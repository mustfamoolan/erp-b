-- Phase 2: Accounting Core
-- Roadmap §13-22 — PHASE 2: ACCOUNTING CORE
--
-- Rules enforced:
-- Rule 10: Never use mutable balances as financial source of truth
-- Rule 11: Every financial movement must have a traceable transaction
-- Rule 13: Every accounting entry must be balanced (enforced in app layer + CHECK)
-- Rule 14: Posted entries must not be silently edited or deleted (status machine)
-- Rule 15: Corrections use REVERSAL/ADJUSTMENT
-- Rule 18: PostgreSQL transactions for atomic operations
-- Rule 19: Database constraints for critical data integrity
-- Rule 21: One accounting engine — NOT two separate ones for Admin and Factory
-- Rule 22: One engine with scope/cost-center dimensions

-- ============================================================
-- 1. FISCAL YEARS — Roadmap §18
-- ============================================================
CREATE TABLE fiscal_years (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(50)  NOT NULL,
    start_date  DATE         NOT NULL,
    end_date    DATE         NOT NULL,
    status      VARCHAR(10)  NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'CLOSED')),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT fiscal_years_name_unique UNIQUE (name),
    -- No overlapping fiscal years
    CONSTRAINT fiscal_years_dates_valid CHECK (start_date < end_date)
);

COMMENT ON TABLE fiscal_years IS 'Annual accounting period containers. Roadmap §18.';

-- ============================================================
-- 2. ACCOUNTING PERIODS — Roadmap §18
-- Closed periods must prevent ordinary posting.
-- ============================================================
CREATE TABLE accounting_periods (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    fiscal_year_id UUID        NOT NULL REFERENCES fiscal_years(id) ON DELETE RESTRICT,
    name           VARCHAR(50) NOT NULL,
    period_number  INT         NOT NULL CHECK (period_number BETWEEN 1 AND 12),
    start_date     DATE        NOT NULL,
    end_date       DATE        NOT NULL,
    status         VARCHAR(10) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'CLOSED')),
    closed_by      UUID        REFERENCES users(id) ON DELETE SET NULL,
    closed_at      TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT accounting_periods_unique   UNIQUE (fiscal_year_id, period_number),
    CONSTRAINT accounting_periods_dates    CHECK (start_date < end_date),
    -- If closed, closed_by must be set
    CONSTRAINT accounting_periods_closed_by CHECK (
        status = 'OPEN' OR (status = 'CLOSED' AND closed_by IS NOT NULL)
    )
);

CREATE INDEX idx_accounting_periods_fiscal_year ON accounting_periods(fiscal_year_id);
CREATE INDEX idx_accounting_periods_status       ON accounting_periods(status);
CREATE INDEX idx_accounting_periods_dates        ON accounting_periods(start_date, end_date);

COMMENT ON TABLE accounting_periods IS 'Monthly periods within a fiscal year. Closed periods block posting. Roadmap §18.';

