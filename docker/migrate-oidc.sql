-- Migration: switch from access_key auth to OIDC subject
-- Run this against the existing database before deploying the OAuth update.

ALTER TABLE mcp_users ADD COLUMN IF NOT EXISTS oidc_subject TEXT UNIQUE;

-- Set the OIDC subject to match the Authelia user UUID.
-- Find your UUID at: https://auth.x1024.net/.well-known/openid-configuration
-- (complete an auth flow and decode the id_token's "sub" claim)
UPDATE mcp_users SET oidc_subject = 'e816ea9a-646c-4c03-a27f-48469abfedcd' WHERE oidc_subject IS NULL;

ALTER TABLE mcp_users ALTER COLUMN oidc_subject SET NOT NULL;
ALTER TABLE mcp_users DROP COLUMN IF EXISTS access_key;
