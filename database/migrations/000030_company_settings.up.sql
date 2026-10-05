-- 000030_company_settings.up.sql

CREATE TABLE company_settings (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_name VARCHAR(255) NOT NULL,
    phone_number VARCHAR(50),
    logo_url     VARCHAR(500),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by   UUID REFERENCES users(id) ON DELETE SET NULL
);

-- We only ever need one row. We can enforce this with a check constraint.
ALTER TABLE company_settings ADD CONSTRAINT check_single_row CHECK (id = '00000000-0000-0000-0000-000000000000');

-- Insert the default settings row
INSERT INTO company_settings (id, company_name) VALUES ('00000000-0000-0000-0000-000000000000', 'معمل و شركة المقاولات');

-- Insert new permission for managing settings
INSERT INTO permissions (code, display_name, grp) VALUES
    ('settings.manage', 'إدارة إعدادات الشركة', 'admin');

-- Add permission to SUPER_ADMIN and ADMINISTRATOR
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name IN ('SUPER_ADMIN', 'ADMINISTRATOR')
AND p.code = 'settings.manage';
