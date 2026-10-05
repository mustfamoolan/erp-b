package handlers_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	appinv "m3aml-erp/internal/application/inventory"
	"m3aml-erp/internal/domain/inventory"
)

// ─────────────────────────────────────────────────────────────────────────────
// MOCK IMPLEMENTATIONS — for Inventory Phase 4 (§25, §26)
// ─────────────────────────────────────────────────────────────────────────────

type mockCategoryRepo struct {
	cats map[uuid.UUID]*inventory.ItemCategory
}

func newMockCategoryRepo() *mockCategoryRepo {
	return &mockCategoryRepo{cats: make(map[uuid.UUID]*inventory.ItemCategory)}
}
func (m *mockCategoryRepo) FindAll(ctx context.Context) ([]inventory.ItemCategory, error) { return nil, nil }
func (m *mockCategoryRepo) FindByID(ctx context.Context, id uuid.UUID) (*inventory.ItemCategory, error) {
	if c, ok := m.cats[id]; ok {
		return c, nil
	}
	return nil, nil
}
func (m *mockCategoryRepo) Save(ctx context.Context, c *inventory.ItemCategory) error {
	m.cats[c.ID] = c
	return nil
}
func (m *mockCategoryRepo) Update(ctx context.Context, c *inventory.ItemCategory) error { return nil }

type mockUnitRepo struct {
	units map[string]*inventory.UnitOfMeasure
}
func newMockUnitRepo() *mockUnitRepo {
	return &mockUnitRepo{units: make(map[string]*inventory.UnitOfMeasure)}
}
func (m *mockUnitRepo) FindAll(ctx context.Context) ([]inventory.UnitOfMeasure, error) { return nil, nil }
func (m *mockUnitRepo) FindByCode(ctx context.Context, code string) (*inventory.UnitOfMeasure, error) {
	if u, ok := m.units[code]; ok {
		return u, nil
	}
	return nil, nil
}
func (m *mockUnitRepo) Save(ctx context.Context, u *inventory.UnitOfMeasure) error {
	m.units[u.Code] = u
	return nil
}

type mockItemRepo struct {
	items map[uuid.UUID]*inventory.Item
}
func newMockItemRepo() *mockItemRepo {
	return &mockItemRepo{items: make(map[uuid.UUID]*inventory.Item)}
}
func (m *mockItemRepo) FindAll(ctx context.Context) ([]inventory.Item, error) { return nil, nil }
func (m *mockItemRepo) FindByID(ctx context.Context, id uuid.UUID) (*inventory.Item, error) { return m.items[id], nil }
func (m *mockItemRepo) FindByCode(ctx context.Context, code string) (*inventory.Item, error) { return nil, nil }
func (m *mockItemRepo) Save(ctx context.Context, item *inventory.Item) error {
	m.items[item.ID] = item
	return nil
}
func (m *mockItemRepo) Update(ctx context.Context, item *inventory.Item) error { return nil }

type mockVariantRepo struct {
	variants map[uuid.UUID]*inventory.ItemVariant
}
func newMockVariantRepo() *mockVariantRepo {
	return &mockVariantRepo{variants: make(map[uuid.UUID]*inventory.ItemVariant)}
}
func (m *mockVariantRepo) FindByItem(ctx context.Context, itemID uuid.UUID) ([]inventory.ItemVariant, error) { return nil, nil }
func (m *mockVariantRepo) FindByID(ctx context.Context, id uuid.UUID) (*inventory.ItemVariant, error) { return m.variants[id], nil }
func (m *mockVariantRepo) FindBySKU(ctx context.Context, sku string) (*inventory.ItemVariant, error) {
	for _, v := range m.variants {
		if v.SKU == sku {
			return v, nil
		}
	}
	return nil, nil
}
func (m *mockVariantRepo) FindByBarcode(ctx context.Context, barcode string) (*inventory.ItemVariant, error) {
	for _, v := range m.variants {
		if v.Barcode == barcode {
			return v, nil
		}
	}
	return nil, nil
}
func (m *mockVariantRepo) Save(ctx context.Context, v *inventory.ItemVariant) error {
	m.variants[v.ID] = v
	return nil
}
func (m *mockVariantRepo) Update(ctx context.Context, v *inventory.ItemVariant) error { return nil }

