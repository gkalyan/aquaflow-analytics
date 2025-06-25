package chat

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ollama/ollama/api"
)

type OllamaService struct {
	client     *api.Client
	db         *sql.DB
	ollamaHost string
	modelName  string
}


type ChatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
	UserID    string `json:"user_id"`
}

type ChatResponse struct {
	Response               string            `json:"response"`
	SessionID              string            `json:"session_id"`
	NeedsClarification     bool              `json:"needs_clarification"`
	ClarificationQuestion  string            `json:"clarification_question,omitempty"`
	EntityMappings         map[string]string `json:"entity_mappings"`
	Confidence             float64           `json:"confidence"`
}

type FeedbackRequest struct {
	SessionID      string `json:"session_id"`
	MessageID      string `json:"message_id"`
	UserQuery      string `json:"user_query"`
	SystemResponse string `json:"system_response"`
	UserCorrection string `json:"user_correction"`
	IsHelpful      bool   `json:"is_helpful"`
}

func NewOllamaService(db *sql.DB) *OllamaService {
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}

	modelName := os.Getenv("OLLAMA_MODEL")
	if modelName == "" {
		modelName = "llama3.2:3b"
	}

	client, err := api.ClientFromEnvironment()
	if err != nil {
		// Try with manual configuration
		client = api.NewClient(nil, nil)
	}

	return &OllamaService{
		client:     client,
		db:         db,
		ollamaHost: ollamaHost,
		modelName:  modelName,
	}
}

func (s *OllamaService) ProcessMessage(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	// Get or create session
	session, err := s.getOrCreateSession(req.SessionID, req.UserID)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("failed to get session: %w", err)
	}

	// Create system prompt with water operations context
	systemPrompt := s.createSystemPrompt()

	// Enrich context with relevant data based on user query
	dataContext, err := s.enrichWithDataContext(ctx, req.Message)
	if err != nil {
		log.Printf("Warning: Failed to enrich data context: %v", err)
	}


	// Add data context to system prompt if available
	if dataContext != "" {
		systemPrompt += "\n\n## Current Data Context:\n" + dataContext
	}

	// Build conversation history
	messages := []api.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	// Add recent conversation history (last 10 messages)
	recentMessages := session.Messages
	if len(recentMessages) > 10 {
		recentMessages = recentMessages[len(recentMessages)-10:]
	}

	for _, msg := range recentMessages {
		messages = append(messages, api.Message{
			Role:    msg.Type, // Use Type field from existing model
			Content: msg.Content,
		})
	}

	// Add current user message
	messages = append(messages, api.Message{
		Role:    "user",
		Content: req.Message,
	})

	// Call Ollama
	response, err := s.callOllama(ctx, messages)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("ollama call failed: %w", err)
	}

	// Store messages in session
	userMsg := ChatMessage{
		ID:        uuid.New().String(),
		Type:      "user",
		Content:   req.Message,
		Timestamp: time.Now(),
	}

	assistantMsg := ChatMessage{
		ID:        uuid.New().String(),
		Type:      "assistant",
		Content:   response,
		Timestamp: time.Now(),
	}

	session.Messages = append(session.Messages, userMsg, assistantMsg)
	session.LastActivity = time.Now()

	// Save session
	if err := s.saveSession(session); err != nil {
		log.Printf("Failed to save session: %v", err)
	}

	// Check for clarification needs
	needsClarification := s.needsClarification(response)
	clarificationQuestion := ""
	if needsClarification {
		clarificationQuestion = s.extractClarificationQuestion(response)
	}

	return ChatResponse{
		Response:               response,
		SessionID:              session.SessionID,
		NeedsClarification:     needsClarification,
		ClarificationQuestion:  clarificationQuestion,
		EntityMappings:         session.EntityMappings,
		Confidence:             0.8, // Default confidence
	}, nil
}

