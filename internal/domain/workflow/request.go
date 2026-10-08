package workflow

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ============================================================
// FINANCIAL REQUEST — Roadmap §31
// The complete approval workflow from Factory → Administration.
// ============================================================

// RequestStatus defines every possible state in the workflow.
// Roadmap §32 — REQUEST WORKFLOW
type RequestStatus string

const (
	// Active states
	RequestDraft              RequestStatus = "DRAFT"
	RequestSubmitted          RequestStatus = "SUBMITTED"
	RequestUnderReview        RequestStatus = "UNDER_REVIEW"
	RequestReviewApproved     RequestStatus = "REVIEW_APPROVED"
	RequestAccountantApproved RequestStatus = "ACCOUNTANT_APPROVED"
	RequestPaymentPending     RequestStatus = "PAYMENT_PENDING"
	RequestDisbursed          RequestStatus = "DISBURSED"
	RequestFactoryReceiptPend RequestStatus = "FACTORY_RECEIPT_PENDING"
	RequestDelivered          RequestStatus = "DELIVERED"
	RequestReceived           RequestStatus = "RECEIVED"
	RequestCompleted          RequestStatus = "COMPLETED"

	// Revision & Terminal rejection/cancellation states
	RequestReturnedForRevision RequestStatus = "RETURNED_FOR_REVISION"
	RequestReviewRejected     RequestStatus = "REVIEW_REJECTED"
	RequestAccountantRejected RequestStatus = "ACCOUNTANT_REJECTED"
	RequestRejected           RequestStatus = "REJECTED"
	RequestCancelled          RequestStatus = "CANCELLED"
)

// RequestType distinguishes financial vs material requests.
type RequestType string

const (
	RequestTypeFinancial RequestType = "FINANCIAL"
	RequestTypeMaterial  RequestType = "MATERIAL"
	RequestTypeAdvance   RequestType = "ADVANCE"
)

// FinancialRequest is created by the Factory Accountant.
// Roadmap §31 — contains all required fields.
// Every request must clearly identify the originating factory.
type FinancialRequest struct {
	ID                uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DocumentNumber    string          `gorm:"type:varchar(50);not null;uniqueIndex"` // e.g. FIN-2026-000001 — §33
	Barcode           string          `gorm:"type:varchar(100);uniqueIndex"`          // machine-readable — §33
	ScopeID           uuid.UUID       `gorm:"type:uuid;not null;index"`               // must = factory scope
	FactoryID         uuid.UUID       `gorm:"type:uuid;not null;index"`               // explicitly identifies the factory
	RequestedBy       uuid.UUID       `gorm:"type:uuid;not null"`
	Type              RequestType     `gorm:"type:varchar(20);not null;default:'FINANCIAL'"`
	
	// Phase 6 Structured Fields
	RequestTypeID        *uuid.UUID      `gorm:"type:uuid" json:"request_type_id,omitempty"` // Links to master data request_types
	ExpenseCategoryID    *uuid.UUID      `gorm:"type:uuid" json:"expense_category_id,omitempty"` // Links to master data expense_categories
	FactoryExpenseTypeID *uuid.UUID      `gorm:"type:uuid" json:"factory_expense_type_id,omitempty"` // Links to master data factory_expense_types
	SupplierName      *string         `gorm:"type:varchar(255)"`
	ReceiverName      *string         `gorm:"type:varchar(255)"`
	ProjectName       *string         `gorm:"type:varchar(255)"`
	Attachments       *string         `gorm:"type:jsonb"` // Store array of attachment URLs as JSON
	ReceivingMethod   *string          `gorm:"type:varchar(50);default:'CASH'"`
	PurchaseNumber    *string          `gorm:"type:varchar(100)"`
	WorkType          *string          `gorm:"type:varchar(255)" json:"work_type,omitempty"`
	BarcodeSKU        *string          `gorm:"type:varchar(255)"`
	ExchangeRate      *decimal.Decimal `gorm:"type:numeric(18,6)"`
	OriginalAmount    *decimal.Decimal `gorm:"type:numeric(18,4)"`
	SignedVoucherURL  *string          `gorm:"type:text"`
	DisbursedAt       *time.Time       `gorm:"type:timestamptz"`
	DeliveredAt       *time.Time       `gorm:"type:timestamptz"`
	
	// Phase 11 Fields (Advance Voucher Fields)
	AdvanceSequenceNumber *int    `gorm:"type:int" json:"advance_sequence_number,omitempty"`
	ReceivingLocation     *string `gorm:"type:varchar(255)" json:"receiving_location,omitempty"`
	ReceiverPhone         *string `gorm:"type:varchar(50)" json:"receiver_phone,omitempty"`

	RequestDate       time.Time        `gorm:"type:date;not null"`
	RequiredDate      *time.Time       `gorm:"type:date"`
	Purpose           string           `gorm:"type:varchar(500);not null"`
	Description       string           `gorm:"type:text"`
	TotalAmount       decimal.Decimal  `gorm:"type:numeric(18,4);not null;default:0"`
	Currency          string           `gorm:"type:varchar(10);not null;default:'IQD'"`
	Status            RequestStatus   `gorm:"type:varchar(40);not null;default:'DRAFT'"`
	CreatedAt         time.Time
	UpdatedAt         time.Time

	RequestedByName   string           `gorm:"->;column:requested_by_name"`

	Items    []RequestItem    `gorm:"foreignKey:RequestID"`
	History  []RequestHistory `gorm:"foreignKey:RequestID"`
}

