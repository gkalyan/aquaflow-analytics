-- =====================================================
-- CREATE DEMO SERIES FOR ETL TESTING
-- =====================================================

-- Create series that match the demo data service expectations
-- Series 1-12 must exist for ETL workers to insert data

INSERT INTO aquaflow.series (dataset_id, parameter_id, series_hash, description) VALUES
-- Series 1: Main Canal Flow Rate (Main Canal + Flow Rate)
(1, 1, 'main_canal_flow', 'Main Canal Flow Rate - Primary water flow measurement'),

-- Series 2: Don Pedro Reservoir Level (Don Pedro Reservoir + Water Level)  
(2, 2, 'don_pedro_level', 'Don Pedro Reservoir Water Level - Daily operational level'),

-- Series 3: Pump Station 3 Pressure (Pump Stations + Pressure)
(3, 3, 'pump_station_3_pressure', 'Pump Station 3 Operating Pressure - Critical operational metric'),

-- Series 4: Main Canal Temperature (Main Canal + Temperature)
(1, 8, 'main_canal_temp', 'Main Canal Water Temperature - Environmental monitoring'),

-- Series 5: Gate 12 Position (Main Canal + Gate Position)
(1, 4, 'gate_12_position', 'Gate 12 Position Control - Flow regulation'),

-- Series 6: Pump Station 1 Flow (Pump Stations + Flow Rate)
(3, 1, 'pump_station_1_flow', 'Pump Station 1 Flow Rate - Secondary distribution'),

-- Series 7: North Branch Flow (Lateral Canals + Flow Rate)  
(4, 1, 'north_branch_flow', 'North Branch Canal Flow Rate - Distribution monitoring'),

-- Series 8: Water Quality pH (Main Canal + custom pH parameter - we'll use efficiency as placeholder)
(1, 5, 'water_quality_ph', 'Water Quality pH Level - Chemical monitoring'),

-- Series 9: Pump Station 2 Status (Pump Stations + Pump Status)
(3, 6, 'pump_station_2_status', 'Pump Station 2 Operational Status - Boolean operational state'),

-- Series 10: Reservoir Inflow (Don Pedro Reservoir + Flow Rate)
(2, 1, 'reservoir_inflow', 'Don Pedro Reservoir Inflow Rate - Source water monitoring'),

-- Series 11: System Efficiency (Main Canal + Efficiency)
(1, 5, 'system_efficiency', 'Overall System Efficiency - Performance metric'),

-- Series 12: Turbidity Level (Main Canal + custom turbidity - we'll use humidity as placeholder)
(1, 10, 'turbidity_level', 'Water Turbidity Level - Quality monitoring');

-- Verify the series were created correctly
SELECT s.series_id, d.dataset_name, p.parameter_name, p.unit, s.description
FROM aquaflow.series s
JOIN aquaflow.datasets d ON s.dataset_id = d.dataset_id  
JOIN aquaflow.parameters p ON s.parameter_id = p.parameter_id
ORDER BY s.series_id;