package service

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

type EnhancedAdaptiveDifficultyService struct {
	*AdaptiveDifficultyService
	historicalAnalyzer *HistoricalDataAnalyzer
	realtimeAdjuster  *RealtimeDifficultyAdjuster
	personalizationEngine *PersonalizationEngine
	mu sync.RWMutex
}

type HistoricalDataAnalyzer struct {
	userHistory map[string]*UserHistoricalData
	analyticsDB map[string]*DifficultyAnalytics
	mu sync.RWMutex
}

type UserHistoricalData struct {
	UserID           string
	VerificationHistory [][]VerificationAttempt
	SuccessPatterns  []SuccessPattern
	FailurePatterns  []FailurePattern
	PreferredTimes    map[int][]time.Time
	AvgDifficulty    float64
	TotalAttempts    int
	SuccessCount     int
	SuccessRate      float64
	AvgTimeByDifficulty map[DifficultyLevel]float64
	LastUpdated      time.Time
}

type VerificationAttempt struct {
	Timestamp        time.Time
	Difficulty       DifficultyLevel
	Success          bool
	ResponseTime     float64
	MethodUsed       string
	RiskScore        float64
}

type SuccessPattern struct {
	Difficulty      DifficultyLevel
	AvgTime         float64
	Count           int
	PreferredMethod string
	TimeOfDay       int
}

type FailurePattern struct {
	Difficulty      DifficultyLevel
	AvgTimeBeforeFail float64
	Count           int
	CommonReasons    []string
}

type DifficultyAnalytics struct {
	TotalAttempts    int
	SuccessRate      float64
	AvgTime          float64
	MedianTime       float64
	MinTime          float64
	MaxTime          float64
	TimeVariance     float64
	DifficultyDistribution map[DifficultyLevel]int
	SuccessByHour    map[int]int
	Trend            string
}

type RealtimeDifficultyAdjuster struct {
	currentAdjustments map[string]*Adjustment
	config            *RealtimeConfig
	mu                sync.RWMutex
}

type Adjustment struct {
	UserID        string
	BaseDifficulty DifficultyLevel
	Adjustment    float64
	Reason        string
	ValidUntil    time.Time
	ConsecutiveOK int
	ConsecutiveFail int
}

type RealtimeConfig struct {
	AdjustmentWindow   time.Duration
	MaxAdjustment      float64
	ConsecutiveOKThreshold int
	ConsecutiveFailThreshold int
	CooldownPeriod     time.Duration
}

type PersonalizationEngine struct {
	userProfiles map[string]*PersonalizedProfile
	globalStats  *GlobalStats
	mu           sync.RWMutex
}

type PersonalizedProfile struct {
	UserID            string
	PreferredMethod   string
	PreferredDifficulty DifficultyLevel
	OptimalHours      []int
	OptimalDays       []int
	AvgSuccessRate    float64
	AvgTime           float64
	ComfortZone       DifficultyLevel
	ChallengeLevel    DifficultyLevel
	SuccessHistory    []SuccessRecord
	LastUpdated       time.Time
}

type SuccessRecord struct {
	Timestamp      time.Time
	Difficulty     DifficultyLevel
	Method         string
	TimeSpent      float64
	UserSatisfaction float64
}

type GlobalStats struct {
	TotalUsers     int
	AvgSuccessRate float64
	MostPopularMethod string
	DifficultyDistribution map[DifficultyLevel]float64
	TimePatterns   map[int]float64
}

