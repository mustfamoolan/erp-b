package handlers

import (
	"github.com/gofiber/fiber/v2"
	"m3aml-erp/internal/application/auth"
)

// AuthHandler handles authentication API endpoints.
type AuthHandler struct {
	authSvc *auth.AuthService
}

func NewAuthHandler(authSvc *auth.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// LoginRequest is the JSON body for POST /api/v1/auth/login
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login handles POST /api/v1/auth/login
// Returns JWT token on success.
// Roadmap §8 — authentication is the backend's responsibility.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "username and password are required",
		})
	}

	resp, err := h.authSvc.Login(c.Context(), auth.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		switch err {
		case auth.ErrInvalidCredentials:
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid username or password",
			})
		case auth.ErrUserInactive:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "account is inactive or suspended",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "authentication failed",
			})
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": fiber.Map{
			"token":      resp.Token,
			"expires_at": resp.ExpiresAt,
			"user": fiber.Map{
				"id":        resp.UserID,
				"username":  resp.Username,
				"full_name": resp.FullName,
			},
			"roles":  resp.Roles,
			"scopes": resp.Scopes,
		},
	})
}
