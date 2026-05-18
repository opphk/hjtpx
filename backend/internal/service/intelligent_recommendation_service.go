package service

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type IntelligentRecommendationService struct {
	userProfiles map[string]*CaptchaUserProfile
	captchaStats map[string]*CaptchaMethodStats
	abTestIntegration *ABTestIntegration
	recommendationEngine *RecommendationEngine
	mu sync.RWMutex
}

type CaptchaUserProfile struct {
	UserID           string
	DeviceInfo       DeviceInfo
	BehaviorPatterns []CaptchaBehaviorPattern
	SuccessHistory   []MethodSuccess
	PreferredMethods []string
	AvgResponseTime  float64
	SuccessRate      float64
	RiskLevel        string
	LastUpdated      time.Time
	UserSegments     []string
}

type DeviceInfo struct {
	DeviceType     string
	OS             string
	Browser        string
	IsMobile       bool
	ScreenSize     string
	TouchEnabled   bool
	NetworkType    string
}

type CaptchaBehaviorPattern struct {
	PatternType    string
	AvgSpeed       float64
	AvgAccuracy    float64
	CommonMistakes []string
	TimeOfDay      int
	SuccessRate    float64
}

type MethodSuccess struct {
	Method    string
	Attempts  int
	Successes int
	AvgTime   float64
	LastUsed  time.Time
}

type CaptchaMethodStats struct {
	Method       string
	TotalAttempts int
	TotalSuccesses int
	AvgTime       float64
	UserRatings   []float64
	SuccessRate   float64
	BestForDevices map[string]float64
	BestForUsers   map[string]float64
}

type ABTestIntegration struct {
	activeTests map[string]*ABTestRecommendation
	mu         sync.RWMutex
}

type ABTestRecommendation struct {
	TestID        string
	VariantID     string
	Recommended   string
	ConversionRate float64
	Traffic       float64
}

type RecommendationEngine struct {
	modelWeights map[string]float64
	featureImportance map[string]float64
	mu           sync.RWMutex
}

type RecommendationRequest struct {
	UserID         string
	DeviceFingerprint string
	ApplicationID   uint
	Context         *RecommendationContext
	UserProfile    *CaptchaUserProfile
}

type RecommendationContext struct {
	Action         string
	SessionID      string
	PreviousMethod  string
	FailureCount   int
	TimeOfDay      int
	IsHighRisk     bool
	UserAgent      string
	IPAddress      string
}

type RecommendationResult struct {
	RecommendedMethod  string
	Confidence         float64
	AlternativeMethods []AlternativeRecommendation
	UserProfileMatch  float64
	ABTestVariant      string
	Reasoning         string
	EstimatedTime      float64
	EstimatedSuccessRate float64
	PersonalizedTips   []string
}

type AlternativeRecommendation struct {
	Method          string
	Score           float64
	Reason          string
	Pros            []string
	Cons            []string
}

func NewIntelligentRecommendationService() *IntelligentRecommendationService {
	service := &IntelligentRecommendationService{
		userProfiles: make(map[string]*CaptchaUserProfile),
		captchaStats: make(map[string]*CaptchaMethodStats),
		abTestIntegration: &ABTestIntegration{
			activeTests: make(map[string]*ABTestRecommendation),
		},
		recommendationEngine: &RecommendationEngine{
			modelWeights: map[string]float64{
				"device_compatibility": 0.25,
				"user_preference":     0.25,
				"success_rate":        0.20,
				"response_time":       0.15,
				"risk_level":          0.15,
			},
			featureImportance: map[string]float64{
				"mobile_optimized":  1.5,
				"touch_support":    1.3,
				"low_bandwidth":    1.2,
				"high_security":    1.0,
				"user_familiarity": 1.8,
				"recent_success":    2.0,
			},
		},
	}
	
	service.initializeDefaultMethods()
	return service
}

func (s *IntelligentRecommendationService) initializeDefaultMethods() {
	defaultMethods := []string{
		"slider", "click", "3d_rotate", "3d_click",
		"lianliankan", "voice", "seamless", "biometrics",
	}
	
	for _, method := range defaultMethods {
		s.captchaStats[method] = &CaptchaMethodStats{
			Method:           method,
			BestForDevices:  make(map[string]float64),
			BestForUsers:    make(map[string]float64),
			UserRatings:     []float64{},
		}
	}
}

