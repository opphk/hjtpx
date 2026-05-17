package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"hjtpx/internal/database"
	"hjtpx/internal/models"

	"gorm.io/gorm"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

type UserRepository struct {
	db    *gorm.DB
	cache *database.RedisCache
}

func NewUserRepository(db *gorm.DB, cache *database.RedisCache) *UserRepository {
	return &UserRepository{
		db:    db,
		cache: cache,
	}
}

func (r *UserRepository) Create(user *models.User) error {
	if err := r.db.Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrUserExists
		}
		return err
	}
	return nil
}

func (r *UserRepository) GetByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(user *models.User) error {
	result := r.db.Save(user)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) Delete(id uint) error {
	result := r.db.Delete(&models.User{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	ctx := context.Background()
	r.cache.Delete(ctx, "user:"+string(rune(id)))

	return nil
}

func (r *UserRepository) List(page, pageSize int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	if err := r.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := r.db.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *UserRepository) UpdateLastLogin(id uint) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Update("last_login", time.Now()).Error
}

func (r *UserRepository) GetByIDWithCache(id uint) (*models.User, error) {
	ctx := context.Background()
	cacheKey := "user:" + string(rune(id))

	exists, _ := r.cache.Exists(ctx, cacheKey)
	if exists {
		var user models.User
		if err := r.cache.Get(ctx, cacheKey, &user); err == nil {
			return &user, nil
		}
	}

	user, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	r.cache.Set(ctx, cacheKey, user, 30*time.Minute)

	return user, nil
}

func (r *UserRepository) UpdateWithCache(user *models.User) error {
	if err := r.Update(user); err != nil {
		return err
	}

	ctx := context.Background()
	cacheKey := "user:" + string(rune(user.ID))
	userJSON, _ := json.Marshal(user)
	r.cache.Set(ctx, cacheKey, userJSON, 30*time.Minute)

	return nil
}

func (r *UserRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&models.User{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *UserRepository) FindAll(page, pageSize int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	if err := r.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := r.db.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *UserRepository) FindByID(id uint) (*models.User, error) {
	return r.GetByID(id)
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	return r.GetByEmail(email)
}
