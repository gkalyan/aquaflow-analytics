package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/aquaflow/demo-data-service/internal/generator"
	"github.com/gin-gonic/gin"
)

type DataHandler struct {
	generator *generator.SCADAGenerator
}

func NewDataHandler(gen *generator.SCADAGenerator) *DataHandler {
	return &DataHandler{
		generator: gen,
	}
}

type HistoricalRequest struct {
	SeriesID  int    `form:"series_id" binding:"required"`
	StartDate string `form:"start_date" binding:"required"`
	EndDate   string `form:"end_date" binding:"required"`
	Page      int    `form:"page,default=1"`
	Limit     int    `form:"limit,default=1000"`
}

type HistoricalResponse struct {
	Data       []generator.DataPoint `json:"data"`
	Page       int                   `json:"page"`
	Limit      int                   `json:"limit"`
	TotalCount int                   `json:"total_count"`
	HasMore    bool                  `json:"has_more"`
}

func (h *DataHandler) GetHistoricalData(c *gin.Context) {
	var req HistoricalRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request parameters",
			"details": err.Error(),
		})
		return
	}

	// Validate series ID range (1-12 for water operations)
	if req.SeriesID < 1 || req.SeriesID > 12 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid series_id",
			"details": "series_id must be between 1 and 12",
		})
		return
	}

	// Validate pagination parameters
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 10000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid limit",
			"details": "limit must be between 1 and 10000",
		})
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid start_date format",
			"details": "use YYYY-MM-DD format",
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid end_date format", 
			"details": "use YYYY-MM-DD format",
		})
		return
	}

	// Validate date range
	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid date range",
			"details": "end_date must be after start_date",
		})
		return
	}

	// Limit date range to prevent excessive data generation
	maxRange := 365 * 24 * time.Hour // 1 year
	if endDate.Sub(startDate) > maxRange {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Date range too large",
			"details": "maximum range is 365 days",
		})
		return
	}

	// Generate data with 15-minute intervals
	interval := 15 * time.Minute
	allData := h.generator.GenerateHistoricalData(req.SeriesID, startDate, endDate, interval)

	// Implement pagination
	totalCount := len(allData)
	startIdx := (req.Page - 1) * req.Limit
	endIdx := startIdx + req.Limit

	if startIdx >= totalCount {
		c.JSON(http.StatusOK, HistoricalResponse{
			Data:       []generator.DataPoint{},
			Page:       req.Page,
			Limit:      req.Limit,
			TotalCount: totalCount,
			HasMore:    false,
		})
		return
	}

	if endIdx > totalCount {
		endIdx = totalCount
	}

	paginatedData := allData[startIdx:endIdx]

	c.JSON(http.StatusOK, HistoricalResponse{
		Data:       paginatedData,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalCount: totalCount,
		HasMore:    endIdx < totalCount,
	})
}

func (h *DataHandler) GetRealtimeData(c *gin.Context) {
	seriesIDStr := c.Query("series_id")
	if seriesIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required parameter",
			"details": "series_id is required",
		})
		return
	}

	seriesID, err := strconv.Atoi(seriesIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid series_id format",
			"details": "series_id must be a valid integer",
		})
		return
	}

	// Validate series ID range
	if seriesID < 1 || seriesID > 12 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid series_id",
			"details": "series_id must be between 1 and 12",
		})
		return
	}

	data := h.generator.GenerateRealtimeData(seriesID)
	if data == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Series not found",
			"details": "No data available for the specified series_id",
		})
		return
	}

	c.JSON(http.StatusOK, data)
}