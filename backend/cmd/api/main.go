package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gkalyan/aquaflow-analytics/internal/config"
	"github.com/gkalyan/aquaflow-analytics/internal/core/db"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to database
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint - SHIP THIS FIRST!
	r.GET("/health", func(c *gin.Context) {
		dbStatus := "connected"
		if err := database.HealthCheck(); err != nil {
			dbStatus = "error: " + err.Error()
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "aquaflow-api",
			"version":   "0.1.0",
			"database":  dbStatus,
			"schema":    cfg.DBSchema,
		})
	})

	// API routes
	api := r.Group("/api")
	{
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "pong", "timestamp": time.Now().Format(time.RFC3339)})
		})

		// Query endpoints - core functionality for Olivia
		api.POST("/query", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "Query endpoint - coming soon",
				"status":  "not_implemented",
			})
		})

		// Morning check template
		api.GET("/morning-check", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "Morning check template - coming soon",
				"status":  "not_implemented",
			})
		})
	}

	log.Printf("Starting AquaFlow Analytics API on port %s", cfg.Port)
	log.Printf("Environment: %s", cfg.DatabaseURL)
	log.Printf("Database Schema: %s", cfg.DBSchema)
	
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}