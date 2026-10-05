package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"m3aml-erp/internal/domain/accounting"
)

// ============================================================
// ACCOUNT REPOSITORY
// ============================================================

type accountRepository struct{ db *gorm.DB }

func NewAccountRepository(db *gorm.DB) *accountRepository { return &accountRepository{db: db} }

func (r *accountRepository) FindAll(ctx context.Context) ([]accounting.Account, error) {
	var accounts []accounting.Account
	return accounts, r.db.WithContext(ctx).Order("sort_order, code").Find(&accounts).Error
}

func (r *accountRepository) FindByID(ctx context.Context, id uuid.UUID) (*accounting.Account, error) {
	var a accounting.Account
	err := r.db.WithContext(ctx).First(&a, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &a, err
}

func (r *accountRepository) FindByCode(ctx context.Context, code string) (*accounting.Account, error) {
	var a accounting.Account
	err := r.db.WithContext(ctx).First(&a, "code = ?", code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &a, err
}

func (r *accountRepository) FindChildren(ctx context.Context, parentID uuid.UUID) ([]accounting.Account, error) {
	var accounts []accounting.Account
	return accounts, r.db.WithContext(ctx).Where("parent_id = ? AND is_active = TRUE", parentID).Order("sort_order, code").Find(&accounts).Error
}

func (r *accountRepository) FindPostable(ctx context.Context) ([]accounting.Account, error) {
	var accounts []accounting.Account
	return accounts, r.db.WithContext(ctx).Where("is_postable = TRUE AND is_active = TRUE").Order("code").Find(&accounts).Error
}

func (r *accountRepository) Save(ctx context.Context, a *accounting.Account) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *accountRepository) Update(ctx context.Context, a *accounting.Account) error {
	return r.db.WithContext(ctx).Save(a).Error
}

// ============================================================
// ACCOUNTING PERIOD REPOSITORY
// ============================================================

type accountingPeriodRepository struct{ db *gorm.DB }

func NewAccountingPeriodRepository(db *gorm.DB) *accountingPeriodRepository {
	return &accountingPeriodRepository{db: db}
}

func (r *accountingPeriodRepository) FindOpenForDate(ctx context.Context, date time.Time) (*accounting.AccountingPeriod, error) {
	var period accounting.AccountingPeriod
	err := r.db.WithContext(ctx).
		Where("start_date <= ? AND end_date >= ? AND status = 'OPEN'", date, date).
		First(&period).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &period, err
}

func (r *accountingPeriodRepository) FindByID(ctx context.Context, id uuid.UUID) (*accounting.AccountingPeriod, error) {
	var period accounting.AccountingPeriod
	err := r.db.WithContext(ctx).First(&period, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &period, err
}

func (r *accountingPeriodRepository) FindByFiscalYear(ctx context.Context, fyID uuid.UUID) ([]accounting.AccountingPeriod, error) {
	var periods []accounting.AccountingPeriod
	return periods, r.db.WithContext(ctx).Where("fiscal_year_id = ?", fyID).Order("period_number").Find(&periods).Error
}

func (r *accountingPeriodRepository) Close(ctx context.Context, periodID uuid.UUID, closedBy uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&accounting.AccountingPeriod{}).
		Where("id = ? AND status = 'OPEN'", periodID).
		Updates(map[string]interface{}{
			"status":     "CLOSED",
			"closed_by":  closedBy,
			"closed_at":  now,
			"updated_at": now,
		}).Error
}

// ============================================================
// JOURNAL ENTRY REPOSITORY
// ============================================================

type journalEntryRepository struct{ db *gorm.DB }

func NewJournalEntryRepository(db *gorm.DB) *journalEntryRepository {
	return &journalEntryRepository{db: db}
}

func (r *journalEntryRepository) FindByID(ctx context.Context, id uuid.UUID) (*accounting.JournalEntry, error) {
	var entry accounting.JournalEntry
	err := r.db.WithContext(ctx).Preload("Lines").First(&entry, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &entry, err
}

func (r *journalEntryRepository) FindByNumber(ctx context.Context, number string) (*accounting.JournalEntry, error) {
	var entry accounting.JournalEntry
	err := r.db.WithContext(ctx).First(&entry, "entry_number = ?", number).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &entry, err
}

func (r *journalEntryRepository) FindByScope(ctx context.Context, scopeID uuid.UUID, limit, offset int) ([]accounting.JournalEntry, int64, error) {
	var entries []accounting.JournalEntry
	var total int64
	q := r.db.WithContext(ctx).Model(&accounting.JournalEntry{}).Where("scope_id = ?", scopeID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("entry_date DESC, entry_number DESC").Limit(limit).Offset(offset).Find(&entries).Error
	return entries, total, err
}

func (r *journalEntryRepository) FindByPeriod(ctx context.Context, periodID uuid.UUID) ([]accounting.JournalEntry, error) {
	var entries []accounting.JournalEntry
	return entries, r.db.WithContext(ctx).Where("period_id = ?", periodID).Order("entry_date, entry_number").Find(&entries).Error
}

func (r *journalEntryRepository) FindBySourceDocument(ctx context.Context, sourceType accounting.SourceType, sourceID uuid.UUID) (*accounting.JournalEntry, error) {
	var entry accounting.JournalEntry
	err := r.db.WithContext(ctx).Where("source_type = ? AND source_id = ?", sourceType, sourceID).First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &entry, err
}

func (r *journalEntryRepository) Save(ctx context.Context, entry *accounting.JournalEntry) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// GORM automatically saves associated 'Lines' due to the foreignKey tag
		if err := tx.Create(entry).Error; err != nil {
			return err
		}
		return nil
	})
}

