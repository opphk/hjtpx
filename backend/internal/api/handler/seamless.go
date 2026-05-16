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

// SeamlessVerifyRequest 无感验证请求
type SeamlessVerifyRequest struct {
	SessionID         string                 `json:"session_id" binding:"required"`
	DeviceFingerprint string                 `json:"device_fingerprint" binding:"required"`
	ApplicationID     uint                   `json:"application_id"`
	UserID            *uint                  `json:"user_id,omitempty"`
	BehaviorData      []BehaviorDataPoint    `json:"behavior_data,omitempty"`
	EnvironmentData   map[string]interface{} `json:"environment_data,omitempty"`
}

// SeamlessVerifyResponse 无感验证响应
type SeamlessVerifyResponse struct {
	Decision  string  `json:"decision"` // allow, challenge, block
	RiskScore float64 `json:"risk_score"`
	Reason    string  `json:"reason,omitempty"`
	Token     string  `json:"token,omitempty"` // 如果允许，返回验证token
}

// SeamlessVerify 无感验证
func SeamlessVerify(c *gin.Context) {
	startTime := time.Now()
	var req SeamlessVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	// 风险评分初始化
	riskScore := 0.0
	decision := "challenge"
	reason := "需要验证"

	// 检查是否是信任设备
	var trustedDevice models.TrustedDevice
	deviceTrusted := false
	if req.UserID != nil {
		database.DB.Where("user_id = ? AND device_fingerprint = ? AND is_trusted = ? AND (expires_at IS NULL OR expires_at > ?)",
			*req.UserID, req.DeviceFingerprint, true, time.Now()).First(&trustedDevice)
		if trustedDevice.ID > 0 {
			deviceTrusted = true
			// 更新最后使用时间
			trustedDevice.LastUsedAt = time.Now()
			trustedDevice.UseCount++
			database.DB.Save(&trustedDevice)
		}
	}

	// 获取应用配置
	var config models.SeamlessConfig
	configEnabled := false
	if req.ApplicationID > 0 {
		database.DB.Where("application_id = ?", req.ApplicationID).First(&config)
		if config.ID > 0 && config.Enabled {
			configEnabled = true
		}
	}

	// 计算风险分数（结合行为分析和环境检测）
	if len(req.BehaviorData) > 0 {
		// 调用行为分析服务
		if behaviorService != nil {
			bdList := make([]models.BehaviorData, 0, len(req.BehaviorData))
			for _, d := range req.BehaviorData {
				dataJSON, _ := json.Marshal(d)
				bdList = append(bdList, models.BehaviorData{
					Data:      string(dataJSON),
					DataType:  d.Event,
					Timestamp: time.UnixMilli(d.Timestamp),
				})
			}
			_, score, _ := behaviorService.VerifyWithBehaviorAnalysis(true, bdList)
			riskScore = score
		}
	}

	// 环境检测分析
	envRisk := analyzeEnvironmentData(req.EnvironmentData)
	if envRisk > riskScore {
		riskScore = envRisk
	}

	// 应用配置决定最终决策
	if configEnabled {
		if riskScore >= config.BlockThreshold {
			decision = "block"
			reason = "高风险设备"
		} else if riskScore < config.ChallengeThreshold {
			decision = "allow"
			reason = "低风险设备"
		}
	} else {
		// 默认配置
		if riskScore >= 70 {
			decision = "block"
			reason = "高风险设备"
		} else if riskScore < 30 && deviceTrusted {
			decision = "allow"
			reason = "信任设备"
		}
	}

	// 生成验证token
	token := ""
	if decision == "allow" {
		token = uuid.New().String()
		// 可以缓存token到Redis，设置过期时间
	}

	// 记录验证日志
	seamlessLog := models.SeamlessVerification{
		SessionID:         req.SessionID,
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
		Decision:  decision,
		RiskScore: riskScore,
		Reason:    reason,
		Token:     token,
	})
}

// TrustDeviceRequest 信任设备请求
type TrustDeviceRequest struct {
	DeviceFingerprint string `json:"device_fingerprint" binding:"required"`
	DeviceName        string `json:"device_name"`
}

// TrustDevice 信任设备
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

	// 检查是否已存在
	var existing models.TrustedDevice
	database.DB.Where("user_id = ? AND device_fingerprint = ?", userID.(uint), req.DeviceFingerprint).First(&existing)

	now := time.Now()
	if existing.ID > 0 {
		existing.IsTrusted = true
		existing.TrustedAt = &now
		existing.LastUsedAt = now
		database.DB.Save(&existing)
	} else {
		trusted := models.TrustedDevice{
			UserID:            userID.(uint),
			DeviceFingerprint: req.DeviceFingerprint,
			DeviceName:        req.DeviceName,
			IsTrusted:         true,
			TrustedAt:         &now,
			LastUsedAt:        now,
			UseCount:          1,
		}
		database.DB.Create(&trusted)
	}

	response.Success(c, gin.H{"message": "设备已信任"})
}

// GetTrustedDevices 获取信任设备列表
func GetTrustedDevices(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	var devices []models.TrustedDevice
	database.DB.Where("user_id = ?", userID.(uint)).Find(&devices)
	response.Success(c, devices)
}

// RevokeTrustedDevice 撤销信任设备
func RevokeTrustedDevice(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	deviceID := c.Param("id")
	var device models.TrustedDevice
	if err := database.DB.Where("id = ? AND user_id = ?", deviceID, userID.(uint)).First(&device).Error; err != nil {
		response.NotFound(c, "设备不存在")
		return
	}

	device.IsTrusted = false
	database.DB.Save(&device)
	response.Success(c, gin.H{"message": "已撤销信任"})
}
