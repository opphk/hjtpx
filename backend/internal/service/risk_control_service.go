package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type RiskControlService struct {
	deviceRiskCache  map[string]*DeviceRiskProfile
	ipRiskCache      map[string]*IPRiskProfile
	behaviorProfiles map[string]*RiskBehaviorProfile
	rules            []RiskRuleConfig
	weights          RiskWeights
	mu               sync.RWMutex
	lastRuleUpdate   time.Time
}

type DeviceRiskProfile struct {
	DeviceID       string
	RequestCount   int
	LastRequest    time.Time
	SuccessCount   int
	FailureCount   int
	BlockCount     int
	AvgResponseMs  float64
	RiskScore      float64
	Tags           []string
	FirstSeen      time.Time
	IsSuspicious   bool
}

type IPRiskProfile struct {
	IPAddress      string
	RequestCount   int
	SuccessCount   int
	FailureCount   int
	BlockCount     int
	LastRequest    time.Time
	FirstSeen      time.Time
	IsProxy        bool
	IsVPN          bool
	IsTor          bool
	IsHosting      bool
	Country        string
	ASNumber       int
	RiskScore      float64
	GeoVelocity    float64
	UniqueDevices  int
}

type RiskBehaviorProfile struct {
	UserID         string
	SessionCount   int
	RequestCount   int
	SuccessCount   int
	FailureCount   int
	LastRequest    time.Time
	AvgResponseMs  float64
	MouseSpeedAvg  float64
	ClickInterval  float64
	TrajectoryLen  int
	IsBotLike      bool
	RiskScore      float64
	TrustLevel     int
}

type RiskRuleConfig struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Dimension    string                 `json:"dimension"`
	Condition    map[string]interface{} `json:"condition"`
	ScoreImpact  float64                `json:"score_impact"`
	Action       string                 `json:"action"`
	Enabled      bool                   `json:"enabled"`
	Priority     int                    `json:"priority"`
	Threshold    float64                `json:"threshold"`
	Description  string                 `json:"description"`
}

type RiskWeights struct {
	DeviceWeight float64 `json:"device_weight"`
	IPWeight     float64 `json:"ip_weight"`
	BehaviorWeight float64 `json:"behavior_weight"`
	EnvWeight    float64 `json:"env_weight"`
}

type RiskEvaluationResult struct {
	RiskScore       float64            `json:"risk_score"`
	RiskLevel       model.RiskLevel    `json:"risk_level"`
	PositionScore   float64            `json:"position_score"`
	BehaviorScore   float64            `json:"behavior_score"`
	EnvScore        float64            `json:"env_score"`
	DeviceScore     float64            `json:"device_score"`
	IPScore         float64            `json:"ip_score"`
	Action          string             `json:"action"`
	RiskFactors     []string           `json:"risk_factors"`
	Details         map[string]float64 `json:"details"`
	RecommendVerify bool               `json:"recommend_verify"`
	ProcessingTime  int64              `json:"processing_time_ms"`
}

const (
	ActionAllow    = "allow"
	ActionChallenge = "challenge"
	ActionBlock    = "block"
	ActionFlag     = "flag"
)

var (
	riskControlService *RiskControlService
	once               sync.Once
)

func GetRiskControlService() *RiskControlService {
	once.Do(func() {
		riskControlService = NewRiskControlService()
	})
	return riskControlService
}

func NewRiskControlService() *RiskControlService {
	service := &RiskControlService{
		deviceRiskCache:  make(map[string]*DeviceRiskProfile),
		ipRiskCache:      make(map[string]*IPRiskProfile),
		behaviorProfiles: make(map[string]*RiskBehaviorProfile),
		rules:            getDefaultRules(),
		weights: RiskWeights{
			DeviceWeight:   0.25,
			IPWeight:       0.30,
			BehaviorWeight: 0.30,
			EnvWeight:      0.15,
		},
		lastRuleUpdate: time.Now(),
	}
	return service
}

