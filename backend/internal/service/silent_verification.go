package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/redis"
)

type SilentVerificationConfig struct {
	Enabled              bool    `json:"enabled"`
	RiskThreshold        float64 `json:"risk_threshold"`
	MinBehaviorDataPoints int     `json:"min_behavior_data_points"`
	MaxVerifyDuration    int64   `json:"max_verify_duration"`
	EnableDeviceCheck    bool    `json:"enable_device_check"`
	EnableBehaviorCheck  bool    `json:"enable_behavior_check"`
	EnableHistoryCheck   bool    `json:"enable_history_check"`
	CacheTTL             int64   `json:"cache_ttl"`
}

type VerificationStrategy struct {
	Level       string   `json:"level"`
	RiskScore   float64  `json:"risk_score"`
	Reasons     []string `json:"reasons"`
	Suggestions []string `json:"suggestions"`
	NeedCaptcha bool     `json:"need_captcha"`
	CaptchaType string   `json:"captcha_type"`
	WaitTime    int      `json:"wait_time"`
}

type DeviceTrustScore struct {
	DeviceFingerprint string  `json:"device_fingerprint"`
	FingerprintMatch  float64 `json:"fingerprint_match"`
	HistoricalScore   float64 `json:"historical_score"`
	LoginFrequency    float64 `json:"login_frequency"`
	GeographicScore   float64 `json:"geographic_score"`
	TotalScore        float64 `json:"total_score"`
}

type BehaviorTrustScore struct {
	MouseTrajectoryScore float64 `json:"mouse_trajectory_score"`
	ClickPatternScore    float64 `json:"click_pattern_score"`
	KeyboardPatternScore float64 `json:"keyboard_pattern_score"`
	ScrollBehaviorScore  float64 `json:"scroll_behavior_score"`
	TouchBehaviorScore   float64 `json:"touch_behavior_score"`
	TotalScore           float64 `json:"total_score"`
}

type HistoryTrustScore struct {
	AccountHistoryScore float64 `json:"account_history_score"`
	LoginFrequencyScore float64 `json:"login_frequency_score"`
	OperationHabitScore float64 `json:"operation_habit_score"`
	CommonDeviceScore   float64 `json:"common_device_score"`
	CommonIPScore       float64 `json:"common_ip_score"`
	TotalScore          float64 `json:"total_score"`
}

type SilentVerifyRequest struct {
	DeviceFingerprint string                 `json:"device_fingerprint"`
	SessionID        string                 `json:"session_id"`
	BehaviorData     []BehaviorDataPoint    `json:"behavior_data"`
	Timestamp        int64                  `json:"timestamp"`
	UserID           uint                  `json:"user_id"`
	IPAddress        string                `json:"ip_address"`
	UserAgent        string                `json:"user_agent"`
}

type SilentVerifyResponse struct {
	Pass        bool                 `json:"pass"`
	RiskLevel   string               `json:"risk_level"`
	NeedCaptcha bool                 `json:"need_captcha"`
	CaptchaType string               `json:"captcha_type"`
	Token       string               `json:"token"`
	Strategy    *VerificationStrategy `json:"strategy,omitempty"`
	WaitTime    int                  `json:"wait_time"`
	Message     string               `json:"message"`
}

type StrategyRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Priority    int      `json:"priority"`
	Conditions  []RuleCondition `json:"conditions"`
	Action      StrategyAction  `json:"action"`
	Enabled     bool     `json:"enabled"`
}

type RuleCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type StrategyAction struct {
	Level       string `json:"level"`
	NeedCaptcha bool   `json:"need_captcha"`
	CaptchaType string `json:"captcha_type"`
	WaitTime    int    `json:"wait_time"`
	Score       float64 `json:"score"`
}

type VerificationCache struct {
	Token       string               `json:"token"`
	Status      string               `json:"status"`
	Request     *SilentVerifyRequest `json:"request"`
	Strategy    *VerificationStrategy `json:"strategy"`
	Result      *SilentVerifyResponse `json:"result"`
	CreatedAt   time.Time            `json:"created_at"`
	ExpiresAt   time.Time            `json:"expires_at"`
}

type RateLimitInfo struct {
	IPCount      int64     `json:"ip_count"`
	UserCount    int64     `json:"user_count"`
	GlobalCount  int64     `json:"global_count"`
	Whitelisted  bool      `json:"whitelisted"`
	LastAccessed time.Time `json:"last_accessed"`
}

