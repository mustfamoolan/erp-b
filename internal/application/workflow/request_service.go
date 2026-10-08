package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"m3aml-erp/bootstrap"
	appaudit "m3aml-erp/internal/application/audit"
	domainaudit "m3aml-erp/internal/domain/audit"
	"m3aml-erp/internal/domain/workflow"
	"m3aml-erp/internal/repositories"
)

// RequestService handles all business logic for Financial/Material Requests.
// Roadmap §31, §32, §33 — full workflow state machine.
// §34 — Financial requests are strongly auditable (CREATE, SUBMIT, APPROVE, REJECT, PAY, RECEIVE, CANCEL).
type RequestService struct {
	requestRepo repositories.FinancialRequestRepository
	itemRepo    repositories.RequestItemRepository
	historyRepo repositories.RequestHistoryRepository
	auditSvc    *appaudit.AuditService
	// docNumberFn generates the document number (injected to allow testability)
	docNumberFn func(ctx context.Context) (string, error)
}

func NewRequestService(
	requestRepo repositories.FinancialRequestRepository,
	itemRepo repositories.RequestItemRepository,
	historyRepo repositories.RequestHistoryRepository,
	auditSvc *appaudit.AuditService,
	docNumberFn func(ctx context.Context) (string, error),
) *RequestService {
	return &RequestService{
		requestRepo: requestRepo,
		itemRepo:    itemRepo,
		historyRepo: historyRepo,
		auditSvc:    auditSvc,
		docNumberFn: docNumberFn,
	}
}

// ─── Create Request ───────────────────────────────────────────────────────────

type CreateRequestInput struct {
	ScopeID           uuid.UUID
	FactoryID         uuid.UUID
	RequestedBy       uuid.UUID
	Type              workflow.RequestType
	RequestTypeID        *uuid.UUID
	ExpenseCategoryID    *uuid.UUID
	FactoryExpenseTypeID *uuid.UUID
	SupplierName      *string
	ReceiverName      *string
	ProjectName       *string
	Attachments       *string // JSON string
	ReceivingMethod   *string
	PurchaseNumber    *string
	WorkType          *string
	BarcodeSKU        *string
	ExchangeRate      *decimal.Decimal
	OriginalAmount    *decimal.Decimal
	AdvanceSequenceNumber *int
	ReceivingLocation     *string
	ReceiverPhone         *string
	Purpose           string
	Description       string
	Currency          string
	Items             []CreateRequestItemInput
}

type CreateRequestItemInput struct {
	VariantID          *uuid.UUID
	ExpenseCategoryID  *uuid.UUID
	Description        string
	Quantity           decimal.Decimal
	UnitID             *uuid.UUID
	EstimatedUnitPrice decimal.Decimal
	Notes              string
	ReceiptNumber      *string
}

