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
	Phone        string         `gorm:"type:varchar(20)" json:"phone"`
	Avatar       string         `gorm:"type:varchar(255)" json:"avatar"`
	Role         string         `gorm:"type:varchar(20);default:admin" json:"role"`
	Status       int            `gorm:"type:int;default:1" json:"status"`
	Department   string         `gorm:"type:varchar(100)" json:"department"`
	Notes        string         `gorm:"type:text" json:"notes"`
	LastLoginAt  *time.Time     `json:"last_login_at,omitempty"`
	LastLoginIP  string         `gorm:"type:varchar(45)" json:"last_login_ip"`
	LoginCount   int            `gorm:"default:0" json:"login_count"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Roles        []Role         `gorm:"many2many:admin_roles" json:"roles,omitempty"`
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
	AdminRoleSuper AdminRole = "super_admin"
	AdminRoleAdmin AdminRole = "admin"
	AdminRoleUser  AdminRole = "operator"
	AdminRoleViewer AdminRole = "viewer"
)

type Permission struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Category    string    `gorm:"type:varchar(50);default:general" json:"category"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Role struct {
	ID          uint         `gorm:"primaryKey" json:"id"`
	Code        string       `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	Name        string       `gorm:"type:varchar(100);not null" json:"name"`
	Description string       `gorm:"type:text" json:"description"`
	IsSystem    bool         `gorm:"default:false" json:"is_system"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Permissions []Permission `gorm:"many2many:role_permissions" json:"permissions,omitempty"`
}

type RolePermission struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	RoleID       uint      `gorm:"not null" json:"role_id"`
	PermissionID uint      `gorm:"not null" json:"permission_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type AdminRoleLink struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	AdminID   uint      `gorm:"not null" json:"admin_id"`
	RoleID    uint      `gorm:"not null" json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

type AdminOperationLog struct {
	ID           uint         `gorm:"primaryKey" json:"id"`
	AdminID      uint         `gorm:"index" json:"admin_id"`
	Username     string       `gorm:"type:varchar(50);not null" json:"username"`
	Action       string       `gorm:"type:varchar(50);not null" json:"action"`
	ResourceType string       `gorm:"type:varchar(50);not null" json:"resource_type"`
	ResourceID   string       `gorm:"type:varchar(100)" json:"resource_id"`
	Details      string       `gorm:"type:jsonb" json:"details"`
	IP           string       `gorm:"type:varchar(45)" json:"ip"`
	UserAgent    string       `gorm:"type:text" json:"user_agent"`
	CreatedAt    time.Time    `json:"created_at"`
}

type AdminSession struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	AdminID      uint      `gorm:"not null;index" json:"admin_id"`
	SessionToken string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"session_token"`
	IPAddress    string    `gorm:"type:varchar(45)" json:"ip_address"`
	UserAgent    string    `gorm:"type:text" json:"user_agent"`
	ExpiresAt    time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

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

type CreateAdminRequest struct {
	Username   string   `json:"username" binding:"required,min=3,max=50"`
	Password  string   `json:"password" binding:"required,min=6"`
	Email     string   `json:"email" binding:"omitempty,email"`
	Nickname  string   `json:"nickname"`
	Phone     string   `json:"phone"`
	Role      string   `json:"role" binding:"required"`
	Department string  `json:"department"`
	Notes     string   `json:"notes"`
}

type UpdateAdminRequest struct {
	Email      string `json:"email" binding:"omitempty,email"`
	Nickname   string `json:"nickname"`
	Phone      string `json:"phone"`
	Status     *int   `json:"status"`
	Department string `json:"department"`
	Notes      string `json:"notes"`
}

type UpdateAdminRoleRequest struct {
	RoleID uint `json:"role_id" binding:"required"`
}

type AssignRolesRequest struct {
	RoleIDs []uint `json:"role_ids" binding:"required"`
}

type CreateRoleRequest struct {
	Code        string   `json:"code" binding:"required,min=3,max=50"`
	Name        string   `json:"name" binding:"required,min=2,max=100"`
	Description string   `json:"description"`
	PermissionIDs []uint `json:"permission_ids"`
}

type UpdateRoleRequest struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	PermissionIDs []uint   `json:"permission_ids"`
}

type AdminDTO struct {
	ID         uint     `json:"id"`
	Username   string   `json:"username"`
	Email      string   `json:"email"`
	Nickname   string   `json:"nickname"`
	Phone      string   `json:"phone"`
	Avatar     string   `json:"avatar"`
	Role       string   `json:"role"`
	Status     int      `json:"status"`
	Department string   `json:"department"`
	LastLogin  *string  `json:"last_login_at,omitempty"`
	LoginCount int      `json:"login_count"`
	CreatedAt  string   `json:"created_at"`
	Roles      []RoleDTO `json:"roles,omitempty"`
}

type RoleDTO struct {
	ID          uint            `json:"id"`
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	IsSystem    bool            `json:"is_system"`
	Permissions []PermissionDTO `json:"permissions,omitempty"`
	CreatedAt   string          `json:"created_at"`
}

type PermissionDTO struct {
	ID          uint   `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

func (a *Admin) ToDTO() *AdminDTO {
	dto := &AdminDTO{
		ID:         a.ID,
		Username:   a.Username,
		Email:      a.Email,
		Nickname:   a.Nickname,
		Phone:      a.Phone,
		Avatar:     a.Avatar,
		Role:       a.Role,
		Status:     a.Status,
		Department: a.Department,
		LoginCount: a.LoginCount,
		CreatedAt:  a.CreatedAt.Format(time.RFC3339),
	}
	if a.LastLoginAt != nil {
		lastLogin := a.LastLoginAt.Format(time.RFC3339)
		dto.LastLogin = &lastLogin
	}
	if len(a.Roles) > 0 {
		dto.Roles = make([]RoleDTO, 0, len(a.Roles))
		for _, role := range a.Roles {
			dto.Roles = append(dto.Roles, *role.ToDTO())
		}
	}
	return dto
}

func (r *Role) ToDTO() *RoleDTO {
	dto := &RoleDTO{
		ID:          r.ID,
		Code:        r.Code,
		Name:        r.Name,
		Description: r.Description,
		IsSystem:    r.IsSystem,
		CreatedAt:   r.CreatedAt.Format(time.RFC3339),
	}
	if len(r.Permissions) > 0 {
		dto.Permissions = make([]PermissionDTO, 0, len(r.Permissions))
		for _, perm := range r.Permissions {
			permDTO := perm.ToDTO()
			dto.Permissions = append(dto.Permissions, PermissionDTO{
				ID:          permDTO.ID,
				Code:        permDTO.Code,
				Name:        permDTO.Name,
				Description: permDTO.Description,
				Category:    permDTO.Category,
			})
		}
	}
	return dto
}

func (p *Permission) ToDTO() *PermissionDTO {
	return &PermissionDTO{
		ID:          p.ID,
		Code:        p.Code,
		Name:        p.Name,
		Description: p.Description,
		Category:    p.Category,
	}
}

type AdminFilter struct {
	Username string
	Email    string
	Role     string
	Status   *int
	Page     int
	PageSize int
}

type RoleFilter struct {
	Code      string
	IsSystem  *bool
	Page      int
	PageSize  int
}
