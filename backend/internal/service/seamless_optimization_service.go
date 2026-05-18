package service

import (
	"math"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

type SeamlessOptimizationService struct {
	continuousLearner *ContinuousLearner
	deviceFingerprint *DeviceFingerprintOptimizer
	disturbanceReducer *DisturbanceReducer
	passRateOptimizer *PassRateOptimizer
	mu sync.RWMutex
}

type ContinuousLearner struct {
	userPatterns map[string]*UserBehaviorPattern
	config      *LearningConfig
	mu          sync.RWMutex
}

type LearningConfig struct {
	LearningRate         float64
	PatternDecay         float64
	MinSamplesForLearning int
	MaxPatternAge        time.Duration
}

type UserBehaviorPattern struct {
	UserID          string
	PreferredTimes  []time.Time
	PreferredDays   []int
	AvgSuccessRate  float64
	TotalAttempts   int
	SuccessfulAttempts int
	AvgResponseTime float64
	LastSeen        time.Time
	Confidence      float64
	DeviceHistory   map[string]int
	LocationHistory []string
}

type DeviceFingerprintOptimizer struct {
	trustScores map[string]*DeviceTrustScore
	weightConfig *FingerprintWeightConfig
	mu          sync.RWMutex
}

type DeviceTrustScore struct {
	Fingerprint   string
	TrustLevel    float64
	TotalUses     int
	SuccessUses   int
	LastUsed      time.Time
	KnownPatterns []string
	AnomalyCount  int
	AvgRiskScore  float64
}

type FingerprintWeightConfig struct {
	CanvasWeight     float64
	WebGLWeight      float64
	FontWeight       float64
	TimezoneWeight   float64
	LanguageWeight   float64
	ScreenWeight     float64
}

type DisturbanceReducer struct {
	userPreferences map[string]*UserPreference
	globalConfig    *DisturbanceConfig
	mu              sync.RWMutex
}

type UserPreference struct {
	UserID          string
	PreferredMethod string
	MinDisturbLevel int
	SkipHours       []int
	SkipDays        []int
	AlwaysVerifyNew bool
	TrustNewDevice  bool
}

type DisturbanceConfig struct {
	MaxChallengesPerDay      int
	MaxChallengesPerWeek     int
	QuietHoursStart          int
	QuietHoursEnd           int
	SkipForLowRisk          bool
	SkipThreshold           float64
	ProgressiveChallenge    bool
}

type PassRateOptimizer struct {
	historicalData map[string]*PassRateData
	thresholds     *PassRateThresholds
	mu             sync.RWMutex
}

type PassRateData struct {
	UserID       string
	DailyStats   map[string]int
	WeeklyStats  map[string]int
	AvgPassRate  float64
	FailPatterns []string
	SuccessPatterns []string
	LastUpdated  time.Time
}

type PassRateThresholds struct {
	LowRiskThreshold    float64
	MediumRiskThreshold float64
	HighRiskThreshold   float64
	MinPassRate         float64
	TargetPassRate      float64
}

func NewSeamlessOptimizationService() *SeamlessOptimizationService {
	return &SeamlessOptimizationService{
		continuousLearner: &ContinuousLearner{
			userPatterns: make(map[string]*UserBehaviorPattern),
			config: &LearningConfig{
				LearningRate:          0.1,
				PatternDecay:          0.05,
				MinSamplesForLearning: 10,
				MaxPatternAge:         30 * 24 * time.Hour,
			},
		},
		deviceFingerprint: &DeviceFingerprintOptimizer{
			trustScores: make(map[string]*DeviceTrustScore),
			weightConfig: &FingerprintWeightConfig{
				CanvasWeight:   0.25,
				WebGLWeight:    0.20,
				FontWeight:     0.15,
				TimezoneWeight: 0.15,
				LanguageWeight: 0.10,
				ScreenWeight:   0.15,
			},
		},
		disturbanceReducer: &DisturbanceReducer{
			userPreferences: make(map[string]*UserPreference),
			globalConfig: &DisturbanceConfig{
				MaxChallengesPerDay:   5,
				MaxChallengesPerWeek:  20,
				QuietHoursStart:        22,
				QuietHoursEnd:          8,
				SkipForLowRisk:         true,
				SkipThreshold:          20.0,
				ProgressiveChallenge:   true,
			},
		},
		passRateOptimizer: &PassRateOptimizer{
			historicalData: make(map[string]*PassRateData),
			thresholds: &PassRateThresholds{
				LowRiskThreshold:    20.0,
				MediumRiskThreshold: 50.0,
				HighRiskThreshold:   80.0,
				MinPassRate:         85.0,
				TargetPassRate:      90.0,
			},
		},
	}
}

func (s *SeamlessOptimizationService) OptimizeSeamlessVerification(
	userID string,
	deviceFingerprint string,
	behaviorData []models.BehaviorData,
	environmentData map[string]interface{},
	previousRiskScore float64,
) (*SeamlessOptimizationResult, error) {
	
	result := &SeamlessOptimizationResult{
		FinalRiskScore: previousRiskScore,
		ShouldChallenge: true,
		OptimizationApplied: []string{},
	}

	userPattern := s.continuousLearner.getUserPattern(userID)
	deviceTrust := s.deviceFingerprint.getDeviceTrust(deviceFingerprint)
	userPref := s.disturbanceReducer.getUserPreference(userID)
	passRateData := s.passRateOptimizer.getPassRateData(userID)

	if userPattern != nil && userPattern.Confidence > 0.7 {
		timeBonus := s.calculateTimeBasedBonus(userPattern)
		result.FinalRiskScore = math.Max(0, result.FinalRiskScore-timeBonus)
		result.OptimizationApplied = append(result.OptimizationApplied, "time_based_optimization")
	}

	if deviceTrust != nil && deviceTrust.TrustLevel > 0.8 {
		deviceBonus := deviceTrust.TrustLevel * 20
		result.FinalRiskScore = math.Max(0, result.FinalRiskScore-deviceBonus)
		result.OptimizationApplied = append(result.OptimizationApplied, "device_trust_optimization")
	}

	if userPref != nil {
		disturbanceReduction := s.calculateDisturbanceReduction(userPref, result.FinalRiskScore)
		if disturbanceReduction > 0 {
			result.FinalRiskScore = math.Max(0, result.FinalRiskScore-disturbanceReduction)
			result.OptimizationApplied = append(result.OptimizationApplied, "disturbance_reduction")
		}
	}

	if passRateData != nil && passRateData.AvgPassRate < s.passRateOptimizer.thresholds.MinPassRate {
		adjustment := (s.passRateOptimizer.thresholds.TargetPassRate - passRateData.AvgPassRate) * 0.5
		result.FinalRiskScore = math.Max(0, result.FinalRiskScore-adjustment)
		result.OptimizationApplied = append(result.OptimizationApplied, "pass_rate_adjustment")
	}

	if result.FinalRiskScore < s.disturbanceReducer.globalConfig.SkipThreshold {
		result.ShouldChallenge = false
		result.SkipReason = "低风险用户，应用无感验证跳过"
		result.OptimizationApplied = append(result.OptimizationApplied, "seamless_skip")
	}

	if s.isInQuietHours() {
		result.ShouldChallenge = false
		result.SkipReason = "安静时段，跳过验证"
		result.OptimizationApplied = append(result.OptimizationApplied, "quiet_hours_skip")
	}

	return result, nil
}

func (s *SeamlessOptimizationService) UpdateUserPattern(
	userID string,
	verificationSuccess bool,
	responseTime time.Duration,
	deviceFingerprint string,
	location string,
) {
	pattern := s.continuousLearner.getOrCreateUserPattern(userID)
	
	pattern.TotalAttempts++
	if verificationSuccess {
		pattern.SuccessfulAttempts++
	}
	
	pattern.AvgResponseTime = pattern.AvgResponseTime*0.9 + responseTime.Seconds()*0.1
	pattern.AvgSuccessRate = float64(pattern.SuccessfulAttempts) / float64(pattern.TotalAttempts) * 100
	pattern.LastSeen = time.Now()
	
	if pattern.DeviceHistory == nil {
		pattern.DeviceHistory = make(map[string]int)
	}
	pattern.DeviceHistory[deviceFingerprint]++
	
	if location != "" {
		pattern.LocationHistory = append(pattern.LocationHistory, location)
		if len(pattern.LocationHistory) > 10 {
			pattern.LocationHistory = pattern.LocationHistory[len(pattern.LocationHistory)-10:]
		}
	}
	
	pattern.Confidence = math.Min(1.0, float64(pattern.TotalAttempts)/100.0)
	
	s.continuousLearner.updateUserPattern(userID, pattern)
}

func (s *SeamlessOptimizationService) UpdateDeviceTrust(
	fingerprint string,
	riskScore float64,
	isSuccessful bool,
	newPatterns []string,
) {
	device := s.deviceFingerprint.getOrCreateDeviceTrust(fingerprint)
	
	device.TotalUses++
	if isSuccessful {
		device.SuccessUses++
	}
	device.LastUsed = time.Now()
	device.AvgRiskScore = device.AvgRiskScore*0.9 + riskScore*0.1
	
	if len(newPatterns) > 0 {
		device.KnownPatterns = append(device.KnownPatterns, newPatterns...)
		if len(device.KnownPatterns) > 50 {
			device.KnownPatterns = device.KnownPatterns[len(device.KnownPatterns)-50:]
		}
	}
	
	if riskScore > 70 {
		device.AnomalyCount++
	}
	
	device.TrustLevel = float64(device.SuccessUses) / float64(device.TotalUses)
	if device.AnomalyCount > 5 {
		device.TrustLevel *= 0.5
	}
	
	s.deviceFingerprint.updateDeviceTrust(fingerprint, device)
}

func (s *SeamlessOptimizationService) calculateTimeBasedBonus(pattern *UserBehaviorPattern) float64 {
	now := time.Now()
	currentHour := now.Hour()
	currentDay := int(now.Weekday())
	
	hourMatch := false
	for _, t := range pattern.PreferredTimes {
		if abs(t.Hour()-currentHour) <= 2 {
			hourMatch = true
			break
		}
	}
	
	dayMatch := false
	for _, d := range pattern.PreferredDays {
		if d == currentDay {
			dayMatch = true
			break
		}
	}
	
	bonus := 0.0
	if hourMatch {
		bonus += 10 * pattern.Confidence
	}
	if dayMatch {
		bonus += 5 * pattern.Confidence
	}
	
	return bonus
}

func (s *SeamlessOptimizationService) calculateDisturbanceReduction(pref *UserPreference, riskScore float64) float64 {
	if pref.MinDisturbLevel >= 3 && riskScore < 30 {
		return 15
	}
	if pref.MinDisturbLevel >= 2 && riskScore < 25 {
		return 10
	}
	if pref.MinDisturbLevel >= 1 && riskScore < 20 {
		return 5
	}
	return 0
}

func (s *SeamlessOptimizationService) isInQuietHours() bool {
	hour := time.Now().Hour()
	start := s.disturbanceReducer.globalConfig.QuietHoursStart
	end := s.disturbanceReducer.globalConfig.QuietHoursEnd
	
	if start > end {
		return hour >= start || hour < end
	}
	return hour >= start && hour < end
}

func (cl *ContinuousLearner) getUserPattern(userID string) *UserBehaviorPattern {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.userPatterns[userID]
}

func (cl *ContinuousLearner) getOrCreateUserPattern(userID string) *UserBehaviorPattern {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	
	if pattern, exists := cl.userPatterns[userID]; exists {
		return pattern
	}
	
	pattern := &UserBehaviorPattern{
		UserID:           userID,
		AvgSuccessRate:   80.0,
		TotalAttempts:    0,
		SuccessfulAttempts: 0,
		AvgResponseTime:  5.0,
		Confidence:       0.1,
		DeviceHistory:    make(map[string]int),
		LocationHistory:  []string{},
	}
	cl.userPatterns[userID] = pattern
	return pattern
}

func (cl *ContinuousLearner) updateUserPattern(userID string, pattern *UserBehaviorPattern) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.userPatterns[userID] = pattern
}

