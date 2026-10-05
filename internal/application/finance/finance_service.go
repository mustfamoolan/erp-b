package finance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	appaccounting "m3aml-erp/internal/application/accounting"
	appaudit "m3aml-erp/internal/application/audit"
	"m3aml-erp/internal/domain/accounting"
	"m3aml-erp/internal/domain/cashbox"
	domainaudit "m3aml-erp/internal/domain/audit"
	"m3aml-erp/internal/repositories"
)

// FinanceService handles cashbox operations and ensures accounting integrity.
// Rule 17: Cash movements must be transactional (atomic).
// Rule 10 / Roadmap §24: Never directly modify cashbox balances.
// §34: Cash transfers are auditable — AuditService is injected.
type FinanceService struct {
	cashboxRepo   repositories.CashboxRepository
	cashTxRepo    repositories.CashTransactionRepository
	accountingSvc *appaccounting.AccountingService
	auditSvc      *appaudit.AuditService // §34
	accountRepo   repositories.AccountRepository
}

func NewFinanceService(
	cashboxRepo repositories.CashboxRepository,
	cashTxRepo repositories.CashTransactionRepository,
	accountingSvc *appaccounting.AccountingService,
	auditSvc *appaudit.AuditService,
	accountRepo repositories.AccountRepository,
) *FinanceService {
	return &FinanceService{
		cashboxRepo:   cashboxRepo,
		cashTxRepo:    cashTxRepo,
		accountingSvc: accountingSvc,
		auditSvc:      auditSvc,
		accountRepo:   accountRepo,
	}
}

// ─── Create Cashbox ────────────────────────────────────────────────────────────

type CreateCashboxRequest struct {
	ScopeID              uuid.UUID
	Name                 string
	AccountID            uuid.UUID
	Currency             string
}

func (s *FinanceService) CreateCashbox(ctx context.Context, req CreateCashboxRequest) (*cashbox.Cashbox, error) {
	if req.Name == "" || req.AccountID == uuid.Nil || req.ScopeID == uuid.Nil {
		return nil, fmt.Errorf("name, account_id, and scope_id are required")
	}
	currency := req.Currency
	if currency == "" {
		currency = "IQD"
	}
	cb := &cashbox.Cashbox{
		ID:                   uuid.New(),
		ScopeID:              req.ScopeID,
		Name:                 req.Name,
		AccountID:            req.AccountID,
		Currency:             currency,
		TargetOpeningBalance: decimal.Zero,
		ImprestBalance:       decimal.Zero,
		Status:               cashbox.CashboxStatusActive,
	}
	if err := s.cashboxRepo.Save(ctx, cb); err != nil {
		return nil, fmt.Errorf("failed to save cashbox: %w", err)
	}
	return cb, nil
}

// ─── Transfer Funds ────────────────────────────────────────────────────────────

type TransferFundsRequest struct {
	FromCashboxID uuid.UUID
	ToCashboxID   uuid.UUID
	Amount        decimal.Decimal
	Currency      string
	Description   string
	PerformedBy   uuid.UUID
	// ApplyImprest: destination is a factory cashbox receiving a normal request payment.
	// The amount first covers the current deficit; any excess raises imprest_balance.
	ApplyImprest bool
}

