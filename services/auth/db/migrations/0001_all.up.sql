CREATE TABLE IF NOT EXISTS users (
  id             bigserial PRIMARY KEY,
  username       text        NOT NULL UNIQUE,
  password_hash  text        NOT NULL,
  role           text        NOT NULL
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id         bigserial PRIMARY KEY,
  role       text   NOT NULL,
  token      text   NOT NULL UNIQUE,
  user_id    bigint NOT NULL,
  expires_at bigint NOT NULL,
  revoked    boolean NOT NULL DEFAULT FALSE,
  jti        text   NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_tokens_jti ON refresh_tokens(jti);
