-- Roadmap correction: there is NO special "imprest funding" request type.
-- The factory uses the normal requests (ADVANCE / FUNDING); the imprest rule
-- is applied automatically on payment. Remove the type added by 000025 (unused).
DELETE FROM request_types
WHERE code = 'IMPREST_FUNDING'
  AND NOT EXISTS (
      SELECT 1 FROM financial_requests fr WHERE fr.request_type_id = request_types.id
  );

-- Safety: if it was somehow used, keep the row but deactivate it (master data is never deleted when used).
UPDATE request_types SET is_active = FALSE WHERE code = 'IMPREST_FUNDING';