// TransferFunds securely moves funds between two cashboxes.
// Creates two CashTransactions + one balanced Journal Entry atomically (Rule 18).
// §34: TRANSFER action is audited on success.
func (s *FinanceService) TransferFunds(ctx context.Context, req TransferFundsRequest) error {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return cashbox.ErrAmountNegative
	}
	if req.FromCashboxID == req.ToCashboxID {
		return fmt.Errorf("cannot transfer to the same cashbox")
	}

	fromBox, err := s.cashboxRepo.FindByID(ctx, req.FromCashboxID)
	if err != nil || fromBox == nil {
		return fmt.Errorf("source cashbox not found")
	}
	if fromBox.Status != cashbox.CashboxStatusActive {
		return cashbox.ErrCashboxInactive
	}

	toBox, err := s.cashboxRepo.FindByID(ctx, req.ToCashboxID)
	if err != nil || toBox == nil {
		return fmt.Errorf("destination cashbox not found")
	}
	if toBox.Status != cashbox.CashboxStatusActive {
		return cashbox.ErrCashboxInactive
	}

	if req.Currency != fromBox.Currency || req.Currency != toBox.Currency {
		return fmt.Errorf("currency mismatch: transfer currency must match both cashboxes")
	}

	now := time.Now()
	fromTx := &cashbox.CashTransaction{
		ID:              uuid.New(),
		CashboxID:       fromBox.ID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Direction:       cashbox.CashDirectionOut,
		SourceType:      "TRANSFER",
		Description:     req.Description,
		PerformedBy:     req.PerformedBy,
		TransactionDate: now,
		Status:          cashbox.CashTransactionCompleted,
	}
	toTx := &cashbox.CashTransaction{
		ID:              uuid.New(),
		CashboxID:       toBox.ID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Direction:       cashbox.CashDirectionIn,
		SourceType:      "TRANSFER",
		Description:     req.Description,
		PerformedBy:     req.PerformedBy,
		TransactionDate: now,
		Status:          cashbox.CashTransactionCompleted,
	}

	var imprestBefore, imprestAfter decimal.Decimal
	transferErr := s.cashTxRepo.ExecuteTransfer(ctx, fromTx, toTx, func(txCtx context.Context) error {
		// Imprest rule (Roadmap Phase 3):
		//   deficit  = max(0, imprest - balance_before)
		//   increase = max(0, amount - deficit)
		// Zero box (imprest 0, balance 0) ⇒ the whole amount becomes imprest.
		if req.ApplyImprest {
			b, a, err := s.applyImprestIncrease(txCtx, toBox.ID, toTx.ID, req.Amount)
			if err != nil {
				return err
			}
			imprestBefore, imprestAfter = b, a
		}

		// Double-entry: Debit destination account, Credit source account
		jeReq := appaccounting.CreateJournalEntryRequest{
			EntryDate:   now,
			Description: req.Description,
			SourceType:  accounting.SourceTypeCashTransfer,
			SourceID:    &fromTx.ID,
			ScopeID:     fromBox.ScopeID,
			CreatedBy:   req.PerformedBy,
			Lines: []appaccounting.JournalLineInput{
				{
					AccountID:   toBox.AccountID,
					Debit:       req.Amount,
					Credit:      decimal.Zero,
					Description: "Cash Transfer Receipt",
					ScopeID:     toBox.ScopeID,
				},
				{
					AccountID:   fromBox.AccountID,
					Debit:       decimal.Zero,
					Credit:      req.Amount,
					Description: "Cash Transfer Disbursement",
					ScopeID:     fromBox.ScopeID,
				},
			},
		}

		entry, err := s.accountingSvc.CreateDraftEntry(txCtx, jeReq)
		if err != nil {
			return fmt.Errorf("accounting entry creation failed: %w", err)
		}
		if err := s.accountingSvc.PostEntry(txCtx, entry.ID, req.PerformedBy); err != nil {
			return fmt.Errorf("accounting entry post failed: %w", err)
		}
		// Link journal back to transactions for audit trail (Rule 21)
		fromTx.JournalEntryID = &entry.ID
		toTx.JournalEntryID = &entry.ID
		return nil
	})

	if transferErr != nil {
		return transferErr
	}

	// §34: Audit TRANSFER — fires AFTER atomic operation succeeds
	if s.auditSvc != nil {
		fromBoxID := fromBox.ID
		newValues := map[string]any{
			"from_cashbox": fromBox.Name,
			"to_cashbox":   toBox.Name,
			"amount":       req.Amount.String(),
			"currency":     req.Currency,
		}
		if req.ApplyImprest {
			newValues["imprest_before"] = imprestBefore.String()
			newValues["imprest_after"] = imprestAfter.String()
		}
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     req.PerformedBy,
			ScopeID:    &fromBox.ScopeID,
			Action:     domainaudit.AuditTransfer,
			EntityType: "cashbox_transfer",
			EntityID:   &fromBoxID,
			NewValues:  newValues,
		})
	}

	return nil
}

