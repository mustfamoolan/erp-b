package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"m3aml-erp/internal/domain/accounting"
)

// AccountRepository defines persistence for the Chart of Accounts.
type AccountRepository interface {
	FindAll(ctx context.Context) ([]accounting.Account, error)
	FindByID(ctx context.Context, id uuid.UUID) (*accounting.Account, error)
	FindByCode(ctx context.Context, code string) (*accounting.Account, error)
	FindChildren(ctx context.Context, parentID uuid.UUID) ([]accounting.Account, error)
	FindPostable(ctx context.Context) ([]accounting.Account, error)
	Save(ctx context.Context, account *accounting.Account) error
	Update(ctx context.Context, account *accounting.Account) error
}

// FiscalYearRepository defines persistence for FiscalYear.
type FiscalYearRepository interface {
	FindCurrent(ctx context.Context) (*accounting.FiscalYear, error)
	FindByID(ctx context.Context, id uuid.UUID) (*accounting.FiscalYear, error)
	FindAll(ctx context.Context) ([]accounting.FiscalYear, error)
	Save(ctx context.Context, fy *accounting.FiscalYear) error
}

// AccountingPeriodRepository defines persistence for AccountingPeriod.
type AccountingPeriodRepository interface {
	// FindOpenForDate returns the OPEN period that contains the given date.
	// Returns nil if no open period covers the date — posting should be rejected.
	FindOpenForDate(ctx context.Context, date time.Time) (*accounting.AccountingPeriod, error)
	FindByID(ctx context.Context, id uuid.UUID) (*accounting.AccountingPeriod, error)
	FindByFiscalYear(ctx context.Context, fiscalYearID uuid.UUID) ([]accounting.AccountingPeriod, error)
	Close(ctx context.Context, periodID uuid.UUID, closedBy uuid.UUID) error
}

// JournalEntryRepository defines persistence for JournalEntry.
// Rule 14: Posted entries must not be silently edited or deleted.
type JournalEntryRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*accounting.JournalEntry, error)
	FindByNumber(ctx context.Context, number string) (*accounting.JournalEntry, error)
	FindByScope(ctx context.Context, scopeID uuid.UUID, limit, offset int) ([]accounting.JournalEntry, int64, error)
	FindByPeriod(ctx context.Context, periodID uuid.UUID) ([]accounting.JournalEntry, error)
	FindBySourceDocument(ctx context.Context, sourceType accounting.SourceType, sourceID uuid.UUID) (*accounting.JournalEntry, error)

	// Save creates a new DRAFT journal entry.
	Save(ctx context.Context, entry *accounting.JournalEntry) error

	// Post transitions a DRAFT entry to POSTED within a DB transaction.
	// Application must have validated balance BEFORE calling Post.
	// Rule 18: Use PostgreSQL transactions.
	Post(ctx context.Context, entryID uuid.UUID, postedBy uuid.UUID) error

	// Void marks a POSTED entry as VOIDED.
	// Rule 14: Does NOT delete. The original remains.
	Void(ctx context.Context, entryID uuid.UUID, voidedBy uuid.UUID) error

	// NextEntryNumber generates the next sequential entry number.
	// Format: JE-YYYY-NNNNNN
	NextEntryNumber(ctx context.Context, year int) (string, error)

	// GetAccountBalance returns the net balance of an account for a scope.
	// Always computed from posted journal lines — Rule 10.
	GetAccountBalance(ctx context.Context, accountID uuid.UUID, scopeID uuid.UUID) (map[string]interface{}, error)
}

// CostCenterRepository defines persistence for CostCenter.
type CostCenterRepository interface {
	FindByScope(ctx context.Context, scopeID uuid.UUID) ([]accounting.CostCenter, error)
	FindByID(ctx context.Context, id uuid.UUID) (*accounting.CostCenter, error)
	Save(ctx context.Context, cc *accounting.CostCenter) error
}
