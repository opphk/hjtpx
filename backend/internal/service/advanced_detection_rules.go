package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

type AdvancedDetectionRule struct {
	Name            string
	Description     string
	Category        string
	Condition       func(*AdvancedDetectionContext) bool
	Weight          float64
	Priority        int
	Severity        float64
	Enabled         bool
	CombinationType string
	RequiredRules   []string
	MinTriggerCount int
	TriggerCount    int
	LastTriggered   time.Time
	HitRate         float64
	FalsePositiveRate float64
	TruePositiveRate  float64
	mu              sync.RWMutex
}

type AdvancedDetectionContext struct {
	IPAddress          string
	UserAgent          string
	Fingerprint        string
	DeviceFingerprint  string
	SessionID          string
	BehaviorFeatures   *BehaviorFeatures
	TrajectoryFeatures *TrajectoryFeatures
	ClickFeatures      *ClickFeatures
	KeyboardFeatures   *KeyboardFeatures
	SliderFeatures     *AdvancedSliderFeatures
	CaptchaFeatures    *CaptchaFeatures
	IPFeatures         *IPFeatures
	DeviceFeatures     *DeviceFeatures
	SessionFeatures    *SessionFeatures
	TimeFeatures       *TimeFeatures
}

type TrajectoryFeatures struct {
	CurvatureAverage     float64
	CurvatureVariance    float64
	CurvatureMax         float64
	CurvatureMin         float64
	Smoothness           float64
	SpeedConsistency     float64
	JitterScore          float64
	AccelerationVariance float64
	TrajectoryScore      float64
}

type ClickFeatures struct {
	IntervalAverage    float64
	IntervalVariance   float64
	PositionVarianceX  float64
	PositionVarianceY  float64
	PositionEntropy    float64
	ClusteringScore    float64
	Regularity         float64
	ReleasePositionX   float64
	ReleasePositionY   float64
	Precision          float64
	DoubleClickRate    float64
}

type KeyboardFeatures struct {
	TypingSpeed      float64
	HoldTimeAverage  float64
	HoldTimeVariance float64
	IntervalVariance float64
	Regularity       float64
	ErrorRate        float64
	RhythmScore      float64
	CommonPairsCount int
}

type AdvancedSliderFeatures struct {
	ReleasePosition   float64
	Precision         float64
	TrajectoryLength  float64
	Directness       float64
	MicroCorrectionCount int
	AverageSpeed     float64
	SpeedVariation   float64
}

type CaptchaFeatures struct {
	AttemptCount      int
	FailureCount      int
	SuccessCount      int
	FailureRate       float64
	AttemptFrequency  float64
	TimeToFirstAttempt float64
	AverageSolveTime  float64
}

type IPFeatures struct {
	Subnet             string
	ASN                int
	Country            string
	IsProxy            bool
	IsVPN              bool
	IsTor              bool
	IsHosting          bool
	Reputation         float64
	RequestCount       int
	RequestFrequency   float64
	UniqueSessions     int
	FailedAttempts     int
}

type DeviceFeatures struct {
	FingerprintHash   string
	ScreenResolution  string
	ColorDepth        int
	Timezone          string
	Language          string
	Platform          string
	CanvasHash        string
	WebGLRenderer     string
	PluginsCount      int
	DoNotTrack        bool
	UserAgent         string
}

type SessionFeatures struct {
	Duration           time.Duration
	PageViews          int
	UniquePages        int
	UniqueSessions     int
	InteractionCount   int
	MouseMovements     int
	ClickCount         int
	KeyStrokes        int
	ScrollEvents       int
	FocusLossCount     int
	AvgTimePerPage     float64
	BounceRate         float64
	IsNewVisitor       bool
}

type TimeFeatures struct {
	HourOfDay         int
	DayOfWeek         int
	IsWeekend         bool
	IsBusinessHour    bool
	TimeSinceLastActivity time.Duration
	ActivityDuration  time.Duration
}

type AdvancedRuleEngine struct {
	rules       map[string]*AdvancedDetectionRule
	ruleGroups  map[string][]string
	contexts    map[string]*AdvancedDetectionContext
	statistics  *RuleStatistics
	combinators map[string]RuleCombinator
	mu          sync.RWMutex
}

type RuleStatistics struct {
	TotalEvaluations    int64
	TotalTriggers       int64
	RuleHitCounts       map[string]int64
	CategoryHitCounts   map[string]int64
	AverageScore        float64
	TopTriggeredRules   []RuleStat
	LastUpdated         time.Time
	mu                  sync.Mutex
}