var (
	defaultConfig = &SilentVerificationConfig{
		Enabled:              true,
		RiskThreshold:        30.0,
		MinBehaviorDataPoints: 20,
		MaxVerifyDuration:    300,
		EnableDeviceCheck:    true,
		EnableBehaviorCheck:  true,
		EnableHistoryCheck:   true,
		CacheTTL:             600,
	}

	strategyRules = []StrategyRule{
		{
			ID:       "rule_very_low_risk",
			Name:     "极低风险",
			Priority: 1,
			Conditions: []RuleCondition{
				{Field: "device_score", Operator: ">=", Value: 80.0},
				{Field: "behavior_score", Operator: ">=", Value: 75.0},
				{Field: "history_score", Operator: ">=", Value: 80.0},
			},
			Action: StrategyAction{Level: "pass", NeedCaptcha: false, CaptchaType: "none", WaitTime: 0, Score: 0},
			Enabled: true,
		},
		{
			ID:       "rule_low_risk",
			Name:     "低风险",
			Priority: 2,
			Conditions: []RuleCondition{
				{Field: "device_score", Operator: ">=", Value: 60.0},
				{Field: "behavior_score", Operator: ">=", Value: 50.0},
				{Field: "history_score", Operator: ">=", Value: 60.0},
			},
			Action: StrategyAction{Level: "pass", NeedCaptcha: false, CaptchaType: "none", WaitTime: 0, Score: 15},
			Enabled: true,
		},
		{
			ID:       "rule_medium_risk",
			Name:     "中等风险",
			Priority: 3,
			Conditions: []RuleCondition{
				{Field: "device_score", Operator: ">=", Value: 40.0},
				{Field: "behavior_score", Operator: ">=", Value: 30.0},
			},
			Action: StrategyAction{Level: "challenge", NeedCaptcha: true, CaptchaType: "slider", WaitTime: 5, Score: 40},
			Enabled: true,
		},
		{
			ID:       "rule_high_risk",
			Name:     "高风险",
			Priority: 4,
			Conditions: []RuleCondition{
				{Field: "device_score", Operator: ">=", Value: 20.0},
				{Field: "behavior_score", Operator: ">=", Value: 20.0},
			},
			Action: StrategyAction{Level: "challenge", NeedCaptcha: true, CaptchaType: "click", WaitTime: 10, Score: 70},
			Enabled: true,
		},
		{
			ID:       "rule_very_high_risk",
			Name:     "极高风险",
			Priority: 5,
			Conditions: []RuleCondition{
				{Field: "device_score", Operator: "<", Value: 20.0},
				{Field: "behavior_score", Operator: "<", Value: 20.0},
			},
			Action: StrategyAction{Level: "block", NeedCaptcha: true, CaptchaType: "click", WaitTime: 30, Score: 90},
			Enabled: true,
		},
	}

	strategyMutex    sync.RWMutex
	configMutex      sync.RWMutex
	rateLimitMutex   sync.RWMutex
	currentConfig    = defaultConfig
	verificationCache = make(map[string]*VerificationCache)
	rateLimitCache    = make(map[string]*RateLimitInfo)
)

type SilentVerificationService struct {
	behaviorService *BehaviorAnalysisService
}

func NewSilentVerificationService() *SilentVerificationService {
	return &SilentVerificationService{
		behaviorService: NewBehaviorAnalysisService(),
	}
}

func (s *SilentVerificationService) ProcessVerification(req *SilentVerifyRequest) (*SilentVerifyResponse, error) {
	startTime := time.Now()

	token := s.generateToken(req)

	deviceScore := s.evaluateDeviceTrust(req)
	behaviorScore := s.evaluateBehaviorTrust(req.BehaviorData)
	historyScore := s.evaluateHistoryTrust(req)

	totalRiskScore := s.calculateCompositeScore(deviceScore, behaviorScore, historyScore)

	strategy := s.determineStrategy(totalRiskScore, deviceScore, behaviorScore, historyScore)

	response := &SilentVerifyResponse{
		Pass:        strategy.Level == "pass",
		RiskLevel:   strategy.Level,
		NeedCaptcha: strategy.NeedCaptcha,
		CaptchaType: strategy.CaptchaType,
		Token:       token,
		Strategy:    strategy,
		WaitTime:    strategy.WaitTime,
		Message:     s.getMessageForStrategy(strategy),
	}

	s.cacheVerification(token, req, strategy, response)

	if strategy.Level == "challenge" {
		go s.triggerAsyncVerification(token, req)
	}

	duration := time.Since(startTime).Milliseconds()
	go s.logVerification(req, strategy, duration)

	return response, nil
}

