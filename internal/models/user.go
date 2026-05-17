package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Email     string         `gorm:"uniqueIndex;not null;size:255" json:"email"`
	Password  string         `gorm:"not null;size:255" json:"-"`
	Username  string         `gorm:"size:100" json:"username"`
	AppID     uint           `gorm:"index" json:"app_id"`
	Status    int            `gorm:"default:1" json:"status"`
	LastLogin time.Time      `json:"last_login"`
}

func (User) TableName() string {
	return "users"
}

type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Username string `json:"username"`
	AppID    uint   `json:"app_id"`
}

type UpdateUserRequest struct {
	Email    string `json:"email" binding:"omitempty,email"`
	Username string `json:"username"`
	Status   int    `json:"status"`
}
