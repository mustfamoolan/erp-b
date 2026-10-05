-- Migration 000020: Salary expense category kind + request attachments

-- ─── 1. Expense category kind ──────────────────────────────────────────────
-- SALARY categories follow a separate request path (defined later by the business);
-- until then the backend refuses to create requests against them.
ALTER TABLE expense_categories
    ADD COLUMN kind VARCHAR(20) NOT NULL DEFAULT 'STANDARD'
        CONSTRAINT expense_categories_kind_check CHECK (kind IN ('STANDARD', 'SALARY'));

-- Salary expense account: child of the header mapped by EXPENSE_CATEGORY_PARENT (no hard-coded parent).
INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '5270', 'رواتب وأجور', 'EXPENSE', m.account_id, TRUE, 5270
FROM account_mappings m
WHERE m.key = 'EXPENSE_CATEGORY_PARENT'
  AND NOT EXISTS (SELECT 1 FROM accounts WHERE code = '5270');

INSERT INTO expense_categories (code, name, account_id, sort_order, kind)
SELECT 'SALARY', 'رواتب', a.id, 5, 'SALARY'
FROM accounts a
WHERE a.code = '5270'
ON CONFLICT (code) DO NOTHING;

-- ─── 2. Request attachments (receipts: images / PDF) ───────────────────────
CREATE TABLE request_attachments (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id   UUID          NOT NULL REFERENCES financial_requests(id) ON DELETE RESTRICT,
    scope_id     UUID          NOT NULL REFERENCES organization_scopes(id) ON DELETE RESTRICT,
    file_name    VARCHAR(255)  NOT NULL,
    stored_name  VARCHAR(255)  NOT NULL UNIQUE,
    mime_type    VARCHAR(100)  NOT NULL
        CONSTRAINT request_attachments_mime_check
        CHECK (mime_type IN ('application/pdf', 'image/jpeg', 'image/png', 'image/webp')),
    size_bytes   BIGINT        NOT NULL CHECK (size_bytes > 0 AND size_bytes <= 5242880),
    uploaded_by  UUID          NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_request_attachments_request_id ON request_attachments(request_id);