func (s *SilentVerificationService) evaluateDeviceTrust(req *SilentVerifyRequest) *DeviceTrustScore {
	score := &DeviceTrustScore{
		DeviceFingerprint: req.DeviceFingerprint,
	}

	score.FingerprintMatch = s.calculateFingerprintMatch(req.DeviceFingerprint)

	score.HistoricalScore = s.getHistoricalDeviceScore(req.DeviceFingerprint, req.UserID)

	score.LoginFrequency = s.calculateLoginFrequency(req.DeviceFingerprint, req.UserID)

	score.GeographicScore = s.evaluateGeographicRisk(req.IPAddress, req.UserID)

	score.TotalScore = score.FingerprintMatch*0.3 + 
		score.HistoricalScore*0.3 + 
		score.LoginFrequency*0.2 + 
		score.GeographicScore*0.2

	return score
}

func (s *SilentVerificationService) calculateFingerprintMatch(fingerprint string) float64 {
	if fingerprint == "" {
		return 0
	}
	
	fingerprintHash := sha256.Sum256([]byte(fingerprint))
	hashStr := hex.EncodeToString(fingerprintHash[:])
	
	var matchScore float64 = 50
	if len(hashStr) >= 32 {
		matchScore = 70
	}
	
	if len(fingerprint) > 20 {
		matchScore += 10
	}
	
	return math.Min(matchScore, 100)
}

func (s *SilentVerificationService) getHistoricalDeviceScore(fingerprint string, userID uint) float64 {
	key := fmt.Sprintf("device:history:%s:%d", fingerprint, userID)
	
	if redis.Client != nil {
		ctx := context.Background()
		exists, err := redis.Client.Exists(ctx, key).Result()
		if err == nil && exists > 0 {
			return 80
		}
	}
	
	return 40
}

func (s *SilentVerificationService) calculateLoginFrequency(fingerprint string, userID uint) float64 {
	key := fmt.Sprintf("device:frequency:%s:%d", fingerprint, userID)
	
	if redis.Client != nil {
		ctx := context.Background()
		count, err := redis.Client.Get(ctx, key).Int64()
		if err == nil {
			if count <= 3 {
				return 90
			} else if count <= 10 {
				return 70
			} else if count <= 30 {
				return 50
			}
		}
	}
	
	return 60
}

func (s *SilentVerificationService) evaluateGeographicRisk(ipAddress string, userID uint) float64 {
	if ipAddress == "" {
		return 50
	}
	
	knownIPs := s.getKnownIPsForUser(userID)
	
	for _, knownIP := range knownIPs {
		if s.ipsInSameRegion(ipAddress, knownIP) {
			return 90
		}
	}
	
	return 60
}

func (s *SilentVerificationService) getKnownIPsForUser(userID uint) []string {
	key := fmt.Sprintf("user:ips:%d", userID)
	
	if redis.Client != nil {
		ctx := context.Background()
		ips, err := redis.Client.SMembers(ctx, key).Result()
		if err == nil {
			return ips
		}
	}
	
	return []string{}
}

func (s *SilentVerificationService) ipsInSameRegion(ip1, ip2 string) bool {
	if ip1 == ip2 {
		return true
	}
	
	parts1 := s.parseIP(ip1)
	parts2 := s.parseIP(ip2)
	
	if len(parts1) < 3 || len(parts2) < 3 {
		return false
	}
	
	return parts1[0] == parts2[0] && parts1[1] == parts2[1]
}

func (s *SilentVerificationService) parseIP(ip string) []int {
	var parts []int
	current := 0
	for i := 0; i < len(ip); i++ {
		if ip[i] == '.' {
			parts = append(parts, current)
			current = 0
		} else {
			current = current*10 + int(ip[i]-'0')
		}
	}
	parts = append(parts, current)
	return parts
}

func (s *SilentVerificationService) evaluateBehaviorTrust(behaviorData []BehaviorDataPoint) *BehaviorTrustScore {
	score := &BehaviorTrustScore{}

	if len(behaviorData) == 0 {
		score.TotalScore = 30
		return score
	}

	score.MouseTrajectoryScore = s.analyzeMouseTrajectoryTrust(behaviorData)
	score.ClickPatternScore = s.analyzeClickPatternTrust(behaviorData)
	score.KeyboardPatternScore = s.analyzeKeyboardPatternTrust(behaviorData)
	score.ScrollBehaviorScore = s.analyzeScrollBehaviorTrust(behaviorData)
	score.TouchBehaviorScore = s.analyzeTouchBehaviorTrust(behaviorData)

	score.TotalScore = score.MouseTrajectoryScore*0.35 +
		score.ClickPatternScore*0.25 +
		score.KeyboardPatternScore*0.15 +
		score.ScrollBehaviorScore*0.15 +
		score.TouchBehaviorScore*0.10

	return score
}

