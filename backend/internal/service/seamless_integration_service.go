package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

type SeamlessIntegrationService struct {
	enhancedService *EnhancedSeamlessService
	originalService *SeamlessOptimizationService
	analytics       *SeamlessAnalytics
	config          *IntegrationConfig
	mu              sync.RWMutex
}

type IntegrationConfig struct {
	EnableEnhancedMode       bool
	HybridMode               bool
	FallbackToOriginal       bool
	EnableAnalytics          bool
	EnableAutoTuning         bool
	PerformanceThreshold     time.Duration
	MaxResponseTime          time.Duration
}

type SeamlessAnalytics struct {
	totalRequests      int64
	enhancedRequests   int64
	originalRequests   int64
	skipRate          float64
	challengeRate     float64
	blockRate         float64
	avgResponseTime   time.Duration
	dimensionAccuracy map[string]float64
	mu                sync.RWMutex
}

type SeamlessVerificationRequest struct {
	SessionID         string                 `json:"session_id" binding:"required"`
	DeviceFingerprint string                 `json:"device_fingerprint" binding:"required"`
	ApplicationID     uint                   `json:"application_id"`
	UserID            *uint                  `json:"user_id,omitempty"`
	BehaviorData      []models.BehaviorData  `json:"behavior_data,omitempty"`
	EnvironmentData   map[string]interface{} `json:"environment_data,omitempty"`
	IPAddress         string                 `json:"ip_address,omitempty"`
	UserAgent         string                 `json:"user_agent,omitempty"`
	UseEnhanced       bool                   `json:"use_enhanced,omitempty"`
}

type SeamlessVerificationResponse struct {
	Decision         string                    `json:"decision"`
	RiskScore        float64                   `json:"risk_score"`
	Reason           string                    `json:"reason,omitempty"`
	Token            string                    `json:"token,omitempty"`
	TrustLevel       float64                   `json:"trust_level"`
	Confidence       float64                   `json:"confidence"`
	ShouldChallenge  bool                     `json:"should_challenge"`
	SkipReason       string                    `json:"skip_reason,omitempty"`
	Optimizations    []string                  `json:"optimizations,omitempty"`
	EnhancedResult   *EnhancedSeamlessResult   `json:"enhanced_result,omitempty"`
	ProcessingTime   int64                     `json:"processing_time_ms"`
	ModeUsed         string                    `json:"mode_used"`
}

func NewSeamlessIntegrationService() *SeamlessIntegrationService {
	return &SeamlessIntegrationService{
		enhancedService: NewEnhancedSeamlessService(),
		originalService: NewSeamlessOptimizationService(),
		analytics: &SeamlessAnalytics{
			dimensionAccuracy: make(map[string]float64),
		},
		config: &IntegrationConfig{
			EnableEnhancedMode:   true,
			HybridMode:           true,
			FallbackToOriginal:   true,
			EnableAnalytics:      true,
			EnableAutoTuning:     true,
			PerformanceThreshold: 100 * time.Millisecond,
			MaxResponseTime:      500 * time.Millisecond,
		},
	}
}

func (s *SeamlessIntegrationService) ProcessVerification(req *SeamlessVerificationRequest) (*SeamlessVerificationResponse, error) {
	startTime := time.Now()
	
	if req.UseEnhanced || s.config.EnableEnhancedMode {
		return s.processEnhancedVerification(req, startTime)
	}
	
	return s.processOriginalVerification(req, startTime)
}

func (s *SeamlessIntegrationService) processEnhancedVerification(req *SeamlessVerificationRequest, startTime time.Time) (*SeamlessVerificationResponse, error) {
	s.analytics.mu.Lock()
	s.analytics.enhancedRequests++
	s.analytics.totalRequests++
	s.analytics.mu.Unlock()
	
	var userID string
	if req.UserID != nil {
		userID = fmt.Sprintf("%d", *req.UserID)
	}
	
	previousRiskScore := s.estimateInitialRiskScore(req)
	
	enhancedResult, err := s.enhancedService.OptimizeVerification(
		userID,
		req.DeviceFingerprint,
		req.BehaviorData,
		req.EnvironmentData,
		previousRiskScore,
	)
	
	if err != nil {
		if s.config.FallbackToOriginal {
			return s.processOriginalVerification(req, startTime)
		}
		return nil, err
	}
	
	processingTime := time.Since(startTime)
	
	if s.config.EnableAutoTuning && processingTime > s.config.PerformanceThreshold {
		s.adjustPerformanceConfig()
	}
	
	s.recordAnalytics(req, enhancedResult, processingTime, "enhanced")
	
	response := &SeamlessVerificationResponse{
		Decision:        s.determineDecision(enhancedResult),
		RiskScore:       enhancedResult.FinalRiskScore,
		TrustLevel:      enhancedResult.TrustLevel,
		Confidence:      enhancedResult.Confidence,
		ShouldChallenge: enhancedResult.ShouldChallenge,
		SkipReason:      enhancedResult.SkipReason,
		Optimizations:   enhancedResult.OptimizationApplied,
		EnhancedResult:  enhancedResult,
		ProcessingTime:  processingTime.Milliseconds(),
		ModeUsed:        "enhanced",
	}
	
	return response, nil
}

