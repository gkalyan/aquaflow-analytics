package cron

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

type Parser struct {
	cronParser cron.Parser
}

func NewParser() *Parser {
	// Use standard cron parser with seconds support
	return &Parser{
		cronParser: cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
	}
}

// ParseSchedule parses a cron expression and returns the next execution time
func (p *Parser) ParseSchedule(cronExpr string) (cron.Schedule, error) {
	schedule, err := p.cronParser.Parse(cronExpr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression '%s': %w", cronExpr, err)
	}
	return schedule, nil
}

// NextExecution calculates the next execution time for a cron expression from the given time
func (p *Parser) NextExecution(cronExpr string, from time.Time) (time.Time, error) {
	schedule, err := p.ParseSchedule(cronExpr)
	if err != nil {
		return time.Time{}, err
	}
	
	next := schedule.Next(from)
	return next, nil
}

// IsScheduleDue checks if a scheduled job is due for execution
func (p *Parser) IsScheduleDue(cronExpr string, lastRun time.Time, now time.Time) (bool, error) {
	schedule, err := p.ParseSchedule(cronExpr)
	if err != nil {
		return false, err
	}
	
	// Get the next scheduled time after the last run
	nextScheduled := schedule.Next(lastRun)
	
	// Job is due if the next scheduled time is now or in the past
	return !nextScheduled.After(now), nil
}

// GetNextRun calculates the next run time after the current time
func (p *Parser) GetNextRun(cronExpr string) (time.Time, error) {
	return p.NextExecution(cronExpr, time.Now())
}

// ValidateCronExpression validates if a cron expression is syntactically correct
func (p *Parser) ValidateCronExpression(cronExpr string) error {
	_, err := p.ParseSchedule(cronExpr)
	return err
}

// GetHumanReadableSchedule converts cron expression to human-readable format
func (p *Parser) GetHumanReadableSchedule(cronExpr string) string {
	scheduleMap := map[string]string{
		"*/15 * * * *": "Every 15 minutes",
		"*/30 * * * *": "Every 30 minutes", 
		"0 * * * *":    "Every hour",
		"0 */2 * * *":  "Every 2 hours",
		"0 */6 * * *":  "Every 6 hours",
		"0 2 * * *":    "Daily at 2:00 AM",
		"0 6 * * *":    "Daily at 6:00 AM",
		"0 0 * * 0":    "Weekly on Sunday",
		"0 0 1 * *":    "Monthly on 1st",
	}
	
	if readable, exists := scheduleMap[cronExpr]; exists {
		return readable
	}
	return cronExpr // Return original if no mapping found
}