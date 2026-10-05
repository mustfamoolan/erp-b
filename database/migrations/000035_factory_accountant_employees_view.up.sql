-- Grant employees.view to FACTORY_ACCOUNTANT so they can select employees for cashbox custody (withdraw/deposit)
INSERT INTO role_permissions (id, role_id, permission_id, created_at)
SELECT
    gen_random_uuid(),
    r.id,
    p.id,
    NOW()
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'FACTORY_ACCOUNTANT'
  AND p.code = 'employees.view'
ON CONFLICT (role_id, permission_id) DO NOTHING;
