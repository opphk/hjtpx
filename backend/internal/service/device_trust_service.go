package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

type DeviceTrustService struct {
	trustCache    map[string]*DeviceTrustInfo
	cacheMutex    sync.RWMutex
	trustHistory   map[string][]*TrustEvent
	historyMutex   sync.RWMutex
}

type DeviceTrustInfo struct {
	Fingerprint   string    `json:"fingerprint"`
	TrustScore    int       `json:"trust_score"`
	TrustLevel    string    `json:"trust_level"`
	VisitCount    int       `json:"visit_count"`
	SuccessCount  int       `json:"success_count"`
	FailCount     int       `json:"fail_count"`
	LastVisit     time.Time `json:"last_visit"`
	FirstVisit    time.Time `json:"first_visit"`
	RiskScore     float64   `json:"risk_score"`
	RiskFactors   []string  `json:"risk_factors"`
	IsVerified    bool      `json:"is_verified"`
	VerifiedAt    *time.Time `json:"verified_at,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

type TrustEvent struct {
	ID          uint      `json:"id"`
	Fingerprint string    `json:"fingerprint"`
	Event       string    `json:"event"`
	ScoreChange int       `json:"score_change"`
	RiskScore   float64   `json:"risk_score"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	Timestamp   time.Time `json:"timestamp"`
}

type TrustDecision struct {
	Pass           bool     `json:"pass"`
	TrustScore     int      `json:"trust_score"`
	TrustLevel     string   `json:"trust_level"`
	RiskScore      float64  `json:"risk_score"`
	RiskFactors    []string `json:"risk_factors"`
	Action         string   `json:"action"`
	ChallengeLevel int      `json:"challenge_level"`
	Message        string   `json:"message"`
}

type FingerprintAnalysis struct {
	Fingerprint  string                 `json:"fingerprint"`
	Hash         string                 `json:"hash"`
	Components   map[string]interface{}  `json:"components"`
	RiskScore    float64                `json:"risk_score"`
	RiskLevel    string                 `json:"risk_level"`
	RiskFactors  []string               `json:"risk_factors"`
	TrustScore   int                    `json:"trust_score"`
	TrustLevel   string                 `json:"trust_level"`
	Confidence   float64                `json:"confidence"`
	IsTrusted    bool                   `json:"is_trusted"`
	IsNew        bool                   `json:"is_new"`
	IsVerified   bool                   `json:"is_verified"`
	SessionCount int                    `json:"session_count"`
}

var (
	deviceTrustInstance *DeviceTrustService
	deviceTrustOnce     sync.Once
)

func GetDeviceTrustService() *DeviceTrustService {
	deviceTrustOnce.Do(func() {
		deviceTrustInstance = NewDeviceTrustService()
	})
	return deviceTrustInstance
}

func NewDeviceTrustService() *DeviceTrustService {
	svc := &DeviceTrustService{
		trustCache:  make(map[string]*DeviceTrustInfo),
		trustHistory: make(map[string][]*TrustEvent),
	}
	go svc.cleanupRoutine()
	return svc
}

func (s *DeviceTrustService) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.cleanupExpiredData()
	}
}

func (s *DeviceTrustService) cleanupExpiredData() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	now := time.Now()
	for fp, info := range s.trustCache {
		if info.ExpiresAt != nil && now.After(*info.ExpiresAt) {
			delete(s.trustCache, fp)
		}
	}
}

func (s *DeviceTrustService) GenerateFingerprintHash(data map[string]interface{}) string {
	var components []string

	if webgl, ok := data["webgl"].(string); ok && webgl != "" {
		components = append(components, webgl)
	}
	if canvas, ok := data["canvas"].(string); ok && canvas != "" {
		components = append(components, canvas)
	}
	if fonts, ok := data["fonts"].(string); ok && fonts != "" {
		components = append(components, fonts)
	}
	if screen, ok := data["screen"].(string); ok && screen != "" {
		components = append(components, screen)
	}
	if platform, ok := data["platform"].(string); ok && platform != "" {
		components = append(components, platform)
	}
	if timezone, ok := data["timezone"].(string); ok && timezone != "" {
		components = append(components, timezone)
	}
	if languages, ok := data["languages"].(string); ok && languages != "" {
		components = append(components, languages)
	}

	if len(components) == 0 {
		return ""
	}

	combined := fmt.Sprintf("%s|%v", strings.Join(components, "|"), time.Now().Unix()/86400)
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])[:32]
}

