-- Rollback Phase 4+5: Inventory

DROP VIEW  IF EXISTS stock_balances           CASCADE;
DROP TABLE IF EXISTS stock_movements           CASCADE;
DROP TABLE IF EXISTS warehouses                CASCADE;
DROP TABLE IF EXISTS item_variant_attributes   CASCADE;
DROP TABLE IF EXISTS item_variants             CASCADE;
DROP TABLE IF EXISTS items                     CASCADE;
DROP TABLE IF EXISTS units_of_measure          CASCADE;
DROP TABLE IF EXISTS item_categories           CASCADE;
