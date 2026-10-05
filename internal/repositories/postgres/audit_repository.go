package postgres

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"m3aml-erp/internal/domain/audit"
	"m3aml-erp/internal/repositories"
)

type auditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) repositories.AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) Save(ctx context.Context, logEntry *audit.AuditLog) error {
	// Audits are immutable, so only Create is supported.
	return r.db.WithContext(ctx).Create(logEntry).Error
}

func (r *auditRepository) FindByEntity(ctx context.Context, entityType string, entityID *uuid.UUID) ([]audit.AuditLog, error) {
	var logs []audit.AuditLog
	err := r.db.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("created_at desc").
		Find(&logs).Error
	return logs, err
}

func (r *auditRepository) FindByScope(ctx context.Context, scopeID uuid.UUID) ([]audit.AuditLog, error) {
	var logs []audit.AuditLog
	err := r.db.WithContext(ctx).
		Where("scope_id = ?", scopeID).
		Order("created_at desc").
		Find(&logs).Error
	return logs, err
}

func (r *auditRepository) FindByUser(ctx context.Context, userID uuid.UUID) ([]audit.AuditLog, error) {
	var logs []audit.AuditLog
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at desc").
		Find(&logs).Error
	return logs, err
}

func (r *auditRepository) FindAll(ctx context.Context) ([]audit.AuditLog, error) {
	var logs []audit.AuditLog
	err := r.db.WithContext(ctx).Order("created_at desc").Find(&logs).Error
	return logs, err
}
