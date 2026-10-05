package purchasing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	
	appaccounting "m3aml-erp/internal/application/accounting"
	appaudit "m3aml-erp/internal/application/audit"
	domainaudit "m3aml-erp/internal/domain/audit"
	"m3aml-erp/internal/domain/cashbox"
	"m3aml-erp/internal/domain/purchasing"
	"m3aml-erp/internal/repositories"
)

type PurchasingService struct {
	invoiceRepo     repositories.PurchaseInvoiceRepository
	itemRepo        repositories.PurchaseInvoiceItemRepository
	cashTxRepo      repositories.CashTransactionRepository
	cashboxRepo     repositories.CashboxRepository
	accountRepo     repositories.AccountRepository
	accountingSvc   *appaccounting.AccountingService
	auditSvc        *appaudit.AuditService
	// docNumberFn generates the document number
	docNumberFn func(ctx context.Context) (string, error)
}

func NewPurchasingService(
	invoiceRepo repositories.PurchaseInvoiceRepository,
	itemRepo repositories.PurchaseInvoiceItemRepository,
	cashTxRepo repositories.CashTransactionRepository,
	cashboxRepo repositories.CashboxRepository,
	accountRepo repositories.AccountRepository,
	accountingSvc *appaccounting.AccountingService,
	auditSvc *appaudit.AuditService,
	docNumberFn func(ctx context.Context) (string, error),
) *PurchasingService {
	return &PurchasingService{
		invoiceRepo:   invoiceRepo,
		itemRepo:      itemRepo,
		cashTxRepo:    cashTxRepo,
		cashboxRepo:   cashboxRepo,
		accountRepo:   accountRepo,
		accountingSvc: accountingSvc,
		auditSvc:      auditSvc,
		docNumberFn:   docNumberFn,
	}
}

type CreateInvoiceInput struct {
	ScopeID       uuid.UUID
	InvoiceNumber string
	InvoiceDate   time.Time
	SupplierName  string
	Discount      decimal.Decimal
	Tax           decimal.Decimal
	Currency      string
	Notes         string
	Attachments   *string
	CreatedBy     uuid.UUID
	Items         []CreateInvoiceItemInput
}

type CreateInvoiceItemInput struct {
	Description string
	Quantity    decimal.Decimal
	UnitID      *uuid.UUID
	UnitPrice   decimal.Decimal
	Notes       string
}

func (s *PurchasingService) CreateInvoice(ctx context.Context, input CreateInvoiceInput) (*purchasing.PurchaseInvoice, error) {
	docNum, err := s.docNumberFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate document number: %w", err)
	}

	total := decimal.Zero
	for _, i := range input.Items {
		total = total.Add(i.UnitPrice.Mul(i.Quantity))
	}
	
	netAmount := total.Sub(input.Discount).Add(input.Tax)

	invoiceID := uuid.New()
	inv := &purchasing.PurchaseInvoice{
		ID:             invoiceID,
		DocumentNumber: docNum,
		InvoiceNumber:  input.InvoiceNumber,
		InvoiceDate:    input.InvoiceDate,
		ScopeID:        input.ScopeID,
		SupplierName:   input.SupplierName,
		TotalAmount:    total,
		Discount:       input.Discount,
		Tax:            input.Tax,
		NetAmount:      netAmount,
		Currency:       input.Currency,
		PaymentStatus:  purchasing.PaymentUnpaid,
		Status:         purchasing.InvoiceDraft,
		Notes:          input.Notes,
		Attachments:    input.Attachments,
		CreatedBy:      input.CreatedBy,
	}

	if err := s.invoiceRepo.Save(ctx, inv); err != nil {
		return nil, fmt.Errorf("save invoice failed: %w", err)
	}

	var items []purchasing.PurchaseInvoiceItem
	for _, i := range input.Items {
		lineTotal := i.UnitPrice.Mul(i.Quantity)
		items = append(items, purchasing.PurchaseInvoiceItem{
			ID:          uuid.New(),
			InvoiceID:   invoiceID,
			Description: i.Description,
			Quantity:    i.Quantity,
			UnitID:      i.UnitID,
			UnitPrice:   i.UnitPrice,
			Total:       lineTotal,
			Notes:       i.Notes,
		})
	}

	if err := s.itemRepo.SaveAll(ctx, items); err != nil {
		return nil, fmt.Errorf("save items failed: %w", err)
	}

	if s.auditSvc != nil {
		scope := input.ScopeID
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     input.CreatedBy,
			ScopeID:    &scope,
			Action:     domainaudit.AuditCreate,
			EntityType: "purchase_invoice",
			EntityID:   &invoiceID,
			NewValues: map[string]any{
				"document_number": docNum,
				"net_amount":      netAmount.String(),
				"supplier":        input.SupplierName,
			},
		})
	}

	return inv, nil
}

