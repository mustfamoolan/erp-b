package masterdata

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"m3aml-erp/bootstrap"
	appaudit "m3aml-erp/internal/application/audit"
	"m3aml-erp/internal/domain/accounting"
	domainaudit "m3aml-erp/internal/domain/audit"
	"m3aml-erp/internal/domain/masterdata"
	"m3aml-erp/internal/repositories"
)

var (
	ErrNotFound = errors.New("master data not found")
	ErrCodeExists = errors.New("code already exists")
	ErrInvalidParent = errors.New("invalid parent account mapping")
)

type MasterDataBundle struct {
	RequestTypes        []masterdata.RequestType        `json:"request_types"`
	ExpenseCategories   []masterdata.ExpenseCategory    `json:"expense_categories"`
	FactoryExpenseTypes []masterdata.FactoryExpenseType `json:"factory_expense_types"`
	ReceivingMethods    []masterdata.ReceivingMethod    `json:"receiving_methods"`
	Units               []masterdata.UnitOfMeasure      `json:"units"`
}

type MasterDataService interface {
	// Bundle for instant single-flight hydration
	GetBundle(ctx context.Context) (*MasterDataBundle, error)

	// Request Types
	GetRequestTypes(ctx context.Context, activeOnly bool) ([]masterdata.RequestType, error)
	CreateRequestType(ctx context.Context, req masterdata.RequestType, userID uuid.UUID) (*masterdata.RequestType, error)
	UpdateRequestType(ctx context.Context, id uuid.UUID, updates masterdata.RequestType, userID uuid.UUID) error
	ToggleRequestType(ctx context.Context, id uuid.UUID, active bool, userID uuid.UUID) error

	// Expense Categories
	GetExpenseCategories(ctx context.Context, activeOnly bool) ([]masterdata.ExpenseCategory, error)
	CreateExpenseCategory(ctx context.Context, req masterdata.ExpenseCategory, userID uuid.UUID) (*masterdata.ExpenseCategory, error)
	UpdateExpenseCategory(ctx context.Context, id uuid.UUID, updates masterdata.ExpenseCategory, userID uuid.UUID) error
	ToggleExpenseCategory(ctx context.Context, id uuid.UUID, active bool, userID uuid.UUID) error

	// Factory Expense Types
	GetFactoryExpenseTypes(ctx context.Context, activeOnly bool) ([]masterdata.FactoryExpenseType, error)
	CreateFactoryExpenseType(ctx context.Context, req masterdata.FactoryExpenseType, userID uuid.UUID) (*masterdata.FactoryExpenseType, error)
	UpdateFactoryExpenseType(ctx context.Context, id uuid.UUID, updates masterdata.FactoryExpenseType, userID uuid.UUID) error
	ToggleFactoryExpenseType(ctx context.Context, id uuid.UUID, active bool, userID uuid.UUID) error

	// Receiving Methods
	GetReceivingMethods(ctx context.Context, activeOnly bool) ([]masterdata.ReceivingMethod, error)
	CreateReceivingMethod(ctx context.Context, req masterdata.ReceivingMethod, userID uuid.UUID) (*masterdata.ReceivingMethod, error)
	UpdateReceivingMethod(ctx context.Context, id uuid.UUID, updates masterdata.ReceivingMethod, userID uuid.UUID) error
	ToggleReceivingMethod(ctx context.Context, id uuid.UUID, active bool, userID uuid.UUID) error

	// Units of Measure
	GetUnits(ctx context.Context, activeOnly bool) ([]masterdata.UnitOfMeasure, error)
	CreateUnit(ctx context.Context, req masterdata.UnitOfMeasure, userID uuid.UUID) (*masterdata.UnitOfMeasure, error)
	UpdateUnit(ctx context.Context, id uuid.UUID, updates masterdata.UnitOfMeasure, userID uuid.UUID) error
	ToggleUnit(ctx context.Context, id uuid.UUID, active bool, userID uuid.UUID) error
}

type service struct {
	repo       masterdata.MasterDataRepository
	accountRepo repositories.AccountRepository
	auditSvc   *appaudit.AuditService
}

func NewMasterDataService(
	repo masterdata.MasterDataRepository,
	accountRepo repositories.AccountRepository,
	auditSvc *appaudit.AuditService,
) MasterDataService {
	return &service{
		repo:       repo,
		accountRepo: accountRepo,
		auditSvc:   auditSvc,
	}
}

