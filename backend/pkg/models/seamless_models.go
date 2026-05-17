package models

import (
	"time"
)

type DeviceFingerprintRecord struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Fingerprint   string    `gorm:"size:64;uniqueIndex:idx_fp_fingerprint" json:"fingerprint"`
	UserID        *uint     `gorm:"index:idx_fp_user" json:"user_id,omitempty"`
	ApplicationID *uint     `gorm:"index:idx_fp_app" json:"application_id,omitempty"`
	CanvasHash    string    `gorm:"size:64" json:"canvas_hash"`
	WebGLHash     string    `gorm:"size:64" json:"webgl_hash"`
	AudioHash     string    `gorm:"size:64" json:"audio_hash"`
	FontHash      string    `gorm:"size:64" json:"font_hash"`
	ScreenHash    string    `gorm:"size:64" json:"screen_hash"`
	Timezone      string    `gorm:"size:50" json:"timezone"`
	Language      string    `gorm:"size:50" json:"language"`
	Platform      string    `gorm:"size:100" json:"platform"`
	UserAgent     string    `gorm:"size:500" json:"user_agent"`
	IPAddress     string    `gorm:"size:45" json:"ip_address"`
	ProxyDetected bool      `gorm:"default:false" json:"proxy_detected"`
	VPNDetected   bool      `gorm:"default:false" json:"vpn_detected"`
	TorDetected   bool      `gorm:"default:false" json:"tor_detected"`
	HostingDetected bool    `gorm:"default:false" json:"hosting_detected"`
	IsBot         bool      `gorm:"default:false" json:"is_bot"`
	FirstSeenAt   time.Time `json:"first_seen_at"`
	LastSeenAt    time.Time `gorm:"index:idx_fp_last_seen" json:"last_seen_at"`
	VisitCount    int       `gorm:"default:1" json:"visit_count"`
	IsTrusted     bool      `gorm:"default:false;index:idx_fp_trusted" json:"is_trusted"`
	TrustLevel    string    `gorm:"size:20;default:unknown" json:"trust_level"`
	RiskScore     float64   `gorm:"default:0" json:"risk_score"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type TrustLevel string

const (
	TrustLevelUnknown  TrustLevel = "unknown"
	TrustLevelNone    TrustLevel = "none"
	TrustLevelLow     TrustLevel = "low"
	TrustLevelMedium  TrustLevel = "medium"
	TrustLevelHigh    TrustLevel = "high"
	TrustLevelFull    TrustLevel = "full"
)

type DeviceTrust struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	UserID          uint       `gorm:"not null;index:idx_trust_user" json:"user_id"`
	FingerprintID  uint       `gorm:"not null;index:idx_trust_fingerprint" json:"fingerprint_id"`
	DeviceFingerprint string   `gorm:"size:64" json:"device_fingerprint"`
	DeviceName      string     `gorm:"size:255" json:"device_name"`
	TrustLevel      TrustLevel `gorm:"size:20;default:unknown" json:"trust_level"`
	TrustScore      float64    `gorm:"default:0" json:"trust_score"`
	IsTrusted       bool       `gorm:"default:false;index:idx_trust_is_trusted" json:"is_trusted"`
	SuccessCount    int        `gorm:"default:0" json:"success_count"`
	FailureCount    int        `gorm:"default:0" json:"failure_count"`
	ContinuousSuccess int       `gorm:"default:0" json:"continuous_success"`
	FirstTrustedAt  *time.Time `json:"first_trusted_at"`
	LastVerifiedAt  *time.Time `json:"last_verified_at"`
	LastFailedAt    *time.Time `json:"last_failed_at"`
	ExpiresAt       *time.Time `json:"expires_at"`
	IsAutoTrusted   bool       `gorm:"default:false" json:"is_auto_trusted"`
	TrustedBy       string     `gorm:"size:50" json:"trusted_by"`
	Note            string     `gorm:"type:text" json:"note"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type TrustLog struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	UserID          uint       `gorm:"not null;index:idx_trust_log_user" json:"user_id"`
	FingerprintID   uint       `gorm:"not null;index:idx_trust_log_fp" json:"fingerprint_id"`
	DeviceFingerprint string   `gorm:"size:64" json:"device_fingerprint"`
	Action          string     `gorm:"size:50;not null" json:"action"`
	OldTrustLevel   TrustLevel `gorm:"size:20" json:"old_trust_level"`
	NewTrustLevel   TrustLevel `gorm:"size:20" json:"new_trust_level"`
	OldTrustScore   float64    `json:"old_trust_score"`
	NewTrustScore   float64    `json:"new_trust_score"`
	Reason          string     `gorm:"type:text" json:"reason"`
	IPAddress       string     `gorm:"size:45" json:"ip_address"`
	UserAgent       string     `gorm:"size:500" json:"user_agent"`
	RiskScore       float64    `json:"risk_score"`
	CreatedAt       time.Time  `json:"created_at"`
}

