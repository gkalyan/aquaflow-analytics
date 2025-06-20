-- =====================================================
-- CORE TABLES (Generic Time-Series)
-- =====================================================

-- Datasets (collections of series)
CREATE TABLE aquaflow.datasets (
    dataset_id SERIAL PRIMARY KEY,
    dataset_name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    system_type VARCHAR(50), -- 'canal', 'reservoir', 'pump_station', 'external'
    location_info JSONB,
    is_active BOOLEAN DEFAULT true,
    data_retention_days INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Parameters (what is being measured)
CREATE TABLE aquaflow.parameters (
    parameter_id SERIAL PRIMARY KEY,
    parameter_name VARCHAR(255) NOT NULL UNIQUE,
    parameter_name_normalized TEXT NOT NULL,
    description TEXT,
    unit VARCHAR(50),
    parameter_type aquaflow.parameter_data_type DEFAULT 'numeric' NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Dimensions (context - where/what is being measured)
CREATE TABLE aquaflow.dimensions (
    dimension_id SERIAL PRIMARY KEY,
    dimension_name VARCHAR(255) NOT NULL,
    dimension_value TEXT NOT NULL,
    dimension_name_normalized TEXT NOT NULL,
    dimension_value_normalized TEXT NOT NULL,
    parent_dimension_id INTEGER REFERENCES aquaflow.dimensions(dimension_id),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Series (combination of dataset + parameter + dimensions)
CREATE TABLE aquaflow.series (
    series_id SERIAL PRIMARY KEY,
    dataset_id INTEGER NOT NULL REFERENCES aquaflow.datasets(dataset_id),
    parameter_id INTEGER NOT NULL REFERENCES aquaflow.parameters(parameter_id),
    series_hash TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT series_must_have_hash CHECK (series_hash IS NOT NULL AND series_hash <> ''),
    CONSTRAINT series_dataset_parameter_hash_unique UNIQUE (dataset_id, parameter_id, series_hash)
);

-- Series-Dimensions junction table
CREATE TABLE aquaflow.series_dimensions (
    series_id INTEGER NOT NULL REFERENCES aquaflow.series(series_id) ON DELETE CASCADE,
    dimension_id INTEGER NOT NULL REFERENCES aquaflow.dimensions(dimension_id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (series_id, dimension_id)
);

-- Series metadata (for thresholds, etc.)
CREATE TABLE aquaflow.series_metadata (
    id SERIAL PRIMARY KEY,
    series_id INTEGER NOT NULL REFERENCES aquaflow.series(series_id) ON DELETE CASCADE,
    metadata_key VARCHAR(255) NOT NULL,
    metadata_value TEXT,
    data_type VARCHAR(50) DEFAULT 'text',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT series_metadata_series_key_unique UNIQUE (series_id, metadata_key)
);