-- Phase 3: Cashboxes
-- Roadmap §23-24 — Cashboxes and Cash Transactions
--
-- Rules enforced:
-- Rule 10: Never use mutable balances as financial source of truth
--          (cashbox balance is always computed from cash_transactions)
-- Rule 11: Every financial movement must have a traceable transaction
-- Rule 18: PostgreSQL transactions for atomic operations
-- Roadmap §7: Cross-scope transfers must be modeled explicitly

-- ============================================================
-- 1. CASHBOXES — Roadmap §23
-- Each cashbox is linked to an accounting account.
-- ============================================================
CREATE TABLE cashboxes (
    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    scope_id   UUID          NOT NULL REFERENCES organization_scopes(id) ON DELETE RESTRICT,
    name       VARCHAR(150)  NOT NULL,
    account_id UUID          NOT NULL REFERENCES accounts(id)            ON DELETE RESTRICT,
    currency   VARCHAR(10)   NOT NULL DEFAULT 'SAR',
    status     VARCHAR(20)   NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT cashboxes_scope_account_unique UNIQUE (scope_id, account_id)
);

CREATE INDEX idx_cashboxes_scope_id   ON cashboxes(scope_id);
CREATE INDEX idx_cashboxes_account_id ON cashboxes(account_id);

COMMENT ON TABLE cashboxes IS 'Cash holding units linked to accounting accounts. Balance is computed from transactions. Roadmap §23.';

-- ============================================================
-- 2. CASH TRANSACTIONS — Roadmap §24
-- Every cash movement must be traceable. Immutable once completed.
-- ============================================================
CREATE TABLE cash_transactions (
    id               UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    cashbox_id       UUID          NOT NULL REFERENCES cashboxes(id)       ON DELETE RESTRICT,
    amount           NUMERIC(18,4) NOT NULL CHECK (amount > 0),
    currency         VARCHAR(10)   NOT NULL DEFAULT 'SAR',
    direction        VARCHAR(10)   NOT NULL CHECK (direction IN ('IN', 'OUT')),
    source_type      VARCHAR(50)   NOT NULL,
    source_id        UUID,
    journal_entry_id UUID          REFERENCES journal_entries(id)           ON DELETE RESTRICT,
    description      TEXT,
    performed_by     UUID          NOT NULL REFERENCES users(id)            ON DELETE RESTRICT,
    transaction_date DATE          NOT NULL,
    status           VARCHAR(20)   NOT NULL DEFAULT 'COMPLETED'
                                   CHECK (status IN ('PENDING', 'COMPLETED', 'CANCELLED')),
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cash_transactions_cashbox_id  ON cash_transactions(cashbox_id);
CREATE INDEX idx_cash_transactions_date        ON cash_transactions(transaction_date);
CREATE INDEX idx_cash_transactions_source      ON cash_transactions(source_type, source_id);
CREATE INDEX idx_cash_transactions_je          ON cash_transactions(journal_entry_id);

COMMENT ON TABLE cash_transactions IS 'Immutable record of every cash movement. Balance computed from here. Rule 10, 11.';

-- ============================================================
-- 3. CASH TRANSFERS — Roadmap §24, §7
-- Explicit cross-scope transfer modeling.
-- Creates two cash_transaction records + one journal entry atomically.
-- ============================================================
CREATE TABLE cash_transfers (
    id                  UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    source_cashbox_id   UUID          NOT NULL REFERENCES cashboxes(id) ON DELETE RESTRICT,
    dest_cashbox_id     UUID          NOT NULL REFERENCES cashboxes(id) ON DELETE RESTRICT,
    source_scope_id     UUID          NOT NULL REFERENCES organization_scopes(id),
    dest_scope_id       UUID          NOT NULL REFERENCES organization_scopes(id),
    amount              NUMERIC(18,4) NOT NULL CHECK (amount > 0),
    currency            VARCHAR(10)   NOT NULL DEFAULT 'SAR',
    reference_document  VARCHAR(100),
    journal_entry_id    UUID          REFERENCES journal_entries(id) ON DELETE RESTRICT,
    performed_by        UUID          NOT NULL REFERENCES users(id)  ON DELETE RESTRICT,
    transfer_date       DATE          NOT NULL,
    status              VARCHAR(20)   NOT NULL DEFAULT 'COMPLETED'
                                      CHECK (status IN ('PENDING', 'COMPLETED', 'CANCELLED')),
    notes               TEXT,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    -- Source and destination cashboxes must be different
    CONSTRAINT cash_transfers_different_cashboxes CHECK (source_cashbox_id != dest_cashbox_id)
);

CREATE INDEX idx_cash_transfers_source_cashbox ON cash_transfers(source_cashbox_id);
CREATE INDEX idx_cash_transfers_dest_cashbox   ON cash_transfers(dest_cashbox_id);
CREATE INDEX idx_cash_transfers_date           ON cash_transfers(transfer_date);

COMMENT ON TABLE cash_transfers IS 'Explicit cross-scope cash transfer. Creates 2 transactions + 1 journal entry atomically. Roadmap §7, §24.';

-- ============================================================
-- 4. CASHBOX BALANCE VIEW
-- Computed from cash_transactions. Never from a stored field.
-- Rule 10.
-- ============================================================
CREATE VIEW cashbox_balances AS
SELECT
    ct.cashbox_id,
    cb.name          AS cashbox_name,
    cb.scope_id,
    cb.currency,
    COALESCE(SUM(CASE WHEN ct.direction = 'IN'  THEN ct.amount ELSE 0 END), 0) AS total_in,
    COALESCE(SUM(CASE WHEN ct.direction = 'OUT' THEN ct.amount ELSE 0 END), 0) AS total_out,
    COALESCE(SUM(CASE WHEN ct.direction = 'IN'  THEN ct.amount ELSE -ct.amount END), 0) AS current_balance
FROM cash_transactions ct
JOIN cashboxes cb ON cb.id = ct.cashbox_id
WHERE ct.status = 'COMPLETED'
GROUP BY ct.cashbox_id, cb.name, cb.scope_id, cb.currency;

COMMENT ON VIEW cashbox_balances IS 'Real-time cashbox balances computed from completed transactions. Rule 10.';

-- ============================================================
-- 5. SEED: Administration Main Cashbox
-- ============================================================
INSERT INTO cashboxes (scope_id, name, account_id, currency)
SELECT
    s.id,
    'الصندوق الرئيسي للإدارة',
    a.id,
    'SAR'
FROM organization_scopes s, accounts a
WHERE s.code = 'ADM' AND a.code = '1101';
