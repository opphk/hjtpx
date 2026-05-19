package model

import (
	"encoding/json"
	"time"
)

type DifficultyAdaptiveConfig struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	AppID       uint      `json:"app_id" gorm:"index"`
	MinLevel    string    `json:"min_level"`
	MaxLevel    string    `json:"max_level"`
	RiskWeights string    `json:"risk_weights" gorm:"type:text"`
	Timeouts    string    `json:"timeouts" gorm:"type:text"`
	RetryConfig string    `json:"retry_config" gorm:"type:text"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DifficultyRiskWeights struct {
	DeviceRisk        float64 `json:"device_risk"`
	BehavioralRisk    float64 `json:"behavioral_risk"`
	HistoricalRisk    float64 `json:"historical_risk"`
	ContextualRisk    float64 `json:"contextual_risk"`
	NetworkRisk       float64 `json:"network_risk"`
	GeolocationRisk   float64 `json:"geolocation_risk"`
	TimePatternRisk   float64 `json:"time_pattern_risk"`
}

type DifficultyTimeouts struct {
	DefaultTimeout time.Duration `json:"default_timeout"`
	GracePeriod    time.Duration `json:"grace_period"`
	MaxExtensions  int           `json:"max_extensions"`
}

type DifficultyRetryConfig struct {
	MaxRetries     int           `json:"max_retries"`
	InitialDelay   time.Duration `json:"initial_delay"`
	MaxDelay       time.Duration `json:"max_delay"`
	Multiplier     float64       `json:"multiplier"`
	JitterFactor   float64       `json:"jitter_factor"`
}

type UserDifficultyProfile struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	UserID            string    `json:"user_id" gorm:"index:idx_user_app;uniqueIndex:idx_user_app,priority:1"`
	AppID             uint      `json:"app_id" gorm:"index:idx_user_app;uniqueIndex:idx_user_app,priority:2"`
	CurrentDifficulty string    `json:"current_difficulty"`
	RiskScore         float64   `json:"risk_score"`
	SuccessRate       float64   `json:"success_rate"`
	TotalAttempts     int       `json:"total_attempts"`
	LastAttemptAt     time.Time `json:"last_attempt_at"`
	LastDifficulty    string    `json:"last_difficulty"`
	ConsecutiveSuccess int      `json:"consecutive_success"`
	ConsecutiveFail   int       `json:"consecutive_fail"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type DifficultyAdjustmentLog struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	UserID            string    `json:"user_id" gorm:"index"`
	AppID             uint      `json:"app_id" gorm:"index"`
	PreviousDifficulty string   `json:"previous_difficulty"`
	NewDifficulty     string    `json:"new_difficulty"`
	AdjustmentReason  string    `json:"adjustment_reason"`
	RiskScoreBefore   float64   `json:"risk_score_before"`
	RiskScoreAfter    float64   `json:"risk_score_after"`
	AdjustmentFactor  float64   `json:"adjustment_factor"`
	SessionID         string    `json:"session_id" gorm:"index"`
	CreatedAt         time.Time `json:"created_at"`
}

