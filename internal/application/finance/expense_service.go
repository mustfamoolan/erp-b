package finance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	appaccounting "m3aml-erp/internal/application/accounting"
	"m3aml-erp/internal/domain/accounting"
	"m3aml-erp/internal/domain/cashbox"
	"m3aml-erp/internal/domain/masterdata"
	"m3aml-erp/internal/repositories"
)

type ExpenseService struct {
	expenseRepo    repositories.ExpenseRepository
	cashboxRepo    repositories.CashboxRepository
	cashTxRepo     repositories.CashTransactionRepository
	accountingSvc  *appaccounting.AccountingService
	accountRepo    repositories.AccountRepository
	masterDataRepo masterdata.MasterDataRepository
}

func NewExpenseService(
	expenseRepo repositories.ExpenseRepository,
	cashboxRepo repositories.CashboxRepository,
	cashTxRepo repositories.CashTransactionRepository,
	accountingSvc *appaccounting.AccountingService,
	accountRepo repositories.AccountRepository,
	masterDataRepo masterdata.MasterDataRepository,
) *ExpenseService {
	return &ExpenseService{
		expenseRepo:    expenseRepo,
		cashboxRepo:    cashboxRepo,
		cashTxRepo:     cashTxRepo,
		accountingSvc:  accountingSvc,
		accountRepo:    accountRepo,
		masterDataRepo: masterDataRepo,
	}
}

type RecordExpenseRequest struct {
	ScopeID           uuid.UUID
	CashboxID         uuid.UUID
	ExpenseCategoryID uuid.UUID
	Amount            decimal.Decimal
	Currency          string
	ExpenseDate       time.Time
	Purpose           string
	PaidTo            string
	Description       string
	PerformedBy       uuid.UUID
}

func (s *ExpenseService) RecordExpense(ctx context.Context, req RecordExpenseRequest) (*cashbox.Expense, error) {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	cb, err := s.cashboxRepo.FindByID(ctx, req.CashboxID)
	if err != nil {
		return nil, fmt.Errorf("cashbox not found: %w", err)
	}

	if cb.ScopeID != req.ScopeID {
		return nil, fmt.Errorf("cashbox does not belong to scope")
	}

	cat, err := s.masterDataRepo.GetExpenseCategoryByID(ctx, req.ExpenseCategoryID)
	if err != nil {
		return nil, fmt.Errorf("expense category not found: %w", err)
	}

	docNum, err := s.expenseRepo.GetNextDocumentNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate document number: %w", err)
	}

	expense := &cashbox.Expense{
		DocumentNumber:    docNum,
		ScopeID:           req.ScopeID,
		CashboxID:         req.CashboxID,
		ExpenseCategoryID: req.ExpenseCategoryID,
		Amount:            req.Amount.InexactFloat64(),
		Currency:          req.Currency,
		ExpenseDate:       req.ExpenseDate,
		Purpose:           req.Purpose,
		PaidTo:            req.PaidTo,
		Description:       req.Description,
		CreatedBy:         req.PerformedBy,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// We need to create the cash transaction
	cashTx := &cashbox.CashTransaction{
		CashboxID:       req.CashboxID,
		Direction:       cashbox.CashDirectionOut,
		Amount:          req.Amount.Neg(), // Outgoing
		Currency:        req.Currency,
		SourceType:      "EXPENSE",
		SourceID:        &expense.ID, // Wait, expense.ID is empty. Let's let gorm handle it. Actually, Expense ID is auto-generated in DB, but we can generate it here to link.
		TransactionDate: req.ExpenseDate,
		Description:     req.Purpose,
		PerformedBy:     req.PerformedBy,
		Status:          cashbox.CashTransactionCompleted,
	}
	expense.ID = uuid.New()
	cashTx.SourceID = &expense.ID
	cashTx.ID = uuid.New()

	postJournal := func(txCtx context.Context) error {
		journalReq := appaccounting.CreateJournalEntryRequest{
			EntryDate:   req.ExpenseDate,
			Description: fmt.Sprintf("صرف نقدي: %s", req.Purpose),
			SourceType:  accounting.SourceTypeExpense,
			SourceID:    &expense.ID,
			ScopeID:     req.ScopeID,
			CreatedBy:   req.PerformedBy,
			Lines: []appaccounting.JournalLineInput{
				{
					AccountID:   cat.AccountID,
					Debit:       req.Amount,
					Credit:      decimal.Zero,
					ScopeID:     req.ScopeID,
					Description: req.Description,
				},
				{
					AccountID:   cb.AccountID,
					Debit:       decimal.Zero,
					Credit:      req.Amount,
					ScopeID:     req.ScopeID,
					Description: req.Description,
				},
			},
		}

		entry, err := s.accountingSvc.CreateDraftEntry(txCtx, journalReq)
		if err != nil {
			return err
		}

		if err := s.accountingSvc.PostEntry(txCtx, entry.ID, req.PerformedBy); err != nil {
			return err
		}

		expense.JournalEntryID = entry.ID
		cashTx.JournalEntryID = &entry.ID

		if err := s.cashTxRepo.Save(txCtx, cashTx); err != nil {
			return err
		}

		return nil
	}

	if err := s.expenseRepo.Create(ctx, expense, postJournal); err != nil {
		return nil, fmt.Errorf("failed to record expense: %w", err)
	}

	return expense, nil
}
