package model

import (
	"encoding/json"
	"time"
)

type CaptchaType string

const (
	CaptchaTypeSlider CaptchaType = "slider"
	CaptchaTypeClick  CaptchaType = "click"
	CaptchaTypeRotate CaptchaType = "rotate"
)

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

type Challenge struct {
	ID          int64           `db:"id" json:"id"`
	ChallengeID string          `db:"challenge_id" json:"challenge_id"`
	Type        CaptchaType    `db:"type" json:"type"`
	Difficulty  Difficulty      `db:"difficulty" json:"difficulty"`
	Data        json.RawMessage `db:"data" json:"data"`
	Solution    json.RawMessage `db:"solution" json:"solution"`
	ExpiresAt   time.Time       `db:"expires_at" json:"expires_at"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

type Attempt struct {
	ID             int64           `db:"id" json:"id"`
	ChallengeID    string          `db:"challenge_id" json:"challenge_id"`
	SessionID      string          `db:"session_id" json:"session_id"`
	UserAnswer     json.RawMessage `db:"user_answer" json:"user_answer"`
	IsValid        bool            `db:"is_valid" json:"is_valid"`
	ResponseTimeMs int             `db:"response_time_ms" json:"response_time_ms"`
	IPAddress      string          `db:"ip_address" json:"ip_address"`
	UserAgent      string          `db:"user_agent" json:"user_agent"`
	Fingerprint    string          `db:"fingerprint" json:"fingerprint"`
	RiskScore      float64         `db:"risk_score" json:"risk_score"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
}

type Admin struct {
	ID           int64      `db:"id" json:"id"`
	Username     string     `db:"username" json:"username"`
	PasswordHash string     `db:"password_hash" json:"-"`
	Role         string     `db:"role" json:"role"`
	IsActive     bool       `db:"is_active" json:"is_active"`
	LastLoginAt  *time.Time `db:"last_login_at" json:"last_login_at"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}

type Session struct {
	ID           int64      `db:"id" json:"id"`
	SessionID    string     `db:"session_id" json:"session_id"`
	Fingerprint  string     `db:"fingerprint" json:"fingerprint"`
	IPAddress    string     `db:"ip_address" json:"ip_address"`
	RiskScore    float64    `db:"risk_score" json:"risk_score"`
	AttemptCount int        `db:"attempt_count" json:"attempt_count"`
	BlockedUntil *time.Time `db:"blocked_until" json:"blocked_until"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
	ExpiresAt    time.Time  `db:"expires_at" json:"expires_at"`
}

type CaptchaStatus string

const (
	CaptchaStatusPending  CaptchaStatus = "pending"
	CaptchaStatusVerified CaptchaStatus = "verified"
	CaptchaStatusExpired  CaptchaStatus = "expired"
	CaptchaStatusFailed  CaptchaStatus = "failed"
)

type AdminUser struct {
	ID           int64     `db:"id" json:"id"`
	Username     string    `db:"username" json:"username"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         string    `db:"role" json:"role"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	LastLoginAt  *time.Time `db:"last_login_at" json:"last_login_at"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type Log struct {
	ID        int64     `db:"id" json:"id"`
	Level     string    `db:"level" json:"level"`
	Message   string    `db:"message" json:"message"`
	Metadata  string    `db:"metadata" json:"metadata"`
	IPAddress string    `db:"ip_address" json:"ip_address"`
	UserID    *int64    `db:"user_id" json:"user_id,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type DailyStats struct {
	Date              string `db:"date" json:"date"`
	ChallengeCount    int64  `db:"challenge_count" json:"challenge_count"`
	AttemptCount      int64  `db:"attempt_count" json:"attempt_count"`
	SuccessCount      int64  `db:"success_count" json:"success_count"`
	SuccessRate       float64 `db:"success_rate" json:"success_rate"`
	AvgResponseTimeMs int64  `db:"avg_response_time_ms" json:"avg_response_time_ms"`
}

type TotalStats struct {
	TotalChallenges   int64   `db:"total_challenges" json:"total_challenges"`
	TotalAttempts     int64   `db:"total_attempts" json:"total_attempts"`
	SuccessCount      int64   `db:"success_count" json:"success_count"`
	SuccessRate       float64 `db:"success_rate" json:"success_rate"`
	AvgResponseTimeMs int64   `db:"avg_response_time_ms" json:"avg_response_time_ms"`
	BlockedSessions   int64   `db:"blocked_sessions" json:"blocked_sessions"`
}
