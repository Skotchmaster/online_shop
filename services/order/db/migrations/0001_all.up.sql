CREATE TABLE IF NOT EXISTS orders (
  id         bigserial PRIMARY KEY,
  user_id    bigint   NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  total      double precision NOT NULL,
  status     text     NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_orders_user_id    ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);

CREATE TABLE IF NOT EXISTS order_items (
  id         bigserial PRIMARY KEY,
  order_id   bigint   NOT NULL,
  user_id    bigint   NOT NULL,
  product_id bigint   NOT NULL,
  quantity   integer  NOT NULL DEFAULT 1,
  price      double precision NOT NULL,
  CONSTRAINT chk_order_items_quantity_gt0 CHECK (quantity > 0),
  CONSTRAINT fk_order_items_order
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_order_items_order_id   ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_user_id    ON order_items(user_id);
CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);
