-- Phase 4.5: COA Hardening and Master Data
-- 1. Create account_mappings
CREATE TABLE account_mappings (
    key          VARCHAR(100) PRIMARY KEY,
    account_id   UUID         NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    description  TEXT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 2. Create request_types
CREATE TABLE request_types (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(50)  NOT NULL UNIQUE,
    name        VARCHAR(150) NOT NULL,
    description TEXT,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    sort_order  INT          NOT NULL DEFAULT 0,
    created_by  UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 3. Create expense_categories
CREATE TABLE expense_categories (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(50)  NOT NULL UNIQUE,
    name        VARCHAR(150) NOT NULL,
    parent_id   UUID         REFERENCES expense_categories(id) ON DELETE RESTRICT,
    account_id  UUID         NOT NULL REFERENCES accounts(id)  ON DELETE RESTRICT,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    sort_order  INT          NOT NULL DEFAULT 0,
    created_by  UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 4. Seed New Accounts
-- Move US Dollar Cashbox from 1110 to 1104 to free up 1110 for Factory Cashboxes
UPDATE accounts SET code = '1104', sort_order = 114 WHERE code = '1110';

-- 1110 صناديق المعامل (Header, parent = 1100)
INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '1110', 'صناديق المعامل', 'ASSET', id, FALSE, 115
FROM accounts WHERE code = '1100';

-- 3100 أرصدة افتتاحية (Postable, parent = 3000)
INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '3100', 'أرصدة افتتاحية', 'EQUITY', id, TRUE, 31
FROM accounts WHERE code = '3000';

-- 5210..5260 (Postable, parent = 5200)
INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '5210', 'مكائن', 'EXPENSE', id, TRUE, 5210 FROM accounts WHERE code = '5200'
UNION ALL
SELECT '5220', 'معدات صغيرة', 'EXPENSE', id, TRUE, 5220 FROM accounts WHERE code = '5200'
UNION ALL
SELECT '5230', 'تنصيب مكائن', 'EXPENSE', id, TRUE, 5230 FROM accounts WHERE code = '5200'
UNION ALL
SELECT '5240', 'أثاث', 'EXPENSE', id, TRUE, 5240 FROM accounts WHERE code = '5200'
UNION ALL
SELECT '5250', 'مواد أولية محلية', 'EXPENSE', id, TRUE, 5250 FROM accounts WHERE code = '5200'
UNION ALL
SELECT '5260', 'مواد أولية مستوردة', 'EXPENSE', id, TRUE, 5260 FROM accounts WHERE code = '5200';

-- 5. Seed account_mappings
INSERT INTO account_mappings (key, account_id, description)
SELECT 'MAIN_CASHBOX', id, 'حساب الصندوق الرئيسي للإدارة' FROM accounts WHERE code = '1101'
UNION ALL
SELECT 'FACTORY_CASHBOX_PARENT', id, 'الحساب الأب لصناديق المعامل' FROM accounts WHERE code = '1110'
UNION ALL
SELECT 'OPENING_BALANCE_EQUITY', id, 'حساب الأرصدة الافتتاحية' FROM accounts WHERE code = '3100'
UNION ALL
SELECT 'EXPENSE_CATEGORY_PARENT', id, 'الحساب الأب لتصنيفات الصرف' FROM accounts WHERE code = '5200';

-- 6. Seed request_types
INSERT INTO request_types (code, name, description, sort_order)
VALUES 
    ('ADVANCE', 'سلفة', 'سلفة مالية تُسوى لاحقاً', 10),
    ('FUNDING', 'تمويل', 'تمويل لتغطية مصروفات', 20);

-- 7. Seed expense_categories
INSERT INTO expense_categories (code, name, account_id, sort_order)
SELECT 'MACHINE', 'مكائن', id, 10 FROM accounts WHERE code = '5210'
UNION ALL
SELECT 'SMALL_EQUIP', 'معدات صغيرة', id, 20 FROM accounts WHERE code = '5220'
UNION ALL
SELECT 'MACHINE_INSTALL', 'تنصيب مكائن', id, 30 FROM accounts WHERE code = '5230'
UNION ALL
SELECT 'FURNITURE', 'أثاث', id, 40 FROM accounts WHERE code = '5240'
UNION ALL
SELECT 'RAW_LOCAL', 'مواد أولية محلية', id, 50 FROM accounts WHERE code = '5250'
UNION ALL
SELECT 'RAW_IMPORT', 'مواد أولية مستوردة', id, 60 FROM accounts WHERE code = '5260';

-- 8. Corrective Migration for Cashboxes
-- Fix cashboxes pointing to header accounts or shared accounts by giving them their own account under 1110
DO $$
DECLARE
    cb RECORD;
    new_acc_id UUID;
    acc_seq INT := 1;
    new_code VARCHAR(20);
BEGIN
    FOR cb IN 
        SELECT c.id as cashbox_id, c.name, c.scope_id 
        FROM cashboxes c
        JOIN accounts a ON c.account_id = a.id
        WHERE a.is_postable = FALSE 
           OR a.code NOT LIKE '1110-%' 
           AND a.code != '1101' -- Skip main cashbox
    LOOP
        new_code := '1110-' || LPAD(acc_seq::TEXT, 4, '0');
        
        -- Create new account
        INSERT INTO accounts (code, name, type, parent_id, is_postable, scope_id, sort_order)
        SELECT new_code, 'صندوق ' || cb.name, 'ASSET', id, TRUE, cb.scope_id, 11100 + acc_seq
        FROM accounts WHERE code = '1110'
        RETURNING id INTO new_acc_id;

        -- Update cashbox
        UPDATE cashboxes SET account_id = new_acc_id WHERE id = cb.cashbox_id;
        
        acc_seq := acc_seq + 1;
    END LOOP;
END $$;

-- 9. DB Constraint: UNIQUE (account_id) on cashboxes
ALTER TABLE cashboxes ADD CONSTRAINT cashboxes_account_id_unique UNIQUE (account_id);
