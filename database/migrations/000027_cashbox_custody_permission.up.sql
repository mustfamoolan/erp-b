INSERT INTO permissions (code, display_name, description, grp, created_at)
VALUES 
    ('cashbox.custody.manage', 'إدارة عهد وسلف الموظفين (سحب/إيداع)', 'يسمح للمستخدم بسحب أو إيداع مبالغ في صندوق المعمل كسلف أو عهد للموظفين', 'cashbox', NOW())
ON CONFLICT (code) DO NOTHING;

INSERT INTO accounts (code, name, type, parent_id, is_postable, sort_order)
SELECT '1120', 'عهد سلف الموظفين', 'ASSET', id, TRUE, 120
FROM accounts WHERE code = '1100'
ON CONFLICT (code) DO NOTHING;

INSERT INTO account_mappings (key, account_id, description)
SELECT 'EMPLOYEE_CUSTODY_ACCOUNT', id, 'حساب عهد سلف الموظفين' FROM accounts WHERE code = '1120'
ON CONFLICT (key) DO NOTHING;
