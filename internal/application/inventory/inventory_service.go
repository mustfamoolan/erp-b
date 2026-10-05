package inventory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	appaudit "m3aml-erp/internal/application/audit"
	domainaudit "m3aml-erp/internal/domain/audit"
	"m3aml-erp/internal/domain/inventory"
	"m3aml-erp/internal/repositories"
)

// InventoryService manages item master data, warehouses, and stock movements.
// Rule 12: Every inventory movement must have a traceable stock movement.
// Rule 16: Never directly overwrite authoritative stock quantity.
// §34: Inventory movements are auditable — AuditService is injected.
type InventoryService struct {
	categoryRepo  repositories.ItemCategoryRepository
	unitRepo      repositories.UnitOfMeasureRepository
	itemRepo      repositories.ItemRepository
	variantRepo   repositories.ItemVariantRepository
	warehouseRepo repositories.WarehouseRepository
	stockRepo     repositories.StockMovementRepository
	auditSvc      *appaudit.AuditService // §34
}

func NewInventoryService(
	catRepo repositories.ItemCategoryRepository,
	unitRepo repositories.UnitOfMeasureRepository,
	itemRepo repositories.ItemRepository,
	variantRepo repositories.ItemVariantRepository,
	warehouseRepo repositories.WarehouseRepository,
	stockRepo repositories.StockMovementRepository,
	auditSvc *appaudit.AuditService,
) *InventoryService {
	return &InventoryService{
		categoryRepo:  catRepo,
		unitRepo:      unitRepo,
		itemRepo:      itemRepo,
		variantRepo:   variantRepo,
		warehouseRepo: warehouseRepo,
		stockRepo:     stockRepo,
		auditSvc:      auditSvc,
	}
}

// ─── Categories & Units ───────────────────────────────────────────────────────

func (s *InventoryService) CreateCategory(ctx context.Context, name string, parentID *uuid.UUID) (*inventory.ItemCategory, error) {
	if name == "" {
		return nil, fmt.Errorf("category name is required")
	}
	cat := &inventory.ItemCategory{
		ID:       uuid.New(),
		Name:     name,
		ParentID: parentID,
	}
	if err := s.categoryRepo.Save(ctx, cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func (s *InventoryService) GetCategories(ctx context.Context) ([]inventory.ItemCategory, error) {
	return s.categoryRepo.FindAll(ctx)
}

func (s *InventoryService) GetUnits(ctx context.Context) ([]inventory.UnitOfMeasure, error) {
	return s.unitRepo.FindAll(ctx)
}

// ─── Items & Variants ─────────────────────────────────────────────────────────

type CreateItemRequest struct {
	CategoryID  uuid.UUID
	Code        string
	Name        string
	Description string
	BaseUnitID  uuid.UUID
}

func (s *InventoryService) CreateItem(ctx context.Context, req CreateItemRequest) (*inventory.Item, error) {
	if req.Code == "" || req.Name == "" {
		return nil, fmt.Errorf("item code and name are required")
	}

	cat, err := s.categoryRepo.FindByID(ctx, req.CategoryID)
	if err != nil || cat == nil {
		return nil, fmt.Errorf("invalid category ID")
	}

	item := &inventory.Item{
		ID:          uuid.New(),
		CategoryID:  req.CategoryID,
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		BaseUnitID:  req.BaseUnitID,
		IsActive:    true,
	}
	if err := s.itemRepo.Save(ctx, item); err != nil {
		return nil, fmt.Errorf("failed to save item: %w", err)
	}
	return item, nil
}

type CreateVariantRequest struct {
	ItemID     uuid.UUID
	Name       string
	SKU        string
	Barcode    string
	UnitID     uuid.UUID
	Attributes map[string]string
}

func (s *InventoryService) CreateVariant(ctx context.Context, req CreateVariantRequest) (*inventory.ItemVariant, error) {
	if req.Name == "" || req.SKU == "" {
		return nil, fmt.Errorf("variant name and SKU are required")
	}

	// Rule 26: SKU is business ID, Barcode is machine. Both must be unique.
	existingSKU, _ := s.variantRepo.FindBySKU(ctx, req.SKU)
	if existingSKU != nil {
		return nil, fmt.Errorf("SKU already exists")
	}

	if req.Barcode != "" {
		existingBarcode, _ := s.variantRepo.FindByBarcode(ctx, req.Barcode)
		if existingBarcode != nil {
			return nil, fmt.Errorf("barcode already exists")
		}
	}

	attrs := []inventory.ItemVariantAttribute{}
	for k, v := range req.Attributes {
		attrs = append(attrs, inventory.ItemVariantAttribute{
			ID:        uuid.New(),
			AttrKey:   k,
			AttrValue: v,
		})
	}

	variant := &inventory.ItemVariant{
		ID:         uuid.New(),
		ItemID:     req.ItemID,
		Name:       req.Name,
		SKU:        req.SKU,
		Barcode:    req.Barcode,
		UnitID:     req.UnitID,
		IsActive:   true,
		Attributes: attrs,
	}

	if err := s.variantRepo.Save(ctx, variant); err != nil {
		return nil, fmt.Errorf("failed to save variant: %w", err)
	}
	return variant, nil
}

func (s *InventoryService) GetItems(ctx context.Context) ([]inventory.Item, error) {
	return s.itemRepo.FindAll(ctx)
}

// ─── Warehouses ───────────────────────────────────────────────────────────────

type CreateWarehouseRequest struct {
	ScopeID uuid.UUID
	Code    string
	Name    string
	Type    inventory.WarehouseType
}

func (s *InventoryService) CreateWarehouse(ctx context.Context, req CreateWarehouseRequest) (*inventory.Warehouse, error) {
	if req.Code == "" || req.Name == "" {
		return nil, fmt.Errorf("code and name are required")
	}
	wh := &inventory.Warehouse{
		ID:       uuid.New(),
		ScopeID:  req.ScopeID,
		Code:     req.Code,
		Name:     req.Name,
		Type:     req.Type,
		IsActive: true,
	}
	if err := s.warehouseRepo.Save(ctx, wh); err != nil {
		return nil, fmt.Errorf("failed to save warehouse: %w", err)
	}
	return wh, nil
}

func (s *InventoryService) GetWarehousesByScope(ctx context.Context, scopeID uuid.UUID) ([]inventory.Warehouse, error) {
	return s.warehouseRepo.FindByScope(ctx, scopeID)
}

// ─── Stock Movements ──────────────────────────────────────────────────────────

type RecordMovementRequest struct {
	WarehouseID   uuid.UUID
	VariantID     uuid.UUID
	Quantity      float64
	MovementType  inventory.StockMovementType
	ReferenceType string
	ReferenceID   *uuid.UUID
	PerformedBy   uuid.UUID
	Notes         string
}

// RecordMovement records an immutable stock movement.
// Rule 12: Every inventory movement must have a traceable stock movement record.
// §34: RECEIPT and ADJUST are audited.
func (s *InventoryService) RecordMovement(ctx context.Context, req RecordMovementRequest) error {
	if req.Quantity == 0 {
		return fmt.Errorf("quantity cannot be zero")
	}

	// Prevent negative stock on outbound movements
	if req.Quantity < 0 {
		currentBal, err := s.stockRepo.GetStockBalance(ctx, req.WarehouseID, req.VariantID)
		if err != nil {
			return fmt.Errorf("failed to get current balance: %w", err)
		}
		if currentBal+req.Quantity < 0 {
			return fmt.Errorf("insufficient stock: current balance is %v", currentBal)
		}
	}

	// Fetch variant to get unit ID
	variant, err := s.variantRepo.FindByID(ctx, req.VariantID)
	if err != nil || variant == nil {
		return fmt.Errorf("invalid variant")
	}

	mov := &inventory.StockMovement{
		ID:            uuid.New(),
		WarehouseID:   req.WarehouseID,
		VariantID:     req.VariantID,
		Quantity:      decimal.NewFromFloat(req.Quantity),
		UnitID:        variant.UnitID,
		MovementType:  req.MovementType,
		ReferenceType: req.ReferenceType,
		ReferenceID:   req.ReferenceID,
		PerformedBy:   req.PerformedBy,
		Notes:         req.Notes,
	}

	if err := s.stockRepo.Save(ctx, mov); err != nil {
		return err
	}

	// §34: Audit the movement type — RECEIPT maps to AuditReceive, ADJUSTMENT to AuditAdjust
	if s.auditSvc != nil {
		action := domainaudit.AuditCreate // default for OPENING_BALANCE, PRODUCTION, etc.
		switch req.MovementType {
		case inventory.StockMovementPurchaseReceipt:
			action = domainaudit.AuditReceive
		case inventory.StockMovementAdjustment:
			action = domainaudit.AuditAdjust
		}
		movID := mov.ID
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     req.PerformedBy,
			Action:     action,
			EntityType: "stock_movement",
			EntityID:   &movID,
			NewValues: map[string]any{
				"warehouse_id":   req.WarehouseID.String(),
				"variant_id":    req.VariantID.String(),
				"quantity":       req.Quantity,
				"movement_type":  string(req.MovementType),
			},
		})
	}

	return nil
}

