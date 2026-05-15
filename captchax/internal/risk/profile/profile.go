package profile

import (
	"time"
)

type UserProfile struct {
	ID              int64           `json:"id"`
	Identifier      string          `json:"identifier"`
	IdentifierType  IdentifierType  `json:"identifier_type"`
	IP              string          `json:"ip"`
	DeviceFingerprint string        `json:"device_fingerprint"`
	CookieID        string          `json:"cookie_id"`
	SessionID       string          `json:"session_id"`

	TotalAttempts   int64           `json:"total_attempts"`
	SuccessCount    int64           `json:"success_count"`
	FailCount       int64           `json:"fail_count"`
	SuccessRate     float64         `json:"success_rate"`

	AvgResponseTime float64         `json:"avg_response_time"`
	MinResponseTime float64         `json:"min_response_time"`
	MaxResponseTime float64         `json:"max_response_time"`

	PreferredCaptchaType string     `json:"preferred_captcha_type"`
	CaptchaTypeDistribution map[string]int64 `json:"captcha_type_distribution"`

	ActiveHours      map[int]int64  `json:"active_hours"`
	ActiveDays       map[int]int64  `json:"active_days"`

	LocationDistribution map[string]int64 `json:"location_distribution"`
	DeviceDistribution   map[string]int64 `json:"device_distribution"`

	TotalRiskEvents int64           `json:"total_risk_events"`
	HighRiskEvents  int64           `json:"high_risk_events"`
	LastRiskEventAt *time.Time     `json:"last_risk_event_at"`

	FirstSeenAt     time.Time       `json:"first_seen_at"`
	LastSeenAt      time.Time       `json:"last_seen_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	CreatedAt       time.Time       `json:"created_at"`
}

type IdentifierType string

const (
	IdentifierTypeIP      IdentifierType = "ip"
	IdentifierTypeDevice  IdentifierType = "device"
	IdentifierTypeCookie  IdentifierType = "cookie"
	IdentifierTypeSession IdentifierType = "session"
)

type ProfileLabel struct {
	LabelType  LabelType `json:"label_type"`
	LabelValue string    `json:"label_value"`
	Score      int       `json:"score"`
	Reason     string    `json:"reason"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type LabelType string

const (
	LabelTypeTrust      LabelType = "trust"
	LabelTypeActivity   LabelType = "activity"
	LabelTypeFrequency  LabelType = "frequency"
	LabelTypeComplexity LabelType = "complexity"
)

type TrustLevel string

const (
	TrustLevelTrusted   TrustLevel = "trusted"
	TrustLevelSuspicious TrustLevel = "suspicious"
	TrustLevelHighRisk  TrustLevel = "high_risk"
)

type ActivityLevel int

const (
	ActivityLevelInactive ActivityLevel = 0
	ActivityLevelLow      ActivityLevel = 1
	ActivityLevelMedium   ActivityLevel = 2
	ActivityLevelHigh     ActivityLevel = 3
	ActivityLevelVeryHigh ActivityLevel = 4
)

type FrequencyLevel int

const (
	FrequencyLevelVeryLow  FrequencyLevel = 0
	FrequencyLevelLow      FrequencyLevel = 1
	FrequencyLevelNormal   FrequencyLevel = 2
	FrequencyLevelHigh     FrequencyLevel = 3
	FrequencyLevelVeryHigh FrequencyLevel = 4
)

type ComplexityLevel int

const (
	ComplexityLevelSimple   ComplexityLevel = 0
	ComplexityLevelNormal   ComplexityLevel = 1
	ComplexityLevelComplex  ComplexityLevel = 2
	ComplexityLevelVeryComplex ComplexityLevel = 3
)

type ProfileLabelSet struct {
	ProfileID   int64         `json:"profile_id"`
	TrustLevel TrustLevel    `json:"trust_level"`
	Activity   ActivityLevel `json:"activity_level"`
	Frequency  FrequencyLevel `json:"frequency_level"`
	Complexity ComplexityLevel `json:"complexity_level"`
	Labels     []ProfileLabel `json:"labels"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

type BehaviorMetrics struct {
	TotalVerifications int64            `json:"total_verifications"`
	SuccessRate        float64          `json:"success_rate"`
	AvgResponseTime    float64          `json:"avg_response_time"`
	RiskScore          float64          `json:"risk_score"`
	DeviceCount        int              `json:"device_count"`
	IPCount            int              `json:"ip_count"`
	LocationCount      int              `json:"location_count"`
	UniqueDays         int              `json:"unique_days"`
	Last7DaysActivity  int64            `json:"last_7_days_activity"`
	Last30DaysActivity int64           `json:"last_30_days_activity"`
	Trend              string           `json:"trend"`
}

type TimeWindow string

const (
	TimeWindowLast24Hours TimeWindow = "24h"
	TimeWindowLast7Days   TimeWindow = "7d"
	TimeWindowLast30Days  TimeWindow = "30d"
	TimeWindowAllTime     TimeWindow = "all"
)

type ProfileFilter struct {
	Identifier     string
	IdentifierType IdentifierType
	TrustLevel     TrustLevel
	ActivityLevel  ActivityLevel
	RiskScoreMin   float64
	RiskScoreMax   float64
	DateFrom       *time.Time
	DateTo         *time.Time
	Page           int
	PageSize       int
}

func (f *ProfileFilter) Offset() int {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	return (f.Page - 1) * f.PageSize
}

func (f *ProfileFilter) Limit() int {
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
	return f.PageSize
}

type ProfileDTO struct {
	ID               int64                  `json:"id"`
	Identifier       string                 `json:"identifier"`
	IdentifierType   IdentifierType         `json:"identifier_type"`
	BehaviorMetrics  BehaviorMetrics        `json:"behavior_metrics"`
	TrustLevel       TrustLevel             `json:"trust_level"`
	ActivityLevel    ActivityLevel          `json:"activity_level"`
	FrequencyLevel   FrequencyLevel         `json:"frequency_level"`
	ComplexityLevel  ComplexityLevel         `json:"complexity_level"`
	CaptchaPreferences map[string]int64     `json:"captcha_preferences"`
	LocationDistribution map[string]int64   `json:"location_distribution"`
	DeviceDistribution map[string]int64     `json:"device_distribution"`
	FirstSeenAt      string                 `json:"first_seen_at"`
	LastSeenAt       string                 `json:"last_seen_at"`
	UpdatedAt        string                 `json:"updated_at"`
}

func (p *UserProfile) ToDTO() *ProfileDTO {
	metrics := BehaviorMetrics{
		TotalVerifications: p.TotalAttempts,
		SuccessRate:        p.SuccessRate,
		AvgResponseTime:    p.AvgResponseTime,
	}

	labels := p.CalculateLabels()

	dto := &ProfileDTO{
		ID:                p.ID,
		Identifier:        p.Identifier,
		IdentifierType:    p.IdentifierType,
		BehaviorMetrics:   metrics,
		TrustLevel:        labels.TrustLevel,
		ActivityLevel:     labels.Activity,
		FrequencyLevel:    labels.Frequency,
		ComplexityLevel:   labels.Complexity,
		CaptchaPreferences: p.CaptchaTypeDistribution,
		LocationDistribution: p.LocationDistribution,
		DeviceDistribution: p.DeviceDistribution,
		FirstSeenAt:       p.FirstSeenAt.Format(time.RFC3339),
		LastSeenAt:        p.LastSeenAt.Format(time.RFC3339),
		UpdatedAt:         p.UpdatedAt.Format(time.RFC3339),
	}

	return dto
}

func (p *UserProfile) CalculateLabels() *ProfileLabelSet {
	labels := &ProfileLabelSet{
		ProfileID: p.ID,
		Labels:    make([]ProfileLabel, 0),
		UpdatedAt: time.Now(),
	}

	labels.TrustLevel = p.calculateTrustLevel()
	labels.Activity = p.calculateActivityLevel()
	labels.Frequency = p.calculateFrequencyLevel()
	labels.Complexity = p.calculateComplexityLevel()

	return labels
}

func (p *UserProfile) calculateTrustLevel() TrustLevel {
	if p.TotalAttempts < 5 {
		return TrustLevelSuspicious
	}

	highRiskRatio := float64(p.HighRiskEvents) / float64(p.TotalRiskEvents+1)

	if p.SuccessRate >= 90 && highRiskRatio < 0.1 {
		return TrustLevelTrusted
	} else if p.SuccessRate < 50 || highRiskRatio > 0.5 {
		return TrustLevelHighRisk
	}

	return TrustLevelSuspicious
}

func (p *UserProfile) calculateActivityLevel() ActivityLevel {
	if p.TotalAttempts == 0 {
		return ActivityLevelInactive
	}

	daysActive := len(p.ActiveDays)
	if daysActive >= 30 {
		return ActivityLevelVeryHigh
	} else if daysActive >= 15 {
		return ActivityLevelHigh
	} else if daysActive >= 7 {
		return ActivityLevelMedium
	} else if daysActive >= 1 {
		return ActivityLevelLow
	}

	return ActivityLevelInactive
}

func (p *UserProfile) calculateFrequencyLevel() FrequencyLevel {
	if p.TotalAttempts == 0 {
		return FrequencyLevelVeryLow
	}

	daysActive := len(p.ActiveDays)
	if daysActive == 0 {
		return FrequencyLevelVeryLow
	}

	avgPerDay := float64(p.TotalAttempts) / float64(daysActive)

	if avgPerDay >= 50 {
		return FrequencyLevelVeryHigh
	} else if avgPerDay >= 20 {
		return FrequencyLevelHigh
	} else if avgPerDay >= 5 {
		return FrequencyLevelNormal
	} else if avgPerDay >= 1 {
		return FrequencyLevelLow
	}

	return FrequencyLevelVeryLow
}

func (p *UserProfile) calculateComplexityLevel() ComplexityLevel {
	deviceCount := len(p.DeviceDistribution)
	ipCount := 0
	for _, count := range p.LocationDistribution {
		ipCount += int(count)
	}

	totalVariety := deviceCount + ipCount

	if totalVariety >= 10 {
		return ComplexityLevelVeryComplex
	} else if totalVariety >= 5 {
		return ComplexityLevelComplex
	} else if totalVariety >= 2 {
		return ComplexityLevelNormal
	}

	return ComplexityLevelSimple
}
