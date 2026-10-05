package accounting

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	appaudit "m3aml-erp/internal/application/audit"
	"m3aml-erp/internal/domain/accounting"
	domainaudit "m3aml-erp/internal/domain/audit"
	"m3aml-erp/internal/repositories"
)

// AccountingService is the single accounting engine for the entire company.
// Rule 21: Do NOT create two independent accounting systems.
// Rule 22: Use one accounting engine with scope/cost-center dimensions.
// §34: Financial actions are strongly auditable — AuditService is injected here.
type AccountingService struct {
	journalRepo repositories.JournalEntryRepository
	accountRepo repositories.AccountRepository
	periodRepo  repositories.AccountingPeriodRepository
	auditSvc    *appaudit.AuditService // §34 — mandatory for financial audit trail
}

func NewAccountingService(
	journalRepo repositories.JournalEntryRepository,
	accountRepo repositories.AccountRepository,
	periodRepo  repositories.AccountingPeriodRepository,
	auditSvc    *appaudit.AuditService,
) *AccountingService {
	return &AccountingService{
		journalRepo: journalRepo,
		accountRepo: accountRepo,
		periodRepo:  periodRepo,
		auditSvc:    auditSvc,
	}
}

// ============================================================
// CreateJournalEntryRequest — what a caller provides.
// ============================================================

// JournalLineInput — what a caller provides per line.
// Debit and Credit MUST always be in IQD (base currency) — Rule 14.
// ForeignCurrency fields are optional and for reference/reporting only.
type JournalLineInput struct {
	AccountID       uuid.UUID
	Debit           decimal.Decimal  // ALWAYS IQD — Rule 14
	Credit          decimal.Decimal  // ALWAYS IQD — Rule 14
	Description     string
	ScopeID         uuid.UUID
	CostCenterID    *uuid.UUID
	Reference       string
	// Optional: original foreign-currency amounts — for reporting only, do not affect balance
	ForeignCurrency *string
	ForeignDebit    decimal.Decimal
	ForeignCredit   decimal.Decimal
	ExchangeRate    *decimal.Decimal
}

type CreateJournalEntryRequest struct {
	EntryDate   time.Time
	Description string
	SourceType  accounting.SourceType
	SourceID    *uuid.UUID
	ScopeID     uuid.UUID
	Lines       []JournalLineInput
	CreatedBy   uuid.UUID
}

