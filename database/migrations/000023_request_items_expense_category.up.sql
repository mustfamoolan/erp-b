ALTER TABLE request_items ADD COLUMN expense_category_id UUID REFERENCES expense_categories(id) ON DELETE SET NULL;
CREATE INDEX idx_request_items_expense_category_id ON request_items(expense_category_id);
