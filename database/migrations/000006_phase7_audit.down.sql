-- Rollback Phase 7: Audit
-- Note: Rules must be dropped before the table.

DROP RULE IF EXISTS audit_logs_no_update ON audit_logs;
DROP RULE IF EXISTS audit_logs_no_delete ON audit_logs;
DROP TABLE IF EXISTS audit_logs CASCADE;