func (s *SeamlessIntegrationService) processOriginalVerification(req *SeamlessVerificationRequest, startTime time.Time) (*SeamlessVerificationResponse, error) {
	s.analytics.mu.Lock()
	s.analytics.originalRequests++
	s.analytics.totalRequests++
	s.analytics.mu.Unlock()
	
	var userID string
	if req.UserID != nil {
		userID = fmt.Sprintf("%d", *req.UserID)
	}
	
	previousRiskScore := s.estimateInitialRiskScore(req)
	
	originalResult, err := s.originalService.OptimizeSeamlessVerification(
		userID,
		req.DeviceFingerprint,
		req.BehaviorData,
		req.EnvironmentData,
		previousRiskScore,
	)
	
	if err != nil {
		return nil, err
	}
	
	processingTime := time.Since(startTime)
	s.recordAnalytics(req, nil, processingTime, "original")
	
	response := &SeamlessVerificationResponse{
		Decision:        s.determineOriginalDecision(originalResult),
		RiskScore:       originalResult.FinalRiskScore,
		ShouldChallenge: originalResult.ShouldChallenge,
		SkipReason:      originalResult.SkipReason,
		Optimizations:   originalResult.OptimizationApplied,
		ProcessingTime:  processingTime.Milliseconds(),
		ModeUsed:        "original",
	}
	
	return response, nil
}

func (s *SeamlessIntegrationService) estimateInitialRiskScore(req *SeamlessVerificationRequest) float64 {
	riskScore := 50.0
	
	if req.UserAgent != "" {
		if containsAnySubstr(req.UserAgent, []string{"bot", "crawler", "spider"}) {
			riskScore += 30
		}
	}
	
	if val, ok := req.EnvironmentData["proxy_detected"].(bool); ok && val {
		riskScore += 25
	}
	
	if val, ok := req.EnvironmentData["tor_exit"].(bool); ok && val {
		riskScore += 30
	}
	
	if val, ok := req.EnvironmentData["vpn_detected"].(bool); ok && val {
		riskScore += 15
	}
	
	if val, ok := req.EnvironmentData["risk_score"].(float64); ok {
		riskScore = (riskScore + val) / 2
	}
	
	return math.Max(0, math.Min(100, riskScore))
}

func (s *SeamlessIntegrationService) determineDecision(result *EnhancedSeamlessResult) string {
	if result.FinalRiskScore >= 70 {
		return "block"
	}
	
	if result.FinalRiskScore < 30 && result.TrustLevel > 0.7 {
		return "allow"
	}
	
	return "challenge"
}

func (s *SeamlessIntegrationService) determineOriginalDecision(result *SeamlessOptimizationResult) string {
	if result.FinalRiskScore >= 70 {
		return "block"
	}
	
	if !result.ShouldChallenge {
		return "allow"
	}
	
	return "challenge"
}

func (s *SeamlessIntegrationService) recordAnalytics(req *SeamlessVerificationRequest, result *EnhancedSeamlessResult, processingTime time.Duration, mode string) {
	s.analytics.mu.Lock()
	defer s.analytics.mu.Unlock()
	
	s.analytics.avgResponseTime = (s.analytics.avgResponseTime + processingTime) / 2
	
	if result != nil {
		if !result.ShouldChallenge {
			s.analytics.skipRate++
		} else {
			s.analytics.challengeRate++
		}
		
		if result.FinalRiskScore >= 70 {
			s.analytics.blockRate++
		}
		
		if result.DimensionScores != nil {
			for dim := range result.DimensionScores {
				if _, exists := s.analytics.dimensionAccuracy[dim]; !exists {
					s.analytics.dimensionAccuracy[dim] = 0.7
				}
			}
		}
	}
}

func (s *SeamlessIntegrationService) adjustPerformanceConfig() {
	s.config.EnableAutoTuning = false
	s.config.EnableAnalytics = true
}

func (s *SeamlessIntegrationService) UpdateVerificationResult(req *SeamlessVerificationRequest, success bool, responseTime time.Duration) {
	var userID string
	if req.UserID != nil {
		userID = fmt.Sprintf("%d", *req.UserID)
	}
	
	s.enhancedService.UpdateLearningFromResult(userID, req.DeviceFingerprint, success, responseTime)
	
	s.enhancedService.RecordChallengeResult(userID, false, success)
}

