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
	Schedule         *string                `json:"schedule,omitempty"`
	NextRun          *time.Time             `json:"next_run,omitempty"`
	JobID            string                 `json:"job_id"`
	RunNumber        int                    `json:"run_number"`
}

type ETLJobLog struct {
	LogID     int                    `json:"log_id"`
	BatchID   string                 `json:"batch_id"`
	Timestamp time.Time              `json:"timestamp"`
	LogLevel  string                 `json:"log_level"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

type JobDefinition struct {
	JobID       string                 `json:"job_id"`
	JobName     string                 `json:"job_name"`
	JobType     string                 `json:"job_type"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type Schedule struct {
	ScheduleID     string    `json:"schedule_id"`
	JobID          string    `json:"job_id"`
	ScheduleName   string    `json:"schedule_name"`
	CronExpression string    `json:"cron_expression"`
	NextRun        time.Time `json:"next_run"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	JobName        string    `json:"job_name,omitempty"`
}

type JobRun struct {
	RunID            string                 `json:"run_id"`
	JobID            string                 `json:"job_id"`
	ScheduleID       *string                `json:"schedule_id,omitempty"`
	RunName          string                 `json:"run_name"`
	Status           string                 `json:"status"`
	TriggerType      string                 `json:"trigger_type"`
	ScheduledFor     time.Time              `json:"scheduled_for"`
	StartedAt        *time.Time             `json:"started_at,omitempty"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	RuntimeParams    map[string]interface{} `json:"runtime_parameters,omitempty"`
	RecordsProcessed int                    `json:"records_processed"`
	RecordsFailed    int                    `json:"records_failed"`
	ErrorMessage     *string                `json:"error_message,omitempty"`
	JobName          string                 `json:"job_name,omitempty"`
	JobType          string                 `json:"job_type,omitempty"`
}

// GetJobs returns all ETL job runs with optional filtering
func (h *ETLHandler) GetJobs(c *gin.Context) {
	status := c.Query("status")
	limit := c.DefaultQuery("limit", "50")

	query := `
		WITH run_numbers AS (
			SELECT run_id, job_id,
				   ROW_NUMBER() OVER (PARTITION BY job_id ORDER BY started_at DESC) as run_number
			FROM aquaflow.etl_job_runs
		)
		SELECT r.run_id as batch_id, r.run_name as job_name, j.job_type, 'scheduled' as load_type,
			   r.status, COALESCE(r.runtime_parameters, j.parameters) as parameters,
			   r.records_processed, r.records_failed, r.started_at, r.completed_at, r.error_message,
			   s.cron_expression as schedule, s.next_run, r.job_id, rn.run_number
		FROM aquaflow.etl_job_runs r
		JOIN aquaflow.etl_jobs_v2 j ON r.job_id = j.job_id
		JOIN run_numbers rn ON r.run_id = rn.run_id
		LEFT JOIN aquaflow.etl_schedules s ON r.schedule_id = s.schedule_id
	`

	args := []interface{}{}
	if status != "" {
		query += " WHERE r.status = $1"
		args = append(args, status)
	}

	query += " ORDER BY r.started_at DESC LIMIT " + limit

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
			&job.ErrorMessage, &job.Schedule, &job.NextRun, &job.JobID, &job.RunNumber,
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

// GetJobDetails returns details for a specific job run
func (h *ETLHandler) GetJobDetails(c *gin.Context) {
	runID := c.Param("id")

	// Validate UUID format
	if _, err := uuid.Parse(runID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run ID format"})
		return
	}

	var job ETLJob
	var paramsJSON []byte

	query := `
		WITH run_numbers AS (
			SELECT run_id, job_id,
				   ROW_NUMBER() OVER (PARTITION BY job_id ORDER BY started_at DESC) as run_number
			FROM aquaflow.etl_job_runs
		)
		SELECT r.run_id as batch_id, r.run_name as job_name, j.job_type, 'scheduled' as load_type,
			   r.status, COALESCE(r.runtime_parameters, j.parameters) as parameters,
			   r.records_processed, r.records_failed, r.started_at, r.completed_at, r.error_message,
			   s.cron_expression as schedule, s.next_run, r.job_id, rn.run_number
		FROM aquaflow.etl_job_runs r
		JOIN aquaflow.etl_jobs_v2 j ON r.job_id = j.job_id
		JOIN run_numbers rn ON r.run_id = rn.run_id
		LEFT JOIN aquaflow.etl_schedules s ON r.schedule_id = s.schedule_id
		WHERE r.run_id = $1
	`

	err := h.db.QueryRow(query, runID).Scan(
		&job.BatchID, &job.JobName, &job.JobType, &job.LoadType,
		&job.Status, &paramsJSON, &job.RecordsProcessed,
		&job.RecordsFailed, &job.StartedAt, &job.CompletedAt,
		&job.ErrorMessage, &job.Schedule, &job.NextRun, &job.JobID, &job.RunNumber,
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

// GetJobLogs returns logs for a specific job run
func (h *ETLHandler) GetJobLogs(c *gin.Context) {
	runID := c.Param("id")
	since := c.Query("since")
	limit := c.DefaultQuery("limit", "100")

	// Validate UUID format
	if _, err := uuid.Parse(runID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run ID format"})
		return
	}

	query := `
		SELECT log_id, run_id as batch_id, timestamp, log_level, message, context
		FROM aquaflow.etl_job_logs_v2
		WHERE run_id = $1
	`

	args := []interface{}{runID}

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

// RestartJob creates a new job run with the same parameters as a failed run
func (h *ETLHandler) RestartJob(c *gin.Context) {
	runID := c.Param("id")

	// Validate UUID format
	if _, err := uuid.Parse(runID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run ID format"})
		return
	}

	// Get the original job run details
	var jobID, scheduleID uuid.UUID
	var runName string
	var paramsJSON []byte

	query := `
		SELECT r.job_id, r.schedule_id, r.run_name, COALESCE(r.runtime_parameters, j.parameters)
		FROM aquaflow.etl_job_runs r
		JOIN aquaflow.etl_jobs_v2 j ON r.job_id = j.job_id
		WHERE r.run_id = $1 AND r.status IN ('failed', 'completed_with_errors')
	`

	err := h.db.QueryRow(query, runID).Scan(&jobID, &scheduleID, &runName, &paramsJSON)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "job run not found or not in failed state"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create new job run
	newRunID := uuid.New()
	insertQuery := `
		INSERT INTO aquaflow.etl_job_runs 
		(run_id, job_id, schedule_id, run_name, runtime_parameters, status, trigger_type, scheduled_for)
		VALUES ($1, $2, $3, $4, $5, 'queued', 'manual', NOW())
	`

	_, err = h.db.Exec(insertQuery, newRunID, jobID, scheduleID, runName+" (Restart)", paramsJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Job run restarted successfully",
		"new_run_id":    newRunID.String(),
		"original_id":   runID,
	})
}

// GetAllLogs returns logs from all ETL job runs with filtering options
func (h *ETLHandler) GetAllLogs(c *gin.Context) {
	jobName := c.Query("job_name")
	logLevel := c.Query("level")
	seriesID := c.Query("series_id")
	since := c.Query("since")
	limit := c.DefaultQuery("limit", "200")

	query := `
		SELECT l.log_id, l.run_id as batch_id, l.timestamp, l.log_level, l.message, l.context,
			   r.run_name as job_name, j.job_type
		FROM aquaflow.etl_job_logs_v2 l
		JOIN aquaflow.etl_job_runs r ON l.run_id = r.run_id
		JOIN aquaflow.etl_jobs_v2 j ON r.job_id = j.job_id
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 0

	if jobName != "" {
		argCount++
		query += fmt.Sprintf(" AND r.run_name ILIKE $%d", argCount)
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

// GetJobDefinitions returns all job definitions
func (h *ETLHandler) GetJobDefinitions(c *gin.Context) {
	limit := c.DefaultQuery("limit", "50")

	query := `
		SELECT job_id, job_name, job_type, description, parameters, created_at, updated_at
		FROM aquaflow.etl_jobs_v2
		ORDER BY created_at DESC LIMIT ` + limit

	rows, err := h.db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	jobs := []JobDefinition{}
	for rows.Next() {
		var job JobDefinition
		var paramsJSON []byte

		err := rows.Scan(
			&job.JobID, &job.JobName, &job.JobType, &job.Description,
			&paramsJSON, &job.CreatedAt, &job.UpdatedAt,
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

// GetSchedules returns all schedules with their job details
func (h *ETLHandler) GetSchedules(c *gin.Context) {
	active := c.Query("active")
	limit := c.DefaultQuery("limit", "50")

	query := `
		SELECT s.schedule_id, s.job_id, s.schedule_name, s.cron_expression, 
		       s.next_run, s.is_active, s.created_at, j.job_name
		FROM aquaflow.etl_schedules s
		JOIN aquaflow.etl_jobs_v2 j ON s.job_id = j.job_id
	`

	args := []interface{}{}
	if active != "" {
		query += " WHERE s.is_active = $1"
		args = append(args, active == "true")
	}

	query += " ORDER BY s.next_run ASC LIMIT " + limit

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	schedules := []Schedule{}
	for rows.Next() {
		var schedule Schedule

		err := rows.Scan(
			&schedule.ScheduleID, &schedule.JobID, &schedule.ScheduleName,
			&schedule.CronExpression, &schedule.NextRun, &schedule.IsActive,
			&schedule.CreatedAt, &schedule.JobName,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		schedules = append(schedules, schedule)
	}

	c.JSON(http.StatusOK, gin.H{
		"schedules": schedules,
		"count":     len(schedules),
	})
}

// GetJobRuns returns all job runs with optional filtering
func (h *ETLHandler) GetJobRuns(c *gin.Context) {
	status := c.Query("status")
	jobID := c.Query("job_id")
	scheduleID := c.Query("schedule_id")
	limit := c.DefaultQuery("limit", "100")

	query := `
		SELECT r.run_id, r.job_id, r.schedule_id, r.run_name, r.status, r.trigger_type,
		       r.started_at, r.started_at, r.completed_at, r.runtime_parameters,
		       r.records_processed, r.records_failed, r.error_message,
		       j.job_name, j.job_type
		FROM aquaflow.etl_job_runs r
		JOIN aquaflow.etl_jobs_v2 j ON r.job_id = j.job_id
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 0

	if status != "" {
		argCount++
		query += fmt.Sprintf(" AND r.status = $%d", argCount)
		args = append(args, status)
	}

	if jobID != "" {
		argCount++
		query += fmt.Sprintf(" AND r.job_id = $%d", argCount)
		args = append(args, jobID)
	}

	if scheduleID != "" {
		argCount++
		query += fmt.Sprintf(" AND r.schedule_id = $%d", argCount)
		args = append(args, scheduleID)
	}

	query += " ORDER BY r.started_at DESC LIMIT " + limit

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	runs := []JobRun{}
	for rows.Next() {
		var run JobRun
		var paramsJSON []byte

		err := rows.Scan(
			&run.RunID, &run.JobID, &run.ScheduleID, &run.RunName,
			&run.Status, &run.TriggerType, &run.ScheduledFor,
			&run.StartedAt, &run.CompletedAt, &paramsJSON,
			&run.RecordsProcessed, &run.RecordsFailed, &run.ErrorMessage,
			&run.JobName, &run.JobType,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Parse runtime parameters JSON
		if len(paramsJSON) > 0 {
			if err := json.Unmarshal(paramsJSON, &run.RuntimeParams); err == nil {
				// Parsed successfully
			}
		}

		runs = append(runs, run)
	}

	c.JSON(http.StatusOK, gin.H{
		"runs":  runs,
		"count": len(runs),
	})
}