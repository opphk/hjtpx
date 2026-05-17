package service

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

const (
	DifficultyLevelEasy   = 1
	DifficultyLevelMedium = 2
	DifficultyLevelHard   = 3
	DifficultyLevelExpert = 4

	AttackTypeNone            = "none"
	AttackTypeBruteForce      = "brute_force"
	AttackTypeBatchAttack     = "batch_attack"
	AttackTypeDistributed     = "distributed_attack"
	AttackTypeReplayAttack    = "replay_attack"
	AttackTypePatternAttack   = "pattern_attack"
	AttackTypeSpeedAttack     = "speed_attack"
	AttackTypeCoordinated     = "coordinated_attack"
)

type AdaptiveDifficultyLevel int

type AdaptiveConfig struct {
	InitialDifficulty   AdaptiveDifficultyLevel `json:"initial_difficulty"`
	MinDifficulty      AdaptiveDifficultyLevel `json:"min_difficulty"`
	MaxDifficulty     AdaptiveDifficultyLevel `json:"max_difficulty"`
	DifficultyStep    int                   `json:"difficulty_step"`
	AdjustmentWindow  int                   `json:"adjustment_window"`
	SuccessRateTarget float64               `json:"success_rate_target"`
	ConfidenceThreshold float64              `json:"confidence_threshold"`
	CooldownPeriod    time.Duration         `json:"cooldown_period"`
	LearningRate      float64               `json:"learning_rate"`
	ExplorationRate   float64               `json:"exploration_rate"`
}

type AdaptiveUserProfile struct {
	UserID          string                 `json:"user_id"`
	DifficultyLevel AdaptiveDifficultyLevel `json:"difficulty_level"`
	SuccessHistory  []bool                `json:"success_history"`
	LastAdjustment  time.Time             `json:"last_adjustment"`
	TotalAttempts   int                   `json:"total_attempts"`
	TotalSuccesses  int                   `json:"total_successes"`
	AvgResponseTime float64               `json:"avg_response_time"`
	BehaviorFeatures *AdaptiveBehaviorFeatures `json:"behavior_features,omitempty"`
	AdaptiveMetrics AdaptiveUserMetrics     `json:"adaptive_metrics"`
	SessionData     map[string]interface{} `json:"session_data"`
	mu              sync.RWMutex
}

type AdaptiveUserMetrics struct {
	SuccessRate       float64   `json:"success_rate"`
	StreakCount       int      `json:"streak_count"`
	MaxStreak         int      `json:"max_streak"`
	FailureCount      int      `json:"failure_count"`
	AverageDifficulty float64   `json:"average_difficulty"`
	RecentDifficulty  []float64 `json:"recent_difficulty"`
	Confidence        float64   `json:"confidence"`
	AbilityEstimate   float64   `json:"ability_estimate"`
	ErrorVariance     float64   `json:"error_variance"`
	LastResponseTime  float64   `json:"last_response_time"`
}

type AdaptiveDifficultyMetrics struct {
	Level               AdaptiveDifficultyLevel `json:"level"`
	SuccessCount        int                  `json:"success_count"`
	FailureCount        int                  `json:"failure_count"`
	TotalAttempts       int                  `json:"total_attempts"`
	AverageResponseTime float64              `json:"average_response_time"`
	MinResponseTime     float64              `json:"min_response_time"`
	MaxResponseTime     float64              `json:"max_response_time"`
	TimeVariance        float64              `json:"time_variance"`
	SuccessRate         float64              `json:"success_rate"`
	ConfusionLevel      float64              `json:"confusion_level"`
	DiscriminationIndex float64              `json:"discrimination_index"`
}

type AdaptiveBehaviorFeatures struct {
	AvgSpeed              float64 `json:"avg_speed"`
	MaxSpeed              float64 `json:"max_speed"`
	MinSpeed              float64 `json:"min_speed"`
	SpeedVariation        float64 `json:"speed_variation"`
	Acceleration          float64 `json:"acceleration"`
	TrajectorySmoothness  float64 `json:"trajectory_smoothness"`
	ClickInterval         float64 `json:"click_interval"`
	ClickPositionVariance float64 `json:"click_position_variance"`
	PathSimilarity        float64 `json:"path_similarity"`
	PathComplexity        float64 `json:"path_complexity"`
	MicroCorrections      int     `json:"micro_corrections"`
	PauseCount            int     `json:"pause_count"`
	HesitationTime        float64 `json:"hesitation_time"`
	IsHumanLike           bool    `json:"is_human_like"`
	BotScore              float64 `json:"bot_score"`
}

type AdaptiveAttackSignature struct {
	Type              string             `json:"type"`
	PatternHash       string            `json:"pattern_hash"`
	Frequency         int               `json:"frequency"`
	FirstSeen         time.Time        `json:"first_seen"`
	LastSeen          time.Time        `json:"last_seen"`
	AffectedEndpoints []string          `json:"affected_endpoints"`
	Indicators        map[string]float64 `json:"indicators"`
	Confidence        float64           `json:"confidence"`
	IsActive          bool              `json:"is_active"`
}

type AdaptiveAttackPattern struct {
	Type             string            `json:"type"`
	SourceIdentifier string            `json:"source_identifier"`
	Attempts         int              `json:"attempts"`
	TimeWindow       time.Duration    `json:"time_window"`
	SuccessRate      float64          `json:"success_rate"`
	AvgInterval      float64          `json:"avg_interval"`
	IntervalVariance float64          `json:"interval_variance"`
	PeakTimes        []int           `json:"peak_times"`
	RequestPatterns  []AdaptiveRequestPattern `json:"request_patterns"`
	Metadata         map[string]interface{} `json:"metadata"`
}

type AdaptiveRequestPattern struct {
	Method       string            `json:"method"`
	Path         string            `json:"path"`
	BodySize     int              `json:"body_size"`
	Headers      map[string]string `json:"headers"`
	Interval     float64           `json:"interval"`
	IsSequential bool             `json:"is_sequential"`
}

type AdaptiveDistributedIndicator struct {
	UniqueIPs          int     `json:"unique_ips"`
	UniqueUserAgents   int     `json:"unique_user_agents"`
	UniqueFingerprints int     `json:"unique_fingerprints"`
	RequestCount       int     `json:"request_count"`
	SuccessRate        float64 `json:"success_rate"`
	AvgResponseTime    float64 `json:"avg_response_time"`
	IsCoordinated      bool    `json:"is_coordinated"`
	CorrelationScore   float64 `json:"correlation_score"`
	TimeClusterScore   float64 `json:"time_cluster_score"`
}

