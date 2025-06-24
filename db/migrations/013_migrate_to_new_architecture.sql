-- =====================================================
-- DATA MIGRATION: OLD ARCHITECTURE → NEW ARCHITECTURE
-- =====================================================
-- This script migrates existing data from the old schema to the new three-tier architecture
-- Run this AFTER 012_elegant_etl_architecture.sql
-- =====================================================

-- =====================================================
-- PHASE 1: MIGRATE EXISTING JOB TEMPLATES TO JOBS_V2
-- =====================================================

-- First, ensure the new schema exists
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'aquaflow' AND table_name = 'etl_jobs_v2') THEN
        RAISE EXCEPTION 'New schema not found. Please run 012_elegant_etl_architecture.sql first.';
    END IF;
END $$;

-- Migrate job templates to job definitions (if etl_job_templates exists)
INSERT INTO aquaflow.etl_jobs_v2 (job_id, job_name, job_type, description, parameters, created_at, updated_at)
SELECT 
    template_id as job_id,
    template_name as job_name,
    job_type,
    'Migrated from job template: ' || template_name as description,
    parameters,
    created_at,
    updated_at
FROM aquaflow.etl_job_templates
WHERE NOT EXISTS (
    SELECT 1 FROM aquaflow.etl_jobs_v2 
    WHERE job_name = aquaflow.etl_job_templates.template_name
)
ON CONFLICT (job_name) DO NOTHING;

-- Migrate schedule information from templates
INSERT INTO aquaflow.etl_schedules (job_id, schedule_name, cron_expression, next_run, last_run, run_count)
SELECT 
    t.template_id as job_id,
    'Default Schedule' as schedule_name,
    t.schedule as cron_expression,
    t.next_run,
    NULL as last_run,
    COALESCE(t.created_job_count, 0) as run_count
FROM aquaflow.etl_job_templates t
WHERE t.schedule IS NOT NULL
  AND t.is_active = true
  AND NOT EXISTS (
    SELECT 1 FROM aquaflow.etl_schedules s 
    WHERE s.job_id = t.template_id
)
ON CONFLICT (job_id, schedule_name) DO NOTHING;

-- =====================================================
-- PHASE 2: MIGRATE EXISTING JOB INSTANCES TO RUNS
-- =====================================================

-- Migrate template-based job instances to runs
INSERT INTO aquaflow.etl_job_runs (
    job_id, 
    schedule_id,
    run_name,
    status,
    trigger_type,
    started_at,
    completed_at,
    records_processed,
    records_failed,
    error_message,
    retry_count,
    runtime_parameters
)
SELECT 
    j.template_id as job_id,
    s.schedule_id,
    j.job_name as run_name,
    CASE 
        WHEN j.status = 'pending' THEN 'queued'
        WHEN j.status = 'running' THEN 'running'
        WHEN j.status = 'completed' THEN 'completed'
        WHEN j.status = 'failed' THEN 'failed'
        WHEN j.status = 'completed_with_errors' THEN 'completed_with_errors'
        ELSE 'failed'
    END as status,
    CASE 
        WHEN j.is_template_instance = true THEN 'scheduled'
        ELSE 'manual'
    END as trigger_type,
    j.started_at,
    j.completed_at,
    j.records_processed,
    j.records_failed,
    j.error_message,
    COALESCE(j.retry_count, 0) as retry_count,
    j.parameters as runtime_parameters
FROM aquaflow.etl_jobs j
LEFT JOIN aquaflow.etl_schedules s ON j.template_id = s.job_id
WHERE j.template_id IS NOT NULL
  AND j.is_template_instance = true
ON CONFLICT DO NOTHING;

