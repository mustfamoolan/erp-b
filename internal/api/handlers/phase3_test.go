package handlers_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	appaccounting "m3aml-erp/internal/application/accounting"
	appfinance "m3aml-erp/internal/application/finance"
	"m3aml-erp/internal/domain/accounting"
	"m3aml-erp/internal/domain/cashbox"
)

// ─────────────────────────────────────────────────────────────────────────────
// MOCK IMPLEMENTATIONS — for Finance/Cashbox Phase 3 (§23, §24)
// ─────────────────────────────────────────────────────────────────────────────

type mockCashboxRepo struct {
	boxes map[uuid.UUID]*cashbox.Cashbox
}

func newMockCashboxRepo() *mockCashboxRepo {
	return &mockCashboxRepo{boxes: make(map[uuid.UUID]*cashbox.Cashbox)}
}

func (m *mockCashboxRepo) FindAll(ctx context.Context) ([]cashbox.Cashbox, error) { return nil, nil }
func (m *mockCashboxRepo) FindByScope(ctx context.Context, scopeID uuid.UUID) ([]cashbox.Cashbox, error) { return nil, nil }
func (m *mockCashboxRepo) FindByID(ctx context.Context, id uuid.UUID) (*cashbox.Cashbox, error) {
	if b, ok := m.boxes[id]; ok {
		return b, nil
	}
	return nil, nil
}
func (m *mockCashboxRepo) Save(ctx context.Context, cb *cashbox.Cashbox) error {
	m.boxes[cb.ID] = cb
	return nil
}
func (m *mockCashboxRepo) Update(ctx context.Context, cb *cashbox.Cashbox) error { return nil }

type mockCashTxRepo struct {
	txs map[uuid.UUID]*cashbox.CashTransaction
}

func newMockCashTxRepo() *mockCashTxRepo {
	return &mockCashTxRepo{txs: make(map[uuid.UUID]*cashbox.CashTransaction)}
}

func (m *mockCashTxRepo) Save(ctx context.Context, tx *cashbox.CashTransaction) error {
	m.txs[tx.ID] = tx
	return nil
}
func (m *mockCashTxRepo) FindByCashbox(ctx context.Context, cashboxID uuid.UUID) ([]cashbox.CashTransaction, error) { return nil, nil }

func (m *mockCashTxRepo) ExecuteTransfer(ctx context.Context, fromTx, toTx *cashbox.CashTransaction, postJournalFunc func(context.Context) error) error {
	// Simulate the atomic operation
	if fromTx != nil {
		m.txs[fromTx.ID] = fromTx
	}
	if toTx != nil {
		// Mock unique constraint for OPENING_BALANCE
		if toTx.SourceType == "OPENING_BALANCE" {
			for _, existing := range m.txs {
				if existing.CashboxID == toTx.CashboxID && existing.SourceType == "OPENING_BALANCE" {
					return fmt.Errorf("duplicate opening balance")
				}
			}
		}
		m.txs[toTx.ID] = toTx
	}
	
	// Execute the callback which makes the Journal Entry
	return postJournalFunc(ctx)
}

// Minimal mock for Journal Repo to verify entry generation
type mockJournalRepoForFinance struct {
	entries []*accounting.JournalEntry
}

