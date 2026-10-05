DELETE FROM role_permissions
WHERE role_id IN (SELECT id FROM roles WHERE name = 'FACTORY_ACCOUNTANT')
  AND permission_id IN (SELECT id FROM permissions WHERE code = 'employees.view');
