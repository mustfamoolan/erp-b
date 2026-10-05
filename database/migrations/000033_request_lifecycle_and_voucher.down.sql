-- Migration 000033 Down
DELETE FROM permissions WHERE code IN ('request.return_revision', 'request.disburse', 'request.deliver');
ALTER TABLE financial_requests DROP COLUMN IF EXISTS signed_voucher_url;
ALTER TABLE financial_requests DROP COLUMN IF EXISTS disbursed_at;
ALTER TABLE financial_requests DROP COLUMN IF EXISTS delivered_at;