func (s *IntelligentRecommendationService) GetRecommendation(req *RecommendationRequest) *RecommendationResult {
	
	profile := s.getOrCreateUserProfile(req.UserID)
	
	abTestVariant := s.getABTestRecommendation(req.Context)
	
	scores := s.calculateMethodScores(profile, req.Context)
	
	sortedMethods := s.sortMethodsByScore(scores)
	
	recommended := sortedMethods[0]
	alternatives := s.generateAlternatives(sortedMethods[1:4], profile)
	
	result := &RecommendationResult{
		RecommendedMethod:    recommended.Method,
		Confidence:           recommended.TotalScore,
		AlternativeMethods:   alternatives,
		UserProfileMatch:    s.calculateProfileMatch(profile, recommended.Method),
		ABTestVariant:       abTestVariant,
		EstimatedTime:       s.estimateResponseTime(recommended.Method, profile),
		EstimatedSuccessRate: s.estimateSuccessRate(recommended.Method, profile),
		PersonalizedTips:    s.generateTips(recommended.Method, profile),
	}
	
	result.Reasoning = s.generateReasoning(result, profile)
	
	return result
}

func (s *IntelligentRecommendationService) getOrCreateUserProfile(userID string) *CaptchaUserProfile {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if profile, exists := s.userProfiles[userID]; exists {
		return profile
	}
	
	profile := &CaptchaUserProfile{
		UserID:           userID,
		SuccessRate:      80.0,
		AvgResponseTime: 5.0,
		RiskLevel:       "medium",
		UserSegments:    []string{"new_user"},
	}
	s.userProfiles[userID] = profile
	return profile
}

func (s *IntelligentRecommendationService) UpdateUserProfile(
	userID string,
	method string,
	success bool,
	responseTime float64,
	deviceInfo *DeviceInfo,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	profile := s.userProfiles[userID]
	if profile == nil {
		profile = s.getOrCreateUserProfile(userID)
	}
	
	found := false
	for i := range profile.SuccessHistory {
		if profile.SuccessHistory[i].Method == method {
			profile.SuccessHistory[i].Attempts++
			if success {
				profile.SuccessHistory[i].Successes++
			}
			profile.SuccessHistory[i].AvgTime = 
				profile.SuccessHistory[i].AvgTime*0.9 + responseTime*0.1
			profile.SuccessHistory[i].LastUsed = time.Now()
			found = true
			break
		}
	}
	
	if !found {
		profile.SuccessHistory = append(profile.SuccessHistory, MethodSuccess{
			Method:    method,
			Attempts:  1,
			Successes: 0,
			AvgTime:   responseTime,
			LastUsed:  time.Now(),
		})
		if success {
			profile.SuccessHistory[len(profile.SuccessHistory)-1].Successes = 1
		}
	}
	
	profile.AvgResponseTime = profile.AvgResponseTime*0.9 + responseTime*0.1
	profile.SuccessRate = s.calculateOverallSuccessRate(profile)
	profile.LastUpdated = time.Now()
	
	if deviceInfo != nil {
		profile.DeviceInfo = *deviceInfo
		profile.UserSegments = s.determineUserSegments(profile)
	}
	
	for _, pref := range profile.PreferredMethods {
		if pref == method {
			return
		}
	}
	
	if success && responseTime < profile.AvgResponseTime {
		profile.PreferredMethods = append([]string{method}, profile.PreferredMethods...)
		if len(profile.PreferredMethods) > 5 {
			profile.PreferredMethods = profile.PreferredMethods[:5]
		}
	}
	
	s.updateCaptchaStats(method, success, responseTime)
}

func (s *IntelligentRecommendationService) calculateMethodScores(
	profile *CaptchaUserProfile,
	context *RecommendationContext,
) map[string]MethodScore {
	
	scores := make(map[string]MethodScore)
	
	for method := range s.captchaStats {
		score := MethodScore{Method: method}
		
		deviceScore := s.calculateDeviceCompatibility(method, &profile.DeviceInfo)
		deviceScore *= s.recommendationEngine.modelWeights["device_compatibility"]
		score.DeviceScore = deviceScore
		
		preferenceScore := s.calculateUserPreference(method, profile)
		preferenceScore *= s.recommendationEngine.modelWeights["user_preference"]
		score.PreferenceScore = preferenceScore
		
		successScore := s.calculateHistoricalSuccess(method, profile)
		successScore *= s.recommendationEngine.modelWeights["success_rate"]
		score.HistoricalScore = successScore
		
		timeScore := s.calculateResponseTimeScore(method, profile)
		timeScore *= s.recommendationEngine.modelWeights["response_time"]
		score.TimeScore = timeScore
		
		riskScore := s.calculateRiskScore(method, context)
		riskScore *= s.recommendationEngine.modelWeights["risk_level"]
		score.RiskScore = riskScore
		
		score.TotalScore = score.DeviceScore + score.PreferenceScore + 
			score.HistoricalScore + score.TimeScore + score.RiskScore
		
		if context != nil && context.PreviousMethod == method {
			score.TotalScore *= 0.7
		}
		
		scores[method] = score
	}
	
	return scores
}

