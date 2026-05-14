package model

import (
	"database/sql"
	"time"
)

type CaptchaLog struct {
	ID         int64          `json:"id"`
	Type       string         `json:"captcha_type"`
	ClientID   string         `json:"client_id"`
	IP         string         `json:"ip"`
	UserAgent  sql.NullString `json:"-"`
	Result     bool           `json:"result"`
	Duration   int            `json:"duration"`
	RiskScore  int            `json:"risk_score"`
	CreatedAt  time.Time     `json:"created_at"`
}

type CaptchaLogDTO struct {
	ID         int64  `json:"id"`
	Type       string `json:"captcha_type"`
	ClientID   string `json:"client_id"`
	IP         string `json:"ip"`
	UserAgent  string `json:"user_agent,omitempty"`
	Result     bool   `json:"result"`
	Duration   int    `json:"duration"`
	RiskScore  int    `json:"risk_score"`
	CreatedAt  string `json:"created_at"`
}

func (c *CaptchaLog) ToDTO() *CaptchaLogDTO {
	dto := &CaptchaLogDTO{
		ID:        c.ID,
		Type:      c.Type,
		ClientID:  c.ClientID,
		IP:        c.IP,
		Result:    c.Result,
		Duration:  c.Duration,
		RiskScore: c.RiskScore,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
	}
	if c.UserAgent.Valid {
		dto.UserAgent = c.UserAgent.String
	}
	return dto
}

type CreateCaptchaLogRequest struct {
	Type      string `json:"captcha_type" binding:"required,oneof=slider click puzzle"`
	ClientID  string `json:"client_id" binding:"required,max=64"`
	IP        string `json:"ip" binding:"required,max=45"`
	UserAgent string `json:"user_agent"`
	Result    bool   `json:"result"`
	Duration  int    `json:"duration" binding:"min=0"`
	RiskScore int    `json:"risk_score" binding:"min=0,max=100"`
}

type CaptchaLogFilter struct {
	StartDate  *time.Time
	EndDate    *time.Time
	Type       string
	ClientID   string
	IP         string
	Result     *bool
	MinScore   int
	MaxScore   int
	Page       int
	PageSize   int
}

func (f *CaptchaLogFilter) Offset() int {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	return (f.Page - 1) * f.PageSize
}

func (f *CaptchaLogFilter) Limit() int {
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
	return f.PageSize
}

type CaptchaLogStats struct {
	TotalCount     int64            `json:"total_count"`
	SuccessCount   int64            `json:"success_count"`
	FailCount      int64            `json:"fail_count"`
	SuccessRate    float64          `json:"success_rate"`
	AvgDuration    float64          `json:"avg_duration"`
	AvgRiskScore   float64          `json:"avg_risk_score"`
	ByType         map[string]int64 `json:"by_type"`
	ByHour         []HourlyStat     `json:"by_hour"`
}

type HourlyStat struct {
	Hour       time.Time `json:"hour"`
	TotalCount int64     `json:"total_count"`
	SuccessCount int64   `json:"success_count"`
	FailCount    int64   `json:"fail_count"`
}
