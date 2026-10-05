-- Revert Phase 6: Structured Financial Requests

ALTER TABLE financial_requests 
    DROP COLUMN IF EXISTS request_type_id,
    DROP COLUMN IF EXISTS expense_category_id,
    DROP COLUMN IF EXISTS supplier_name,
    DROP COLUMN IF EXISTS receiver_name,
    DROP COLUMN IF EXISTS project_name,
    DROP COLUMN IF EXISTS attachments;
