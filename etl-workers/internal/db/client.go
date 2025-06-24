package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type Client struct {
	db *sql.DB
}

type ETLJob struct {
	BatchID          uuid.UUID              `json:"batch_id"`
	JobName          string                 `json:"job_name"`
	JobType          string                 `json:"job_type"`
	LoadType         string                 `json:"load_type"`
	Status           string                 `json:"status"`
	Parameters       map[string]interface{} `json:"parameters"`
	RecordsProcessed int                    `json:"records_processed"`
	RecordsFailed    int                    `json:"records_failed"`
	StartedAt        time.Time              `json:"started_at"`
	CompletedAt      *time.Time             `json:"completed_at"`
	ErrorMessage     *string                `json:"error_message"`
}

func NewClient(db *sql.DB) *Client {
	return &Client{db: db}
}

func (c *Client) GetNextPendingJob() (*ETLJob, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var job ETLJob
	var paramsJSON []byte

	// Lock the job run for processing (check both old and new tables for backward compatibility)
	query := `
		SELECT r.run_id as batch_id, r.run_name as job_name, j.job_type, 'scheduled' as load_type, 
			   r.status, COALESCE(r.runtime_parameters, j.parameters) as parameters,
			   r.records_processed, r.records_failed, r.started_at
		FROM aquaflow.etl_job_runs r
		JOIN aquaflow.etl_jobs_v2 j ON r.job_id = j.job_id
		WHERE r.status = 'queued'
		  AND j.is_active = true
		ORDER BY r.started_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	err = tx.QueryRow(query).Scan(
		&job.BatchID, &job.JobName, &job.JobType, &job.LoadType,
		&job.Status, &paramsJSON, &job.RecordsProcessed,
		&job.RecordsFailed, &job.StartedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Parse parameters
	if len(paramsJSON) > 0 {
		if err := json.Unmarshal(paramsJSON, &job.Parameters); err != nil {
			return nil, err
		}
	}

	// Update status to running
	updateQuery := `
		UPDATE aquaflow.etl_job_runs 
		SET status = 'running', started_at = NOW(), updated_at = NOW()
		WHERE run_id = $1
	`
	if _, err := tx.Exec(updateQuery, job.BatchID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &job, nil
}

func (c *Client) UpdateJobStatus(batchID uuid.UUID, status string, recordsProcessed, recordsFailed int, errorMsg *string) error {
	// Try to update job run first (new architecture)
	query := `
		UPDATE aquaflow.etl_job_runs 
		SET status = $2, 
			records_processed = $3,
			records_failed = $4,
			error_message = $5,
			updated_at = NOW()
		WHERE run_id = $1
	`
	
	// Handle nil error message properly
	var errorParam interface{}
	if errorMsg != nil {
		errorParam = *errorMsg
	} else {
		errorParam = nil
	}
	
	result, err := c.db.Exec(query, batchID, status, recordsProcessed, recordsFailed, errorParam)
	if err != nil {
		return err
	}
	
	// Check if any rows were affected (run exists in new table)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	// If no rows affected, try the old table for backward compatibility
	if rowsAffected == 0 {
		oldQuery := `
			UPDATE aquaflow.etl_jobs 
			SET status = $2, 
				records_processed = $3,
				records_failed = $4,
				error_message = $5
			WHERE batch_id = $1
		`
		_, err = c.db.Exec(oldQuery, batchID, status, recordsProcessed, recordsFailed, errorParam)
		if err != nil {
			return err
		}
	}
	
	// Update completed_at separately for terminal statuses
	if status == "completed" || status == "failed" || status == "completed_with_errors" {
		// Try new table first
		completedQuery := `UPDATE aquaflow.etl_job_runs SET completed_at = NOW() WHERE run_id = $1`
		result, err = c.db.Exec(completedQuery, batchID)
		if err != nil {
			return err
		}
		
		// If no rows affected, try old table
		if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
			oldCompletedQuery := `UPDATE aquaflow.etl_jobs SET completed_at = NOW() WHERE batch_id = $1`
			_, err = c.db.Exec(oldCompletedQuery, batchID)
		}
	}
	
	return err
}

func (c *Client) InsertNumericValues(values []NumericValue) error {
	if len(values) == 0 {
		return nil
	}

	// Prepare bulk insert with ON CONFLICT to handle duplicates
	query := `
		INSERT INTO aquaflow.numeric_values (series_id, time_point, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (series_id, time_point, version) DO NOTHING
	`

	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, v := range values {
		if _, err := stmt.Exec(v.SeriesID, v.Timestamp, v.Value); err != nil {
			return err
		}
	}

	return tx.Commit()
}

type NumericValue struct {
	Timestamp time.Time
	SeriesID  int
	Value     float64
}

// HealthCheck verifies database connectivity
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// IncrementRetryCount increments the retry count for a failed job
func (c *Client) IncrementRetryCount(batchID uuid.UUID) error {
	query := `
		UPDATE aquaflow.etl_jobs 
		SET retry_count = COALESCE(retry_count, 0) + 1,
		    last_retry_at = NOW()
		WHERE batch_id = $1
	`
	_, err := c.db.Exec(query, batchID)
	return err
}

// GetJobRetryCount returns the current retry count for a job
func (c *Client) GetJobRetryCount(batchID uuid.UUID) (int, error) {
	var count sql.NullInt64
	query := `SELECT retry_count FROM aquaflow.etl_jobs WHERE batch_id = $1`
	err := c.db.QueryRow(query, batchID).Scan(&count)
	if err != nil {
		return 0, err
	}
	if count.Valid {
		return int(count.Int64), nil
	}
	return 0, nil
}