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
	Points              []TracePoint `json:"points"`
	TotalTime           int64        `json:"total_time"`
	StartX              float64      `json:"start_x"`
	StartY              float64      `json:"start_y"`
	EndX                float64      `json:"end_x"`
	EndY                float64      `json:"end_y"`
	DeviceInfo          string       `json:"device_info"`
	ClickData           []ClickInfo  `json:"click_data,omitempty"`
	ScrollData          []ScrollInfo `json:"scroll_data,omitempty"`
	PointCount          int          `json:"point_count"`
	TotalDistance       float64      `json:"total_distance"`
	AvgDistance         float64      `json:"avg_distance"`
	AvgSpeed            float64      `json:"avg_speed"`
	SpeedVariance       float64      `json:"speed_variance"`
	MinSpeed            float64      `json:"min_speed"`
	MaxSpeed            float64      `json:"max_speed"`
	DirectionChanges    int          `json:"direction_changes"`
	AvgCurvature        float64      `json:"avg_curvature"`
	CurvatureVariance   float64      `json:"curvature_variance"`
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

type JitterDetectionResult struct {
	JitterCount       int     `json:"jitter_count"`
	JitterRatio       float64 `json:"jitter_ratio"`
	AvgJitterAmplitude float64 `json:"avg_jitter_amplitude"`
	MaxJitterAmplitude float64 `json:"max_jitter_amplitude"`
	JitterFrequency   float64 `json:"jitter_frequency"`
	IsJittery        bool     `json:"is_jittery"`
	JitterScore       float64 `json:"jitter_score"`
	JitterPositions   []int   `json:"jitter_positions"`
}

type TrajectoryCurvatureResult struct {
	AvgCurvature       float64 `json:"avg_curvature"`
	MaxCurvature       float64 `json:"max_curvature"`
	MinCurvature       float64 `json:"min_curvature"`
	CurvatureVariance  float64 `json:"curvature_variance"`
	CurvatureSkewness  float64 `json:"curvature_skewness"`
	CurvatureKurtosis  float64 `json:"curvature_kurtosis"`
	CurvatureEntropy   float64 `json:"curvature_entropy"`
	SharpTurnCount     int     `json:"sharp_turn_count"`
	SmoothTurnCount    int     `json:"smooth_turn_count"`
	DirectionChanges   int     `json:"direction_changes"`
	CurvatureScore     float64 `json:"curvature_score"`
}

type SpeedCurveFitResult struct {
	Coefficients      []float64 `json:"coefficients"`
	FittedSpeeds      []float64 `json:"fitted_speeds"`
	Residuals         []float64 `json:"residuals"`
	RMSE              float64   `json:"rmse"`
	R2Score           float64   `json:"r2_score"`
	Degree            int       `json:"degree"`
	FittedCurvePoints []float64 `json:"fitted_curve_points"`
	SpeedFluctuation  float64   `json:"speed_fluctuation"`
	AccelerationPattern string  `json:"acceleration_pattern"`
}

type TrajectorySmoothnessResult struct {
	SmoothnessScore   float64 `json:"smoothness_score"`
	AvgAngularChange  float64 `json:"avg_angular_change"`
	MaxAngularChange  float64 `json:"max_angular_change"`
	AngularVariance   float64 `json:"angular_variance"`
	LinearDeviation   float64 `json:"linear_deviation"`
	PathEfficiency    float64 `json:"path_efficiency"`
	MovementContinuity float64 `json:"movement_continuity"`
	SmoothRatio       float64 `json:"smooth_ratio"`
	RaggedRatio       float64 `json:"ragged_ratio"`
	OverallFluidity   float64 `json:"overall_fluidity"`
}

type EnhancedTrajectoryAnalysis struct {
	JitterResult       *JitterDetectionResult        `json:"jitter_result"`
	CurvatureResult    *TrajectoryCurvatureResult   `json:"curvature_result"`
	SpeedFitResult     *SpeedCurveFitResult          `json:"speed_fit_result"`
	SmoothnessResult   *TrajectorySmoothnessResult   `json:"smoothness_result"`
	OverallScore       float64                       `json:"overall_score"`
	AnomalyIndicators  []string                      `json:"anomaly_indicators"`
	ConfidenceLevel    float64                        `json:"confidence_level"`
}
