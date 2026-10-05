package identity

import (
	"time"

	"github.com/google/uuid"
)

// PermissionCode is the string identifier of a permission (Roadmap §10).
// Format: "resource.action"
type PermissionCode string

// Full permission catalogue. Every code here MUST be seeded in a migration
// (see 000019_granular_permissions) and every protected endpoint uses exactly one.
// Deletion is intentionally absent: master/financial data is deactivated, never deleted (AGENTS.md §1.13, §2.5.2).
const (
	// ── Users ───────────────────────────────────────────────
	PermUsersView         PermissionCode = "users.view"
	PermUsersCreate       PermissionCode = "users.create"
	PermUsersUpdate       PermissionCode = "users.update"
	PermUsersAssignRoles  PermissionCode = "users.assign_roles"
	PermUsersAssignScopes PermissionCode = "users.assign_scopes"

	// ── Roles & permissions ─────────────────────────────────
	PermRolesView   PermissionCode = "roles.view"
	PermRolesCreate PermissionCode = "roles.create"
	PermRolesUpdate PermissionCode = "roles.update"

	// ── Factories / scopes & areas ──────────────────────────
	PermFactoriesView   PermissionCode = "factories.view"
	PermFactoriesCreate PermissionCode = "factories.create"
	PermFactoriesUpdate PermissionCode = "factories.update"
	PermAreasView       PermissionCode = "areas.view"
	PermAreasCreate     PermissionCode = "areas.create"
	PermAreasUpdate     PermissionCode = "areas.update"

	// ── Employees ───────────────────────────────────────────
	PermEmployeesView   PermissionCode = "employees.view"
	PermEmployeesCreate PermissionCode = "employees.create"
	PermEmployeesUpdate PermissionCode = "employees.update"

	// ── Master data (request types, expense categories) ─────
	PermMasterDataView   PermissionCode = "master_data.view"
	PermMasterDataCreate PermissionCode = "master_data.create"
	PermMasterDataUpdate PermissionCode = "master_data.update"
	PermMasterDataToggle PermissionCode = "master_data.toggle"

	// ── Exchange rates ──────────────────────────────────────
	PermExchangeRateView   PermissionCode = "exchange_rate.view"
	PermExchangeRateUpdate PermissionCode = "exchange_rate.update"

	// ── Accounting ──────────────────────────────────────────
	PermAccountingView          PermissionCode = "accounting.view"
	PermAccountingJournalCreate PermissionCode = "accounting.journal_create"
	PermAccountingPost          PermissionCode = "accounting.post"
	PermAccountingReverse       PermissionCode = "accounting.reverse"
	PermAccountingClosePeriod   PermissionCode = "accounting.close_period"

	// ── Cashboxes ───────────────────────────────────────────
	PermCashboxView           PermissionCode = "cashbox.view"
	PermCashboxCreate         PermissionCode = "cashbox.create"
	PermCashboxOpeningBalance PermissionCode = "cashbox.opening_balance"
	PermCashboxTransfer       PermissionCode = "cashbox.transfer"
	PermCashboxCustodyManage  PermissionCode = "cashbox.custody.manage"

	// ── Factory expenses ────────────────────────────────────
	PermExpensesCreate PermissionCode = "expenses.create"

	// ── Inventory & warehouses ──────────────────────────────
	PermWarehouseView     PermissionCode = "warehouse.view"
	PermWarehouseCreate   PermissionCode = "warehouse.create"
	PermItemsCreate       PermissionCode = "items.create"
	PermWarehouseReceive  PermissionCode = "warehouse.receive"
	PermWarehouseIssue    PermissionCode = "warehouse.issue"
	PermWarehouseTransfer PermissionCode = "warehouse.transfer"
	PermWarehouseAdjust   PermissionCode = "warehouse.adjust"

	// ── Financial requests (workflow) ───────────────────────
	PermRequestView         PermissionCode = "request.view"
	PermRequestViewAll      PermissionCode = "request.view_all"
	PermRequestCreate       PermissionCode = "request.create"
	PermRequestSubmit       PermissionCode = "request.submit"
	PermRequestCancel       PermissionCode = "request.cancel"
	PermRequestAttach       PermissionCode = "request.attach"
	PermRequestReview       PermissionCode = "request.review"
	PermRequestAuditApprove PermissionCode = "request.audit_approve"
	PermRequestApprove      PermissionCode = "request.approve"
	PermRequestReject          PermissionCode = "request.reject"
	PermRequestReturnRevision  PermissionCode = "request.return_revision"
	PermRequestDisburse        PermissionCode = "request.disburse"
	PermRequestDeliver         PermissionCode = "request.deliver"
	PermRequestPay             PermissionCode = "request.pay"
	PermRequestReceive         PermissionCode = "request.receive"

	// ── Purchases ───────────────────────────────────────────
	PermPurchasesView    PermissionCode = "purchases.view"
	PermPurchasesCreate  PermissionCode = "purchases.create"
	PermPurchasesApprove PermissionCode = "purchases.approve"
	PermPurchasesPay     PermissionCode = "purchases.pay"

	// ── Audit & reports ─────────────────────────────────────
	PermAuditView   PermissionCode = "audit.view"
	PermReportsView PermissionCode = "reports.view"

	// ── Scope ───────────────────────────────────────────────
	PermScopeAll PermissionCode = "scope.all"
)

// Permission defines a granular action that can be allowed or denied (Roadmap §10).
type Permission struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code        PermissionCode `gorm:"type:varchar(100);not null;uniqueIndex"`
	DisplayName string         `gorm:"type:varchar(150);not null"`
	Description string         `gorm:"type:text"`
	Group       string         `gorm:"column:grp;type:varchar(50);not null"` // e.g. "request", "cashbox"
	CreatedAt   time.Time
}

func (Permission) TableName() string {
	return "permissions"
}

// RolePermission assigns a Permission to a Role (Roadmap §10).
type RolePermission struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RoleID       uuid.UUID      `gorm:"type:uuid;not null;index"`
	PermissionID uuid.UUID      `gorm:"type:uuid;not null;index"`
	CreatedAt    time.Time
}

func (RolePermission) TableName() string {
	return "role_permissions"
}
