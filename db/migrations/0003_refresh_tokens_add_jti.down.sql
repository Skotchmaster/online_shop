DROP INDEX IF EXISTS idx_refresh_tokens_jti;
ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS jti;