// CreateDraftEntry creates a new DRAFT journal entry.
// Validates all lines before saving. Does NOT post.
func (s *AccountingService) CreateDraftEntry(ctx context.Context, req CreateJournalEntryRequest) (*accounting.JournalEntry, error) {
	if len(req.Lines) < 2 {
		return nil, fmt.Errorf("a journal entry must have at least 2 lines")
	}

	// 1. Verify there is an OPEN period for the entry date — Roadmap §18
	period, err := s.periodRepo.FindOpenForDate(ctx, req.EntryDate)
	if err != nil {
		return nil, fmt.Errorf("period lookup: %w", err)
	}
	if period == nil {
		return nil, accounting.ErrPeriodClosed
	}

	// 2. Build and validate lines
	lines := make([]accounting.JournalLine, 0, len(req.Lines))
	for _, li := range req.Lines {
		// Verify account is postable — Roadmap §14
		acct, err := s.accountRepo.FindByID(ctx, li.AccountID)
		if err != nil {
			return nil, fmt.Errorf("account lookup: %w", err)
		}
		if acct == nil || !acct.IsActive {
			return nil, fmt.Errorf("account %s not found or inactive", li.AccountID)
		}
		if !acct.IsPostable {
			return nil, accounting.ErrNotPostable
		}

		line := accounting.JournalLine{
			ID:              uuid.New(),
			AccountID:       li.AccountID,
			Debit:           li.Debit,
			Credit:          li.Credit,
			Description:     li.Description,
			ScopeID:         li.ScopeID,
			CostCenterID:    li.CostCenterID,
			Reference:       li.Reference,
			// Multi-currency reference fields — for reporting only
			ForeignCurrency: li.ForeignCurrency,
			ForeignDebit:    li.ForeignDebit,
			ForeignCredit:   li.ForeignCredit,
			ExchangeRate:    li.ExchangeRate,
		}
		// Domain validation: debit/credit rules
		if err := line.Validate(); err != nil {
			return nil, fmt.Errorf("line validation: %w", err)
		}
		lines = append(lines, line)
	}

	// 3. Balance validation — SUM(debit) == SUM(credit) — Roadmap §16, Rule 13
	if err := accounting.ValidateBalance(lines); err != nil {
		return nil, err
	}

	// 4. Generate entry number
	entryNumber, err := s.journalRepo.NextEntryNumber(ctx, req.EntryDate.Year())
	if err != nil {
		return nil, fmt.Errorf("entry number generation: %w", err)
	}

	// 5. Build and save DRAFT entry
	entry := &accounting.JournalEntry{
		ID:          uuid.New(),
		EntryNumber: entryNumber,
		EntryDate:   req.EntryDate,
		Description: req.Description,
		SourceType:  req.SourceType,
		SourceID:    req.SourceID,
		ScopeID:     req.ScopeID,
		PeriodID:    period.ID,
		Status:      accounting.JournalStatusDraft,
		CreatedBy:   req.CreatedBy,
		Lines:       lines,
	}

	if err := s.journalRepo.Save(ctx, entry); err != nil {
		return nil, fmt.Errorf("save journal entry: %w", err)
	}

	// §34: Audit CREATE — best-effort, non-blocking
	if s.auditSvc != nil {
		scopeID := req.ScopeID
		entryID := entry.ID
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     req.CreatedBy,
			ScopeID:    &scopeID,
			Action:     domainaudit.AuditCreate,
			EntityType: "journal_entry",
			EntityID:   &entryID,
			NewValues:  map[string]any{"entry_number": entry.EntryNumber, "status": "DRAFT", "description": entry.Description},
		})
	}

	return entry, nil
}

// PostEntry transitions a DRAFT entry to POSTED.
// Re-validates balance before posting — never trusts prior state.
// Rule 13: Every accounting entry must be balanced.
// Rule 18: Executed within a PostgreSQL transaction (in the repo).
func (s *AccountingService) PostEntry(ctx context.Context, entryID uuid.UUID, postedBy uuid.UUID) error {
	entry, err := s.journalRepo.FindByID(ctx, entryID)
	if err != nil {
		return fmt.Errorf("post entry: lookup failed: %w", err)
	}
	if entry == nil {
		return fmt.Errorf("journal entry not found: %s", entryID)
	}
	if entry.Status != accounting.JournalStatusDraft {
		return accounting.ErrEntryAlreadyPosted
	}

	// Re-validate balance before posting — defensive, Rule 13
	if err := accounting.ValidateBalance(entry.Lines); err != nil {
		return err
	}

	// Verify period is still OPEN — Roadmap §18
	period, err := s.periodRepo.FindByID(ctx, entry.PeriodID)
	if err != nil {
		return fmt.Errorf("period lookup: %w", err)
	}
	if period == nil || period.Status == accounting.AccountingPeriodClosed {
		return accounting.ErrPeriodClosed
	}

	if err := s.journalRepo.Post(ctx, entryID, postedBy); err != nil {
		return err
	}

	// §34: Audit POST — best-effort
	if s.auditSvc != nil {
		scopeID := entry.ScopeID
		eid := entryID
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     postedBy,
			ScopeID:    &scopeID,
			Action:     domainaudit.AuditPost,
			EntityType: "journal_entry",
			EntityID:   &eid,
			OldValues:  map[string]any{"status": "DRAFT"},
			NewValues:  map[string]any{"status": "POSTED"},
		})
	}

	return nil
}