func NewEnhancedAdaptiveDifficultyService() *EnhancedAdaptiveDifficultyService {
	service := &EnhancedAdaptiveDifficultyService{
		AdaptiveDifficultyService: NewAdaptiveDifficultyService(),
		historicalAnalyzer: &HistoricalDataAnalyzer{
			userHistory: make(map[string]*UserHistoricalData),
			analyticsDB: make(map[string]*DifficultyAnalytics),
		},
		realtimeAdjuster: &RealtimeDifficultyAdjuster{
			currentAdjustments: make(map[string]*Adjustment),
			config: &RealtimeConfig{
				AdjustmentWindow:   10 * time.Minute,
				MaxAdjustment:      2.0,
				ConsecutiveOKThreshold: 3,
				ConsecutiveFailThreshold: 2,
				CooldownPeriod:     5 * time.Minute,
			},
		},
		personalizationEngine: &PersonalizationEngine{
			userProfiles: make(map[string]*PersonalizedProfile),
			globalStats: &GlobalStats{
				DifficultyDistribution: make(map[DifficultyLevel]float64),
				TimePatterns: make(map[int]float64),
			},
		},
	}
	
	service.personalizationEngine.globalStats.DifficultyDistribution[DifficultyEasy] = 0.2
	service.personalizationEngine.globalStats.DifficultyDistribution[DifficultyMedium] = 0.4
	service.personalizationEngine.globalStats.DifficultyDistribution[DifficultyHard] = 0.3
	service.personalizationEngine.globalStats.DifficultyDistribution[DifficultyExpert] = 0.1
	
	return service
}

func (s *EnhancedAdaptiveDifficultyService) GetEnhancedDifficulty(
	userID string,
	context *DifficultyContext,
) (DifficultyLevel, *DifficultyRecommendation) {
	
	baseDifficulty := s.AdaptiveDifficultyService.GetDifficulty(userID)
	
	historicalData := s.historicalAnalyzer.getUserHistory(userID)
	
	personalizedDifficulty := s.personalizationEngine.getPersonalizedDifficulty(
		userID,
		baseDifficulty,
		historicalData,
	)
	
	realtimeAdjustment := s.realtimeAdjuster.getAdjustment(userID)
	
	finalDifficulty := s.calculateFinalDifficulty(
		personalizedDifficulty,
		realtimeAdjustment,
		context,
	)
	
	recommendation := &DifficultyRecommendation{
		RecommendedDifficulty: finalDifficulty,
		BaseDifficulty:        baseDifficulty,
		PersonalizationBonus:  s.calculatePersonalizationBonus(userID, historicalData),
		HistoricalAdjustment:  s.calculateHistoricalAdjustment(historicalData),
		RealtimeAdjustment:   realtimeAdjustment,
		Confidence:           s.calculateRecommendationConfidence(userID),
		AlternativeMethods:   s.suggestAlternativeMethods(userID, finalDifficulty),
		Reasoning:            s.generateReasoning(finalDifficulty, baseDifficulty, historicalData),
	}
	
	return finalDifficulty, recommendation
}

func (s *EnhancedAdaptiveDifficultyService) UpdateDifficultyWithResult(
	userID string,
	difficulty DifficultyLevel,
	success bool,
	responseTime time.Duration,
	method string,
) {
	s.AdaptiveDifficultyService.UpdateProfile(userID, success, responseTime)
	
	attempt := VerificationAttempt{
		Timestamp:    time.Now(),
		Difficulty:   difficulty,
		Success:      success,
		ResponseTime: responseTime.Seconds(),
		MethodUsed:   method,
	}
	s.historicalAnalyzer.recordAttempt(userID, attempt)
	
	s.realtimeAdjuster.updateAdjustment(userID, difficulty, success, responseTime)
	
	s.personalizationEngine.updateProfile(userID, difficulty, method, responseTime)
	
	s.analyzeAndAdapt(userID)
}

func (s *EnhancedAdaptiveDifficultyService) analyzeAndAdapt(userID string) {
	history := s.historicalAnalyzer.getUserHistory(userID)
	if history == nil || len(history.VerificationHistory) < 5 {
		return
	}
	
	analytics := s.historicalAnalyzer.calculateAnalytics(userID)
	
	if analytics.SuccessRate < 70 && history.AvgDifficulty < 60 {
		s.AdaptiveDifficultyService.UpdateConfig(&DifficultyConfig{
			EasyThreshold:   15.0,
			MediumThreshold: 35.0,
			HardThreshold:   55.0,
			ExpertThreshold: 75.0,
			FailureWeight:   10.0,
			SuccessWeight:   -3.0,
			TimePenalty:     1.5,
		})
	} else if analytics.SuccessRate > 95 && history.AvgDifficulty > 60 {
		s.AdaptiveDifficultyService.UpdateConfig(&DifficultyConfig{
			EasyThreshold:   25.0,
			MediumThreshold: 45.0,
			HardThreshold:   65.0,
			ExpertThreshold: 85.0,
			FailureWeight:   20.0,
			SuccessWeight:   -8.0,
			TimePenalty:     3.0,
		})
	}
}

