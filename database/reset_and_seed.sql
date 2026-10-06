-- ==============================================================================
-- DATABASE RESET & CLEAN SEEDER
-- ==============================================================================

BEGIN;

-- 1. Truncate all transactions, ledger, and logs (Excluding views like account_balances, stock_balances)
TRUNCATE TABLE 
    request_history, 
    request_items, 
    request_attachments, 
    financial_requests,
    cash_transactions, 
    cash_transfers, 
    expenses,
    purchase_invoice_items, 
    purchase_invoices,
    journal_lines, 
    journal_entries, 
    stock_movements, 
    audit_logs,
    employees
CASCADE;

-- 2. Reset cashbox balances
UPDATE cashboxes SET target_opening_balance = 0, imprest_balance = 0;

-- 3. Clean users and roles
TRUNCATE TABLE user_roles, user_scope_access, users CASCADE;
TRUNCATE TABLE role_permissions, roles CASCADE;

-- 4. Re-create the 5 canonical roles with explicit Arabic names
INSERT INTO roles (id, name, display_name, description, is_system, created_at, updated_at) VALUES
('33ebfb2a-439a-4a84-a16b-0f8331f8f359', 'ADMINISTRATOR', 'المدير العام (كامل الصلاحيات)', 'له كل الصلاحيات المطلقة وإدارة النظام بالكامل والوصول لجميع المعامل والمناطق', TRUE, NOW(), NOW()),
('bbcd755c-5e45-4ec3-bb0f-ebc39b77e006', 'SUPER_ADMIN', 'المدير العام (كامل الصلاحيات)', 'له كل الصلاحيات المطلقة وإدارة النظام بالكامل والوصول لجميع المعامل والمناطق', TRUE, NOW(), NOW()),
('3cb0bc21-4e38-43e5-a06a-c12deabce1c1', 'FACTORY_ACCOUNTANT', 'محاسب معمل', 'يرفع طلب، كشف حركات صندوق، وعرض إيداع وسحب من حركات الصندوق', TRUE, NOW(), NOW()),
('b204a576-7b8a-4775-8e43-84399d914d21', 'AUDITOR', 'مدقق الإدارة', 'يعرض الطلبات التي ليست مدققة فقط، يوافق عليها أو يلغي أو يطلب إعادة نظر مع ذكر السبب', TRUE, NOW(), NOW()),
('9d8df3fc-d24e-4c41-8853-8d94e8ed4d70', 'ACCOUNTANT', 'محاسب الإدارة', 'يطبع A4، يعرض التي تخصه فقط (محاسب إدارة)، ويوافق أو يرفض أو يطلب إعادة نظر مع ذكر السبب', TRUE, NOW(), NOW()),
('4e91cf23-5e34-4512-8822-26cb443210aa', 'CENTRAL_CASHIER', 'محاسب المركز', 'يصرف المبلغ ويسلم المبلغ وإرفاق الملف وصل فقط A5', TRUE, NOW(), NOW());

-- 5. Seed Role Permissions

-- A. ADMINISTRATOR & SUPER_ADMIN -> ALL 64 PERMISSIONS
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('ADMINISTRATOR', 'SUPER_ADMIN');

-- B. FACTORY_ACCOUNTANT -> إنشاء وتقديم طلبات، كشف رصيد وسجل الصندوق، سحب وإيداع
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'request.create',
    'request.submit',
    'request.view',
    'request.cancel',
    'request.attach',
    'request.receive',
    'cashbox.view',
    'cashbox.custody.manage',
    'cashbox.opening_balance',
    'expenses.create',
    'purchases.create',
    'purchases.view',
    'employees.view',
    'master_data.view',
    'factories.view',
    'areas.view',
    'users.view'
)
WHERE r.name = 'FACTORY_ACCOUNTANT';

-- C. AUDITOR -> مدقق الإدارة: يعرض الطلبات غير المدققة، يوافق، يرفض، إعادة نظر مع ذكر السبب
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'request.view',
    'request.view_all',
    'request.review',
    'request.audit_approve',
    'request.reject',
    'request.return_revision',
    'users.view',
    'factories.view',
    'areas.view'
)
WHERE r.name = 'AUDITOR';

