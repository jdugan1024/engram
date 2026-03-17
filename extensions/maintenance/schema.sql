-- Home Maintenance Tracker schema for the Go server.
-- RLS uses current_setting('app.current_user_id') set per-transaction by the server.

CREATE TABLE IF NOT EXISTS maintenance_tasks (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    name            TEXT        NOT NULL,
    category        TEXT,
    location        TEXT,
    frequency_days  INTEGER,
    next_due        DATE,
    last_completed  DATE,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS maintenance_logs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    task_id         UUID        NOT NULL REFERENCES maintenance_tasks(id) ON DELETE CASCADE,
    completed_date  DATE        NOT NULL DEFAULT CURRENT_DATE,
    performed_by    TEXT,
    cost            DECIMAL(10,2),
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_maintenance_tasks_user ON maintenance_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_logs_user  ON maintenance_logs(user_id);

ALTER TABLE maintenance_tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE maintenance_logs  ENABLE ROW LEVEL SECURITY;

CREATE POLICY maintenance_tasks_rls ON maintenance_tasks
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

CREATE POLICY maintenance_logs_rls ON maintenance_logs
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

-- Auto-update updated_at on maintenance_tasks edits.
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS maintenance_tasks_updated_at ON maintenance_tasks;
CREATE TRIGGER maintenance_tasks_updated_at
    BEFORE UPDATE ON maintenance_tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