// ComputeImprestIncrease returns how much the imprest (رصيد المداورة) rises
// when `amount` is received by a factory cashbox.
//   deficit  = max(0, imprest - balanceBefore)   — what the factory consumed
//   increase = max(0, amount - deficit)          — the excess above the deficit
func ComputeImprestIncrease(imprest, balanceBefore, amount decimal.Decimal) decimal.Decimal {
	deficit := imprest.Sub(balanceBefore)
	if deficit.IsNegative() {
		deficit = decimal.Zero
	}
	increase := amount.Sub(deficit)
	if increase.IsNegative() {
		return decimal.Zero
	}
	return increase
}

// applyImprestIncrease runs inside the transfer DB transaction. It locks the cashbox row,
// derives the balance BEFORE this transfer from cash_transactions (Rule 10), and raises
// imprest_balance by the excess only. Returns (imprestBefore, imprestAfter).
func (s *FinanceService) applyImprestIncrease(txCtx context.Context, cashboxID, newTxID uuid.UUID, amount decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
	gormTx, ok := txCtx.Value("tx").(*gorm.DB)
	if !ok || gormTx == nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("imprest update requires an active DB transaction")
	}

	var locked struct{ ImprestBalance decimal.Decimal }
	if err := gormTx.Raw(`SELECT imprest_balance FROM cashboxes WHERE id = ? FOR UPDATE`, cashboxID).
		Scan(&locked).Error; err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("failed to lock cashbox for imprest update: %w", err)
	}

	var bal struct{ Balance decimal.Decimal }
	if err := gormTx.Raw(`
		SELECT COALESCE(SUM(CASE WHEN direction = 'IN' THEN amount ELSE -amount END), 0) AS balance
		FROM cash_transactions
		WHERE cashbox_id = ? AND status = 'COMPLETED' AND id <> ?`, cashboxID, newTxID).
		Scan(&bal).Error; err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("failed to compute cashbox balance: %w", err)
	}

	increase := ComputeImprestIncrease(locked.ImprestBalance, bal.Balance, amount)
	after := locked.ImprestBalance.Add(increase)
	if increase.IsPositive() {
		if err := gormTx.Exec(`UPDATE cashboxes SET imprest_balance = ?, updated_at = NOW() WHERE id = ?`, after, cashboxID).Error; err != nil {
			return decimal.Zero, decimal.Zero, fmt.Errorf("failed to update imprest balance: %w", err)
		}
	}
	return locked.ImprestBalance, after, nil
}

// GetCashboxes returns all cashboxes for a scope.
func (s *FinanceService) GetCashboxes(ctx context.Context, scopeID uuid.UUID) ([]cashbox.Cashbox, error) {
	return s.cashboxRepo.FindByScope(ctx, scopeID)
}

// GetCashboxTransactions returns all transactions for a cashbox.
func (s *FinanceService) GetCashboxTransactions(ctx context.Context, cashboxID uuid.UUID) ([]cashbox.CashTransaction, error) {
	return s.cashTxRepo.FindByCashbox(ctx, cashboxID)
}

// ─── Establish Opening Balance (Roadmap §9, Phase 4) ──────────────────────────

type EstablishOpeningBalanceRequest struct {
	CashboxID   uuid.UUID
	PerformedBy uuid.UUID
}

