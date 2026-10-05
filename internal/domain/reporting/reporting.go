package reporting

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DateRange defines a common filter for reports
type DateRange struct {
	From *time.Time `json:"from,omitempty"`
	To   *time.Time `json:"to,omitempty"`
}

// ─── Accounting Reports ────────────────────────────────────────────────────────

type AccountStatementLine struct {
	Date          time.Time       `json:"date"`
	JournalID     uuid.UUID       `json:"journal_id"`
	EntryNumber   string          `json:"entry_number"`
	Description   string          `json:"description"`
	Debit         decimal.Decimal `json:"debit"`
	Credit        decimal.Decimal `json:"credit"`
	Balance       decimal.Decimal `json:"balance"`       // Running balance
	Currency      string          `json:"currency"`
	ForeignDebit  decimal.Decimal `json:"foreign_debit"`
	ForeignCredit decimal.Decimal `json:"foreign_credit"`
}

type AccountStatementReport struct {
	AccountID       uuid.UUID              `json:"account_id"`
	AccountCode     string                 `json:"account_code"`
	AccountName     string                 `json:"account_name"`
	OpeningBalance  decimal.Decimal        `json:"opening_balance"`
	ClosingBalance  decimal.Decimal        `json:"closing_balance"`
	TotalDebit      decimal.Decimal        `json:"total_debit"`
	TotalCredit     decimal.Decimal        `json:"total_credit"`
	Lines           []AccountStatementLine `json:"lines"`
}

type TrialBalanceLine struct {
	AccountID   uuid.UUID       `json:"account_id"`
	AccountCode string          `json:"account_code"`
	AccountName string          `json:"account_name"`
	Debit       decimal.Decimal `json:"debit"`
	Credit      decimal.Decimal `json:"credit"`
	Balance     decimal.Decimal `json:"balance"` // Positive = Debit, Negative = Credit
}

type TrialBalanceReport struct {
	TotalDebit  decimal.Decimal    `json:"total_debit"`
	TotalCredit decimal.Decimal    `json:"total_credit"`
	Lines       []TrialBalanceLine `json:"lines"`
}

// ─── Inventory Reports ─────────────────────────────────────────────────────────

type StockBalanceLine struct {
	WarehouseID   uuid.UUID       `json:"warehouse_id"`
	WarehouseName string          `json:"warehouse_name"`
	VariantID     uuid.UUID       `json:"variant_id"`
	SKU           string          `json:"sku"`
	MaterialName  string          `json:"material_name"`
	Quantity      decimal.Decimal `json:"quantity"`
}

type StockBalanceReport struct {
	Lines []StockBalanceLine `json:"lines"`
}

// ─── Cashbox Reports ───────────────────────────────────────────────────────────

type CashboxStatementReport struct {
	CashboxID      uuid.UUID              `json:"cashbox_id"`
	CashboxName    string                 `json:"cashbox_name"`
	ScopeID        uuid.UUID              `json:"scope_id"`
	AccountID      uuid.UUID              `json:"account_id"`
	OpeningBalance decimal.Decimal        `json:"opening_balance"`
	ClosingBalance decimal.Decimal        `json:"closing_balance"`
	Lines          []AccountStatementLine `json:"lines"` // Reuses AccountStatementLine
}
