package generator

import (
	"math"
	"math/rand"
	"time"
)

type SCADAGenerator struct {
	seriesConfig map[int]SeriesConfig
}

type SeriesConfig struct {
	Name      string
	Unit      string
	BaseValue float64
	MinValue  float64
	MaxValue  float64
	Pattern   PatternType
}

type PatternType int

const (
	PatternDaily PatternType = iota
	PatternSeasonal
	PatternOperational
	PatternConstant
)

type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	SeriesID  int       `json:"series_id"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
}

func NewSCADAGenerator() *SCADAGenerator {
	return &SCADAGenerator{
		seriesConfig: map[int]SeriesConfig{
			// Updated to match database series IDs 9-20
			9:  {Name: "Main Canal Flow Rate", Unit: "CFS", BaseValue: 1000, MinValue: 800, MaxValue: 1200, Pattern: PatternDaily},
			10: {Name: "Don Pedro Reservoir Level", Unit: "feet", BaseValue: 800, MinValue: 780, MaxValue: 820, Pattern: PatternSeasonal},
			11: {Name: "Pump Station 3 Pressure", Unit: "PSI", BaseValue: 50, MinValue: 40, MaxValue: 60, Pattern: PatternOperational},
			12: {Name: "Main Canal Temperature", Unit: "°F", BaseValue: 65, MinValue: 50, MaxValue: 85, Pattern: PatternDaily},
			13: {Name: "Gate 12 Position", Unit: "%", BaseValue: 75, MinValue: 0, MaxValue: 100, Pattern: PatternOperational},
			14: {Name: "Pump Station 1 Flow", Unit: "CFS", BaseValue: 500, MinValue: 300, MaxValue: 700, Pattern: PatternOperational},
			15: {Name: "North Branch Flow", Unit: "CFS", BaseValue: 400, MinValue: 200, MaxValue: 600, Pattern: PatternDaily},
			16: {Name: "Water Quality pH", Unit: "pH", BaseValue: 7.5, MinValue: 6.5, MaxValue: 8.5, Pattern: PatternConstant},
			17: {Name: "Pump Station 2 Status", Unit: "boolean", BaseValue: 1, MinValue: 0, MaxValue: 1, Pattern: PatternOperational},
			18: {Name: "Reservoir Inflow", Unit: "CFS", BaseValue: 1100, MinValue: 800, MaxValue: 1500, Pattern: PatternSeasonal},
			19: {Name: "System Efficiency", Unit: "%", BaseValue: 85, MinValue: 70, MaxValue: 95, Pattern: PatternOperational},
			20: {Name: "Turbidity Level", Unit: "NTU", BaseValue: 2.5, MinValue: 0.5, MaxValue: 5.0, Pattern: PatternConstant},
		},
	}
}

func (g *SCADAGenerator) GenerateHistoricalData(seriesID int, start, end time.Time, interval time.Duration) []DataPoint {
	config, exists := g.seriesConfig[seriesID]
	if !exists {
		return nil
	}

	var data []DataPoint
	current := start

	for current.Before(end) || current.Equal(end) {
		value := g.generateValue(config, current)
		data = append(data, DataPoint{
			Timestamp: current,
			SeriesID:  seriesID,
			Value:     value,
			Unit:      config.Unit,
		})
		current = current.Add(interval)
	}

	return data
}

func (g *SCADAGenerator) GenerateRealtimeData(seriesID int) *DataPoint {
	config, exists := g.seriesConfig[seriesID]
	if !exists {
		return nil
	}

	now := time.Now()
	value := g.generateValue(config, now)

	return &DataPoint{
		Timestamp: now,
		SeriesID:  seriesID,
		Value:     value,
		Unit:      config.Unit,
	}
}

func (g *SCADAGenerator) generateValue(config SeriesConfig, t time.Time) float64 {
	var value float64

	switch config.Pattern {
	case PatternDaily:
		// Daily sinusoidal pattern
		hourOfDay := float64(t.Hour()) + float64(t.Minute())/60.0
		dailyFactor := math.Sin((hourOfDay - 6) * math.Pi / 12) // Peak at noon
		amplitude := (config.MaxValue - config.MinValue) / 2
		value = config.BaseValue + amplitude*dailyFactor*0.3
		
	case PatternSeasonal:
		// Seasonal pattern with slow changes
		dayOfYear := float64(t.YearDay())
		seasonalFactor := math.Sin((dayOfYear - 80) * 2 * math.Pi / 365) // Peak in summer
		amplitude := (config.MaxValue - config.MinValue) / 2
		value = config.BaseValue + amplitude*seasonalFactor*0.5
		
	case PatternOperational:
		// Operational pattern with state changes
		// Simulate pump cycles, gate operations, etc.
		hourOfDay := t.Hour()
		if hourOfDay >= 6 && hourOfDay <= 18 { // Daytime operations
			value = config.BaseValue + (config.MaxValue-config.BaseValue)*0.7
		} else {
			value = config.BaseValue - (config.BaseValue-config.MinValue)*0.3
		}
		
	case PatternConstant:
		// Relatively stable with small variations
		value = config.BaseValue
	}

	// Add random noise
	noise := (rand.Float64() - 0.5) * (config.MaxValue - config.MinValue) * 0.05
	value += noise

	// Ensure within bounds
	if value < config.MinValue {
		value = config.MinValue
	} else if value > config.MaxValue {
		value = config.MaxValue
	}

	// Round to reasonable precision
	return math.Round(value*100) / 100
}