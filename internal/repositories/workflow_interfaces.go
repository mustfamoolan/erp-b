package repositories

import (
	"context"

	"github.com/google/uuid"
	"m3aml-erp/internal/domain/workflow"
)

// FinancialRequestRepository defines persistence for FinancialRequests (Roadmap §31, §32)
type FinancialRequestRepository interface {
	Save(ctx context.Context, req *workflow.FinancialRequest) error
	Update(ctx context.Context, req *workflow.FinancialRequest) error
	FindByID(ctx context.Context, id uuid.UUID) (*workflow.FinancialRequest, error)
	FindByDocumentNumber(ctx context.Context, number string) (*workflow.FinancialRequest, error)
	FindByScope(ctx context.Context, scopeID uuid.UUID) ([]workflow.FinancialRequest, error)
	FindAll(ctx context.Context) ([]workflow.FinancialRequest, error)
}

// RequestItemRepository defines persistence for RequestItems
type RequestItemRepository interface {
	SaveAll(ctx context.Context, items []workflow.RequestItem) error
	FindByRequest(ctx context.Context, requestID uuid.UUID) ([]workflow.RequestItem, error)
}

// RequestHistoryRepository writes the immutable audit trail of status transitions (Rule 17)
type RequestHistoryRepository interface {
	Save(ctx context.Context, h *workflow.RequestHistory) error
	FindByRequest(ctx context.Context, requestID uuid.UUID) ([]workflow.RequestHistory, error)
}
