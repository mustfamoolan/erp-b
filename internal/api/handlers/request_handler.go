package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"m3aml-erp/internal/api/middleware"
	appfinance "m3aml-erp/internal/application/finance"
	appwf "m3aml-erp/internal/application/workflow"
	"m3aml-erp/internal/domain/cashbox"
	"m3aml-erp/internal/domain/identity"
	"m3aml-erp/internal/domain/organization"
	"m3aml-erp/internal/domain/workflow"
	"m3aml-erp/internal/repositories"
)

// RequestHandler handles the HTTP layer for Financial/Material Requests (Roadmap §31, §32, §33)
type RequestHandler struct {
	svc         *appwf.RequestService
	userRepo    repositories.UserRepository
	financeSvc  *appfinance.FinanceService
	scopeRepo   repositories.ScopeRepository
	cashboxRepo repositories.CashboxRepository
}

func NewRequestHandler(
	svc *appwf.RequestService,
	userRepo repositories.UserRepository,
	financeSvc *appfinance.FinanceService,
	scopeRepo repositories.ScopeRepository,
	cashboxRepo repositories.CashboxRepository,
) *RequestHandler {
	return &RequestHandler{
		svc:         svc,
		userRepo:    userRepo,
		financeSvc:  financeSvc,
		scopeRepo:   scopeRepo,
		cashboxRepo: cashboxRepo,
	}
}

// ─── Create Request ───────────────────────────────────────────────────────────

type createRequestItemJSON struct {
	VariantID          *string `json:"variant_id"`
	ExpenseCategoryID  *string `json:"expense_category_id"`
	Description        string  `json:"description"`
	Quantity           string  `json:"quantity"`
	UnitID             *string `json:"unit_id"`
	EstimatedUnitPrice string  `json:"estimated_unit_price"`
	Notes              string  `json:"notes"`
}

type createRequestJSON struct {
	ScopeID           string                  `json:"scope_id"`
	FactoryID         string                  `json:"factory_id"`
	Type              string                  `json:"type"`
	RequestTypeID        *string                 `json:"request_type_id"`
	ExpenseCategoryID    *string                 `json:"expense_category_id"`
	FactoryExpenseTypeID *string                 `json:"factory_expense_type_id"`
	SupplierName      *string                 `json:"supplier_name"`
	ReceiverName      *string                 `json:"receiver_name"`
	ProjectName       *string                 `json:"project_name"`
	Attachments       *string                 `json:"attachments"` // JSON string representation
	ReceivingMethod   *string                 `json:"receiving_method"`
	PurchaseNumber    *string                 `json:"purchase_number"`
	WorkType          *string                 `json:"work_type"`
	BarcodeSKU        *string                 `json:"barcode_sku"`
	ExchangeRate      *string                 `json:"exchange_rate"`
	OriginalAmount    *string                 `json:"original_amount"`
	Purpose           string                  `json:"purpose"`
	Description       string                  `json:"description"`
	Currency          string                  `json:"currency"`
	Items             []createRequestItemJSON `json:"items"`
}

