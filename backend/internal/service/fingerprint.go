package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type DeviceFingerprintService struct{}

func NewDeviceFingerprintService() *DeviceFingerprintService {
	return &DeviceFingerprintService{}
}

type FingerprintData struct {
	UserAgent           string   `json:"user_agent"`
	ScreenWidth         int      `json:"screen_width"`
	ScreenHeight        int      `json:"screen_height"`
	ColorDepth          int      `json:"color_depth"`
	Timezone            string   `json:"timezone"`
	Language            string   `json:"language"`
	Platform            string   `json:"platform"`
	HardwareConcurrency int      `json:"hardware_concurrency"`
	DeviceMemory        int64    `json:"device_memory"`
	TouchPoints         int      `json:"touch_points"`
	WebGLVendor         string   `json:"webgl_vendor"`
	WebGLRenderer       string   `json:"webgl_renderer"`
	CanvasFingerprint   string   `json:"canvas_fingerprint"`
	AudioFingerprint    string   `json:"audio_fingerprint"`
	Fonts               []string `json:"fonts"`
	Plugins             []string `json:"plugins"`
	DoNotTrack          bool     `json:"do_not_track"`
	CookiesEnabled      bool     `json:"cookies_enabled"`
	LocalStorage        bool     `json:"local_storage"`
	SessionStorage      bool     `json:"session_storage"`
}

type FingerprintHash struct {
	UserAgentHash string `json:"user_agent_hash"`
	ScreenHash    string `json:"screen_hash"`
	BrowserHash   string `json:"browser_hash"`
	PlatformHash  string `json:"platform_hash"`
	CanvasHash    string `json:"canvas_hash"`
	WebGLHash     string `json:"webgl_hash"`
	AudioHash     string `json:"audio_hash"`
}

type RiskAssessment struct {
	Score         float64   `json:"score"`
	Level         string    `json:"level"`
	Factors       []string  `json:"factors"`
	IsNewDevice   bool      `json:"is_new_device"`
	IsSharedDevice bool     `json:"is_shared_device"`
	Similarity    float64   `json:"similarity"`
}

type CollectedFingerprint struct {
	FingerprintID uint   `json:"fingerprint_id"`
	Hash          string `json:"hash"`
	RiskLevel     string `json:"risk_level"`
}

type DeviceInfo struct {
	ID              uint      `json:"id"`
	Hash            string    `json:"hash"`
	UserAgent       string    `json:"user_agent"`
	ScreenInfo      string    `json:"screen_info"`
	BrowserInfo     string    `json:"browser_info"`
	PlatformInfo    string    `json:"platform_info"`
	FirstSeenAt     time.Time `json:"first_seen_at"`
	LastSeenAt      time.Time `json:"last_seen_at"`
	VisitCount      int       `json:"visit_count"`
	IsTrusted       bool      `json:"is_trusted"`
	RiskLevel       string    `json:"risk_level"`
}

type SimilarDevice struct {
	DeviceID       uint      `json:"device_id"`
	Similarity     float64   `json:"similarity"`
	FirstSeenAt    time.Time `json:"first_seen_at"`
	LastSeenAt     time.Time `json:"last_seen_at"`
	VisitCount     int       `json:"visit_count"`
}

var fingerprintWeights = map[string]float64{
	"user_agent":       1.5,
	"screen":           1.2,
	"browser":          1.3,
	"platform":         1.0,
	"canvas":           2.0,
	"webgl":            1.8,
	"audio":            1.5,
}

