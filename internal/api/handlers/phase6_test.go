package handlers_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	appwf "m3aml-erp/internal/application/workflow"
	"m3aml-erp/internal/domain/workflow"
)

// ─────────────────────────────────────────────────────────────────────────────
// MOCK IMPLEMENTATIONS — Phase 6 (Workflow / Financial Request)
// ─────────────────────────────────────────────────────────────────────────────

type mockRequestRepo struct {
	requests map[uuid.UUID]*workflow.FinancialRequest
}

func newMockRequestRepo() *mockRequestRepo {
	return &mockRequestRepo{requests: make(map[uuid.UUID]*workflow.FinancialRequest)}
}
func (m *mockRequestRepo) Save(ctx context.Context, req *workflow.FinancialRequest) error {
	m.requests[req.ID] = req
	return nil
}
func (m *mockRequestRepo) Update(ctx context.Context, req *workflow.FinancialRequest) error {
	m.requests[req.ID] = req
	return nil
}
func (m *mockRequestRepo) FindByID(ctx context.Context, id uuid.UUID) (*workflow.FinancialRequest, error) {
	if r, ok := m.requests[id]; ok {
		return r, nil
	}
	return nil, nil
}
func (m *mockRequestRepo) FindByDocumentNumber(ctx context.Context, number string) (*workflow.FinancialRequest, error) {
	return nil, nil
}
func (m *mockRequestRepo) FindByScope(ctx context.Context, scopeID uuid.UUID) ([]workflow.FinancialRequest, error) {
	var reqs []workflow.FinancialRequest
	for _, r := range m.requests {
		if r.ScopeID == scopeID {
			reqs = append(reqs, *r)
		}
	}
	return reqs, nil
}
func (m *mockRequestRepo) FindAll(ctx context.Context) ([]workflow.FinancialRequest, error) {
	var reqs []workflow.FinancialRequest
	for _, r := range m.requests {
		reqs = append(reqs, *r)
	}
	return reqs, nil
}

type mockRequestItemRepo struct{}

func (m *mockRequestItemRepo) SaveAll(ctx context.Context, items []workflow.RequestItem) error {
	return nil
}
func (m *mockRequestItemRepo) FindByRequest(ctx context.Context, requestID uuid.UUID) ([]workflow.RequestItem, error) {
	return nil, nil
}

type mockRequestHistoryRepo struct {
	history []workflow.RequestHistory
}

