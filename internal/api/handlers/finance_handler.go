package handlers

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"m3aml-erp/internal/api/middleware"
	appfinance "m3aml-erp/internal/application/finance"
	"m3aml-erp/internal/domain/cashbox"
	"m3aml-erp/internal/domain/identity"
	"m3aml-erp/internal/repositories"
)

type FinanceHandler struct {
	financeSvc  *appfinance.FinanceService
	cashboxRepo repositories.CashboxRepository
	userRepo    repositories.UserRepository
}

func NewFinanceHandler(svc *appfinance.FinanceService, cashboxRepo repositories.CashboxRepository, userRepo repositories.UserRepository) *FinanceHandler {
	return &FinanceHandler{financeSvc: svc, cashboxRepo: cashboxRepo, userRepo: userRepo}
}

// ListCashboxes GET /api/v1/finance/cashboxes?scope_id=...
// Rule 19: Scope isolation enforced — factory user sees only their scope's cashboxes.
func (h *FinanceHandler) ListCashboxes(c *fiber.Ctx) error {
	queryScopeID := c.Query("scope_id")

	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthenticated"})
	}
	userID, _ := uuid.Parse(userIDStr)

	// Admin view — can filter by scope_id query param or list all
	if queryScopeID != "" {
		scopeID, err := uuid.Parse(queryScopeID)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid scope_id"})
		}

		// Ensure user has access to this specific scope
		if err := middleware.EnsureScopeAccess(c.Context(), h.userRepo, userID, scopeID); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "access denied to requested scope"})
		}

		boxes, err := h.cashboxRepo.FindByScope(c.Context(), scopeID)
		if err != nil {
			fmt.Printf("ListCashboxes error: %v\n", err)
			return c.Status(500).JSON(fiber.Map{"error": "failed to fetch cashboxes for scope"})
		}
		fmt.Printf("ListCashboxes found %d boxes for scope %s\n", len(boxes), scopeID)
		return c.JSON(fiber.Map{"data": boxes})
	}

	// If no scope_id provided, return all cashboxes the user has access to.
	// If the user has global scope (scope.all), they get everything.
	// Otherwise, they get cashboxes for their allowed scopes.

	perms, _ := h.userRepo.GetUserPermissions(c.Context(), userID, uuid.Nil)
	hasGlobalScope := false
	for _, p := range perms {
		if p == identity.PermScopeAll {
			hasGlobalScope = true
			break
		}
	}

	if hasGlobalScope {
		boxes, err := h.cashboxRepo.FindAll(c.Context())
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to fetch cashboxes"})
		}
		return c.JSON(fiber.Map{"data": boxes})
	}

	// Fetch cashboxes for user's specific scopes
	allowedScopes, err := h.userRepo.GetUserScopeAccess(c.Context(), userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to verify scope access"})
	}

	var allBoxes []cashbox.Cashbox
	for _, sID := range allowedScopes {
		boxes, err := h.cashboxRepo.FindByScope(c.Context(), sID)
		if err == nil {
			allBoxes = append(allBoxes, boxes...)
		}
	}

	return c.JSON(fiber.Map{"data": allBoxes})
}

type createCashboxRequest struct {
	ScopeID              string `json:"scope_id"`
	Name                 string `json:"name"`
	AccountID            string `json:"account_id"`
	Currency             string `json:"currency"`
	TargetOpeningBalance string `json:"target_opening_balance"`
}

// CreateCashbox POST /api/v1/finance/cashboxes
// Authorization: Requires cashbox.view permission (enforced in router by RequirePermission middleware)
func (h *FinanceHandler) CreateCashbox(c *fiber.Ctx) error {
	var req createCashboxRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name is required"})
	}
	scopeID, err := uuid.Parse(req.ScopeID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scope_id"})
	}
	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid account_id"})
	}

	box, err := h.financeSvc.CreateCashbox(c.Context(), appfinance.CreateCashboxRequest{
		ScopeID:   scopeID,
		Name:      req.Name,
		AccountID: accountID,
		Currency:  req.Currency,
	})
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"data": box})
}