-- Migrate standalone jobs (non-template) as both job definitions and runs
WITH standalone_jobs AS (
    SELECT 
        gen_random_uuid() as new_job_id,
        job_name,
        job_type,
        parameters,
        started_at,
        batch_id
    FROM aquaflow.etl_jobs 
    WHERE template_id IS NULL OR is_template_instance IS NULL OR is_template_instance = false
),
inserted_jobs AS (
    INSERT INTO aquaflow.etl_jobs_v2 (job_id, job_name, job_type, description, parameters, created_at)
    SELECT 
        new_job_id,
        job_name || ' (Migrated)',
        job_type,
        'Migrated standalone job',
        parameters,
        started_at
    FROM standalone_jobs
    ON CONFLICT (job_name) DO UPDATE SET job_name = EXCLUDED.job_name || ' (' || gen_random_uuid() || ')'
    RETURNING job_id, job_name, created_at
)
INSERT INTO aquaflow.etl_job_runs (
    job_id,
    run_name,
    status,
    trigger_type,
    started_at,
    completed_at,
    records_processed,
    records_failed,
    error_message,
    retry_count,
    runtime_parameters
)
SELECT 
    ij.job_id,
    j.job_name as run_name,
    CASE 
        WHEN j.status = 'pending' THEN 'queued'
        WHEN j.status = 'running' THEN 'running'
        WHEN j.status = 'completed' THEN 'completed'
        WHEN j.status = 'failed' THEN 'failed'
        WHEN j.status = 'completed_with_errors' THEN 'completed_with_errors'
        ELSE 'completed'
    END as status,
    'manual' as trigger_type,
    j.started_at,
    j.completed_at,
    j.records_processed,
    j.records_failed,
    j.error_message,
    COALESCE(j.retry_count, 0) as retry_count,
    j.parameters as runtime_parameters
FROM aquaflow.etl_jobs j
JOIN inserted_jobs ij ON (j.job_name || ' (Migrated)') = ij.job_name OR j.job_name LIKE ij.job_name || ' (%'
WHERE j.template_id IS NULL OR j.is_template_instance IS NULL OR j.is_template_instance = false;

-- =====================================================
-- PHASE 3: MIGRATE EXISTING LOGS TO NEW STRUCTURE
-- =====================================================

-- First, update existing logs with run_id where possible
UPDATE aquaflow.etl_job_logs 
SET run_id = (
    SELECT r.run_id 
    FROM aquaflow.etl_job_runs r
    JOIN aquaflow.etl_jobs j ON r.job_id = (
        SELECT j2.template_id 
        FROM aquaflow.etl_jobs j2 
        WHERE j2.batch_id = aquaflow.etl_job_logs.batch_id
        LIMIT 1
    )
    WHERE j.batch_id = aquaflow.etl_job_logs.batch_id
    LIMIT 1
)
WHERE run_id IS NULL 
  AND batch_id IS NOT NULL;

-- Migrate logs to new structure
INSERT INTO aquaflow.etl_job_logs_v2 (
    run_id,
    timestamp,
    log_level,
    message,
    context,
    component
)
SELECT 
    l.run_id,
    l.timestamp,
    l.log_level,
    l.message,
    l.context,
    'migrated' as component
FROM aquaflow.etl_job_logs l
WHERE l.run_id IS NOT NULL
ON CONFLICT DO NOTHING;

-- =====================================================
-- PHASE 4: UPDATE STATISTICS AND METADATA
-- =====================================================

-- Update run counts in schedules based on actual runs
UPDATE aquaflow.etl_schedules 
SET run_count = (
    SELECT COUNT(*) 
    FROM aquaflow.etl_job_runs r 
    WHERE r.schedule_id = aquaflow.etl_schedules.schedule_id
),
failure_count = (
    SELECT COUNT(*) 
    FROM aquaflow.etl_job_runs r 
    WHERE r.schedule_id = aquaflow.etl_schedules.schedule_id 
      AND r.status IN ('failed', 'completed_with_errors')
),
last_run = (
    SELECT MAX(r.completed_at) 
    FROM aquaflow.etl_job_runs r 
    WHERE r.schedule_id = aquaflow.etl_schedules.schedule_id
      AND r.status IN ('completed', 'completed_with_errors')
);

