-- =====================================================
-- ELEGANT ETL ARCHITECTURE: JOBS → SCHEDULES → RUNS
-- =====================================================
-- This migration creates a three-tier architecture similar to Microsoft Power Automate:
-- 1. ETL Jobs: WHAT to run (job definitions)
-- 2. ETL Schedules: WHEN to run (schedule configurations) 
-- 3. ETL Job Runs: Execution instances and history
-- =====================================================

-- =====================================================
-- TIER 1: ETL JOBS (Job Definitions)
-- =====================================================
CREATE TABLE IF NOT EXISTS aquaflow.etl_jobs_v2 (
    job_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_name VARCHAR(255) NOT NULL UNIQUE,
    job_type VARCHAR(50) NOT NULL CHECK (job_type IN ('historical_load', 'realtime_sync', 'data_validation', 'cleanup')),
    description TEXT,
    parameters JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_by VARCHAR(100) DEFAULT 'system',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version INTEGER DEFAULT 1,
    tags TEXT[] DEFAULT ARRAY[]::TEXT[]
);

-- =====================================================
-- TIER 2: ETL SCHEDULES (When to Run)
-- =====================================================
CREATE TABLE IF NOT EXISTS aquaflow.etl_schedules (
    schedule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES aquaflow.etl_jobs_v2(job_id) ON DELETE CASCADE,
    schedule_name VARCHAR(255) NOT NULL,
    cron_expression VARCHAR(100) NOT NULL,
    timezone VARCHAR(50) DEFAULT 'UTC',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    next_run TIMESTAMP WITH TIME ZONE,
    last_run TIMESTAMP WITH TIME ZONE,
    run_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    UNIQUE(job_id, schedule_name)
);

-- =====================================================
-- TIER 3: ETL JOB RUNS (Execution History)
-- =====================================================
CREATE TABLE IF NOT EXISTS aquaflow.etl_job_runs (
    run_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES aquaflow.etl_jobs_v2(job_id),
    schedule_id UUID REFERENCES aquaflow.etl_schedules(schedule_id),
    run_name VARCHAR(255) NOT NULL, -- e.g., "Real-time Data Sync - 2025-06-24 06:30"
    status VARCHAR(50) DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'completed', 'failed', 'cancelled', 'completed_with_errors')),
    trigger_type VARCHAR(50) DEFAULT 'scheduled' CHECK (trigger_type IN ('scheduled', 'manual', 'retry', 'api')),
    
    -- Execution tracking
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER,
    
    -- Data processing metrics
    records_processed INTEGER DEFAULT 0,
    records_failed INTEGER DEFAULT 0,
    records_skipped INTEGER DEFAULT 0,
    
    -- Error handling
    error_message TEXT,
    error_category VARCHAR(50), -- 'transient', 'data', 'system', 'configuration'
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    
    -- Runtime parameters (can override job defaults)
    runtime_parameters JSONB,
    
    -- Execution context
    worker_id VARCHAR(100),
    execution_node VARCHAR(100),
    memory_used_mb INTEGER,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =====================================================
-- UPDATED ETL JOB LOGS (Linked to Runs)
-- =====================================================
-- Update existing logs table to link to runs instead of jobs
ALTER TABLE aquaflow.etl_job_logs 
ADD COLUMN IF NOT EXISTS run_id UUID REFERENCES aquaflow.etl_job_runs(run_id);

