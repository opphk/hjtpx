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
	ErrLogNotFound = errors.New("verification log not found")
)

type VerificationLogRepository struct {
	db    *gorm.DB
	cache *database.RedisCache
}

func NewVerificationLogRepository(db *gorm.DB, cache *database.RedisCache) *VerificationLogRepository {
	return &VerificationLogRepository{
		db:    db,
		cache: cache,
	}
}

func (r *VerificationLogRepository) Create(log *models.VerificationLog) error {
	if err := r.db.Create(log).Error; err != nil {
		return err
	}

	ctx := context.Background()
	cacheKey := "verification_log:" + string(rune(log.ID))
	r.cache.Set(ctx, cacheKey, log, 24*time.Hour)

	return nil
}

func (r *VerificationLogRepository) GetByID(id uint) (*models.VerificationLog, error) {
	ctx := context.Background()
	cacheKey := "verification_log:" + string(rune(id))

	exists, _ := r.cache.Exists(ctx, cacheKey)
	if exists {
		var log models.VerificationLog
		if err := r.cache.Get(ctx, cacheKey, &log); err == nil {
			return &log, nil
		}
	}

	var log models.VerificationLog
	if err := r.db.First(&log, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLogNotFound
		}
		return nil, err
	}

	r.cache.Set(ctx, cacheKey, &log, 24*time.Hour)

	return &log, nil
}

func (r *VerificationLogRepository) GetByToken(token string) ([]models.VerificationLog, error) {
	var logs []models.VerificationLog
	if err := r.db.Where("token = ?", token).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *VerificationLogRepository) List(page, pageSize int) ([]models.VerificationLog, int64, error) {
	var logs []models.VerificationLog
	var total int64

	if err := r.db.Model(&models.VerificationLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := r.db.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *VerificationLogRepository) ListByUserID(userID uint, page, pageSize int) ([]models.VerificationLog, int64, error) {
	var logs []models.VerificationLog
	var total int64

	query := r.db.Model(&models.VerificationLog{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *VerificationLogRepository) ListByIPAddress(ipAddress string, page, pageSize int) ([]models.VerificationLog, int64, error) {
	var logs []models.VerificationLog
	var total int64

	query := r.db.Model(&models.VerificationLog{}).Where("ip_address = ?", ipAddress)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *VerificationLogRepository) ListByAppID(appID uint, page, pageSize int) ([]models.VerificationLog, int64, error) {
	var logs []models.VerificationLog
	var total int64

	query := r.db.Model(&models.VerificationLog{}).Where("app_id = ?", appID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *VerificationLogRepository) GetStatsByUserID(userID uint, startTime, endTime time.Time) (*models.VerificationStats, error) {
	var stats models.VerificationStats

	type CountResult struct {
		Total int64
	}

	type ResultCount struct {
		Result  string
		Count   int64
	}

	var total CountResult
	if err := r.db.Model(&models.VerificationLog{}).
		Where("user_id = ? AND created_at BETWEEN ? AND ?", userID, startTime, endTime).
		Select("COUNT(*) as total").
		Scan(&total).Error; err != nil {
		return nil, err
	}
	stats.TotalAttempts = total.Total

	var results []ResultCount
	if err := r.db.Model(&models.VerificationLog{}).
		Where("user_id = ? AND created_at BETWEEN ? AND ?", userID, startTime, endTime).
		Select("result, COUNT(*) as count").
		Group("result").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	for _, rc := range results {
		if rc.Result == string(models.ResultSuccess) {
			stats.SuccessCount = rc.Count
		} else if rc.Result == string(models.ResultFailed) || rc.Result == string(models.ResultInvalid) {
			stats.FailedCount += rc.Count
		}
	}

	if stats.TotalAttempts > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalAttempts) * 100
	}

	var avgResponse struct {
		Avg float64
	}
	if err := r.db.Model(&models.VerificationLog{}).
		Where("user_id = ? AND created_at BETWEEN ? AND ?", userID, startTime, endTime).
		Select("AVG(response_time) as avg").
		Scan(&avgResponse).Error; err != nil {
		return nil, err
	}
	stats.AvgResponseTime = avgResponse.Avg

	return &stats, nil
}

func (r *VerificationLogRepository) GetStatsByIPAddress(ipAddress string, startTime, endTime time.Time) (*models.VerificationStats, error) {
	var stats models.VerificationStats

	type CountResult struct {
		Total int64
	}

	var total CountResult
	if err := r.db.Model(&models.VerificationLog{}).
		Where("ip_address = ? AND created_at BETWEEN ? AND ?", ipAddress, startTime, endTime).
		Select("COUNT(*) as total").
		Scan(&total).Error; err != nil {
		return nil, err
	}
	stats.TotalAttempts = total.Total

	type ResultCount struct {
		Result string
		Count  int64
	}

	var results []ResultCount
	if err := r.db.Model(&models.VerificationLog{}).
		Where("ip_address = ? AND created_at BETWEEN ? AND ?", ipAddress, startTime, endTime).
		Select("result, COUNT(*) as count").
		Group("result").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	for _, rc := range results {
		if rc.Result == string(models.ResultSuccess) {
			stats.SuccessCount = rc.Count
		} else if rc.Result == string(models.ResultFailed) || rc.Result == string(models.ResultInvalid) {
			stats.FailedCount += rc.Count
		}
	}

	if stats.TotalAttempts > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalAttempts) * 100
	}

	var avgResponse struct {
		Avg float64
	}
	if err := r.db.Model(&models.VerificationLog{}).
		Where("ip_address = ? AND created_at BETWEEN ? AND ?", ipAddress, startTime, endTime).
		Select("AVG(response_time) as avg").
		Scan(&avgResponse).Error; err != nil {
		return nil, err
	}
	stats.AvgResponseTime = avgResponse.Avg

	return &stats, nil
}

func (r *VerificationLogRepository) GetRecentByIPAddress(ipAddress string, duration time.Duration) ([]models.VerificationLog, error) {
	var logs []models.VerificationLog
	startTime := time.Now().Add(-duration)

	if err := r.db.Where("ip_address = ? AND created_at >= ?", ipAddress, startTime).
		Order("created_at DESC").
		Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

func (r *VerificationLogRepository) Delete(id uint) error {
	result := r.db.Delete(&models.VerificationLog{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrLogNotFound
	}

	ctx := context.Background()
	cacheKey := "verification_log:" + string(rune(id))
	r.cache.Delete(ctx, cacheKey)

	return nil
}

func (r *VerificationLogRepository) CleanOldLogs(days int) (int64, error) {
	cutoffTime := time.Now().AddDate(0, 0, -days)
	result := r.db.Where("created_at < ?", cutoffTime).Delete(&models.VerificationLog{})
	return result.RowsAffected, result.Error
}
