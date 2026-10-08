package masterdata

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RequestType represents a configurable financial request type (e.g., Advance, Funding)
type RequestType struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code        string     `gorm:"type:varchar(50);not null;uniqueIndex" json:"code"`
	Name        string     `gorm:"type:varchar(150);not null" json:"name"`
	Description string     `gorm:"type:text" json:"description"`
	IsActive    bool       `gorm:"not null;default:true" json:"is_active"`
	SortOrder   int        `gorm:"not null;default:0" json:"sort_order"`
	CreatedBy   *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (RequestType) TableName() string { return "request_types" }

// ExpenseCategory represents a configurable expense category (e.g., Furniture, Machines)
type ExpenseCategory struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code      string     `gorm:"type:varchar(50);not null;uniqueIndex" json:"code"`
	Name      string     `gorm:"type:varchar(150);not null" json:"name"`
	ParentID  *uuid.UUID `gorm:"type:uuid" json:"parent_id,omitempty"`
	AccountID uuid.UUID  `gorm:"type:uuid;not null" json:"account_id"`
	IsActive  bool       `gorm:"not null;default:true" json:"is_active"`
	SortOrder int        `gorm:"not null;default:0" json:"sort_order"`
	Kind      string     `gorm:"type:varchar(20);not null;default:STANDARD" json:"kind"`
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (ExpenseCategory) TableName() string { return "expense_categories" }

// FactoryExpenseType represents a configurable factory expense category/work type (e.g. Setup, Construction, Purchase, Operational)
type FactoryExpenseType struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code      string     `gorm:"type:varchar(50);not null;uniqueIndex" json:"code"`
	Name      string     `gorm:"type:varchar(150);not null" json:"name"`
	IsActive  bool       `gorm:"not null;default:true" json:"is_active"`
	SortOrder int        `gorm:"not null;default:0" json:"sort_order"`
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (FactoryExpenseType) TableName() string { return "factory_expense_types" }

// Expense category kinds. SALARY follows a dedicated request path (not yet defined),
// so generic financial requests against it are refused by the backend.
const (
	ExpenseCategoryKindStandard = "STANDARD"
	ExpenseCategoryKindSalary   = "SALARY"
)

// AccountMapping holds mappings for dynamically resolving system accounts
type AccountMapping struct {
	Key         string    `gorm:"type:varchar(100);primaryKey"`
	AccountID   uuid.UUID `gorm:"type:uuid;not null"`
	Description string    `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (AccountMapping) TableName() string { return "account_mappings" }

// MasterDataRepository defines database operations for master data
type MasterDataRepository interface {
	// Request Types
	GetRequestTypes(ctx context.Context, activeOnly bool) ([]RequestType, error)
	GetRequestTypeByID(ctx context.Context, id uuid.UUID) (*RequestType, error)
	GetRequestTypeByCode(ctx context.Context, code string) (*RequestType, error)
	CreateRequestType(ctx context.Context, rt *RequestType) error
	UpdateRequestType(ctx context.Context, rt *RequestType) error

	// Expense Categories
	GetExpenseCategories(ctx context.Context, activeOnly bool) ([]ExpenseCategory, error)
	GetExpenseCategoryByID(ctx context.Context, id uuid.UUID) (*ExpenseCategory, error)
	GetExpenseCategoryByCode(ctx context.Context, code string) (*ExpenseCategory, error)
	CreateExpenseCategory(ctx context.Context, ec *ExpenseCategory) error
	UpdateExpenseCategory(ctx context.Context, ec *ExpenseCategory) error

	// Factory Expense Types
	GetFactoryExpenseTypes(ctx context.Context, activeOnly bool) ([]FactoryExpenseType, error)
	GetFactoryExpenseTypeByID(ctx context.Context, id uuid.UUID) (*FactoryExpenseType, error)
	GetFactoryExpenseTypeByCode(ctx context.Context, code string) (*FactoryExpenseType, error)
	CreateFactoryExpenseType(ctx context.Context, fet *FactoryExpenseType) error
	UpdateFactoryExpenseType(ctx context.Context, fet *FactoryExpenseType) error

	// Receiving Methods
	GetReceivingMethods(ctx context.Context, activeOnly bool) ([]ReceivingMethod, error)
	GetReceivingMethodByID(ctx context.Context, id uuid.UUID) (*ReceivingMethod, error)
	GetReceivingMethodByCode(ctx context.Context, code string) (*ReceivingMethod, error)
	CreateReceivingMethod(ctx context.Context, rm *ReceivingMethod) error
	UpdateReceivingMethod(ctx context.Context, rm *ReceivingMethod) error

	// Units of Measure
	GetUnits(ctx context.Context, activeOnly bool) ([]UnitOfMeasure, error)
	GetUnitByID(ctx context.Context, id uuid.UUID) (*UnitOfMeasure, error)
	GetUnitByCode(ctx context.Context, code string) (*UnitOfMeasure, error)
	CreateUnit(ctx context.Context, u *UnitOfMeasure) error
	UpdateUnit(ctx context.Context, u *UnitOfMeasure) error

	// Account Mappings
	GetAccountMapping(ctx context.Context, key string) (*AccountMapping, error)

	// Transactions
	ExecuteInTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

// ReceivingMethod represents a configurable receiving method (e.g., Cash, Transfer, Check, Card)
type ReceivingMethod struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code      string     `gorm:"type:varchar(50);not null;uniqueIndex" json:"code"`
	Name      string     `gorm:"type:varchar(150);not null" json:"name"`
	IsActive  bool       `gorm:"not null;default:true" json:"is_active"`
	SortOrder int        `gorm:"not null;default:0" json:"sort_order"`
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (ReceivingMethod) TableName() string { return "receiving_methods" }

// UnitOfMeasure represents a configurable unit of measure (e.g., Piece, Meter, Box, KG)
type UnitOfMeasure struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code      string    `gorm:"type:varchar(20);not null;uniqueIndex" json:"code"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	IsActive  bool      `gorm:"not null;default:true" json:"is_active"`
	SortOrder int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

func (UnitOfMeasure) TableName() string { return "units_of_measure" }

