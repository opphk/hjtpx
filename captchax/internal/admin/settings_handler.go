package admin

import (
	"context"
	"database/sql"
	"net/http"

	"captchax/internal/model"
	"captchax/pkg/response"

	"github.com/gin-gonic/gin"
)

type SettingsHandlers struct {
	db *sql.DB
}

func NewSettingsHandlers(db *sql.DB) *SettingsHandlers {
	return &SettingsHandlers{db: db}
}

func (h *SettingsHandlers) GetSettings(c *gin.Context) {
	ctx := c.Request.Context()

	query := `SELECT id, site_name, site_description, jwt_expiry_hours, min_password_length, captcha_difficulty, captcha_types, email_notification, webhook_url, updated_at FROM system_settings ORDER BY id LIMIT 1`

	var settings model.SystemSettings
	err := h.db.QueryRowContext(ctx, query).Scan(
		&settings.ID, &settings.SiteName, &settings.SiteDescription,
		&settings.JWTExpiryHours, &settings.MinPasswordLength,
		&settings.CaptchaDifficulty, &settings.CaptchaTypes,
		&settings.EmailNotification, &settings.WebhookURL, &settings.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		settings = model.SystemSettings{
			ID:                1,
			SiteName:          "CaptchaX",
			SiteDescription:   "CaptchaX 验证码管理系统",
			JWTExpiryHours:    24,
			MinPasswordLength: 8,
			CaptchaDifficulty: "medium",
			CaptchaTypes:      "slider,click,rotate",
			EmailNotification: false,
			WebhookURL:        "",
		}
		h.createDefaultSettings(ctx)
	} else if err != nil {
		response.InternalError(c, "failed to get settings")
		return
	}

	response.Success(c, settings.ToDTO())
}

func (h *SettingsHandlers) UpdateSettings(c *gin.Context) {
	var req model.UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	ctx := c.Request.Context()

	var current model.SystemSettings
	err := h.db.QueryRowContext(ctx, `SELECT id, site_name, site_description, jwt_expiry_hours, min_password_length, captcha_difficulty, captcha_types, email_notification, webhook_url, updated_at FROM system_settings ORDER BY id LIMIT 1`).Scan(
		&current.ID, &current.SiteName, &current.SiteDescription,
		&current.JWTExpiryHours, &current.MinPasswordLength,
		&current.CaptchaDifficulty, &current.CaptchaTypes,
		&current.EmailNotification, &current.WebhookURL, &current.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		h.createDefaultSettings(ctx)
		current = model.SystemSettings{ID: 1}
	} else if err != nil {
		response.InternalError(c, "failed to get settings")
		return
	}

	if req.SiteName != nil {
		current.SiteName = *req.SiteName
	}
	if req.SiteDescription != nil {
		current.SiteDescription = *req.SiteDescription
	}
	if req.JWTExpiryHours != nil {
		current.JWTExpiryHours = *req.JWTExpiryHours
	}
	if req.MinPasswordLength != nil {
		current.MinPasswordLength = *req.MinPasswordLength
	}
	if req.CaptchaDifficulty != nil {
		current.CaptchaDifficulty = *req.CaptchaDifficulty
	}
	if req.CaptchaTypes != nil {
		current.CaptchaTypes = *req.CaptchaTypes
	}
	if req.EmailNotification != nil {
		current.EmailNotification = *req.EmailNotification
	}
	if req.WebhookURL != nil {
		current.WebhookURL = *req.WebhookURL
	}

	query := `UPDATE system_settings SET site_name=$1, site_description=$2, jwt_expiry_hours=$3, min_password_length=$4, captcha_difficulty=$5, captcha_types=$6, email_notification=$7, webhook_url=$8, updated_at=NOW() WHERE id=$9`
	_, err = h.db.ExecContext(ctx, query,
		current.SiteName, current.SiteDescription, current.JWTExpiryHours,
		current.MinPasswordLength, current.CaptchaDifficulty, current.CaptchaTypes,
		current.EmailNotification, current.WebhookURL, current.ID,
	)
	if err != nil {
		response.InternalError(c, "failed to update settings")
		return
	}

	username, _ := c.Get("username")
	uname := ""
	if username != nil {
		uname = username.(string)
	}
	h.logAudit(ctx, 0, uname, "update_settings", "Updated system settings", c.ClientIP(), c.Request.UserAgent())

	response.Success(c, current.ToDTO())
}

func (h *SettingsHandlers) ShowSettingsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "settings.html", gin.H{
		"title": "CaptchaX System Settings",
	})
}

func (h *SettingsHandlers) createDefaultSettings(ctx context.Context) {
	query := `INSERT INTO system_settings (site_name, site_description, jwt_expiry_hours, min_password_length, captcha_difficulty, captcha_types, email_notification, webhook_url)
		VALUES ('CaptchaX', 'CaptchaX 验证码管理系统', 24, 8, 'medium', 'slider,click,rotate', false, '')
		ON CONFLICT DO NOTHING`
	_, _ = h.db.ExecContext(ctx, query)
}

func (h *SettingsHandlers) logAudit(ctx context.Context, userID uint, username, action, detail, ip, userAgent string) {
	if username == "" {
		username = "system"
	}
	if ip == "" {
		ip = "127.0.0.1"
	}
	query := `INSERT INTO audit_logs (user_id, username, action, detail, ip_address, user_agent, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW())`
	_, _ = h.db.ExecContext(ctx, query, userID, username, action, detail, ip, userAgent)
}