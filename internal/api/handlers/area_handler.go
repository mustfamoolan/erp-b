package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"m3aml-erp/internal/application/organization"
	domainorg "m3aml-erp/internal/domain/organization"
)

type AreaHandler struct {
	svc organization.AreaService
}

func NewAreaHandler(svc organization.AreaService) *AreaHandler {
	return &AreaHandler{svc: svc}
}

type CreateAreaRequest struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Governorate string `json:"governorate"`
	Description string `json:"description"`
}

func (h *AreaHandler) CreateArea(c *fiber.Ctx) error {
	var req CreateAreaRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	area, err := h.svc.CreateArea(c.Context(), req.Name, req.Code, req.Governorate, req.Description)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": area})
}

type UpdateAreaRequest struct {
	Name        string               `json:"name"`
	Code        string               `json:"code"`
	Governorate string               `json:"governorate"`
	Description string               `json:"description"`
	Status      domainorg.AreaStatus `json:"status"`
}

func (h *AreaHandler) UpdateArea(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid area id"})
	}

	var req UpdateAreaRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	area, err := h.svc.UpdateArea(c.Context(), id, req.Name, req.Code, req.Governorate, req.Description, req.Status)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"data": area})
}

func (h *AreaHandler) ListAreas(c *fiber.Ctx) error {
	areas, err := h.svc.ListAreas(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"data": areas})
}

func (h *AreaHandler) GetArea(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid area id"})
	}

	area, err := h.svc.GetArea(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if area == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "area not found"})
	}

	return c.JSON(fiber.Map{"data": area})
}
