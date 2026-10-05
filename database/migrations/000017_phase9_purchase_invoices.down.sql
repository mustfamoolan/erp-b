DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE grp = 'purchases');
DELETE FROM permissions WHERE grp = 'purchases';

DROP TABLE IF EXISTS purchase_invoice_items;
DROP TABLE IF EXISTS purchase_invoices;