// TransferStock moves stock between two warehouses atomically (Rule 12).
// §34: TRANSFER action is audited on success.
func (s *InventoryService) TransferStock(ctx context.Context, fromWH, toWH, variantID, performedBy uuid.UUID, qty float64, notes string) error {
	if qty <= 0 {
		return fmt.Errorf("transfer quantity must be positive")
	}

	currentBal, err := s.stockRepo.GetStockBalance(ctx, fromWH, variantID)
	if err != nil {
		return fmt.Errorf("failed to get current balance: %w", err)
	}
	if currentBal-qty < 0 {
		return fmt.Errorf("insufficient stock for transfer")
	}

	variant, err := s.variantRepo.FindByID(ctx, variantID)
	if err != nil || variant == nil {
		return fmt.Errorf("invalid variant")
	}

	outMov := &inventory.StockMovement{
		ID:            uuid.New(),
		WarehouseID:   fromWH,
		VariantID:     variantID,
		Quantity:      decimal.NewFromFloat(-qty),
		UnitID:        variant.UnitID,
		MovementType:  inventory.StockMovementTransferOut,
		ReferenceType: "TRANSFER",
		PerformedBy:   performedBy,
		Notes:         notes,
	}

	inMov := &inventory.StockMovement{
		ID:            uuid.New(),
		WarehouseID:   toWH,
		VariantID:     variantID,
		Quantity:      decimal.NewFromFloat(qty),
		UnitID:        variant.UnitID,
		MovementType:  inventory.StockMovementTransferIn,
		ReferenceType: "TRANSFER",
		PerformedBy:   performedBy,
		Notes:         notes,
	}

	if err := s.stockRepo.ExecuteTransfer(ctx, outMov, inMov); err != nil {
		return err
	}

	// §34: Audit TRANSFER — fires after successful atomic stock movement
	if s.auditSvc != nil {
		outMovID := outMov.ID
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     performedBy,
			Action:     domainaudit.AuditTransfer,
			EntityType: "stock_transfer",
			EntityID:   &outMovID,
			NewValues: map[string]any{
				"from_warehouse": fromWH.String(),
				"to_warehouse":   toWH.String(),
				"variant_id":    variantID.String(),
				"quantity":       qty,
			},
		})
	}

	return nil
}
