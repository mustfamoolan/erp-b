-- Migration 000033: Request Lifecycle States, Delivery Voucher, and Granular Permissions
-- Adds RETURNED_FOR_REVISION, DISBURSED, DELIVERED, REJECTED to status check
-- Adds signed_voucher_url, disbursed_at, delivered_at to financial_requests
-- Seeds granular request permissions (return_revision, disburse, deliver)

-- 1. Update check constraint on financial_requests
ALTER TABLE financial_requests DROP CONSTRAINT IF EXISTS financial_requests_status_check;
ALTER TABLE financial_requests ADD CONSTRAINT financial_requests_status_check CHECK (
    status IN (
        'DRAFT',
        'SUBMITTED',
        'UNDER_REVIEW',
        'REVIEW_APPROVED',
        'ACCOUNTANT_APPROVED',
        'PAYMENT_PENDING',
        'DISBURSED',
        'FACTORY_RECEIPT_PENDING',
        'DELIVERED',
        'RECEIVED',
        'COMPLETED',
        'RETURNED_FOR_REVISION',
        'REVIEW_REJECTED',
        'ACCOUNTANT_REJECTED',
        'REJECTED',
        'CANCELLED'
    )
);

-- 2. Add columns to financial_requests
ALTER TABLE financial_requests ADD COLUMN IF NOT EXISTS signed_voucher_url TEXT;
ALTER TABLE financial_requests ADD COLUMN IF NOT EXISTS disbursed_at TIMESTAMPTZ;
ALTER TABLE financial_requests ADD COLUMN IF NOT EXISTS delivered_at TIMESTAMPTZ;

-- 3. Upsert granular permissions
INSERT INTO permissions (code, display_name, description, grp) VALUES
    ('request.return_revision', 'طلب إعادة نظر', 'إعادة الطلب إلى المعمل للتعديل مع كتابة سبب إلزامي', 'request'),
    ('request.disburse',        'صرف السلفة وتوليد الوصل', 'تأكيد صرف السلفة وتوليد وصل التسليم الرسمي للطباعة', 'request'),
    ('request.deliver',         'تسليم السلفة وإرفاق الوصل', 'تأكيد تسليم السلفة ورفع الوصل الموقع وإجراء الأثر المالي لصندوق المعمل', 'request')
ON CONFLICT (code) DO UPDATE
    SET display_name = EXCLUDED.display_name,
        description  = EXCLUDED.description,
        grp          = EXCLUDED.grp;

-- 4. Assign permissions to roles
-- Super Admin and Administrator get all
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('SUPER_ADMIN', 'ADMINISTRATOR')
  AND p.code IN ('request.return_revision', 'request.disburse', 'request.deliver')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Auditor gets return_revision
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'request.return_revision'
WHERE r.name = 'AUDITOR'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Accountant gets return_revision, disburse, deliver
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'ACCOUNTANT'
  AND p.code IN ('request.return_revision', 'request.disburse', 'request.deliver')
ON CONFLICT (role_id, permission_id) DO NOTHING;
