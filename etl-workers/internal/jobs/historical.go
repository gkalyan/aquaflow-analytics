package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/aquaflow/etl-workers/internal/db"
	"github.com/aquaflow/etl-workers/internal/logger"
)

type HistoricalLoadJob struct {
	db     *db.Client
	logger *logger.ETLLogger
}

type HistoricalDataResponse struct {
	Data       []DataPoint `json:"data"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalCount int         `json:"total_count"`
	HasMore    bool        `json:"has_more"`
}


func NewHistoricalLoadJob(dbClient *db.Client, logger *logger.ETLLogger) *HistoricalLoadJob {
	return &HistoricalLoadJob{
		db:     dbClient,
		logger: logger,
	}
}

func (h *HistoricalLoadJob) Execute(ctx context.Context, job *db.ETLJob) error {
	h.logger.Info(job.BatchID, "Starting historical data load", map[string]interface{}{
		"parameters": job.Parameters,
	})

	// Extract parameters
	sourceURL, ok := job.Parameters["source_url"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid source_url parameter")
	}

	startDate, ok := job.Parameters["start_date"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid start_date parameter")
	}

	endDate, ok := job.Parameters["end_date"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid end_date parameter")
	}

	seriesIDsRaw, ok := job.Parameters["series_ids"].([]interface{})
	if !ok {
		return fmt.Errorf("missing or invalid series_ids parameter")
	}

	batchSize := 1000
	if bs, ok := job.Parameters["batch_size"].(float64); ok {
		batchSize = int(bs)
	}

	// Convert series IDs to int slice
	var seriesIDs []int
	for _, id := range seriesIDsRaw {
		if fid, ok := id.(float64); ok {
			seriesIDs = append(seriesIDs, int(fid))
		}
	}

	totalProcessed := 0
	totalFailed := 0

	// Process each series
	for _, seriesID := range seriesIDs {
		h.logger.Info(job.BatchID, "Loading data for series", map[string]interface{}{
			"series_id": seriesID,
			"job_type":  job.JobType,
			"job_name":  job.JobName,
		})

		processed, failed, err := h.loadSeriesData(ctx, job, sourceURL, seriesID, startDate, endDate, batchSize)
		if err != nil {
			h.logger.Error(job.BatchID, "Failed to load series data", map[string]interface{}{
				"series_id": seriesID,
				"job_name":  job.JobName,
				"error":     err.Error(),
			})
			totalFailed += failed
		}

		totalProcessed += processed
		totalFailed += failed

		// Update job progress
		h.db.UpdateJobStatus(job.BatchID, "running", totalProcessed, totalFailed, nil)

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	// Final status update
	status := "completed"
	if totalFailed > 0 {
		status = "completed_with_errors"
	}

	h.logger.Info(job.BatchID, "Historical load completed", map[string]interface{}{
		"total_processed": totalProcessed,
		"total_failed":    totalFailed,
		"status":          status,
	})

	return h.db.UpdateJobStatus(job.BatchID, status, totalProcessed, totalFailed, nil)
}

func (h *HistoricalLoadJob) loadSeriesData(ctx context.Context, job *db.ETLJob, baseURL string, seriesID int, startDate, endDate string, batchSize int) (processed, failed int, err error) {
	page := 1
	hasMore := true

	for hasMore {
		// Build URL with parameters
		u, _ := url.Parse(baseURL)
		q := u.Query()
		q.Set("series_id", fmt.Sprintf("%d", seriesID))
		q.Set("start_date", startDate)
		q.Set("end_date", endDate)
		q.Set("page", fmt.Sprintf("%d", page))
		q.Set("limit", fmt.Sprintf("%d", batchSize))
		u.RawQuery = q.Encode()

		h.logger.Debug(job.BatchID, "Fetching page", map[string]interface{}{
			"url":       u.String(),
			"page":      page,
			"series_id": seriesID,
			"job_name":  job.JobName,
		})

		// Fetch data
		req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if err != nil {
			return processed, failed, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return processed, failed, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return processed, failed, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}

		// Parse response
		var histResp HistoricalDataResponse
		if err := json.NewDecoder(resp.Body).Decode(&histResp); err != nil {
			return processed, failed, err
		}

		// Convert to database format
		var values []db.NumericValue
		for _, dp := range histResp.Data {
			values = append(values, db.NumericValue{
				Timestamp: dp.Timestamp,
				SeriesID:  dp.SeriesID,
				Value:     dp.Value,
			})
		}

		// Insert batch
		if err := h.db.InsertNumericValues(values); err != nil {
			h.logger.Error(job.BatchID, "Failed to insert batch", map[string]interface{}{
				"series_id":    seriesID,
				"page":         page,
				"batch_size":   len(values),
				"job_name":     job.JobName,
				"error":        err.Error(),
			})
			failed += len(values)
		} else {
			processed += len(values)
			h.logger.Info(job.BatchID, "Inserted records successfully", map[string]interface{}{
				"series_id":    seriesID,
				"page":         page,
				"batch_size":   len(values),
				"job_name":     job.JobName,
				"records_inserted": len(values),
			})
		}

		hasMore = histResp.HasMore
		page++

		// Check context cancellation
		select {
		case <-ctx.Done():
			return processed, failed, ctx.Err()
		default:
		}
	}

	return processed, failed, nil
}