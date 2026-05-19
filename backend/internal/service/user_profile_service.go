package service

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

type UserProfileService struct{}

func NewUserProfileService() *UserProfileService {
	return &UserProfileService{}
}

type UserProfile struct {
	UserID        uint                 `json:"user_id"`
	Username      string               `json:"username"`
	Email         string               `json:"email"`
	CreatedAt     time.Time            `json:"created_at"`
	ProfileData   UserProfileData      `json:"profile_data"`
	RiskProfile   RiskProfile          `json:"risk_profile"`
	BehaviorProfile BehaviorProfile    `json:"behavior_profile"`
	DeviceProfile DeviceProfile        `json:"device_profile"`
	ActivityTimeline []ActivityPoint  `json:"activity_timeline"`
}

type UserProfileData struct {
	TotalSessions      int64              `json:"total_sessions"`
	TotalCaptchas      int64              `json:"total_captchas"`
	SuccessRate        float64            `json:"success_rate"`
	AvgSolveTime       float64            `json:"avg_solve_time"`
	TotalInteractions  int64              `json:"total_interactions"`
	FirstActivity      *time.Time         `json:"first_activity"`
	LastActivity       *time.Time         `json:"last_activity"`
	ActivityDays       int                `json:"activity_days"`
	TrustLevel         string             `json:"trust_level"`
	Tags               []string           `json:"tags"`
}

type RiskProfile struct {
	RiskScore        float64            `json:"risk_score"`
	RiskLevel         string             `json:"risk_level"`
	RiskFactors       []RiskFactor       `json:"risk_factors"`
	ThreatHistory     []ThreatEvent      `json:"threat_history"`
	Recommendations   []string          `json:"recommendations"`
}

type RiskFactor struct {
	Factor     string  `json:"factor"`
	Score      float64 `json:"score"`
	Weight     float64 `json:"weight"`
	Severity   string  `json:"severity"`
}

type ThreatEvent struct {
	Type        string     `json:"type"`
	Severity    string     `json:"severity"`
	Timestamp   time.Time  `json:"timestamp"`
	Description string     `json:"description"`
}

type BehaviorProfile struct {
	BehavioralPatterns []Pattern        `json:"behavioral_patterns"`
	HabitScore         float64           `json:"habit_score"`
	ConsistencyScore   float64           `json:"consistency_score"`
	TypicalHours       []int             `json:"typical_hours"`
	TypicalDays        []int             `json:"typical_days"`
	GeoDistribution    map[string]int    `json:"geo_distribution"`
}

type Pattern struct {
	Type        string   `json:"type"`
	Frequency   int      `json:"frequency"`
	Confidence  float64  `json:"confidence"`
	Description string   `json:"description"`
}

type DeviceProfile struct {
	PrimaryDevice   DeviceInfo         `json:"primary_device"`
	AllDevices      []DeviceInfo       `json:"all_devices"`
	BrowserProfile  BrowserProfile     `json:"browser_profile"`
	NetworkProfile  NetworkProfile     `json:"network_profile"`
}

type DeviceInfo struct {
	DeviceID      string    `json:"device_id"`
	DeviceType   string    `json:"device_type"`
	OS           string     `json:"os"`
	Browser      string     `json:"browser"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
	UsageCount   int64     `json:"usage_count"`
	IsTrusted    bool      `json:"is_trusted"`
}

type BrowserProfile struct {
	PrimaryBrowser   string             `json:"primary_browser"`
	BrowserVersions []BrowserVersion   `json:"browser_versions"`
	CanvasSupport   bool               `json:"canvas_support"`
	WebGLSupport    bool               `json:"webgl_support"`
	PluginCount     int                `json:"plugin_count"`
}

type BrowserVersion struct {
	Browser   string `json:"browser"`
	Version   string `json:"version"`
	Count     int64  `json:"count"`
}

type NetworkProfile struct {
	IPAddresses     []IPInfo        `json:"ip_addresses"`
	ISPInfo        string           `json:"isp_info"`
	ProxyUsage     float64          `json:"proxy_usage"`
	VPNUsage       float64          `json:"vpn_usage"`
	AvgLatency     float64          `json:"avg_latency"`
}

type IPInfo struct {
	IPAddress     string    `json:"ip_address"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
	Count        int64     `json:"count"`
	Country      string    `json:"country"`
	City         string    `json:"city"`
}

type ActivityPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	EventType   string    `json:"event_type"`
	Description string    `json:"description"`
	RiskLevel   string    `json:"risk_level"`
}

func (s *UserProfileService) GenerateUserProfile(userID uint) (*UserProfile, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	profile := &UserProfile{
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}

	profile.ProfileData = s.analyzeProfileData(userID)
	profile.RiskProfile = s.analyzeRiskProfile(userID)
	profile.BehaviorProfile = s.analyzeBehaviorProfile(userID)
	profile.DeviceProfile = s.analyzeDeviceProfile(userID)
	profile.ActivityTimeline = s.generateActivityTimeline(userID)

	return profile, nil
}

func (s *UserProfileService) analyzeProfileData(userID uint) UserProfileData {
	data := UserProfileData{
		Tags: make([]string, 0),
	}

	var sessions []models.Session
	database.DB.Where("user_id = ?", userID).Find(&sessions)
	data.TotalSessions = int64(len(sessions))

	var captchas []models.CaptchaRecord
	database.DB.Where("user_id = ?", userID).Find(&captchas)
	data.TotalCaptchas = int64(len(captchas))

	if data.TotalCaptchas > 0 {
		var successCount int64
		database.DB.Model(&models.CaptchaRecord{}).
			Where("user_id = ? AND status = ?", userID, "success").
			Count(&successCount)
		data.SuccessRate = float64(successCount) / float64(data.TotalCaptchas) * 100
	}

	if len(captchas) > 0 {
		var totalTime float64
		for _, c := range captchas {
			if c.SolveTime > 0 {
				totalTime += float64(c.SolveTime)
			}
		}
		data.AvgSolveTime = totalTime / float64(len(captchas))
	}

	data.TotalInteractions = data.TotalSessions + data.TotalCaptchas

	var firstSession, lastSession models.Session
	if err := database.DB.Where("user_id = ?", userID).Order("created_at ASC").First(&firstSession).Error; err == nil {
		data.FirstActivity = &firstSession.CreatedAt
	}
	if err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").First(&lastSession).Error; err == nil {
		data.LastActivity = &lastSession.CreatedAt
	}

	if data.FirstActivity != nil && data.LastActivity != nil {
		data.ActivityDays = int(data.LastActivity.Sub(*data.FirstActivity).Hours()/24) + 1
	}

	data.TrustLevel = s.calculateTrustLevel(data)

	if data.SuccessRate > 90 && data.TotalCaptchas > 10 {
		data.Tags = append(data.Tags, "高可信用户")
	}
	if data.TotalCaptchas > 100 {
		data.Tags = append(data.Tags, "活跃用户")
	}
	if data.SuccessRate < 50 && data.TotalCaptchas > 5 {
		data.Tags = append(data.Tags, "需关注")
	}

	return data
}

func (s *UserProfileService) calculateTrustLevel(data UserProfileData) string {
	score := 0

	if data.SuccessRate >= 95 {
		score += 40
	} else if data.SuccessRate >= 85 {
		score += 30
	} else if data.SuccessRate >= 70 {
		score += 20
	} else {
		score += 10
	}

	if data.TotalCaptchas >= 100 {
		score += 30
	} else if data.TotalCaptchas >= 50 {
		score += 20
	} else if data.TotalCaptchas >= 10 {
		score += 10
	}

	if data.ActivityDays >= 30 {
		score += 20
	} else if data.ActivityDays >= 7 {
		score += 10
	}

	if score >= 80 {
		return "非常高"
	} else if score >= 60 {
		return "高"
	} else if score >= 40 {
		return "中"
	} else if score >= 20 {
		return "低"
	}
	return "非常低"
}