type BehaviorTrail struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          *uint     `gorm:"index:idx_trail_user" json:"user_id,omitempty"`
	FingerprintID   uint      `gorm:"not null;index:idx_trail_fingerprint" json:"fingerprint_id"`
	SessionID       string    `gorm:"size:100;index:idx_trail_session" json:"session_id"`
	ApplicationID   *uint     `gorm:"index:idx_trail_app" json:"application_id,omitempty"`
	IPAddress       string    `gorm:"size:45;index:idx_trail_ip" json:"ip_address"`
	IPCountry       string    `gorm:"size:10" json:"ip_country"`
	IPRegion        string    `gorm:"size:50" json:"ip_region"`
	IPCity          string    `gorm:"size:50" json:"ip_city"`
	ISP             string    `gorm:"size:100" json:"isp"`
	UserAgent       string    `gorm:"size:500" json:"user_agent"`
	Browser         string    `gorm:"size:50" json:"browser"`
	BrowserVersion  string    `gorm:"size:50" json:"browser_version"`
	OS              string    `gorm:"size:50" json:"os"`
	OSVersion       string    `gorm:"size:50" json:"os_version"`
	DeviceType      string    `gorm:"size:50" json:"device_type"`
	DeviceBrand     string    `gorm:"size:50" json:"device_brand"`
	DeviceModel     string    `gorm:"size:100" json:"device_model"`
	ScreenWidth     int       `json:"screen_width"`
	ScreenHeight    int       `json:"screen_height"`
	Referrer        string    `gorm:"size:500" json:"referrer"`
	LandingPage     string    `gorm:"size:500" json:"landing_page"`
	VisitedPages    string    `gorm:"type:text" json:"visited_pages"`
	TimeOnSite      int       `json:"time_on_site"`
	ClickCount      int       `json:"click_count"`
	ScrollDepth     int       `json:"scroll_depth"`
	MouseMovement   bool      `gorm:"default:false" json:"mouse_movement"`
	HasTouchEvent   bool      `gorm:"default:false" json:"has_touch_event"`
	IsAnomaly       bool      `gorm:"default:false" json:"is_anomaly"`
	AnomalyReasons  string    `gorm:"type:text" json:"anomaly_reasons"`
	RiskScore       float64   `gorm:"default:0" json:"risk_score"`
	CreatedAt       time.Time `json:"created_at"`
}