func (s *service) GetBundle(ctx context.Context) (*MasterDataBundle, error) {
	return bootstrap.CacheRemember("md:bundle", 24*time.Hour, func() (*MasterDataBundle, error) {
		rt, err := s.repo.GetRequestTypes(ctx, false)
		if err != nil {
			return nil, err
		}
		ec, err := s.repo.GetExpenseCategories(ctx, false)
		if err != nil {
			return nil, err
		}
		fet, err := s.repo.GetFactoryExpenseTypes(ctx, false)
		if err != nil {
			return nil, err
		}
		rm, err := s.repo.GetReceivingMethods(ctx, false)
		if err != nil {
			return nil, err
		}
		u, err := s.repo.GetUnits(ctx, false)
		if err != nil {
			return nil, err
		}
		return &MasterDataBundle{
			RequestTypes:        rt,
			ExpenseCategories:   ec,
			FactoryExpenseTypes: fet,
			ReceivingMethods:    rm,
			Units:               u,
		}, nil
	})
}

func (s *service) GetRequestTypes(ctx context.Context, activeOnly bool) ([]masterdata.RequestType, error) {
	key := fmt.Sprintf("md:request_types:%t", activeOnly)
	return bootstrap.CacheRemember(key, 24*time.Hour, func() ([]masterdata.RequestType, error) {
		return s.repo.GetRequestTypes(ctx, activeOnly)
	})
}

func (s *service) CreateRequestType(ctx context.Context, req masterdata.RequestType, userID uuid.UUID) (*masterdata.RequestType, error) {
	existing, _ := s.repo.GetRequestTypeByCode(ctx, req.Code)
	if existing != nil {
		return nil, ErrCodeExists
	}

	req.ID = uuid.New()
	req.CreatedBy = &userID
	req.IsActive = true

	if err := s.repo.CreateRequestType(ctx, &req); err != nil {
		return nil, err
	}

	bootstrap.InvalidateMasterDataCache()

	s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
		EntityID:   &req.ID,
		EntityType: "master_data_request_type",
		Action:     domainaudit.AuditCreate,
		UserID:     userID,
		NewValues:  req,
	})

	return &req, nil
}

func (s *service) UpdateRequestType(ctx context.Context, id uuid.UUID, updates masterdata.RequestType, userID uuid.UUID) error {
	rt, err := s.repo.GetRequestTypeByID(ctx, id)
	if err != nil {
		return err
	}
	if rt == nil {
		return ErrNotFound
	}

	oldValues := *rt

	rt.Name = updates.Name
	rt.Description = updates.Description
	rt.SortOrder = updates.SortOrder

	if err := s.repo.UpdateRequestType(ctx, rt); err != nil {
		return err
	}

	bootstrap.InvalidateMasterDataCache()

	s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
		EntityID:   &rt.ID,
		EntityType: "master_data_request_type",
		Action:     domainaudit.AuditUpdate,
		UserID:     userID,
		OldValues:  oldValues,
		NewValues:  rt,
	})

	return nil
}

func (s *service) ToggleRequestType(ctx context.Context, id uuid.UUID, active bool, userID uuid.UUID) error {
	rt, err := s.repo.GetRequestTypeByID(ctx, id)
	if err != nil {
		return err
	}
	if rt == nil {
		return ErrNotFound
	}

	oldValues := *rt
	rt.IsActive = active

	if err := s.repo.UpdateRequestType(ctx, rt); err != nil {
		return err
	}

	bootstrap.InvalidateMasterDataCache()

	s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
		EntityID:   &rt.ID,
		EntityType: "master_data_request_type",
		Action:     domainaudit.AuditUpdate,
		UserID:     userID,
		OldValues:  oldValues,
		NewValues:  rt,
	})

	return nil
}

func (s *service) GetExpenseCategories(ctx context.Context, activeOnly bool) ([]masterdata.ExpenseCategory, error) {
	key := fmt.Sprintf("md:expense_categories:%t", activeOnly)
	return bootstrap.CacheRemember(key, 24*time.Hour, func() ([]masterdata.ExpenseCategory, error) {
		return s.repo.GetExpenseCategories(ctx, activeOnly)
	})
}

