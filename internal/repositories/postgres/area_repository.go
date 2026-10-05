package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"m3aml-erp/internal/domain/organization"
	"m3aml-erp/internal/repositories"
)

type areaRepository struct {
	db *gorm.DB
}

// NewAreaRepository creates a new instance of AreaRepository.
func NewAreaRepository(db *gorm.DB) repositories.AreaRepository {
	return &areaRepository{db: db}
}

func (r *areaRepository) FindAll(ctx context.Context) ([]organization.Area, error) {
	var areas []organization.Area
	result := r.db.WithContext(ctx).Find(&areas)
	return areas, result.Error
}

func (r *areaRepository) FindByID(ctx context.Context, id uuid.UUID) (*organization.Area, error) {
	var area organization.Area
	result := r.db.WithContext(ctx).First(&area, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &area, nil
}

func (r *areaRepository) FindByCode(ctx context.Context, code string) (*organization.Area, error) {
	var area organization.Area
	result := r.db.WithContext(ctx).First(&area, "code = ?", code)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &area, nil
}

func (r *areaRepository) Save(ctx context.Context, area *organization.Area) error {
	return r.db.WithContext(ctx).Create(area).Error
}

func (r *areaRepository) Update(ctx context.Context, area *organization.Area) error {
	return r.db.WithContext(ctx).Save(area).Error
}

func (r *areaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&organization.Area{}, "id = ?", id).Error
}
