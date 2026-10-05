package organization

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"m3aml-erp/internal/api/middleware"
	apporg "m3aml-erp/internal/application/organization"
)

type CompanySettingsHandler struct {
	service *apporg.CompanySettingsService
}

func NewCompanySettingsHandler(service *apporg.CompanySettingsService) *CompanySettingsHandler {
	return &CompanySettingsHandler{service: service}
}

func (h *CompanySettingsHandler) Get(c *fiber.Ctx) error {
	settings, err := h.service.GetSettings(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to get company settings"})
	}

	return c.JSON(fiber.Map{"data": settings})
}

type UpdateCompanySettingsRequest struct {
	CompanyName string `json:"company_name"`
	PhoneNumber string `json:"phone_number"`
	LogoUrl     string `json:"logo_url"`
}

func (h *CompanySettingsHandler) Update(c *fiber.Ctx) error {
	var req UpdateCompanySettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.CompanyName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Company name is required"})
	}

	userIDStr, ok := c.Locals(middleware.LocalUserID).(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthenticated"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid user id"})
	}

	settings, err := h.service.UpdateSettings(c.Context(), userID, req.CompanyName, req.PhoneNumber, req.LogoUrl)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update settings"})
	}

	return c.JSON(fiber.Map{"data": settings})
}