type mockWarehouseRepo struct{}
func (m *mockWarehouseRepo) FindAll(ctx context.Context) ([]inventory.Warehouse, error) { return nil, nil }
func (m *mockWarehouseRepo) FindByScope(ctx context.Context, scopeID uuid.UUID) ([]inventory.Warehouse, error) { return nil, nil }
func (m *mockWarehouseRepo) FindByID(ctx context.Context, id uuid.UUID) (*inventory.Warehouse, error) { return nil, nil }
func (m *mockWarehouseRepo) Save(ctx context.Context, warehouse *inventory.Warehouse) error { return nil }

type mockStockRepo struct{}
func (m *mockStockRepo) Save(ctx context.Context, movement *inventory.StockMovement) error { return nil }
func (m *mockStockRepo) FindByWarehouse(ctx context.Context, warehouseID uuid.UUID) ([]inventory.StockMovement, error) { return nil, nil }
func (m *mockStockRepo) GetStockBalance(ctx context.Context, warehouseID, variantID uuid.UUID) (float64, error) { return 0, nil }
func (m *mockStockRepo) ExecuteTransfer(ctx context.Context, outMovement, inMovement *inventory.StockMovement) error { return nil }

// ─────────────────────────────────────────────────────────────────────────────
// TESTS
// ─────────────────────────────────────────────────────────────────────────────

func TestInventoryDomain_ItemMasterDataCreation(t *testing.T) {
	catRepo := newMockCategoryRepo()
	unitRepo := newMockUnitRepo()
	itemRepo := newMockItemRepo()
	variantRepo := newMockVariantRepo()
	warehouseRepo := &mockWarehouseRepo{}
	stockRepo := &mockStockRepo{}

	invSvc := appinv.NewInventoryService(catRepo, unitRepo, itemRepo, variantRepo, warehouseRepo, stockRepo, nil)

	// 1. Setup Base Data
	cat, _ := invSvc.CreateCategory(context.Background(), "Raw Materials", nil)
	unitID := uuid.New()

	// 2. Create Base Item (Iron)
	item, err := invSvc.CreateItem(context.Background(), appinv.CreateItemRequest{
		CategoryID:  cat.ID,
		Code:        "RM-IRON",
		Name:        "Iron",
		Description: "Raw Iron Material",
		BaseUnitID:  unitID,
	})
	if err != nil {
		t.Fatalf("Failed to create item: %v", err)
	}

	// 3. Create Variant (Iron Sheet 3mm)
	variant1, err := invSvc.CreateVariant(context.Background(), appinv.CreateVariantRequest{
		ItemID:  item.ID,
		Name:    "Iron Sheet 3mm",
		SKU:     "RM-IRON-SHEET-3MM",
		Barcode: "1234567890123",
		UnitID:  unitID,
		Attributes: map[string]string{
			"Thickness": "3mm",
			"Width":     "120cm",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create variant: %v", err)
	}
	if variant1.SKU != "RM-IRON-SHEET-3MM" {
		t.Errorf("expected SKU RM-IRON-SHEET-3MM, got %s", variant1.SKU)
	}

	t.Log("✅ PASS: Base Item and Variant created successfully (Rule 25)")

	// 4. Test Duplicate SKU Rejection (Rule 26 implicitly unique)
	_, err = invSvc.CreateVariant(context.Background(), appinv.CreateVariantRequest{
		ItemID:  item.ID,
		Name:    "Another Iron Sheet",
		SKU:     "RM-IRON-SHEET-3MM", // Duplicate
		Barcode: "9999999999999",
		UnitID:  unitID,
	})
	if err == nil {
		t.Fatalf("Expected error when creating variant with duplicate SKU")
	}
	t.Log("✅ PASS: Duplicate SKU rejected (Rule 26)")

	// 5. Test Duplicate Barcode Rejection
	_, err = invSvc.CreateVariant(context.Background(), appinv.CreateVariantRequest{
		ItemID:  item.ID,
		Name:    "Another Iron Sheet",
		SKU:     "RM-IRON-SHEET-4MM",
		Barcode: "1234567890123", // Duplicate
		UnitID:  unitID,
	})
	if err == nil {
		t.Fatalf("Expected error when creating variant with duplicate Barcode")
	}
	t.Log("✅ PASS: Duplicate Barcode rejected (Rule 26)")
}
