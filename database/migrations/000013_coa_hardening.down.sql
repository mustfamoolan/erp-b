-- Down migration for Phase 4.5
ALTER TABLE cashboxes DROP CONSTRAINT IF EXISTS cashboxes_account_id_unique;

DROP TABLE IF EXISTS expense_categories;
DROP TABLE IF EXISTS request_types;
DROP TABLE IF EXISTS account_mappings;

-- We shouldn't drop accounts if they have journal lines.
-- For safety in down migration, we just set them to inactive.
UPDATE accounts SET is_active = FALSE WHERE code IN (
    '1110', '3100', '5210', '5220', '5230', '5240', '5250', '5260'
) OR code LIKE '1110-%';

UPDATE accounts SET code = '1110', sort_order = 115 WHERE code = '1104';
