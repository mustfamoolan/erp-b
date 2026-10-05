ALTER TABLE organization_scopes ADD COLUMN area_id UUID REFERENCES areas(id) ON DELETE SET NULL;