// CreateRequest creates a new request in DRAFT status with generated document number and barcode.
func (s *RequestService) CreateRequest(ctx context.Context, input CreateRequestInput) (*workflow.FinancialRequest, error) {
	if input.Purpose == "" {
		return nil, fmt.Errorf("purpose is required")
	}
	if input.ScopeID == uuid.Nil || input.FactoryID == uuid.Nil {
		return nil, fmt.Errorf("scope_id and factory_id are required")
	}
	if len(input.Items) == 0 {
		return nil, fmt.Errorf("request must have at least one item")
	}

	docNum, err := s.docNumberFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate document number: %w", err)
	}

	barcode := docNum

	var total decimal.Decimal
	for _, item := range input.Items {
		total = total.Add(item.EstimatedUnitPrice.Mul(item.Quantity))
	}
	if total.IsZero() && input.OriginalAmount != nil && !input.OriginalAmount.IsZero() {
		total = *input.OriginalAmount
	}

	currency := input.Currency
	if currency == "" {
		currency = "SAR"
	}

	var advSeq *int
	if input.AdvanceSequenceNumber != nil && *input.AdvanceSequenceNumber > 0 {
		advSeq = input.AdvanceSequenceNumber
	} else {
		next, err := s.requestRepo.GetNextAdvanceSequence(ctx, input.FactoryID)
		if err == nil {
			advSeq = &next
		}
	}

	reqID := uuid.New()
	req := &workflow.FinancialRequest{
		ID:                reqID,
		DocumentNumber:    docNum,
		Barcode:           barcode,
		ScopeID:           input.ScopeID,
		FactoryID:         input.FactoryID,
		RequestedBy:       input.RequestedBy,
		Type:              input.Type,
		RequestTypeID:        input.RequestTypeID,
		ExpenseCategoryID:    input.ExpenseCategoryID,
		FactoryExpenseTypeID: input.FactoryExpenseTypeID,
		SupplierName:      input.SupplierName,
		ReceiverName:      input.ReceiverName,
		ProjectName:       input.ProjectName,
		Attachments:       input.Attachments,
		ReceivingMethod:   input.ReceivingMethod,
		PurchaseNumber:    input.PurchaseNumber,
		WorkType:          input.WorkType,
		BarcodeSKU:        input.BarcodeSKU,
		ExchangeRate:      input.ExchangeRate,
		OriginalAmount:    input.OriginalAmount,
		AdvanceSequenceNumber: advSeq,
		ReceivingLocation:     input.ReceivingLocation,
		ReceiverPhone:         input.ReceiverPhone,
		Purpose:           input.Purpose,
		Description:       input.Description,
		TotalAmount:       total,
		Currency:          currency,
		Status:            workflow.RequestSubmitted,
	}

	if err := s.requestRepo.Save(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to save request: %w", err)
	}

	var items []workflow.RequestItem
	for _, i := range input.Items {
		lineTotal := i.EstimatedUnitPrice.Mul(i.Quantity)
		items = append(items, workflow.RequestItem{
			ID:                 uuid.New(),
			RequestID:          reqID,
			VariantID:          i.VariantID,
			ExpenseCategoryID:  i.ExpenseCategoryID,
			Description:        i.Description,
			Quantity:           i.Quantity,
			UnitID:             i.UnitID,
			EstimatedUnitPrice: i.EstimatedUnitPrice,
			EstimatedTotal:     lineTotal,
			Notes:              i.Notes,
			ReceiptNumber:      i.ReceiptNumber,
		})
	}
	if err := s.itemRepo.SaveAll(ctx, items); err != nil {
		return nil, fmt.Errorf("failed to save request items: %w", err)
	}

	// Local history
	s.appendHistory(ctx, reqID, "", workflow.RequestSubmitted, "SUBMIT", input.RequestedBy, "تم إنشاء الطلب وتقديمه للتدقيق الإداري")

	// Global Audit — §34
	if s.auditSvc != nil {
		scope := input.ScopeID
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     input.RequestedBy,
			ScopeID:    &scope,
			Action:     domainaudit.AuditSubmit,
			EntityType: "financial_request",
			EntityID:   &reqID,
			NewValues: map[string]any{
				"document_number": docNum,
				"total_amount":    total.String(),
				"purpose":         input.Purpose,
				"status":          "SUBMITTED",
			},
		})
	}

	bootstrap.InvalidateRequestsCache(input.ScopeID.String())

	return req, nil
}

// ─── Workflow Transitions ─────────────────────────────────────────────────────

type TransitionInput struct {
	RequestID   uuid.UUID
	PerformedBy uuid.UUID
	NewStatus   workflow.RequestStatus
	Action      string
	Notes       string
}

