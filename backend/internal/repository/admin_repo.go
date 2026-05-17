package repository

import (
	"context"
	"errors"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

var (
	ErrAdminNotFound      = errors.New("admin user not found")
	ErrAdminAlreadyExists = errors.New("admin user already exists")
)

type AdminRepository interface {
	Create(ctx context.Context, admin *models.Admin) error
	GetByID(ctx context.Context, id uint) (*models.Admin, error)
	GetByUsername(ctx context.Context, username string) (*models.Admin, error)
	GetByEmail(ctx context.Context, email string) (*models.Admin, error)
	Update(ctx context.Context, admin *models.Admin) error
	UpdateLastLogin(ctx context.Context, adminID uint, loginTime time.Time, ip string) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, page, pageSize int) ([]*models.Admin, int64, error)
	UpdateStatus(ctx context.Context, adminID uint, status string) error
	IncrementLoginCount(ctx context.Context, adminID uint) error
	CreateLoginLog(ctx context.Context, log *models.AdminLoginLog) error
	GetLoginLogs(ctx context.Context, adminID uint, page, pageSize int) ([]*models.AdminLoginLog, int64, error)
}

type adminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) Create(ctx context.Context, admin *models.Admin) error {
	var existing models.Admin
	result := r.db.WithContext(ctx).Where("username = ? OR email = ?", admin.Username, admin.Email).First(&existing)
	if result.Error == nil {
		return ErrAdminAlreadyExists
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	return r.db.WithContext(ctx).Create(admin).Error
}

func (r *adminRepository) GetByID(ctx context.Context, id uint) (*models.Admin, error) {
	var admin models.Admin
	result := r.db.WithContext(ctx).First(&admin, id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrAdminNotFound
	}
	return &admin, result.Error
}

func (r *adminRepository) GetByUsername(ctx context.Context, username string) (*models.Admin, error) {
	var admin models.Admin
	result := r.db.WithContext(ctx).Where("username = ?", username).First(&admin)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrAdminNotFound
	}
	return &admin, result.Error
}

func (r *adminRepository) GetByEmail(ctx context.Context, email string) (*models.Admin, error) {
	var admin models.Admin
	result := r.db.WithContext(ctx).Where("email = ?", email).First(&admin)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrAdminNotFound
	}
	return &admin, result.Error
}

func (r *adminRepository) Update(ctx context.Context, admin *models.Admin) error {
	result := r.db.WithContext(ctx).Save(admin)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAdminNotFound
	}
	return nil
}

func (r *adminRepository) UpdateLastLogin(ctx context.Context, adminID uint, loginTime time.Time, ip string) error {
	result := r.db.WithContext(ctx).Model(&models.Admin{}).
		Where("id = ?", adminID).
		Updates(map[string]interface{}{
			"last_login_at": loginTime,
			"last_login_ip": ip,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAdminNotFound
	}
	return nil
}

func (r *adminRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.Admin{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAdminNotFound
	}
	return nil
}

func (r *adminRepository) List(ctx context.Context, page, pageSize int) ([]*models.Admin, int64, error) {
	var admins []*models.Admin
	var total int64

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	err := r.db.WithContext(ctx).Model(&models.Admin{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = r.db.WithContext(ctx).
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&admins).Error

	return admins, total, err
}

func (r *adminRepository) UpdateStatus(ctx context.Context, adminID uint, status string) error {
	result := r.db.WithContext(ctx).Model(&models.Admin{}).
		Where("id = ?", adminID).
		Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAdminNotFound
	}
	return nil
}

func (r *adminRepository) IncrementLoginCount(ctx context.Context, adminID uint) error {
	result := r.db.WithContext(ctx).Model(&models.Admin{}).
		Where("id = ?", adminID).
		UpdateColumn("login_count", gorm.Expr("login_count + ?", 1))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAdminNotFound
	}
	return nil
}

func (r *adminRepository) CreateLoginLog(ctx context.Context, log *models.AdminLoginLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *adminRepository) GetLoginLogs(ctx context.Context, adminID uint, page, pageSize int) ([]*models.AdminLoginLog, int64, error) {
	var logs []*models.AdminLoginLog
	var total int64

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	query := r.db.WithContext(ctx).Model(&models.AdminLoginLog{}).Where("admin_id = ?", adminID)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, total, err
}
