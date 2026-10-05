package postgres

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	domainaccounting "m3aml-erp/internal/domain/accounting"
)

// ============================================================
// ExchangeRateRepository — PostgreSQL implementation.
// Rule 15: DELETE operations are intentionally absent — rates are immutable history.
// Rule 12: NUMERIC precision enforced at DB level (see migration 000007).
// ============================================================

type exchangeRateRepository struct {
	db *gorm.DB
}

// NewExchangeRateRepository creates a new PostgreSQL-backed ExchangeRateRepository.
func NewExchangeRateRepository(db *gorm.DB) *exchangeRateRepository {
	return &exchangeRateRepository{db: db}
}

// Save persists a new exchange rate. Returns error if pair+date already exists.
func (r *exchangeRateRepository) Save(ctx context.Context, rate *domainaccounting.ExchangeRate) error {
	if err := r.db.WithContext(ctx).Create(rate).Error; err != nil {
		return fmt.Errorf("exchange_rate: save failed: %w", err)
	}
	return nil
}

// FindLatest returns the most recent exchange rate for a currency pair.
// Uses effective_date DESC ordering — always returns the latest active rate.
func (r *exchangeRateRepository) FindLatest(ctx context.Context, fromCurrency, toCurrency string) (*domainaccounting.ExchangeRate, error) {
	var rate domainaccounting.ExchangeRate
	err := r.db.WithContext(ctx).
		Where("from_currency = ? AND to_currency = ?", fromCurrency, toCurrency).
		Order("effective_date DESC, created_at DESC").
		First(&rate).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("exchange_rate: find latest failed: %w", err)
	}
	return &rate, nil
}

// FindForDate returns the rate effective on or before the given date.
// Used for historical transaction reconstruction — Rule 15.
func (r *exchangeRateRepository) FindForDate(ctx context.Context, fromCurrency, toCurrency string, date time.Time) (*domainaccounting.ExchangeRate, error) {
	var rate domainaccounting.ExchangeRate
	err := r.db.WithContext(ctx).
		Where("from_currency = ? AND to_currency = ? AND effective_date <= ?", fromCurrency, toCurrency, date.Format("2006-01-02")).
		Order("effective_date DESC, created_at DESC").
		First(&rate).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("exchange_rate: find for date failed: %w", err)
	}
	return &rate, nil
}

// ListHistory returns all historical rates for a currency pair, newest first.
func (r *exchangeRateRepository) ListHistory(ctx context.Context, fromCurrency, toCurrency string) ([]*domainaccounting.ExchangeRate, error) {
	var rates []*domainaccounting.ExchangeRate
	err := r.db.WithContext(ctx).
		Where("from_currency = ? AND to_currency = ?", fromCurrency, toCurrency).
		Order("effective_date DESC, created_at DESC").
		Find(&rates).Error
	if err != nil {
		return nil, fmt.Errorf("exchange_rate: list history failed: %w", err)
	}
	return rates, nil
}
