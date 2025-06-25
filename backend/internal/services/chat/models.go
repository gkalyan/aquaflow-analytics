package chat

import (
	"time"
)

// ChatMessage represents a single message in a conversation
type ChatMessage struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	UserID    string                 `json:"user_id"`
	Type      string                 `json:"type"` // "user", "assistant", "clarification", "system"
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ChatSession represents an ongoing conversation
type ChatSession struct {
	SessionID        string                 `json:"session_id"`
	UserID          string                 `json:"user_id"`
	Messages        []ChatMessage          `json:"messages"`
	EntityMappings  map[string]string      `json:"entity_mappings"`
	Context         map[string]interface{} `json:"context"`
	LastActivity    time.Time              `json:"last_activity"`
	IsActive        bool                   `json:"is_active"`
}

// QueryRequest represents a user query
type QueryRequest struct {
	Query     string            `json:"query" binding:"required"`
	SessionID string            `json:"session_id" binding:"required"`
	UserID    string            `json:"user_id" binding:"required"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// QueryResponse represents the system's response to a query
type QueryResponse struct {
	Answer                string                 `json:"answer"`
	SQL                   string                 `json:"sql,omitempty"`
	Data                  interface{}            `json:"data,omitempty"`
	Intent               string                 `json:"intent"`
	Confidence           float64                `json:"confidence"`
	NeedsClarification   bool                   `json:"needs_clarification"`
	ClarificationQuestions []string            `json:"clarification_questions,omitempty"`
	Entities             map[string]string      `json:"entities"`
	SessionID            string                 `json:"session_id"`
	MessageID            string                 `json:"message_id"`
	ResponseTime         int                    `json:"response_time_ms"`
}

// ClarificationRequest represents a user's response to clarification questions
type ClarificationRequest struct {
	SessionID     string `json:"session_id" binding:"required"`
	MessageID     string `json:"message_id" binding:"required"`
	Clarification string `json:"clarification" binding:"required"`
	UserChoice    string `json:"user_choice" binding:"required"`
}

// LearningFeedback represents user feedback for improving the system
type LearningFeedback struct {
	SessionID      string            `json:"session_id" binding:"required"`
	MessageID      string            `json:"message_id" binding:"required"`
	OriginalQuery  string            `json:"original_query" binding:"required"`
	CorrectSQL     string            `json:"correct_sql,omitempty"`
	EntityMappings map[string]string `json:"entity_mappings,omitempty"`
	Helpful        bool              `json:"helpful"`
	Comments       string            `json:"comments,omitempty"`
}

// LLMServiceRequest represents a request to the local LLM service
type LLMServiceRequest struct {
	Query     string                   `json:"query"`
	SessionID string                   `json:"session_id"`
	UserID    string                   `json:"user_id"`
	Context   []map[string]interface{} `json:"context"`
}

// LLMServiceResponse represents a response from the local LLM service
type LLMServiceResponse struct {
	SQL                    string            `json:"sql"`
	Intent                 string            `json:"intent"`
	Confidence             float64           `json:"confidence"`
	NeedsClarification     bool              `json:"needs_clarification"`
	ClarificationQuestions []string          `json:"clarification_questions"`
	Entities               map[string]string `json:"entities"`
	SessionID              string            `json:"session_id"`
}

// UserPreferences represents user-specific settings
type UserPreferences struct {
	UserID           string            `json:"user_id"`
	EntityShortcuts  map[string]string `json:"entity_shortcuts"`
	DetailLevel      string            `json:"detail_level"` // "brief", "normal", "detailed"
	PreferredUnits   string            `json:"preferred_units"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// ConversationSummary represents a summary of recent conversations
type ConversationSummary struct {
	SessionID     string    `json:"session_id"`
	UserID        string    `json:"user_id"`
	LastQuery     string    `json:"last_query"`
	MessageCount  int       `json:"message_count"`
	LastActivity  time.Time `json:"last_activity"`
	Duration      int       `json:"duration_minutes"`
}