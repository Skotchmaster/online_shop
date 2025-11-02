CREATE TABLE IF NOT EXISTS cart_items (
  id         bigserial PRIMARY KEY,
  user_id    bigint   NOT NULL,
  product_id bigint   NOT NULL,
  quantity   integer  NOT NULL DEFAULT 1,
  CONSTRAINT chk_cart_items_quantity_gt0 CHECK (quantity > 0),
  CONSTRAINT uniq_cart_user_product UNIQUE (user_id, product_id)
);

CREATE INDEX IF NOT EXISTS idx_cart_items_user_id    ON cart_items(user_id);
CREATE INDEX IF NOT EXISTS idx_cart_items_product_id ON cart_items(product_id);
