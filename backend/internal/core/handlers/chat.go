package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gkalyan/aquaflow-analytics/internal/core/db"
	"github.com/gkalyan/aquaflow-analytics/internal/services/chat"
)

type ChatHandler struct {
	ollamaService *chat.OllamaService
}

func NewChatHandler(database *db.DB) *ChatHandler {
	return &ChatHandler{
		ollamaService: chat.NewOllamaService(database.DB),
	}
}

func (h *ChatHandler) Chat(c *gin.Context) {
	var req chat.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Process message with integrated Ollama service
	ctx := context.Background()
	response, err := h.ollamaService.ProcessMessage(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process message",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *ChatHandler) Feedback(c *gin.Context) {
	var req chat.FeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Process feedback with integrated service
	ctx := context.Background()
	err := h.ollamaService.ProcessFeedback(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process feedback",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "feedback_received",
		"session_id": req.SessionID,
	})
}

func (h *ChatHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	sessionData, err := h.ollamaService.GetSessionInfo(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	c.JSON(http.StatusOK, sessionData)
}

func (h *ChatHandler) ClearSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	err := h.ollamaService.ClearSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "session_cleared",
		"session_id": sessionID,
	})
}