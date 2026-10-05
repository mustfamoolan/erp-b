package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"m3aml-erp/internal/domain/accounting"
)

type fiscalYearRepository struct{ db *gorm.DB }

func NewFiscalYearRepository(db *gorm.DB) *fiscalYearRepository {
	return &fiscalYearRepository{db: db}
}

func (r *fiscalYearRepository) FindCurrent(ctx context.Context) (*accounting.FiscalYear, error) {
	var fy accounting.FiscalYear
	err := r.db.WithContext(ctx).
		Where("status = 'OPEN'").
		Order("start_date DESC").
		First(&fy).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &fy, err
}

func (r *fiscalYearRepository) FindByID(ctx context.Context, id uuid.UUID) (*accounting.FiscalYear, error) {
	var fy accounting.FiscalYear
	err := r.db.WithContext(ctx).Preload("Periods").First(&fy, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &fy, err
}

func (r *fiscalYearRepository) FindAll(ctx context.Context) ([]accounting.FiscalYear, error) {
	var fys []accounting.FiscalYear
	return fys, r.db.WithContext(ctx).Order("start_date DESC").Find(&fys).Error
}

func (r *fiscalYearRepository) Save(ctx context.Context, fy *accounting.FiscalYear) error {
	return r.db.WithContext(ctx).Create(fy).Error
}
