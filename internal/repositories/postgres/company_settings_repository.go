package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"m3aml-erp/internal/domain/organization"
	"m3aml-erp/internal/repositories"
)

type companySettingsRepository struct {
	db *gorm.DB
}

func NewCompanySettingsRepository(db *gorm.DB) repositories.CompanySettingsRepository {
	return &companySettingsRepository{db: db}
}

func (r *companySettingsRepository) Get(ctx context.Context) (*organization.CompanySettings, error) {
	var settings organization.CompanySettings
	// There is only one row with this exact ID
	id := uuid.MustParse("00000000-0000-0000-0000-000000000000")
	if err := r.db.WithContext(ctx).First(&settings, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Fallback: try getting any existing settings row
			if errFallback := r.db.WithContext(ctx).Order("updated_at DESC").First(&settings).Error; errFallback == nil {
				return &settings, nil
			}
			return nil, errors.New("company settings not found")
		}
		return nil, err
	}
	return &settings, nil
}

func (r *companySettingsRepository) Update(ctx context.Context, settings *organization.CompanySettings) error {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000000")
	settings.ID = id

	res := r.db.WithContext(ctx).Model(&organization.CompanySettings{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"company_name": settings.CompanyName,
			"phone_number": settings.PhoneNumber,
			"logo_url":     settings.LogoUrl,
			"updated_at":   settings.UpdatedAt,
			"updated_by":   settings.UpdatedBy,
		})

	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return r.db.WithContext(ctx).Exec(
			`INSERT INTO company_settings (id, company_name, phone_number, logo_url, updated_at, updated_by)
			 VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT (id) DO UPDATE SET
			 company_name = EXCLUDED.company_name,
			 phone_number = EXCLUDED.phone_number,
			 logo_url = EXCLUDED.logo_url,
			 updated_at = EXCLUDED.updated_at,
			 updated_by = EXCLUDED.updated_by`,
			id, settings.CompanyName, settings.PhoneNumber, settings.LogoUrl, settings.UpdatedAt, settings.UpdatedBy,
		).Error
	}

	return nil
}
