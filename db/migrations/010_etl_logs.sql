-- =====================================================
-- ETL JOB LOGS AND UPDATES
-- =====================================================

-- Add new columns to etl_jobs table
ALTER TABLE aquaflow.etl_jobs 
ADD COLUMN IF NOT EXISTS job_type VARCHAR(50) NOT NULL DEFAULT 'historical_load',
ADD COLUMN IF NOT EXISTS parameters JSONB,
ADD COLUMN IF NOT EXISTS schedule VARCHAR(100),
ADD COLUMN IF NOT EXISTS next_run TIMESTAMP WITH TIME ZONE;

-- Create ETL job logs table
CREATE TABLE IF NOT EXISTS aquaflow.etl_job_logs (
    log_id SERIAL PRIMARY KEY,
    batch_id UUID REFERENCES aquaflow.etl_jobs(batch_id),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    log_level VARCHAR(10) NOT NULL CHECK (log_level IN ('DEBUG', 'INFO', 'WARN', 'ERROR')),
    message TEXT NOT NULL,
    context JSONB
);

-- Create indexes for efficient querying
CREATE INDEX idx_etl_job_logs_batch_timestamp ON aquaflow.etl_job_logs(batch_id, timestamp DESC);
CREATE INDEX idx_etl_job_logs_timestamp ON aquaflow.etl_job_logs(timestamp DESC);
CREATE INDEX idx_etl_jobs_status ON aquaflow.etl_jobs(status);
CREATE INDEX idx_etl_jobs_next_run ON aquaflow.etl_jobs(next_run) WHERE next_run IS NOT NULL;

-- Sample ETL job configurations
INSERT INTO aquaflow.etl_jobs (job_name, job_type, parameters, status) VALUES 
(
    'Historical Data Load - 2 Months', 
    'historical_load', 
    '{
        "source_url": "http://demo-data-service:8090/api/historical",
        "start_date": "2025-04-23",
        "end_date": "2025-06-23",
        "series_ids": [1,2,3,4,5,6,7,8,9,10,11,12],
        "batch_size": 1000
    }'::jsonb, 
    'pending'
),
(
    'Real-time Data Sync', 
    'realtime_sync',
    '{
        "source_url": "http://demo-data-service:8090/api/realtime",
        "series_ids": [1,2,3,4,5,6,7,8,9,10,11,12],
        "sync_interval": 30
    }'::jsonb, 
    'pending'
);