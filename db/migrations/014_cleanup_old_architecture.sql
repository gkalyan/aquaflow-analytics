-- =====================================================
-- CLEANUP OLD ARCHITECTURE DATA
-- =====================================================
-- This script safely removes data from the old ETL architecture
-- Run this AFTER confirming the new architecture is working properly
-- =====================================================

-- Create backup tables first for safety
CREATE TABLE IF NOT EXISTS aquaflow.etl_jobs_backup AS SELECT * FROM aquaflow.etl_jobs;
CREATE TABLE IF NOT EXISTS aquaflow.etl_job_templates_backup AS SELECT * FROM aquaflow.etl_job_templates;
CREATE TABLE IF NOT EXISTS aquaflow.etl_job_logs_backup AS SELECT * FROM aquaflow.etl_job_logs;

-- Log the cleanup operation
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
    'Starting cleanup of old ETL architecture',
    json_build_object(
        'cleanup_date', NOW(),
        'old_jobs_count', (SELECT COUNT(*) FROM aquaflow.etl_jobs),
        'old_templates_count', (SELECT COUNT(*) FROM aquaflow.etl_job_templates),
        'old_logs_count', (SELECT COUNT(*) FROM aquaflow.etl_job_logs)
    ),
    'cleanup'
WHERE EXISTS (SELECT 1 FROM aquaflow.etl_job_runs);

-- =====================================================
-- PHASE 1: DISABLE OLD SCHEDULER (if still running)
-- =====================================================

-- Mark old job templates as inactive to prevent new job creation
UPDATE aquaflow.etl_job_templates SET is_active = false;

-- Update old jobs to prevent workers from picking them up
UPDATE aquaflow.etl_jobs 
SET status = 'archived'
WHERE status IN ('pending', 'queued');

-- =====================================================
-- PHASE 2: CLEAN UP OLD JOBS TABLE
-- =====================================================

-- Delete old job instances (keeping some for reference if needed)
DELETE FROM aquaflow.etl_jobs 
WHERE batch_id IN (
    -- Delete jobs that have been migrated to runs
    SELECT j.batch_id 
    FROM aquaflow.etl_jobs j
    WHERE j.is_template_instance = true
      AND j.completed_at < NOW() - INTERVAL '7 days'
);

-- Clean up old manual jobs that are completed
DELETE FROM aquaflow.etl_jobs 
WHERE (is_template_instance IS NULL OR is_template_instance = false)
  AND status IN ('completed', 'failed', 'completed_with_errors')
  AND completed_at < NOW() - INTERVAL '7 days';

-- =====================================================
-- PHASE 3: CLEAN UP OLD LOGS
-- =====================================================

-- Delete old logs that have been migrated (keep recent ones for reference)
DELETE FROM aquaflow.etl_job_logs 
WHERE timestamp < NOW() - INTERVAL '7 days'
  AND batch_id NOT IN (
    -- Keep logs for jobs that still exist
    SELECT batch_id FROM aquaflow.etl_jobs
  );

-- =====================================================
-- PHASE 4: CREATE CLEANUP REPORT
-- =====================================================

CREATE TEMP VIEW cleanup_report AS
SELECT 
    'Old Jobs Remaining' as metric,
    COUNT(*) as count
FROM aquaflow.etl_jobs
UNION ALL
SELECT 
    'Old Templates Remaining' as metric,
    COUNT(*) as count
FROM aquaflow.etl_job_templates
UNION ALL
SELECT 
    'Old Logs Remaining' as metric,
    COUNT(*) as count
FROM aquaflow.etl_job_logs
UNION ALL
SELECT 
    'New Jobs Active' as metric,
    COUNT(*) as count
FROM aquaflow.etl_jobs_v2
WHERE is_active = true
UNION ALL
SELECT 
    'New Schedules Active' as metric,
    COUNT(*) as count
FROM aquaflow.etl_schedules
WHERE is_active = true
UNION ALL
SELECT 
    'New Runs Total' as metric,
    COUNT(*) as count
FROM aquaflow.etl_job_runs
UNION ALL
SELECT 
    'New Logs Total' as metric,
    COUNT(*) as count
