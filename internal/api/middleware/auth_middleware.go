package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"m3aml-erp/internal/application/auth"
	"m3aml-erp/internal/domain/identity"
	"m3aml-erp/internal/repositories"
)

// ContextKey type for storing values in fiber.Ctx locals.
type ContextKey string

const (
	LocalUserID      = "user_id"
	LocalUsername    = "username"
	LocalScopeIDs    = "scope_ids"
	LocalPermissions = "permissions"
)

// AuthMiddleware validates JWT and populates context with user identity.
// Security MUST be enforced here — never rely on WPF UI restrictions (Roadmap Rule 8, Rule 9).
func AuthMiddleware(authSvc *auth.AuthService, userRepo repositories.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing or invalid authorization header",
			})
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := authSvc.ParseToken(tokenStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired token",
			})
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid user id in token",
			})
		}

		user, err := userRepo.FindByID(c.Context(), userID)
		if err != nil || user == nil || user.Status != identity.UserStatusActive {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "user account is inactive or does not exist",
			})
		}

		c.Locals(LocalUserID, claims.UserID)
		c.Locals(LocalUsername, claims.Username)
		return c.Next()
	}
}

// EnsureScopeAccess verifies if a user has access to a specific scope, either
// through direct user_scope_access mapping or via global 'scope.all' permission.
func EnsureScopeAccess(ctx context.Context, userRepo repositories.UserRepository, userID uuid.UUID, requestedScope uuid.UUID) error {
	// First check if user has global scope.all permission
	perms, err := userRepo.GetUserPermissions(ctx, userID, uuid.Nil)
	if err == nil {
		for _, p := range perms {
			if p == identity.PermScopeAll {
				return nil // Global access granted
			}
		}
	}

	// Fetch user's allowed scopes from DB — ALWAYS from backend, never from client
	allowedScopes, err := userRepo.GetUserScopeAccess(ctx, userID)
	if err != nil {
		return err
	}

	for _, sid := range allowedScopes {
		if sid == requestedScope {
			return nil // Scope-specific access granted
		}
	}

	return fiber.ErrForbidden
}

// ScopeMiddleware verifies the requesting user has access to the scope in the URL param.
// This enforces Rule 7: Factory A user MUST NOT access Factory B.
// Rule 8: This MUST happen on the backend — not the WPF.
func ScopeMiddleware(userRepo repositories.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userIDStr, ok := c.Locals(LocalUserID).(string)
		if !ok || userIDStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthenticated"})
		}
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user id"})
		}

		// The scope being accessed comes from the URL param or query
		scopeIDStr := c.Params("scope_id", c.Query("scope_id"))
		if scopeIDStr == "" {
			return c.Next() // no scope required for this endpoint
		}
		requestedScope, err := uuid.Parse(scopeIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid scope_id"})
		}

		err = EnsureScopeAccess(c.Context(), userRepo, userID, requestedScope)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "access denied to requested scope",
			})
		}

		return c.Next()
	}
}

// RequirePermission checks that the authenticated user has a specific permission in context.
// Uses the DB-resolved permission list — never trusts client-side data (Rule 8).
func RequirePermission(userRepo repositories.UserRepository, required identity.PermissionCode) fiber.Handler {
	return RequirePermissionFor(userRepo, func(*fiber.Ctx) identity.PermissionCode { return required })
}

// RequirePermissionFor resolves the required permission per request (e.g. from the body),
// so one endpoint serving several operation kinds still demands the specific permission
// for the operation being performed (AGENTS.md §2.2).
func RequirePermissionFor(userRepo repositories.UserRepository, resolve func(c *fiber.Ctx) identity.PermissionCode) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userIDStr, _ := c.Locals(LocalUserID).(string)
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthenticated"})
		}

		required := resolve(c)
		if required == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unsupported operation"})
		}

		scopeIDStr := c.Params("scope_id", c.Query("scope_id"))
		var scopeID uuid.UUID
		if scopeIDStr != "" {
			scopeID, _ = uuid.Parse(scopeIDStr)
		}

		perms, err := userRepo.GetUserPermissions(c.Context(), userID, scopeID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "permission check failed"})
		}

		for _, p := range perms {
			if p == required {
				c.Locals(LocalPermissions, perms)
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":    "permission denied",
			"required": string(required),
		})
	}
}
