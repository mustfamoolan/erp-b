package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	
	"m3aml-erp/internal/api/middleware"
	appidentity "m3aml-erp/internal/application/identity"
	"m3aml-erp/internal/domain/identity"
	"m3aml-erp/internal/domain/organization"
	"m3aml-erp/internal/repositories"
)

// EmployeeHandler handles employee management endpoints.
// Roadmap §8 — Employees are linked to users and scopes.
type EmployeeHandler struct {
	employeeRepo repositories.EmployeeRepository
	scopeRepo    repositories.ScopeRepository
	userSvc      *appidentity.UserService
	roleRepo     repositories.RoleRepository
	areaRepo     repositories.AreaRepository
	userRepo     repositories.UserRepository
}

func NewEmployeeHandler(
	employeeRepo repositories.EmployeeRepository,
	scopeRepo    repositories.ScopeRepository,
	userSvc      *appidentity.UserService,
	roleRepo     repositories.RoleRepository,
	areaRepo     repositories.AreaRepository,
	userRepo     repositories.UserRepository,
) *EmployeeHandler {
	return &EmployeeHandler{
		employeeRepo: employeeRepo,
		scopeRepo:    scopeRepo,
		userSvc:      userSvc,
		roleRepo:     roleRepo,
		areaRepo:     areaRepo,
		userRepo:     userRepo,
	}
}

// ─── GET /api/v1/employees?scope_id=... ─────────────────────

// ListEmployees returns employees for a scope.
// Rule 7: The backend enforces that the caller has access to the requested scope.
func (h *EmployeeHandler) ListEmployees(c *fiber.Ctx) error {
	type extendedEmployee struct {
		organization.Employee
		Username   string      `json:"username"`
		ScopeIDs   []uuid.UUID `json:"scope_ids"`
		RoleName   string      `json:"role_name"`
		ScopeName  string      `json:"scope_name"`
		AreaName   string      `json:"area_name"`
	}

	var employees []organization.Employee
	var err error

	userIDStr, ok := c.Locals(middleware.LocalUserID).(string)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthenticated"})
	}
	userID, _ := uuid.Parse(userIDStr)

	scopeIDStr := c.Query("scope_id")
	if scopeIDStr == "" {
		// If no scope_id provided, return all employees IF user has scope.all
		perms, _ := h.userRepo.GetUserPermissions(c.Context(), userID, uuid.Nil)
		hasGlobalScope := false
		for _, p := range perms {
			if p == identity.PermScopeAll {
				hasGlobalScope = true
				break
			}
		}

		if hasGlobalScope {
			employees, err = h.employeeRepo.FindAll(c.Context())
		} else {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "scope_id is required or lack global permission"})
		}
	} else {
		scopeID, parseErr := uuid.Parse(scopeIDStr)
		if parseErr != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid scope_id"})
		}

		if err := middleware.EnsureScopeAccess(c.Context(), h.userRepo, userID, scopeID); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "access denied to requested scope"})
		}

		employees, err = h.employeeRepo.FindByScope(c.Context(), scopeID)
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch employees"})
	}

	extEmployees := make([]extendedEmployee, 0, len(employees))
	for _, emp := range employees {
		ext := extendedEmployee{Employee: emp, ScopeIDs: make([]uuid.UUID, 0)}
		
		if h.scopeRepo != nil {
			scope, _ := h.scopeRepo.FindByID(c.Context(), emp.ScopeID)
			if scope != nil {
				ext.ScopeName = scope.Name
			}
		}

		user, _ := h.userSvc.FindUserByEmployeeID(c.Context(), emp.ID)
		if user != nil {
			ext.Username = user.Username
			
			var scopeNames []string
			var areaNames []string
			areaIDMap := make(map[uuid.UUID]bool)

			if ext.ScopeName != "" {
				scopeNames = append(scopeNames, ext.ScopeName)
				if h.scopeRepo != nil && emp.ScopeID != uuid.Nil {
					if primaryScope, _ := h.scopeRepo.FindByID(c.Context(), emp.ScopeID); primaryScope != nil && primaryScope.AreaID != nil {
						if area, _ := h.areaRepo.FindByID(c.Context(), *primaryScope.AreaID); area != nil {
							areaNames = append(areaNames, area.Name)
							areaIDMap[area.ID] = true
						}
					}
				}
			}
			
			for _, sa := range user.ScopeAccess {
				ext.ScopeIDs = append(ext.ScopeIDs, sa.ScopeID)
				if h.scopeRepo != nil {
					if extraScope, _ := h.scopeRepo.FindByID(c.Context(), sa.ScopeID); extraScope != nil {
						if sa.ScopeID != emp.ScopeID {
							scopeNames = append(scopeNames, extraScope.Name)
						}
						if extraScope.AreaID != nil && !areaIDMap[*extraScope.AreaID] {
							if area, _ := h.areaRepo.FindByID(c.Context(), *extraScope.AreaID); area != nil {
								areaNames = append(areaNames, area.Name)
								areaIDMap[area.ID] = true
							}
						}
					}
				}
			}
			if len(scopeNames) > 0 {
				ext.ScopeName = strings.Join(scopeNames, " + ")
			}
			if len(areaNames) > 0 {
				ext.AreaName = strings.Join(areaNames, " + ")
			}

			if len(user.Roles) > 0 && h.roleRepo != nil {
				role, _ := h.roleRepo.FindByID(c.Context(), user.Roles[0].RoleID)
				if role != nil {
					ext.RoleName = string(role.Name)
				}
			}
		}
		extEmployees = append(extEmployees, ext)
	}

	return c.JSON(fiber.Map{"data": extEmployees})
}

