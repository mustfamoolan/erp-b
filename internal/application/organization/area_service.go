package organization

import (
	"context"
	"errors"
	"strings"
	"time"

	"m3aml-erp/bootstrap"
	"m3aml-erp/internal/domain/organization"
	"m3aml-erp/internal/repositories"

	"github.com/google/uuid"
)

type AreaService interface {
	CreateArea(ctx context.Context, name, code, governorate, description string) (*organization.Area, error)
	UpdateArea(ctx context.Context, id uuid.UUID, name, code, governorate, description string, status organization.AreaStatus) (*organization.Area, error)
	GetArea(ctx context.Context, id uuid.UUID) (*organization.Area, error)
	ListAreas(ctx context.Context) ([]organization.Area, error)
	DeleteArea(ctx context.Context, id uuid.UUID) error
}

type areaService struct {
	repo repositories.AreaRepository
}

func NewAreaService(repo repositories.AreaRepository) AreaService {
	return &areaService{repo: repo}
}

func (s *areaService) CreateArea(ctx context.Context, name, code, governorate, description string) (*organization.Area, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}
	if code == "" {
		code = "AREA-" + strings.ToUpper(uuid.New().String()[:6])
	}
	if governorate == "" {
		governorate = "بغداد"
	}

	existing, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("area with this code already exists")
	}

	area := &organization.Area{
		Name:        name,
		Code:        code,
		Governorate: governorate,
		Description: description,
		Status:      organization.AreaStatusActive,
	}

	if err := s.repo.Save(ctx, area); err != nil {
		return nil, err
	}

	bootstrap.InvalidateOrgCache()

	return area, nil
}

func (s *areaService) UpdateArea(ctx context.Context, id uuid.UUID, name, code, governorate, description string, status organization.AreaStatus) (*organization.Area, error) {
	area, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if area == nil {
		return nil, errors.New("area not found")
	}

	if code != area.Code {
		existing, err := s.repo.FindByCode(ctx, code)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != area.ID {
			return nil, errors.New("area with this code already exists")
		}
	}

	area.Name = name
	area.Code = code
	if governorate != "" {
		area.Governorate = governorate
	}
	area.Description = description
	if status != "" {
		area.Status = status
	}

	if err := s.repo.Update(ctx, area); err != nil {
		return nil, err
	}

	bootstrap.InvalidateOrgCache()

	return area, nil
}

func (s *areaService) GetArea(ctx context.Context, id uuid.UUID) (*organization.Area, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *areaService) ListAreas(ctx context.Context) ([]organization.Area, error) {
	return bootstrap.CacheRemember("org:areas:all", 12*time.Hour, func() ([]organization.Area, error) {
		return s.repo.FindAll(ctx)
	})
}

func (s *areaService) DeleteArea(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err == nil {
		bootstrap.InvalidateOrgCache()
	}
	return err
}