// Transition validates and applies a status change (§32 state machine).
func (s *RequestService) Transition(ctx context.Context, input TransitionInput) (*workflow.FinancialRequest, error) {
	req, err := s.requestRepo.FindByID(ctx, input.RequestID)
	if err != nil || req == nil {
		return nil, fmt.Errorf("request not found")
	}

	// Mandatory note enforcement for rejection and revision
	if input.NewStatus == workflow.RequestReturnedForRevision || 
	   input.NewStatus == workflow.RequestRejected || 
	   input.NewStatus == workflow.RequestReviewRejected || 
	   input.NewStatus == workflow.RequestAccountantRejected {
		if strings.TrimSpace(input.Notes) == "" {
			return nil, fmt.Errorf("notes explaining the reason are required for rejection or revision")
		}
	}

	if !req.Status.CanTransitionTo(input.NewStatus) {
		return nil, fmt.Errorf("invalid transition: %s → %s", req.Status, input.NewStatus)
	}

	from := req.Status
	req.Status = input.NewStatus

	if err := s.requestRepo.Update(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to update request status: %w", err)
	}

	// Local history
	s.appendHistory(ctx, req.ID, from, input.NewStatus, input.Action, input.PerformedBy, input.Notes)

	// Global Audit — §34 mapping Action
	if s.auditSvc != nil {
		var auditAction domainaudit.AuditAction
		switch input.Action {
		case "SUBMIT":
			auditAction = domainaudit.AuditSubmit
		case "APPROVE", "ACCOUNTANT_APPROVE":
			auditAction = domainaudit.AuditApprove
		case "REJECT", "ACCOUNTANT_REJECT":
			auditAction = domainaudit.AuditReject
		case "RETURN_REVISION", "RETURN_FOR_REVISION":
			auditAction = domainaudit.AuditReject
		case "DISBURSE", "MARK_PAYMENT_PENDING":
			auditAction = domainaudit.AuditPay
		case "DELIVER", "RECEIVE", "MARK_FACTORY_RECEIPT_PENDING":
			auditAction = domainaudit.AuditReceive
		case "CANCEL":
			auditAction = domainaudit.AuditVoid
		default:
			auditAction = domainaudit.AuditUpdate
		}

		scope := req.ScopeID
		reqID := req.ID
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     input.PerformedBy,
			ScopeID:    &scope,
			Action:     auditAction,
			EntityType: "financial_request",
			EntityID:   &reqID,
			OldValues:  map[string]any{"status": string(from)},
			NewValues:  map[string]any{"status": string(input.NewStatus), "notes": input.Notes},
		})
	}

	bootstrap.InvalidateRequestsCache(req.ScopeID.String())

	return req, nil
}

// Disburse transitions request to DISBURSED and stamps DisbursedAt.
func (s *RequestService) Disburse(ctx context.Context, requestID, performedBy uuid.UUID, notes string) (*workflow.FinancialRequest, error) {
	req, err := s.requestRepo.FindByID(ctx, requestID)
	if err != nil || req == nil {
		return nil, fmt.Errorf("request not found")
	}

	if !req.Status.CanTransitionTo(workflow.RequestDisbursed) {
		return nil, fmt.Errorf("cannot disburse request in status: %s", req.Status)
	}

	from := req.Status
	now := time.Now()
	req.Status = workflow.RequestDisbursed
	req.DisbursedAt = &now

	if err := s.requestRepo.Update(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to update request status: %w", err)
	}

	s.appendHistory(ctx, req.ID, from, workflow.RequestDisbursed, "DISBURSE", performedBy, notes)

	if s.auditSvc != nil {
		scope := req.ScopeID
		reqID := req.ID
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     performedBy,
			ScopeID:    &scope,
			Action:     domainaudit.AuditPay,
			EntityType: "financial_request",
			EntityID:   &reqID,
			OldValues:  map[string]any{"status": string(from)},
			NewValues:  map[string]any{"status": string(workflow.RequestDisbursed), "notes": notes},
		})
	}

	bootstrap.InvalidateRequestsCache(req.ScopeID.String())

	return req, nil
}

