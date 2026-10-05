DROP TABLE IF EXISTS request_attachments;

DELETE FROM expense_categories WHERE code = 'SALARY'
  AND NOT EXISTS (SELECT 1 FROM financial_requests fr WHERE fr.expense_category_id = expense_categories.id)
  AND NOT EXISTS (SELECT 1 FROM expenses e WHERE e.expense_category_id = expense_categories.id);

DELETE FROM accounts WHERE code = '5270'
  AND NOT EXISTS (SELECT 1 FROM journal_lines jl WHERE jl.account_id = accounts.id)
  AND NOT EXISTS (SELECT 1 FROM expense_categories ec WHERE ec.account_id = accounts.id);

ALTER TABLE expense_categories DROP COLUMN IF EXISTS kind;