// CreateRequest POST /api/v1/requests
func (h *RequestHandler) CreateRequest(c *fiber.Ctx) error {
	var body createRequestJSON
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	scopeID, err := uuid.Parse(body.ScopeID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scope_id"})
	}
	factoryID, err := uuid.Parse(body.FactoryID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid factory_id"})
	}

	userIDStr := c.Locals("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	var items []appwf.CreateRequestItemInput
	for _, it := range body.Items {
		qty, _ := decimal.NewFromString(it.Quantity)
		price, _ := decimal.NewFromString(it.EstimatedUnitPrice)

		var variantID *uuid.UUID
		if it.VariantID != nil {
			parsed, e := uuid.Parse(*it.VariantID)
			if e == nil {
				variantID = &parsed
			}
		}
		var unitID *uuid.UUID
		if it.UnitID != nil {
			parsed, e := uuid.Parse(*it.UnitID)
			if e == nil {
				unitID = &parsed
			}
		}
		var expCatID *uuid.UUID
		if it.ExpenseCategoryID != nil {
			parsed, e := uuid.Parse(*it.ExpenseCategoryID)
			if e == nil {
				expCatID = &parsed
			}
		}

		items = append(items, appwf.CreateRequestItemInput{
			VariantID:          variantID,
			ExpenseCategoryID:  expCatID,
			Description:        it.Description,
			Quantity:           qty,
			UnitID:             unitID,
			EstimatedUnitPrice: price,
			Notes:              it.Notes,
		})
	}

	reqType := workflow.RequestType(body.Type)
	if reqType == "" {
		reqType = workflow.RequestTypeFinancial
	}

	var reqTypeID *uuid.UUID
	if body.RequestTypeID != nil && *body.RequestTypeID != "" {
		parsed, err := uuid.Parse(*body.RequestTypeID)
		if err == nil {
			reqTypeID = &parsed
		}
	}

	var expCatID *uuid.UUID
	if body.ExpenseCategoryID != nil && *body.ExpenseCategoryID != "" {
		parsed, err := uuid.Parse(*body.ExpenseCategoryID)
		if err == nil {
			expCatID = &parsed
		}
	}

	var factoryExpTypeID *uuid.UUID
	if body.FactoryExpenseTypeID != nil && *body.FactoryExpenseTypeID != "" {
		parsed, err := uuid.Parse(*body.FactoryExpenseTypeID)
		if err == nil {
			factoryExpTypeID = &parsed
		}
	}

	var exchangeRate *decimal.Decimal
	if body.ExchangeRate != nil && *body.ExchangeRate != "" {
		parsed, err := decimal.NewFromString(*body.ExchangeRate)
		if err == nil {
			exchangeRate = &parsed
		}
	}

	var originalAmount *decimal.Decimal
	if body.OriginalAmount != nil && *body.OriginalAmount != "" {
		parsed, err := decimal.NewFromString(*body.OriginalAmount)
		if err == nil {
			originalAmount = &parsed
		}
	}

	req, err := h.svc.CreateRequest(c.Context(), appwf.CreateRequestInput{
		ScopeID:           scopeID,
		FactoryID:         factoryID,
		RequestedBy:       userID,
		Type:              reqType,
		RequestTypeID:        reqTypeID,
		ExpenseCategoryID:    expCatID,
		FactoryExpenseTypeID: factoryExpTypeID,
		SupplierName:      body.SupplierName,
		ReceiverName:      body.ReceiverName,
		ProjectName:       body.ProjectName,
		Attachments:       body.Attachments,
		ReceivingMethod:   body.ReceivingMethod,
		PurchaseNumber:    body.PurchaseNumber,
		WorkType:          body.WorkType,
		BarcodeSKU:        body.BarcodeSKU,
		ExchangeRate:      exchangeRate,
		OriginalAmount:    originalAmount,
		Purpose:           body.Purpose,
		Description:       body.Description,
		Currency:          body.Currency,
		Items:             items,
	})
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"data": req})
}

// ─── Get Requests ─────────────────────────────────────────────────────────────

// GetAll GET /api/v1/requests — Administration view, filtered by user's allowed scopes
func (h *RequestHandler) GetAll(c *fiber.Ctx) error {
	reqs, err := h.svc.GetAll(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch requests"})
	}

	userIDStr, ok := c.Locals(middleware.LocalUserID).(string)
	if ok && userIDStr != "" {
		userID, err := uuid.Parse(userIDStr)
		if err == nil {
			// Check if user has global scope.all permission (Administrator)
			perms, _ := h.userRepo.GetUserPermissions(c.Context(), userID, uuid.Nil)
			hasScopeAll := false
			for _, p := range perms {
				if p == identity.PermScopeAll {
					hasScopeAll = true
					break
				}
			}

			// If not super-admin/scope.all, filter to only requests in user's assigned scopes
			if !hasScopeAll {
				allowedScopes, err := h.userRepo.GetUserScopeAccess(c.Context(), userID)
				if err != nil || len(allowedScopes) == 0 {
					reqs = []workflow.FinancialRequest{}
				} else {
					scopeSet := make(map[uuid.UUID]bool)
					for _, s := range allowedScopes {
						scopeSet[s] = true
					}
					var filtered []workflow.FinancialRequest
					for _, r := range reqs {
						if scopeSet[r.ScopeID] || scopeSet[r.FactoryID] {
							filtered = append(filtered, r)
						}
					}
					reqs = filtered
				}
			}
		}
	}

	return c.JSON(fiber.Map{"data": reqs})
}

