package cashbox

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ============================================================
// CASHBOX — Roadmap §23
// Each cashbox is linked to an accounting account.
// Balance is NEVER stored directly — computed from transactions.
// Rule 10: Never use mutable balances as financial source of truth.
// ============================================================

// CashboxStatus defines the state of a cashbox.
type CashboxStatus string

const (
	CashboxStatusActive   CashboxStatus = "ACTIVE"
	CashboxStatusInactive CashboxStatus = "INACTIVE"
)

// Cashbox represents a physical or virtual cash holding.
// Always linked to an accounting Account — cash is tracked via the ledger.
// Currency default changed to IQD (Iraqi Dinar) — Multi-Currency extension.
type Cashbox struct {
	ID                   uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ScopeID              uuid.UUID     `gorm:"type:uuid;not null;index" json:"scope_id"` // which scope owns this cashbox
	Name                 string        `gorm:"type:varchar(150);not null" json:"name"`
	AccountID            uuid.UUID     `gorm:"type:uuid;not null;index" json:"account_id"` // linked to Chart of Accounts (e.g. 1101)
	Currency             string        `gorm:"type:varchar(10);not null;default:'IQD'" json:"currency"` // IQD (default) or USD
	TargetOpeningBalance decimal.Decimal `gorm:"type:numeric(18,4);not null;default:0" json:"target_opening_balance"`   // Roadmap §9: Target Opening Balance
	Status               CashboxStatus `gorm:"type:varchar(20);not null;default:'ACTIVE'" json:"status"`
	ImprestBalance       decimal.Decimal `gorm:"type:numeric(19,4);not null;default:0" json:"imprest_balance"`
	CurrentBalance       decimal.Decimal `gorm:"->;column:current_balance" json:"current_balance"` // populated from view
	CreatedAt            time.Time     `json:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at"`
}

func (Cashbox) TableName() string { return "cashboxes" }

// ============================================================
// CASH TRANSACTION — Roadmap §24
// Every movement must reference a cashbox, amount, direction,
// source document, accounting entry, user, date, status.
// Rule 11: Every financial movement must have a traceable transaction.
// Never directly modify cashbox balances — Rule 10.
// ============================================================

// CashTransactionDirection indicates money flow.
type CashTransactionDirection string

const (
	CashDirectionIn  CashTransactionDirection = "IN"
	CashDirectionOut CashTransactionDirection = "OUT"
)

// CashTransactionStatus tracks the lifecycle.
type CashTransactionStatus string

const (
	CashTransactionPending   CashTransactionStatus = "PENDING"
	CashTransactionCompleted CashTransactionStatus = "COMPLETED"
	CashTransactionCancelled CashTransactionStatus = "CANCELLED"
)

// CashTransaction records every cash movement. IMMUTABLE once completed.
// Multi-Currency: amount is in the cashbox's own currency (IQD or USD).
// BaseAmount is always in IQD for consolidated reporting — Rule 12.
type CashTransaction struct {
	ID               uuid.UUID                `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CashboxID        uuid.UUID                `gorm:"type:uuid;not null;index"`
	Amount           decimal.Decimal          `gorm:"type:numeric(18,4);not null"`          // in cashbox currency (IQD or USD)
	Currency         string                   `gorm:"type:varchar(10);not null;default:'IQD'"` // IQD default
	BaseAmount       *decimal.Decimal         `gorm:"type:numeric(18,4)"` // always IQD; NULL if currency=IQD (same)
	ExchangeRate     *decimal.Decimal         `gorm:"type:numeric(18,6)"` // rate used; NULL if IQD transaction
	Direction        CashTransactionDirection `gorm:"type:varchar(10);not null"`
	SourceType       string                   `gorm:"type:varchar(50);not null"` // TRANSFER, REQUEST_PAYMENT, EXPENSE...
	SourceID         *uuid.UUID               `gorm:"type:uuid;index"`
	JournalEntryID   *uuid.UUID               `gorm:"type:uuid;index"` // accounting entry — Rule 11
	Description      string                   `gorm:"type:text"`
	PerformedBy      uuid.UUID                `gorm:"type:uuid;not null"`
	TransactionDate  time.Time                `gorm:"type:date;not null"`
	Status           CashTransactionStatus    `gorm:"type:varchar(20);not null;default:'COMPLETED'"`
	CreatedAt        time.Time
}

func (CashTransaction) TableName() string { return "cash_transactions" }

// ============================================================
// CASH TRANSFER — Roadmap §24 (Cross-scope operations)
// Transfer from Main Cashbox → Factory Cashbox must be atomic.
// Rule 18: PostgreSQL transactions.
// Roadmap §7: Cross-scope operations must be modeled explicitly.
// ============================================================

// CashTransferStatus tracks the transfer lifecycle.
type CashTransferStatus string

const (
	CashTransferPending   CashTransferStatus = "PENDING"
	CashTransferCompleted CashTransferStatus = "COMPLETED"
	CashTransferCancelled CashTransferStatus = "CANCELLED"
)

// CashTransfer models an explicit transfer between two cashboxes.
// Both transactions and the accounting entry are created atomically.
// Multi-Currency: BaseAmount is always IQD for accounting purposes — Rule 12.
type CashTransfer struct {
	ID                 uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SourceCashboxID    uuid.UUID          `gorm:"type:uuid;not null;index"`
	DestCashboxID      uuid.UUID          `gorm:"type:uuid;not null;index"`
	SourceScopeID      uuid.UUID          `gorm:"type:uuid;not null"` // cross-scope tracking — Roadmap §7
	DestScopeID        uuid.UUID          `gorm:"type:uuid;not null"`
	Amount             decimal.Decimal    `gorm:"type:numeric(18,4);not null"`          // in source cashbox currency
	Currency           string             `gorm:"type:varchar(10);not null;default:'IQD'"` // IQD default
	BaseAmount         *decimal.Decimal   `gorm:"type:numeric(18,4)"` // always IQD; NULL if IQD transfer
	SourceExchangeRate *decimal.Decimal   `gorm:"type:numeric(18,6)"` // rate for source cashbox if foreign
	DestExchangeRate   *decimal.Decimal   `gorm:"type:numeric(18,6)"` // rate for dest cashbox if foreign
	ReferenceDocument  string             `gorm:"type:varchar(100)"` // e.g. purchase request number
	JournalEntryID     *uuid.UUID         `gorm:"type:uuid;index"`
	PerformedBy        uuid.UUID          `gorm:"type:uuid;not null"`
	TransferDate       time.Time          `gorm:"type:date;not null"`
	Status             CashTransferStatus `gorm:"type:varchar(20);not null;default:'COMPLETED'"`
	Notes              string             `gorm:"type:text"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (CashTransfer) TableName() string { return "cash_transfers" }

// ============================================================
// VALIDATION ERRORS
// ============================================================

var (
	ErrAmountNegative  = errors.New("cash transaction amount must be greater than zero")
	ErrCashboxInactive = errors.New("cannot perform transaction on an inactive cashbox")
)
