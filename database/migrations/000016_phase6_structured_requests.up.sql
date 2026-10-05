-- Phase 6: Structured Financial Requests
-- Add Master Data references and new structured fields to financial_requests.

ALTER TABLE financial_requests 
    ADD COLUMN request_type_id UUID REFERENCES request_types(id) ON DELETE RESTRICT,
    ADD COLUMN expense_category_id UUID REFERENCES expense_categories(id) ON DELETE RESTRICT,
    ADD COLUMN supplier_name VARCHAR(255),
    ADD COLUMN receiver_name VARCHAR(255),
    ADD COLUMN project_name VARCHAR(255),
    ADD COLUMN attachments JSONB;

-- In a real scenario, we might want to migrate existing data. 
-- For now, new fields will be nullable, but we can enforce them later or via application logic.
