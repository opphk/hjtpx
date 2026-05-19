package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

var (
	adaptiveSecurityOnce   = &sync.Once{}
	adaptiveSecurityService *AdaptiveSecurityService
)

func initAdaptiveSecurityServices() {
	adaptiveSecurityOnce.Do(func() {
		adaptiveSecurityService = NewAdaptiveSecurityService()
	})
}

type AdaptiveSecurityService struct {
	threatIntel      *service.ThreatIntelligenceService
	dynamicDefense   *service.DynamicDefenseService
	aiDetector       *service.AIAttackDetectorService
	honeypot         *service.HoneypotService
	autoResponse     *service.AutoResponseService
	botDetection     *service.BotDetectionService
	behaviorAnalysis *service.RealTimeBehaviorAnalysisService
	config           *AdaptiveSecurityConfig
	enabled          bool
	mu               sync.RWMutex
}

type AdaptiveSecurityConfig struct {
	ThreatIntelEnabled   bool
	DynamicDefenseEnabled bool
	AIAttackEnabled      bool
	HoneypotEnabled      bool
	AutoResponseEnabled  bool
	BotDetectionEnabled  bool
	BehaviorAnalysisEnabled bool
	IntegrationMode     IntegrationMode
	ResponseDelay       time.Duration
	MaxProcessingTime   time.Duration
}

type IntegrationMode string

const (
	IntegrationModeSequential IntegrationMode = "sequential"
	IntegrationModeParallel   IntegrationMode = "parallel"
	IntegrationModeHybrid     IntegrationMode = "hybrid"
)

func NewAdaptiveSecurityService() *AdaptiveSecurityService {
	service := &AdaptiveSecurityService{
		threatIntel:      service.NewThreatIntelligenceService(),
		dynamicDefense:   service.NewDynamicDefenseService(),
		aiDetector:       service.NewAIAttackDetectorService(),
		honeypot:         service.NewHoneypotService(),
		autoResponse:     service.NewAutoResponseService(),
		botDetection:     service.NewBotDetectionService(),
		behaviorAnalysis: service.NewRealTimeBehaviorAnalysisService(),
		config: &AdaptiveSecurityConfig{
			ThreatIntelEnabled:    true,
			DynamicDefenseEnabled: true,
			AIAttackEnabled:       true,
			HoneypotEnabled:       true,
			AutoResponseEnabled:   true,
			BotDetectionEnabled:   true,
			BehaviorAnalysisEnabled: true,
			IntegrationMode:       IntegrationModeHybrid,
			ResponseDelay:          100 * time.Millisecond,
			MaxProcessingTime:     500 * time.Millisecond,
		},
		enabled: true,
	}

	return service
}

type AdaptiveSecurityResult struct {
	ShouldBlock       bool
	ShouldChallenge   bool
	RiskScore         float64
	ThreatLevel       int
	ThreatTypes       []string
	Recommendations   []string
	ActionTaken       []string
	ProcessingTime    time.Duration
	ComponentsUsed    []string
	Confidence        float64
	ThreatIntelScore  float64
	DynamicDefenseScore float64
	AIAttackScore     float64
	HoneypotTriggered bool
}

type MiddlewareConfig struct {
	Enabled           bool
	ExcludePaths      []string
	RequireAllChecks  bool
	BlockOnAnyThreat  bool
	ChallengeOnMedium bool
	NotifyOnBlock     bool
	UseHoneypots      bool
	UseAutoResponse   bool
}

var DefaultAdaptiveSecurityConfig = MiddlewareConfig{
	Enabled:           true,
	RequireAllChecks:  false,
	BlockOnAnyThreat:  false,
	ChallengeOnMedium: true,
	NotifyOnBlock:     true,
	UseHoneypots:      true,
	UseAutoResponse:   true,
}

