-- Set search path to aquaflow schema
SET search_path TO aquaflow, public;

-- =====================================================
-- INITIAL DATA FOR WATER OPERATIONS
-- =====================================================

-- Insert sample datasets
INSERT INTO aquaflow.datasets (dataset_name, description, system_type) VALUES
('Main Canal', 'Primary water distribution canal', 'canal'),
('Don Pedro Reservoir', 'Main storage reservoir', 'reservoir'),
('Pump Stations', 'Booster pump stations', 'pump_station'),
('Lateral Canals', 'Secondary distribution canals', 'canal'),
('USGS Monitoring', 'External river gauge data', 'external'),
('NWS Weather', 'Weather station data', 'external');

-- Insert parameters
INSERT INTO aquaflow.parameters (parameter_name, unit, parameter_type, parameter_name_normalized) VALUES
('Flow Rate', 'CFS', 'numeric', aquaflow.normalize_text('Flow Rate')),
('Water Level', 'feet', 'numeric', aquaflow.normalize_text('Water Level')),
('Pressure', 'PSI', 'numeric', aquaflow.normalize_text('Pressure')),
('Gate Position', '%', 'numeric', aquaflow.normalize_text('Gate Position')),
('Efficiency', '%', 'numeric', aquaflow.normalize_text('Efficiency')),
('Pump Status', 'RPM', 'numeric', aquaflow.normalize_text('Pump Status')),
('River Discharge', 'CFS', 'numeric', aquaflow.normalize_text('River Discharge')),
('Temperature', '°F', 'numeric', aquaflow.normalize_text('Temperature')),
('Precipitation', 'inches', 'numeric', aquaflow.normalize_text('Precipitation')),
('Humidity', '%', 'numeric', aquaflow.normalize_text('Humidity')),
('ET0', 'inches/day', 'numeric', aquaflow.normalize_text('ET0'));

-- Insert dimensions (locations)
INSERT INTO aquaflow.dimensions (dimension_name, dimension_value, dimension_name_normalized, dimension_value_normalized, metadata) VALUES
('Location', 'Mile 0', aquaflow.normalize_text('Location'), aquaflow.normalize_text('Mile 0'), '{"type": "canal_point"}'),
('Location', 'Mile 6', aquaflow.normalize_text('Location'), aquaflow.normalize_text('Mile 6'), '{"type": "canal_point"}'),
('Location', 'Mile 12', aquaflow.normalize_text('Location'), aquaflow.normalize_text('Mile 12'), '{"type": "canal_point"}'),
('Location', 'Mile 18', aquaflow.normalize_text('Location'), aquaflow.normalize_text('Mile 18'), '{"type": "canal_point"}'),
('Location', 'Mile 24', aquaflow.normalize_text('Location'), aquaflow.normalize_text('Mile 24'), '{"type": "canal_point"}'),
('Location', 'Station 7', aquaflow.normalize_text('Location'), aquaflow.normalize_text('Station 7'), '{"type": "monitoring_station"}'),
('Location', 'Pump Station 1', aquaflow.normalize_text('Location'), aquaflow.normalize_text('Pump Station 1'), '{"type": "pump_station"}'),
('Location', 'Pump Station 2', aquaflow.normalize_text('Location'), aquaflow.normalize_text('Pump Station 2'), '{"type": "pump_station"}'),
('Location', 'Gate 4', aquaflow.normalize_text('Location'), aquaflow.normalize_text('Gate 4'), '{"type": "control_gate"}'),
('Location', 'Gate 7', aquaflow.normalize_text('Location'), aquaflow.normalize_text('Gate 7'), '{"type": "control_gate"}'),
('USGS Site', '11289650', aquaflow.normalize_text('USGS Site'), aquaflow.normalize_text('11289650'), '{"name": "Tuolumne R bl La Grange Dam"}'),
('NWS Station', 'KMOD', aquaflow.normalize_text('NWS Station'), aquaflow.normalize_text('KMOD'), '{"name": "Modesto City Airport"}');

-- Insert query templates
INSERT INTO aquaflow.query_templates (template_name, template_key, query_pattern, sql_template, response_template) VALUES
(
    'Morning Check',
    'morning_check',
    ARRAY['morning check', 'system status', 'overnight summary'],
    'WITH latest_values AS (
        SELECT s.series_id, 
               aquaflow.get_series_display_name(s.series_id) as name, 
               nv.value, 
               nv.time_point, 
               p.unit,
               CASE 
                   WHEN nv.value < (sm_min.metadata_value)::numeric THEN ''low''
                   WHEN nv.value > (sm_max.metadata_value)::numeric THEN ''high''
                   ELSE ''normal''
               END as status
        FROM aquaflow.series s
        JOIN aquaflow.parameters p ON s.parameter_id = p.parameter_id
        JOIN LATERAL (
            SELECT value, time_point 
            FROM aquaflow.numeric_values 
            WHERE series_id = s.series_id 
            ORDER BY time_point DESC 
            LIMIT 1
        ) nv ON true
        LEFT JOIN aquaflow.series_metadata sm_min ON s.series_id = sm_min.series_id 
            AND sm_min.metadata_key = ''threshold_normal_min''
        LEFT JOIN aquaflow.series_metadata sm_max ON s.series_id = sm_max.series_id 
            AND sm_max.metadata_key = ''threshold_normal_max''
        WHERE nv.time_point > NOW() - INTERVAL ''24 hours''
    )
    SELECT * FROM latest_values;',
    'System check complete. {anomaly_count} parameters outside normal range.'
),
(
    'Efficiency Report',
    'efficiency_report',
    ARRAY['efficiency', 'performance', 'system efficiency'],
    'SELECT 
        AVG(nv.value) as avg_efficiency,
        MIN(nv.value) as min_efficiency,
        MAX(nv.value) as max_efficiency
     FROM aquaflow.numeric_values nv
     JOIN aquaflow.series s ON nv.series_id = s.series_id
     JOIN aquaflow.parameters p ON s.parameter_id = p.parameter_id
     WHERE p.parameter_name = ''Efficiency''
     AND nv.time_point > NOW() - INTERVAL ''7 days''',
    'System efficiency this week: {avg_efficiency}% (Range: {min_efficiency}% - {max_efficiency}%)'
);