// ReverseEntry creates a mirror DRAFT entry with all debits/credits swapped.
// Rule 15: Financial corrections must use reversal/adjustment mechanisms.
// Rule 14: The original entry remains — it is NOT deleted.
func (s *AccountingService) ReverseEntry(ctx context.Context, entryID uuid.UUID, reason string, reversedBy uuid.UUID) (*accounting.JournalEntry, error) {
	original, err := s.journalRepo.FindByID(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("reverse entry: lookup failed: %w", err)
	}
	if original == nil {
		return nil, fmt.Errorf("journal entry not found: %s", entryID)
	}
	if original.Status != accounting.JournalStatusPosted {
		return nil, fmt.Errorf("only POSTED entries can be reversed")
	}

	// Swap debit/credit on each line
	reversalLines := make([]JournalLineInput, 0, len(original.Lines))
	for _, l := range original.Lines {
		reversalLines = append(reversalLines, JournalLineInput{
			AccountID:    l.AccountID,
			Debit:        l.Credit, // swapped
			Credit:       l.Debit, // swapped
			Description:  fmt.Sprintf("عكس: %s", l.Description),
			ScopeID:      l.ScopeID,
			CostCenterID: l.CostCenterID,
			Reference:    original.EntryNumber,
		})
	}

	reversalID := original.ID
	req := CreateJournalEntryRequest{
		EntryDate:   time.Now(),
		Description: fmt.Sprintf("عكس القيد %s — %s", original.EntryNumber, reason),
		SourceType:  accounting.SourceTypeReversal,
		SourceID:    &reversalID,
		ScopeID:     original.ScopeID,
		Lines:       reversalLines,
		CreatedBy:   reversedBy,
	}
	reversal, err := s.CreateDraftEntry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create reversal: %w", err)
	}

	// Mark the original as REVERSED — Rule 14: original remains visible
	_ = s.journalRepo.Void(ctx, entryID, reversedBy)

	// §34: Audit REVERSE — best-effort
	if s.auditSvc != nil {
		scopeID := original.ScopeID
		eid := entryID
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     reversedBy,
			ScopeID:    &scopeID,
			Action:     domainaudit.AuditReverse,
			EntityType: "journal_entry",
			EntityID:   &eid,
			OldValues:  map[string]any{"status": "POSTED", "entry_number": original.EntryNumber},
			NewValues:  map[string]any{"status": "REVERSED", "reversal_entry": reversal.EntryNumber, "reason": reason},
		})
	}

	return reversal, nil
}

// GetAccountBalance returns the current balance of an account for a scope.
// Always computed from posted journal lines — Rule 10.
func (s *AccountingService) GetAccountBalance(ctx context.Context, accountID uuid.UUID, scopeID uuid.UUID) (map[string]interface{}, error) {
	return s.journalRepo.GetAccountBalance(ctx, accountID, scopeID)
}

// ClosePeriod closes an accounting period.
// §34: Audit CLOSE_PERIOD.
func (s *AccountingService) ClosePeriod(ctx context.Context, periodID uuid.UUID, closedBy uuid.UUID) error {
	period, err := s.periodRepo.FindByID(ctx, periodID)
	if err != nil || period == nil {
		return fmt.Errorf("period not found")
	}
	if period.Status == accounting.AccountingPeriodClosed {
		return fmt.Errorf("period is already closed")
	}

	if err := s.periodRepo.Close(ctx, periodID, closedBy); err != nil {
		return err
	}

	// §34: Audit CLOSE_PERIOD — best-effort
	if s.auditSvc != nil {
		eid := periodID
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     closedBy,
			Action:     domainaudit.AuditClosePeriod,
			EntityType: "accounting_period",
			EntityID:   &eid,
			OldValues:  map[string]any{"status": "OPEN"},
			NewValues:  map[string]any{"status": "CLOSED"},
		})
	}

	return nil
}