func (s *IntelligentRecommendationService) calculateDeviceCompatibility(
	method string,
	device *DeviceInfo,
) float64 {
	
	score := 0.5
	
	switch method {
	case "slider":
		if device.TouchEnabled && device.IsMobile {
			score = 0.9
		} else if !device.IsMobile {
			score = 0.95
		}
	case "click", "3d_rotate":
		if device.TouchEnabled {
			score = 0.85
		} else {
			score = 0.9
		}
	case "voice":
		if device.IsMobile {
			score = 0.95
		} else {
			score = 0.7
		}
	case "3d_click", "lianliankan":
		if device.IsMobile && !device.TouchEnabled {
			score = 0.3
		} else {
			score = 0.85
		}
	case "seamless":
		if profile := s.getUserProfileByDevice(device); profile != nil {
			score = 0.95
		}
	case "biometrics":
		if device.DeviceType == "mobile" || device.DeviceType == "tablet" {
			score = 0.95
		} else {
			score = 0.4
		}
	}
	
	return score
}

func (s *IntelligentRecommendationService) calculateUserPreference(
	method string,
	profile *CaptchaUserProfile,
) float64 {
	
	if len(profile.PreferredMethods) == 0 {
		return 0.5
	}
	
	for i, pref := range profile.PreferredMethods {
		if pref == method {
			return 1.0 - float64(i)*0.15
		}
	}
	
	return 0.3
}

func (s *IntelligentRecommendationService) calculateHistoricalSuccess(
	method string,
	profile *CaptchaUserProfile,
) float64 {
	
	for _, success := range profile.SuccessHistory {
		if success.Method == method && success.Attempts > 0 {
			successRate := float64(success.Successes) / float64(success.Attempts)
			timeBonus := 0.0
			if success.AvgTime < profile.AvgResponseTime {
				timeBonus = 0.1
			}
			return math.Min(1.0, successRate+timeBonus)
		}
	}
	
	return 0.5
}

func (s *IntelligentRecommendationService) calculateResponseTimeScore(
	method string,
	profile *CaptchaUserProfile,
) float64 {
	
	stats := s.captchaStats[method]
	if stats == nil || stats.AvgTime == 0 {
		return 0.5
	}
	
	profileAvgTime := profile.AvgResponseTime
	methodAvgTime := stats.AvgTime
	
	if methodAvgTime < profileAvgTime {
		return 0.9
	} else if methodAvgTime < profileAvgTime*1.5 {
		return 0.7
	}
	return 0.5
}

func (s *IntelligentRecommendationService) calculateRiskScore(
	method string,
	context *RecommendationContext,
) float64 {
	
	if context == nil || !context.IsHighRisk {
		return 0.5
	}
	
	highSecurityMethods := map[string]float64{
		"biometrics":   0.9,
		"3d_click":     0.85,
		"lianliankan":  0.8,
		"voice":        0.75,
		"3d_rotate":    0.7,
	}
	
	if score, exists := highSecurityMethods[method]; exists {
		return score
	}
	
	return 0.6
}

func (s *IntelligentRecommendationService) getABTestRecommendation(
	context *RecommendationContext,
) string {
	
	s.abTestIntegration.mu.RLock()
	defer s.abTestIntegration.mu.RUnlock()
	
	if context == nil || context.SessionID == "" {
		return ""
	}
	
	for _, test := range s.abTestIntegration.activeTests {
		if test.Traffic > 0.9 {
			return test.Recommended
		}
	}
	
	return ""
}

func (s *IntelligentRecommendationService) sortMethodsByScore(
	scores map[string]MethodScore,
) []MethodScore {
	
	result := make([]MethodScore, 0, len(scores))
	for _, score := range scores {
		result = append(result, score)
	}
	
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].TotalScore > result[i].TotalScore {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	
	return result
}