func (s *DeviceTrustService) AnalyzeFingerprint(fingerprint string, data map[string]interface{}) *FingerprintAnalysis {
	analysis := &FingerprintAnalysis{
		Fingerprint: fingerprint,
		Components: data,
		RiskFactors: make([]string, 0),
		Confidence: 0.5,
	}

	if fingerprint == "" {
		analysis.RiskScore = 50
		analysis.RiskLevel = "high"
		analysis.TrustScore = 30
		analysis.Confidence = 0.3
		analysis.RiskFactors = append(analysis.RiskFactors, "missing_fingerprint")
		return analysis
	}

	analysis.Hash = s.GenerateFingerprintHash(data)

	s.cacheMutex.RLock()
	trustInfo, exists := s.trustCache[fingerprint]
	s.cacheMutex.RUnlock()

	if !exists {
		analysis.IsNew = true
		analysis.TrustScore = 50
		analysis.RiskScore = 25
		analysis.RiskLevel = "medium"
		analysis.Confidence = 0.6
		analysis.RiskFactors = append(analysis.RiskFactors, "new_device")
		return analysis
	}

	analysis.TrustScore = trustInfo.TrustScore
	analysis.RiskScore = trustInfo.RiskScore
	analysis.SessionCount = trustInfo.VisitCount

	if trustInfo.RiskScore >= 60 {
		analysis.RiskLevel = "high"
		analysis.RiskFactors = trustInfo.RiskFactors
	} else if trustInfo.RiskScore >= 30 {
		analysis.RiskLevel = "medium"
	} else {
		analysis.RiskLevel = "low"
	}

	analysis.TrustLevel = s.calculateTrustLevel(trustInfo.TrustScore)
	analysis.IsTrusted = trustInfo.TrustScore >= 70 && trustInfo.RiskScore < 30
	analysis.IsVerified = trustInfo.IsVerified

	if trustInfo.VerifiedAt != nil {
		timeSinceVerify := time.Since(*trustInfo.VerifiedAt)
		if timeSinceVerify < 24*time.Hour {
			analysis.Confidence = 0.95
		} else if timeSinceVerify < 7*24*time.Hour {
			analysis.Confidence = 0.85
		} else {
			analysis.Confidence = 0.7
		}
	}

	if trustInfo.VisitCount >= 10 {
		analysis.Confidence += 0.1
	}
	if trustInfo.SuccessCount >= 5 {
		analysis.Confidence += 0.05
	}

	analysis.Confidence = math.Min(analysis.Confidence, 1.0)

	return analysis
}

func (s *DeviceTrustService) EvaluateTrust(fingerprint string, data map[string]interface{}) *TrustDecision {
	decision := &TrustDecision{
		TrustScore:     50,
		TrustLevel:    "medium",
		RiskScore:     0,
		RiskFactors:   make([]string, 0),
		ChallengeLevel: 1,
		Pass:          false,
	}

	if fingerprint == "" {
		decision.Action = "block"
		decision.ChallengeLevel = 3
		decision.Message = "缺少设备指纹"
		decision.RiskFactors = append(decision.RiskFactors, "missing_fingerprint")
		return decision
	}

	analysis := s.AnalyzeFingerprint(fingerprint, data)
	decision.TrustScore = analysis.TrustScore
	decision.RiskScore = analysis.RiskScore
	decision.RiskFactors = analysis.RiskFactors

	if analysis.IsNew {
		decision.ChallengeLevel = 2
	}

	if s.checkRiskFactors(data) {
		decision.RiskScore = math.Min(decision.RiskScore+30, 100)
		decision.RiskFactors = append(decision.RiskFactors, s.getRiskIndicators(data)...)
		decision.ChallengeLevel = 3
	}

	decision.TrustLevel = s.calculateTrustLevel(decision.TrustScore)

	if decision.TrustScore >= 80 && decision.RiskScore < 20 {
		decision.Pass = true
		decision.Action = "allow"
		decision.ChallengeLevel = 0
		decision.Message = "信任验证通过"
	} else if decision.TrustScore >= 60 && decision.RiskScore < 40 {
		decision.Pass = true
		decision.Action = "allow_with_monitoring"
		decision.ChallengeLevel = 1
		decision.Message = "验证通过，建议监控"
	} else if decision.TrustScore >= 40 {
		decision.Pass = false
		decision.Action = "challenge"
		decision.ChallengeLevel = 2
		decision.Message = "需要完成验证"
	} else {
		decision.Pass = false
		decision.Action = "block"
		decision.ChallengeLevel = 3
		decision.Message = "验证失败，访问被拒绝"
	}

	return decision
}