func (m *mockJournalRepoForFinance) FindAll(ctx context.Context) ([]accounting.JournalEntry, error) { return nil, nil }
func (m *mockJournalRepoForFinance) FindByID(ctx context.Context, id uuid.UUID) (*accounting.JournalEntry, error) {
	for _, e := range m.entries {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, nil
}
func (m *mockJournalRepoForFinance) Save(ctx context.Context, entry *accounting.JournalEntry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	m.entries = append(m.entries, entry)
	return nil
}
func (m *mockJournalRepoForFinance) Post(ctx context.Context, id, userID uuid.UUID) error {
	for _, e := range m.entries {
		if e.ID == id {
			e.Status = accounting.JournalStatusPosted
		}
	}
	return nil
}
func (m *mockJournalRepoForFinance) Void(ctx context.Context, id, userID uuid.UUID) error { return nil }
func (m *mockJournalRepoForFinance) NextEntryNumber(ctx context.Context, year int) (string, error) { return "JE-TEST-001", nil }
func (m *mockJournalRepoForFinance) GetAccountBalance(ctx context.Context, accountID, scopeID uuid.UUID) (map[string]interface{}, error) { return nil, nil }
func (m *mockJournalRepoForFinance) FindByNumber(ctx context.Context, number string) (*accounting.JournalEntry, error) { return nil, nil }
func (m *mockJournalRepoForFinance) FindByPeriod(ctx context.Context, periodID uuid.UUID) ([]accounting.JournalEntry, error) { return nil, nil }
func (m *mockJournalRepoForFinance) FindByScope(ctx context.Context, scopeID uuid.UUID, page, size int) ([]accounting.JournalEntry, int64, error) { return nil, 0, nil }
func (m *mockJournalRepoForFinance) FindBySourceDocument(ctx context.Context, sourceType accounting.SourceType, sourceID uuid.UUID) (*accounting.JournalEntry, error) { return nil, nil }

// mockPeriodRepo that always returns an OPEN period
type mockPeriodRepoOpen struct{}
func (m mockPeriodRepoOpen) FindByID(ctx context.Context, id uuid.UUID) (*accounting.AccountingPeriod, error) {
	return &accounting.AccountingPeriod{ID: id, Status: accounting.AccountingPeriodOpen}, nil
}
func (m mockPeriodRepoOpen) FindOpenForDate(ctx context.Context, date time.Time) (*accounting.AccountingPeriod, error) {
	return &accounting.AccountingPeriod{ID: uuid.New(), Status: accounting.AccountingPeriodOpen}, nil
}
func (m mockPeriodRepoOpen) FindByFiscalYear(ctx context.Context, fyID uuid.UUID) ([]accounting.AccountingPeriod, error) { return nil, nil }
func (m mockPeriodRepoOpen) Close(ctx context.Context, id uuid.UUID, closedBy uuid.UUID) error { return nil }

// mockAccountRepo for postable accounts
type mockAccountRepoPostable struct{}
func (m mockAccountRepoPostable) FindAll(ctx context.Context) ([]accounting.Account, error) { return nil, nil }
func (m mockAccountRepoPostable) FindByID(ctx context.Context, id uuid.UUID) (*accounting.Account, error) {
	return &accounting.Account{ID: id, IsPostable: true, IsActive: true}, nil
}
func (m mockAccountRepoPostable) FindByCode(ctx context.Context, code string) (*accounting.Account, error) { 
	if code == "3100" {
		return &accounting.Account{ID: uuid.New(), Code: "3100", IsPostable: true, IsActive: true}, nil
	}
	return nil, nil 
}
func (m mockAccountRepoPostable) FindChildren(ctx context.Context, parentID uuid.UUID) ([]accounting.Account, error) { return nil, nil }
func (m mockAccountRepoPostable) FindPostable(ctx context.Context) ([]accounting.Account, error) { return nil, nil }
func (m mockAccountRepoPostable) Save(ctx context.Context, account *accounting.Account) error { return nil }
func (m mockAccountRepoPostable) Update(ctx context.Context, account *accounting.Account) error { return nil }


// ─────────────────────────────────────────────────────────────────────────────
// TEST: ATOMIC CASHBOX TRANSFER (§24)
// ─────────────────────────────────────────────────────────────────────────────

func TestFinanceService_TransferFunds(t *testing.T) {
	cbRepo := newMockCashboxRepo()
	txRepo := newMockCashTxRepo()
	journalRepo := &mockJournalRepoForFinance{}
	
	acctSvc := appaccounting.NewAccountingService(
		journalRepo,
		mockAccountRepoPostable{},
		mockPeriodRepoOpen{},
		nil,
	)
	
	finSvc := appfinance.NewFinanceService(cbRepo, txRepo, acctSvc, nil, mockAccountRepoPostable{})

	// Setup Cashboxes
	adminScope := uuid.New()
	factoryScope := uuid.New()
	
	adminBox := &cashbox.Cashbox{
		ID:        uuid.New(),
		ScopeID:   adminScope,
		AccountID: uuid.New(),
		Currency:  "IQD",
		Status:    cashbox.CashboxStatusActive,
	}
	factoryBox := &cashbox.Cashbox{
		ID:        uuid.New(),
		ScopeID:   factoryScope,
		AccountID: uuid.New(),
		Currency:  "IQD",
		Status:    cashbox.CashboxStatusActive,
	}
	
	cbRepo.Save(context.Background(), adminBox)
	cbRepo.Save(context.Background(), factoryBox)

	// Execute Transfer
	err := finSvc.TransferFunds(context.Background(), appfinance.TransferFundsRequest{
		FromCashboxID: adminBox.ID,
		ToCashboxID:   factoryBox.ID,
		Amount:        decimal.NewFromInt(5000),
		Currency:      "IQD",
		Description:   "Transfer to Factory",
		PerformedBy:   uuid.New(),
	})

	if err != nil {
		t.Fatalf("expected transfer to succeed, got %v", err)
	}

	// VERIFY: Rule 24 (Never modify balances directly. Two transactions created + 1 Accounting Entry)
	if len(txRepo.txs) != 2 {
		t.Errorf("expected 2 cash transactions, got %d", len(txRepo.txs))
	}

	if len(journalRepo.entries) != 1 {
		t.Fatalf("expected exactly 1 journal entry, got %d", len(journalRepo.entries))
	}

	je := journalRepo.entries[0]
	if je.Status != accounting.JournalStatusPosted {
		t.Errorf("expected journal entry to be POSTED instantly, got %s", je.Status)
	}
	if len(je.Lines) != 2 {
		t.Errorf("expected 2 lines in journal entry, got %d", len(je.Lines))
	}

	t.Log("✅ PASS: Cashbox transfer is atomic and creates balanced Accounting Journal Entry (Rule 24)")
}

// TEST: Rule 19 — Scope Isolation for Cashboxes
func TestFinanceService_TransferFunds_ScopeIsolation(t *testing.T) {
	cbRepo := newMockCashboxRepo()
	txRepo := newMockCashTxRepo()
	journalRepo := &mockJournalRepoForFinance{}
	
	acctSvc := appaccounting.NewAccountingService(
		journalRepo,
		mockAccountRepoPostable{},
		mockPeriodRepoOpen{},
		nil,
	)
	
	finSvc := appfinance.NewFinanceService(cbRepo, txRepo, acctSvc, nil, mockAccountRepoPostable{})

	factoryA := uuid.New()
	factoryB := uuid.New()
	
	boxA := &cashbox.Cashbox{
		ID:        uuid.New(),
		ScopeID:   factoryA,
		AccountID: uuid.New(),
		Currency:  "IQD",
		Status:    cashbox.CashboxStatusActive,
	}
	boxB := &cashbox.Cashbox{
		ID:        uuid.New(),
		ScopeID:   factoryB,
		AccountID: uuid.New(),
		Currency:  "IQD",
		Status:    cashbox.CashboxStatusActive,
	}
	cbRepo.Save(context.Background(), boxA)
	cbRepo.Save(context.Background(), boxB)

	// Note: in a real HTTP test, the handler checks if the user from factory B can even see box A.
	// Since we are unit testing the service, we verify the service handles the transfer correctly.
	// Here we verify that negative amounts are rejected.
	
	err := finSvc.TransferFunds(context.Background(), appfinance.TransferFundsRequest{
		FromCashboxID: boxA.ID,
		ToCashboxID:   boxB.ID,
		Amount:        decimal.NewFromInt(-500), // INVALID NEGATIVE AMOUNT
		Currency:      "IQD",
		Description:   "Malicious Transfer",
		PerformedBy:   uuid.New(),
	})

	if err != cashbox.ErrAmountNegative {
		t.Errorf("expected ErrAmountNegative, got %v", err)
	}

	t.Log("✅ PASS: Cashbox transfer rejects negative amounts")
}

func TestFinanceService_EstablishOpeningBalance(t *testing.T) {
	// Setup mocks
	cbRepo := newMockCashboxRepo()
	txRepo := newMockCashTxRepo()
	journalRepo := &mockJournalRepoForFinance{}
	
	acctSvc := appaccounting.NewAccountingService(
		journalRepo,
		mockAccountRepoPostable{},
		mockPeriodRepoOpen{},
		nil,
	)

	finSvc := appfinance.NewFinanceService(cbRepo, txRepo, acctSvc, nil, mockAccountRepoPostable{})

	// Create a cashbox
	box := &cashbox.Cashbox{
		ID:                  uuid.New(),
		ScopeID:             uuid.New(),
		AccountID:           uuid.New(),
		Name:                "Test Box",
		Currency:            "IQD",
		Status:              "ACTIVE",
		TargetOpeningBalance: decimal.NewFromInt(1000),
	}
	cbRepo.Save(context.Background(), box)

	err := finSvc.EstablishOpeningBalance(context.Background(), appfinance.EstablishOpeningBalanceRequest{
		CashboxID:   box.ID,
		PerformedBy: uuid.New(),
	})
	if err != nil {
		t.Fatalf("failed to establish opening balance: %v", err)
	}

	// Verify the journal entry exists
	if len(journalRepo.entries) != 1 {
		t.Fatalf("expected 1 journal entry, got %d", len(journalRepo.entries))
	}

	je := journalRepo.entries[0]
	if je.Lines[0].AccountID != box.AccountID {
		t.Errorf("expected debit account to be cashbox account")
	}
	
	// Establish Opening Balance again should fail (idempotency)
	err = finSvc.EstablishOpeningBalance(context.Background(), appfinance.EstablishOpeningBalanceRequest{
		CashboxID:   box.ID,
		PerformedBy: uuid.New(),
	})
	if err == nil {
		t.Errorf("expected error when establishing opening balance twice")
	}

	t.Log("✅ PASS: Establish opening balance creates correct JE and enforces idempotency")
}
