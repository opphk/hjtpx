package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"
)

type BiometricsV15Service struct {
	profiles map[string]*MultimodalBiometricProfile
}

type MultimodalBiometricProfile struct {
	UserID               string                `json:"user_id"`
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at"`
	MousePressureProfile *MousePressureProfile `json:"mouse_pressure_profile,omitempty"`
	TouchForceProfile    *TouchForceProfile    `json:"touch_force_profile,omitempty"`
	EyeTrackingProfile   *EyeTrackingProfile   `json:"eye_tracking_profile,omitempty"`
	VerificationCount    int                   `json:"verification_count"`
	ConfidenceScore      float64               `json:"confidence_score"`
	FeatureVector        []float64             `json:"feature_vector"`
}

type MousePressureProfile struct {
	AveragePressure     float64 `json:"average_pressure"`
	PressureStdDev      float64 `json:"pressure_std_dev"`
	MaxPressure         float64 `json:"max_pressure"`
	MinPressure         float64 `json:"min_pressure"`
	PressureRange       float64 `json:"pressure_range"`
	PressureSkewness    float64 `json:"pressure_skewness"`
	PressureKurtosis    float64 `json:"pressure_kurtosis"`
	AvgForce            float64 `json:"avg_force"`
	ForceStdDev         float64 `json:"force_std_dev"`
	DownPressureAvg     float64 `json:"down_pressure_avg"`
	UpPressureAvg       float64 `json:"up_pressure_avg"`
	AvgSpeed            float64 `json:"avg_speed"`
	SpeedStd            float64 `json:"speed_std"`
	MaxSpeed            float64 `json:"max_speed"`
	AvgAcceleration     float64 `json:"avg_acceleration"`
	AvgJerk             float64 `json:"avg_jerk"`
	MovementEntropy     float64 `json:"movement_entropy"`
	DirectionHorizontal float64 `json:"direction_horizontal"`
	DirectionVertical   float64 `json:"direction_vertical"`
	ClickCount          int     `json:"click_count"`
	AvgClickDuration    float64 `json:"avg_click_duration"`
	DragCount           int     `json:"drag_count"`
	AvgDragPressure     float64 `json:"avg_drag_pressure"`
}

type TouchForceProfile struct {
	TouchCount       int     `json:"touch_count"`
	AvgForce         float64 `json:"avg_force"`
	ForceStdDev      float64 `json:"force_std_dev"`
	MaxForce         float64 `json:"max_force"`
	MinForce         float64 `json:"min_force"`
	ForceRange       float64 `json:"force_range"`
	ForceSkewness    float64 `json:"force_skewness"`
	AvgPressure      float64 `json:"avg_pressure"`
	PressureStdDev   float64 `json:"pressure_std_dev"`
	AvgSpeed         float64 `json:"avg_speed"`
	SpeedStd         float64 `json:"speed_std"`
	SwipeCount       int     `json:"swipe_count"`
	DirectionEntropy float64 `json:"direction_entropy"`
	AvgSwipeSpeed    float64 `json:"avg_swipe_speed"`
	AvgSwipeForce    float64 `json:"avg_swipe_force"`
	AvgAngle         float64 `json:"avg_angle"`
	AvgDistance      float64 `json:"avg_distance"`
	AvgDuration      float64 `json:"avg_duration"`
	PinchCount       int     `json:"pinch_count"`
	AvgPinchScale    float64 `json:"avg_pinch_scale"`
	AvgPinchRotation float64 `json:"avg_pinch_rotation"`
}

type EyeTrackingProfile struct {
	GazeCount              int     `json:"gaze_count"`
	AvgX                   float64 `json:"avg_x"`
	AvgY                   float64 `json:"avg_y"`
	XStd                   float64 `json:"x_std"`
	YStd                   float64 `json:"y_std"`
	CoverageArea           float64 `json:"coverage_area"`
	AvgPupilSize           float64 `json:"avg_pupil_size"`
	PupilStd               float64 `json:"pupil_std"`
	BlinkCount             int     `json:"blink_count"`
	BlinkRate              float64 `json:"blink_rate"`
	AvgBlinkDuration       float64 `json:"avg_blink_duration"`
	AvgBlinkInterval       float64 `json:"avg_blink_interval"`
	FixationCount          int     `json:"fixation_count"`
	AvgFixationDuration    float64 `json:"avg_fixation_duration"`
	AvgDispersion          float64 `json:"avg_dispersion"`
	SaccadeCount           int     `json:"saccade_count"`
	AvgSaccadeSpeed        float64 `json:"avg_saccade_speed"`
	MaxSaccadeSpeed        float64 `json:"max_saccade_speed"`
	DwellCount             int     `json:"dwell_count"`
	AvgDwellDuration       float64 `json:"avg_dwell_duration"`
	LongestDwell           float64 `json:"longest_dwell"`
	AttentionRatio         float64 `json:"attention_ratio"`
	ScanPatternTopLeft     float64 `json:"scan_pattern_top_left"`
	ScanPatternTopRight    float64 `json:"scan_pattern_top_right"`
	ScanPatternBottomLeft  float64 `json:"scan_pattern_bottom_left"`
	ScanPatternBottomRight float64 `json:"scan_pattern_bottom_right"`
}