func (FinancialRequest) TableName() string { return "financial_requests" }

// RequestItem is a line item in a financial/material request.
// Roadmap §31 — request_items
type RequestItem struct {
	ID                uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RequestID         uuid.UUID       `gorm:"type:uuid;not null;index"`
	VariantID         *uuid.UUID      `gorm:"type:uuid;index"` // optional: links to item master — §25
	ExpenseCategoryID *uuid.UUID      `gorm:"type:uuid;index"` // added for phase 4.5/6 support for multiple categories per request
	Description       string          `gorm:"type:varchar(500);not null"`
	Quantity          decimal.Decimal `gorm:"type:numeric(18,4);not null"`
	UnitID            *uuid.UUID      `gorm:"type:uuid"`
	EstimatedUnitPrice decimal.Decimal `gorm:"type:numeric(18,4);not null;default:0"`
	EstimatedTotal    decimal.Decimal `gorm:"type:numeric(18,4);not null;default:0"`
	Notes             string          `gorm:"type:text"`
	ReceiptNumber     *string         `gorm:"type:varchar(100)" json:"receipt_number,omitempty"`
}

func (RequestItem) TableName() string { return "request_items" }

// RequestHistory is an immutable audit record of every status transition.
// Rule 17: Every important workflow transition must be audited.
// This is the document history — Roadmap §40.
type RequestHistory struct {
	ID              uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RequestID       uuid.UUID     `gorm:"type:uuid;not null;index"`
	FromStatus      RequestStatus `gorm:"type:varchar(40)"`
	ToStatus        RequestStatus `gorm:"type:varchar(40);not null"`
	Action          string        `gorm:"type:varchar(50);not null"` // SUBMIT, APPROVE, REJECT, PAY, RECEIVE...
	PerformedBy     uuid.UUID     `gorm:"type:uuid;not null"`
	PerformedByName string        `gorm:"->;column:performed_by_name"`
	Notes           string        `gorm:"type:text"`
	CreatedAt       time.Time
}

func (RequestHistory) TableName() string { return "request_history" }

// ============================================================
// VALID TRANSITIONS — enforced in the application service.
// Roadmap §32 — Rule 4: Never change architecture silently.
// ============================================================

// ValidTransitions defines the allowed status state machine.
var ValidTransitions = map[RequestStatus][]RequestStatus{
	RequestDraft:               {RequestSubmitted, RequestCancelled},
	RequestSubmitted:           {RequestUnderReview, RequestReviewApproved, RequestReviewRejected, RequestRejected, RequestReturnedForRevision, RequestCancelled},
	RequestUnderReview:         {RequestReviewApproved, RequestReviewRejected, RequestRejected, RequestReturnedForRevision},
	RequestReviewApproved:      {RequestAccountantApproved, RequestAccountantRejected, RequestRejected, RequestReturnedForRevision},
	RequestReturnedForRevision: {RequestSubmitted, RequestCancelled},
	RequestAccountantApproved:  {RequestDisbursed, RequestPaymentPending},
	RequestDisbursed:           {RequestDelivered, RequestFactoryReceiptPend},
	RequestPaymentPending:      {RequestDisbursed, RequestFactoryReceiptPend, RequestDelivered},
	RequestFactoryReceiptPend:  {RequestDelivered, RequestReceived},
	RequestReceived:            {RequestCompleted},
	RequestDelivered:           {RequestCompleted},
	// Terminal states — no further transitions allowed
	RequestCompleted:          {},
	RequestReviewRejected:     {},
	RequestAccountantRejected: {},
	RequestRejected:           {},
	RequestCancelled:          {},
}

// CanTransitionTo checks if the given transition is valid.
func (from RequestStatus) CanTransitionTo(to RequestStatus) bool {
	allowed, exists := ValidTransitions[from]
	if !exists {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}