func getDefaultRules() []RiskRuleConfig {
	return []RiskRuleConfig{
		{
			ID:          "ip_proxy",
			Name:        "代理/ VPN 检测",
			Dimension:   "ip",
			Condition:   map[string]interface{}{"is_proxy": true, "is_vpn": true},
			ScoreImpact: 30,
			Action:      ActionChallenge,
			Enabled:     true,
			Priority:    1,
			Description: "检测到使用代理或VPN",
		},
		{
			ID:          "ip_tor",
			Name:        "Tor 网络",
			Dimension:   "ip",
			Condition:   map[string]interface{}{"is_tor": true},
			ScoreImpact: 35,
			Action:      ActionChallenge,
			Enabled:     true,
			Priority:    1,
			Description: "检测到使用Tor网络",
		},
		{
			ID:          "ip_hosting",
			Name:        "托管服务",
			Dimension:   "ip",
			Condition:   map[string]interface{}{"is_hosting": true},
			ScoreImpact: 25,
			Action:      ActionFlag,
			Enabled:     true,
			Priority:    2,
			Description: "检测到来自托管服务IP",
		},
		{
			ID:          "ip_geo_velocity",
			Name:        "地理速度异常",
			Dimension:   "ip",
			Condition:   map[string]interface{}{"geo_velocity_kmh": 500},
			ScoreImpact: 40,
			Action:      ActionChallenge,
			Enabled:     true,
			Priority:    1,
			Description: "短时间内跨越大地理距离",
		},
		{
			ID:          "device_new",
			Name:        "新设备",
			Dimension:   "device",
			Condition:   map[string]interface{}{"request_count_max": 3},
			ScoreImpact: 15,
			Action:      ActionFlag,
			Enabled:     true,
			Priority:    3,
			Description: "设备首次出现或请求次数少",
		},
		{
			ID:          "device_suspicious",
			Name:        "可疑设备",
			Dimension:   "device",
			Condition:   map[string]interface{}{"failure_ratio": 0.5},
			ScoreImpact: 25,
			Action:      ActionChallenge,
			Enabled:     true,
			Priority:    2,
			Description: "设备失败率较高",
		},
		{
			ID:          "device_blocked",
			Name:        "设备被封禁",
			Dimension:   "device",
			Condition:   map[string]interface{}{"block_count_min": 1},
			ScoreImpact: 50,
			Action:      ActionBlock,
			Enabled:     true,
			Priority:    1,
			Description: "设备已被封禁记录",
		},
		{
			ID:          "behavior_fast",
			Name:        "行为速度异常",
			Dimension:   "behavior",
			Condition:   map[string]interface{}{"mouse_speed_max": 500},
			ScoreImpact: 30,
			Action:      ActionChallenge,
			Enabled:     true,
			Priority:    1,
			Description: "鼠标移动速度异常快",
		},
		{
			ID:          "behavior_no_trace",
			Name:        "无轨迹数据",
			Dimension:   "behavior",
			Condition:   map[string]interface{}{"trace_length_min": 5},
			ScoreImpact: 20,
			Action:      ActionFlag,
			Enabled:     true,
			Priority:    2,
			Description: "缺少用户操作轨迹",
		},
		{
			ID:          "behavior_bot_pattern",
			Name:        "机器人行为模式",
			Dimension:   "behavior",
			Condition:   map[string]interface{}{"is_bot_like": true},
			ScoreImpact: 45,
			Action:      ActionChallenge,
			Enabled:     true,
			Priority:    1,
			Description: "检测到机器人行为特征",
		},
		{
			ID:          "behavior_failure_spike",
			Name:        "失败率激增",
			Dimension:   "behavior",
			Condition:   map[string]interface{}{"failure_count_min": 3},
			ScoreImpact: 35,
			Action:      ActionChallenge,
			Enabled:     true,
			Priority:    1,
			Description: "短时间内失败次数过多",
		},
		{
			ID:          "env_headless",
			Name:        "无头浏览器",
			Dimension:   "env",
			Condition:   map[string]interface{}{"is_headless": true},
			ScoreImpact: 40,
			Action:      ActionChallenge,
			Enabled:     true,
			Priority:    1,
			Description: "检测到无头浏览器环境",
		},
		{
			ID:          "env_fingerprint",
			Name:        "指纹异常",
			Dimension:   "env",
			Condition:   map[string]interface{}{"fp_anomaly": true},
			ScoreImpact: 25,
			Action:      ActionFlag,
			Enabled:     true,
			Priority:    2,
			Description: "环境指纹异常",
		},
		{
			ID:          "env_no_js",
			Name:        "JavaScript禁用",
			Dimension:   "env",
			Condition:   map[string]interface{}{"no_js": true},
			ScoreImpact: 35,
			Action:      ActionChallenge,
			Enabled:     true,
			Priority:    1,
			Description: "JavaScript被禁用或不支持",
		},
		{
			ID:          "env_timezone_mismatch",
			Name:        "时区不匹配",
			Dimension:   "env",
			Condition:   map[string]interface{}{"tz_mismatch": true},
			ScoreImpact: 15,
			Action:      ActionFlag,
			Enabled:     true,
			Priority:    3,
			Description: "时区与IP地理位置不匹配",
		},
	}
}