-- Create new optimized logs table for the new architecture
CREATE TABLE IF NOT EXISTS aquaflow.etl_job_logs_v2 (
    log_id BIGSERIAL PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES aquaflow.etl_job_runs(run_id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    log_level VARCHAR(10) NOT NULL CHECK (log_level IN ('DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL')),
    message TEXT NOT NULL,
    context JSONB,
    component VARCHAR(50), -- 'scheduler', 'worker', 'processor', 'validator'
    stack_trace TEXT,
    correlation_id UUID -- for tracing related log entries
);

-- =====================================================
-- INDEXES FOR PERFORMANCE
-- =====================================================

-- ETL Jobs indexes
CREATE INDEX idx_etl_jobs_v2_active ON aquaflow.etl_jobs_v2(is_active) WHERE is_active = true;
CREATE INDEX idx_etl_jobs_v2_type ON aquaflow.etl_jobs_v2(job_type);
CREATE INDEX idx_etl_jobs_v2_tags ON aquaflow.etl_jobs_v2 USING GIN(tags);

-- ETL Schedules indexes  
CREATE INDEX idx_etl_schedules_job_id ON aquaflow.etl_schedules(job_id);
CREATE INDEX idx_etl_schedules_next_run ON aquaflow.etl_schedules(next_run) WHERE is_active = true AND next_run IS NOT NULL;
CREATE INDEX idx_etl_schedules_active ON aquaflow.etl_schedules(is_active);

-- ETL Job Runs indexes
CREATE INDEX idx_etl_job_runs_job_id ON aquaflow.etl_job_runs(job_id);
CREATE INDEX idx_etl_job_runs_schedule_id ON aquaflow.etl_job_runs(schedule_id);
CREATE INDEX idx_etl_job_runs_status ON aquaflow.etl_job_runs(status);
CREATE INDEX idx_etl_job_runs_started_at ON aquaflow.etl_job_runs(started_at DESC);
CREATE INDEX idx_etl_job_runs_job_started ON aquaflow.etl_job_runs(job_id, started_at DESC);
CREATE INDEX idx_etl_job_runs_schedule_started ON aquaflow.etl_job_runs(schedule_id, started_at DESC);

-- ETL Job Logs indexes
CREATE INDEX idx_etl_job_logs_v2_run_id ON aquaflow.etl_job_logs_v2(run_id);
CREATE INDEX idx_etl_job_logs_v2_timestamp ON aquaflow.etl_job_logs_v2(timestamp DESC);
CREATE INDEX idx_etl_job_logs_v2_run_timestamp ON aquaflow.etl_job_logs_v2(run_id, timestamp DESC);
CREATE INDEX idx_etl_job_logs_v2_level ON aquaflow.etl_job_logs_v2(log_level);

-- =====================================================
-- TRIGGERS FOR AUTOMATIC UPDATES
-- =====================================================

-- Update timestamps automatically
CREATE OR REPLACE FUNCTION aquaflow.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply to all tables with updated_at
CREATE TRIGGER update_etl_jobs_v2_updated_at BEFORE UPDATE ON aquaflow.etl_jobs_v2 FOR EACH ROW EXECUTE FUNCTION aquaflow.update_updated_at_column();
CREATE TRIGGER update_etl_schedules_updated_at BEFORE UPDATE ON aquaflow.etl_schedules FOR EACH ROW EXECUTE FUNCTION aquaflow.update_updated_at_column();
CREATE TRIGGER update_etl_job_runs_updated_at BEFORE UPDATE ON aquaflow.etl_job_runs FOR EACH ROW EXECUTE FUNCTION aquaflow.update_updated_at_column();

-- Auto-calculate duration when run completes
CREATE OR REPLACE FUNCTION aquaflow.calculate_run_duration()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.completed_at IS NOT NULL AND OLD.completed_at IS NULL THEN
        NEW.duration_seconds = EXTRACT(EPOCH FROM (NEW.completed_at - NEW.started_at));
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER calculate_etl_run_duration BEFORE UPDATE ON aquaflow.etl_job_runs FOR EACH ROW EXECUTE FUNCTION aquaflow.calculate_run_duration();

-- =====================================================
-- SAMPLE DATA FOR NEW ARCHITECTURE
-- =====================================================

-- Insert job definitions
INSERT INTO aquaflow.etl_jobs_v2 (job_name, job_type, description, parameters, tags) VALUES 
(
    'Real-time Data Sync',
    'realtime_sync',
    'Synchronizes real-time data from external sources every 15 minutes',
    '{
        "source_url": "http://demo-data-service:8090/api/realtime",
        "series_ids": [1,2,3,4,5,6,7,8,9,10,11,12],
        "sync_interval": 30,
        "timeout_seconds": 300
    }'::jsonb,
    ARRAY['production', 'realtime', 'high-priority']
),
(
    'Hourly Flow Sync', 
    'realtime_sync',
    'Hourly synchronization of flow data for operational monitoring',
    '{
        "source_url": "http://demo-data-service:8090/api/realtime",
        "series_ids": [1,2,3,4,5,6,7,8,9,10,11,12],
        "sync_interval": 30,
        "batch_size": 1000
    }'::jsonb,
    ARRAY['production', 'monitoring']
),
(
    'Weekly Infrastructure Check',
    'historical_load',
    'Weekly load of infrastructure data for compliance reporting',
    '{
        "source_url": "http://demo-data-service:8090/api/historical", 
        "start_date": "DYNAMIC_WEEK_START",
        "end_date": "DYNAMIC_WEEK_END",
        "series_ids": [8,9,10,11,12],
        "batch_size": 1000
    }'::jsonb,
    ARRAY['compliance', 'weekly', 'infrastructure']
),
(
    'Daily System Health Check',
    'historical_load', 
    'Daily health check of all system parameters',
    '{
        "source_url": "http://demo-data-service:8090/api/historical",
        "start_date": "DYNAMIC_DAY_START", 
        "end_date": "DYNAMIC_DAY_END",
        "series_ids": [1,2,3,4,5,6,7],
        "batch_size": 500
    }'::jsonb,
    ARRAY['health-check', 'daily', 'system']
);