type RuleStat struct {
	Name        string
	HitCount    int64
	HitRate     float64
	Accuracy    float64
}

type RuleCombinator interface {
	Combine([]*AdvancedDetectionRule, *AdvancedDetectionContext) bool
}

type ANDCombinator struct{}

type ORCombinator struct{}

type ThresholdCombinator struct {
	MinTriggers int
}

type WeightedSumCombinator struct {
	Threshold float64
}

func NewAdvancedRuleEngine() *AdvancedRuleEngine {
	engine := &AdvancedRuleEngine{
		rules:      make(map[string]*AdvancedDetectionRule),
		ruleGroups: make(map[string][]string),
		contexts:   make(map[string]*AdvancedDetectionContext),
		statistics: &RuleStatistics{
			RuleHitCounts:     make(map[string]int64),
			CategoryHitCounts: make(map[string]int64),
			TopTriggeredRules: make([]RuleStat, 0),
		},
		combinators: make(map[string]RuleCombinator),
	}

	engine.combinators["AND"] = &ANDCombinator{}
	engine.combinators["OR"] = &ORCombinator{}
	engine.combinators["THRESHOLD"] = &ThresholdCombinator{MinTriggers: 2}
	engine.combinators["WEIGHTED"] = &WeightedSumCombinator{Threshold: 0.5}

	engine.initializeAdvancedRules()

	return engine
}

func (ac *ANDCombinator) Combine(rules []*AdvancedDetectionRule, ctx *AdvancedDetectionContext) bool {
	for _, rule := range rules {
		if !rule.Condition(ctx) {
			return false
		}
	}
	return len(rules) > 0
}

func (oc *ORCombinator) Combine(rules []*AdvancedDetectionRule, ctx *AdvancedDetectionContext) bool {
	for _, rule := range rules {
		if rule.Condition(ctx) {
			return true
		}
	}
	return false
}

func (tc *ThresholdCombinator) Combine(rules []*AdvancedDetectionRule, ctx *AdvancedDetectionContext) bool {
	triggered := 0
	for _, rule := range rules {
		if rule.Condition(ctx) {
			triggered++
		}
	}
	return triggered >= tc.MinTriggers
}

func (wc *WeightedSumCombinator) Combine(rules []*AdvancedDetectionRule, ctx *AdvancedDetectionContext) bool {
	totalWeight := 0.0
	triggeredWeight := 0.0

	for _, rule := range rules {
		totalWeight += rule.Weight
		if rule.Condition(ctx) {
			triggeredWeight += rule.Weight
		}
	}

	if totalWeight == 0 {
		return false
	}

	return (triggeredWeight / totalWeight) >= wc.Threshold
}