type MultimodalBiometricData struct {
	SessionID          string             `json:"session_id"`
	UserID             string             `json:"user_id"`
	CollectionDuration int64              `json:"collection_duration"`
	Timestamp          int64              `json:"timestamp"`
	MousePressure      *MousePressureData `json:"mouse_pressure,omitempty"`
	TouchForce         *TouchForceData    `json:"touch_force,omitempty"`
	EyeTracking        *EyeTrackingData   `json:"eye_tracking,omitempty"`
	DeviceInfo         *DeviceInfo        `json:"device_info,omitempty"`
}

type MousePressureData struct {
	PressureData     []PressurePoint   `json:"pressure_data"`
	PressureAnalysis *PressureAnalysis `json:"pressure_analysis,omitempty"`
	ClickAnalysis    *ClickAnalysis    `json:"click_analysis,omitempty"`
	DragAnalysis     *DragAnalysis     `json:"drag_analysis,omitempty"`
	MovementAnalysis *MovementAnalysis `json:"movement_analysis,omitempty"`
	Timestamp        int64             `json:"timestamp"`
}

type PressurePoint struct {
	Type      string  `json:"type"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Timestamp int64   `json:"timestamp"`
	Pressure  float64 `json:"pressure"`
	Force     float64 `json:"force"`
}

type PressureAnalysis struct {
	AveragePressure  float64 `json:"average_pressure"`
	PressureStd      float64 `json:"pressure_std"`
	AverageForce     float64 `json:"average_force"`
	ForceStd         float64 `json:"force_std"`
	MaxPressure      float64 `json:"max_pressure"`
	MinPressure      float64 `json:"min_pressure"`
	DownPressureAvg  float64 `json:"down_pressure_avg"`
	UpPressureAvg    float64 `json:"up_pressure_avg"`
	PressureRange    float64 `json:"pressure_range"`
	PressureSkewness float64 `json:"pressure_skewness"`
	PressureKurtosis float64 `json:"pressure_kurtosis"`
}

type ClickAnalysis struct {
	ClickCount        int     `json:"click_count"`
	AvgClickDuration  float64 `json:"avg_click_duration"`
	ClickDurationStd  float64 `json:"click_duration_std"`
	AvgDistance       float64 `json:"avg_distance"`
	AvgStartPressure  float64 `json:"avg_start_pressure"`
	AvgEndPressure    float64 `json:"avg_end_pressure"`
	PressureChangeAvg float64 `json:"pressure_change_avg"`
}

type DragAnalysis struct {
	DragCount       int     `json:"drag_count"`
	AvgDragDuration float64 `json:"avg_drag_duration"`
	AvgDragDistance float64 `json:"avg_drag_distance"`
	AvgDragSpeed    float64 `json:"avg_drag_speed"`
	AvgDragPressure float64 `json:"avg_drag_pressure"`
	DragPressureStd float64 `json:"drag_pressure_std"`
}

type MovementAnalysis struct {
	AvgSpeed            float64 `json:"avg_speed"`
	SpeedStd            float64 `json:"speed_std"`
	MaxSpeed            float64 `json:"max_speed"`
	MinSpeed            float64 `json:"min_speed"`
	AvgAcceleration     float64 `json:"avg_acceleration"`
	AccelerationStd     float64 `json:"acceleration_std"`
	AvgJerk             float64 `json:"avg_jerk"`
	JerkStd             float64 `json:"jerk_std"`
	MovementEntropy     float64 `json:"movement_entropy"`
	DirectionHorizontal float64 `json:"direction_horizontal"`
	DirectionVertical   float64 `json:"direction_vertical"`
}

type TouchForceData struct {
	TouchEvents        []TouchEvent        `json:"touch_events"`
	GestureEvents      []GestureEvent      `json:"gesture_events"`
	SwipeEvents        []SwipeEvent        `json:"swipe_events"`
	PinchEvents        []PinchEvent        `json:"pinch_events"`
	ForceAnalysis      *TouchForceAnalysis `json:"force_analysis,omitempty"`
	SwipeAnalysis      *SwipeAnalysis      `json:"swipe_analysis,omitempty"`
	MultitouchAnalysis *MultiTouchAnalysis `json:"multitouch_analysis,omitempty"`
	Timestamp          int64               `json:"timestamp"`
}

type TouchEvent struct {
	Type      string  `json:"type"`
	ID        int     `json:"id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Force     float64 `json:"force"`
	Pressure  float64 `json:"pressure"`
	Speed     float64 `json:"speed,omitempty"`
	Timestamp int64   `json:"timestamp"`
	Duration  int64   `json:"duration,omitempty"`
}

type GestureEvent struct {
	Type          string  `json:"type"`
	Scale         float64 `json:"scale"`
	Rotation      float64 `json:"rotation"`
	ScaleDelta    float64 `json:"scale_delta,omitempty"`
	RotationDelta float64 `json:"rotation_delta,omitempty"`
	TotalScale    float64 `json:"total_scale,omitempty"`
	TotalRotation float64 `json:"total_rotation,omitempty"`
	Timestamp     int64   `json:"timestamp"`
}

