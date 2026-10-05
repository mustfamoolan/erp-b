package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"m3aml-erp/internal/domain/identity"
	"m3aml-erp/internal/repositories"
)

// RoleHandler handles role management endpoints.
// Roadmap §9 — Roles.
type RoleHandler struct {
	roleRepo repositories.RoleRepository
}

func NewRoleHandler(roleRepo repositories.RoleRepository) *RoleHandler {
	return &RoleHandler{roleRepo: roleRepo}
}

// ListRoles GET /api/v1/roles
func (h *RoleHandler) ListRoles(c *fiber.Ctx) error {
	roles, err := h.roleRepo.FindAll(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch roles"})
	}
	return c.JSON(fiber.Map{"data": roles})
}

// GetRole GET /api/v1/roles/:id
func (h *RoleHandler) GetRole(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid role id"})
	}
	role, err := h.roleRepo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "lookup failed"})
	}
	if role == nil {
		return c.Status(404).JSON(fiber.Map{"error": "role not found"})
	}

	// Fetch permissions for this role
	perms, err := h.roleRepo.FindRolePermissions(c.Context(), role.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to load permissions"})
	}

	var permIDs []string
	for _, p := range perms {
		permIDs = append(permIDs, p.ID.String())
	}

	return c.JSON(fiber.Map{"data": map[string]interface{}{
		"id":           role.ID,
		"name":         role.Name,
		"display_name": role.DisplayName,
		"description":  role.Description,
		"is_system":    role.IsSystem,
		"permissions":  permIDs,
	}})
}

// ListPermissions GET /api/v1/permissions
func (h *RoleHandler) ListPermissions(c *fiber.Ctx) error {
	perms, err := h.roleRepo.FindAllPermissions(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch permissions"})
	}

	// Group by Group
	grouped := make(map[string][]identity.Permission)
	for _, p := range perms {
		grouped[p.Group] = append(grouped[p.Group], p)
	}

	return c.JSON(fiber.Map{"data": grouped})
}

// CreateRoleRequest
type createRoleRequest struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// CreateRole POST /api/v1/roles
func (h *RoleHandler) CreateRole(c *fiber.Ctx) error {
	var req createRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Name == "" || req.DisplayName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name and display_name are required"})
	}

	role := &identity.Role{
		ID:          uuid.New(),
		Name:        identity.RoleName(req.Name),
		DisplayName: req.DisplayName,
		Description: req.Description,
		IsSystem:    false,
	}

	if err := h.roleRepo.Save(c.Context(), role); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create role"})
	}

	for _, permIDStr := range req.Permissions {
		if pid, err := uuid.Parse(permIDStr); err == nil {
			_ = h.roleRepo.AssignPermission(c.Context(), role.ID, pid)
		}
	}

	return c.Status(201).JSON(fiber.Map{"data": role})
}

// UpdateRole PUT /api/v1/roles/:id
func (h *RoleHandler) UpdateRole(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid role id"})
	}

	var req createRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	role, err := h.roleRepo.FindByID(c.Context(), id)
	if err != nil || role == nil {
		return c.Status(404).JSON(fiber.Map{"error": "role not found"})
	}

	// System roles (like SUPER_ADMIN) shouldn't have their core names/permissions removed,
	// but let's assume we can edit display_name and description.
	if req.DisplayName != "" {
		role.DisplayName = req.DisplayName
	}
	role.Description = req.Description

	if err := h.roleRepo.Update(c.Context(), role); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update role"})
	}

	// Update permissions
	if len(req.Permissions) > 0 {
		_ = h.roleRepo.ClearPermissions(c.Context(), role.ID)
		for _, permIDStr := range req.Permissions {
			if pid, err := uuid.Parse(permIDStr); err == nil {
				_ = h.roleRepo.AssignPermission(c.Context(), role.ID, pid)
			}
		}
	}

	return c.JSON(fiber.Map{"data": role})
}
