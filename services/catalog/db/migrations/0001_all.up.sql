CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS products (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name        text NOT NULL,
  description text NOT NULL,
  price       bigint NOT NULL CHECK (price >= 0),
  count       integer NOT NULL DEFAULT 0 CHECK (count >= 0)
);

CREATE INDEX IF NOT EXISTS idx_products_name ON products (name);

CREATE EXTENSION IF NOT EXISTS unaccent;

CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE products
  ADD COLUMN IF NOT EXISTS search_vector tsvector
  DEFAULT ''::tsvector;

CREATE OR REPLACE FUNCTION products_search_vector_update()
RETURNS trigger AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('russian', unaccent(coalesce(NEW.name, '') || ' ' || coalesce(NEW.description, ''))), 'A')
    ||
    setweight(to_tsvector('english', unaccent(coalesce(NEW.name, '') || ' ' || coalesce(NEW.description, ''))), 'B');
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_products_search_vector_update ON products;

CREATE TRIGGER trg_products_search_vector_update
BEFORE INSERT OR UPDATE OF name, description
ON products
FOR EACH ROW
EXECUTE FUNCTION products_search_vector_update();

UPDATE products
SET search_vector =
  setweight(to_tsvector('russian', unaccent(coalesce(name, '') || ' ' || coalesce(description, ''))), 'A')
  ||
  setweight(to_tsvector('english', unaccent(coalesce(name, '') || ' ' || coalesce(description, ''))), 'B');

CREATE INDEX IF NOT EXISTS idx_products_search_vector
  ON products USING GIN (search_vector);

CREATE INDEX IF NOT EXISTS idx_products_name_trgm
  ON products USING GIN (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_products_description_trgm
  ON products USING GIN (description gin_trgm_ops);
