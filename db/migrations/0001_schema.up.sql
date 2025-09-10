-- USERS
CREATE TABLE IF NOT EXISTS users (
  id             bigserial PRIMARY KEY,
  username       text        NOT NULL UNIQUE,
  password_hash  text        NOT NULL,
  role           text        NOT NULL
);

-- PRODUCTS
CREATE TABLE IF NOT EXISTS products (
  id           bigserial PRIMARY KEY,
  name         text             NOT NULL,
  description  text             NOT NULL,
  price        double precision NOT NULL,
  count        integer          NOT NULL DEFAULT 0
);

-- REFRESH TOKENS
CREATE TABLE IF NOT EXISTS refresh_tokens (
  id         bigserial PRIMARY KEY,
  role       text        NOT NULL,
  token      text        NOT NULL UNIQUE,
  user_id    bigint      NOT NULL,
  expires_at bigint      NOT NULL,
  revoked    boolean     NOT NULL DEFAULT FALSE,
  CONSTRAINT fk_refresh_tokens_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- CART ITEMS
CREATE TABLE IF NOT EXISTS cart_items (
  id         bigserial PRIMARY KEY,
  user_id    bigint   NOT NULL,
  product_id bigint   NOT NULL,
  quantity   integer  NOT NULL DEFAULT 1,
  CONSTRAINT chk_cart_items_quantity_gt0 CHECK (quantity > 0),
  CONSTRAINT fk_cart_items_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_cart_items_product
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cart_items_user_id ON cart_items(user_id);
CREATE INDEX IF NOT EXISTS idx_cart_items_product_id ON cart_items(product_id);

-- ORDERS
CREATE TABLE IF NOT EXISTS orders (
  id         bigserial PRIMARY KEY,
  user_id    bigint             NOT NULL,
  created_at bigint             NOT NULL,
  total      double precision   NOT NULL,
  status     text               NOT NULL,
  CONSTRAINT fk_orders_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
);
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);

-- ORDER ITEMS
CREATE TABLE IF NOT EXISTS order_items (
  id         bigserial PRIMARY KEY,
  order_id   bigint   NOT NULL,
  user_id    bigint   NOT NULL,
  product_id bigint   NOT NULL,
  quantity   integer  NOT NULL DEFAULT 1,
  CONSTRAINT chk_order_items_quantity_gt0 CHECK (quantity > 0),
  CONSTRAINT fk_order_items_order
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
  CONSTRAINT fk_order_items_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
  CONSTRAINT fk_order_items_product
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT
);
CREATE INDEX IF NOT EXISTS idx_order_items_order_id   ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_user_id    ON order_items(user_id);
CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);