// GetByScope GET /api/v1/requests/scope/:scopeID — Factory view (own scope only)
func (h *RequestHandler) GetByScope(c *fiber.Ctx) error {
	scopeID, err := uuid.Parse(c.Params("scopeID"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scopeID"})
	}

	userIDStr, ok := c.Locals(middleware.LocalUserID).(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthenticated"})
	}
	userID, _ := uuid.Parse(userIDStr)

	// Ensure user has access to the requested scope
	if err := middleware.EnsureScopeAccess(c.Context(), h.userRepo, userID, scopeID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "access denied to requested scope"})
	}

	reqs, err := h.svc.GetByScope(c.Context(), scopeID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch requests"})
	}
	return c.JSON(fiber.Map{"data": reqs})
}

// GetByID GET /api/v1/requests/:id
func (h *RequestHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	req, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	userIDStr, ok := c.Locals(middleware.LocalUserID).(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthenticated"})
	}
	userID, _ := uuid.Parse(userIDStr)

	if err := middleware.EnsureScopeAccess(c.Context(), h.userRepo, userID, req.ScopeID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "access denied to request's scope"})
	}

	return c.JSON(fiber.Map{"data": req})
}

// GetHistory GET /api/v1/requests/:id/history
func (h *RequestHandler) GetHistory(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	history, err := h.svc.GetHistory(c.Context(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch history"})
	}
	return c.JSON(fiber.Map{"data": history})
}

// ─── Workflow Transitions ─────────────────────────────────────────────────────

type transitionJSON struct {
	Notes string `json:"notes"`
}

func (h *RequestHandler) doTransition(c *fiber.Ctx, action string, newStatus workflow.RequestStatus) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	var body transitionJSON
	c.BodyParser(&body) // notes are optional

	userIDStr, ok := c.Locals(middleware.LocalUserID).(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthenticated"})
	}
	userID, _ := uuid.Parse(userIDStr)

	// Enforce scope access
	currentReq, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "request not found"})
	}
	if err := middleware.EnsureScopeAccess(c.Context(), h.userRepo, userID, currentReq.ScopeID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "access denied to request's scope"})
	}

	req, err := h.svc.Transition(c.Context(), appwf.TransitionInput{
		RequestID:   id,
		PerformedBy: userID,
		NewStatus:   newStatus,
		Action:      action,
		Notes:       body.Notes,
	})
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": req, "message": action + " successful"})
}

// Submit POST /api/v1/requests/:id/submit  — Factory Accountant
func (h *RequestHandler) Submit(c *fiber.Ctx) error {
	return h.doTransition(c, "SUBMIT", workflow.RequestSubmitted)
}

// StartReview POST /api/v1/requests/:id/review — Administration Auditor
func (h *RequestHandler) StartReview(c *fiber.Ctx) error {
	return h.doTransition(c, "START_REVIEW", workflow.RequestUnderReview)
}

// Approve POST /api/v1/requests/:id/approve — Administration Auditor
func (h *RequestHandler) Approve(c *fiber.Ctx) error {
	return h.doTransition(c, "APPROVE", workflow.RequestReviewApproved)
}

// AccountantApprove POST /api/v1/requests/:id/accountant-approve — Administration Accountant
func (h *RequestHandler) AccountantApprove(c *fiber.Ctx) error {
	return h.doTransition(c, "ACCOUNTANT_APPROVE", workflow.RequestAccountantApproved)
}

// Disburse POST /api/v1/requests/:id/disburse — Central Cashier
// Step 1: Disburses request and generates delivery receipt voucher. No cash movement yet.
func (h *RequestHandler) Disburse(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request id"})
	}
	var body transitionJSON
	_ = c.BodyParser(&body)

	userIDStr, ok := c.Locals(middleware.LocalUserID).(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthenticated"})
	}
	userID, _ := uuid.Parse(userIDStr)

	req, err := h.svc.Disburse(c.Context(), id, userID, body.Notes)
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": req, "message": "تم صرف السلفة وتوليد وصل التسليم للطباعة"})
}