func (s *UserProfileService) analyzeRiskProfile(userID uint) RiskProfile {
	profile := RiskProfile{
		RiskFactors:     make([]RiskFactor, 0),
		ThreatHistory:   make([]ThreatEvent, 0),
		Recommendations: make([]string, 0),
	}

	var failedCaptchas int64
	database.DB.Model(&models.CaptchaRecord{}).
		Where("user_id = ? AND status = ?", userID, "failed").
		Count(&failedCaptchas)

	var totalCaptchas int64
	database.DB.Model(&models.CaptchaRecord{}).
		Where("user_id = ?", userID).
		Count(&totalCaptchas)

	if totalCaptchas > 0 {
		failRate := float64(failedCaptchas) / float64(totalCaptchas)
		
		if failRate > 0.5 {
			profile.RiskFactors = append(profile.RiskFactors, RiskFactor{
				Factor:   "高失败率",
				Score:    failRate * 100,
				Weight:   0.4,
				Severity: "high",
			})
		} else if failRate > 0.3 {
			profile.RiskFactors = append(profile.RiskFactors, RiskFactor{
				Factor:   "较高失败率",
				Score:    failRate * 100,
				Weight:   0.3,
				Severity: "medium",
			})
		}
	}

	var suspiciousLogs []models.Log
	database.DB.Where("user_id = ? AND level = ?", userID, "warning").
		Order("created_at DESC").
		Limit(5).
		Find(&suspiciousLogs)

	if len(suspiciousLogs) > 0 {
		profile.RiskFactors = append(profile.RiskFactors, RiskFactor{
			Factor:   "可疑日志",
			Score:    float64(len(suspiciousLogs)) * 10,
			Weight:   0.2,
			Severity: "medium",
		})
	}

	totalScore := 0.0
	for _, rf := range profile.RiskFactors {
		totalScore += rf.Score * rf.Weight
	}
	profile.RiskScore = math.Min(totalScore, 100)

	if profile.RiskScore >= 70 {
		profile.RiskLevel = "极高"
		profile.Recommendations = append(profile.Recommendations, "建议立即审核此用户")
	} else if profile.RiskScore >= 50 {
		profile.RiskLevel = "高"
		profile.Recommendations = append(profile.Recommendations, "建议增加验证强度")
	} else if profile.RiskScore >= 30 {
		profile.RiskLevel = "中"
		profile.Recommendations = append(profile.Recommendations, "建议持续监控")
	} else {
		profile.RiskLevel = "低"
		profile.Recommendations = append(profile.Recommendations, "用户行为正常")
	}

	return profile
}

func (s *UserProfileService) analyzeBehaviorProfile(userID uint) BehaviorProfile {
	profile := BehaviorProfile{
		BehavioralPatterns: make([]Pattern, 0),
		GeoDistribution:    make(map[string]int),
		TypicalHours:       make([]int, 0),
		TypicalDays:        make([]int, 0),
	}

	var sessions []models.Session
	database.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(100).
		Find(&sessions)

	hourCounts := make(map[int]int)
	dayCounts := make(map[int]int)

	for _, session := range sessions {
		hourCounts[session.CreatedAt.Hour()]++
		dayCounts[int(session.CreatedAt.Weekday())]++
	}

	for hour, count := range hourCounts {
		if count >= len(sessions)/5 {
			profile.TypicalHours = append(profile.TypicalHours, hour)
		}
	}

	for day, count := range dayCounts {
		if count >= len(sessions)/7 {
			profile.TypicalDays = append(profile.TypicalDays, day)
		}
	}

	if len(profile.TypicalHours) >= 2 {
		profile.HabitScore = 70.0
	}
	if len(profile.TypicalDays) >= 4 {
		profile.HabitScore += 15.0
	}
	if len(sessions) >= 20 {
		profile.HabitScore += 15.0
	}

	if len(sessions) > 0 {
		profile.ConsistencyScore = 100.0 - (float64(len(profile.TypicalHours)) / 24.0 * 100)
	}

	profile.BehavioralPatterns = append(profile.BehavioralPatterns, Pattern{
		Type:        "登录时段",
		Frequency:  len(profile.TypicalHours),
		Confidence: profile.HabitScore,
		Description: fmt.Sprintf("用户通常在 %d 个不同时间段活跃", len(profile.TypicalHours)),
	})

	return profile
}

