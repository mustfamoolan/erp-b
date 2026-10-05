-- Phase 10: Factory Expenses
-- Roadmap §28

CREATE SEQUENCE expense_seq START 1;

CREATE TABLE expenses (
    id                  UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    document_number     VARCHAR(50)    NOT NULL,
    scope_id            UUID           NOT NULL REFERENCES organization_scopes(id) ON DELETE RESTRICT,
    cashbox_id          UUID           NOT NULL REFERENCES cashboxes(id) ON DELETE RESTRICT,
    expense_category_id UUID           NOT NULL REFERENCES expense_categories(id) ON DELETE RESTRICT,
    amount              NUMERIC(18,4)  NOT NULL CHECK (amount > 0),
    currency            VARCHAR(10)    NOT NULL DEFAULT 'IQD',
    expense_date        DATE           NOT NULL,
    purpose             VARCHAR(500)   NOT NULL,
    paid_to             VARCHAR(255)   NOT NULL,
    description         TEXT,
    journal_entry_id    UUID           REFERENCES journal_entries(id) ON DELETE RESTRICT,
    created_by          UUID           NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at          TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT expenses_document_number_unique UNIQUE (document_number)
);

CREATE INDEX idx_expenses_scope_id   ON expenses(scope_id);
CREATE INDEX idx_expenses_cashbox_id ON expenses(cashbox_id);
CREATE INDEX idx_expenses_date       ON expenses(expense_date);

COMMENT ON TABLE expenses IS 'Factory cash expenses recorded by the accountant. Roadmap §28.';
