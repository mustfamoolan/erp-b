package organization

import (
	"time"

	"github.com/google/uuid"
)

// ScopeType defines the type of an organizational scope.
type ScopeType string

const (
	ScopeTypeAdministration ScopeType = "ADMINISTRATION"
	ScopeTypeFactory        ScopeType = "FACTORY"
)

// ScopeStatus defines the status of a scope.
type ScopeStatus string

const (
	ScopeStatusActive   ScopeStatus = "ACTIVE"
	ScopeStatusInactive ScopeStatus = "INACTIVE"
)

// OrganizationScope represents an organizational unit (Administration or Factory).
// Every operational record MUST be traceable to a scope (Roadmap §5, §6).
type OrganizationScope struct {
	ID          uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Type        ScopeType   `gorm:"type:varchar(20);not null" json:"type"`
	Name        string      `gorm:"type:varchar(150);not null;uniqueIndex" json:"name"`
	Code        string      `gorm:"type:varchar(50);not null;uniqueIndex" json:"code"`
	ParentID    *uuid.UUID  `gorm:"type:uuid;index" json:"parent_id"`
	AreaID      *uuid.UUID  `gorm:"type:uuid;index" json:"area_id"`
	Status      ScopeStatus `gorm:"type:varchar(20);not null;default:'ACTIVE'" json:"status"`
	Location    string      `gorm:"type:varchar(255);default:''" json:"location"`
	ManagerName string      `gorm:"type:varchar(150);default:''" json:"manager_name"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

func (OrganizationScope) TableName() string {
	return "organization_scopes"
}
