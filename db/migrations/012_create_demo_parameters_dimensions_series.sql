-- =====================================================
-- CREATE MISSING PARAMETERS, DIMENSIONS, AND SERIES FOR DEMO DATA
-- =====================================================

-- First, add missing parameters with normalized names
INSERT INTO aquaflow.parameters (parameter_name, parameter_name_normalized, unit, description, parameter_type) VALUES
('pH Level', 'ph_level', 'pH', 'Water acidity/alkalinity measurement', 'numeric'),
('Turbidity', 'turbidity', 'NTU', 'Water clarity measurement in Nephelometric Turbidity Units', 'numeric'),
('Operational Status', 'operational_status', 'boolean', 'Equipment operational state (on/off, active/inactive)', 'boolean');

-- Add specific location dimensions for our demo series
INSERT INTO aquaflow.dimensions (dimension_name, dimension_value, dimension_name_normalized, dimension_value_normalized) VALUES
-- Main Canal specific locations
('Location', 'Main Canal Mile 8', 'location', 'main_canal_mile_8'),
('Location', 'Gate 12', 'location', 'gate_12'),

-- Pump station specific locations  
('Location', 'Pump Station 3', 'location', 'pump_station_3'),

-- Reservoir specific locations
('Location', 'Don Pedro Reservoir', 'location', 'don_pedro_reservoir'),
('Location', 'Don Pedro Intake', 'location', 'don_pedro_intake'),

-- Lateral canal locations
('Location', 'North Branch Canal', 'location', 'north_branch_canal'),

-- System-wide location for efficiency monitoring
('Location', 'System Wide', 'location', 'system_wide');

-- Now create the 12 series with proper parameter + dimension combinations
INSERT INTO aquaflow.series (dataset_id, parameter_id, series_hash, description) VALUES
-- Series 1: Main Canal Flow Rate at Mile 8
(1, 1, 'main_canal_flow_mile8', 'Main Canal Flow Rate at Mile 8 - Primary flow measurement'),

-- Series 2: Don Pedro Reservoir Water Level  
(2, 2, 'don_pedro_level', 'Don Pedro Reservoir Water Level - Daily operational level'),

-- Series 3: Pump Station 3 Pressure
(3, 3, 'pump_station_3_pressure', 'Pump Station 3 Operating Pressure - Critical operational metric'),

-- Series 4: Main Canal Temperature at Mile 8
(1, 8, 'main_canal_temp_mile8', 'Main Canal Water Temperature at Mile 8 - Environmental monitoring'),

-- Series 5: Gate 12 Position
(1, 4, 'gate_12_position', 'Gate 12 Position Control - Flow regulation'),

-- Series 6: Pump Station 1 Flow Rate
(3, 1, 'pump_station_1_flow', 'Pump Station 1 Flow Rate - Secondary distribution'),

-- Series 7: North Branch Canal Flow Rate
(4, 1, 'north_branch_flow', 'North Branch Canal Flow Rate - Distribution monitoring'),

-- Series 8: Main Canal pH Level (using new pH parameter)
(1, (SELECT parameter_id FROM aquaflow.parameters WHERE parameter_name = 'pH Level'), 'main_canal_ph_mile8', 'Main Canal pH Level at Mile 8 - Water quality monitoring'),

-- Series 9: Pump Station 2 Operational Status (using new status parameter)
(3, (SELECT parameter_id FROM aquaflow.parameters WHERE parameter_name = 'Operational Status'), 'pump_station_2_status', 'Pump Station 2 Operational Status - Equipment state monitoring'),

-- Series 10: Reservoir Inflow at Don Pedro Intake
(2, 1, 'don_pedro_inflow', 'Don Pedro Reservoir Inflow Rate - Source water monitoring'),

-- Series 11: System-wide Efficiency
(1, 5, 'system_efficiency', 'System-wide Operational Efficiency - Performance metric'),

-- Series 12: Main Canal Turbidity (using new turbidity parameter)
(1, (SELECT parameter_id FROM aquaflow.parameters WHERE parameter_name = 'Turbidity'), 'main_canal_turbidity_mile8', 'Main Canal Turbidity at Mile 8 - Water quality monitoring');

-- Now link each series to its appropriate dimensions
INSERT INTO aquaflow.series_dimensions (series_id, dimension_id) VALUES
-- Series 1: Main Canal Flow at Mile 8
(1, (SELECT dimension_id FROM aquaflow.dimensions WHERE dimension_value = 'Main Canal Mile 8')),

-- Series 2: Don Pedro Reservoir Level
(2, (SELECT dimension_id FROM aquaflow.dimensions WHERE dimension_value = 'Don Pedro Reservoir')),

-- Series 3: Pump Station 3 Pressure  
(3, (SELECT dimension_id FROM aquaflow.dimensions WHERE dimension_value = 'Pump Station 3')),

-- Series 4: Main Canal Temperature at Mile 8
(4, (SELECT dimension_id FROM aquaflow.dimensions WHERE dimension_value = 'Main Canal Mile 8')),

-- Series 5: Gate 12 Position
(5, (SELECT dimension_id FROM aquaflow.dimensions WHERE dimension_value = 'Gate 12')),

-- Series 6: Pump Station 1 Flow
(6, (SELECT dimension_id FROM aquaflow.dimensions WHERE dimension_value = 'Pump Station 1')),

-- Series 7: North Branch Canal Flow
(7, (SELECT dimension_id FROM aquaflow.dimensions WHERE dimension_value = 'North Branch Canal')),

-- Series 8: Main Canal pH at Mile 8
(8, (SELECT dimension_id FROM aquaflow.dimensions WHERE dimension_value = 'Main Canal Mile 8')),

-- Series 9: Pump Station 2 Status
(9, (SELECT dimension_id FROM aquaflow.dimensions WHERE dimension_value = 'Pump Station 2')),

-- Series 10: Don Pedro Inflow
(10, (SELECT dimension_id FROM aquaflow.dimensions WHERE dimension_value = 'Don Pedro Intake')),

-- Series 11: System-wide Efficiency
(11, (SELECT dimension_id FROM aquaflow.dimensions WHERE dimension_value = 'System Wide')),

-- Series 12: Main Canal Turbidity at Mile 8
(12, (SELECT dimension_id FROM aquaflow.dimensions WHERE dimension_value = 'Main Canal Mile 8'));

-- Verification query: Show the complete series structure
SELECT 
  s.series_id,
  d.dataset_name,
  p.parameter_name,
  p.unit,
  dim.dimension_name || ': ' || dim.dimension_value AS location,
  s.description
FROM aquaflow.series s
JOIN aquaflow.datasets d ON s.dataset_id = d.dataset_id
JOIN aquaflow.parameters p ON s.parameter_id = p.parameter_id
LEFT JOIN aquaflow.series_dimensions sd ON s.series_id = sd.series_id
LEFT JOIN aquaflow.dimensions dim ON sd.dimension_id = dim.dimension_id
ORDER BY s.series_id;