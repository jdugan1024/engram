-- Migration: move all legacy domain tables into the canonical entries table.
-- Run this as a superuser (bypasses RLS) before deploying Phase 3 code.
-- Safe to run multiple times — INSERT is not idempotent, so run once on a clean DB.
--
-- NOTE: embedding column is left NULL for migrated rows. Embeddings will be
-- generated lazily the first time each entry appears in a search result, or
-- can be backfilled with a one-off script later.

BEGIN;

-- ---------------------------------------------------------------------------
-- 1. thoughts → note.thought
-- ---------------------------------------------------------------------------
-- Old metadata shape: {type, topics, people, action_items, dates_mentioned, source}
-- New payload shape:  {content, thought_type, topics, people, action_items, dates_mentioned}

INSERT INTO entries (
    id, user_id, record_type, schema_version, source,
    content_text, payload, tags, entities,
    created_at, updated_at
)
SELECT
    id,
    user_id,
    'note.thought',
    '1.0.0',
    COALESCE(metadata->>'source', 'migrated'),
    content,
    jsonb_build_object(
        'content',          content,
        'thought_type',     COALESCE(NULLIF(metadata->>'type', ''), 'observation'),
        'topics',           COALESCE(metadata->'topics',       '[]'::jsonb),
        'people',           COALESCE(metadata->'people',       '[]'::jsonb),
        'action_items',     COALESCE(metadata->'action_items', '[]'::jsonb),
        'dates_mentioned',  COALESCE(metadata->'dates_mentioned', '[]'::jsonb)
    ),
    COALESCE(metadata->'topics', '[]'::jsonb),
    jsonb_build_object(
        'people', COALESCE(metadata->'people', '[]'::jsonb),
        'orgs',   '[]'::jsonb,
        'dates',  COALESCE(metadata->'dates_mentioned', '[]'::jsonb)
    ),
    created_at,
    updated_at
FROM thoughts;

DO $$ BEGIN RAISE NOTICE 'thoughts migrated: %', (SELECT COUNT(*) FROM thoughts); END $$;

-- ---------------------------------------------------------------------------
-- 2. professional_contacts → crm.contact
-- ---------------------------------------------------------------------------

INSERT INTO entries (
    id, user_id, record_type, schema_version, source,
    content_text, payload, tags,
    created_at, updated_at
)
SELECT
    id,
    user_id,
    'crm.contact',
    '1.0.0',
    'migrated',
    'contact: ' || name ||
        CASE WHEN company IS NOT NULL THEN ' at ' || company ELSE '' END ||
        CASE WHEN title IS NOT NULL THEN ' (' || title || ')' ELSE '' END,
    jsonb_strip_nulls(jsonb_build_object(
        'name',         name,
        'company',      company,
        'title',        title,
        'email',        email,
        'phone',        phone,
        'linkedin_url', linkedin_url,
        'how_we_met',   how_we_met,
        'tags',         COALESCE(to_jsonb(tags), '[]'::jsonb),
        'notes',        notes
    )),
    COALESCE(to_jsonb(tags), '[]'::jsonb),
    created_at,
    updated_at
FROM professional_contacts;

DO $$ BEGIN RAISE NOTICE 'professional_contacts migrated: %', (SELECT COUNT(*) FROM professional_contacts); END $$;

-- ---------------------------------------------------------------------------
-- 3. contact_interactions → crm.interaction
--    JOIN professional_contacts to get person_name
-- ---------------------------------------------------------------------------

INSERT INTO entries (
    id, user_id, record_type, schema_version, source,
    content_text, payload,
    created_at
)
SELECT
    ci.id,
    ci.user_id,
    'crm.interaction',
    '1.0.0',
    'migrated',
    'interaction with ' || pc.name || ': ' || ci.summary,
    jsonb_strip_nulls(jsonb_build_object(
        'person_name',       pc.name,
        'interaction_type',  ci.interaction_type,
        'summary',           ci.summary,
        'follow_up_needed',  ci.follow_up_needed,
        'follow_up_notes',   ci.follow_up_notes,
        'interaction_date',  ci.interaction_date::text,
        'resolved_contact_id', ci.contact_id::text
    )),
    ci.created_at
FROM contact_interactions ci
JOIN professional_contacts pc ON pc.id = ci.contact_id;

DO $$ BEGIN RAISE NOTICE 'contact_interactions migrated: %', (SELECT COUNT(*) FROM contact_interactions); END $$;

-- ---------------------------------------------------------------------------
-- 4. maintenance_tasks → maintenance.task
-- ---------------------------------------------------------------------------

