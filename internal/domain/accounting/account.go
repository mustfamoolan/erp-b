package accounting

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================
// ACCOUNT TYPES — Roadmap §15
// The accounting engine must understand the normal debit/credit
// nature of each type.
// ============================================================

// AccountType defines the fundamental accounting classification.
type AccountType string

const (
	AccountTypeAsset     AccountType = "ASSET"
	AccountTypeLiability AccountType = "LIABILITY"
	AccountTypeEquity    AccountType = "EQUITY"
	AccountTypeRevenue   AccountType = "REVENUE"
	AccountTypeExpense   AccountType = "EXPENSE"
)

// NormalBalance returns whether the account naturally increases on Debit or Credit.
func (t AccountType) NormalBalance() string {
	switch t {
	case AccountTypeAsset, AccountTypeExpense:
		return "DEBIT"
	default:
		return "CREDIT"
	}
}

// ============================================================
// ACCOUNT — Roadmap §14
// Hierarchical Chart of Accounts. Data-driven hierarchy.
// Do NOT hard-code account IDs inside business logic.
// ============================================================

// Account represents a single account in the Chart of Accounts.
type Account struct {
	ID          uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code        string      `gorm:"type:varchar(20);not null;uniqueIndex"` // e.g. "1101"
	Name        string      `gorm:"type:varchar(200);not null"`
	Type        AccountType `gorm:"type:varchar(20);not null"`
	ParentID    *uuid.UUID  `gorm:"type:uuid;index"`              // nil = root account
	IsPostable  bool        `gorm:"not null;default:true"`        // false = header/summary only
	ScopeID     *uuid.UUID  `gorm:"type:uuid;index"`              // nil = company-wide account
	Description string      `gorm:"type:text"`
	IsActive    bool        `gorm:"not null;default:true"`
	SortOrder   int         `gorm:"not null;default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Account) TableName() string { return "accounts" }

// ============================================================
// COST CENTER — Roadmap §20
// Cost centers are NOT the same as accounts.
// Designed for future: Factory A, Factory B, Production Line 1...
// ============================================================

// CostCenter represents a cost center for dimensional accounting.
type CostCenter struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ScopeID  uuid.UUID `gorm:"type:uuid;not null;index"`
	Code     string    `gorm:"type:varchar(50);not null;uniqueIndex"`
	Name     string    `gorm:"type:varchar(150);not null"`
	IsActive bool      `gorm:"not null;default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (CostCenter) TableName() string { return "cost_centers" }

// ============================================================
// FISCAL YEAR — Roadmap §18
// ============================================================

// FiscalYearStatus represents the lifecycle of a fiscal year.
type FiscalYearStatus string

const (
	FiscalYearStatusOpen   FiscalYearStatus = "OPEN"
	FiscalYearStatusClosed FiscalYearStatus = "CLOSED"
)

// FiscalYear defines the company's accounting year.
type FiscalYear struct {
	ID        uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string           `gorm:"type:varchar(50);not null;uniqueIndex"` // e.g. "FY2026"
	StartDate time.Time        `gorm:"type:date;not null"`
	EndDate   time.Time        `gorm:"type:date;not null"`
	Status    FiscalYearStatus `gorm:"type:varchar(10);not null;default:'OPEN'"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Periods []AccountingPeriod `gorm:"foreignKey:FiscalYearID"`
}

func (FiscalYear) TableName() string { return "fiscal_years" }

// ============================================================
// ACCOUNTING PERIOD — Roadmap §18
// Period states: OPEN, CLOSED
// Closed periods must prevent ordinary posting.
// ============================================================

// AccountingPeriodStatus represents the state of a period.
type AccountingPeriodStatus string

const (
	AccountingPeriodOpen   AccountingPeriodStatus = "OPEN"
	AccountingPeriodClosed AccountingPeriodStatus = "CLOSED"
)

// AccountingPeriod represents a month within a fiscal year.
type AccountingPeriod struct {
	ID           uuid.UUID              `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	FiscalYearID uuid.UUID              `gorm:"type:uuid;not null;index"`
	Name         string                 `gorm:"type:varchar(50);not null"`  // e.g. "September 2026"
	PeriodNumber int                    `gorm:"not null"`                   // 1-12
	StartDate    time.Time              `gorm:"type:date;not null"`
	EndDate      time.Time              `gorm:"type:date;not null"`
	Status       AccountingPeriodStatus `gorm:"type:varchar(10);not null;default:'OPEN'"`
	ClosedBy     *uuid.UUID             `gorm:"type:uuid"`
	ClosedAt     *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (AccountingPeriod) TableName() string { return "accounting_periods" }
