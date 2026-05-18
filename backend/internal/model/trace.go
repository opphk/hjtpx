package model

import (
	"time"
)

type TracePoint struct {
	Timestamp int64   `json:"t"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Event     string  `json:"e"`
}

type TraceData struct {
	Points     []TracePoint `json:"points"`
	TotalTime  int64        `json:"total_time"`
	StartX     float64      `json:"start_x"`
	StartY     float64      `json:"start_y"`
	EndX       float64      `json:"end_x"`
	EndY       float64      `json:"end_y"`
	DeviceInfo string       `json:"device_info"`
}

type TraceFeatures struct {
	ID                int64     `json:"id"`
	SessionID         string    `json:"session_id"`
	TotalTime         int64     `json:"total_time"`
	MoveCount         int       `json:"move_count"`
	AvgSpeed          float64   `json:"avg_speed"`
	MaxSpeed          float64   `json:"max_speed"`
	MinSpeed          float64   `json:"min_speed"`
	SpeedVariance     float64   `json:"speed_variance"`
	MaxAcceleration   float64   `json:"max_acceleration"`
	AvgAcceleration   float64   `json:"avg_acceleration"`
	AccelVariance     float64   `json:"accel_variance"`
	Smoothness        float64   `json:"smoothness"`
	PauseCount        int       `json:"pause_count"`
	TotalDistance     float64   `json:"total_distance"`
	DirectDistance    float64   `json:"direct_distance"`
	PathRatio         float64   `json:"path_ratio"`
	AvgCurvature      float64   `json:"avg_curvature"`
	MaxCurvature      float64   `json:"max_curvature"`
	JitterFrequency   float64   `json:"jitter_frequency"`
	JitterAmplitude   float64   `json:"jitter_amplitude"`
	SpeedChangeRate   float64   `json:"speed_change_rate"`
	DirectionChange   float64   `json:"direction_change"`
	RiskFactors       []string  `json:"risk_factors"`
	CreatedAt         time.Time `json:"created_at"`
}

type TraceScore struct {
	TotalScore  float64            `json:"total_score"`
	SpeedScore  float64            `json:"speed_score"`
	AccelScore  float64            `json:"accel_score"`
	SmoothScore float64            `json:"smooth_score"`
	PauseScore  float64            `json:"pause_score"`
	Features    map[string]float64 `json:"features"`
	RiskFactors []string           `json:"risk_factors"`
}
