package purchasing

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type InvoiceStatus string

const (
	InvoiceDraft     InvoiceStatus = "DRAFT"
	InvoiceApproved  InvoiceStatus = "APPROVED"
	InvoiceCancelled InvoiceStatus = "CANCELLED"
)

type PaymentStatus string

const (
	PaymentUnpaid  PaymentStatus = "UNPAID"
	PaymentPartial PaymentStatus = "PARTIAL"
	PaymentPaid    PaymentStatus = "PAID"
)

type PurchaseInvoice struct {
	ID             uuid.UUID
	DocumentNumber string
	InvoiceNumber  string
	InvoiceDate    time.Time
	ScopeID        uuid.UUID
	SupplierName   string
	TotalAmount    decimal.Decimal
	Discount       decimal.Decimal
	Tax            decimal.Decimal
	NetAmount      decimal.Decimal
	Currency       string
	PaymentStatus  PaymentStatus
	Status         InvoiceStatus
	Notes          string
	Attachments    *string // JSON
	CreatedBy      uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time

	Items []PurchaseInvoiceItem `gorm:"foreignKey:InvoiceID"`
}

type PurchaseInvoiceItem struct {
	ID          uuid.UUID
	InvoiceID   uuid.UUID
	Description string
	Quantity    decimal.Decimal
	UnitID      *uuid.UUID
	UnitPrice   decimal.Decimal
	Total       decimal.Decimal
	Notes       string
}
