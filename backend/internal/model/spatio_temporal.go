package model

import "time"

type TimePatternType string

const (
	TimePatternDaily   TimePatternType = "daily"
	TimePatternWeekly  TimePatternType = "weekly"
	TimePatternMonthly TimePatternType = "monthly"
	TimePatternCustom TimePatternType = "custom"
)

type LocationAccuracy string

const (
	LocationAccuracyCity    LocationAccuracy = "city"
	LocationAccuracyRegion  LocationAccuracy = "region"
	LocationAccuracyCountry LocationAccuracy = "country"
	LocationAccuracyIP      LocationAccuracy = "ip"
	LocationAccuracyGPS     LocationAccuracy = "gps"
)

type SpatioTemporalPoint struct {
	Timestamp  int64           `json:"timestamp"`
	Latitude   float64         `json:"latitude"`
	Longitude  float64         `json:"longitude"`
	Altitude   float64         `json:"altitude,omitempty"`
	IPAddress  string          `json:"ip_address,omitempty"`
	UserAgent  string          `json:"user_agent,omitempty"`
	DeviceID   string          `json:"device_id,omitempty"`
	Accuracy   LocationAccuracy `json:"accuracy"`
	Confidence float64         `json:"confidence"`
	Velocity   float64         `json:"velocity,omitempty"`
	Heading    float64         `json:"heading,omitempty"`
}

type TimeWindow struct {
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`
	Duration  int64 `json:"duration"`
}

type SpatioTemporalPattern struct {
	PatternID        string                   `json:"pattern_id"`
	PatternType      TimePatternType          `json:"pattern_type"`
	Points           []SpatioTemporalPoint    `json:"points"`
	Centroid         []float64                `json:"centroid"`
	TimeWindow       TimeWindow               `json:"time_window"`
	BehaviorFeatures map[string]float64       `json:"behavior_features"`
	AnomalyScore     float64                  `json:"anomaly_score"`
	Confidence       float64                  `json:"confidence"`
	Frequency        float64                  `json:"frequency"`
}

type BehaviorFlow struct {
	FlowID        string                    `json:"flow_id"`
	UserID        string                    `json:"user_id"`
	Points        []SpatioTemporalPoint     `json:"points"`
	StartTime     int64                     `json:"start_time"`
	EndTime       int64                     `json:"end_time"`
	TotalDistance float64                   `json:"total_distance"`
	AvgVelocity   float64                   `json:"avg_velocity"`
	MaxVelocity   float64                   `json:"max_velocity"`
	MinVelocity   float64                   `json:"min_velocity"`
	Trajectory    []TrajectoryPoint         `json:"trajectory"`
	Anomalies     []AnomalousBehavior       `json:"anomalies"`
	RiskScore     float64                   `json:"risk_score"`
	PatternType   TimePatternType           `json:"pattern_type"`
}

type TrajectoryPoint struct {
	Timestamp int64   `json:"timestamp"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Z         float64 `json:"z,omitempty"`
	Velocity  float64 `json:"velocity"`
	Acceleration float64 `json:"acceleration"`
	Jerk      float64 `json:"jerk"`
	Direction float64 `json:"direction"`
}

type AnomalousBehavior struct {
	AnomalyID       string    `json:"anomaly_id"`
	AnomalyType     string    `json:"anomaly_type"`
	Timestamp       int64     `json:"timestamp"`
	Location        []float64 `json:"location"`
	Severity        float64   `json:"severity"`
	Description     string    `json:"description"`
	Confidence      float64   `json:"confidence"`
	RiskContribution float64 `json:"risk_contribution"`
}

type BehaviorPrediction struct {
	PredictionID      string              `json:"prediction_id"`
	UserID            string              `json:"user_id"`
	PredictedLocation []float64           `json:"predicted_location"`
	PredictedTime     int64               `json:"predicted_time"`
	PredictionWindow  int64               `json:"prediction_window"`
	Confidence        float64             `json:"confidence"`
	Method            string              `json:"method"`
	Features          map[string]float64  `json:"features"`
	Trajectory        []TrajectoryPoint   `json:"trajectory"`
	AnomalyIndicators []string            `json:"anomaly_indicators"`
}

type RiskScore struct {
	ScoreID          string                 `json:"score_id"`
	UserID           string                 `json:"user_id"`
	OverallScore     float64                `json:"overall_score"`
	LocationScore    float64                `json:"location_score"`
	TimeScore        float64                `json:"time_score"`
	BehaviorScore    float64                `json:"behavior_score"`
	VelocityScore    float64                `json:"velocity_score"`
	PatternScore     float64                `json:"pattern_score"`
	RiskLevel        string                 `json:"risk_level"`
	RiskFactors      []string               `json:"risk_factors"`
	Recommendations  []string               `json:"recommendations"`
	CalculatedAt     int64                  `json:"calculated_at"`
	ValidUntil       int64                  `json:"valid_until"`
}

type SpatioTemporalSession struct {
	SessionID        string                   `json:"session_id"`
	UserID           string                   `json:"user_id"`
	TargetPattern    *SpatioTemporalPattern   `json:"target_pattern"`
	ChallengePoints  []SpatioTemporalPoint    `json:"challenge_points"`
	CorrectOption    string                   `json:"correct_option"`
	Status           string                   `json:"status"`
	VerifyCount      int                      `json:"verify_count"`
	MaxAttempts      int                      `json:"max_attempts"`
	CreatedAt        time.Time                `json:"created_at"`
	ExpiredAt        time.Time                `json:"expired_at"`
	Difficulty       string                   `json:"difficulty"`
	ClientIP         string                   `json:"client_ip"`
	UserAgent        string                   `json:"user_agent"`
	BehaviorFlows    []BehaviorFlow           `json:"behavior_flows,omitempty"`
	RiskScores       []RiskScore              `json:"risk_scores,omitempty"`
}

