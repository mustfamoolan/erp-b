package postgres

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"m3aml-erp/internal/domain/inventory"
	"m3aml-erp/internal/repositories"
)

// ─── ItemCategory Repository ──────────────────────────────────────────────────

type itemCategoryRepository struct {
	db *gorm.DB
}

func NewItemCategoryRepository(db *gorm.DB) repositories.ItemCategoryRepository {
	return &itemCategoryRepository{db: db}
}

func (r *itemCategoryRepository) FindAll(ctx context.Context) ([]inventory.ItemCategory, error) {
	var cats []inventory.ItemCategory
	err := r.db.WithContext(ctx).Order("name asc").Find(&cats).Error
	return cats, err
}

func (r *itemCategoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.ItemCategory, error) {
	var cat inventory.ItemCategory
	err := r.db.WithContext(ctx).First(&cat, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &cat, err
}

func (r *itemCategoryRepository) Save(ctx context.Context, cat *inventory.ItemCategory) error {
	return r.db.WithContext(ctx).Create(cat).Error
}

func (r *itemCategoryRepository) Update(ctx context.Context, cat *inventory.ItemCategory) error {
	return r.db.WithContext(ctx).Save(cat).Error
}

// ─── UnitOfMeasure Repository ─────────────────────────────────────────────────

type unitOfMeasureRepository struct {
	db *gorm.DB
}

func NewUnitOfMeasureRepository(db *gorm.DB) repositories.UnitOfMeasureRepository {
	return &unitOfMeasureRepository{db: db}
}

func (r *unitOfMeasureRepository) FindAll(ctx context.Context) ([]inventory.UnitOfMeasure, error) {
	var units []inventory.UnitOfMeasure
	err := r.db.WithContext(ctx).Order("code asc").Find(&units).Error
	return units, err
}

func (r *unitOfMeasureRepository) FindByCode(ctx context.Context, code string) (*inventory.UnitOfMeasure, error) {
	var unit inventory.UnitOfMeasure
	err := r.db.WithContext(ctx).First(&unit, "code = ?", code).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &unit, err
}

func (r *unitOfMeasureRepository) Save(ctx context.Context, unit *inventory.UnitOfMeasure) error {
	return r.db.WithContext(ctx).Create(unit).Error
}

// ─── Item Repository ──────────────────────────────────────────────────────────

type itemRepository struct {
	db *gorm.DB
}

func NewItemRepository(db *gorm.DB) repositories.ItemRepository {
	return &itemRepository{db: db}
}

func (r *itemRepository) FindAll(ctx context.Context) ([]inventory.Item, error) {
	var items []inventory.Item
	err := r.db.WithContext(ctx).Order("name asc").Preload("Variants").Preload("Variants.Attributes").Find(&items).Error
	return items, err
}

func (r *itemRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.Item, error) {
	var item inventory.Item
	err := r.db.WithContext(ctx).Preload("Variants").Preload("Variants.Attributes").First(&item, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &item, err
}

func (r *itemRepository) FindByCode(ctx context.Context, code string) (*inventory.Item, error) {
	var item inventory.Item
	err := r.db.WithContext(ctx).Preload("Variants").Preload("Variants.Attributes").First(&item, "code = ?", code).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &item, err
}

func (r *itemRepository) Save(ctx context.Context, item *inventory.Item) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *itemRepository) Update(ctx context.Context, item *inventory.Item) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// ─── ItemVariant Repository ───────────────────────────────────────────────────

type itemVariantRepository struct {
	db *gorm.DB
}

func NewItemVariantRepository(db *gorm.DB) repositories.ItemVariantRepository {
	return &itemVariantRepository{db: db}
}

func (r *itemVariantRepository) FindByItem(ctx context.Context, itemID uuid.UUID) ([]inventory.ItemVariant, error) {
	var vars []inventory.ItemVariant
	err := r.db.WithContext(ctx).Where("item_id = ?", itemID).Preload("Attributes").Order("name asc").Find(&vars).Error
	return vars, err
}

func (r *itemVariantRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.ItemVariant, error) {
	var variant inventory.ItemVariant
	err := r.db.WithContext(ctx).Preload("Attributes").First(&variant, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &variant, err
}

func (r *itemVariantRepository) FindBySKU(ctx context.Context, sku string) (*inventory.ItemVariant, error) {
	var variant inventory.ItemVariant
	err := r.db.WithContext(ctx).Preload("Attributes").First(&variant, "sku = ?", sku).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &variant, err
}

func (r *itemVariantRepository) FindByBarcode(ctx context.Context, barcode string) (*inventory.ItemVariant, error) {
	var variant inventory.ItemVariant
	err := r.db.WithContext(ctx).Preload("Attributes").First(&variant, "barcode = ?", barcode).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &variant, err
}

func (r *itemVariantRepository) Save(ctx context.Context, variant *inventory.ItemVariant) error {
	return r.db.WithContext(ctx).Create(variant).Error
}

func (r *itemVariantRepository) Update(ctx context.Context, variant *inventory.ItemVariant) error {
	return r.db.WithContext(ctx).Save(variant).Error
}

// ─── Warehouse Repository ─────────────────────────────────────────────────────

type warehouseRepository struct {
	db *gorm.DB
}

func NewWarehouseRepository(db *gorm.DB) repositories.WarehouseRepository {
	return &warehouseRepository{db: db}
}

func (r *warehouseRepository) FindAll(ctx context.Context) ([]inventory.Warehouse, error) {
	var whs []inventory.Warehouse
	err := r.db.WithContext(ctx).Order("name asc").Find(&whs).Error
	return whs, err
}

func (r *warehouseRepository) FindByScope(ctx context.Context, scopeID uuid.UUID) ([]inventory.Warehouse, error) {
	var whs []inventory.Warehouse
	err := r.db.WithContext(ctx).Where("scope_id = ?", scopeID).Order("name asc").Find(&whs).Error
	return whs, err
}

func (r *warehouseRepository) FindByID(ctx context.Context, id uuid.UUID) (*inventory.Warehouse, error) {
	var wh inventory.Warehouse
	err := r.db.WithContext(ctx).First(&wh, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &wh, err
}

func (r *warehouseRepository) Save(ctx context.Context, warehouse *inventory.Warehouse) error {
	return r.db.WithContext(ctx).Create(warehouse).Error
}

// ─── Stock Movement Repository ────────────────────────────────────────────────

type stockMovementRepository struct {
	db *gorm.DB
}

func NewStockMovementRepository(db *gorm.DB) repositories.StockMovementRepository {
	return &stockMovementRepository{db: db}
}

func (r *stockMovementRepository) Save(ctx context.Context, movement *inventory.StockMovement) error {
	return r.db.WithContext(ctx).Create(movement).Error
}

func (r *stockMovementRepository) FindByWarehouse(ctx context.Context, warehouseID uuid.UUID) ([]inventory.StockMovement, error) {
	var movs []inventory.StockMovement
	err := r.db.WithContext(ctx).Where("warehouse_id = ?", warehouseID).Order("created_at desc").Find(&movs).Error
	return movs, err
}

func (r *stockMovementRepository) GetStockBalance(ctx context.Context, warehouseID, variantID uuid.UUID) (float64, error) {
	var balance float64
	err := r.db.WithContext(ctx).
		Model(&inventory.StockMovement{}).
		Where("warehouse_id = ? AND variant_id = ?", warehouseID, variantID).
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&balance).Error
	return balance, err
}

func (r *stockMovementRepository) ExecuteTransfer(ctx context.Context, outMovement, inMovement *inventory.StockMovement) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(outMovement).Error; err != nil {
			return err
		}
		if err := tx.Create(inMovement).Error; err != nil {
			return err
		}
		return nil
	})
}

