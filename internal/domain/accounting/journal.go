package accounting

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ============================================================
// JOURNAL ENTRY — Roadmap §16
// Source of truth for ALL financial movements.
// Rule 10: Never use mutable balances as financial source of truth.
// Rule 11: Every financial movement must have a traceable transaction.
// Rule 13: Every accounting entry must be balanced.
// Rule 14: Posted entries MUST NOT be silently edited or deleted.
// ============================================================

// JournalEntryStatus represents the lifecycle of a journal entry.
// Roadmap §17 — ACCOUNTING LIFECYCLE
type JournalEntryStatus string

const (
	JournalStatusDraft    JournalEntryStatus = "DRAFT"
	JournalStatusPosted   JournalEntryStatus = "POSTED"
	JournalStatusVoided   JournalEntryStatus = "VOIDED"
	JournalStatusReversed JournalEntryStatus = "REVERSED"
)

// SourceType identifies which domain document created this journal entry.
// Enables source-document relationships (Roadmap §13 Phase 2).
type SourceType string

const (
	SourceTypeManual          SourceType = "MANUAL"
	SourceTypeOpeningBalance  SourceType = "OPENING_BALANCE"
	SourceTypeCashTransfer    SourceType = "CASH_TRANSFER"
	SourceTypeCashboxReceipt  SourceType = "CASHBOX_RECEIPT"
	SourceTypePurchaseRequest SourceType = "PURCHASE_REQUEST"
	SourceTypeExpense         SourceType = "EXPENSE"
	SourceTypeReversal        SourceType = "REVERSAL"
	SourceTypeAdjustment      SourceType = "ADJUSTMENT"
)

// JournalEntry is the header of a double-entry accounting record.
// It is IMMUTABLE once posted — corrections use REVERSAL or ADJUSTMENT.
type JournalEntry struct {
	ID             uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EntryNumber    string             `gorm:"type:varchar(50);not null;uniqueIndex"` // e.g. "JE-2026-000001"
	EntryDate      time.Time          `gorm:"type:date;not null"`
	Description    string             `gorm:"type:text;not null"`
	SourceType     SourceType         `gorm:"type:varchar(50);not null;default:'MANUAL'"`
	SourceID       *uuid.UUID         `gorm:"type:uuid;index"` // FK to originating document
	ScopeID        uuid.UUID          `gorm:"type:uuid;not null;index"` // which scope owns this entry
	PeriodID       uuid.UUID          `gorm:"type:uuid;not null;index"` // must be an OPEN period
	Status         JournalEntryStatus `gorm:"type:varchar(20);not null;default:'DRAFT'"`
	PostedBy       *uuid.UUID         `gorm:"type:uuid"`
	PostedAt       *time.Time
	ReversalOf     *uuid.UUID         `gorm:"type:uuid;index"` // if this is a reversal
	CreatedBy      uuid.UUID          `gorm:"type:uuid;not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time

	Lines []JournalLine `gorm:"foreignKey:JournalEntryID"`
}

func (JournalEntry) TableName() string { return "journal_entries" }

// ============================================================
// JOURNAL LINE — Roadmap §16
// Rules:
//   debit  >= 0
//   credit >= 0
//   NOT (debit > 0 AND credit > 0)
//   SUM(debit) = SUM(credit)  ← enforced at Post time in the service
// ============================================================

// JournalLine is a single debit or credit line in a journal entry.
// IMPORTANT: Debit and Credit are ALWAYS in IQD (base currency) — Rule 14.
// Foreign currency amounts are stored separately for reference and reporting only.
type JournalLine struct {
	ID             uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	JournalEntryID uuid.UUID        `gorm:"type:uuid;not null;index"`
	AccountID      uuid.UUID        `gorm:"type:uuid;not null;index"`
	Debit          decimal.Decimal  `gorm:"type:numeric(18,4);not null;default:0"`  // ALWAYS IQD — Rule 14
	Credit         decimal.Decimal  `gorm:"type:numeric(18,4);not null;default:0"`  // ALWAYS IQD — Rule 14
	Description    string           `gorm:"type:text"`
	ScopeID        uuid.UUID        `gorm:"type:uuid;not null;index"` // dimension: Roadmap §21
	CostCenterID   *uuid.UUID       `gorm:"type:uuid;index"`          // dimension: Roadmap §20
	Reference      string           `gorm:"type:varchar(100)"`
	CreatedAt      time.Time

	// --- Multi-Currency Reference Fields ---
	// These store the original foreign-currency amounts for reporting ONLY.
	// They do NOT affect accounting balance. Balance is always computed from Debit/Credit (IQD).
	ForeignCurrency *string          `gorm:"type:varchar(10)"` // e.g. "USD"; NULL = IQD-only transaction
	ForeignDebit    decimal.Decimal  `gorm:"type:numeric(18,4);not null;default:0"` // original debit in foreign currency
	ForeignCredit   decimal.Decimal  `gorm:"type:numeric(18,4);not null;default:0"` // original credit in foreign currency
	ExchangeRate    *decimal.Decimal `gorm:"type:numeric(18,6)"`                    // rate used at time of posting
}

func (JournalLine) TableName() string { return "journal_lines" }

// ============================================================
// DOMAIN VALIDATION — enforced BEFORE any DB write
// ============================================================

// ErrLineDebitCreditBoth is returned when a line has both debit and credit.
var ErrLineDebitCreditBoth = errors.New("a journal line cannot have both debit and credit")

// ErrLineNegative is returned when debit or credit is negative.
var ErrLineNegative = errors.New("debit and credit must be >= 0")

// ErrEntryNotBalanced is returned when SUM(debit) != SUM(credit).
// An unbalanced journal entry MUST NOT be posted — Roadmap §16.
var ErrEntryNotBalanced = errors.New("journal entry is not balanced: sum(debit) must equal sum(credit)")

// ErrNotPostable is returned when posting to a non-postable account.
var ErrNotPostable = errors.New("cannot post to a header/summary account")

// ErrPeriodClosed is returned when the accounting period is closed.
// Closed periods must prevent ordinary posting — Roadmap §18.
var ErrPeriodClosed = errors.New("accounting period is closed — cannot post")

// ErrEntryAlreadyPosted is returned on illegal edit of a posted entry.
// Rule 14: Posted entries must not be silently edited or deleted.
var ErrEntryAlreadyPosted = errors.New("posted journal entry cannot be modified — use reversal")

// Validate validates a single JournalLine before saving.
func (l *JournalLine) Validate() error {
	zero := decimal.Zero
	if l.Debit.LessThan(zero) || l.Credit.LessThan(zero) {
		return ErrLineNegative
	}
	if l.Debit.GreaterThan(zero) && l.Credit.GreaterThan(zero) {
		return ErrLineDebitCreditBoth
	}
	return nil
}

// ValidateBalance checks that SUM(debit) == SUM(credit) across all lines.
// Must be called before posting. Rule 13.
func ValidateBalance(lines []JournalLine) error {
	totalDebit  := decimal.Zero
	totalCredit := decimal.Zero
	for _, l := range lines {
		totalDebit  = totalDebit.Add(l.Debit)
		totalCredit = totalCredit.Add(l.Credit)
	}
	if !totalDebit.Equal(totalCredit) {
		return ErrEntryNotBalanced
	}
	return nil
}
