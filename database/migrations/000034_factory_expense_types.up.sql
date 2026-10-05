-- Migration 000034: Factory Expense Types (أنواع مصاريف المعامل)
CREATE TABLE IF NOT EXISTS factory_expense_types (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(50) NOT NULL UNIQUE,
    name        VARCHAR(150) NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order  INT NOT NULL DEFAULT 0,
    created_by  UUID REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add reference column to financial_requests if not exists
ALTER TABLE financial_requests
    ADD COLUMN IF NOT EXISTS factory_expense_type_id UUID REFERENCES factory_expense_types(id) ON DELETE RESTRICT;

-- Seed initial standard factory expense types
INSERT INTO factory_expense_types (code, name, sort_order) VALUES
    ('SETUP', 'تأسيسي', 10),
    ('CONSTRUCTION', 'إنشائي', 20),
    ('PURCHASE', 'شراء', 30),
    ('OPERATIONAL', 'تشغيلي', 40)
ON CONFLICT (code) DO NOTHING;
