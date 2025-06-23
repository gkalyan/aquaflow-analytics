package db

import (
	"database/sql"
	"encoding/json"
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
		return nil, err
	}
	defer tx.Rollback()

	var job ETLJob
	var paramsJSON []byte

	// Lock the job for processing
	query := `
		SELECT batch_id, job_name, job_type, load_type, status, parameters, 
			   records_processed, records_failed, started_at
		FROM aquaflow.etl_jobs
		WHERE status = 'pending'
		ORDER BY started_at ASC
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
		UPDATE aquaflow.etl_jobs 
		SET status = 'running', started_at = NOW()
		WHERE batch_id = $1
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
	// Use separate queries to avoid parameter type confusion
	query := `
		UPDATE aquaflow.etl_jobs 
		SET status = $2, 
			records_processed = $3,
			records_failed = $4,
			error_message = $5
		WHERE batch_id = $1
	`
	
	// Handle nil error message properly
	var errorParam interface{}
	if errorMsg != nil {
		errorParam = *errorMsg
	} else {
		errorParam = nil
	}
	
	_, err := c.db.Exec(query, batchID, status, recordsProcessed, recordsFailed, errorParam)
	if err != nil {
		return err
	}
	
	// Update completed_at separately for terminal statuses
	if status == "completed" || status == "failed" || status == "completed_with_errors" {
		completedQuery := `UPDATE aquaflow.etl_jobs SET completed_at = NOW() WHERE batch_id = $1`
		_, err = c.db.Exec(completedQuery, batchID)
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