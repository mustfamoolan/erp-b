package audit

import (
	"time"

	"github.com/google/uuid"
)

// AuditAction defines all trackable actions in the system.
// Roadmap §34 — centralized audit system.
type AuditAction string

const (
	AuditCreate      AuditAction = "CREATE"
	AuditUpdate      AuditAction = "UPDATE"
	AuditSubmit      AuditAction = "SUBMIT"
	AuditApprove     AuditAction = "APPROVE"
	AuditReject      AuditAction = "REJECT"
	AuditPay         AuditAction = "PAY"
	AuditReceive     AuditAction = "RECEIVE"
	AuditPost        AuditAction = "POST"
	AuditVoid        AuditAction = "VOID"
	AuditReverse     AuditAction = "REVERSE"
	AuditTransfer    AuditAction = "TRANSFER"
	AuditAdjust      AuditAction = "ADJUST"
	AuditLogin       AuditAction = "LOGIN"
	AuditLogout      AuditAction = "LOGOUT"
	AuditClosePeriod AuditAction = "CLOSE_PERIOD"
)

// AuditLog is an immutable record of every important system action.
// Roadmap §34. Append-only — never updated or deleted (enforced in DB).
// Rule 17: Every important workflow transition must be audited.
// Roadmap §35: IMMUTABILITY — the original record remains visible.
type AuditLog struct {
	ID         uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID     uuid.UUID   `gorm:"type:uuid;not null;index"`
	ScopeID    *uuid.UUID  `gorm:"type:uuid;index"`
	Action     AuditAction `gorm:"type:varchar(50);not null"`
	EntityType string      `gorm:"type:varchar(100);not null"` // "financial_request", "journal_entry"...
	EntityID   *uuid.UUID  `gorm:"type:uuid;index"`
	OldValues  []byte      `gorm:"type:jsonb"`
	NewValues  []byte      `gorm:"type:jsonb"`
	IPAddress  string      `gorm:"type:varchar(50)"`
	Device     string      `gorm:"type:varchar(200)"`
	CreatedAt  time.Time
}

func (AuditLog) TableName() string { return "audit_logs" }
