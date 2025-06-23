package main

import (
	"log"
	"os"

	"github.com/aquaflow/demo-data-service/internal/generator"
	"github.com/aquaflow/demo-data-service/internal/handlers"
	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	r := gin.Default()

	// Initialize generator
	gen := generator.NewSCADAGenerator()
	h := handlers.NewDataHandler(gen)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// API routes
	api := r.Group("/api")
	{
		api.GET("/historical", h.GetHistoricalData)
		api.GET("/realtime", h.GetRealtimeData)
	}

	log.Printf("Demo Data Service starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}