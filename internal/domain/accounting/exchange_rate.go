package accounting

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ============================================================
// EXCHANGE RATE — Multi-Currency Support
// Rule 15: Exchange rate records MUST NOT be deleted — full audit history.
// Rule 12: Rate stored as NUMERIC(18,6) — no floating-point.
// Roadmap §7: Added in phase 7 (multi-currency extension).
// ============================================================

// ExchangeRate records a historical exchange rate between two currencies.
// The system base currency is always IQD (Iraqi Dinar).
// Example: FromCurrency=USD, ToCurrency=IQD, Rate=1310.000000
// Meaning: 1 USD = 1310 IQD
type ExchangeRate struct {
	ID            uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	FromCurrency  string          `gorm:"type:varchar(10);not null"`            // e.g. "USD"
	ToCurrency    string          `gorm:"type:varchar(10);not null;default:'IQD'"` // always "IQD" in this system
	Rate          decimal.Decimal `gorm:"type:numeric(18,6);not null"`           // how many ToCurrency = 1 FromCurrency
	EffectiveDate time.Time       `gorm:"type:date;not null"`                    // date this rate became active
	SetBy         uuid.UUID       `gorm:"type:uuid;not null"`                    // user who set it — Rule 21
	CreatedAt     time.Time
}

func (ExchangeRate) TableName() string { return "exchange_rates" }

// Convert converts an amount from FromCurrency to ToCurrency (IQD) using this rate.
// Example: Convert(100) where Rate=1310 → returns 131000 IQD
// Rule 12: Uses decimal arithmetic — no floating-point.
func (er ExchangeRate) Convert(amount decimal.Decimal) decimal.Decimal {
	return amount.Mul(er.Rate)
}

// ConvertBack converts an IQD amount back to the foreign currency.
// Example: ConvertBack(131000) where Rate=1310 → returns 100 USD
func (er ExchangeRate) ConvertBack(baseAmount decimal.Decimal) decimal.Decimal {
	if er.Rate.IsZero() {
		return decimal.Zero
	}
	return baseAmount.Div(er.Rate)
}
