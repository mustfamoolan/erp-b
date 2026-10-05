package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"m3aml-erp/bootstrap"
	"m3aml-erp/config"
	"m3aml-erp/internal/api/handlers"
	apimw "m3aml-erp/internal/api/middleware"
	appaccounting "m3aml-erp/internal/application/accounting"
	appaudit "m3aml-erp/internal/application/audit"
	"m3aml-erp/internal/application/auth"
	appfinance "m3aml-erp/internal/application/finance"
	appidentity "m3aml-erp/internal/application/identity"
	appinv "m3aml-erp/internal/application/inventory"
	apporg "m3aml-erp/internal/application/organization"
	apppurchasing "m3aml-erp/internal/application/purchasing"
	appwf "m3aml-erp/internal/application/workflow"
	"m3aml-erp/internal/domain/identity"
	"m3aml-erp/internal/repositories/postgres"
	reportingrepo "m3aml-erp/internal/repositories/postgres/reporting"
	appreporting "m3aml-erp/internal/application/reporting"
	reportinghandlers "m3aml-erp/internal/api/handlers/reporting"
	
	appmasterdata "m3aml-erp/internal/application/masterdata"
	mdpostgres "m3aml-erp/internal/repositories/postgres/masterdata"
	orghandlers "m3aml-erp/internal/api/handlers/organization"
)