FROM aquaflow.etl_job_logs_v2;

-- Display cleanup report
SELECT * FROM cleanup_report ORDER BY metric;

-- =====================================================
-- PHASE 5: UPDATE WORKER TO USE NEW ARCHITECTURE ONLY
-- =====================================================

-- Create a view to show migration status
CREATE OR REPLACE VIEW aquaflow.vw_architecture_status AS
SELECT 
    'Migration Status' as check_type,
    CASE 
        WHEN new_jobs > 0 AND new_schedules > 0 AND new_runs > 0 THEN 'NEW_ARCHITECTURE_ACTIVE'
        ELSE 'MIGRATION_INCOMPLETE'
    END as status,
    json_build_object(
        'new_jobs', new_jobs,
        'new_schedules', new_schedules,
        'new_runs', new_runs,
        'new_logs', new_logs,
        'old_jobs_remaining', old_jobs,
        'old_templates_remaining', old_templates,
        'old_logs_remaining', old_logs
    ) as details
FROM (
    SELECT 
        (SELECT COUNT(*) FROM aquaflow.etl_jobs_v2 WHERE is_active = true) as new_jobs,
        (SELECT COUNT(*) FROM aquaflow.etl_schedules WHERE is_active = true) as new_schedules,
        (SELECT COUNT(*) FROM aquaflow.etl_job_runs) as new_runs,
        (SELECT COUNT(*) FROM aquaflow.etl_job_logs_v2) as new_logs,
        (SELECT COUNT(*) FROM aquaflow.etl_jobs) as old_jobs,
        (SELECT COUNT(*) FROM aquaflow.etl_job_templates) as old_templates,
        (SELECT COUNT(*) FROM aquaflow.etl_job_logs) as old_logs
) counts;

-- Show final status
SELECT * FROM aquaflow.vw_architecture_status;

-- =====================================================
-- PHASE 6: LOG COMPLETION
-- =====================================================

INSERT INTO aquaflow.etl_job_logs_v2 (
    run_id, 
    timestamp, 
    log_level, 
    message, 
    context, 
    component
) 
SELECT 
    (SELECT run_id FROM aquaflow.etl_job_runs ORDER BY started_at DESC LIMIT 1),
    NOW(),
    'INFO',
    'Cleanup of old ETL architecture completed',
    json_build_object(
        'cleanup_completed_date', NOW(),
        'old_jobs_remaining', (SELECT COUNT(*) FROM aquaflow.etl_jobs),
        'old_templates_remaining', (SELECT COUNT(*) FROM aquaflow.etl_job_templates),
        'old_logs_remaining', (SELECT COUNT(*) FROM aquaflow.etl_job_logs),
        'new_architecture_status', 'ACTIVE'
    ),
    'cleanup'
WHERE EXISTS (SELECT 1 FROM aquaflow.etl_job_runs);

-- =====================================================
-- OPTIONAL PHASE 7: DROP OLD TABLES (COMMENTED OUT FOR SAFETY)
-- =====================================================

-- WARNING: Only run these commands if you're 100% sure the new architecture is working
-- and you have confirmed backups

-- -- Drop old tables (UNCOMMENT ONLY IF CERTAIN)
-- -- DROP TABLE IF EXISTS aquaflow.etl_job_logs;
-- -- DROP TABLE IF EXISTS aquaflow.etl_jobs;
-- -- DROP TABLE IF EXISTS aquaflow.etl_job_templates;

-- -- Drop old indexes
-- -- DROP INDEX IF EXISTS aquaflow.idx_etl_job_logs_batch_timestamp;
-- -- DROP INDEX IF EXISTS aquaflow.idx_etl_job_logs_timestamp;
-- -- DROP INDEX IF EXISTS aquaflow.idx_etl_jobs_status;
-- -- DROP INDEX IF EXISTS aquaflow.idx_etl_jobs_next_run;

COMMENT ON SCHEMA aquaflow IS 'AquaFlow Analytics ETL Schema - Cleaned up old architecture on ' || NOW();