func (s *service) CreateExpenseCategory(ctx context.Context, req masterdata.ExpenseCategory, userID uuid.UUID) (*masterdata.ExpenseCategory, error) {
	existing, _ := s.repo.GetExpenseCategoryByCode(ctx, req.Code)
	if existing != nil {
		return nil, ErrCodeExists
	}

	err := s.repo.ExecuteInTx(ctx, func(txCtx context.Context) error {
		// Find the parent account mapping
		mapping, err := s.repo.GetAccountMapping(txCtx, "EXPENSE_CATEGORY_PARENT")
		if err != nil {
			return err
		}
		if mapping == nil {
			return ErrInvalidParent
		}

		parentAccount, err := s.accountRepo.FindByID(txCtx, mapping.AccountID)
		if err != nil || parentAccount == nil {
			return ErrInvalidParent
		}

		// Find children to determine next code
		children, err := s.accountRepo.FindChildren(txCtx, parentAccount.ID)
		if err != nil {
			return err
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
		
		// If no children, start with 10. Otherwise maxCode + 10
		nextNum := maxCode + 10
		if nextNum == 10 && len(children) == 0 {
			nextNum = 10
		} else if len(children) > 0 && maxCode == 0 {
			nextNum = 10
		}
		
		newCode := fmt.Sprintf("%s%02d", parentAccount.Code, nextNum)
		// Some existing seed accounts might be 5210, 5220... which means suffix is 10, 20.
		// We want to avoid collisions. A safe approach is to just check if it exists:
		for {
			acc, _ := s.accountRepo.FindByCode(txCtx, newCode)
			if acc == nil {
				break
			}
			nextNum += 10
			newCode = fmt.Sprintf("%s%02d", parentAccount.Code, nextNum)
		}

		newAccount := accounting.Account{
			ID:         uuid.New(),
			Code:       newCode,
			Name:       "مصروفات - " + req.Name,
			Type:       accounting.AccountTypeExpense,
			ParentID:   &parentAccount.ID,
			IsPostable: true,
			IsActive:   true,
			SortOrder:  parentAccount.SortOrder*100 + nextNum,
		}

		if err := s.accountRepo.Save(txCtx, &newAccount); err != nil {
			return err
		}

		req.ID = uuid.New()
		req.AccountID = newAccount.ID
		req.CreatedBy = &userID
		req.IsActive = true

		if err := s.repo.CreateExpenseCategory(txCtx, &req); err != nil {
			return err
		}
		
		return nil
	})

	if err != nil {
		return nil, err
	}

	bootstrap.InvalidateMasterDataCache()

	s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
		EntityID:   &req.ID,
		EntityType: "master_data_expense_category",
		Action:     "CREATE",
		UserID:     userID,
		NewValues:  req,
	})

	return &req, nil
}

func (s *service) UpdateExpenseCategory(ctx context.Context, id uuid.UUID, updates masterdata.ExpenseCategory, userID uuid.UUID) error {
	ec, err := s.repo.GetExpenseCategoryByID(ctx, id)
	if err != nil {
		return err
	}
	if ec == nil {
		return ErrNotFound
	}

	oldValues := *ec

	ec.Name = updates.Name
	ec.ParentID = updates.ParentID
	ec.SortOrder = updates.SortOrder

	if err := s.repo.UpdateExpenseCategory(ctx, ec); err != nil {
		return err
	}

	bootstrap.InvalidateMasterDataCache()

	s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
		EntityID:   &ec.ID,
		EntityType: "master_data_expense_category",
		Action:     domainaudit.AuditUpdate,
		UserID:     userID,
		OldValues:  oldValues,
		NewValues:  ec,
	})

	return nil
}

func (s *service) ToggleExpenseCategory(ctx context.Context, id uuid.UUID, active bool, userID uuid.UUID) error {
	ec, err := s.repo.GetExpenseCategoryByID(ctx, id)
	if err != nil {
		return err
	}
	if ec == nil {
		return ErrNotFound
	}

	oldValues := *ec
	ec.IsActive = active

	if err := s.repo.UpdateExpenseCategory(ctx, ec); err != nil {
		return err
	}

	bootstrap.InvalidateMasterDataCache()

	s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
		EntityID:   &ec.ID,
		EntityType: "master_data_expense_category",
		Action:     domainaudit.AuditUpdate,
		UserID:     userID,
		OldValues:  oldValues,
		NewValues:  ec,
	})

	return nil
}