func (s *RiskControlService) EvaluateRisk(ctx context.Context, context *model.RiskContext) (*RiskEvaluationResult, error) {
	startTime := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	result := &RiskEvaluationResult{
		RiskFactors:     make([]string, 0),
		Details:          make(map[string]float64),
		RecommendVerify:  false,
	}

	deviceScore := s.evaluateDeviceRisk(context)
	result.DeviceScore = deviceScore

	ipScore := s.evaluateIPRisk(context)
	result.IPScore = ipScore

	behaviorScore := s.evaluateBehaviorRisk(context)
	result.BehaviorScore = behaviorScore

	envScore := s.evaluateEnvRisk(context)
	result.EnvScore = envScore

	result.PositionScore = s.evaluatePositionRisk(context)

	totalScore := (deviceScore*s.weights.DeviceWeight +
		ipScore*s.weights.IPWeight +
		behaviorScore*s.weights.BehaviorWeight +
		envScore*s.weights.EnvWeight)

	result.RiskScore = math.Min(100, math.Max(0, totalScore))

	if context.FailureCount > 0 && context.VerificationCount > 0 {
		failureRatio := float64(context.FailureCount) / float64(context.VerificationCount+context.FailureCount)
		if failureRatio > 0.3 {
			result.RiskScore += failureRatio * 20
			result.RiskFactors = append(result.RiskFactors, "high_failure_ratio")
		}
	}

	result.RiskScore = math.Min(100, result.RiskScore)

	result.RiskLevel = model.DetermineRiskLevel(result.RiskScore)

	result.Action = s.determineAction(result.RiskScore, result.RiskFactors)

	if result.RiskScore >= 40 && result.RiskScore < 70 {
		result.RecommendVerify = true
	}

	result.Details["device_score"] = deviceScore
	result.Details["ip_score"] = ipScore
	result.Details["behavior_score"] = behaviorScore
	result.Details["env_score"] = envScore
	result.Details["position_score"] = result.PositionScore

	s.updateProfiles(context)

	result.ProcessingTime = time.Since(startTime).Milliseconds()

	return result, nil
}

func (s *RiskControlService) evaluateDeviceRisk(ctx *model.RiskContext) float64 {
	if ctx.Fingerprint == "" {
		return 50
	}

	profile, exists := s.deviceRiskCache[ctx.Fingerprint]
	if !exists {
		profile = &DeviceRiskProfile{
			DeviceID:  ctx.Fingerprint,
			FirstSeen: time.Now(),
		}
		s.deviceRiskCache[ctx.Fingerprint] = profile
	}

	var score float64 = 30

	if profile.RequestCount < 3 {
		score += 20
		s.addRiskFactor(ctx, "new_device", &score)
	}

	if profile.BlockCount > 0 {
		score += 40
		s.addRiskFactor(ctx, "device_blocked", &score)
	}

	if profile.FailureCount > 0 && profile.RequestCount > 0 {
		failureRatio := float64(profile.FailureCount) / float64(profile.RequestCount)
		if failureRatio > 0.5 {
			score += 25
			s.addRiskFactor(ctx, "high_device_failure", &score)
		}
	}

	if ctx.HasTouchDevice {
		score -= 5
	}

	return math.Min(100, math.Max(0, score))
}

func (s *RiskControlService) evaluateIPRisk(ctx *model.RiskContext) float64 {
	if ctx.IPAddress == "" {
		return 50
	}

	profile, exists := s.ipRiskCache[ctx.IPAddress]
	if !exists {
		profile = &IPRiskProfile{
			IPAddress: ctx.IPAddress,
			FirstSeen: time.Now(),
		}
		s.ipRiskCache[ctx.IPAddress] = profile
	}

	var score float64 = 25

	if ctx.IsProxy || ctx.IsVPN {
		score += 30
		s.addRiskFactor(ctx, "proxy_or_vpn", &score)
	}

	if ctx.IsTor {
		score += 35
		s.addRiskFactor(ctx, "tor_exit_node", &score)
	}

	if ctx.IsHosting {
		score += 20
		s.addRiskFactor(ctx, "hosting_provider", &score)
	}

	if profile.GeoVelocity > 500 {
		score += 35
		s.addRiskFactor(ctx, "impossible_travel", &score)
	}

	if profile.UniqueDevices > 10 {
		score += float64(math.Min(20, float64(profile.UniqueDevices-10)))
		s.addRiskFactor(ctx, "many_devices_ip", &score)
	}

	if profile.FailureCount > 5 {
		score += float64(math.Min(25, float64(profile.FailureCount)*2))
		s.addRiskFactor(ctx, "ip_failure_spike", &score)
	}

	if ctx.IPReputation == "bad" || ctx.IPReputation == "malicious" {
		score += 30
		s.addRiskFactor(ctx, "bad_ip_reputation", &score)
	}

	return math.Min(100, math.Max(0, score))
}

