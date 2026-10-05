package reportingrepo

import (
	"context"
	"m3aml-erp/internal/domain/reporting"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"github.com/shopspring/decimal"
)

type Repository struct {
	db *gorm.DB
}

func NewReportingRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GetAccountStatement fetches the journal lines for an account and calculates the running balance
func (r *Repository) GetAccountStatement(ctx context.Context, accountID uuid.UUID, filter reporting.DateRange) (*reporting.AccountStatementReport, error) {
	var account struct {
		ID   uuid.UUID
		Code string
		Name string
	}
	if err := r.db.WithContext(ctx).Table("accounts").Select("id, code, name").Where("id = ?", accountID).First(&account).Error; err != nil {
		return nil, err
	}

	report := &reporting.AccountStatementReport{
		AccountID:   account.ID,
		AccountCode: account.Code,
		AccountName: account.Name,
		TotalDebit:  decimal.Zero,
		TotalCredit: decimal.Zero,
	}

	// Calculate opening balance before 'From' date if specified
	if filter.From != nil {
		var opening struct {
			SumDebit  decimal.Decimal
			SumCredit decimal.Decimal
		}
		r.db.WithContext(ctx).Table("journal_lines").
			Select("COALESCE(SUM(debit), 0) as sum_debit, COALESCE(SUM(credit), 0) as sum_credit").
			Joins("JOIN journal_entries je ON je.id = journal_lines.journal_entry_id").
			Where("journal_lines.account_id = ? AND je.status = 'POSTED' AND je.entry_date < ?", accountID, *filter.From).
			Scan(&opening)
		report.OpeningBalance = opening.SumDebit.Sub(opening.SumCredit)
	}

	// Fetch lines
	query := r.db.WithContext(ctx).Table("journal_lines").
		Select("je.entry_date as date, je.id as journal_id, je.entry_number, journal_lines.description, journal_lines.debit, journal_lines.credit, journal_lines.currency, journal_lines.foreign_debit, journal_lines.foreign_credit").
		Joins("JOIN journal_entries je ON je.id = journal_lines.journal_entry_id").
		Where("journal_lines.account_id = ? AND je.status = 'POSTED'", accountID)

	if filter.From != nil {
		query = query.Where("je.entry_date >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("je.entry_date <= ?", *filter.To)
	}

	query = query.Order("je.entry_date ASC, je.created_at ASC")

	var rawLines []reporting.AccountStatementLine
	if err := query.Scan(&rawLines).Error; err != nil {
		return nil, err
	}

	runningBalance := report.OpeningBalance
	for i := range rawLines {
		runningBalance = runningBalance.Add(rawLines[i].Debit).Sub(rawLines[i].Credit)
		rawLines[i].Balance = runningBalance
		report.TotalDebit = report.TotalDebit.Add(rawLines[i].Debit)
		report.TotalCredit = report.TotalCredit.Add(rawLines[i].Credit)
	}

	report.Lines = rawLines
	report.ClosingBalance = runningBalance

	return report, nil
}

// GetTrialBalance fetches the aggregated debits and credits for all postable accounts
func (r *Repository) GetTrialBalance(ctx context.Context, filter reporting.DateRange) (*reporting.TrialBalanceReport, error) {
	query := r.db.WithContext(ctx).Table("accounts").
		Select("accounts.id as account_id, accounts.code as account_code, accounts.name as account_name, COALESCE(SUM(jl.debit), 0) as debit, COALESCE(SUM(jl.credit), 0) as credit").
		Joins("LEFT JOIN journal_lines jl ON jl.account_id = accounts.id").
		Joins("LEFT JOIN journal_entries je ON je.id = jl.journal_entry_id AND je.status = 'POSTED'")

	if filter.From != nil {
		query = query.Where("je.entry_date >= ? OR je.entry_date IS NULL", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("je.entry_date <= ? OR je.entry_date IS NULL", *filter.To)
	}

	query = query.Where("accounts.is_postable = ?", true).
		Group("accounts.id, accounts.code, accounts.name").
		Order("accounts.code ASC")

	var lines []reporting.TrialBalanceLine
	if err := query.Scan(&lines).Error; err != nil {
		return nil, err
	}

	report := &reporting.TrialBalanceReport{
		TotalDebit:  decimal.Zero,
		TotalCredit: decimal.Zero,
	}

	for i := range lines {
		lines[i].Balance = lines[i].Debit.Sub(lines[i].Credit)
		report.TotalDebit = report.TotalDebit.Add(lines[i].Debit)
		report.TotalCredit = report.TotalCredit.Add(lines[i].Credit)
	}
	report.Lines = lines

	return report, nil
}

// GetStockBalance fetches current aggregated stock grouped by warehouse and variant
func (r *Repository) GetStockBalance(ctx context.Context, scopeID *uuid.UUID) (*reporting.StockBalanceReport, error) {
	query := r.db.WithContext(ctx).Table("inventory_stock as st").
		Select("st.warehouse_id, w.name as warehouse_name, st.variant_id, v.sku, m.name as material_name, st.quantity").
		Joins("JOIN warehouses w ON w.id = st.warehouse_id").
		Joins("JOIN material_variants v ON v.id = st.variant_id").
		Joins("JOIN materials m ON m.id = v.material_id").
		Where("st.quantity > 0")

	if scopeID != nil && *scopeID != uuid.Nil {
		query = query.Where("w.scope_id = ?", *scopeID)
	}

	query = query.Order("w.name ASC, m.name ASC, v.sku ASC")

	var lines []reporting.StockBalanceLine
	if err := query.Scan(&lines).Error; err != nil {
		return nil, err
	}

	return &reporting.StockBalanceReport{Lines: lines}, nil
}
