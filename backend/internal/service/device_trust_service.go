package service

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

type DeviceTrustService struct {
	db               *gorm.DB
	fpService        *DeviceFingerprintService
	cache            map[string]*DeviceTrustCache
	cacheMu          sync.RWMutex
	defaultConfig     TrustConfig
}

type TrustConfig struct {
	AutoTrustAfter    int             `json:"auto_trust_after"`
	TrustDurationDays int             `json:"trust_duration_days"`
	ChallengeThreshold float64         `json:"challenge_threshold"`
	BlockThreshold    float64         `json:"block_threshold"`
	MinTrustScore     float64         `json:"min_trust_score"`
	MaxTrustScore     float64         `json:"max_trust_score"`
}

type DeviceTrustCache struct {
	Trust      *models.DeviceTrust
	LastAccess time.Time
}

type TrustEvaluation struct {
	DeviceFingerprint string             `json:"device_fingerprint"`
	UserID           uint               `json:"user_id"`
	TrustLevel       models.TrustLevel   `json:"trust_level"`
	TrustScore       float64            `json:"trust_score"`
	IsTrusted        bool               `json:"is_trusted"`
	CanAutoTrust     bool               `json:"can_auto_trust"`
	Decision         string             `json:"decision"`
	Factors          []TrustFactor      `json:"factors"`
	ExpiresAt        *time.Time         `json:"expires_at"`
	LastVerifiedAt   *time.Time         `json:"last_verified_at"`
}

type TrustFactor struct {
	Name      string  `json:"name"`
	Weight    float64 `json:"weight"`
	Score     float64 `json:"score"`
	Positive  bool    `json:"positive"`
	Reason    string  `json:"reason"`
}

type TrustUpdateRequest struct {
	UserID           uint   `json:"user_id" binding:"required"`
	Fingerprint      string `json:"fingerprint" binding:"required"`
	Action           string `json:"action" binding:"required"`
	Success          bool   `json:"success"`
	FailureReason    string `json:"failure_reason"`
	RiskScore        float64 `json:"risk_score"`
	VerifiedBy       string `json:"verified_by"`
	Note             string `json:"note"`
}

const (
	TrustActionVerify   = "verify"
	TrustActionTrust    = "trust"
	TrustActionRevoke   = "revoke"
	TrustActionUpdate   = "update"
)

func NewDeviceTrustService(db *gorm.DB, fpService *DeviceFingerprintService) *DeviceTrustService {
	svc := &DeviceTrustService{
		db:        db,
		fpService: fpService,
		cache:     make(map[string]*DeviceTrustCache),
		defaultConfig: TrustConfig{
			AutoTrustAfter:    3,
			TrustDurationDays: 30,
			ChallengeThreshold: 30,
			BlockThreshold:    70,
			MinTrustScore:     0,
			MaxTrustScore:     100,
		},
	}
	go svc.cleanupCache()
	return svc
}

func (s *DeviceTrustService) cleanupCache() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		s.cacheMu.Lock()
		cutoff := time.Now().Add(-10 * time.Minute)
		for key, cached := range s.cache {
			if cached.LastAccess.Before(cutoff) {
				delete(s.cache, key)
			}
		}
		s.cacheMu.Unlock()
	}
}

func (s *DeviceTrustService) GetTrustByID(deviceID interface{}) (*models.DeviceTrust, error) {
	var trust models.DeviceTrust
	idStr, ok := deviceID.(string)
	if !ok {
		return nil, fmt.Errorf("invalid device ID")
	}
	err := s.db.First(&trust, idStr).Error
	return &trust, err
}