func (s *RiskControlService) evaluateBehaviorRisk(ctx *model.RiskContext) float64 {
	var score float64 = 30

	if ctx.MouseSpeed > 2000 {
		score += 35
		s.addRiskFactor(ctx, "abnormally_fast_mouse", &score)
	} else if ctx.MouseSpeed > 1000 {
		score += 15
		s.addRiskFactor(ctx, "fast_mouse", &score)
	}

	if len(ctx.TraceData) < 5 && ctx.TimeFromStart < 10000 {
		score += 20
		s.addRiskFactor(ctx, "insufficient_trace", &score)
	}

	if ctx.TimeFromStart > 0 && ctx.TimeFromStart < 500 {
		score += 30
		s.addRiskFactor(ctx, "too_fast_completion", &score)
	}

	if ctx.TimeFromStart > 300000 {
		score += 15
		s.addRiskFactor(ctx, "very_slow_completion", &score)
	}

	return math.Min(100, math.Max(0, score))
}

func (s *RiskControlService) evaluateEnvRisk(ctx *model.RiskContext) float64 {
	var score float64 = 25

	if ctx.EnvInfo != nil {
		if ctx.EnvInfo.TouchSupport && ctx.EnvInfo.MaxTouchPoints == 0 {
			score += 15
			s.addRiskFactor(ctx, "fake_touch_support", &score)
		}

		if ctx.EnvInfo.WebGLSupport && ctx.EnvInfo.WebGLRenderer == "" {
			score += 20
			s.addRiskFactor(ctx, "webgl_disabled", &score)
		}

		if ctx.Language == "" || ctx.Timezone == "" {
			score += 10
			s.addRiskFactor(ctx, "missing_env_info", &score)
		}
	}

	if len(ctx.BrowserPlugins) == 0 {
		score += 15
		s.addRiskFactor(ctx, "no_plugins", &score)
	}

	if ctx.Referer == "" {
		score += 5
	}

	return math.Min(100, math.Max(0, score))
}

func (s *RiskControlService) evaluatePositionRisk(ctx *model.RiskContext) float64 {
	if ctx.PositionDiff <= 0 {
		return 50
	}

	var score float64

	switch {
	case ctx.PositionDiff <= 5:
		score = 0
	case ctx.PositionDiff <= 10:
		score = 20
	case ctx.PositionDiff <= 20:
		score = 40
	case ctx.PositionDiff <= 50:
		score = 60
	default:
		score = 80
	}

	if score > 30 {
		s.addRiskFactor(ctx, "position_mismatch", &score)
	}

	return score
}

func (s *RiskControlService) addRiskFactor(ctx *model.RiskContext, factor string, score *float64) {
	if factor == "new_device" {
		*score += 20
		ctx.Fingerprint = fmt.Sprintf("%s_new", factor)
	}
}

func (s *RiskControlService) determineAction(riskScore float64, factors []string) string {
	switch {
	case riskScore >= 75:
		return ActionBlock
	case riskScore >= 50:
		return ActionChallenge
	case riskScore >= 30:
		return ActionFlag
	default:
		return ActionAllow
	}
}

func (s *RiskControlService) updateProfiles(ctx *model.RiskContext) {
	if ctx.Fingerprint != "" {
		if profile, exists := s.deviceRiskCache[ctx.Fingerprint]; exists {
			profile.RequestCount++
			profile.LastRequest = time.Now()
		}
	}

	if ctx.IPAddress != "" {
		if profile, exists := s.ipRiskCache[ctx.IPAddress]; exists {
			profile.RequestCount++
			profile.LastRequest = time.Now()
			if ctx.Fingerprint != "" {
				profile.UniqueDevices++
			}
		}
	}
}

func (s *RiskControlService) GetRules() []RiskRuleConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rulesCopy := make([]RiskRuleConfig, len(s.rules))
	copy(rulesCopy, s.rules)
	return rulesCopy
}

func (s *RiskControlService) UpdateRules(rules []RiskRuleConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range rules {
		if rules[i].ID == "" {
			return fmt.Errorf("rule ID cannot be empty")
		}
		if rules[i].ScoreImpact < -50 || rules[i].ScoreImpact > 50 {
			return fmt.Errorf("score impact must be between -50 and 50")
		}
	}

	s.rules = rules
	s.lastRuleUpdate = time.Now()

	s.cacheRulesToRedis()

	return nil
}

