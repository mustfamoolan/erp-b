-- Phase 4+5: Item Master Data + Warehouse Foundation
-- Roadmap §25-30

-- ============================================================
-- 1. ITEM CATEGORIES
-- ============================================================
CREATE TABLE item_categories (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(150) NOT NULL,
    parent_id  UUID        REFERENCES item_categories(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_item_categories_parent ON item_categories(parent_id);

-- ============================================================
-- 2. UNITS OF MEASURE
-- ============================================================
CREATE TABLE units_of_measure (
    id         UUID       PRIMARY KEY DEFAULT gen_random_uuid(),
    code       VARCHAR(20) NOT NULL,
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT units_code_unique UNIQUE (code)
);

-- Seed basic units
INSERT INTO units_of_measure (code, name) VALUES
    ('PCS',  'قطعة'),
    ('KG',   'كيلوغرام'),
    ('G',    'غرام'),
    ('TON',  'طن'),
    ('M',    'متر'),
    ('M2',   'متر مربع'),
    ('M3',   'متر مكعب'),
    ('L',    'لتر'),
    ('BOX',  'صندوق'),
    ('ROLL', 'لفة'),
    ('SET',  'مجموعة');

-- ============================================================
-- 3. ITEMS (base records)
-- ============================================================
CREATE TABLE items (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id  UUID        NOT NULL REFERENCES item_categories(id) ON DELETE RESTRICT,
    code         VARCHAR(50) NOT NULL,
    name         VARCHAR(200) NOT NULL,
    description  TEXT,
    base_unit_id UUID        NOT NULL REFERENCES units_of_measure(id) ON DELETE RESTRICT,
    is_active    BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT items_code_unique UNIQUE (code)
);

CREATE INDEX idx_items_category_id ON items(category_id);

-- ============================================================
-- 4. ITEM VARIANTS — Roadmap §25
-- Each variant has its own SKU and Barcode.
-- SKU = business identifier, Barcode = machine-readable (§26).
-- They are NOT the same thing.
-- ============================================================
CREATE TABLE item_variants (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id    UUID        NOT NULL REFERENCES items(id) ON DELETE RESTRICT,
    name       VARCHAR(200) NOT NULL,
    sku        VARCHAR(100) NOT NULL,    -- business identifier — §26
    barcode    VARCHAR(100),             -- machine-readable — §26
    unit_id    UUID        NOT NULL REFERENCES units_of_measure(id) ON DELETE RESTRICT,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT item_variants_sku_unique     UNIQUE (sku),
    CONSTRAINT item_variants_barcode_unique UNIQUE (barcode)
);

CREATE INDEX idx_item_variants_item_id ON item_variants(item_id);
CREATE INDEX idx_item_variants_sku     ON item_variants(sku);
CREATE INDEX idx_item_variants_barcode ON item_variants(barcode);

-- ============================================================
-- 5. ITEM VARIANT ATTRIBUTES
-- ============================================================
CREATE TABLE item_variant_attributes (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id UUID        NOT NULL REFERENCES item_variants(id) ON DELETE CASCADE,
    attr_key   VARCHAR(100) NOT NULL,
    attr_value VARCHAR(200) NOT NULL
);

CREATE INDEX idx_variant_attributes_variant ON item_variant_attributes(variant_id);

-- ============================================================
-- 6. WAREHOUSES — Roadmap §27
-- Scope ownership is mandatory — Rule 5.
-- ============================================================
CREATE TABLE warehouses (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    scope_id   UUID        NOT NULL REFERENCES organization_scopes(id) ON DELETE RESTRICT,
    code       VARCHAR(50) NOT NULL,
    name       VARCHAR(150) NOT NULL,
    type       VARCHAR(30) NOT NULL DEFAULT 'GENERAL'
               CHECK (type IN ('RAW_MATERIAL', 'FINISHED_GOODS', 'CONSUMABLES', 'GENERAL')),
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT warehouses_code_unique UNIQUE (code)
);

CREATE INDEX idx_warehouses_scope_id ON warehouses(scope_id);

COMMENT ON TABLE warehouses IS 'Warehouses owned by a scope. Scope isolation enforced by backend. Rule 5.';

-- ============================================================
-- 7. STOCK MOVEMENTS — Roadmap §28
-- Stock quantity is reconstructable from movements.
-- Rule 12: Every inventory movement must have a traceable record.
-- Rule 16: Corrections use ADJUSTMENT movements.
-- NEVER overwrite stock directly.
-- ============================================================
CREATE TABLE stock_movements (
    id             UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    warehouse_id   UUID           NOT NULL REFERENCES warehouses(id)     ON DELETE RESTRICT,
    variant_id     UUID           NOT NULL REFERENCES item_variants(id)  ON DELETE RESTRICT,
    quantity       NUMERIC(18,4)  NOT NULL,  -- positive = IN, negative = OUT
    unit_id        UUID           NOT NULL REFERENCES units_of_measure(id),
    movement_type  VARCHAR(30)    NOT NULL
                   CHECK (movement_type IN (
                       'OPENING_BALANCE','PURCHASE_RECEIPT','TRANSFER_IN','TRANSFER_OUT',
                       'CONSUMPTION','PRODUCTION','ADJUSTMENT','RETURN','ISSUE','RECEIPT'
                   )),
    reference_type VARCHAR(50),
    reference_id   UUID,
    performed_by   UUID           NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    notes          TEXT,
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stock_movements_warehouse  ON stock_movements(warehouse_id);
CREATE INDEX idx_stock_movements_variant    ON stock_movements(variant_id);
CREATE INDEX idx_stock_movements_type       ON stock_movements(movement_type);
CREATE INDEX idx_stock_movements_reference  ON stock_movements(reference_type, reference_id);
CREATE INDEX idx_stock_movements_date       ON stock_movements(created_at);

COMMENT ON TABLE stock_movements IS 'Immutable stock movement log. Stock reconstructed from here. Rule 12, 16.';

-- ============================================================
-- 8. STOCK BALANCE VIEW — Roadmap §28
-- Never from a stored quantity field — Rule 12.
-- ============================================================
CREATE VIEW stock_balances AS
SELECT
    sm.warehouse_id,
    w.name         AS warehouse_name,
    w.scope_id,
    sm.variant_id,
    iv.name        AS variant_name,
    iv.sku,
    sm.unit_id,
    SUM(sm.quantity) AS current_quantity
FROM stock_movements sm
JOIN warehouses    w  ON w.id  = sm.warehouse_id
JOIN item_variants iv ON iv.id = sm.variant_id
GROUP BY sm.warehouse_id, w.name, w.scope_id, sm.variant_id, iv.name, iv.sku, sm.unit_id;

COMMENT ON VIEW stock_balances IS 'Real-time stock computed from movements. Never from stored fields. Rule 12.';
