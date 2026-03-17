-- Professional CRM schema for the Go server.
-- RLS uses current_setting('app.current_user_id') set per-transaction by the server.

CREATE TABLE IF NOT EXISTS professional_contacts (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    name            TEXT        NOT NULL,
    company         TEXT,
    title           TEXT,
    email           TEXT,
    phone           TEXT,
    linkedin_url    TEXT,
    how_we_met      TEXT,
    tags            TEXT[]      DEFAULT '{}',
    notes           TEXT,
    last_contacted  DATE,
    follow_up_date  DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS contact_interactions (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    contact_id        UUID        NOT NULL REFERENCES professional_contacts(id) ON DELETE CASCADE,
    interaction_type  TEXT        NOT NULL,
    summary           TEXT        NOT NULL,
    follow_up_needed  BOOLEAN     NOT NULL DEFAULT false,
    follow_up_notes   TEXT,
    interaction_date  DATE        NOT NULL DEFAULT CURRENT_DATE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS opportunities (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    contact_id          UUID        REFERENCES professional_contacts(id) ON DELETE SET NULL,
    title               TEXT        NOT NULL,
    description         TEXT,
    stage               TEXT        NOT NULL DEFAULT 'identified',
    value               DECIMAL(12,2),
    expected_close_date DATE,
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_professional_contacts_user   ON professional_contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_contact_interactions_user     ON contact_interactions(user_id);
CREATE INDEX IF NOT EXISTS idx_opportunities_user            ON opportunities(user_id);

ALTER TABLE professional_contacts  ENABLE ROW LEVEL SECURITY;
ALTER TABLE contact_interactions   ENABLE ROW LEVEL SECURITY;
ALTER TABLE opportunities          ENABLE ROW LEVEL SECURITY;

CREATE POLICY professional_contacts_rls ON professional_contacts
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

CREATE POLICY contact_interactions_rls ON contact_interactions
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

CREATE POLICY opportunities_rls ON opportunities
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

-- Auto-update updated_at.
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS professional_contacts_updated_at ON professional_contacts;
CREATE TRIGGER professional_contacts_updated_at
    BEFORE UPDATE ON professional_contacts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

DROP TRIGGER IF EXISTS opportunities_updated_at ON opportunities;
CREATE TRIGGER opportunities_updated_at
    BEFORE UPDATE ON opportunities
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
