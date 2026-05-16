package models

import (
	"time"
)

type DeviceTrust struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Fingerprint       string    `gorm:"size:64;uniqueIndex" json:"fingerprint"`
	TrustScore        int       `gorm:"default:50;comment:'可信度评分0-100'" json:"trust_score"`
	TrustLevel        string    `gorm:"size:20;default:medium;comment:'minimal/low/medium/high/full'" json:"trust_level"`
	VisitCount        int       `gorm:"default:1" json:"visit_count"`
	SuccessfulLogins  int       `gorm:"default:0" json:"successful_logins"`
	FailedLogins      int       `gorm:"default:0" json:"failed_logins"`
	LastVisit         time.Time `json:"last_visit"`
	FirstVisit        time.Time `json:"first_visit"`
	LastSuccessfulLogin *time.Time `json:"last_successful_login,omitempty"`
	LastFailedLogin   *time.Time `json:"last_failed_login,omitempty"`
	IsWhitelisted     bool      `gorm:"default:false" json:"is_whitelisted"`
	WhitelistedAt     *time.Time `json:"whitelisted_at,omitempty"`
	WhitelistedBy     *uint     `json:"whitelisted_by,omitempty"`
	WhitelistReason   string    `gorm:"size:255" json:"whitelist_reason,omitempty"`
	RiskScore         float64   `gorm:"default:0" json:"risk_score"`
	RiskFactors       string    `gorm:"type:text;comment:'JSON格式的风险因素'" json:"risk_factors,omitempty"`
	IPAddress         string    `gorm:"size:45" json:"ip_address"`
	UserAgent         string    `gorm:"size:500" json:"user_agent"`
	IsVerified        bool      `gorm:"default:false;comment:'是否已通过验证'" json:"is_verified"`
	VerificationCount int       `gorm:"default:0" json:"verification_count"`
	LastVerified      *time.Time `json:"last_verified,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type WhitelistEntry struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	Target         string     `gorm:"size:255;not null;index" json:"target"`
	Type           string     `gorm:"size:50;not null;index;comment:'fingerprint/ip/user/application'" json:"type"`
	TargetID       string     `gorm:"size:64;index" json:"target_id,omitempty"`
	Reason         string     `gorm:"size:255" json:"reason,omitempty"`
	AddedBy        uint       `gorm:"default:0" json:"added_by"`
	AddedByName    string     `gorm:"size:100" json:"added_by_name,omitempty"`
	ApplicationID  *uint      `gorm:"index" json:"application_id,omitempty"`
	Status         string     `gorm:"size:20;default:active;comment:'active/expired/revoked'" json:"status"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type TrustHistory struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Fingerprint   string    `gorm:"size:64;index" json:"fingerprint"`
	IPAddress     string    `gorm:"size:45" json:"ip_address"`
	Event         string    `gorm:"size:50;not null" json:"event"`
	EventType     string    `gorm:"size:50" json:"event_type"`
	PreviousScore int       `json:"previous_score"`
	NewScore      int       `json:"new_score"`
	PreviousLevel string    `gorm:"size:20" json:"previous_level"`
	NewLevel      string    `gorm:"size:20" json:"new_level"`
	RiskScore     float64   `json:"risk_score"`
	RiskFactors   string    `gorm:"type:text" json:"risk_factors,omitempty"`
	SessionID     string    `gorm:"size:100" json:"session_id,omitempty"`
	RequestData   string    `gorm:"type:text" json:"request_data,omitempty"`
	UserAgent     string    `gorm:"size:500" json:"user_agent,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type ProgressiveVerification struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	SessionID        string     `gorm:"size:100;index" json:"session_id"`
	Fingerprint      string     `gorm:"size:64;index" json:"fingerprint"`
	CurrentLevel     int        `gorm:"default:0;comment:'0-静默/1-轻量/2-中等/3-严格'" json:"current_level"`
	TargetLevel      int        `gorm:"default:0" json:"target_level"`
	RequiredScore    int        `gorm:"default:0" json:"required_score"`
	CurrentScore     int        `gorm:"default:0" json:"current_score"`
	TrustScore       int        `gorm:"default:50" json:"trust_score"`
	Passed           bool       `gorm:"default:false" json:"passed"`
	VerificationType string     `gorm:"size:50;default:silent" json:"verification_type"`
	VerificationData string     `gorm:"type:text" json:"verification_data,omitempty"`
	ChallengeToken   string     `gorm:"size:100" json:"challenge_token,omitempty"`
	ChallengeResult  string     `gorm:"size:50" json:"challenge_result,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	ExpiresAt        time.Time  `json:"expires_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

const (
	TrustLevelMinimal = "minimal"
	TrustLevelLow     = "low"
	TrustLevelMedium  = "medium"
	TrustLevelHigh    = "high"
	TrustLevelFull    = "full"
)

const (
	VerificationTypeSilent   = "silent"
	VerificationTypeLight    = "light"
	VerificationTypeModerate  = "moderate"
	VerificationTypeStrict   = "strict"
)

const (
	WhitelistTypeFingerprint = "fingerprint"
	WhitelistTypeIP         = "ip"
	WhitelistTypeUser       = "user"
	WhitelistTypeApplication = "application"
)

const (
	EventTypeLogin         = "login"
	EventTypeLoginSuccess  = "login_success"
	EventTypeLoginFailed   = "login_failed"
	EventTypeTrustIncrease = "trust_increase"
	EventTypeTrustDecrease = "trust_decrease"
	EventTypeWhitelist     = "whitelist"
	EventTypeVerify        = "verify"
	EventTypeRiskDetected  = "risk_detected"
)
