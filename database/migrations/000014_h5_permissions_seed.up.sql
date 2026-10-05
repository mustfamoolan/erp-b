-- Migration 000014: Seed H5 Permissions

INSERT INTO permissions (code, display_name, grp) VALUES
    ('cashbox.create',          'Create Cashbox',          'cashbox'),
    ('cashbox.opening_balance', 'Establish Opening Balance','cashbox'),
    ('scope.all',               'Access All Scopes',       'admin')
ON CONFLICT (code) DO NOTHING;

-- Assign scope.all to SUPER_ADMIN role
INSERT INTO role_permissions (id, role_id, permission_id, created_at)
SELECT
    gen_random_uuid(),
    r.id,
    p.id,
    NOW()
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'SUPER_ADMIN'
  AND p.code IN (
    'scope.all',
    'cashbox.create',
    'cashbox.opening_balance'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Ensure ACCOUNTANT has cashbox.opening_balance (assuming admin accountant does it)
INSERT INTO role_permissions (id, role_id, permission_id, created_at)
SELECT
    gen_random_uuid(),
    r.id,
    p.id,
    NOW()
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'ACCOUNTANT'
  AND p.code IN (
    'cashbox.create',
    'cashbox.opening_balance'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;
