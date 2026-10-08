package masterdata

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"m3aml-erp/internal/domain/masterdata"
)

type repository struct {
	db *gorm.DB
}

// NewMasterDataRepository creates a new postgres repository for master data
func NewMasterDataRepository(db *gorm.DB) masterdata.MasterDataRepository {
	return &repository{db: db}
}

// getTx gets the transaction from context or falls back to db
func (r *repository) getTx(ctx context.Context) *gorm.DB {
	tx, ok := ctx.Value("tx").(*gorm.DB)
	if ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *repository) GetRequestTypes(ctx context.Context, activeOnly bool) ([]masterdata.RequestType, error) {
	var types []masterdata.RequestType
	query := r.getTx(ctx).Order("sort_order asc")
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	err := query.Find(&types).Error
	return types, err
}

func (r *repository) GetRequestTypeByID(ctx context.Context, id uuid.UUID) (*masterdata.RequestType, error) {
	var rt masterdata.RequestType
	err := r.getTx(ctx).First(&rt, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rt, err
}

func (r *repository) GetRequestTypeByCode(ctx context.Context, code string) (*masterdata.RequestType, error) {
	var rt masterdata.RequestType
	err := r.getTx(ctx).First(&rt, "code = ?", code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rt, err
}

func (r *repository) CreateRequestType(ctx context.Context, rt *masterdata.RequestType) error {
	return r.getTx(ctx).Create(rt).Error
}

func (r *repository) UpdateRequestType(ctx context.Context, rt *masterdata.RequestType) error {
	return r.getTx(ctx).Save(rt).Error
}

func (r *repository) GetExpenseCategories(ctx context.Context, activeOnly bool) ([]masterdata.ExpenseCategory, error) {
	var cats []masterdata.ExpenseCategory
	query := r.getTx(ctx).Order("sort_order asc")
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	err := query.Find(&cats).Error
	return cats, err
}

func (r *repository) GetExpenseCategoryByID(ctx context.Context, id uuid.UUID) (*masterdata.ExpenseCategory, error) {
	var ec masterdata.ExpenseCategory
	err := r.getTx(ctx).First(&ec, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &ec, err
}

func (r *repository) GetExpenseCategoryByCode(ctx context.Context, code string) (*masterdata.ExpenseCategory, error) {
	var ec masterdata.ExpenseCategory
	err := r.getTx(ctx).First(&ec, "code = ?", code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &ec, err
}

func (r *repository) CreateExpenseCategory(ctx context.Context, ec *masterdata.ExpenseCategory) error {
	return r.getTx(ctx).Create(ec).Error
}

func (r *repository) UpdateExpenseCategory(ctx context.Context, ec *masterdata.ExpenseCategory) error {
	return r.getTx(ctx).Save(ec).Error
}

func (r *repository) GetFactoryExpenseTypes(ctx context.Context, activeOnly bool) ([]masterdata.FactoryExpenseType, error) {
	var types []masterdata.FactoryExpenseType
	query := r.getTx(ctx).Order("sort_order asc")
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	err := query.Find(&types).Error
	return types, err
}

func (r *repository) GetFactoryExpenseTypeByID(ctx context.Context, id uuid.UUID) (*masterdata.FactoryExpenseType, error) {
	var fet masterdata.FactoryExpenseType
	err := r.getTx(ctx).First(&fet, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &fet, err
}

func (r *repository) GetFactoryExpenseTypeByCode(ctx context.Context, code string) (*masterdata.FactoryExpenseType, error) {
	var fet masterdata.FactoryExpenseType
	err := r.getTx(ctx).First(&fet, "code = ?", code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &fet, err
}

func (r *repository) CreateFactoryExpenseType(ctx context.Context, fet *masterdata.FactoryExpenseType) error {
	return r.getTx(ctx).Create(fet).Error
}

func (r *repository) UpdateFactoryExpenseType(ctx context.Context, fet *masterdata.FactoryExpenseType) error {
	return r.getTx(ctx).Save(fet).Error
}

func (r *repository) GetReceivingMethods(ctx context.Context, activeOnly bool) ([]masterdata.ReceivingMethod, error) {
	var methods []masterdata.ReceivingMethod
	query := r.getTx(ctx).Order("sort_order asc")
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	err := query.Find(&methods).Error
	return methods, err
}

func (r *repository) GetReceivingMethodByID(ctx context.Context, id uuid.UUID) (*masterdata.ReceivingMethod, error) {
	var rm masterdata.ReceivingMethod
	err := r.getTx(ctx).First(&rm, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rm, err
}

func (r *repository) GetReceivingMethodByCode(ctx context.Context, code string) (*masterdata.ReceivingMethod, error) {
	var rm masterdata.ReceivingMethod
	err := r.getTx(ctx).First(&rm, "code = ?", code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rm, err
}

func (r *repository) CreateReceivingMethod(ctx context.Context, rm *masterdata.ReceivingMethod) error {
	return r.getTx(ctx).Create(rm).Error
}

func (r *repository) UpdateReceivingMethod(ctx context.Context, rm *masterdata.ReceivingMethod) error {
	return r.getTx(ctx).Save(rm).Error
}

func (r *repository) GetUnits(ctx context.Context, activeOnly bool) ([]masterdata.UnitOfMeasure, error) {
	var units []masterdata.UnitOfMeasure
	query := r.getTx(ctx).Order("sort_order asc, name asc")
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	err := query.Find(&units).Error
	return units, err
}

func (r *repository) GetUnitByID(ctx context.Context, id uuid.UUID) (*masterdata.UnitOfMeasure, error) {
	var u masterdata.UnitOfMeasure
	err := r.getTx(ctx).First(&u, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *repository) GetUnitByCode(ctx context.Context, code string) (*masterdata.UnitOfMeasure, error) {
	var u masterdata.UnitOfMeasure
	err := r.getTx(ctx).First(&u, "code = ?", code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *repository) CreateUnit(ctx context.Context, u *masterdata.UnitOfMeasure) error {
	return r.getTx(ctx).Create(u).Error
}

func (r *repository) UpdateUnit(ctx context.Context, u *masterdata.UnitOfMeasure) error {
	return r.getTx(ctx).Save(u).Error
}

func (r *repository) GetAccountMapping(ctx context.Context, key string) (*masterdata.AccountMapping, error) {
	var m masterdata.AccountMapping
	err := r.getTx(ctx).First(&m, "key = ?", key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &m, err
}

func (r *repository) ExecuteInTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	// If already in a transaction, just reuse it
	if _, ok := ctx.Value("tx").(*gorm.DB); ok {
		return fn(ctx)
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, "tx", tx)
		return fn(txCtx)
	})
}
