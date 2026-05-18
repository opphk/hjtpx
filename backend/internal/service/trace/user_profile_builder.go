package trace

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type UserProfileBuilder struct {
	profiles       map[string]*UserProfile
	mu             sync.RWMutex
	profileStorage *ProfileStorage
}

type UserProfile struct {
	UserID               string                 `json:"user_id"`
	CreatedAt            time.Time              `json:"created_at"`
	LastUpdatedAt        time.Time              `json:"last_updated_at"`
	BehaviorPatterns     *BehaviorPatterns      `json:"behavior_patterns"`
	DeviceFingerprints   []string               `json:"device_fingerprints"`
	LocationHistory      []LocationRecord       `json:"location_history"`
	SessionHistory       []SessionRecord        `json:"session_history"`
	RiskScoreHistory     []RiskScoreRecord      `json:"risk_score_history"`
	TypicalBehaviorTimes []TimeSlot            `json:"typical_behavior_times"`
	ProfileConfidence    float64               `json:"profile_confidence"`
	AnomalyCount         int                   `json:"anomaly_count"`
}

type BehaviorPatterns struct {
	TypicalSessionDuration    float64              `json:"typical_session_duration"`
	SessionDurationVariance   float64              `json:"session_duration_variance"`
	TypicalActionsPerSession  int                  `json:"typical_actions_per_session"`
	ActionIntervalStats       *IntervalStatistics  `json:"action_interval_stats"`
	TimeOfDayDistribution     map[string]float64   `json:"time_of_day_distribution"`
	DayOfWeekDistribution     map[string]float64   `json:"day_of_week_distribution"`
	CommonNavigationPatterns  []string             `json:"common_navigation_patterns"`
	TypicalClickPatterns      *ClickPatternStats   `json:"typical_click_patterns"`
	TypicalScrollPatterns     *ScrollPatternStats  `json:"typical_scroll_patterns"`
	TypingSpeedStats          *TypingSpeedStats    `json:"typing_speed_stats"`
}

type IntervalStatistics struct {
	Average       float64 `json:"average"`
	Variance      float64 `json:"variance"`
	Min           float64 `json:"min"`
	Max           float64 `json:"max"`
}

type ClickPatternStats struct {
	AverageInterval     float64 `json:"average_interval"`
	IntervalVariance    float64 `json:"interval_variance"`
	ClickCountPerMinute float64 `json:"click_count_per_minute"`
	TypicalClickAreas   []Area  `json:"typical_click_areas"`
}

type ScrollPatternStats struct {
	AverageScrollSpeed     float64 `json:"average_scroll_speed"`
	ScrollSpeedVariance    float64 `json:"scroll_speed_variance"`
	TypicalScrollDepth     float64 `json:"typical_scroll_depth"`
}

type TypingSpeedStats struct {
	AverageWPM         float64 `json:"average_wpm"`
	WPMVariance        float64 `json:"wpm_variance"`
	AverageHoldTime    float64 `json:"average_hold_time"`
	HoldTimeVariance   float64 `json:"hold_time_variance"`
}

type Area struct {
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Radius float64 `json:"radius"`
}

type LocationRecord struct {
	IP          string    `json:"ip"`
	Country     string    `json:"country"`
	Region      string    `json:"region"`
	City        string    `json:"city"`
	Timestamp   time.Time `json:"timestamp"`
	Confidence  float64   `json:"confidence"`
}

