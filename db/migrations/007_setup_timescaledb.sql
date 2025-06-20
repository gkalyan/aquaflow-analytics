-- =====================================================
-- TIMESCALEDB SETUP
-- =====================================================

-- Convert value tables to hypertables BEFORE inserting any data
SELECT create_hypertable('aquaflow.numeric_values', 'time_point', chunk_time_interval => INTERVAL '7 days');
SELECT create_hypertable('aquaflow.text_values', 'time_point', chunk_time_interval => INTERVAL '7 days');
SELECT create_hypertable('aquaflow.boolean_values', 'time_point', chunk_time_interval => INTERVAL '7 days');

-- Create indexes on hypertables
CREATE INDEX idx_numeric_values_series_time ON aquaflow.numeric_values (series_id, time_point DESC);
CREATE INDEX idx_text_values_series_time ON aquaflow.text_values (series_id, time_point DESC);
CREATE INDEX idx_boolean_values_series_time ON aquaflow.boolean_values (series_id, time_point DESC);