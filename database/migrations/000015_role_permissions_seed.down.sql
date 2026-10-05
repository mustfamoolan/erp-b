-- Rollback: Remove role-permission assignments added in 000008
DELETE FROM role_permissions
WHERE (role_id, permission_id) IN (
    SELECT rp.role_id, rp.permission_id
    FROM role_permissions rp
    JOIN roles r ON rp.role_id = r.id
    WHERE r.name IN ('FACTORY_ACCOUNTANT','FACTORY_EMPLOYEE','AUDITOR','ACCOUNTANT','ADMINISTRATOR','SUPER_ADMIN')
);
