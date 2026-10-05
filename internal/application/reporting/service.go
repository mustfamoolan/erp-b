package appreporting

import (
	"context"
	"m3aml-erp/internal/domain/reporting"
	reportingrepo "m3aml-erp/internal/repositories/postgres/reporting"

	"github.com/google/uuid"
)

type Service struct {
	repo *reportingrepo.Repository
}

func NewReportingService(repo *reportingrepo.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetAccountStatement(ctx context.Context, accountID uuid.UUID, filter reporting.DateRange) (*reporting.AccountStatementReport, error) {
	// TODO: Consider scope checks if specific accounts belong strictly to scopes.
	// Currently assuming if the user has reports.view, they can view statements for accessible accounts.
	return s.repo.GetAccountStatement(ctx, accountID, filter)
}

func (s *Service) GetTrialBalance(ctx context.Context, filter reporting.DateRange) (*reporting.TrialBalanceReport, error) {
	return s.repo.GetTrialBalance(ctx, filter)
}

func (s *Service) GetStockBalance(ctx context.Context, scopeID *uuid.UUID) (*reporting.StockBalanceReport, error) {
	return s.repo.GetStockBalance(ctx, scopeID)
}

// CashboxStatement is essentially an AccountStatement for a cashbox's linked account.
func (s *Service) GetCashboxStatement(ctx context.Context, accountID uuid.UUID, cashboxID, scopeID uuid.UUID, cashboxName string, filter reporting.DateRange) (*reporting.CashboxStatementReport, error) {
	stmt, err := s.repo.GetAccountStatement(ctx, accountID, filter)
	if err != nil {
		return nil, err
	}

	return &reporting.CashboxStatementReport{
		CashboxID:      cashboxID,
		CashboxName:    cashboxName,
		ScopeID:        scopeID,
		AccountID:      accountID,
		OpeningBalance: stmt.OpeningBalance,
		ClosingBalance: stmt.ClosingBalance,
		Lines:          stmt.Lines,
	}, nil
}