type SwipeEvent struct {
	StartX    float64 `json:"start_x"`
	StartY    float64 `json:"start_y"`
	EndX      float64 `json:"end_x"`
	EndY      float64 `json:"end_y"`
	Distance  float64 `json:"distance"`
	Duration  float64 `json:"duration"`
	Speed     float64 `json:"speed"`
	Direction string  `json:"direction"`
	Angle     float64 `json:"angle"`
	AvgForce  float64 `json:"avg_force"`
	Timestamp int64   `json:"timestamp"`
}

type PinchEvent struct {
	ScaleFactor float64 `json:"scale_factor"`
	Rotation    float64 `json:"rotation"`
	Timestamp   int64   `json:"timestamp"`
}

type TouchForceAnalysis struct {
	TouchCount    int     `json:"touch_count"`
	AvgForce      float64 `json:"avg_force"`
	ForceStdDev   float64 `json:"force_std_dev"`
	MaxForce      float64 `json:"max_force"`
	MinForce      float64 `json:"min_force"`
	ForceRange    float64 `json:"force_range"`
	ForceSkewness float64 `json:"force_skewness"`
	AvgPressure   float64 `json:"avg_pressure"`
	PressureStd   float64 `json:"pressure_std"`
	AvgSpeed      float64 `json:"avg_speed"`
	SpeedStd      float64 `json:"speed_std"`
}

type SwipeAnalysis struct {
	SwipeCount       int            `json:"swipe_count"`
	Directions       map[string]int `json:"directions"`
	DirectionEntropy float64        `json:"direction_entropy"`
	AvgSpeed         float64        `json:"avg_speed"`
	SpeedStd         float64        `json:"speed_std"`
	AvgForce         float64        `json:"avg_force"`
	ForceStd         float64        `json:"force_std"`
	AvgAngle         float64        `json:"avg_angle"`
	AngleStd         float64        `json:"angle_std"`
	AvgDistance      float64        `json:"avg_distance"`
	AvgDuration      float64        `json:"avg_duration"`
}

type MultiTouchAnalysis struct {
	GestureCount     int     `json:"gesture_count"`
	PinchCount       int     `json:"pinch_count"`
	AvgPinchScale    float64 `json:"avg_pinch_scale"`
	AvgPinchRotation float64 `json:"avg_pinch_rotation"`
	MaxPinchScale    float64 `json:"max_pinch_scale"`
	MinPinchScale    float64 `json:"min_pinch_scale"`
}

type EyeTrackingData struct {
	GazeData         []GazePoint       `json:"gaze_data"`
	BlinkData        []BlinkEvent      `json:"blink_data"`
	FixationData     []FixationPoint   `json:"fixation_data"`
	SaccadeData      []SaccadeEvent    `json:"saccade_data"`
	DwellData        []DwellEvent      `json:"dwell_data"`
	FocusData        []FocusEvent      `json:"focus_data"`
	GazeAnalysis     *GazeAnalysis     `json:"gaze_analysis,omitempty"`
	BlinkAnalysis    *BlinkAnalysis    `json:"blink_analysis,omitempty"`
	FixationAnalysis *FixationAnalysis `json:"fixation_analysis,omitempty"`
	SaccadeAnalysis  *SaccadeAnalysis  `json:"saccade_analysis,omitempty"`
	DwellAnalysis    *DwellAnalysis    `json:"dwell_analysis,omitempty"`
	FocusAnalysis    *FocusAnalysis    `json:"focus_analysis,omitempty"`
	Timestamp        int64             `json:"timestamp"`
}

type GazePoint struct {
	Type       string  `json:"type"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	PupilSize  float64 `json:"pupil_size,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	Timestamp  int64   `json:"timestamp"`
}

type BlinkEvent struct {
	Duration  int64 `json:"duration"`
	Timestamp int64 `json:"timestamp"`
}

type FixationPoint struct {
	CentroidX    float64 `json:"centroid_x"`
	CentroidY    float64 `json:"centroid_y"`
	Duration     float64 `json:"duration"`
	PointCount   int     `json:"point_count"`
	Dispersion   float64 `json:"dispersion"`
	AvgPupilSize float64 `json:"avg_pupil_size"`
	Timestamp    int64   `json:"timestamp"`
}

type SaccadeEvent struct {
	StartX     float64 `json:"start_x"`
	StartY     float64 `json:"start_y"`
	EndX       float64 `json:"end_x"`
	EndY       float64 `json:"end_y"`
	Speed      float64 `json:"speed"`
	Duration   float64 `json:"duration"`
	TargetArea string  `json:"target_area"`
	Timestamp  int64   `json:"timestamp"`
}

type DwellEvent struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Duration  float64 `json:"duration"`
	Target    string  `json:"target"`
	Timestamp int64   `json:"timestamp"`
}

type FocusEvent struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
	PageTime  int64  `json:"page_time"`
}

