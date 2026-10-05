package organization

import (
	"context"
	"time"

	"github.com/google/uuid"

	"m3aml-erp/internal/application/audit"
	orgdomain "m3aml-erp/internal/domain/organization"
	"m3aml-erp/internal/repositories"
)

type CompanySettingsService struct {
	repo         repositories.CompanySettingsRepository
	auditService *audit.AuditService
}

func NewCompanySettingsService(repo repositories.CompanySettingsRepository, auditService *audit.AuditService) *CompanySettingsService {
	return &CompanySettingsService{
		repo:         repo,
		auditService: auditService,
	}
}

func (s *CompanySettingsService) GetSettings(ctx context.Context) (*orgdomain.CompanySettings, error) {
	return s.repo.Get(ctx)
}

func (s *CompanySettingsService) UpdateSettings(ctx context.Context, userID uuid.UUID, companyName, phoneNumber, logoUrl string) (*orgdomain.CompanySettings, error) {
	settings, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}

	oldSettings := *settings

	settings.CompanyName = companyName
	settings.PhoneNumber = phoneNumber
	settings.LogoUrl = logoUrl
	settings.UpdatedAt = time.Now()
	settings.UpdatedBy = &userID

	if err := s.repo.Update(ctx, settings); err != nil {
		return nil, err
	}

	// Audit log
	_ = s.auditService.RecordAudit(ctx, audit.RecordAuditInput{
		Action:     "UPDATE",
		UserID:     userID,
		EntityType: "company_settings",
		EntityID:   &settings.ID,
		ScopeID:    nil, // Global
		OldValues: map[string]interface{}{
			"company_name": oldSettings.CompanyName,
			"phone_number": oldSettings.PhoneNumber,
			"logo_url":     oldSettings.LogoUrl,
		},
		NewValues: map[string]interface{}{
			"company_name": settings.CompanyName,
			"phone_number": settings.PhoneNumber,
			"logo_url":     settings.LogoUrl,
		},
	})

	return settings, nil
}
