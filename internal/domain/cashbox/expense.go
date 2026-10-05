package cashbox

import (
	"time"

	"github.com/google/uuid"
)

// Expense represents a factory cash expense (Roadmap §28).
type Expense struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DocumentNumber    string    `gorm:"type:varchar(50);not null;uniqueIndex"`
	ScopeID           uuid.UUID `gorm:"type:uuid;not null;index"`
	CashboxID         uuid.UUID `gorm:"type:uuid;not null;index"`
	ExpenseCategoryID uuid.UUID `gorm:"type:uuid;not null"`
	Amount            float64   `gorm:"type:numeric(18,4);not null"`
	Currency          string    `gorm:"type:varchar(10);not null;default:'IQD'"`
	ExpenseDate       time.Time `gorm:"type:date;not null"`
	Purpose           string    `gorm:"type:varchar(500);not null"`
	PaidTo            string    `gorm:"type:varchar(255);not null"`
	Description       string    `gorm:"type:text"`
	JournalEntryID    uuid.UUID `gorm:"type:uuid"`
	CreatedBy         uuid.UUID `gorm:"type:uuid;not null"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (Expense) TableName() string {
	return "expenses"
}
