package cashbox

import (
	"time"

	"github.com/google/uuid"
)

// Expense represents a factory cash expense (Roadmap §28).
type Expense struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DocumentNumber    string    `gorm:"type:varchar(50);not null;uniqueIndex" json:"document_number"`
	ScopeID           uuid.UUID `gorm:"type:uuid;not null;index" json:"scope_id"`
	CashboxID         uuid.UUID `gorm:"type:uuid;not null;index" json:"cashbox_id"`
	ExpenseCategoryID uuid.UUID `gorm:"type:uuid;not null" json:"expense_category_id"`
	Amount            float64   `gorm:"type:numeric(18,4);not null" json:"amount"`
	Currency          string    `gorm:"type:varchar(10);not null;default:'IQD'" json:"currency"`
	ExpenseDate       time.Time `gorm:"type:date;not null" json:"expense_date"`
	Purpose           string    `gorm:"type:varchar(500);not null" json:"purpose"`
	PaidTo            string    `gorm:"type:varchar(255);not null" json:"paid_to"`
	Description       string    `gorm:"type:text" json:"description"`
	JournalEntryID    uuid.UUID `gorm:"type:uuid" json:"journal_entry_id,omitempty"`
	CreatedBy         uuid.UUID `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (Expense) TableName() string {
	return "expenses"
}
