package model

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

type AnomalyPatternType string

const (
	AnomalyConstantSpeed    AnomalyPatternType = "constant_speed"
	AnomalyLinearPath      AnomalyPatternType = "linear_path"
	AnomalyNoPause         AnomalyPatternType = "no_pause"
	AnomalyMechanicalClick  AnomalyPatternType = "mechanical_click"
	AnomalyRapidFire       AnomalyPatternType = "rapid_fire"
	AnomalyRepeatingPath   AnomalyPatternType = "repeating_path"
	AnomalyZeroJitter      AnomalyPatternType = "zero_jitter"
	AnomalySuperhumanSpeed AnomalyPatternType = "superhuman_speed"
	AnomalyPerfectRegular  AnomalyPatternType = "perfect_regular"
	AnomalyNoMicroAdjust   AnomalyPatternType = "no_micro_adjust"
	AnomalyPatternDev      AnomalyPatternType = "pattern_deviation"
	AnomalyBurstBehavior   AnomalyPatternType = "burst_behavior"
	AnomalyImpossibleMouse AnomalyPatternType = "impossible_mouse"
	AnomalyCopyPaste       AnomalyPatternType = "copy_paste"
	AnomalyAutoFill        AnomalyPatternType = "auto_fill"
)

type AnomalyPattern struct {
	Type        AnomalyPatternType `json:"type"`
	Severity    float64           `json:"severity"`
	Confidence  float64           `json:"confidence"`
	Evidence    []string          `json:"evidence"`
	Timestamp   time.Time         `json:"timestamp"`
	Description string            `json:"description"`
}

type EnhancedRiskResult struct {
	RiskLevel          RiskLevel          `json:"risk_level"`
	RiskScore          float64           `json:"risk_score"`
	PositionScore      float64           `json:"position_score"`
	TraceScore         float64           `json:"trace_score"`
	EnvScore           float64           `json:"env_score"`
	RiskFactors        []string          `json:"risk_factors"`
	Action             string            `json:"action"`
	RecommendVerify    bool              `json:"recommend_verify"`
	HumanProbability   float64           `json:"human_probability"`
	Details            map[string]float64 `json:"details,omitempty"`
	AnomalyPatterns    []AnomalyPattern  `json:"anomaly_patterns,omitempty"`
	CompositeScore     float64           `json:"composite_score,omitempty"`
	ConfidenceLevel    float64           `json:"confidence_level,omitempty"`
	ThreatLevel        string            `json:"threat_level,omitempty"`
	BehavioralEntropy  float64           `json:"behavioral_entropy,omitempty"`
	PatternDeviation  float64           `json:"pattern_deviation,omitempty"`
	HistoricalSimilarity float64         `json:"historical_similarity,omitempty"`
}

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
	ClickCount        int               `json:"click_count"`
	LastKnownLocation string            `json:"last_known_location"`
	CurrentLocation   string            `json:"current_location"`
	DeviceReputationScore float64        `json:"device_reputation_score"`
	RiskScore         float64           `json:"risk_score"`
	ApplicationID     *uint             `json:"application_id"`
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

func (r *EnhancedRiskResult) AddAnomalyPattern(pattern AnomalyPattern) {
	r.AnomalyPatterns = append(r.AnomalyPatterns, pattern)
}

func (r *EnhancedRiskResult) CalculateCompositeScore() float64 {
	if len(r.AnomalyPatterns) == 0 {
		return r.RiskScore
	}

	var totalSeverity float64
	var maxSeverity float64
	for _, pattern := range r.AnomalyPatterns {
		totalSeverity += pattern.Severity * pattern.Confidence
		if pattern.Severity > maxSeverity {
			maxSeverity = pattern.Severity
		}
	}

	patternScore := totalSeverity / float64(len(r.AnomalyPatterns))

	compositeScore := r.RiskScore*0.6 + patternScore*0.4

	if maxSeverity > 80 {
		compositeScore = math.Max(compositeScore, maxSeverity)
	}

	return math.Min(compositeScore, 100)
}

func (r *EnhancedRiskResult) DetermineThreatLevel() string {
	if r.RiskScore >= 90 || len(r.AnomalyPatterns) >= 5 {
		return "critical"
	} else if r.RiskScore >= 70 || len(r.AnomalyPatterns) >= 3 {
		return "high"
	} else if r.RiskScore >= 50 || len(r.AnomalyPatterns) >= 2 {
		return "medium"
	}
	return "low"
}

func (r *EnhancedRiskResult) CalculateConfidenceLevel() float64 {
	baseConfidence := 0.7

	if len(r.AnomalyPatterns) > 0 {
		patternConfidence := 0.0
		for _, pattern := range r.AnomalyPatterns {
			patternConfidence += pattern.Confidence
		}
		patternConfidence /= float64(len(r.AnomalyPatterns))
		baseConfidence += patternConfidence * 0.2
	}

	dataPoints := len(r.Details)
	if dataPoints > 10 {
		baseConfidence += 0.1
	} else if dataPoints > 5 {
		baseConfidence += 0.05
	}

	return math.Min(baseConfidence, 0.99)
}

func (r *EnhancedRiskResult) GenerateReport() string {
	report := "Enhanced Risk Analysis Report\n"
	report += "============================\n"
	report += fmt.Sprintf("Risk Level: %s\n", r.RiskLevel)
	report += fmt.Sprintf("Risk Score: %.2f\n", r.RiskScore)
	report += fmt.Sprintf("Composite Score: %.2f\n", r.CompositeScore)
	report += fmt.Sprintf("Threat Level: %s\n", r.ThreatLevel)
	report += fmt.Sprintf("Confidence: %.2f%%\n", r.ConfidenceLevel*100)

	if len(r.AnomalyPatterns) > 0 {
		report += "\nDetected Anomaly Patterns:\n"
		for i, pattern := range r.AnomalyPatterns {
			report += fmt.Sprintf("%d. [%s] Severity: %.2f, Confidence: %.2f%%\n",
				i+1, pattern.Type, pattern.Severity, pattern.Confidence*100)
			report += fmt.Sprintf("   Description: %s\n", pattern.Description)
			if len(pattern.Evidence) > 0 {
				report += fmt.Sprintf("   Evidence: %s\n", strings.Join(pattern.Evidence, ", "))
			}
		}
	}

	report += fmt.Sprintf("\nRecommended Action: %s\n", r.Action)

	return report
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
