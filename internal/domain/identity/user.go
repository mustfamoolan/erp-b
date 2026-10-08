package identity

import (
	"time"

	"github.com/google/uuid"
)

// UserStatus represents the status of a user account.
type UserStatus string

const (
	UserStatusActive    UserStatus = "ACTIVE"
	UserStatusInactive  UserStatus = "INACTIVE"
	UserStatusSuspended UserStatus = "SUSPENDED"
)

// User is an authenticated identity in the system (Roadmap §8).
// A user may have role(s), scope access, permissions, and an employee relationship.
type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Username     string     `gorm:"type:varchar(100);not null;uniqueIndex"`
	Email        string     `gorm:"type:varchar(255);not null;uniqueIndex"`
	PasswordHash string     `gorm:"type:varchar(255);not null"`
	FullName     string     `gorm:"type:varchar(200);not null"`
	Status       UserStatus `gorm:"type:varchar(20);not null;default:'ACTIVE'"`
	EmployeeID   *uuid.UUID `gorm:"type:uuid;index"` // optional link to Employee
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Relations (loaded on demand)
	Roles       []UserRole       `gorm:"foreignKey:UserID"`
	ScopeAccess []UserScopeAccess `gorm:"foreignKey:UserID"`
}

func (User) TableName() string {
	return "users"
}

// RoleName defines the system roles (Roadmap §9).
type RoleName string

const (
	RoleSuperAdmin        RoleName = "SUPER_ADMIN"
	RoleAdministrator     RoleName = "ADMINISTRATOR"
	RoleAuditor           RoleName = "AUDITOR"
	RoleAccountant        RoleName = "ACCOUNTANT"
	RoleCentralCashier    RoleName = "CENTRAL_CASHIER"
	RoleFactoryAccountant RoleName = "FACTORY_ACCOUNTANT"
	RoleFactoryEmployee   RoleName = "FACTORY_EMPLOYEE"
)

// Role is a named collection of permissions (Roadmap §9).
type Role struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        RoleName  `gorm:"type:varchar(50);not null;uniqueIndex" json:"name"`
	DisplayName string    `gorm:"type:varchar(100);not null" json:"display_name"`
	Description string    `gorm:"type:text" json:"description"`
	IsSystem    bool      `gorm:"not null;default:false" json:"is_system"` // system roles cannot be deleted
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Permissions []RolePermission `gorm:"foreignKey:RoleID" json:"permissions,omitempty"`
}

func (Role) TableName() string {
	return "roles"
}

// UserRole assigns a Role to a User, optionally scoped (Roadmap §9).
// Security is ALWAYS enforced by the backend — not by the WPF UI (Roadmap §3).
type UserRole struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	RoleID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	ScopeID   *uuid.UUID `gorm:"type:uuid;index"` // nil = company-wide, non-nil = specific scope
	CreatedAt time.Time
}

func (UserRole) TableName() string {
	return "user_roles"
}

// UserScopeAccess explicitly grants a user access to an organizational scope (Roadmap §8).
// Factory A user MUST NOT automatically access Factory B (Roadmap §2.2, Rule 7).
type UserScopeAccess struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	ScopeID   uuid.UUID `gorm:"type:uuid;not null;index"`
	GrantedAt time.Time
	GrantedBy uuid.UUID `gorm:"type:uuid;not null"`
}

func (UserScopeAccess) TableName() string {
	return "user_scope_access"
}