-- ============================================================
-- 3. CHART OF ACCOUNTS — Roadmap §14
-- Hierarchical, data-driven. Do NOT hard-code account IDs in
-- business logic (roadmap rule: §14).
-- Account hierarchy: parent_id = NULL means root.
-- ============================================================
CREATE TABLE accounts (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(20)  NOT NULL,
    name        VARCHAR(200) NOT NULL,
    type        VARCHAR(20)  NOT NULL CHECK (type IN ('ASSET', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE')),
    parent_id   UUID         REFERENCES accounts(id) ON DELETE RESTRICT,
    is_postable BOOLEAN      NOT NULL DEFAULT TRUE,  -- FALSE = header/summary only
    scope_id    UUID         REFERENCES organization_scopes(id) ON DELETE RESTRICT, -- NULL = company-wide
    description TEXT,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    sort_order  INT          NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT accounts_code_unique UNIQUE (code)
);

CREATE INDEX idx_accounts_type      ON accounts(type);
CREATE INDEX idx_accounts_parent_id ON accounts(parent_id);
CREATE INDEX idx_accounts_scope_id  ON accounts(scope_id);
CREATE INDEX idx_accounts_active    ON accounts(is_active);

COMMENT ON TABLE accounts IS 'Chart of Accounts — hierarchical, data-driven. Roadmap §14.';
COMMENT ON COLUMN accounts.is_postable IS 'FALSE for header/group accounts. Only postable accounts accept journal lines.';
COMMENT ON COLUMN accounts.scope_id    IS 'NULL = company-wide account. Non-null = scope-specific account (e.g. Factory A cashbox).';

-- ============================================================
-- 4. COST CENTERS — Roadmap §20
-- Cost centers are NOT accounts. They are analytical dimensions.
-- ============================================================
CREATE TABLE cost_centers (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    scope_id   UUID        NOT NULL REFERENCES organization_scopes(id) ON DELETE RESTRICT,
    code       VARCHAR(50) NOT NULL,
    name       VARCHAR(150) NOT NULL,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT cost_centers_code_unique UNIQUE (code)
);

CREATE INDEX idx_cost_centers_scope_id ON cost_centers(scope_id);

COMMENT ON TABLE cost_centers IS 'Analytical cost centers. Not accounts. Used for dimensional reporting. Roadmap §20.';

-- ============================================================
-- 5. JOURNAL ENTRIES — Roadmap §16
-- Immutable once posted. Corrections via REVERSAL/ADJUSTMENT.
-- Rule 14: Posted entries must not be silently edited or deleted.
-- ============================================================
CREATE TABLE journal_entries (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_number VARCHAR(50) NOT NULL,
    entry_date   DATE        NOT NULL,
    description  TEXT        NOT NULL,
    source_type  VARCHAR(50) NOT NULL DEFAULT 'MANUAL',
    source_id    UUID,                            -- FK to originating document (polymorphic)
    scope_id     UUID        NOT NULL REFERENCES organization_scopes(id) ON DELETE RESTRICT,
    period_id    UUID        NOT NULL REFERENCES accounting_periods(id)  ON DELETE RESTRICT,
    status       VARCHAR(20) NOT NULL DEFAULT 'DRAFT'
                             CHECK (status IN ('DRAFT', 'POSTED', 'VOIDED', 'REVERSED')),
    posted_by    UUID        REFERENCES users(id) ON DELETE SET NULL,
    posted_at    TIMESTAMPTZ,
    reversal_of  UUID        REFERENCES journal_entries(id) ON DELETE RESTRICT,
    created_by   UUID        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT journal_entries_number_unique UNIQUE (entry_number),
    -- Posted entries must have posted_by
    CONSTRAINT journal_entries_posted_by CHECK (
        status != 'POSTED' OR (status = 'POSTED' AND posted_by IS NOT NULL)
    )
);

CREATE INDEX idx_journal_entries_scope_id  ON journal_entries(scope_id);
CREATE INDEX idx_journal_entries_period_id ON journal_entries(period_id);
CREATE INDEX idx_journal_entries_status    ON journal_entries(status);
CREATE INDEX idx_journal_entries_date      ON journal_entries(entry_date);
CREATE INDEX idx_journal_entries_source    ON journal_entries(source_type, source_id);

COMMENT ON TABLE journal_entries IS 'Double-entry accounting journal. Immutable once posted. Roadmap §16-17.';

-- ============================================================
-- 6. JOURNAL LINES — Roadmap §16
-- Database-level constraints enforce debit/credit rules.
-- Application layer enforces SUM(debit) = SUM(credit).
-- ============================================================
CREATE TABLE journal_lines (
    id               UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    journal_entry_id UUID           NOT NULL REFERENCES journal_entries(id) ON DELETE RESTRICT,
    account_id       UUID           NOT NULL REFERENCES accounts(id)        ON DELETE RESTRICT,
    debit            NUMERIC(18,4)  NOT NULL DEFAULT 0 CHECK (debit >= 0),
    credit           NUMERIC(18,4)  NOT NULL DEFAULT 0 CHECK (credit >= 0),
    description      TEXT,
    scope_id         UUID           NOT NULL REFERENCES organization_scopes(id) ON DELETE RESTRICT,
    cost_center_id   UUID           REFERENCES cost_centers(id) ON DELETE SET NULL,
    reference        VARCHAR(100),
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    -- A line cannot have both debit AND credit — Roadmap §16
    CONSTRAINT journal_lines_not_both CHECK (NOT (debit > 0 AND credit > 0)),
    -- At least one of debit or credit must be non-zero (no zero-value lines)
    CONSTRAINT journal_lines_not_zero CHECK (debit > 0 OR credit > 0)
);

CREATE INDEX idx_journal_lines_entry_id    ON journal_lines(journal_entry_id);
CREATE INDEX idx_journal_lines_account_id  ON journal_lines(account_id);
CREATE INDEX idx_journal_lines_scope_id    ON journal_lines(scope_id);

COMMENT ON TABLE journal_lines IS 'Individual debit/credit lines. DB constraints enforce debit/credit rules. App enforces balance. Roadmap §16.';

-- ============================================================
-- 7. ACCOUNT BALANCES VIEW
-- Computed from journal_lines — never from a manually edited field.
-- Rule 10: Never use mutable balances as financial source of truth.
-- ============================================================
CREATE VIEW account_balances AS
SELECT
    jl.account_id,
    a.code               AS account_code,
    a.name               AS account_name,
    a.type               AS account_type,
    jl.scope_id,
    jl.cost_center_id,
    SUM(jl.debit)        AS total_debit,
    SUM(jl.credit)       AS total_credit,
    SUM(jl.debit) - SUM(jl.credit) AS net_balance
FROM journal_lines jl
JOIN journal_entries je ON je.id = jl.journal_entry_id
JOIN accounts a         ON a.id  = jl.account_id
WHERE je.status = 'POSTED'   -- only posted entries affect balances
GROUP BY
    jl.account_id, a.code, a.name, a.type,
    jl.scope_id, jl.cost_center_id;

COMMENT ON VIEW account_balances IS 'Computed balances from posted journal lines. Never from mutable fields. Rule 10.';

-- ============================================================
-- 8. ENTRY NUMBER SEQUENCE — one per fiscal year
-- Format: JE-YYYY-NNNNNN
-- ============================================================
CREATE SEQUENCE journal_entry_seq START 1;

-- ============================================================
-- 9. SEED: Fiscal Year 2026
-- ============================================================
INSERT INTO fiscal_years (name, start_date, end_date, status)
VALUES ('FY2026', '2026-01-01', '2026-12-31', 'OPEN');

-- Seed 12 monthly periods for FY2026
INSERT INTO accounting_periods (fiscal_year_id, name, period_number, start_date, end_date, status)
SELECT
    fy.id,
    TO_CHAR(generate_series, 'Month YYYY'),
    EXTRACT(MONTH FROM generate_series)::INT,
    DATE_TRUNC('month', generate_series)::DATE,
    (DATE_TRUNC('month', generate_series) + INTERVAL '1 month - 1 day')::DATE,
    'OPEN'
FROM
    fiscal_years fy,
    GENERATE_SERIES('2026-01-01'::DATE, '2026-12-01'::DATE, '1 month'::INTERVAL) AS generate_series
WHERE fy.name = 'FY2026';

-- ============================================================
-- 10. SEED: Chart of Accounts — Roadmap §14 example structure
-- ============================================================

-- === ROOT ACCOUNTS (non-postable headers) ===
INSERT INTO accounts (code, name, type, is_postable, sort_order) VALUES
    ('1000', 'الأصول',              'ASSET',     FALSE, 10),
    ('2000', 'الالتزامات',          'LIABILITY', FALSE, 20),
    ('3000', 'حقوق الملكية',        'EQUITY',    FALSE, 30),
    ('4000', 'الإيرادات',           'REVENUE',   FALSE, 40),
    ('5000', 'المصروفات',           'EXPENSE',   FALSE, 50);

-- === ASSET CHILDREN ===
INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '1100', 'النقدية والبنوك', 'ASSET', id, FALSE, 11
FROM accounts WHERE code = '1000';

INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '1200', 'المخزون',         'ASSET', id, FALSE, 12
FROM accounts WHERE code = '1000';

INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '1300', 'ذمم مدينة',       'ASSET', id, FALSE, 13
FROM accounts WHERE code = '1000';

-- === CASH ACCOUNTS (postable) ===
INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '1101', 'صندوق الإدارة الرئيسي', 'ASSET', id, TRUE, 111
FROM accounts WHERE code = '1100';

INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '1102', 'صندوق مصنع أ',          'ASSET', id, TRUE, 112
FROM accounts WHERE code = '1100';

INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '1103', 'صندوق مصنع ب',          'ASSET', id, TRUE, 113
FROM accounts WHERE code = '1100';

-- === LIABILITY CHILDREN ===
INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '2100', 'موردون',           'LIABILITY', id, TRUE, 21
FROM accounts WHERE code = '2000';

INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '2200', 'التزامات أخرى',   'LIABILITY', id, TRUE, 22
FROM accounts WHERE code = '2000';

-- === EXPENSE CHILDREN ===
INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '5100', 'مصروفات الإدارة', 'EXPENSE',   id, TRUE, 51
FROM accounts WHERE code = '5000';

INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '5200', 'مصروفات المصانع', 'EXPENSE',   id, FALSE, 52
FROM accounts WHERE code = '5000';

INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '5201', 'مصروفات مصنع أ',  'EXPENSE',   id, TRUE, 521
FROM accounts WHERE code = '5200';

INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '5202', 'مصروفات مصنع ب',  'EXPENSE',   id, TRUE, 522
FROM accounts WHERE code = '5200';

-- === SEED: Administration Cost Center ===
INSERT INTO cost_centers (scope_id, code, name)
SELECT id, 'CC-ADM', 'الإدارة العامة'
FROM organization_scopes WHERE code = 'ADM';