func (s *DeviceFingerprintService) CollectFingerprint(userID uint, data FingerprintData, ipAddress string) (*CollectedFingerprint, error) {
	ctx := context.Background()
	fingerprintHash := s.GenerateFingerprintHash(data)
	combinedHash := s.CombineHashes(fingerprintHash)
	
	var existing models.DeviceFingerprint
	err := database.DB.Where("fingerprint_hash = ? AND user_id = ?", combinedHash, userID).First(&existing).Error
	
	var device models.DeviceFingerprint
	if err == nil {
		device = existing
		device.LastSeenAt = time.Now()
		device.VisitCount++
		database.DB.Save(&device)
	} else {
		device = models.DeviceFingerprint{
			UserID:          userID,
			FingerprintHash: combinedHash,
			UserAgent:       data.UserAgent,
			ScreenInfo:      fmt.Sprintf("%dx%d", data.ScreenWidth, data.ScreenHeight),
			BrowserInfo:     fmt.Sprintf("%s|%s|%dbit", data.WebGLVendor, data.WebGLRenderer, data.ColorDepth),
			PlatformInfo:    fmt.Sprintf("%s|%s", data.Platform, data.Timezone),
			CanvasHash:      fingerprintHash.CanvasHash,
			WebGLHash:       fingerprintHash.WebGLHash,
			AudioHash:       fingerprintHash.AudioHash,
			FirstSeenAt:     time.Now(),
			LastSeenAt:      time.Now(),
			VisitCount:      1,
			IsTrusted:       false,
			RiskLevel:       "low",
		}
		database.DB.Create(&device)
	}
	
	riskLevel := s.AssessRisk(userID, device, data, ipAddress)
	device.RiskLevel = riskLevel
	database.DB.Save(&device)
	
	history := models.DeviceHistory{
		FingerprintID: device.ID,
		IPAddress:     ipAddress,
		LoginTime:     time.Now(),
		LoginSuccess:  true,
		UserAgent:     data.UserAgent,
	}
	database.DB.Create(&history)
	
	cacheKey := fmt.Sprintf("fingerprint:%d:%s", userID, combinedHash)
	if redis.Client != nil {
		redis.Client.Set(ctx, cacheKey, combinedHash, 24*time.Hour)
	}
	
	return &CollectedFingerprint{
		FingerprintID: device.ID,
		Hash:          combinedHash,
		RiskLevel:     riskLevel,
	}, nil
}

func (s *DeviceFingerprintService) GenerateFingerprintHash(data FingerprintData) FingerprintHash {
	userAgentHash := s.hashString(data.UserAgent)
	
	screenInfo := fmt.Sprintf("%dx%d_%d", data.ScreenWidth, data.ScreenHeight, data.ColorDepth)
	screenHash := s.hashString(screenInfo)
	
	browserInfo := fmt.Sprintf("%s|%s|%d|%d", data.WebGLVendor, data.WebGLRenderer, data.TouchPoints, data.HardwareConcurrency)
	browserHash := s.hashString(browserInfo)
	
	platformInfo := fmt.Sprintf("%s|%s|%s", data.Platform, data.Timezone, data.Language)
	platformHash := s.hashString(platformInfo)
	
	canvasHash := data.CanvasFingerprint
	if canvasHash == "" {
		canvasHash = s.hashString(fmt.Sprintf("%s|%s|%s", data.UserAgent, data.Platform, data.WebGLRenderer))
	}
	
	webglInfo := fmt.Sprintf("%s|%s", data.WebGLVendor, data.WebGLRenderer)
	webglHash := s.hashString(webglInfo)
	
	audioHash := data.AudioFingerprint
	if audioHash == "" {
		audioHash = s.hashString(fmt.Sprintf("%s|%s|%d", data.Platform, data.Language, data.DeviceMemory))
	}
	
	return FingerprintHash{
		UserAgentHash: userAgentHash,
		ScreenHash:    screenHash,
		BrowserHash:   browserHash,
		PlatformHash:  platformHash,
		CanvasHash:    canvasHash,
		WebGLHash:     webglHash,
		AudioHash:     audioHash,
	}
}

