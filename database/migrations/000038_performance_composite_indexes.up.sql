-- Phase 12: High-Performance Composite Indexes for Millions of Records
-- Optimizes queries for factory views, auditor/accountant queues, cashbox ledgers, and master data lookups.

-- 1. Financial Requests composite indexes
CREATE INDEX IF NOT EXISTS idx_freq_scope_created_desc ON financial_requests(scope_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_freq_factory_created_desc ON financial_requests(factory_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_freq_status_created_desc ON financial_requests(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_freq_status_scope_created ON financial_requests(status, scope_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_freq_adv_seq ON financial_requests(factory_id, advance_sequence_number);

-- 2. Request Items & History composite indexes
CREATE INDEX IF NOT EXISTS idx_request_items_request_id ON request_items(request_id);
CREATE INDEX IF NOT EXISTS idx_request_items_exp_cat ON request_items(expense_category_id);
CREATE INDEX IF NOT EXISTS idx_request_history_req_created ON request_history(request_id, created_at ASC);

-- 3. Cash Transactions & Cashboxes indexes
CREATE INDEX IF NOT EXISTS idx_cash_tx_box_created ON cash_transactions(cashbox_id, created_at DESC);

-- 4. Scopes & Areas indexes
CREATE INDEX IF NOT EXISTS idx_scopes_type ON organization_scopes(type);
CREATE INDEX IF NOT EXISTS idx_scopes_area_id ON organization_scopes(area_id);
CREATE INDEX IF NOT EXISTS idx_areas_status ON areas(status);
