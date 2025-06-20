-- =====================================================
-- WATER-SPECIFIC TABLES
-- =====================================================

-- SCADA Integration Mappings
CREATE TABLE aquaflow.scada_mappings (
    mapping_id SERIAL PRIMARY KEY,
    scada_tag VARCHAR(255) NOT NULL UNIQUE,
    series_id INTEGER REFERENCES aquaflow.series(series_id),
    scale_factor NUMERIC DEFAULT 1.0,
    value_offset NUMERIC DEFAULT 0.0,
    last_value NUMERIC,
    last_update TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Query Templates for common questions
CREATE TABLE aquaflow.query_templates (
    template_id SERIAL PRIMARY KEY,
    template_name VARCHAR(255) NOT NULL,
    template_key VARCHAR(100) NOT NULL UNIQUE,
    query_pattern TEXT[],
    sql_template TEXT NOT NULL,
    response_template TEXT,
    parameters JSONB,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Query Response Cache
CREATE TABLE aquaflow.query_cache (
    cache_id SERIAL PRIMARY KEY,
    query_text TEXT NOT NULL,
    query_hash VARCHAR(64) NOT NULL,
    response JSONB NOT NULL,
    response_time_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    hit_count INTEGER DEFAULT 0
);

-- Simple ETL tracking for demo data
CREATE TABLE aquaflow.etl_jobs (
    batch_id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    job_name VARCHAR(255),
    load_type VARCHAR(50) DEFAULT 'demo',
    status VARCHAR(50) DEFAULT 'queued',
    records_processed INTEGER DEFAULT 0,
    records_failed INTEGER DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    notes TEXT
);