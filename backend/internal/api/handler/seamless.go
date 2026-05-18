package handler

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type SeamlessVerifyRequest struct {
	SessionID         string                 `json:"session_id" binding:"required"`
	DeviceFingerprint string                 `json:"device_fingerprint" binding:"required"`
	ApplicationID     uint                   `json:"application_id"`
	UserID            *uint                  `json:"user_id,omitempty"`
	BehaviorData      []BehaviorDataPoint    `json:"behavior_data,omitempty"`
	EnvironmentData   map[string]interface{} `json:"environment_data,omitempty"`
	FingerprintHash   string                 `json:"fingerprint_hash,omitempty"`
	FeatureCount      int                    `json:"feature_count,omitempty"`
	RiskScore         *RiskScoreData         `json:"risk_score,omitempty"`
}

type BehaviorDataPoint struct {
	Event      string                 `json:"event"`
	Timestamp  int64                  `json:"timestamp"`
	Data       map[string]interface{} `json:"data,omitempty"`
	X          float64                `json:"x,omitempty"`
	Y          float64                `json:"y,omitempty"`
	Target     string                 `json:"target,omitempty"`
	ClickX     float64                `json:"click_x,omitempty"`
	ClickY     float64                `json:"click_y,omitempty"`
	KeyCode    string                 `json:"key_code,omitempty"`
	Delay      int64                  `json:"delay,omitempty"`
	HoldTime   int64                  `json:"hold_time,omitempty"`
}

type RiskScoreData struct {
	TotalScore  float64                `json:"total_score"`
	Level       string                 `json:"level"`
	RiskFactors map[string]float64     `json:"risk_factors"`
	Details     []string               `json:"details,omitempty"`
}

type SeamlessVerifyResponse struct {
	Decision     string  `json:"decision"`
	RiskScore    float64 `json:"risk_score"`
	TrustLevel   float64 `json:"trust_level"`
	Reason       string  `json:"reason,omitempty"`
	Token        string  `json:"token,omitempty"`
	Trusted      bool    `json:"trusted"`
	SessionID    string  `json:"session_id,omitempty"`
	CacheHit     bool    `json:"cache_hit,omitempty"`
	RemainingTTL int64   `json:"remaining_ttl,omitempty"`
}

type SeamlessConfigRequest struct {
	ApplicationID              uint    `json:"application_id" binding:"required"`
	Enabled                    bool    `json:"enabled"`
	ChallengeThreshold         float64 `json:"challenge_threshold"`
	BlockThreshold            float64 `json:"block_threshold"`
	TrustExpiryDays            int     `json:"trust_expiry_days"`
	RequireBehaviorAnalysis    bool    `json:"require_behavior_analysis"`
	ForceVerificationThreshold float64 `json:"force_verification_threshold"`
	EnableAutoTrust            bool    `json:"enable_auto_trust"`
	MinTrustLevel              float64 `json:"min_trust_level"`
	MaxDevicePerUser           int     `json:"max_device_per_user"`
}

type SeamlessConfigResponse struct {
	Success bool                  `json:"success"`
	Config  *models.SeamlessConfig `json:"config,omitempty"`
}

var (
	behaviorService interface{}
	deviceService    interface{}
)

func SetBehaviorService(service interface{}) {
	behaviorService = service
}

func SetDeviceDetectionService(service interface{}) {
	deviceService = service
}