type TimeoutEvent struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       string    `json:"user_id" gorm:"index"`
	SessionID    string    `json:"session_id" gorm:"index"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Duration     time.Duration `json:"duration"`
	Extensions   int       `json:"extensions"`
	WasCompleted bool      `json:"was_completed"`
	CreatedAt    time.Time `json:"created_at"`
}

type RetryEvent struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      string    `json:"user_id" gorm:"index"`
	SessionID   string    `json:"session_id" gorm:"index"`
	RetryNumber int       `json:"retry_number"`
	DelayUsed   time.Duration `json:"delay_used"`
	WasSuccess  bool      `json:"was_success"`
	CreatedAt   time.Time `json:"created_at"`
}

type AdaptiveDifficultyMetrics struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	AppID             uint      `json:"app_id" gorm:"index"`
	TotalUsers        int       `json:"total_users"`
	AvgRiskScore      float64   `json:"avg_risk_score"`
	DifficultyDist    string    `json:"difficulty_dist" gorm:"type:text"`
	TimeoutRate       float64   `json:"timeout_rate"`
	RetryRate         float64   `json:"retry_rate"`
	SuccessRateByLevel string   `json:"success_rate_by_level" gorm:"type:text"`
	AvgAttemptsPerUser float64 `json:"avg_attempts_per_user"`
	RecordedAt        time.Time `json:"recorded_at"`
}

func (d *DifficultyAdaptiveConfig) GetRiskWeights() (*DifficultyRiskWeights, error) {
	if d.RiskWeights == "" {
		return &DifficultyRiskWeights{
			DeviceRisk:      0.15,
			BehavioralRisk:  0.25,
			HistoricalRisk:  0.20,
			ContextualRisk:  0.15,
			NetworkRisk:     0.10,
			GeolocationRisk: 0.08,
			TimePatternRisk: 0.07,
		}, nil
	}

	var weights DifficultyRiskWeights
	err := json.Unmarshal([]byte(d.RiskWeights), &weights)
	return &weights, err
}

func (d *DifficultyAdaptiveConfig) SetRiskWeights(weights *DifficultyRiskWeights) error {
	data, err := json.Marshal(weights)
	if err != nil {
		return err
	}
	d.RiskWeights = string(data)
	return nil
}

func (d *DifficultyAdaptiveConfig) GetTimeouts() (*DifficultyTimeouts, error) {
	if d.Timeouts == "" {
		return &DifficultyTimeouts{
			DefaultTimeout: 60 * time.Second,
			GracePeriod:    10 * time.Second,
			MaxExtensions:  2,
		}, nil
	}

	var timeouts DifficultyTimeouts
	err := json.Unmarshal([]byte(d.Timeouts), &timeouts)
	return &timeouts, err
}

func (d *DifficultyAdaptiveConfig) SetTimeouts(timeouts *DifficultyTimeouts) error {
	data, err := json.Marshal(timeouts)
	if err != nil {
		return err
	}
	d.Timeouts = string(data)
	return nil
}

func (d *DifficultyAdaptiveConfig) GetRetryConfig() (*DifficultyRetryConfig, error) {
	if d.RetryConfig == "" {
		return &DifficultyRetryConfig{
			MaxRetries:   3,
			InitialDelay: 5 * time.Second,
			MaxDelay:     60 * time.Second,
			Multiplier:   2.0,
			JitterFactor: 0.2,
		}, nil
	}

	var config DifficultyRetryConfig
	err := json.Unmarshal([]byte(d.RetryConfig), &config)
	return &config, err
}

func (d *DifficultyAdaptiveConfig) SetRetryConfig(config *DifficultyRetryConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	d.RetryConfig = string(data)
	return nil
}

func (u *UserDifficultyProfile) CalculateSuccessRate() float64 {
	if u.TotalAttempts == 0 {
		return 0
	}
	return float64(u.ConsecutiveSuccess) / float64(u.TotalAttempts) * 100
}

func (u *UserDifficultyProfile) ShouldDecreaseDifficulty() bool {
	return u.ConsecutiveFail >= 3 || u.SuccessRate < 60
}

func (u *UserDifficultyProfile) ShouldIncreaseDifficulty() bool {
	return u.ConsecutiveSuccess >= 5 && u.SuccessRate > 90
}

func (m *AdaptiveDifficultyMetrics) GetDifficultyDistribution() (map[string]int, error) {
	if m.DifficultyDist == "" {
		return map[string]int{
			"easy":   0,
			"medium": 0,
			"hard":   0,
			"expert": 0,
		}, nil
	}

	var dist map[string]int
	err := json.Unmarshal([]byte(m.DifficultyDist), &dist)
	return dist, err
}

func (m *AdaptiveDifficultyMetrics) SetDifficultyDistribution(dist map[string]int) error {
	data, err := json.Marshal(dist)
	if err != nil {
		return err
	}
	m.DifficultyDist = string(data)
	return nil
}

func (m *AdaptiveDifficultyMetrics) GetSuccessRateByLevel() (map[string]float64, error) {
	if m.SuccessRateByLevel == "" {
		return map[string]float64{
			"easy":   0,
			"medium": 0,
			"hard":   0,
			"expert": 0,
		}, nil
	}

	var rates map[string]float64
	err := json.Unmarshal([]byte(m.SuccessRateByLevel), &rates)
	return rates, err
}

func (m *AdaptiveDifficultyMetrics) SetSuccessRateByLevel(rates map[string]float64) error {
	data, err := json.Marshal(rates)
	if err != nil {
		return err
	}
	m.SuccessRateByLevel = string(data)
	return nil
}
