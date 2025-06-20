-- =====================================================
-- VALUE TABLES (Time-Series Data)
-- =====================================================

-- Numeric values (main data table)
CREATE TABLE aquaflow.numeric_values (
    series_id INTEGER NOT NULL REFERENCES aquaflow.series(series_id),
    time_point TIMESTAMP WITH TIME ZONE NOT NULL,
    value NUMERIC NOT NULL,
    quality_code CHAR(1) DEFAULT 'G', -- G=Good, Q=Questionable, B=Bad
    version INTEGER DEFAULT 1 NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    import_batch_id UUID,
    PRIMARY KEY (series_id, time_point, version)
);

-- Text values
CREATE TABLE aquaflow.text_values (
    series_id INTEGER NOT NULL REFERENCES aquaflow.series(series_id),
    time_point TIMESTAMP WITH TIME ZONE NOT NULL,
    value TEXT NOT NULL,
    version INTEGER DEFAULT 1 NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    import_batch_id UUID,
    PRIMARY KEY (series_id, time_point, version)
);

-- Boolean values
CREATE TABLE aquaflow.boolean_values (
    series_id INTEGER NOT NULL REFERENCES aquaflow.series(series_id),
    time_point TIMESTAMP WITH TIME ZONE NOT NULL,
    value BOOLEAN NOT NULL,
    version INTEGER DEFAULT 1 NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    import_batch_id UUID,
    PRIMARY KEY (series_id, time_point, version)
);