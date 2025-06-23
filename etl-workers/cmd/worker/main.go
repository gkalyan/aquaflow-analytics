package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aquaflow/etl-workers/internal/db"
	"github.com/aquaflow/etl-workers/internal/jobs"
	"github.com/aquaflow/etl-workers/internal/logger"
	_ "github.com/lib/pq"
)

func main() {
	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	database, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Initialize components
	dbClient := db.NewClient(database)
	etlLogger := logger.NewETLLogger(database)
	
	// Create job processor
	processor := jobs.NewProcessor(dbClient, etlLogger)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received, stopping worker...")
		cancel()
	}()

	log.Println("ETL Worker starting...")

	// Main processing loop
	ticker := time.NewTicker(5 * time.Second) // Poll every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Worker stopped")
			return
		case <-ticker.C:
			if err := processor.ProcessNextJob(ctx); err != nil {
				if err != jobs.ErrNoJobsAvailable {
					log.Printf("Error processing job: %v", err)
				}
			}
		}
	}
}