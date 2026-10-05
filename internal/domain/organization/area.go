package organization

import (
	"time"

	"github.com/google/uuid"
)

// AreaStatus defines the status of an area.
type AreaStatus string

const (
	AreaStatusActive   AreaStatus = "ACTIVE"
	AreaStatusInactive AreaStatus = "INACTIVE"
)

// Area represents a geographical or logical region that contains factories (Roadmap Phase 1).
type Area struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string     `gorm:"type:varchar(200);not null;unique" json:"name"`
	Code        string     `gorm:"type:varchar(50);not null;unique" json:"code"`
	Description string     `gorm:"type:text" json:"description"`
	Status      AreaStatus `gorm:"type:varchar(20);not null;default:'ACTIVE'" json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

func (Area) TableName() string {
	return "areas"
}