func main() {
	// ── 1. Bootstrap ──────────────────────────────────────────
	bootstrap.InitializeConfig()
	bootstrap.InitializeLogger()
	bootstrap.InitializeCache()
	bootstrap.InitializeDatabase()
	bootstrap.RunMigrations() // always runs migrations before serving
	bootstrap.DB.Exec("UPDATE financial_requests SET status = 'SUBMITTED' WHERE status = 'DRAFT'")

	// ── 2. Repositories ───────────────────────────────────────
	// Phase 1: Identity & Organization
	userRepo := postgres.NewUserRepository(bootstrap.DB)
	roleRepo := postgres.NewRoleRepository(bootstrap.DB)
	scopeRepo := postgres.NewScopeRepository(bootstrap.DB)
	employeeRepo := postgres.NewEmployeeRepository(bootstrap.DB)
	areaRepo := postgres.NewAreaRepository(bootstrap.DB)

	// Phase 2: Accounting
	accountRepo := postgres.NewAccountRepository(bootstrap.DB)
	periodRepo := postgres.NewAccountingPeriodRepository(bootstrap.DB)
	fyRepo := postgres.NewFiscalYearRepository(bootstrap.DB)
	journalRepo := postgres.NewJournalEntryRepository(bootstrap.DB)

	// ── 3. Application Services ───────────────────────────────
	auditSvc := appaudit.NewAuditService(postgres.NewAuditRepository(bootstrap.DB))
	authSvc := auth.NewAuthService(userRepo, scopeRepo, roleRepo, areaRepo, config.Global.App.Key, auditSvc)
	userSvc := appidentity.NewUserService(userRepo, roleRepo)
	areaSvc := apporg.NewAreaService(areaRepo)
	accountingSvc := appaccounting.NewAccountingService(journalRepo, accountRepo, periodRepo, auditSvc)
	scopeSvc := apporg.NewScopeService(scopeRepo, accountRepo, bootstrap.DB, auditSvc)

	companySettingsRepo := postgres.NewCompanySettingsRepository(bootstrap.DB)
	companySettingsSvc := apporg.NewCompanySettingsService(companySettingsRepo, auditSvc)

	cashboxRepo := postgres.NewCashboxRepository(bootstrap.DB)
	cashTxRepo := postgres.NewCashTransactionRepository(bootstrap.DB)
	financeSvc := appfinance.NewFinanceService(cashboxRepo, cashTxRepo, accountingSvc, auditSvc, accountRepo)

	// Phase 4.5: Master Data
	mdRepo := mdpostgres.NewMasterDataRepository(bootstrap.DB)
	masterDataSvc := appmasterdata.NewMasterDataService(mdRepo, accountRepo, auditSvc)

	// Multi-Currency: Exchange Rate Service — migration 000007
	exchangeRateRepo := postgres.NewExchangeRateRepository(bootstrap.DB)
	exchangeRateSvc  := appaccounting.NewExchangeRateService(exchangeRateRepo, auditSvc)

	// ── 4. HTTP Handlers ─────────────────────────────────────
	authHandler := handlers.NewAuthHandler(authSvc)
	userHandler := handlers.NewUserHandler(userSvc, roleRepo, scopeRepo)
	roleHandler := handlers.NewRoleHandler(roleRepo)
	scopeHandler := handlers.NewScopeHandler(scopeSvc, financeSvc)
	employeeHandler := handlers.NewEmployeeHandler(employeeRepo, scopeRepo, userSvc, roleRepo, areaRepo, userRepo)
	areaHandler := handlers.NewAreaHandler(areaSvc)
	companySettingsHandler := orghandlers.NewCompanySettingsHandler(companySettingsSvc)
	accountingHandler   := handlers.NewAccountingHandler(accountingSvc, accountRepo, periodRepo, fyRepo, userRepo)
	exchangeRateHandler := handlers.NewExchangeRateHandler(exchangeRateSvc)
	masterDataHandler   := handlers.NewMasterDataHandler(masterDataSvc)

	// ── 5. Fiber ─────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		AppName:      config.Global.App.Name + " ERP",
		ErrorHandler: errorHandler,
		BodyLimit:    25 * 1024 * 1024, // 25 MB max body limit for PDF and attachment uploads
	})
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
	}))

	// Serve uploaded files statically
	app.Static("/uploads", "./uploads")

	// ── 6. Global Middlewares (Phase 11) ──────────────────────
	app.Use(apimw.SecurityHeaders())
	app.Use(apimw.RateLimiter())

	// ── 7. Routes ─────────────────────────────────────────────
	api := app.Group("/api")
	v1 := api.Group("/v1")

	// Health check — public
	v1.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "version": "v1", "service": config.Global.App.Name})
	})

	// Auth — public
	v1.Post("/auth/login", authHandler.Login)

	// ─── Protected routes (require valid JWT) ────────────────
	p := v1.Group("", apimw.AuthMiddleware(authSvc, userRepo))

	// Generic Upload
	p.Post("/upload", func(c *fiber.Ctx) error {
		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "No file uploaded"})
		}
		if err := os.MkdirAll("./uploads", 0755); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to create uploads directory"})
		}
		cleanName := strings.ReplaceAll(filepath.Base(file.Filename), " ", "_")
		filename := fmt.Sprintf("%d_%s", time.Now().Unix(), cleanName)
		if err := c.SaveFile(file, fmt.Sprintf("./uploads/%s", filename)); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to save file"})
		}
		// Return full URL
		port := config.Global.App.Port
		if port == "" {
			port = "8080"
		}
		return c.JSON(fiber.Map{"url": fmt.Sprintf("http://localhost:%s/uploads/%s", port, filename)})
	})

	// ════════════════════════════════════════════════════════
	// PHASE 1 — IDENTITY & ORGANIZATION FOUNDATION (§12)
	// ════════════════════════════════════════════════════════

	// Roles — §9
	roles := p.Group("/roles")
	roles.Get("/", apimw.RequirePermission(userRepo, identity.PermRolesView), roleHandler.ListRoles)
	roles.Get("/:id", apimw.RequirePermission(userRepo, identity.PermRolesView), roleHandler.GetRole)
	roles.Post("/", apimw.RequirePermission(userRepo, identity.PermRolesCreate), roleHandler.CreateRole)
	roles.Put("/:id", apimw.RequirePermission(userRepo, identity.PermRolesUpdate), roleHandler.UpdateRole)

	// Permissions mapping
	p.Get("/permissions", apimw.RequirePermission(userRepo, identity.PermRolesView), roleHandler.ListPermissions)

	// Users — §8
	users := p.Group("/users",
		apimw.RequirePermission(userRepo, identity.PermUsersView),
	)
	users.Get("/", userHandler.ListUsers)
	users.Post("/", userHandler.CreateUser)
	users.Get("/:id", userHandler.GetUser)

	// User → Role assignment — §9
	users.Post("/:id/roles", userHandler.AssignRole)
	users.Delete("/:id/roles/:role_id", userHandler.RevokeRole)

	// User → Scope access — §8, Rule 7
	// This is the ONLY way a user gets scope access. No automatic grants.
	users.Post("/:id/scopes", userHandler.GrantScope)
	users.Delete("/:id/scopes/:scope_id", userHandler.RevokeScope)

	// Scopes / Factories — §5
	scopes := p.Group("/scopes")
	scopes.Get("/", scopeHandler.ListScopes)
	scopes.Get("/factories", scopeHandler.ListFactories)
	scopes.Get("/:id", scopeHandler.GetScope)
	scopes.Post("/",
		apimw.RequirePermission(userRepo, identity.PermFactoriesCreate),
		scopeHandler.CreateScope,
	)
	scopes.Put("/:id",
		apimw.RequirePermission(userRepo, identity.PermFactoriesUpdate),
		scopeHandler.UpdateScope,
	)

	// Employees — §8
	// ScopeMiddleware enforces Rule 7 on scope-specific queries
	employees := p.Group("/employees",
		apimw.RequirePermission(userRepo, identity.PermEmployeesView),
	)
	employees.Get("/", employeeHandler.ListEmployees) // ?scope_id= required
	employees.Post("/", employeeHandler.CreateEmployee)
	employees.Get("/:id", employeeHandler.GetEmployee)
	employees.Put("/:id", employeeHandler.UpdateEmployee)

	// Areas — Phase 1
	areas := p.Group("/areas",
		apimw.RequirePermission(userRepo, identity.PermAreasView),
	)
	areas.Get("/", areaHandler.ListAreas)
	areas.Post("/", areaHandler.CreateArea)
	areas.Get("/:id", areaHandler.GetArea)
	areas.Put("/:id", areaHandler.UpdateArea)

	// Company Settings
	settings := p.Group("/settings")
	settings.Get("/company", companySettingsHandler.Get)
	settings.Put("/company",
		apimw.RequirePermission(userRepo, "settings.manage"),
		companySettingsHandler.Update,
	)

	// ════════════════════════════════════════════════════════
	// PHASE 2 — ACCOUNTING CORE (§13)
	// ════════════════════════════════════════════════════════
	acct := p.Group("/accounting")

	// Chart of Accounts
	acct.Get("/accounts", accountingHandler.ListAccounts)
	acct.Get("/accounts/:id", accountingHandler.GetAccount)
	acct.Get("/accounts/:id/balance", accountingHandler.GetAccountBalance)

	// Fiscal Years
	acct.Get("/fiscal-years", func(c *fiber.Ctx) error {
		fys, err := fyRepo.FindAll(c.Context())
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to fetch fiscal years"})
		}
		return c.JSON(fiber.Map{"data": fys})
	})

	// Accounting Periods
	acct.Get("/periods", accountingHandler.ListPeriods)
	acct.Post("/periods/:id/close",
		apimw.RequirePermission(userRepo, identity.PermAccountingClosePeriod),
		accountingHandler.ClosePeriod,
	)

	// Journal Entries
	acct.Post("/journal",
		apimw.RequirePermission(userRepo, identity.PermAccountingPost),
		accountingHandler.CreateJournalEntry,
	)
	acct.Post("/journal/:id/post",
		apimw.RequirePermission(userRepo, identity.PermAccountingPost),
		accountingHandler.PostJournalEntry,
	)

	// ════════════════════════════════════════════════════════
	// PHASE 3 — CASHBOXES & FINANCE (§23)
	// ════════════════════════════════════════════════════════

	financeHandler := handlers.NewFinanceHandler(financeSvc, cashboxRepo, userRepo)

	fin := p.Group("/finance")
	// Rule 32: Every API endpoint MUST have Authorization
	fin.Get("/cashboxes",
		apimw.RequirePermission(userRepo, identity.PermCashboxView),
		financeHandler.ListCashboxes,
	)
	fin.Post("/cashboxes",
		apimw.RequirePermission(userRepo, identity.PermCashboxTransfer), // only cashbox managers can create cashboxes
		financeHandler.CreateCashbox,
	)
	fin.Post("/cashboxes/transfer",
		apimw.RequirePermission(userRepo, identity.PermCashboxTransfer),
		financeHandler.TransferFunds,
	)
	fin.Post("/cashboxes/:id/opening-balance",
		apimw.RequirePermission(userRepo, identity.PermCashboxTransfer),
		financeHandler.EstablishOpeningBalance,
	)
	fin.Get("/cashboxes/:id/transactions",
		apimw.RequirePermission(userRepo, identity.PermCashboxView),
		financeHandler.GetCashboxTransactions,
	)
	fin.Post("/cashboxes/:id/withdraw",
		apimw.RequirePermission(userRepo, identity.PermCashboxCustodyManage),
		financeHandler.WithdrawCustody,
	)
	fin.Post("/cashboxes/:id/deposit",
		apimw.RequirePermission(userRepo, identity.PermCashboxCustodyManage),
		financeHandler.DepositCustody,
	)
	// ════════════════════════════════════════════════════════
	// ════════════════════════════════════════════════════════
	// PHASE 4 & 5 — INVENTORY & WAREHOUSE FOUNDATION (§25-30)
	// ════════════════════════════════════════════════════════
	itemCategoryRepo := postgres.NewItemCategoryRepository(bootstrap.DB)
	unitRepo := postgres.NewUnitOfMeasureRepository(bootstrap.DB)
	itemRepo := postgres.NewItemRepository(bootstrap.DB)
	variantRepo := postgres.NewItemVariantRepository(bootstrap.DB)
	warehouseRepo := postgres.NewWarehouseRepository(bootstrap.DB)
	stockRepo := postgres.NewStockMovementRepository(bootstrap.DB)

	invSvc := appinv.NewInventoryService(itemCategoryRepo, unitRepo, itemRepo, variantRepo, warehouseRepo, stockRepo, auditSvc)
	invHandler := handlers.NewInventoryHandler(invSvc)

	inv := p.Group("/inventory")
	inv.Get("/categories", apimw.RequirePermission(userRepo, identity.PermWarehouseView), invHandler.GetCategories)
	inv.Post("/categories", apimw.RequirePermission(userRepo, identity.PermWarehouseView), invHandler.CreateCategory)
	inv.Get("/units", apimw.RequirePermission(userRepo, identity.PermWarehouseView), invHandler.GetUnits)
	inv.Get("/items", apimw.RequirePermission(userRepo, identity.PermWarehouseView), invHandler.GetItems)
	
	// Phase 4: Master Data
	inv.Post("/items", apimw.RequirePermission(userRepo, identity.PermWarehouseView), invHandler.CreateItem)
	inv.Post("/variants", apimw.RequirePermission(userRepo, identity.PermWarehouseView), invHandler.CreateVariant)
	
	// Phase 5: Warehouse & Stock
	inv.Post("/warehouses", apimw.RequirePermission(userRepo, identity.PermWarehouseView), invHandler.CreateWarehouse)
	inv.Get("/warehouses/:scopeID", apimw.RequirePermission(userRepo, identity.PermWarehouseView), invHandler.GetWarehousesByScope)
	inv.Post("/movements", apimw.RequirePermission(userRepo, identity.PermWarehouseReceive), invHandler.RecordMovement)
	inv.Post("/transfers", apimw.RequirePermission(userRepo, identity.PermWarehouseTransfer), invHandler.TransferStock)

	// ════════════════════════════════════════════════════════
	// PHASE 6 — FINANCIAL / MATERIAL REQUEST (§31, §32, §33)
	// ════════════════════════════════════════════════════════
	freqRepo := postgres.NewFinancialRequestRepository(bootstrap.DB)
	freqItemRepo := postgres.NewRequestItemRepository(bootstrap.DB)
	freqHistoryRepo := postgres.NewRequestHistoryRepository(bootstrap.DB)

	docNumFn := func(ctx context.Context) (string, error) {
		return postgres.GenerateDocumentNumber(ctx, bootstrap.DB, "FIN")
	}

	requestSvc := appwf.NewRequestService(freqRepo, freqItemRepo, freqHistoryRepo, auditSvc, docNumFn)
	requestHandler := handlers.NewRequestHandler(requestSvc, userRepo, financeSvc, scopeRepo, cashboxRepo)

	req := p.Group("/requests")

	// Factory Accountant — Create, view own requests
	req.Post("", apimw.RequirePermission(userRepo, identity.PermRequestCreate), requestHandler.CreateRequest)
	req.Get("/scope/:scopeID", apimw.RequirePermission(userRepo, identity.PermRequestView), requestHandler.GetByScope)
	req.Get("/:id", apimw.RequirePermission(userRepo, identity.PermRequestView), requestHandler.GetByID)
	req.Get("/:id/history", apimw.RequirePermission(userRepo, identity.PermRequestView), requestHandler.GetHistory)

	// Factory Accountant — Workflow: submit, receive, cancel
	req.Post("/:id/submit", apimw.RequirePermission(userRepo, identity.PermRequestCreate), requestHandler.Submit)
	req.Post("/:id/receive", apimw.RequirePermission(userRepo, identity.PermRequestReceive), requestHandler.MarkReceived)
	req.Post("/:id/complete", apimw.RequirePermission(userRepo, identity.PermRequestReceive), requestHandler.Complete)
	req.Post("/:id/cancel", apimw.RequirePermission(userRepo, identity.PermRequestCreate), requestHandler.Cancel)

	// Administration Auditor — Review
	req.Get("", apimw.RequirePermission(userRepo, identity.PermRequestReview), requestHandler.GetAll)
	req.Post("/:id/review", apimw.RequirePermission(userRepo, identity.PermRequestReview), requestHandler.StartReview)
	req.Post("/:id/approve", apimw.RequirePermission(userRepo, identity.PermRequestReview), requestHandler.Approve)

	// Administration Accountant — Accountant Approve & Payment
	req.Post("/:id/accountant-approve", apimw.RequirePermission(userRepo, identity.PermRequestApprove), requestHandler.AccountantApprove)
	req.Post("/:id/payment-pending", apimw.RequirePermission(userRepo, identity.PermRequestPay), requestHandler.MarkPaymentPending)
	req.Post("/:id/factory-receipt-pending", apimw.RequirePermission(userRepo, identity.PermRequestPay), requestHandler.MarkFactoryReceiptPending)

	// Return Revision (Auditor / Management Accountant)
	req.Post("/:id/return-revision", apimw.RequirePermission(userRepo, identity.PermRequestReturnRevision), requestHandler.ReturnRevision)

	// Central Cashier — Disburse & Deliver
	req.Post("/:id/disburse", apimw.RequirePermission(userRepo, identity.PermRequestDisburse), requestHandler.Disburse)
	req.Post("/:id/deliver", apimw.RequirePermission(userRepo, identity.PermRequestDeliver), requestHandler.Deliver)

	// Rejection (Auditor / Accountant)
	req.Post("/:id/reject", apimw.RequirePermission(userRepo, identity.PermRequestReject), requestHandler.Reject)

	// ════════════════════════════════════════════════════════
	// PHASE 7 — AUDIT SYSTEM (§34, §35)
	// ════════════════════════════════════════════════════════
	auditHandler := handlers.NewAuditHandler(auditSvc)

	aud := p.Group("/audit")
	aud.Get("", apimw.RequirePermission(userRepo, identity.PermAuditView), auditHandler.GetAll)
	aud.Get("/entity/:type/:id", apimw.RequirePermission(userRepo, identity.PermAuditView), auditHandler.GetByEntity)
	aud.Get("/scope/:scopeID", apimw.RequirePermission(userRepo, identity.PermAuditView), auditHandler.GetByScope)
	aud.Get("/user/:userID", apimw.RequirePermission(userRepo, identity.PermAuditView), auditHandler.GetByUser)

	// ════════════════════════════════════════════════════════
	// PHASE 11 — REPORTING FOUNDATION
	// ════════════════════════════════════════════════════════
	reportingRepo := reportingrepo.NewReportingRepository(bootstrap.DB)
	reportingSvc := appreporting.NewReportingService(reportingRepo)
	reportingHandler := reportinghandlers.NewReportingHandler(reportingSvc)

	rep := p.Group("/reports", apimw.RequirePermission(userRepo, identity.PermReportsView))
	rep.Get("/accounting/statement/:accountId", reportingHandler.GetAccountStatement)
	rep.Get("/accounting/trial-balance", reportingHandler.GetTrialBalance)
	rep.Get("/inventory/stock-balance", reportingHandler.GetStockBalance)

	// ════════════════════════════════════════════════════════
	// PHASE 4.5 — MASTER DATA 
	// ════════════════════════════════════════════════════════
	md := p.Group("/master-data")
	
	md.Get("/request-types", masterDataHandler.GetRequestTypes)
	md.Post("/request-types", apimw.RequirePermission(userRepo, identity.PermMasterDataCreate), masterDataHandler.CreateRequestType)
	md.Put("/request-types/:id", apimw.RequirePermission(userRepo, identity.PermMasterDataUpdate), masterDataHandler.UpdateRequestType)
	md.Patch("/request-types/:id/toggle", apimw.RequirePermission(userRepo, identity.PermMasterDataToggle), masterDataHandler.ToggleRequestType)

	md.Get("/expense-categories", masterDataHandler.GetExpenseCategories)
	md.Post("/expense-categories", apimw.RequirePermission(userRepo, identity.PermMasterDataCreate), masterDataHandler.CreateExpenseCategory)
	md.Put("/expense-categories/:id", apimw.RequirePermission(userRepo, identity.PermMasterDataUpdate), masterDataHandler.UpdateExpenseCategory)
	md.Patch("/expense-categories/:id/toggle", apimw.RequirePermission(userRepo, identity.PermMasterDataToggle), masterDataHandler.ToggleExpenseCategory)

	md.Get("/factory-expense-types", masterDataHandler.GetFactoryExpenseTypes)
	md.Post("/factory-expense-types", apimw.RequirePermission(userRepo, identity.PermMasterDataCreate), masterDataHandler.CreateFactoryExpenseType)
	md.Put("/factory-expense-types/:id", apimw.RequirePermission(userRepo, identity.PermMasterDataUpdate), masterDataHandler.UpdateFactoryExpenseType)
	md.Patch("/factory-expense-types/:id/toggle", apimw.RequirePermission(userRepo, identity.PermMasterDataToggle), masterDataHandler.ToggleFactoryExpenseType)

	// ════════════════════════════════════════════════════════
	// EXCHANGE RATES — Multi-Currency (migration 000007)
	// Rule 20: Only SUPER_ADMIN may update rates — enforced here.
	// ════════════════════════════════════════════════════════
	ex := p.Group("/exchange-rates")
	ex.Get("/",       exchangeRateHandler.GetCurrentRates)
	ex.Get("/history", exchangeRateHandler.GetHistory)
	ex.Get("/convert", exchangeRateHandler.ConvertAmount) // preview only — no DB write
	ex.Post("/",
		apimw.RequirePermission(userRepo, identity.PermExchangeRateUpdate), // SUPER_ADMIN permission guard
		exchangeRateHandler.SetRate,
	)

	// ════════════════════════════════════════════════════════
	// PHASE 9 — PURCHASING (Purchase Invoices)
	// ════════════════════════════════════════════════════════
	purchaseInvoiceRepo := postgres.NewPurchaseInvoiceRepository(bootstrap.DB)
	purchaseInvoiceItemRepo := postgres.NewPurchaseInvoiceItemRepository(bootstrap.DB)
	
	// We need a document number generator for purchasing
	purchDocNumFn := func(ctx context.Context) (string, error) {
		return "PI-" + time.Now().Format("20060102150405"), nil
	}

	purchasingSvc := apppurchasing.NewPurchasingService(
		purchaseInvoiceRepo,
		purchaseInvoiceItemRepo,
		cashTxRepo,
		cashboxRepo,
		accountRepo,
		accountingSvc,
		auditSvc,
		purchDocNumFn,
	)
	purchasingHandler := handlers.NewPurchasingHandler(purchasingSvc, userRepo)

	purch := p.Group("/purchases")
	purch.Post("/",
		apimw.RequirePermission(userRepo, identity.PermPurchasesCreate),
		purchasingHandler.CreateInvoice,
	)
	purch.Get("/",
		apimw.RequirePermission(userRepo, identity.PermPurchasesView),
		purchasingHandler.GetInvoices,
	)
	purch.Get("/:id",
		apimw.RequirePermission(userRepo, identity.PermPurchasesView),
		purchasingHandler.GetInvoice,
	)
	purch.Post("/:id/pay",
		apimw.RequirePermission(userRepo, identity.PermRequestPay),
		purchasingHandler.PayInvoice,
	)

	// ════════════════════════════════════════════════════════
	// PHASE 10 — FACTORY EXPENSES (§28)
	// ════════════════════════════════════════════════════════
	expenseRepo := postgres.NewExpenseRepository(bootstrap.DB)
	expenseSvc := appfinance.NewExpenseService(
		expenseRepo,
		cashboxRepo,
		cashTxRepo,
		accountingSvc,
		accountRepo,
		mdRepo,
	)
	expenseHandler := handlers.NewExpenseHandler(expenseSvc, userRepo)

	exp := p.Group("/expenses")
	exp.Post("/",
		apimw.RequirePermission(userRepo, identity.PermExpensesCreate),
		expenseHandler.RecordExpense,
	)

	// ── 7. Start ─────────────────────────────────────────────
	port := config.Global.App.Port
	if port == "" {
		port = "8080"
	}
	fmt.Printf("🚀 %s ERP started on :%s\n", config.Global.App.Name, port)
	log.Fatal(app.Listen(":" + port))
}

// errorHandler centralizes all error responses.
func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}
	return c.Status(code).JSON(fiber.Map{"error": err.Error()})
}