// Deliver stamps receiver_name, signed_voucher_url, delivered_at and transitions to DELIVERED.
func (s *RequestService) Deliver(ctx context.Context, requestID, performedBy uuid.UUID, receiverName, signedVoucherURL, notes string) (*workflow.FinancialRequest, error) {
	if strings.TrimSpace(receiverName) == "" {
		return nil, fmt.Errorf("receiver name is required")
	}
	if strings.TrimSpace(signedVoucherURL) == "" {
		return nil, fmt.Errorf("signed delivery voucher attachment is required")
	}

	req, err := s.requestRepo.FindByID(ctx, requestID)
	if err != nil || req == nil {
		return nil, fmt.Errorf("request not found")
	}

	if !req.Status.CanTransitionTo(workflow.RequestDelivered) {
		return nil, fmt.Errorf("cannot deliver request in status: %s", req.Status)
	}

	from := req.Status
	now := time.Now()
	req.Status = workflow.RequestDelivered
	req.ReceiverName = &receiverName
	req.SignedVoucherURL = &signedVoucherURL
	req.DeliveredAt = &now

	if err := s.requestRepo.Update(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to update request status: %w", err)
	}

	s.appendHistory(ctx, req.ID, from, workflow.RequestDelivered, "DELIVER", performedBy, notes)

	if s.auditSvc != nil {
		scope := req.ScopeID
		reqID := req.ID
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     performedBy,
			ScopeID:    &scope,
			Action:     domainaudit.AuditReceive,
			EntityType: "financial_request",
			EntityID:   &reqID,
			OldValues:  map[string]any{"status": string(from)},
			NewValues: map[string]any{
				"status":             string(workflow.RequestDelivered),
				"receiver_name":      receiverName,
				"signed_voucher_url": signedVoucherURL,
				"notes":              notes,
			},
		})
	}

	bootstrap.InvalidateRequestsCache(req.ScopeID.String())

	return req, nil
}

// GetByID returns a request with its items and history (Redis Cached)
func (s *RequestService) GetByID(ctx context.Context, id uuid.UUID) (*workflow.FinancialRequest, error) {
	key := fmt.Sprintf("requests:detail:%s", id)
	return bootstrap.CacheRemember(key, 10*time.Minute, func() (*workflow.FinancialRequest, error) {
		req, err := s.requestRepo.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if req == nil {
			return nil, fmt.Errorf("request not found")
		}
		return req, nil
	})
}

// GetByScope returns all requests for a given scope (factory) — Rule 5 (Redis Cached)
func (s *RequestService) GetByScope(ctx context.Context, scopeID uuid.UUID) ([]workflow.FinancialRequest, error) {
	key := fmt.Sprintf("requests:scope:%s", scopeID)
	return bootstrap.CacheRemember(key, 5*time.Minute, func() ([]workflow.FinancialRequest, error) {
		return s.requestRepo.FindByScope(ctx, scopeID)
	})
}

// GetAll returns all requests (Administration view) — Rule 7 (Redis Cached)
func (s *RequestService) GetAll(ctx context.Context) ([]workflow.FinancialRequest, error) {
	return bootstrap.CacheRemember("requests:all", 5*time.Minute, func() ([]workflow.FinancialRequest, error) {
		return s.requestRepo.FindAll(ctx)
	})
}

// GetHistory returns the immutable audit trail for a request
func (s *RequestService) GetHistory(ctx context.Context, requestID uuid.UUID) ([]workflow.RequestHistory, error) {
	return s.historyRepo.FindByRequest(ctx, requestID)
}

// GetNextAdvanceSequence returns the next sequential advance number for a factory
func (s *RequestService) GetNextAdvanceSequence(ctx context.Context, factoryID uuid.UUID) (int, error) {
	return s.requestRepo.GetNextAdvanceSequence(ctx, factoryID)
}

// ─── Internal Helpers ─────────────────────────────────────────────────────────

func (s *RequestService) appendHistory(ctx context.Context, requestID uuid.UUID, from, to workflow.RequestStatus, action string, performedBy uuid.UUID, notes string) {
	h := &workflow.RequestHistory{
		ID:          uuid.New(),
		RequestID:   requestID,
		FromStatus:  from,
		ToStatus:    to,
		Action:      action,
		PerformedBy: performedBy,
		Notes:       notes,
	}
	_ = s.historyRepo.Save(ctx, h) // best-effort audit; non-blocking
}