func (s *EnhancedAdaptiveDifficultyService) GetUserAnalytics(userID string) *UserAnalyticsReport {
	history := s.historicalAnalyzer.getUserHistory(userID)
	analytics := s.historicalAnalyzer.calculateAnalytics(userID)
	profile := s.personalizationEngine.getProfile(userID)
	
	report := &UserAnalyticsReport{
		UserID:          userID,
		TotalAttempts:   history.TotalAttempts,
		SuccessCount:    history.SuccessCount,
		SuccessRate:     analytics.SuccessRate,
		AvgTime:         analytics.AvgTime,
		MedianTime:      analytics.MedianTime,
		Trend:           analytics.Trend,
		PreferredMethod: profile.PreferredMethod,
		OptimalTimes:    profile.OptimalHours,
		AvgDifficulty:   history.AvgDifficulty,
	}
	
	return report
}

func (s *EnhancedAdaptiveDifficultyService) calculateFinalDifficulty(
	personalized DifficultyLevel,
	adjustment *Adjustment,
	context *DifficultyContext,
) DifficultyLevel {
	
	difficultyScore := s.difficultyToScore(personalized)
	
	if adjustment != nil && time.Now().Before(adjustment.ValidUntil) {
		difficultyScore += adjustment.Adjustment
	}
	
	if context != nil {
		if context.HighRiskContext {
			difficultyScore += 1.0
		}
		if context.TimeSensitive {
			difficultyScore -= 0.5
		}
		if context.UserRequestedDifficulty != "" {
			requestedDifficulty := DifficultyLevel(context.UserRequestedDifficulty)
			requestedScore := s.difficultyToScore(requestedDifficulty)
			difficultyScore = difficultyScore*0.7 + requestedScore*0.3
		}
	}
	
	difficultyScore = math.Max(0, math.Min(4, difficultyScore))
	
	return s.scoreToDifficulty(difficultyScore)
}

func (s *EnhancedAdaptiveDifficultyService) calculatePersonalizationBonus(userID string, history *UserHistoricalData) float64 {
	if history == nil || len(history.SuccessPatterns) == 0 {
		return 0
	}
	
	bonus := 0.0
	if len(history.SuccessPatterns) >= 3 {
		avgSuccessTime := 0.0
		for _, sp := range history.SuccessPatterns {
			avgSuccessTime += sp.AvgTime
		}
		avgSuccessTime /= float64(len(history.SuccessPatterns))
		
		if avgSuccessTime < 3.0 {
			bonus += 0.5
		} else if avgSuccessTime < 5.0 {
			bonus += 0.3
		}
	}
	
	if history.SuccessCount > history.TotalAttempts*90/100 {
		bonus += 0.3
	}
	
	return bonus
}

func (s *EnhancedAdaptiveDifficultyService) calculateHistoricalAdjustment(history *UserHistoricalData) float64 {
	if history == nil {
		return 0
	}
	
	if len(history.FailurePatterns) == 0 {
		return -0.5
	}
	
	maxFailureDifficulty := float64(0)
	for _, fp := range history.FailurePatterns {
		difficultyScore := s.difficultyToScore(fp.Difficulty)
		if difficultyScore > maxFailureDifficulty {
			maxFailureDifficulty = difficultyScore
		}
	}
	
	return -maxFailureDifficulty * 0.1
}

func (s *EnhancedAdaptiveDifficultyService) calculateRecommendationConfidence(userID string) float64 {
	history := s.historicalAnalyzer.getUserHistory(userID)
	if history == nil {
		return 0.3
	}
	
	confidence := math.Min(1.0, float64(history.TotalAttempts)/20.0)
	confidence *= history.SuccessRate / 100.0
	
	return confidence
}

