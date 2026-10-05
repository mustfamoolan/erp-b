-- Rollback Phase 2: Accounting Core

DROP VIEW  IF EXISTS account_balances         CASCADE;
DROP TABLE IF EXISTS journal_lines             CASCADE;
DROP TABLE IF EXISTS journal_entries           CASCADE;
DROP TABLE IF EXISTS cost_centers              CASCADE;
DROP TABLE IF EXISTS accounts                  CASCADE;
DROP TABLE IF EXISTS accounting_periods        CASCADE;
DROP TABLE IF EXISTS fiscal_years              CASCADE;
DROP SEQUENCE IF EXISTS journal_entry_seq;
