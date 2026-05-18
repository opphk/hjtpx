package model

import (
	"encoding/json"
	"sort"
	"time"
)

type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

type RiskLog struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	SessionID   string    `json:"session_id" gorm:"size:100;index:idx_risk_session"`
	RiskLevel   RiskLevel `json:"risk_level" gorm:"size:20;index:idx_risk_level"`
	RiskScore   float64   `json:"risk_score" gorm:"index:idx_risk_score"`
	RiskFactors string    `json:"risk_factors" gorm:"type:text"`
	ActionTaken string    `json:"action_taken" gorm:"size:50"`
	IPAddress   string    `json:"ip_address" gorm:"size:50;index:idx_risk_ip"`
	UserAgent   string    `json:"user_agent" gorm:"size:500"`
	Fingerprint string    `json:"fingerprint" gorm:"size:64;index:idx_risk_fingerprint"`
	DeviceInfo  string    `json:"device_info" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at" gorm:"index:idx_risk_created"`
}

type RiskResult struct {
	RiskLevel        RiskLevel          `json:"risk_level"`
	RiskScore        float64            `json:"risk_score"`
	PositionScore    float64            `json:"position_score"`
	TraceScore       float64            `json:"trace_score"`
	EnvScore         float64            `json:"env_score"`
	RiskFactors      []string           `json:"risk_factors"`
	Action           string             `json:"action"`
	RecommendVerify  bool               `json:"recommend_verify"`
	HumanProbability float64            `json:"human_probability"`
	Details          map[string]float64 `json:"details,omitempty"`
}

type RiskContext struct {
	SessionID         string            `json:"session_id"`
	IPAddress         string            `json:"ip_address"`
	UserAgent         string            `json:"user_agent"`
	Fingerprint       string            `json:"fingerprint"`
	DeviceInfo        map[string]string `json:"device_info"`
	PositionDiff      int               `json:"position_diff"`
	TraceData         []TracePoint      `json:"trace_data"`
	EnvInfo           *EnvInfo          `json:"env_info"`
	VerificationCount int               `json:"verification_count"`
	FailureCount      int               `json:"failure_count"`
	TimeFromStart     int64             `json:"time_from_start"`
	MouseSpeed        float64           `json:"mouse_speed"`
	HasTouchDevice    bool              `json:"has_touch_device"`
	BrowserPlugins    []string          `json:"browser_plugins"`
	Language          string            `json:"language"`
	Timezone          string            `json:"timezone"`
	ScreenRes         string            `json:"screen_res"`
	Referer           string            `json:"referer"`
	IsProxy           bool              `json:"is_proxy"`
	IsVPN             bool              `json:"is_vpn"`
	IsTor             bool              `json:"is_tor"`
	IsHosting         bool              `json:"is_hosting"`
	IPReputation      string            `json:"ip_reputation"`
	Country           string            `json:"country"`
	ASNumber          int               `json:"as_number"`
}

type RiskRule struct {
	Name            string                  `json:"name"`
	Condition       func(*RiskContext) bool `json:"-"`
	ConditionConfig map[string]interface{}  `json:"condition_config,omitempty"`
	Action          string                  `json:"action"`
	Score           float64                 `json:"score"`
	Priority        int                     `json:"priority"`
	Enabled         bool                    `json:"enabled"`
	Reason          string                  `json:"reason"`
}

type RiskStatistics struct {
	TotalCount     int64            `json:"total_count"`
	PassCount      int64            `json:"pass_count"`
	ReviewCount    int64            `json:"review_count"`
	BlockCount     int64            `json:"block_count"`
	AvgRiskScore   float64          `json:"avg_risk_score"`
	RiskLevelStats map[string]int64 `json:"risk_level_stats"`
	TopRiskFactors []RiskFactorStat `json:"top_risk_factors"`
	TrendByHour    []HourlyStat     `json:"trend_by_hour"`
	TopOffenders   []IPStat         `json:"top_offenders"`
}

type RiskFactorStat struct {
	Factor   string  `json:"factor"`
	Count    int64   `json:"count"`
	AvgScore float64 `json:"avg_score"`
}

type HourlyStat struct {
	Hour         time.Time `json:"hour"`
	TotalCount   int64     `json:"total_count"`
	PassCount    int64     `json:"pass_count"`
	BlockCount   int64     `json:"block_count"`
	AvgRiskScore float64   `json:"avg_risk_score"`
}

type IPStat struct {
	IPAddress  string    `json:"ip_address"`
	BlockCount int64     `json:"block_count"`
	TotalCount int64     `json:"total_count"`
	LastSeen   time.Time `json:"last_seen"`
}

func (r *RiskLog) SetRiskFactors(factors []string) error {
	data, err := json.Marshal(factors)
	if err != nil {
		return err
	}
	r.RiskFactors = string(data)
	return nil
}

func (r *RiskLog) GetRiskFactors() ([]string, error) {
	var factors []string
	if r.RiskFactors == "" {
		return factors, nil
	}
	err := json.Unmarshal([]byte(r.RiskFactors), &factors)
	return factors, err
}

func (r *RiskResult) AddRiskFactor(factor string) {
	for _, f := range r.RiskFactors {
		if f == factor {
			return
		}
	}
	r.RiskFactors = append(r.RiskFactors, factor)
}

func (r *RiskResult) SortRiskFactors() {
	sort.Strings(r.RiskFactors)
}

func (r *RiskResult) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ParseRiskResult(data string) (*RiskResult, error) {
	var result RiskResult
	err := json.Unmarshal([]byte(data), &result)
	return &result, err
}

func (rc *RiskContext) HasHighRiskIndicators() bool {
	if rc.IsProxy || rc.IsVPN || rc.IsTor {
		return true
	}
	if rc.FailureCount >= 3 {
		return true
	}
	if rc.MouseSpeed > 2000 {
		return true
	}
	if rc.TimeFromStart > 0 && rc.TimeFromStart < 500 {
		return true
	}
	return false
}

func (rc *RiskContext) GetTrustScore() float64 {
	score := 100.0

	if rc.VerificationCount > 5 {
		score += 10
	}
	if rc.FailureCount == 0 && rc.VerificationCount > 0 {
		score += 15
	}
	if !rc.IsProxy && !rc.IsVPN && !rc.IsTor {
		score += 20
	}
	if rc.HasTouchDevice {
		score += 5
	}
	if rc.Timezone != "" && rc.Language != "" {
		score += 5
	}

	return score
}

func DetermineRiskLevel(score float64) RiskLevel {
	switch {
	case score >= 80:
		return RiskLevelLow
	case score >= 60:
		return RiskLevelMedium
	case score >= 40:
		return RiskLevelHigh
	default:
		return RiskLevelCritical
	}
}

func CalculateHumanProbability(riskScore float64) float64 {
	if riskScore >= 100 {
		return 99.0
	}
	if riskScore <= 0 {
		return 1.0
	}
	return 1.0 + riskScore*0.98
}

func NewRiskContext() *RiskContext {
	return &RiskContext{
		TraceData:      make([]TracePoint, 0),
		BrowserPlugins: make([]string, 0),
		EnvInfo:        &EnvInfo{},
		DeviceInfo:     make(map[string]string),
	}
}