func (s *DeviceTrustService) checkRiskFactors(data map[string]interface{}) bool {
	if _, ok := data["webdriver"]; ok {
		wd, _ := data["webdriver"].(string)
		if wd == "wd:true" || wd == "true" {
			return true
		}
	}

	if _, ok := data["headless"]; ok {
		if hl, ok := data["headless"].(bool); ok && hl {
			return true
		}
	}

	if webgl, ok := data["webgl"].(string); ok {
		webglLower := strings.ToLower(webgl)
		riskIndicators := []string{"swiftshader", "llvmpipe", "mesa", "virtualbox", "vmware"}
		for _, indicator := range riskIndicators {
			if strings.Contains(webglLower, indicator) {
				return true
			}
		}
	}

	if fonts, ok := data["fonts"].(string); ok {
		fontCount := len(strings.Split(fonts, ","))
		if fontCount < 3 {
			return true
		}
	}

	return false
}

func (s *DeviceTrustService) getRiskIndicators(data map[string]interface{}) []string {
	indicators := make([]string, 0)

	if _, ok := data["webdriver"]; ok {
		indicators = append(indicators, "automation_detected")
	}

	if _, ok := data["headless"]; ok {
		indicators = append(indicators, "headless_browser")
	}

	if webgl, ok := data["webgl"].(string); ok {
		if strings.Contains(strings.ToLower(webgl), "swiftshader") {
			indicators = append(indicators, "software_renderer")
		}
	}

	if fonts, ok := data["fonts"].(string); ok {
		if len(strings.Split(fonts, ",")) < 3 {
			indicators = append(indicators, "minimal_fonts")
		}
	}

	return indicators
}

func (s *DeviceTrustService) calculateTrustLevel(score int) string {
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
		return models.TrustLevelMinimal
	}
}

func (s *DeviceTrustService) UpdateTrustScore(fingerprint string, event string, ipAddress string, userAgent string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	info, exists := s.trustCache[fingerprint]
	if !exists {
		info = &DeviceTrustInfo{
			Fingerprint: fingerprint,
			TrustScore:  50,
			TrustLevel: models.TrustLevelMedium,
			FirstVisit: time.Now(),
			RiskFactors: make([]string, 0),
		}
		s.trustCache[fingerprint] = info
	}

	info.LastVisit = time.Now()
	info.VisitCount++

	trustEvent := &TrustEvent{
		Fingerprint: fingerprint,
		Event:       event,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Timestamp:   time.Now(),
	}

	switch event {
	case models.EventTypeLoginSuccess:
		info.SuccessCount++
		info.TrustScore = min(100, info.TrustScore+5)
		trustEvent.ScoreChange = 5
		if info.FailCount > 0 {
			info.FailCount = max(0, info.FailCount-1)
		}
	case models.EventTypeLoginFailed:
		info.FailCount++
		info.TrustScore = max(0, info.TrustScore-10)
		trustEvent.ScoreChange = -10
	case models.EventTypeVerify:
		info.TrustScore = min(100, info.TrustScore+3)
		trustEvent.ScoreChange = 3
	case models.EventTypeRiskDetected:
		info.TrustScore = max(0, info.TrustScore-15)
		trustEvent.ScoreChange = -15
	}

	info.TrustLevel = s.calculateTrustLevel(info.TrustScore)

	s.historyMutex.Lock()
	s.trustHistory[fingerprint] = append(s.trustHistory[fingerprint], trustEvent)
	if len(s.trustHistory[fingerprint]) > 100 {
		s.trustHistory[fingerprint] = s.trustHistory[fingerprint][len(s.trustHistory[fingerprint])-100:]
	}
	s.historyMutex.Unlock()
}

func (s *DeviceTrustService) GetTrustInfo(fingerprint string) *DeviceTrustInfo {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	if info, exists := s.trustCache[fingerprint]; exists {
		return info
	}
	return nil
}

