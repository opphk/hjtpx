package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username        string          `gorm:"size:100;uniqueIndex;not null" json:"username"`
	Email           string          `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash    string          `gorm:"size:255;not null" json:"password_hash"`
	IsVerified      bool            `gorm:"default:false" json:"is_verified"`
	VerifiedAt      *time.Time      `gorm:"default:null" json:"verified_at,omitempty"`
	Applications    []Application   `gorm:"foreignKey:UserID" json:"applications,omitempty"`
	Verifications   []Verification  `gorm:"foreignKey:UserID" json:"verifications,omitempty"`
}

type Admin struct {
	gorm.Model
	Username       string `gorm:"size:100;uniqueIndex;not null" json:"username"`
	PasswordHash   string `gorm:"size:255;not null" json:"password_hash"`
	IsSuperAdmin   bool   `gorm:"default:false" json:"is_super_admin"`
}

type Application struct {
	gorm.Model
	Name           string          `gorm:"size:255;not null" json:"name"`
	UserID         uint            `gorm:"not null;index" json:"user_id"`
	Description    string          `gorm:"type:text" json:"description,omitempty"`
	APIKey         string          `gorm:"size:255;uniqueIndex" json:"api_key"`
	IsActive       bool            `gorm:"default:true" json:"is_active"`
	User           User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Verifications  []Verification  `gorm:"foreignKey:ApplicationID" json:"verifications,omitempty"`
}

type Verification struct {
	gorm.Model
	ApplicationID  uint            `gorm:"not null;index" json:"application_id"`
	UserID         uint            `gorm:"not null;index" json:"user_id"`
	SessionID      string          `gorm:"size:100;index" json:"session_id"`
	CaptchaType    string          `gorm:"size:50" json:"captcha_type"`
	Status         string          `gorm:"size:50;not null;default:pending" json:"status"`
	IPAddress      string          `gorm:"size:50" json:"ip_address"`
	UserAgent      string          `gorm:"size:500" json:"user_agent"`
	RiskScore      float64         `gorm:"default:0" json:"risk_score"`
	BehaviorData   []BehaviorData  `gorm:"foreignKey:VerificationID" json:"behavior_data,omitempty"`
	Application    Application     `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
	User           User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type BehaviorData struct {
	gorm.Model
	SessionID      string         `gorm:"size:100;index" json:"session_id"`
	VerificationID uint           `gorm:"not null;index" json:"verification_id"`
	Data           string         `gorm:"type:text" json:"data"`
	DataType       string         `gorm:"size:100" json:"data_type"`
	Timestamp      time.Time      `json:"timestamp"`
	Verification   Verification   `gorm:"foreignKey:VerificationID" json:"verification,omitempty"`
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

type SilentVerification struct {
	gorm.Model
	Token             string  `gorm:"size:100;uniqueIndex;not null" json:"token"`
	SessionID         string  `gorm:"size:100;index" json:"session_id"`
	UserID            uint    `gorm:"not null;index" json:"user_id"`
	ApplicationID     uint    `gorm:"not null;index" json:"application_id"`
	DeviceFingerprint string  `gorm:"size:255" json:"device_fingerprint"`
	RiskLevel         string  `gorm:"size:20;not null;default:'low'" json:"risk_level"`
	RiskScore         float64 `gorm:"default:0" json:"risk_score"`
	DeviceScore       float64 `gorm:"default:0" json:"device_score"`
	BehaviorScore     float64 `gorm:"default:0" json:"behavior_score"`
	HistoryScore      float64 `gorm:"default:0" json:"history_score"`
	Status            string  `gorm:"size:20;not null;default:'pending'" json:"status"`
	NeedCaptcha       bool    `gorm:"default:false" json:"need_captcha"`
	CaptchaType       string  `gorm:"size:20" json:"captcha_type"`
	CaptchaToken      string  `gorm:"size:100" json:"captcha_token"`
	IPAddress         string  `gorm:"size:50" json:"ip_address"`
	UserAgent         string  `gorm:"size:500" json:"user_agent"`
	VerifyDuration    int64   `json:"verify_duration"`
	BehaviorData      string  `gorm:"type:text" json:"behavior_data"`
	User              User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Application       Application `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
}

type DeviceTrustHistory struct {
	gorm.Model
	DeviceFingerprint string    `gorm:"size:255;not null;index" json:"device_fingerprint"`
	UserID           uint      `gorm:"not null;index" json:"user_id"`
	TrustScore       float64   `gorm:"default:0" json:"trust_score"`
	LastSeen         time.Time `json:"last_seen"`
	LoginCount       int       `gorm:"default:0" json:"login_count"`
	IPAddress        string    `gorm:"size:50" json:"ip_address"`
	IsTrusted        bool      `gorm:"default:false" json:"is_trusted"`
	User             User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type UserBehaviorProfile struct {
	gorm.Model
	UserID             uint      `gorm:"not null;uniqueIndex" json:"user_id"`
	AvgTypingSpeed     float64   `gorm:"default:0" json:"avg_typing_speed"`
	AvgMouseSpeed      float64   `gorm:"default:0" json:"avg_mouse_speed"`
	ClickFrequency     float64   `gorm:"default:0" json:"click_frequency"`
	ScrollFrequency    float64   `gorm:"default:0" json:"scroll_frequency"`
	CommonLoginTime    string    `gorm:"size:50" json:"common_login_time"`
	CommonLocation     string    `gorm:"size:255" json:"common_location"`
	LastUpdated        time.Time `json:"last_updated"`
	SampleSize         int       `gorm:"default:0" json:"sample_size"`
	User               User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type DeviceFingerprint struct {
	gorm.Model
	UserID          uint           `gorm:"not null;index" json:"user_id"`
	FingerprintHash string         `gorm:"size:64;uniqueIndex" json:"fingerprint_hash"`
	UserAgent       string         `gorm:"type:text" json:"user_agent"`
	ScreenInfo      string         `gorm:"type:text" json:"screen_info"`
	BrowserInfo     string         `gorm:"type:text" json:"browser_info"`
	PlatformInfo    string         `gorm:"type:text" json:"platform_info"`
	CanvasHash      string         `gorm:"size:64" json:"canvas_hash"`
	WebGLHash       string         `gorm:"size:64" json:"webgl_hash"`
	AudioHash       string         `gorm:"size:64" json:"audio_hash"`
	FirstSeenAt     time.Time      `json:"first_seen_at"`
	LastSeenAt      time.Time      `json:"last_seen_at"`
	VisitCount      int            `gorm:"default:1" json:"visit_count"`
	IsTrusted       bool           `gorm:"default:false" json:"is_trusted"`
	RiskLevel       string         `gorm:"size:20" json:"risk_level"`
	DeviceHistory   []DeviceHistory `gorm:"foreignKey:FingerprintID" json:"device_history,omitempty"`
}

type DeviceHistory struct {
	gorm.Model
	FingerprintID uint           `gorm:"not null;index" json:"fingerprint_id"`
	IPAddress     string         `gorm:"size:50" json:"ip_address"`
	Location      string         `gorm:"size:255" json:"location"`
	LoginTime     time.Time      `json:"login_time"`
	LoginSuccess  bool           `gorm:"default:false" json:"login_success"`
	UserAgent     string         `gorm:"type:text" json:"user_agent"`
	DeviceFingerprint DeviceFingerprint `gorm:"foreignKey:FingerprintID" json:"device_fingerprint,omitempty"`
}
