package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gkalyan/aquaflow-analytics/internal/config"
	"github.com/gkalyan/aquaflow-analytics/internal/core/db"
	"github.com/gkalyan/aquaflow-analytics/internal/core/handlers"
	"github.com/gkalyan/aquaflow-analytics/internal/core/middleware"
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

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(cfg.JWTSecret)
	etlHandler := handlers.NewETLHandler(database)
	chatHandler := handlers.NewChatHandler(database)

	// Auth routes (no middleware)
	auth := r.Group("/api/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/logout", authHandler.Logout)
	}

	// API routes (protected)
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "pong", "timestamp": time.Now().Format(time.RFC3339)})
		})

		// User routes
		api.GET("/me", authHandler.GetCurrentUser)

		// Database schema information endpoint
		api.GET("/schema", func(c *gin.Context) {
			var tableCount, coreTableCount int
			
			// Get total table count
			database.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'aquaflow'").Scan(&tableCount)
			
			// Get core table count
			database.QueryRow(`
				SELECT COUNT(*) FROM information_schema.tables 
				WHERE table_schema = 'aquaflow' 
				AND table_name IN ('datasets', 'parameters', 'series', 'numeric_values')
			`).Scan(&coreTableCount)

			c.JSON(200, gin.H{
				"schema": "aquaflow",
				"total_tables": tableCount,
				"core_tables": coreTableCount,
				"status": "ready",
				"timestamp": time.Now().Format(time.RFC3339),
			})
		})

		// Chat endpoints - natural language query processing
		chat := api.Group("/chat")
		{
			chat.POST("", chatHandler.Chat)
			chat.POST("/feedback", chatHandler.Feedback)
			chat.GET("/sessions/:session_id", chatHandler.GetSession)
			chat.DELETE("/sessions/:session_id", chatHandler.ClearSession)
		}

		// Legacy query endpoint (redirect to chat)
		api.POST("/query", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "Please use /api/chat endpoint for natural language queries",
				"status":  "deprecated",
				"new_endpoint": "/api/chat",
			})
		})

		// Morning check template
		api.GET("/morning-check", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "Morning check template - coming soon",
				"status":  "not_implemented",
			})
		})

		// ETL management endpoints
		etl := api.Group("/etl")
		{
			// Legacy endpoints (job runs displayed as jobs)
			etl.GET("/jobs", etlHandler.GetJobs)
			etl.GET("/jobs/:id", etlHandler.GetJobDetails)
			etl.GET("/jobs/:id/logs", etlHandler.GetJobLogs)
			etl.POST("/jobs/:id/restart", etlHandler.RestartJob)
			etl.GET("/logs", etlHandler.GetAllLogs)
			
			// New three-tier architecture endpoints
			etl.GET("/job-definitions", etlHandler.GetJobDefinitions)
			etl.GET("/schedules", etlHandler.GetSchedules)
			etl.GET("/runs", etlHandler.GetJobRuns)
		}
	}

	log.Printf("Starting AquaFlow Analytics API on port %s", cfg.Port)
	log.Printf("Environment: %s", cfg.DatabaseURL)
	log.Printf("Database Schema: %s", cfg.DBSchema)
	
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}