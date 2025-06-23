package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gkalyan/aquaflow-analytics/internal/core/db"
	"github.com/google/uuid"
)

type ETLHandler struct {
	db *db.DB
}

func NewETLHandler(database *db.DB) *ETLHandler {
	return &ETLHandler{db: database}
}

type ETLJob struct {
	BatchID          string                 `json:"batch_id"`
	JobName          string                 `json:"job_name"`
	JobType          string                 `json:"job_type"`
	LoadType         string                 `json:"load_type"`
	Status           string                 `json:"status"`
	Parameters       map[string]interface{} `json:"parameters"`
	RecordsProcessed int                    `json:"records_processed"`
	RecordsFailed    int                    `json:"records_failed"`
	StartedAt        time.Time              `json:"started_at"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage     *string                `json:"error_message,omitempty"`
}

type ETLJobLog struct {
	LogID     int                    `json:"log_id"`
	BatchID   string                 `json:"batch_id"`
	Timestamp time.Time              `json:"timestamp"`
	LogLevel  string                 `json:"log_level"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// GetJobs returns all ETL jobs with optional filtering
func (h *ETLHandler) GetJobs(c *gin.Context) {
	status := c.Query("status")
	limit := c.DefaultQuery("limit", "50")

	query := `
		SELECT batch_id, job_name, job_type, load_type, status, parameters,
			   records_processed, records_failed, started_at, completed_at, error_message
		FROM aquaflow.etl_jobs
	`

	args := []interface{}{}
	if status != "" {
		query += " WHERE status = $1"
		args = append(args, status)
	}

	query += " ORDER BY started_at DESC LIMIT " + limit

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	jobs := []ETLJob{}
	for rows.Next() {
		var job ETLJob
		var paramsJSON []byte

		err := rows.Scan(
			&job.BatchID, &job.JobName, &job.JobType, &job.LoadType,
			&job.Status, &paramsJSON, &job.RecordsProcessed,
			&job.RecordsFailed, &job.StartedAt, &job.CompletedAt,
			&job.ErrorMessage,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Parse parameters JSON
		if len(paramsJSON) > 0 {
			if err := json.Unmarshal(paramsJSON, &job.Parameters); err == nil {
				// Parsed successfully
			}
		}

		jobs = append(jobs, job)
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

// GetJobDetails returns details for a specific job
func (h *ETLHandler) GetJobDetails(c *gin.Context) {
	batchID := c.Param("id")

	// Validate UUID format
	if _, err := uuid.Parse(batchID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid batch ID format"})
		return
	}

	var job ETLJob
	var paramsJSON []byte

	query := `
		SELECT batch_id, job_name, job_type, load_type, status, parameters,
			   records_processed, records_failed, started_at, completed_at, error_message
		FROM aquaflow.etl_jobs
		WHERE batch_id = $1
	`

	err := h.db.QueryRow(query, batchID).Scan(
		&job.BatchID, &job.JobName, &job.JobType, &job.LoadType,
		&job.Status, &paramsJSON, &job.RecordsProcessed,
		&job.RecordsFailed, &job.StartedAt, &job.CompletedAt,
		&job.ErrorMessage,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Parse parameters JSON
	if len(paramsJSON) > 0 {
		if err := json.Unmarshal(paramsJSON, &job.Parameters); err == nil {
			// Parsed successfully
		}
	}

	c.JSON(http.StatusOK, job)
}

// GetJobLogs returns logs for a specific job
func (h *ETLHandler) GetJobLogs(c *gin.Context) {
	batchID := c.Param("id")
	since := c.Query("since")
	limit := c.DefaultQuery("limit", "100")

	// Validate UUID format
	if _, err := uuid.Parse(batchID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid batch ID format"})
		return
	}

	query := `
		SELECT log_id, batch_id, timestamp, log_level, message, context
		FROM aquaflow.etl_job_logs
		WHERE batch_id = $1
	`

	args := []interface{}{batchID}

	if since != "" {
		sinceTime, err := time.Parse(time.RFC3339, since)
		if err == nil {
			query += " AND timestamp > $2"
			args = append(args, sinceTime)
		}
	}

	query += " ORDER BY timestamp DESC LIMIT " + limit

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	logs := []ETLJobLog{}
	for rows.Next() {
		var log ETLJobLog
		var contextJSON []byte

		err := rows.Scan(
			&log.LogID, &log.BatchID, &log.Timestamp,
			&log.LogLevel, &log.Message, &contextJSON,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Parse context JSON
		if len(contextJSON) > 0 {
			if err := json.Unmarshal(contextJSON, &log.Context); err == nil {
				// Parsed successfully
			}
		}

		logs = append(logs, log)
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"count": len(logs),
	})
}

// RestartJob creates a new job with the same parameters as a failed job
func (h *ETLHandler) RestartJob(c *gin.Context) {
	batchID := c.Param("id")

	// Validate UUID format
	if _, err := uuid.Parse(batchID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid batch ID format"})
		return
	}

	// Get the original job
	var jobName, jobType string
	var paramsJSON []byte

	query := `
		SELECT job_name, job_type, parameters
		FROM aquaflow.etl_jobs
		WHERE batch_id = $1 AND status IN ('failed', 'completed_with_errors')
	`

	err := h.db.QueryRow(query, batchID).Scan(&jobName, &jobType, &paramsJSON)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found or not in failed state"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create new job
	newBatchID := uuid.New()
	insertQuery := `
		INSERT INTO aquaflow.etl_jobs (batch_id, job_name, job_type, parameters, status)
		VALUES ($1, $2, $3, $4, 'pending')
	`

	_, err = h.db.Exec(insertQuery, newBatchID, jobName+" (Restart)", jobType, paramsJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Job restarted successfully",
		"new_batch_id":  newBatchID.String(),
		"original_id":   batchID,
	})
}

// GetAllLogs returns logs from all ETL jobs with filtering options
func (h *ETLHandler) GetAllLogs(c *gin.Context) {
	jobName := c.Query("job_name")
	logLevel := c.Query("level")
	seriesID := c.Query("series_id")
	since := c.Query("since")
	limit := c.DefaultQuery("limit", "200")

	query := `
		SELECT l.log_id, l.batch_id, l.timestamp, l.log_level, l.message, l.context,
			   j.job_name, j.job_type
		FROM aquaflow.etl_job_logs l
		JOIN aquaflow.etl_jobs j ON l.batch_id = j.batch_id
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 0

	if jobName != "" {
		argCount++
		query += fmt.Sprintf(" AND j.job_name ILIKE $%d", argCount)
		args = append(args, "%"+jobName+"%")
	}

	if logLevel != "" {
		argCount++
		query += fmt.Sprintf(" AND l.log_level = $%d", argCount)
		args = append(args, logLevel)
	}

	if seriesID != "" {
		argCount++
		query += fmt.Sprintf(" AND l.context->>'series_id' = $%d", argCount)
		args = append(args, seriesID)
	}

	if since != "" {
		sinceTime, err := time.Parse(time.RFC3339, since)
		if err == nil {
			argCount++
			query += fmt.Sprintf(" AND l.timestamp > $%d", argCount)
			args = append(args, sinceTime)
		}
	}

	query += " ORDER BY l.timestamp DESC LIMIT " + limit

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type LogWithJobInfo struct {
		ETLJobLog
		JobName string `json:"job_name"`
		JobType string `json:"job_type"`
	}

	logs := []LogWithJobInfo{}
	for rows.Next() {
		var log LogWithJobInfo
		var contextJSON []byte

		err := rows.Scan(
			&log.LogID, &log.BatchID, &log.Timestamp,
			&log.LogLevel, &log.Message, &contextJSON,
			&log.JobName, &log.JobType,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Parse context JSON
		if len(contextJSON) > 0 {
			if err := json.Unmarshal(contextJSON, &log.Context); err == nil {
				// Parsed successfully
			}
		}

		logs = append(logs, log)
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"count": len(logs),
	})
}