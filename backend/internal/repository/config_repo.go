package repository

import (
	"context"
	"errors"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

var (
	ErrConfigNotFound      = errors.New("config not found")
	ErrConfigAlreadyExists = errors.New("config already exists")
)

type ConfigRepo struct {
	db *gorm.DB
}

func NewConfigRepo(db *gorm.DB) *ConfigRepo {
	return &ConfigRepo{db: db}
}

func (r *ConfigRepo) GetAll() ([]models.Config, error) {
	var configs []models.Config
	err := r.db.Order("sort_order ASC, key ASC").Find(&configs).Error
	return configs, err
}

func (r *ConfigRepo) GetByKey(ctx context.Context, key string) (*models.Config, error) {
	var config models.Config
	err := r.db.WithContext(ctx).Where("`key` = ?", key).First(&config).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrConfigNotFound
	}
	return &config, err
}

func (r *ConfigRepo) GetByGroup(ctx context.Context, group string) ([]models.Config, error) {
	var configs []models.Config
	err := r.db.WithContext(ctx).Where("`group` = ? AND is_visible = ?", group, true).
		Order("sort_order ASC, key ASC").Find(&configs).Error
	return configs, err
}

func (r *ConfigRepo) Create(ctx context.Context, config *models.Config) error {
	var existing models.Config
	result := r.db.WithContext(ctx).Where("`key` = ?", config.Key).First(&existing)
	if result.Error == nil {
		return ErrConfigAlreadyExists
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	return r.db.WithContext(ctx).Create(config).Error
}

func (r *ConfigRepo) Update(ctx context.Context, key string, value string) error {
	result := r.db.WithContext(ctx).Model(&models.Config{}).
		Where("`key` = ?", key).
		Updates(map[string]interface{}{
			"value":      value,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrConfigNotFound
	}
	return nil
}

func (r *ConfigRepo) Upsert(key string, value string) error {
	var config models.Config
	result := r.db.Where("`key` = ?", key).First(&config)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		group := "default"
		if len(key) > 0 {
			parts := splitConfigKey(key)
			if len(parts) > 1 {
				group = parts[0]
			}
		}
		config = models.Config{
			Key:       key,
			Value:     value,
			Group:     group,
			IsVisible: true,
		}
		return r.db.Create(&config).Error
	}
	if result.Error != nil {
		return result.Error
	}
	config.Value = value
	return r.db.Save(&config).Error
}

func (r *ConfigRepo) Delete(ctx context.Context, key string) error {
	result := r.db.WithContext(ctx).Where("`key` = ?", key).Delete(&models.Config{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrConfigNotFound
	}
	return nil
}

func (r *ConfigRepo) BatchUpdate(ctx context.Context, configs map[string]string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for key, value := range configs {
			result := tx.Model(&models.Config{}).
				Where("`key` = ?", key).
				Updates(map[string]interface{}{
					"value":      value,
					"updated_at": time.Now(),
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				group := "default"
				if len(key) > 0 {
					parts := splitConfigKey(key)
					if len(parts) > 1 {
						group = parts[0]
					}
				}
				config := models.Config{
					Key:       key,
					Value:     value,
					Group:     group,
					IsVisible: true,
				}
				if err := tx.Create(&config).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func splitConfigKey(key string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(key); i++ {
		if key[i] == '.' {
			parts = append(parts, key[start:i])
			start = i + 1
		}
	}
	parts = append(parts, key[start:])
	return parts
}
