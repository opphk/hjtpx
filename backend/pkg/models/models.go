package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username        string          `gorm:"size:100;uniqueIndex;not null" json:"username"`
	Email           string          `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash    string          `gorm:"size:255;not null" json:"-"`
	Nickname        string          `gorm:"size:100" json:"nickname"`
	Avatar          string          `gorm:"size:500" json:"avatar"`
	Phone           string          `gorm:"size:20" json:"phone"`
	Bio             string          `gorm:"size:500" json:"bio"`
	IsVerified      bool            `gorm:"default:false" json:"is_verified"`
	VerifiedAt      *time.Time      `json:"verified_at,omitempty"`
	VerificationToken string        `gorm:"size:100" json:"-"`
	PasswordResetToken string       `gorm:"size:100" json:"-"`
	PasswordResetAt *time.Time      `json:"password_reset_at,omitempty"`
	LoginCount      int             `gorm:"default:0" json:"login_count"`
	LastLoginAt     *time.Time      `json:"last_login_at,omitempty"`
	LastLoginIP     string          `gorm:"size:50" json:"last_login_ip"`
	Status          string          `gorm:"size:20;default:active" json:"status"`
	Applications    []Application   `gorm:"foreignKey:UserID" json:"applications,omitempty"`
	Verifications   []Verification  `gorm:"foreignKey:UserID" json:"verifications,omitempty"`
}

type Admin struct {
	gorm.Model
	Username       string `gorm:"size:100;uniqueIndex;not null" json:"username"`
	PasswordHash   string `gorm:"size:255;not null" json:"-"`
	IsSuperAdmin   bool   `gorm:"default:false" json:"is_super_admin"`
}

type Application struct {
	gorm.Model
	Name           string          `gorm:"size:255;not null" json:"name"`
	UserID         uint            `gorm:"not null;index" json:"user_id"`
	Description    string          `gorm:"type:text" json:"description,omitempty"`
	APIKey         string          `gorm:"size:255;uniqueIndex" json:"api_key"`
	Domain         string          `gorm:"size:255" json:"domain,omitempty"`
	Website        string          `gorm:"size:255" json:"website,omitempty"`
	IsActive       bool            `gorm:"default:true" json:"is_active"`
	Config         string          `gorm:"type:text" json:"config,omitempty"`
	User           User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Verifications  []Verification  `gorm:"foreignKey:ApplicationID" json:"verifications,omitempty"`
	APIKeyHistories []APIKeyHistory `gorm:"foreignKey:ApplicationID" json:"api_key_histories,omitempty"`
}

type APIKeyHistory struct {
	gorm.Model
	ApplicationID uint       `gorm:"not null;index" json:"application_id"`
	OldAPIKey     string     `gorm:"size:255" json:"old_api_key"`
	NewAPIKey     string     `gorm:"size:255" json:"new_api_key"`
	ChangedAt     time.Time  `json:"changed_at"`
	Application   Application `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
}

type Verification struct {
	gorm.Model
	ApplicationID  *uint           `gorm:"index" json:"application_id,omitempty"`
	UserID         *uint           `gorm:"index" json:"user_id,omitempty"`
	SessionID      string          `gorm:"size:100;index" json:"session_id"`
	CaptchaType    string          `gorm:"size:50" json:"captcha_type"`
	Status         string          `gorm:"size:50;not null;default:pending" json:"status"`
	IPAddress      string          `gorm:"size:50" json:"ip_address"`
	UserAgent      string          `gorm:"size:500" json:"user_agent"`
	RiskScore      float64         `gorm:"default:0" json:"risk_score"`
	Duration       int64           `gorm:"comment:'验证耗时(毫秒)'" json:"duration"`
	BehaviorData   []BehaviorData  `gorm:"foreignKey:VerificationID" json:"behavior_data,omitempty"`
	Application    *Application    `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
	User           *User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type BehaviorData struct {
	gorm.Model
	VerificationID uint           `gorm:"not null;index" json:"verification_id"`
	Data           string         `gorm:"type:text" json:"data"`
	DataType       string         `gorm:"size:100" json:"data_type"`
	Timestamp      time.Time      `json:"timestamp"`
	Verification   Verification   `gorm:"foreignKey:VerificationID" json:"verification,omitempty"`
}

type Blacklist struct {
	gorm.Model
	Target         string    `gorm:"size:255;not null;index" json:"target"`
	Type           string    `gorm:"size:50;not null;index" json:"type"`
	Source         string    `gorm:"size:50;default:manual" json:"source"`
	Reason         string    `gorm:"type:text" json:"reason,omitempty"`
	Action         string    `gorm:"size:50;default:block" json:"action"`
	Status         string    `gorm:"size:50;default:active" json:"status"`
	Note           string    `gorm:"type:text" json:"note,omitempty"`
	CreatedBy      uint      `gorm:"default:0" json:"created_by"`
	HitCount       int       `gorm:"default:0" json:"hit_count"`
	ApplicationIDs string    `gorm:"type:text" json:"application_ids,omitempty"`
	Expiration     string    `gorm:"size:50" json:"expiration,omitempty"`
}

type VerificationLog struct {
	gorm.Model
	VerificationID uint           `gorm:"not null;index" json:"verification_id"`
	SessionID      string         `gorm:"size:100;index" json:"session_id"`
	ApplicationID  uint           `gorm:"not null;index" json:"application_id"`
	CaptchaType    string         `gorm:"size:50" json:"captcha_type"`
	Status         string         `gorm:"size:50;not null" json:"status"`
	IPAddress      string         `gorm:"size:50" json:"ip_address"`
	UserAgent      string         `gorm:"size:500" json:"user_agent"`
	RiskScore      float64        `gorm:"default:0" json:"risk_score"`
	AnalysisResult string         `gorm:"type:text" json:"analysis_result"`
	Duration       int64          `gorm:"comment:'验证耗时(毫秒)'" json:"duration"`
	Verification   Verification   `gorm:"foreignKey:VerificationID" json:"verification,omitempty"`
	Application    Application    `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
}

type DeviceFingerprint struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Fingerprint   string    `gorm:"uniqueIndex;size:64" json:"fingerprint"`
	CanvasHash    string    `gorm:"size:64" json:"canvas_hash"`
	WebGLVendor   string    `gorm:"size:100" json:"webgl_vendor"`
	WebGLRenderer string    `gorm:"size:100" json:"webgl_renderer"`
	UserAgent     string    `gorm:"size:500" json:"user_agent"`
	IPAddress     string    `gorm:"size:45" json:"ip_address"`
	ScreenInfo    string    `gorm:"size:100" json:"screen_info"`
	Timezone      string    `gorm:"size:100" json:"timezone"`
	Language      string    `gorm:"size:50" json:"language"`
	Fonts         string    `gorm:"size:500" json:"fonts"`
	Plugins       string    `gorm:"size:500" json:"plugins"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
	VisitCount    int       `gorm:"default:1" json:"visit_count"`
	IsBot         bool      `gorm:"default:false" json:"is_bot"`
	RiskLevel     string    `gorm:"size:20;default:low" json:"risk_level"`
	RiskScore     float64   `gorm:"default:0" json:"risk_score"`
	ProxyDetected bool      `gorm:"default:false" json:"proxy_detected"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