type GazeAnalysis struct {
	GazeCount              int     `json:"gaze_count"`
	AvgX                   float64 `json:"avg_x"`
	AvgY                   float64 `json:"avg_y"`
	XStd                   float64 `json:"x_std"`
	YStd                   float64 `json:"y_std"`
	CoverageArea           float64 `json:"coverage_area"`
	AvgPupilSize           float64 `json:"avg_pupil_size"`
	PupilStd               float64 `json:"pupil_std"`
	ScanPatternTopLeft     float64 `json:"scan_pattern_top_left"`
	ScanPatternTopRight    float64 `json:"scan_pattern_top_right"`
	ScanPatternBottomLeft  float64 `json:"scan_pattern_bottom_left"`
	ScanPatternBottomRight float64 `json:"scan_pattern_bottom_right"`
}

type BlinkAnalysis struct {
	BlinkCount       int     `json:"blink_count"`
	BlinkRate        float64 `json:"blink_rate"`
	AvgBlinkDuration float64 `json:"avg_blink_duration"`
	BlinkDurationStd float64 `json:"blink_duration_std"`
	AvgInterval      float64 `json:"avg_interval"`
	IntervalStd      float64 `json:"interval_std"`
	MinDuration      float64 `json:"min_duration"`
	MaxDuration      float64 `json:"max_duration"`
}

type FixationAnalysis struct {
	FixationCount  int     `json:"fixation_count"`
	AvgDuration    float64 `json:"avg_duration"`
	DurationStd    float64 `json:"duration_std"`
	AvgDispersion  float64 `json:"avg_dispersion"`
	DispersionStd  float64 `json:"dispersion_std"`
	AvgPupilSize   float64 `json:"avg_pupil_size"`
	PupilStd       float64 `json:"pupil_std"`
	LongFixations  int     `json:"long_fixations"`
	ShortFixations int     `json:"short_fixations"`
}

type SaccadeAnalysis struct {
	SaccadeCount int     `json:"saccade_count"`
	AvgSpeed     float64 `json:"avg_speed"`
	SpeedStd     float64 `json:"speed_std"`
	AvgDuration  float64 `json:"avg_duration"`
	DurationStd  float64 `json:"duration_std"`
	AvgDistance  float64 `json:"avg_distance"`
	DistanceStd  float64 `json:"distance_std"`
	MaxSpeed     float64 `json:"max_speed"`
}

type DwellAnalysis struct {
	DwellCount   int            `json:"dwell_count"`
	AvgDuration  float64        `json:"avg_duration"`
	DurationStd  float64        `json:"duration_std"`
	TargetTypes  map[string]int `json:"target_types"`
	TopTargets   map[string]int `json:"top_targets"`
	LongestDwell float64        `json:"longest_dwell"`
}

type FocusAnalysis struct {
	FocusCount        int     `json:"focus_count"`
	BlurCount         int     `json:"blur_count"`
	BlurRate          int     `json:"blur_rate"`
	TotalBlurDuration int64   `json:"total_blur_duration"`
	FocusLostCount    int     `json:"focus_lost_count"`
	AttentionRatio    float64 `json:"attention_ratio"`
}

type DeviceInfo struct {
	UserAgent           string  `json:"user_agent"`
	ScreenWidth         int     `json:"screen_width"`
	ScreenHeight        int     `json:"screen_height"`
	WindowWidth         int     `json:"window_width"`
	WindowHeight        int     `json:"window_height"`
	DevicePixelRatio    float64 `json:"device_pixel_ratio"`
	TouchSupport        bool    `json:"touch_support"`
	Platform            string  `json:"platform"`
	Language            string  `json:"language"`
	HardwareConcurrency int     `json:"hardware_concurrency"`
	MaxTouchPoints      int     `json:"max_touch_points"`
}

type FusionVerificationResult struct {
	IsVerified        bool         `json:"is_verified"`
	OverallConfidence float64      `json:"overall_confidence"`
	ModalScores       *ModalScores `json:"modal_scores"`
	FusionScore       float64      `json:"fusion_score"`
	DecisionDetails   string       `json:"decision_details"`
	RiskLevel         string       `json:"risk_level"`
}

type ModalScores struct {
	MousePressureScore float64 `json:"mouse_pressure_score"`
	TouchForceScore    float64 `json:"touch_force_score"`
	EyeTrackingScore   float64 `json:"eye_tracking_score"`
}

type FusionWeights struct {
	MousePressure float64 `json:"mouse_pressure"`
	TouchForce    float64 `json:"touch_force"`
	EyeTracking   float64 `json:"eye_tracking"`
}

func NewBiometricsV15Service() *BiometricsV15Service {
	return &BiometricsV15Service{
		profiles: make(map[string]*MultimodalBiometricProfile),
	}
}

func (s *BiometricsV15Service) RegisterMultimodalProfile(userID string, data *MultimodalBiometricData) (*MultimodalBiometricProfile, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	profile, exists := s.profiles[userID]
	if !exists {
		profile = &MultimodalBiometricProfile{
			UserID:    userID,
			CreatedAt: time.Now(),
		}
	}

	profile.UpdatedAt = time.Now()

	if data.MousePressure != nil {
		mouseProfile := s.extractMousePressureProfile(data.MousePressure)
		profile.MousePressureProfile = mouseProfile
	}

	if data.TouchForce != nil {
		touchProfile := s.extractTouchForceProfile(data.TouchForce)
		profile.TouchForceProfile = touchProfile
	}

	if data.EyeTracking != nil {
		eyeProfile := s.extractEyeTrackingProfile(data.EyeTracking)
		profile.EyeTrackingProfile = eyeProfile
	}

	profile.FeatureVector = s.generateFeatureVector(profile)
	profile.VerificationCount++
	profile.ConfidenceScore = math.Min(1.0, float64(profile.VerificationCount)/10.0)

	s.profiles[userID] = profile
	return profile, nil
}

