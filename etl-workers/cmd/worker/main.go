package main

import (
	"context"
	"database/sql"
	"fmt"
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

func connectWithRetry(dbURL string, maxRetries int) (*sql.DB, error) {
	var database *sql.DB
	var err error
	
	backoff := time.Second
	maxBackoff := 30 * time.Second
	
	for i := 0; i <= maxRetries; i++ {
		log.Printf("Attempting database connection (attempt %d/%d)...", i+1, maxRetries+1)
		
		database, err = sql.Open("postgres", dbURL)
		if err != nil {
			log.Printf("Failed to open database connection: %v", err)
			if i < maxRetries {
				log.Printf("Retrying in %v...", backoff)
				time.Sleep(backoff)
				backoff = backoff * 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}
			return nil, err
		}
		
		// Test the connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = database.PingContext(ctx)
		cancel()
		
		if err != nil {
			log.Printf("Failed to ping database: %v", err)
			database.Close()
			if i < maxRetries {
				log.Printf("Retrying in %v...", backoff)
				time.Sleep(backoff)
				backoff = backoff * 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}
			return nil, err
		}
		
		// Configure connection pool
		database.SetMaxOpenConns(10)
		database.SetMaxIdleConns(5)
		database.SetConnMaxLifetime(time.Hour)
		
		log.Println("Successfully connected to database")
		return database, nil
	}
	
	return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries+1, err)
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	
	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Connect with retry logic
	database, err := connectWithRetry(dbURL, 10)
	if err != nil {
		log.Fatal("Failed to establish database connection:", err)
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

	log.Println("ETL Worker started successfully")
	log.Printf("Worker configuration: PollInterval=5s, MaxConnections=%d", 10)

	// Health check goroutine
	go func() {
		healthTicker := time.NewTicker(30 * time.Second)
		defer healthTicker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-healthTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := database.PingContext(ctx); err != nil {
					log.Printf("WARNING: Database health check failed: %v", err)
				}
				cancel()
			}
		}
	}()

	// Main processing loop
	ticker := time.NewTicker(5 * time.Second) // Poll every 5 seconds
	defer ticker.Stop()

	consecutiveErrors := 0
	maxConsecutiveErrors := 5

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutdown complete")
			return
		case <-ticker.C:
			if err := processor.ProcessNextJob(ctx); err != nil {
				if err != jobs.ErrNoJobsAvailable {
					consecutiveErrors++
					log.Printf("Error processing job (consecutive errors: %d): %v", consecutiveErrors, err)
					
					if consecutiveErrors >= maxConsecutiveErrors {
						log.Printf("CRITICAL: Too many consecutive errors (%d), backing off for 30 seconds", consecutiveErrors)
						time.Sleep(30 * time.Second)
						consecutiveErrors = 0
					}
				}
			} else {
				// Reset error counter on successful processing
				if consecutiveErrors > 0 {
					consecutiveErrors = 0
				}
			}
		}
	}
}