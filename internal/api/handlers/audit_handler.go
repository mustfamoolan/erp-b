package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	appaudit "m3aml-erp/internal/application/audit"
)

// AuditHandler exposes read-only endpoints for the audit log.
// Writing to the audit log is done internally by services.
type AuditHandler struct {
	svc *appaudit.AuditService
}

func NewAuditHandler(svc *appaudit.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// GetAll GET /api/v1/audit
func (h *AuditHandler) GetAll(c *fiber.Ctx) error {
	logs, err := h.svc.GetAll(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch audit logs"})
	}
	return c.JSON(fiber.Map{"data": logs})
}

// GetByEntity GET /api/v1/audit/entity/:type/:id
func (h *AuditHandler) GetByEntity(c *fiber.Ctx) error {
	entityType := c.Params("type")
	entityID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid entity id"})
	}

	logs, err := h.svc.GetByEntity(c.Context(), entityType, &entityID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch audit logs"})
	}
	return c.JSON(fiber.Map{"data": logs})
}

// GetByScope GET /api/v1/audit/scope/:scopeID
func (h *AuditHandler) GetByScope(c *fiber.Ctx) error {
	scopeID, err := uuid.Parse(c.Params("scopeID"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scope id"})
	}

	logs, err := h.svc.GetByScope(c.Context(), scopeID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch audit logs"})
	}
	return c.JSON(fiber.Map{"data": logs})
}

// GetByUser GET /api/v1/audit/user/:userID
func (h *AuditHandler) GetByUser(c *fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("userID"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user id"})
	}

	logs, err := h.svc.GetByUser(c.Context(), userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch audit logs"})
	}
	return c.JSON(fiber.Map{"data": logs})
}