func SeamlessVerify(c *gin.Context) {
	startTime := time.Now()
	var req SeamlessVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	var appConfig models.SeamlessConfig
	configEnabled := false
	if req.ApplicationID > 0 {
		database.DB.Where("application_id = ?", req.ApplicationID).First(&appConfig)
		if appConfig.ID > 0 {
			configEnabled = appConfig.Enabled
		}
	}

	if !configEnabled && req.ApplicationID > 0 {
		response.Success(c, SeamlessVerifyResponse{
			Decision:  "allow",
			RiskScore: 0,
			TrustLevel: 100,
			Reason:    "seamless_disabled",
			Trusted:   true,
		})
		return
	}

	trustStatus := checkDeviceTrust(req.UserID, req.DeviceFingerprint, req.FingerprintHash)
	if trustStatus.IsTrusted {
		remainingTTL := int64(0)
		if trustStatus.ExpiresAt != nil {
			remainingTTL = int64(time.Until(*trustStatus.ExpiresAt).Seconds())
		}
		if remainingTTL > 0 {
			response.Success(c, SeamlessVerifyResponse{
				Decision:     "allow",
				RiskScore:     0,
				TrustLevel:    100,
				Reason:       "device_trusted",
				Trusted:       true,
				CacheHit:      true,
				RemainingTTL:  remainingTTL,
				SessionID:     req.SessionID,
			})
			return
		}
	}

	riskScore := calculateRiskScore(req, appConfig)

	decision := determineDecision(riskScore, trustStatus, appConfig)
	reason := getDecisionReason(decision, riskScore, trustStatus)

	trustLevel := 100 - riskScore
	if trustStatus.IsTrusted {
		trustLevel = 100
	}

	token := ""
	sessionID := req.SessionID
	if decision == "allow" {
		token = uuid.New().String()
		sessionID = uuid.New().String()
	}

	seamlessLog := models.SeamlessVerification{
		SessionID:         sessionID,
		ApplicationID:     &req.ApplicationID,
		UserID:            req.UserID,
		DeviceFingerprint: req.DeviceFingerprint,
		Decision:          decision,
		RiskScore:         riskScore,
		Reason:            reason,
		IPAddress:         c.ClientIP(),
		UserAgent:         c.GetHeader("User-Agent"),
		Duration:          time.Since(startTime).Milliseconds(),
	}
	database.DB.Create(&seamlessLog)

	response.Success(c, SeamlessVerifyResponse{
		Decision:   decision,
		RiskScore:  riskScore,
		TrustLevel: trustLevel,
		Reason:     reason,
		Token:      token,
		Trusted:    decision == "allow",
		SessionID:  sessionID,
	})
}

func checkDeviceTrust(userID *uint, fingerprint string, fingerprintHash string) DeviceTrustStatus {
	status := DeviceTrustStatus{IsTrusted: false}

	if userID == nil {
		if fingerprintHash != "" {
			var trusted models.TrustedDevice
			query := database.DB.Where("device_fingerprint = ? AND is_trusted = ? AND (expires_at IS NULL OR expires_at > ?)",
				fingerprint, true, time.Now())

			if query.First(&trusted).Error == nil {
				status.IsTrusted = true
				status.ExpiresAt = trusted.ExpiresAt
				status.TrustLevel = float64(trusted.UseCount) * 10
				if status.TrustLevel > 100 {
					status.TrustLevel = 100
				}

				trusted.LastUsedAt = time.Now()
				trusted.UseCount++
				database.DB.Save(&trusted)
			}
		}
		return status
	}

	var trusted models.TrustedDevice
	query := database.DB.Where("user_id = ? AND device_fingerprint = ? AND is_trusted = ? AND (expires_at IS NULL OR expires_at > ?)",
		*userID, fingerprint, true, time.Now())

	if query.First(&trusted).Error == nil {
		status.IsTrusted = true
		status.ExpiresAt = trusted.ExpiresAt
		status.TrustLevel = float64(trusted.UseCount) * 10
		if status.TrustLevel > 100 {
			status.TrustLevel = 100
		}

		trusted.LastUsedAt = time.Now()
		trusted.UseCount++
		database.DB.Save(&trusted)
	}

	return status
}

type DeviceTrustStatus struct {
	IsTrusted  bool
	ExpiresAt  *time.Time
	TrustLevel float64
}

func calculateRiskScore(req SeamlessVerifyRequest, config models.SeamlessConfig) float64 {
	riskScore := 0.0

	if req.RiskScore != nil {
		return req.RiskScore.TotalScore
	}

	if len(req.BehaviorData) > 0 {
		behaviorRisk := analyzeBehaviorRisk(req.BehaviorData)
		riskScore += behaviorRisk * 0.4
	}

	envRisk := analyzeEnvironmentRisk(req.EnvironmentData)
	riskScore += envRisk * 0.3

	fpRisk := analyzeFingerprintRisk(req.FingerprintHash, req.FeatureCount)
	riskScore += fpRisk * 0.3

	return riskScore
}

func analyzeBehaviorRisk(behaviorData []BehaviorDataPoint) float64 {
	if len(behaviorData) == 0 {
		return 50.0
	}

	risk := 0.0
	anomalyCount := 0

	clickCount := 0
	keyCount := 0
	mouseCount := 0

	for i, data := range behaviorData {
		switch data.Event {
		case "click":
			clickCount++
			if i > 0 && behaviorData[i-1].Event == "click" {
				if data.Timestamp-behaviorData[i-1].Timestamp < 50 {
					anomalyCount++
				}
			}
		case "keydown", "keyup":
			keyCount++
			if data.Delay > 0 && data.Delay < 20 {
				anomalyCount++
			}
		case "mousemove":
			mouseCount++
		}
	}

	if clickCount == 0 && keyCount == 0 && mouseCount < 5 {
		risk += 30
	}

	if anomalyCount > len(behaviorData)*0.3 {
		risk += 40
	}

	if keyCount > 0 {
		keyDelays := make([]int64, 0)
		for _, data := range behaviorData {
			if (data.Event == "keydown" || data.Event == "keyup") && data.Delay > 0 {
				keyDelays = append(keyDelays, data.Delay)
			}
		}
		if len(keyDelays) > 2 {
			variance := calculateVariance(keyDelays)
			if variance < 100 {
				risk += 25
			}
		}
	}

	return minFloat(risk, 100)
}

