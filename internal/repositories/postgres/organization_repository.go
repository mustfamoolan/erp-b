package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"m3aml-erp/internal/domain/organization"
)

// scopeRepository is the PostgreSQL implementation of ScopeRepository.
type scopeRepository struct {
	db *gorm.DB
}

func NewScopeRepository(db *gorm.DB) *scopeRepository {
	return &scopeRepository{db: db}
}

func (r *scopeRepository) FindAll(ctx context.Context) ([]organization.OrganizationScope, error) {
	var scopes []organization.OrganizationScope
	return scopes, r.db.WithContext(ctx).Order("type, name").Find(&scopes).Error
}

func (r *scopeRepository) FindByID(ctx context.Context, id uuid.UUID) (*organization.OrganizationScope, error) {
	var scope organization.OrganizationScope
	err := r.db.WithContext(ctx).First(&scope, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &scope, err
}

func (r *scopeRepository) FindByType(ctx context.Context, scopeType organization.ScopeType) ([]organization.OrganizationScope, error) {
	var scopes []organization.OrganizationScope
	return scopes, r.db.WithContext(ctx).Where("type = ? AND status = 'ACTIVE'", scopeType).Find(&scopes).Error
}

func (r *scopeRepository) Save(ctx context.Context, scope *organization.OrganizationScope) error {
	return r.db.WithContext(ctx).Create(scope).Error
}

func (r *scopeRepository) Update(ctx context.Context, scope *organization.OrganizationScope) error {
	return r.db.WithContext(ctx).Save(scope).Error
}

// employeeRepository is the PostgreSQL implementation of EmployeeRepository.
type employeeRepository struct {
	db *gorm.DB
}

func NewEmployeeRepository(db *gorm.DB) *employeeRepository {
	return &employeeRepository{db: db}
}

func (r *employeeRepository) FindAll(ctx context.Context) ([]organization.Employee, error) {
	var employees []organization.Employee
	return employees, r.db.WithContext(ctx).Order("full_name").Find(&employees).Error
}

func (r *employeeRepository) FindByScope(ctx context.Context, scopeID uuid.UUID) ([]organization.Employee, error) {
	var employees []organization.Employee
	return employees, r.db.WithContext(ctx).
		Where("scope_id = ?", scopeID).
		Order("full_name").
		Find(&employees).Error
}

func (r *employeeRepository) FindByID(ctx context.Context, id uuid.UUID) (*organization.Employee, error) {
	var emp organization.Employee
	err := r.db.WithContext(ctx).First(&emp, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &emp, err
}

func (r *employeeRepository) FindByEmployeeNo(ctx context.Context, no string) (*organization.Employee, error) {
	var emp organization.Employee
	err := r.db.WithContext(ctx).First(&emp, "employee_no = ?", no).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &emp, err
}

func (r *employeeRepository) Save(ctx context.Context, emp *organization.Employee) error {
	return r.db.WithContext(ctx).Create(emp).Error
}

func (r *employeeRepository) Update(ctx context.Context, emp *organization.Employee) error {
	return r.db.WithContext(ctx).Save(emp).Error
}
