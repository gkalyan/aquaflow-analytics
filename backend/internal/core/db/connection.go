package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func Connect(databaseURL string) (*DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Ensure we're using the aquaflow schema
	if _, err := db.Exec("SET search_path TO aquaflow, public"); err != nil {
		log.Printf("Warning: Could not set search_path to aquaflow: %v", err)
	}

	log.Println("Successfully connected to database with aquaflow schema")
	return &DB{db}, nil
}

func (db *DB) HealthCheck() error {
	// Test basic connectivity
	if err := db.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Test schema access - try to query a table that should exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'aquaflow'").Scan(&count)
	if err != nil {
		return fmt.Errorf("schema health check failed: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("aquaflow schema appears to be empty or not accessible")
	}

	return nil
}