func (s *BiometricsV15Service) extractMousePressureProfile(data *MousePressureData) *MousePressureProfile {
	profile := &MousePressureProfile{}

	if data.PressureAnalysis != nil {
		profile.AveragePressure = data.PressureAnalysis.AveragePressure
		profile.PressureStdDev = data.PressureAnalysis.PressureStd
		profile.MaxPressure = data.PressureAnalysis.MaxPressure
		profile.MinPressure = data.PressureAnalysis.MinPressure
		profile.PressureRange = data.PressureAnalysis.PressureRange
		profile.PressureSkewness = data.PressureAnalysis.PressureSkewness
		profile.PressureKurtosis = data.PressureAnalysis.PressureKurtosis
		profile.AvgForce = data.PressureAnalysis.AverageForce
		profile.ForceStdDev = data.PressureAnalysis.ForceStd
		profile.DownPressureAvg = data.PressureAnalysis.DownPressureAvg
		profile.UpPressureAvg = data.PressureAnalysis.UpPressureAvg
	}

	if data.ClickAnalysis != nil {
		profile.ClickCount = data.ClickAnalysis.ClickCount
		profile.AvgClickDuration = data.ClickAnalysis.AvgClickDuration
	}

	if data.MovementAnalysis != nil {
		profile.AvgSpeed = data.MovementAnalysis.AvgSpeed
		profile.SpeedStd = data.MovementAnalysis.SpeedStd
		profile.MaxSpeed = data.MovementAnalysis.MaxSpeed
		profile.AvgAcceleration = data.MovementAnalysis.AvgAcceleration
		profile.AvgJerk = data.MovementAnalysis.AvgJerk
		profile.MovementEntropy = data.MovementAnalysis.MovementEntropy
		profile.DirectionHorizontal = data.MovementAnalysis.DirectionHorizontal
		profile.DirectionVertical = data.MovementAnalysis.DirectionVertical
	}

	if data.DragAnalysis != nil {
		profile.DragCount = data.DragAnalysis.DragCount
		profile.AvgDragPressure = data.DragAnalysis.AvgDragPressure
	}

	return profile
}

func (s *BiometricsV15Service) extractTouchForceProfile(data *TouchForceData) *TouchForceProfile {
	profile := &TouchForceProfile{}

	if data.ForceAnalysis != nil {
		profile.TouchCount = data.ForceAnalysis.TouchCount
		profile.AvgForce = data.ForceAnalysis.AvgForce
		profile.ForceStdDev = data.ForceAnalysis.ForceStdDev
		profile.MaxForce = data.ForceAnalysis.MaxForce
		profile.MinForce = data.ForceAnalysis.MinForce
		profile.ForceRange = data.ForceAnalysis.ForceRange
		profile.ForceSkewness = data.ForceAnalysis.ForceSkewness
		profile.AvgPressure = data.ForceAnalysis.AvgPressure
		profile.PressureStdDev = data.ForceAnalysis.PressureStd
		profile.AvgSpeed = data.ForceAnalysis.AvgSpeed
		profile.SpeedStd = data.ForceAnalysis.SpeedStd
	}

	if data.SwipeAnalysis != nil {
		profile.SwipeCount = data.SwipeAnalysis.SwipeCount
		profile.DirectionEntropy = data.SwipeAnalysis.DirectionEntropy
		profile.AvgSwipeSpeed = data.SwipeAnalysis.AvgSpeed
		profile.AvgSwipeForce = data.SwipeAnalysis.AvgForce
		profile.AvgAngle = data.SwipeAnalysis.AvgAngle
		profile.AvgDistance = data.SwipeAnalysis.AvgDistance
		profile.AvgDuration = data.SwipeAnalysis.AvgDuration
	}

	if data.MultitouchAnalysis != nil {
		profile.PinchCount = data.MultitouchAnalysis.PinchCount
		profile.AvgPinchScale = data.MultitouchAnalysis.AvgPinchScale
		profile.AvgPinchRotation = data.MultitouchAnalysis.AvgPinchRotation
	}

	return profile
}

