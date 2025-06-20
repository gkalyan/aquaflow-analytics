# Database Migrations

## Current Status

**NOTE**: The aquaflow schema and all tables have been manually created and are already present in the database. These migration files are provided for reference and documentation purposes only.

## Migration Files

1. **001_create_schema.sql** - Creates aquaflow schema and enables extensions
2. **002_create_types.sql** - Custom enum types
3. **003_create_core_tables.sql** - Core time-series tables (datasets, parameters, dimensions, series)
4. **004_create_value_tables.sql** - Value tables (numeric_values, text_values, boolean_values)
5. **005_create_water_tables.sql** - Water-specific tables (scada_mappings, query_templates, etc.)
6. **006_create_indexes.sql** - Database indexes for performance
7. **007_setup_timescaledb.sql** - TimescaleDB hypertables setup
8. **008_create_functions.sql** - Helper functions for data processing
9. **009_initial_data.sql** - Sample data for development

## Usage

Since the schema already exists, these migrations are **NOT** automatically run by Docker. To run them manually if needed:

```bash
# Connect to the database
docker-compose exec timescaledb psql -U aquaflow -d aquaflowdb

# Run a specific migration (if needed)
\i /path/to/migration/file.sql
```

## Schema Verification

To verify the current schema exists:

```sql
-- Check if aquaflow schema exists
SELECT schema_name FROM information_schema.schemata WHERE schema_name = 'aquaflow';

-- List all tables in aquaflow schema
SELECT table_name FROM information_schema.tables WHERE table_schema = 'aquaflow';

-- Check hypertables
SELECT * FROM timescaledb_information.hypertables WHERE schema_name = 'aquaflow';
```