type deliverJSON struct {
	ReceiverName     string `json:"receiver_name"`
	SignedVoucherURL string `json:"signed_voucher_url"`
	Notes            string `json:"notes"`
}

// Deliver POST /api/v1/requests/:id/deliver — Central Cashier
// Step 2: Requires recipient name + signed voucher upload.
// Triggers the immediate financial movement to the factory cashbox & accounting journal!
func (h *RequestHandler) Deliver(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request id"})
	}
	var body deliverJSON
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if strings.TrimSpace(body.ReceiverName) == "" {
		return c.Status(422).JSON(fiber.Map{"error": "اسم المستلم مطلوب لتأكيد تسليم السلفة"})
	}
	if strings.TrimSpace(body.SignedVoucherURL) == "" {
		return c.Status(422).JSON(fiber.Map{"error": "يرجى إرفاق صورة وصل التسليم الموقّع من المستلم"})
	}

	userIDStr, ok := c.Locals(middleware.LocalUserID).(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthenticated"})
	}
	userID, _ := uuid.Parse(userIDStr)

	// 1. Deliver in request service (updates receiver_name, signed_voucher_url, status to DELIVERED)
	req, err := h.svc.Deliver(c.Context(), id, userID, body.ReceiverName, body.SignedVoucherURL, body.Notes)
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}

	// 2. Financial Movement: Transfer funds from Admin cashbox to Factory cashbox!
	// Crucial rule: Proceeds normally even if admin cashbox currently has 0 balance or is not physically funded.
	ctx := c.Context()

	// Look up factory cashbox
	factoryBoxes, err := h.cashboxRepo.FindByScope(ctx, req.ScopeID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "factory cashbox lookup failed: " + err.Error()})
	}
	var factoryBox *cashbox.Cashbox
	for i := range factoryBoxes {
		if factoryBoxes[i].Currency == req.Currency && factoryBoxes[i].Status == cashbox.CashboxStatusActive {
			factoryBox = &factoryBoxes[i]
			break
		}
	}
	if factoryBox == nil {
		for i := range factoryBoxes {
			if factoryBoxes[i].Currency == "IQD" && factoryBoxes[i].Status == cashbox.CashboxStatusActive {
				factoryBox = &factoryBoxes[i]
				break
			}
		}
	}
	if factoryBox == nil {
		for i := range factoryBoxes {
			if factoryBoxes[i].Status == cashbox.CashboxStatusActive {
				factoryBox = &factoryBoxes[i]
				break
			}
		}
	}
	if factoryBox == nil {
		return c.Status(500).JSON(fiber.Map{"error": "لم يتم العثور على صندوق نشط للمصنع لتحويل السلفة إليه"})
	}

	transferCurrency := factoryBox.Currency

	// Look up Admin Scope and Admin Cashbox
	adminScopes, err := h.scopeRepo.FindByType(ctx, organization.ScopeTypeAdministration)
	if err != nil || len(adminScopes) == 0 {
		return c.Status(500).JSON(fiber.Map{"error": "admin scope not found"})
	}
	adminScope := adminScopes[0]

	adminBoxes, err := h.cashboxRepo.FindByScope(ctx, adminScope.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "admin cashbox lookup failed: " + err.Error()})
	}
	var adminBox *cashbox.Cashbox
	for i := range adminBoxes {
		if adminBoxes[i].Currency == transferCurrency && adminBoxes[i].Status == cashbox.CashboxStatusActive {
			adminBox = &adminBoxes[i]
			break
		}
	}
	if adminBox == nil {
		for i := range adminBoxes {
			if adminBoxes[i].Status == cashbox.CashboxStatusActive {
				adminBox = &adminBoxes[i]
				break
			}
		}
	}
	if adminBox == nil {
		return c.Status(500).JSON(fiber.Map{"error": "لم يتم العثور على صندوق نشط للإدارة العامة للصرف منه"})
	}

	// Execute transfer — covers deficit, raises imprest, records double-entry journal & cash transactions
	err = h.financeSvc.TransferFunds(ctx, appfinance.TransferFundsRequest{
		FromCashboxID: adminBox.ID,
		ToCashboxID:   factoryBox.ID,
		Amount:        req.TotalAmount,
		Currency:      transferCurrency,
		Description:   "تسليم سلفة طلب رقم " + req.DocumentNumber + " للمستلم " + body.ReceiverName + ": " + req.Purpose,
		PerformedBy:   userID,
		ApplyImprest:  true,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to process financial transfer: " + err.Error()})
	}

	return c.JSON(fiber.Map{"data": req, "message": "تم تسليم السلفة بنجاح وإيداع المبلغ في صندوق المعمل"})
}

