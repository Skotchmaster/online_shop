CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS cart_items (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    uuid NOT NULL,
  product_id uuid NOT NULL,
  quantity   integer NOT NULL DEFAULT 1 CHECK (quantity > 0)
);

CREATE INDEX IF NOT EXISTS idx_cart_items_user_id
  ON cart_items(user_id);

CREATE UNIQUE INDEX IF NOT EXISTS ux_cart_items_user_product
  ON cart_items(user_id, product_id);
