-- ERP Database Verification Script
-- Run this manually against your PostgreSQL to confirm Phase 1 is applied correctly.
-- Roadmap §12 — Phase 1 tests

-- 1. Verify all tables exist
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public'
ORDER BY table_name;

-- Expected output:
-- employees
-- organization_scopes
-- permissions
-- role_permissions
-- roles
-- user_roles
-- user_scope_access
-- users

-- 2. Verify Administration scope was seeded
SELECT id, type, name, code, status FROM organization_scopes;

-- 3. Verify all 6 roles seeded
SELECT name, display_name, is_system FROM roles ORDER BY name;

-- 4. Verify all permissions seeded (should be 25)
SELECT COUNT(*) as total_permissions FROM permissions;
SELECT grp, COUNT(*) FROM permissions GROUP BY grp ORDER BY grp;

-- 5. Test scope isolation constraint:
-- A user cannot have user_scope_access duplicates
-- Attempt duplicate insert should fail:
-- INSERT INTO user_scope_access (user_id, scope_id, granted_by) VALUES (x, y, z);
-- INSERT INTO user_scope_access (user_id, scope_id, granted_by) VALUES (x, y, z); -- should fail

-- 6. Verify unique constraints on organization_scopes
SELECT constraint_name, constraint_type 
FROM information_schema.table_constraints 
WHERE table_name = 'organization_scopes';

-- 7. Verify FK constraints exist on user_scope_access
SELECT
    tc.constraint_name,
    tc.constraint_type,
    kcu.column_name,
    ccu.table_name AS foreign_table,
    ccu.column_name AS foreign_column
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage ccu ON tc.constraint_name = ccu.constraint_name
WHERE tc.table_name = 'user_scope_access'
  AND tc.constraint_type = 'FOREIGN KEY';
