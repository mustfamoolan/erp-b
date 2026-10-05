package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"m3aml-erp/internal/api/middleware"
	appfinance "m3aml-erp/internal/application/finance"
	"m3aml-erp/internal/repositories"
)

type ExpenseHandler struct {
	expenseSvc *appfinance.ExpenseService
	userRepo   repositories.UserRepository
}

func NewExpenseHandler(expenseSvc *appfinance.ExpenseService, userRepo repositories.UserRepository) *ExpenseHandler {
	return &ExpenseHandler{expenseSvc: expenseSvc, userRepo: userRepo}
}

type recordExpenseRequest struct {
	ScopeID           uuid.UUID `json:"scope_id"`
	CashboxID         uuid.UUID `json:"cashbox_id"`
	ExpenseCategoryID uuid.UUID `json:"expense_category_id"`
	Amount            float64   `json:"amount"`
	Currency          string    `json:"currency"`
	ExpenseDate       string    `json:"expense_date"`
	Purpose           string    `json:"purpose"`
	PaidTo            string    `json:"paid_to"`
	Description       string    `json:"description"`
}

func (h *ExpenseHandler) RecordExpense(c *fiber.Ctx) error {
	var req recordExpenseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	userIDStr := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	if err := middleware.EnsureScopeAccess(c.Context(), h.userRepo, userID, req.ScopeID); err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
	}

	expDate, err := time.Parse(time.RFC3339, req.ExpenseDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid expense_date format. Use RFC3339."})
	}

	expenseReq := appfinance.RecordExpenseRequest{
		ScopeID:           req.ScopeID,
		CashboxID:         req.CashboxID,
		ExpenseCategoryID: req.ExpenseCategoryID,
		Amount:            decimal.NewFromFloat(req.Amount),
		Currency:          req.Currency,
		ExpenseDate:       expDate,
		Purpose:           req.Purpose,
		PaidTo:            req.PaidTo,
		Description:       req.Description,
		PerformedBy:       userID,
	}

	expense, err := h.expenseSvc.RecordExpense(c.Context(), expenseReq)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(expense)
}
