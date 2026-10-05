package reportinghandler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	appreporting "m3aml-erp/internal/application/reporting"
	"m3aml-erp/internal/domain/reporting"
)

type Handler struct {
	svc *appreporting.Service
}

func NewReportingHandler(svc *appreporting.Service) *Handler {
	return &Handler{svc: svc}
}

func parseDateRange(c *fiber.Ctx) reporting.DateRange {
	var dr reporting.DateRange
	fromStr := c.Query("from")
	toStr := c.Query("to")

	if fromStr != "" {
		t, err := time.Parse("2006-01-02", fromStr)
		if err == nil {
			dr.From = &t
		}
	}
	if toStr != "" {
		// End of day
		t, err := time.Parse("2006-01-02", toStr)
		if err == nil {
			t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			dr.To = &t
		}
	}
	return dr
}

// GetAccountStatement GET /api/v1/reports/accounting/statement/:accountId
func (h *Handler) GetAccountStatement(c *fiber.Ctx) error {
	accountID, err := uuid.Parse(c.Params("accountId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid account id"})
	}

	filter := parseDateRange(c)

	report, err := h.svc.GetAccountStatement(c.Context(), accountID, filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate account statement"})
	}

	return c.JSON(fiber.Map{"data": report})
}

// GetTrialBalance GET /api/v1/reports/accounting/trial-balance
func (h *Handler) GetTrialBalance(c *fiber.Ctx) error {
	filter := parseDateRange(c)

	report, err := h.svc.GetTrialBalance(c.Context(), filter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate trial balance"})
	}

	return c.JSON(fiber.Map{"data": report})
}

// GetStockBalance GET /api/v1/reports/inventory/stock-balance
func (h *Handler) GetStockBalance(c *fiber.Ctx) error {
	var scopeID *uuid.UUID
	scopeStr := c.Query("scope_id")
	if scopeStr != "" {
		if id, err := uuid.Parse(scopeStr); err == nil {
			scopeID = &id
		}
	}

	report, err := h.svc.GetStockBalance(c.Context(), scopeID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate stock balance"})
	}

	return c.JSON(fiber.Map{"data": report})
}