// ─── GET /api/v1/employees/:id ──────────────────────────────

// GetEmployee returns a single employee.
func (h *EmployeeHandler) GetEmployee(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid employee id"})
	}
	emp, err := h.employeeRepo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "lookup failed"})
	}
	if emp == nil {
		return c.Status(404).JSON(fiber.Map{"error": "employee not found"})
	}
	
	type extendedEmployee struct {
		*organization.Employee
		Username   string      `json:"username"`
		ScopeIDs   []uuid.UUID `json:"scope_ids"`
		RoleName   string      `json:"role_name"`
	}

	ext := extendedEmployee{Employee: emp, ScopeIDs: make([]uuid.UUID, 0)}

	user, _ := h.userSvc.FindUserByEmployeeID(c.Context(), emp.ID)
	if user != nil {
		ext.Username = user.Username
		for _, sa := range user.ScopeAccess {
			ext.ScopeIDs = append(ext.ScopeIDs, sa.ScopeID)
		}
		// Get role name by looking up the role
		if len(user.Roles) > 0 && h.roleRepo != nil {
			role, _ := h.roleRepo.FindByID(c.Context(), user.Roles[0].RoleID)
			if role != nil {
				ext.RoleName = string(role.Name)
			}
		}
	}

	return c.JSON(fiber.Map{"data": ext})
}

// ─── POST /api/v1/employees ─────────────────────────────────

type createEmployeeRequest struct {
	ScopeID    string   `json:"scope_id"`
	ScopeIDs   []string `json:"scope_ids"` // Multiple factories they can manage
	EmployeeNo string   `json:"employee_no"`
	FullName   string   `json:"full_name"`
	JobTitle   string   `json:"job_title"`
	NationalID string   `json:"national_id"`
	Phone      string   `json:"phone"`
	Email      string   `json:"email"`
	Photo      *string  `json:"photo,omitempty"`
	HireDate   string   `json:"hire_date"` // YYYY-MM-DD
	Status     string   `json:"status"`
	// Optional User Creation
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	RoleName   string `json:"role_name,omitempty"`
}

// CreateEmployee creates a new employee in the given scope.
// Rule 5: Every employee must belong to a scope.
func (h *EmployeeHandler) CreateEmployee(c *fiber.Ctx) error {
	var req createEmployeeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.ScopeID == "" && len(req.ScopeIDs) > 0 {
		req.ScopeID = req.ScopeIDs[0]
	}
	if req.ScopeID == "" {
		scopes, _ := h.scopeRepo.FindAll(c.Context())
		if len(scopes) > 0 {
			req.ScopeID = scopes[0].ID.String()
		}
	}
	if req.FullName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "full_name is required"})
	}

	if req.JobTitle == "" {
		if req.RoleName != "" {
			req.JobTitle = req.RoleName
		} else {
			req.JobTitle = "موظف"
		}
	}

	if req.EmployeeNo == "" {
		employees, _ := h.employeeRepo.FindAll(c.Context())
		maxNo := 100000
		for _, e := range employees {
			if val, err := strconv.Atoi(e.EmployeeNo); err == nil {
				if val > maxNo {
					maxNo = val
				}
			}
		}
		req.EmployeeNo = strconv.Itoa(maxNo + 1)
	}

	scopeID, err := uuid.Parse(req.ScopeID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scope_id"})
	}

	// Verify scope exists — Rule 5
	scope, err := h.scopeRepo.FindByID(c.Context(), scopeID)
	if err != nil || scope == nil {
		return c.Status(404).JSON(fiber.Map{"error": "scope not found"})
	}

	// Check duplicate employee number
	existing, err := h.employeeRepo.FindByEmployeeNo(c.Context(), req.EmployeeNo)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "lookup failed"})
	}
	if existing != nil {
		return c.Status(409).JSON(fiber.Map{"error": "employee number already exists"})
	}

	var natID *string
	if req.NationalID != "" {
		natID = &req.NationalID
	}

	emp := &organization.Employee{
		ID:         uuid.New(),
		ScopeID:    scopeID,
		EmployeeNo: req.EmployeeNo,
		FullName:   req.FullName,
		JobTitle:   req.JobTitle,
		NationalID: natID,
		Phone:      req.Phone,
		Email:      req.Email,
		Photo:      req.Photo,
		Status:     organization.EmployeeStatusActive,
	}

	if err := h.employeeRepo.Save(c.Context(), emp); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create employee: " + err.Error()})
	}

	// Create User account if ANY scopes or role is assigned (scopes MUST be attached to a user)
	needUser := req.Username != "" || req.RoleName != "" || len(req.ScopeIDs) > 0
	if needUser && h.userSvc != nil {
		if req.Username == "" {
			req.Username = "emp" + req.EmployeeNo
		}
		if req.Password == "" {
			req.Password = "12345678" // Default password
		}

		userReq := appidentity.CreateUserRequest{
			Username:   req.Username,
			Email:      req.Email, // Email could be empty, but CreateUser checks it
			Password:   req.Password,
			FullName:   req.FullName,
			EmployeeID: &emp.ID,
		}
		if userReq.Email == "" {
			userReq.Email = req.Username + "@m3aml.local" // dummy email if not provided
		}
		user, err := h.userSvc.CreateUser(c.Context(), userReq)
		if err == nil {
			grantedByStr, _ := c.Locals("user_id").(string)
			grantedBy, _ := uuid.Parse(grantedByStr)
			if grantedBy == uuid.Nil {
				grantedBy = emp.ID // fallback
			}
			// Grant scope access to the newly created user for this factory
			h.userSvc.GrantScopeAccess(c.Context(), user.ID, scopeID, grantedBy)
			
			// Also grant for any additional scopes
			for _, sid := range req.ScopeIDs {
				parsedSid, err := uuid.Parse(sid)
				if err == nil && parsedSid != scopeID {
					h.userSvc.GrantScopeAccess(c.Context(), user.ID, parsedSid, grantedBy)
				}
			}

			// Assign role if provided
			if req.RoleName != "" && h.roleRepo != nil {
				role, _ := h.roleRepo.FindByName(c.Context(), identity.RoleName(req.RoleName))
				if role != nil {
					h.userSvc.AssignRole(c.Context(), user.ID, role.ID, &scopeID)
				}
			}
		} else {
			fmt.Printf("ERROR: Failed to create user for employee %s: %v\n", emp.EmployeeNo, err)
			return c.Status(500).JSON(fiber.Map{"error": "employee created but user account failed: " + err.Error()})
		}
	}

	return c.Status(201).JSON(fiber.Map{"data": emp})
}

