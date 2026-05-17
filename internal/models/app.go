package models

import (
	"time"

	"gorm.io/gorm"
)

type App struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"not null;size:100" json:"name"`
	AppKey    string         `gorm:"uniqueIndex;not null;size:64" json:"app_key"`
	AppSecret string         `gorm:"not null;size:255" json:"app_secret"`
	Status    int            `gorm:"default:1" json:"status"`
	Domain    string         `gorm:"size:255" json:"domain"`
	OwnerID   uint           `gorm:"index" json:"owner_id"`
}

func (App) TableName() string {
	return "apps"
}

type CreateAppRequest struct {
	Name      string `json:"name" binding:"required"`
	AppKey    string `json:"app_key" binding:"required"`
	AppSecret string `json:"app_secret" binding:"required"`
	Domain    string `json:"domain"`
	OwnerID   uint   `json:"owner_id"`
}

type UpdateAppRequest struct {
	Name      string `json:"name"`
	AppSecret string `json:"app_secret"`
	Status    int    `json:"status"`
	Domain    string `json:"domain"`
}
