CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  username      text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  role          text NOT NULL
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token      text NOT NULL,
  jti        text NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  revoked    boolean NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_refresh_user_exp
  ON refresh_tokens (user_id, expires_at);