// ─── PUT /api/v1/employees/:id ──────────────────────────────

func (h *EmployeeHandler) UpdateEmployee(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var req createEmployeeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	emp, err := h.employeeRepo.FindByID(c.Context(), id)
	if err != nil || emp == nil {
		return c.Status(404).JSON(fiber.Map{"error": "employee not found"})
	}

	if req.FullName != "" { emp.FullName = req.FullName }
	if req.JobTitle != "" { emp.JobTitle = req.JobTitle }
	if req.NationalID != "" { emp.NationalID = &req.NationalID }
	if req.Phone != "" { emp.Phone = req.Phone }
	if req.Email != "" { emp.Email = req.Email }
	if req.Photo != nil { emp.Photo = req.Photo }
	if req.Status != "" { emp.Status = organization.EmployeeStatus(req.Status) }
	
	if req.ScopeID != "" {
		if scopeID, err := uuid.Parse(req.ScopeID); err == nil {
			emp.ScopeID = scopeID
		}
	}

	if err := h.employeeRepo.Update(c.Context(), emp); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update employee: " + err.Error()})
	}

	// Update the associated User's scopes and roles
	user, _ := h.userSvc.FindUserByEmployeeID(c.Context(), emp.ID)
	
	if user == nil && (req.Username != "" || req.RoleName != "") {
		if req.Username == "" {
			req.Username = "emp" + emp.EmployeeNo
		}
		if req.Password == "" {
			req.Password = "123456"
		}
		// Create a user for this employee
		newUser, err := h.userSvc.CreateUser(c.Context(), appidentity.CreateUserRequest{
			Username:   req.Username,
			Password:   req.Password,
			Email:      req.Email,
			FullName:   req.FullName,
			EmployeeID: &emp.ID,
		})
		if err == nil {
			user = newUser
		}
	} else if user != nil && (req.Username != "" || req.Password != "") {
		_ = h.userSvc.UpdateUserCredentials(c.Context(), user.ID, req.Username, req.Password)
	}

	if user != nil {
		// Only clear scopes if the request explicitly included the array (even if empty, it'll be a non-nil array in JSON). 
		// Wait, if it's not provided, we don't clear? Actually let's just clear and assign.
		_ = h.userSvc.ClearUserScopes(c.Context(), user.ID)
		if len(req.ScopeIDs) > 0 {
			for _, sid := range req.ScopeIDs {
				if parsedSID, err := uuid.Parse(sid); err == nil {
					_ = h.userSvc.GrantScopeAccess(c.Context(), user.ID, parsedSID, user.ID)
				}
			}
		}

		if req.RoleName != "" && h.roleRepo != nil {
			_ = h.userSvc.ClearUserRoles(c.Context(), user.ID)
			role, _ := h.roleRepo.FindByName(c.Context(), identity.RoleName(req.RoleName))
			if role != nil {
				_ = h.userSvc.AssignRole(c.Context(), user.ID, role.ID, &emp.ScopeID)
			}
		}
	}

	return c.JSON(fiber.Map{"data": emp})
}

