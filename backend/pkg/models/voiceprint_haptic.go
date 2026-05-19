package models

import (
	"time"
)

type VoiceprintCaptchaSession struct {
	ID             int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID      string     `gorm:"size:100;uniqueIndex;not null" json:"session_id"`
	Pattern        string     `gorm:"type:text;not null" json:"pattern"`
	TargetPhrase   string     `gorm:"size:50;not null" json:"target_phrase"`
	Status         string     `gorm:"size:50;default:pending" json:"status"`
	VerifyCount    int        `gorm:"default:0" json:"verify_count"`
	MaxAttempts    int        `gorm:"default:3" json:"max_attempts"`
	SimilarityScore float64   `gorm:"default:0" json:"similarity_score"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiredAt      time.Time  `json:"expired_at"`
	VerifiedAt     *time.Time `json:"verified_at"`
	ClientIP       string     `gorm:"size:50" json:"client_ip"`
	UserAgent      string     `gorm:"size:500" json:"user_agent"`
	Fingerprint    string     `gorm:"size:255" json:"fingerprint"`
}

func (VoiceprintCaptchaSession) TableName() string {
	return "voiceprint_captcha_sessions"
}

type HapticCaptchaSession struct {
	ID             int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID      string     `gorm:"size:100;uniqueIndex;not null" json:"session_id"`
	Pattern        string     `gorm:"type:text;not null" json:"pattern"`
	PatternType    string     `gorm:"size:50;default:sequence" json:"pattern_type"`
	Difficulty     string     `gorm:"size:20;default:medium" json:"difficulty"`
	Status         string     `gorm:"size:50;default:pending" json:"status"`
	VerifyCount    int        `gorm:"default:0" json:"verify_count"`
	MaxAttempts    int        `gorm:"default:3" json:"max_attempts"`
	MatchScore     float64    `gorm:"default:0" json:"match_score"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiredAt      time.Time  `json:"expired_at"`
	VerifiedAt     *time.Time `json:"verified_at"`
	ClientIP       string     `gorm:"size:50" json:"client_ip"`
	UserAgent      string     `gorm:"size:500" json:"user_agent"`
	Fingerprint    string     `gorm:"size:255" json:"fingerprint"`
}

func (HapticCaptchaSession) TableName() string {
	return "haptic_captcha_sessions"
}
