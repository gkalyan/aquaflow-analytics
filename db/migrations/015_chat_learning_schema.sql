-- =====================================================
-- CHAT LEARNING SYSTEM SCHEMA
-- =====================================================
-- This migration creates tables for the chat-based query system
-- with local LLM learning capabilities
-- =====================================================

-- =====================================================
-- CHAT SESSIONS AND MESSAGES
-- =====================================================

-- Chat sessions for conversation management
CREATE TABLE IF NOT EXISTS aquaflow.chat_sessions (
    session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    message_count INTEGER DEFAULT 0,
    last_query TEXT,
    entity_mappings JSONB DEFAULT '{}',
    context JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true
);

-- Chat messages within sessions
CREATE TABLE IF NOT EXISTS aquaflow.chat_messages (
    message_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES aquaflow.chat_sessions(session_id) ON DELETE CASCADE,
    user_id VARCHAR(100) NOT NULL,
    message_type VARCHAR(20) NOT NULL CHECK (message_type IN ('user', 'assistant', 'clarification', 'system')),
    content TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =====================================================
-- LEARNING AND FEEDBACK SYSTEM
-- =====================================================

-- Learning feedback from users
CREATE TABLE IF NOT EXISTS aquaflow.chat_learning_feedback (
    feedback_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES aquaflow.chat_sessions(session_id),
    message_id UUID NOT NULL REFERENCES aquaflow.chat_messages(message_id),
    original_query TEXT NOT NULL,
    correct_sql TEXT,
    entity_mappings JSONB,
    helpful BOOLEAN NOT NULL,
    comments TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed BOOLEAN DEFAULT false,
    processed_at TIMESTAMP WITH TIME ZONE
);

-- Query patterns learned from user interactions
CREATE TABLE IF NOT EXISTS aquaflow.chat_learned_patterns (
    pattern_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(100),
    original_query TEXT NOT NULL,
    normalized_query TEXT NOT NULL,
    intent VARCHAR(50) NOT NULL,
    entities JSONB NOT NULL,
    generated_sql TEXT,
    confidence_score REAL DEFAULT 0.0,
    success_count INTEGER DEFAULT 1,
    failure_count INTEGER DEFAULT 0,
    last_used TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Entity mappings learned from user corrections
CREATE TABLE IF NOT EXISTS aquaflow.chat_entity_mappings (
    mapping_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(100),
    abbreviation VARCHAR(50) NOT NULL,
    full_form VARCHAR(255) NOT NULL,
    entity_type VARCHAR(50) NOT NULL, -- 'location', 'parameter', 'system', 'unit'
    confidence_score REAL DEFAULT 1.0,
    usage_count INTEGER DEFAULT 1,
    last_used TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, abbreviation, entity_type)
);

-- =====================================================
-- USER PREFERENCES AND ANALYTICS
-- =====================================================

-- User preferences for chat interface
CREATE TABLE IF NOT EXISTS aquaflow.chat_user_preferences (
    user_id VARCHAR(100) PRIMARY KEY,
    entity_shortcuts JSONB DEFAULT '{}',
    detail_level VARCHAR(20) DEFAULT 'normal' CHECK (detail_level IN ('brief', 'normal', 'detailed')),
    preferred_units VARCHAR(20) DEFAULT 'imperial' CHECK (preferred_units IN ('imperial', 'metric')),
    auto_clarify BOOLEAN DEFAULT true,
    save_history BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Analytics for learning system performance
CREATE TABLE IF NOT EXISTS aquaflow.chat_analytics (
    analytic_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(100),
    date_recorded DATE DEFAULT CURRENT_DATE,
    total_queries INTEGER DEFAULT 0,
    successful_queries INTEGER DEFAULT 0,
    clarification_requests INTEGER DEFAULT 0,
    feedback_submissions INTEGER DEFAULT 0,
    avg_confidence_score REAL DEFAULT 0.0,
    avg_response_time_ms INTEGER DEFAULT 0,
    unique_entities_learned INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, date_recorded)
);

-- =====================================================
-- LLM TRAINING DATA PREPARATION
-- =====================================================

-- Training data for model fine-tuning
CREATE TABLE IF NOT EXISTS aquaflow.chat_training_data (
    training_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    input_query TEXT NOT NULL,
    expected_sql TEXT NOT NULL,
    expected_entities JSONB NOT NULL,
    expected_intent VARCHAR(50) NOT NULL,
    context JSONB,
    quality_score REAL DEFAULT 1.0, -- 0.0 to 1.0
    source_type VARCHAR(50) DEFAULT 'user_feedback', -- 'user_feedback', 'synthetic', 'expert'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    used_for_training BOOLEAN DEFAULT false,
    training_batch_id UUID
);

-- Query processing performance metrics
CREATE TABLE IF NOT EXISTS aquaflow.chat_performance_metrics (
    metric_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES aquaflow.chat_sessions(session_id),
    query_text TEXT NOT NULL,
    intent_detected VARCHAR(50),
    entities_extracted JSONB,
    sql_generated TEXT,
    confidence_score REAL,
    processing_time_ms INTEGER,
    execution_success BOOLEAN,
    user_satisfaction REAL, -- 0.0 to 1.0, from feedback
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =====================================================
-- INDEXES FOR PERFORMANCE
-- =====================================================

-- Chat sessions indexes
CREATE INDEX idx_chat_sessions_user_id ON aquaflow.chat_sessions(user_id);
CREATE INDEX idx_chat_sessions_last_activity ON aquaflow.chat_sessions(last_activity DESC);
CREATE INDEX idx_chat_sessions_active ON aquaflow.chat_sessions(is_active) WHERE is_active = true;

-- Chat messages indexes
CREATE INDEX idx_chat_messages_session_id ON aquaflow.chat_messages(session_id);
CREATE INDEX idx_chat_messages_created_at ON aquaflow.chat_messages(created_at DESC);
CREATE INDEX idx_chat_messages_type ON aquaflow.chat_messages(message_type);

-- Learning feedback indexes
CREATE INDEX idx_chat_feedback_session_id ON aquaflow.chat_learning_feedback(session_id);
CREATE INDEX idx_chat_feedback_processed ON aquaflow.chat_learning_feedback(processed) WHERE processed = false;
CREATE INDEX idx_chat_feedback_created_at ON aquaflow.chat_learning_feedback(created_at DESC);

-- Learned patterns indexes
CREATE INDEX idx_chat_patterns_user_id ON aquaflow.chat_learned_patterns(user_id);
CREATE INDEX idx_chat_patterns_intent ON aquaflow.chat_learned_patterns(intent);
CREATE INDEX idx_chat_patterns_confidence ON aquaflow.chat_learned_patterns(confidence_score DESC);
CREATE INDEX idx_chat_patterns_last_used ON aquaflow.chat_learned_patterns(last_used DESC);

-- Entity mappings indexes
CREATE INDEX idx_chat_entities_user_id ON aquaflow.chat_entity_mappings(user_id);
CREATE INDEX idx_chat_entities_type ON aquaflow.chat_entity_mappings(entity_type);
CREATE INDEX idx_chat_entities_usage ON aquaflow.chat_entity_mappings(usage_count DESC);

-- Analytics indexes
CREATE INDEX idx_chat_analytics_user_date ON aquaflow.chat_analytics(user_id, date_recorded);
CREATE INDEX idx_chat_analytics_date ON aquaflow.chat_analytics(date_recorded DESC);

-- Training data indexes
CREATE INDEX idx_chat_training_intent ON aquaflow.chat_training_data(expected_intent);
CREATE INDEX idx_chat_training_quality ON aquaflow.chat_training_data(quality_score DESC);
CREATE INDEX idx_chat_training_unused ON aquaflow.chat_training_data(used_for_training) WHERE used_for_training = false;

-- Performance metrics indexes
CREATE INDEX idx_chat_metrics_session ON aquaflow.chat_performance_metrics(session_id);
CREATE INDEX idx_chat_metrics_created_at ON aquaflow.chat_performance_metrics(created_at DESC);
CREATE INDEX idx_chat_metrics_confidence ON aquaflow.chat_performance_metrics(confidence_score DESC);

-- =====================================================
-- TRIGGERS FOR AUTOMATIC UPDATES
-- =====================================================

-- Update chat session activity and message count
CREATE OR REPLACE FUNCTION aquaflow.update_chat_session_activity()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE aquaflow.chat_sessions 
    SET 
        last_activity = NOW(),
        message_count = (
            SELECT COUNT(*) 
            FROM aquaflow.chat_messages 
            WHERE session_id = NEW.session_id
        ),
        last_query = CASE 
            WHEN NEW.message_type = 'user' THEN NEW.content 
            ELSE (
                SELECT content 
                FROM aquaflow.chat_messages 
                WHERE session_id = NEW.session_id 
                AND message_type = 'user' 
                ORDER BY created_at DESC 
                LIMIT 1
            )
        END
    WHERE session_id = NEW.session_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_chat_session_activity
    AFTER INSERT ON aquaflow.chat_messages
    FOR EACH ROW
    EXECUTE FUNCTION aquaflow.update_chat_session_activity();

-- Update user preferences timestamp
CREATE OR REPLACE FUNCTION aquaflow.update_chat_preferences_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_chat_preferences_timestamp
    BEFORE UPDATE ON aquaflow.chat_user_preferences
    FOR EACH ROW
    EXECUTE FUNCTION aquaflow.update_chat_preferences_timestamp();

-- Update entity mapping usage
CREATE OR REPLACE FUNCTION aquaflow.update_entity_usage()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE aquaflow.chat_entity_mappings 
    SET 
        usage_count = usage_count + 1,
        last_used = NOW()
    WHERE user_id = NEW.user_id 
    AND abbreviation = ANY(SELECT jsonb_object_keys(NEW.entity_mappings)::text);
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_entity_usage
    AFTER INSERT ON aquaflow.chat_learned_patterns
    FOR EACH ROW
    EXECUTE FUNCTION aquaflow.update_entity_usage();

-- =====================================================
-- INITIAL DATA AND SETUP
-- =====================================================

-- Insert default entity mappings for water operations
INSERT INTO aquaflow.chat_entity_mappings (user_id, abbreviation, full_form, entity_type, confidence_score) VALUES
('default', 'mc', 'main canal', 'location', 1.0),
('default', 'ps1', 'pump station 1', 'location', 1.0),
('default', 'ps2', 'pump station 2', 'location', 1.0),
('default', 'dp', 'don pedro reservoir', 'location', 1.0),
('default', 'cfs', 'flow rate', 'parameter', 1.0),
('default', 'psi', 'pressure', 'parameter', 1.0),
('default', 'ft', 'feet', 'unit', 1.0),
('default', 'lvl', 'water level', 'parameter', 1.0),
('default', 'eff', 'efficiency', 'parameter', 1.0),
('default', 'temp', 'temperature', 'parameter', 1.0),
('default', 'gate', 'gate position', 'parameter', 1.0)
ON CONFLICT (user_id, abbreviation, entity_type) DO NOTHING;

-- Insert sample training data for common water operations queries
INSERT INTO aquaflow.chat_training_data (input_query, expected_sql, expected_entities, expected_intent, quality_score, source_type) VALUES
(
    'What is the flow rate at Main Canal?',
    'SELECT nv.value, nv.time_point FROM aquaflow.numeric_values nv JOIN aquaflow.series s ON nv.series_id = s.series_id JOIN aquaflow.datasets d ON s.dataset_id = d.dataset_id JOIN aquaflow.parameters p ON s.parameter_id = p.parameter_id WHERE d.dataset_name = ''Main Canal'' AND p.parameter_name = ''Flow Rate'' ORDER BY nv.time_point DESC LIMIT 1',
    '{"location": "main canal", "parameter": "flow rate"}',
    'status',
    1.0,
    'expert'
),
(
    'Show me pump station 1 status',
    'SELECT p.parameter_name, nv.value, p.unit, nv.time_point FROM aquaflow.numeric_values nv JOIN aquaflow.series s ON nv.series_id = s.series_id JOIN aquaflow.datasets d ON s.dataset_id = d.dataset_id JOIN aquaflow.parameters p ON s.parameter_id = p.parameter_id WHERE d.dataset_name = ''Pump Stations'' AND EXISTS (SELECT 1 FROM aquaflow.series_dimensions sd JOIN aquaflow.dimensions dim ON sd.dimension_id = dim.dimension_id WHERE sd.series_id = s.series_id AND dim.dimension_value = ''Pump Station 1'') ORDER BY nv.time_point DESC',
    '{"location": "pump station 1"}',
    'status',
    1.0,
    'expert'
),
(
    'Why is pressure low at station 7?',
    'SELECT nv.value, nv.time_point, CASE WHEN nv.value < (sm.metadata_value)::numeric THEN ''below normal'' ELSE ''normal'' END as status FROM aquaflow.numeric_values nv JOIN aquaflow.series s ON nv.series_id = s.series_id JOIN aquaflow.parameters p ON s.parameter_id = p.parameter_id LEFT JOIN aquaflow.series_metadata sm ON s.series_id = sm.series_id AND sm.metadata_key = ''threshold_normal_min'' WHERE p.parameter_name = ''Pressure'' AND s.series_id IN (SELECT sd.series_id FROM aquaflow.series_dimensions sd JOIN aquaflow.dimensions d ON sd.dimension_id = d.dimension_id WHERE d.dimension_value LIKE ''%Station 7%'') ORDER BY nv.time_point DESC LIMIT 24',
    '{"location": "station 7", "parameter": "pressure"}',
    'investigation',
    1.0,
    'expert'
);

-- Create view for easy access to user learning data
CREATE OR REPLACE VIEW aquaflow.vw_user_learning_summary AS
SELECT 
    u.user_id,
    COUNT(DISTINCT cs.session_id) as total_sessions,
    COUNT(DISTINCT cm.message_id) as total_messages,
    COUNT(DISTINCT clf.feedback_id) as feedback_given,
    COUNT(DISTINCT clp.pattern_id) as patterns_learned,
    COUNT(DISTINCT cem.mapping_id) as entities_learned,
    AVG(cpm.confidence_score) as avg_confidence,
    MAX(cs.last_activity) as last_activity
FROM aquaflow.chat_user_preferences u
LEFT JOIN aquaflow.chat_sessions cs ON u.user_id = cs.user_id
LEFT JOIN aquaflow.chat_messages cm ON cs.session_id = cm.session_id
LEFT JOIN aquaflow.chat_learning_feedback clf ON cs.session_id = clf.session_id
LEFT JOIN aquaflow.chat_learned_patterns clp ON u.user_id = clp.user_id
LEFT JOIN aquaflow.chat_entity_mappings cem ON u.user_id = cem.user_id
LEFT JOIN aquaflow.chat_performance_metrics cpm ON cs.session_id = cpm.session_id
GROUP BY u.user_id;

COMMENT ON TABLE aquaflow.chat_sessions IS 'Chat sessions for conversation management and context tracking';
COMMENT ON TABLE aquaflow.chat_messages IS 'Individual messages within chat sessions';
COMMENT ON TABLE aquaflow.chat_learning_feedback IS 'User feedback for improving query understanding';
COMMENT ON TABLE aquaflow.chat_learned_patterns IS 'Query patterns learned from user interactions';
COMMENT ON TABLE aquaflow.chat_entity_mappings IS 'Entity abbreviations and mappings learned from users';
COMMENT ON TABLE aquaflow.chat_training_data IS 'Training data for LLM fine-tuning';
COMMENT ON VIEW aquaflow.vw_user_learning_summary IS 'Summary of user learning and interaction statistics';