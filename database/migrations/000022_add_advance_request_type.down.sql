-- Revert ADVANCE from the CHECK constraint

ALTER TABLE financial_requests DROP CONSTRAINT IF EXISTS financial_requests_type_check;

ALTER TABLE financial_requests ADD CONSTRAINT financial_requests_type_check 
CHECK (type IN ('FINANCIAL', 'MATERIAL'));
