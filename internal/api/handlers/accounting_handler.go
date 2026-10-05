package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"m3aml-erp/internal/api/middleware"
	appaccounting "m3aml-erp/internal/application/accounting"
	"m3aml-erp/internal/domain/accounting"
	"m3aml-erp/internal/repositories"
)

// AccountingHandler handles all accounting API endpoints.
// One handler — one engine — Roadmap Rule 21, 22.
type AccountingHandler struct {
	accountingSvc *appaccounting.AccountingService
	accountRepo   repositories.AccountRepository
	periodRepo    repositories.AccountingPeriodRepository
	fyRepo        repositories.FiscalYearRepository
	userRepo      repositories.UserRepository
}

func NewAccountingHandler(
	svc  *appaccounting.AccountingService,
	acct repositories.AccountRepository,
	per  repositories.AccountingPeriodRepository,
	fy   repositories.FiscalYearRepository,
	ur   repositories.UserRepository,
) *AccountingHandler {
	return &AccountingHandler{
		accountingSvc: svc,
		accountRepo:   acct,
		periodRepo:    per,
		fyRepo:        fy,
		userRepo:      ur,
	}
}

// ===== CHART OF ACCOUNTS =====

// ListAccounts GET /api/v1/accounting/accounts
func (h *AccountingHandler) ListAccounts(c *fiber.Ctx) error {
	accounts, err := h.accountRepo.FindAll(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch accounts"})
	}
	return c.JSON(fiber.Map{"data": accounts})
}

// GetAccount GET /api/v1/accounting/accounts/:id
func (h *AccountingHandler) GetAccount(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid account id"})
	}
	acct, err := h.accountRepo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "lookup failed"})
	}
	if acct == nil {
		return c.Status(404).JSON(fiber.Map{"error": "account not found"})
	}
	return c.JSON(fiber.Map{"data": acct})
}

// GetAccountBalance GET /api/v1/accounting/accounts/:id/balance?scope_id=...
// Balance is always computed from posted journal lines — Rule 10.
func (h *AccountingHandler) GetAccountBalance(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid account id"})
	}
	scopeID, err := uuid.Parse(c.Query("scope_id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "scope_id is required"})
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

	balance, err := h.accountingSvc.GetAccountBalance(c.Context(), id, scopeID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "balance computation failed"})
	}
	return c.JSON(fiber.Map{"data": balance})
}

// ===== ACCOUNTING PERIODS =====

// ListPeriods GET /api/v1/accounting/periods?fiscal_year_id=...
func (h *AccountingHandler) ListPeriods(c *fiber.Ctx) error {
	fyIDStr := c.Query("fiscal_year_id")
	if fyIDStr == "" {
		return c.Status(400).JSON(fiber.Map{"error": "fiscal_year_id is required"})
	}
	fyID, err := uuid.Parse(fyIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid fiscal_year_id"})
	}
	periods, err := h.periodRepo.FindByFiscalYear(c.Context(), fyID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch periods"})
	}
	return c.JSON(fiber.Map{"data": periods})
}

// ClosePeriod POST /api/v1/accounting/periods/:id/close
// Requires accounting.close_period permission (enforced in middleware).
func (h *AccountingHandler) ClosePeriod(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid period id"})
	}
	userIDStr, _ := c.Locals("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	if err := h.periodRepo.Close(c.Context(), id, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to close period: " + err.Error()})
	}
	return c.JSON(fiber.Map{"message": "period closed successfully"})
}

// ===== JOURNAL ENTRIES =====

type journalLineInput struct {
	AccountID    string `json:"account_id"`
	Debit        string `json:"debit"`
	Credit       string `json:"credit"`
	Description  string `json:"description"`
	ScopeID      string `json:"scope_id"`
	CostCenterID string `json:"cost_center_id"`
	Reference    string `json:"reference"`
}

type createJournalEntryRequest struct {
	EntryDate   string             `json:"entry_date"`   // YYYY-MM-DD
	Description string             `json:"description"`
	SourceType  string             `json:"source_type"`
	ScopeID     string             `json:"scope_id"`
	Lines       []journalLineInput `json:"lines"`
}

// CreateJournalEntry POST /api/v1/accounting/journal
func (h *AccountingHandler) CreateJournalEntry(c *fiber.Ctx) error {
	var req createJournalEntryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	entryDate, err := time.Parse("2006-01-02", req.EntryDate)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "entry_date must be YYYY-MM-DD"})
	}
	scopeID, err := uuid.Parse(req.ScopeID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scope_id"})
	}
	userIDStr, _ := c.Locals("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	// Build lines
	lines := make([]appaccounting.JournalLineInput, 0, len(req.Lines))
	for _, l := range req.Lines {
		acctID, err := uuid.Parse(l.AccountID)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid account_id in line"})
		}
		lineScopeID, err := uuid.Parse(l.ScopeID)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid scope_id in line"})
		}
		debit, _  := decimal.NewFromString(l.Debit)
		credit, _ := decimal.NewFromString(l.Credit)

		li := appaccounting.JournalLineInput{
			AccountID:   acctID,
			Debit:       debit,
			Credit:      credit,
			Description: l.Description,
			ScopeID:     lineScopeID,
			Reference:   l.Reference,
		}
		if l.CostCenterID != "" {
			ccID, err := uuid.Parse(l.CostCenterID)
			if err == nil {
				li.CostCenterID = &ccID
			}
		}
		lines = append(lines, li)
	}

	sourceType := accounting.SourceType(req.SourceType)
	if sourceType == "" {
		sourceType = accounting.SourceTypeManual
	}

	entry, err := h.accountingSvc.CreateDraftEntry(c.Context(), appaccounting.CreateJournalEntryRequest{
		EntryDate:   entryDate,
		Description: req.Description,
		SourceType:  sourceType,
		ScopeID:     scopeID,
		Lines:       lines,
		CreatedBy:   userID,
	})
	if err != nil {
		switch err {
		case accounting.ErrEntryNotBalanced:
			return c.Status(422).JSON(fiber.Map{"error": err.Error()})
		case accounting.ErrPeriodClosed:
			return c.Status(422).JSON(fiber.Map{"error": err.Error()})
		case accounting.ErrNotPostable:
			return c.Status(422).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
	}
	return c.Status(201).JSON(fiber.Map{"data": entry})
}

// PostJournalEntry POST /api/v1/accounting/journal/:id/post
func (h *AccountingHandler) PostJournalEntry(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid entry id"})
	}
	userIDStr, _ := c.Locals("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	if err := h.accountingSvc.PostEntry(c.Context(), id, userID); err != nil {
		switch err {
		case accounting.ErrEntryAlreadyPosted:
			return c.Status(409).JSON(fiber.Map{"error": err.Error()})
		case accounting.ErrEntryNotBalanced:
			return c.Status(422).JSON(fiber.Map{"error": err.Error()})
		case accounting.ErrPeriodClosed:
			return c.Status(422).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
	}
	return c.JSON(fiber.Map{"message": "journal entry posted successfully"})
}

// GetJournalEntry GET /api/v1/accounting/journal/:id
func (h *AccountingHandler) GetJournalEntry(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid entry id"})
	}
	// TODO: inject journalRepo for direct reads
	_ = id
	return c.JSON(fiber.Map{"message": "not implemented yet"})
}
