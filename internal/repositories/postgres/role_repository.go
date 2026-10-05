package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"m3aml-erp/internal/domain/identity"
)

type roleRepository struct{ db *gorm.DB }

func NewRoleRepository(db *gorm.DB) *roleRepository { return &roleRepository{db: db} }

func (r *roleRepository) FindAll(ctx context.Context) ([]identity.Role, error) {
	var roles []identity.Role
	return roles, r.db.WithContext(ctx).Order("name").Find(&roles).Error
}

func (r *roleRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.Role, error) {
	var role identity.Role
	err := r.db.WithContext(ctx).First(&role, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &role, err
}

func (r *roleRepository) FindByName(ctx context.Context, name identity.RoleName) (*identity.Role, error) {
	var role identity.Role
	err := r.db.WithContext(ctx).First(&role, "name = ?", name).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &role, err
}

func (r *roleRepository) Save(ctx context.Context, role *identity.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *roleRepository) AssignPermission(ctx context.Context, roleID uuid.UUID, permissionID uuid.UUID) error {
	rp := identity.RolePermission{
		ID:           uuid.New(),
		RoleID:       roleID,
		PermissionID: permissionID,
	}
	return r.db.WithContext(ctx).Create(&rp).Error
}

func (r *roleRepository) RemovePermission(ctx context.Context, roleID uuid.UUID, permissionID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Delete(&identity.RolePermission{}).Error
}

func (r *roleRepository) ClearPermissions(ctx context.Context, roleID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("role_id = ?", roleID).
		Delete(&identity.RolePermission{}).Error
}

func (r *roleRepository) Update(ctx context.Context, role *identity.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *roleRepository) FindAllPermissions(ctx context.Context) ([]identity.Permission, error) {
	var permissions []identity.Permission
	return permissions, r.db.WithContext(ctx).Order("grp, display_name").Find(&permissions).Error
}

func (r *roleRepository) FindRolePermissions(ctx context.Context, roleID uuid.UUID) ([]identity.Permission, error) {
	var permissions []identity.Permission
	err := r.db.WithContext(ctx).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Find(&permissions).Error
	return permissions, err
}

// ────────────────────────────────────────────────────────────
// UserScopeAccess Repository extension on userRepository
// ────────────────────────────────────────────────────────────

// GrantScopeAccess grants a user explicit access to a scope — Rule 7.
func (r *userRepository) GrantScopeAccess(ctx context.Context, userID uuid.UUID, scopeID uuid.UUID, grantedBy uuid.UUID) error {
	access := identity.UserScopeAccess{
		ID:        uuid.New(),
		UserID:    userID,
		ScopeID:   scopeID,
		GrantedAt: time.Now(),
		GrantedBy: grantedBy,
	}
	return r.db.WithContext(ctx).Create(&access).Error
}

// RevokeScopeAccess removes a user's access to a scope.
func (r *userRepository) RevokeScopeAccess(ctx context.Context, userID uuid.UUID, scopeID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND scope_id = ?", userID, scopeID).
		Delete(&identity.UserScopeAccess{}).Error
}

func (r *userRepository) ClearUserScopes(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&identity.UserScopeAccess{}).Error
}

func (r *userRepository) ClearUserRoles(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&identity.UserRole{}).Error
}

// AssignRole assigns a role to a user, optionally scoped.
func (r *userRepository) AssignRole(ctx context.Context, userID uuid.UUID, roleID uuid.UUID, scopeID *uuid.UUID) error {
	userRole := identity.UserRole{
		ID:      uuid.New(),
		UserID:  userID,
		RoleID:  roleID,
		ScopeID: scopeID,
	}
	return r.db.WithContext(ctx).Create(&userRole).Error
}

// RevokeRole removes a role from a user.
func (r *userRepository) RevokeRole(ctx context.Context, userID uuid.UUID, roleID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&identity.UserRole{}).Error
}

// FindAll returns all users ordered by full_name.
func (r *userRepository) FindAll(ctx context.Context) ([]identity.User, error) {
	var users []identity.User
	return users, r.db.WithContext(ctx).
		Preload("Roles").
		Preload("ScopeAccess").
		Order("full_name").
		Find(&users).Error
}
