-- =====================================================
-- ETL JOB TEMPLATES AND SCHEDULING
-- =====================================================

-- Create job templates table for recurring scheduled jobs
CREATE TABLE IF NOT EXISTS aquaflow.etl_job_templates (
    template_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name VARCHAR(255) NOT NULL UNIQUE,
    job_type VARCHAR(50) NOT NULL,
    load_type VARCHAR(50) DEFAULT 'scheduled',
    schedule VARCHAR(100) NOT NULL, -- Cron expression
    parameters JSONB NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    next_run TIMESTAMP WITH TIME ZONE,
    last_created_job_id UUID,
    created_job_count INTEGER DEFAULT 0
);

-- Add template reference to existing etl_jobs table
ALTER TABLE aquaflow.etl_jobs 
ADD COLUMN IF NOT EXISTS template_id UUID REFERENCES aquaflow.etl_job_templates(template_id),
ADD COLUMN IF NOT EXISTS is_template_instance BOOLEAN DEFAULT false;

-- Create indexes for efficient querying
CREATE INDEX idx_etl_job_templates_next_run ON aquaflow.etl_job_templates(next_run) WHERE is_active = true AND next_run IS NOT NULL;
CREATE INDEX idx_etl_job_templates_active ON aquaflow.etl_job_templates(is_active);
CREATE INDEX idx_etl_jobs_template_id ON aquaflow.etl_jobs(template_id);

-- Insert sample job templates
INSERT INTO aquaflow.etl_job_templates (template_name, job_type, schedule, parameters, next_run) VALUES 
(
    'Real-time Data Sync',
    'realtime_sync',
    '*/15 * * * *', -- Every 15 minutes
    '{
        "source_url": "http://demo-data-service:8090/api/realtime",
        "series_ids": [1,2,3,4,5,6,7,8,9,10,11,12],
        "sync_interval": 30
    }'::jsonb,
    NOW() + INTERVAL '1 minute' -- Start in 1 minute
),
(
    'Hourly Flow Sync',
    'realtime_sync', 
    '0 * * * *', -- Every hour
    '{
        "source_url": "http://demo-data-service:8090/api/realtime",
        "series_ids": [1,2,3,4,5,6,7,8,9,10,11,12],
        "sync_interval": 30
    }'::jsonb,
    NOW() + INTERVAL '5 minutes' -- Start in 5 minutes
),
(
    'Weekly Infrastructure Check',
    'historical_load',
    '0 2 * * 0', -- Every Sunday at 2 AM
    '{
        "source_url": "http://demo-data-service:8090/api/historical",
        "start_date": "DYNAMIC_WEEK_START",
        "end_date": "DYNAMIC_WEEK_END", 
        "series_ids": [8,9,10,11,12],
        "batch_size": 1000
    }'::jsonb,
    NOW() + INTERVAL '10 minutes' -- Start in 10 minutes for testing
),
(
    'Daily System Health Check',
    'historical_load',
    '0 6 * * *', -- Every day at 6 AM
    '{
        "source_url": "http://demo-data-service:8090/api/historical",
        "start_date": "DYNAMIC_DAY_START",
        "end_date": "DYNAMIC_DAY_END",
        "series_ids": [1,2,3,4,5,6,7],
        "batch_size": 500
    }'::jsonb,
    NOW() + INTERVAL '15 minutes' -- Start in 15 minutes for testing
);

-- Update existing jobs to mark them as non-template instances  
UPDATE aquaflow.etl_jobs SET is_template_instance = false WHERE template_id IS NULL;