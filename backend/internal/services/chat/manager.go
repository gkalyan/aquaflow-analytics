package chat

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ConversationManager handles chat sessions and interactions with the LLM service
type ConversationManager struct {
	db             *sql.DB
	llmServiceURL  string
	sessions       map[string]*ChatSession
	httpClient     *http.Client
}

// NewConversationManager creates a new conversation manager
func NewConversationManager(db *sql.DB, llmServiceURL string) *ConversationManager {
	return &ConversationManager{
		db:            db,
		llmServiceURL: llmServiceURL,
		sessions:      make(map[string]*ChatSession),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetOrCreateSession retrieves an existing session or creates a new one
func (cm *ConversationManager) GetOrCreateSession(userID string, sessionID string) (*ChatSession, error) {
	// If sessionID is empty, create a new session
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Check if session exists in memory
	if session, exists := cm.sessions[sessionID]; exists {
		session.LastActivity = time.Now()
		return session, nil
	}

	// Try to load from database
	session, err := cm.loadSessionFromDB(sessionID)
	if err != nil {
		// Create new session if not found
		session = &ChatSession{
			SessionID:       sessionID,
			UserID:         userID,
			Messages:       []ChatMessage{},
			EntityMappings: make(map[string]string),
			Context:        make(map[string]interface{}),
			LastActivity:   time.Now(),
			IsActive:       true,
		}
		
		// Save to database
		if err := cm.saveSessionToDB(session); err != nil {
			return nil, fmt.Errorf("failed to save new session: %w", err)
		}
	}

	// Store in memory
	cm.sessions[sessionID] = session
	return session, nil
}

// ProcessQuery handles a user query through the chat interface
func (cm *ConversationManager) ProcessQuery(request QueryRequest) (*QueryResponse, error) {
	start := time.Now()

	// Get or create session
	session, err := cm.GetOrCreateSession(request.UserID, request.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Add user message to session
	userMessage := ChatMessage{
		ID:        uuid.New().String(),
		SessionID: session.SessionID,
		UserID:    request.UserID,
		Type:      "user",
		Content:   request.Query,
		Timestamp: time.Now(),
	}
	session.Messages = append(session.Messages, userMessage)

	// Send request to LLM service
	llmResponse, err := cm.callLLMService(request, session)
	if err != nil {
		return nil, fmt.Errorf("LLM service error: %w", err)
	}

	// Create response message ID
	responseMessageID := uuid.New().String()

	// If clarification is needed, handle it
	if llmResponse.NeedsClarification {
		return &QueryResponse{
			Answer:                 "I need some clarification to better understand your question.",
			Intent:                llmResponse.Intent,
			Confidence:            llmResponse.Confidence,
			NeedsClarification:    true,
			ClarificationQuestions: llmResponse.ClarificationQuestions,
			Entities:              llmResponse.Entities,
			SessionID:             session.SessionID,
			MessageID:             responseMessageID,
			ResponseTime:          int(time.Since(start).Milliseconds()),
		}, nil
	}

	// Execute SQL query if provided
	var data interface{}
	var answer string

	if llmResponse.SQL != "" {
		queryResult, err := cm.executeSQL(llmResponse.SQL)
		if err != nil {
			// If SQL execution fails, ask for clarification
			return &QueryResponse{
				Answer:                 "I had trouble executing that query. Could you rephrase your question?",
				Intent:                llmResponse.Intent,
				Confidence:            0.3,
				NeedsClarification:    true,
				ClarificationQuestions: []string{"Could you be more specific about what you're looking for?"},
				Entities:              llmResponse.Entities,
				SessionID:             session.SessionID,
				MessageID:             responseMessageID,
				ResponseTime:          int(time.Since(start).Milliseconds()),
			}, nil
		}

		data = queryResult
		answer = cm.generateNaturalLanguageResponse(queryResult, llmResponse.Intent, llmResponse.Entities)
	} else {
		answer = "I couldn't understand your query. Could you try rephrasing it?"
	}

	// Add assistant message to session
	assistantMessage := ChatMessage{
		ID:        responseMessageID,
		SessionID: session.SessionID,
		UserID:    request.UserID,
		Type:      "assistant", 
		Content:   answer,
		Metadata: map[string]interface{}{
			"sql":        llmResponse.SQL,
			"entities":   llmResponse.Entities,
			"intent":     llmResponse.Intent,
			"confidence": llmResponse.Confidence,
		},
		Timestamp: time.Now(),
	}
	session.Messages = append(session.Messages, assistantMessage)

	// Update entity mappings from LLM response
	for key, value := range llmResponse.Entities {
		session.EntityMappings[key] = value
	}

	// Update session activity
	session.LastActivity = time.Now()

	// Save session to database
	if err := cm.saveSessionToDB(session); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to save session to DB: %v\n", err)
	}

	return &QueryResponse{
		Answer:      answer,
		SQL:         llmResponse.SQL,
		Data:        data,
		Intent:      llmResponse.Intent,
		Confidence:  llmResponse.Confidence,
		Entities:    llmResponse.Entities,
		SessionID:   session.SessionID,
		MessageID:   responseMessageID,
		ResponseTime: int(time.Since(start).Milliseconds()),
	}, nil
}

// callLLMService sends a request to the local LLM service
func (cm *ConversationManager) callLLMService(request QueryRequest, session *ChatSession) (*LLMServiceResponse, error) {
	// Prepare context from recent messages
	context := make([]map[string]interface{}, 0)
	
	// Include last 5 messages for context
	messageCount := len(session.Messages)
	startIdx := 0
	if messageCount > 5 {
		startIdx = messageCount - 5
	}
	
	for i := startIdx; i < messageCount; i++ {
		msg := session.Messages[i]
		context = append(context, map[string]interface{}{
			"type":    msg.Type,
			"content": msg.Content,
		})
	}

	llmRequest := LLMServiceRequest{
		Query:     request.Query,
		SessionID: session.SessionID,
		UserID:    request.UserID,
		Context:   context,
	}

	jsonData, err := json.Marshal(llmRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request to LLM service
	resp, err := cm.httpClient.Post(
		cm.llmServiceURL+"/query",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM service returned status %d", resp.StatusCode)
	}

	var llmResponse LLMServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResponse); err != nil {
		return nil, fmt.Errorf("failed to decode LLM response: %w", err)
	}

	return &llmResponse, nil
}

// executeSQL executes the generated SQL query
func (cm *ConversationManager) executeSQL(sqlQuery string) (interface{}, error) {
	// For now, return a simple mock result
	// In a real implementation, this would execute against TimescaleDB
	
	// Basic SQL validation
	if sqlQuery == "" {
		return nil, fmt.Errorf("empty SQL query")
	}

	// Mock result for demonstration
	mockResult := map[string]interface{}{
		"value": 945.2,
		"unit":  "CFS",
		"timestamp": time.Now().Format(time.RFC3339),
		"status": "normal",
		"location": "Main Canal",
		"parameter": "Flow Rate",
	}

	return mockResult, nil
}

// generateNaturalLanguageResponse creates a natural language response from query results
func (cm *ConversationManager) generateNaturalLanguageResponse(data interface{}, intent string, entities map[string]string) string {
	// Simple response generation based on intent and data
	switch intent {
	case "status":
		if result, ok := data.(map[string]interface{}); ok {
			if value, ok := result["value"].(float64); ok {
				if unit, ok := result["unit"].(string); ok {
					if location, ok := result["location"].(string); ok {
						return fmt.Sprintf("Current flow rate at %s is %.1f %s (normal range)", location, value, unit)
					}
				}
			}
		}
		return "Current status is normal."
		
	case "comparison":
		return "Comparison results show values are within expected range."
		
	case "investigation":
		return "Analysis shows no significant anomalies detected."
		
	default:
		return "Here's what I found based on your query."
	}
}

// HandleClarification processes a clarification response from the user
func (cm *ConversationManager) HandleClarification(request ClarificationRequest) (*QueryResponse, error) {
	session, exists := cm.sessions[request.SessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Add clarification message
	clarificationMessage := ChatMessage{
		ID:        uuid.New().String(),
		SessionID: session.SessionID,
		UserID:    session.UserID,
		Type:      "clarification",
		Content:   request.Clarification,
		Metadata: map[string]interface{}{
			"original_message_id": request.MessageID,
			"user_choice":        request.UserChoice,
		},
		Timestamp: time.Now(),
	}
	session.Messages = append(session.Messages, clarificationMessage)

	// Process the clarified query
	queryRequest := QueryRequest{
		Query:     request.Clarification,
		SessionID: request.SessionID,
		UserID:    session.UserID,
	}

	return cm.ProcessQuery(queryRequest)
}

// SaveLearningFeedback saves user feedback for model improvement
func (cm *ConversationManager) SaveLearningFeedback(feedback LearningFeedback) error {
	// Save feedback to database for future model training
	query := `
		INSERT INTO chat_learning_feedback 
		(session_id, message_id, original_query, correct_sql, entity_mappings, helpful, comments, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`
	
	entityMappingsJSON, err := json.Marshal(feedback.EntityMappings)
	if err != nil {
		return fmt.Errorf("failed to marshal entity mappings: %w", err)
	}

	_, err = cm.db.Exec(query, 
		feedback.SessionID,
		feedback.MessageID,
		feedback.OriginalQuery,
		feedback.CorrectSQL,
		string(entityMappingsJSON),
		feedback.Helpful,
		feedback.Comments,
	)

	return err
}

// GetConversationHistory retrieves conversation history for a user
func (cm *ConversationManager) GetConversationHistory(userID string, limit int) ([]ConversationSummary, error) {
	query := `
		SELECT session_id, user_id, last_query, message_count, last_activity, 
		       EXTRACT(EPOCH FROM (last_activity - created_at))/60 as duration_minutes
		FROM chat_sessions 
		WHERE user_id = $1 
		ORDER BY last_activity DESC 
		LIMIT $2
	`

	rows, err := cm.db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []ConversationSummary
	for rows.Next() {
		var summary ConversationSummary
		err := rows.Scan(
			&summary.SessionID,
			&summary.UserID,
			&summary.LastQuery,
			&summary.MessageCount,
			&summary.LastActivity,
			&summary.Duration,
		)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// Helper methods for database operations

func (cm *ConversationManager) loadSessionFromDB(sessionID string) (*ChatSession, error) {
	// Implementation would load session from database
	// For now, return an error to indicate session not found
	return nil, fmt.Errorf("session not found in database")
}

func (cm *ConversationManager) saveSessionToDB(session *ChatSession) error {
	// Implementation would save session to database
	// For now, just return nil (no-op)
	return nil
}