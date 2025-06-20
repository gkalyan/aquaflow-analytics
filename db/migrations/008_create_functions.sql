-- =====================================================
-- HELPER FUNCTIONS
-- =====================================================

-- Normalize text for case-insensitive lookups
CREATE OR REPLACE FUNCTION aquaflow.normalize_text(input_text TEXT) 
RETURNS TEXT AS $$
BEGIN
    RETURN LOWER(TRIM(input_text));
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Generate series hash from parameter and dimensions
CREATE OR REPLACE FUNCTION aquaflow.generate_series_hash(p_parameter_id INTEGER, p_dimension_ids INTEGER[]) 
RETURNS TEXT AS $$
DECLARE
    hash_string TEXT := '';
    dim_id INTEGER;
    dim_value TEXT;
    parameter_name TEXT;
BEGIN
    -- Get normalized parameter name
    SELECT parameter_name_normalized INTO parameter_name
    FROM aquaflow.parameters 
    WHERE parameter_id = p_parameter_id;
    
    -- Start with parameter name
    hash_string := COALESCE(parameter_name, '') || '_';
    
    -- Sort dimension IDs for consistent hash generation
    SELECT array_agg(id ORDER BY id) INTO p_dimension_ids 
    FROM unnest(p_dimension_ids) AS id;
    
    -- Concatenate normalized dimension values
    FOREACH dim_id IN ARRAY p_dimension_ids
    LOOP
        SELECT dimension_value_normalized INTO dim_value
        FROM aquaflow.dimensions 
        WHERE dimension_id = dim_id;
        
        hash_string := hash_string || COALESCE(dim_value, '') || '_';
    END LOOP;
    
    -- Return SHA256 hash
    RETURN encode(digest(hash_string, 'sha256'), 'hex');
END;
$$ LANGUAGE plpgsql;

-- Get display name for series (generated, not stored)
CREATE OR REPLACE FUNCTION aquaflow.get_series_display_name(p_series_id INTEGER)
RETURNS TEXT AS $$
DECLARE
    display_name TEXT;
    param_name TEXT;
    dim_values TEXT[];
BEGIN
    -- Get parameter name
    SELECT p.parameter_name INTO param_name
    FROM aquaflow.series s
    JOIN aquaflow.parameters p ON s.parameter_id = p.parameter_id
    WHERE s.series_id = p_series_id;
    
    -- Get dimension values
    SELECT ARRAY_AGG(d.dimension_value ORDER BY d.dimension_name)
    INTO dim_values
    FROM aquaflow.series_dimensions sd
    JOIN aquaflow.dimensions d ON sd.dimension_id = d.dimension_id
    WHERE sd.series_id = p_series_id;
    
    -- Combine into display name
    display_name := param_name || ' at ' || array_to_string(dim_values, ', ');
    
    RETURN display_name;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Get latest value with status
CREATE OR REPLACE FUNCTION aquaflow.get_series_status(p_series_id INTEGER)
RETURNS TABLE(
    value NUMERIC,
    time_point TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20),
    deviation_pct NUMERIC
) AS $$
DECLARE
    threshold_min NUMERIC;
    threshold_max NUMERIC;
BEGIN
    -- Get thresholds from metadata
    SELECT 
        (metadata_value)::NUMERIC,
        (SELECT metadata_value FROM aquaflow.series_metadata 
         WHERE series_id = p_series_id AND metadata_key = 'threshold_normal_max')::NUMERIC
    INTO threshold_min, threshold_max
    FROM aquaflow.series_metadata
    WHERE series_id = p_series_id AND metadata_key = 'threshold_normal_min';
    
    -- Get latest value with status
    RETURN QUERY
    SELECT 
        nv.value,
        nv.time_point,
        CASE 
            WHEN threshold_min IS NULL OR threshold_max IS NULL THEN 'unknown'
            WHEN nv.value < threshold_min THEN 'low'
            WHEN nv.value > threshold_max THEN 'high'
            ELSE 'normal'
        END as status,
        CASE 
            WHEN threshold_min IS NOT NULL AND nv.value < threshold_min 
            THEN ((nv.value - threshold_min) / threshold_min * 100)
            WHEN threshold_max IS NOT NULL AND nv.value > threshold_max 
            THEN ((nv.value - threshold_max) / threshold_max * 100)
            ELSE 0
        END as deviation_pct
    FROM aquaflow.numeric_values nv
    WHERE nv.series_id = p_series_id
    ORDER BY nv.time_point DESC
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;