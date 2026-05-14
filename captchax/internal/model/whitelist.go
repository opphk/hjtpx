package model

import (
	"database/sql"
	"time"
)

type Whitelist struct {
	ID        int64          `json:"id"`
	IP        string         `json:"ip"`
	Domain    sql.NullString `json:"-"`
	Reason    sql.NullString `json:"-"`
	CreatedAt time.Time      `json:"created_at"`
}

type WhitelistDTO struct {
	ID        int64  `json:"id"`
	IP        string `json:"ip"`
	Domain    string `json:"domain,omitempty"`
	Reason    string `json:"reason,omitempty"`
	CreatedAt string `json:"created_at"`
}

func (w *Whitelist) ToDTO() *WhitelistDTO {
	dto := &WhitelistDTO{
		ID:        w.ID,
		IP:        w.IP,
		CreatedAt: w.CreatedAt.Format(time.RFC3339),
	}
	if w.Domain.Valid {
		dto.Domain = w.Domain.String
	}
	if w.Reason.Valid {
		dto.Reason = w.Reason.String
	}
	return dto
}

type CreateWhitelistRequest struct {
	IP     string `json:"ip" binding:"required,max=45"`
	Domain string `json:"domain" binding:"max=255"`
	Reason string `json:"reason"`
}

type UpdateWhitelistRequest struct {
	Domain string `json:"domain" binding:"max=255"`
	Reason string `json:"reason"`
}

type WhitelistFilter struct {
	IP     string
	Domain string
	Page   int
	PageSize int
}

func (f *WhitelistFilter) Offset() int {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	return (f.Page - 1) * f.PageSize
}

func (f *WhitelistFilter) Limit() int {
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
	return f.PageSize
}
