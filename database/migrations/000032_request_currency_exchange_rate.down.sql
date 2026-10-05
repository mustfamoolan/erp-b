ALTER TABLE financial_requests
    DROP COLUMN IF EXISTS exchange_rate,
    DROP COLUMN IF EXISTS original_amount;
