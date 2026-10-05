-- Phase 7: Centralized Audit System
-- Roadmap §34-35 — AUDIT SYSTEM

-- ============================================================
-- AUDIT LOG — centralized, append-only
-- Rule 17: Every important workflow transition must be audited.
-- Roadmap §34 — tracks every critical action.
-- Roadmap §35 — IMMUTABILITY: no silent deletes.
-- ============================================================
CREATE TABLE audit_logs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    scope_id    UUID        REFERENCES organization_scopes(id),
    action      VARCHAR(50) NOT NULL
                CHECK (action IN (
                    'CREATE','UPDATE','SUBMIT','APPROVE','REJECT',
                    'PAY','RECEIVE','POST','VOID','REVERSE','TRANSFER',
                    'ADJUST','LOGIN','LOGOUT','CLOSE_PERIOD'
                )),
    entity_type VARCHAR(100) NOT NULL,
    entity_id   UUID,
    old_values  JSONB,
    new_values  JSONB,
    ip_address  VARCHAR(50),
    device      VARCHAR(200),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user_id     ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_scope_id    ON audit_logs(scope_id);
CREATE INDEX idx_audit_logs_action      ON audit_logs(action);
CREATE INDEX idx_audit_logs_entity      ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_date        ON audit_logs(created_at);

-- Audit log is append-only — prevent UPDATE and DELETE
-- This enforces Roadmap §35 IMMUTABILITY at the database level.
CREATE RULE audit_logs_no_update AS ON UPDATE TO audit_logs DO INSTEAD NOTHING;
CREATE RULE audit_logs_no_delete AS ON DELETE TO audit_logs DO INSTEAD NOTHING;

COMMENT ON TABLE audit_logs IS 'Append-only audit log. UPDATE and DELETE are blocked by rules. Roadmap §34-35.';
COMMENT ON COLUMN audit_logs.old_values IS 'JSON snapshot of record before change.';
COMMENT ON COLUMN audit_logs.new_values IS 'JSON snapshot of record after change.';