func AdaptiveSecurityMiddleware(config ...MiddlewareConfig) gin.HandlerFunc {
	initAdaptiveSecurityServices()

	cfg := DefaultAdaptiveSecurityConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		if !cfg.Enabled || !adaptiveSecurityService.enabled {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		for _, excluded := range cfg.ExcludePaths {
			if path == excluded || hasPathPrefix(path, excluded+"/") {
				c.Next()
				return
			}
		}

		startTime := time.Now()

		result := adaptiveSecurityService.ProcessRequest(c.Request)

		result.ProcessingTime = time.Since(startTime)

		c.Set("adaptive_security", result)
		c.Set("security_risk_score", result.RiskScore)
		c.Set("security_threat_level", result.ThreatLevel)
		c.Set("security_threat_types", result.ThreatTypes)

		if result.ShouldBlock {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":         "Access denied",
				"code":          http.StatusForbidden,
				"message":       "Security policy violation detected",
				"threat_level":   result.ThreatLevel,
				"threat_types":   result.ThreatTypes,
				"recommendations": result.Recommendations,
			})
			return
		}

		if cfg.ChallengeOnMedium && result.ThreatLevel >= 3 && result.ThreatLevel < 5 {
			c.Set("require_challenge", true)
			c.Set("challenge_type", "adaptive_security")
		}

		if result.ShouldChallenge {
			c.Set("require_challenge", true)
			c.Set("challenge_type", "adaptive_security")
			c.Set("risk_score", result.RiskScore)
		}

		c.Next()
	}
}

func hasPathPrefix(path, prefix string) bool {
	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
}

func (s *AdaptiveSecurityService) ProcessRequest(r *http.Request) *AdaptiveSecurityResult {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.MaxProcessingTime)
	defer cancel()

	result := &AdaptiveSecurityResult{
		ThreatTypes:     []string{},
		Recommendations: []string{},
		ActionTaken:     []string{},
		ComponentsUsed:  []string{},
	}

	sessionID := r.Header.Get("X-Session-ID")
	if sessionID == "" {
		sessionID = generateSessionID(r)
	}

	switch s.config.IntegrationMode {
	case IntegrationModeSequential:
		s.processSequential(ctx, r, sessionID, result)
	case IntegrationModeParallel:
		s.processParallel(ctx, r, sessionID, result)
	case IntegrationModeHybrid:
		s.processHybrid(ctx, r, sessionID, result)
	}

	s.calculateFinalScore(result)

	return result
}

func (s *AdaptiveSecurityService) processSequential(ctx context.Context, r *http.Request, sessionID string, result *AdaptiveSecurityResult) {
	if s.config.ThreatIntelEnabled {
		s.runThreatIntelligence(ctx, r, result)
		result.ComponentsUsed = append(result.ComponentsUsed, "threat_intel")
	}

	if s.config.DynamicDefenseEnabled {
		s.runDynamicDefense(ctx, r, result)
		result.ComponentsUsed = append(result.ComponentsUsed, "dynamic_defense")
	}

	if s.config.AIAttackEnabled {
		s.runAIAttackDetection(ctx, r, sessionID, result)
		result.ComponentsUsed = append(result.ComponentsUsed, "ai_attack_detector")
	}

	if s.config.BotDetectionEnabled {
		s.runBotDetection(r, result)
		result.ComponentsUsed = append(result.ComponentsUsed, "bot_detection")
	}

	if s.config.HoneypotEnabled {
		s.runHoneypotCheck(ctx, r, result)
		result.ComponentsUsed = append(result.ComponentsUsed, "honeypot")
	}

	if s.config.AutoResponseEnabled && result.ShouldBlock {
		s.runAutoResponse(ctx, r, result)
		result.ComponentsUsed = append(result.ComponentsUsed, "auto_response")
	}
}