func (s *DeviceFingerprintService) CombineHashes(hashes FingerprintHash) string {
	combined := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s",
		hashes.UserAgentHash,
		hashes.ScreenHash,
		hashes.BrowserHash,
		hashes.PlatformHash,
		hashes.CanvasHash,
		hashes.WebGLHash,
		hashes.AudioHash,
	)
	return s.hashString(combined)
}

func (s *DeviceFingerprintService) hashString(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

func (s *DeviceFingerprintService) AssessRisk(userID uint, device models.DeviceFingerprint, data FingerprintData, ipAddress string) string {
	assessment := s.CalculateRiskScore(userID, device, data, ipAddress)
	return assessment.Level
}

func (s *DeviceFingerprintService) CalculateRiskScore(userID uint, device models.DeviceFingerprint, data FingerprintData, ipAddress string) RiskAssessment {
	assessment := RiskAssessment{
		Factors: []string{},
	}
	
	if device.ID == 0 || device.VisitCount == 0 {
		assessment.IsNewDevice = true
		assessment.Score += 30
		assessment.Factors = append(assessment.Factors, "新设备首次访问")
	}
	
	var userDevices []models.DeviceFingerprint
	database.DB.Where("user_id = ?", userID).Find(&userDevices)
	
	if len(userDevices) > 3 {
		assessment.IsSharedDevice = true
		assessment.Score += 20
		assessment.Factors = append(assessment.Factors, fmt.Sprintf("用户使用设备数量异常: %d", len(userDevices)))
	}
	
	var sharedCount int64
	database.DB.Model(&models.DeviceFingerprint{}).
		Where("fingerprint_hash = ? AND user_id != ?", device.FingerprintHash, userID).
		Count(&sharedCount)
	
	if sharedCount > 0 {
		assessment.Score += 25
		assessment.Factors = append(assessment.Factors, fmt.Sprintf("设备被多个账号使用: %d个账号", sharedCount+1))
	}
	
	if device.RiskLevel == "high" {
		assessment.Score += 30
		assessment.Factors = append(assessment.Factors, "历史高风险设备")
	}
	
	var recentLogins int64
	since := time.Now().Add(-24 * time.Hour)
	database.DB.Model(&models.DeviceHistory{}).
		Where("fingerprint_id = ? AND login_time > ?", device.ID, since).
		Count(&recentLogins)
	
	if recentLogins > 10 {
		assessment.Score += 15
		assessment.Factors = append(assessment.Factors, fmt.Sprintf("24小时内登录次数异常: %d次", recentLogins))
	}
	
	if data.DoNotTrack {
		assessment.Score += 10
		assessment.Factors = append(assessment.Factors, "启用DoNotTrack追踪")
	}
	
	if !data.CookiesEnabled {
		assessment.Score += 5
		assessment.Factors = append(assessment.Factors, "Cookie被禁用")
	}
	
	if !data.LocalStorage || !data.SessionStorage {
		assessment.Score += 5
		assessment.Factors = append(assessment.Factors, "本地存储异常")
	}
	
	if len(data.Fonts) < 3 {
		assessment.Score += 10
		assessment.Factors = append(assessment.Factors, "字体列表异常")
	}
	
	if len(data.Plugins) == 0 {
		assessment.Score += 5
		assessment.Factors = append(assessment.Factors, "无插件信息")
	}
	
	assessment.Score = math.Min(assessment.Score, 100)
	
	if assessment.Score < 30 {
		assessment.Level = "low"
	} else if assessment.Score < 60 {
		assessment.Level = "medium"
	} else {
		assessment.Level = "high"
	}
	
	return assessment
}

func (s *DeviceFingerprintService) VerifyFingerprint(userID uint, fingerprintID uint, providedHash string) (bool, RiskAssessment, []SimilarDevice, error) {
	var device models.DeviceFingerprint
	if err := database.DB.First(&device, fingerprintID).Error; err != nil {
		return false, RiskAssessment{}, nil, err
	}
	
	if device.UserID != userID {
		return false, RiskAssessment{}, nil, fmt.Errorf("设备不属于该用户")
	}
	
	similarity := s.CalculateSimilarity(device.FingerprintHash, providedHash)
	
	assessment := RiskAssessment{
		Similarity: similarity,
		IsNewDevice: false,
	}
	
	if similarity > 0.9 {
		assessment.Score = 0
		assessment.Level = "low"
	} else if similarity > 0.7 {
		assessment.Score = 30
		assessment.Level = "medium"
		assessment.Factors = append(assessment.Factors, "设备指纹部分匹配")
	} else {
		assessment.Score = 80
		assessment.Level = "high"
		assessment.Factors = append(assessment.Factors, "设备指纹不匹配")
	}
	
	similarDevices := s.FindSimilarDevices(userID, device.FingerprintHash)
	
	return similarity > 0.7, assessment, similarDevices, nil
}

func (s *DeviceFingerprintService) CalculateSimilarity(hash1, hash2 string) float64 {
	if len(hash1) != len(hash2) {
		return 0
	}
	
	matchingBits := 0
	totalBits := len(hash1) * 4
	
	for i := 0; i < len(hash1); i++ {
		if hash1[i] == hash2[i] {
			matchingBits += 4
		}
	}
	
	return float64(matchingBits) / float64(totalBits)
}

func (s *DeviceFingerprintService) FindSimilarDevices(userID uint, currentHash string) []SimilarDevice {
	var devices []models.DeviceFingerprint
	database.DB.Where("user_id = ?", userID).Find(&devices)
	
	var similarDevices []SimilarDevice
	
	for _, device := range devices {
		if device.FingerprintHash == currentHash {
			continue
		}
		
		similarity := s.CalculateSimilarity(currentHash, device.FingerprintHash)
		
		if similarity > 0.5 {
			similarDevices = append(similarDevices, SimilarDevice{
				DeviceID:    device.ID,
				Similarity:  similarity,
				FirstSeenAt: device.FirstSeenAt,
				LastSeenAt:  device.LastSeenAt,
				VisitCount:  device.VisitCount,
			})
		}
	}
	
	sort.Slice(similarDevices, func(i, j int) bool {
		return similarDevices[i].Similarity > similarDevices[j].Similarity
	})
	
	if len(similarDevices) > 5 {
		similarDevices = similarDevices[:5]
	}
	
	return similarDevices
}

func (s *DeviceFingerprintService) GetUserDevices(userID uint) ([]DeviceInfo, error) {
	var devices []models.DeviceFingerprint
	if err := database.DB.Where("user_id = ?", userID).Order("last_seen_at DESC").Find(&devices).Error; err != nil {
		return nil, err
	}
	
	var deviceInfos []DeviceInfo
	for _, device := range devices {
		deviceInfos = append(deviceInfos, DeviceInfo{
			ID:           device.ID,
			Hash:         device.FingerprintHash,
			UserAgent:    device.UserAgent,
			ScreenInfo:   device.ScreenInfo,
			BrowserInfo:  device.BrowserInfo,
			PlatformInfo: device.PlatformInfo,
			FirstSeenAt:  device.FirstSeenAt,
			LastSeenAt:   device.LastSeenAt,
			VisitCount:   device.VisitCount,
			IsTrusted:    device.IsTrusted,
			RiskLevel:    device.RiskLevel,
		})
	}
	
	return deviceInfos, nil
}

func (s *DeviceFingerprintService) TrustDevice(userID uint, deviceID uint) error {
	var device models.DeviceFingerprint
	if err := database.DB.First(&device, deviceID).Error; err != nil {
		return err
	}
	
	if device.UserID != userID {
		return fmt.Errorf("设备不属于该用户")
	}
	
	device.IsTrusted = true
	return database.DB.Save(&device).Error
}

func (s *DeviceFingerprintService) UntrustDevice(userID uint, deviceID uint) error {
	var device models.DeviceFingerprint
	if err := database.DB.First(&device, deviceID).Error; err != nil {
		return err
	}
	
	if device.UserID != userID {
		return fmt.Errorf("设备不属于该用户")
	}
	
	device.IsTrusted = false
	return database.DB.Save(&device).Error
}

func (s *DeviceFingerprintService) GetDeviceHistory(deviceID uint, limit int) ([]models.DeviceHistory, error) {
	var history []models.DeviceHistory
	err := database.DB.Where("fingerprint_id = ?", deviceID).
		Order("login_time DESC").
		Limit(limit).
		Find(&history).Error
	
	return history, err
}

func (s *DeviceFingerprintService) CleanupOldDevices(userID uint, keepTrusted bool, daysOld int) (int64, error) {
	var devices []models.DeviceFingerprint
	query := database.DB.Where("user_id = ? AND last_seen_at < ?", userID, time.Now().AddDate(0, 0, -daysOld))
	
	if keepTrusted {
		query = query.Where("is_trusted = ?", false)
	}
	
	if err := query.Find(&devices).Error; err != nil {
		return 0, err
	}
	
	var count int64 = int64(len(devices))
	
	if count > 0 {
		var deviceIDs []uint
		for _, device := range devices {
			deviceIDs = append(deviceIDs, device.ID)
		}
		
		database.DB.Where("fingerprint_id IN ?", deviceIDs).Delete(&models.DeviceHistory{})
		database.DB.Where("id IN ?", deviceIDs).Delete(&models.DeviceFingerprint{})
	}
	
	return count, nil
}

func (s *DeviceFingerprintService) ExportFingerprintData(userID uint) (string, error) {
	devices, err := s.GetUserDevices(userID)
	if err != nil {
		return "", err
	}
	
	var allHistory []map[string]interface{}
	for _, device := range devices {
		history, _ := s.GetDeviceHistory(device.ID, 100)
		for _, h := range history {
			allHistory = append(allHistory, map[string]interface{}{
				"device_id":   device.ID,
				"login_time":  h.LoginTime,
				"ip_address":  h.IPAddress,
				"location":   h.Location,
				"login_success": h.LoginSuccess,
			})
		}
	}
	
	data := map[string]interface{}{
		"user_id":       userID,
		"export_time":   time.Now(),
		"devices":       devices,
		"login_history": allHistory,
	}
	
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	
	return string(jsonData), nil
}

func (s *DeviceFingerprintService) AnonymizeFingerprintData(userID uint) error {
	return database.DB.Model(&models.DeviceFingerprint{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"user_agent":    "REDACTED",
			"screen_info":   "REDACTED",
			"browser_info":  "REDACTED",
			"platform_info": "REDACTED",
		}).Error
}

func (s *DeviceFingerprintService) CheckDeviceAnomalies(userID uint) ([]string, error) {
	var anomalies []string
	
	var devices []models.DeviceFingerprint
	database.DB.Where("user_id = ?", userID).Find(&devices)
	
	if len(devices) > 10 {
		anomalies = append(anomalies, fmt.Sprintf("设备数量过多: %d", len(devices)))
	}
	
	platforms := make(map[string]int)
	browsers := make(map[string]int)
	
	for _, device := range devices {
		if device.PlatformInfo != "" {
			platform := strings.Split(device.PlatformInfo, "|")[0]
			platforms[platform]++
		}
		if device.BrowserInfo != "" {
			browser := strings.Split(device.BrowserInfo, "|")[0]
			browsers[browser]++
		}
	}
	
	for platform, count := range platforms {
		if count > 5 {
			anomalies = append(anomalies, fmt.Sprintf("平台 %s 使用设备过多: %d", platform, count))
		}
	}
	
	for browser, count := range browsers {
		if count > 5 {
			anomalies = append(anomalies, fmt.Sprintf("浏览器 %s 使用设备过多: %d", browser, count))
		}
	}
	
	return anomalies, nil
}
