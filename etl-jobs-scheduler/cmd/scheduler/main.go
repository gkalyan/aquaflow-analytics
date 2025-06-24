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

	"github.com/aquaflow/etl-jobs-scheduler/internal/db"
	"github.com/aquaflow/etl-jobs-scheduler/internal/scheduler"
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
		database.SetMaxOpenConns(5)  // Scheduler doesn't need many connections
		database.SetMaxIdleConns(2)
		database.SetConnMaxLifetime(time.Hour)
		
		log.Println("Successfully connected to database")
		return database, nil
	}
	
	return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries+1, err)
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	
	log.Println("ETL Jobs Scheduler starting up...")
	
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
	schedulerLogger := log.New(os.Stdout, "[SCHEDULER] ", log.LstdFlags)
	schedulerInstance := scheduler.NewScheduler(dbClient, schedulerLogger)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received, stopping scheduler...")
		cancel()
	}()

	// Configuration
	checkInterval := 30 * time.Second // Check for due jobs every 30 seconds
	if intervalEnv := os.Getenv("SCHEDULER_CHECK_INTERVAL"); intervalEnv != "" {
		if parsed, err := time.ParseDuration(intervalEnv); err == nil {
			checkInterval = parsed
		} else {
			log.Printf("WARNING: Invalid SCHEDULER_CHECK_INTERVAL '%s', using default %v", intervalEnv, checkInterval)
		}
	}

	log.Printf("ETL Jobs Scheduler started successfully")
	log.Printf("Configuration: CheckInterval=%v, MaxConnections=%d", checkInterval, 5)

	// Health check goroutine
	go func() {
		healthTicker := time.NewTicker(5 * time.Minute) // Health check every 5 minutes
		defer healthTicker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-healthTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := database.PingContext(ctx); err != nil {
					log.Printf("WARNING: Database health check failed: %v", err)
				} else {
					// Log active templates count
					if count, err := dbClient.GetActiveSchedulesCount(); err == nil {
						log.Printf("Health check OK - monitoring %d active schedules", count)
					}
				}
				cancel()
			}
		}
	}()

	// Start the scheduler
	if err := schedulerInstance.Start(ctx, checkInterval); err != nil {
		if err == context.Canceled {
			log.Println("Scheduler shutdown completed")
		} else {
			log.Printf("Scheduler error: %v", err)
		}
	}
}