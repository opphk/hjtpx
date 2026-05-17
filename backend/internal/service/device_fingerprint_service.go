package service

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

type DeviceFingerprintService struct {
	db            *gorm.DB
	cache         map[string]*CachedFingerprint
	cacheMu       sync.RWMutex
	ipCache       map[string]*models.IPReputation
	ipCacheMu     sync.RWMutex
	knownPatterns map[string]bool
	patternMu     sync.RWMutex
}

type CachedFingerprint struct {
	Fingerprint *models.DeviceFingerprintRecord
	LastAccess  time.Time
}

type BrowserFeatures struct {
	CanvasHash     string   `json:"canvas_hash"`
	WebGLHash     string   `json:"webgl_hash"`
	AudioHash     string   `json:"audio_hash"`
	FontHash      string   `json:"font_hash"`
	ScreenHash    string   `json:"screen_hash"`
	Timezone      string   `json:"timezone"`
	Language      string   `json:"language"`
	Platform      string   `json:"platform"`
	ScreenWidth   int      `json:"screen_width"`
	ScreenHeight  int      `json:"screen_height"`
	ColorDepth    int      `json:"color_depth"`
	DeviceMemory  int      `json:"device_memory"`
	HardwareConcurrency int `json:"hardware_concurrency"`
	MaxTouchPoints int     `json:"max_touch_points"`
	WebGLVendor   string   `json:"webgl_vendor"`
	WebGLRenderer string   `json:"webgl_renderer"`
	Fonts         []string `json:"fonts"`
	Plugins       []string `json:"plugins"`
	MimeTypes     []string `json:"mime_types"`
}

type FingerprintRequest struct {
	Fingerprint   string         `json:"fingerprint"`
	Features      BrowserFeatures `json:"features"`
	UserID        *uint          `json:"user_id,omitempty"`
	ApplicationID *uint          `json:"application_id,omitempty"`
	SessionID     string         `json:"session_id"`
	IPAddress     string         `json:"ip_address"`
	UserAgent     string         `json:"user_agent"`
	Referrer      string         `json:"referrer"`
}

type FingerprintResult struct {
	Fingerprint   string               `json:"fingerprint"`
	RiskScore     float64              `json:"risk_score"`
	RiskLevel     string               `json:"risk_level"`
	IsBot         bool                 `json:"is_bot"`
	IsProxy       bool                 `json:"is_proxy"`
	IsVPN         bool                 `json:"is_vpn"`
	IsTor         bool                 `json:"is_tor"`
	IsTrusted     bool                 `json:"is_trusted"`
	TrustLevel    models.TrustLevel     `json:"trust_level"`
	Factors       []RiskFactor          `json:"factors"`
	IPReputation  *models.IPReputation  `json:"ip_reputation,omitempty"`
	FirstSeen     time.Time            `json:"first_seen"`
	LastSeen      time.Time            `json:"last_seen"`
	VisitCount    int                  `json:"visit_count"`
	IsNewDevice   bool                 `json:"is_new_device"`
}

type RiskFactor struct {
	Name     string  `json:"name"`
	Weight   float64 `json:"weight"`
	Score    float64 `json:"score"`
	Critical bool    `json:"critical"`
}

func NewDeviceFingerprintService(db *gorm.DB) *DeviceFingerprintService {
	svc := &DeviceFingerprintService{
		db:            db,
		cache:         make(map[string]*CachedFingerprint),
		ipCache:       make(map[string]*models.IPReputation),
		knownPatterns: make(map[string]bool),
	}
	go svc.cleanupCache()
	return svc
}