-- Update job activity based on recent runs
UPDATE aquaflow.etl_jobs_v2 
SET is_active = (
    SELECT COUNT(*) > 0
    FROM aquaflow.etl_job_runs r 
    WHERE r.job_id = aquaflow.etl_jobs_v2.job_id
      AND r.started_at > NOW() - INTERVAL '30 days'
);

-- =====================================================
-- PHASE 5: CREATE HELPFUL MIGRATION REPORT
-- =====================================================

-- Create temporary migration report
CREATE TEMP TABLE migration_report AS
SELECT 
    'Jobs Migrated' as metric,
    COUNT(*) as count
FROM aquaflow.etl_jobs_v2
UNION ALL
SELECT 
    'Schedules Migrated' as metric,
    COUNT(*) as count
FROM aquaflow.etl_schedules
UNION ALL
SELECT 
    'Runs Migrated' as metric,
    COUNT(*) as count
FROM aquaflow.etl_job_runs
UNION ALL
SELECT 
    'Logs Migrated' as metric,
    COUNT(*) as count
FROM aquaflow.etl_job_logs_v2
UNION ALL
SELECT 
    'Original Jobs' as metric,
    COUNT(*) as count
FROM aquaflow.etl_jobs
UNION ALL
SELECT 
    'Original Templates' as metric,
    COUNT(*) as count
FROM aquaflow.etl_job_templates
WHERE EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'aquaflow' AND table_name = 'etl_job_templates')
UNION ALL
SELECT 
    'Original Logs' as metric,
    COUNT(*) as count
FROM aquaflow.etl_job_logs;

-- Display migration report
SELECT * FROM migration_report ORDER BY metric;

-- =====================================================
-- PHASE 6: VALIDATION AND CLEANUP PREPARATION
-- =====================================================

-- Create backup table references for validation
CREATE OR REPLACE VIEW aquaflow.vw_migration_validation AS
SELECT 
    'Data Integrity Check' as check_type,
    CASE 
        WHEN jobs_count > 0 AND schedules_count > 0 AND runs_count > 0 THEN 'PASS'
        ELSE 'FAIL'
    END as status,
    json_build_object(
        'jobs', jobs_count,
        'schedules', schedules_count, 
        'runs', runs_count,
        'logs', logs_count
    ) as details
FROM (
    SELECT 
        (SELECT COUNT(*) FROM aquaflow.etl_jobs_v2) as jobs_count,
        (SELECT COUNT(*) FROM aquaflow.etl_schedules) as schedules_count,
        (SELECT COUNT(*) FROM aquaflow.etl_job_runs) as runs_count,
        (SELECT COUNT(*) FROM aquaflow.etl_job_logs_v2) as logs_count
) counts;

-- Show validation results
SELECT * FROM aquaflow.vw_migration_validation;

-- Add migration metadata
INSERT INTO aquaflow.etl_job_logs_v2 (
    run_id,
    timestamp,
    log_level,
    message,
    context,
    component
) 
SELECT 
    (SELECT run_id FROM aquaflow.etl_job_runs LIMIT 1),
    NOW(),
    'INFO',
    'Migration completed successfully',
    json_build_object(
        'migration_date', NOW(),
        'migration_version', '013',
        'jobs_migrated', (SELECT COUNT(*) FROM aquaflow.etl_jobs_v2),
        'schedules_migrated', (SELECT COUNT(*) FROM aquaflow.etl_schedules),
        'runs_migrated', (SELECT COUNT(*) FROM aquaflow.etl_job_runs)
    ),
    'migration'
WHERE EXISTS (SELECT 1 FROM aquaflow.etl_job_runs);

COMMENT ON SCHEMA aquaflow IS 'AquaFlow Analytics ETL Schema - Migrated to elegant three-tier architecture on ' || NOW();