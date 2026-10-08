package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"m3aml-erp/internal/domain/identity"
)

// userRepository is the PostgreSQL implementation of UserRepository.
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a PostgreSQL-backed UserRepository.
func NewUserRepository(db *gorm.DB) *userRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.User, error) {
	var user identity.User
	err := r.db.WithContext(ctx).
		Preload("Roles").
		Preload("ScopeAccess").
		First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*identity.User, error) {
	var user identity.User
	err := r.db.WithContext(ctx).First(&user, "username = ? OR full_name = ?", username, username).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*identity.User, error) {
	var user identity.User
	err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *userRepository) FindByEmployeeID(ctx context.Context, empID uuid.UUID) (*identity.User, error) {
	var user identity.User
	err := r.db.WithContext(ctx).
		Preload("Roles").
		Preload("ScopeAccess").
		Where("employee_id = ?", empID).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *userRepository) Save(ctx context.Context, user *identity.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) Update(ctx context.Context, user *identity.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// GetUserScopeAccess returns all scope IDs the user has explicit access to.
// This is the enforcement mechanism for Rule 7: Factory A cannot access Factory B.
func (r *userRepository) GetUserScopeAccess(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var accesses []identity.UserScopeAccess
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&accesses).Error
	if err != nil {
		return nil, err
	}
	scopeIDs := make([]uuid.UUID, 0, len(accesses))
	for _, a := range accesses {
		scopeIDs = append(scopeIDs, a.ScopeID)
	}
	return scopeIDs, nil
}

// GetUserPermissions resolves all permissions for a user within a specific scope.
// Joins user_roles → role_permissions → permissions where role is global or matches scopeID.
// If scopeID is uuid.Nil (empty), returns permissions from ALL scopes the user has access to.
// This is correct behavior: permission check happens at middleware level,
// scope enforcement happens at handler level (Roadmap Rule 7, Rule 8).
func (r *userRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID, scopeID uuid.UUID) ([]identity.PermissionCode, error) {
	var codes []string
	var err error

	if scopeID == uuid.Nil {
		// No scope specified — return permissions from all roles the user has (any scope).
		// Handler-level scope enforcement will restrict data access.
		err = r.db.WithContext(ctx).Raw(`
			SELECT DISTINCT p.code
			FROM permissions p
			JOIN role_permissions rp ON rp.permission_id = p.id
			JOIN user_roles ur ON ur.role_id = rp.role_id
			WHERE ur.user_id = ?
		`, userID).Scan(&codes).Error
	} else {
		// Scope specified — only roles that are global (NULL scope) or match the specific scope.
		err = r.db.WithContext(ctx).Raw(`
			SELECT DISTINCT p.code
			FROM permissions p
			JOIN role_permissions rp ON rp.permission_id = p.id
			JOIN user_roles ur ON ur.role_id = rp.role_id
			WHERE ur.user_id = ?
			  AND (ur.scope_id IS NULL OR ur.scope_id = ?)
		`, userID, scopeID).Scan(&codes).Error
	}

	if err != nil {
		return nil, err
	}
	perms := make([]identity.PermissionCode, 0, len(codes))
	for _, c := range codes {
		perms = append(perms, identity.PermissionCode(c))
	}
	return perms, nil
}

