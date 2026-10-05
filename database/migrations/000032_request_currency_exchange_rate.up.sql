ALTER TABLE financial_requests
    ADD COLUMN IF NOT EXISTS exchange_rate NUMERIC(18,6),
    ADD COLUMN IF NOT EXISTS original_amount NUMERIC(18,4);

COMMENT ON COLUMN financial_requests.exchange_rate IS 'سعر الصرف المستخدم في حال كان الطلب بالدولار الأمريكي USD';
COMMENT ON COLUMN financial_requests.original_amount IS 'المبلغ بالعملة الأصلية المدخلة قبل التحويل إلى الدينار العراقي IQD';