-- D. ACCOUNTANT -> محاسب الإدارة: يعرض الطلبات المدققة، مصادقة، طباعة A4، رفض، إعادة نظر
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'request.view',
    'request.view_all',
    'request.approve',
    'request.reject',
    'request.return_revision',
    'reports.view',
    'users.view',
    'factories.view',
    'areas.view'
)
WHERE r.name = 'ACCOUNTANT';

-- E. CENTRAL_CASHIER -> محاسب المركز: صرف، تسليم وإرفاق وصل A5
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'request.view',
    'request.view_all',
    'request.disburse',
    'request.pay',
    'request.deliver',
    'request.attach',
    'cashbox.view',
    'cashbox.transfer',
    'users.view',
    'factories.view',
    'areas.view'
)
WHERE r.name = 'CENTRAL_CASHIER';

-- 6. Insert the single Administrator User
-- Username: admin | Password: 12345678
INSERT INTO users (id, username, email, password_hash, full_name, status, created_at, updated_at)
VALUES (
    'a61be584-37bb-43c3-87a9-b72ea4b40a6f',
    'admin',
    'admin@m3aml.local',
    '$2a$12$PEjeX5Ju90tplRh2OOoGB.lq5X..yAaV4AUu8wdIhqnTkb3DxH99y',
    'المدير العام (System Administrator)',
    'ACTIVE',
    NOW(),
    NOW()
);

-- Assign ADMINISTRATOR and SUPER_ADMIN roles to admin
INSERT INTO user_roles (id, user_id, role_id, scope_id, created_at)
SELECT gen_random_uuid(), 'a61be584-37bb-43c3-87a9-b72ea4b40a6f', r.id, NULL, NOW()
FROM roles r
WHERE r.name IN ('ADMINISTRATOR', 'SUPER_ADMIN');

-- Grant all organization scopes to admin for universal access
INSERT INTO user_scope_access (id, user_id, scope_id, granted_at, granted_by)
SELECT gen_random_uuid(), 'a61be584-37bb-43c3-87a9-b72ea4b40a6f', id, NOW(), 'a61be584-37bb-43c3-87a9-b72ea4b40a6f'
FROM organization_scopes;

-- 7. Seed Master Data (Request Types, Expense Categories, Factory Expense Types, Settings, etc.)
INSERT INTO company_settings (id, company_name) 
VALUES ('00000000-0000-0000-0000-000000000000', 'معمل و شركة المقاولات')
ON CONFLICT (id) DO UPDATE SET company_name = EXCLUDED.company_name;

INSERT INTO request_types (code, name, description, sort_order)
VALUES 
    ('ADVANCE', 'سلفة', 'سلفة مالية تُسوى لاحقاً', 10),
    ('FUNDING', 'تمويل', 'تمويل لتغطية مصروفات', 20),
    ('IMPREST_FUNDING', 'طلب تعزيز رصيد المداورة', 'طلب لتعزيز رصيد المداورة للمعمل', 30)
ON CONFLICT (code) DO NOTHING;

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
SELECT 'RAW_IMPORT', 'مواد أولية مستوردة', id, 60 FROM accounts WHERE code = '5260'
ON CONFLICT (code) DO NOTHING;

INSERT INTO factory_expense_types (code, name, sort_order) VALUES
    ('SETUP', 'تأسيسي', 10),
    ('CONSTRUCTION', 'إنشائي', 20),
    ('PURCHASE', 'شراء', 30),
    ('OPERATIONAL', 'تشغيلي', 40)
ON CONFLICT (code) DO NOTHING;

INSERT INTO exchange_rates (from_currency, to_currency, rate, effective_date, set_by)
SELECT
    'USD',
    'IQD',
    1310.000000,
    CURRENT_DATE,
    'a61be584-37bb-43c3-87a9-b72ea4b40a6f'
ON CONFLICT DO NOTHING;

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

COMMIT;

