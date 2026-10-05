INSERT INTO role_permissions (id, role_id, permission_id, created_at)
SELECT
    gen_random_uuid(),
    r.id,
    p.id,
    NOW()
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('FACTORY_ACCOUNTANT', 'ACCOUNTANT', 'ADMINISTRATOR', 'SUPER_ADMIN')
  AND p.code = 'cashbox.custody.manage'
ON CONFLICT (role_id, permission_id) DO NOTHING;