func (s *AdaptiveSecurityService) processParallel(ctx context.Context, r *http.Request, sessionID string, result *AdaptiveSecurityResult) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	if s.config.ThreatIntelEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localResult := &AdaptiveSecurityResult{}
			s.runThreatIntelligence(ctx, r, localResult)
			mu.Lock()
			result.ThreatIntelScore = localResult.ThreatIntelScore
			result.ThreatTypes = append(result.ThreatTypes, localResult.ThreatTypes...)
			result.Recommendations = append(result.Recommendations, localResult.Recommendations...)
			result.ComponentsUsed = append(result.ComponentsUsed, "threat_intel")
			mu.Unlock()
		}()
	}

	if s.config.DynamicDefenseEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localResult := &AdaptiveSecurityResult{}
			s.runDynamicDefense(ctx, r, localResult)
			mu.Lock()
			result.DynamicDefenseScore = localResult.DynamicDefenseScore
			result.ThreatTypes = append(result.ThreatTypes, localResult.ThreatTypes...)
			result.Recommendations = append(result.Recommendations, localResult.Recommendations...)
			result.ComponentsUsed = append(result.ComponentsUsed, "dynamic_defense")
			mu.Unlock()
		}()
	}

	if s.config.AIAttackEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localResult := &AdaptiveSecurityResult{}
			s.runAIAttackDetection(ctx, r, sessionID, localResult)
			mu.Lock()
			result.AIAttackScore = localResult.AIAttackScore
			result.ThreatTypes = append(result.ThreatTypes, localResult.ThreatTypes...)
			result.Recommendations = append(result.Recommendations, localResult.Recommendations...)
			result.ComponentsUsed = append(result.ComponentsUsed, "ai_attack_detector")
			mu.Unlock()
		}()
	}

	if s.config.BotDetectionEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localResult := &AdaptiveSecurityResult{}
			s.runBotDetection(r, localResult)
			mu.Lock()
			result.ThreatTypes = append(result.ThreatTypes, localResult.ThreatTypes...)
			result.Recommendations = append(result.Recommendations, localResult.Recommendations...)
			result.ComponentsUsed = append(result.ComponentsUsed, "bot_detection")
			mu.Unlock()
		}()
	}

	if s.config.HoneypotEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localResult := &AdaptiveSecurityResult{}
			s.runHoneypotCheck(ctx, r, localResult)
			mu.Lock()
			result.HoneypotTriggered = localResult.HoneypotTriggered
			result.ThreatTypes = append(result.ThreatTypes, localResult.ThreatTypes...)
			result.Recommendations = append(result.Recommendations, localResult.Recommendations...)
			result.ComponentsUsed = append(result.ComponentsUsed, "honeypot")
			mu.Unlock()
		}()
	}

	wg.Wait()
}

func (s *AdaptiveSecurityService) processHybrid(ctx context.Context, r *http.Request, sessionID string, result *AdaptiveSecurityResult) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	wg.Add(2)
	go func() {
		defer wg.Done()
		s.runThreatIntelligence(ctx, r, result)
		mu.Lock()
		result.ComponentsUsed = append(result.ComponentsUsed, "threat_intel")
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		s.runDynamicDefense(ctx, r, result)
		mu.Lock()
		result.ComponentsUsed = append(result.ComponentsUsed, "dynamic_defense")
		mu.Unlock()
	}()

	wg.Wait()

	if result.RiskScore < 0.3 {
		return
	}

	wg.Add(3)
	go func() {
		defer wg.Done()
		s.runAIAttackDetection(ctx, r, sessionID, result)
		mu.Lock()
		result.ComponentsUsed = append(result.ComponentsUsed, "ai_attack_detector")
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		s.runBotDetection(r, result)
		mu.Lock()
		result.ComponentsUsed = append(result.ComponentsUsed, "bot_detection")
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		s.runHoneypotCheck(ctx, r, result)
		mu.Lock()
		result.ComponentsUsed = append(result.ComponentsUsed, "honeypot")
		mu.Unlock()
	}()

	wg.Wait()

	if s.config.AutoResponseEnabled && result.ShouldBlock {
		s.runAutoResponse(ctx, r, result)
		result.ComponentsUsed = append(result.ComponentsUsed, "auto_response")
	}
}