func (s *SeamlessIntegrationService) GetAnalytics() *SeamlessAnalyticsData {
	s.analytics.mu.Lock()
	defer s.analytics.mu.Unlock()
	
	total := float64(s.analytics.totalRequests)
	if total == 0 {
		total = 1
	}
	
	return &SeamlessAnalyticsData{
		TotalRequests:      s.analytics.totalRequests,
		EnhancedRequests:   s.analytics.enhancedRequests,
		OriginalRequests:   s.analytics.originalRequests,
		SkipRate:           float64(s.analytics.skipRate) / total,
		ChallengeRate:      float64(s.analytics.challengeRate) / total,
		BlockRate:          float64(s.analytics.blockRate) / total,
		AvgResponseTime:    s.analytics.avgResponseTime,
		DimensionAccuracy: s.analytics.dimensionAccuracy,
	}
}

type SeamlessAnalyticsData struct {
	TotalRequests      int64              `json:"total_requests"`
	EnhancedRequests   int64              `json:"enhanced_requests"`
	OriginalRequests   int64              `json:"original_requests"`
	SkipRate           float64            `json:"skip_rate"`
	ChallengeRate      float64            `json:"challenge_rate"`
	BlockRate          float64            `json:"block_rate"`
	AvgResponseTime    time.Duration      `json:"avg_response_time"`
	DimensionAccuracy  map[string]float64 `json:"dimension_accuracy"`
}

func (s *SeamlessIntegrationService) GetTrustBreakdown(userID string, deviceFingerprint string) map[string]interface{} {
	return s.enhancedService.GetTrustScoreBreakdown(userID, deviceFingerprint)
}

func (s *SeamlessIntegrationService) SetUserPreference(userID string, pref *UserDisturbanceProfile) {
	s.enhancedService.SetUserPreference(userID, pref)
}

func (s *SeamlessIntegrationService) GetUserPreference(userID string) *UserDisturbanceProfile {
	return s.enhancedService.GetUserPreference(userID)
}

func (s *SeamlessIntegrationService) OptimizeThresholds() map[string]float64 {
	return s.enhancedService.OptimizeDisturbanceThresholds()
}

func (s *SeamlessIntegrationService) CleanupOldData(maxAge time.Duration) int {
	return s.enhancedService.CleanupOldData(maxAge)
}

func (s *SeamlessIntegrationService) ValidateFingerprint(fingerprint string) (bool, float64) {
	return s.enhancedService.ValidateFingerprintStability(fingerprint, 3, 30*24*time.Hour)
}

func (s *SeamlessIntegrationService) GetConfig() *IntegrationConfig {
	return s.config
}

func (s *SeamlessIntegrationService) UpdateConfig(newConfig *IntegrationConfig) {
	s.config = newConfig
}

func (s *SeamlessIntegrationService) GenerateFingerprintFromComponents(components *EnhancedFingerprintComponents) string {
	return s.enhancedService.GenerateEnhancedFingerprint(components)
}

func (s *SeamlessIntegrationService) ExportAnalyticsReport() string {
	data := s.GetAnalytics()
	report := map[string]interface{}{
		"generated_at":           time.Now().Format(time.RFC3339),
		"total_verifications":    data.TotalRequests,
		"enhanced_mode_usage":    float64(data.EnhancedRequests) / float64(math.Max(1, float64(data.TotalRequests))) * 100,
		"skip_rate":              data.SkipRate * 100,
		"challenge_rate":         data.ChallengeRate * 100,
		"block_rate":             data.BlockRate * 100,
		"average_response_time":  data.AvgResponseTime.String(),
		"dimension_accuracy":     data.DimensionAccuracy,
	}
	
	jsonBytes, _ := json.MarshalIndent(report, "", "  ")
	return string(jsonBytes)
}

func containsAnySubstr(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) && 
		   (s == substr || len(s) > len(substr) && 
		    (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr)) {
			return true
		}
	}
	return false
}

func (s *SeamlessIntegrationService) CompareModePerformance(requests []SeamlessVerificationRequest) map[string]interface{} {
	enhancedTimes := make([]time.Duration, 0, len(requests))
	originalTimes := make([]time.Duration, 0, len(requests))
	
	for _, req := range requests {
		start := time.Now()
		if req.UseEnhanced || s.config.EnableEnhancedMode {
			s.processEnhancedVerification(&req, start)
		} else {
			s.processOriginalVerification(&req, start)
		}
		
		if req.UseEnhanced {
			enhancedTimes = append(enhancedTimes, time.Since(start))
		} else {
			originalTimes = append(originalTimes, time.Since(start))
		}
	}
	
	result := make(map[string]interface{})
	
	if len(enhancedTimes) > 0 {
		avgEnhanced := calculateAverageDuration(enhancedTimes)
		result["enhanced_avg_time"] = avgEnhanced.String()
		result["enhanced_sample_size"] = len(enhancedTimes)
	}
	
	if len(originalTimes) > 0 {
		avgOriginal := calculateAverageDuration(originalTimes)
		result["original_avg_time"] = avgOriginal.String()
		result["original_sample_size"] = len(originalTimes)
	}
	
	return result
}

func calculateAverageDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	var sum int64
	for _, d := range durations {
		sum += d.Nanoseconds()
	}
	
	return time.Duration(sum / int64(len(durations)))
}
