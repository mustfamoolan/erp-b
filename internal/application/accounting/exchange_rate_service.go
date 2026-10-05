package accounting

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	appaudit "m3aml-erp/internal/application/audit"
	domainaccounting "m3aml-erp/internal/domain/accounting"
	domainaudit "m3aml-erp/internal/domain/audit"
	"m3aml-erp/internal/repositories"
)

// ============================================================
// ExchangeRateService — manages exchange rate data and currency conversion.
// Rule 15: Rates are never deleted — full history is always preserved.
// Rule 12: All amounts and rates use decimal.Decimal — no floating-point.
// Rule 20: Only SUPER_ADMIN may update exchange rates — enforced at API layer.
// ============================================================

// ErrNoExchangeRate is returned when no rate is found for a currency pair.
var ErrNoExchangeRate = errors.New("no exchange rate found for the specified currency pair and date")

// ErrInvalidRate is returned when a rate of zero or negative is provided.
var ErrInvalidRate = errors.New("exchange rate must be greater than zero")

// ExchangeRateService handles exchange rate lifecycle and currency conversion.
type ExchangeRateService struct {
	rateRepo repositories.ExchangeRateRepository
	auditSvc *appaudit.AuditService
}

func NewExchangeRateService(
	rateRepo repositories.ExchangeRateRepository,
	auditSvc *appaudit.AuditService,
) *ExchangeRateService {
	return &ExchangeRateService{
		rateRepo: rateRepo,
		auditSvc: auditSvc,
	}
}

// ============================================================
// SetRate — create or update an exchange rate for a date.
// Rule 21: Always records an audit trail.
// ============================================================

type SetRateInput struct {
	FromCurrency  string
	ToCurrency    string
	Rate          decimal.Decimal
	EffectiveDate time.Time
	SetBy         uuid.UUID
}

// SetRate saves a new exchange rate record.
// If a rate already exists for the same pair+date, it returns an error.
// To update a rate, callers must provide a new effective_date (or the same date — DB has a unique constraint).
func (s *ExchangeRateService) SetRate(ctx context.Context, input SetRateInput) (*domainaccounting.ExchangeRate, error) {
	if input.Rate.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidRate
	}

	rate := &domainaccounting.ExchangeRate{
		ID:            uuid.New(),
		FromCurrency:  input.FromCurrency,
		ToCurrency:    input.ToCurrency,
		Rate:          input.Rate,
		EffectiveDate: input.EffectiveDate,
		SetBy:         input.SetBy,
	}

	if err := s.rateRepo.Save(ctx, rate); err != nil {
		return nil, fmt.Errorf("SetRate: %w", err)
	}

	// §34: Audit SET_EXCHANGE_RATE — Rule 21
	if s.auditSvc != nil {
		setBy := input.SetBy
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     input.SetBy,
			ScopeID:    nil, // company-wide setting — no scope
			Action:     domainaudit.AuditCreate,
			EntityType: "exchange_rate",
			EntityID:   &setBy,
			NewValues: map[string]any{
				"from_currency":  input.FromCurrency,
				"to_currency":    input.ToCurrency,
				"rate":           input.Rate.String(),
				"effective_date": input.EffectiveDate.Format("2006-01-02"),
			},
		})
	}

	return rate, nil
}

// ============================================================
// GetCurrentRate — fetch the most recent rate for a pair.
// ============================================================

// GetCurrentRate returns the latest exchange rate for a currency pair.
// Returns ErrNoExchangeRate if no rate has been configured.
func (s *ExchangeRateService) GetCurrentRate(ctx context.Context, fromCurrency, toCurrency string) (*domainaccounting.ExchangeRate, error) {
	rate, err := s.rateRepo.FindLatest(ctx, fromCurrency, toCurrency)
	if err != nil {
		return nil, fmt.Errorf("GetCurrentRate: %w", err)
	}
	if rate == nil {
		return nil, fmt.Errorf("%w: %s → %s", ErrNoExchangeRate, fromCurrency, toCurrency)
	}
	return rate, nil
}

// ============================================================
// ConvertToBase — convert a foreign currency amount to IQD.
// Used by FinanceService and AccountingService when building journal entries.
// ============================================================

// ConvertToBase converts amount in fromCurrency to IQD using the current exchange rate.
// Returns: (baseAmountIQD, rateUsed, error).
// Rule 12: Uses decimal.Decimal — no floating-point.
func (s *ExchangeRateService) ConvertToBase(ctx context.Context, amount decimal.Decimal, fromCurrency string) (decimal.Decimal, decimal.Decimal, error) {
	// IQD is already the base — no conversion needed
	if fromCurrency == "IQD" || fromCurrency == "" {
		return amount, decimal.NewFromInt(1), nil
	}

	rate, err := s.GetCurrentRate(ctx, fromCurrency, "IQD")
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("ConvertToBase: %w", err)
	}

	base := rate.Convert(amount) // amount × rate
	return base, rate.Rate, nil
}

// ============================================================
// ListHistory — fetch historical rates for a currency pair.
// ============================================================

// ListHistory returns all historical exchange rates for a pair, newest first.
func (s *ExchangeRateService) ListHistory(ctx context.Context, fromCurrency, toCurrency string) ([]*domainaccounting.ExchangeRate, error) {
	rates, err := s.rateRepo.ListHistory(ctx, fromCurrency, toCurrency)
	if err != nil {
		return nil, fmt.Errorf("ListHistory: %w", err)
	}
	return rates, nil
}
