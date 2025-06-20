package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
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
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "aquaflow-api",
			"version":   "0.1.0",
			"database":  "connected", // TODO: Add actual DB health check
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Starting AquaFlow Analytics API on port %s", port)
	log.Printf("Environment: %s", os.Getenv("GIN_MODE"))
	
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}