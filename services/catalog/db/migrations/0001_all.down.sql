DROP INDEX IF EXISTS idx_products_description_trgm;
DROP INDEX IF EXISTS idx_products_name_trgm;

DROP INDEX IF EXISTS idx_products_search_vector;

DROP TRIGGER IF EXISTS trg_products_search_vector_update ON products;
DROP FUNCTION IF EXISTS products_search_vector_update();

ALTER TABLE products
  DROP COLUMN IF EXISTS search_vector;

DROP INDEX IF EXISTS idx_products_name;
DROP TABLE IF EXISTS products;