func (s *BiometricsV15Service) extractEyeTrackingProfile(data *EyeTrackingData) *EyeTrackingProfile {
	profile := &EyeTrackingProfile{}

	if data.GazeAnalysis != nil {
		profile.GazeCount = data.GazeAnalysis.GazeCount
		profile.AvgX = data.GazeAnalysis.AvgX
		profile.AvgY = data.GazeAnalysis.AvgY
		profile.XStd = data.GazeAnalysis.XStd
		profile.YStd = data.GazeAnalysis.YStd
		profile.CoverageArea = data.GazeAnalysis.CoverageArea
		profile.AvgPupilSize = data.GazeAnalysis.AvgPupilSize
		profile.PupilStd = data.GazeAnalysis.PupilStd
		profile.ScanPatternTopLeft = data.GazeAnalysis.ScanPatternTopLeft
		profile.ScanPatternTopRight = data.GazeAnalysis.ScanPatternTopRight
		profile.ScanPatternBottomLeft = data.GazeAnalysis.ScanPatternBottomLeft
		profile.ScanPatternBottomRight = data.GazeAnalysis.ScanPatternBottomRight
	}

	if data.BlinkAnalysis != nil {
		profile.BlinkCount = data.BlinkAnalysis.BlinkCount
		profile.BlinkRate = data.BlinkAnalysis.BlinkRate
		profile.AvgBlinkDuration = data.BlinkAnalysis.AvgBlinkDuration
		profile.AvgBlinkInterval = data.BlinkAnalysis.AvgInterval
	}

	if data.FixationAnalysis != nil {
		profile.FixationCount = data.FixationAnalysis.FixationCount
		profile.AvgFixationDuration = data.FixationAnalysis.AvgDuration
		profile.AvgDispersion = data.FixationAnalysis.AvgDispersion
	}

	if data.SaccadeAnalysis != nil {
		profile.SaccadeCount = data.SaccadeAnalysis.SaccadeCount
		profile.AvgSaccadeSpeed = data.SaccadeAnalysis.AvgSpeed
		profile.MaxSaccadeSpeed = data.SaccadeAnalysis.MaxSpeed
	}

	if data.DwellAnalysis != nil {
		profile.DwellCount = data.DwellAnalysis.DwellCount
		profile.AvgDwellDuration = data.DwellAnalysis.AvgDuration
		profile.LongestDwell = data.DwellAnalysis.LongestDwell
	}

	if data.FocusAnalysis != nil {
		profile.AttentionRatio = data.FocusAnalysis.AttentionRatio
	}

	return profile
}

func (s *BiometricsV15Service) generateFeatureVector(profile *MultimodalBiometricProfile) []float64 {
	features := []float64{}

	if profile.MousePressureProfile != nil {
		mp := profile.MousePressureProfile
		features = append(features,
			mp.AveragePressure,
			mp.PressureStdDev,
			mp.MaxPressure,
			mp.MinPressure,
			mp.PressureRange,
			mp.PressureSkewness,
			mp.PressureKurtosis,
			mp.AvgForce,
			mp.AvgSpeed,
			mp.SpeedStd,
			mp.MaxSpeed,
			mp.AvgAcceleration,
			mp.MovementEntropy,
			mp.DirectionHorizontal,
			mp.DirectionVertical,
			float64(mp.ClickCount),
			mp.AvgClickDuration,
		)
	}

	if profile.TouchForceProfile != nil {
		tp := profile.TouchForceProfile
		features = append(features,
			float64(tp.TouchCount),
			tp.AvgForce,
			tp.ForceStdDev,
			tp.MaxForce,
			tp.ForceRange,
			tp.AvgPressure,
			tp.AvgSpeed,
			float64(tp.SwipeCount),
			tp.DirectionEntropy,
			tp.AvgSwipeSpeed,
			tp.AvgAngle,
			tp.AvgDistance,
			float64(tp.PinchCount),
			tp.AvgPinchScale,
		)
	}

	if profile.EyeTrackingProfile != nil {
		ep := profile.EyeTrackingProfile
		features = append(features,
			float64(ep.GazeCount),
			ep.AvgX,
			ep.AvgY,
			ep.XStd,
			ep.YStd,
			ep.CoverageArea,
			ep.AvgPupilSize,
			float64(ep.BlinkCount),
			ep.BlinkRate,
			ep.AvgBlinkDuration,
			float64(ep.FixationCount),
			ep.AvgFixationDuration,
			ep.AvgDispersion,
			float64(ep.SaccadeCount),
			ep.AvgSaccadeSpeed,
			ep.AttentionRatio,
			ep.ScanPatternTopLeft,
			ep.ScanPatternTopRight,
		)
	}

	return features
}

func (s *BiometricsV15Service) VerifyMultimodal(userID string, data *MultimodalBiometricData) (*FusionVerificationResult, error) {
	profile, exists := s.profiles[userID]
	if !exists {
		return &FusionVerificationResult{
			IsVerified:        false,
			OverallConfidence: 0,
			DecisionDetails:   "No profile found for user",
			RiskLevel:         "high",
		}, nil
	}

	var mouseScore float64 = 0.5
	var touchScore float64 = 0.5
	var eyeScore float64 = 0.5

	if data.MousePressure != nil {
		sampleProfile := s.extractMousePressureProfile(data.MousePressure)
		mouseScore = s.compareMousePressureProfiles(profile.MousePressureProfile, sampleProfile)
	}

	if data.TouchForce != nil {
		sampleProfile := s.extractTouchForceProfile(data.TouchForce)
		touchScore = s.compareTouchForceProfiles(profile.TouchForceProfile, sampleProfile)
	}

	if data.EyeTracking != nil {
		sampleProfile := s.extractEyeTrackingProfile(data.EyeTracking)
		eyeScore = s.compareEyeTrackingProfiles(profile.EyeTrackingProfile, sampleProfile)
	}

	modalScores := &ModalScores{
		MousePressureScore: mouseScore,
		TouchForceScore:    touchScore,
		EyeTrackingScore:   eyeScore,
	}

	fusionScore := s.calculateFusionScore(modalScores)

	isVerified := fusionScore >= 0.85
	riskLevel := s.determineRiskLevel(modalScores, fusionScore)

	return &FusionVerificationResult{
		IsVerified:        isVerified,
		OverallConfidence: fusionScore,
		ModalScores:       modalScores,
		FusionScore:       fusionScore,
		DecisionDetails:   fmt.Sprintf("Fusion verification completed with %.2f%% confidence", fusionScore*100),
		RiskLevel:         riskLevel,
	}, nil
}