func (s *OllamaService) callOllama(ctx context.Context, messages []api.Message) (string, error) {
	// Check if model is available
	listResp, err := s.client.List(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to connect to Ollama: %w", err)
	}

	modelAvailable := false
	for _, model := range listResp.Models {
		if model.Name == s.modelName {
			modelAvailable = true
			break
		}
	}

	if !modelAvailable {
		return "", fmt.Errorf("model %s is not available. Please pull the model first", s.modelName)
	}

	// Create chat request
	req := &api.ChatRequest{
		Model:    s.modelName,
		Messages: messages,
		Options: map[string]interface{}{
			"temperature": 0.7,
			"top_p":       0.9,
		},
	}

	// Call Ollama
	resp := ""
	err = s.client.Chat(ctx, req, func(resp_chunk api.ChatResponse) error {
		resp += resp_chunk.Message.Content
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("Ollama chat request failed: %w", err)
	}

	if resp == "" {
		return "", fmt.Errorf("received empty response from model")
	}

	return resp, nil
}


func (s *OllamaService) createSystemPrompt() string {
	return `You are Afa, an AI assistant for AquaFlow Analytics - a water district operations platform.

Your role is to help water operations managers by:
1. Understanding natural language queries about water infrastructure and operations
2. Converting them into appropriate database queries or actions
3. Providing data-driven insights and operational recommendations
4. Asking clarifying questions when user intent is unclear

## CRITICAL RULES:

### Data Accuracy Requirements:
- **NEVER fabricate, estimate, or make up any numerical values, measurements, or operational data**
- **ONLY use data that is explicitly provided in the Current Data Context section**
- If no real data is available, clearly state "No current data available" instead of providing fake numbers
- Always cite the source timestamp when presenting measurements
- Be transparent about data limitations or gaps

### Response Style:
- **Keep responses very concise** - 1-2 sentences for simple status queries
- **Lead with the direct answer first** - state the measurement immediately
- **Avoid explanatory text** like "Based on the Current Data Context" unless essential
- **Use bullet points sparingly** - only for multiple distinct systems
- **Skip courtesy questions** like "Would you like more information?" unless user asks for help

## Database Schema Overview (aquaflow schema):

### Core Data Structure:
**datasets** → **series** → **numeric_values** (TimescaleDB hypertable)
- A **dataset** represents a logical grouping of related measurements (e.g., "Main Canal System", "Pump Station Network")
- A **series** is a unique combination of a parameter + dimensions (e.g., "Flow Rate at Main Canal Mile 5")
- **parameters** define what is being measured (flow_rate, water_level, pressure, pump_status)
- **dimensions** provide context like location, equipment_id, measurement_point, depth
- **numeric_values** contains the actual time-series measurements with timestamps

### Common Operational Terminology:
- **MC** = Main Canal
- **PS1, PS2, etc.** = Pump Station 1, 2, etc.
- **Flow rates** typically in CFS (cubic feet per second) or MGD (million gallons per day)
- **Water levels** in feet above sea level or depth measurements
- **Operational status** = running, stopped, maintenance, alarm states

### When Processing Queries:
1. Check if relevant data is provided in the Current Data Context
2. If data is available, present it concisely with timestamps
3. If no data is available, clearly state this fact
4. For ambiguous queries, suggest specific interpretations based on available data
5. Always focus on actionable operational insights

### Response Examples:
**Good (concise with real data):**
"Main Canal flow rate: 810.43 CFS (at 00:00)."

**Bad (lengthy and potentially fabricated):**
"Based on our comprehensive monitoring system, I can tell you that the Main Canal is currently operating at approximately 300 CFS, which represents normal operational parameters for this time of day..."

You have access to live operational data through the database. When Current Data Context is provided, use ONLY that data. When no context is provided, clearly state data is not available.

**Professional Boundaries**: For non-water operations questions, provide a brief helpful answer but suggest focusing on water system queries for your specialized assistance.

Always respond in a helpful, professional tone focused on supporting water district operations and decision-making.`
}

func (s *OllamaService) needsClarification(response string) bool {
	clarificationIndicators := []string{
		"could you clarify",
		"what do you mean by",
		"are you referring to",
		"which specific",
		"can you be more specific",
		"do you mean",
		"unclear",
	}

	responseLower := strings.ToLower(response)
	for _, indicator := range clarificationIndicators {
		if strings.Contains(responseLower, indicator) {
			return true
		}
	}
	return false
}

func (s *OllamaService) extractClarificationQuestion(response string) string {
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		if strings.Contains(line, "?") && 
		   (strings.Contains(strings.ToLower(line), "clarify") ||
			strings.Contains(strings.ToLower(line), "mean") ||
			strings.Contains(strings.ToLower(line), "specific") ||
			strings.Contains(strings.ToLower(line), "referring")) {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func (s *OllamaService) getOrCreateSession(sessionID, userID string) (*ChatSession, error) {
	// Try to get existing session
	session, err := s.getSession(sessionID)
	if err == nil {
		session.LastActivity = time.Now()
		return session, nil
	}

	// Create new session
	session = &ChatSession{
		SessionID:      sessionID,
		UserID:         userID,
		Messages:       []ChatMessage{},
		EntityMappings: make(map[string]string),
		Context:        make(map[string]interface{}),
		LastActivity:   time.Now(),
		IsActive:       true,
	}

	return session, nil
}

func (s *OllamaService) getSession(sessionID string) (*ChatSession, error) {
	// For now, return empty session (in-memory only)
	// TODO: Implement actual database storage
	return nil, fmt.Errorf("session not found")
}

func (s *OllamaService) saveSession(session *ChatSession) error {
	// For now, do nothing (in-memory only)
	// TODO: Implement actual database storage
	log.Printf("Session %s saved (in-memory)", session.SessionID)
	return nil
}

func (s *OllamaService) ProcessFeedback(ctx context.Context, feedback FeedbackRequest) error {
	// For now, just log feedback
	// TODO: Implement learning from feedback
	log.Printf("Feedback received for session %s: helpful=%v", 
		feedback.SessionID, feedback.IsHelpful)
	
	if feedback.UserCorrection != "" {
		log.Printf("User correction: %s", feedback.UserCorrection)
	}
	
	return nil
}

func (s *OllamaService) GetSessionInfo(sessionID string) (map[string]interface{}, error) {
	session, err := s.getSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found")
	}

	return map[string]interface{}{
		"session_id":      session.SessionID,
		"message_count":   len(session.Messages),
		"entity_mappings": session.EntityMappings,
		"last_activity":   session.LastActivity,
	}, nil
}

func (s *OllamaService) ClearSession(sessionID string) error {
	// For now, do nothing (in-memory only)
	// TODO: Implement actual session clearing
	log.Printf("Session %s cleared", sessionID)
	return nil
}

// enrichWithDataContext analyzes the user query and fetches relevant data to provide context
func (s *OllamaService) enrichWithDataContext(ctx context.Context, query string) (string, error) {
	context := ""
	queryLower := strings.ToLower(query)
	
	// Check for current data requests
	if s.isCurrentDataQuery(queryLower) {
		// Get recent data samples
		recentData, err := s.getRecentDataSamples(ctx)
		if err == nil && recentData != "" {
			context += "Recent Measurements:\n" + recentData + "\n"
		}
	}
	
	// Check for specific infrastructure queries
	if s.isInfrastructureQuery(queryLower) {
		infraData, err := s.getInfrastructureStatus(ctx, queryLower)
		if err == nil && infraData != "" {
			context += "Infrastructure Status:\n" + infraData + "\n"
		}
	}
	
	// Get available datasets and parameters for general queries
	if s.isSchemaQuery(queryLower) {
		schemaInfo, err := s.getSchemaInfo(ctx)
		if err == nil && schemaInfo != "" {
			context += "Available Data:\n" + schemaInfo + "\n"
		}
	}
	
	return context, nil
}

// isCurrentDataQuery checks if the user is asking for current/live data
func (s *OllamaService) isCurrentDataQuery(query string) bool {
	currentIndicators := []string{
		"current", "now", "latest", "today", "status", "live", "real-time",
		"what is", "show me", "tell me", "how much", "what's the",
	}
	
	for _, indicator := range currentIndicators {
		if strings.Contains(query, indicator) {
			return true
		}
	}
	return false
}

// isInfrastructureQuery checks if the user is asking about specific infrastructure
func (s *OllamaService) isInfrastructureQuery(query string) bool {
	infraIndicators := []string{
		"canal", "pump", "reservoir", "station", "ps1", "ps2", "mc", "main canal",
		"flow", "level", "pressure", "status", "operational",
	}
	
	for _, indicator := range infraIndicators {
		if strings.Contains(query, indicator) {
			return true
		}
	}
	return false
}

// isSchemaQuery checks if the user is asking about available data/capabilities
func (s *OllamaService) isSchemaQuery(query string) bool {
	schemaIndicators := []string{
		"what can", "what data", "what information", "available", "datasets",
		"parameters", "what do you have", "show me data", "list",
	}
	
	for _, indicator := range schemaIndicators {
		if strings.Contains(query, indicator) {
			return true
		}
	}
	return false
}

// getRecentDataSamples fetches the most recent measurements from key series
func (s *OllamaService) getRecentDataSamples(ctx context.Context) (string, error) {
	query := `
		SELECT 
			d.dataset_name as dataset,
			p.parameter_name as parameter,
			STRING_AGG(dim.dimension_name || '=' || dim.dimension_value, ', ') as dimensions,
			nv.value,
			nv.time_point,
			p.unit
		FROM aquaflow.numeric_values nv
		JOIN aquaflow.series s ON nv.series_id = s.series_id
		JOIN aquaflow.parameters p ON s.parameter_id = p.parameter_id
		JOIN aquaflow.datasets d ON s.dataset_id = d.dataset_id
		LEFT JOIN aquaflow.series_dimensions sd ON s.series_id = sd.series_id
		LEFT JOIN aquaflow.dimensions dim ON sd.dimension_id = dim.dimension_id
		WHERE nv.time_point >= NOW() - INTERVAL '1 hour'
		GROUP BY d.dataset_name, p.parameter_name, nv.value, nv.time_point, p.unit, nv.series_id
		ORDER BY nv.time_point DESC
		LIMIT 10
	`
	
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to query recent data: %w", err)
	}
	defer rows.Close()
	
	var results []string
	for rows.Next() {
		var dataset, parameter, dimensions, unit string
		var value float64
		var timestamp time.Time
		
		err := rows.Scan(&dataset, &parameter, &dimensions, &value, &timestamp, &unit)
		if err != nil {
			continue
		}
		
		results = append(results, fmt.Sprintf(
			"- %s %s: %.2f %s (at %s) [%s]",
			dataset, parameter, value, unit, 
			timestamp.Format("15:04"), dimensions,
		))
	}
	
	if len(results) == 0 {
		return "No recent data available in the last hour.", nil
	}
	
	return strings.Join(results, "\n"), nil
}

// getInfrastructureStatus gets status information for specific infrastructure
func (s *OllamaService) getInfrastructureStatus(ctx context.Context, query string) (string, error) {
	// This would be enhanced with specific infrastructure queries
	// For now, return a sample of operational parameters
	sqlQuery := `
		SELECT 
			d.dataset_name as dataset,
			p.parameter_name as parameter,
			STRING_AGG(dim.dimension_name || '=' || dim.dimension_value, ', ') as dimensions,
			nv.value,
			nv.time_point,
			p.unit
		FROM aquaflow.numeric_values nv
		JOIN aquaflow.series s ON nv.series_id = s.series_id
		JOIN aquaflow.parameters p ON s.parameter_id = p.parameter_id
		JOIN aquaflow.datasets d ON s.dataset_id = d.dataset_id
		LEFT JOIN aquaflow.series_dimensions sd ON s.series_id = sd.series_id
		LEFT JOIN aquaflow.dimensions dim ON sd.dimension_id = dim.dimension_id
		WHERE nv.time_point >= NOW() - INTERVAL '30 minutes'
		AND (
			p.parameter_name ILIKE '%flow%' OR 
			p.parameter_name ILIKE '%level%' OR 
			p.parameter_name ILIKE '%status%' OR
			p.parameter_name ILIKE '%pressure%'
		)
		GROUP BY d.dataset_name, p.parameter_name, nv.value, nv.time_point, p.unit, nv.series_id
		ORDER BY nv.time_point DESC
		LIMIT 5
	`
	
	rows, err := s.db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return "", fmt.Errorf("failed to query infrastructure status: %w", err)
	}
	defer rows.Close()
	
	var results []string
	for rows.Next() {
		var dataset, parameter, dimensions, unit string
		var value float64
		var timestamp time.Time
		
		err := rows.Scan(&dataset, &parameter, &dimensions, &value, &timestamp, &unit)
		if err != nil {
			continue
		}
		
		results = append(results, fmt.Sprintf(
			"- %s %s: %.2f %s (%s ago) [%s]",
			dataset, parameter, value, unit,
			formatTimeSince(timestamp), dimensions,
		))
	}
	
	if len(results) == 0 {
		return "No recent infrastructure data available.", nil
	}
	
	return strings.Join(results, "\n"), nil
}

