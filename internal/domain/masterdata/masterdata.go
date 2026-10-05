package masterdata

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RequestType represents a configurable financial request type (e.g., Advance, Funding)
type RequestType struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code        string    `gorm:"type:varchar(50);not null;uniqueIndex"`
	Name        string    `gorm:"type:varchar(150);not null"`
	Description string    `gorm:"type:text"`
	IsActive    bool      `gorm:"not null;default:true"`
	SortOrder   int       `gorm:"not null;default:0"`
	CreatedBy   *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (RequestType) TableName() string { return "request_types" }

// ExpenseCategory represents a configurable expense category (e.g., Furniture, Machines)
type ExpenseCategory struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code      string     `gorm:"type:varchar(50);not null;uniqueIndex"`
	Name      string     `gorm:"type:varchar(150);not null"`
	ParentID  *uuid.UUID `gorm:"type:uuid"`
	AccountID uuid.UUID  `gorm:"type:uuid;not null"`
	IsActive  bool       `gorm:"not null;default:true"`
	SortOrder int        `gorm:"not null;default:0"`
	Kind      string     `gorm:"type:varchar(20);not null;default:STANDARD"`
	CreatedBy *uuid.UUID  `gorm:"type:uuid"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ExpenseCategory) TableName() string { return "expense_categories" }

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

	// Account Mappings
	GetAccountMapping(ctx context.Context, key string) (*AccountMapping, error)

	// Transactions
	ExecuteInTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