func (s *PurchasingService) ApproveInvoice(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	inv, err := s.invoiceRepo.FindByID(ctx, id)
	if err != nil || inv == nil {
		return fmt.Errorf("invoice not found")
	}
	if inv.Status != purchasing.InvoiceDraft {
		return fmt.Errorf("only draft invoices can be approved")
	}

	inv.Status = purchasing.InvoiceApproved
	if err := s.invoiceRepo.Update(ctx, inv); err != nil {
		return err
	}

	if s.auditSvc != nil {
		scope := inv.ScopeID
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     userID,
			ScopeID:    &scope,
			Action:     domainaudit.AuditApprove,
			EntityType: "purchase_invoice",
			EntityID:   &inv.ID,
		})
	}

	return nil
}

// PayInvoice handles payment from a specific factory cashbox.
func (s *PurchasingService) PayInvoice(ctx context.Context, invoiceID uuid.UUID, cashboxID uuid.UUID, amount decimal.Decimal, expenseCategoryID uuid.UUID, userID uuid.UUID) error {
	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil || inv == nil {
		return fmt.Errorf("invoice not found")
	}
	if inv.Status != purchasing.InvoiceApproved {
		return fmt.Errorf("invoice must be approved before payment")
	}
	if inv.PaymentStatus == purchasing.PaymentPaid {
		return fmt.Errorf("invoice is already fully paid")
	}

	cb, err := s.cashboxRepo.FindByID(ctx, cashboxID)
	if err != nil || cb == nil {
		return fmt.Errorf("cashbox not found")
	}

	// Verify the cashbox belongs to the same scope
	if cb.ScopeID != inv.ScopeID {
		return fmt.Errorf("cashbox does not belong to the invoice's factory")
	}

	// Find the expense account linked to the category
	expenseCategory, err := s.accountRepo.FindByID(ctx, expenseCategoryID)
	if err != nil || expenseCategory == nil {
		return fmt.Errorf("expense category not found")
	}

	now := time.Now()
	
	// Create CashTransaction OUT
	tx := &cashbox.CashTransaction{
		ID:              uuid.New(),
		CashboxID:       cb.ID,
		Amount:          amount,
		Currency:        inv.Currency,
		Direction:       cashbox.CashDirectionOut,
		SourceType:      "PURCHASE_PAYMENT",
		Description:     "Payment for Invoice " + inv.DocumentNumber,
		PerformedBy:     userID,
		TransactionDate: now,
		Status:          cashbox.CashTransactionCompleted,
	}

	// Atomically execute transfer and journal entry
	err = s.cashTxRepo.ExecuteTransfer(ctx, tx, nil, func(txCtx context.Context) error {
		// Debit Expense Account, Credit Cashbox Account
		jeReq := appaccounting.CreateJournalEntryRequest{
			EntryDate:   now,
			Description: "Purchase Payment " + inv.DocumentNumber,
			SourceType:  "PURCHASE_PAYMENT",
			SourceID:    &tx.ID,
			ScopeID:     inv.ScopeID,
			CreatedBy:   userID,
			Lines: []appaccounting.JournalLineInput{
				{
					AccountID:   expenseCategory.ID, // Debit Expense
					Debit:       amount,
					Credit:      decimal.Zero,
					Description: "Purchase Expense for " + inv.SupplierName,
					ScopeID:     inv.ScopeID,
				},
				{
					AccountID:   cb.AccountID, // Credit Cashbox
					Debit:       decimal.Zero,
					Credit:      amount,
					Description: "Cash Payment for " + inv.SupplierName,
					ScopeID:     inv.ScopeID,
				},
			},
		}

		entry, accErr := s.accountingSvc.CreateDraftEntry(txCtx, jeReq)
		if accErr != nil {
			return fmt.Errorf("journal entry creation failed: %w", accErr)
		}
		if accErr := s.accountingSvc.PostEntry(txCtx, entry.ID, userID); accErr != nil {
			return fmt.Errorf("journal entry post failed: %w", accErr)
		}
		tx.JournalEntryID = &entry.ID
		return nil
	})

	if err != nil {
		return err
	}

	// Update Invoice status
	// For simplicity, we just mark it PAID. 
	// In reality we should track total paid vs net amount.
	inv.PaymentStatus = purchasing.PaymentPaid
	if err := s.invoiceRepo.Update(ctx, inv); err != nil {
		return err
	}

	if s.auditSvc != nil {
		scope := inv.ScopeID
		_ = s.auditSvc.RecordAudit(ctx, appaudit.RecordAuditInput{
			UserID:     userID,
			ScopeID:    &scope,
			Action:     domainaudit.AuditPay,
			EntityType: "purchase_invoice",
			EntityID:   &inv.ID,
			NewValues: map[string]any{
				"amount_paid": amount.String(),
			},
		})
	}

	return nil
}

func (s *PurchasingService) GetByScope(ctx context.Context, scopeID uuid.UUID) ([]purchasing.PurchaseInvoice, error) {
	return s.invoiceRepo.FindByScope(ctx, scopeID)
}

func (s *PurchasingService) GetByID(ctx context.Context, id uuid.UUID) (*purchasing.PurchaseInvoice, error) {
	return s.invoiceRepo.FindByID(ctx, id)
}
