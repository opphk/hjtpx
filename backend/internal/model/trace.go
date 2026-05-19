package model

import (
	"time"
)

type TracePoint struct {
	Timestamp int64   `json:"t"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Event     string  `json:"e"`
	Pressure  float64 `json:"pressure,omitempty"`
	TouchSize float64 `json:"touch_size,omitempty"`
	ScrollDelta float64 `json:"scroll_delta,omitempty"`
	Angle     float64 `json:"angle,omitempty"`
	HoldTime  int64   `json:"hold_time,omitempty"`
}

type TraceData struct {
	Points     []TracePoint `json:"points"`
	TotalTime  int64        `json:"total_time"`
	StartX     float64      `json:"start_x"`
	StartY     float64      `json:"start_y"`
	EndX       float64      `json:"end_x"`
	EndY       float64      `json:"end_y"`
	DeviceInfo string       `json:"device_info"`
	ClickData  []ClickInfo  `json:"click_data,omitempty"`
	ScrollData []ScrollInfo `json:"scroll_data,omitempty"`
	PointCount         int     `json:"point_count"`
	TotalDistance     float64 `json:"total_distance"`
	AvgDistance       float64 `json:"avg_distance"`
	AvgSpeed          float64 `json:"avg_speed"`
	SpeedVariance     float64 `json:"speed_variance"`
	MinSpeed          float64 `json:"min_speed"`
	MaxSpeed          float64 `json:"max_speed"`
	DirectionChanges  int     `json:"direction_changes"`
	AvgCurvature      float64 `json:"avg_curvature"`
	CurvatureVariance float64 `json:"curvature_variance"`
}

type ClickInfo struct {
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Timestamp  int64   `json:"timestamp"`
	Pressure   float64 `json:"pressure"`
	HoldTime   int64   `json:"hold_time"`
	ClickType  string  `json:"click_type"`
	IsTargeted bool    `json:"is_targeted"`
}

type ScrollInfo struct {
	Timestamp int64   `json:"timestamp"`
	DeltaX    float64 `json:"delta_x"`
	DeltaY    float64 `json:"delta_y"`
	Velocity  float64 `json:"velocity"`
	Direction string  `json:"direction"`
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
	CurvatureVariance float64   `json:"curvature_variance"`
	CurvatureSkewness float64   `json:"curvature_skewness"`
	CurvatureEntropy  float64   `json:"curvature_entropy"`
	AvgPressure       float64   `json:"avg_pressure"`
	PressureVariance  float64   `json:"pressure_variance"`
	MaxPressure       float64   `json:"max_pressure"`
	MinPressure       float64   `json:"min_pressure"`
	ClickCount        int       `json:"click_count"`
	AvgClickInterval  float64   `json:"avg_click_interval"`
	ClickRegularity   float64   `json:"click_regularity"`
	ClickAreaSize     float64   `json:"click_area_size"`
	TargetedClickRate float64   `json:"targeted_click_rate"`
	ScrollCount       int       `json:"scroll_count"`
	AvgScrollVelocity float64   `json:"avg_scroll_velocity"`
	ScrollRegularity   float64   `json:"scroll_regularity"`
	ScrollDirectionEntropy float64 `json:"scroll_direction_entropy"`
	MovementFluidity  float64   `json:"movement_fluidity"`
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
