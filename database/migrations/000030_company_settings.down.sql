-- 000030_company_settings.down.sql

DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code = 'settings.manage');
DELETE FROM permissions WHERE code = 'settings.manage';

DROP TABLE IF EXISTS company_settings;
