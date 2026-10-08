package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"m3aml-erp/bootstrap"
	appidentity "m3aml-erp/internal/application/identity"
	"m3aml-erp/internal/repositories"
)

// UserHandler handles all user management API endpoints.
// Roadmap §8 — Users, Roles, Scope Access.
// Rule 8: Security enforced on the backend, never in WPF.
type UserHandler struct {
	userSvc   *appidentity.UserService
	roleRepo  repositories.RoleRepository
	scopeRepo repositories.ScopeRepository
}

func NewUserHandler(
	userSvc   *appidentity.UserService,
	roleRepo  repositories.RoleRepository,
	scopeRepo repositories.ScopeRepository,
) *UserHandler {
	return &UserHandler{
		userSvc:   userSvc,
		roleRepo:  roleRepo,
		scopeRepo: scopeRepo,
	}
}

// ─── GET /api/v1/users ──────────────────────────────────────

// ListUsers returns all users. Requires users.manage permission.
func (h *UserHandler) ListUsers(c *fiber.Ctx) error {
	users, err := h.userSvc.ListUsers(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch users"})
	}
	// Never expose password_hash in response
	type safeUser struct {
		ID         string `json:"id"`
		Username   string `json:"username"`
		Email      string `json:"email"`
		FullName   string `json:"full_name"`
		Name       string `json:"name"`
		Status     string `json:"status"`
		EmployeeID string `json:"employee_id,omitempty"`
	}
	result := make([]safeUser, 0, len(users))
	for _, u := range users {
		su := safeUser{
			ID:       u.ID.String(),
			Username: u.Username,
			Email:    u.Email,
			FullName: u.FullName,
			Name:     u.FullName,
			Status:   string(u.Status),
		}
		if u.EmployeeID != nil {
			su.EmployeeID = u.EmployeeID.String()
		}
		result = append(result, su)
	}
	return c.JSON(fiber.Map{"data": result})
}

// ─── GET /api/v1/users/lookup ──────────────────────────────

type LookupUser struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

// ListUsersLookup returns basic user info (id, name, full_name, username) for display resolution.
// Cached in Redis with fast zero-latency response.
func (h *UserHandler) ListUsersLookup(c *fiber.Ctx) error {
	result, err := bootstrap.CacheRemember("org:users:lookup", 1*time.Hour, func() ([]LookupUser, error) {
		users, err := h.userSvc.ListUsers(c.Context())
		if err != nil {
			return nil, err
		}
		res := make([]LookupUser, 0, len(users))
		for _, u := range users {
			displayName := u.FullName
			if displayName == "" {
				displayName = u.Username
			}
			res = append(res, LookupUser{
				ID:       u.ID.String(),
				FullName: displayName,
				Name:     displayName,
				Username: u.Username,
			})
		}
		return res, nil
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch users"})
	}
	return c.JSON(fiber.Map{"data": result})
}

// ─── GET /api/v1/users/:id ──────────────────────────────────

// GetUser returns a single user. Never exposes password_hash.
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user id"})
	}
	user, err := h.userSvc.GetUser(c.Context(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "lookup failed"})
	}
	if user == nil {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}
	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"full_name": user.FullName,
			"status":    user.Status,
			"roles":     user.Roles,
			"scopes":    user.ScopeAccess,
		},
	})
}

// ─── POST /api/v1/users ─────────────────────────────────────

type createUserRequest struct {
	Username   string `json:"username"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	FullName   string `json:"full_name"`
	EmployeeID string `json:"employee_id,omitempty"`
}

// CreateUser creates a new user. Requires users.manage permission.
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	var req createUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.Username == "" || req.Email == "" || req.Password == "" || req.FullName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "username, email, password, full_name are required"})
	}
	if len(req.Password) < 8 {
		return c.Status(400).JSON(fiber.Map{"error": "password must be at least 8 characters"})
	}

	createReq := appidentity.CreateUserRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	}
	if req.EmployeeID != "" {
		empID, err := uuid.Parse(req.EmployeeID)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid employee_id"})
		}
		createReq.EmployeeID = &empID
	}

	user, err := h.userSvc.CreateUser(c.Context(), createReq)
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{
		"data": fiber.Map{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"full_name": user.FullName,
			"status":    user.Status,
		},
	})
}

// ─── POST /api/v1/users/:id/roles ───────────────────────────

type assignRoleRequest struct {
	RoleID  string `json:"role_id"`
	ScopeID string `json:"scope_id,omitempty"` // optional — nil = company-wide role
}

// AssignRole assigns a role to a user, optionally scoped.
// Rule 7: scope-scoped roles restrict access to a specific factory.
func (h *UserHandler) AssignRole(c *fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user id"})
	}

	var req assignRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid role_id"})
	}

	var scopeID *uuid.UUID
	if req.ScopeID != "" {
		sid, err := uuid.Parse(req.ScopeID)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid scope_id"})
		}
		scopeID = &sid
	}

	if err := h.userSvc.AssignRole(c.Context(), userID, roleID, scopeID); err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "role assigned successfully"})
}

// ─── DELETE /api/v1/users/:id/roles/:role_id ────────────────

// RevokeRole removes a role from a user.
func (h *UserHandler) RevokeRole(c *fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user id"})
	}
	roleID, err := uuid.Parse(c.Params("role_id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid role_id"})
	}
	if err := h.userSvc.RevokeRole(c.Context(), userID, roleID); err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "role revoked successfully"})
}

// ─── POST /api/v1/users/:id/scopes ──────────────────────────

type grantScopeRequest struct {
	ScopeID string `json:"scope_id"`
}

// GrantScope grants a user explicit access to a scope.
// Rule 7: This is the ONLY way a user gets scope access.
// No automatic access is ever granted.
func (h *UserHandler) GrantScope(c *fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user id"})
	}

	var req grantScopeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	scopeID, err := uuid.Parse(req.ScopeID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scope_id"})
	}

	grantedByStr, _ := c.Locals("user_id").(string)
	grantedBy, _ := uuid.Parse(grantedByStr)

	if err := h.userSvc.GrantScopeAccess(c.Context(), userID, scopeID, grantedBy); err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "scope access granted"})
}

// ─── DELETE /api/v1/users/:id/scopes/:scope_id ──────────────

// RevokeScope removes a user's access to a scope.
func (h *UserHandler) RevokeScope(c *fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user id"})
	}
	scopeID, err := uuid.Parse(c.Params("scope_id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scope_id"})
	}
	if err := h.userSvc.RevokeScopeAccess(c.Context(), userID, scopeID); err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "scope access revoked"})
}

// ─── PUT /api/v1/users/:id/credentials ──────────────────────

type updateCredentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UpdateCredentials updates username and/or password.
func (h *UserHandler) UpdateCredentials(c *fiber.Ctx) error {
	idStr := c.Params("id")
	var userID uuid.UUID
	var err error
	if idStr == "me" {
		currentUserID, ok := c.Locals("user_id").(string)
		if !ok || currentUserID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		userID, err = uuid.Parse(currentUserID)
	} else {
		userID, err = uuid.Parse(idStr)
	}
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	var req updateCredentialsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Username == "" && req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "username or password required"})
	}

	if req.Password != "" && len(req.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "password must be at least 6 characters"})
	}

	if err := h.userSvc.UpdateUserCredentials(c.Context(), userID, req.Username, req.Password); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "credentials updated successfully"})
}
