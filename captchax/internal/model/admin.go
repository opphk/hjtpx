package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Admin struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	PasswordHash string         `gorm:"type:varchar(128);not null" json:"-"`
	Email        string         `gorm:"type:varchar(100);uniqueIndex" json:"email"`
	Nickname     string         `gorm:"type:varchar(100)" json:"nickname"`
	Role         string         `gorm:"type:varchar(20);default:admin" json:"role"`
	Status       int            `gorm:"type:int;default:1" json:"status"`
	LastLoginAt  *time.Time     `json:"last_login_at,omitempty"`
	LastLoginIP  string         `gorm:"type:varchar(45)" json:"last_login_ip"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (a *Admin) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	a.PasswordHash = string(hash)
	return nil
}

func (a *Admin) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password))
	return err == nil
}

type AdminRole string

const (
	AdminRoleSuper AdminRole = "super"
	AdminRoleAdmin AdminRole = "admin"
	AdminRoleUser  AdminRole = "user"
)

type App struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	AppID       string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"app_id"`
	AppSecret   string         `gorm:"type:varchar(128);not null" json:"-"`
	Name        string         `gorm:"type:varchar(100);not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	OwnerID     uint           `gorm:"not null" json:"owner_id"`
	Owner       *Admin         `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Status      int            `gorm:"type:int;default:1" json:"status"`
	Domain      string         `gorm:"type:varchar(255)" json:"domain"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (a *App) SetSecret(secret string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	a.AppSecret = string(hash)
	return nil
}

func (a *App) CheckSecret(secret string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(a.AppSecret), []byte(secret))
	return err == nil
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	Admin Admin  `json:"admin"`
}

type CreateAppRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Domain      string `json:"domain"`
}

type UpdateAppRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Domain      string `json:"domain"`
	Status      int    `json:"status"`
}
