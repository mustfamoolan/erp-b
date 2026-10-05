package repositories

import (
	"context"
	"m3aml-erp/internal/domain/inventory"

	"github.com/google/uuid"
)

// ItemCategoryRepository defines persistence for ItemCategory
type ItemCategoryRepository interface {
	FindAll(ctx context.Context) ([]inventory.ItemCategory, error)
	FindByID(ctx context.Context, id uuid.UUID) (*inventory.ItemCategory, error)
	Save(ctx context.Context, category *inventory.ItemCategory) error
	Update(ctx context.Context, category *inventory.ItemCategory) error
}

// UnitOfMeasureRepository defines persistence for UOM
type UnitOfMeasureRepository interface {
	FindAll(ctx context.Context) ([]inventory.UnitOfMeasure, error)
	FindByCode(ctx context.Context, code string) (*inventory.UnitOfMeasure, error)
	Save(ctx context.Context, unit *inventory.UnitOfMeasure) error
}

// ItemRepository defines persistence for base Items
type ItemRepository interface {
	FindAll(ctx context.Context) ([]inventory.Item, error)
	FindByID(ctx context.Context, id uuid.UUID) (*inventory.Item, error)
	FindByCode(ctx context.Context, code string) (*inventory.Item, error)
	Save(ctx context.Context, item *inventory.Item) error
	Update(ctx context.Context, item *inventory.Item) error
}

// ItemVariantRepository defines persistence for ItemVariants
type ItemVariantRepository interface {
	FindByItem(ctx context.Context, itemID uuid.UUID) ([]inventory.ItemVariant, error)
	FindByID(ctx context.Context, id uuid.UUID) (*inventory.ItemVariant, error)
	FindBySKU(ctx context.Context, sku string) (*inventory.ItemVariant, error)
	FindByBarcode(ctx context.Context, barcode string) (*inventory.ItemVariant, error)
	Save(ctx context.Context, variant *inventory.ItemVariant) error
	Update(ctx context.Context, variant *inventory.ItemVariant) error
}

// WarehouseRepository defines persistence for Warehouses
type WarehouseRepository interface {
	FindAll(ctx context.Context) ([]inventory.Warehouse, error)
	FindByScope(ctx context.Context, scopeID uuid.UUID) ([]inventory.Warehouse, error)
	FindByID(ctx context.Context, id uuid.UUID) (*inventory.Warehouse, error)
	Save(ctx context.Context, warehouse *inventory.Warehouse) error
}

// StockMovementRepository defines persistence for Stock Movements
type StockMovementRepository interface {
	Save(ctx context.Context, movement *inventory.StockMovement) error
	FindByWarehouse(ctx context.Context, warehouseID uuid.UUID) ([]inventory.StockMovement, error)
	GetStockBalance(ctx context.Context, warehouseID, variantID uuid.UUID) (float64, error)
	// ExecuteTransfer moves stock between two warehouses atomically
	ExecuteTransfer(ctx context.Context, outMovement, inMovement *inventory.StockMovement) error
}