func (s *EnhancedAdaptiveDifficultyService) suggestAlternativeMethods(userID string, difficulty DifficultyLevel) []string {
	methods := []string{}
	
	switch difficulty {
	case DifficultyEasy:
		methods = append(methods, "slider_simple", "click_simple", "seamless")
	case DifficultyMedium:
		methods = append(methods, "slider", "click", "3d_rotate")
	case DifficultyHard:
		methods = append(methods, "3d_click", "lianliankan", "voice")
	case DifficultyExpert:
		methods = append(methods, "3d_complete", "multi_step", "biometrics")
	}
	
	profile := s.personalizationEngine.getProfile(userID)
	if profile != nil && profile.PreferredMethod != "" {
		for i, m := range methods {
			if m == profile.PreferredMethod {
				methods = append([]string{m}, append(methods[:i], methods[i+1:]...)...)
				break
			}
		}
	}
	
	return methods
}

func (s *EnhancedAdaptiveDifficultyService) generateReasoning(
	final, base DifficultyLevel,
	history *UserHistoricalData,
) string {
	
	reasoning := fmt.Sprintf("基于用户历史表现，推荐难度从 %s 调整为 %s。", base, final)
	
	if history != nil && history.TotalAttempts > 0 {
		reasoning += fmt.Sprintf(" 总验证次数: %d, 成功率: %.1f%%", 
			history.TotalAttempts, history.SuccessRate)
	}
	
	if final == DifficultyEasy {
		reasoning += " 考虑到用户表现良好，降低难度以提升用户体验。"
	} else if final == DifficultyHard || final == DifficultyExpert {
		reasoning += " 检测到异常行为，提升难度以增强安全性。"
	}
	
	return reasoning
}

func (s *EnhancedAdaptiveDifficultyService) difficultyToScore(d DifficultyLevel) float64 {
	switch d {
	case DifficultyEasy:
		return 0
	case DifficultyMedium:
		return 1
	case DifficultyHard:
		return 2
	case DifficultyExpert:
		return 3
	default:
		return 1
	}
}

func (s *EnhancedAdaptiveDifficultyService) scoreToDifficulty(score float64) DifficultyLevel {
	if score < 0.5 {
		return DifficultyEasy
	} else if score < 1.5 {
		return DifficultyMedium
	} else if score < 2.5 {
		return DifficultyHard
	} else {
		return DifficultyExpert
	}
}

func (hda *HistoricalDataAnalyzer) getUserHistory(userID string) *UserHistoricalData {
	hda.mu.RLock()
	defer hda.mu.RUnlock()
	return hda.userHistory[userID]
}

func (hda *HistoricalDataAnalyzer) recordAttempt(userID string, attempt VerificationAttempt) {
	hda.mu.Lock()
	defer hda.mu.Unlock()
	
	history, exists := hda.userHistory[userID]
	if !exists {
		history = &UserHistoricalData{
			UserID:               userID,
			VerificationHistory:  make([][]VerificationAttempt, 0),
			PreferredTimes:        make(map[int][]time.Time),
			AvgTimeByDifficulty:  make(map[DifficultyLevel]float64),
		}
		hda.userHistory[userID] = history
	}
	
	history.VerificationHistory = append(history.VerificationHistory, []VerificationAttempt{attempt})
	history.TotalAttempts++
	if attempt.Success {
		history.SuccessCount++
	}
	history.LastUpdated = time.Now()
	
	hour := attempt.Timestamp.Hour()
	history.PreferredTimes[hour] = append(history.PreferredTimes[hour], attempt.Timestamp)
	
	pattern := SuccessPattern{
		Difficulty: attempt.Difficulty,
		AvgTime:    attempt.ResponseTime,
		Count:      1,
		TimeOfDay:  hour,
	}
	history.SuccessPatterns = append(history.SuccessPatterns, pattern)
	
	limitHistory(history)
}

