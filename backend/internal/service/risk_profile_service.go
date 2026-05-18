package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"gorm.io/gorm"
)

type ProfileType string

const (
	ProfileTypeDevice     ProfileType = "device"
	ProfileTypeIP         ProfileType = "ip"
	ProfileTypeBehavior   ProfileType = "behavior"
	ProfileTypeGeo        ProfileType = "geo"
	ProfileTypeUnified    ProfileType = "unified"
)

type DeviceProfile struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	Fingerprint       string    `json:"fingerprint" gorm:"uniqueIndex;size:128"`
	UserAgent         string    `json:"user_agent" gorm:"size:500"`
	ScreenResolution  string    `json:"screen_resolution" gorm:"size:20"`
	ColorDepth        int       `json:"color_depth"`
	Timezone          string    `json:"timezone" gorm:"size:50"`
	Language          string    `json:"language" gorm:"size:20"`
	Platform          string    `json:"platform" gorm:"size:50"`
	HardwareConcurrency int      `json:"hardware_concurrency"`
	DeviceMemory      float64   `json:"device_memory"`
	TouchPoints       int       `json:"touch_points"`
	CanvasFingerprint string    `json:"canvas_fingerprint" gorm:"size:128"`
	WebGLFingerprint  string    `json:"webgl_fingerprint" gorm:"size:128"`
	AudioFingerprint  string    `json:"audio_fingerprint" gorm:"size:128"`
	HasAdBlock        bool      `json:"has_ad_block"`
	HasFlash          bool      `json:"has_flash"`
	HasWebSocket      bool      `json:"has_web_socket"`
	IsTor             bool      `json:"is_tor"`
	IsVPN             bool      `json:"is_vpn"`
	IsProxy           bool      `json:"is_proxy"`
	IsHosting         bool      `json:"is_hosting"`
	IsMobile          bool      `json:"is_mobile"`
	IsBot             bool      `json:"is_bot"`
	RiskScore         float64   `json:"risk_score"`
	TrustLevel        int       `json:"trust_level"`
	RequestCount      int64     `json:"request_count"`
	BlockCount        int64     `json:"block_count"`
	FirstSeenAt       time.Time `json:"first_seen_at"`
	LastSeenAt        time.Time `json:"last_seen_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type IPProfile struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	IPAddress       string    `json:"ip_address" gorm:"uniqueIndex;size:50"`
	Country         string    `json:"country" gorm:"size:10"`
	Region          string    `json:"region" gorm:"size:100"`
	City            string    `json:"city" gorm:"size:100"`
	ISP             string    `json:"isp" gorm:"size:200"`
	ASN             int       `json:"asn"`
	IsProxy         bool      `json:"is_proxy"`
	IsVPN           bool      `json:"is_vpn"`
	IsTor           bool      `json:"is_tor"`
	IsHosting       bool      `json:"is_hosting"`
	IsDatacenter    bool      `json:"is_datacenter"`
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
	ThreatLevel     int       `json:"threat_level"`
	RiskScore       float64   `json:"risk_score"`
	RequestCount    int64     `json:"request_count"`
	BlockCount      int64     `json:"block_count"`
	UniqueDevices   int64     `json:"unique_devices"`
	CountryCode     string    `json:"country_code" gorm:"size:10"`
	PostalCode      string    `json:"postal_code" gorm:"size:20"`
	Timezone        string    `json:"timezone" gorm:"size:50"`
	FirstSeenAt     time.Time `json:"first_seen_at"`
	LastSeenAt      time.Time `json:"last_seen_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type BehaviorProfile struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	SessionID         string    `json:"session_id" gorm:"uniqueIndex;size:100"`
	Fingerprint       string    `json:"fingerprint" gorm:"index;size:128"`
	IPAddress         string    `json:"ip_address" gorm:"index;size:50"`
	MouseSpeed        float64   `json:"mouse_speed"`
	MouseAcceleration float64   `json:"mouse_acceleration"`
	ClickFrequency    float64   `json:"click_frequency"`
	ScrollSpeed       float64   `json:"scroll_speed"`
	KeyboardSpeed     float64   `json:"keyboard_speed"`
	PathEfficiency    float64   `json:"path_efficiency"`
	Straightness      float64   `json:"straightness"`
	TotalClicks       int       `json:"total_clicks"`
	TotalMoves        int       `json:"total_moves"`
	AvgPauseDuration  float64   `json:"avg_pause_duration"`
	IdleTime          int64     `json:"idle_time"`
	TrajectoryPoints  int       `json:"trajectory_points"`
	IsHuman           bool      `json:"is_human"`
	Confidence        float64   `json:"confidence"`
	AnomalyScore      float64   `json:"anomaly_score"`
	RiskScore         float64   `json:"risk_score"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type GeoProfile struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	Fingerprint       string    `json:"fingerprint" gorm:"index;size:128"`
	IPAddress         string    `json:"ip_address" gorm:"index;size:50"`
	CurrentCountry    string    `json:"current_country" gorm:"size:10"`
	CurrentRegion     string    `json:"current_region" gorm:"size:100"`
	CurrentCity       string    `json:"current_city" gorm:"size:100"`
	CurrentLatitude   float64   `json:"current_latitude"`
	CurrentLongitude  float64   `json:"current_longitude"`
	LastCountry       string    `json:"last_country" gorm:"size:10"`
	LastRegion        string    `json:"last_region" gorm:"size:100"`
	LastCity          string    `json:"last_city" gorm:"size:100"`
	LastLatitude      float64   `json:"last_latitude"`
	LastLongitude     float64   `json:"last_longitude"`
	TravelDistance    float64   `json:"travel_distance"`
	TravelSpeed       float64   `json:"travel_speed"`
	TimeSinceLastSeen int64     `json:"time_since_last_seen"`
	CountryChanges    int       `json:"country_changes"`
	RegionChanges     int       `json:"region_changes"`
	IsImpossibleTravel bool     `json:"is_impossible_travel"`
	RiskScore         float64   `json:"risk_score"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type UnifiedRiskProfile struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	Fingerprint       string    `json:"fingerprint" gorm:"uniqueIndex;size:128"`
	IPAddress         string    `json:"ip_address" gorm:"index;size:50"`
	DeviceRiskScore   float64   `json:"device_risk_score"`
	IPRiskScore       float64   `json:"ip_risk_score"`
	BehaviorScore     float64   `json:"behavior_score"`
	GeoScore          float64   `json:"geo_score"`
	HistoricalScore   float64   `json:"historical_score"`
	TimeScore         float64   `json:"time_score"`
	SessionScore      float64   `json:"session_score"`
	OverallRiskScore  float64   `json:"overall_risk_score"`
	RiskLevel         string    `json:"risk_level" gorm:"size:20"`
	TrustLevel        int       `json:"trust_level"`
	RiskFactors       string    `json:"risk_factors" gorm:"type:text"`
	LastRiskAction    string    `json:"last_risk_action" gorm:"size:50"`
	LastRiskEventAt   *time.Time `json:"last_risk_event_at"`
	RequestCount      int64     `json:"request_count"`
	SuccessCount      int64     `json:"success_count"`
	BlockCount        int64     `json:"block_count"`
	FailCount         int64     `json:"fail_count"`
	UniqueIPs         int64     `json:"unique_ips"`
	UniqueSessions    int64     `json:"unique_sessions"`
	FirstSeenAt       time.Time `json:"first_seen_at"`
	LastSeenAt        time.Time `json:"last_seen_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type RiskProfileService struct {
	mu    sync.RWMutex
}

var profileServiceInstance *RiskProfileService
var profileServiceOnce sync.Once

func NewRiskProfileService() *RiskProfileService {
	profileServiceOnce.Do(func() {
		profileServiceInstance = &RiskProfileService{}
	})
	return profileServiceInstance
}

func (s *RiskProfileService) CreateOrUpdateDeviceProfile(ctx context.Context, fingerprint string, deviceData map[string]interface{}) (*DeviceProfile, error) {
	var profile DeviceProfile
	err := database.DB.Where("fingerprint = ?", fingerprint).First(&profile).Error

	now := time.Now()
	isNew := err == gorm.ErrRecordNotFound

	if isNew {
		profile = DeviceProfile{
			Fingerprint:  fingerprint,
			FirstSeenAt:  now,
			TrustLevel:   50,
			RiskScore:   100.0,
		}
	}

	if ua, ok := deviceData["user_agent"].(string); ok {
		profile.UserAgent = ua
	}
	if res, ok := deviceData["screen_resolution"].(string); ok {
		profile.ScreenResolution = res
	}
	if cd, ok := deviceData["color_depth"].(int); ok {
		profile.ColorDepth = cd
	}
	if tz, ok := deviceData["timezone"].(string); ok {
		profile.Timezone = tz
	}
	if lang, ok := deviceData["language"].(string); ok {
		profile.Language = lang
	}
	if plat, ok := deviceData["platform"].(string); ok {
		profile.Platform = plat
	}
	if hc, ok := deviceData["hardware_concurrency"].(int); ok {
		profile.HardwareConcurrency = hc
	}
	if dm, ok := deviceData["device_memory"].(float64); ok {
		profile.DeviceMemory = dm
	}
	if tp, ok := deviceData["touch_points"].(int); ok {
		profile.TouchPoints = tp
	}
	if canvas, ok := deviceData["canvas_fingerprint"].(string); ok {
		profile.CanvasFingerprint = canvas
	}
	if webgl, ok := deviceData["webgl_fingerprint"].(string); ok {
		profile.WebGLFingerprint = webgl
	}
	if audio, ok := deviceData["audio_fingerprint"].(string); ok {
		profile.AudioFingerprint = audio
	}
	if ab, ok := deviceData["has_ad_block"].(bool); ok {
		profile.HasAdBlock = ab
	}
	if flash, ok := deviceData["has_flash"].(bool); ok {
		profile.HasFlash = flash
	}
	if ws, ok := deviceData["has_web_socket"].(bool); ok {
		profile.HasWebSocket = ws
	}
	if isMobile, ok := deviceData["is_mobile"].(bool); ok {
		profile.IsMobile = isMobile
	}

	profile.LastSeenAt = now
	profile.RequestCount++

	s.analyzeDeviceRisk(&profile)

	if isNew {
		err = database.DB.Create(&profile).Error
	} else {
		err = database.DB.Save(&profile).Error
	}

	if err == nil {
		s.cacheDeviceProfile(ctx, &profile)
	}

	return &profile, err
}

func (s *RiskProfileService) analyzeDeviceRisk(profile *DeviceProfile) {
	riskScore := 100.0

	if profile.HasAdBlock {
		riskScore -= 5
	}
	if profile.HasFlash {
		riskScore -= 10
	}
	if profile.TouchPoints == 0 && !profile.IsMobile {
		riskScore -= 15
	}
	if profile.HardwareConcurrency > 8 {
		riskScore -= 5
	}
	if profile.DeviceMemory < 2 {
		riskScore -= 10
	}

	if strings.Contains(strings.ToLower(profile.UserAgent), "headless") {
		riskScore -= 30
		profile.IsBot = true
	}
	if strings.Contains(strings.ToLower(profile.UserAgent), "phantom") {
		riskScore -= 35
		profile.IsBot = true
	}
	if strings.Contains(strings.ToLower(profile.UserAgent), "selenium") {
		riskScore -= 25
		profile.IsBot = true
	}

	profile.RiskScore = riskScore

	if riskScore >= 80 {
		profile.TrustLevel = 100
	} else if riskScore >= 60 {
		profile.TrustLevel = 75
	} else if riskScore >= 40 {
		profile.TrustLevel = 50
	} else if riskScore >= 20 {
		profile.TrustLevel = 25
	} else {
		profile.TrustLevel = 0
	}
}

func (s *RiskProfileService) CreateOrUpdateIPProfile(ctx context.Context, ipAddress string, ipData map[string]interface{}) (*IPProfile, error) {
	var profile IPProfile
	err := database.DB.Where("ip_address = ?", ipAddress).First(&profile).Error

	now := time.Now()
	isNew := err == gorm.ErrRecordNotFound

	if isNew {
		profile = IPProfile{
			IPAddress:   ipAddress,
			FirstSeenAt: now,
			ThreatLevel: 0,
			RiskScore:  100.0,
		}
	}

	if country, ok := ipData["country"].(string); ok {
		profile.Country = country
	}
	if region, ok := ipData["region"].(string); ok {
		profile.Region = region
	}
	if city, ok := ipData["city"].(string); ok {
		profile.City = city
	}
	if isp, ok := ipData["isp"].(string); ok {
		profile.ISP = isp
	}
	if asn, ok := ipData["asn"].(int); ok {
		profile.ASN = asn
	}
	if lat, ok := ipData["latitude"].(float64); ok {
		profile.Latitude = lat
	}
	if lng, ok := ipData["longitude"].(float64); ok {
		profile.Longitude = lng
	}
	if countryCode, ok := ipData["country_code"].(string); ok {
		profile.CountryCode = countryCode
	}

	profile.LastSeenAt = now
	profile.RequestCount++

	s.analyzeIPRisk(ctx, &profile)

	if isNew {
		err = database.DB.Create(&profile).Error
	} else {
		err = database.DB.Save(&profile).Error
	}

	if err == nil {
		s.cacheIPProfile(ctx, &profile)
	}

	return &profile, err
}

func (s *RiskProfileService) analyzeIPRisk(ctx context.Context, profile *IPProfile) {
	riskScore := 100.0

	var recentBlocks int64
	database.DB.Model(&struct {
		tableName struct{} `gorm:"table:risk_events"`
	}{}).Where("ip_address = ? AND created_at > ?", profile.IPAddress, time.Now().Add(-24*time.Hour)).Count(&recentBlocks)

	riskScore -= float64(recentBlocks) * 5

	if profile.IsProxy || profile.IsVPN || profile.IsTor {
		riskScore -= 15
	}
	if profile.IsHosting || profile.IsDatacenter {
		riskScore -= 10
	}

	suspiciousCountries := map[string]bool{"NG": true, "PK": true, "BD": true}
	if suspiciousCountries[profile.CountryCode] {
		riskScore -= 5
	}

	if profile.ThreatLevel >= 3 {
		riskScore -= 20
	}

	profile.RiskScore = mathMax(0, riskScore)

	redis.GetClient().Set(ctx, fmt.Sprintf("ip:risk:%s", profile.IPAddress), profile.RiskScore, 10*time.Minute)
}

func (s *RiskProfileService) CreateOrUpdateBehaviorProfile(sessionID string, behaviorData map[string]interface{}) (*BehaviorProfile, error) {
	var profile BehaviorProfile
	err := database.DB.Where("session_id = ?", sessionID).First(&profile).Error

	isNew := err == gorm.ErrRecordNotFound

	if isNew {
		profile = BehaviorProfile{
			SessionID:    sessionID,
			RiskScore:    100.0,
			Confidence:   0.5,
		}
	}

	if fp, ok := behaviorData["fingerprint"].(string); ok {
		profile.Fingerprint = fp
	}
	if ip, ok := behaviorData["ip_address"].(string); ok {
		profile.IPAddress = ip
	}
	if ms, ok := behaviorData["mouse_speed"].(float64); ok {
		profile.MouseSpeed = ms
	}
	if ma, ok := behaviorData["mouse_acceleration"].(float64); ok {
		profile.MouseAcceleration = ma
	}
	if cf, ok := behaviorData["click_frequency"].(float64); ok {
		profile.ClickFrequency = cf
	}
	if ss, ok := behaviorData["scroll_speed"].(float64); ok {
		profile.ScrollSpeed = ss
	}
	if ks, ok := behaviorData["keyboard_speed"].(float64); ok {
		profile.KeyboardSpeed = ks
	}
	if pe, ok := behaviorData["path_efficiency"].(float64); ok {
		profile.PathEfficiency = pe
	}
	if st, ok := behaviorData["straightness"].(float64); ok {
		profile.Straightness = st
	}
	if tc, ok := behaviorData["total_clicks"].(int); ok {
		profile.TotalClicks = tc
	}
	if tm, ok := behaviorData["total_moves"].(int); ok {
		profile.TotalMoves = tm
	}
	if apd, ok := behaviorData["avg_pause_duration"].(float64); ok {
		profile.AvgPauseDuration = apd
	}
	if it, ok := behaviorData["idle_time"].(int64); ok {
		profile.IdleTime = it
	}
	if tp, ok := behaviorData["trajectory_points"].(int); ok {
		profile.TrajectoryPoints = tp
	}

	s.analyzeBehaviorRisk(&profile)

	if isNew {
		err = database.DB.Create(&profile).Error
	} else {
		err = database.DB.Save(&profile).Error
	}

	return &profile, err
}

func (s *RiskProfileService) analyzeBehaviorRisk(profile *BehaviorProfile) {
	riskScore := 100.0

	if profile.MouseSpeed > 2000 {
		riskScore -= 20
	}
	if profile.PathEfficiency > 0.95 {
		riskScore -= 25
	}
	if profile.Straightness > 0.9 {
		riskScore -= 15
	}
	if profile.ClickFrequency > 10 {
		riskScore -= 10
	}
	if profile.TrajectoryPoints < 5 && profile.TotalMoves > 20 {
		riskScore -= 20
	}
	if profile.AvgPauseDuration < 50 && profile.TotalClicks > 3 {
		riskScore -= 15
	}

	profile.RiskScore = mathMax(0, riskScore)

	profile.Confidence = 1.0 - (profile.RiskScore / 100.0)
	profile.IsHuman = profile.RiskScore >= 50 && profile.Confidence >= 0.5

	profile.AnomalyScore = 100.0 - profile.RiskScore
}

func (s *RiskProfileService) CreateOrUpdateGeoProfile(fingerprint string, ipAddress string, geoData map[string]interface{}) (*GeoProfile, error) {
	var profile GeoProfile
	err := database.DB.Where("fingerprint = ?", fingerprint).Order("created_at DESC").First(&profile).Error

	isNew := err == gorm.ErrRecordNotFound || profile.ID == 0

	if isNew {
		profile = GeoProfile{
			Fingerprint: fingerprint,
			IPAddress:   ipAddress,
			RiskScore:  100.0,
		}
	}

	if cc, ok := geoData["current_country"].(string); ok {
		if profile.CurrentCountry != "" && profile.CurrentCountry != cc {
			profile.CountryChanges++
		}
		profile.LastCountry = profile.CurrentCountry
		profile.CurrentCountry = cc
	}
	if cr, ok := geoData["current_region"].(string); ok {
		if profile.CurrentRegion != "" && profile.CurrentRegion != cr {
			profile.RegionChanges++
		}
		profile.LastRegion = profile.CurrentRegion
		profile.CurrentRegion = cr
	}
	if ct, ok := geoData["current_city"].(string); ok {
		profile.LastCity = profile.CurrentCity
		profile.CurrentCity = ct
	}
	if lat, ok := geoData["current_latitude"].(float64); ok {
		profile.LastLatitude = profile.CurrentLatitude
		profile.CurrentLatitude = lat
	}
	if lng, ok := geoData["current_longitude"].(float64); ok {
		profile.LastLongitude = profile.CurrentLongitude
		profile.CurrentLongitude = lng
	}

	s.analyzeGeoRisk(&profile)

	if isNew {
		err = database.DB.Create(&profile).Error
	} else {
		err = database.DB.Save(&profile).Error
	}

	return &profile, err
}

func (s *RiskProfileService) analyzeGeoRisk(profile *GeoProfile) {
	riskScore := 100.0

	if profile.TravelDistance > 1000 {
		timeSince := time.Since(profile.UpdatedAt).Hours()
		requiredSpeed := profile.TravelDistance / mathMax(1, timeSince)
		if requiredSpeed > 800 {
			riskScore -= 40
			profile.IsImpossibleTravel = true
		} else {
			riskScore -= 20
			profile.TravelSpeed = requiredSpeed
		}
	}

	if profile.CountryChanges > 2 {
		riskScore -= 30
	}
	if profile.RegionChanges > 5 {
		riskScore -= 20
	}

	profile.RiskScore = mathMax(0, riskScore)
}

func (s *RiskProfileService) CreateOrUpdateUnifiedProfile(ctx context.Context, fingerprint string, ipAddress string) (*UnifiedRiskProfile, error) {
	var profile UnifiedRiskProfile
	err := database.DB.Where("fingerprint = ?", fingerprint).First(&profile).Error

	now := time.Now()
	isNew := err == gorm.ErrRecordNotFound

	if isNew {
		profile = UnifiedRiskProfile{
			Fingerprint:     fingerprint,
			IPAddress:       ipAddress,
			RiskLevel:      "low",
			TrustLevel:     50,
			FirstSeenAt:    now,
		}
	}

	s.calculateUnifiedRiskScore(ctx, &profile)

	profile.LastSeenAt = now
	profile.RequestCount++

	if isNew {
		err = database.DB.Create(&profile).Error
	} else {
		err = database.DB.Save(&profile).Error
	}

	if err == nil {
		s.cacheUnifiedProfile(ctx, &profile)
	}

	return &profile, err
}

func (s *RiskProfileService) calculateUnifiedRiskScore(ctx context.Context, profile *UnifiedRiskProfile) {
	var deviceProfile DeviceProfile
	if err := database.DB.Where("fingerprint = ?", profile.Fingerprint).First(&deviceProfile).Error; err == nil {
		profile.DeviceRiskScore = deviceProfile.RiskScore
	}

	var ipProfile IPProfile
	if err := database.DB.Where("ip_address = ?", profile.IPAddress).First(&ipProfile).Error; err == nil {
		profile.IPRiskScore = ipProfile.RiskScore
	}

	var geoProfile GeoProfile
	if err := database.DB.Where("fingerprint = ?", profile.Fingerprint).Order("created_at DESC").First(&geoProfile).Error; err == nil {
		profile.GeoScore = 100.0 - geoProfile.RiskScore
	}

	profile.HistoricalScore = 100.0
	if profile.BlockCount > 0 {
		profile.HistoricalScore -= float64(profile.BlockCount) * 10
	}

	hour := time.Now().Hour()
	if hour < 2 || hour > 23 {
		profile.TimeScore = 70
	} else {
		profile.TimeScore = 100
	}

	profile.SessionScore = 100.0
	if profile.FailCount > 0 {
		profile.SessionScore -= float64(profile.FailCount) * 15
	}

	weights := map[string]float64{
		"device":     0.15,
		"ip":         0.20,
		"behavior":   0.25,
		"geo":        0.10,
		"historical": 0.20,
		"time":       0.05,
		"session":    0.05,
	}

	profile.OverallRiskScore = (
		profile.DeviceRiskScore * weights["device"] +
		profile.IPRiskScore * weights["ip"] +
		profile.BehaviorScore * weights["behavior"] +
		profile.GeoScore * weights["geo"] +
		profile.HistoricalScore * weights["historical"] +
		profile.TimeScore * weights["time"] +
		profile.SessionScore * weights["session"])


	switch {
	case profile.OverallRiskScore >= 80:
		profile.RiskLevel = "low"
		profile.TrustLevel = 100
	case profile.OverallRiskScore >= 60:
		profile.RiskLevel = "medium"
		profile.TrustLevel = 75
	case profile.OverallRiskScore >= 40:
		profile.RiskLevel = "high"
		profile.TrustLevel = 50
	default:
		profile.RiskLevel = "critical"
		profile.TrustLevel = 0
	}
}

func (s *RiskProfileService) GetDeviceProfile(fingerprint string) (*DeviceProfile, error) {
	var profile DeviceProfile
	err := database.DB.Where("fingerprint = ?", fingerprint).First(&profile).Error
	return &profile, err
}

func (s *RiskProfileService) GetIPProfile(ipAddress string) (*IPProfile, error) {
	var profile IPProfile
	err := database.DB.Where("ip_address = ?", ipAddress).First(&profile).Error
	return &profile, err
}

func (s *RiskProfileService) GetBehaviorProfile(sessionID string) (*BehaviorProfile, error) {
	var profile BehaviorProfile
	err := database.DB.Where("session_id = ?", sessionID).First(&profile).Error
	return &profile, err
}

func (s *RiskProfileService) GetGeoProfile(fingerprint string) (*GeoProfile, error) {
	var profile GeoProfile
	err := database.DB.Where("fingerprint = ?", fingerprint).Order("created_at DESC").First(&profile).Error
	return &profile, err
}

func (s *RiskProfileService) GetUnifiedProfile(fingerprint string) (*UnifiedRiskProfile, error) {
	var profile UnifiedRiskProfile
	err := database.DB.Where("fingerprint = ?", fingerprint).First(&profile).Error
	return &profile, err
}

func (s *RiskProfileService) cacheDeviceProfile(ctx context.Context, profile *DeviceProfile) {
	key := fmt.Sprintf("profile:device:%s", profile.Fingerprint)
	data, _ := json.Marshal(profile)
	redis.GetClient().Set(ctx, key, data, 30*time.Minute)
}

func (s *RiskProfileService) cacheIPProfile(ctx context.Context, profile *IPProfile) {
	key := fmt.Sprintf("profile:ip:%s", profile.IPAddress)
	data, _ := json.Marshal(profile)
	redis.GetClient().Set(ctx, key, data, 15*time.Minute)
}

func (s *RiskProfileService) cacheUnifiedProfile(ctx context.Context, profile *UnifiedRiskProfile) {
	key := fmt.Sprintf("profile:unified:%s", profile.Fingerprint)
	data, _ := json.Marshal(profile)
	redis.GetClient().Set(ctx, key, data, 10*time.Minute)
}

func (s *RiskProfileService) GetCachedDeviceProfile(ctx context.Context, fingerprint string) (*DeviceProfile, error) {
	key := fmt.Sprintf("profile:device:%s", fingerprint)
	data, err := redis.GetClient().Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var profile DeviceProfile
	if err := json.Unmarshal([]byte(data), &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *RiskProfileService) GetCachedIPProfile(ctx context.Context, ipAddress string) (*IPProfile, error) {
	key := fmt.Sprintf("profile:ip:%s", ipAddress)
	data, err := redis.GetClient().Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var profile IPProfile
	if err := json.Unmarshal([]byte(data), &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *RiskProfileService) RecordRiskEvent(ctx context.Context, fingerprint string, ipAddress string, eventType string, action string, riskScore float64) error {
	event := struct {
		Fingerprint string    `json:"fingerprint"`
		IPAddress   string    `json:"ip_address"`
		EventType   string    `json:"event_type"`
		Action      string    `json:"action"`
		RiskScore   float64   `json:"risk_score"`
		Timestamp   time.Time `json:"timestamp"`
	}{
		Fingerprint: fingerprint,
		IPAddress:   ipAddress,
		EventType:   eventType,
		Action:      action,
		RiskScore:   riskScore,
		Timestamp:   time.Now(),
	}

	eventJSON, _ := json.Marshal(event)

	redis.GetClient().LPush(ctx, fmt.Sprintf("device:%s:history", fingerprint), eventJSON)
	redis.GetClient().LTrim(ctx, fmt.Sprintf("device:%s:history", fingerprint), 0, 999)

	redis.GetClient().LPush(ctx, fmt.Sprintf("ip:%s:history", ipAddress), eventJSON)
	redis.GetClient().LTrim(ctx, fmt.Sprintf("ip:%s:history", ipAddress), 0, 999)

	return nil
}

func (s *RiskProfileService) GetDeviceHistory(ctx context.Context, fingerprint string, limit int64) ([]map[string]interface{}, error) {
	key := fmt.Sprintf("device:%s:history", fingerprint)
	events, err := redis.GetClient().LRange(ctx, key, 0, limit-1).Result()
	if err != nil {
		return nil, err
	}

	var history []map[string]interface{}
	for _, eventStr := range events {
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(eventStr), &event); err == nil {
			history = append(history, event)
		}
	}
	return history, nil
}

func (s *RiskProfileService) GetIPHistory(ctx context.Context, ipAddress string, limit int64) ([]map[string]interface{}, error) {
	key := fmt.Sprintf("ip:%s:history", ipAddress)
	events, err := redis.GetClient().LRange(ctx, key, 0, limit-1).Result()
	if err != nil {
		return nil, err
	}

	var history []map[string]interface{}
	for _, eventStr := range events {
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(eventStr), &event); err == nil {
			history = append(history, event)
		}
	}
	return history, nil
}

func mathMax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
