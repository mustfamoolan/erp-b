package repositories

import (
	"context"

	"github.com/google/uuid"
	"m3aml-erp/internal/domain/identity"
	"m3aml-erp/internal/domain/organization"
)

// ScopeRepository defines the persistence contract for OrganizationScope.
// Security enforcement is ALWAYS on the backend — Rule 8.
type ScopeRepository interface {
	FindAll(ctx context.Context) ([]organization.OrganizationScope, error)
	FindByID(ctx context.Context, id uuid.UUID) (*organization.OrganizationScope, error)
	FindByType(ctx context.Context, scopeType organization.ScopeType) ([]organization.OrganizationScope, error)
	Save(ctx context.Context, scope *organization.OrganizationScope) error
	Update(ctx context.Context, scope *organization.OrganizationScope) error
}

// AreaRepository defines the persistence contract for Area (Phase 1).
type AreaRepository interface {
	FindAll(ctx context.Context) ([]organization.Area, error)
	FindByID(ctx context.Context, id uuid.UUID) (*organization.Area, error)
	FindByCode(ctx context.Context, code string) (*organization.Area, error)
	Save(ctx context.Context, area *organization.Area) error
	Update(ctx context.Context, area *organization.Area) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// EmployeeRepository defines the persistence contract for Employee.
type EmployeeRepository interface {
	FindAll(ctx context.Context) ([]organization.Employee, error)
	FindByScope(ctx context.Context, scopeID uuid.UUID) ([]organization.Employee, error)
	FindByID(ctx context.Context, id uuid.UUID) (*organization.Employee, error)
	FindByEmployeeNo(ctx context.Context, no string) (*organization.Employee, error)
	Save(ctx context.Context, emp *organization.Employee) error
	Update(ctx context.Context, emp *organization.Employee) error
}

// UserRepository defines the persistence contract for User.
// Roadmap §8 — Users, Roles, Permissions, Scope access.
type UserRepository interface {
	// Lookups
	FindAll(ctx context.Context) ([]identity.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*identity.User, error)
	FindByUsername(ctx context.Context, username string) (*identity.User, error)
	FindByEmail(ctx context.Context, email string) (*identity.User, error)
	FindByEmployeeID(ctx context.Context, empID uuid.UUID) (*identity.User, error)

	// Persistence
	Save(ctx context.Context, user *identity.User) error
	Update(ctx context.Context, user *identity.User) error

	// Scope access — enforces Rule 7: Factory A cannot access Factory B.
	GetUserScopeAccess(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GrantScopeAccess(ctx context.Context, userID uuid.UUID, scopeID uuid.UUID, grantedBy uuid.UUID) error
	RevokeScopeAccess(ctx context.Context, userID uuid.UUID, scopeID uuid.UUID) error
	ClearUserScopes(ctx context.Context, userID uuid.UUID) error

	// Role assignment
	AssignRole(ctx context.Context, userID uuid.UUID, roleID uuid.UUID, scopeID *uuid.UUID) error
	RevokeRole(ctx context.Context, userID uuid.UUID, roleID uuid.UUID) error
	ClearUserRoles(ctx context.Context, userID uuid.UUID) error

	// Permission resolution — always resolved from DB, never from client.
	GetUserPermissions(ctx context.Context, userID uuid.UUID, scopeID uuid.UUID) ([]identity.PermissionCode, error)
}

// RoleRepository defines the persistence contract for Role.
type RoleRepository interface {
	FindAll(ctx context.Context) ([]identity.Role, error)
	FindByID(ctx context.Context, id uuid.UUID) (*identity.Role, error)
	FindByName(ctx context.Context, name identity.RoleName) (*identity.Role, error)
	Save(ctx context.Context, role *identity.Role) error
	Update(ctx context.Context, role *identity.Role) error
	AssignPermission(ctx context.Context, roleID uuid.UUID, permissionID uuid.UUID) error
	RemovePermission(ctx context.Context, roleID uuid.UUID, permissionID uuid.UUID) error
	ClearPermissions(ctx context.Context, roleID uuid.UUID) error
	FindAllPermissions(ctx context.Context) ([]identity.Permission, error)
	FindRolePermissions(ctx context.Context, roleID uuid.UUID) ([]identity.Permission, error)
}

// CompanySettingsRepository defines the persistence contract for CompanySettings.
type CompanySettingsRepository interface {
	Get(ctx context.Context) (*organization.CompanySettings, error)
	Update(ctx context.Context, settings *organization.CompanySettings) error
}
