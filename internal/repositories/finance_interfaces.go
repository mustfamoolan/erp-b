package repositories

import (
	"context"
	"m3aml-erp/internal/domain/cashbox"

	"github.com/google/uuid"
)

// CashboxRepository defines persistence operations for Cashbox.
// Rule 7: Scope isolation — FindByScope ensures factory users only see their cashboxes.
type CashboxRepository interface {
	FindAll(ctx context.Context) ([]cashbox.Cashbox, error)
	FindByScope(ctx context.Context, scopeID uuid.UUID) ([]cashbox.Cashbox, error)
	FindByID(ctx context.Context, id uuid.UUID) (*cashbox.Cashbox, error)
	Save(ctx context.Context, cb *cashbox.Cashbox) error
	Update(ctx context.Context, cb *cashbox.Cashbox) error
}

// CashTransactionRepository defines persistence for Cash Transactions.
// Rule 11: Every financial movement must have a traceable transaction.
type CashTransactionRepository interface {
	Save(ctx context.Context, tx *cashbox.CashTransaction) error
	FindByCashbox(ctx context.Context, cashboxID uuid.UUID) ([]cashbox.CashTransaction, error)
	// ExecuteTransfer runs a cashbox transfer + journal entry atomically (Rule 18).
	ExecuteTransfer(ctx context.Context, fromTx, toTx *cashbox.CashTransaction, postJournalFunc func(context.Context) error) error
}

type ExpenseRepository interface {
	Create(ctx context.Context, expense *cashbox.Expense, postJournalFunc func(context.Context) error) error
	GetByID(ctx context.Context, id uuid.UUID) (*cashbox.Expense, error)
	ListByScope(ctx context.Context, scopeID uuid.UUID) ([]cashbox.Expense, error)
	GetNextDocumentNumber(ctx context.Context) (string, error)
}
