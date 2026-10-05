DROP INDEX IF EXISTS idx_request_items_expense_category_id;
ALTER TABLE request_items DROP COLUMN IF EXISTS expense_category_id;