func (s *IntelligentRecommendationService) generateAlternatives(
	methods []MethodScore,
	profile *CaptchaUserProfile,
) []AlternativeRecommendation {
	
	alts := make([]AlternativeRecommendation, 0, len(methods))
	
	for _, m := range methods {
		alt := AlternativeRecommendation{
			Method: m.Method,
			Score:  m.TotalScore,
			Pros:   s.getMethodPros(m.Method, profile),
			Cons:   s.getMethodCons(m.Method, profile),
		}
		alts = append(alts, alt)
	}
	
	return alts
}

func (s *IntelligentRecommendationService) getMethodPros(
	method string,
	profile *CaptchaUserProfile,
) []string {
	
	pros := []string{}
	
	switch method {
	case "slider":
		pros = append(pros, "操作简单直观", "适合多种设备")
	case "click":
		pros = append(pros, "快速完成", "用户熟悉度高")
	case "3d_rotate":
		pros = append(pros, "安全性较高", "视觉效果好")
	case "voice":
		pros = append(pros, "适合视障用户", "移动端体验佳")
	case "seamless":
		pros = append(pros, "无感知验证", "用户体验最佳")
	}
	
	return pros
}

func (s *IntelligentRecommendationService) getMethodCons(
	method string,
	profile *CaptchaUserProfile,
) []string {
	
	cons := []string{}
	
	switch method {
	case "slider":
		cons = append(cons, "可能需要精确操作")
	case "click":
		cons = append(cons, "安全性相对较低")
	case "3d_rotate":
		cons = append(cons, "耗时可能较长")
	case "voice":
		cons = append(cons, "环境噪音可能影响")
	case "seamless":
		cons = append(cons, "需要足够的行为数据")
	}
	
	return cons
}

func (s *IntelligentRecommendationService) calculateProfileMatch(
	profile *CaptchaUserProfile,
	method string,
) float64 {
	
	match := 0.5
	
	for _, pref := range profile.PreferredMethods {
		if pref == method {
			match += 0.3
			break
		}
	}
	
	device := profile.DeviceInfo
	if device.IsMobile && (method == "voice" || method == "seamless") {
		match += 0.2
	}
	
	for _, segment := range profile.UserSegments {
		if segment == "premium" && method == "biometrics" {
			match += 0.1
		}
		if segment == "new_user" && method == "click" {
			match += 0.1
		}
	}
	
	return math.Min(1.0, match)
}

func (s *IntelligentRecommendationService) estimateResponseTime(
	method string,
	profile *CaptchaUserProfile,
) float64 {
	
	stats := s.captchaStats[method]
	if stats != nil && stats.AvgTime > 0 {
		return stats.AvgTime * 0.8
	}
	
	defaultTimes := map[string]float64{
		"slider":      4.0,
		"click":       3.0,
		"3d_rotate":   6.0,
		"3d_click":    5.0,
		"lianliankan": 8.0,
		"voice":       5.0,
		"seamless":    0.0,
		"biometrics":  2.0,
	}
	
	if t, exists := defaultTimes[method]; exists {
		return t
	}
	return 5.0
}

func (s *IntelligentRecommendationService) estimateSuccessRate(
	method string,
	profile *CaptchaUserProfile,
) float64 {
	
	for _, success := range profile.SuccessHistory {
		if success.Method == method && success.Attempts >= 3 {
			return float64(success.Successes) / float64(success.Attempts) * 100
		}
	}
	
	stats := s.captchaStats[method]
	if stats != nil && stats.TotalAttempts > 100 {
		return stats.SuccessRate
	}
	
	return 85.0
}

func (s *IntelligentRecommendationService) generateTips(
	method string,
	profile *CaptchaUserProfile,
) []string {
	
	tips := []string{}
	
	switch method {
	case "slider":
		tips = append(tips, "保持匀速滑动效果更佳")
		if profile.DeviceInfo.IsMobile {
			tips = append(tips, "移动端建议双手操作")
		}
	case "click":
		tips = append(tips, "注意图片中的所有目标")
	case "3d_rotate":
		tips = append(tips, "仔细观察旋转后的图片")
	case "voice":
		tips = append(tips, "在安静环境中效果更好")
	case "seamless":
		tips = append(tips, "正常使用即可，系统会自动验证")
	}
	
	return tips
}

