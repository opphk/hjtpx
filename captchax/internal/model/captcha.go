package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Captcha struct {
	ID         string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	AppID      string         `gorm:"type:varchar(64);index;not null" json:"app_id"`
	Type       string         `gorm:"type:varchar(20);not null" json:"type"`
	Answer     string         `gorm:"type:varchar(128);not null" json:"-"`
	ImageData  string         `gorm:"type:text;not null" json:"image_data"`
	Status     int            `gorm:"type:int;default:0" json:"status"`
	Attempts   int            `gorm:"type:int;default:0" json:"attempts"`
	ClientInfo string         `gorm:"type:varchar(512)" json:"client_info"`
	UserAgent  string         `gorm:"type:varchar(512)" json:"user_agent"`
	IPAddress  string         `gorm:"type:varchar(45)" json:"ip_address"`
	ExpiredAt  time.Time      `gorm:"index" json:"expired_at"`
	VerifiedAt *time.Time     `json:"verified_at,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (c *Captcha) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.ExpiredAt.IsZero() {
		c.ExpiredAt = time.Now().Add(5 * time.Minute)
	}
	return nil
}

type CaptchaStatus int

const (
	CaptchaStatusPending  CaptchaStatus = 0
	CaptchaStatusVerified CaptchaStatus = 1
	CaptchaStatusExpired  CaptchaStatus = 2
	CaptchaStatusFailed   CaptchaStatus = 3
)

type CaptchaType string

const (
	CaptchaTypeImage  CaptchaType = "image"
	CaptchaTypeSlider CaptchaType = "slider"
	CaptchaTypeRotate CaptchaType = "rotate"
)

type VerifyRequest struct {
	CaptchaID string `json:"captcha_id" binding:"required"`
	Answer    string `json:"answer" binding:"required"`
}

type VerifyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
