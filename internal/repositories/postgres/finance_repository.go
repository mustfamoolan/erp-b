package postgres

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"m3aml-erp/internal/domain/cashbox"
	"m3aml-erp/internal/repositories"
)

// ─── Cashbox Repository ───────────────────────────────────────────────────────

type cashboxRepository struct {
	db *gorm.DB
}

func NewCashboxRepository(db *gorm.DB) repositories.CashboxRepository {
	return &cashboxRepository{db: db}
}

func (r *cashboxRepository) FindAll(ctx context.Context) ([]cashbox.Cashbox, error) {
	var boxes []cashbox.Cashbox
	err := r.db.WithContext(ctx).
		Table("cashboxes").
		Select("cashboxes.*, COALESCE(cb.current_balance, 0) as current_balance").
		Joins("LEFT JOIN cashbox_balances cb ON cb.cashbox_id = cashboxes.id").
		Order("cashboxes.name asc").
		Scan(&boxes).Error
	return boxes, err
}

func (r *cashboxRepository) FindByScope(ctx context.Context, scopeID uuid.UUID) ([]cashbox.Cashbox, error) {
	var boxes []cashbox.Cashbox
	err := r.db.WithContext(ctx).
		Table("cashboxes").
		Select("cashboxes.*, COALESCE(cb.current_balance, 0) as current_balance").
		Joins("LEFT JOIN cashbox_balances cb ON cb.cashbox_id = cashboxes.id").
		Where("cashboxes.scope_id = ?", scopeID).
		Order("cashboxes.name asc").
		Scan(&boxes).Error
	return boxes, err
}

func (r *cashboxRepository) FindByID(ctx context.Context, id uuid.UUID) (*cashbox.Cashbox, error) {
	var cb cashbox.Cashbox
	if err := r.db.WithContext(ctx).
		Table("cashboxes").
		Select("cashboxes.*, COALESCE(cb.current_balance, 0) as current_balance").
		Joins("LEFT JOIN cashbox_balances cb ON cb.cashbox_id = cashboxes.id").
		Where("cashboxes.id = ?", id).
		Scan(&cb).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	if cb.ID == uuid.Nil {
		return nil, nil // Not found
	}
	return &cb, nil
}

func (r *cashboxRepository) Save(ctx context.Context, cb *cashbox.Cashbox) error {
	return r.db.WithContext(ctx).Create(cb).Error
}

func (r *cashboxRepository) Update(ctx context.Context, cb *cashbox.Cashbox) error {
	return r.db.WithContext(ctx).Save(cb).Error
}

// ─── CashTransaction Repository ──────────────────────────────────────────────

type cashTransactionRepository struct {
	db *gorm.DB
}

func NewCashTransactionRepository(db *gorm.DB) repositories.CashTransactionRepository {
	return &cashTransactionRepository{db: db}
}

func (r *cashTransactionRepository) Save(ctx context.Context, tx *cashbox.CashTransaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *cashTransactionRepository) FindByCashbox(ctx context.Context, cashboxID uuid.UUID) ([]cashbox.CashTransaction, error) {
	var txs []cashbox.CashTransaction
	err := r.db.WithContext(ctx).
		Table("cash_transactions").
		Select("cash_transactions.*, COALESCE(u.full_name, u.username, 'النظام') AS performed_by_name").
		Joins("LEFT JOIN users u ON u.id = cash_transactions.performed_by").
		Where("cash_transactions.cashbox_id = ?", cashboxID).
		Order("cash_transactions.transaction_date desc, cash_transactions.created_at desc").
		Find(&txs).Error
	return txs, err
}

// ExecuteTransfer creates two CashTransactions and a Journal Entry atomically (Rule 18).
func (r *cashTransactionRepository) ExecuteTransfer(
	ctx context.Context,
	fromTx, toTx *cashbox.CashTransaction,
	postJournalFunc func(context.Context) error,
) error {
	return r.db.WithContext(ctx).Transaction(func(gormTx *gorm.DB) error {
		if fromTx != nil {
			if err := gormTx.Create(fromTx).Error; err != nil {
				return err
			}
		}
		if toTx != nil {
			if err := gormTx.Create(toTx).Error; err != nil {
				return err
			}
		}
		// Pass the transaction context to the journal function
		txCtx := context.WithValue(ctx, "tx", gormTx)
		return postJournalFunc(txCtx)
	})
}
