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
	ErrCaptchaNotFound = errors.New("captcha not found")
	ErrCaptchaExpired  = errors.New("captcha expired")
	ErrCaptchaInvalid = errors.New("invalid captcha")
)

type CaptchaRepository struct {
	db    *gorm.DB
	cache *database.RedisCache
}

func NewCaptchaRepository(db *gorm.DB, cache *database.RedisCache) *CaptchaRepository {
	return &CaptchaRepository{
		db:    db,
		cache: cache,
	}
}

func (r *CaptchaRepository) Create(captcha *models.Captcha) error {
	if err := r.db.Create(captcha).Error; err != nil {
		return err
	}

	ctx := context.Background()
	cacheKey := "captcha:" + captcha.Token
	r.cache.Set(ctx, cacheKey, captcha, time.Until(captcha.ExpiresAt))

	return nil
}

func (r *CaptchaRepository) GetByToken(token string) (*models.Captcha, error) {
	ctx := context.Background()
	cacheKey := "captcha:" + token

	exists, _ := r.cache.Exists(ctx, cacheKey)
	if exists {
		var captcha models.Captcha
		if err := r.cache.Get(ctx, cacheKey, &captcha); err == nil {
			return &captcha, nil
		}
	}

	var captcha models.Captcha
	if err := r.db.Where("token = ?", token).First(&captcha).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCaptchaNotFound
		}
		return nil, err
	}

	if captcha.IsExpired() {
		return &captcha, ErrCaptchaExpired
	}

	r.cache.Set(ctx, cacheKey, &captcha, time.Until(captcha.ExpiresAt))

	return &captcha, nil
}

func (r *CaptchaRepository) GetByID(id uint) (*models.Captcha, error) {
	var captcha models.Captcha
	if err := r.db.First(&captcha, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCaptchaNotFound
		}
		return nil, err
	}
	return &captcha, nil
}

func (r *CaptchaRepository) Update(captcha *models.Captcha) error {
	result := r.db.Save(captcha)
	if result.Error != nil {
		return result.Error
	}

	ctx := context.Background()
	cacheKey := "captcha:" + captcha.Token
	if captcha.IsExpired() {
		r.cache.Delete(ctx, cacheKey)
	} else {
		r.cache.Set(ctx, cacheKey, captcha, time.Until(captcha.ExpiresAt))
	}

	return nil
}

func (r *CaptchaRepository) Delete(token string) error {
	result := r.db.Where("token = ?", token).Delete(&models.Captcha{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCaptchaNotFound
	}

	ctx := context.Background()
	cacheKey := "captcha:" + token
	r.cache.Delete(ctx, cacheKey)

	return nil
}

func (r *CaptchaRepository) Verify(token, challenge string) (*models.Captcha, error) {
	captcha, err := r.GetByToken(token)
	if err != nil {
		return nil, err
	}

	if captcha.IsExpired() {
		captcha.Status = models.CaptchaStatusExpired
		r.Update(captcha)
		return captcha, ErrCaptchaExpired
	}

	if captcha.Challenge != challenge {
		captcha.VerifyCount++
		r.Update(captcha)
		return captcha, ErrCaptchaInvalid
	}

	if captcha.Status != models.CaptchaStatusPending {
		return captcha, ErrCaptchaInvalid
	}

	captcha.Status = models.CaptchaStatusVerified
	if err := r.Update(captcha); err != nil {
		return nil, err
	}

	return captcha, nil
}

func (r *CaptchaRepository) IncrementVerifyCount(token string) error {
	captcha, err := r.GetByToken(token)
	if err != nil {
		return err
	}

	captcha.VerifyCount++
	return r.Update(captcha)
}

func (r *CaptchaRepository) MarkAsExpired(token string) error {
	captcha, err := r.GetByToken(token)
	if err != nil {
		return err
	}

	captcha.Status = models.CaptchaStatusExpired
	return r.Update(captcha)
}

func (r *CaptchaRepository) ListByUserID(userID uint, page, pageSize int) ([]models.Captcha, int64, error) {
	var captchas []models.Captcha
	var total int64

	query := r.db.Model(&models.Captcha{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&captchas).Error; err != nil {
		return nil, 0, err
	}

	return captchas, total, nil
}

func (r *CaptchaRepository) ListByAppID(appID uint, page, pageSize int) ([]models.Captcha, int64, error) {
	var captchas []models.Captcha
	var total int64

	query := r.db.Model(&models.Captcha{}).Where("app_id = ?", appID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&captchas).Error; err != nil {
		return nil, 0, err
	}

	return captchas, total, nil
}

func (r *CaptchaRepository) CleanExpired() (int64, error) {
	result := r.db.Where("expires_at < ? AND status = ?", time.Now(), models.CaptchaStatusPending).
		Delete(&models.Captcha{})
	return result.RowsAffected, result.Error
}

func (r *CaptchaRepository) CacheSet(ctx context.Context, captcha *models.Captcha) error {
	cacheKey := "captcha:" + captcha.Token
	ttl := time.Until(captcha.ExpiresAt)
	if ttl <= 0 {
		return nil
	}
	return r.cache.Set(ctx, cacheKey, captcha, ttl)
}

func (r *CaptchaRepository) CacheGet(ctx context.Context, token string) (*models.Captcha, error) {
	cacheKey := "captcha:" + token
	var captcha models.Captcha
	if err := r.cache.Get(ctx, cacheKey, &captcha); err != nil {
		return nil, ErrCaptchaNotFound
	}
	return &captcha, nil
}

func (r *CaptchaRepository) CacheDelete(ctx context.Context, token string) error {
	cacheKey := "captcha:" + token
	return r.cache.Delete(ctx, cacheKey)
}

func (r *CaptchaRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&models.Captcha{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *CaptchaRepository) FindAll(page, pageSize int, status string) ([]models.Captcha, int64, error) {
	var captchas []models.Captcha
	var total int64

	query := r.db.Model(&models.Captcha{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&captchas).Error; err != nil {
		return nil, 0, err
	}

	return captchas, total, nil
}

func (r *CaptchaRepository) FindByToken(token string) (*models.Captcha, error) {
	return r.GetByToken(token)
}

func (r *CaptchaRepository) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var total int64
	r.db.Model(&models.Captcha{}).Count(&total)
	stats["total"] = total

	var pending int64
	r.db.Model(&models.Captcha{}).Where("status = ?", models.CaptchaStatusPending).Count(&pending)
	stats["pending"] = pending

	var verified int64
	r.db.Model(&models.Captcha{}).Where("status = ?", models.CaptchaStatusVerified).Count(&verified)
	stats["verified"] = verified

	var expired int64
	r.db.Model(&models.Captcha{}).Where("status = ?", models.CaptchaStatusExpired).Count(&expired)
	stats["expired"] = expired

	var failed int64
	r.db.Model(&models.Captcha{}).Where("status = ?", models.CaptchaStatusFailed).Count(&failed)
	stats["failed"] = failed

	return stats, nil
}
