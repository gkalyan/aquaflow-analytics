-- Create aquaflow schema
CREATE SCHEMA IF NOT EXISTS aquaflow;
SET search_path TO aquaflow, public;

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
CREATE EXTENSION IF NOT EXISTS pgcrypto;