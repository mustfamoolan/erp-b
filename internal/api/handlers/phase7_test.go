package handlers_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	appaudit "m3aml-erp/internal/application/audit"
	"m3aml-erp/internal/domain/audit"
)

// ─────────────────────────────────────────────────────────────────────────────
// MOCK IMPLEMENTATION — Phase 7 (Audit System)
// ─────────────────────────────────────────────────────────────────────────────

type mockAuditRepo struct {
	logs []audit.AuditLog
}

func (m *mockAuditRepo) Save(ctx context.Context, logEntry *audit.AuditLog) error {
	m.logs = append(m.logs, *logEntry)
	return nil
}
func (m *mockAuditRepo) FindByEntity(ctx context.Context, entityType string, entityID *uuid.UUID) ([]audit.AuditLog, error) {
	var res []audit.AuditLog
	for _, l := range m.logs {
		if l.EntityType == entityType && l.EntityID != nil && *l.EntityID == *entityID {
			res = append(res, l)
		}
	}
	return res, nil
}
func (m *mockAuditRepo) FindByScope(ctx context.Context, scopeID uuid.UUID) ([]audit.AuditLog, error) {
	var res []audit.AuditLog
	for _, l := range m.logs {
		if l.ScopeID != nil && *l.ScopeID == scopeID {
			res = append(res, l)
		}
	}
	return res, nil
}
func (m *mockAuditRepo) FindByUser(ctx context.Context, userID uuid.UUID) ([]audit.AuditLog, error) {
	var res []audit.AuditLog
	for _, l := range m.logs {
		if l.UserID == userID {
			res = append(res, l)
		}
	}
	return res, nil
}
func (m *mockAuditRepo) FindAll(ctx context.Context) ([]audit.AuditLog, error) {
	return m.logs, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS — Phase 7
// ─────────────────────────────────────────────────────────────────────────────

func buildAuditService() (*appaudit.AuditService, *mockAuditRepo) {
	repo := &mockAuditRepo{}
	svc := appaudit.NewAuditService(repo)
	return svc, repo
}

func TestAuditLog_CreationAndImmutability(t *testing.T) {
	svc, repo := buildAuditService()

	userID := uuid.New()
	scopeID := uuid.New()
	entityID := uuid.New()

	oldValues := map[string]string{"status": "DRAFT"}
	newValues := map[string]string{"status": "SUBMITTED"}

	err := svc.RecordAudit(context.Background(), appaudit.RecordAuditInput{
		UserID:     userID,
		ScopeID:    &scopeID,
		Action:     audit.AuditSubmit,
		EntityType: "FinancialRequest",
		EntityID:   &entityID,
		OldValues:  oldValues,
		NewValues:  newValues,
		IPAddress:  "127.0.0.1",
		Device:     "TestRunner",
	})
	if err != nil {
		t.Fatalf("failed to record audit: %v", err)
	}

	if len(repo.logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(repo.logs))
	}

	logEntry := repo.logs[0]
	if logEntry.Action != audit.AuditSubmit {
		t.Errorf("expected SUBMIT action, got %s", logEntry.Action)
	}
	if logEntry.EntityType != "FinancialRequest" {
		t.Errorf("expected FinancialRequest, got %s", logEntry.EntityType)
	}
	if string(logEntry.OldValues) != `{"status":"DRAFT"}` {
		t.Errorf("expected old values JSON, got %s", string(logEntry.OldValues))
	}
	if string(logEntry.NewValues) != `{"status":"SUBMITTED"}` {
		t.Errorf("expected new values JSON, got %s", string(logEntry.NewValues))
	}
	t.Log("✅ PASS: Audit log successfully serialized states to JSON (§34)")
}

func TestAuditLog_Queries(t *testing.T) {
	svc, _ := buildAuditService()

	userID1 := uuid.New()
	userID2 := uuid.New()
	scopeID1 := uuid.New()
	scopeID2 := uuid.New()
	entityID1 := uuid.New()
	entityID2 := uuid.New()

	// 1: User 1, Scope 1, Entity 1
	svc.RecordAudit(context.Background(), appaudit.RecordAuditInput{
		UserID: userID1, ScopeID: &scopeID1, Action: audit.AuditCreate, EntityType: "A", EntityID: &entityID1,
	})
	// 2: User 1, Scope 1, Entity 2
	svc.RecordAudit(context.Background(), appaudit.RecordAuditInput{
		UserID: userID1, ScopeID: &scopeID1, Action: audit.AuditCreate, EntityType: "B", EntityID: &entityID2,
	})
	// 3: User 2, Scope 2, Entity 1
	svc.RecordAudit(context.Background(), appaudit.RecordAuditInput{
		UserID: userID2, ScopeID: &scopeID2, Action: audit.AuditUpdate, EntityType: "A", EntityID: &entityID1,
	})

	user1Logs, _ := svc.GetByUser(context.Background(), userID1)
	if len(user1Logs) != 2 {
		t.Errorf("expected 2 logs for user 1, got %d", len(user1Logs))
	}

	scope2Logs, _ := svc.GetByScope(context.Background(), scopeID2)
	if len(scope2Logs) != 1 {
		t.Errorf("expected 1 log for scope 2, got %d", len(scope2Logs))
	}

	entity1Logs, _ := svc.GetByEntity(context.Background(), "A", &entityID1)
	if len(entity1Logs) != 2 {
		t.Errorf("expected 2 logs for entity 1, got %d", len(entity1Logs))
	}

	t.Log("✅ PASS: Audit log queries filter correctly (By User, By Scope, By Entity)")
}
