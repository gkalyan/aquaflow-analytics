package jobs

import "time"

type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	SeriesID  int       `json:"series_id"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
}