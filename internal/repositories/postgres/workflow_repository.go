package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"m3aml-erp/internal/domain/workflow"
	"m3aml-erp/internal/repositories"
)

// ─── Financial Request Repository ────────────────────────────────────────────

type financialRequestRepository struct {
	db *gorm.DB
}

func NewFinancialRequestRepository(db *gorm.DB) repositories.FinancialRequestRepository {
	return &financialRequestRepository{db: db}
}

func (r *financialRequestRepository) Save(ctx context.Context, req *workflow.FinancialRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *financialRequestRepository) Update(ctx context.Context, req *workflow.FinancialRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}

func (r *financialRequestRepository) FindByID(ctx context.Context, id uuid.UUID) (*workflow.FinancialRequest, error) {
	var req workflow.FinancialRequest
	err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("History").
		First(&req, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &req, err
}

func (r *financialRequestRepository) FindByDocumentNumber(ctx context.Context, number string) (*workflow.FinancialRequest, error) {
	var req workflow.FinancialRequest
	err := r.db.WithContext(ctx).First(&req, "document_number = ?", number).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &req, err
}

func (r *financialRequestRepository) FindByScope(ctx context.Context, scopeID uuid.UUID) ([]workflow.FinancialRequest, error) {
	var reqs []workflow.FinancialRequest
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("scope_id = ?", scopeID).
		Order("created_at desc").
		Find(&reqs).Error
	return reqs, err
}

func (r *financialRequestRepository) FindAll(ctx context.Context) ([]workflow.FinancialRequest, error) {
	var reqs []workflow.FinancialRequest
	err := r.db.WithContext(ctx).Preload("Items").Order("created_at desc").Find(&reqs).Error
	return reqs, err
}

// ─── Request Item Repository ──────────────────────────────────────────────────

type requestItemRepository struct {
	db *gorm.DB
}

func NewRequestItemRepository(db *gorm.DB) repositories.RequestItemRepository {
	return &requestItemRepository{db: db}
}

func (r *requestItemRepository) SaveAll(ctx context.Context, items []workflow.RequestItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&items).Error
}

func (r *requestItemRepository) FindByRequest(ctx context.Context, requestID uuid.UUID) ([]workflow.RequestItem, error) {
	var items []workflow.RequestItem
	err := r.db.WithContext(ctx).Where("request_id = ?", requestID).Find(&items).Error
	return items, err
}

// ─── Request History Repository ───────────────────────────────────────────────

type requestHistoryRepository struct {
	db *gorm.DB
}

func NewRequestHistoryRepository(db *gorm.DB) repositories.RequestHistoryRepository {
	return &requestHistoryRepository{db: db}
}

func (r *requestHistoryRepository) Save(ctx context.Context, h *workflow.RequestHistory) error {
	return r.db.WithContext(ctx).Create(h).Error
}

func (r *requestHistoryRepository) FindByRequest(ctx context.Context, requestID uuid.UUID) ([]workflow.RequestHistory, error) {
	var history []workflow.RequestHistory
	err := r.db.WithContext(ctx).
		Where("request_id = ?", requestID).
		Order("created_at asc").
		Find(&history).Error
	return history, err
}

// ─── Document Number Generator ────────────────────────────────────────────────
// Generates sequential document numbers like FIN-2026-000001 (Roadmap §33)

func GenerateDocumentNumber(ctx context.Context, db *gorm.DB, prefix string) (string, error) {
	year := time.Now().Year()
	var count int64
	err := db.WithContext(ctx).
		Model(&workflow.FinancialRequest{}).
		Where("EXTRACT(YEAR FROM created_at) = ? AND document_number LIKE ?", year, fmt.Sprintf("%s-%d-%%", prefix, year)).
		Count(&count).Error
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d-%06d", prefix, year, count+1), nil
}
