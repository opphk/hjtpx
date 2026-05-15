package model

import "time"

type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"default:0" json:"user_id"`
	Username  string    `gorm:"type:varchar(255);default:''" json:"username"`
	Action    string    `gorm:"type:varchar(100);not null" json:"action"`
	Detail    string    `gorm:"type:text" json:"detail"`
	IPAddress string    `gorm:"type:varchar(45);default:''" json:"ip_address"`
	UserAgent string    `gorm:"type:text;default:''" json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

func (a *AuditLog) ToDTO() *AuditLogDTO {
	return &AuditLogDTO{
		ID:        a.ID,
		UserID:    a.UserID,
		Username:  a.Username,
		Action:    a.Action,
		Detail:    a.Detail,
		IPAddress: a.IPAddress,
		UserAgent: a.UserAgent,
		CreatedAt: a.CreatedAt.Format(time.RFC3339),
	}
}

type AuditLogDTO struct {
	ID        uint   `json:"id"`
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	Action    string `json:"action"`
	Detail    string `json:"detail"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
	CreatedAt string `json:"created_at"`
}

type AuditLogFilter struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	Action    string `json:"action"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
}