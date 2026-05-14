package model

import (
	"database/sql"
	"time"
)

type Config struct {
	ID          int64          `json:"id"`
	Key         string         `json:"key"`
	Value       string         `json:"value"`
	Description sql.NullString `json:"-"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type ConfigDTO struct {
	ID          int64  `json:"id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	UpdatedAt   string `json:"updated_at"`
}

func (c *Config) ToDTO() *ConfigDTO {
	dto := &ConfigDTO{
		ID:        c.ID,
		Key:       c.Key,
		Value:     c.Value,
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
	}
	if c.Description.Valid {
		dto.Description = c.Description.String
	}
	return dto
}

type UpdateConfigRequest struct {
	Key   string `json:"key" binding:"required,max=100"`
	Value string `json:"value" binding:"required"`
}

type CreateConfigRequest struct {
	Key         string `json:"key" binding:"required,max=100"`
	Value       string `json:"value" binding:"required"`
	Description string `json:"description"`
}

type SystemConfig struct {
	MaxAttemptsPerIP   int  `json:"max_attempts_per_ip"`
	BlockDurationMins int  `json:"block_duration_minutes"`
	RiskThreshold      int  `json:"risk_threshold"`
	SessionTimeoutSecs int  `json:"session_timeout_seconds"`
	EnableWhitelist    bool `json:"enable_whitelist"`
	EnableBlacklist    bool `json:"enable_blacklist"`
}

func DefaultSystemConfig() *SystemConfig {
	return &SystemConfig{
		MaxAttemptsPerIP:   10,
		BlockDurationMins:  30,
		RiskThreshold:      70,
		SessionTimeoutSecs: 300,
		EnableWhitelist:    true,
		EnableBlacklist:    true,
	}
}
