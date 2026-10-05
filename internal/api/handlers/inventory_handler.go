package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	appinv "m3aml-erp/internal/application/inventory"
	"m3aml-erp/internal/domain/inventory"
)

type InventoryHandler struct {
	invSvc *appinv.InventoryService
}

func NewInventoryHandler(svc *appinv.InventoryService) *InventoryHandler {
	return &InventoryHandler{invSvc: svc}
}

// GetCategories GET /api/v1/inventory/categories
func (h *InventoryHandler) GetCategories(c *fiber.Ctx) error {
	cats, err := h.invSvc.GetCategories(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch categories"})
	}
	return c.JSON(fiber.Map{"data": cats})
}

type createCategoryReq struct {
	Name     string `json:"name"`
	ParentID string `json:"parent_id"`
}

// CreateCategory POST /api/v1/inventory/categories
func (h *InventoryHandler) CreateCategory(c *fiber.Ctx) error {
	var req createCategoryReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	var parentID *uuid.UUID
	if req.ParentID != "" {
		id, err := uuid.Parse(req.ParentID)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid parent_id"})
		}
		parentID = &id
	}

	cat, err := h.invSvc.CreateCategory(c.Context(), req.Name, parentID)
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"data": cat})
}

// GetUnits GET /api/v1/inventory/units
func (h *InventoryHandler) GetUnits(c *fiber.Ctx) error {
	units, err := h.invSvc.GetUnits(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch units"})
	}
	return c.JSON(fiber.Map{"data": units})
}

// GetItems GET /api/v1/inventory/items
func (h *InventoryHandler) GetItems(c *fiber.Ctx) error {
	items, err := h.invSvc.GetItems(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch items"})
	}
	return c.JSON(fiber.Map{"data": items})
}

type createItemReq struct {
	CategoryID  string `json:"category_id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	BaseUnitID  string `json:"base_unit_id"`
}

// CreateItem POST /api/v1/inventory/items
func (h *InventoryHandler) CreateItem(c *fiber.Ctx) error {
	var req createItemReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	catID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid category_id"})
	}
	unitID, err := uuid.Parse(req.BaseUnitID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid base_unit_id"})
	}

	item, err := h.invSvc.CreateItem(c.Context(), appinv.CreateItemRequest{
		CategoryID:  catID,
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		BaseUnitID:  unitID,
	})
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"data": item})
}

type createVariantReq struct {
	ItemID     string            `json:"item_id"`
	Name       string            `json:"name"`
	SKU        string            `json:"sku"`
	Barcode    string            `json:"barcode"`
	UnitID     string            `json:"unit_id"`
	Attributes map[string]string `json:"attributes"`
}

// CreateVariant POST /api/v1/inventory/variants
func (h *InventoryHandler) CreateVariant(c *fiber.Ctx) error {
	var req createVariantReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	itemID, err := uuid.Parse(req.ItemID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid item_id"})
	}
	unitID, err := uuid.Parse(req.UnitID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid unit_id"})
	}

	variant, err := h.invSvc.CreateVariant(c.Context(), appinv.CreateVariantRequest{
		ItemID:     itemID,
		Name:       req.Name,
		SKU:        req.SKU,
		Barcode:    req.Barcode,
		UnitID:     unitID,
		Attributes: req.Attributes,
	})
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"data": variant})
}

// ─── Warehouses ───────────────────────────────────────────────────────────────

type createWarehouseReq struct {
	ScopeID string `json:"scope_id"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	Type    string `json:"type"`
}

// CreateWarehouse POST /api/v1/inventory/warehouses
func (h *InventoryHandler) CreateWarehouse(c *fiber.Ctx) error {
	var req createWarehouseReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	scopeID, err := uuid.Parse(req.ScopeID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scope_id"})
	}

	wh, err := h.invSvc.CreateWarehouse(c.Context(), appinv.CreateWarehouseRequest{
		ScopeID: scopeID,
		Code:    req.Code,
		Name:    req.Name,
		Type:    inventory.WarehouseType(req.Type),
	})
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"data": wh})
}

// GetWarehousesByScope GET /api/v1/inventory/warehouses/:scopeID
func (h *InventoryHandler) GetWarehousesByScope(c *fiber.Ctx) error {
	scopeID, err := uuid.Parse(c.Params("scopeID"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scopeID"})
	}
	whs, err := h.invSvc.GetWarehousesByScope(c.Context(), scopeID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch warehouses"})
	}
	return c.JSON(fiber.Map{"data": whs})
}

// ─── Stock Movements ──────────────────────────────────────────────────────────

type recordMovementReq struct {
	WarehouseID   string  `json:"warehouse_id"`
	VariantID     string  `json:"variant_id"`
	Quantity      float64 `json:"quantity"`
	MovementType  string  `json:"movement_type"`
	ReferenceType string  `json:"reference_type"`
	ReferenceID   *string `json:"reference_id,omitempty"`
	Notes         string  `json:"notes"`
}

// RecordMovement POST /api/v1/inventory/movements
func (h *InventoryHandler) RecordMovement(c *fiber.Ctx) error {
	var req recordMovementReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	warehouseID, err := uuid.Parse(req.WarehouseID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid warehouse_id"})
	}
	variantID, err := uuid.Parse(req.VariantID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid variant_id"})
	}
	var refID *uuid.UUID
	if req.ReferenceID != nil {
		parsed, err := uuid.Parse(*req.ReferenceID)
		if err == nil {
			refID = &parsed
		}
	}
	userIDStr, _ := c.Locals("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	err = h.invSvc.RecordMovement(c.Context(), appinv.RecordMovementRequest{
		WarehouseID:   warehouseID,
		VariantID:     variantID,
		Quantity:      req.Quantity,
		MovementType:  inventory.StockMovementType(req.MovementType),
		ReferenceType: req.ReferenceType,
		ReferenceID:   refID,
		PerformedBy:   userID,
		Notes:         req.Notes,
	})
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"message": "movement recorded"})
}

type transferStockReq struct {
	FromWarehouseID string  `json:"from_warehouse_id"`
	ToWarehouseID   string  `json:"to_warehouse_id"`
	VariantID       string  `json:"variant_id"`
	Quantity        float64 `json:"quantity"`
	Notes           string  `json:"notes"`
}

// TransferStock POST /api/v1/inventory/transfers
func (h *InventoryHandler) TransferStock(c *fiber.Ctx) error {
	var req transferStockReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	fromWH, err := uuid.Parse(req.FromWarehouseID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid from_warehouse_id"})
	}
	toWH, err := uuid.Parse(req.ToWarehouseID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid to_warehouse_id"})
	}
	variantID, err := uuid.Parse(req.VariantID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid variant_id"})
	}
	userIDStr, _ := c.Locals("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	err = h.invSvc.TransferStock(c.Context(), fromWH, toWH, variantID, userID, req.Quantity, req.Notes)
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"message": "stock transferred atomically"})
}
