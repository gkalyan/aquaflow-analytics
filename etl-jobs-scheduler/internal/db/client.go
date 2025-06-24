package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type Client struct {
	db *sql.DB
}

type Job struct {
	JobID       uuid.UUID              `json:"job_id"`
	JobName     string                 `json:"job_name"`
	JobType     string                 `json:"job_type"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	IsActive    bool                   `json:"is_active"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type Schedule struct {
	ScheduleID     uuid.UUID  `json:"schedule_id"`
	JobID          uuid.UUID  `json:"job_id"`
	ScheduleName   string     `json:"schedule_name"`
	CronExpression string     `json:"cron_expression"`
	Timezone       string     `json:"timezone"`
	IsActive       bool       `json:"is_active"`
	NextRun        *time.Time `json:"next_run"`
	LastRun        *time.Time `json:"last_run"`
	RunCount       int        `json:"run_count"`
	FailureCount   int        `json:"failure_count"`
}

type JobRun struct {
	RunID              uuid.UUID              `json:"run_id"`
	JobID              uuid.UUID              `json:"job_id"`
	ScheduleID         *uuid.UUID             `json:"schedule_id"`
	RunName            string                 `json:"run_name"`
	Status             string                 `json:"status"`
	TriggerType        string                 `json:"trigger_type"`
	StartedAt          time.Time              `json:"started_at"`
	CompletedAt        *time.Time             `json:"completed_at"`
	DurationSeconds    *int                   `json:"duration_seconds"`
	RecordsProcessed   int                    `json:"records_processed"`
	RecordsFailed      int                    `json:"records_failed"`
	RecordsSkipped     int                    `json:"records_skipped"`
	ErrorMessage       *string                `json:"error_message"`
	ErrorCategory      *string                `json:"error_category"`
	RetryCount         int                    `json:"retry_count"`
	MaxRetries         int                    `json:"max_retries"`
	RuntimeParameters  map[string]interface{} `json:"runtime_parameters"`
	WorkerID           *string                `json:"worker_id"`
}

func NewClient(db *sql.DB) *Client {
	return &Client{db: db}
}

// GetDueSchedules returns all active schedules that are due for execution
func (c *Client) GetDueSchedules(now time.Time) ([]Schedule, error) {
	query := `
		SELECT s.schedule_id, s.job_id, s.schedule_name, s.cron_expression, s.timezone,
			   s.is_active, s.next_run, s.last_run, s.run_count, s.failure_count
		FROM aquaflow.etl_schedules s
		JOIN aquaflow.etl_jobs_v2 j ON s.job_id = j.job_id
		WHERE s.is_active = true 
		  AND j.is_active = true
		  AND s.next_run IS NOT NULL 
		  AND s.next_run <= $1
		ORDER BY s.next_run ASC
	`

	rows, err := c.db.Query(query, now)
	if err != nil {
		return nil, fmt.Errorf("failed to query due schedules: %w", err)
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var schedule Schedule

		err := rows.Scan(
			&schedule.ScheduleID, &schedule.JobID, &schedule.ScheduleName, &schedule.CronExpression,
			&schedule.Timezone, &schedule.IsActive, &schedule.NextRun, &schedule.LastRun,
			&schedule.RunCount, &schedule.FailureCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}

		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// GetJobForSchedule returns the job definition for a given schedule
func (c *Client) GetJobForSchedule(scheduleID uuid.UUID) (*Job, error) {
	query := `
		SELECT j.job_id, j.job_name, j.job_type, j.description, j.parameters,
			   j.is_active, j.created_at, j.updated_at
		FROM aquaflow.etl_jobs_v2 j
		JOIN aquaflow.etl_schedules s ON j.job_id = s.job_id
		WHERE s.schedule_id = $1
	`

	var job Job
	var paramsJSON []byte

	err := c.db.QueryRow(query, scheduleID).Scan(
		&job.JobID, &job.JobName, &job.JobType, &job.Description,
		&paramsJSON, &job.IsActive, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get job for schedule: %w", err)
	}

	// Parse parameters JSON
	if len(paramsJSON) > 0 {
		if err := json.Unmarshal(paramsJSON, &job.Parameters); err != nil {
			return nil, fmt.Errorf("failed to parse job parameters: %w", err)
		}
	}

	return &job, nil
}

// CreateJobRun creates a new ETL job run from a schedule
func (c *Client) CreateJobRun(schedule Schedule, job Job, scheduledFor time.Time) (*JobRun, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create new run ID
	newRunID := uuid.New()

	// Process dynamic parameters
	processedParams, err := c.processDynamicParameters(job.Parameters, scheduledFor)
	if err != nil {
		return nil, fmt.Errorf("failed to process dynamic parameters: %w", err)
	}

	paramsJSON, err := json.Marshal(processedParams)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	// Insert new job run
	insertQuery := `
		INSERT INTO aquaflow.etl_job_runs (
			run_id, job_id, schedule_id, run_name, status, trigger_type,
			started_at, runtime_parameters
		) VALUES ($1, $2, $3, $4, 'queued', 'scheduled', $5, $6)
	`

	runName := fmt.Sprintf("%s - %s", job.JobName, scheduledFor.Format("2006-01-02 15:04"))

	_, err = tx.Exec(insertQuery,
		newRunID, job.JobID, schedule.ScheduleID, runName, 
		scheduledFor, paramsJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert job run: %w", err)
	}

	// Update schedule statistics
	updateScheduleQuery := `
		UPDATE aquaflow.etl_schedules 
		SET run_count = run_count + 1,
			updated_at = NOW()
		WHERE schedule_id = $1
	`
	_, err = tx.Exec(updateScheduleQuery, schedule.ScheduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return the created job run
	jobRun := &JobRun{
		RunID:             newRunID,
		JobID:             job.JobID,
		ScheduleID:        &schedule.ScheduleID,
		RunName:           runName,
		Status:            "queued",
		TriggerType:       "scheduled",
		StartedAt:         scheduledFor,
		RecordsProcessed:  0,
		RecordsFailed:     0,
		RecordsSkipped:    0,
		RetryCount:        0,
		MaxRetries:        3,
		RuntimeParameters: processedParams,
	}

	return jobRun, nil
}

// UpdateScheduleNextRun updates the next_run time for a schedule
func (c *Client) UpdateScheduleNextRun(scheduleID uuid.UUID, nextRun time.Time) error {
	query := `
		UPDATE aquaflow.etl_schedules 
		SET next_run = $1, updated_at = NOW()
		WHERE schedule_id = $2
	`
	_, err := c.db.Exec(query, nextRun, scheduleID)
	if err != nil {
		return fmt.Errorf("failed to update schedule next run: %w", err)
	}
	return nil
}

// HealthCheck verifies database connectivity
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// GetActiveSchedulesCount returns the count of active schedules
func (c *Client) GetActiveSchedulesCount() (int, error) {
	var count int
	query := `
		SELECT COUNT(*) 
		FROM aquaflow.etl_schedules s
		JOIN aquaflow.etl_jobs_v2 j ON s.job_id = j.job_id
		WHERE s.is_active = true AND j.is_active = true
	`
	err := c.db.QueryRow(query).Scan(&count)
	return count, err
}

// processDynamicParameters processes parameters with dynamic date placeholders
func (c *Client) processDynamicParameters(params map[string]interface{}, scheduledFor time.Time) (map[string]interface{}, error) {
	processed := make(map[string]interface{})

	for key, value := range params {
		switch v := value.(type) {
		case string:
			processed[key] = c.processDynamicDateString(v, scheduledFor)
		default:
			processed[key] = value
		}
	}

	return processed, nil
}

// processDynamicDateString replaces dynamic date placeholders with actual dates
func (c *Client) processDynamicDateString(input string, scheduledFor time.Time) string {
	now := scheduledFor

	replacements := map[string]string{
		"DYNAMIC_WEEK_START": now.AddDate(0, 0, -int(now.Weekday())).Format("2006-01-02"),
		"DYNAMIC_WEEK_END":   now.AddDate(0, 0, 6-int(now.Weekday())).Format("2006-01-02"),
		"DYNAMIC_DAY_START":  now.Format("2006-01-02"),
		"DYNAMIC_DAY_END":    now.Format("2006-01-02"),
		"DYNAMIC_YESTERDAY":  now.AddDate(0, 0, -1).Format("2006-01-02"),
		"DYNAMIC_MONTH_START": time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02"),
		"DYNAMIC_MONTH_END":   time.Date(now.Year(), now.Month()+1, 0, 23, 59, 59, 0, now.Location()).Format("2006-01-02"),
	}

	result := input
	for placeholder, replacement := range replacements {
		result = strings.ReplaceAll(result, placeholder, replacement)
	}

	return result
}