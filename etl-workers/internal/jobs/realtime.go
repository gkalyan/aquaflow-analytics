package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/aquaflow/etl-workers/internal/db"
	"github.com/aquaflow/etl-workers/internal/logger"
)

type RealtimeSyncJob struct {
	db     *db.Client
	logger *logger.ETLLogger
}

func NewRealtimeSyncJob(dbClient *db.Client, logger *logger.ETLLogger) *RealtimeSyncJob {
	return &RealtimeSyncJob{
		db:     dbClient,
		logger: logger,
	}
}

func (r *RealtimeSyncJob) Execute(ctx context.Context, job *db.ETLJob) error {
	r.logger.Info(job.BatchID, "Starting realtime data sync", map[string]interface{}{
		"parameters": job.Parameters,
	})

	// Extract parameters
	sourceURL, ok := job.Parameters["source_url"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid source_url parameter")
	}

	seriesIDsRaw, ok := job.Parameters["series_ids"].([]interface{})
	if !ok {
		return fmt.Errorf("missing or invalid series_ids parameter")
	}

	syncInterval := 30
	if si, ok := job.Parameters["sync_interval"].(float64); ok {
		syncInterval = int(si)
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

	// Single sync cycle for all series
	for _, seriesID := range seriesIDs {
		if err := r.syncSeriesData(ctx, job, sourceURL, seriesID); err != nil {
			r.logger.Errorf(job.BatchID, "Failed to sync series %d: %v", seriesID, err)
			totalFailed++
		} else {
			totalProcessed++
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	// Update job status
	status := "completed"
	if totalFailed > 0 {
		status = "completed_with_errors"
	}

	r.logger.Info(job.BatchID, "Realtime sync completed", map[string]interface{}{
		"total_processed": totalProcessed,
		"total_failed":    totalFailed,
		"sync_interval":   syncInterval,
	})

	// Update the next run time if this is a scheduled job
	if job.Parameters["schedule"] != nil {
		r.logger.Info(job.BatchID, "Scheduling next run", map[string]interface{}{
			"next_run_seconds": syncInterval,
		})
	}

	return r.db.UpdateJobStatus(job.BatchID, status, totalProcessed, totalFailed, nil)
}

func (r *RealtimeSyncJob) syncSeriesData(ctx context.Context, job *db.ETLJob, baseURL string, seriesID int) error {
	// Build URL with series_id parameter
	u, _ := url.Parse(baseURL)
	q := u.Query()
	q.Set("series_id", fmt.Sprintf("%d", seriesID))
	u.RawQuery = q.Encode()

	r.logger.Debug(job.BatchID, "Fetching realtime data", map[string]interface{}{
		"url":       u.String(),
		"series_id": seriesID,
	})

	// Fetch data
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Parse response
	var dataPoint DataPoint
	if err := json.NewDecoder(resp.Body).Decode(&dataPoint); err != nil {
		return err
	}

	// Insert single value
	values := []db.NumericValue{{
		Timestamp: dataPoint.Timestamp,
		SeriesID:  dataPoint.SeriesID,
		Value:     dataPoint.Value,
	}}

	if err := r.db.InsertNumericValues(values); err != nil {
		return fmt.Errorf("failed to insert value: %w", err)
	}

	r.logger.Debug(job.BatchID, "Inserted realtime value", map[string]interface{}{
		"series_id": seriesID,
		"value":     dataPoint.Value,
		"timestamp": dataPoint.Timestamp,
	})

	return nil
}