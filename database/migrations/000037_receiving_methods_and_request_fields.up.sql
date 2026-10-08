CREATE TABLE IF NOT EXISTS receiving_methods (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code       VARCHAR(50) NOT NULL UNIQUE,
    name       VARCHAR(150) NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO receiving_methods (code, name, sort_order) VALUES
    ('CASH', 'نقد', 1),
    ('TRANSFER', 'حوالة', 2),
    ('CHECK', 'صك', 3),
    ('CARD', 'بطاقة مصرفية', 4),
    ('ZAIN_CASH', 'زين كاش', 5),
    ('INSTALLMENTS', 'دفعات', 6)
ON CONFLICT (code) DO NOTHING;

ALTER TABLE units_of_measure ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE units_of_measure ADD COLUMN IF NOT EXISTS sort_order INT NOT NULL DEFAULT 0;

ALTER TABLE financial_requests ADD COLUMN IF NOT EXISTS advance_sequence_number INT;
ALTER TABLE financial_requests ADD COLUMN IF NOT EXISTS receiving_location VARCHAR(255);
ALTER TABLE financial_requests ADD COLUMN IF NOT EXISTS receiver_phone VARCHAR(50);

ALTER TABLE request_items ADD COLUMN IF NOT EXISTS receipt_number VARCHAR(100);
