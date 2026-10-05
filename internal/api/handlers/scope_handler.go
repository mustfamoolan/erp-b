package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	appfinance "m3aml-erp/internal/application/finance"
	apporg "m3aml-erp/internal/application/organization"
	"m3aml-erp/internal/domain/organization"
)

// ScopeHandler handles organizational scope endpoints.
type ScopeHandler struct {
	scopeSvc   *apporg.ScopeService
	financeSvc *appfinance.FinanceService
}

func NewScopeHandler(scopeSvc *apporg.ScopeService, financeSvc *appfinance.FinanceService) *ScopeHandler {
	return &ScopeHandler{scopeSvc: scopeSvc, financeSvc: financeSvc}
}

// ListScopes GET /api/v1/scopes
func (h *ScopeHandler) ListScopes(c *fiber.Ctx) error {
	scopes, err := h.scopeSvc.GetAllScopes(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch scopes"})
	}
	return c.JSON(fiber.Map{"data": scopes})
}

// ListFactories GET /api/v1/scopes/factories
func (h *ScopeHandler) ListFactories(c *fiber.Ctx) error {
	areaIDStr := c.Query("area_id")
	scopes, err := h.scopeSvc.GetFactories(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch factories"})
	}

	if areaIDStr != "" {
		areaID, err := uuid.Parse(areaIDStr)
		if err == nil {
			var filtered []organization.OrganizationScope
			for _, s := range scopes {
				if s.AreaID != nil && *s.AreaID == areaID {
					filtered = append(filtered, s)
				}
			}
			scopes = filtered
		}
	}

	return c.JSON(fiber.Map{"data": scopes})
}

// GetScope GET /api/v1/scopes/:id
func (h *ScopeHandler) GetScope(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid scope id"})
	}

	scope, err := h.scopeSvc.GetScopeByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "lookup failed"})
	}
	if scope == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "scope not found"})
	}

	return c.JSON(fiber.Map{"data": scope})
}

// createScopeRequest is the body for POST /api/v1/scopes
type createScopeRequest struct {
	Type                 string  `json:"type"`
	Name                 string  `json:"name"`
	Code                 string  `json:"code"`
	Location             string  `json:"location"`
	ManagerName          string  `json:"manager_name"`
	AreaID               string  `json:"area_id"`
	TargetOpeningBalance float64 `json:"target_opening_balance"`
}

// CreateScope POST /api/v1/scopes
// Only SUPER_ADMIN or ADMINISTRATOR with factories.manage permission.
func (h *ScopeHandler) CreateScope(c *fiber.Ctx) error {
	var req createScopeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}
	if req.Code == "" {
		req.Code = "FAC-" + strings.ToUpper(uuid.New().String()[:6])
	}
	scopeType := organization.ScopeType(req.Type)
	if scopeType != organization.ScopeTypeAdministration && scopeType != organization.ScopeTypeFactory {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "type must be ADMINISTRATION or FACTORY"})
	}

	scopeReq := organization.OrganizationScope{
		Name:        req.Name,
		Code:        req.Code,
		Location:    req.Location,
		ManagerName: req.ManagerName,
	}

	if req.AreaID != "" {
		parsedArea, err := uuid.Parse(req.AreaID)
		if err == nil {
			scopeReq.AreaID = &parsedArea
		}
	}

	// Assuming UserID comes from auth middleware
	userIDRaw := c.Locals("user_id")
	var userID uuid.UUID
	if uidStr, ok := userIDRaw.(string); ok {
		userID, _ = uuid.Parse(uidStr)
	}

	if scopeType == organization.ScopeTypeFactory {
		createdScope, err := h.scopeSvc.CreateFactory(c.Context(), scopeReq, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": createdScope})
	}

	// For administration, we don't have atomic creation method yet, so return error for now or add to service.
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "creation of non-factory scopes not fully supported here yet"})
}

// UpdateScope PUT /api/v1/scopes/:id
// Only SUPER_ADMIN or ADMINISTRATOR with factories.manage permission.
func (h *ScopeHandler) UpdateScope(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid scope id"})
	}

	scope, err := h.scopeSvc.GetScopeByID(c.Context(), id)
	if err != nil || scope == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "scope not found"})
	}

	var req createScopeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if req.Name != "" {
		scope.Name = req.Name
	}
	if req.Code != "" {
		scope.Code = req.Code
	}
	if req.Location != "" {
		scope.Location = req.Location
	}
	if req.ManagerName != "" {
		scope.ManagerName = req.ManagerName
	}
	if req.AreaID != "" {
		parsedArea, err := uuid.Parse(req.AreaID)
		if err == nil {
			scope.AreaID = &parsedArea
		}
	}

	userIDRaw := c.Locals("user_id")
	var userID uuid.UUID
	if uidStr, ok := userIDRaw.(string); ok {
		userID, _ = uuid.Parse(uidStr)
	}

	if err := h.scopeSvc.UpdateScope(c.Context(), scope, req.TargetOpeningBalance, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update scope"})
	}
	return c.JSON(fiber.Map{"data": scope})
}
