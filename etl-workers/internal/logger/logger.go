package logger

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
)

type ETLLogger struct {
	db *sql.DB
}

type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
)

func NewETLLogger(db *sql.DB) *ETLLogger {
	return &ETLLogger{db: db}
}

func (l *ETLLogger) Log(batchID uuid.UUID, level LogLevel, message string, context map[string]interface{}) {
	// Add timestamp to context
	if context == nil {
		context = make(map[string]interface{})
	}
	context["logged_at"] = time.Now().Format(time.RFC3339)
	
	// Log to stdout with structured format
	contextStr := ""
	if len(context) > 0 {
		if ctxJSON, err := json.Marshal(context); err == nil {
			contextStr = string(ctxJSON)
		}
	}
	log.Printf("[%s] [%s] %s %s", batchID.String()[:8], level, message, contextStr)

	// Log to database - try new table first, fallback to old table
	contextJSON, _ := json.Marshal(context)
	
	// Try new table first (etl_job_logs_v2 with run_id)
	newQuery := `
		INSERT INTO aquaflow.etl_job_logs_v2 (run_id, log_level, message, context, component)
		VALUES ($1, $2, $3, $4, 'worker')
	`
	
	if _, err := l.db.Exec(newQuery, batchID, string(level), message, contextJSON); err != nil {
		// Fallback to old table (etl_job_logs with batch_id)
		oldQuery := `
			INSERT INTO aquaflow.etl_job_logs (batch_id, log_level, message, context)
			VALUES ($1, $2, $3, $4)
		`
		if _, err := l.db.Exec(oldQuery, batchID, string(level), message, contextJSON); err != nil {
			log.Printf("Failed to write log to database (both tables): %v", err)
		}
	}
}

// LogWithStackTrace logs an error with stack trace
func (l *ETLLogger) LogWithStackTrace(batchID uuid.UUID, level LogLevel, message string, err error) {
	context := map[string]interface{}{
		"error": err.Error(),
		"stack_trace": string(debug.Stack()),
	}
	l.Log(batchID, level, message, context)
}

func (l *ETLLogger) Debug(batchID uuid.UUID, message string, context ...map[string]interface{}) {
	ctx := make(map[string]interface{})
	if len(context) > 0 {
		ctx = context[0]
	}
	l.Log(batchID, DEBUG, message, ctx)
}

func (l *ETLLogger) Info(batchID uuid.UUID, message string, context ...map[string]interface{}) {
	ctx := make(map[string]interface{})
	if len(context) > 0 {
		ctx = context[0]
	}
	l.Log(batchID, INFO, message, ctx)
}

func (l *ETLLogger) Warn(batchID uuid.UUID, message string, context ...map[string]interface{}) {
	ctx := make(map[string]interface{})
	if len(context) > 0 {
		ctx = context[0]
	}
	l.Log(batchID, WARN, message, ctx)
}

func (l *ETLLogger) Error(batchID uuid.UUID, message string, context ...map[string]interface{}) {
	ctx := make(map[string]interface{})
	if len(context) > 0 {
		ctx = context[0]
	}
	l.Log(batchID, ERROR, message, ctx)
}

func (l *ETLLogger) Infof(batchID uuid.UUID, format string, args ...interface{}) {
	l.Info(batchID, fmt.Sprintf(format, args...))
}

func (l *ETLLogger) Errorf(batchID uuid.UUID, format string, args ...interface{}) {
	l.Error(batchID, fmt.Sprintf(format, args...))
}

// Job lifecycle logging methods
func (l *ETLLogger) LogJobStart(batchID uuid.UUID, jobName, jobType string, parameters map[string]interface{}) {
	l.Info(batchID, "JOB_STARTED", map[string]interface{}{
		"job_name": jobName,
		"job_type": jobType,
		"parameters": parameters,
		"event": "job_start",
	})
}

func (l *ETLLogger) LogJobProgress(batchID uuid.UUID, jobName string, processed, failed, total int) {
	l.Info(batchID, "JOB_PROGRESS", map[string]interface{}{
		"job_name": jobName,
		"records_processed": processed,
		"records_failed": failed,
		"total_records": total,
		"progress_percent": float64(processed) / float64(total) * 100,
		"event": "job_progress",
	})
}

func (l *ETLLogger) LogJobComplete(batchID uuid.UUID, jobName string, processed, failed int, duration time.Duration) {
	l.Info(batchID, "JOB_COMPLETED", map[string]interface{}{
		"job_name": jobName,
		"records_processed": processed,
		"records_failed": failed,
		"duration_seconds": duration.Seconds(),
		"event": "job_complete",
	})
}

func (l *ETLLogger) LogJobError(batchID uuid.UUID, jobName string, err error, withStackTrace bool) {
	if withStackTrace {
		l.LogWithStackTrace(batchID, ERROR, fmt.Sprintf("JOB_ERROR: %s", jobName), err)
	} else {
		l.Error(batchID, "JOB_ERROR", map[string]interface{}{
			"job_name": jobName,
			"error": err.Error(),
			"event": "job_error",
		})
	}
}