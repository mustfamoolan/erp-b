DROP TABLE IF EXISTS receiving_methods CASCADE;
ALTER TABLE units_of_measure DROP COLUMN IF EXISTS is_active;
ALTER TABLE units_of_measure DROP COLUMN IF EXISTS sort_order;
ALTER TABLE financial_requests DROP COLUMN IF EXISTS advance_sequence_number;
ALTER TABLE financial_requests DROP COLUMN IF EXISTS receiving_location;
ALTER TABLE financial_requests DROP COLUMN IF EXISTS receiver_phone;
ALTER TABLE request_items DROP COLUMN IF EXISTS receipt_number;