func (s *DeviceFingerprintService) cleanupCache() {
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

func (s *DeviceFingerprintService) GenerateFingerprint(req *FingerprintRequest) (*models.DeviceFingerprintRecord, error) {
	hash := s.computeFingerprintHash(req)
	
	var record models.DeviceFingerprintRecord
	err := s.db.Where("fingerprint = ?", hash).First(&record).Error
	if err == nil {
		record.LastSeenAt = time.Now()
		record.VisitCount++
		s.db.Save(&record)
		return &record, nil
	}

	record = models.DeviceFingerprintRecord{
		Fingerprint:     hash,
		UserID:          req.UserID,
		ApplicationID:   req.ApplicationID,
		CanvasHash:      req.Features.CanvasHash,
		WebGLHash:       req.Features.WebGLHash,
		AudioHash:       req.Features.AudioHash,
		FontHash:        req.Features.FontHash,
		ScreenHash:      req.Features.ScreenHash,
		Timezone:        req.Features.Timezone,
		Language:        req.Features.Language,
		Platform:        req.Features.Platform,
		UserAgent:       req.UserAgent,
		IPAddress:       req.IPAddress,
		FirstSeenAt:     time.Now(),
		LastSeenAt:      time.Now(),
		VisitCount:      1,
		RiskScore:       0,
	}

	if req.Features.ScreenWidth > 0 && req.Features.ScreenHeight > 0 {
		screenInfo := fmt.Sprintf("%dx%d", req.Features.ScreenWidth, req.Features.ScreenHeight)
		record.ScreenHash = screenInfo
	}

	record.RiskScore = s.calculateRiskScore(&record, req)

	if record.RiskScore >= 70 {
		record.IsBot = true
	} else if record.RiskScore >= 50 {
		record.TrustLevel = "high"
	} else if record.RiskScore >= 30 {
		record.TrustLevel = "medium"
	} else {
		record.TrustLevel = "low"
	}

	err = s.db.Create(&record).Error
	return &record, err
}

func (s *DeviceFingerprintService) computeFingerprintHash(req *FingerprintRequest) string {
	hasher := sha256.New()
	
	hasher.Write([]byte(req.Features.CanvasHash))
	hasher.Write([]byte(req.Features.WebGLHash))
	hasher.Write([]byte(req.Features.AudioHash))
	hasher.Write([]byte(req.Features.FontHash))
	hasher.Write([]byte(req.Features.ScreenHash))
	hasher.Write([]byte(req.Features.Timezone))
	hasher.Write([]byte(req.Features.Language))
	hasher.Write([]byte(req.Features.Platform))
	
	if req.Features.WebGLVendor != "" {
		hasher.Write([]byte(req.Features.WebGLVendor))
		hasher.Write([]byte(req.Features.WebGLRenderer))
	}
	
	fontsCombined := strings.Join(req.Features.Fonts, ",")
	hasher.Write([]byte(fontsCombined))
	
	hasher.Write([]byte(req.IPAddress))
	hasher.Write([]byte(req.UserAgent))
	
	return hex.EncodeToString(hasher.Sum(nil))[:32]
}

func (s *DeviceFingerprintService) calculateRiskScore(fp *models.DeviceFingerprintRecord, req *FingerprintRequest) float64 {
	var score float64

	if fp.CanvasHash == "" || fp.CanvasHash == "no_canvas" {
		score += 15
	}

	if fp.WebGLHash == "" || fp.WebGLHash == "no_webgl" {
		score += 10
	}

	if fp.FontHash == "" || len(req.Features.Fonts) < 5 {
		score += 10
	}

	if s.isSoftwareRenderer(fp.WebGLHash) {
		score += 25
	}

	if s.isAutomationDetected(req) {
		score += 35
	}

	if s.isProxyDetected(req.IPAddress, req.UserAgent) {
		score += 20
	}

	if s.isVPNDetected(req.IPAddress) {
		score += 15
	}

	if s.isTorExitNode(req.IPAddress) {
		score += 30
	}

	if fp.VisitCount < 3 {
		score += 10
	}

	return minFloat(score, 100)
}

func (s *DeviceFingerprintService) isSoftwareRenderer(renderer string) bool {
	if renderer == "" {
		return false
	}
	softwareIndicators := []string{
		"swiftshader", "llvmpipe", "mesa", "virtualbox",
		"vmware", "parallels", "microsoft basic",
	}
	rendererLower := strings.ToLower(renderer)
	for _, indicator := range softwareIndicators {
		if strings.Contains(rendererLower, indicator) {
			return true
		}
	}
	return false
}

func (s *DeviceFingerprintService) isAutomationDetected(req *FingerprintRequest) bool {
	uaLower := strings.ToLower(req.UserAgent)
	automationIndicators := []string{
		"headless", "phantom", "puppeteer", "playwright",
		"selenium", "webdriver", "chromium", "electron",
	}
	for _, indicator := range automationIndicators {
		if strings.Contains(uaLower, indicator) {
			return true
		}
	}
	return false
}

func (s *DeviceFingerprintService) isProxyDetected(ipAddress, userAgent string) bool {
	if ipAddress == "" {
		return false
	}

	ip := net.ParseIP(ipAddress)
	if ip != nil {
		if ip.IsPrivate() || ip.IsLoopback() {
			return false
		}
	}

	proxyHeaders := []string{
		"X-Forwarded-For", "X-Real-IP", "Via", "Proxy-Agent",
		"X-ProxyID", "Forwarded", "Client-IP",
	}

	for _, header := range proxyHeaders {
		if strings.Contains(userAgent, header) {
			return true
		}
	}

	return false
}

func (s *DeviceFingerprintService) isVPNDetected(ipAddress string) bool {
	return false
}

func (s *DeviceFingerprintService) isTorExitNode(ipAddress string) bool {
	return false
}

func (s *DeviceFingerprintService) GetFingerprint(fingerprint string) (*models.DeviceFingerprintRecord, error) {
	s.cacheMu.RLock()
	if cached, exists := s.cache[fingerprint]; exists {
		cached.LastAccess = time.Now()
		s.cacheMu.RUnlock()
		return cached.Fingerprint, nil
	}
	s.cacheMu.RUnlock()

	var record models.DeviceFingerprintRecord
	err := s.db.Where("fingerprint = ?", fingerprint).First(&record).Error
	if err != nil {
		return nil, err
	}

	s.cacheMu.Lock()
	s.cache[fingerprint] = &CachedFingerprint{
		Fingerprint: &record,
		LastAccess:  time.Now(),
	}
	s.cacheMu.Unlock()

	return &record, nil
}

func (s *DeviceFingerprintService) AnalyzeFingerprint(fingerprint string) (*FingerprintResult, error) {
	record, err := s.GetFingerprint(fingerprint)
	if err != nil {
		return nil, err
	}

	var ipReputation *models.IPReputation
	s.ipCacheMu.RLock()
	ipReputation, _ = s.ipCache[record.IPAddress]
	s.ipCacheMu.RUnlock()

	if ipReputation == nil {
		var ipRec models.IPReputation
		err := s.db.Where("ip_address = ?", record.IPAddress).First(&ipRec).Error
		if err == nil {
			ipReputation = &ipRec
			s.ipCacheMu.Lock()
			s.ipCache[record.IPAddress] = &ipRec
			s.ipCacheMu.Unlock()
		}
	}

	result := &FingerprintResult{
		Fingerprint:  record.Fingerprint,
		RiskScore:    record.RiskScore,
		RiskLevel:    record.TrustLevel,
		IsBot:        record.IsBot,
		IsProxy:      record.ProxyDetected,
		IsVPN:        record.VPNDetected,
		IsTor:        record.TorDetected,
		IsTrusted:    record.IsTrusted,
		TrustLevel:   models.TrustLevel(record.TrustLevel),
		FirstSeen:    record.FirstSeenAt,
		LastSeen:     record.LastSeenAt,
		VisitCount:   record.VisitCount,
		IsNewDevice:  record.VisitCount <= 1,
		IPReputation: ipReputation,
	}

	if record.RiskScore >= 70 {
		result.RiskLevel = "critical"
		result.Factors = append(result.Factors, RiskFactor{
			Name:     "High Risk Score",
			Weight:   1.0,
			Score:    record.RiskScore,
			Critical: true,
		})
	}

	if record.IsBot {
		result.Factors = append(result.Factors, RiskFactor{
			Name:     "Bot Detected",
			Weight:   1.0,
			Score:    100,
			Critical: true,
		})
	}

	if record.ProxyDetected {
		result.Factors = append(result.Factors, RiskFactor{
			Name:   "Proxy Detected",
			Weight: 0.5,
			Score:  30,
		})
	}

	if record.TorDetected {
		result.Factors = append(result.Factors, RiskFactor{
			Name:   "Tor Exit Node",
			Weight: 0.6,
			Score:  40,
		})
	}

	if record.VisitCount < 3 {
		result.Factors = append(result.Factors, RiskFactor{
			Name:   "New Device",
			Weight: 0.3,
			Score:  15,
		})
	}

	return result, nil
}

func (s *DeviceFingerprintService) UpdateFingerprintRiskScore(fingerprint string, riskScore float64, isBot bool) error {
	return s.db.Model(&models.DeviceFingerprintRecord{}).
		Where("fingerprint = ?", fingerprint).
		Updates(map[string]interface{}{
			"risk_score": riskScore,
			"is_bot":     isBot,
			"updated_at": time.Now(),
		}).Error
}

func (s *DeviceFingerprintService) SetFingerprintTrustLevel(fingerprint string, trustLevel models.TrustLevel, isTrusted bool) error {
	return s.db.Model(&models.DeviceFingerprintRecord{}).
		Where("fingerprint = ?", fingerprint).
		Updates(map[string]interface{}{
			"trust_level": string(trustLevel),
			"is_trusted":  isTrusted,
			"updated_at": time.Now(),
		}).Error
}

func (s *DeviceFingerprintService) ListFingerprints(userID *uint, page, pageSize int) ([]models.DeviceFingerprintRecord, int64, error) {
	var records []models.DeviceFingerprintRecord
	var total int64

	query := s.db.Model(&models.DeviceFingerprintRecord{})
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("last_seen_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error

	return records, total, err
}

func (s *DeviceFingerprintService) GetFingerprintStats(userID *uint) (map[string]interface{}, error) {
	var totalCount, botCount, proxyCount, vpnCount int64
	var avgRiskScore float64

	query := s.db.Model(&models.DeviceFingerprintRecord{})
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	query.Count(&totalCount)
	query.Where("is_bot = ?", true).Count(&botCount)
	query.Where("proxy_detected = ?", true).Count(&proxyCount)
	query.Where("vpn_detected = ?", true).Count(&vpnCount)

	var sumRisk float64
	var riskCount int64
	s.db.Model(&models.DeviceFingerprintRecord{}).
		Where("user_id = ? AND risk_score > 0", *userID).
		Select("COALESCE(SUM(risk_score), 0)").Row().Scan(&sumRisk)
	s.db.Model(&models.DeviceFingerprintRecord{}).
		Where("user_id = ? AND risk_score > 0", *userID).
		Count(&riskCount)

	if riskCount > 0 {
		avgRiskScore = sumRisk / float64(riskCount)
	}

	riskDistribution := make(map[string]int64)
	var lowCount, mediumCount, highCount, criticalCount int64
	s.db.Model(&models.DeviceFingerprintRecord{}).
		Where("user_id = ? AND risk_score < 30", *userID).
		Count(&lowCount)
	s.db.Model(&models.DeviceFingerprintRecord{}).
		Where("user_id = ? AND risk_score >= 30 AND risk_score < 60", *userID).
		Count(&mediumCount)
	s.db.Model(&models.DeviceFingerprintRecord{}).
		Where("user_id = ? AND risk_score >= 60 AND risk_score < 80", *userID).
		Count(&highCount)
	s.db.Model(&models.DeviceFingerprintRecord{}).
		Where("user_id = ? AND risk_score >= 80", *userID).
		Count(&criticalCount)
	riskDistribution["low"] = lowCount
	riskDistribution["medium"] = mediumCount
	riskDistribution["high"] = highCount
	riskDistribution["critical"] = criticalCount

	return map[string]interface{}{
		"total_count":       totalCount,
		"bot_count":         botCount,
		"proxy_count":       proxyCount,
		"vpn_count":         vpnCount,
		"average_risk_score": avgRiskScore,
		"risk_distribution":  riskDistribution,
	}, nil
}

func (s *DeviceFingerprintService) ParseBrowserFeatures(featuresJSON string) (*BrowserFeatures, error) {
	var features BrowserFeatures
	if featuresJSON == "" {
		return &features, nil
	}
	err := json.Unmarshal([]byte(featuresJSON), &features)
	return &features, err
}

func (s *DeviceFingerprintService) ExtractFromRequest(r *http.Request) (string, string, string) {
	clientIP := s.getRealIP(r)
	userAgent := r.UserAgent()
	
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			clientIP = strings.TrimSpace(parts[0])
		}
	}
	
	return clientIP, userAgent, r.Referer()
}

