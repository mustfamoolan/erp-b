package identity

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"m3aml-erp/internal/domain/identity"
	"m3aml-erp/internal/repositories"
)

// UserService handles all user management business logic.
// Business rules ALWAYS belong in Go — Rule 20.
// Never trust the WPF UI for security — Rule 9.
type UserService struct {
	userRepo  repositories.UserRepository
	roleRepo  repositories.RoleRepository
}

func NewUserService(userRepo repositories.UserRepository, roleRepo repositories.RoleRepository) *UserService {
	return &UserService{userRepo: userRepo, roleRepo: roleRepo}
}

// CreateUserRequest holds data needed to create a new user.
type CreateUserRequest struct {
	Username   string
	Email      string
	Password   string
	FullName   string
	EmployeeID *uuid.UUID
}

// ListUsers returns all users. Password hashes are never returned.
func (s *UserService) ListUsers(ctx context.Context) ([]identity.User, error) {
	return s.userRepo.FindAll(ctx)
}

// GetUser returns a single user by ID.
func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*identity.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

func (s *UserService) FindUserByEmployeeID(ctx context.Context, empID uuid.UUID) (*identity.User, error) {
	return s.userRepo.FindByEmployeeID(ctx, empID)
}

// CreateUser creates a new user with hashed password.
// Enforces uniqueness on username and email.
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) (*identity.User, error) {
	if req.Username == "" || req.Email == "" || req.Password == "" || req.FullName == "" {
		return nil, fmt.Errorf("username, email, password, and full_name are required")
	}

	// Check duplicate username
	existing, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("username '%s' is already taken", req.Username)
	}

	// Check duplicate email
	existingEmail, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	if existingEmail != nil {
		return nil, fmt.Errorf("email '%s' is already registered", req.Email)
	}

	// Hash password — bcrypt, never store plaintext — Rule 20
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("password hashing failed: %w", err)
	}

	user := &identity.User{
		ID:           uuid.New(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		FullName:     req.FullName,
		Status:       identity.UserStatusActive,
		EmployeeID:   req.EmployeeID,
	}
	if err := s.userRepo.Save(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}
	return user, nil
}

// UpdateUserCredentials updates the username and/or password of a user
func (s *UserService) UpdateUserCredentials(ctx context.Context, userID uuid.UUID, username, password string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}

	updated := false
	if username != "" && user.Username != username {
		user.Username = username
		updated = true
	}
	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		user.PasswordHash = string(hash)
		updated = true
	}

	if updated {
		return s.userRepo.Update(ctx, user)
	}
	return nil
}

// GrantScopeAccess grants a user explicit access to a scope.
// Rule 7: Factory A user MUST NOT automatically access Factory B.
// This is the ONLY mechanism by which scope access is granted.
func (s *UserService) GrantScopeAccess(ctx context.Context, targetUserID uuid.UUID, scopeID uuid.UUID, grantedBy uuid.UUID) error {
	// Verify user exists
	user, err := s.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("scope grant: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Check not already granted
	currentScopes, err := s.userRepo.GetUserScopeAccess(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("scope grant: %w", err)
	}
	for _, sid := range currentScopes {
		if sid == scopeID {
			return fmt.Errorf("user already has access to this scope")
		}
	}

	return s.userRepo.GrantScopeAccess(ctx, targetUserID, scopeID, grantedBy)
}

// RevokeScopeAccess removes a user's explicit access to a scope.
func (s *UserService) RevokeScopeAccess(ctx context.Context, userID uuid.UUID, scopeID uuid.UUID) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("scope revoke: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}
	return s.userRepo.RevokeScopeAccess(ctx, userID, scopeID)
}

// AssignRole assigns a role to a user, optionally scoped to a specific scope.
// Rule 9: Roles must not be the only security mechanism.
func (s *UserService) AssignRole(ctx context.Context, userID uuid.UUID, roleID uuid.UUID, scopeID *uuid.UUID) error {
	// Verify user exists
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("assign role: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Verify role exists
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("assign role: %w", err)
	}
	if role == nil {
		return fmt.Errorf("role not found")
	}

	return s.userRepo.AssignRole(ctx, userID, roleID, scopeID)
}

// RevokeRole removes a role from a user.
func (s *UserService) RevokeRole(ctx context.Context, userID uuid.UUID, roleID uuid.UUID) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("revoke role: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}
	return s.userRepo.RevokeRole(ctx, userID, roleID)
}

func (s *UserService) ClearUserScopes(ctx context.Context, userID uuid.UUID) error {
	return s.userRepo.ClearUserScopes(ctx, userID)
}

func (s *UserService) ClearUserRoles(ctx context.Context, userID uuid.UUID) error {
	return s.userRepo.ClearUserRoles(ctx, userID)
}