func (s *SilentVerificationService) analyzeMouseTrajectoryTrust(data []BehaviorDataPoint) float64 {
	var moveEvents []BehaviorDataPoint
	for _, d := range data {
		if d.Event == "mousemove" || d.Event == "move" {
			moveEvents = append(moveEvents, d)
		}
	}

	if len(moveEvents) < 10 {
		return 20
	}

	totalSpeed := 0.0
	speedCount := 0
	directionChanges := 0
	var prevAngle float64 = -1

	for i := 1; i < len(moveEvents); i++ {
		dx := float64(moveEvents[i].X - moveEvents[i-1].X)
		dy := float64(moveEvents[i].Y - moveEvents[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		
		dt := float64(moveEvents[i].Timestamp - moveEvents[i-1].Timestamp)
		if dt > 0 {
			speed := distance / dt * 1000
			totalSpeed += speed
			speedCount++
		}

		if i > 1 {
			angle := math.Atan2(dy, dx)
			if prevAngle >= 0 && math.Abs(angle-prevAngle) > 0.5 {
				directionChanges++
			}
			prevAngle = angle
		}
	}

	avgSpeed := 0.0
	if speedCount > 0 {
		avgSpeed = totalSpeed / float64(speedCount)
	}

	if avgSpeed > 10 || avgSpeed < 0.05 {
		return 30
	}

	if directionChanges < len(moveEvents)/10 {
		return 40
	}

	return 75
}

func (s *SilentVerificationService) analyzeClickPatternTrust(data []BehaviorDataPoint) float64 {
	var clickEvents []BehaviorDataPoint
	for _, d := range data {
		if d.Event == "click" || d.Event == "mousedown" {
			clickEvents = append(clickEvents, d)
		}
	}

	if len(clickEvents) == 0 {
		return 50
	}

	if len(clickEvents) < 2 {
		return 40
	}

	var intervals []float64
	for i := 1; i < len(clickEvents); i++ {
		interval := float64(clickEvents[i].Timestamp - clickEvents[i-1].Timestamp)
		intervals = append(intervals, interval)
	}

	avgInterval := 0.0
	for _, interval := range intervals {
		avgInterval += interval
	}
	avgInterval /= float64(len(intervals))

	variance := 0.0
	for _, interval := range intervals {
		variance += math.Pow(interval-avgInterval, 2)
	}
	variance /= float64(len(intervals))
	stdDev := math.Sqrt(variance)

	regularity := 1.0
	if avgInterval > 0 {
		regularity = 1 - (stdDev / avgInterval)
	}

	if regularity > 0.95 && len(clickEvents) > 3 {
		return 20
	}

	if regularity < 0.3 {
		return 85
	}

	return 60
}

func (s *SilentVerificationService) analyzeKeyboardPatternTrust(data []BehaviorDataPoint) float64 {
	var keyEvents []BehaviorDataPoint
	for _, d := range data {
		if d.Event == "keydown" || d.Event == "keyup" || d.Event == "keypress" {
			keyEvents = append(keyEvents, d)
		}
	}

	if len(keyEvents) == 0 {
		return 50
	}

	if len(keyEvents) < 3 {
		return 40
	}

	var intervals []float64
	for i := 1; i < len(keyEvents); i++ {
		interval := float64(keyEvents[i].Timestamp - keyEvents[i-1].Timestamp)
		intervals = append(intervals, interval)
	}

	avgInterval := 0.0
	for _, interval := range intervals {
		avgInterval += interval
	}
	avgInterval /= float64(len(intervals))

	if avgInterval < 50 && len(keyEvents) > 5 {
		return 25
	}

	return 70
}

func (s *SilentVerificationService) analyzeScrollBehaviorTrust(data []BehaviorDataPoint) float64 {
	var scrollEvents []BehaviorDataPoint
	for _, d := range data {
		if d.Event == "scroll" || d.Event == "wheel" {
			scrollEvents = append(scrollEvents, d)
		}
	}

	if len(scrollEvents) == 0 {
		return 50
	}

	if len(scrollEvents) > 0 && len(scrollEvents) < 3 {
		return 60
	}

	var intervals []float64
	for i := 1; i < len(scrollEvents); i++ {
		interval := float64(scrollEvents[i].Timestamp - scrollEvents[i-1].Timestamp)
		intervals = append(intervals, interval)
	}

	variance := 0.0
	avgInterval := 0.0
	for _, interval := range intervals {
		avgInterval += interval
	}
	avgInterval /= float64(len(intervals))

	for _, interval := range intervals {
		variance += math.Pow(interval-avgInterval, 2)
	}
	variance /= float64(len(intervals))
	stdDev := math.Sqrt(variance)

	if stdDev < avgInterval*0.2 {
		return 35
	}

	return 70
}

func (s *SilentVerificationService) analyzeTouchBehaviorTrust(data []BehaviorDataPoint) float64 {
	var touchEvents []BehaviorDataPoint
	for _, d := range data {
		if d.Event == "touchstart" || d.Event == "touchmove" || d.Event == "touchend" {
			touchEvents = append(touchEvents, d)
		}
	}

	if len(touchEvents) == 0 {
		return 50
	}

	return 65
}

func (s *SilentVerificationService) evaluateHistoryTrust(req *SilentVerifyRequest) *HistoryTrustScore {
	score := &HistoryTrustScore{}

	score.AccountHistoryScore = s.getAccountHistoryScore(req.UserID)
	score.LoginFrequencyScore = s.getLoginFrequencyScore(req.UserID)
	score.OperationHabitScore = s.getOperationHabitScore(req.UserID)
	score.CommonDeviceScore = s.getCommonDeviceScore(req.DeviceFingerprint, req.UserID)
	score.CommonIPScore = s.getCommonIPScore(req.IPAddress, req.UserID)

	score.TotalScore = score.AccountHistoryScore*0.2 +
		score.LoginFrequencyScore*0.2 +
		score.OperationHabitScore*0.2 +
		score.CommonDeviceScore*0.2 +
		score.CommonIPScore*0.2

	return score
}

func (s *SilentVerificationService) getAccountHistoryScore(userID uint) float64 {
	if userID == 0 {
		return 50
	}
	
	key := fmt.Sprintf("user:history:age:%d", userID)
	
	if redis.Client != nil {
		ctx := context.Background()
		days, err := redis.Client.Get(ctx, key).Int64()
		if err == nil {
			if days > 180 {
				return 90
			} else if days > 90 {
				return 75
			} else if days > 30 {
				return 60
			}
		}
	}
	
	return 55
}

func (s *SilentVerificationService) getLoginFrequencyScore(userID uint) float64 {
	key := fmt.Sprintf("user:logins:%d:24h", userID)
	
	if redis.Client != nil {
		ctx := context.Background()
		count, err := redis.Client.Get(ctx, key).Int64()
		if err == nil {
			if count <= 5 {
				return 85
			} else if count <= 15 {
				return 65
			} else if count <= 30 {
				return 45
			}
		}
	}
	
	return 50
}

func (s *SilentVerificationService) getOperationHabitScore(userID uint) float64 {
	key := fmt.Sprintf("user:habits:%d", userID)
	
	if redis.Client != nil {
		ctx := context.Background()
		exists, err := redis.Client.Exists(ctx, key).Result()
		if err == nil && exists > 0 {
			return 80
		}
	}
	
	return 50
}

func (s *SilentVerificationService) getCommonDeviceScore(fingerprint string, userID uint) float64 {
	if fingerprint == "" || userID == 0 {
		return 50
	}
	
	key := fmt.Sprintf("user:devices:%d", userID)
	
	if redis.Client != nil {
		ctx := context.Background()
		isMember, err := redis.Client.SIsMember(ctx, key, fingerprint).Result()
		if err == nil && isMember {
			return 90
		}
	}
	
	return 40
}

func (s *SilentVerificationService) getCommonIPScore(ipAddress string, userID uint) float64 {
	if ipAddress == "" || userID == 0 {
		return 50
	}
	
	key := fmt.Sprintf("user:ips:%d", userID)
	
	if redis.Client != nil {
		ctx := context.Background()
		isMember, err := redis.Client.SIsMember(ctx, key, ipAddress).Result()
		if err == nil && isMember {
			return 85
		}
	}
	
	return 45
}

func (s *SilentVerificationService) calculateCompositeScore(device *DeviceTrustScore, behavior *BehaviorTrustScore, history *HistoryTrustScore) float64 {
	deviceWeight := 0.30
	behaviorWeight := 0.40
	historyWeight := 0.30

	if !currentConfig.EnableDeviceCheck {
		deviceWeight = 0
		behaviorWeight = 0.55
		historyWeight = 0.45
	}
	if !currentConfig.EnableBehaviorCheck {
		behaviorWeight = 0
		if currentConfig.EnableDeviceCheck {
			deviceWeight = 0.55
			historyWeight = 0.45
		} else {
			historyWeight = 1.0
		}
	}
	if !currentConfig.EnableHistoryCheck {
		historyWeight = 0
		if currentConfig.EnableDeviceCheck && currentConfig.EnableBehaviorCheck {
			deviceWeight = 0.45
			behaviorWeight = 0.55
		} else if currentConfig.EnableDeviceCheck {
			deviceWeight = 1.0
		} else {
			behaviorWeight = 1.0
		}
	}

	composite := device.TotalScore*deviceWeight +
		behavior.TotalScore*behaviorWeight +
		history.TotalScore*historyWeight

	return math.Max(0, math.Min(100, 100-composite))
}

func (s *SilentVerificationService) determineStrategy(totalRiskScore float64, device *DeviceTrustScore, behavior *BehaviorTrustScore, history *HistoryTrustScore) *VerificationStrategy {
	strategyMutex.RLock()
	defer strategyMutex.RUnlock()

	activeRules := make([]StrategyRule, 0)
	for _, rule := range strategyRules {
		if rule.Enabled {
			activeRules = append(activeRules, rule)
		}
	}

	sort.Slice(activeRules, func(i, j int) bool {
		return activeRules[i].Priority < activeRules[j].Priority
	})

	for _, rule := range activeRules {
		if s.matchConditions(rule.Conditions, device, behavior, history) {
			return &VerificationStrategy{
				Level:       rule.Action.Level,
				RiskScore:   rule.Action.Score,
				NeedCaptcha: rule.Action.NeedCaptcha,
				CaptchaType: rule.Action.CaptchaType,
				WaitTime:    rule.Action.WaitTime,
				Reasons:     s.generateRiskReasons(device, behavior, history),
				Suggestions: s.generateSuggestions(rule.Action),
			}
		}
	}

	return &VerificationStrategy{
		Level:       "challenge",
		RiskScore:   totalRiskScore,
		NeedCaptcha: true,
		CaptchaType: "slider",
		WaitTime:    5,
		Reasons:     s.generateRiskReasons(device, behavior, history),
		Suggestions: []string{"完成验证后继续"},
	}
}

func (s *SilentVerificationService) matchConditions(conditions []RuleCondition, device *DeviceTrustScore, behavior *BehaviorTrustScore, history *HistoryTrustScore) bool {
	for _, cond := range conditions {
		var fieldValue float64
		switch cond.Field {
		case "device_score":
			fieldValue = device.TotalScore
		case "behavior_score":
			fieldValue = behavior.TotalScore
		case "history_score":
			fieldValue = history.TotalScore
		default:
			continue
		}

		condValue, ok := cond.Value.(float64)
		if !ok {
			continue
		}

		switch cond.Operator {
		case ">=":
			if fieldValue < condValue {
				return false
			}
		case "<=":
			if fieldValue > condValue {
				return false
			}
		case ">":
			if fieldValue <= condValue {
				return false
			}
		case "<":
			if fieldValue >= condValue {
				return false
			}
		case "==":
			if fieldValue != condValue {
				return false
			}
		case "!=":
			if fieldValue == condValue {
				return false
			}
		}
	}
	return true
}

func (s *SilentVerificationService) generateRiskReasons(device *DeviceTrustScore, behavior *BehaviorTrustScore, history *HistoryTrustScore) []string {
	var reasons []string

	if device.TotalScore < 40 {
		reasons = append(reasons, "设备指纹可信度低")
	}
	if behavior.TotalScore < 30 {
		reasons = append(reasons, "行为模式异常")
	}
	if history.TotalScore < 40 {
		reasons = append(reasons, "历史记录可信度低")
	}
	if device.FingerprintMatch < 50 {
		reasons = append(reasons, "设备指纹匹配度低")
	}
	if behavior.MouseTrajectoryScore < 40 {
		reasons = append(reasons, "鼠标轨迹异常")
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "正常验证")
	}

	return reasons
}

func (s *SilentVerificationService) generateSuggestions(action StrategyAction) []string {
	var suggestions []string

	switch action.Level {
	case "pass":
		suggestions = append(suggestions, "验证通过，可继续操作")
	case "challenge":
		suggestions = append(suggestions, "请完成安全验证")
		if action.CaptchaType == "slider" {
			suggestions = append(suggestions, "拖动滑块完成验证")
		} else if action.CaptchaType == "click" {
			suggestions = append(suggestions, "按顺序点击图片完成验证")
		}
	case "block":
		suggestions = append(suggestions, "验证失败，请稍后重试")
		suggestions = append(suggestions, "如有问题请联系客服")
	}

	return suggestions
}

func (s *SilentVerificationService) getMessageForStrategy(strategy *VerificationStrategy) string {
	switch strategy.Level {
	case "pass":
		return "验证通过"
	case "challenge":
		return "请完成验证"
	case "block":
		return "验证被拦截"
	default:
		return "验证处理中"
	}
}

func (s *SilentVerificationService) generateToken(req *SilentVerifyRequest) string {
	data := fmt.Sprintf("%s:%s:%d:%d",
		req.DeviceFingerprint,
		req.SessionID,
		req.UserID,
		time.Now().UnixNano(),
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (s *SilentVerificationService) cacheVerification(token string, req *SilentVerifyRequest, strategy *VerificationStrategy, response *SilentVerifyResponse) {
	cache := &VerificationCache{
		Token:     token,
		Status:    "pending",
		Request:   req,
		Strategy:  strategy,
		Result:    response,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Duration(currentConfig.CacheTTL) * time.Second),
	}

	cacheKey := fmt.Sprintf("silent_verify:%s", token)
	
	if redis.Client != nil {
		ctx := context.Background()
		data, _ := json.Marshal(cache)
		redis.Client.Set(ctx, cacheKey, data, time.Duration(currentConfig.CacheTTL)*time.Second)
	} else {
		strategyMutex.Lock()
		verificationCache[token] = cache
		strategyMutex.Unlock()
	}
}

func (s *SilentVerificationService) triggerAsyncVerification(token string, req *SilentVerifyRequest) {
	time.Sleep(2 * time.Second)

	s.updateVerificationStatus(token, "completed")
}

func (s *SilentVerificationService) updateVerificationStatus(token string, status string) {
	cacheKey := fmt.Sprintf("silent_verify:%s", token)

	if redis.Client != nil {
		ctx := context.Background()
		data, err := redis.Client.Get(ctx, cacheKey).Bytes()
		if err == nil {
			var cache VerificationCache
			if json.Unmarshal(data, &cache) == nil {
				cache.Status = status
				newData, _ := json.Marshal(cache)
				ttl := time.Until(cache.ExpiresAt)
				if ttl > 0 {
					redis.Client.Set(ctx, cacheKey, newData, ttl)
				}
			}
		}
	} else {
		strategyMutex.Lock()
		defer strategyMutex.Unlock()
		if cache, ok := verificationCache[token]; ok {
			cache.Status = status
		}
	}
}

func (s *SilentVerificationService) GetVerificationStatus(token string) (*VerificationCache, error) {
	cacheKey := fmt.Sprintf("silent_verify:%s", token)

	if redis.Client != nil {
		ctx := context.Background()
		data, err := redis.Client.Get(ctx, cacheKey).Bytes()
		if err != nil {
			return nil, fmt.Errorf("verification not found")
		}
		var cache VerificationCache
		if err := json.Unmarshal(data, &cache); err != nil {
			return nil, err
		}
		return &cache, nil
	}

	strategyMutex.RLock()
	defer strategyMutex.RUnlock()
	if cache, ok := verificationCache[token]; ok {
		return cache, nil
	}
	return nil, fmt.Errorf("verification not found")
}

func (s *SilentVerificationService) logVerification(req *SilentVerifyRequest, strategy *VerificationStrategy, duration int64) {
	logData := map[string]interface{}{
		"device_fingerprint": req.DeviceFingerprint,
		"session_id":        req.SessionID,
		"user_id":           req.UserID,
		"ip_address":        req.IPAddress,
		"risk_level":        strategy.Level,
		"risk_score":        strategy.RiskScore,
		"need_captcha":      strategy.NeedCaptcha,
		"captcha_type":      strategy.CaptchaType,
		"duration_ms":       duration,
		"timestamp":         time.Now().Unix(),
	}

	logJSON, _ := json.Marshal(logData)
	logKey := fmt.Sprintf("verification:log:%d", time.Now().Unix())

	if redis.Client != nil {
		ctx := context.Background()
		redis.Client.LPush(ctx, "verification:logs", logJSON)
		redis.Client.Expire(ctx, "verification:logs", 7*24*time.Hour)
		redis.Client.LPush(ctx, logKey, logJSON)
		redis.Client.Expire(ctx, logKey, 7*24*time.Hour)
	}
}

func (s *SilentVerificationService) CheckRateLimit(ipAddress string, userID uint) (bool, error) {
	rateLimitMutex.Lock()
	defer rateLimitMutex.Unlock()

	whitelistKey := fmt.Sprintf("rate_limit:whitelist:%s", ipAddress)
	if redis.Client != nil {
		ctx := context.Background()
		isWhitelisted, err := redis.Client.Get(ctx, whitelistKey).Bool()
		if err == nil && isWhitelisted {
			return true, nil
		}
	}

	ipLimit := 100
	userLimit := 50
	globalLimit := 10000

	ipKey := fmt.Sprintf("rate_limit:ip:%s", ipAddress)
	userKey := fmt.Sprintf("rate_limit:user:%d", userID)
	globalKey := "rate_limit:global"

	if redis.Client != nil {
		ctx := context.Background()

		ipCount, _ := redis.Client.Incr(ctx, ipKey).Result()
		if ipCount == 1 {
			redis.Client.Expire(ctx, ipKey, time.Minute)
		}

		if ipCount > int64(ipLimit) {
			return false, fmt.Errorf("IP rate limit exceeded")
		}

		if userID > 0 {
			userCount, _ := redis.Client.Incr(ctx, userKey).Result()
			if userCount == 1 {
				redis.Client.Expire(ctx, userKey, time.Minute)
			}
			if userCount > int64(userLimit) {
				return false, fmt.Errorf("user rate limit exceeded")
			}
		}

		globalCount, _ := redis.Client.Incr(ctx, globalKey).Result()
		if globalCount == 1 {
			redis.Client.Expire(ctx, globalKey, time.Minute)
		}
		if globalCount > int64(globalLimit) {
			return false, fmt.Errorf("global rate limit exceeded")
		}
	}

	return true, nil
}

func (s *SilentVerificationService) GetConfig() *SilentVerificationConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return currentConfig
}

func (s *SilentVerificationService) UpdateConfig(config *SilentVerificationConfig) {
	configMutex.Lock()
	defer configMutex.Unlock()
	currentConfig = config

	if redis.Client != nil {
		ctx := context.Background()
		data, _ := json.Marshal(config)
		redis.Client.Set(ctx, "silent_verification:config", data, 0)
	}
}

func (s *SilentVerificationService) GetStrategyRules() []StrategyRule {
	strategyMutex.RLock()
	defer strategyMutex.RUnlock()
	return strategyRules
}

func (s *SilentVerificationService) UpdateStrategyRule(ruleID string, rule StrategyRule) error {
	strategyMutex.Lock()
	defer strategyMutex.Unlock()

	for i, r := range strategyRules {
		if r.ID == ruleID {
			strategyRules[i] = rule
			return nil
		}
	}

	return fmt.Errorf("rule not found: %s", ruleID)
}

func (s *SilentVerificationService) GetVerificationStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	if redis.Client != nil {
		ctx := context.Background()

		totalLogs, _ := redis.Client.LLen(ctx, "verification:logs").Result()
		stats["total_verifications"] = totalLogs

		recentLogs, _ := redis.Client.LRange(ctx, "verification:logs", 0, 100).Result()

		passCount := 0
		challengeCount := 0
		blockCount := 0

		for _, logStr := range recentLogs {
			var logData map[string]interface{}
			if json.Unmarshal([]byte(logStr), &logData) == nil {
				level, _ := logData["risk_level"].(string)
				switch level {
				case "pass":
					passCount++
				case "challenge":
					challengeCount++
				case "block":
					blockCount++
				}
			}
		}

		stats["pass_count"] = passCount
		stats["challenge_count"] = challengeCount
		stats["block_count"] = blockCount

		if passCount+challengeCount+blockCount > 0 {
			passRate := float64(passCount) / float64(passCount+challengeCount+blockCount) * 100
			stats["pass_rate"] = passRate
		}
	}

	return stats, nil
}

func (s *SilentVerificationService) DegradeToNormalVerification() *VerificationStrategy {
	return &VerificationStrategy{
		Level:       "challenge",
		RiskScore:   50,
		NeedCaptcha: true,
		CaptchaType: "slider",
		WaitTime:    3,
		Reasons:     []string{"降级为普通验证"},
		Suggestions: []string{"请完成滑块验证"},
	}
}
