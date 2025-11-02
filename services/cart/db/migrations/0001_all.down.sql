DROP INDEX IF EXISTS idx_outbox_unsent;
DROP INDEX IF EXISTS idx_products_desc_trgm;
DROP INDEX IF EXISTS idx_products_name_trgm;
DROP INDEX IF EXISTS idx_products_search_vector;
DROP INDEX IF EXISTS idx_reservations_product;

DROP TRIGGER IF EXISTS trg_products_search_vector ON products;
DROP FUNCTION IF EXISTS products_update_search_vector();
ALTER TABLE products DROP COLUMN IF EXISTS search_vector;

DROP TABLE IF EXISTS outbox;
DROP TABLE IF EXISTS reservations;
DROP TABLE IF EXISTS products;

