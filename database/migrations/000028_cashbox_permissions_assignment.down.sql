DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'cashbox.custody.manage')
  AND role_id IN (SELECT id FROM roles WHERE name IN ('FACTORY_ACCOUNTANT', 'ACCOUNTANT', 'ADMINISTRATOR', 'SUPER_ADMIN'));
