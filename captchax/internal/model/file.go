package model

import (
	"time"

	"gorm.io/gorm"
)

type File struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       uint           `gorm:"index;not null" json:"user_id"`
	Filename     string         `gorm:"type:varchar(255);not null" json:"filename"`
	OriginalName string         `gorm:"type:varchar(255);not null" json:"original_name"`
	MimeType     string         `gorm:"type:varchar(127);not null" json:"mime_type"`
	Size         int64          `gorm:"not null" json:"size"`
	Path         string         `gorm:"type:varchar(512);not null" json:"path"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (f *File) TableName() string {
	return "files"
}