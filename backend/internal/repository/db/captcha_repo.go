package db

import (
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

type CaptchaRepository struct{}

func NewCaptchaRepository() *CaptchaRepository {
	return &CaptchaRepository{}
}

func (r *CaptchaRepository) Create(session *models.CaptchaSession) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	return db.Create(session).Error
}

func (r *CaptchaRepository) GetBySessionID(sessionID string) (*models.CaptchaSession, error) {
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

	return db.Model(&models.CaptchaSession{}).
		Where("session_id = ?", sessionID).
		Updates(updates).Error
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

	return db.Where("session_id = ?", sessionID).Delete(&models.CaptchaSession{}).Error
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