func (dfo *DeviceFingerprintOptimizer) getDeviceTrust(fingerprint string) *DeviceTrustScore {
	dfo.mu.RLock()
	defer dfo.mu.RUnlock()
	return dfo.trustScores[fingerprint]
}

func (dfo *DeviceFingerprintOptimizer) getOrCreateDeviceTrust(fingerprint string) *DeviceTrustScore {
	dfo.mu.Lock()
	defer dfo.mu.Unlock()
	
	if trust, exists := dfo.trustScores[fingerprint]; exists {
		return trust
	}
	
	trust := &DeviceTrustScore{
		Fingerprint:   fingerprint,
		TrustLevel:    0.5,
		TotalUses:     0,
		SuccessUses:   0,
		KnownPatterns: []string{},
	}
	dfo.trustScores[fingerprint] = trust
	return trust
}

func (dfo *DeviceFingerprintOptimizer) updateDeviceTrust(fingerprint string, trust *DeviceTrustScore) {
	dfo.mu.Lock()
	defer dfo.mu.Unlock()
	dfo.trustScores[fingerprint] = trust
}

func (dr *DisturbanceReducer) getUserPreference(userID string) *UserPreference {
	dr.mu.RLock()
	defer dr.mu.RUnlock()
	return dr.userPreferences[userID]
}