func (s *DeviceTrustService) GetTrustHistory(fingerprint string, limit int) []*TrustEvent {
	s.historyMutex.RLock()
	defer s.historyMutex.RUnlock()

	events := s.trustHistory[fingerprint]
	if len(events) == 0 {
		return []*TrustEvent{}
	}

	start := 0
	if len(events) > limit {
		start = len(events) - limit
	}
	return events[start:]
}

func (s *DeviceTrustService) MarkAsVerified(fingerprint string, duration time.Duration) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	info, exists := s.trustCache[fingerprint]
	if !exists {
		info = &DeviceTrustInfo{
			Fingerprint: fingerprint,
			TrustScore:  50,
			TrustLevel: models.TrustLevelMedium,
			FirstVisit: time.Now(),
			RiskFactors: make([]string, 0),
		}
		s.trustCache[fingerprint] = info
	}

	now := time.Now()
	info.IsVerified = true
	info.VerifiedAt = &now
	info.TrustScore = min(100, info.TrustScore+20)
	info.TrustLevel = s.calculateTrustLevel(info.TrustScore)

	expiresAt := now.Add(duration)
	info.ExpiresAt = &expiresAt
}

func (s *DeviceTrustService) SetRiskScore(fingerprint string, riskScore float64, riskFactors []string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	info, exists := s.trustCache[fingerprint]
	if !exists {
		info = &DeviceTrustInfo{
			Fingerprint: fingerprint,
			TrustScore:  50,
			TrustLevel: models.TrustLevelMedium,
			RiskFactors: make([]string, 0),
		}
		s.trustCache[fingerprint] = info
	}

	info.RiskScore = riskScore
	info.RiskFactors = riskFactors

	trustPenalty := int(riskScore * 0.5)
	info.TrustScore = max(0, info.TrustScore-trustPenalty)
	info.TrustLevel = s.calculateTrustLevel(info.TrustScore)
}

func (s *DeviceTrustService) ExportTrustData(fingerprint string) (string, error) {
	info := s.GetTrustInfo(fingerprint)
	if info == nil {
		return "{}", fmt.Errorf("fingerprint not found")
	}

	data, err := json.Marshal(info)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (s *DeviceTrustService) ImportTrustData(fingerprint string, data string) error {
	var info DeviceTrustInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return err
	}

	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	info.Fingerprint = fingerprint
	s.trustCache[fingerprint] = &info
	return nil
}

func (s *DeviceTrustService) RemoveFingerprint(fingerprint string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	delete(s.trustCache, fingerprint)

	s.historyMutex.Lock()
	delete(s.trustHistory, fingerprint)
	s.historyMutex.Unlock()
}

func (s *DeviceTrustService) GetStatistics() map[string]interface{} {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	stats := map[string]interface{}{
		"total_fingerprints": len(s.trustCache),
		"trust_distribution": map[string]int{
			"full":    0,
			"high":    0,
			"medium": 0,
			"low":     0,
			"minimal": 0,
		},
		"average_trust_score": 0.0,
		"verified_count":      0,
	}

	totalScore := 0
	for _, info := range s.trustCache {
		stats["trust_distribution"].(map[string]int)[info.TrustLevel]++
		totalScore += info.TrustScore
		if info.IsVerified {
			stats["verified_count"] = stats["verified_count"].(int) + 1
		}
	}

	if len(s.trustCache) > 0 {
		stats["average_trust_score"] = float64(totalScore) / float64(len(s.trustCache))
	}

	return stats
}

type ContextKey string

const (
	ContextKeyFingerprint ContextKey = "fingerprint"
	ContextKeyTrustScore  ContextKey = "trust_score"
	ContextKeyRiskScore  ContextKey = "risk_score"
)

func (s *DeviceTrustService) ShouldSkipVerification(ctx context.Context, fingerprint string) (bool, string) {
	if fingerprint == "" {
		return false, "missing_fingerprint"
	}

	info := s.GetTrustInfo(fingerprint)
	if info == nil {
		return false, "new_device"
	}

	if info.IsVerified && info.ExpiresAt != nil {
		if time.Now().Before(*info.ExpiresAt) {
			return true, "verified_device"
		}
	}

	if info.TrustScore >= 90 && info.RiskScore < 10 {
		return true, "high_trust"
	}

	return false, ""
}
