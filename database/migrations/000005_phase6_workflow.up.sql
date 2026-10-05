-- Phase 6: Financial Request Workflow
-- Roadmap §31-33

-- ============================================================
-- 1. FINANCIAL REQUESTS — Roadmap §31
-- ============================================================
CREATE SEQUENCE financial_request_seq START 1;

CREATE TABLE financial_requests (
    id              UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    document_number VARCHAR(50)    NOT NULL,
    barcode         VARCHAR(100),
    scope_id        UUID           NOT NULL REFERENCES organization_scopes(id) ON DELETE RESTRICT,
    factory_id      UUID           NOT NULL REFERENCES organization_scopes(id) ON DELETE RESTRICT,
    requested_by    UUID           NOT NULL REFERENCES users(id)               ON DELETE RESTRICT,
    type            VARCHAR(20)    NOT NULL DEFAULT 'FINANCIAL'
                                   CHECK (type IN ('FINANCIAL', 'MATERIAL')),
    request_date    DATE           NOT NULL,
    required_date   DATE,
    purpose         VARCHAR(500)   NOT NULL,
    description     TEXT,
    total_amount    NUMERIC(18,4)  NOT NULL DEFAULT 0,
    currency        VARCHAR(10)    NOT NULL DEFAULT 'SAR',
    status          VARCHAR(40)    NOT NULL DEFAULT 'DRAFT'
                    CHECK (status IN (
                        'DRAFT','SUBMITTED','UNDER_REVIEW','REVIEW_APPROVED',
                        'ACCOUNTANT_APPROVED','PAYMENT_PENDING','FACTORY_RECEIPT_PENDING',
                        'RECEIVED','COMPLETED',
                        'REVIEW_REJECTED','ACCOUNTANT_REJECTED','CANCELLED'
                    )),
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT financial_requests_number_unique  UNIQUE (document_number),
    CONSTRAINT financial_requests_barcode_unique UNIQUE (barcode),
    -- The request must belong to a factory scope (not administration)
    CONSTRAINT financial_requests_scope_check CHECK (scope_id = factory_id)
);

CREATE INDEX idx_financial_requests_scope_id   ON financial_requests(scope_id);
CREATE INDEX idx_financial_requests_factory_id ON financial_requests(factory_id);
CREATE INDEX idx_financial_requests_status     ON financial_requests(status);
CREATE INDEX idx_financial_requests_date       ON financial_requests(request_date);
CREATE INDEX idx_financial_requests_barcode    ON financial_requests(barcode);

COMMENT ON TABLE financial_requests IS 'Financial/material requests from factories to administration. Roadmap §31.';
COMMENT ON COLUMN financial_requests.document_number IS 'Format: FIN-YYYY-NNNNNN (Roadmap §33)';
COMMENT ON COLUMN financial_requests.barcode IS 'Machine-readable barcode for scanning (Roadmap §33). Different from item barcode.';

-- ============================================================
-- 2. REQUEST ITEMS — Roadmap §31
-- ============================================================
CREATE TABLE request_items (
    id                   UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id           UUID          NOT NULL REFERENCES financial_requests(id) ON DELETE CASCADE,
    variant_id           UUID          REFERENCES item_variants(id)               ON DELETE SET NULL,
    description          VARCHAR(500)  NOT NULL,
    quantity             NUMERIC(18,4) NOT NULL CHECK (quantity > 0),
    unit_id              UUID          REFERENCES units_of_measure(id),
    estimated_unit_price NUMERIC(18,4) NOT NULL DEFAULT 0,
    estimated_total      NUMERIC(18,4) NOT NULL DEFAULT 0,
    notes                TEXT
);

CREATE INDEX idx_request_items_request_id ON request_items(request_id);

-- ============================================================
-- 3. REQUEST HISTORY — immutable audit trail for every transition
-- Rule 17: Every workflow transition must be audited.
-- ============================================================
CREATE TABLE request_history (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id   UUID        NOT NULL REFERENCES financial_requests(id) ON DELETE CASCADE,
    from_status  VARCHAR(40),
    to_status    VARCHAR(40) NOT NULL,
    action       VARCHAR(50) NOT NULL,
    performed_by UUID        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    notes        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_request_history_request_id ON request_history(request_id);
CREATE INDEX idx_request_history_date       ON request_history(created_at);

COMMENT ON TABLE request_history IS 'Immutable audit trail for every request status transition. Rule 17.';
