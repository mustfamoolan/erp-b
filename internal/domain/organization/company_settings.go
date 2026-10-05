package organization

import (
	"time"

	"github.com/google/uuid"
)

// CompanySettings holds the global configuration for the company.
type CompanySettings struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CompanyName string     `gorm:"type:varchar(255);not null" json:"company_name"`
	PhoneNumber string     `gorm:"type:varchar(50)" json:"phone_number"`
	LogoUrl     string     `gorm:"type:varchar(500)" json:"logo_url"`
	UpdatedAt   time.Time  `json:"updated_at"`
	UpdatedBy   *uuid.UUID `gorm:"type:uuid" json:"updated_by"`
}

func (CompanySettings) TableName() string {
	return "company_settings"
}
