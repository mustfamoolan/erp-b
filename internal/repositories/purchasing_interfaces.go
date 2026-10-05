package repositories

import (
	"context"
	"m3aml-erp/internal/domain/purchasing"

	"github.com/google/uuid"
)

type PurchaseInvoiceRepository interface {
	Save(ctx context.Context, invoice *purchasing.PurchaseInvoice) error
	Update(ctx context.Context, invoice *purchasing.PurchaseInvoice) error
	FindByID(ctx context.Context, id uuid.UUID) (*purchasing.PurchaseInvoice, error)
	FindByScope(ctx context.Context, scopeID uuid.UUID) ([]purchasing.PurchaseInvoice, error)
}

type PurchaseInvoiceItemRepository interface {
	SaveAll(ctx context.Context, items []purchasing.PurchaseInvoiceItem) error
	FindByInvoiceID(ctx context.Context, invoiceID uuid.UUID) ([]purchasing.PurchaseInvoiceItem, error)
}
