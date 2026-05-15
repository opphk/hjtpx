package model

import "time"

type SystemSettings struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	SiteName          string    `gorm:"type:varchar(255);default:CaptchaX" json:"site_name"`
	SiteDescription   string    `gorm:"type:text" json:"site_description"`
	JWTExpiryHours    int       `gorm:"default:24" json:"jwt_expiry_hours"`
	MinPasswordLength int       `gorm:"default:8" json:"min_password_length"`
	CaptchaDifficulty string    `gorm:"type:varchar(20);default:medium" json:"captcha_difficulty"`
	CaptchaTypes      string    `gorm:"type:text" json:"captcha_types"`
	EmailNotification bool      `gorm:"default:false" json:"email_notification"`
	WebhookURL        string    `gorm:"type:varchar(500)" json:"webhook_url"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (s *SystemSettings) ToDTO() *SystemSettingsDTO {
	return &SystemSettingsDTO{
		ID:                s.ID,
		SiteName:          s.SiteName,
		SiteDescription:   s.SiteDescription,
		JWTExpiryHours:    s.JWTExpiryHours,
		MinPasswordLength: s.MinPasswordLength,
		CaptchaDifficulty: s.CaptchaDifficulty,
		CaptchaTypes:      s.CaptchaTypes,
		EmailNotification: s.EmailNotification,
		WebhookURL:        s.WebhookURL,
		UpdatedAt:         s.UpdatedAt.Format(time.RFC3339),
	}
}

type SystemSettingsDTO struct {
	ID                uint   `json:"id"`
	SiteName          string `json:"site_name"`
	SiteDescription   string `json:"site_description"`
	JWTExpiryHours    int    `json:"jwt_expiry_hours"`
	MinPasswordLength int    `json:"min_password_length"`
	CaptchaDifficulty string `json:"captcha_difficulty"`
	CaptchaTypes      string `json:"captcha_types"`
	EmailNotification bool   `json:"email_notification"`
	WebhookURL        string `json:"webhook_url"`
	UpdatedAt         string `json:"updated_at"`
}

type UpdateSettingsRequest struct {
	SiteName          *string `json:"site_name"`
	SiteDescription   *string `json:"site_description"`
	JWTExpiryHours    *int    `json:"jwt_expiry_hours"`
	MinPasswordLength *int    `json:"min_password_length"`
	CaptchaDifficulty *string `json:"captcha_difficulty"`
	CaptchaTypes      *string `json:"captcha_types"`
	EmailNotification *bool   `json:"email_notification"`
	WebhookURL        *string `json:"webhook_url"`
}