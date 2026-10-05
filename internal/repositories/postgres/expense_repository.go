package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"m3aml-erp/internal/domain/cashbox"
	"m3aml-erp/internal/repositories"
)

type expenseRepository struct {
	db *gorm.DB
}

func NewExpenseRepository(db *gorm.DB) repositories.ExpenseRepository {
	return &expenseRepository{db: db}
}

func (r *expenseRepository) Create(ctx context.Context, expense *cashbox.Expense, postJournalFunc func(context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(expense).Error; err != nil {
			return err
		}
		// Context with this tx
		txCtx := context.WithValue(ctx, "tx", tx)
		if postJournalFunc != nil {
			if err := postJournalFunc(txCtx); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *expenseRepository) GetByID(ctx context.Context, id uuid.UUID) (*cashbox.Expense, error) {
	var expense cashbox.Expense
	err := r.db.WithContext(ctx).First(&expense, id).Error
	if err != nil {
		return nil, err
	}
	return &expense, nil
}

func (r *expenseRepository) ListByScope(ctx context.Context, scopeID uuid.UUID) ([]cashbox.Expense, error) {
	var expenses []cashbox.Expense
	err := r.db.WithContext(ctx).Where("scope_id = ?", scopeID).Order("expense_date desc, created_at desc").Find(&expenses).Error
	return expenses, err
}

func (r *expenseRepository) GetNextDocumentNumber(ctx context.Context) (string, error) {
	var nextVal int64
	err := r.db.WithContext(ctx).Raw("SELECT nextval('expense_seq')").Scan(&nextVal).Error
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("EXP-%06d", nextVal), nil
}
