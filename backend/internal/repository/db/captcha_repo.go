package db

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"gorm.io/gorm"
)

type CaptchaRepository struct {
	cache *redis.EnhancedCache
}

func NewCaptchaRepository() *CaptchaRepository {
	return &CaptchaRepository{
		cache: redis.GetEnhancedCache(),
	}
}

func (r *CaptchaRepository) getCacheKey(sessionID string) string {
	return fmt.Sprintf("captcha:session:%s", sessionID)
}

func (r *CaptchaRepository) Create(session *models.CaptchaSession) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	if err := db.Create(session).Error; err != nil {
		return err
	}
	
	if r.cache != nil && session.SessionID != "" {
		ctx := redis.GetContext()
		r.cache.Set(ctx, r.getCacheKey(session.SessionID), mustMarshal(session), &redis.SetOptions{
			TTL:   10 * time.Minute,
			Level: redis.CacheLevelL1,
		})
	}
	
	return nil
}

func (r *CaptchaRepository) GetBySessionID(sessionID string) (*models.CaptchaSession, error) {
	if r.cache != nil && sessionID != "" {
		ctx := redis.GetContext()
		var session models.CaptchaSession
		if err := r.cache.GetJSON(ctx, r.getCacheKey(sessionID), &session, nil); err == nil {
			return &session, nil
		}
	}
	
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var session models.CaptchaSession
	err := db.Where("session_id = ?", sessionID).First(&session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	
	if r.cache != nil && session.SessionID != "" {
		ctx := redis.GetContext()
		r.cache.Set(ctx, r.getCacheKey(session.SessionID), mustMarshal(&session), &redis.SetOptions{
			TTL:   10 * time.Minute,
			Level: redis.CacheLevelL1,
		})
	}
	
	return &session, nil
}

func (r *CaptchaRepository) GetBySessionIDNoCache(sessionID string) (*models.CaptchaSession, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var session models.CaptchaSession
	err := db.Where("session_id = ?", sessionID).First(&session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

func (r *CaptchaRepository) GetBySessionIDs(sessionIDs []string) ([]*models.CaptchaSession, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}

	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var sessions []*models.CaptchaSession
	err := db.Where("session_id IN ?", sessionIDs).Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *CaptchaRepository) GetBySessionIDsBatch(sessionIDs []string) ([]*models.CaptchaSession, error) {
	if len(sessionIDs) == 0 {
		return []*models.CaptchaSession{}, nil
	}

	var results []*models.CaptchaSession
	var uncachedIDs []string
	
	if r.cache != nil {
		ctx := redis.GetContext()
		cacheKeys := make([]string, len(sessionIDs))
		for i, id := range sessionIDs {
			cacheKeys[i] = r.getCacheKey(id)
		}
		
		cached, err := r.cache.MGet(ctx, cacheKeys, &redis.GetOptions{Level: redis.CacheLevelL1})
		if err == nil {
			for _, id := range sessionIDs {
				if val, ok := cached[r.getCacheKey(id)]; ok {
					var session models.CaptchaSession
					if mustUnmarshal(val, &session) == nil {
						results = append(results, &session)
						continue
					}
				}
				uncachedIDs = append(uncachedIDs, id)
			}
		} else {
			uncachedIDs = sessionIDs
		}
	} else {
		uncachedIDs = sessionIDs
	}

	if len(uncachedIDs) > 0 {
		db := database.GetDB()
		if db == nil {
			return results, nil
		}
		
		var sessions []*models.CaptchaSession
		if err := db.Where("session_id IN ?", uncachedIDs).Find(&sessions).Error; err != nil {
			return results, err
		}
		
		results = append(results, sessions...)
		
		if r.cache != nil && len(sessions) > 0 {
			ctx := redis.GetContext()
			pipeData := make(map[string][]byte)
			for _, session := range sessions {
				if session.SessionID != "" {
					pipeData[r.getCacheKey(session.SessionID)] = mustMarshal(session)
				}
			}
			if len(pipeData) > 0 {
				pipe := redis.GetClient().Pipeline()
				for key, data := range pipeData {
					pipe.Set(ctx, key, data, 10*time.Minute)
				}
				pipe.Exec(ctx)
			}
		}
	}

	return results, nil
}

func (r *CaptchaRepository) UpdateStatus(sessionID string, status string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	updates := map[string]interface{}{
		"status": status,
	}

	if status == "verified" {
		now := time.Now()
		updates["verified_at"] = &now
	}

	if err := db.Model(&models.CaptchaSession{}).
		Where("session_id = ?", sessionID).
		Updates(updates).Error; err != nil {
		return err
	}

	if r.cache != nil {
		ctx := redis.GetContext()
		r.cache.Delete(ctx, r.getCacheKey(sessionID), &redis.DeleteOptions{Level: redis.CacheLevelL1})
	}

	return nil
}

func (r *CaptchaRepository) UpdateVerifyCount(sessionID string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	return db.Model(&models.CaptchaSession{}).
		Where("session_id = ?", sessionID).
		Update("verify_count", gorm.Expr("verify_count + 1")).Error
}

func (r *CaptchaRepository) UpdateRiskScore(sessionID string, riskScore, traceScore, envScore float64) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	return db.Model(&models.CaptchaSession{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]interface{}{
			"risk_score":  riskScore,
			"trace_score": traceScore,
			"env_score":   envScore,
		}).Error
}

func (r *CaptchaRepository) Delete(sessionID string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	if err := db.Where("session_id = ?", sessionID).Delete(&models.CaptchaSession{}).Error; err != nil {
		return err
	}

	if r.cache != nil {
		ctx := redis.GetContext()
		r.cache.Delete(ctx, r.getCacheKey(sessionID), &redis.DeleteOptions{Level: redis.CacheLevelBoth})
	}

	return nil
}

func (r *CaptchaRepository) GetExpiredSessions(olderThan time.Duration) ([]*models.CaptchaSession, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	cutoff := time.Now().Add(-olderThan)
	var sessions []*models.CaptchaSession

	err := db.Where("expired_at < ? AND status = ?", cutoff, "pending").
		Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *CaptchaRepository) CleanupExpired(olderThan time.Duration) (int64, error) {
	db := database.GetDB()
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	cutoff := time.Now().Add(-olderThan)
	result := db.Where("expired_at < ?", cutoff).Delete(&models.CaptchaSession{})
	return result.RowsAffected, result.Error
}

func (r *CaptchaRepository) GetPendingSessionsByAppID(appID string, limit int) ([]*models.CaptchaSession, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var sessions []*models.CaptchaSession
	err := db.Where("application_id = ? AND status = ?", appID, "pending").
		Order("created_at DESC").
		Limit(limit).
		Find(&sessions).Error
	return sessions, err
}

func (r *CaptchaRepository) CountByStatus(status string) (int64, error) {
	db := database.GetDB()
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	var count int64
	err := db.Model(&models.CaptchaSession{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

func (r *CaptchaRepository) CreateVoiceSession(session *models.VoiceCaptchaSession) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	return db.Create(session).Error
}

func (r *CaptchaRepository) GetVoiceSession(sessionID string) (*models.VoiceCaptchaSession, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var session models.VoiceCaptchaSession
	err := db.Where("session_id = ?", sessionID).First(&session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &session, nil
}

func (r *CaptchaRepository) IncrementVoiceVerifyCount(sessionID string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	return db.Model(&models.VoiceCaptchaSession{}).
		Where("session_id = ?", sessionID).
		Update("verify_count", gorm.Expr("verify_count + 1")).Error
}

func (r *CaptchaRepository) MarkVoiceAsVerified(sessionID string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	now := time.Now()
	return db.Model(&models.VoiceCaptchaSession{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]interface{}{
			"status":      "verified",
			"verified_at": &now,
		}).Error
}

func mustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

func mustUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
