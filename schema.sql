-- Open Brain: self-hosted PostgreSQL schema
-- Run this against a non-superuser role (see notes at bottom).

-- Extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "vector";

-- ---------------------------------------------------------------------------
-- Users
-- Each user is identified by their OIDC subject from Authelia.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mcp_users (
    id           UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    name         TEXT        NOT NULL,
    oidc_subject TEXT        NOT NULL UNIQUE,
    created_at   TIMESTAMPTZ DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- Thoughts
-- Identical to the original schema, with user_id added.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS thoughts (
    id         UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id    UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    content    TEXT        NOT NULL,
    embedding  VECTOR(1536),
    metadata   JSONB       DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS thoughts_embedding_idx  ON thoughts USING hnsw (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS thoughts_metadata_idx   ON thoughts USING gin  (metadata);
CREATE INDEX IF NOT EXISTS thoughts_created_at_idx ON thoughts (created_at DESC);
CREATE INDEX IF NOT EXISTS thoughts_user_id_idx    ON thoughts (user_id);

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS thoughts_updated_at ON thoughts;
CREATE TRIGGER thoughts_updated_at
    BEFORE UPDATE ON thoughts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ---------------------------------------------------------------------------
-- Row Level Security
--
-- The Go server calls `SET LOCAL app.current_user_id = '<uuid>'` at the start
-- of every transaction. These policies enforce isolation automatically.
--
-- IMPORTANT: The DATABASE_URL role must NOT be a superuser — superusers bypass
-- RLS. Create a dedicated role:
--
--   CREATE ROLE app_user LOGIN PASSWORD 'changeme';
--   GRANT CONNECT ON DATABASE yourdb TO app_user;
--   GRANT USAGE ON SCHEMA public TO app_user;
--   GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO app_user;
--   GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO app_user;
--   ALTER DEFAULT PRIVILEGES IN SCHEMA public
--     GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_user;
-- ---------------------------------------------------------------------------
ALTER TABLE thoughts ENABLE ROW LEVEL SECURITY;

CREATE POLICY thoughts_user_isolation ON thoughts
    FOR ALL
    USING (
        current_setting('app.current_user_id', true) IS NOT NULL
        AND user_id = current_setting('app.current_user_id', true)::uuid
    )
    WITH CHECK (
        current_setting('app.current_user_id', true) IS NOT NULL
        AND user_id = current_setting('app.current_user_id', true)::uuid
    );

-- mcp_users is readable by the app role but not subject to per-user RLS
-- (the server needs to look up any user by oidc_subject at auth time).

-- ---------------------------------------------------------------------------
-- Semantic search function
--
-- SECURITY INVOKER (the default) means RLS on `thoughts` applies automatically
-- because the function runs in the caller's security context.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION match_thoughts(
    query_embedding VECTOR(1536),
    match_threshold FLOAT DEFAULT 0.5,
    match_count     INT   DEFAULT 10
)
RETURNS TABLE (
    id         UUID,
    content    TEXT,
    metadata   JSONB,
    similarity FLOAT,
    created_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY INVOKER
AS $$
BEGIN
    RETURN QUERY
    SELECT
        t.id,
        t.content,
        t.metadata,
        1 - (t.embedding <=> query_embedding) AS similarity,
        t.created_at
    FROM thoughts t
    WHERE 1 - (t.embedding <=> query_embedding) > match_threshold
    ORDER BY t.embedding <=> query_embedding
    LIMIT match_count;
END;
$$;