func (s *UserProfileService) analyzeDeviceProfile(userID uint) DeviceProfile {
	profile := DeviceProfile{
		AllDevices: make([]DeviceInfo, 0),
	}

	var sessions []models.Session
	database.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&sessions)

	deviceCounts := make(map[string]*DeviceInfo)
	
	for _, session := range sessions {
		deviceID := session.DeviceFingerprint
		if deviceID == "" {
			deviceID = "unknown"
		}

		if _, exists := deviceCounts[deviceID]; !exists {
			deviceCounts[deviceID] = &DeviceInfo{
				DeviceID:    deviceID,
				DeviceType:  s.inferDeviceType(session.UserAgent),
				OS:          s.inferOS(session.UserAgent),
				Browser:     s.inferBrowser(session.UserAgent),
				FirstSeen:   session.CreatedAt,
				LastSeen:    session.CreatedAt,
				UsageCount:  0,
			}
		}

		deviceCounts[deviceID].UsageCount++
		if session.CreatedAt.Before(deviceCounts[deviceID].FirstSeen) {
			deviceCounts[deviceID].FirstSeen = session.CreatedAt
		}
		if session.CreatedAt.After(deviceCounts[deviceID].LastSeen) {
			deviceCounts[deviceID].LastSeen = session.CreatedAt
		}
	}

	for _, device := range deviceCounts {
		device.IsTrusted = device.UsageCount >= 5
		profile.AllDevices = append(profile.AllDevices, *device)
	}

	if len(profile.AllDevices) > 0 {
		maxUsage := int64(0)
		for _, device := range profile.AllDevices {
			if device.UsageCount > maxUsage {
				maxUsage = device.UsageCount
				profile.PrimaryDevice = device
			}
		}
	}

	return profile
}

func (s *UserProfileService) inferDeviceType(userAgent string) string {
	if userAgent == "" {
		return "Unknown"
	}
	if contains(userAgent, "Mobile") || contains(userAgent, "Android") || contains(userAgent, "iPhone") {
		return "Mobile"
	}
	if contains(userAgent, "Tablet") || contains(userAgent, "iPad") {
		return "Tablet"
	}
	return "Desktop"
}

func (s *UserProfileService) inferOS(userAgent string) string {
	if contains(userAgent, "Windows") {
		return "Windows"
	}
	if contains(userAgent, "Mac") || contains(userAgent, "OS X") {
		return "macOS"
	}
	if contains(userAgent, "Linux") {
		return "Linux"
	}
	if contains(userAgent, "Android") {
		return "Android"
	}
	if contains(userAgent, "iOS") || contains(userAgent, "iPhone") || contains(userAgent, "iPad") {
		return "iOS"
	}
	return "Unknown"
}