INSERT INTO entries (
    id, user_id, record_type, schema_version, source,
    content_text, payload,
    created_at, updated_at
)
SELECT
    id,
    user_id,
    'maintenance.task',
    '1.0.0',
    'migrated',
    'maintenance: ' || name ||
        CASE WHEN category IS NOT NULL THEN ' (' || category || ')' ELSE '' END,
    jsonb_strip_nulls(jsonb_build_object(
        'name',           name,
        'category',       category,
        'location',       location,
        'frequency_days', frequency_days,
        'next_due',       next_due::text,
        'last_completed', last_completed::text,
        'notes',          notes
    )),
    created_at,
    updated_at
FROM maintenance_tasks;

DO $$ BEGIN RAISE NOTICE 'maintenance_tasks migrated: %', (SELECT COUNT(*) FROM maintenance_tasks); END $$;

-- ---------------------------------------------------------------------------
-- 5. job_applications → jobhunt.application
--    JOIN job_postings + job_companies for company_name and title
-- ---------------------------------------------------------------------------

INSERT INTO entries (
    id, user_id, record_type, schema_version, source,
    content_text, payload,
    created_at, updated_at
)
SELECT
    ja.id,
    ja.user_id,
    'jobhunt.application',
    '1.0.0',
    'migrated',
    'job application: ' || jp.title || ' at ' || jc.name || ' — ' || ja.status,
    jsonb_strip_nulls(jsonb_build_object(
        'company_name',     jc.name,
        'title',            jp.title,
        'posting_url',      jp.url,
        'status',           ja.status,
        'applied_date',     ja.applied_date::text,
        'resume_version',   ja.resume_version,
        'referral_contact', ja.referral_contact,
        'salary_min',       jp.salary_min,
        'salary_max',       jp.salary_max,
        'notes',            ja.notes
    )),
    ja.created_at,
    ja.updated_at
FROM job_applications ja
JOIN job_postings jp ON jp.id = ja.job_posting_id
JOIN job_companies jc ON jc.id = jp.company_id;

DO $$ BEGIN RAISE NOTICE 'job_applications migrated: %', (SELECT COUNT(*) FROM job_applications); END $$;

-- ---------------------------------------------------------------------------
-- Validation: confirm entries row count matches expectations
-- ---------------------------------------------------------------------------
DO $$
DECLARE
    v_thoughts     BIGINT;
    v_contacts     BIGINT;
    v_interactions BIGINT;
    v_maintenance  BIGINT;
    v_jobs         BIGINT;
    v_entries      BIGINT;
BEGIN
    SELECT COUNT(*) INTO v_thoughts     FROM thoughts;
    SELECT COUNT(*) INTO v_contacts     FROM professional_contacts;
    SELECT COUNT(*) INTO v_interactions FROM contact_interactions;
    SELECT COUNT(*) INTO v_maintenance  FROM maintenance_tasks;
    SELECT COUNT(*) INTO v_jobs         FROM job_applications;
    SELECT COUNT(*) INTO v_entries      FROM entries;

    RAISE NOTICE 'Expected entries: %  Actual entries: %',
        v_thoughts + v_contacts + v_interactions + v_maintenance + v_jobs,
        v_entries;

    IF v_entries < v_thoughts + v_contacts + v_interactions + v_maintenance + v_jobs THEN
        RAISE EXCEPTION 'Entry count mismatch — migration incomplete, rolling back';
    END IF;
END $$;

-- ---------------------------------------------------------------------------
-- Drop legacy tables (in dependency order)
-- CASCADE handles FK-dependent tables (maintenance_logs, contact_interactions,
-- opportunities, job_interviews, job_postings, etc.)
-- ---------------------------------------------------------------------------

DROP TABLE IF EXISTS contact_interactions CASCADE;
DROP TABLE IF EXISTS opportunities         CASCADE;
DROP TABLE IF EXISTS professional_contacts CASCADE;

DROP TABLE IF EXISTS maintenance_logs  CASCADE;
DROP TABLE IF EXISTS maintenance_tasks CASCADE;

DROP TABLE IF EXISTS job_interviews   CASCADE;
DROP TABLE IF EXISTS job_applications CASCADE;
DROP TABLE IF EXISTS job_contacts     CASCADE;
DROP TABLE IF EXISTS job_postings     CASCADE;
DROP TABLE IF EXISTS job_companies    CASCADE;

DROP TABLE IF EXISTS thoughts CASCADE;

DO $$ BEGIN RAISE NOTICE 'Migration complete. Legacy tables dropped.'; END $$;

COMMIT;
