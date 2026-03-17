-- Meal Planning schema for the Go server.
-- RLS uses current_setting('app.current_user_id') set per-transaction by the server.

CREATE TABLE IF NOT EXISTS recipes (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    name                TEXT        NOT NULL,
    cuisine             TEXT,
    prep_time_minutes   INTEGER,
    cook_time_minutes   INTEGER,
    servings            INTEGER,
    ingredients         JSONB       NOT NULL DEFAULT '[]',
    instructions        JSONB       NOT NULL DEFAULT '[]',
    tags                TEXT[]      DEFAULT '{}',
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS meal_plans (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    week_start      DATE        NOT NULL,
    day_of_week     TEXT        NOT NULL,
    meal_type       TEXT        NOT NULL,
    recipe_id       UUID        REFERENCES recipes(id) ON DELETE SET NULL,
    custom_meal     TEXT,
    servings        INTEGER     DEFAULT 4,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, week_start, day_of_week, meal_type)
);

CREATE TABLE IF NOT EXISTS shopping_lists (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    week_start      DATE        NOT NULL,
    item_name       TEXT        NOT NULL,
    quantity        TEXT,
    unit            TEXT,
    recipe_source   TEXT,
    purchased       BOOLEAN     NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_recipes_user        ON recipes(user_id);
CREATE INDEX IF NOT EXISTS idx_meal_plans_user      ON meal_plans(user_id);
CREATE INDEX IF NOT EXISTS idx_shopping_lists_user  ON shopping_lists(user_id);

ALTER TABLE recipes        ENABLE ROW LEVEL SECURITY;
ALTER TABLE meal_plans     ENABLE ROW LEVEL SECURITY;
ALTER TABLE shopping_lists ENABLE ROW LEVEL SECURITY;

CREATE POLICY recipes_rls ON recipes
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

CREATE POLICY meal_plans_rls ON meal_plans
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

CREATE POLICY shopping_lists_rls ON shopping_lists
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

-- Auto-update updated_at on recipes edits.
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS recipes_updated_at ON recipes;
CREATE TRIGGER recipes_updated_at
    BEFORE UPDATE ON recipes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