func (s *IntelligentRecommendationService) generateReasoning(
	result *RecommendationResult,
	profile *CaptchaUserProfile,
) string {
	
	reasoning := fmt.Sprintf("基于您的使用历史和设备信息，推荐使用 %s 验证码。", result.RecommendedMethod)
	
	if len(profile.PreferredMethods) > 0 && profile.PreferredMethods[0] == result.RecommendedMethod {
		reasoning += " 您之前使用此方法的成功率很高。"
	}
	
	if profile.DeviceInfo.IsMobile && (result.RecommendedMethod == "voice" || result.RecommendedMethod == "seamless") {
		reasoning += " 此方法在移动设备上体验最佳。"
	}
	
	reasoning += fmt.Sprintf(" 预计完成时间: %.1f 秒，成功率: %.1f%%。", 
		result.EstimatedTime, result.EstimatedSuccessRate)
	
	return reasoning
}

func (s *IntelligentRecommendationService) calculateOverallSuccessRate(
	profile *CaptchaUserProfile,
) float64 {
	
	if len(profile.SuccessHistory) == 0 {
		return 80.0
	}
	
	totalSuccess := 0
	totalAttempts := 0
	
	for _, success := range profile.SuccessHistory {
		totalSuccess += success.Successes
		totalAttempts += success.Attempts
	}
	
	if totalAttempts == 0 {
		return 80.0
	}
	
	return float64(totalSuccess) / float64(totalAttempts) * 100
}

func (s *IntelligentRecommendationService) determineUserSegments(
	profile *CaptchaUserProfile,
) []string {
	
	segments := []string{}
	
	if len(profile.SuccessHistory) < 5 {
		segments = append(segments, "new_user")
	} else if profile.SuccessRate > 95 {
		segments = append(segments, "expert_user")
	}
	
	if profile.AvgResponseTime < 3.0 {
		segments = append(segments, "power_user")
	}
	
	if profile.DeviceInfo.IsMobile {
		segments = append(segments, "mobile_user")
	} else {
		segments = append(segments, "desktop_user")
	}
	
	return segments
}

func (s *IntelligentRecommendationService) updateCaptchaStats(
	method string,
	success bool,
	responseTime float64,
) {
	
	stats := s.captchaStats[method]
	if stats == nil {
		stats = &CaptchaMethodStats{
			Method:           method,
			BestForDevices:  make(map[string]float64),
			BestForUsers:    make(map[string]float64),
			UserRatings:     []float64{},
		}
		s.captchaStats[method] = stats
	}
	
	stats.TotalAttempts++
	if success {
		stats.TotalSuccesses++
	}
	stats.AvgTime = stats.AvgTime*0.9 + responseTime*0.1
	stats.SuccessRate = float64(stats.TotalSuccesses) / float64(stats.TotalAttempts) * 100
}

func (s *IntelligentRecommendationService) getUserProfileByDevice(
	device *DeviceInfo,
) *CaptchaUserProfile {
	return nil
}

type MethodScore struct {
	Method          string
	DeviceScore     float64
	PreferenceScore float64
	HistoricalScore float64
	TimeScore       float64
	RiskScore       float64
	TotalScore      float64
}

func (s *IntelligentRecommendationService) GetTopRecommendations(
	userID string,
	limit int,
) []string {
	
	profile := s.getOrCreateUserProfile(userID)
	
	req := &RecommendationRequest{
		UserID: userID,
		Context: &RecommendationContext{},
	}
	
	result := s.GetRecommendation(req)
	
	methods := []string{result.RecommendedMethod}
	for _, alt := range result.AlternativeMethods {
		if len(methods) >= limit {
			break
		}
		methods = append(methods, alt.Method)
	}
	
	if len(profile.PreferredMethods) > len(methods) {
		lenToUse := len(profile.PreferredMethods)
		if lenToUse > limit {
			lenToUse = limit
		}
		methods = profile.PreferredMethods[:lenToUse]
	}
	
	return methods
}

func (s *IntelligentRecommendationService) GetMethodStatistics(method string) *CaptchaMethodStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.captchaStats[method]
}

func (s *IntelligentRecommendationService) SetABTestVariant(
	testID string,
	variantID string,
	recommended string,
	traffic float64,
) {
	s.abTestIntegration.mu.Lock()
	defer s.abTestIntegration.mu.Unlock()
	
	s.abTestIntegration.activeTests[testID] = &ABTestRecommendation{
		TestID:          testID,
		VariantID:       variantID,
		Recommended:     recommended,
		Traffic:         traffic,
		ConversionRate:  0.0,
	}
}
