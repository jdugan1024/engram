-- Family Calendar schema for the Go server.
-- RLS uses current_setting('app.current_user_id') set per-transaction by the server.

CREATE TABLE IF NOT EXISTS family_members (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    name            TEXT        NOT NULL,
    relationship    TEXT,
    date_of_birth   DATE,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS activities (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    family_member_id  UUID        REFERENCES family_members(id) ON DELETE SET NULL,
    title             TEXT        NOT NULL,
    activity_type     TEXT,
    location          TEXT,
    start_date        DATE        NOT NULL,
    end_date          DATE,
    start_time        TIME,
    end_time          TIME,
    day_of_week       TEXT,
    recurring         BOOLEAN     NOT NULL DEFAULT false,
    notes             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS important_dates (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    family_member_id      UUID        REFERENCES family_members(id) ON DELETE SET NULL,
    title                 TEXT        NOT NULL,
    event_date            DATE        NOT NULL,
    recurring_yearly      BOOLEAN     NOT NULL DEFAULT false,
    reminder_days_before  INTEGER     DEFAULT 7,
    notes                 TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_family_members_user  ON family_members(user_id);
CREATE INDEX IF NOT EXISTS idx_activities_user       ON activities(user_id);
CREATE INDEX IF NOT EXISTS idx_important_dates_user  ON important_dates(user_id);

ALTER TABLE family_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE activities     ENABLE ROW LEVEL SECURITY;
ALTER TABLE important_dates ENABLE ROW LEVEL SECURITY;

CREATE POLICY family_members_rls ON family_members
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

CREATE POLICY activities_rls ON activities
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

CREATE POLICY important_dates_rls ON important_dates
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

-- Auto-update updated_at on activities edits.
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS activities_updated_at ON activities;
CREATE TRIGGER activities_updated_at
    BEFORE UPDATE ON activities
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
