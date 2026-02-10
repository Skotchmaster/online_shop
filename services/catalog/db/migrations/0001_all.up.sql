CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS products (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name        text NOT NULL,
  description text NOT NULL,
  price       bigint NOT NULL CHECK (price >= 0),
  count       integer NOT NULL DEFAULT 0 CHECK (count >= 0)
);

CREATE INDEX IF NOT EXISTS idx_products_name ON products (name);
