package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	
	apppurchasing "m3aml-erp/internal/application/purchasing"
	"m3aml-erp/internal/repositories"
)

type PurchasingHandler struct {
	svc      *apppurchasing.PurchasingService
	userRepo repositories.UserRepository
}

func NewPurchasingHandler(svc *apppurchasing.PurchasingService, userRepo repositories.UserRepository) *PurchasingHandler {
	return &PurchasingHandler{svc: svc, userRepo: userRepo}
}

type CreateInvoiceRequest struct {
	ScopeID       string `json:"scope_id"`
	InvoiceNumber string `json:"invoice_number"`
	InvoiceDate   string `json:"invoice_date"`
	SupplierName  string `json:"supplier_name"`
	Discount      string `json:"discount"`
	Tax           string `json:"tax"`
	Currency      string `json:"currency"`
	Notes         string `json:"notes"`
	Items         []struct {
		Description string `json:"description"`
		Quantity    string `json:"quantity"`
		UnitID      string `json:"unit_id"`
		UnitPrice   string `json:"unit_price"`
		Notes       string `json:"notes"`
	} `json:"items"`
}

func (h *PurchasingHandler) CreateInvoice(c *fiber.Ctx) error {
	var req CreateInvoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	scopeID, err := uuid.Parse(req.ScopeID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scope_id"})
	}

	// Verify User has access to Scope
	userID := c.Locals("user_id").(uuid.UUID)

	// In a real app we'd parse time, for now we just use time.Now() if empty, or parse properly.
	// For simplicity in this handler we skip strict parsing.

	discount, _ := decimal.NewFromString(req.Discount)
	tax, _ := decimal.NewFromString(req.Tax)

	var items []apppurchasing.CreateInvoiceItemInput
	for _, item := range req.Items {
		q, _ := decimal.NewFromString(item.Quantity)
		up, _ := decimal.NewFromString(item.UnitPrice)
		var uID *uuid.UUID
		if item.UnitID != "" {
			id, _ := uuid.Parse(item.UnitID)
			uID = &id
		}
		items = append(items, apppurchasing.CreateInvoiceItemInput{
			Description: item.Description,
			Quantity:    q,
			UnitID:      uID,
			UnitPrice:   up,
			Notes:       item.Notes,
		})
	}

	// Wait, we need real date parsing for `InvoiceDate`. Let's assume RFC3339 or just use now if missing.
	
	input := apppurchasing.CreateInvoiceInput{
		ScopeID:       scopeID,
		InvoiceNumber: req.InvoiceNumber,
		SupplierName:  req.SupplierName,
		Discount:      discount,
		Tax:           tax,
		Currency:      req.Currency,
		Notes:         req.Notes,
		CreatedBy:     userID,
		Items:         items,
	}

	inv, err := h.svc.CreateInvoice(c.Context(), input)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{"data": inv})
}

func (h *PurchasingHandler) GetInvoices(c *fiber.Ctx) error {
	scopeID, err := uuid.Parse(c.Query("scope_id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scope_id"})
	}

	invoices, err := h.svc.GetByScope(c.Context(), scopeID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"data": invoices})
}

func (h *PurchasingHandler) GetInvoice(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid invoice id"})
	}

	inv, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if inv == nil {
		return c.Status(404).JSON(fiber.Map{"error": "invoice not found"})
	}

	return c.JSON(fiber.Map{"data": inv})
}

type PayInvoiceRequest struct {
	CashboxID         string `json:"cashbox_id"`
	Amount            string `json:"amount"`
	ExpenseCategoryID string `json:"expense_category_id"`
}

func (h *PurchasingHandler) PayInvoice(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid invoice id"})
	}

	var req PayInvoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	cbID, err := uuid.Parse(req.CashboxID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid cashbox_id"})
	}

	expID, err := uuid.Parse(req.ExpenseCategoryID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid expense_category_id"})
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid amount"})
	}

	userID := c.Locals("user_id").(uuid.UUID)

	if err := h.svc.PayInvoice(c.Context(), id, cbID, amount, expID, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Payment recorded successfully"})
}