type SpatioTemporalRequest struct {
	UserID             string                 `json:"user_id"`
	PatternType        TimePatternType        `json:"pattern_type"`
	Difficulty         string                 `json:"difficulty"`
	ClientIP           string                 `json:"client_ip"`
	UserAgent          string                 `json:"user_agent"`
	CurrentLocation    *SpatioTemporalPoint   `json:"current_location,omitempty"`
	IncludePredictions bool                   `json:"include_predictions"`
	PredictionWindow   int64                  `json:"prediction_window"`
}

type SpatioTemporalResponse struct {
	SessionID         string                  `json:"session_id"`
	TargetPattern     *SpatioTemporalPattern  `json:"target_pattern"`
	ChallengePoints   []SpatioTemporalPoint   `json:"challenge_points"`
	Instructions      string                  `json:"instructions"`
	Options           []ChallengeOption       `json:"options"`
	ExpiresIn         int64                   `json:"expires_in"`
	ExpiresAt         int64                   `json:"expires_at"`
	BehaviorFlow      *BehaviorFlow           `json:"behavior_flow,omitempty"`
	Prediction        *BehaviorPrediction     `json:"prediction,omitempty"`
	RiskScore         *RiskScore              `json:"risk_score,omitempty"`
}

type ChallengeOption struct {
	OptionID   string                `json:"option_id"`
	Point      SpatioTemporalPoint   `json:"point"`
	IsCorrect  bool                  `json:"is_correct"`
}

type SpatioTemporalVerifyRequest struct {
	SessionID      string                 `json:"session_id"`
	SelectedOption string                 `json:"selected_option"`
	UserLocation   *SpatioTemporalPoint   `json:"user_location"`
	ResponseTime   int64                  `json:"response_time"`
	BehaviorData   map[string]interface{} `json:"behavior_data,omitempty"`
}

type SpatioTemporalVerifyResponse struct {
	Success   bool                           `json:"success"`
	Score     float64                        `json:"score"`
	Message   string                         `json:"message"`
	Details   *VerifyDetails                 `json:"details,omitempty"`
	Analytics *SpatioTemporalAnalytics      `json:"analytics,omitempty"`
	RiskScore *RiskScore                    `json:"risk_score,omitempty"`
}

type VerifyDetails struct {
	LocationMatchScore float64 `json:"location_match_score"`
	TimePatternScore   float64 `json:"time_pattern_score"`
	BehaviorMatchScore float64 `json:"behavior_match_score"`
	DistanceToCentroid float64 `json:"distance_to_centroid"`
	TimeWindowMatch     float64 `json:"time_window_match"`
	AnomalyScore       float64 `json:"anomaly_score"`
	VelocityScore       float64 `json:"velocity_score"`
}

type SpatioTemporalAnalytics struct {
	LocationConfidence  float64   `json:"location_confidence"`
	TimeConsistency     float64   `json:"time_consistency"`
	BehaviorConsistency float64   `json:"behavior_consistency"`
	VelocityConsistency float64   `json:"velocity_consistency"`
	RiskLevel           string    `json:"risk_level"`
	RiskFactors         []string  `json:"risk_factors"`
}

type ContinuousBehaviorData struct {
	UserID        string              `json:"user_id"`
	SessionID     string              `json:"session_id"`
	Points        []SpatioTemporalPoint `json:"points"`
	StartTime     int64               `json:"start_time"`
	EndTime       int64               `json:"end_time"`
	Duration      int64               `json:"duration"`
	SampleRate    float64             `json:"sample_rate"`
	TotalDistance float64             `json:"total_distance"`
	AvgVelocity   float64             `json:"avg_velocity"`
	MaxVelocity   float64             `json:"max_velocity"`
	MinVelocity   float64             `json:"min_velocity"`
	StdVelocity   float64             `json:"std_velocity"`
	Anomalies     []AnomalousBehavior `json:"anomalies"`
}

type TrajectoryPredictionRequest struct {
	UserID           string                    `json:"user_id"`
	HistoricalData   []SpatioTemporalPoint    `json:"historical_data"`
	CurrentLocation  []float64                 `json:"current_location"`
	CurrentTime      int64                     `json:"current_time"`
	PredictionSteps  int                       `json:"prediction_steps"`
	PredictionMethod string                    `json:"prediction_method"`
}

type TrajectoryPredictionResponse struct {
	Predictions    []BehaviorPrediction `json:"predictions"`
	Confidence     float64             `json:"confidence"`
	Method         string              `json:"method"`
	ModelVersion   string              `json:"model_version"`
}

type RiskAssessmentRequest struct {
	UserID        string                   `json:"user_id"`
	BehaviorData  *ContinuousBehaviorData `json:"behavior_data"`
	ContextData   map[string]interface{}  `json:"context_data,omitempty"`
	Threshold     float64                 `json:"threshold"`
}

type RiskAssessmentResponse struct {
	AssessmentID  string      `json:"assessment_id"`
	RiskScore    *RiskScore  `json:"risk_score"`
	Anomalies    []AnomalousBehavior `json:"anomalies"`
	Recommendations []string `json:"recommendations"`
	Factors      map[string]float64 `json:"factors"`
}