// EstablishOpeningBalance officially records the opening financial entry.
// It checks that no transactions currently exist, and deposits the TargetOpeningBalance.
func (s *FinanceService) EstablishOpeningBalance(ctx context.Context, req EstablishOpeningBalanceRequest) error {
	cb, err := s.cashboxRepo.FindByID(ctx, req.CashboxID)
	if err != nil || cb == nil {
		return fmt.Errorf("cashbox not found")
	}

	if cb.TargetOpeningBalance.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("target opening balance must be greater than zero")
	}

	// Make sure no transactions exist yet to prevent duplication.
	existingTxs, err := s.cashTxRepo.FindByCashbox(ctx, cb.ID)
	if err != nil {
		return fmt.Errorf("failed to check existing transactions: %w", err)
	}
	if len(existingTxs) > 0 {
		return fmt.Errorf("cashbox already has transactions, opening balance cannot be established again")
	}

	// Create the opening balance transaction.
	now := time.Now()
	tx := &cashbox.CashTransaction{
		ID:              uuid.New(),
		CashboxID:       cb.ID,
		Amount:          cb.TargetOpeningBalance,
		Currency:        cb.Currency,
		Direction:       cashbox.CashDirectionIn,
		SourceType:      "OPENING_BALANCE",
		Description:     "Initial Factory Opening Balance Allocation",
		PerformedBy:     req.PerformedBy,
		TransactionDate: now,
		Status:          cashbox.CashTransactionCompleted,
	}

	err = s.cashTxRepo.ExecuteTransfer(ctx, nil, tx, func(txCtx context.Context) error {
		// using DB instance attached to the context (if available, otherwise fallback)
		// For proper architecture, we could add a CheckExists method, but since we're in a transaction,
		// we'll rely on the partial unique index we created for exact safety, and also do a basic check here if possible.
		// Since we don't have direct DB access here, the DB index will catch race conditions.
		
		// Find the Opening Balance account (3100)
		openingBalanceAccount, eqErr := s.accountRepo.FindByCode(txCtx, "3100")
		if eqErr != nil || openingBalanceAccount == nil {
			return fmt.Errorf("opening balance account (3100) not found")
		}

		jeReq := appaccounting.CreateJournalEntryRequest{
			EntryDate:   now,
			Description: "Factory Opening Balance",
			SourceType:  "OPENING_BALANCE",
			SourceID:    &tx.ID,
			ScopeID:     cb.ScopeID,
			CreatedBy:   req.PerformedBy,
			Lines: []appaccounting.JournalLineInput{
				{
					AccountID:   cb.AccountID, // Debit the Cashbox
					Debit:       cb.TargetOpeningBalance,
					Credit:      decimal.Zero,
					Description: "Opening Cash",
					ScopeID:     cb.ScopeID,
				},
				{
					AccountID:   openingBalanceAccount.ID, // Credit 3100
					Debit:       decimal.Zero,
					Credit:      cb.TargetOpeningBalance,
					Description: "Opening Balance Equity Offset",
					ScopeID:     cb.ScopeID,
				},
			},
		}

		entry, accErr := s.accountingSvc.CreateDraftEntry(txCtx, jeReq)
		if accErr != nil {
			return fmt.Errorf("accounting entry creation failed: %w", accErr)
		}
		if accErr := s.accountingSvc.PostEntry(txCtx, entry.ID, req.PerformedBy); accErr != nil {
			return fmt.Errorf("accounting entry post failed: %w", accErr)
		}
		tx.JournalEntryID = &entry.ID
		return nil
	})

	if err != nil {
		return err
	}

	// Audit
	if s.auditSvc != nil {
		cbID := cb.ID
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     req.PerformedBy,
			ScopeID:    &cb.ScopeID,
			Action:     "ESTABLISH_OPENING_BALANCE",
			EntityType: "cashbox",
			EntityID:   &cbID,
			NewValues: map[string]any{
				"cashbox_name":   cb.Name,
				"amount":         cb.TargetOpeningBalance.String(),
				"currency":       cb.Currency,
			},
		})
	}

	return nil
}

// ─── Employee Custody ────────────────────────────────────────────────────────
type CustodyRequest struct {
	CashboxID   uuid.UUID
	EmployeeID  *uuid.UUID
	PersonName  string
	Amount      decimal.Decimal
	Currency    string
	Description string
	PerformedBy uuid.UUID
}

