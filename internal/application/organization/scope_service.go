package organization

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	appaudit "m3aml-erp/internal/application/audit"
	"m3aml-erp/bootstrap"
	"m3aml-erp/internal/domain/accounting"
	domainaudit "m3aml-erp/internal/domain/audit"
	"m3aml-erp/internal/domain/organization"
	"m3aml-erp/internal/repositories"
)

var ErrScopeNotFound  = errors.New("scope not found")
var ErrDuplicateScope = errors.New("scope with this name or code already exists")

// ScopeService handles business operations on organizational scopes.
// Roadmap §5 — every record must trace to a scope.
type ScopeService struct {
	scopeRepo   repositories.ScopeRepository
	accountRepo repositories.AccountRepository
	db          *gorm.DB
	auditSvc    *appaudit.AuditService
}

func NewScopeService(
	scopeRepo repositories.ScopeRepository,
	accountRepo repositories.AccountRepository,
	db *gorm.DB,
	auditSvc *appaudit.AuditService,
) *ScopeService {
	return &ScopeService{
		scopeRepo:   scopeRepo,
		accountRepo: accountRepo,
		db:          db,
		auditSvc:    auditSvc,
	}
}

// CreateFactory creates a new Factory scope atomically with its cashbox, account, and warehouses.
// Permission check is done in the middleware — this service enforces business rules.
func (s *ScopeService) CreateFactory(ctx context.Context, req organization.OrganizationScope, userID uuid.UUID) (*organization.OrganizationScope, error) {
	if req.Name == "" || req.Code == "" {
		return nil, fmt.Errorf("factory name and code are required")
	}

	req.ID = uuid.New()
	req.Type = organization.ScopeTypeFactory
	req.Status = organization.ScopeStatusActive

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, "tx", tx)

		// 1. Save Scope
		if err := s.scopeRepo.Save(txCtx, &req); err != nil {
			return fmt.Errorf("save scope: %w", err)
		}

		now := time.Now()

		// 2. Find Parent Account for Cashboxes (FACTORY_CASHBOX_PARENT)
		var mapping struct {
			AccountID uuid.UUID
		}
		if err := tx.Table("account_mappings").Select("account_id").Where("key = ?", "FACTORY_CASHBOX_PARENT").Take(&mapping).Error; err != nil {
			return fmt.Errorf("FACTORY_CASHBOX_PARENT mapping not found: %w", err)
		}

		parentAccount, err := s.accountRepo.FindByID(txCtx, mapping.AccountID)
		if err != nil || parentAccount == nil {
			return fmt.Errorf("parent account not found: %w", err)
		}

		// Generate next sub-account code
		children, err := s.accountRepo.FindChildren(txCtx, parentAccount.ID)
		if err != nil {
			return fmt.Errorf("find children: %w", err)
		}

		maxCode := 0
		for _, child := range children {
			if strings.HasPrefix(child.Code, parentAccount.Code) {
				suffix := strings.TrimPrefix(child.Code, parentAccount.Code)
				if num, err := strconv.Atoi(suffix); err == nil {
					if num > maxCode {
						maxCode = num
					}
				}
			}
		}

		nextNum := maxCode + 1
		if nextNum == 1 && len(children) > 0 { // Just in case it didn't parse
			nextNum = len(children) + 1
		}
		newCode := fmt.Sprintf("%s%04d", parentAccount.Code, nextNum)

		for {
			acc, _ := s.accountRepo.FindByCode(txCtx, newCode)
			if acc == nil {
				break
			}
			nextNum++
			newCode = fmt.Sprintf("%s%04d", parentAccount.Code, nextNum)
		}

		newAccount := accounting.Account{
			ID:          uuid.New(),
			Code:        newCode,
			Name:        "صندوق " + req.Name,
			Type:        parentAccount.Type,
			IsPostable:  true,
			ParentID:    &parentAccount.ID,
			ScopeID:     &req.ID,
			SortOrder:   parentAccount.SortOrder*10000 + nextNum,
		}

		if err := s.accountRepo.Save(txCtx, &newAccount); err != nil {
			return fmt.Errorf("create account: %w", err)
		}

		// 3. Create Cashbox (Imprest 0, TargetOpening 0)
		cashboxID := uuid.New()
		if err := tx.Exec(`
			INSERT INTO cashboxes (id, scope_id, name, account_id, currency, target_opening_balance, imprest_balance, status, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, cashboxID, req.ID, newAccount.Name, newAccount.ID, "IQD", 0.0, 0.0, "ACTIVE", now, now).Error; err != nil {
			return fmt.Errorf("create cashbox: %w", err)
		}

		// 4. Create Default Warehouses
		if err := tx.Exec(`
			INSERT INTO warehouses (id, scope_id, code, name, type, is_active, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, uuid.New(), req.ID, "WH-RAW-"+req.Code, "مستودع المواد الأولية", "RAW_MATERIAL", true, now, now).Error; err != nil {
			return fmt.Errorf("create raw warehouse: %w", err)
		}
		
		if err := tx.Exec(`
			INSERT INTO warehouses (id, scope_id, code, name, type, is_active, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, uuid.New(), req.ID, "WH-FIN-"+req.Code, "مستودع المنتجات التامة", "FINISHED_GOODS", true, now, now).Error; err != nil {
			return fmt.Errorf("create finished warehouse: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 5. Audit
	if s.auditSvc != nil {
		s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			EntityID:   &req.ID,
			EntityType: "organization_scope",
			Action:     domainaudit.AuditCreate,
			UserID:     userID,
			NewValues:  req,
		})
	}

	bootstrap.InvalidateOrgCache()

	return &req, nil
}

// GetAllScopes returns all scopes ordered by type and name.
func (s *ScopeService) GetAllScopes(ctx context.Context) ([]organization.OrganizationScope, error) {
	return bootstrap.CacheRemember("org:scopes:all", 12*time.Hour, func() ([]organization.OrganizationScope, error) {
		return s.scopeRepo.FindAll(ctx)
	})
}

// GetFactories returns all active factory scopes.
func (s *ScopeService) GetFactories(ctx context.Context) ([]organization.OrganizationScope, error) {
	return bootstrap.CacheRemember("org:factories:all", 12*time.Hour, func() ([]organization.OrganizationScope, error) {
		return s.scopeRepo.FindByType(ctx, organization.ScopeTypeFactory)
	})
}

// GetScopeByID returns a scope by its ID.
func (s *ScopeService) GetScopeByID(ctx context.Context, id uuid.UUID) (*organization.OrganizationScope, error) {
	return s.scopeRepo.FindByID(ctx, id)
}

// UpdateScope updates a scope and optionally its cashbox target opening balance.
func (s *ScopeService) UpdateScope(ctx context.Context, scope *organization.OrganizationScope, targetOpeningBalance float64, userID uuid.UUID) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, "tx", tx)

		if err := s.scopeRepo.Update(txCtx, scope); err != nil {
			return err
		}

		if targetOpeningBalance >= 0 {
			if err := tx.Exec(`
				UPDATE cashboxes 
				SET target_opening_balance = ? 
				WHERE scope_id = ? 
				AND NOT EXISTS (
					SELECT 1 FROM cash_transactions 
					WHERE cash_transactions.cashbox_id = cashboxes.id 
					AND source_type = 'OPENING_BALANCE' 
					AND status = 'COMPLETED'
				)
			`, targetOpeningBalance, scope.ID).Error; err != nil {
				return err
			}
		}

		if s.auditSvc != nil {
			s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
				EntityID:   &scope.ID,
				EntityType: "organization_scope",
				Action:     domainaudit.AuditUpdate,
				UserID:     userID,
				NewValues:  *scope,
			})
		}

		return nil
	})

	if err == nil {
		bootstrap.InvalidateOrgCache()
	}

	return err
}
