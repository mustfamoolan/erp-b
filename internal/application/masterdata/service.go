package masterdata

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
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

type MasterDataService interface {
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

func (s *service) GetRequestTypes(ctx context.Context, activeOnly bool) ([]masterdata.RequestType, error) {
	return s.repo.GetRequestTypes(ctx, activeOnly)
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
	return s.repo.GetExpenseCategories(ctx, activeOnly)
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