type transferFundsRequest struct {
	FromCashboxID string `json:"from_cashbox_id"`
	ToCashboxID   string `json:"to_cashbox_id"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	Description   string `json:"description"`
}

// TransferFunds POST /api/v1/finance/cashboxes/transfer
// Authorization: Requires cashbox.transfer permission (enforced in router by RequirePermission middleware)
// Rule 19: Scope isolation — service layer validates both cashboxes are accessible
func (h *FinanceHandler) TransferFunds(c *fiber.Ctx) error {
	var req transferFundsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	fromID, err := uuid.Parse(req.FromCashboxID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid from_cashbox_id"})
	}
	toID, err := uuid.Parse(req.ToCashboxID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid to_cashbox_id"})
	}
	if req.Description == "" {
		return c.Status(400).JSON(fiber.Map{"error": "description is required"})
	}
	amt, err := decimal.NewFromString(req.Amount)
	if err != nil || amt.LessThanOrEqual(decimal.Zero) {
		return c.Status(400).JSON(fiber.Map{"error": "amount must be a positive number"})
	}

	// Extract performing user from JWT (set by auth middleware)
	userIDStr, _ := c.Locals(middleware.LocalUserID).(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid user identity in token"})
	}

	// Verify access to fromCashbox
	fromBox, err := h.cashboxRepo.FindByID(c.Context(), fromID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "from_cashbox not found"})
	}
	if err := middleware.EnsureScopeAccess(c.Context(), h.userRepo, userID, fromBox.ScopeID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "access denied to source cashbox scope"})
	}

	// Verify access to toCashbox
	toBox, err := h.cashboxRepo.FindByID(c.Context(), toID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "to_cashbox not found"})
	}
	if err := middleware.EnsureScopeAccess(c.Context(), h.userRepo, userID, toBox.ScopeID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "access denied to destination cashbox scope"})
	}

	err = h.financeSvc.TransferFunds(c.Context(), appfinance.TransferFundsRequest{
		FromCashboxID: fromID,
		ToCashboxID:   toID,
		Amount:        amt,
		Currency:      req.Currency,
		Description:   req.Description,
		PerformedBy:   userID,
	})
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "funds transferred successfully"})
}

// EstablishOpeningBalance POST /api/v1/finance/cashboxes/:id/opening-balance
// Authorization: Requires cashbox.opening_balance permission
func (h *FinanceHandler) EstablishOpeningBalance(c *fiber.Ctx) error {
	idParam := c.Params("id")
	cashboxID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid cashbox id"})
	}

	userIDStr, _ := c.Locals(middleware.LocalUserID).(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid user identity in token"})
	}

	// Fetch cashbox to get its scope
	box, err := h.cashboxRepo.FindByID(c.Context(), cashboxID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "cashbox not found"})
	}

	// Verify user has access to this cashbox scope
	if err := middleware.EnsureScopeAccess(c.Context(), h.userRepo, userID, box.ScopeID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "access denied to cashbox scope"})
	}

	err = h.financeSvc.EstablishOpeningBalance(c.Context(), appfinance.EstablishOpeningBalanceRequest{
		CashboxID:   cashboxID,
		PerformedBy: userID,
	})
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique_opening_balance") {
			return c.Status(409).JSON(fiber.Map{"error": "opening balance already exists for this cashbox"})
		}
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "opening balance established successfully"})
}

// GetCashboxTransactions GET /api/v1/finance/cashboxes/:id/transactions
func (h *FinanceHandler) GetCashboxTransactions(c *fiber.Ctx) error {
	idParam := c.Params("id")
	cashboxID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid cashbox id"})
	}

	userIDStr, ok := c.Locals(middleware.LocalUserID).(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthenticated"})
	}
	userID, _ := uuid.Parse(userIDStr)

	// Fetch cashbox to get its scope
	box, err := h.cashboxRepo.FindByID(c.Context(), cashboxID)
	if err != nil || box == nil {
		fmt.Printf("GetCashboxTransactions Error finding cashbox %s: %v, box=%v\n", cashboxID, err, box)
		return c.Status(404).JSON(fiber.Map{"error": "cashbox not found"})
	}

	// Verify user has access to this cashbox scope
	if err := middleware.EnsureScopeAccess(c.Context(), h.userRepo, userID, box.ScopeID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "access denied to cashbox scope"})
	}

	txs, err := h.financeSvc.GetCashboxTransactions(c.Context(), cashboxID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch transactions"})
	}

	return c.JSON(fiber.Map{"data": txs})
}

// ─── Employee Custody ────────────────────────────────────────────────────────

type custodyRequestPayload struct {
	EmployeeID  string `json:"employee_id"`
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
}

// WithdrawCustody POST /api/v1/finance/cashboxes/:id/withdraw
// Authorization: Requires cashbox.custody.manage
func (h *FinanceHandler) WithdrawCustody(c *fiber.Ctx) error {
	idParam := c.Params("id")
	cashboxID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid cashbox id"})
	}

	var req custodyRequestPayload
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	empID, err := uuid.Parse(req.EmployeeID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid employee_id"})
	}

	amt, err := decimal.NewFromString(req.Amount)
	if err != nil || amt.LessThanOrEqual(decimal.Zero) {
		return c.Status(400).JSON(fiber.Map{"error": "amount must be positive"})
	}

	userIDStr, _ := c.Locals(middleware.LocalUserID).(string)
	userID, _ := uuid.Parse(userIDStr)

	// Verify user has access to this cashbox scope
	box, err := h.cashboxRepo.FindByID(c.Context(), cashboxID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "cashbox not found"})
	}
	if err := middleware.EnsureScopeAccess(c.Context(), h.userRepo, userID, box.ScopeID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "access denied to cashbox scope"})
	}

	err = h.financeSvc.WithdrawCustody(c.Context(), appfinance.CustodyRequest{
		CashboxID:   cashboxID,
		EmployeeID:  empID,
		Amount:      amt,
		Currency:    req.Currency,
		Description: req.Description,
		PerformedBy: userID,
	})
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "custody withdrawn successfully"})
}

// DepositCustody POST /api/v1/finance/cashboxes/:id/deposit
// Authorization: Requires cashbox.custody.manage
func (h *FinanceHandler) DepositCustody(c *fiber.Ctx) error {
	idParam := c.Params("id")
	cashboxID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid cashbox id"})
	}

	var req custodyRequestPayload
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	empID, err := uuid.Parse(req.EmployeeID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid employee_id"})
	}

	amt, err := decimal.NewFromString(req.Amount)
	if err != nil || amt.LessThanOrEqual(decimal.Zero) {
		return c.Status(400).JSON(fiber.Map{"error": "amount must be positive"})
	}

	userIDStr, _ := c.Locals(middleware.LocalUserID).(string)
	userID, _ := uuid.Parse(userIDStr)

	// Verify user has access to this cashbox scope
	box, err := h.cashboxRepo.FindByID(c.Context(), cashboxID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "cashbox not found"})
	}
	if err := middleware.EnsureScopeAccess(c.Context(), h.userRepo, userID, box.ScopeID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "access denied to cashbox scope"})
	}

	err = h.financeSvc.DepositCustody(c.Context(), appfinance.CustodyRequest{
		CashboxID:   cashboxID,
		EmployeeID:  empID,
		Amount:      amt,
		Currency:    req.Currency,
		Description: req.Description,
		PerformedBy: userID,
	})
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "custody deposited successfully"})
}