type SessionRecord struct {
	SessionID     string    `json:"session_id"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	DurationMs    int64     `json:"duration_ms"`
	ActionCount   int       `json:"action_count"`
	SuccessCount  int       `json:"success_count"`
	FailureCount  int       `json:"failure_count"`
	DeviceID      string    `json:"device_id"`
	IP            string    `json:"ip"`
	UserAgent     string    `json:"user_agent"`
}

type RiskScoreRecord struct {
	Score      float64   `json:"score"`
	Timestamp  time.Time `json:"timestamp"`
	Reason     string    `json:"reason"`
}

type TimeSlot struct {
	Hour    int     `json:"hour"`
	Day     int     `json:"day"`
	Weight  float64 `json:"weight"`
}

type ProfileStorage struct {
	data      map[string]*UserProfile
	mu        sync.RWMutex
	lastClean time.Time
}

func NewUserProfileBuilder() *UserProfileBuilder {
	return &UserProfileBuilder{
		profiles:       make(map[string]*UserProfile),
		profileStorage: NewProfileStorage(),
	}
}

func NewProfileStorage() *ProfileStorage {
	return &ProfileStorage{
		data:      make(map[string]*UserProfile),
		lastClean: time.Now(),
	}
}

func (b *UserProfileBuilder) GetOrCreateProfile(userID string) *UserProfile {
	b.mu.RLock()
	profile, exists := b.profiles[userID]
	b.mu.RUnlock()

	if exists {
		return profile
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if profile, exists := b.profiles[userID]; exists {
		return profile
	}

	profile = &UserProfile{
		UserID:               userID,
		CreatedAt:            time.Now(),
		LastUpdatedAt:        time.Now(),
		BehaviorPatterns:     NewBehaviorPatterns(),
		DeviceFingerprints:   make([]string, 0),
		LocationHistory:      make([]LocationRecord, 0),
		SessionHistory:       make([]SessionRecord, 0),
		RiskScoreHistory:     make([]RiskScoreRecord, 0),
		TypicalBehaviorTimes: make([]TimeSlot, 0),
		ProfileConfidence:    0.0,
		AnomalyCount:         0,
	}

	b.profiles[userID] = profile
	b.profileStorage.StoreProfile(profile)

	return profile
}

func NewBehaviorPatterns() *BehaviorPatterns {
	return &BehaviorPatterns{
		TimeOfDayDistribution:    make(map[string]float64),
		DayOfWeekDistribution:    make(map[string]float64),
		CommonNavigationPatterns: make([]string, 0),
		TypicalClickPatterns:     &ClickPatternStats{
			TypicalClickAreas: make([]Area, 0),
		},
		TypicalScrollPatterns: &ScrollPatternStats{},
		TypingSpeedStats:      &TypingSpeedStats{},
		ActionIntervalStats:   &IntervalStatistics{},
	}
}

func (b *UserProfileBuilder) UpdateProfileWithTrace(userID string, traceData *model.TraceData) error {
	if traceData == nil {
		return errors.New("trace data is nil")
	}

	profile := b.GetOrCreateProfile(userID)

	b.mu.Lock()
	defer b.mu.Unlock()

	profile.LastUpdatedAt = time.Now()

	b.updateBehaviorPatterns(profile, traceData)
	b.updateSessionHistory(profile, traceData)
	b.updateRiskScoreHistory(profile, traceData)
	b.updateProfileConfidence(profile)

	b.profileStorage.StoreProfile(profile)

	return nil
}

func (b *UserProfileBuilder) updateBehaviorPatterns(profile *UserProfile, traceData *model.TraceData) {
	if len(traceData.Points) < 2 {
		return
	}

	patterns := profile.BehaviorPatterns

	duration := float64(traceData.Points[len(traceData.Points)-1].Timestamp - traceData.Points[0].Timestamp) / 1000.0
	if duration > 0 {
		if patterns.TypicalSessionDuration == 0 {
			patterns.TypicalSessionDuration = duration
		} else {
			patterns.TypicalSessionDuration = (patterns.TypicalSessionDuration*0.9 + duration*0.1)
		}
	}

	patterns.TypicalActionsPerSession = (patterns.TypicalActionsPerSession*9 + len(traceData.Points)) / 10

	now := time.Now()
	hourKey := fmt.Sprintf("%02d", now.Hour())
	patterns.TimeOfDayDistribution[hourKey]++

	dayKey := now.Weekday().String()
	patterns.DayOfWeekDistribution[dayKey]++

	patterns.ActionIntervalStats = b.calculateIntervalStats(traceData)
	patterns.TypicalScrollPatterns = b.calculateScrollPatterns(traceData)
}

func (b *UserProfileBuilder) calculateIntervalStats(traceData *model.TraceData) *IntervalStatistics {
	if len(traceData.Points) < 2 {
		return &IntervalStatistics{}
	}

	intervals := []float64{}
	for i := 1; i < len(traceData.Points); i++ {
		intervals = append(intervals, float64(traceData.Points[i].Timestamp-traceData.Points[i-1].Timestamp))
	}

	if len(intervals) == 0 {
		return &IntervalStatistics{}
	}

	stats := &IntervalStatistics{
		Min: intervals[0],
		Max: intervals[0],
	}

	sum := 0.0
	for _, interval := range intervals {
		sum += interval
		if interval < stats.Min {
			stats.Min = interval
		}
		if interval > stats.Max {
			stats.Max = interval
		}
	}

	stats.Average = sum / float64(len(intervals))

	sumVariance := 0.0
	for _, interval := range intervals {
		sumVariance += math.Pow(interval-stats.Average, 2)
	}
	stats.Variance = sumVariance / float64(len(intervals))

	return stats
}

func (b *UserProfileBuilder) calculateScrollPatterns(traceData *model.TraceData) *ScrollPatternStats {
	if len(traceData.Points) < 2 {
		return &ScrollPatternStats{}
	}

	speeds := []float64{}
	for i := 1; i < len(traceData.Points); i++ {
		dx := float64(traceData.Points[i].X - traceData.Points[i-1].X)
		dy := float64(traceData.Points[i].Y - traceData.Points[i-1].Y)
		dt := float64(traceData.Points[i].Timestamp - traceData.Points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, math.Sqrt(dx*dx+dy*dy)/dt)
		}
	}

	if len(speeds) == 0 {
		return &ScrollPatternStats{}
	}

	stats := &ScrollPatternStats{}
	sum := 0.0
	for _, speed := range speeds {
		sum += speed
	}
	stats.AverageScrollSpeed = sum / float64(len(speeds))

	sumVariance := 0.0
	for _, speed := range speeds {
		sumVariance += math.Pow(speed-stats.AverageScrollSpeed, 2)
	}
	stats.ScrollSpeedVariance = sumVariance / float64(len(speeds))

	minY, maxY := traceData.Points[0].Y, traceData.Points[0].Y
	for _, point := range traceData.Points {
		if point.Y < minY {
			minY = point.Y
		}
		if point.Y > maxY {
			maxY = point.Y
		}
	}
	stats.TypicalScrollDepth = float64(maxY - minY)

	return stats
}

func (b *UserProfileBuilder) updateSessionHistory(profile *UserProfile, traceData *model.TraceData) {
	if len(traceData.Points) < 2 {
		return
	}

	session := SessionRecord{
		SessionID:    fmt.Sprintf("%s:%d", profile.UserID, time.Now().UnixNano()),
		StartTime:    time.Unix(0, traceData.Points[0].Timestamp*1e6),
		EndTime:      time.Unix(0, traceData.Points[len(traceData.Points)-1].Timestamp*1e6),
		DurationMs:   traceData.Points[len(traceData.Points)-1].Timestamp - traceData.Points[0].Timestamp,
		ActionCount:  len(traceData.Points),
		SuccessCount: len(traceData.Points),
		FailureCount: 0,
	}

	profile.SessionHistory = append(profile.SessionHistory, session)

	if len(profile.SessionHistory) > 1000 {
		profile.SessionHistory = profile.SessionHistory[len(profile.SessionHistory)-1000:]
	}
}

func (b *UserProfileBuilder) updateRiskScoreHistory(profile *UserProfile, traceData *model.TraceData) {
	riskScore := b.calculateRiskScore(profile, traceData)
	record := RiskScoreRecord{
		Score:     riskScore,
		Timestamp: time.Now(),
		Reason:    "behavior_analysis",
	}

	profile.RiskScoreHistory = append(profile.RiskScoreHistory, record)

	if len(profile.RiskScoreHistory) > 100 {
		profile.RiskScoreHistory = profile.RiskScoreHistory[len(profile.RiskScoreHistory)-100:]
	}
}

func (b *UserProfileBuilder) calculateRiskScore(profile *UserProfile, traceData *model.TraceData) float64 {
	if len(traceData.Points) < 2 {
		return 0.5
	}

	risk := 0.0

	patterns := profile.BehaviorPatterns
	duration := float64(traceData.Points[len(traceData.Points)-1].Timestamp - traceData.Points[0].Timestamp) / 1000.0

	if patterns.TypicalSessionDuration > 0 {
		durationRatio := duration / patterns.TypicalSessionDuration
		if durationRatio < 0.1 || durationRatio > 10 {
			risk += 15
		}
	}

	actionCount := len(traceData.Points)
	if patterns.TypicalActionsPerSession > 0 {
		actionRatio := float64(actionCount) / float64(patterns.TypicalActionsPerSession)
		if actionRatio < 0.2 || actionRatio > 5 {
			risk += 10
		}
	}

	if patterns.ActionIntervalStats.Variance > 0 {
		if patterns.ActionIntervalStats.Variance < 100 {
			risk += 20
		}
	}

	now := time.Now()
	hourKey := fmt.Sprintf("%02d", now.Hour())
	if patterns.TimeOfDayDistribution[hourKey] < 0.1*float64(len(profile.SessionHistory)) {
		risk += 10
	}

	if profile.AnomalyCount > 5 {
		risk += float64(profile.AnomalyCount) * 2
	}

	return math.Min(risk, 100)
}

func (b *UserProfileBuilder) updateProfileConfidence(profile *UserProfile) {
	sessionCount := len(profile.SessionHistory)
	if sessionCount == 0 {
		profile.ProfileConfidence = 0.0
		return
	}

	confidence := math.Min(float64(sessionCount)/10.0, 1.0)

	anomalyRatio := float64(profile.AnomalyCount) / float64(sessionCount)
	confidence *= (1 - anomalyRatio*0.5)

	profile.ProfileConfidence = math.Max(0, confidence)
}

func (b *UserProfileBuilder) AddDeviceFingerprint(userID, fingerprint string) {
	profile := b.GetOrCreateProfile(userID)

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, fp := range profile.DeviceFingerprints {
		if fp == fingerprint {
			return
		}
	}

	profile.DeviceFingerprints = append(profile.DeviceFingerprints, fingerprint)
	profile.LastUpdatedAt = time.Now()

	b.profileStorage.StoreProfile(profile)
}

func (b *UserProfileBuilder) AddLocation(userID, ip, country, region, city string) {
	profile := b.GetOrCreateProfile(userID)

	b.mu.Lock()
	defer b.mu.Unlock()

	record := LocationRecord{
		IP:         ip,
		Country:    country,
		Region:     region,
		City:       city,
		Timestamp:  time.Now(),
		Confidence: 0.9,
	}

	profile.LocationHistory = append(profile.LocationHistory, record)

	if len(profile.LocationHistory) > 100 {
		profile.LocationHistory = profile.LocationHistory[len(profile.LocationHistory)-100:]
	}

	profile.LastUpdatedAt = time.Now()
	b.profileStorage.StoreProfile(profile)
}

func (b *UserProfileBuilder) GetProfile(userID string) (*UserProfile, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	profile, exists := b.profiles[userID]
	return profile, exists
}

func (b *UserProfileBuilder) RemoveProfile(userID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.profiles[userID]; !exists {
		return errors.New("profile not found")
	}

	delete(b.profiles, userID)
	b.profileStorage.RemoveProfile(userID)

	return nil
}

func (b *UserProfileBuilder) DetectAnomalousBehavior(userID string, traceData *model.TraceData) (*AnomalyDetectionResult, error) {
	profile, exists := b.GetProfile(userID)
	if !exists {
		return nil, errors.New("profile not found")
	}

	if traceData == nil || len(traceData.Points) < 2 {
		return nil, errors.New("invalid trace data")
	}

	result := &AnomalyDetectionResult{
		UserID:       userID,
		IsAnomalous:  false,
		Anomalies:    make([]AnomalyDetail, 0),
		Confidence:   0.0,
		OverallScore: 0.0,
	}

	b.detectSessionDurationAnomaly(profile, traceData, result)
	b.detectActionRateAnomaly(profile, traceData, result)
	b.detectTimeAnomaly(profile, result)
	b.detectLocationAnomaly(profile, result)

	result.IsAnomalous = len(result.Anomalies) > 0
	result.OverallScore = b.calculateAnomalyScore(result.Anomalies)
	result.Confidence = profile.ProfileConfidence

	if result.IsAnomalous {
		b.mu.Lock()
		profile.AnomalyCount++
		profile.LastUpdatedAt = time.Now()
		b.mu.Unlock()
	}

	return result, nil
}

type AnomalyDetectionResult struct {
	UserID       string         `json:"user_id"`
	IsAnomalous  bool           `json:"is_anomalous"`
	Anomalies    []AnomalyDetail `json:"anomalies"`
	Confidence   float64        `json:"confidence"`
	OverallScore float64        `json:"overall_score"`
}

type AnomalyDetail struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
	Confidence  float64 `json:"confidence"`
}

func (b *UserProfileBuilder) detectSessionDurationAnomaly(profile *UserProfile, traceData *model.TraceData, result *AnomalyDetectionResult) {
	patterns := profile.BehaviorPatterns
	if patterns.TypicalSessionDuration == 0 {
		return
	}

	duration := float64(traceData.Points[len(traceData.Points)-1].Timestamp - traceData.Points[0].Timestamp) / 1000.0
	durationRatio := duration / patterns.TypicalSessionDuration

	if durationRatio < 0.1 {
		result.Anomalies = append(result.Anomalies, AnomalyDetail{
			Type:        "short_session",
			Description: fmt.Sprintf("会话时间异常短暂 (%.1fs vs 典型 %.1fs)", duration, patterns.TypicalSessionDuration),
			Score:       30,
			Confidence:  0.8,
		})
	} else if durationRatio > 10 {
		result.Anomalies = append(result.Anomalies, AnomalyDetail{
			Type:        "long_session",
			Description: fmt.Sprintf("会话时间异常漫长 (%.1fs vs 典型 %.1fs)", duration, patterns.TypicalSessionDuration),
			Score:       20,
			Confidence:  0.7,
		})
	}
}

func (b *UserProfileBuilder) detectActionRateAnomaly(profile *UserProfile, traceData *model.TraceData, result *AnomalyDetectionResult) {
	patterns := profile.BehaviorPatterns
	if patterns.TypicalActionsPerSession == 0 {
		return
	}

	actionCount := len(traceData.Points)
	duration := float64(traceData.Points[len(traceData.Points)-1].Timestamp - traceData.Points[0].Timestamp) / 1000.0

	if duration > 0 {
		rate := float64(actionCount) / duration
		typicalRate := float64(patterns.TypicalActionsPerSession) / patterns.TypicalSessionDuration

		if typicalRate > 0 {
			rateRatio := rate / typicalRate

			if rateRatio > 3 {
				result.Anomalies = append(result.Anomalies, AnomalyDetail{
					Type:        "high_action_rate",
					Description: fmt.Sprintf("操作频率异常高 (%.1f/s vs 典型 %.1f/s)", rate, typicalRate),
					Score:       35,
					Confidence:  0.85,
				})
			} else if rateRatio < 0.2 {
				result.Anomalies = append(result.Anomalies, AnomalyDetail{
					Type:        "low_action_rate",
					Description: fmt.Sprintf("操作频率异常低 (%.1f/s vs 典型 %.1f/s)", rate, typicalRate),
					Score:       20,
					Confidence:  0.7,
				})
			}
		}
	}
}

func (b *UserProfileBuilder) detectTimeAnomaly(profile *UserProfile, result *AnomalyDetectionResult) {
	if len(profile.SessionHistory) < 10 {
		return
	}

	patterns := profile.BehaviorPatterns
	now := time.Now()
	hourKey := fmt.Sprintf("%02d", now.Hour())
	dayKey := now.Weekday().String()

	totalSessions := float64(len(profile.SessionHistory))
	hourCount := patterns.TimeOfDayDistribution[hourKey]
	dayCount := patterns.DayOfWeekDistribution[dayKey]

	if hourCount/totalSessions < 0.02 {
		result.Anomalies = append(result.Anomalies, AnomalyDetail{
			Type:        "unusual_time",
			Description: fmt.Sprintf("非典型访问时间 (小时 %s)", hourKey),
			Score:       25,
			Confidence:  0.75,
		})
	}

	if dayCount/totalSessions < 0.05 {
		result.Anomalies = append(result.Anomalies, AnomalyDetail{
			Type:        "unusual_day",
			Description: fmt.Sprintf("非典型访问日期 (%s)", dayKey),
			Score:       20,
			Confidence:  0.7,
		})
	}
}

func (b *UserProfileBuilder) detectLocationAnomaly(profile *UserProfile, result *AnomalyDetectionResult) {
	if len(profile.LocationHistory) < 3 {
		return
	}

	locationCounts := make(map[string]int)
	for _, loc := range profile.LocationHistory {
		locationCounts[loc.Country]++
	}

	if len(locationCounts) > 1 {
		maxCount := 0
		var mainCountry string
		for country, count := range locationCounts {
			if count > maxCount {
				maxCount = count
				mainCountry = country
			}
		}

		ratio := float64(maxCount) / float64(len(profile.LocationHistory))
		if ratio < 0.7 {
			result.Anomalies = append(result.Anomalies, AnomalyDetail{
				Type:        "location_change",
				Description: fmt.Sprintf("检测到多个国家访问 (主要: %s)", mainCountry),
				Score:       30,
				Confidence:  0.8,
			})
		}
	}
}

func (b *UserProfileBuilder) calculateAnomalyScore(anomalies []AnomalyDetail) float64 {
	score := 0.0
	for _, anomaly := range anomalies {
		score += anomaly.Score * anomaly.Confidence
	}
	return math.Min(score, 100)
}

func (b *UserProfileBuilder) ExportProfile(userID string) ([]byte, error) {
	profile, exists := b.GetProfile(userID)
	if !exists {
		return nil, errors.New("profile not found")
	}

	return json.MarshalIndent(profile, "", "  ")
}

func (b *UserProfileBuilder) ImportProfile(data []byte) error {
	var profile UserProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.profiles[profile.UserID] = &profile
	b.profileStorage.StoreProfile(&profile)

	return nil
}

func (b *UserProfileBuilder) GetAllProfiles() []*UserProfile {
	b.mu.RLock()
	defer b.mu.RUnlock()

	profiles := make([]*UserProfile, 0, len(b.profiles))
	for _, profile := range b.profiles {
		profiles = append(profiles, profile)
	}

	return profiles
}

func (b *UserProfileBuilder) GetProfileCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.profiles)
}

func (b *UserProfileBuilder) CleanupOldProfiles(maxAge time.Duration) int {
	b.mu.Lock()
	defer b.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for userID, profile := range b.profiles {
		if profile.LastUpdatedAt.Before(cutoff) {
			delete(b.profiles, userID)
			b.profileStorage.RemoveProfile(userID)
			removed++
		}
	}

	return removed
}

func (s *ProfileStorage) StoreProfile(profile *UserProfile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[profile.UserID] = profile
}

func (s *ProfileStorage) GetProfile(userID string) (*UserProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profile, exists := s.data[userID]
	return profile, exists
}

func (s *ProfileStorage) RemoveProfile(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, userID)
}

func (s *ProfileStorage) ExportAll() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type ExportData struct {
		Profiles    map[string]*UserProfile `json:"profiles"`
		ExportedAt  time.Time              `json:"exported_at"`
	}

	data := ExportData{
		Profiles:   s.data,
		ExportedAt: time.Now(),
	}

	return json.MarshalIndent(data, "", "  ")
}

func (b *UserProfileBuilder) GetProfileStatistics(userID string) (*ProfileStatistics, error) {
	profile, exists := b.GetProfile(userID)
	if !exists {
		return nil, errors.New("profile not found")
	}

	stats := &ProfileStatistics{
		UserID:              userID,
		ProfileAgeDays:      int(time.Since(profile.CreatedAt).Hours() / 24),
		TotalSessions:       len(profile.SessionHistory),
		TotalAnomalies:      profile.AnomalyCount,
		ProfileConfidence:   profile.ProfileConfidence,
		DeviceCount:         len(profile.DeviceFingerprints),
		LocationCount:       len(b.getUniqueLocations(profile)),
		AvgSessionDuration:  b.calculateAvgSessionDuration(profile),
		AvgActionsPerSession: b.calculateAvgActionsPerSession(profile),
		RecentRiskScore:     b.getRecentRiskScore(profile),
	}

	return stats, nil
}

type ProfileStatistics struct {
	UserID              string  `json:"user_id"`
	ProfileAgeDays      int     `json:"profile_age_days"`
	TotalSessions       int     `json:"total_sessions"`
	TotalAnomalies      int     `json:"total_anomalies"`
	ProfileConfidence   float64 `json:"profile_confidence"`
	DeviceCount         int     `json:"device_count"`
	LocationCount       int     `json:"location_count"`
	AvgSessionDuration  float64 `json:"avg_session_duration_sec"`
	AvgActionsPerSession int    `json:"avg_actions_per_session"`
	RecentRiskScore     float64 `json:"recent_risk_score"`
}

func (b *UserProfileBuilder) getUniqueLocations(profile *UserProfile) []string {
	locations := make(map[string]bool)
	for _, loc := range profile.LocationHistory {
		key := loc.Country + "-" + loc.City
		locations[key] = true
	}

	result := make([]string, 0, len(locations))
	for loc := range locations {
		result = append(result, loc)
	}
	sort.Strings(result)

	return result
}

func (b *UserProfileBuilder) calculateAvgSessionDuration(profile *UserProfile) float64 {
	if len(profile.SessionHistory) == 0 {
		return 0
	}

	total := 0.0
	for _, session := range profile.SessionHistory {
		total += float64(session.DurationMs) / 1000.0
	}

	return total / float64(len(profile.SessionHistory))
}

func (b *UserProfileBuilder) calculateAvgActionsPerSession(profile *UserProfile) int {
	if len(profile.SessionHistory) == 0 {
		return 0
	}

	total := 0
	for _, session := range profile.SessionHistory {
		total += session.ActionCount
	}

	return total / len(profile.SessionHistory)
}

func (b *UserProfileBuilder) getRecentRiskScore(profile *UserProfile) float64 {
	if len(profile.RiskScoreHistory) == 0 {
		return 0
	}

	recent := profile.RiskScoreHistory[len(profile.RiskScoreHistory)-1]
	return recent.Score
}

func (b *UserProfileBuilder) MergeProfiles(sourceUserID, targetUserID string) error {
	source, exists := b.GetProfile(sourceUserID)
	if !exists {
		return errors.New("source profile not found")
	}

	target := b.GetOrCreateProfile(targetUserID)

	b.mu.Lock()
	defer b.mu.Unlock()

	target.SessionHistory = append(target.SessionHistory, source.SessionHistory...)
	if len(target.SessionHistory) > 1000 {
		target.SessionHistory = target.SessionHistory[len(target.SessionHistory)-1000:]
	}

	target.RiskScoreHistory = append(target.RiskScoreHistory, source.RiskScoreHistory...)
	if len(target.RiskScoreHistory) > 100 {
		target.RiskScoreHistory = target.RiskScoreHistory[len(target.RiskScoreHistory)-100:]
	}

	target.LocationHistory = append(target.LocationHistory, source.LocationHistory...)
	if len(target.LocationHistory) > 100 {
		target.LocationHistory = target.LocationHistory[len(target.LocationHistory)-100:]
	}

	for _, fp := range source.DeviceFingerprints {
		found := false
		for _, targetFp := range target.DeviceFingerprints {
			if fp == targetFp {
				found = true
				break
			}
		}
		if !found {
			target.DeviceFingerprints = append(target.DeviceFingerprints, fp)
		}
	}

	target.AnomalyCount += source.AnomalyCount
	target.LastUpdatedAt = time.Now()

	delete(b.profiles, sourceUserID)
	b.profileStorage.RemoveProfile(sourceUserID)
	b.profileStorage.StoreProfile(target)

	return nil
}