// Post transitions a DRAFT entry to POSTED within a transaction.
// Rule 18: Use PostgreSQL transactions for atomic operations.
// Balance validation MUST be done BEFORE calling Post.
func (r *journalEntryRepository) Post(ctx context.Context, entryID uuid.UUID, postedBy uuid.UUID) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&accounting.JournalEntry{}).
		Where("id = ? AND status = 'DRAFT'", entryID).
		Updates(map[string]interface{}{
			"status":     "POSTED",
			"posted_by":  postedBy,
			"posted_at":  now,
			"updated_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return accounting.ErrEntryAlreadyPosted
	}
	return nil
}

// Void marks a POSTED entry as VOIDED. Does NOT delete — Rule 14.
func (r *journalEntryRepository) Void(ctx context.Context, entryID uuid.UUID, voidedBy uuid.UUID) error {
	_ = voidedBy // TODO: Phase 7 — log to audit trail
	result := r.db.WithContext(ctx).Model(&accounting.JournalEntry{}).
		Where("id = ? AND status = 'POSTED'", entryID).
		Updates(map[string]interface{}{
			"status":     "VOIDED",
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("entry not found or not in POSTED status")
	}
	return nil
}

// NextEntryNumber generates JE-YYYY-NNNNNN using the DB sequence.
func (r *journalEntryRepository) NextEntryNumber(ctx context.Context, year int) (string, error) {
	var seq int64
	if err := r.db.WithContext(ctx).Raw("SELECT nextval('journal_entry_seq')").Scan(&seq).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("JE-%d-%06d", year, seq), nil
}

// GetAccountBalance returns balance computed from posted journal lines — Rule 10.
func (r *journalEntryRepository) GetAccountBalance(ctx context.Context, accountID uuid.UUID, scopeID uuid.UUID) (map[string]interface{}, error) {
	var result struct {
		TotalDebit  float64
		TotalCredit float64
		NetBalance  float64
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(jl.debit),  0) AS total_debit,
			COALESCE(SUM(jl.credit), 0) AS total_credit,
			COALESCE(SUM(jl.debit) - SUM(jl.credit), 0) AS net_balance
		FROM journal_lines jl
		JOIN journal_entries je ON je.id = jl.journal_entry_id
		WHERE jl.account_id = ?
		  AND jl.scope_id   = ?
		  AND je.status     = 'POSTED'
	`, accountID, scopeID).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"total_debit":  result.TotalDebit,
		"total_credit": result.TotalCredit,
		"net_balance":  result.NetBalance,
	}, nil
}