func (s *DeviceFingerprintService) getRealIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *DeviceFingerprintService) DetectProxyByHeaders(r *http.Request) bool {
	proxyHeaders := []string{
		"X-Forwarded-For", "X-Proxy-Id", "X-Real-IP", "Via",
		"Proxy-Agent", "Client-IP", "WL-Proxy-Agent", "Proxy-Authorization",
	}

	for _, header := range proxyHeaders {
		if r.Header.Get(header) != "" {
			return true
		}
	}

	return false
}

func (s *DeviceFingerprintService) ParseUserAgent(ua string) (browser, version, os, osVersion string) {
	browserRegex := regexp.MustCompile(`(Chrome|Firefox|Safari|Edge|Opera|IE|Edg|Vivaldi|Brave)[/ ](\d+[\.\d]*)`)
	browserMatch := browserRegex.FindStringSubmatch(ua)
	if len(browserMatch) >= 3 {
		browser = browserMatch[1]
		version = browserMatch[2]
	}

	osRegex := regexp.MustCompile(`(Windows|Mac OS X|Linux|Android|iOS|iPhone|iPad)[ ]?([\d_\.]*)`)
	osMatch := osRegex.FindStringSubmatch(ua)
	if len(osMatch) >= 2 {
		os = osMatch[1]
		if len(osMatch) >= 3 {
			osVersion = osMatch[2]
		}
	}

	return
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func (s *DeviceFingerprintService) GetOrCreateIPReputation(ipAddress string) (*models.IPReputation, error) {
	s.ipCacheMu.RLock()
	if cached, exists := s.ipCache[ipAddress]; exists {
		s.ipCacheMu.RUnlock()
		return cached, nil
	}
	s.ipCacheMu.RUnlock()

	var ipRec models.IPReputation
	err := s.db.Where("ip_address = ?", ipAddress).First(&ipRec).Error
	if err == nil {
		s.ipCacheMu.Lock()
		s.ipCache[ipAddress] = &ipRec
		s.ipCacheMu.Unlock()
		return &ipRec, nil
	}

	ipRec = models.IPReputation{
		IPAddress:       ipAddress,
		ThreatLevel:     "low",
		ReputationScore: 100,
		FirstSeenAt:     time.Now(),
		LastSeenAt:      time.Now(),
	}

	if ip := net.ParseIP(ipAddress); ip != nil {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() {
			ipRec.ThreatLevel = "none"
			ipRec.ReputationScore = 100
			ipRec.IsResidential = true
		}
	}

	err = s.db.Create(&ipRec).Error
	if err == nil {
		s.ipCacheMu.Lock()
		s.ipCache[ipAddress] = &ipRec
		s.ipCacheMu.Unlock()
	}

	return &ipRec, err
}

func (s *DeviceFingerprintService) UpdateIPReputation(ipAddress string, updates map[string]interface{}) error {
	delete(s.ipCache, ipAddress)
	return s.db.Model(&models.IPReputation{}).
		Where("ip_address = ?", ipAddress).
		Updates(updates).Error
}

func (s *DeviceFingerprintService) GenerateClientFingerprintScript() string {
	return `(function() { return { canvas: function() { return 'no_canvas'; }, webgl: function() { return 'no_webgl'; }, audio: function() { return 'no_audio'; }, screen: function() { return screen.width + 'x' + screen.height; }, timezone: function() { return Intl.DateTimeFormat().resolvedOptions().timeZone; }, language: function() { return navigator.language; }, platform: function() { return navigator.platform; } }; })();`
}

func (s *DeviceFingerprintService) HashString(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

func (s *DeviceFingerprintService) ParseScreenInfo(screenInfo string) (width, height, colorDepth int) {
	parts := strings.Split(screenInfo, "x")
	if len(parts) >= 2 {
		width, _ = strconv.Atoi(parts[0])
		height, _ = strconv.Atoi(parts[1])
		if len(parts) >= 3 {
			colorDepth, _ = strconv.Atoi(parts[2])
		}
	}
	return
}
