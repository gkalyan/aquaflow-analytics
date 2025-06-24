package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aquaflow/etl-workers/internal/db"
	"github.com/aquaflow/etl-workers/internal/logger"
)

var (
	ErrNoJobsAvailable = errors.New("no jobs available")
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
)

// ErrorType categorizes errors for retry logic
type ErrorType int

const (
	ErrorTypeTransient ErrorType = iota // Network, timeout - auto retry
	ErrorTypeData                       // Bad data - pause job
	ErrorTypeSystem                     // System error - alert
)

type Processor struct {
	db     *db.Client
	logger *logger.ETLLogger
}

type JobHandler interface {
	Execute(ctx context.Context, job *db.ETLJob) error
}

func NewProcessor(dbClient *db.Client, logger *logger.ETLLogger) *Processor {
	return &Processor{
		db:     dbClient,
		logger: logger,
	}
}

func (p *Processor) ProcessNextJob(ctx context.Context) error {
	// Get next pending job
	job, err := p.db.GetNextPendingJob()
	if err != nil {
		return fmt.Errorf("failed to get next job: %w", err)
	}

	if job == nil {
		return ErrNoJobsAvailable
	}

	// Log job start
	p.logger.LogJobStart(job.BatchID, job.JobName, job.JobType, job.Parameters)
	startTime := time.Now()

	// Select handler based on job type
	var handler JobHandler
	switch job.JobType {
	case "historical_load":
		handler = NewHistoricalLoadJob(p.db, p.logger)
	case "realtime_sync":
		handler = NewRealtimeSyncJob(p.db, p.logger)
	default:
		errMsg := fmt.Sprintf("unknown job type: %s", job.JobType)
		p.logger.Error(job.BatchID, errMsg)
		p.db.UpdateJobStatus(job.BatchID, "failed", 0, 0, &errMsg)
		return fmt.Errorf(errMsg)
	}

	// Execute the job
	if err := handler.Execute(ctx, job); err != nil {
		duration := time.Since(startTime)
		errMsg := err.Error()
		
		// Categorize error
		errorType := p.categorizeError(err)
		
		// Get retry count
		retryCount, _ := p.db.GetJobRetryCount(job.BatchID)
		
		// Handle based on error type
		switch errorType {
		case ErrorTypeTransient:
			// Check retry limit
			if retryCount >= 3 { // Max retries hardcoded for now
				p.logger.Error(job.BatchID, "Max retries exceeded", map[string]interface{}{
					"job_name": job.JobName,
					"retry_count": retryCount,
					"error": errMsg,
				})
				p.db.UpdateJobStatus(job.BatchID, "failed", job.RecordsProcessed, job.RecordsFailed, &errMsg)
			} else {
				// Increment retry count and set back to pending
				p.db.IncrementRetryCount(job.BatchID)
				p.db.UpdateJobStatus(job.BatchID, "pending", job.RecordsProcessed, job.RecordsFailed, &errMsg)
				p.logger.Warn(job.BatchID, "Transient error, will retry", map[string]interface{}{
					"job_name": job.JobName,
					"retry_count": retryCount + 1,
					"error": errMsg,
				})
			}
			
		case ErrorTypeData:
			// Log error with stack trace
			p.logger.LogJobError(job.BatchID, job.JobName, err, true)
			// Mark as failed - requires manual intervention
			p.db.UpdateJobStatus(job.BatchID, "failed", job.RecordsProcessed, job.RecordsFailed, &errMsg)
			
		case ErrorTypeSystem:
			// Log critical error with stack trace
			p.logger.LogJobError(job.BatchID, job.JobName, err, true)
			p.db.UpdateJobStatus(job.BatchID, "failed", job.RecordsProcessed, job.RecordsFailed, &errMsg)
		}
		
		// Log completion even for failed jobs
		p.logger.LogJobComplete(job.BatchID, job.JobName, job.RecordsProcessed, job.RecordsFailed, duration)
		
		return err
	}

	// Log successful completion
	duration := time.Since(startTime)
	p.logger.LogJobComplete(job.BatchID, job.JobName, job.RecordsProcessed, job.RecordsFailed, duration)
	
	return nil
}

// categorizeError determines the type of error for retry logic
func (p *Processor) categorizeError(err error) ErrorType {
	errStr := err.Error()
	
	// Network/connection errors - transient
	if errors.Is(err, context.DeadlineExceeded) || 
	   errors.Is(err, context.Canceled) ||
	   containsAny(errStr, []string{"connection refused", "timeout", "EOF", "broken pipe", "no such host"}) {
		return ErrorTypeTransient
	}
	
	// Data validation errors
	if containsAny(errStr, []string{"invalid", "validation", "bad request", "400", "422"}) {
		return ErrorTypeData
	}
	
	// System errors
	return ErrorTypeSystem
}

func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(strings.ToLower(s), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}