func (s *DeviceTrustService) GetTrustLevel(userID uint, fingerprint string) (*TrustEvaluation, error) {
	cacheKey := fmt.Sprintf("%d:%s", userID, fingerprint)
	s.cacheMu.RLock()
	if cached, exists := s.cache[cacheKey]; exists {
		cached.LastAccess = time.Now()
		s.cacheMu.RUnlock()
		return s.toTrustEvaluation(cached.Trust), nil
	}
	s.cacheMu.RUnlock()

	var trust models.DeviceTrust
	err := s.db.Where("user_id = ? AND device_fingerprint = ?", userID, fingerprint).First(&trust).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return s.evaluateNewDevice(userID, fingerprint)
		}
		return nil, err
	}

	if trust.ExpiresAt != nil && trust.ExpiresAt.Before(time.Now()) {
		trust.IsTrusted = false
		trust.TrustLevel = "expired"
	}

	s.cacheMu.Lock()
	s.cache[cacheKey] = &DeviceTrustCache{
		Trust:      &trust,
		LastAccess: time.Now(),
	}
	s.cacheMu.Unlock()

	return s.toTrustEvaluation(&trust), nil
}

func (s *DeviceTrustService) evaluateNewDevice(userID uint, fingerprint string) (*TrustEvaluation, error) {
	fpRecord, err := s.fpService.GetFingerprint(fingerprint)
	if err != nil {
		return &TrustEvaluation{
			DeviceFingerprint: fingerprint,
			UserID:           userID,
			TrustLevel:       models.TrustLevelUnknown,
			TrustScore:       0,
			IsTrusted:        false,
			CanAutoTrust:     false,
			Decision:         "challenge",
			Factors:          []TrustFactor{},
		}, nil
	}

	evaluation := &TrustEvaluation{
		DeviceFingerprint: fingerprint,
		UserID:           userID,
		TrustLevel:       models.TrustLevelNone,
		TrustScore:       0,
		IsTrusted:        false,
		CanAutoTrust:     false,
		Decision:         "challenge",
		Factors:          []TrustFactor{},
	}

	if fpRecord.RiskScore < 30 {
		evaluation.Factors = append(evaluation.Factors, TrustFactor{
			Name:     "Low Risk Device",
			Weight:   0.3,
			Score:    20,
			Positive: true,
			Reason:   "Device has low risk score",
		})
	}

	if fpRecord.VisitCount >= 3 {
		evaluation.Factors = append(evaluation.Factors, TrustFactor{
			Name:     "Multiple Visits",
			Weight:   0.2,
			Score:    15,
			Positive: true,
			Reason:   "Device has been seen multiple times",
		})
	}

	if !fpRecord.ProxyDetected && !fpRecord.VPNDetected && !fpRecord.TorDetected {
		evaluation.Factors = append(evaluation.Factors, TrustFactor{
			Name:     "Clean Network",
			Weight:   0.3,
			Score:    25,
			Positive: true,
			Reason:   "No proxy, VPN, or Tor detected",
		})
	}

	if !fpRecord.IsBot {
		evaluation.Factors = append(evaluation.Factors, TrustFactor{
			Name:     "Not Bot",
			Weight:   0.2,
			Score:    20,
			Positive: true,
			Reason:   "Not identified as automated bot",
		})
	}

	evaluation.TrustScore = s.calculateTrustScore(evaluation.Factors)
	evaluation.TrustLevel = s.scoreToTrustLevel(evaluation.TrustScore)

	return evaluation, nil
}

func (s *DeviceTrustService) toTrustEvaluation(trust *models.DeviceTrust) *TrustEvaluation {
	evaluation := &TrustEvaluation{
		DeviceFingerprint: trust.DeviceFingerprint,
		UserID:           trust.UserID,
		TrustLevel:       trust.TrustLevel,
		TrustScore:       trust.TrustScore,
		IsTrusted:        trust.IsTrusted,
		CanAutoTrust:     trust.ContinuousSuccess >= s.defaultConfig.AutoTrustAfter,
		Decision:         s.trustLevelToDecision(trust.TrustLevel, trust.TrustScore),
		Factors:          []TrustFactor{},
		ExpiresAt:        trust.ExpiresAt,
		LastVerifiedAt:   trust.LastVerifiedAt,
	}

	evaluation.Factors = append(evaluation.Factors, TrustFactor{
		Name:     "Successful Verifications",
		Weight:   0.3,
		Score:    float64(trust.SuccessCount) * 5,
		Positive: true,
		Reason:   fmt.Sprintf("%d successful verifications", trust.SuccessCount),
	})

	if trust.FailureCount > 0 {
		evaluation.Factors = append(evaluation.Factors, TrustFactor{
			Name:     "Failed Verifications",
			Weight:   0.2,
			Score:    float64(trust.FailureCount) * 10,
			Positive: false,
			Reason:   fmt.Sprintf("%d failed verifications", trust.FailureCount),
		})
	}

	if trust.ContinuousSuccess > 0 {
		evaluation.Factors = append(evaluation.Factors, TrustFactor{
			Name:     "Continuous Success",
			Weight:   0.3,
			Score:    float64(trust.ContinuousSuccess) * 10,
			Positive: trust.ContinuousSuccess >= 3,
			Reason:   fmt.Sprintf("%d continuous successful verifications", trust.ContinuousSuccess),
		})
	}

	if trust.IsAutoTrusted {
		evaluation.Factors = append(evaluation.Factors, TrustFactor{
			Name:     "Auto Trusted",
			Weight:   0.2,
			Score:    20,
			Positive: true,
			Reason:   "Device was automatically trusted",
		})
	}

	return evaluation
}