func analyzeEnvironmentRisk(envData map[string]interface{}) float64 {
	if envData == nil {
		return 20.0
	}

	risk := 0.0

	if isEmulator, ok := envData["isEmulator"].(bool); ok && isEmulator {
		risk += 50
	}

	if isVirtual, ok := envData["isVirtual"].(bool); ok && isVirtual {
		risk += 40
	}

	if isContainer, ok := envData["isContainer"].(bool); ok && isContainer {
		risk += 30
	}

	if isHeadless, ok := envData["isHeadlessBrowser"].(bool); ok && isHeadless {
		risk += 45
	}

	if webdriver, ok := envData["webdriverStatus"].(bool); ok && webdriver {
		risk += 50
	}

	return minFloat(risk, 100)
}

func analyzeFingerprintRisk(fingerprintHash string, featureCount int) float64 {
	risk := 0.0

	if fingerprintHash == "" {
		risk += 30
	}

	expectedFeatures := 30
	if featureCount < expectedFeatures/2 {
		risk += 40
	} else if featureCount < expectedFeatures {
		risk += 20
	}

	return minFloat(risk, 100)
}

func calculateVariance(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := int64(0)
	for _, v := range values {
		sum += v
	}
	mean := float64(sum) / float64(len(values))

	variance := 0.0
	for _, v := range values {
		diff := float64(v) - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return variance
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func determineDecision(riskScore float64, trustStatus DeviceTrustStatus, config models.SeamlessConfig) string {
	if trustStatus.IsTrusted {
		return "allow"
	}

	challengeThreshold := config.ChallengeThreshold
	blockThreshold := config.BlockThreshold

	if challengeThreshold == 0 {
		challengeThreshold = 30
	}
	if blockThreshold == 0 {
		blockThreshold = 70
	}

	if riskScore >= blockThreshold {
		return "block"
	}

	if riskScore < challengeThreshold {
		return "allow"
	}

	return "challenge"
}

func getDecisionReason(decision string, riskScore float64, trustStatus DeviceTrustStatus) string {
	switch decision {
	case "allow":
		if trustStatus.IsTrusted {
			return "device_trusted"
		}
		return "low_risk"
	case "block":
		if riskScore >= 80 {
			return "high_risk"
		}
		return "risk_exceeded_threshold"
	case "challenge":
		return "medium_risk"
	}
	return "unknown"
}

func TrustDevice(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	var req TrustDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	uid := userID.(uint)
	var appConfig models.SeamlessConfig
	database.DB.Where("application_id = ?", req.ApplicationID).First(&appConfig)

	expiryDays := appConfig.TrustExpiryDays
	if expiryDays == 0 {
		expiryDays = 30
	}

	expiresAt := time.Now().AddDate(0, 0, expiryDays)

	var existing models.TrustedDevice
	database.DB.Where("user_id = ? AND device_fingerprint = ?", uid, req.DeviceFingerprint).First(&existing)

	if existing.ID > 0 {
		existing.IsTrusted = true
		now := time.Now()
		existing.TrustedAt = &now
		existing.ExpiresAt = &expiresAt
		existing.LastUsedAt = now
		existing.DeviceName = req.DeviceName
		database.DB.Save(&existing)
	} else {
		maxDevices := appConfig.MaxDevicePerUser
		if maxDevices == 0 {
			maxDevices = 10
		}

		var deviceCount int64
		database.DB.Model(&models.TrustedDevice{}).Where("user_id = ? AND is_trusted = ?", uid, true).Count(&deviceCount)

		if deviceCount >= int64(maxDevices) {
			var oldestDevice models.TrustedDevice
			database.DB.Where("user_id = ? AND is_trusted = ?", uid, true).Order("last_used_at ASC").First(&oldestDevice)
			if oldestDevice.ID > 0 {
				database.DB.Delete(&oldestDevice)
			}
		}

		trusted := models.TrustedDevice{
			UserID:            uid,
			DeviceFingerprint: req.DeviceFingerprint,
			DeviceName:        req.DeviceName,
			IsTrusted:         true,
			ExpiresAt:         &expiresAt,
		}
		now := time.Now()
		trusted.TrustedAt = &now
		trusted.LastUsedAt = now
		trusted.UseCount = 1

		database.DB.Create(&trusted)
	}

	response.Success(c, gin.H{
		"message":      "设备已信任",
		"expires_at":   expiresAt,
		"expiry_days":  expiryDays,
	})
}

type TrustDeviceRequest struct {
	DeviceFingerprint string `json:"device_fingerprint" binding:"required"`
	DeviceName        string `json:"device_name"`
	ApplicationID      uint   `json:"application_id"`
}

func GetTrustedDevices(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	uid := userID.(uint)
	var devices []models.TrustedDevice
	database.DB.Where("user_id = ? AND is_trusted = ?", uid, true).Find(&devices)

	result := make([]TrustedDeviceInfo, 0, len(devices))
	for _, d := range devices {
		info := TrustedDeviceInfo{
			ID:                 d.ID,
			DeviceName:         d.DeviceName,
			IsTrusted:          d.IsTrusted,
			TrustedAt:          d.TrustedAt,
			ExpiresAt:          d.ExpiresAt,
			LastUsedAt:         d.LastUsedAt,
			UseCount:           d.UseCount,
		}

		if d.ExpiresAt != nil {
			info.RemainingDays = int(time.Until(*d.ExpiresAt).Hours() / 24)
		}

		result = append(result, info)
	}

	response.Success(c, result)
}

type TrustedDeviceInfo struct {
	ID             uint       `json:"id"`
	DeviceName     string     `json:"device_name"`
	IsTrusted      bool       `json:"is_trusted"`
	TrustedAt      *time.Time `json:"trusted_at"`
	ExpiresAt      *time.Time `json:"expires_at"`
	LastUsedAt     time.Time  `json:"last_used_at"`
	UseCount       int        `json:"use_count"`
	RemainingDays  int        `json:"remaining_days"`
}

func RevokeTrustedDevice(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	deviceID := c.Param("id")
	uid := userID.(uint)

	var device models.TrustedDevice
	if err := database.DB.Where("id = ? AND user_id = ?", deviceID, uid).First(&device).Error; err != nil {
		response.NotFound(c, "设备不存在")
		return
	}

	device.IsTrusted = false
	device.ExpiresAt = nil
	database.DB.Save(&device)

	response.Success(c, gin.H{"message": "已撤销信任"})
}

func GetSeamlessConfig(c *gin.Context) {
	appID := c.Query("application_id")
	if appID == "" {
		response.BadRequest(c, "缺少 application_id 参数")
		return
	}

	var appConfig models.SeamlessConfig
	if err := database.DB.Where("application_id = ?", appID).First(&appConfig).Error; err != nil {
		response.Success(c, gin.H{
			"enabled":                    false,
			"challenge_threshold":        30,
			"block_threshold":            70,
			"trust_expiry_days":          30,
			"require_behavior_analysis":  true,
			"force_verification_threshold": 80,
			"enable_auto_trust":          true,
			"min_trust_level":            60,
			"max_device_per_user":         10,
		})
		return
	}

	response.Success(c, gin.H{
		"enabled":                     appConfig.Enabled,
		"challenge_threshold":          appConfig.ChallengeThreshold,
		"block_threshold":              appConfig.BlockThreshold,
		"trust_expiry_days":            appConfig.TrustExpiryDays,
		"require_behavior_analysis":    appConfig.RequireBehaviorAnalysis,
		"force_verification_threshold": appConfig.ForceVerificationThreshold,
		"enable_auto_trust":            appConfig.EnableAutoTrust,
		"min_trust_level":              appConfig.MinTrustLevel,
		"max_device_per_user":          appConfig.MaxDevicePerUser,
	})
}

func UpdateSeamlessConfig(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	var req SeamlessConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	uid := userID.(uint)

	var app models.Application
	if err := database.DB.Where("id = ? AND user_id = ?", req.ApplicationID, uid).First(&app).Error; err != nil {
		response.NotFound(c, "应用不存在")
		return
	}

	var config models.SeamlessConfig
	if err := database.DB.Where("application_id = ?", req.ApplicationID).First(&config).Error; err != nil {
		config = models.SeamlessConfig{
			ApplicationID: req.ApplicationID,
		}
	}

	config.Enabled = req.Enabled
	config.ChallengeThreshold = req.ChallengeThreshold
	config.BlockThreshold = req.BlockThreshold
	config.TrustExpiryDays = req.TrustExpiryDays
	config.RequireBehaviorAnalysis = req.RequireBehaviorAnalysis
	config.ForceVerificationThreshold = req.ForceVerificationThreshold
	config.EnableAutoTrust = req.EnableAutoTrust
	config.MinTrustLevel = req.MinTrustLevel
	config.MaxDevicePerUser = req.MaxDevicePerUser

	if config.ID == 0 {
		database.DB.Create(&config)
	} else {
		database.DB.Save(&config)
	}

	response.Success(c, gin.H{
		"message": "配置已更新",
		"config":  config,
	})
}

func CleanupExpiredTrusts() error {
	now := time.Now()
	return database.DB.Model(&models.TrustedDevice{}).
		Where("expires_at IS NOT NULL AND expires_at < ?", now).
		Update("is_trusted", false).Error
}

func GetSeamlessStats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	uid := userID.(uint)

	var totalDevices int64
	database.DB.Model(&models.TrustedDevice{}).Where("user_id = ?", uid).Count(&totalDevices)

	var activeDevices int64
	database.DB.Model(&models.TrustedDevice{}).Where("user_id = ? AND is_trusted = ? AND (expires_at IS NULL OR expires_at > ?)", uid, true, time.Now()).Count(&activeDevices)

	var expiringDevices int64
	database.DB.Model(&models.TrustedDevice{}).Where("user_id = ? AND is_trusted = ? AND expires_at > ? AND expires_at < ?",
		uid, true, time.Now(), time.Now().AddDate(0, 0, 7)).Count(&expiringDevices)

	var recentVerifications int64
	database.DB.Model(&models.SeamlessVerification{}).Where("user_id = ? AND created_at > ?", uid, time.Now().AddDate(0, 0, -7)).Count(&recentVerifications)

	var blockedCount int64
	database.DB.Model(&models.SeamlessVerification{}).Where("user_id = ? AND decision = ? AND created_at > ?", uid, "block", time.Now().AddDate(0, 0, -7)).Count(&blockedCount)

	response.Success(c, gin.H{
		"total_devices":          totalDevices,
		"active_devices":        activeDevices,
		"expiring_devices":      expiringDevices,
		"recent_verifications":  recentVerifications,
		"blocked_count":         blockedCount,
		"block_rate":            float64(blockedCount) / float64(recentVerifications) * 100,
	})
}

func ExportDeviceData(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	uid := userID.(uint)

	var devices []models.TrustedDevice
	database.DB.Where("user_id = ? AND is_trusted = ?", uid, true).Find(&devices)

	data := make([]map[string]interface{}, 0, len(devices))
	for _, d := range devices {
		data = append(data, map[string]interface{}{
			"device_fingerprint": d.DeviceFingerprint,
			"device_name":        d.DeviceName,
			"trusted_at":          d.TrustedAt,
			"expires_at":          d.ExpiresAt,
			"last_used_at":       d.LastUsedAt,
			"use_count":           d.UseCount,
		})
	}

	response.Success(c, gin.H{
		"devices": data,
		"exported_at": time.Now(),
	})
}

func ImportDeviceData(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	var req struct {
		Devices []map[string]interface{} `json:"devices" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	uid := userID.(uint)
	imported := 0
	skipped := 0

	for _, deviceData := range req.Devices {
		fingerprint, ok := deviceData["device_fingerprint"].(string)
		if !ok || fingerprint == "" {
			skipped++
			continue
		}

		var existing models.TrustedDevice
		if database.DB.Where("user_id = ? AND device_fingerprint = ?", uid, fingerprint).First(&existing).Error == nil {
			skipped++
			continue
		}

		deviceName := ""
		if name, ok := deviceData["device_name"].(string); ok {
			deviceName = name
		}

		expiresAt := time.Now().AddDate(0, 0, 30)
		if expStr, ok := deviceData["expires_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, expStr); err == nil {
				expiresAt = t
			}
		}

		device := models.TrustedDevice{
			UserID:            uid,
			DeviceFingerprint: fingerprint,
			DeviceName:        deviceName,
			IsTrusted:         true,
			ExpiresAt:         &expiresAt,
		}
		now := time.Now()
		device.TrustedAt = &now
		device.LastUsedAt = now
		device.UseCount = 1

		if err := database.DB.Create(&device).Error; err == nil {
			imported++
		} else {
			skipped++
		}
	}

	response.Success(c, gin.H{
		"imported": imported,
		"skipped":  skipped,
		"message":  "设备数据导入完成",
	})
}
