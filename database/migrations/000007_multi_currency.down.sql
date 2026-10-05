-- ============================================================
-- Migration 000007 ROLLBACK: Multi-Currency Support
-- ============================================================

-- 8. حذف Seed أسعار الصرف
DELETE FROM exchange_rates WHERE from_currency = 'USD' AND to_currency = 'IQD';

-- 7. حذف صندوق الدولار
DELETE FROM cashboxes WHERE currency = 'USD';

-- 6. حذف الحسابات المضافة
DELETE FROM accounts WHERE code = '6100';
DELETE FROM accounts WHERE code = '6000';
DELETE FROM accounts WHERE code = '1110';

-- 5. Rollback cash_transfers
ALTER TABLE cash_transfers
    DROP COLUMN IF EXISTS base_amount,
    DROP COLUMN IF EXISTS source_exchange_rate,
    DROP COLUMN IF EXISTS dest_exchange_rate;

-- 4. Rollback cash_transactions
ALTER TABLE cash_transactions
    DROP COLUMN IF EXISTS base_amount,
    DROP COLUMN IF EXISTS exchange_rate;

-- 3. Rollback journal_lines
ALTER TABLE journal_lines
    DROP COLUMN IF EXISTS foreign_currency,
    DROP COLUMN IF EXISTS foreign_debit,
    DROP COLUMN IF EXISTS foreign_credit,
    DROP COLUMN IF EXISTS exchange_rate;

-- 2. حذف جدول أسعار الصرف
DROP TABLE IF EXISTS exchange_rates;

-- 1. إعادة العملة الافتراضية إلى SAR
ALTER TABLE financial_requests ALTER COLUMN currency SET DEFAULT 'SAR';
ALTER TABLE cashboxes          ALTER COLUMN currency SET DEFAULT 'SAR';
ALTER TABLE cash_transactions  ALTER COLUMN currency SET DEFAULT 'SAR';
ALTER TABLE cash_transfers     ALTER COLUMN currency SET DEFAULT 'SAR';