-- ============================================================
-- Migration 000008: Seed Role-Permission Assignments
-- Roadmap §9, §10 — each role must have its permissions
-- SUPER_ADMIN inherits all via bypass in middleware
-- ============================================================

-- ────────────────────────────────────────────────────────────
-- FACTORY_ACCOUNTANT permissions
-- Can: create/view/submit/receive/complete requests, view cashbox,
--      view warehouse, view accounting
-- ────────────────────────────────────────────────────────────
INSERT INTO role_permissions (id, role_id, permission_id, created_at)
SELECT
    gen_random_uuid(),
    r.id,
    p.id,
    NOW()
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'FACTORY_ACCOUNTANT'
  AND p.code IN (
    'request.create',
    'request.view',
    'request.receive',
    'cashbox.view',
    'warehouse.view',
    'warehouse.receive',
    'accounting.view',
    'audit.view',
    'reports.view'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ────────────────────────────────────────────────────────────
-- FACTORY_EMPLOYEE permissions
-- Can: view requests, view warehouse
-- ────────────────────────────────────────────────────────────
INSERT INTO role_permissions (id, role_id, permission_id, created_at)
SELECT
    gen_random_uuid(),
    r.id,
    p.id,
    NOW()
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'FACTORY_EMPLOYEE'
  AND p.code IN (
    'request.view',
    'warehouse.view'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ────────────────────────────────────────────────────────────
-- AUDITOR permissions
-- Can: view and review requests, view audit
-- ────────────────────────────────────────────────────────────
INSERT INTO role_permissions (id, role_id, permission_id, created_at)
SELECT
    gen_random_uuid(),
    r.id,
    p.id,
    NOW()
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'AUDITOR'
  AND p.code IN (
    'request.view',
    'request.review',
    'request.reject',
    'audit.view',
    'reports.view',
    'accounting.view'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ────────────────────────────────────────────────────────────
-- ACCOUNTANT permissions
-- Can: view, approve/reject requests, pay, view cashbox & accounting
-- ────────────────────────────────────────────────────────────
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
    'request.view',
    'request.approve',
    'request.reject',
    'request.pay',
    'cashbox.view',
    'cashbox.transfer',
    'accounting.view',
    'accounting.post',
    'audit.view',
    'reports.view'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ────────────────────────────────────────────────────────────
-- ADMINISTRATOR permissions
-- Can: everything except super-admin specific ops
-- ────────────────────────────────────────────────────────────
INSERT INTO role_permissions (id, role_id, permission_id, created_at)
SELECT
    gen_random_uuid(),
    r.id,
    p.id,
    NOW()
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'ADMINISTRATOR'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ────────────────────────────────────────────────────────────
-- SUPER_ADMIN permissions
-- Gets ALL permissions (also has bypass in middleware)
-- ────────────────────────────────────────────────────────────
INSERT INTO role_permissions (id, role_id, permission_id, created_at)
SELECT
    gen_random_uuid(),
    r.id,
    p.id,
    NOW()
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'SUPER_ADMIN'
ON CONFLICT (role_id, permission_id) DO NOTHING;