type AdaptiveDetectionResult struct {
	IsAttack              bool                         `json:"is_attack"`
	AttackType            string                       `json:"attack_type"`
	Confidence            float64                      `json:"confidence"`
	Severity              int                          `json:"severity"`
	AffectedResources     []string                     `json:"affected_resources"`
	RecommendedAction     string                       `json:"recommended_action"`
	Indicators           map[string]float64           `json:"indicators"`
	SourceIdentifiers     []string                     `json:"source_identifiers"`
	PatternSignature      *AdaptiveAttackSignature     `json:"pattern_signature,omitempty"`
	DistributedIndicators *AdaptiveDistributedIndicator `json:"distributed_indicators,omitempty"`
	Timestamp             time.Time                    `json:"timestamp"`
}

type AdaptiveLearningModel struct {
	Weights        map[string]float64           `json:"weights"`
	Thresholds     map[string]float64           `json:"thresholds"`
	FeatureStats   map[string]AdaptiveStats     `json:"feature_stats"`
	AttackPatterns map[string]*AdaptiveAttackPattern `json:"attack_patterns"`
	Version        int                          `json:"version"`
	LastUpdate     time.Time                    `json:"last_update"`
	mu             sync.RWMutex
}

type AdaptiveStats struct {
	Count   int     `json:"count"`
	Mean    float64 `json:"mean"`
	Variance float64 `json:"variance"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
}

type AdaptiveModelUpdate struct {
	Type        string                 `json:"type"`
	FeatureName string                 `json:"feature_name"`
	OldValue    float64                `json:"old_value"`
	NewValue    float64                `json:"new_value"`
	Timestamp   time.Time              `json:"timestamp"`
	Confidence  float64                `json:"confidence"`
	Reason      string                 `json:"reason"`
}

type AdaptiveABTestVariant struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	TrafficPercent  int               `json:"traffic_percent"`
	ConversionRate  float64           `json:"conversion_rate"`
	SampleSize      int               `json:"sample_size"`
	SuccessCount    int               `json:"success_count"`
	Metrics         map[string]float64 `json:"metrics"`
	IsControl       bool              `json:"is_control"`
}

type AdaptiveABTestExperiment struct {
	ID            string                         `json:"id"`
	Name          string                         `json:"name"`
	Variants      []*AdaptiveABTestVariant       `json:"variants"`
	StartTime     time.Time                     `json:"start_time"`
	Status        string                         `json:"status"`
	TargetMetric  string                         `json:"target_metric"`
	MinSampleSize int                           `json:"min_sample_size"`
	Results       map[string]*AdaptiveABTestResult `json:"results"`
}

type AdaptiveABTestResult struct {
	VariantID      string  `json:"variant_id"`
	Conversions    int     `json:"conversions"`
	SampleSize     int     `json:"sample_size"`
	ConversionRate float64 `json:"conversion_rate"`
	Improvement    float64 `json:"improvement"`
	Confidence     float64 `json:"confidence"`
	IsSignificant  bool    `json:"is_significant"`
	PValue         float64 `json:"p_value"`
}

type AdaptiveService struct {
	config              *AdaptiveConfig
	userProfiles        map[string]*AdaptiveUserProfile
	difficultyMetrics   map[AdaptiveDifficultyLevel]*AdaptiveDifficultyMetrics
	learningModel       *AdaptiveLearningModel
	attackSignatures    map[string]*AdaptiveAttackSignature
	activeExperiments   map[string]*AdaptiveABTestExperiment
	eventHistory        []AdaptiveEvent
	mu                  sync.RWMutex
}

type AdaptiveEvent struct {
	UserID       string                      `json:"user_id"`
	EventType    string                      `json:"event_type"`
	Difficulty   AdaptiveDifficultyLevel      `json:"difficulty"`
	Success      bool                        `json:"success"`
	ResponseTime float64                     `json:"response_time"`
	Timestamp    time.Time                  `json:"timestamp"`
	Features     *AdaptiveBehaviorFeatures    `json:"features,omitempty"`
	AttackInfo   *AdaptiveDetectionResult    `json:"attack_info,omitempty"`
}

type AdaptiveVerificationRequest struct {
	UserID       string                    `json:"user_id" binding:"required"`
	SessionID    string                   `json:"session_id"`
	BehaviorData []models.BehaviorData    `json:"behavior_data"`
	Success      bool                     `json:"success"`
	ResponseTime float64                  `json:"response_time"`
	Metadata     map[string]interface{}   `json:"metadata"`
}

type AdaptiveVerificationResponse struct {
	RecommendedDifficulty AdaptiveDifficultyLevel    `json:"recommended_difficulty"`
	CurrentDifficulty    AdaptiveDifficultyLevel    `json:"current_difficulty"`
	AdjustedDifficulty   AdaptiveDifficultyLevel    `json:"adjusted_difficulty"`
	AttackDetection      *AdaptiveDetectionResult   `json:"attack_detection,omitempty"`
	Confidence           float64                   `json:"confidence"`
	Metrics              *AdaptiveUserMetrics      `json:"metrics"`
	NeedsChallenge       bool                     `json:"needs_challenge"`
	ChallengeType        string                   `json:"challenge_type,omitempty"`
}

func NewAdaptiveService() *AdaptiveService {
	return &AdaptiveService{
		config: &AdaptiveConfig{
			InitialDifficulty:   AdaptiveDifficultyLevel(DifficultyLevelMedium),
			MinDifficulty:      AdaptiveDifficultyLevel(DifficultyLevelEasy),
			MaxDifficulty:     AdaptiveDifficultyLevel(DifficultyLevelExpert),
			DifficultyStep:    1,
			AdjustmentWindow:  10,
			SuccessRateTarget: 0.75,
			ConfidenceThreshold: 0.85,
			CooldownPeriod:    30 * time.Second,
			LearningRate:      0.1,
			ExplorationRate:   0.15,
		},
		userProfiles:      make(map[string]*AdaptiveUserProfile),
		difficultyMetrics: make(map[AdaptiveDifficultyLevel]*AdaptiveDifficultyMetrics),
		learningModel: &AdaptiveLearningModel{
			Weights:        make(map[string]float64),
			Thresholds:    make(map[string]float64),
			FeatureStats:  make(map[string]AdaptiveStats),
			AttackPatterns: make(map[string]*AdaptiveAttackPattern),
			Version:       1,
			LastUpdate:    time.Now(),
		},
		attackSignatures:  make(map[string]*AdaptiveAttackSignature),
		activeExperiments: make(map[string]*AdaptiveABTestExperiment),
		eventHistory:      make([]AdaptiveEvent, 0, 10000),
	}
}

func (s *AdaptiveService) ProcessVerification(req *AdaptiveVerificationRequest) *AdaptiveVerificationResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile := s.getOrCreateUserProfile(req.UserID)
	profile.mu.Lock()

	var features *AdaptiveBehaviorFeatures
	if len(req.BehaviorData) > 0 {
		features = s.extractFeatures(req.BehaviorData)
		profile.BehaviorFeatures = features
	}

	attackResult := s.detectAttack(req, features)

	metrics := s.updateMetrics(profile, req.Success, req.ResponseTime)

	currentLevel := profile.DifficultyLevel
	recommendedLevel := s.calculateRecommendedDifficulty(profile)

	adjustedLevel := s.adjustDifficulty(profile, recommendedLevel, attackResult)

	if attackResult != nil && attackResult.IsAttack {
		s.recordAttackPattern(attackResult)
	}

	event := AdaptiveEvent{
		UserID:       req.UserID,
		EventType:    "verification",
		Difficulty:   adjustedLevel,
		Success:      req.Success,
		ResponseTime: req.ResponseTime,
		Timestamp:    time.Now(),
		Features:     features,
		AttackInfo:   attackResult,
	}
	s.eventHistory = append(s.eventHistory, event)
	if len(s.eventHistory) > 10000 {
		s.eventHistory = s.eventHistory[len(s.eventHistory)-5000:]
	}

	profile.mu.Unlock()

	return &AdaptiveVerificationResponse{
		RecommendedDifficulty: recommendedLevel,
		CurrentDifficulty:     currentLevel,
		AdjustedDifficulty:    adjustedLevel,
		AttackDetection:       attackResult,
		Confidence:            metrics.Confidence,
		Metrics:               &metrics,
		NeedsChallenge:       attackResult != nil && attackResult.IsAttack,
		ChallengeType:         s.getChallengeType(attackResult),
	}
}

func (s *AdaptiveService) getOrCreateUserProfile(userID string) *AdaptiveUserProfile {
	profile, exists := s.userProfiles[userID]
	if !exists {
		profile = &AdaptiveUserProfile{
			UserID:          userID,
			DifficultyLevel: AdaptiveDifficultyLevel(DifficultyLevelMedium),
			SuccessHistory: make([]bool, 0),
			LastAdjustment: time.Now(),
			SessionData:    make(map[string]interface{}),
			AdaptiveMetrics: AdaptiveUserMetrics{
				RecentDifficulty: make([]float64, 0),
				AbilityEstimate: 0.5,
			},
		}
		s.userProfiles[userID] = profile
	}
	return profile
}

func (s *AdaptiveService) extractFeatures(behaviorData []models.BehaviorData) *AdaptiveBehaviorFeatures {
	features := &AdaptiveBehaviorFeatures{}

	if len(behaviorData) == 0 {
		return features
	}

	var trajectory []TrajectoryPoint
	var clicks []ClickData

	for _, bd := range behaviorData {
		var dp BehaviorDataPoint
		if err := json.Unmarshal([]byte(bd.Data), &dp); err == nil {
			trajectory = append(trajectory, TrajectoryPoint{
				X:         dp.X,
				Y:         dp.Y,
				Timestamp: dp.Timestamp,
			})
			if dp.Event == "click" {
				clicks = append(clicks, ClickData{
					X:         dp.X,
					Y:         dp.Y,
					Timestamp: dp.Timestamp,
				})
			}
		}
	}

	if len(trajectory) >= 2 {
		features.AvgSpeed = CalculateAverageSpeed(trajectory)
		features.MaxSpeed = CalculateMaxSpeed(trajectory)
		features.MinSpeed = CalculateMinSpeed(trajectory)
		features.SpeedVariation = CalculateSpeedVariation(trajectory)
		features.Acceleration = CalculateAcceleration(trajectory)
		features.TrajectorySmoothness = CalculateTrajectorySmoothness(trajectory)
		features.PathComplexity = CalculatePathComplexity(trajectory)
		features.PathSimilarity = CompareWithHumanTrajectory(trajectory)

		if features.AvgSpeed > 1500 || features.TrajectorySmoothness > 0.95 {
			features.BotScore = math.Min(features.BotScore+0.3, 1.0)
		}
	}

	if len(clicks) >= 2 {
		features.ClickInterval = CalculateClickInterval(clicks)
		features.ClickPositionVariance = CalculateClickPositionVariance(clicks)
	}

	s.updateFeatureStats(features)

	return features
}

func (s *AdaptiveService) updateFeatureStats(features *AdaptiveBehaviorFeatures) {
	s.learningModel.mu.Lock()
	defer s.learningModel.mu.Unlock()

	featureMap := map[string]float64{
		"avg_speed":             features.AvgSpeed,
		"max_speed":             features.MaxSpeed,
		"speed_variation":       features.SpeedVariation,
		"acceleration":          features.Acceleration,
		"trajectory_smoothness": features.TrajectorySmoothness,
		"path_complexity":       features.PathComplexity,
		"path_similarity":       features.PathSimilarity,
		"click_interval":        features.ClickInterval,
	}

	for name, value := range featureMap {
		stats, exists := s.learningModel.FeatureStats[name]
		if !exists {
			stats = AdaptiveStats{
				Count: 0,
				Mean:  0,
				Min:   math.MaxFloat64,
				Max:   -math.MaxFloat64,
			}
		}

		stats.Count++
		oldMean := stats.Mean
		stats.Mean = oldMean + (value-oldMean)/float64(stats.Count)
		stats.Variance = stats.Variance + (value-oldMean)*(value-stats.Mean)
		if stats.Count > 1 {
			stats.Variance = stats.Variance / float64(stats.Count-1)
		}

		if value < stats.Min {
			stats.Min = value
		}
		if value > stats.Max {
			stats.Max = value
		}

		s.learningModel.FeatureStats[name] = stats
	}
}

func (s *AdaptiveService) updateMetrics(profile *AdaptiveUserProfile, success bool, responseTime float64) AdaptiveUserMetrics {
	profile.TotalAttempts++
	if success {
		profile.TotalSuccesses++
		profile.AdaptiveMetrics.SuccessRate = float64(profile.TotalSuccesses) / float64(profile.TotalAttempts)
		profile.AdaptiveMetrics.StreakCount++
		if profile.AdaptiveMetrics.StreakCount > profile.AdaptiveMetrics.MaxStreak {
			profile.AdaptiveMetrics.MaxStreak = profile.AdaptiveMetrics.StreakCount
		}
		profile.AdaptiveMetrics.FailureCount = 0
	} else {
		profile.AdaptiveMetrics.FailureCount++
		profile.AdaptiveMetrics.StreakCount = 0
		profile.AdaptiveMetrics.SuccessRate = float64(profile.TotalSuccesses) / float64(profile.TotalAttempts)
	}

	profile.SuccessHistory = append(profile.SuccessHistory, success)
	if len(profile.SuccessHistory) > s.config.AdjustmentWindow {
		profile.SuccessHistory = profile.SuccessHistory[len(profile.SuccessHistory)-s.config.AdjustmentWindow:]
	}

	if responseTime > 0 {
		if profile.AdaptiveMetrics.LastResponseTime > 0 {
			profile.AdaptiveMetrics.LastResponseTime = profile.AdaptiveMetrics.LastResponseTime*0.9 + responseTime*0.1
		} else {
			profile.AdaptiveMetrics.LastResponseTime = responseTime
		}
	}

	recentDifficulty := float64(profile.DifficultyLevel)
	profile.AdaptiveMetrics.RecentDifficulty = append(profile.AdaptiveMetrics.RecentDifficulty, recentDifficulty)
	if len(profile.AdaptiveMetrics.RecentDifficulty) > s.config.AdjustmentWindow {
		profile.AdaptiveMetrics.RecentDifficulty = profile.AdaptiveMetrics.RecentDifficulty[len(profile.AdaptiveMetrics.RecentDifficulty)-s.config.AdjustmentWindow:]
	}

	var sum float64
	for _, d := range profile.AdaptiveMetrics.RecentDifficulty {
		sum += d
	}
	profile.AdaptiveMetrics.AverageDifficulty = sum / float64(len(profile.AdaptiveMetrics.RecentDifficulty))

	s.updateAbilityEstimate(&profile.AdaptiveMetrics, success, responseTime)

	profile.AdaptiveMetrics.Confidence = s.calculateConfidence(&profile.AdaptiveMetrics)

	return profile.AdaptiveMetrics
}

func (s *AdaptiveService) updateAbilityEstimate(metrics *AdaptiveUserMetrics, success bool, responseTime float64) {
	baseAbility := metrics.AbilityEstimate

	difficultyWeight := metrics.AverageDifficulty / float64(s.config.MaxDifficulty)

	var abilityChange float64
	if success {
		if responseTime > 0 && metrics.LastResponseTime > 0 {
			timeRatio := metrics.LastResponseTime / responseTime
			abilityChange = s.config.LearningRate * (1.0 - baseAbility) * difficultyWeight * timeRatio
		} else {
			abilityChange = s.config.LearningRate * (1.0 - baseAbility) * difficultyWeight
		}
	} else {
		abilityChange = -s.config.LearningRate * baseAbility * (1.0 - difficultyWeight) * 0.5
	}

	metrics.AbilityEstimate = math.Max(0, math.Min(1, baseAbility+abilityChange))
}

func (s *AdaptiveService) calculateConfidence(metrics *AdaptiveUserMetrics) float64 {
	if len(metrics.RecentDifficulty) < 3 {
		return 0.5
	}

	window := s.config.AdjustmentWindow
	if len(metrics.RecentDifficulty) < window {
		window = len(metrics.RecentDifficulty)
	}

	patternCount := 0
	for i := len(metrics.RecentDifficulty) - window; i < len(metrics.RecentDifficulty)-1; i++ {
		if metrics.RecentDifficulty[i] == metrics.RecentDifficulty[i+1] {
			patternCount++
		}
	}

	baseConfidence := 0.5

	difficultyStability := 1.0 - (float64(patternCount) / float64(window))

	timeStability := 1.0
	if metrics.LastResponseTime > 100 {
		expectedTime := 2000.0 * (5.0 - metrics.AverageDifficulty) / 4.0
		timeDiff := math.Abs(metrics.LastResponseTime - expectedTime)
		timeStability = math.Max(0, 1.0-timeDiff/expectedTime)
	}

	confidence := baseConfidence*0.4 + difficultyStability*0.3 + timeStability*0.3

	return math.Max(0, math.Min(1, confidence))
}

func (s *AdaptiveService) calculateRecommendedDifficulty(profile *AdaptiveUserProfile) AdaptiveDifficultyLevel {
	metrics := profile.AdaptiveMetrics

	ability := metrics.AbilityEstimate
	successRate := metrics.SuccessRate
	streak := metrics.StreakCount
	failures := metrics.FailureCount

	var targetDifficulty float64

	abilityWeight := 0.4
	successWeight := 0.35
	streakWeight := 0.15
	failureWeight := 0.1

	baseDifficulty := float64(profile.DifficultyLevel)

	abilityAdjustment := (ability - 0.5) * 2

	successDiff := successRate - s.config.SuccessRateTarget

	targetDifficulty = baseDifficulty +
		abilityWeight*abilityAdjustment +
		successWeight*successDiff*2 +
		streakWeight*float64(streak)*0.1 -
		failureWeight*float64(failures)*0.2

	targetDifficulty = math.Max(float64(s.config.MinDifficulty), targetDifficulty)
	targetDifficulty = math.Min(float64(s.config.MaxDifficulty), targetDifficulty)

	rounded := int(math.Round(targetDifficulty))
	if rounded < int(s.config.MinDifficulty) {
		rounded = int(s.config.MinDifficulty)
	}
	if rounded > int(s.config.MaxDifficulty) {
		rounded = int(s.config.MaxDifficulty)
	}

	return AdaptiveDifficultyLevel(rounded)
}

func (s *AdaptiveService) adjustDifficulty(profile *AdaptiveUserProfile, recommended AdaptiveDifficultyLevel, attackResult *AdaptiveDetectionResult) AdaptiveDifficultyLevel {
	if time.Since(profile.LastAdjustment) < s.config.CooldownPeriod {
		return profile.DifficultyLevel
	}

	current := profile.DifficultyLevel
	adjusted := current

	diff := int(recommended) - int(current)

	if math.Abs(float64(diff)) >= float64(s.config.DifficultyStep) {
		if diff > 0 {
			adjusted = current + AdaptiveDifficultyLevel(s.config.DifficultyStep)
		} else {
			adjusted = current - AdaptiveDifficultyLevel(s.config.DifficultyStep)
		}
	} else if diff != 0 {
		adjusted = recommended
	}

	if adjusted < s.config.MinDifficulty {
		adjusted = s.config.MinDifficulty
	}
	if adjusted > s.config.MaxDifficulty {
		adjusted = s.config.MaxDifficulty
	}

	if attackResult != nil && attackResult.IsAttack && attackResult.Severity >= 3 {
		if adjusted < AdaptiveDifficultyLevel(DifficultyLevelHard) {
			adjusted = AdaptiveDifficultyLevel(DifficultyLevelHard)
		}
	}

	if adjusted != current {
		profile.DifficultyLevel = adjusted
		profile.LastAdjustment = time.Now()
	}

	return adjusted
}

func (s *AdaptiveService) detectAttack(req *AdaptiveVerificationRequest, features *AdaptiveBehaviorFeatures) *AdaptiveDetectionResult {
	result := &AdaptiveDetectionResult{
		Timestamp: time.Now(),
		Indicators: make(map[string]float64),
	}

	indicators := make(map[string]float64)
	var maxConfidence float64
	var attackType string

	batchScore := s.detectBatchAttack(req)
	indicators["batch_attack_score"] = batchScore
	if batchScore > 0.7 && batchScore > maxConfidence {
		maxConfidence = batchScore
		attackType = AttackTypeBatchAttack
	}

	distributedScore := s.detectDistributedAttack(req)
	indicators["distributed_attack_score"] = distributedScore
	if distributedScore > 0.6 && distributedScore > maxConfidence {
		maxConfidence = distributedScore
		attackType = AttackTypeDistributed
	}

	replayScore := s.detectReplayAttack(req)
	indicators["replay_attack_score"] = replayScore
	if replayScore > 0.8 && replayScore > maxConfidence {
		maxConfidence = replayScore
		attackType = AttackTypeReplayAttack
	}

	speedScore := s.detectSpeedAttack(req, features)
	indicators["speed_attack_score"] = speedScore
	if speedScore > 0.75 && speedScore > maxConfidence {
		maxConfidence = speedScore
		attackType = AttackTypeSpeedAttack
	}

	patternScore := s.detectPatternAttack(req)
	indicators["pattern_attack_score"] = patternScore
	if patternScore > 0.7 && patternScore > maxConfidence {
		maxConfidence = patternScore
		attackType = AttackTypePatternAttack
	}

	coordinatedScore := s.detectCoordinatedAttack(req)
	indicators["coordinated_attack_score"] = coordinatedScore
	if coordinatedScore > 0.65 && coordinatedScore > maxConfidence {
		maxConfidence = coordinatedScore
		attackType = AttackTypeCoordinated
	}

	result.Indicators = indicators
	result.Confidence = maxConfidence
	result.AttackType = attackType
	result.IsAttack = maxConfidence >= 0.7

	if result.IsAttack {
		result.Severity = s.calculateSeverity(result)
		result.RecommendedAction = s.getRecommendedAction(result)
		result.SourceIdentifiers = s.extractSourceIdentifiers(req)
		result.AffectedResources = s.extractAffectedResources(req)

		sig := s.findMatchingSignature(result)
		if sig != nil {
			result.PatternSignature = sig
		}
	}

	return result
}

func (s *AdaptiveService) detectBatchAttack(req *AdaptiveVerificationRequest) float64 {
	score := 0.0

	userID := req.UserID
	var sameUserCount int
	var recentCount int
	windowStart := time.Now().Add(-5 * time.Minute)

	for i := len(s.eventHistory) - 1; i >= 0 && i >= len(s.eventHistory)-100; i-- {
		event := s.eventHistory[i]
		if event.UserID == userID {
			recentCount++
		}
		if event.Timestamp.After(windowStart) && event.UserID == userID {
			sameUserCount++
		}
	}

	if sameUserCount > 20 {
		score += 0.4
	} else if sameUserCount > 10 {
		score += 0.2
	}

	if recentCount > 50 {
		score += 0.3
	} else if recentCount > 30 {
		score += 0.15
	}

	var rapidFailures int
	for i := len(s.eventHistory) - 1; i >= 0 && i >= len(s.eventHistory)-10; i-- {
		if s.eventHistory[i].UserID == userID && !s.eventHistory[i].Success {
			rapidFailures++
		}
	}
	if rapidFailures >= 5 {
		score += 0.3
	} else if rapidFailures >= 3 {
		score += 0.15
	}

	return math.Min(score, 1.0)
}

func (s *AdaptiveService) detectDistributedAttack(req *AdaptiveVerificationRequest) float64 {
	score := 0.0

	var recentEvents []AdaptiveEvent
	windowStart := time.Now().Add(-1 * time.Minute)
	for _, event := range s.eventHistory {
		if event.Timestamp.After(windowStart) {
			recentEvents = append(recentEvents, event)
		}
	}

	if len(recentEvents) < 10 {
		return 0.0
	}

	userSet := make(map[string]bool)
	for _, event := range recentEvents {
		userSet[event.UserID] = userSet[event.UserID]
	}

	if len(userSet) > 50 {
		score += 0.3
	} else if len(userSet) > 30 {
		score += 0.2
	}

	var successCount int
	for _, event := range recentEvents {
		if event.Success {
			successCount++
		}
	}
	successRate := float64(successCount) / float64(len(recentEvents))

	if successRate < 0.1 {
		score += 0.4
	} else if successRate < 0.3 {
		score += 0.2
	}

	var rapidEvents int
	lastMinuteStart := time.Now().Add(-1 * time.Minute)
	for _, event := range recentEvents {
		if event.Timestamp.After(lastMinuteStart) {
			rapidEvents++
		}
	}
	if rapidEvents > 100 {
		score += 0.3
	} else if rapidEvents > 50 {
		score += 0.15
	}

	return math.Min(score, 1.0)
}

func (s *AdaptiveService) detectReplayAttack(req *AdaptiveVerificationRequest) float64 {
	score := 0.0

	userID := req.UserID
	var recentEvents []AdaptiveEvent
	windowStart := time.Now().Add(-30 * time.Second)
	for _, event := range s.eventHistory {
		if event.Timestamp.After(windowStart) && event.UserID == userID {
			recentEvents = append(recentEvents, event)
		}
	}

	if len(recentEvents) >= 3 {
		var similarCount int
		for i := range recentEvents[:len(recentEvents)-1] {
			timeDiff := math.Abs(recentEvents[i].ResponseTime - recentEvents[i+1].ResponseTime)
			if timeDiff < 50 {
				similarCount++
			}
		}
		if similarCount >= len(recentEvents)-1 {
			score += 0.5
		}
	}

	if req.Metadata != nil {
		if _, ok := req.Metadata["session_token"]; ok {
			var repeatCount int
			for _, event := range s.eventHistory {
				if event.UserID == userID {
					if event.Features != nil && event.Features.BotScore > 0.7 {
						repeatCount++
					}
				}
			}
			if repeatCount > 5 {
				score += 0.3
			}
		}
	}

	if req.ResponseTime > 0 && req.ResponseTime < 100 {
		score += 0.2
	}

	return math.Min(score, 1.0)
}

func (s *AdaptiveService) detectSpeedAttack(req *AdaptiveVerificationRequest, features *AdaptiveBehaviorFeatures) float64 {
	score := 0.0

	if features != nil {
		if features.AvgSpeed > 2000 {
			score += 0.4
		} else if features.AvgSpeed > 1000 {
			score += 0.2
		}

		if features.TrajectorySmoothness > 0.95 {
			score += 0.3
		} else if features.TrajectorySmoothness > 0.9 {
			score += 0.15
		}

		if features.PathComplexity < 0.1 {
			score += 0.2
		}

		if features.MicroCorrections == 0 && features.PauseCount == 0 {
			score += 0.2
		}

		if features.BotScore > 0.7 {
			score += 0.3
		}
	}

	if req.ResponseTime > 0 && req.ResponseTime < 500 {
		score += 0.2
	}

	return math.Min(score, 1.0)
}

func (s *AdaptiveService) detectPatternAttack(req *AdaptiveVerificationRequest) float64 {
	score := 0.0

	userID := req.UserID
	var recentEvents []AdaptiveEvent
	windowStart := time.Now().Add(-5 * time.Minute)
	for _, event := range s.eventHistory {
		if event.Timestamp.After(windowStart) && event.UserID == userID {
			recentEvents = append(recentEvents, event)
		}
	}

	if len(recentEvents) < 3 {
		return 0.0
	}

	var sameDifficultyCount int
	targetLevel := recentEvents[len(recentEvents)-1].Difficulty
	for i := len(recentEvents) - 1; i >= 0 && i >= len(recentEvents)-5; i-- {
		if recentEvents[i].Difficulty == targetLevel {
			sameDifficultyCount++
		}
	}
	if sameDifficultyCount >= 5 {
		score += 0.3
	}

	var allSuccess bool = true
	var allFail bool = true
	for _, event := range recentEvents {
		if event.Success {
			allFail = false
		} else {
			allSuccess = false
		}
	}
	if allSuccess || allFail {
		score += 0.3
	}

	var similarIntervals int
	for i := 1; i < len(recentEvents); i++ {
		if recentEvents[i].ResponseTime > 0 && recentEvents[i-1].ResponseTime > 0 {
			ratio := recentEvents[i].ResponseTime / recentEvents[i-1].ResponseTime
			if ratio > 0.9 && ratio < 1.1 {
				similarIntervals++
			}
		}
	}
	if similarIntervals >= len(recentEvents)-2 {
		score += 0.4
	}

	return math.Min(score, 1.0)
}

func (s *AdaptiveService) detectCoordinatedAttack(req *AdaptiveVerificationRequest) float64 {
	score := 0.0

	var recentEvents []AdaptiveEvent
	windowStart := time.Now().Add(-2 * time.Minute)
	for _, event := range s.eventHistory {
		if event.Timestamp.After(windowStart) {
			recentEvents = append(recentEvents, event)
		}
	}

	if len(recentEvents) < 20 {
		return 0.0
	}

	var successCount int
	for _, event := range recentEvents {
		if event.Success {
			successCount++
		}
	}
	successRate := float64(successCount) / float64(len(recentEvents))

	if successRate < 0.05 {
		score += 0.4
	} else if successRate < 0.15 {
		score += 0.2
	}

	var timeClusters int
	intervalHistogram := make(map[int]int)
	for _, event := range recentEvents {
		second := event.Timestamp.Second()
		intervalHistogram[second/10]++
	}
	for _, count := range intervalHistogram {
		if count > len(recentEvents)/5 {
			timeClusters++
		}
	}
	if timeClusters >= 3 {
		score += 0.3
	}

	var uniqueUsers int
	userSet := make(map[string]bool)
	for _, event := range recentEvents {
		userSet[event.UserID] = true
	}
	uniqueUsers = len(userSet)
	if uniqueUsers > 100 && float64(uniqueUsers)/float64(len(recentEvents)) > 0.9 {
		score += 0.3
	}

	return math.Min(score, 1.0)
}

func (s *AdaptiveService) calculateSeverity(result *AdaptiveDetectionResult) int {
	severity := 0

	if result.Confidence >= 0.95 {
		severity += 3
	} else if result.Confidence >= 0.85 {
		severity += 2
	} else if result.Confidence >= 0.7 {
		severity += 1
	}

	switch result.AttackType {
	case AttackTypeDistributed, AttackTypeCoordinated:
		severity += 2
	case AttackTypeBatchAttack, AttackTypeBruteForce:
		severity += 1
	}

	if len(result.SourceIdentifiers) > 10 {
		severity += 2
	} else if len(result.SourceIdentifiers) > 5 {
		severity += 1
	}

	minSeverity := severity
	if minSeverity > 5 {
		minSeverity = 5
	}
	return minSeverity
}

func (s *AdaptiveService) getRecommendedAction(result *AdaptiveDetectionResult) string {
	if result.Severity >= 4 {
		return "block"
	} else if result.Severity >= 3 {
		return "challenge_captcha"
	} else if result.Severity >= 2 {
		return "require_verification"
	}
	return "log_only"
}

func (s *AdaptiveService) extractSourceIdentifiers(req *AdaptiveVerificationRequest) []string {
	identifiers := make([]string, 0)
	identifiers = append(identifiers, req.UserID)
	if req.SessionID != "" {
		identifiers = append(identifiers, req.SessionID)
	}
	return identifiers
}

func (s *AdaptiveService) extractAffectedResources(req *AdaptiveVerificationRequest) []string {
	return []string{"/api/verify", "/api/challenge"}
}

func (s *AdaptiveService) getChallengeType(attackResult *AdaptiveDetectionResult) string {
	if attackResult == nil || !attackResult.IsAttack {
		return ""
	}

	switch attackResult.AttackType {
	case AttackTypeSpeedAttack, AttackTypePatternAttack:
		return "behavior_analysis"
	case AttackTypeBatchAttack, AttackTypeBruteForce:
		return "captcha"
	case AttackTypeDistributed, AttackTypeCoordinated:
		return "advanced_captcha"
	default:
		return "standard_captcha"
	}
}

func (s *AdaptiveService) recordAttackPattern(result *AdaptiveDetectionResult) {
	if result == nil || !result.IsAttack {
		return
	}

	s.learningModel.mu.Lock()
	defer s.learningModel.mu.Unlock()

	sigID := s.generateSignatureID(result)
	sig, exists := s.attackSignatures[sigID]

	if !exists {
		sig = &AdaptiveAttackSignature{
			Type:        result.AttackType,
			PatternHash: sigID,
			FirstSeen:   time.Now(),
			Indicators:  make(map[string]float64),
		}
		s.attackSignatures[sigID] = sig
	}

	sig.LastSeen = time.Now()
	sig.Frequency++
	sig.IsActive = true
	sig.Confidence = result.Confidence

	for k, v := range result.Indicators {
		if existing, ok := sig.Indicators[k]; ok {
			sig.Indicators[k] = existing*0.9 + v*0.1
		} else {
			sig.Indicators[k] = v
		}
	}
}

func (s *AdaptiveService) generateSignatureID(result *AdaptiveDetectionResult) string {
	hasher := fnv.New64a()
	hasher.Write([]byte(result.AttackType))
	hasher.Write([]byte(fmt.Sprintf("%d", result.Severity)))
	for _, source := range result.SourceIdentifiers {
		hasher.Write([]byte(source))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func (s *AdaptiveService) findMatchingSignature(result *AdaptiveDetectionResult) *AdaptiveAttackSignature {
	if result == nil {
		return nil
	}

	for _, sig := range s.attackSignatures {
		if !sig.IsActive {
			continue
		}
		if sig.Type != result.AttackType {
			continue
		}

		confidenceDiff := math.Abs(sig.Confidence - result.Confidence)
		if confidenceDiff < 0.2 {
			return sig
		}
	}

	return nil
}

func (s *AdaptiveService) CreateExperiment(name string, variants []*AdaptiveABTestVariant) (*AdaptiveABTestExperiment, error) {
	totalTraffic := 0
	for _, v := range variants {
		totalTraffic += v.TrafficPercent
	}
	if totalTraffic != 100 {
		return nil, fmt.Errorf("traffic percentage must sum to 100")
	}

	experiment := &AdaptiveABTestExperiment{
		ID:            fmt.Sprintf("exp_%d", time.Now().UnixNano()),
		Name:          name,
		Variants:      variants,
		StartTime:     time.Now(),
		Status:        "running",
		TargetMetric:  "conversion_rate",
		MinSampleSize: 100,
		Results:       make(map[string]*AdaptiveABTestResult),
	}

	for _, v := range variants {
		experiment.Results[v.ID] = &AdaptiveABTestResult{
			VariantID:      v.ID,
			Conversions:    0,
			SampleSize:     0,
			ConversionRate: 0,
		}
	}

	s.mu.Lock()
	s.activeExperiments[experiment.ID] = experiment
	s.mu.Unlock()

	return experiment, nil
}

func (s *AdaptiveService) AssignVariant(experimentID string, userID string) (*AdaptiveABTestVariant, error) {
	s.mu.RLock()
	experiment, exists := s.activeExperiments[experimentID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("experiment not found")
	}

	if experiment.Status != "running" {
		return nil, fmt.Errorf("experiment is not running")
	}

	hash := fnv.New64a()
	hash.Write([]byte(userID))
	hash.Write([]byte(experiment.ID))
	hashNum := hash.Sum64()

	cumulative := 0
	selected := experiment.Variants[0]
	for _, variant := range experiment.Variants {
		cumulative += variant.TrafficPercent
		if int(hashNum%100) < cumulative {
			selected = variant
			break
		}
	}

	return selected, nil
}

func (s *AdaptiveService) RecordConversion(experimentID, variantID string, success bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	experiment, exists := s.activeExperiments[experimentID]
	if !exists {
		return fmt.Errorf("experiment not found")
	}

	result, exists := experiment.Results[variantID]
	if !exists {
		return fmt.Errorf("variant not found")
	}

	result.SampleSize++
	if success {
		result.Conversions++
	}

	if result.SampleSize > 0 {
		result.ConversionRate = float64(result.Conversions) / float64(result.SampleSize)
	}

	return nil
}

func (s *AdaptiveService) AnalyzeExperiment(experimentID string) (*AdaptiveABTestExperiment, error) {
	s.mu.RLock()
	experiment, exists := s.activeExperiments[experimentID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("experiment not found")
	}

	var controlResult *AdaptiveABTestResult
	for _, variant := range experiment.Variants {
		if result, ok := experiment.Results[variant.ID]; ok {
			if variant.Name == "control" || variant.IsControl {
				controlResult = result
				break
			}
		}
	}

	if controlResult == nil {
		for _, result := range experiment.Results {
			controlResult = result
			break
		}
	}

	for _, result := range experiment.Results {
		if result == controlResult {
			continue
		}

		if controlResult.SampleSize > 0 && result.SampleSize > 0 {
			p1 := float64(controlResult.Conversions) / float64(controlResult.SampleSize)
			p2 := float64(result.Conversions) / float64(result.SampleSize)

			if p1 > 0 {
				result.Improvement = ((p2 - p1) / p1) * 100
			}

			result.Confidence = s.calculateStatisticalConfidence(
				controlResult.SampleSize, controlResult.Conversions,
				result.SampleSize, result.Conversions,
			)

			result.IsSignificant = result.Confidence >= 95.0
			result.PValue = 1.0 - (result.Confidence / 100.0)
		}
	}

	return experiment, nil
}

func (s *AdaptiveService) calculateStatisticalConfidence(n1, c1, n2, c2 int) float64 {
	if n1 == 0 || n2 == 0 {
		return 0
	}

	p1 := float64(c1) / float64(n1)
	p2 := float64(c2) / float64(n2)

	pooled := float64(c1+c2) / float64(n1+n2)
	se := math.Sqrt(pooled * (1 - pooled) * (1/float64(n1) + 1/float64(n2)))

	if se == 0 {
		return 0
	}

	zScore := math.Abs(p2 - p1) / se

	confidence := (1 - 2*(1-s.normalCDF(zScore))) * 100

	return math.Max(0, math.Min(100, confidence))
}

func (s *AdaptiveService) normalCDF(x float64) float64 {
	return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}

func (s *AdaptiveService) UpdateLearningModel(update *AdaptiveModelUpdate) error {
	s.learningModel.mu.Lock()
	defer s.learningModel.mu.Unlock()

	switch update.Type {
	case "weight":
		s.learningModel.Weights[update.FeatureName] = update.NewValue
	case "threshold":
		s.learningModel.Thresholds[update.FeatureName] = update.NewValue
	case "feature_stat":
		if stats, ok := s.learningModel.FeatureStats[update.FeatureName]; ok {
			stats.Mean = update.NewValue
			s.learningModel.FeatureStats[update.FeatureName] = stats
		}
	}

	s.learningModel.Version++
	s.learningModel.LastUpdate = time.Now()

	return nil
}

func (s *AdaptiveService) GetLearningModel() *AdaptiveLearningModel {
	s.learningModel.mu.RLock()
	defer s.learningModel.mu.RUnlock()

	return &AdaptiveLearningModel{
		Weights:      s.learningModel.Weights,
		Thresholds:  s.learningModel.Thresholds,
		FeatureStats: s.learningModel.FeatureStats,
		Version:     s.learningModel.Version,
		LastUpdate:  s.learningModel.LastUpdate,
	}
}

func (s *AdaptiveService) GetDifficultyMetrics(level AdaptiveDifficultyLevel) *AdaptiveDifficultyMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.difficultyMetrics[level]
}

func (s *AdaptiveService) GetAllDifficultyMetrics() map[AdaptiveDifficultyLevel]*AdaptiveDifficultyMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[AdaptiveDifficultyLevel]*AdaptiveDifficultyMetrics)
	for k, v := range s.difficultyMetrics {
		result[k] = v
	}
	return result
}

func (s *AdaptiveService) GetUserProfile(userID string) *AdaptiveUserProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.userProfiles[userID]
}

func (s *AdaptiveService) GetActiveAttackSignatures() []*AdaptiveAttackSignature {
	s.mu.RLock()
	defer s.mu.RUnlock()

	signatures := make([]*AdaptiveAttackSignature, 0)
	for _, sig := range s.attackSignatures {
		if sig.IsActive {
			signatures = append(signatures, sig)
		}
	}
	return signatures
}

func (s *AdaptiveService) GetEventHistory(userID string, limit int) []AdaptiveEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]AdaptiveEvent, 0)
	for i := len(s.eventHistory) - 1; i >= 0 && len(events) < limit; i-- {
		if s.eventHistory[i].UserID == userID {
			events = append(events, s.eventHistory[i])
		}
	}
	return events
}

func (s *AdaptiveService) CleanupOldData(olderThan time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)

	newHistory := make([]AdaptiveEvent, 0, len(s.eventHistory))
	for _, event := range s.eventHistory {
		if event.Timestamp.After(cutoff) {
			newHistory = append(newHistory, event)
		}
	}
	s.eventHistory = newHistory

	for id, sig := range s.attackSignatures {
		if sig.LastSeen.Before(cutoff) {
			sig.IsActive = false
			delete(s.attackSignatures, id)
		}
	}

	for id, profile := range s.userProfiles {
		if time.Since(profile.LastAdjustment) > olderThan*2 {
			delete(s.userProfiles, id)
		}
	}
}

func (s *AdaptiveService) SyncThreatIntelligence() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, sig := range s.attackSignatures {
		if sig.Frequency > 100 && sig.Confidence > 0.9 {
			s.learningModel.AttackPatterns[sig.PatternHash] = &AdaptiveAttackPattern{
				Type:             sig.Type,
				SourceIdentifier: sig.PatternHash,
				Attempts:         sig.Frequency,
				TimeWindow:       time.Since(sig.FirstSeen),
				SuccessRate:      0,
				AvgInterval:      0,
				RequestPatterns:  []AdaptiveRequestPattern{},
			}
		}
	}

	return nil
}

func CalculateAdaptiveBotProbability(features *AdaptiveBehaviorFeatures) float64 {
	if features == nil {
		return 0.5
	}

	score := 0.0

	if features.AvgSpeed > 1500 {
		score += 0.25
	} else if features.AvgSpeed > 800 {
		score += 0.1
	}

	if features.TrajectorySmoothness > 0.95 {
		score += 0.25
	} else if features.TrajectorySmoothness > 0.9 {
		score += 0.1
	}

	if features.Acceleration < 0.1 && features.AvgSpeed > 100 {
		score += 0.2
	}

	if features.PathComplexity < 0.2 {
		score += 0.15
	}

	if features.PathSimilarity < 0.3 {
		score += 0.2
	}

	if features.MicroCorrections == 0 && features.PauseCount == 0 {
		score += 0.15
	}

	return math.Min(score, 1.0)
}

type AdaptiveChallenge struct {
	Type        string                     `json:"type"`
	Difficulty  AdaptiveDifficultyLevel    `json:"difficulty"`
	Parameters  map[string]interface{}     `json:"parameters"`
	TimeLimit   time.Duration              `json:"time_limit"`
	MaxAttempts int                        `json:"max_attempts"`
}

func GenerateAdaptiveChallenge(level AdaptiveDifficultyLevel) *AdaptiveChallenge {
	challenge := &AdaptiveChallenge{
		Type:       "standard",
		Difficulty: level,
		Parameters: make(map[string]interface{}),
		TimeLimit:  30 * time.Second,
		MaxAttempts: 3,
	}

	switch level {
	case AdaptiveDifficultyLevel(DifficultyLevelEasy):
		challenge.Parameters["puzzle_pieces"] = 3
		challenge.Parameters["distortion"] = 0.2
		challenge.Parameters["noise_level"] = 0.1
	case AdaptiveDifficultyLevel(DifficultyLevelMedium):
		challenge.Parameters["puzzle_pieces"] = 5
		challenge.Parameters["distortion"] = 0.4
		challenge.Parameters["noise_level"] = 0.2
	case AdaptiveDifficultyLevel(DifficultyLevelHard):
		challenge.Parameters["puzzle_pieces"] = 7
		challenge.Parameters["distortion"] = 0.6
		challenge.Parameters["noise_level"] = 0.35
	case AdaptiveDifficultyLevel(DifficultyLevelExpert):
		challenge.Parameters["puzzle_pieces"] = 9
		challenge.Parameters["distortion"] = 0.8
		challenge.Parameters["noise_level"] = 0.5
		challenge.TimeLimit = 45 * time.Second
	}

	return challenge
}

type AdaptiveEnsembleDetector struct {
	detectors []AdaptiveAttackDetector
	weights   []float64
}

type AdaptiveAttackDetector interface {
	Detect(req *AdaptiveVerificationRequest, features *AdaptiveBehaviorFeatures) float64
	GetName() string
}

func NewAdaptiveEnsembleDetector() *AdaptiveEnsembleDetector {
	return &AdaptiveEnsembleDetector{
		detectors: make([]AdaptiveAttackDetector, 0),
		weights:   make([]float64, 0),
	}
}

func (e *AdaptiveEnsembleDetector) AddDetector(detector AdaptiveAttackDetector, weight float64) {
	e.detectors = append(e.detectors, detector)
	e.weights = append(e.weights, weight)
}

func (e *AdaptiveEnsembleDetector) Detect(req *AdaptiveVerificationRequest, features *AdaptiveBehaviorFeatures) float64 {
	if len(e.detectors) == 0 {
		return 0.0
	}

	var totalScore float64
	var totalWeight float64

	for i, detector := range e.detectors {
		score := detector.Detect(req, features)
		totalScore += score * e.weights[i]
		totalWeight += e.weights[i]
	}

	if totalWeight > 0 {
		return totalScore / totalWeight
	}

	return 0.0
}

type AdaptiveStatisticalDetector struct {
	name           string
	threshold      float64
	windowSize    int
	meanExpected   float64
	varianceExpected float64
}

func NewAdaptiveStatisticalDetector(name string, threshold float64) *AdaptiveStatisticalDetector {
	return &AdaptiveStatisticalDetector{
		name:            name,
		threshold:       threshold,
		windowSize:     100,
		meanExpected:   1000,
		varianceExpected: 500,
	}
}

func (d *AdaptiveStatisticalDetector) Detect(req *AdaptiveVerificationRequest, features *AdaptiveBehaviorFeatures) float64 {
	score := 0.0

	if req.ResponseTime > 0 {
		zScore := (req.ResponseTime - d.meanExpected) / math.Sqrt(d.varianceExpected)
		if math.Abs(zScore) > 3 {
			score += 0.5
		}
	}

	return math.Min(score, 1.0)
}

func (d *AdaptiveStatisticalDetector) GetName() string {
	return d.name
}