-- Insert schedule configurations
INSERT INTO aquaflow.etl_schedules (job_id, schedule_name, cron_expression, next_run)
SELECT 
    j.job_id,
    CASE 
        WHEN j.job_name = 'Real-time Data Sync' THEN 'Every 15 minutes'
        WHEN j.job_name = 'Hourly Flow Sync' THEN 'Every hour'
        WHEN j.job_name = 'Weekly Infrastructure Check' THEN 'Weekly on Sunday at 2 AM'
        WHEN j.job_name = 'Daily System Health Check' THEN 'Daily at 6 AM'
    END as schedule_name,
    CASE 
        WHEN j.job_name = 'Real-time Data Sync' THEN '*/15 * * * *'
        WHEN j.job_name = 'Hourly Flow Sync' THEN '0 * * * *'
        WHEN j.job_name = 'Weekly Infrastructure Check' THEN '0 2 * * 0'
        WHEN j.job_name = 'Daily System Health Check' THEN '0 6 * * *'
    END as cron_expression,
    CASE 
        WHEN j.job_name = 'Real-time Data Sync' THEN NOW() + INTERVAL '2 minutes'
        WHEN j.job_name = 'Hourly Flow Sync' THEN NOW() + INTERVAL '5 minutes'
        WHEN j.job_name = 'Weekly Infrastructure Check' THEN NOW() + INTERVAL '10 minutes'
        WHEN j.job_name = 'Daily System Health Check' THEN NOW() + INTERVAL '15 minutes'
    END as next_run
FROM aquaflow.etl_jobs_v2 j;

-- =====================================================
-- HELPFUL VIEWS FOR COMMON QUERIES
-- =====================================================

-- View: Current job status with latest run info
CREATE OR REPLACE VIEW aquaflow.vw_job_status AS
SELECT 
    j.job_id,
    j.job_name,
    j.job_type,
    j.is_active as job_active,
    s.schedule_id,
    s.schedule_name,
    s.cron_expression,
    s.next_run,
    s.is_active as schedule_active,
    lr.run_id as last_run_id,
    lr.status as last_run_status,
    lr.started_at as last_run_started,
    lr.completed_at as last_run_completed,
    lr.records_processed as last_run_records,
    lr.error_message as last_run_error
FROM aquaflow.etl_jobs_v2 j
LEFT JOIN aquaflow.etl_schedules s ON j.job_id = s.job_id
LEFT JOIN LATERAL (
    SELECT * FROM aquaflow.etl_job_runs r 
    WHERE r.job_id = j.job_id 
    ORDER BY r.started_at DESC 
    LIMIT 1
) lr ON true;

-- View: Recent runs with job context
CREATE OR REPLACE VIEW aquaflow.vw_recent_runs AS
SELECT 
    r.run_id,
    r.run_name,
    j.job_name,
    j.job_type,
    s.schedule_name,
    s.cron_expression,
    r.status,
    r.trigger_type,
    r.started_at,
    r.completed_at,
    r.duration_seconds,
    r.records_processed,
    r.records_failed,
    r.error_message,
    r.retry_count
FROM aquaflow.etl_job_runs r
JOIN aquaflow.etl_jobs_v2 j ON r.job_id = j.job_id
LEFT JOIN aquaflow.etl_schedules s ON r.schedule_id = s.schedule_id
ORDER BY r.started_at DESC;

-- =====================================================
-- CONSTRAINTS AND VALIDATION
-- =====================================================

-- Ensure run names are descriptive
ALTER TABLE aquaflow.etl_job_runs 
ADD CONSTRAINT chk_run_name_not_empty CHECK (LENGTH(TRIM(run_name)) > 0);

-- Ensure valid cron expressions (basic check)
ALTER TABLE aquaflow.etl_schedules
ADD CONSTRAINT chk_cron_expression_format CHECK (cron_expression ~ '^[*,/0-9-]+\s+[*,/0-9-]+\s+[*,/0-9-]+\s+[*,/0-9-]+\s+[*,/0-9-]+$');

-- Ensure duration is positive when set
ALTER TABLE aquaflow.etl_job_runs
ADD CONSTRAINT chk_positive_duration CHECK (duration_seconds IS NULL OR duration_seconds >= 0);

-- Ensure retry count doesn't exceed max retries
ALTER TABLE aquaflow.etl_job_runs
ADD CONSTRAINT chk_retry_count_valid CHECK (retry_count <= max_retries);

COMMENT ON TABLE aquaflow.etl_jobs_v2 IS 'Job definitions - WHAT to run';
COMMENT ON TABLE aquaflow.etl_schedules IS 'Schedule configurations - WHEN to run';  
COMMENT ON TABLE aquaflow.etl_job_runs IS 'Execution instances - actual run history';
COMMENT ON VIEW aquaflow.vw_job_status IS 'Current status of all jobs with latest run information';
COMMENT ON VIEW aquaflow.vw_recent_runs IS 'Recent job runs with full context';