func (s *service) GetFactoryExpenseTypes(ctx context.Context, activeOnly bool) ([]masterdata.FactoryExpenseType, error) {
	key := fmt.Sprintf("md:factory_expense_types:%t", activeOnly)
	return bootstrap.CacheRemember(key, 24*time.Hour, func() ([]masterdata.FactoryExpenseType, error) {
		return s.repo.GetFactoryExpenseTypes(ctx, activeOnly)
	})
}

func (s *service) CreateFactoryExpenseType(ctx context.Context, req masterdata.FactoryExpenseType, userID uuid.UUID) (*masterdata.FactoryExpenseType, error) {
	existing, _ := s.repo.GetFactoryExpenseTypeByCode(ctx, req.Code)
	if existing != nil {
		return nil, ErrCodeExists
	}

	req.ID = uuid.New()
	req.CreatedBy = &userID
	req.IsActive = true

	if err := s.repo.CreateFactoryExpenseType(ctx, &req); err != nil {
		return nil, err
	}

	bootstrap.InvalidateMasterDataCache()

	s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
		EntityID:   &req.ID,
		EntityType: "master_data_factory_expense_type",
		Action:     domainaudit.AuditCreate,
		UserID:     userID,
		NewValues:  req,
	})

	return &req, nil
}

func (s *service) UpdateFactoryExpenseType(ctx context.Context, id uuid.UUID, updates masterdata.FactoryExpenseType, userID uuid.UUID) error {
	fet, err := s.repo.GetFactoryExpenseTypeByID(ctx, id)
	if err != nil {
		return err
	}
	if fet == nil {
		return ErrNotFound
	}

	oldValues := *fet

	fet.Name = updates.Name
	fet.SortOrder = updates.SortOrder

	if err := s.repo.UpdateFactoryExpenseType(ctx, fet); err != nil {
		return err
	}

	bootstrap.InvalidateMasterDataCache()

	s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
		EntityID:   &fet.ID,
		EntityType: "master_data_factory_expense_type",
		Action:     domainaudit.AuditUpdate,
		UserID:     userID,
		OldValues:  oldValues,
		NewValues:  fet,
	})

	return nil
}

func (s *service) ToggleFactoryExpenseType(ctx context.Context, id uuid.UUID, active bool, userID uuid.UUID) error {
	fet, err := s.repo.GetFactoryExpenseTypeByID(ctx, id)
	if err != nil {
		return err
	}
	if fet == nil {
		return ErrNotFound
	}

	oldValues := *fet
	fet.IsActive = active

	if err := s.repo.UpdateFactoryExpenseType(ctx, fet); err != nil {
		return err
	}

	bootstrap.InvalidateMasterDataCache()

	s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
		EntityID:   &fet.ID,
		EntityType: "master_data_factory_expense_type",
		Action:     domainaudit.AuditUpdate,
		UserID:     userID,
		OldValues:  oldValues,
		NewValues:  fet,
	})

	return nil
}

// ─── Receiving Methods ───────────────────────────────────────────────────────

func (s *service) GetReceivingMethods(ctx context.Context, activeOnly bool) ([]masterdata.ReceivingMethod, error) {
	key := fmt.Sprintf("md:receiving_methods:%t", activeOnly)
	return bootstrap.CacheRemember(key, 24*time.Hour, func() ([]masterdata.ReceivingMethod, error) {
		return s.repo.GetReceivingMethods(ctx, activeOnly)
	})
}

func (s *service) CreateReceivingMethod(ctx context.Context, req masterdata.ReceivingMethod, userID uuid.UUID) (*masterdata.ReceivingMethod, error) {
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("code and name are required")
	}

	existing, err := s.repo.GetReceivingMethodByCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrCodeExists
	}

	req.ID = uuid.New()
	req.IsActive = true
	req.CreatedBy = &userID

	if err := s.repo.CreateReceivingMethod(ctx, &req); err != nil {
		return nil, err
	}

	bootstrap.InvalidateMasterDataCache()

	s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
		EntityID:   &req.ID,
		EntityType: "master_data_receiving_method",
		Action:     domainaudit.AuditCreate,
		UserID:     userID,
		NewValues:  req,
	})

	return &req, nil
}

