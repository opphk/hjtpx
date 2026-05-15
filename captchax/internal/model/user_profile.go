package model

import (
	"database/sql"
	"encoding/json"
	"time"
)

type UserProfile struct {
	ID                    int64           `json:"id"`
	Identifier           string          `json:"identifier"`
	IdentifierType       string          `json:"identifier_type"`
	IP                   string          `json:"ip"`
	DeviceFingerprint    string          `json:"device_fingerprint"`
	CookieID             string          `json:"cookie_id"`
	SessionID            string          `json:"session_id"`
	TotalAttempts        int64           `json:"total_attempts"`
	SuccessCount         int64           `json:"success_count"`
	FailCount            int64           `json:"fail_count"`
	SuccessRate          float64         `json:"success_rate"`
	AvgResponseTime      float64         `json:"avg_response_time"`
	MinResponseTime      float64         `json:"min_response_time"`
	MaxResponseTime      float64         `json:"max_response_time"`
	PreferredCaptchaType string          `json:"preferred_captcha_type"`
	CaptchaTypeDistribution sql.NullString `json:"-"`
	ActiveHours          sql.NullString  `json:"-"`
	ActiveDays           sql.NullString  `json:"-"`
	LocationDistribution sql.NullString  `json:"-"`
	DeviceDistribution   sql.NullString  `json:"-"`
	TotalRiskEvents      int64           `json:"total_risk_events"`
	HighRiskEvents       int64           `json:"high_risk_events"`
	LastRiskEventAt      sql.NullTime    `json:"last_risk_event_at,omitempty"`
	FirstSeenAt          time.Time       `json:"first_seen_at"`
	LastSeenAt           time.Time       `json:"last_seen_at"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

type UserProfileDTO struct {
	ID                    int64            `json:"id"`
	Identifier           string           `json:"identifier"`
	IdentifierType       string           `json:"identifier_type"`
	IP                   string           `json:"ip"`
	DeviceFingerprint    string           `json:"device_fingerprint,omitempty"`
	TotalAttempts        int64            `json:"total_attempts"`
	SuccessCount         int64            `json:"success_count"`
	FailCount            int64            `json:"fail_count"`
	SuccessRate          float64          `json:"success_rate"`
	AvgResponseTime      float64          `json:"avg_response_time"`
	PreferredCaptchaType string           `json:"preferred_captcha_type"`
	CaptchaDistribution   map[string]int64 `json:"captcha_distribution,omitempty"`
	ActiveHours          map[int]int64    `json:"active_hours,omitempty"`
	ActiveDays           map[int]int64    `json:"active_days,omitempty"`
	LocationDistribution map[string]int64 `json:"location_distribution,omitempty"`
	DeviceDistribution   map[string]int64 `json:"device_distribution,omitempty"`
	TotalRiskEvents      int64            `json:"total_risk_events"`
	HighRiskEvents       int64            `json:"high_risk_events"`
	LastRiskEventAt      *string         `json:"last_risk_event_at,omitempty"`
	FirstSeenAt          string           `json:"first_seen_at"`
	LastSeenAt           string           `json:"last_seen_at"`
	UpdatedAt            string           `json:"updated_at"`
}

func (u *UserProfile) ToDTO() *UserProfileDTO {
	dto := &UserProfileDTO{
		ID:                    u.ID,
		Identifier:           u.Identifier,
		IdentifierType:       u.IdentifierType,
		IP:                   u.IP,
		DeviceFingerprint:    u.DeviceFingerprint,
		TotalAttempts:        u.TotalAttempts,
		SuccessCount:         u.SuccessCount,
		FailCount:            u.FailCount,
		SuccessRate:          u.SuccessRate,
		AvgResponseTime:      u.AvgResponseTime,
		PreferredCaptchaType: u.PreferredCaptchaType,
		TotalRiskEvents:      u.TotalRiskEvents,
		HighRiskEvents:       u.HighRiskEvents,
		FirstSeenAt:          u.FirstSeenAt.Format(time.RFC3339),
		LastSeenAt:           u.LastSeenAt.Format(time.RFC3339),
		UpdatedAt:            u.UpdatedAt.Format(time.RFC3339),
	}

	if u.CaptchaTypeDistribution.Valid {
		var dist map[string]int64
		if err := json.Unmarshal([]byte(u.CaptchaTypeDistribution.String), &dist); err == nil {
			dto.CaptchaDistribution = dist
		}
	}

	if u.ActiveHours.Valid {
		var hours map[int]int64
		if err := json.Unmarshal([]byte(u.ActiveHours.String), &hours); err == nil {
			dto.ActiveHours = hours
		}
	}

	if u.ActiveDays.Valid {
		var days map[int]int64
		if err := json.Unmarshal([]byte(u.ActiveDays.String), &days); err == nil {
			dto.ActiveDays = days
		}
	}

	if u.LocationDistribution.Valid {
		var locations map[string]int64
		if err := json.Unmarshal([]byte(u.LocationDistribution.String), &locations); err == nil {
			dto.LocationDistribution = locations
		}
	}

	if u.DeviceDistribution.Valid {
		var devices map[string]int64
		if err := json.Unmarshal([]byte(u.DeviceDistribution.String), &devices); err == nil {
			dto.DeviceDistribution = devices
		}
	}

	if u.LastRiskEventAt.Valid {
		t := u.LastRiskEventAt.Time.Format(time.RFC3339)
		dto.LastRiskEventAt = &t
	}

	return dto
}

type CreateUserProfileRequest struct {
	Identifier        string  `json:"identifier" binding:"required"`
	IdentifierType    string  `json:"identifier_type" binding:"required,oneof=ip device cookie session"`
	IP                string  `json:"ip"`
	DeviceFingerprint string  `json:"device_fingerprint"`
	CookieID          string  `json:"cookie_id"`
	SessionID         string  `json:"session_id"`
}

type UpdateUserProfileRequest struct {
	IP                string  `json:"ip"`
	DeviceFingerprint string  `json:"device_fingerprint"`
	CookieID          string  `json:"cookie_id"`
	SessionID         string  `json:"session_id"`
}

type UserProfileFilter struct {
	Identifier     string  `form:"identifier"`
	IdentifierType string  `form:"identifier_type"`
	TrustLevel     string  `form:"trust_level"`
	DateFrom       string  `form:"date_from"`
	DateTo         string  `form:"date_to"`
	Page            int    `form:"page,default=1"`
	PageSize        int    `form:"page_size,default=20"`
}

type UserProfileAnalysis struct {
	ProfileID       int64           `json:"profile_id"`
	Identifier     string          `json:"identifier"`
	TrustLevel     string          `json:"trust_level"`
	ActivityLevel  int             `json:"activity_level"`
	FrequencyLevel int             `json:"frequency_level"`
	ComplexityLevel int             `json:"complexity_level"`
	RiskScore      int             `json:"risk_score"`
	Anomalies      []AnomalyInfo   `json:"anomalies,omitempty"`
	Recommendations []string       `json:"recommendations,omitempty"`
}

type AnomalyInfo struct {
	Type        string      `json:"type"`
	Severity    int         `json:"severity"`
	Description string      `json:"description"`
}

type RefreshProfileRequest struct {
	Identifier    string `json:"identifier" binding:"required"`
	IdentifierType string `json:"identifier_type" binding:"required,oneof=ip device cookie session"`
}

type RefreshProfileResponse struct {
	Profile *UserProfileDTO `json:"profile"`
	Message string          `json:"message"`
}
