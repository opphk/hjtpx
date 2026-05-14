package config

import "time"

type RiskConfig struct {
	SlideSpeedThresholdFast   time.Duration
	SlideSpeedThresholdSlow   time.Duration
	SmoothnessThreshold       float64
	JitterThreshold           float64
	MaxFailureCount           int
	CriticalFailureCount      int
	BlockDuration             time.Duration
	HighFrequencyThreshold    int64
	RiskScoreThresholds       RiskScoreThresholds
}

type RiskScoreThresholds struct {
	Low      int
	Medium   int
	High     int
	Critical int
}

func DefaultRiskConfig() *RiskConfig {
	return &RiskConfig{
		SlideSpeedThresholdFast:   1 * time.Second,
		SlideSpeedThresholdSlow:   30 * time.Second,
		SmoothnessThreshold:       0.95,
		JitterThreshold:           0.1,
		MaxFailureCount:           3,
		CriticalFailureCount:      5,
		BlockDuration:             30 * time.Minute,
		HighFrequencyThreshold:    100,
		RiskScoreThresholds: RiskScoreThresholds{
			Low:      0,
			Medium:   25,
			High:     50,
			Critical: 80,
		},
	}
}
