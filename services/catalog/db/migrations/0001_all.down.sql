DROP INDEX IF EXISTS idx_products_description_trgm;
DROP INDEX IF EXISTS idx_products_name_trgm;

DROP INDEX IF EXISTS idx_products_search_vector;

ALTER TABLE products
  DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_products_name;
DROP TABLE IF EXISTS products;