type BehaviorPattern struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            uint      `gorm:"not null;index:idx_pattern_user" json:"user_id"`
	FingerprintID     uint      `gorm:"not null;index:idx_pattern_fp" json:"fingerprint_id"`
	PatternType       string    `gorm:"size:50;not null" json:"pattern_type"`
	PatternData       string    `gorm:"type:text" json:"pattern_data"`
	Confidence        float64   `gorm:"default:0" json:"confidence"`
	SampleCount       int       `gorm:"default:1" json:"sample_count"`
	LastMatchedAt     *time.Time `json:"last_matched_at"`
	IsAnomaly         bool      `gorm:"default:false" json:"is_anomaly"`
	AnomalyScore      float64   `gorm:"default:0" json:"anomaly_score"`
	FirstSeenAt       time.Time `json:"first_seen_at"`
	LastSeenAt        time.Time `json:"last_seen_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type AnomalyRecord struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          *uint     `gorm:"index:idx_anomaly_user" json:"user_id,omitempty"`
	FingerprintID   uint      `gorm:"not null;index:idx_anomaly_fp" json:"fingerprint_id"`
	SessionID       string    `gorm:"size:100" json:"session_id"`
	AnomalyType     string    `gorm:"size:50;not null" json:"anomaly_type"`
	Severity        string    `gorm:"size:20;not null" json:"severity"`
	Description     string    `gorm:"type:text" json:"description"`
	Evidence        string    `gorm:"type:text" json:"evidence"`
	RiskScore       float64   `json:"risk_score"`
	ActionTaken     string    `gorm:"size:50" json:"action_taken"`
	IPAddress       string    `gorm:"size:45" json:"ip_address"`
	IsResolved      bool      `gorm:"default:false" json:"is_resolved"`
	ResolvedAt      *time.Time `json:"resolved_at"`
	ResolvedBy      uint      `json:"resolved_by"`
	ResolveNote     string    `gorm:"type:text" json:"resolve_note"`
	CreatedAt       time.Time `json:"created_at"`
}

type VerificationRule struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"size:100;not null" json:"name"`
	Description     string    `gorm:"type:text" json:"description"`
	RuleType        string    `gorm:"size:50;not null" json:"rule_type"`
	Priority        int       `gorm:"default:0" json:"priority"`
	Condition       string    `gorm:"type:text" json:"condition"`
	ConditionType   string    `gorm:"size:20;default:and" json:"condition_type"`
	Action          string    `gorm:"size:50;not null" json:"action"`
	ActionValue     string    `gorm:"size:100" json:"action_value"`
	RiskScoreWeight float64   `gorm:"default:0" json:"risk_score_weight"`
	IsEnabled       bool      `gorm:"default:true" json:"is_enabled"`
	IsGlobal        bool      `gorm:"default:false" json:"is_global"`
	ApplicationIDs  string    `gorm:"type:text" json:"application_ids"`
	EffectStartTime *time.Time `json:"effect_start_time"`
	EffectEndTime   *time.Time `json:"effect_end_time"`
	HitCount        int       `gorm:"default:0" json:"hit_count"`
	LastHitAt       *time.Time `json:"last_hit_at"`
	CreatedBy       uint      `json:"created_by"`
	UpdatedBy       uint      `json:"updated_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type VerificationDecision struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	SessionID       string    `gorm:"size:100;index:idx_decision_session" json:"session_id"`
	FingerprintID   uint      `gorm:"not null;index:idx_decision_fp" json:"fingerprint_id"`
	UserID          *uint     `gorm:"index:idx_decision_user" json:"user_id,omitempty"`
	ApplicationID   *uint     `gorm:"index:idx_decision_app" json:"application_id,omitempty"`
	Decision        string    `gorm:"size:20;not null" json:"decision"`
	DecisionType    string    `gorm:"size:50" json:"decision_type"`
	BaseRiskScore   float64   `json:"base_risk_score"`
	FinalRiskScore  float64   `json:"final_risk_score"`
	TrustScore      float64   `json:"trust_score"`
	Factors         string    `gorm:"type:text" json:"factors"`
	MatchedRules    string    `gorm:"type:text" json:"matched_rules"`
	UnmatchedRules  string    `gorm:"type:text" json:"unmatched_rules"`
	Recommendation  string    `gorm:"size:50" json:"recommendation"`
	ChallengeType   string    `gorm:"size:50" json:"challenge_type"`
	ProcessingTime  int64     `json:"processing_time"`
	IsOverridden    bool      `gorm:"default:false" json:"is_overridden"`
	OverrideBy      uint      `json:"override_by"`
	OverrideReason  string    `gorm:"type:text" json:"override_reason"`
	CreatedAt       time.Time `json:"created_at"`
}

type SeamlessVerificationSession struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	SessionID       string    `gorm:"size:100;uniqueIndex:idx_seamless_session_id" json:"session_id"`
	FingerprintID   uint      `gorm:"not null;index:idx_seamless_fp" json:"fingerprint_id"`
	DeviceFingerprint string  `gorm:"size:64" json:"device_fingerprint"`
	UserID          *uint     `gorm:"index:idx_seamless_user" json:"user_id,omitempty"`
	ApplicationID   *uint     `gorm:"index:idx_seamless_app" json:"application_id,omitempty"`
	RequestPath     string    `gorm:"size:500" json:"request_path"`
	RequestMethod   string    `gorm:"size:10" json:"request_method"`
	IPAddress       string    `gorm:"size:45" json:"ip_address"`
	IPCountry       string    `gorm:"size:10" json:"ip_country"`
	IPRegion        string    `gorm:"size:50" json:"ip_region"`
	IPCity          string    `gorm:"size:50" json:"ip_city"`
	UserAgent       string    `gorm:"size:500" json:"user_agent"`
	Decision        string    `gorm:"size:20;not null" json:"decision"`
	RiskScore       float64   `json:"risk_score"`
	TrustScore      float64   `json:"trust_score"`
	TrustLevel      TrustLevel `gorm:"size:20" json:"trust_level"`
	IsChallenge     bool      `gorm:"default:false" json:"is_challenge"`
	ChallengeType   string    `gorm:"size:50" json:"challenge_type"`
	ChallengePassed *bool     `json:"challenge_passed"`
	ChallengeTime   int64     `json:"challenge_time"`
	Factors         string    `gorm:"type:text" json:"factors"`
	IsAnomaly       bool      `gorm:"default:false" json:"is_anomaly"`
	AnomalyReasons  string    `gorm:"type:text" json:"anomaly_reasons"`
	ProcessingTime  int64     `json:"processing_time"`
	CreatedAt       time.Time `json:"created_at"`
}

type IPReputation struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	IPAddress       string    `gorm:"size:45;uniqueIndex:idx_ip_address" json:"ip_address"`
	Country         string    `gorm:"size:10" json:"country"`
	Region          string    `gorm:"size:50" json:"region"`
	City            string    `gorm:"size:50" json:"city"`
	ISP             string    `gorm:"size:100" json:"isp"`
	IsProxy         bool      `gorm:"default:false" json:"is_proxy"`
	IsVPN           bool      `gorm:"default:false" json:"is_vpn"`
	IsTor           bool      `gorm:"default:false" json:"is_tor"`
	IsHosting       bool      `gorm:"default:false" json:"is_hosting"`
	IsDatacenter    bool      `gorm:"default:false" json:"is_datacenter"`
	IsResidential   bool      `gorm:"default:false" json:"is_residential"`
	IsMobile        bool      `gorm:"default:false" json:"is_mobile"`
	ThreatLevel     string    `gorm:"size:20;default:low" json:"threat_level"`
	ThreatTypes     string    `gorm:"type:text" json:"threat_types"`
	ReputationScore float64   `gorm:"default:100" json:"reputation_score"`
	RequestCount    int       `gorm:"default:0" json:"request_count"`
	BlockCount      int       `gorm:"default:0" json:"block_count"`
	FirstSeenAt     time.Time `json:"first_seen_at"`
	LastSeenAt      time.Time `json:"last_seen_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
