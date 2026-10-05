CREATE TABLE purchase_invoices (
    id UUID PRIMARY KEY,
    document_number VARCHAR(50) NOT NULL UNIQUE,
    invoice_number VARCHAR(100), -- From supplier
    invoice_date DATE NOT NULL,
    scope_id UUID NOT NULL REFERENCES organization_scopes(id),
    supplier_name VARCHAR(255) NOT NULL,
    total_amount DECIMAL(19,4) NOT NULL,
    discount DECIMAL(19,4) NOT NULL DEFAULT 0.0,
    tax DECIMAL(19,4) NOT NULL DEFAULT 0.0,
    net_amount DECIMAL(19,4) NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'IQD',
    payment_status VARCHAR(50) NOT NULL DEFAULT 'UNPAID', -- UNPAID, PARTIAL, PAID
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT', -- DRAFT, APPROVED, CANCELLED
    notes TEXT,
    attachments JSONB,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_purchase_invoices_scope_id ON purchase_invoices(scope_id);
CREATE INDEX idx_purchase_invoices_status ON purchase_invoices(status);
CREATE INDEX idx_purchase_invoices_payment_status ON purchase_invoices(payment_status);

CREATE TABLE purchase_invoice_items (
    id UUID PRIMARY KEY,
    invoice_id UUID NOT NULL REFERENCES purchase_invoices(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    quantity DECIMAL(19,4) NOT NULL,
    unit_id UUID REFERENCES units_of_measure(id),
    unit_price DECIMAL(19,4) NOT NULL,
    total DECIMAL(19,4) NOT NULL,
    notes TEXT
);

CREATE INDEX idx_purchase_invoice_items_invoice_id ON purchase_invoice_items(invoice_id);

-- Insert Permissions (schema: code, display_name, description, grp)
INSERT INTO permissions (code, display_name, description, grp) VALUES
    ('purchases.create',  'Create Purchase Invoices',  'Can create and edit draft purchase invoices', 'purchases'),
    ('purchases.view',    'View Purchase Invoices',    'Can view purchase invoices',                  'purchases'),
    ('purchases.approve', 'Approve Purchase Invoices', 'Can approve purchase invoices',               'purchases'),
    ('purchases.pay',     'Pay Purchase Invoices',     'Can record payments for purchase invoices',   'purchases')
ON CONFLICT (code) DO NOTHING;