func (s *DeviceTrustService) trustLevelToDecision(level models.TrustLevel, score float64) string {
	switch level {
	case models.TrustLevelFull:
		return "allow"
	case models.TrustLevelHigh:
		if score >= 80 {
			return "allow"
		}
		return "challenge"
	case models.TrustLevelMedium:
		if score >= 70 {
			return "allow"
		}
		return "challenge"
	case models.TrustLevelLow:
		if score >= 60 {
			return "challenge"
		}
		return "block"
	case models.TrustLevelNone:
		return "block"
	default:
		return "challenge"
	}
}

func (s *DeviceTrustService) UpdateTrust(req *TrustUpdateRequest) (*models.DeviceTrust, error) {
	var trust models.DeviceTrust
	err := s.db.Where("user_id = ? AND device_fingerprint = ?", req.UserID, req.Fingerprint).First(&trust).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return s.createTrust(req)
		}
		return nil, err
	}

	return s.modifyTrust(&trust, req)
}

func (s *DeviceTrustService) createTrust(req *TrustUpdateRequest) (*models.DeviceTrust, error) {
	now := time.Now()
	trust := &models.DeviceTrust{
		UserID:            req.UserID,
		DeviceFingerprint: req.Fingerprint,
		TrustLevel:        models.TrustLevelLow,
		TrustScore:        50,
		IsTrusted:         false,
		FirstTrustedAt:    &now,
		LastVerifiedAt:    &now,
		SuccessCount:      0,
		FailureCount:      0,
		ContinuousSuccess: 0,
	}

	if req.Action == TrustActionTrust {
		trust.IsTrusted = true
		trust.TrustLevel = models.TrustLevelMedium
		trust.TrustScore = 70
		trust.ExpiresAt = s.calculateExpiryTime()
		trust.TrustedBy = req.VerifiedBy
	}

	if req.Success {
		trust.SuccessCount++
		trust.ContinuousSuccess++
	} else {
		trust.FailureCount++
		trust.ContinuousSuccess = 0
		trust.LastFailedAt = &now
	}

	err := s.db.Create(trust).Error
	if err != nil {
		return nil, err
	}

	s.recordTrustLog(trust, req)

	return trust, nil
}