func (s *BiometricsV15Service) calculateFusionScore(scores *ModalScores) float64 {
	weights := s.getAdaptiveWeights(scores)

	mouseWeight := weights.MousePressure
	touchWeight := weights.TouchForce
	eyeWeight := weights.EyeTracking

	sumWeights := mouseWeight + touchWeight + eyeWeight
	if sumWeights == 0 {
		return 0.5
	}

	mouseWeight /= sumWeights
	touchWeight /= sumWeights
	eyeWeight /= sumWeights

	weightedScore := scores.MousePressureScore*mouseWeight +
		scores.TouchForceScore*touchWeight +
		scores.EyeTrackingScore*eyeWeight

	return math.Min(1.0, math.Max(0.0, weightedScore))
}

func (s *BiometricsV15Service) getAdaptiveWeights(scores *ModalScores) *FusionWeights {
	weights := &FusionWeights{
		MousePressure: 0.4,
		TouchForce:    0.3,
		EyeTracking:   0.3,
	}

	totalScore := scores.MousePressureScore + scores.TouchForceScore + scores.EyeTrackingScore
	if totalScore == 0 {
		return weights
	}

	mouseReliability := scores.MousePressureScore / totalScore
	touchReliability := scores.TouchForceScore / totalScore
	eyeReliability := scores.EyeTrackingScore / totalScore

	weights.MousePressure = 0.3 + 0.2*mouseReliability
	weights.TouchForce = 0.3 + 0.2*touchReliability
	weights.EyeTracking = 0.3 + 0.2*eyeReliability

	return weights
}

func (s *BiometricsV15Service) determineRiskLevel(scores *ModalScores, fusionScore float64) string {
	modalityCount := 0
	if scores.MousePressureScore > 0.3 {
		modalityCount++
	}
	if scores.TouchForceScore > 0.3 {
		modalityCount++
	}
	if scores.EyeTrackingScore > 0.3 {
		modalityCount++
	}

	if fusionScore >= 0.9 && modalityCount >= 2 {
		return "low"
	} else if fusionScore >= 0.7 && modalityCount >= 1 {
		return "medium"
	}
	return "high"
}

func (s *BiometricsV15Service) compareMousePressureProfiles(profile, sample *MousePressureProfile) float64 {
	if profile == nil || sample == nil {
		return 0.5
	}

	score := 0.0
	weights := 0.0

	features := []struct {
		profileVal, sampleVal float64
		weight                float64
	}{
		{profile.AveragePressure, sample.AveragePressure, 0.15},
		{profile.PressureStdDev, sample.PressureStdDev, 0.10},
		{profile.MaxPressure, sample.MaxPressure, 0.08},
		{profile.PressureRange, sample.PressureRange, 0.07},
		{profile.PressureSkewness, sample.PressureSkewness, 0.08},
		{profile.AvgForce, sample.AvgForce, 0.12},
		{profile.AvgSpeed, sample.AvgSpeed, 0.10},
		{profile.SpeedStd, sample.SpeedStd, 0.08},
		{profile.MaxSpeed, sample.MaxSpeed, 0.07},
		{profile.MovementEntropy, sample.MovementEntropy, 0.08},
		{profile.DirectionHorizontal, sample.DirectionHorizontal, 0.04},
		{profile.DirectionVertical, sample.DirectionVertical, 0.03},
	}

	for _, f := range features {
		if f.profileVal > 0 && f.sampleVal > 0 {
			featureScore := s.calculateSimilarityScore(f.profileVal, f.sampleVal, 0.3)
			score += featureScore * f.weight
			weights += f.weight
		}
	}

	if weights > 0 {
		return score / weights
	}
	return 0.5
}

func (s *BiometricsV15Service) compareTouchForceProfiles(profile, sample *TouchForceProfile) float64 {
	if profile == nil || sample == nil {
		return 0.5
	}

	score := 0.0
	weights := 0.0

	features := []struct {
		profileVal, sampleVal float64
		weight                float64
	}{
		{profile.AvgForce, sample.AvgForce, 0.15},
		{profile.ForceStdDev, sample.ForceStdDev, 0.10},
		{profile.MaxForce, sample.MaxForce, 0.08},
		{profile.AvgPressure, sample.AvgPressure, 0.12},
		{profile.AvgSpeed, sample.AvgSpeed, 0.10},
		{profile.DirectionEntropy, sample.DirectionEntropy, 0.08},
		{profile.AvgSwipeSpeed, sample.AvgSwipeSpeed, 0.10},
		{profile.AvgAngle, sample.AvgAngle, 0.07},
		{profile.AvgPinchScale, sample.AvgPinchScale, 0.10},
		{profile.AvgPinchRotation, sample.AvgPinchRotation, 0.10},
	}

	for _, f := range features {
		if f.profileVal > 0 && f.sampleVal > 0 {
			featureScore := s.calculateSimilarityScore(f.profileVal, f.sampleVal, 0.35)
			score += featureScore * f.weight
			weights += f.weight
		}
	}

	if weights > 0 {
		return score / weights
	}
	return 0.5
}

