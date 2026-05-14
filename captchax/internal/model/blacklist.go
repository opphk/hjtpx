package model

import (
	"database/sql"
	"time"
)

type Blacklist struct {
	ID        int64          `json:"id"`
	IP        string         `json:"ip"`
	Reason    sql.NullString `json:"-"`
	ExpireAt  sql.NullTime   `json:"-"`
	CreatedAt time.Time      `json:"created_at"`
}

type BlacklistDTO struct {
	ID        int64  `json:"id"`
	IP        string `json:"ip"`
	Reason    string `json:"reason,omitempty"`
	ExpireAt  string `json:"expire_at,omitempty"`
	CreatedAt string `json:"created_at"`
	IsActive  bool   `json:"is_active"`
}

func (b *Blacklist) ToDTO() *BlacklistDTO {
	dto := &BlacklistDTO{
		ID:        b.ID,
		IP:        b.IP,
		CreatedAt: b.CreatedAt.Format(time.RFC3339),
		IsActive:  true,
	}
	if b.Reason.Valid {
		dto.Reason = b.Reason.String
	}
	if b.ExpireAt.Valid {
		dto.ExpireAt = b.ExpireAt.Time.Format(time.RFC3339)
		if b.ExpireAt.Time.Before(time.Now()) {
			dto.IsActive = false
		}
	}
	return dto
}

type CreateBlacklistRequest struct {
	IP       string     `json:"ip" binding:"required,max=45"`
	Reason   string     `json:"reason"`
	ExpireAt *time.Time `json:"expire_at"`
}

type UpdateBlacklistRequest struct {
	Reason   string     `json:"reason"`
	ExpireAt *time.Time `json:"expire_at"`
}

type BlacklistFilter struct {
	IP         string
	ActiveOnly bool
	Page       int
	PageSize   int
}

func (f *BlacklistFilter) Offset() int {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	return (f.Page - 1) * f.PageSize
}

func (f *BlacklistFilter) Limit() int {
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
	return f.PageSize
}