func (s *DeviceTrustService) modifyTrust(trust *models.DeviceTrust, req *TrustUpdateRequest) (*models.DeviceTrust, error) {
	now := time.Now()
	oldLevel := trust.TrustLevel
	oldScore := trust.TrustScore

	switch req.Action {
	case TrustActionVerify:
		trust.LastVerifiedAt = &now
		if req.Success {
			trust.SuccessCount++
			trust.ContinuousSuccess++
			trust.TrustScore = math.Min(trust.TrustScore+5, 100)
			trust.TrustLevel = s.scoreToTrustLevel(trust.TrustScore)

			if trust.ContinuousSuccess >= s.defaultConfig.AutoTrustAfter && !trust.IsTrusted {
				trust.IsTrusted = true
				trust.IsAutoTrusted = true
				trust.ExpiresAt = s.calculateExpiryTime()
				trust.TrustedBy = "auto"
			}
		} else {
			trust.FailureCount++
			trust.ContinuousSuccess = 0
			trust.LastFailedAt = &now
			trust.TrustScore = math.Max(trust.TrustScore-10, 0)
			trust.TrustLevel = s.scoreToTrustLevel(trust.TrustScore)

			if trust.FailureCount >= 5 {
				trust.IsTrusted = false
				trust.IsAutoTrusted = false
				trust.ExpiresAt = nil
			}
		}

	case TrustActionTrust:
		trust.IsTrusted = true
		trust.TrustLevel = models.TrustLevelHigh
		trust.TrustScore = 90
		trust.ExpiresAt = s.calculateExpiryTime()
		trust.TrustedBy = req.VerifiedBy

	case TrustActionRevoke:
		trust.IsTrusted = false
		trust.IsAutoTrusted = false
		trust.TrustLevel = models.TrustLevelNone
		trust.TrustScore = 0
		trust.ExpiresAt = nil
		trust.ContinuousSuccess = 0

	case TrustActionUpdate:
		if req.RiskScore > 0 {
			trust.TrustScore = math.Max(0, trust.TrustScore-req.RiskScore/10)
			trust.TrustLevel = s.scoreToTrustLevel(trust.TrustScore)
		}
	}

	trust.UpdatedAt = now
	if trust.Note == "" && req.Note != "" {
		trust.Note = req.Note
	}

	err := s.db.Save(trust).Error
	if err != nil {
		return nil, err
	}

	req.FailureReason = fmt.Sprintf("Level: %s->%s, Score: %.1f->%.1f", oldLevel, trust.TrustLevel, oldScore, trust.TrustScore)
	s.recordTrustLog(trust, req)

	return trust, nil
}

func (s *DeviceTrustService) calculateExpiryTime() *time.Time {
	expiry := time.Now().Add(time.Duration(s.defaultConfig.TrustDurationDays) * 24 * time.Hour)
	return &expiry
}

func (s *DeviceTrustService) scoreToTrustLevel(score float64) models.TrustLevel {
	switch {
	case score >= 90:
		return models.TrustLevelFull
	case score >= 75:
		return models.TrustLevelHigh
	case score >= 50:
		return models.TrustLevelMedium
	case score >= 25:
		return models.TrustLevelLow
	default:
		return models.TrustLevelNone
	}
}

func (s *DeviceTrustService) calculateTrustScore(factors []TrustFactor) float64 {
	var totalScore, totalWeight float64

	for _, factor := range factors {
		if factor.Positive {
			totalScore += factor.Score * factor.Weight
		} else {
			totalScore -= factor.Score * factor.Weight
		}
		totalWeight += factor.Weight
	}

	if totalWeight == 0 {
		return 50
	}

	normalizedScore := (totalScore / totalWeight) + 50
	return math.Max(0, math.Min(100, normalizedScore))
}

func (s *DeviceTrustService) recordTrustLog(trust *models.DeviceTrust, req *TrustUpdateRequest) {
	oldTrust, _ := s.GetTrustLevel(req.UserID, req.Fingerprint)

	log := &models.TrustLog{
		UserID:            req.UserID,
		FingerprintID:     trust.ID,
		DeviceFingerprint: req.Fingerprint,
		Action:            req.Action,
		IPAddress:         "",
		UserAgent:         "",
		RiskScore:         req.RiskScore,
		CreatedAt:         time.Now(),
	}

	if oldTrust != nil {
		log.OldTrustLevel = oldTrust.TrustLevel
		log.OldTrustScore = oldTrust.TrustScore
	}

	log.NewTrustLevel = trust.TrustLevel
	log.NewTrustScore = trust.TrustScore
	log.Reason = req.Note

	if req.FailureReason != "" {
		log.Reason = req.FailureReason
	}

	s.db.Create(log)
}

