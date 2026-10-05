package organization

import (
	"time"

	"github.com/google/uuid"
)

// EmployeeStatus defines the employment status.
type EmployeeStatus string

const (
	EmployeeStatusActive     EmployeeStatus = "ACTIVE"
	EmployeeStatusInactive   EmployeeStatus = "INACTIVE"
	EmployeeStatusTerminated EmployeeStatus = "TERMINATED"
)

// Employee represents a real person employed by the company (Roadmap §5).
// An employee belongs to one organizational scope (Administration or a Factory).
type Employee struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ScopeID     uuid.UUID      `gorm:"type:uuid;not null;index"` // which factory or administration
	EmployeeNo  string         `gorm:"type:varchar(50);not null;uniqueIndex"`
	FullName    string         `gorm:"type:varchar(200);not null"`
	NationalID  *string        `gorm:"type:varchar(50)"`
	Phone       string         `gorm:"type:varchar(30)"`
	Email       string         `gorm:"type:varchar(255)"`
	Photo       *string        `gorm:"type:text"`
	JobTitle    string         `gorm:"type:varchar(150)"`
	HireDate    time.Time      `gorm:"type:date;not null"`
	Status      EmployeeStatus `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	Notes       string         `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Employee) TableName() string {
	return "employees"
}
