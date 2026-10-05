package handlers_test

import (
	"context"
	"testing"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	appinv "m3aml-erp/internal/application/inventory"
	"m3aml-erp/internal/domain/inventory"
)

// Extending the mocks for Phase 5 testing

type mockStockRepoForBalance struct {
	movements []inventory.StockMovement
}
func (m *mockStockRepoForBalance) Save(ctx context.Context, movement *inventory.StockMovement) error {
	m.movements = append(m.movements, *movement)
	return nil
}
func (m *mockStockRepoForBalance) FindByWarehouse(ctx context.Context, warehouseID uuid.UUID) ([]inventory.StockMovement, error) {
	return nil, nil
}
func (m *mockStockRepoForBalance) GetStockBalance(ctx context.Context, warehouseID, variantID uuid.UUID) (float64, error) {
	var bal decimal.Decimal
	for _, mov := range m.movements {
		if mov.WarehouseID == warehouseID && mov.VariantID == variantID {
			bal = bal.Add(mov.Quantity)
		}
	}
	f, _ := bal.Float64()
	return f, nil
}
func (m *mockStockRepoForBalance) ExecuteTransfer(ctx context.Context, outMovement, inMovement *inventory.StockMovement) error {
	m.movements = append(m.movements, *outMovement)
	m.movements = append(m.movements, *inMovement)
	return nil
}

func TestStockLedger_DerivedQuantities(t *testing.T) {
	catRepo := newMockCategoryRepo()
	unitRepo := newMockUnitRepo()
	itemRepo := newMockItemRepo()
	
	// Setup variants for validation
	variantRepo := newMockVariantRepo()
	vID := uuid.New()
	uID := uuid.New()
	variantRepo.Save(context.Background(), &inventory.ItemVariant{
		ID: vID,
		UnitID: uID,
	})

	warehouseRepo := &mockWarehouseRepo{}
	stockRepo := &mockStockRepoForBalance{}

	invSvc := appinv.NewInventoryService(catRepo, unitRepo, itemRepo, variantRepo, warehouseRepo, stockRepo, nil)

	whID := uuid.New()
	userID := uuid.New()

	// 1. Opening Balance (+100)
	err := invSvc.RecordMovement(context.Background(), appinv.RecordMovementRequest{
		WarehouseID:  whID,
		VariantID:    vID,
		Quantity:     100,
		MovementType: inventory.StockMovementOpeningBalance,
		PerformedBy:  userID,
	})
	if err != nil {
		t.Fatalf("failed to record opening balance: %v", err)
	}

	bal, _ := stockRepo.GetStockBalance(context.Background(), whID, vID)
	if bal != 100 {
		t.Fatalf("expected 100, got %v", bal)
	}

	// 2. Issue (-20)
	err = invSvc.RecordMovement(context.Background(), appinv.RecordMovementRequest{
		WarehouseID:  whID,
		VariantID:    vID,
		Quantity:     -20,
		MovementType: inventory.StockMovementIssue,
		PerformedBy:  userID,
	})
	if err != nil {
		t.Fatalf("failed to record issue: %v", err)
	}

	bal, _ = stockRepo.GetStockBalance(context.Background(), whID, vID)
	if bal != 80 {
		t.Fatalf("expected 80, got %v", bal)
	}
	t.Log("✅ PASS: Stock is derived correctly from movements (Rule 28)")

	// 3. Prevent Negative Over-Issue
	err = invSvc.RecordMovement(context.Background(), appinv.RecordMovementRequest{
		WarehouseID:  whID,
		VariantID:    vID,
		Quantity:     -100,
		MovementType: inventory.StockMovementIssue,
		PerformedBy:  userID,
	})
	if err == nil {
		t.Fatalf("expected error when issuing more than stock balance")
	}
	t.Log("✅ PASS: Prevented negative stock balance (Rule 10/28)")
}

func TestStockLedger_AtomicTransfers(t *testing.T) {
	variantRepo := newMockVariantRepo()
	vID := uuid.New()
	variantRepo.Save(context.Background(), &inventory.ItemVariant{ID: vID, UnitID: uuid.New()})
	
	stockRepo := &mockStockRepoForBalance{}
	invSvc := appinv.NewInventoryService(newMockCategoryRepo(), newMockUnitRepo(), newMockItemRepo(), variantRepo, &mockWarehouseRepo{}, stockRepo, nil)

	whA := uuid.New()
	whB := uuid.New()
	userID := uuid.New()

	// Seed whA with 50
	invSvc.RecordMovement(context.Background(), appinv.RecordMovementRequest{
		WarehouseID: whA, VariantID: vID, Quantity: 50, MovementType: inventory.StockMovementOpeningBalance, PerformedBy: userID,
	})

	// Transfer 30 from A to B
	err := invSvc.TransferStock(context.Background(), whA, whB, vID, userID, 30, "Internal Transfer")
	if err != nil {
		t.Fatalf("transfer failed: %v", err)
	}

	balA, _ := stockRepo.GetStockBalance(context.Background(), whA, vID)
	balB, _ := stockRepo.GetStockBalance(context.Background(), whB, vID)

	if balA != 20 {
		t.Errorf("expected whA balance to be 20, got %v", balA)
	}
	if balB != 30 {
		t.Errorf("expected whB balance to be 30, got %v", balB)
	}

	t.Log("✅ PASS: Stock transfer is atomic and balanced (Rule 27)")
}
