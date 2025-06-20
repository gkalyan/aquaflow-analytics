-- =====================================================
-- INDEXES
-- =====================================================

-- Core indexes
CREATE INDEX idx_series_hash ON aquaflow.series(series_hash);
CREATE INDEX idx_dimensions_normalized ON aquaflow.dimensions(dimension_name_normalized, dimension_value_normalized);
CREATE UNIQUE INDEX dimensions_normalized_unique_idx ON aquaflow.dimensions(dimension_name_normalized, dimension_value_normalized);
CREATE INDEX idx_parameters_normalized ON aquaflow.parameters(parameter_name_normalized);
CREATE INDEX idx_series_metadata_key ON aquaflow.series_metadata(metadata_key);

-- Water-specific indexes
CREATE INDEX idx_scada_active ON aquaflow.scada_mappings(is_active) WHERE is_active = true;
CREATE INDEX idx_query_cache_hash ON aquaflow.query_cache(query_hash);
CREATE INDEX idx_query_cache_expires ON aquaflow.query_cache(expires_at);