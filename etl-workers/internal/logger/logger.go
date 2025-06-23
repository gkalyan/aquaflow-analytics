package logger

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

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
	// Log to stdout
	log.Printf("[%s] [%s] %s", batchID.String()[:8], level, message)

	// Log to database
	contextJSON, _ := json.Marshal(context)
	
	query := `
		INSERT INTO aquaflow.etl_job_logs (batch_id, log_level, message, context)
		VALUES ($1, $2, $3, $4)
	`
	
	if _, err := l.db.Exec(query, batchID, string(level), message, contextJSON); err != nil {
		log.Printf("Failed to write log to database: %v", err)
	}
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