func (s *AdaptiveSecurityService) runThreatIntelligence(ctx context.Context, r *http.Request, result *AdaptiveSecurityResult) {
	ip := getClientIP(r)

	assessment, err := s.threatIntel.GetComprehensiveThreatAssessment(ctx, ip, "", r.URL.String())
	if err != nil {
		return
	}

	result.ThreatIntelScore = assessment.CombinedScore
	result.ThreatTypes = append(result.ThreatTypes, assessment.ThreatTypes...)
	result.Recommendations = append(result.Recommendations, assessment.Recommendations...)

	if assessment.ShouldBlock {
		result.ShouldBlock = true
		result.ActionTaken = append(result.ActionTaken, "threat_intel_block")
	}
}

func (s *AdaptiveSecurityService) runDynamicDefense(ctx context.Context, r *http.Request, result *AdaptiveSecurityResult) {
	defenseResult, err := s.dynamicDefense.EvaluateRequest(ctx, r.Request)
	if err != nil {
		return
	}

	result.DynamicDefenseScore = defenseResult.RiskScore
	if defenseResult.ThreatLevel > result.ThreatLevel {
		result.ThreatLevel = int(defenseResult.ThreatLevel)
	}
	result.ThreatTypes = append(result.ThreatTypes, defenseResult.ThreatLevel.String())
	result.Recommendations = append(result.Recommendations, defenseResult.Recommendations...)

	if defenseResult.ShouldBlock {
		result.ShouldBlock = true
		result.ActionTaken = append(result.ActionTaken, "dynamic_defense_block")
	}

	if defenseResult.ShouldChallenge {
		result.ShouldChallenge = true
		result.ActionTaken = append(result.ActionTaken, "dynamic_defense_challenge")
	}
}

func (s *AdaptiveSecurityService) runAIAttackDetection(ctx context.Context, r *http.Request, sessionID string, result *AdaptiveSecurityResult) {
	sessionID = r.Header.Get("X-Session-ID")
	if sessionID == "" {
		sessionID = generateSessionID(r)
	}

	detectionResult, err := s.aiDetector.DetectAttack(ctx, r.Request, sessionID)
	if err != nil {
		return
	}

	if detectionResult.IsAttack {
		result.AIAttackScore = detectionResult.Confidence * 100
		result.ThreatTypes = append(result.ThreatTypes, string(detectionResult.AttackType))
		result.Recommendations = append(result.Recommendations, detectionResult.MitigationActions...)
		result.ActionTaken = append(result.ActionTaken, fmt.Sprintf("ai_detected:%s", detectionResult.AttackType))
	}
}

func (s *AdaptiveSecurityService) runBotDetection(r *http.Request, result *AdaptiveSecurityResult) {
	additionalData := make(map[string]string)
	additionalData["X-Screen-Info"] = r.Header.Get("X-Screen-Info")
	additionalData["X-Timezone"] = r.Header.Get("X-Timezone")
	additionalData["X-Canvas-Hash"] = r.Header.Get("X-Canvas-Hash")
	additionalData["X-WebGL-Hash"] = r.Header.Get("X-WebGL-Hash")

	detectionResult := s.botDetection.DetectBot(r.Request, additionalData)

	if detectionResult.IsBot {
		result.ThreatTypes = append(result.ThreatTypes, "bot")
		result.Recommendations = append(result.Recommendations, "Enable bot verification")

		if detectionResult.ShouldBlock {
			result.ShouldBlock = true
			result.ActionTaken = append(result.ActionTaken, "bot_block")
		}

		if detectionResult.ChallengeType != "" {
			result.ShouldChallenge = true
			result.ActionTaken = append(result.ActionTaken, fmt.Sprintf("bot_challenge:%s", detectionResult.ChallengeType))
		}
	}
}

func (s *AdaptiveSecurityService) runHoneypotCheck(ctx context.Context, r *http.Request, result *AdaptiveSecurityResult) {
	honeypotResponse, err := s.honeypot.EvaluateRequest(ctx, r.Request)
	if err != nil {
		return
	}

	if honeypotResponse.IsHoneypot {
		result.HoneypotTriggered = true
		result.ThreatTypes = append(result.ThreatTypes, "honeypot_interaction")
		result.ShouldBlock = true
		result.ActionTaken = append(result.ActionTaken, "honeypot_triggered")
		result.Recommendations = append(result.Recommendations, "Log and investigate attacker")
	}
}

