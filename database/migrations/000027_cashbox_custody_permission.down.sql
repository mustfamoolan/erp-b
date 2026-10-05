DELETE FROM account_mappings WHERE key = 'EMPLOYEE_CUSTODY_ACCOUNT';
DELETE FROM accounts WHERE code = '1120';
DELETE FROM permissions WHERE code = 'cashbox.custody.manage';