func (s *BiometricsV15Service) compareEyeTrackingProfiles(profile, sample *EyeTrackingProfile) float64 {
	if profile == nil || sample == nil {
		return 0.5
	}

	score := 0.0
	weights := 0.0

	features := []struct {
		profileVal, sampleVal float64
		weight                float64
	}{
		{profile.AvgX, sample.AvgX, 0.08},
		{profile.AvgY, sample.AvgY, 0.08},
		{profile.XStd, sample.XStd, 0.10},
		{profile.YStd, sample.YStd, 0.10},
		{profile.CoverageArea, sample.CoverageArea, 0.08},
		{profile.AvgPupilSize, sample.AvgPupilSize, 0.10},
		{profile.PupilStd, sample.PupilStd, 0.08},
		{profile.BlinkRate, sample.BlinkRate, 0.10},
		{profile.AvgBlinkDuration, sample.AvgBlinkDuration, 0.08},
		{profile.AvgFixationDuration, sample.AvgFixationDuration, 0.08},
		{profile.AvgDispersion, sample.AvgDispersion, 0.07},
		{profile.AvgSaccadeSpeed, sample.AvgSaccadeSpeed, 0.07},
		{profile.AttentionRatio, sample.AttentionRatio, 0.08},
	}

	for _, f := range features {
		if f.profileVal > 0 && f.sampleVal > 0 {
			featureScore := s.calculateSimilarityScore(f.profileVal, f.sampleVal, 0.3)
			score += featureScore * f.weight
			weights += f.weight
		}
	}

	if weights > 0 {
		return score / weights
	}
	return 0.5
}

func (s *BiometricsV15Service) calculateSimilarityScore(value1, value2, maxDiffRatio float64) float64 {
	if value1 <= 0 || value2 <= 0 {
		return 0.5
	}

	diffRatio := math.Abs(value1-value2) / math.Max(value1, value2)

	if diffRatio <= maxDiffRatio {
		return 1.0 - (diffRatio / maxDiffRatio)
	}

	return 0.0
}

func (s *BiometricsV15Service) GetProfile(userID string) (*MultimodalBiometricProfile, bool) {
	profile, exists := s.profiles[userID]
	return profile, exists
}

func (s *BiometricsV15Service) DeleteProfile(userID string) bool {
	if _, exists := s.profiles[userID]; exists {
		delete(s.profiles, userID)
		return true
	}
	return false
}

func (s *BiometricsV15Service) ParseBiometricData(dataJSON []byte) (*MultimodalBiometricData, error) {
	var data MultimodalBiometricData
	err := json.Unmarshal(dataJSON, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *BiometricsV15Service) SerializeProfile(profile *MultimodalBiometricProfile) ([]byte, error) {
	return json.Marshal(profile)
}

func (s *BiometricsV15Service) DeserializeProfile(data []byte) (*MultimodalBiometricProfile, error) {
	var profile MultimodalBiometricProfile
	err := json.Unmarshal(data, &profile)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *BiometricsV15Service) CalculateCosineSimilarity(vec1, vec2 []float64) float64 {
	if len(vec1) != len(vec2) || len(vec1) == 0 {
		return 0
	}

	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0

	for i := 0; i < len(vec1); i++ {
		dotProduct += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}

	if norm1 == 0 || norm2 == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

func (s *BiometricsV15Service) CalculateEuclideanDistance(vec1, vec2 []float64) float64 {
	if len(vec1) != len(vec2) || len(vec1) == 0 {
		return 0
	}

	sumSquares := 0.0
	for i := 0; i < len(vec1); i++ {
		diff := vec1[i] - vec2[i]
		sumSquares += diff * diff
	}

	return math.Sqrt(sumSquares)
}

func (s *BiometricsV15Service) NormalizeFeatureVector(vec []float64) []float64 {
	if len(vec) == 0 {
		return vec
	}

	values := make([]float64, len(vec))
	copy(values, vec)

	sort.Float64s(values)

	q1 := values[len(values)/4]
	q3 := values[3*len(values)/4]
	iqr := q3 - q1

	if iqr == 0 {
		mean := meanFloat64(vec)
		for i := range values {
			if values[i] == 0 {
				values[i] = mean
			}
		}
		return values
	}

	for i := range values {
		normalized := (values[i] - q1) / iqr
		values[i] = math.Max(0, math.Min(1, normalized))
	}

	return values
}

func meanFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDevFloat64(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	avg := meanFloat64(values)
	sumSquares := 0.0
	for _, v := range values {
		diff := v - avg
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares / float64(len(values)))
}
