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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format"})
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "series_id is required"})
		return
	}

	seriesID, err := strconv.Atoi(seriesIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid series_id"})
		return
	}

	data := h.generator.GenerateRealtimeData(seriesID)
	if data == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "series not found"})
		return
	}

	c.JSON(http.StatusOK, data)
}