func (are *AdvancedRuleEngine) initializeAdvancedRules() {
	rules := []AdvancedDetectionRule{
		{
			Name:            "trajectory_speed_too_fast",
			Description:     "轨迹移动速度异常快",
			Category:        "speed",
			Condition:       are.checkTrajectorySpeedTooFast,
			Weight:          35,
			Priority:        1,
			Severity:        0.85,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
		{
			Name:            "trajectory_speed_too_slow",
			Description:     "轨迹移动速度异常慢",
			Category:        "speed",
			Condition:       are.checkTrajectorySpeedTooSlow,
			Weight:          15,
			Priority:        3,
			Severity:        0.4,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
		{
			Name:            "trajectory_curvature_too_smooth",
			Description:     "轨迹曲率异常平滑",
			Category:        "trajectory",
			Condition:       are.checkTrajectoryCurvatureSmooth,
			Weight:          30,
			Priority:        2,
			Severity:        0.75,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
		{
			Name:            "trajectory_curvature_abnormal_jitter",
			Description:     "轨迹曲率异常抖动",
			Category:        "trajectory",
			Condition:       are.checkTrajectoryCurvatureJitter,
			Weight:          25,
			Priority:        2,
			Severity:        0.65,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
		{
			Name:            "click_interval_too_short",
			Description:     "点击时间间隔过短",
			Category:        "click",
			Condition:       are.checkClickIntervalTooShort,
			Weight:          28,
			Priority:        2,
			Severity:        0.7,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
		{
			Name:            "click_interval_too_long",
			Description:     "点击时间间隔过长",
			Category:        "click",
			Condition:       are.checkClickIntervalTooLong,
			Weight:          12,
			Priority:        4,
			Severity:        0.35,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
		{
			Name:            "click_position_too_concentrated",
			Description:     "点击位置集中度过高",
			Category:        "click",
			Condition:       are.checkClickPositionConcentrated,
			Weight:          25,
			Priority:        2,
			Severity:        0.65,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
		{
			Name:            "slider_release_precision_too_high",
			Description:     "滑块释放位置精确度过高",
			Category:        "slider",
			Condition:       are.checkSliderReleasePrecision,
			Weight:          35,
			Priority:        1,
			Severity:        0.9,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
		{
			Name:            "captcha_attempt_frequency_too_high",
			Description:     "验证码尝试频率异常",
			Category:        "captcha",
			Condition:       are.checkCaptchaAttemptFrequency,
			Weight:          30,
			Priority:        2,
			Severity:        0.8,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
		{
			Name:            "captcha_failure_rate_too_high",
			Description:     "验证码失败率异常",
			Category:        "captcha",
			Condition:       are.checkCaptchaFailureRate,
			Weight:          32,
			Priority:        1,
			Severity:        0.85,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
		{
			Name:            "device_fingerprint_duplicate",
			Description:     "设备指纹重复检测",
			Category:        "device",
			Condition:       are.checkDeviceFingerprintDuplicate,
			Weight:          40,
			Priority:        1,
			Severity:        0.95,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
		{
			Name:            "ip_subnet_abnormal_access",
			Description:     "IP段异常访问",
			Category:        "ip",
			Condition:       are.checkIPSubnetAbnormal,
			Weight:          28,
			Priority:        2,
			Severity:        0.75,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
		{
			Name:            "session_behavior_abnormal",
			Description:     "会话行为异常",
			Category:        "session",
			Condition:       are.checkSessionBehaviorAbnormal,
			Weight:          30,
			Priority:        2,
			Severity:        0.8,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
		{
			Name:            "mouse_trajectory_mechanical",
			Description:     "鼠标移动轨迹机械",
			Category:        "trajectory",
			Condition:       are.checkMouseTrajectoryMechanical,
			Weight:          32,
			Priority:        1,
			Severity:        0.85,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
		{
			Name:            "keyboard_input_rhythm_abnormal",
			Description:     "键盘输入节奏异常",
			Category:        "keyboard",
			Condition:       are.checkKeyboardInputRhythm,
			Weight:          28,
			Priority:        2,
			Severity:        0.7,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
		{
			Name:            "combined_trajectory_anomaly",
			Description:     "综合轨迹异常(曲率+速度)",
			Category:        "combined",
			Condition:       are.checkCombinedTrajectoryAnomaly,
			Weight:          45,
			Priority:        1,
			Severity:        0.95,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{"trajectory_curvature_too_smooth", "trajectory_speed_too_fast"},
			MinTriggerCount: 2,
		},
		{
			Name:            "combined_click_pattern_anomaly",
			Description:     "综合点击模式异常(间隔+位置)",
			Category:        "combined",
			Condition:       are.checkCombinedClickPatternAnomaly,
			Weight:          42,
			Priority:        1,
			Severity:        0.9,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{"click_interval_too_short", "click_position_too_concentrated"},
			MinTriggerCount: 2,
		},
		{
			Name:            "captcha_rapid_retry",
			Description:     "验证码快速重试",
			Category:        "captcha",
			Condition:       are.checkCaptchaRapidRetry,
			Weight:          38,
			Priority:        1,
			Severity:        0.88,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{"captcha_failure_rate_too_high", "captcha_attempt_frequency_too_high"},
			MinTriggerCount: 2,
		},
		{
			Name:            "device_ip_session_correlation",
			Description:     "设备IP会话关联异常",
			Category:        "combined",
			Condition:       are.checkDeviceIPSessionCorrelation,
			Weight:          35,
			Priority:        2,
			Severity:        0.82,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{"device_fingerprint_duplicate", "ip_subnet_abnormal_access"},
			MinTriggerCount: 2,
		},
		{
			Name:            "proxy_vpn_tor_detection",
			Description:     "代理/VPN/Tor检测",
			Category:        "network",
			Condition:       are.checkProxyVPNTor,
			Weight:          30,
			Priority:        2,
			Severity:        0.75,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
		{
			Name:            "timing_behavior_inconsistency",
			Description:     "时间行为不一致",
			Category:        "timing",
			Condition:       are.checkTimingBehaviorInconsistency,
			Weight:          25,
			Priority:        3,
			Severity:        0.6,
			Enabled:         true,
			CombinationType: "AND",
			RequiredRules:   []string{},
			MinTriggerCount: 1,
		},
	}

	for i := range rules {
		are.rules[rules[i].Name] = &rules[i]
	}

	are.ruleGroups["speed"] = []string{"trajectory_speed_too_fast", "trajectory_speed_too_slow"}
	are.ruleGroups["trajectory"] = []string{"trajectory_curvature_too_smooth", "trajectory_curvature_abnormal_jitter", "mouse_trajectory_mechanical"}
	are.ruleGroups["click"] = []string{"click_interval_too_short", "click_interval_too_long", "click_position_too_concentrated"}
	are.ruleGroups["slider"] = []string{"slider_release_precision_too_high"}
	are.ruleGroups["captcha"] = []string{"captcha_attempt_frequency_too_high", "captcha_failure_rate_too_high", "captcha_rapid_retry"}
	are.ruleGroups["device"] = []string{"device_fingerprint_duplicate"}
	are.ruleGroups["ip"] = []string{"ip_subnet_abnormal_access"}
	are.ruleGroups["session"] = []string{"session_behavior_abnormal"}
	are.ruleGroups["keyboard"] = []string{"keyboard_input_rhythm_abnormal"}
	are.ruleGroups["combined"] = []string{"combined_trajectory_anomaly", "combined_click_pattern_anomaly", "device_ip_session_correlation"}
	are.ruleGroups["network"] = []string{"proxy_vpn_tor_detection"}
	are.ruleGroups["timing"] = []string{"timing_behavior_inconsistency"}
}

func (are *AdvancedRuleEngine) checkTrajectorySpeedTooFast(ctx *AdvancedDetectionContext) bool {
	if ctx.BehaviorFeatures == nil && ctx.TrajectoryFeatures == nil {
		return false
	}

	var avgSpeed float64
	if ctx.BehaviorFeatures != nil {
		avgSpeed = ctx.BehaviorFeatures.AvgSpeed
	}
	if ctx.TrajectoryFeatures != nil && ctx.TrajectoryFeatures.SpeedConsistency > 0 {
		avgSpeed = ctx.TrajectoryFeatures.SpeedConsistency * 2000
	}

	return avgSpeed > 1800
}

func (are *AdvancedRuleEngine) checkTrajectorySpeedTooSlow(ctx *AdvancedDetectionContext) bool {
	if ctx.BehaviorFeatures == nil {
		return false
	}

	return ctx.BehaviorFeatures.AvgSpeed < 10 && ctx.BehaviorFeatures.AvgSpeed > 0
}

func (are *AdvancedRuleEngine) checkTrajectoryCurvatureSmooth(ctx *AdvancedDetectionContext) bool {
	if ctx.TrajectoryFeatures == nil {
		return false
	}

	return ctx.TrajectoryFeatures.CurvatureAverage < 0.01 && ctx.TrajectoryFeatures.CurvatureVariance < 0.001
}

func (are *AdvancedRuleEngine) checkTrajectoryCurvatureJitter(ctx *AdvancedDetectionContext) bool {
	if ctx.TrajectoryFeatures == nil {
		return false
	}

	return ctx.TrajectoryFeatures.CurvatureVariance > 0.8 && ctx.TrajectoryFeatures.JitterScore > 0.5
}

func (are *AdvancedRuleEngine) checkClickIntervalTooShort(ctx *AdvancedDetectionContext) bool {
	if ctx.ClickFeatures == nil {
		return false
	}

	return ctx.ClickFeatures.IntervalAverage > 0 && ctx.ClickFeatures.IntervalAverage < 30
}

func (are *AdvancedRuleEngine) checkClickIntervalTooLong(ctx *AdvancedDetectionContext) bool {
	if ctx.ClickFeatures == nil {
		return false
	}

	return ctx.ClickFeatures.IntervalAverage > 5000
}

func (are *AdvancedRuleEngine) checkClickPositionConcentrated(ctx *AdvancedDetectionContext) bool {
	if ctx.ClickFeatures == nil {
		return false
	}

	positionVariance := ctx.ClickFeatures.PositionVarianceX + ctx.ClickFeatures.PositionVarianceY

	return ctx.ClickFeatures.ClusteringScore > 0.9 || (ctx.ClickFeatures.PositionEntropy < 1.5 && positionVariance < 1000)
}

func (are *AdvancedRuleEngine) checkSliderReleasePrecision(ctx *AdvancedDetectionContext) bool {
	if ctx.SliderFeatures == nil {
		return false
	}

	return ctx.SliderFeatures.Precision > 0.98 && ctx.SliderFeatures.Directness > 0.99
}

func (are *AdvancedRuleEngine) checkCaptchaAttemptFrequency(ctx *AdvancedDetectionContext) bool {
	if ctx.CaptchaFeatures == nil {
		return false
	}

	return ctx.CaptchaFeatures.AttemptFrequency > 5 && ctx.CaptchaFeatures.AttemptCount > 3
}

func (are *AdvancedRuleEngine) checkCaptchaFailureRate(ctx *AdvancedDetectionContext) bool {
	if ctx.CaptchaFeatures == nil {
		return false
	}

	return ctx.CaptchaFeatures.FailureRate > 0.7 && ctx.CaptchaFeatures.AttemptCount >= 3
}

func (are *AdvancedRuleEngine) checkDeviceFingerprintDuplicate(ctx *AdvancedDetectionContext) bool {
	if ctx.DeviceFeatures == nil || ctx.SessionFeatures == nil {
		return false
	}

	hash := ctx.DeviceFeatures.FingerprintHash
	if hash == "" {
		hasher := sha256.New()
		hasher.Write([]byte(ctx.DeviceFeatures.ScreenResolution))
		hasher.Write([]byte(ctx.DeviceFeatures.CanvasHash))
		hasher.Write([]byte(ctx.DeviceFeatures.WebGLRenderer))
		hash = hex.EncodeToString(hasher.Sum(nil))
	}

	fingerprintKey := fmt.Sprintf("device:%s", hash)
	count := are.getContextCount(fingerprintKey)

	return count > 5
}

func (are *AdvancedRuleEngine) checkIPSubnetAbnormal(ctx *AdvancedDetectionContext) bool {
	if ctx.IPFeatures == nil || ctx.SessionFeatures == nil {
		return false
	}

	if ctx.IPFeatures.IsProxy || ctx.IPFeatures.IsVPN || ctx.IPFeatures.IsTor || ctx.IPFeatures.IsHosting {
		if ctx.SessionFeatures.UniqueSessions > 3 {
			return true
		}
	}

	if ctx.IPFeatures.RequestFrequency > 100 {
		return true
	}

	return false
}

func (are *AdvancedRuleEngine) checkSessionBehaviorAbnormal(ctx *AdvancedDetectionContext) bool {
	if ctx.SessionFeatures == nil {
		return false
	}

	if ctx.SessionFeatures.Duration > 0 {
		interactionsPerMinute := float64(ctx.SessionFeatures.InteractionCount) / ctx.SessionFeatures.Duration.Minutes()
		if interactionsPerMinute > 100 {
			return true
		}
	}

	if ctx.SessionFeatures.BounceRate > 0.9 {
		return true
	}

	if ctx.SessionFeatures.FocusLossCount == 0 && ctx.SessionFeatures.Duration > 5*time.Minute {
		return true
	}

	return false
}

func (are *AdvancedRuleEngine) checkMouseTrajectoryMechanical(ctx *AdvancedDetectionContext) bool {
	if ctx.TrajectoryFeatures == nil || ctx.BehaviorFeatures == nil {
		return false
	}

	speedTooConstant := ctx.TrajectoryFeatures.SpeedConsistency > 0.98
	curvatureTooLow := ctx.TrajectoryFeatures.CurvatureAverage < 0.005
	smoothnessTooHigh := ctx.TrajectoryFeatures.Smoothness > 0.98
	accelerationTooLow := ctx.TrajectoryFeatures.AccelerationVariance < 0.001

	mechanicalScore := 0
	if speedTooConstant {
		mechanicalScore++
	}
	if curvatureTooLow {
		mechanicalScore++
	}
	if smoothnessTooHigh {
		mechanicalScore++
	}
	if accelerationTooLow {
		mechanicalScore++
	}

	return mechanicalScore >= 3
}

func (are *AdvancedRuleEngine) checkKeyboardInputRhythm(ctx *AdvancedDetectionContext) bool {
	if ctx.KeyboardFeatures == nil {
		return false
	}

	if ctx.KeyboardFeatures.TypingSpeed > 15 {
		return true
	}

	if ctx.KeyboardFeatures.HoldTimeVariance < 10 && ctx.KeyboardFeatures.HoldTimeAverage < 50 {
		return true
	}

	if ctx.KeyboardFeatures.Regularity > 0.98 {
		return true
	}

	return false
}

func (are *AdvancedRuleEngine) checkCombinedTrajectoryAnomaly(ctx *AdvancedDetectionContext) bool {
	smoothCurvature := are.checkTrajectoryCurvatureSmooth(ctx)
	fastSpeed := are.checkTrajectorySpeedTooFast(ctx)

	return smoothCurvature && fastSpeed
}

func (are *AdvancedRuleEngine) checkCombinedClickPatternAnomaly(ctx *AdvancedDetectionContext) bool {
	shortInterval := are.checkClickIntervalTooShort(ctx)
	concentratedPosition := are.checkClickPositionConcentrated(ctx)

	return shortInterval && concentratedPosition
}

func (are *AdvancedRuleEngine) checkCaptchaRapidRetry(ctx *AdvancedDetectionContext) bool {
	highFailureRate := are.checkCaptchaFailureRate(ctx)
	highFrequency := are.checkCaptchaAttemptFrequency(ctx)

	return highFailureRate && highFrequency
}

func (are *AdvancedRuleEngine) checkDeviceIPSessionCorrelation(ctx *AdvancedDetectionContext) bool {
	duplicateDevice := are.checkDeviceFingerprintDuplicate(ctx)
	abnormalIP := are.checkIPSubnetAbnormal(ctx)

	return duplicateDevice && abnormalIP
}

func (are *AdvancedRuleEngine) checkProxyVPNTor(ctx *AdvancedDetectionContext) bool {
	if ctx.IPFeatures == nil {
		return false
	}

	return ctx.IPFeatures.IsProxy || ctx.IPFeatures.IsVPN || ctx.IPFeatures.IsTor
}

func (are *AdvancedRuleEngine) checkTimingBehaviorInconsistency(ctx *AdvancedDetectionContext) bool {
	if ctx.TimeFeatures == nil {
		return false
	}

	hour := ctx.TimeFeatures.HourOfDay
	isWeekend := ctx.TimeFeatures.IsWeekend
	activityDuration := ctx.TimeFeatures.ActivityDuration

	if isWeekend && hour >= 9 && hour <= 17 {
		if activityDuration < 30*time.Second {
			return true
		}
	}

	if !isWeekend && (hour < 9 || hour > 18) {
		if activityDuration < 20*time.Second {
			return true
		}
	}

	return false
}

func (are *AdvancedRuleEngine) getContextCount(key string) int {
	are.mu.RLock()
	defer are.mu.RUnlock()

	count := 0
	for _, ctx := range are.contexts {
		if ctx.SessionID == key {
			count++
		}
	}
	return count
}

func (are *AdvancedRuleEngine) Evaluate(ctx *AdvancedDetectionContext) *AdvancedRuleResult {
	are.mu.Lock()
	defer are.mu.Unlock()

	result := &AdvancedRuleResult{
		Timestamp:     time.Now(),
		TriggeredRules: make([]string, 0),
		RuleScores:    make(map[string]float64),
		CategoryScores: make(map[string]float64),
		Details:       make(map[string]interface{}),
	}

	if ctx == nil {
		return result
	}

	categoryScores := make(map[string]float64)
	categoryWeights := make(map[string]float64)

	for name, rule := range are.rules {
		if !rule.Enabled {
			continue
		}

		if rule.Condition(ctx) {
			rule.TriggerCount++
			rule.LastTriggered = time.Now()
			result.TriggeredRules = append(result.TriggeredRules, name)

			score := rule.Weight * rule.Severity
			result.RuleScores[name] = score

			categoryScores[rule.Category] += score
			categoryWeights[rule.Category] += rule.Weight

			are.statistics.RuleHitCounts[name]++
			are.statistics.CategoryHitCounts[rule.Category]++
		}
	}

	for category, score := range categoryScores {
		if categoryWeights[category] > 0 {
			normalizedScore := score / categoryWeights[category]
			result.CategoryScores[category] = normalizedScore
		}
	}

	totalScore := 0.0
	totalCategoryWeight := 0.0
	for category := range categoryWeights {
		categoryWeight := are.getCategoryWeight(category)
		totalScore += result.CategoryScores[category] * categoryWeight
		totalCategoryWeight += categoryWeight
	}

	if totalCategoryWeight > 0 {
		result.TotalScore = totalScore / totalCategoryWeight
	}

	result.TotalScore = math.Min(math.Max(result.TotalScore, 0), 1)
	result.IsBot = result.TotalScore > 0.5
	result.Confidence = are.calculateConfidence(result)

	are.statistics.TotalEvaluations++
	are.statistics.TotalTriggers += int64(len(result.TriggeredRules))
	are.updateTopTriggeredRules()

	return result
}

func (are *AdvancedRuleEngine) getCategoryWeight(category string) float64 {
	weights := map[string]float64{
		"speed":      0.15,
		"trajectory": 0.20,
		"click":      0.15,
		"slider":     0.15,
		"captcha":    0.10,
		"device":     0.08,
		"ip":         0.05,
		"session":    0.05,
		"keyboard":   0.05,
		"combined":   0.10,
		"network":    0.05,
		"timing":     0.02,
	}

	if weight, exists := weights[category]; exists {
		return weight
	}
	return 0.05
}

func (are *AdvancedRuleEngine) calculateConfidence(result *AdvancedRuleResult) float64 {
	confidence := 0.5

	if len(result.TriggeredRules) >= 5 {
		confidence += 0.15
	}

	if len(result.TriggeredRules) >= 10 {
		confidence += 0.15
	}

	categories := make(map[string]bool)
	for _, ruleName := range result.TriggeredRules {
		if rule, exists := are.rules[ruleName]; exists {
			categories[rule.Category] = true
		}
	}

	if len(categories) >= 4 {
		confidence += 0.15
	}

	highSeverityCount := 0
	for _, ruleName := range result.TriggeredRules {
		if rule, exists := are.rules[ruleName]; exists {
			if rule.Severity > 0.8 {
				highSeverityCount++
			}
		}
	}

	if highSeverityCount >= 2 {
		confidence += 0.1
	}

	return math.Min(confidence, 0.99)
}

func (are *AdvancedRuleEngine) updateTopTriggeredRules() {
	are.statistics.mu.Lock()
	defer are.statistics.mu.Unlock()

	stats := make([]RuleStat, 0, len(are.statistics.RuleHitCounts))
	for name, count := range are.statistics.RuleHitCounts {
		stat := RuleStat{
			Name:     name,
			HitCount: count,
		}
		if are.statistics.TotalEvaluations > 0 {
			stat.HitRate = float64(count) / float64(are.statistics.TotalEvaluations)
		}
		stats = append(stats, stat)
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].HitCount > stats[j].HitCount
	})

	if len(stats) > 10 {
		stats = stats[:10]
	}

	are.statistics.TopTriggeredRules = stats
	are.statistics.LastUpdated = time.Now()
}

type AdvancedRuleResult struct {
	TotalScore      float64
	TriggeredRules  []string
	RuleScores      map[string]float64
	CategoryScores  map[string]float64
	IsBot           bool
	Confidence      float64
	Timestamp       time.Time
	Details         map[string]interface{}
	Recommendations []string
}

func (are *AdvancedRuleEngine) GetStatistics() *RuleStatistics {
	are.statistics.mu.Lock()
	defer are.statistics.mu.Unlock()

	return &RuleStatistics{
		TotalEvaluations:    are.statistics.TotalEvaluations,
		TotalTriggers:        are.statistics.TotalTriggers,
		RuleHitCounts:        are.statistics.RuleHitCounts,
		CategoryHitCounts:    are.statistics.CategoryHitCounts,
		TopTriggeredRules:    are.statistics.TopTriggeredRules,
		LastUpdated:          are.statistics.LastUpdated,
	}
}

func (are *AdvancedRuleEngine) SetRuleWeight(ruleName string, weight float64) error {
	are.mu.Lock()
	defer are.mu.Unlock()

	rule, exists := are.rules[ruleName]
	if !exists {
		return fmt.Errorf("rule %s not found", ruleName)
	}

	rule.Weight = math.Max(0, math.Min(100, weight))
	return nil
}

func (are *AdvancedRuleEngine) SetCategoryWeight(category string, weight float64) {
	are.mu.Lock()
	defer are.mu.Unlock()

	weight = math.Max(0, math.Min(1, weight))
}

func (are *AdvancedRuleEngine) GetRule(ruleName string) (*AdvancedDetectionRule, bool) {
	are.mu.RLock()
	defer are.mu.RUnlock()

	rule, exists := are.rules[ruleName]
	return rule, exists
}

func (are *AdvancedRuleEngine) GetRulesByCategory(category string) []*AdvancedDetectionRule {
	are.mu.RLock()
	defer are.mu.RUnlock()

	ruleNames, exists := are.ruleGroups[category]
	if !exists {
		return []*AdvancedDetectionRule{}
	}

	rules := make([]*AdvancedDetectionRule, 0, len(ruleNames))
	for _, name := range ruleNames {
		if rule, exists := are.rules[name]; exists {
			rules = append(rules, rule)
		}
	}

	return rules
}

func (are *AdvancedRuleEngine) GetAllRules() []*AdvancedDetectionRule {
	are.mu.RLock()
	defer are.mu.RUnlock()

	rules := make([]*AdvancedDetectionRule, 0, len(are.rules))
	for _, rule := range are.rules {
		rules = append(rules, rule)
	}

	return rules
}

func (are *AdvancedRuleEngine) EnableRule(ruleName string) error {
	are.mu.Lock()
	defer are.mu.Unlock()

	rule, exists := are.rules[ruleName]
	if !exists {
		return fmt.Errorf("rule %s not found", ruleName)
	}

	rule.Enabled = true
	return nil
}

func (are *AdvancedRuleEngine) DisableRule(ruleName string) error {
	are.mu.Lock()
	defer are.mu.Unlock()

	rule, exists := are.rules[ruleName]
	if !exists {
		return fmt.Errorf("rule %s not found", ruleName)
	}

	rule.Enabled = false
	return nil
}

func (are *AdvancedRuleEngine) CreateCombinedRule(name string, ruleNames []string, combinatorType string) error {
	are.mu.Lock()
	defer are.mu.Unlock()

	_, exists := are.rules[name]
	if exists {
		return fmt.Errorf("rule %s already exists", name)
	}

	combinator, exists := are.combinators[combinatorType]
	if !exists {
		return fmt.Errorf("unknown combinator type: %s", combinatorType)
	}

	combinedRule := &AdvancedDetectionRule{
		Name:            name,
		Description:     fmt.Sprintf("Combined rule using %s combinator", combinatorType),
		Category:        "combined",
		Weight:          50,
		Priority:        1,
		Severity:        0.9,
		Enabled:         true,
		CombinationType: combinatorType,
		RequiredRules:   ruleNames,
		MinTriggerCount: len(ruleNames),
	}

	for _, reqName := range ruleNames {
		if _, exists := are.rules[reqName]; !exists {
			return fmt.Errorf("required rule %s not found", reqName)
		}
	}

	combinedRule.Condition = func(ctx *AdvancedDetectionContext) bool {
		rules := make([]*AdvancedDetectionRule, 0, len(ruleNames))
		for _, ruleName := range ruleNames {
			if rule, exists := are.rules[ruleName]; exists {
				rules = append(rules, rule)
			}
		}

		return combinator.Combine(rules, ctx)
	}

	are.rules[name] = combinedRule
	are.ruleGroups["combined"] = append(are.ruleGroups["combined"], name)

	return nil
}

func (are *AdvancedRuleEngine) ExportConfiguration() string {
	are.mu.RLock()
	defer are.mu.RUnlock()

	var lines []string
	lines = append(lines, "=== Advanced Detection Rules Configuration ===")
	lines = append(lines, fmt.Sprintf("\nTotal Rules: %d", len(are.rules)))
	lines = append(lines, fmt.Sprintf("Total Evaluations: %d", are.statistics.TotalEvaluations))
	lines = append(lines, fmt.Sprintf("Total Triggers: %d\n", are.statistics.TotalTriggers))

	for category, ruleNames := range are.ruleGroups {
		lines = append(lines, fmt.Sprintf("\n[%s]", strings.ToUpper(category)))
		for _, name := range ruleNames {
			if rule, exists := are.rules[name]; exists {
				enabled := "disabled"
				if rule.Enabled {
					enabled = "enabled"
				}
				lines = append(lines, fmt.Sprintf("  %s (Weight: %.1f, Severity: %.2f, %s)",
					rule.Name, rule.Weight, rule.Severity, enabled))
			}
		}
	}

	return strings.Join(lines, "\n")
}
