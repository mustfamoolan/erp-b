-- ============================================================
-- Migration 000007: Multi-Currency Support
-- IQD (دينار عراقي) كعملة أساسية + USD مع سعر صرف متغير
--
-- القواعد المُطبقة:
-- Rule #12: NUMERIC(18,4) — لا floating-point في المبالغ المالية
-- Rule #13: الرصيد يُحسب من journal_lines — لا يُخزن مباشرة
-- Rule #14: SUM(debit) = SUM(credit) بالدينار العراقي IQD
-- Rule #15: exchange_rates لا تُحذف — تاريخ محفوظ
-- Rule #17: تحويل النقدية يجب أن يكون ذرياً
-- ============================================================

-- ============================================================
-- 1. تغيير العملة الافتراضية من SAR إلى IQD في جميع الجداول
-- ============================================================
ALTER TABLE cashboxes         ALTER COLUMN currency SET DEFAULT 'IQD';
ALTER TABLE cash_transactions ALTER COLUMN currency SET DEFAULT 'IQD';
ALTER TABLE cash_transfers    ALTER COLUMN currency SET DEFAULT 'IQD';
ALTER TABLE financial_requests ALTER COLUMN currency SET DEFAULT 'IQD';

-- تحديث السجلات الموجودة التي تحمل SAR
UPDATE cashboxes          SET currency = 'IQD' WHERE currency = 'SAR';
UPDATE cash_transactions  SET currency = 'IQD' WHERE currency = 'SAR';
UPDATE cash_transfers     SET currency = 'IQD' WHERE currency = 'SAR';
UPDATE financial_requests SET currency = 'IQD' WHERE currency = 'SAR';

-- ============================================================
-- 2. جدول أسعار الصرف التاريخية
-- Rule #12: NUMERIC(18,6) للدقة العالية في أسعار الصرف
-- Rule #15: لا حذف — كل سعر تاريخي محفوظ للتدقيق
-- ============================================================
CREATE TABLE exchange_rates (
    id              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    from_currency   VARCHAR(10)   NOT NULL,
    to_currency     VARCHAR(10)   NOT NULL,
    rate            NUMERIC(18,6) NOT NULL CHECK (rate > 0),
    effective_date  DATE          NOT NULL DEFAULT CURRENT_DATE,
    set_by          UUID          NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT exchange_rates_pair_date UNIQUE (from_currency, to_currency, effective_date)
);

CREATE INDEX idx_exchange_rates_pair ON exchange_rates(from_currency, to_currency);
CREATE INDEX idx_exchange_rates_date ON exchange_rates(effective_date DESC);

COMMENT ON TABLE  exchange_rates                IS 'تاريخ أسعار الصرف. سجل لا يُحذف — Rule 15.';
COMMENT ON COLUMN exchange_rates.rate           IS 'كم IQD يساوي 1 وحدة from_currency. مثال: 1310.000000 = 1 USD = 1310 IQD';
COMMENT ON COLUMN exchange_rates.from_currency  IS 'العملة المصدر (مثال: USD)';
COMMENT ON COLUMN exchange_rates.to_currency    IS 'العملة الهدف — دائماً IQD في هذا النظام';
COMMENT ON COLUMN exchange_rates.effective_date IS 'تاريخ سريان هذا السعر';
COMMENT ON COLUMN exchange_rates.set_by         IS 'المستخدم الذي حدّث السعر — للتدقيق Rule 21';

-- ============================================================
-- 3. إضافة حقول العملة الأجنبية في journal_lines
-- قاعدة أساسية:
--   debit/credit  → دائماً بالدينار IQD (للتوازن المحاسبي Rule 14)
--   foreign_*     → القيمة الأجنبية الأصلية (للمرجعية والتقارير فقط)
-- ============================================================
ALTER TABLE journal_lines
    ADD COLUMN foreign_currency VARCHAR(10)   DEFAULT NULL,
    ADD COLUMN foreign_debit    NUMERIC(18,4) NOT NULL DEFAULT 0,
    ADD COLUMN foreign_credit   NUMERIC(18,4) NOT NULL DEFAULT 0,
    ADD COLUMN exchange_rate    NUMERIC(18,6) DEFAULT NULL;

