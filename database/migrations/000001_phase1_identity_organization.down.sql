-- Rollback Phase 1: Identity & Organization Foundation
-- Order: reverse of creation (foreign key constraints)

DROP TABLE IF EXISTS user_scope_access    CASCADE;
DROP TABLE IF EXISTS user_roles            CASCADE;
DROP TABLE IF EXISTS users                 CASCADE;
DROP TABLE IF EXISTS role_permissions      CASCADE;
DROP TABLE IF EXISTS permissions           CASCADE;
DROP TABLE IF EXISTS roles                 CASCADE;
DROP TABLE IF EXISTS employees             CASCADE;
DROP TABLE IF EXISTS organization_scopes   CASCADE;
