package postgres

import (
	"context"
	"m3aml-erp/internal/domain/purchasing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PurchaseInvoiceRepository struct {
	db *gorm.DB
}

func NewPurchaseInvoiceRepository(db *gorm.DB) *PurchaseInvoiceRepository {
	return &PurchaseInvoiceRepository{db: db}
}

func (r *PurchaseInvoiceRepository) Save(ctx context.Context, invoice *purchasing.PurchaseInvoice) error {
	db := r.db.WithContext(ctx)
	if tx, ok := ctx.Value("tx").(*gorm.DB); ok {
		db = tx
	}
	return db.Create(invoice).Error
}

func (r *PurchaseInvoiceRepository) Update(ctx context.Context, invoice *purchasing.PurchaseInvoice) error {
	db := r.db.WithContext(ctx)
	if tx, ok := ctx.Value("tx").(*gorm.DB); ok {
		db = tx
	}
	return db.Save(invoice).Error
}

func (r *PurchaseInvoiceRepository) FindByID(ctx context.Context, id uuid.UUID) (*purchasing.PurchaseInvoice, error) {
	var invoice purchasing.PurchaseInvoice
	err := r.db.WithContext(ctx).Preload("Items").First(&invoice, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &invoice, nil
}

func (r *PurchaseInvoiceRepository) FindByScope(ctx context.Context, scopeID uuid.UUID) ([]purchasing.PurchaseInvoice, error) {
	var invoices []purchasing.PurchaseInvoice
	err := r.db.WithContext(ctx).Where("scope_id = ?", scopeID).Order("created_at desc").Find(&invoices).Error
	return invoices, err
}

type PurchaseInvoiceItemRepository struct {
	db *gorm.DB
}

func NewPurchaseInvoiceItemRepository(db *gorm.DB) *PurchaseInvoiceItemRepository {
	return &PurchaseInvoiceItemRepository{db: db}
}

func (r *PurchaseInvoiceItemRepository) SaveAll(ctx context.Context, items []purchasing.PurchaseInvoiceItem) error {
	if len(items) == 0 {
		return nil
	}
	db := r.db.WithContext(ctx)
	if tx, ok := ctx.Value("tx").(*gorm.DB); ok {
		db = tx
	}
	return db.Create(&items).Error
}

func (r *PurchaseInvoiceItemRepository) FindByInvoiceID(ctx context.Context, invoiceID uuid.UUID) ([]purchasing.PurchaseInvoiceItem, error) {
	var items []purchasing.PurchaseInvoiceItem
	err := r.db.WithContext(ctx).Where("invoice_id = ?", invoiceID).Find(&items).Error
	return items, err
}
