-- Rollback Phase 3: Cashboxes

DROP VIEW  IF EXISTS cashbox_balances   CASCADE;
DROP TABLE IF EXISTS cash_transfers     CASCADE;
DROP TABLE IF EXISTS cash_transactions  CASCADE;
DROP TABLE IF EXISTS cashboxes          CASCADE;
