package jobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/aquaflow/etl-workers/internal/db"
	"github.com/aquaflow/etl-workers/internal/logger"
)

var ErrNoJobsAvailable = errors.New("no jobs available")

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

	p.logger.Infof(job.BatchID, "Starting job: %s (type: %s)", job.JobName, job.JobType)

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
		errMsg := err.Error()
		p.logger.Errorf(job.BatchID, "Job failed: %s", errMsg)
		p.db.UpdateJobStatus(job.BatchID, "failed", 0, 0, &errMsg)
		return err
	}

	p.logger.Info(job.BatchID, "Job completed successfully")
	return nil
}