package repository

import (
	"context"
	"errors"
	"time"

	"hjtpx/internal/database"
	"hjtpx/internal/models"

	"gorm.io/gorm"
)

var (
	ErrAppNotFound = errors.New("app not found")
	ErrAppExists   = errors.New("app already exists")
)

type AppRepository struct {
	db    *gorm.DB
	cache *database.RedisCache
}

func NewAppRepository(db *gorm.DB, cache *database.RedisCache) *AppRepository {
	return &AppRepository{
		db:    db,
		cache: cache,
	}
}

func (r *AppRepository) Create(app *models.App) error {
	if err := r.db.Create(app).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrAppExists
		}
		return err
	}
	return nil
}

func (r *AppRepository) GetByID(id uint) (*models.App, error) {
	var app models.App
	if err := r.db.First(&app, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAppNotFound
		}
		return nil, err
	}
	return &app, nil
}

func (r *AppRepository) GetByAppKey(appKey string) (*models.App, error) {
	var app models.App
	if err := r.db.Where("app_key = ?", appKey).First(&app).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAppNotFound
		}
		return nil, err
	}
	return &app, nil
}

func (r *AppRepository) Update(app *models.App) error {
	result := r.db.Save(app)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAppNotFound
	}
	return nil
}

func (r *AppRepository) Delete(id uint) error {
	result := r.db.Delete(&models.App{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAppNotFound
	}

	ctx := context.Background()
	r.cache.Delete(ctx, "app:"+string(rune(id)))

	return nil
}

func (r *AppRepository) List(page, pageSize int) ([]models.App, int64, error) {
	var apps []models.App
	var total int64

	if err := r.db.Model(&models.App{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := r.db.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&apps).Error; err != nil {
		return nil, 0, err
	}

	return apps, total, nil
}

func (r *AppRepository) FindAll(page, pageSize int) ([]models.App, int64, error) {
	return r.List(page, pageSize)
}

func (r *AppRepository) FindByID(id uint) (*models.App, error) {
	return r.GetByID(id)
}

func (r *AppRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&models.App{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *AppRepository) GetByIDWithCache(id uint) (*models.App, error) {
	ctx := context.Background()
	cacheKey := "app:" + string(rune(id))

	exists, _ := r.cache.Exists(ctx, cacheKey)
	if exists {
		var app models.App
		if err := r.cache.Get(ctx, cacheKey, &app); err == nil {
			return &app, nil
		}
	}

	app, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	r.cache.Set(ctx, cacheKey, app, 30*time.Minute)

	return app, nil
}

func (r *AppRepository) ListByOwnerID(ownerID uint, page, pageSize int) ([]models.App, int64, error) {
	var apps []models.App
	var total int64

	query := r.db.Model(&models.App{}).Where("owner_id = ?", ownerID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&apps).Error; err != nil {
		return nil, 0, err
	}

	return apps, total, nil
}