func (hda *HistoricalDataAnalyzer) calculateAnalytics(userID string) *DifficultyAnalytics {
	hda.mu.RLock()
	defer hda.mu.RUnlock()
	
	history := hda.userHistory[userID]
	if history == nil {
		return &DifficultyAnalytics{}
	}
	
	analytics := &DifficultyAnalytics{
		DifficultyDistribution: make(map[DifficultyLevel]int),
		SuccessByHour:        make(map[int]int),
	}
	
	if len(history.VerificationHistory) == 0 {
		return analytics
	}
	
	analytics.TotalAttempts = history.TotalAttempts
	analytics.SuccessRate = float64(history.SuccessCount) / float64(history.TotalAttempts) * 100
	
	allTimes := make([]float64, 0)
	for _, dayHistory := range history.VerificationHistory {
		for _, attempt := range dayHistory {
			allTimes = append(allTimes, attempt.ResponseTime)
			analytics.DifficultyDistribution[attempt.Difficulty]++
			
			if attempt.Success {
				analytics.SuccessByHour[attempt.Timestamp.Hour()]++
			}
		}
	}
	
	if len(allTimes) > 0 {
		analytics.AvgTime = meanFloat(allTimes)
		sort.Float64s(allTimes)
		analytics.MedianTime = allTimes[len(allTimes)/2]
		analytics.MinTime = allTimes[0]
		analytics.MaxTime = allTimes[len(allTimes)-1]
		analytics.TimeVariance = varianceFloat(allTimes)
	}
	
	analytics.Trend = calculateTrend(history.VerificationHistory)
	
	return analytics
}

func (rda *RealtimeDifficultyAdjuster) getAdjustment(userID string) *Adjustment {
	rda.mu.RLock()
	defer rda.mu.RUnlock()
	return rda.currentAdjustments[userID]
}

func (rda *RealtimeDifficultyAdjuster) updateAdjustment(
	userID string,
	difficulty DifficultyLevel,
	success bool,
	responseTime time.Duration,
) {
	rda.mu.Lock()
	defer rda.mu.Unlock()
	
	adj, exists := rda.currentAdjustments[userID]
	if !exists {
		adj = &Adjustment{
			UserID:         userID,
			BaseDifficulty: difficulty,
			Adjustment:    0,
			ValidUntil:    time.Now().Add(rda.config.AdjustmentWindow),
		}
		rda.currentAdjustments[userID] = adj
	}
	
	if success {
		adj.ConsecutiveOK++
		adj.ConsecutiveFail = 0
		
		if adj.ConsecutiveOK >= rda.config.ConsecutiveOKThreshold {
			adj.Adjustment = math.Max(-rda.config.MaxAdjustment, adj.Adjustment-0.5)
			adj.Reason = "连续成功，降低难度"
			adj.ValidUntil = time.Now().Add(rda.config.AdjustmentWindow)
		}
	} else {
		adj.ConsecutiveFail++
		adj.ConsecutiveOK = 0
		
		if adj.ConsecutiveFail >= rda.config.ConsecutiveFailThreshold {
			adj.Adjustment = math.Min(rda.config.MaxAdjustment, adj.Adjustment+1.0)
			adj.Reason = "连续失败，提升难度"
			adj.ValidUntil = time.Now().Add(rda.config.AdjustmentWindow)
		}
	}
	
	if responseTime.Seconds() > 15 && !success {
		adj.Adjustment = math.Max(-rda.config.MaxAdjustment, adj.Adjustment-0.3)
		adj.Reason = "超时未完成，降低难度"
	}
}

func (pe *PersonalizationEngine) getPersonalizedDifficulty(
	userID string,
	base DifficultyLevel,
	history *UserHistoricalData,
) DifficultyLevel {
	
	profile := pe.getProfile(userID)
	if profile != nil && profile.ComfortZone != "" {
		comfortScore := pe.difficultyToScore(profile.ComfortZone)
		baseScore := pe.difficultyToScore(base)
		
		combinedScore := baseScore*0.6 + comfortScore*0.4
		return pe.scoreToDifficulty(combinedScore)
	}
	
	return base
}