// ReturnRevision POST /api/v1/requests/:id/return-revision — Auditor or Management Accountant
func (h *RequestHandler) ReturnRevision(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	var body transitionJSON
	_ = c.BodyParser(&body)
	if strings.TrimSpace(body.Notes) == "" {
		return c.Status(422).JSON(fiber.Map{"error": "ملاحظة سبب إعادة النظر مطلوبة إجبارياً"})
	}

	userIDStr, ok := c.Locals(middleware.LocalUserID).(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthenticated"})
	}
	userID, _ := uuid.Parse(userIDStr)

	current, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "request not found"})
	}
	if current.Status != workflow.RequestSubmitted && current.Status != workflow.RequestUnderReview && current.Status != workflow.RequestReviewApproved {
		return c.Status(422).JSON(fiber.Map{"error": "لا يمكن طلب إعادة النظر للطلب في حالته الحالية"})
	}

	req, err := h.svc.Transition(c.Context(), appwf.TransitionInput{
		RequestID:   id,
		PerformedBy: userID,
		NewStatus:   workflow.RequestReturnedForRevision,
		Action:      "RETURN_REVISION",
		Notes:       body.Notes,
	})
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": req, "message": "تمت إعادة الطلب للمعمل لإعادة النظر والتعديل"})
}

// Reject POST /api/v1/requests/:id/reject — Auditor or Accountant
func (h *RequestHandler) Reject(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	var body transitionJSON
	_ = c.BodyParser(&body)
	if strings.TrimSpace(body.Notes) == "" {
		return c.Status(422).JSON(fiber.Map{"error": "ملاحظة سبب الرفض مطلوبة إجبارياً"})
	}

	userIDStr, ok := c.Locals(middleware.LocalUserID).(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthenticated"})
	}
	userID, _ := uuid.Parse(userIDStr)

	current, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "request not found"})
	}

	var rejectStatus workflow.RequestStatus
	switch current.Status {
	case workflow.RequestSubmitted, workflow.RequestUnderReview:
		rejectStatus = workflow.RequestReviewRejected
	case workflow.RequestReviewApproved:
		rejectStatus = workflow.RequestAccountantRejected
	default:
		return c.Status(422).JSON(fiber.Map{"error": "cannot reject request in current status"})
	}

	req, err := h.svc.Transition(c.Context(), appwf.TransitionInput{
		RequestID:   id,
		PerformedBy: userID,
		NewStatus:   rejectStatus,
		Action:      "REJECT",
		Notes:       body.Notes,
	})
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": req, "message": "request rejected"})
}

// MarkPaymentPending POST /api/v1/requests/:id/payment-pending — Legacy compatibility
func (h *RequestHandler) MarkPaymentPending(c *fiber.Ctx) error {
	return h.Disburse(c)
}

// MarkFactoryReceiptPending POST /api/v1/requests/:id/factory-receipt-pending — Legacy compatibility
func (h *RequestHandler) MarkFactoryReceiptPending(c *fiber.Ctx) error {
	return h.Deliver(c)
}

// MarkReceived POST /api/v1/requests/:id/receive — Factory Accountant
func (h *RequestHandler) MarkReceived(c *fiber.Ctx) error {
	return h.doTransition(c, "RECEIVE", workflow.RequestReceived)
}

// Complete POST /api/v1/requests/:id/complete
func (h *RequestHandler) Complete(c *fiber.Ctx) error {
	return h.doTransition(c, "COMPLETE", workflow.RequestCompleted)
}

// Cancel POST /api/v1/requests/:id/cancel — Factory Accountant (DRAFT/SUBMITTED only)
func (h *RequestHandler) Cancel(c *fiber.Ctx) error {
	return h.doTransition(c, "CANCEL", workflow.RequestCancelled)
}