func (s *AdaptiveSecurityService) runAutoResponse(ctx context.Context, r *http.Request, result *AdaptiveSecurityResult) {
	threat := &service.ThreatContext{
		IP:          getClientIP(r),
		ThreatLevel: result.ThreatLevel,
		ThreatTypes: result.ThreatTypes,
		Confidence:  result.Confidence,
	}

	responseResult, err := s.autoResponse.ProcessThreat(ctx, threat)
	if err == nil && responseResult.Success {
		result.ActionTaken = append(result.ActionTaken, fmt.Sprintf("auto_response:%s", responseResult.ActionID))
	}
}

func (s *AdaptiveSecurityService) calculateFinalScore(result *AdaptiveSecurityResult) {
	var totalScore float64
	var count float64

	if result.ThreatIntelScore > 0 {
		totalScore += result.ThreatIntelScore * 0.25
		count++
	}

	if result.DynamicDefenseScore > 0 {
		totalScore += result.DynamicDefenseScore * 0.30
		count++
	}

	if result.AIAttackScore > 0 {
		totalScore += result.AIAttackScore * 0.30
		count++
	}

	if result.HoneypotTriggered {
		totalScore += 100
		count++
	}

	if count > 0 {
		result.RiskScore = totalScore
	}

	if result.RiskScore >= 80 {
		result.ThreatLevel = 5
		result.ShouldBlock = true
	} else if result.RiskScore >= 60 {
		result.ThreatLevel = 4
		result.ShouldChallenge = true
	} else if result.RiskScore >= 40 {
		result.ThreatLevel = 3
		result.ShouldChallenge = true
	} else if result.RiskScore >= 20 {
		result.ThreatLevel = 2
	} else {
		result.ThreatLevel = 1
	}

	result.Confidence = math.Min(count/4.0, 1.0)

	if result.ThreatLevel >= 4 {
		result.Recommendations = append(result.Recommendations, "立即阻止并通知安全团队")
	}
}

func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	if idx := strings.Index(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

func generateSessionID(r *http.Request) string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func GetAdaptiveSecurityService() *AdaptiveSecurityService {
	initAdaptiveSecurityServices()
	return adaptiveSecurityService
}

func (s *AdaptiveSecurityService) GetSecurityStatistics() map[string]interface{} {
	stats := map[string]interface{}{
		"enabled": s.enabled,
		"config": map[string]interface{}{
			"threat_intel_enabled":       s.config.ThreatIntelEnabled,
			"dynamic_defense_enabled":    s.config.DynamicDefenseEnabled,
			"ai_attack_enabled":          s.config.AIAttackEnabled,
			"honeypot_enabled":           s.config.HoneypotEnabled,
			"auto_response_enabled":      s.config.AutoResponseEnabled,
			"bot_detection_enabled":     s.config.BotDetectionEnabled,
			"behavior_analysis_enabled": s.config.BehaviorAnalysisEnabled,
			"integration_mode":           s.config.IntegrationMode,
		},
	}

	if s.threatIntel != nil {
		stats["threat_intelligence"] = s.threatIntel.GetThreatStatistics()
	}

	if s.dynamicDefense != nil {
		stats["dynamic_defense"] = s.dynamicDefense.GetDefenseStatistics()
	}

	if s.aiDetector != nil {
		stats["ai_attack_detector"] = s.aiDetector.GetAttackStatistics()
	}

	if s.honeypot != nil {
		stats["honeypot"] = s.honeypot.GetHoneypotStatistics()
	}

	if s.autoResponse != nil {
		stats["auto_response"] = s.autoResponse.GetResponseStatistics()
	}

	return stats
}

func (s *AdaptiveSecurityService) Enable() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = true
}

func (s *AdaptiveSecurityService) Disable() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = false
}

func (s *AdaptiveSecurityService) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

