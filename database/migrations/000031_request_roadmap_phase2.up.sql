ALTER TABLE financial_requests
    ADD COLUMN receiving_method VARCHAR(50) DEFAULT 'CASH',
    ADD COLUMN purchase_number VARCHAR(100),
    ADD COLUMN work_type VARCHAR(255),
    ADD COLUMN barcode_sku VARCHAR(255);