func (s *service) UpdateReceivingMethod(ctx context.Context, id uuid.UUID, updates masterdata.ReceivingMethod, userID uuid.UUID) error {
	rm, err := s.repo.GetReceivingMethodByID(ctx, id)
	if err != nil {
		return err
	}
	if rm == nil {
		return ErrNotFound
	}

	oldValues := *rm
	rm.Name = updates.Name
	rm.SortOrder = updates.SortOrder

	if err := s.repo.UpdateReceivingMethod(ctx, rm); err != nil {
		return err
	}

	bootstrap.InvalidateMasterDataCache()

	s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
		EntityID:   &rm.ID,
		EntityType: "master_data_receiving_method",
		Action:     domainaudit.AuditUpdate,
		UserID:     userID,
		OldValues:  oldValues,
		NewValues:  rm,
	})

	return nil
}

func (s *service) ToggleReceivingMethod(ctx context.Context, id uuid.UUID, active bool, userID uuid.UUID) error {
	rm, err := s.repo.GetReceivingMethodByID(ctx, id)
	if err != nil {
		return err
	}
	if rm == nil {
		return ErrNotFound
	}

	oldValues := *rm
	rm.IsActive = active

	if err := s.repo.UpdateReceivingMethod(ctx, rm); err != nil {
		return err
	}

	bootstrap.InvalidateMasterDataCache()

	s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
		EntityID:   &rm.ID,
		EntityType: "master_data_receiving_method",
		Action:     domainaudit.AuditUpdate,
		UserID:     userID,
		OldValues:  oldValues,
		NewValues:  rm,
	})

	return nil
}

// ─── Units of Measure ────────────────────────────────────────────────────────

func (s *service) GetUnits(ctx context.Context, activeOnly bool) ([]masterdata.UnitOfMeasure, error) {
	key := fmt.Sprintf("md:units:%t", activeOnly)
	return bootstrap.CacheRemember(key, 24*time.Hour, func() ([]masterdata.UnitOfMeasure, error) {
		return s.repo.GetUnits(ctx, activeOnly)
	})
}

func (s *service) CreateUnit(ctx context.Context, req masterdata.UnitOfMeasure, userID uuid.UUID) (*masterdata.UnitOfMeasure, error) {
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("code and name are required")
	}

	existing, err := s.repo.GetUnitByCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrCodeExists
	}

	req.ID = uuid.New()
	req.IsActive = true

	if err := s.repo.CreateUnit(ctx, &req); err != nil {
		return nil, err
	}

	bootstrap.InvalidateMasterDataCache()

	s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
		EntityID:   &req.ID,
		EntityType: "master_data_unit_of_measure",
		Action:     domainaudit.AuditCreate,
		UserID:     userID,
		NewValues:  req,
	})

	return &req, nil
}

func (s *service) UpdateUnit(ctx context.Context, id uuid.UUID, updates masterdata.UnitOfMeasure, userID uuid.UUID) error {
	u, err := s.repo.GetUnitByID(ctx, id)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrNotFound
	}

	oldValues := *u
	u.Name = updates.Name
	u.SortOrder = updates.SortOrder

	if err := s.repo.UpdateUnit(ctx, u); err != nil {
		return err
	}

	bootstrap.InvalidateMasterDataCache()

	s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
		EntityID:   &u.ID,
		EntityType: "master_data_unit_of_measure",
		Action:     domainaudit.AuditUpdate,
		UserID:     userID,
		OldValues:  oldValues,
		NewValues:  u,
	})

	return nil
}

func (s *service) ToggleUnit(ctx context.Context, id uuid.UUID, active bool, userID uuid.UUID) error {
	u, err := s.repo.GetUnitByID(ctx, id)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrNotFound
	}

	oldValues := *u
	u.IsActive = active

	if err := s.repo.UpdateUnit(ctx, u); err != nil {
		return err
	}

	bootstrap.InvalidateMasterDataCache()

	s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
		EntityID:   &u.ID,
		EntityType: "master_data_unit_of_measure",
		Action:     domainaudit.AuditUpdate,
		UserID:     userID,
		OldValues:  oldValues,
		NewValues:  u,
	})

	return nil
}

