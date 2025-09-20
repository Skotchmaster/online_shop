ALTER TABLE refresh_tokens
  ADD COLUMN IF NOT EXISTS jti text;

ALTER TABLE refresh_tokens
  ALTER COLUMN jti SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_tokens_jti
  ON refresh_tokens (jti);