// getSchemaInfo returns information about available datasets and parameters
func (s *OllamaService) getSchemaInfo(ctx context.Context) (string, error) {
	query := `
		SELECT 
			d.dataset_name as dataset,
			p.parameter_name as parameter,
			p.unit,
			COUNT(s.series_id) as series_count
		FROM aquaflow.datasets d
		JOIN aquaflow.series s ON d.dataset_id = s.dataset_id
		JOIN aquaflow.parameters p ON s.parameter_id = p.parameter_id
		GROUP BY d.dataset_name, p.parameter_name, p.unit
		ORDER BY d.dataset_name, p.parameter_name
		LIMIT 20
	`
	
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to query schema info: %w", err)
	}
	defer rows.Close()
	
	var results []string
	currentDataset := ""
	
	for rows.Next() {
		var dataset, parameter, unit string
		var seriesCount int
		
		err := rows.Scan(&dataset, &parameter, &unit, &seriesCount)
		if err != nil {
			continue
		}
		
		if dataset != currentDataset {
			if currentDataset != "" {
				results = append(results, "")
			}
			results = append(results, fmt.Sprintf("Dataset: %s", dataset))
			currentDataset = dataset
		}
		
		results = append(results, fmt.Sprintf(
			"  - %s (%s) - %d series",
			parameter, unit, seriesCount,
		))
	}
	
	if len(results) == 0 {
		return "No datasets available.", nil
	}
	
	return strings.Join(results, "\n"), nil
}

// formatTimeSince returns a human-readable time difference
func formatTimeSince(t time.Time) string {
	duration := time.Since(t)
	if duration < time.Minute {
		return fmt.Sprintf("%.0fs", duration.Seconds())
	} else if duration < time.Hour {
		return fmt.Sprintf("%.0fm", duration.Minutes())
	} else {
		return fmt.Sprintf("%.1fh", duration.Hours())
	}
}