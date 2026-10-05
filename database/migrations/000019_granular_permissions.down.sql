-- Revert 000019: restore coarse permissions and map back from granular ones.
INSERT INTO permissions (code, display_name, grp) VALUES
    ('users.manage',     'Manage Users',     'admin'),
    ('roles.manage',     'Manage Roles',     'admin'),
    ('factories.manage', 'Manage Factories', 'admin'),
    ('employees.manage', 'Manage Employees', 'admin')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT DISTINCT rp.role_id, pn.id
FROM role_permissions rp
JOIN permissions po ON po.id = rp.permission_id
JOIN permissions pn ON pn.code = CASE po.code
        WHEN 'users.create'     THEN 'users.manage'
        WHEN 'roles.create'     THEN 'roles.manage'
        WHEN 'factories.create' THEN 'factories.manage'
        WHEN 'employees.create' THEN 'employees.manage'
    END
ON CONFLICT (role_id, permission_id) DO NOTHING;

DELETE FROM permissions WHERE code IN (
    'users.view','users.create','users.update','users.assign_roles','users.assign_scopes',
    'roles.view','roles.create','roles.update',
    'factories.view','factories.create','factories.update',
    'areas.view','areas.create','areas.update',
    'employees.view','employees.create','employees.update',
    'master_data.view','master_data.create','master_data.update','master_data.toggle',
    'exchange_rate.view','exchange_rate.update',
    'accounting.journal_create',
    'expenses.create',
    'warehouse.create','items.create',
    'request.view_all','request.submit','request.cancel','request.attach','request.audit_approve'
);
