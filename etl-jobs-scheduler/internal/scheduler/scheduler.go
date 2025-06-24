package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aquaflow/etl-jobs-scheduler/internal/cron"
	"github.com/aquaflow/etl-jobs-scheduler/internal/db"
)

type Scheduler struct {
	db         *db.Client
	cronParser *cron.Parser
	logger     *log.Logger
}

type SchedulerStats struct {
	JobsCreated        int
	TemplatesProcessed int
	Errors             int
	LastRunTime        time.Time
}

func NewScheduler(dbClient *db.Client, logger *log.Logger) *Scheduler {
	return &Scheduler{
		db:         dbClient,
		cronParser: cron.NewParser(),
		logger:     logger,
	}
}

// RunSchedulingCycle executes one complete scheduling cycle
func (s *Scheduler) RunSchedulingCycle(ctx context.Context) (*SchedulerStats, error) {
	stats := &SchedulerStats{
		LastRunTime: time.Now(),
	}

	s.logger.Printf("Starting scheduling cycle at %s", stats.LastRunTime.Format(time.RFC3339))

	// Get all due schedules
	dueSchedules, err := s.db.GetDueSchedules(stats.LastRunTime)
	if err != nil {
		stats.Errors++
		return stats, fmt.Errorf("failed to get due schedules: %w", err)
	}

	if len(dueSchedules) == 0 {
		s.logger.Println("No due schedules found")
		return stats, nil
	}

	s.logger.Printf("Found %d due schedules", len(dueSchedules))

	// Process each due schedule
	for _, schedule := range dueSchedules {
		if err := s.processSchedule(ctx, schedule, stats); err != nil {
			s.logger.Printf("ERROR: Failed to process schedule %s: %v", schedule.ScheduleName, err)
			stats.Errors++
			continue
		}
		stats.TemplatesProcessed++
	}

	s.logger.Printf("Scheduling cycle completed: %d jobs created, %d templates processed, %d errors",
		stats.JobsCreated, stats.TemplatesProcessed, stats.Errors)

	return stats, nil
}

// processSchedule handles a single due schedule
func (s *Scheduler) processSchedule(ctx context.Context, schedule db.Schedule, stats *SchedulerStats) error {
	s.logger.Printf("Processing schedule: %s (cron: %s)", schedule.ScheduleName, schedule.CronExpression)

	// Validate that the schedule is actually due
	if schedule.NextRun == nil || schedule.NextRun.After(stats.LastRunTime) {
		return fmt.Errorf("schedule %s is not due yet", schedule.ScheduleName)
	}

	// Get the job definition for this schedule
	job, err := s.db.GetJobForSchedule(schedule.ScheduleID)
	if err != nil {
		return fmt.Errorf("failed to get job for schedule: %w", err)
	}

	// Create job run from schedule
	jobRun, err := s.db.CreateJobRun(schedule, *job, *schedule.NextRun)
	if err != nil {
		return fmt.Errorf("failed to create job run: %w", err)
	}

	s.logger.Printf("Created job run: %s (ID: %s)", jobRun.RunName, jobRun.RunID.String())
	stats.JobsCreated++

	// Calculate next run time
	nextRun, err := s.cronParser.NextExecution(schedule.CronExpression, stats.LastRunTime)
	if err != nil {
		return fmt.Errorf("failed to calculate next run time: %w", err)
	}

	// Update schedule with next run time
	if err := s.db.UpdateScheduleNextRun(schedule.ScheduleID, nextRun); err != nil {
		return fmt.Errorf("failed to update schedule next run: %w", err)
	}

	s.logger.Printf("Updated schedule %s next run to: %s", schedule.ScheduleName, nextRun.Format(time.RFC3339))

	return nil
}

// Start begins the scheduler with the specified interval
func (s *Scheduler) Start(ctx context.Context, checkInterval time.Duration) error {
	s.logger.Printf("Starting ETL Jobs Scheduler with %v check interval", checkInterval)

	// Initial health check
	if err := s.db.HealthCheck(ctx); err != nil {
		return fmt.Errorf("initial database health check failed: %w", err)
	}

	// Log active schedules count
	count, err := s.db.GetActiveSchedulesCount()
	if err != nil {
		s.logger.Printf("WARNING: Failed to get active schedules count: %v", err)
	} else {
		s.logger.Printf("Monitoring %d active schedules", count)
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Run initial cycle
	if _, err := s.RunSchedulingCycle(ctx); err != nil {
		s.logger.Printf("ERROR: Initial scheduling cycle failed: %v", err)
	}

	// Main scheduling loop
	for {
		select {
		case <-ctx.Done():
			s.logger.Println("Scheduler shutdown requested")
			return ctx.Err()
		case <-ticker.C:
			// Run health check periodically
			if err := s.db.HealthCheck(ctx); err != nil {
				s.logger.Printf("WARNING: Database health check failed: %v", err)
				continue
			}

			// Run scheduling cycle
			if _, err := s.RunSchedulingCycle(ctx); err != nil {
				s.logger.Printf("ERROR: Scheduling cycle failed: %v", err)
			}
		}
	}
}

// ValidateTemplateSchedule validates a cron expression for a template
func (s *Scheduler) ValidateTemplateSchedule(cronExpr string) error {
	return s.cronParser.ValidateCronExpression(cronExpr)
}

// GetHumanReadableSchedule returns a human-readable format of a cron expression
func (s *Scheduler) GetHumanReadableSchedule(cronExpr string) string {
	return s.cronParser.GetHumanReadableSchedule(cronExpr)
}

// GetNextRunTime calculates when a template will next run
func (s *Scheduler) GetNextRunTime(cronExpr string) (time.Time, error) {
	return s.cronParser.GetNextRun(cronExpr)
}