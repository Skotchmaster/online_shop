CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE products
  ADD COLUMN IF NOT EXISTS search_vector tsvector;

CREATE OR REPLACE FUNCTION products_update_search_vector() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  NEW.search_vector :=
      setweight(to_tsvector('russian', unaccent(coalesce(NEW.name,''))), 'A') ||
      setweight(to_tsvector('english', unaccent(coalesce(NEW.name,''))), 'A') ||
      setweight(to_tsvector('russian', unaccent(coalesce(NEW.description,''))), 'B') ||
      setweight(to_tsvector('english', unaccent(coalesce(NEW.description,''))), 'B');
  RETURN NEW;
END
$$;

DROP TRIGGER IF EXISTS trg_products_search_vector ON products;
CREATE TRIGGER trg_products_search_vector
BEFORE INSERT OR UPDATE OF name, description
ON products
FOR EACH ROW EXECUTE FUNCTION products_update_search_vector();

UPDATE products SET name = name;

CREATE INDEX IF NOT EXISTS idx_products_search_vector ON products USING GIN (search_vector);
CREATE INDEX IF NOT EXISTS idx_products_name_trgm ON products USING GIN (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_products_desc_trgm ON products USING GIN (description gin_trgm_ops);
