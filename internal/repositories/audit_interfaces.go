package repositories

import (
	"context"

	"github.com/google/uuid"
	"m3aml-erp/internal/domain/audit"
)

// AuditRepository handles centralized auditing persistence.
// Roadmap §34, §35
type AuditRepository interface {
	Save(ctx context.Context, logEntry *audit.AuditLog) error
	FindByEntity(ctx context.Context, entityType string, entityID *uuid.UUID) ([]audit.AuditLog, error)
	FindByScope(ctx context.Context, scopeID uuid.UUID) ([]audit.AuditLog, error)
	FindByUser(ctx context.Context, userID uuid.UUID) ([]audit.AuditLog, error)
	FindAll(ctx context.Context) ([]audit.AuditLog, error)
}
