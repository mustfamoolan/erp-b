package audit

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	domainaudit "m3aml-erp/internal/domain/audit"
	"m3aml-erp/internal/repositories"
)

// AuditService handles the recording and querying of audit logs.
// §34, §35 — centralized audit system. Immutable, append-only.
type AuditService struct {
	repo repositories.AuditRepository
}

func NewAuditService(repo repositories.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

// ─── RecordAuditInput ─────────────────────────────────────────────────────────

type RecordAuditInput struct {
	UserID     uuid.UUID
	ScopeID    *uuid.UUID
	Action     domainaudit.AuditAction
	EntityType string
	EntityID   *uuid.UUID
	OldValues  any
	NewValues  any
	IPAddress  string
	Device     string
}

// RecordAudit logs an important action.
// OldValues and NewValues are marshaled to JSONB.
// This is a best-effort call — it NEVER blocks the caller's operation on failure.
func (s *AuditService) RecordAudit(ctx context.Context, input RecordAuditInput) error {
	var oldJSON, newJSON []byte

	if input.OldValues != nil {
		b, err := json.Marshal(input.OldValues)
		if err == nil {
			oldJSON = b
		}
	}
	if input.NewValues != nil {
		b, err := json.Marshal(input.NewValues)
		if err == nil {
			newJSON = b
		}
	}

	logEntry := &domainaudit.AuditLog{
		ID:         uuid.New(),
		UserID:     input.UserID,
		ScopeID:    input.ScopeID,
		Action:     input.Action,
		EntityType: input.EntityType,
		EntityID:   input.EntityID,
		OldValues:  oldJSON,
		NewValues:  newJSON,
		IPAddress:  input.IPAddress,
		Device:     input.Device,
	}

	return s.repo.Save(ctx, logEntry)
}

// ─── Query Methods ────────────────────────────────────────────────────────────

func (s *AuditService) GetByEntity(ctx context.Context, entityType string, entityID *uuid.UUID) ([]domainaudit.AuditLog, error) {
	return s.repo.FindByEntity(ctx, entityType, entityID)
}

func (s *AuditService) GetByScope(ctx context.Context, scopeID uuid.UUID) ([]domainaudit.AuditLog, error) {
	return s.repo.FindByScope(ctx, scopeID)
}

func (s *AuditService) GetByUser(ctx context.Context, userID uuid.UUID) ([]domainaudit.AuditLog, error) {
	return s.repo.FindByUser(ctx, userID)
}

func (s *AuditService) GetAll(ctx context.Context) ([]domainaudit.AuditLog, error) {
	return s.repo.FindAll(ctx)
}
