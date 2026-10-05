package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"m3aml-erp/internal/application/masterdata"
	domain "m3aml-erp/internal/domain/masterdata"
)

type MasterDataHandler struct {
	service masterdata.MasterDataService
}

func NewMasterDataHandler(service masterdata.MasterDataService) *MasterDataHandler {
	return &MasterDataHandler{service: service}
}

// RequestTypeInput defines the payload for creating/updating a request type
type RequestTypeInput struct {
	Code        string `json:"code" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

// ExpenseCategoryInput defines the payload for creating/updating an expense category
type ExpenseCategoryInput struct {
	Code      string `json:"code" validate:"required"`
	Name      string `json:"name" validate:"required"`
	SortOrder int    `json:"sort_order"`
}

// -- Request Types --

func (h *MasterDataHandler) GetRequestTypes(c *fiber.Ctx) error {
	activeOnly := c.QueryBool("active_only", false)
	types, err := h.service.GetRequestTypes(c.Context(), activeOnly)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(types)
}

func (h *MasterDataHandler) CreateRequestType(c *fiber.Ctx) error {
	var input RequestTypeInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	userID := c.Locals("user_id").(uuid.UUID) // Assumes auth middleware

	rt := domain.RequestType{
		Code:        input.Code,
		Name:        input.Name,
		Description: input.Description,
		SortOrder:   input.SortOrder,
	}

	created, err := h.service.CreateRequestType(c.Context(), rt, userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(created)
}

func (h *MasterDataHandler) UpdateRequestType(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	var input RequestTypeInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	userID := c.Locals("user_id").(uuid.UUID)

	updates := domain.RequestType{
		Name:        input.Name,
		Description: input.Description,
		SortOrder:   input.SortOrder,
	}

	if err := h.service.UpdateRequestType(c.Context(), id, updates, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Request type updated successfully"})
}

func (h *MasterDataHandler) ToggleRequestType(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	var input struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	userID := c.Locals("user_id").(uuid.UUID)

	if err := h.service.ToggleRequestType(c.Context(), id, input.IsActive, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Request type status updated"})
}

// -- Expense Categories --

func (h *MasterDataHandler) GetExpenseCategories(c *fiber.Ctx) error {
	activeOnly := c.QueryBool("active_only", false)
	cats, err := h.service.GetExpenseCategories(c.Context(), activeOnly)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(cats)
}

func (h *MasterDataHandler) CreateExpenseCategory(c *fiber.Ctx) error {
	var input ExpenseCategoryInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	userID := c.Locals("user_id").(uuid.UUID)

	ec := domain.ExpenseCategory{
		Code:      input.Code,
		Name:      input.Name,
		SortOrder: input.SortOrder,
	}

	created, err := h.service.CreateExpenseCategory(c.Context(), ec, userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(created)
}

func (h *MasterDataHandler) UpdateExpenseCategory(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	var input ExpenseCategoryInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	userID := c.Locals("user_id").(uuid.UUID)

	updates := domain.ExpenseCategory{
		Name:      input.Name,
		SortOrder: input.SortOrder,
	}

	if err := h.service.UpdateExpenseCategory(c.Context(), id, updates, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Expense category updated successfully"})
}

func (h *MasterDataHandler) ToggleExpenseCategory(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	var input struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	userID := c.Locals("user_id").(uuid.UUID)

	if err := h.service.ToggleExpenseCategory(c.Context(), id, input.IsActive, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Expense category status updated"})
}