func (s *AdaptiveSecurityService) UpdateConfig(config *AdaptiveSecurityConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config.ThreatIntelEnabled {
		s.config.ThreatIntelEnabled = config.ThreatIntelEnabled
	}
	if config.DynamicDefenseEnabled {
		s.config.DynamicDefenseEnabled = config.DynamicDefenseEnabled
	}
	if config.AIAttackEnabled {
		s.config.AIAttackEnabled = config.AIAttackEnabled
	}
	if config.HoneypotEnabled {
		s.config.HoneypotEnabled = config.HoneypotEnabled
	}
	if config.AutoResponseEnabled {
		s.config.AutoResponseEnabled = config.AutoResponseEnabled
	}
	if config.BotDetectionEnabled {
		s.config.BotDetectionEnabled = config.BotDetectionEnabled
	}
	if config.BehaviorAnalysisEnabled {
		s.config.BehaviorAnalysisEnabled = config.BehaviorAnalysisEnabled
	}
	if config.IntegrationMode != "" {
		s.config.IntegrationMode = config.IntegrationMode
	}
	if config.ResponseDelay > 0 {
		s.config.ResponseDelay = config.ResponseDelay
	}
	if config.MaxProcessingTime > 0 {
		s.config.MaxProcessingTime = config.MaxProcessingTime
	}
}

func (s *AdaptiveSecurityService) GetConfig() *AdaptiveSecurityConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	configCopy := *s.config
	return &configCopy
}

func (s *AdaptiveSecurityService) GetThreatIntel() *service.ThreatIntelligenceService {
	return s.threatIntel
}

func (s *AdaptiveSecurityService) GetDynamicDefense() *service.DynamicDefenseService {
	return s.dynamicDefense
}

func (s *AdaptiveSecurityService) GetAIAttackDetector() *service.AIAttackDetectorService {
	return s.aiDetector
}

func (s *AdaptiveSecurityService) GetHoneypot() *service.HoneypotService {
	return s.honeypot
}

func (s *AdaptiveSecurityService) GetAutoResponse() *service.AutoResponseService {
	return s.autoResponse
}

func (s *AdaptiveSecurityService) GetBotDetection() *service.BotDetectionService {
	return s.botDetection
}

func (s *AdaptiveSecurityService) ExportConfiguration() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config := map[string]interface{}{
		"adaptive_security": s.config,
		"enabled":           s.enabled,
	}

	return json.MarshalIndent(config, "", "  ")
}

func (s *AdaptiveSecurityService) ImportConfiguration(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	if enabled, ok := config["enabled"].(bool); ok {
		s.enabled = enabled
	}

	return nil
}

func (s *AdaptiveSecurityService) PerformHealthCheck() *HealthCheckResult {
	result := &HealthCheckResult{
		Timestamp: time.Now(),
		Healthy:    true,
		Components: map[string]*ComponentHealth{},
	}

	if s.threatIntel != nil {
		result.Components["threat_intelligence"] = &ComponentHealth{
			Name:   "Threat Intelligence",
			Status: "healthy",
		}
	}

	if s.dynamicDefense != nil {
		result.Components["dynamic_defense"] = &ComponentHealth{
			Name:   "Dynamic Defense",
			Status: "healthy",
		}
	}

	if s.aiDetector != nil {
		result.Components["ai_attack_detector"] = &ComponentHealth{
			Name:   "AI Attack Detector",
			Status: "healthy",
		}
	}

	if s.honeypot != nil {
		result.Components["honeypot"] = &ComponentHealth{
			Name:   "Honeypot",
			Status: "healthy",
		}
	}

	if s.autoResponse != nil {
		result.Components["auto_response"] = &ComponentHealth{
			Name:   "Auto Response",
			Status: "healthy",
		}
	}

	return result
}

type HealthCheckResult struct {
	Timestamp  time.Time
	Healthy    bool
	Components map[string]*ComponentHealth
}

type ComponentHealth struct {
	Name   string
	Status string
	Error  string
}

var stringsReplacer = strings.NewReplacer()

func init() {
	_ = stringsReplacer
}
