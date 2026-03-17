-- Household Knowledge Base schema for the Go server.
-- RLS uses current_setting('app.current_user_id') set per-transaction by the server,
-- rather than Supabase's auth.uid().

CREATE TABLE IF NOT EXISTS household_items (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    category    TEXT,
    location    TEXT,
    details     JSONB       NOT NULL DEFAULT '{}',
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS household_vendors (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    name         TEXT        NOT NULL,
    service_type TEXT,
    phone        TEXT,
    email        TEXT,
    website      TEXT,
    notes        TEXT,
    rating       INTEGER     CHECK (rating >= 1 AND rating <= 5),
    last_used    DATE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_household_items_user ON household_items(user_id);
CREATE INDEX IF NOT EXISTS idx_household_vendors_user ON household_vendors(user_id);

ALTER TABLE household_items  ENABLE ROW LEVEL SECURITY;
ALTER TABLE household_vendors ENABLE ROW LEVEL SECURITY;

CREATE POLICY household_items_rls ON household_items
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

CREATE POLICY household_vendors_rls ON household_vendors
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

-- Auto-update updated_at on household_items edits.
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS household_items_updated_at ON household_items;
CREATE TRIGGER household_items_updated_at
    BEFORE UPDATE ON household_items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