COMMENT ON COLUMN journal_lines.debit           IS 'دائماً بالدينار العراقي IQD — للتوازن المحاسبي Rule 14';
COMMENT ON COLUMN journal_lines.credit          IS 'دائماً بالدينار العراقي IQD — للتوازن المحاسبي Rule 14';
COMMENT ON COLUMN journal_lines.foreign_currency IS 'العملة الأجنبية إن وجدت (مثال: USD). NULL = عملية بالدينار فقط';
COMMENT ON COLUMN journal_lines.foreign_debit   IS 'المبلغ الأجنبي الأصلي — للمرجعية فقط، لا للتوازن';
COMMENT ON COLUMN journal_lines.foreign_credit  IS 'المبلغ الأجنبي الأصلي — للمرجعية فقط، لا للتوازن';
COMMENT ON COLUMN journal_lines.exchange_rate   IS 'سعر الصرف المستخدم وقت تسجيل القيد';

-- ============================================================
-- 4. إضافة BaseAmount في cash_transactions
-- amount = بعملة الصندوق الأصلية (IQD أو USD)
-- base_amount = المبلغ المكافئ بالدينار IQD دائماً
-- Rule #12: كلاهما NUMERIC(18,4)
-- ============================================================
ALTER TABLE cash_transactions
    ADD COLUMN base_amount   NUMERIC(18,4) DEFAULT NULL,
    ADD COLUMN exchange_rate NUMERIC(18,6) DEFAULT NULL;

COMMENT ON COLUMN cash_transactions.amount       IS 'المبلغ بعملة الصندوق الأصلية (IQD أو USD)';
COMMENT ON COLUMN cash_transactions.base_amount  IS 'المبلغ المكافئ بالدينار العراقي IQD دائماً';
COMMENT ON COLUMN cash_transactions.exchange_rate IS 'سعر الصرف المستخدم — NULL إذا كانت العملة IQD';

-- ============================================================
-- 5. إضافة BaseAmount في cash_transfers
-- ============================================================
ALTER TABLE cash_transfers
    ADD COLUMN base_amount          NUMERIC(18,4) DEFAULT NULL,
    ADD COLUMN source_exchange_rate NUMERIC(18,6) DEFAULT NULL,
    ADD COLUMN dest_exchange_rate   NUMERIC(18,6) DEFAULT NULL;

COMMENT ON COLUMN cash_transfers.base_amount          IS 'المبلغ المحوّل بالدينار IQD دائماً';
COMMENT ON COLUMN cash_transfers.source_exchange_rate IS 'سعر الصرف للصندوق المصدر إن كان بعملة أجنبية';
COMMENT ON COLUMN cash_transfers.dest_exchange_rate   IS 'سعر الصرف للصندوق الوجهة إن كان بعملة أجنبية';

-- ============================================================
-- 6. إضافة حسابات محاسبية جديدة في الشجرة
-- صندوق الدولار + حساب فروق أسعار الصرف
-- ============================================================

-- صندوق الدولار الأمريكي (تحت مجموعة النقدية والبنوك 1100)
INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '1110', 'صندوق الدولار الأمريكي', 'ASSET', id, TRUE, 115
FROM accounts WHERE code = '1100';

-- مجموعة الإيرادات والمكاسب الأخرى (header — غير قابلة للترحيل)
INSERT INTO accounts (code, name, type, is_postable, sort_order)
VALUES ('6000', 'الإيرادات والمكاسب الأخرى', 'REVENUE', FALSE, 60);

-- حساب مكاسب وخسائر فروق أسعار الصرف (قابل للترحيل)
INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '6100', 'مكاسب وخسائر فروق أسعار الصرف', 'REVENUE', id, TRUE, 610
FROM accounts WHERE code = '6000';

-- ============================================================
-- 7. Seed: صندوق الدولار الأمريكي للإدارة
-- ============================================================
INSERT INTO cashboxes (scope_id, name, account_id, currency, status)
SELECT
    s.id,
    'صندوق الدولار الأمريكي',
    a.id,
    'USD',
    'ACTIVE'
FROM organization_scopes s
CROSS JOIN accounts a
WHERE s.code = 'ADM'
  AND a.code = '1110';

-- ============================================================
-- 8. Seed: سعر الصرف الأولي — 1 USD = 1310 IQD
-- ============================================================
INSERT INTO exchange_rates (from_currency, to_currency, rate, effective_date, set_by)
SELECT
    'USD',
    'IQD',
    1310.000000,
    CURRENT_DATE,
    u.id
FROM users u
WHERE u.username = 'admin'
LIMIT 1;