func (s *UserProfileService) inferBrowser(userAgent string) string {
	if contains(userAgent, "Chrome") && !contains(userAgent, "Edg") {
		return "Chrome"
	}
	if contains(userAgent, "Firefox") {
		return "Firefox"
	}
	if contains(userAgent, "Safari") && !contains(userAgent, "Chrome") {
		return "Safari"
	}
	if contains(userAgent, "Edg") {
		return "Edge"
	}
	if contains(userAgent, "Opera") || contains(userAgent, "OPR") {
		return "Opera"
	}
	return "Unknown"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (s *UserProfileService) generateActivityTimeline(userID uint) []ActivityPoint {
	timeline := make([]ActivityPoint, 0)

	var sessions []models.Session
	database.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(50).
		Find(&sessions)

	for _, session := range sessions {
		timeline = append(timeline, ActivityPoint{
			Timestamp:   session.CreatedAt,
			EventType:   "session",
			Description: fmt.Sprintf("新会话 (IP: %s)", session.IPAddress),
			RiskLevel:   "low",
		})
	}

	var captchas []models.CaptchaRecord
	database.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(20).
		Find(&captchas)

	for _, captcha := range captchas {
		riskLevel := "low"
		if captcha.Status == "failed" {
			riskLevel = "medium"
		} else if captcha.Status == "blocked" {
			riskLevel = "high"
		}

		timeline = append(timeline, ActivityPoint{
			Timestamp:   captcha.CreatedAt,
			EventType:   "captcha",
			Description: fmt.Sprintf("验证码 %s", captcha.Status),
			RiskLevel:   riskLevel,
		})
	}

	return timeline
}

func (s *UserProfileService) GetUserProfileSummary(userID uint) (map[string]interface{}, error) {
	profile, err := s.GenerateUserProfile(userID)
	if err != nil {
		return nil, err
	}

	summary := map[string]interface{}{
		"user_id":       profile.UserID,
		"username":      profile.Username,
		"trust_level":   profile.ProfileData.TrustLevel,
		"risk_level":    profile.RiskProfile.RiskLevel,
		"risk_score":    profile.RiskProfile.RiskScore,
		"success_rate":  profile.ProfileData.SuccessRate,
		"total_sessions": profile.ProfileData.TotalSessions,
		"total_captchas": profile.ProfileData.TotalCaptchas,
		"device_count":  len(profile.DeviceProfile.AllDevices),
		"tags":          profile.ProfileData.Tags,
	}

	return summary, nil
}

func (s *UserProfileService) ExportUserProfile(userID uint, format string) ([]byte, error) {
	profile, err := s.GenerateUserProfile(userID)
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return json.MarshalIndent(profile, "", "  ")
	case "csv":
		return s.exportToCSV(profile)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func (s *UserProfileService) exportToCSV(profile *UserProfile) ([]byte, error) {
	csv := "Field,Value\n"
	csv += fmt.Sprintf("User ID,%d\n", profile.UserID)
	csv += fmt.Sprintf("Username,%s\n", profile.Username)
	csv += fmt.Sprintf("Email,%s\n", profile.Email)
	csv += fmt.Sprintf("Trust Level,%s\n", profile.ProfileData.TrustLevel)
	csv += fmt.Sprintf("Risk Level,%s\n", profile.RiskProfile.RiskLevel)
	csv += fmt.Sprintf("Risk Score,%.2f\n", profile.RiskProfile.RiskScore)
	csv += fmt.Sprintf("Success Rate,%.2f%%\n", profile.ProfileData.SuccessRate)
	csv += fmt.Sprintf("Total Sessions,%d\n", profile.ProfileData.TotalSessions)
	csv += fmt.Sprintf("Total Captchas,%d\n", profile.ProfileData.TotalCaptchas)
	csv += fmt.Sprintf("Device Count,%d\n", len(profile.DeviceProfile.AllDevices))
	
	return []byte(csv), nil
}

type UserProfileListResult struct {
	Data       []UserProfileSummary `json:"data"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
}

type UserProfileSummary struct {
	UserID        uint     `json:"user_id"`
	Username      string   `json:"username"`
	TrustLevel    string   `json:"trust_level"`
	RiskLevel     string   `json:"risk_level"`
	RiskScore     float64  `json:"risk_score"`
	SuccessRate   float64  `json:"success_rate"`
	TotalSessions int64    `json:"total_sessions"`
	Tags          []string `json:"tags"`
}

func (s *UserProfileService) ListUserProfiles(page, pageSize int, trustLevel, riskLevel string) (*UserProfileListResult, error) {
	var users []models.User
	var total int64

	query := database.DB.Model(&models.User{})
	
	if trustLevel != "" {
		query = query.Where("id IN (?)", s.getUsersByTrustLevel(trustLevel))
	}
	
	if riskLevel != "" {
		query = query.Where("id IN (?)", s.getUsersByRiskLevel(riskLevel))
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, err
	}

	summaries := make([]UserProfileSummary, 0, len(users))
	for _, user := range users {
		profile, err := s.GenerateUserProfile(user.ID)
		if err != nil {
			continue
		}
		
		summaries = append(summaries, UserProfileSummary{
			UserID:        profile.UserID,
			Username:      profile.Username,
			TrustLevel:    profile.ProfileData.TrustLevel,
			RiskLevel:     profile.RiskProfile.RiskLevel,
			RiskScore:     profile.RiskProfile.RiskScore,
			SuccessRate:   profile.ProfileData.SuccessRate,
			TotalSessions: profile.ProfileData.TotalSessions,
			Tags:          profile.ProfileData.Tags,
		})
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &UserProfileListResult{
		Data:       summaries,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *UserProfileService) getUsersByTrustLevel(level string) []uint {
	var userIDs []uint
	
	var sessions []models.Session
	database.DB.Select("DISTINCT user_id").Find(&sessions)
	
	userIDs = make([]uint, 0)
	for _, session := range sessions {
		if session.UserID > 0 {
			userIDs = append(userIDs, session.UserID)
		}
	}
	
	return userIDs
}

func (s *UserProfileService) getUsersByRiskLevel(level string) []uint {
	var userIDs []uint
	
	var sessions []models.Session
	database.DB.Select("DISTINCT user_id").Find(&sessions)
	
	userIDs = make([]uint, 0)
	for _, session := range sessions {
		if session.UserID > 0 {
			userIDs = append(userIDs, session.UserID)
		}
	}
	
	return userIDs
}

type ProfileComparison struct {
	User1        UserProfileSummary `json:"user1"`
	User2        UserProfileSummary `json:"user2"`
	Similarities []string          `json:"similarities"`
	Differences  []string          `json:"differences"`
	Recommendations []string       `json:"recommendations"`
}

func (s *UserProfileService) CompareUserProfiles(userID1, userID2 uint) (*ProfileComparison, error) {
	profile1, err := s.GenerateUserProfile(userID1)
	if err != nil {
		return nil, fmt.Errorf("failed to generate profile for user %d: %w", userID1, err)
	}

	profile2, err := s.GenerateUserProfile(userID2)
	if err != nil {
		return nil, fmt.Errorf("failed to generate profile for user %d: %w", userID2, err)
	}

	comparison := &ProfileComparison{
		User1: UserProfileSummary{
			UserID:        profile1.UserID,
			Username:      profile1.Username,
			TrustLevel:    profile1.ProfileData.TrustLevel,
			RiskLevel:     profile1.RiskProfile.RiskLevel,
			RiskScore:     profile1.RiskProfile.RiskScore,
			SuccessRate:   profile1.ProfileData.SuccessRate,
			TotalSessions: profile1.ProfileData.TotalSessions,
			Tags:          profile1.ProfileData.Tags,
		},
		User2: UserProfileSummary{
			UserID:        profile2.UserID,
			Username:      profile2.Username,
			TrustLevel:    profile2.ProfileData.TrustLevel,
			RiskLevel:     profile2.RiskProfile.RiskLevel,
			RiskScore:     profile2.RiskProfile.RiskScore,
			SuccessRate:   profile2.ProfileData.SuccessRate,
			TotalSessions: profile2.ProfileData.TotalSessions,
			Tags:          profile2.ProfileData.Tags,
		},
		Similarities:      make([]string, 0),
		Differences:       make([]string, 0),
		Recommendations:   make([]string, 0),
	}

	if profile1.ProfileData.TrustLevel == profile2.ProfileData.TrustLevel {
		comparison.Similarities = append(comparison.Similarities, fmt.Sprintf("信任等级相同: %s", profile1.ProfileData.TrustLevel))
	} else {
		comparison.Differences = append(comparison.Differences, fmt.Sprintf("信任等级不同: %s vs %s", profile1.ProfileData.TrustLevel, profile2.ProfileData.TrustLevel))
	}

	if profile1.RiskProfile.RiskLevel == profile2.RiskProfile.RiskLevel {
		comparison.Similarities = append(comparison.Similarities, fmt.Sprintf("风险等级相同: %s", profile1.RiskProfile.RiskLevel))
	} else {
		comparison.Differences = append(comparison.Differences, fmt.Sprintf("风险等级不同: %s vs %s", profile1.RiskProfile.RiskLevel, profile2.RiskProfile.RiskLevel))
	}

	rateDiff := math.Abs(profile1.ProfileData.SuccessRate - profile2.ProfileData.SuccessRate)
	if rateDiff < 5 {
		comparison.Similarities = append(comparison.Similarities, fmt.Sprintf("成功率相近 (差异 %.2f%%)", rateDiff))
	} else {
		comparison.Differences = append(comparison.Differences, fmt.Sprintf("成功率差异较大 (差异 %.2f%%)", rateDiff))
	}

	if len(profile1.DeviceProfile.AllDevices) == len(profile2.DeviceProfile.AllDevices) {
		comparison.Similarities = append(comparison.Similarities, fmt.Sprintf("使用设备数量相同: %d", len(profile1.DeviceProfile.AllDevices)))
	} else {
		comparison.Differences = append(comparison.Differences, fmt.Sprintf("使用设备数量不同: %d vs %d", len(profile1.DeviceProfile.AllDevices), len(profile2.DeviceProfile.AllDevices)))
	}

	if rateDiff < 5 && profile1.RiskProfile.RiskLevel == profile2.RiskProfile.RiskLevel {
		comparison.Recommendations = append(comparison.Recommendations, "两个用户画像相似，可以考虑将相同策略应用于两者")
	}

	if profile1.RiskProfile.RiskScore > profile2.RiskProfile.RiskScore*1.5 {
		comparison.Recommendations = append(comparison.Recommendations, fmt.Sprintf("用户1风险更高(%.2f)，建议对用户1实施更严格的验证策略", profile1.RiskProfile.RiskScore))
	}

	return comparison, nil
}
