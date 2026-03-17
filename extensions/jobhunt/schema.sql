-- Job Hunt Pipeline schema for the Go server.
-- RLS uses current_setting('app.current_user_id') set per-transaction by the server.

CREATE TABLE IF NOT EXISTS job_companies (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    name              TEXT        NOT NULL,
    industry          TEXT,
    website           TEXT,
    size              TEXT,
    location          TEXT,
    remote_policy     TEXT,
    notes             TEXT,
    glassdoor_rating  DECIMAL(2,1),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS job_postings (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    company_id      UUID        NOT NULL REFERENCES job_companies(id) ON DELETE CASCADE,
    title           TEXT        NOT NULL,
    url             TEXT,
    salary_min      INTEGER,
    salary_max      INTEGER,
    requirements    TEXT,
    nice_to_haves   TEXT,
    source          TEXT,
    posted_date     DATE,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS job_applications (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    job_posting_id    UUID        NOT NULL REFERENCES job_postings(id) ON DELETE CASCADE,
    status            TEXT        NOT NULL DEFAULT 'applied',
    applied_date      DATE        NOT NULL DEFAULT CURRENT_DATE,
    resume_version    TEXT,
    cover_letter_notes TEXT,
    referral_contact  TEXT,
    notes             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS job_interviews (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    application_id    UUID        NOT NULL REFERENCES job_applications(id) ON DELETE CASCADE,
    interview_type    TEXT        NOT NULL,
    scheduled_at      TIMESTAMPTZ NOT NULL,
    duration_minutes  INTEGER     DEFAULT 60,
    interviewer_name  TEXT,
    interviewer_title TEXT,
    status            TEXT        NOT NULL DEFAULT 'scheduled',
    feedback          TEXT,
    rating            INTEGER     CHECK (rating >= 1 AND rating <= 5),
    notes             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS job_contacts (
    id                          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                     UUID        NOT NULL REFERENCES mcp_users(id) ON DELETE CASCADE,
    company_id                  UUID        REFERENCES job_companies(id) ON DELETE SET NULL,
    name                        TEXT        NOT NULL,
    title                       TEXT,
    email                       TEXT,
    phone                       TEXT,
    role_in_process             TEXT,
    notes                       TEXT,
    professional_crm_contact_id UUID,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_job_companies_user     ON job_companies(user_id);
CREATE INDEX IF NOT EXISTS idx_job_postings_user      ON job_postings(user_id);
CREATE INDEX IF NOT EXISTS idx_job_applications_user  ON job_applications(user_id);
CREATE INDEX IF NOT EXISTS idx_job_interviews_user    ON job_interviews(user_id);
CREATE INDEX IF NOT EXISTS idx_job_contacts_user      ON job_contacts(user_id);

ALTER TABLE job_companies    ENABLE ROW LEVEL SECURITY;
ALTER TABLE job_postings     ENABLE ROW LEVEL SECURITY;
ALTER TABLE job_applications ENABLE ROW LEVEL SECURITY;
ALTER TABLE job_interviews   ENABLE ROW LEVEL SECURITY;
ALTER TABLE job_contacts     ENABLE ROW LEVEL SECURITY;

CREATE POLICY job_companies_rls ON job_companies
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

CREATE POLICY job_postings_rls ON job_postings
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

CREATE POLICY job_applications_rls ON job_applications
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

CREATE POLICY job_interviews_rls ON job_interviews
    FOR ALL
    USING (user_id = current_setting('app.current_user_id')::uuid);

CREATE POLICY job_contacts_rls ON job_contacts
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

DROP TRIGGER IF EXISTS job_companies_updated_at ON job_companies;
CREATE TRIGGER job_companies_updated_at
    BEFORE UPDATE ON job_companies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

DROP TRIGGER IF EXISTS job_postings_updated_at ON job_postings;
CREATE TRIGGER job_postings_updated_at
    BEFORE UPDATE ON job_postings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

DROP TRIGGER IF EXISTS job_applications_updated_at ON job_applications;
CREATE TRIGGER job_applications_updated_at
    BEFORE UPDATE ON job_applications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
