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

// FactoryExpenseTypeInput defines the payload for creating/updating a factory expense type
type FactoryExpenseTypeInput struct {
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
	return c.JSON(fiber.Map{"data": types})
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

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": created})
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
	return c.JSON(fiber.Map{"data": cats})
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

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": created})
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

// -- Factory Expense Types --

func (h *MasterDataHandler) GetFactoryExpenseTypes(c *fiber.Ctx) error {
	activeOnly := c.QueryBool("active_only", false)
	types, err := h.service.GetFactoryExpenseTypes(c.Context(), activeOnly)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": types})
}

func (h *MasterDataHandler) CreateFactoryExpenseType(c *fiber.Ctx) error {
	var input FactoryExpenseTypeInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	userID := c.Locals("user_id").(uuid.UUID)

	fet := domain.FactoryExpenseType{
		Code:      input.Code,
		Name:      input.Name,
		SortOrder: input.SortOrder,
	}

	created, err := h.service.CreateFactoryExpenseType(c.Context(), fet, userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": created})
}

func (h *MasterDataHandler) UpdateFactoryExpenseType(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	var input FactoryExpenseTypeInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	userID := c.Locals("user_id").(uuid.UUID)

	updates := domain.FactoryExpenseType{
		Name:      input.Name,
		SortOrder: input.SortOrder,
	}

	if err := h.service.UpdateFactoryExpenseType(c.Context(), id, updates, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Factory expense type updated successfully"})
}

func (h *MasterDataHandler) ToggleFactoryExpenseType(c *fiber.Ctx) error {
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

	if err := h.service.ToggleFactoryExpenseType(c.Context(), id, input.IsActive, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Factory expense type status updated"})
}

// ReceivingMethodInput defines the payload for creating/updating a receiving method
type ReceivingMethodInput struct {
	Code      string `json:"code" validate:"required"`
	Name      string `json:"name" validate:"required"`
	SortOrder int    `json:"sort_order"`
}

// UnitInput defines the payload for creating/updating a unit of measure
type UnitInput struct {
	Code      string `json:"code" validate:"required"`
	Name      string `json:"name" validate:"required"`
	SortOrder int    `json:"sort_order"`
}

// -- Receiving Methods --

func (h *MasterDataHandler) GetReceivingMethods(c *fiber.Ctx) error {
	activeOnly := c.QueryBool("active_only", false)
	methods, err := h.service.GetReceivingMethods(c.Context(), activeOnly)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": methods})
}

func (h *MasterDataHandler) CreateReceivingMethod(c *fiber.Ctx) error {
	var input ReceivingMethodInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var userID uuid.UUID
	if uid, ok := c.Locals("user_id").(uuid.UUID); ok {
		userID = uid
	} else if uids, ok := c.Locals("user_id").(string); ok {
		userID, _ = uuid.Parse(uids)
	}

	rm := domain.ReceivingMethod{
		Code:      input.Code,
		Name:      input.Name,
		SortOrder: input.SortOrder,
	}

	created, err := h.service.CreateReceivingMethod(c.Context(), rm, userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": created})
}

func (h *MasterDataHandler) UpdateReceivingMethod(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	var input ReceivingMethodInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var userID uuid.UUID
	if uid, ok := c.Locals("user_id").(uuid.UUID); ok {
		userID = uid
	} else if uids, ok := c.Locals("user_id").(string); ok {
		userID, _ = uuid.Parse(uids)
	}

	updates := domain.ReceivingMethod{
		Name:      input.Name,
		SortOrder: input.SortOrder,
	}

	if err := h.service.UpdateReceivingMethod(c.Context(), id, updates, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Receiving method updated successfully"})
}

func (h *MasterDataHandler) ToggleReceivingMethod(c *fiber.Ctx) error {
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

	var userID uuid.UUID
	if uid, ok := c.Locals("user_id").(uuid.UUID); ok {
		userID = uid
	} else if uids, ok := c.Locals("user_id").(string); ok {
		userID, _ = uuid.Parse(uids)
	}

	if err := h.service.ToggleReceivingMethod(c.Context(), id, input.IsActive, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Receiving method status updated"})
}

// -- Units of Measure --

func (h *MasterDataHandler) GetUnits(c *fiber.Ctx) error {
	activeOnly := c.QueryBool("active_only", false)
	units, err := h.service.GetUnits(c.Context(), activeOnly)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": units})
}

func (h *MasterDataHandler) CreateUnit(c *fiber.Ctx) error {
	var input UnitInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var userID uuid.UUID
	if uid, ok := c.Locals("user_id").(uuid.UUID); ok {
		userID = uid
	} else if uids, ok := c.Locals("user_id").(string); ok {
		userID, _ = uuid.Parse(uids)
	}

	u := domain.UnitOfMeasure{
		Code:      input.Code,
		Name:      input.Name,
		SortOrder: input.SortOrder,
	}

	created, err := h.service.CreateUnit(c.Context(), u, userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": created})
}

func (h *MasterDataHandler) UpdateUnit(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	var input UnitInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var userID uuid.UUID
	if uid, ok := c.Locals("user_id").(uuid.UUID); ok {
		userID = uid
	} else if uids, ok := c.Locals("user_id").(string); ok {
		userID, _ = uuid.Parse(uids)
	}

	updates := domain.UnitOfMeasure{
		Name:      input.Name,
		SortOrder: input.SortOrder,
	}

	if err := h.service.UpdateUnit(c.Context(), id, updates, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Unit updated successfully"})
}

func (h *MasterDataHandler) ToggleUnit(c *fiber.Ctx) error {
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

	var userID uuid.UUID
	if uid, ok := c.Locals("user_id").(uuid.UUID); ok {
		userID = uid
	} else if uids, ok := c.Locals("user_id").(string); ok {
		userID, _ = uuid.Parse(uids)
	}

	if err := h.service.ToggleUnit(c.Context(), id, input.IsActive, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Unit status updated"})
}

