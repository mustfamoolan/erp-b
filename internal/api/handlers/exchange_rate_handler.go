package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	appaccounting "m3aml-erp/internal/application/accounting"
	apimw "m3aml-erp/internal/api/middleware"
)

// ============================================================
// ExchangeRateHandler — REST API for currency exchange rates.
// GET  /api/v1/exchange-rates           → GetCurrentRates (all users)
// GET  /api/v1/exchange-rates/history   → GetHistory (all users)
// GET  /api/v1/exchange-rates/convert   → ConvertAmount preview (all users)
// POST /api/v1/exchange-rates           → SetRate (SUPER_ADMIN only — enforced in router)
// Rule 20: Mutation is restricted at the router level via RequirePermission.
// Rule 12: All amounts are NUMERIC — decimal.Decimal throughout.
// ============================================================

type ExchangeRateHandler struct {
	exchangeRateSvc *appaccounting.ExchangeRateService
}

func NewExchangeRateHandler(exchangeRateSvc *appaccounting.ExchangeRateService) *ExchangeRateHandler {
	return &ExchangeRateHandler{exchangeRateSvc: exchangeRateSvc}
}

// ============================================================
// GetCurrentRates — GET /api/v1/exchange-rates
// Returns the current exchange rates for all supported currency pairs.
// ============================================================
func (h *ExchangeRateHandler) GetCurrentRates(c *fiber.Ctx) error {
	ctx := c.Context()

	rate, err := h.exchangeRateSvc.GetCurrentRate(ctx, "USD", "IQD")
	if err != nil {
		// No rate configured yet is not a server error — return empty
		return c.JSON(fiber.Map{
			"data": fiber.Map{
				"usd_to_iqd": nil,
				"message":    "لم يتم تحديد سعر الصرف بعد",
			},
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"id":             rate.ID,
			"from_currency":  rate.FromCurrency,
			"to_currency":    rate.ToCurrency,
			"rate":           rate.Rate,
			"effective_date": rate.EffectiveDate.Format("2006-01-02"),
			"created_at":     rate.CreatedAt,
		},
	})
}

// ============================================================
// GetHistory — GET /api/v1/exchange-rates/history?from=USD&to=IQD
// Returns all historical exchange rates for a pair, newest first.
// ============================================================
func (h *ExchangeRateHandler) GetHistory(c *fiber.Ctx) error {
	ctx := c.Context()

	fromCurrency := c.Query("from", "USD")
	toCurrency := c.Query("to", "IQD")

	rates, err := h.exchangeRateSvc.ListHistory(ctx, fromCurrency, toCurrency)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "فشل في جلب سجل أسعار الصرف",
		})
	}

	result := make([]fiber.Map, 0, len(rates))
	for _, r := range rates {
		result = append(result, fiber.Map{
			"id":             r.ID,
			"from_currency":  r.FromCurrency,
			"to_currency":    r.ToCurrency,
			"rate":           r.Rate,
			"effective_date": r.EffectiveDate.Format("2006-01-02"),
			"created_at":     r.CreatedAt,
		})
	}

	return c.JSON(fiber.Map{"data": result, "total": len(result)})
}

// ============================================================
// ConvertAmount — GET /api/v1/exchange-rates/convert?amount=100&from=USD&to=IQD
// Preview-only conversion. Does NOT write to DB. Used by WPF clients.
// ============================================================
func (h *ExchangeRateHandler) ConvertAmount(c *fiber.Ctx) error {
	ctx := c.Context()

	amountStr := c.Query("amount", "1")
	fromCurrency := c.Query("from", "USD")

	amount, err := decimal.NewFromString(amountStr)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "المبلغ غير صالح — يجب أن يكون رقماً أكبر من صفر",
		})
	}

	base, rateUsed, err := h.exchangeRateSvc.ConvertToBase(ctx, amount, fromCurrency)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"original_amount":   amount,
			"from_currency":     fromCurrency,
			"base_amount_iqd":   base,
			"to_currency":       "IQD",
			"exchange_rate_used": rateUsed,
		},
	})
}

// ============================================================
// SetRate — POST /api/v1/exchange-rates
// Creates a new exchange rate record. SUPER_ADMIN only (enforced at router).
// ============================================================

type setRateRequest struct {
	FromCurrency  string  `json:"from_currency"`
	ToCurrency    string  `json:"to_currency"`
	Rate          string  `json:"rate"` // string to avoid floating-point in JSON — Rule 12
	EffectiveDate string  `json:"effective_date"` // "2026-09-29"
}

func (h *ExchangeRateHandler) SetRate(c *fiber.Ctx) error {
	ctx := c.Context()

	// Extract authenticated user from JWT context — Rule 21
	userIDStr, _ := c.Locals(apimw.LocalUserID).(string)
	setByID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "غير مصرح"})
	}

	var req setRateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "بيانات غير صالحة"})
	}

	// Validate currency codes
	if req.FromCurrency == "" || req.ToCurrency == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "from_currency و to_currency مطلوبان",
		})
	}

	// Parse rate — Rule 12: use decimal, not float
	rate, err := decimal.NewFromString(req.Rate)
	if err != nil || rate.LessThanOrEqual(decimal.Zero) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "سعر الصرف يجب أن يكون رقماً أكبر من صفر",
		})
	}

	// Parse effective date
	effectiveDate := time.Now()
	if req.EffectiveDate != "" {
		parsed, err := time.Parse("2006-01-02", req.EffectiveDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "تنسيق التاريخ غير صالح — استخدم YYYY-MM-DD",
			})
		}
		effectiveDate = parsed
	}

	saved, err := h.exchangeRateSvc.SetRate(ctx, appaccounting.SetRateInput{
		FromCurrency:  req.FromCurrency,
		ToCurrency:    req.ToCurrency,
		Rate:          rate,
		EffectiveDate: effectiveDate,
		SetBy:         setByID,
	})
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": fiber.Map{
			"id":             saved.ID,
			"from_currency":  saved.FromCurrency,
			"to_currency":    saved.ToCurrency,
			"rate":           saved.Rate,
			"effective_date": saved.EffectiveDate.Format("2006-01-02"),
		},
		"message": "تم حفظ سعر الصرف بنجاح",
	})
}
