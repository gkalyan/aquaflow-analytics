-- Create custom types in aquaflow schema
CREATE TYPE aquaflow.parameter_data_type AS ENUM ('text', 'numeric', 'boolean', 'blob', 'media');