func (s *DeviceTrustService) ListTrustedDevices(userID uint, page, pageSize int) ([]models.DeviceTrust, int64, error) {
	var devices []models.DeviceTrust
	var total int64

	s.db.Model(&models.DeviceTrust{}).Where("user_id = ?", userID).Count(&total)

	offset := (page - 1) * pageSize
	err := s.db.Where("user_id = ?", userID).
		Order("last_verified_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&devices).Error

	return devices, total, err
}

func (s *DeviceTrustService) RevokeTrust(userID uint, fingerprint string, reason string) error {
	req := &TrustUpdateRequest{
		UserID:      userID,
		Fingerprint: fingerprint,
		Action:      TrustActionRevoke,
		Note:        reason,
	}
	_, err := s.UpdateTrust(req)
	return err
}

func (s *DeviceTrustService) ManualTrust(userID uint, fingerprint string, trustedBy string, note string) error {
	req := &TrustUpdateRequest{
		UserID:      userID,
		Fingerprint: fingerprint,
		Action:      TrustActionTrust,
		VerifiedBy:  trustedBy,
		Note:        note,
	}
	_, err := s.UpdateTrust(req)
	return err
}

func (s *DeviceTrustService) CheckDeviceExpired() error {
	now := time.Now()
	return s.db.Model(&models.DeviceTrust{}).
		Where("is_trusted = ? AND expires_at IS NOT NULL AND expires_at < ?", true, now).
		Updates(map[string]interface{}{
			"is_trusted": false,
			"trust_level": "expired",
			"updated_at": now,
		}).Error
}

func (s *DeviceTrustService) GetTrustStats(userID uint) (map[string]interface{}, error) {
	var totalDevices, trustedDevices, expiredDevices int64
	var avgTrustScore float64

	s.db.Model(&models.DeviceTrust{}).Where("user_id = ?", userID).Count(&totalDevices)
	s.db.Model(&models.DeviceTrust{}).Where("user_id = ? AND is_trusted = ?", userID, true).Count(&trustedDevices)
	s.db.Model(&models.DeviceTrust{}).Where("user_id = ? AND is_trusted = ? AND expires_at IS NOT NULL AND expires_at < ?", userID, true, time.Now()).Count(&expiredDevices)

	var sumScore float64
	var scoreCount int64
	s.db.Model(&models.DeviceTrust{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(trust_score), 0)").Row().Scan(&sumScore)
	s.db.Model(&models.DeviceTrust{}).
		Where("user_id = ? AND trust_score > 0", userID).
		Count(&scoreCount)

	if scoreCount > 0 {
		avgTrustScore = sumScore / float64(scoreCount)
	}

	levelDistribution := make(map[string]int64)
	for _, level := range []models.TrustLevel{models.TrustLevelFull, models.TrustLevelHigh, models.TrustLevelMedium, models.TrustLevelLow, models.TrustLevelNone} {
		var count int64
		s.db.Model(&models.DeviceTrust{}).
			Where("user_id = ? AND trust_level = ?", userID, level).
			Count(&count)
		levelDistribution[string(level)] = count
	}

	return map[string]interface{}{
		"total_devices":        totalDevices,
		"trusted_devices":      trustedDevices,
		"expired_devices":      expiredDevices,
		"average_trust_score":  avgTrustScore,
		"level_distribution":   levelDistribution,
	}, nil
}

func (s *DeviceTrustService) GetTrustHistory(userID uint, fingerprint string, limit int) ([]models.TrustLog, error) {
	var logs []models.TrustLog
	query := s.db.Where("user_id = ? AND device_fingerprint = ?", userID, fingerprint)

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Order("created_at DESC").Find(&logs).Error
	return logs, err
}

func (s *DeviceTrustService) ExportTrustReport(userID uint) (map[string]interface{}, error) {
	stats, err := s.GetTrustStats(userID)
	if err != nil {
		return nil, err
	}

	var devices []models.DeviceTrust
	s.db.Where("user_id = ?", userID).
		Order("trust_score DESC").
		Limit(100).
		Find(&devices)

	highRiskDevices := []models.DeviceTrust{}
	trustedDevices := []models.DeviceTrust{}

	for _, device := range devices {
		if device.TrustScore < 30 {
			highRiskDevices = append(highRiskDevices, device)
		}
		if device.IsTrusted {
			trustedDevices = append(trustedDevices, device)
		}
	}

	return map[string]interface{}{
		"generated_at":       time.Now(),
		"stats":              stats,
		"high_risk_devices":  highRiskDevices,
		"trusted_devices":    trustedDevices,
	}, nil
}