func (m *mockRequestHistoryRepo) Save(ctx context.Context, h *workflow.RequestHistory) error {
	m.history = append(m.history, *h)
	return nil
}
func (m *mockRequestHistoryRepo) FindByRequest(ctx context.Context, requestID uuid.UUID) ([]workflow.RequestHistory, error) {
	var result []workflow.RequestHistory
	for _, h := range m.history {
		if h.RequestID == requestID {
			result = append(result, h)
		}
	}
	return result, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS — Phase 6
// ─────────────────────────────────────────────────────────────────────────────

func buildRequestService() (*appwf.RequestService, *mockRequestRepo, *mockRequestHistoryRepo) {
	reqRepo := newMockRequestRepo()
	itemRepo := &mockRequestItemRepo{}
	histRepo := &mockRequestHistoryRepo{}
	counter := 0
	docNumFn := func(ctx context.Context) (string, error) {
		counter++
		return "FIN-2026-000001", nil
	}
	svc := appwf.NewRequestService(reqRepo, itemRepo, histRepo, nil, docNumFn)
	return svc, reqRepo, histRepo
}

func TestFinancialRequest_Creation_WithDocumentNumber(t *testing.T) {
	svc, _, _ := buildRequestService()

	scopeID := uuid.New()
	factoryID := uuid.New()
	userID := uuid.New()

	req, err := svc.CreateRequest(context.Background(), appwf.CreateRequestInput{
		ScopeID:     scopeID,
		FactoryID:   factoryID,
		RequestedBy: userID,
		Type:        workflow.RequestTypeFinancial,
		Purpose:     "Purchase Raw Materials",
		Currency:    "SAR",
		Items: []appwf.CreateRequestItemInput{
			{
				Description:        "Iron Sheet 3mm",
				Quantity:           decimal.NewFromFloat(10),
				EstimatedUnitPrice: decimal.NewFromFloat(150),
			},
		},
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if req.DocumentNumber == "" {
		t.Errorf("expected document number to be generated (§33)")
	}
	if req.Status != workflow.RequestDraft {
		t.Errorf("expected DRAFT status, got: %s", req.Status)
	}
	if req.FactoryID != factoryID {
		t.Errorf("request must clearly identify the originating factory (§31)")
	}
	// Total = 10 * 150 = 1500
	if req.TotalAmount.String() != "1500" {
		t.Errorf("expected total 1500, got %s", req.TotalAmount.String())
	}
	t.Logf("✅ PASS: Request created with DocumentNumber=%s, Status=DRAFT, Total=1500 (§31, §33)", req.DocumentNumber)
}

func TestFinancialRequest_MustHaveItems(t *testing.T) {
	svc, _, _ := buildRequestService()

	_, err := svc.CreateRequest(context.Background(), appwf.CreateRequestInput{
		ScopeID:     uuid.New(),
		FactoryID:   uuid.New(),
		RequestedBy: uuid.New(),
		Type:        workflow.RequestTypeFinancial,
		Purpose:     "No items",
		Items:       []appwf.CreateRequestItemInput{},
	})
	if err == nil {
		t.Fatal("expected error for request with no items")
	}
	t.Log("✅ PASS: Request without items is rejected")
}

func TestFinancialRequest_FullWorkflow(t *testing.T) {
	svc, _, histRepo := buildRequestService()

	userID := uuid.New()

	// 1. Create in DRAFT
	req, err := svc.CreateRequest(context.Background(), appwf.CreateRequestInput{
		ScopeID:     uuid.New(),
		FactoryID:   uuid.New(),
		RequestedBy: userID,
		Type:        workflow.RequestTypeFinancial,
		Purpose:     "E2E Workflow Test",
		Items: []appwf.CreateRequestItemInput{
			{Description: "Item A", Quantity: decimal.NewFromFloat(5), EstimatedUnitPrice: decimal.NewFromFloat(100)},
		},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	t.Logf("✅ PASS: Step 1 — DRAFT created")

	transitions := []struct {
		action string
		to     workflow.RequestStatus
	}{
		{"SUBMIT", workflow.RequestSubmitted},
		{"START_REVIEW", workflow.RequestUnderReview},
		{"APPROVE", workflow.RequestReviewApproved},
		{"ACCOUNTANT_APPROVE", workflow.RequestAccountantApproved},
		{"MARK_PAYMENT_PENDING", workflow.RequestPaymentPending},
		{"MARK_FACTORY_RECEIPT_PENDING", workflow.RequestFactoryReceiptPend},
		{"RECEIVE", workflow.RequestReceived},
		{"COMPLETE", workflow.RequestCompleted},
	}

	for _, tr := range transitions {
		req, err = svc.Transition(context.Background(), appwf.TransitionInput{
			RequestID:   req.ID,
			PerformedBy: userID,
			NewStatus:   tr.to,
			Action:      tr.action,
		})
		if err != nil {
			t.Fatalf("transition %s → %s failed: %v", tr.action, tr.to, err)
		}
		if req.Status != tr.to {
			t.Errorf("expected status %s, got %s", tr.to, req.Status)
		}
		t.Logf("✅ PASS: Step — %s → %s", tr.action, tr.to)
	}

	// Verify all transitions are audited in history (Rule 17)
	history, _ := histRepo.FindByRequest(context.Background(), req.ID)
	// 1 CREATE + 8 transitions = 9 history entries
	if len(history) < 9 {
		t.Errorf("expected at least 9 history entries, got %d (Rule 17 audit trail)", len(history))
	}
	t.Logf("✅ PASS: All transitions audited — %d history entries (Rule 17)", len(history))
}

func TestFinancialRequest_InvalidTransition_Rejected(t *testing.T) {
	svc, _, _ := buildRequestService()

	req, _ := svc.CreateRequest(context.Background(), appwf.CreateRequestInput{
		ScopeID:     uuid.New(),
		FactoryID:   uuid.New(),
		RequestedBy: uuid.New(),
		Type:        workflow.RequestTypeFinancial,
		Purpose:     "Test",
		Items: []appwf.CreateRequestItemInput{
			{Description: "Item", Quantity: decimal.NewFromFloat(1), EstimatedUnitPrice: decimal.NewFromFloat(100)},
		},
	})

	// Try to jump directly from DRAFT to APPROVED (invalid) — §32 state machine
	_, err := svc.Transition(context.Background(), appwf.TransitionInput{
		RequestID:   req.ID,
		PerformedBy: uuid.New(),
		NewStatus:   workflow.RequestReviewApproved,
		Action:      "APPROVE",
	})
	if err == nil {
		t.Fatal("expected error for invalid transition DRAFT → REVIEW_APPROVED")
	}
	t.Log("✅ PASS: Invalid transition DRAFT → REVIEW_APPROVED rejected (§32 state machine)")
}

func TestFinancialRequest_ScopeIsolation(t *testing.T) {
	svc, _, _ := buildRequestService()

	factoryAScope := uuid.New()
	factoryBScope := uuid.New()
	userID := uuid.New()

	// Create 2 requests for Factory A and 1 for Factory B
	svc.CreateRequest(context.Background(), appwf.CreateRequestInput{
		ScopeID:   factoryAScope,
		FactoryID: uuid.New(),
		RequestedBy: userID,
		Type:        workflow.RequestTypeFinancial,
		Purpose: "Factory A Request 1",
		Items: []appwf.CreateRequestItemInput{{Description: "i", Quantity: decimal.NewFromFloat(1), EstimatedUnitPrice: decimal.NewFromFloat(10)}},
	})
	svc.CreateRequest(context.Background(), appwf.CreateRequestInput{
		ScopeID:   factoryAScope,
		FactoryID: uuid.New(),
		RequestedBy: userID,
		Type:        workflow.RequestTypeFinancial,
		Purpose: "Factory A Request 2",
		Items: []appwf.CreateRequestItemInput{{Description: "i", Quantity: decimal.NewFromFloat(1), EstimatedUnitPrice: decimal.NewFromFloat(10)}},
	})
	svc.CreateRequest(context.Background(), appwf.CreateRequestInput{
		ScopeID:   factoryBScope,
		FactoryID: uuid.New(),
		RequestedBy: userID,
		Type:        workflow.RequestTypeFinancial,
		Purpose: "Factory B Request",
		Items: []appwf.CreateRequestItemInput{{Description: "i", Quantity: decimal.NewFromFloat(1), EstimatedUnitPrice: decimal.NewFromFloat(10)}},
	})

	aRequests, _ := svc.GetByScope(context.Background(), factoryAScope)
	bRequests, _ := svc.GetByScope(context.Background(), factoryBScope)

	if len(aRequests) != 2 {
		t.Errorf("expected 2 requests for Factory A, got %d", len(aRequests))
	}
	if len(bRequests) != 1 {
		t.Errorf("expected 1 request for Factory B, got %d", len(bRequests))
	}
	t.Log("✅ PASS: Factory A cannot see Factory B requests — scope isolation (Rule 7)")
}