func (dr *DisturbanceReducer) SetUserPreference(userID string, pref *UserPreference) {
	dr.mu.Lock()
	defer dr.mu.Unlock()
	dr.userPreferences[userID] = pref
}

func (pro *PassRateOptimizer) getPassRateData(userID string) *PassRateData {
	pro.mu.RLock()
	defer pro.mu.RUnlock()
	return pro.historicalData[userID]
}

func (pro *PassRateOptimizer) UpdatePassRate(userID string, success bool) {
	pro.mu.Lock()
	defer pro.mu.Unlock()
	
	data, exists := pro.historicalData[userID]
	if !exists {
		data = &PassRateData{
			UserID:      userID,
			DailyStats:  make(map[string]int),
			WeeklyStats: make(map[string]int),
		}
		pro.historicalData[userID] = data
	}
	
	today := time.Now().Format("2006-01-02")
	data.DailyStats[today]++
	if success {
		data.SuccessPatterns = append(data.SuccessPatterns, today)
	} else {
		data.FailPatterns = append(data.FailPatterns, today)
	}
	
	totalSuccess := len(data.SuccessPatterns)
	total := len(data.SuccessPatterns) + len(data.FailPatterns)
	if total > 0 {
		data.AvgPassRate = float64(totalSuccess) / float64(total) * 100
	}
	
	data.LastUpdated = time.Now()
}

type SeamlessOptimizationResult struct {
	FinalRiskScore      float64
	ShouldChallenge     bool
	SkipReason          string
	OptimizationApplied []string
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