func (s *FinanceService) WithdrawCustody(ctx context.Context, req CustodyRequest) error {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return cashbox.ErrAmountNegative
	}
	desc := req.Description
	if req.PersonName != "" {
		if desc != "" {
			desc = fmt.Sprintf("المستلم: %s — %s", req.PersonName, desc)
		} else {
			desc = fmt.Sprintf("المستلم: %s", req.PersonName)
		}
	} else if desc == "" {
		desc = "سحب سلفة نقدية"
	}

	box, err := s.cashboxRepo.FindByID(ctx, req.CashboxID)
	if err != nil {
		return err
	}

	tx := &cashbox.CashTransaction{
		ID:              uuid.New(),
		CashboxID:       req.CashboxID,
		Direction:       cashbox.CashDirectionOut,
		Amount:          req.Amount,
		Currency:        req.Currency,
		SourceType:      "EMPLOYEE_CUSTODY_WITHDRAW",
		SourceID:        req.EmployeeID,
		Description:     desc,
		PerformedBy:     req.PerformedBy,
		TransactionDate: time.Now(),
		Status:          cashbox.CashTransactionCompleted,
	}

	return s.cashTxRepo.ExecuteTransfer(ctx, tx, nil, func(txCtx context.Context) error {
		custodyAcct, err := s.accountRepo.FindByCode(txCtx, "1120")
		if err != nil || custodyAcct == nil {
			return fmt.Errorf("employee custody account (1120) not found: %w", err)
		}

		jeReq := appaccounting.CreateJournalEntryRequest{
			EntryDate:   time.Now(),
			Description: desc,
			SourceType:  "CASH_TRANSACTION",
			SourceID:    &tx.ID,
			ScopeID:     box.ScopeID,
			CreatedBy:   req.PerformedBy,
			Lines: []appaccounting.JournalLineInput{
				{
					AccountID:   custodyAcct.ID,
					Debit:       req.Amount,
					Credit:      decimal.Zero,
					Description: desc,
					ScopeID:     box.ScopeID,
				},
				{
					AccountID:   box.AccountID,
					Debit:       decimal.Zero,
					Credit:      req.Amount,
					Description: "Cashbox Withdrawal",
					ScopeID:     box.ScopeID,
				},
			},
		}

		entry, err := s.accountingSvc.CreateDraftEntry(txCtx, jeReq)
		if err != nil {
			return fmt.Errorf("accounting entry creation failed: %w", err)
		}
		if err := s.accountingSvc.PostEntry(txCtx, entry.ID, req.PerformedBy); err != nil {
			return fmt.Errorf("accounting entry post failed: %w", err)
		}

		tx.JournalEntryID = &entry.ID
		return nil
	})
}

func (s *FinanceService) DepositCustody(ctx context.Context, req CustodyRequest) error {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return cashbox.ErrAmountNegative
	}
	desc := req.Description
	if req.PersonName != "" {
		if desc != "" {
			desc = fmt.Sprintf("المسلّم: %s — %s", req.PersonName, desc)
		} else {
			desc = fmt.Sprintf("المسلّم: %s", req.PersonName)
		}
	} else if desc == "" {
		desc = "إيداع (إرجاع) سلفة"
	}

	box, err := s.cashboxRepo.FindByID(ctx, req.CashboxID)
	if err != nil {
		return err
	}

	tx := &cashbox.CashTransaction{
		ID:              uuid.New(),
		CashboxID:       req.CashboxID,
		Direction:       cashbox.CashDirectionIn,
		Amount:          req.Amount,
		Currency:        req.Currency,
		SourceType:      "EMPLOYEE_CUSTODY_DEPOSIT",
		SourceID:        req.EmployeeID,
		Description:     desc,
		PerformedBy:     req.PerformedBy,
		TransactionDate: time.Now(),
		Status:          cashbox.CashTransactionCompleted,
	}

	return s.cashTxRepo.ExecuteTransfer(ctx, nil, tx, func(txCtx context.Context) error {
		custodyAcct, err := s.accountRepo.FindByCode(txCtx, "1120")
		if err != nil || custodyAcct == nil {
			return fmt.Errorf("employee custody account (1120) not found: %w", err)
		}

		jeReq := appaccounting.CreateJournalEntryRequest{
			EntryDate:   time.Now(),
			Description: desc,
			SourceType:  "CASH_TRANSACTION",
			SourceID:    &tx.ID,
			ScopeID:     box.ScopeID,
			CreatedBy:   req.PerformedBy,
			Lines: []appaccounting.JournalLineInput{
				{
					AccountID:   box.AccountID,
					Debit:       req.Amount,
					Credit:      decimal.Zero,
					Description: "Cashbox Deposit",
					ScopeID:     box.ScopeID,
				},
				{
					AccountID:   custodyAcct.ID,
					Debit:       decimal.Zero,
					Credit:      req.Amount,
					Description: desc,
					ScopeID:     box.ScopeID,
				},
			},
		}

		entry, err := s.accountingSvc.CreateDraftEntry(txCtx, jeReq)
		if err != nil {
			return fmt.Errorf("accounting entry creation failed: %w", err)
		}
		if err := s.accountingSvc.PostEntry(txCtx, entry.ID, req.PerformedBy); err != nil {
			return fmt.Errorf("accounting entry post failed: %w", err)
		}

		tx.JournalEntryID = &entry.ID
		return nil
	})
}
