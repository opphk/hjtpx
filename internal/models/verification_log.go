package models

import (
	"time"

	"gorm.io/gorm"
)

type VerificationResult string
type VerificationLevel string

const (
	ResultSuccess    VerificationResult = "success"
	ResultFailed     VerificationResult = "failed"
	ResultExpired    VerificationResult = "expired"
	ResultInvalid    VerificationResult = "invalid"
	ResultRateLimit  VerificationResult = "rate_limit"

	LevelLow    VerificationLevel = "low"
	LevelMedium VerificationLevel = "medium"
	LevelHigh   VerificationLevel = "high"
)

type VerificationLog struct {
	ID              uint               `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
	DeletedAt       gorm.DeletedAt     `gorm:"index" json:"-"`
	UserID          uint               `gorm:"index" json:"user_id"`
	CaptchaID       uint               `gorm:"index" json:"captcha_id"`
	Token           string             `gorm:"index;size:64" json:"token"`
	Result          VerificationResult `gorm:"not null;size:20" json:"result"`
	Level           VerificationLevel  `gorm:"size:20" json:"level"`
	IPAddress       string             `gorm:"size:45;index" json:"ip_address"`
	UserAgent       string             `gorm:"size:512" json:"user_agent"`
	AppID           uint               `gorm:"index" json:"app_id"`
	ErrorMessage    string             `gorm:"type:text" json:"error_message,omitempty"`
	ResponseTime    int64              `gorm:"default:0" json:"response_time"`
	Challenge       string             `gorm:"size:255" json:"challenge"`
	BrowserFingerprint string         `gorm:"size:255" json:"browser_fingerprint"`
}

func (VerificationLog) TableName() string {
	return "verification_logs"
}

type CreateVerificationLogRequest struct {
	UserID          uint               `json:"user_id"`
	CaptchaID       uint               `json:"captcha_id"`
	Token           string             `json:"token"`
	Result          VerificationResult `json:"result"`
	Level           VerificationLevel  `json:"level"`
	IPAddress       string             `json:"ip_address"`
	UserAgent       string             `json:"user_agent"`
	AppID           uint               `json:"app_id"`
	ErrorMessage    string             `json:"error_message"`
	ResponseTime    int64              `json:"response_time"`
	Challenge       string             `json:"challenge"`
	BrowserFingerprint string          `json:"browser_fingerprint"`
}

type VerificationStats struct {
	TotalAttempts   int64   `json:"total_attempts"`
	SuccessCount    int64   `json:"success_count"`
	FailedCount     int64   `json:"failed_count"`
	SuccessRate     float64 `json:"success_rate"`
	AvgResponseTime float64 `json:"avg_response_time"`
}
