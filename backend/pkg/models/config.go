package models

import (
	"time"
)

type Config struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"size:255;not null;uniqueIndex:idx_config_key" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	Group     string    `gorm:"size:100;index:idx_config_group" json:"group"`
	Name      string    `gorm:"size:255" json:"name"`
	Desc      string    `gorm:"size:500" json:"desc"`
	Type      string    `gorm:"size:50;default:string" json:"type"`
	Options   string    `gorm:"type:text" json:"options"`
	IsVisible bool      `gorm:"default:true" json:"is_visible"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Config) TableName() string {
	return "configs"
}
