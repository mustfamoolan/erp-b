package repositories

import (
	"context"
	"time"

	domainaccounting "m3aml-erp/internal/domain/accounting"
)

// ============================================================
// ExchangeRateRepository — interface for exchange rate persistence.
// Rule 15: Records must NEVER be deleted — full audit history is mandatory.
// Rule 12: Rate precision is NUMERIC(18,6) — enforced at DB level.
// ============================================================

// ExchangeRateRepository defines the persistence contract for exchange rates.
type ExchangeRateRepository interface {
	// Save persists a new exchange rate record.
	// Returns error if a record for the same pair+date already exists.
	Save(ctx context.Context, rate *domainaccounting.ExchangeRate) error

	// FindLatest returns the most recent exchange rate for a currency pair.
	// Returns nil, nil if no rate has been set yet.
	FindLatest(ctx context.Context, fromCurrency, toCurrency string) (*domainaccounting.ExchangeRate, error)

	// FindForDate returns the exchange rate effective on or before the given date.
	// Used for historical transaction reconstruction.
	FindForDate(ctx context.Context, fromCurrency, toCurrency string, date time.Time) (*domainaccounting.ExchangeRate, error)

	// ListHistory returns all historical rates for a currency pair, ordered by date DESC.
	ListHistory(ctx context.Context, fromCurrency, toCurrency string) ([]*domainaccounting.ExchangeRate, error)
}
