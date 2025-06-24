package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gkalyan/aquaflow-analytics/internal/core/db"
)

type ETLControlHandler struct {
	db *db.DB
}

func NewETLControlHandler(database *db.DB) *ETLControlHandler {
	return &ETLControlHandler{db: database}
}

// PauseJobRequest represents the request to pause a job
type PauseJobRequest struct {
	Reason string `json:"reason"`
}

// PauseJob pauses a specific job type
func (h *ETLControlHandler) PauseJob(c *gin.Context) {
	jobName := c.Param("name")
	
	var req PauseJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	
	// Insert or update job control record
	query := `
		INSERT INTO aquaflow.etl_job_control (job_name, is_paused, paused_at, paused_by, pause_reason)
		VALUES ($1, true, NOW(), $2, $3)
		ON CONFLICT (job_name) 
		DO UPDATE SET 
			is_paused = true,
			paused_at = NOW(),
			paused_by = $2,
			pause_reason = $3,
			updated_at = NOW()
	`
	
	user := c.GetString("username")
	if user == "" {
		user = "system"
	}
	
	_, err := h.db.Exec(query, jobName, user, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Job '%s' has been paused", jobName),
		"job_name": jobName,
		"paused_by": user,
		"reason": req.Reason,
	})
}

// ResumeJob resumes a paused job type
func (h *ETLControlHandler) ResumeJob(c *gin.Context) {
	jobName := c.Param("name")
	
	// Update job control record
	query := `
		UPDATE aquaflow.etl_job_control 
		SET is_paused = false,
			paused_at = NULL,
			paused_by = NULL,
			pause_reason = NULL,
			updated_at = NOW()
		WHERE job_name = $1
	`
	
	result, err := h.db.Exec(query, jobName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "job control record not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Job '%s' has been resumed", jobName),
		"job_name": jobName,
	})
}

// GetJobControls returns the current job control settings
func (h *ETLControlHandler) GetJobControls(c *gin.Context) {
	query := `
		SELECT job_name, is_paused, paused_at, paused_by, pause_reason, created_at, updated_at
		FROM aquaflow.etl_job_control
		ORDER BY job_name
	`
	
	rows, err := h.db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	
	type JobControl struct {
		JobName     string     `json:"job_name"`
		IsPaused    bool       `json:"is_paused"`
		PausedAt    *time.Time `json:"paused_at,omitempty"`
		PausedBy    *string    `json:"paused_by,omitempty"`
		PauseReason *string    `json:"pause_reason,omitempty"`
		CreatedAt   time.Time  `json:"created_at"`
		UpdatedAt   time.Time  `json:"updated_at"`
	}
	
	controls := []JobControl{}
	for rows.Next() {
		var control JobControl
		
		err := rows.Scan(
			&control.JobName, &control.IsPaused, &control.PausedAt,
			&control.PausedBy, &control.PauseReason, &control.CreatedAt,
			&control.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		
		controls = append(controls, control)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"controls": controls,
		"count": len(controls),
	})
}