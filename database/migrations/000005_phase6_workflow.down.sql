-- Rollback Phase 6: Workflow

DROP TABLE IF EXISTS request_history      CASCADE;
DROP TABLE IF EXISTS request_items        CASCADE;
DROP TABLE IF EXISTS financial_requests   CASCADE;
DROP SEQUENCE IF EXISTS financial_request_seq;