func (pe *PersonalizationEngine) getProfile(userID string) *PersonalizedProfile {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.userProfiles[userID]
}

func (pe *PersonalizationEngine) updateProfile(
	userID string,
	difficulty DifficultyLevel,
	method string,
	responseTime time.Duration,
) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	
	profile, exists := pe.userProfiles[userID]
	if !exists {
		profile = &PersonalizedProfile{
			UserID:           userID,
			AvgSuccessRate:   80.0,
			AvgTime:          5.0,
			SuccessHistory:   make([]SuccessRecord, 0),
		}
		pe.userProfiles[userID] = profile
	}
	
	record := SuccessRecord{
		Timestamp:      time.Now(),
		Difficulty:     difficulty,
		Method:         method,
		TimeSpent:      responseTime.Seconds(),
		UserSatisfaction: 5.0,
	}
	profile.SuccessHistory = append(profile.SuccessHistory, record)
	
	if len(profile.SuccessHistory) > 100 {
		profile.SuccessHistory = profile.SuccessHistory[len(profile.SuccessHistory)-100:]
	}
	
	profile.AvgTime = profile.AvgTime*0.9 + responseTime.Seconds()*0.1
	profile.LastUpdated = time.Now()
	
	if method != "" {
		profile.PreferredMethod = method
	}
}

func (pe *PersonalizationEngine) difficultyToScore(d DifficultyLevel) float64 {
	switch d {
	case DifficultyEasy:
		return 0
	case DifficultyMedium:
		return 1
	case DifficultyHard:
		return 2
	case DifficultyExpert:
		return 3
	default:
		return 1
	}
}

func (pe *PersonalizationEngine) scoreToDifficulty(score float64) DifficultyLevel {
	if score < 0.5 {
		return DifficultyEasy
	} else if score < 1.5 {
		return DifficultyMedium
	} else if score < 2.5 {
		return DifficultyHard
	} else {
		return DifficultyExpert
	}
}

type DifficultyContext struct {
	HighRiskContext       bool
	TimeSensitive         bool
	UserRequestedDifficulty string
}

type DifficultyRecommendation struct {
	RecommendedDifficulty DifficultyLevel
	BaseDifficulty        DifficultyLevel
	PersonalizationBonus  float64
	HistoricalAdjustment  float64
	RealtimeAdjustment    *Adjustment
	Confidence            float64
	AlternativeMethods    []string
	Reasoning             string
}

type UserAnalyticsReport struct {
	UserID          string
	TotalAttempts   int
	SuccessCount    int
	SuccessRate     float64
	AvgTime         float64
	MedianTime      float64
	Trend           string
	PreferredMethod string
	OptimalTimes    []int
	AvgDifficulty   float64
}

func limitHistory(history *UserHistoricalData) {
	maxDays := 30
	if len(history.VerificationHistory) > maxDays {
		history.VerificationHistory = history.VerificationHistory[len(history.VerificationHistory)-maxDays:]
	}
}

func calculateTrend(history [][]VerificationAttempt) string {
	if len(history) < 7 {
		return "stable"
	}
	
	recentSuccess := 0
	recentTotal := 0
	oldSuccess := 0
	oldTotal := 0
	
	for i, day := range history {
		for _, attempt := range day {
			if i >= len(history)-3 {
				recentTotal++
				if attempt.Success {
					recentSuccess++
				}
			}
			if i < len(history)-7 && i >= len(history)-7 {
				oldTotal++
				if attempt.Success {
					oldSuccess++
				}
			}
		}
	}
	
	if recentTotal == 0 || oldTotal == 0 {
		return "stable"
	}
	
	recentRate := float64(recentSuccess) / float64(recentTotal)
	oldRate := float64(oldSuccess) / float64(oldTotal)
	
	diff := recentRate - oldRate
	if diff > 0.1 {
		return "improving"
	} else if diff < -0.1 {
		return "declining"
	}
	return "stable"
}

func meanFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func varianceFloat(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := meanFloat(values)
	sum := 0.0
	for _, v := range values {
		sum += math.Pow(v-mean, 2)
	}
	return sum / float64(len(values))
}