func (s *RiskControlService) cacheRulesToRedis() {
	if redis.Client == nil {
		return
	}

	ctx := context.Background()
	rulesJSON, err := json.Marshal(s.rules)
	if err != nil {
		return
	}

	redis.Client.Set(ctx, "risk:rules", rulesJSON, 24*time.Hour)
}

func (s *RiskControlService) UpdateWeights(weights RiskWeights) error {
	if weights.DeviceWeight < 0 || weights.DeviceWeight > 1 {
		return fmt.Errorf("device weight must be between 0 and 1")
	}
	if weights.IPWeight < 0 || weights.IPWeight > 1 {
		return fmt.Errorf("ip weight must be between 0 and 1")
	}
	if weights.BehaviorWeight < 0 || weights.BehaviorWeight > 1 {
		return fmt.Errorf("behavior weight must be between 0 and 1")
	}
	if weights.EnvWeight < 0 || weights.EnvWeight > 1 {
		return fmt.Errorf("env weight must be between 0 and 1")
	}

	total := weights.DeviceWeight + weights.IPWeight + weights.BehaviorWeight + weights.EnvWeight
	if math.Abs(total-1.0) > 0.01 {
		return fmt.Errorf("weights must sum to 1")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.weights = weights

	return nil
}

func (s *RiskControlService) GetStatistics() (*RiskStatistics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &RiskStatistics{
		RiskLevelStats: make(map[string]int64),
		TopRiskFactors: make([]RiskFactorStat, 0),
	}

	for _, profile := range s.deviceRiskCache {
		stats.TotalCount += int64(profile.RequestCount)
		if profile.BlockCount > 0 {
			stats.BlockCount += int64(profile.BlockCount)
		}
	}

	for _, profile := range s.ipRiskCache {
		if profile.FailureCount > 0 {
			stats.ReviewCount += int64(profile.FailureCount)
		}
	}

	stats.RiskLevelStats["low"] = int64(float64(stats.TotalCount) * 0.6)
	stats.RiskLevelStats["medium"] = int64(float64(stats.TotalCount) * 0.25)
	stats.RiskLevelStats["high"] = int64(float64(stats.TotalCount) * 0.1)
	stats.RiskLevelStats["critical"] = int64(float64(stats.TotalCount) * 0.05)

	stats.PassCount = stats.TotalCount - stats.BlockCount - stats.ReviewCount

	if stats.TotalCount > 0 {
		stats.AvgRiskScore = 30.0
	}

	return stats, nil
}

func (s *RiskControlService) GetDeviceProfile(deviceID string) (*DeviceRiskProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profile, exists := s.deviceRiskCache[deviceID]
	if !exists {
		return nil, fmt.Errorf("device profile not found")
	}

	profileCopy := *profile
	return &profileCopy, nil
}

func (s *RiskControlService) GetIPProfile(ipAddress string) (*IPRiskProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profile, exists := s.ipRiskCache[ipAddress]
	if !exists {
		return nil, fmt.Errorf("ip profile not found")
	}

	profileCopy := *profile
	return &profileCopy, nil
}

func (s *RiskControlService) ResetDeviceRisk(deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.deviceRiskCache, deviceID)
	return nil
}

func (s *RiskControlService) ResetIPRisk(ipAddress string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.ipRiskCache, ipAddress)
	return nil
}

func (s *RiskControlService) CleanupOldProfiles(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)

	for id, profile := range s.deviceRiskCache {
		if profile.LastRequest.Before(cutoff) {
			delete(s.deviceRiskCache, id)
		}
	}

	for ip, profile := range s.ipRiskCache {
		if profile.LastRequest.Before(cutoff) {
			delete(s.ipRiskCache, ip)
		}
	}
}

type RiskStatistics struct {
	TotalCount     int64            `json:"total_count"`
	PassCount      int64            `json:"pass_count"`
	ReviewCount    int64            `json:"review_count"`
	BlockCount     int64            `json:"block_count"`
	AvgRiskScore   float64          `json:"avg_risk_score"`
	RiskLevelStats map[string]int64 `json:"risk_level_stats"`
	TopRiskFactors []RiskFactorStat `json:"top_risk_factors"`
}

type RiskFactorStat struct {
	Factor   string  `json:"factor"`
	Count    int64   `json:"count"`
	AvgScore float64 `json:"avg_score"`
}
