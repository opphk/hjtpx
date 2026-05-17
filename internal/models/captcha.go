package models

import (
	"time"

	"gorm.io/gorm"
)

type CaptchaType string
type CaptchaStatus int

const (
	CaptchaTypeImage CaptchaType = "image"
	CaptchaTypeVideo CaptchaType = "video"
	CaptchaTypeAudio CaptchaType = "audio"

	CaptchaStatusPending   CaptchaStatus = 0
	CaptchaStatusVerified  CaptchaStatus = 1
	CaptchaStatusExpired   CaptchaStatus = 2
	CaptchaStatusFailed    CaptchaStatus = 3
)

type Captcha struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Token       string         `gorm:"uniqueIndex;not null;size:64" json:"token"`
	Challenge   string         `gorm:"not null;size:255" json:"challenge"`
	Type        CaptchaType    `gorm:"not null;size:20" json:"type"`
	Status      CaptchaStatus  `gorm:"default:0;index" json:"status"`
	ExpiresAt   time.Time      `gorm:"index" json:"expires_at"`
	UserID      uint           `gorm:"index" json:"user_id"`
	AppID       uint           `gorm:"index" json:"app_id"`
	IPAddress   string         `gorm:"size:45" json:"ip_address"`
	UserAgent   string         `gorm:"size:512" json:"user_agent"`
	VerifyCount int            `gorm:"default:0" json:"verify_count"`
	MaxVerify   int            `gorm:"default:3" json:"max_verify"`
	Metadata    string         `gorm:"type:text" json:"metadata"`
}

func (Captcha) TableName() string {
	return "captchas"
}

func (c *Captcha) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

func (c *Captcha) IsVerified() bool {
	return c.Status == CaptchaStatusVerified
}

func (c *Captcha) CanVerify() bool {
	return !c.IsExpired() && c.VerifyCount < c.MaxVerify && c.Status == CaptchaStatusPending
}

type CreateCaptchaRequest struct {
	Type      CaptchaType `json:"type" binding:"required"`
	UserID    uint        `json:"user_id"`
	AppID     uint        `json:"app_id"`
	IPAddress string      `json:"ip_address"`
	UserAgent string      `json:"user_agent"`
	Metadata  string      `json:"metadata"`
}

type VerifyCaptchaRequest struct {
	Token     string `json:"token" binding:"required"`
	Challenge string `json:"challenge" binding:"required"`
}

type CaptchaResponse struct {
	Token     string      `json:"token"`
	Type      CaptchaType `json:"type"`
	ExpiresAt int64       `json:"expires_at"`
	ImageURL  string      `json:"image_url,omitempty"`
}
