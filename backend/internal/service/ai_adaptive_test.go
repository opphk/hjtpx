package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

func TestNewAdaptiveService(t *testing.T) {
	service := NewAdaptiveService()

	if service == nil {
		t.Fatal("Expected service to be created")
	}

	if service.config == nil {
		t.Fatal("Expected config to be initialized")
	}

	if service.userProfiles == nil {
		t.Fatal("Expected userProfiles to be initialized")
	}

	if service.learningModel == nil {
		t.Fatal("Expected learningModel to be initialized")
	}

	if service.attackSignatures == nil {
		t.Fatal("Expected attackSignatures to be initialized")
	}
}

func TestAdaptiveService_ProcessVerification(t *testing.T) {
	service := NewAdaptiveService()

	tests := []struct {
		name         string
		userID       string
		success      bool
		responseTime float64
	}{
		{
			name:         "First verification for new user",
			userID:       "user_001",
			success:      true,
			responseTime: 2000,
		},
		{
			name:         "Failed verification",
			userID:       "user_002",
			success:      false,
			responseTime: 3000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &AdaptiveVerificationRequest{
				UserID:       tt.userID,
				Success:      tt.success,
				ResponseTime: tt.responseTime,
				Metadata:     make(map[string]interface{}),
			}

			resp := service.ProcessVerification(req)

			if resp == nil {
				t.Fatal("Expected response to not be nil")
			}

			if resp.CurrentDifficulty == 0 {
				t.Fatal("Expected current difficulty to be set")
			}

			if resp.RecommendedDifficulty == 0 {
				t.Fatal("Expected recommended difficulty to be set")
			}

			if resp.Metrics == nil {
				t.Fatal("Expected metrics to be set")
			}

			profile := service.GetUserProfile(tt.userID)
			if profile == nil {
				t.Fatal("Expected user profile to be created")
			}

			if profile.TotalAttempts != 1 {
				t.Errorf("Expected 1 attempt, got %d", profile.TotalAttempts)
			}
		})
	}
}

func TestAdaptiveService_DifficultyAdjustment(t *testing.T) {
	service := NewAdaptiveService()

	userID := "user_diff_test"

	profile := service.getOrCreateUserProfile(userID)
	profile.DifficultyLevel = AdaptiveDifficultyLevel(DifficultyLevelMedium)
	service.config.SuccessRateTarget = 0.75

	for i := 0; i < 15; i++ {
		req := &AdaptiveVerificationRequest{
			UserID:       userID,
			Success:      true,
			ResponseTime: 1500,
			Metadata:     make(map[string]interface{}),
		}
		service.ProcessVerification(req)
	}

	profile = service.GetUserProfile(userID)
	if profile.AdaptiveMetrics.SuccessRate < 0.7 {
		t.Errorf("Expected success rate > 0.7, got %f", profile.AdaptiveMetrics.SuccessRate)
	}
}

func TestAdaptiveService_AbilityEstimate(t *testing.T) {
	service := NewAdaptiveService()

	userID := "user_ability_test"

	for i := 0; i < 10; i++ {
		req := &AdaptiveVerificationRequest{
			UserID:       userID,
			Success:      true,
			ResponseTime: 1000 + float64(i*100),
			Metadata:     make(map[string]interface{}),
		}
		service.ProcessVerification(req)
	}

	profile := service.GetUserProfile(userID)

	if profile.AdaptiveMetrics.AbilityEstimate < 0.5 {
		t.Errorf("Expected ability estimate >= 0.5 after successful verifications, got %f", profile.AdaptiveMetrics.AbilityEstimate)
	}

	metrics := profile.AdaptiveMetrics
	metrics.AbilityEstimate = 0.8
	service.updateAbilityEstimate(&metrics, false, 5000)

	if metrics.AbilityEstimate >= 0.8 {
		t.Errorf("Expected ability estimate to decrease after failure, got %f", metrics.AbilityEstimate)
	}
}

func TestAdaptiveService_AttackDetection(t *testing.T) {
	service := NewAdaptiveService()

	tests := []struct {
		name           string
		setupFunc      func()
		userID         string
		expectedAttack bool
	}{
		{
			name: "Normal user should not be detected as attack",
			setupFunc: func() {
				service.eventHistory = make([]AdaptiveEvent, 0)
			},
			userID:         "normal_user",
			expectedAttack: false,
		},
		{
			name: "Rapid failed attempts should trigger batch attack detection",
			setupFunc: func() {
				service.eventHistory = make([]AdaptiveEvent, 0)
				userID := "batch_attack_user"
				for i := 0; i < 60; i++ {
					service.eventHistory = append(service.eventHistory, AdaptiveEvent{
						UserID:    userID,
						Success:   false,
						Timestamp: time.Now().Add(-time.Duration(5-i/10) * time.Minute),
					})
				}
			},
			userID:         "batch_attack_user",
			expectedAttack: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()

			req := &AdaptiveVerificationRequest{
				UserID:       tt.userID,
				Success:      false,
				ResponseTime: 100,
				Metadata:     make(map[string]interface{}),
			}

			resp := service.ProcessVerification(req)

			if tt.expectedAttack && resp.AttackDetection == nil {
				t.Error("Expected attack detection result")
			}

			if tt.expectedAttack && resp.AttackDetection != nil && !resp.AttackDetection.IsAttack {
				t.Error("Expected attack to be detected")
			}
		})
	}
}

func TestAdaptiveService_BatchAttackDetection(t *testing.T) {
	service := NewAdaptiveService()

	userID := "batch_user"
	service.eventHistory = make([]AdaptiveEvent, 0)

	for i := 0; i < 60; i++ {
		service.eventHistory = append(service.eventHistory, AdaptiveEvent{
			UserID:    userID,
			Success:   false,
			Timestamp: time.Now().Add(-time.Duration(i/10) * time.Minute),
		})
	}

	req := &AdaptiveVerificationRequest{
		UserID:       userID,
		Success:      false,
		ResponseTime: 50,
		Metadata:     make(map[string]interface{}),
	}

	score := service.detectBatchAttack(req)

	if score < 0.5 {
		t.Errorf("Expected batch attack score > 0.5, got %f", score)
	}
}

func TestAdaptiveService_DistributedAttackDetection(t *testing.T) {
	service := NewAdaptiveService()

	service.eventHistory = make([]AdaptiveEvent, 0)

	for i := 0; i < 60; i++ {
		service.eventHistory = append(service.eventHistory, AdaptiveEvent{
			UserID:    fmt.Sprintf("user_%d", i%50),
			Success:   false,
			Timestamp: time.Now().Add(-time.Duration(i) * time.Second),
		})
	}

	req := &AdaptiveVerificationRequest{
		UserID:       "distributed_attacker",
		Success:      false,
		ResponseTime: 100,
		Metadata:     make(map[string]interface{}),
	}

	score := service.detectDistributedAttack(req)

	if score < 0.3 {
		t.Errorf("Expected distributed attack score >= 0.3, got %f", score)
	}
}

func TestAdaptiveService_SpeedAttackDetection(t *testing.T) {
	service := NewAdaptiveService()

	tests := []struct {
		name     string
		features *AdaptiveBehaviorFeatures
		minScore float64
	}{
		{
			name: "High speed bot-like behavior",
			features: &AdaptiveBehaviorFeatures{
				AvgSpeed:             2500,
				TrajectorySmoothness: 0.98,
				PathComplexity:       0.05,
				MicroCorrections:     0,
				PauseCount:           0,
				BotScore:             0.8,
			},
			minScore: 0.6,
		},
		{
			name: "Human-like behavior",
			features: &AdaptiveBehaviorFeatures{
				AvgSpeed:             500,
				TrajectorySmoothness: 0.7,
				PathComplexity:       0.4,
				MicroCorrections:     10,
				PauseCount:           5,
				BotScore:             0.1,
			},
			minScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &AdaptiveVerificationRequest{
				UserID:       "speed_user",
				Success:      true,
				ResponseTime: 300,
				Metadata:     make(map[string]interface{}),
			}

			score := service.detectSpeedAttack(req, tt.features)

			if score < tt.minScore {
				t.Errorf("Expected speed attack score >= %f, got %f", tt.minScore, score)
			}
		})
	}
}

func TestAdaptiveService_PatternAttackDetection(t *testing.T) {
	service := NewAdaptiveService()

	userID := "pattern_user"
	service.eventHistory = make([]AdaptiveEvent, 0)

	for i := 0; i < 10; i++ {
		service.eventHistory = append(service.eventHistory, AdaptiveEvent{
			UserID:       userID,
			Success:      true,
			Difficulty:   AdaptiveDifficultyLevel(DifficultyLevelMedium),
			ResponseTime: 2000,
			Timestamp:    time.Now().Add(-time.Duration(10-i) * time.Minute),
		})
	}

	req := &AdaptiveVerificationRequest{
		UserID:       userID,
		Success:      true,
		ResponseTime: 2000,
		Metadata:     make(map[string]interface{}),
	}

	score := service.detectPatternAttack(req)

	if score < 0.3 {
		t.Errorf("Expected pattern attack score >= 0.3 for regular pattern, got %f", score)
	}
}

func TestAdaptiveService_ReplayAttackDetection(t *testing.T) {
	service := NewAdaptiveService()

	userID := "replay_user"
	service.eventHistory = make([]AdaptiveEvent, 0)

	for i := 0; i < 5; i++ {
		service.eventHistory = append(service.eventHistory, AdaptiveEvent{
			UserID:       userID,
			Success:      false,
			ResponseTime: 2000,
			Timestamp:    time.Now().Add(-time.Duration(5-i) * time.Second),
		})
	}

	req := &AdaptiveVerificationRequest{
		UserID:       userID,
		Success:      false,
		ResponseTime: 1999,
		Metadata:     make(map[string]interface{}),
	}

	score := service.detectReplayAttack(req)

	if score < 0.3 {
		t.Errorf("Expected replay attack score >= 0.3 for similar response times, got %f", score)
	}
}

func TestAdaptiveService_CoordinatedAttackDetection(t *testing.T) {
	service := NewAdaptiveService()

	service.eventHistory = make([]AdaptiveEvent, 0)

	for i := 0; i < 100; i++ {
		service.eventHistory = append(service.eventHistory, AdaptiveEvent{
			UserID:    fmt.Sprintf("coord_user_%d", i),
			Success:   false,
			Timestamp: time.Now().Add(-time.Duration(i/10) * time.Second),
		})
	}

	req := &AdaptiveVerificationRequest{
		UserID:       "coordinated_attacker",
		Success:      false,
		ResponseTime: 50,
		Metadata:     make(map[string]interface{}),
	}

	score := service.detectCoordinatedAttack(req)

	if score < 0.3 {
		t.Errorf("Expected coordinated attack score >= 0.3, got %f", score)
	}
}

func TestAdaptiveService_SeverityCalculation(t *testing.T) {
	service := NewAdaptiveService()

	tests := []struct {
		name                string
		confidence          float64
		attackType          string
		sourceCount         int
		expectedMinSeverity int
	}{
		{
			name:                "High confidence distributed attack",
			confidence:          0.95,
			attackType:         AttackTypeDistributed,
			sourceCount:         20,
			expectedMinSeverity: 4,
		},
		{
			name:                "Medium confidence batch attack",
			confidence:          0.8,
			attackType:         AttackTypeBatchAttack,
			sourceCount:         5,
			expectedMinSeverity: 2,
		},
		{
			name:                "Low confidence speed attack",
			confidence:          0.75,
			attackType:         AttackTypeSpeedAttack,
			sourceCount:         2,
			expectedMinSeverity: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &AdaptiveDetectionResult{
				Confidence:        tt.confidence,
				AttackType:       tt.attackType,
				SourceIdentifiers: make([]string, tt.sourceCount),
			}

			severity := service.calculateSeverity(result)

			if severity < tt.expectedMinSeverity {
				t.Errorf("Expected severity >= %d, got %d", tt.expectedMinSeverity, severity)
			}
		})
	}
}

func TestAdaptiveService_RecommendedAction(t *testing.T) {
	service := NewAdaptiveService()

	tests := []struct {
		severity       int
		expectedAction string
	}{
		{5, "block"},
		{4, "block"},
		{3, "challenge_captcha"},
		{2, "require_verification"},
		{1, "log_only"},
	}

	for _, tt := range tests {
		result := &AdaptiveDetectionResult{Severity: tt.severity}
		action := service.getRecommendedAction(result)

		if action != tt.expectedAction {
			t.Errorf("For severity %d, expected action %s, got %s", tt.severity, tt.expectedAction, action)
		}
	}
}

func TestAdaptiveService_ABTestCreation(t *testing.T) {
	service := NewAdaptiveService()

	variants := []*AdaptiveABTestVariant{
		{ID: "v1", Name: "control", TrafficPercent: 50},
		{ID: "v2", Name: "variant_a", TrafficPercent: 50},
	}

	experiment, err := service.CreateExperiment("test_experiment", variants)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if experiment == nil {
		t.Fatal("Expected experiment to be created")
	}

	if experiment.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", experiment.Status)
	}

	if len(experiment.Variants) != 2 {
		t.Errorf("Expected 2 variants, got %d", len(experiment.Variants))
	}
}

func TestAdaptiveService_ABTestAssignment(t *testing.T) {
	service := NewAdaptiveService()

	variants := []*AdaptiveABTestVariant{
		{ID: "v1", Name: "control", TrafficPercent: 50},
		{ID: "v2", Name: "variant_a", TrafficPercent: 50},
	}

	experiment, err := service.CreateExperiment("test_experiment", variants)
	if err != nil {
		t.Fatalf("Failed to create experiment: %v", err)
	}

	assignmentCounts := make(map[string]int)
	for i := 0; i < 100; i++ {
		variant, err := service.AssignVariant(experiment.ID, fmt.Sprintf("user_%d", i))
		if err != nil {
			t.Fatalf("Failed to assign variant: %v", err)
		}
		assignmentCounts[variant.ID]++
	}

	if len(assignmentCounts) != 2 {
		t.Errorf("Expected assignments to both variants, got counts: %v", assignmentCounts)
	}

	for _, count := range assignmentCounts {
		if count < 20 || count > 80 {
			t.Errorf("Unbalanced assignment: %v", assignmentCounts)
		}
	}
}

func TestAdaptiveService_RecordConversion(t *testing.T) {
	service := NewAdaptiveService()

	variants := []*AdaptiveABTestVariant{
		{ID: "v1", Name: "control", TrafficPercent: 50},
		{ID: "v2", Name: "variant_a", TrafficPercent: 50},
	}

	experiment, err := service.CreateExperiment("test_experiment", variants)
	if err != nil {
		t.Fatalf("Failed to create experiment: %v", err)
	}

	err = service.RecordConversion(experiment.ID, "v1", true)
	if err != nil {
		t.Fatalf("Failed to record conversion: %v", err)
	}

	err = service.RecordConversion(experiment.ID, "v1", false)
	if err != nil {
		t.Fatalf("Failed to record non-conversion: %v", err)
	}

	exp, err := service.AnalyzeExperiment(experiment.ID)
	if err != nil {
		t.Fatalf("Failed to analyze experiment: %v", err)
	}

	result := exp.Results["v1"]
	if result.SampleSize != 2 {
		t.Errorf("Expected sample size 2, got %d", result.SampleSize)
	}
	if result.Conversions != 1 {
		t.Errorf("Expected 1 conversion, got %d", result.Conversions)
	}
}

func TestAdaptiveService_StatisticalConfidence(t *testing.T) {
	service := NewAdaptiveService()

	tests := []struct {
		name    string
		n1, c1  int
		n2, c2  int
		minConf float64
	}{
		{"Large sample with clear difference", 1000, 100, 1000, 150, 95.0},
		{"Small sample with no difference", 10, 5, 10, 5, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := service.calculateStatisticalConfidence(tt.n1, tt.c1, tt.n2, tt.c2)

			if tt.name == "Large sample with clear difference" && confidence < tt.minConf {
				t.Errorf("Expected confidence >= %f, got %f", tt.minConf, confidence)
			}
		})
	}
}

func TestAdaptiveService_LearningModelUpdate(t *testing.T) {
	service := NewAdaptiveService()

	initialVersion := service.learningModel.Version

	update := &AdaptiveModelUpdate{
		Type:        "weight",
		FeatureName: "test_feature",
		OldValue:    0.5,
		NewValue:    0.7,
		Timestamp:   time.Now(),
	}

	err := service.UpdateLearningModel(update)
	if err != nil {
		t.Fatalf("Failed to update learning model: %v", err)
	}

	if service.learningModel.Version <= initialVersion {
		t.Error("Expected model version to be incremented")
	}

	model := service.GetLearningModel()
	if model == nil {
		t.Fatal("Expected learning model to be retrievable")
	}
}

func TestAdaptiveService_UserProfileRetrieval(t *testing.T) {
	service := NewAdaptiveService()

	userID := "profile_test_user"

	profile := service.GetUserProfile(userID)
	if profile != nil {
		t.Error("Expected nil profile for non-existent user")
	}

	req := &AdaptiveVerificationRequest{
		UserID:       userID,
		Success:      true,
		ResponseTime: 1000,
		Metadata:     make(map[string]interface{}),
	}
	service.ProcessVerification(req)

	profile = service.GetUserProfile(userID)
	if profile == nil {
		t.Fatal("Expected profile to be created after verification")
	}

	if profile.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, profile.UserID)
	}
}

func TestAdaptiveService_EventHistory(t *testing.T) {
	service := NewAdaptiveService()

	userID := "history_test_user"

	for i := 0; i < 20; i++ {
		req := &AdaptiveVerificationRequest{
			UserID:       userID,
			Success:      i%2 == 0,
			ResponseTime: float64(1000 + i*100),
			Metadata:     make(map[string]interface{}),
		}
		service.ProcessVerification(req)
	}

	events := service.GetEventHistory(userID, 10)
	if len(events) != 10 {
		t.Errorf("Expected 10 events, got %d", len(events))
	}

	events = service.GetEventHistory(userID, 100)
	if len(events) != 20 {
		t.Errorf("Expected 20 events, got %d", len(events))
	}
}

func TestAdaptiveService_CleanupOldData(t *testing.T) {
	service := NewAdaptiveService()

	userID := "cleanup_user"
	service.eventHistory = append(service.eventHistory, AdaptiveEvent{
		UserID:    userID,
		Success:   true,
		Timestamp: time.Now().Add(-2 * time.Hour),
	})

	service.eventHistory = append(service.eventHistory, AdaptiveEvent{
		UserID:    userID,
		Success:   true,
		Timestamp: time.Now(),
	})

	initialLen := len(service.eventHistory)

	service.CleanupOldData(1 * time.Hour)

	if len(service.eventHistory) >= initialLen {
		t.Error("Expected event history to be cleaned up")
	}
}

func TestAdaptiveService_ThreatIntelligenceSync(t *testing.T) {
	service := NewAdaptiveService()

	for i := 0; i < 150; i++ {
		sig := &AdaptiveAttackSignature{
			Type:       AttackTypeBatchAttack,
			Frequency:  150,
			Confidence: 0.95,
			PatternHash: fmt.Sprintf("hash_%d", i),
			FirstSeen:  time.Now().Add(-1 * time.Hour),
			LastSeen:   time.Now(),
		}
		service.attackSignatures[fmt.Sprintf("sig_%d", i)] = sig
	}

	err := service.SyncThreatIntelligence()
	if err != nil {
		t.Fatalf("Failed to sync threat intelligence: %v", err)
	}

	if len(service.learningModel.AttackPatterns) == 0 {
		t.Error("Expected attack patterns to be synced to learning model")
	}
}

func TestCalculateAdaptiveBotProbability(t *testing.T) {
	tests := []struct {
		name     string
		features *AdaptiveBehaviorFeatures
		minProb  float64
		maxProb  float64
	}{
		{
			name: "Definite bot",
			features: &AdaptiveBehaviorFeatures{
				AvgSpeed:             3000,
				TrajectorySmoothness: 0.99,
				Acceleration:         0.05,
				PathComplexity:       0.05,
				PathSimilarity:       0.1,
				MicroCorrections:     0,
				PauseCount:          0,
			},
			minProb: 0.8,
			maxProb: 1.0,
		},
		{
			name: "Definite human",
			features: &AdaptiveBehaviorFeatures{
				AvgSpeed:             400,
				TrajectorySmoothness: 0.6,
				Acceleration:         0.5,
				PathComplexity:       0.5,
				PathSimilarity:       0.8,
				MicroCorrections:     15,
				PauseCount:          8,
			},
			minProb: 0.0,
			maxProb: 0.4,
		},
		{
			name:     "Nil features",
			features: nil,
			minProb:  0.4,
			maxProb:  0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prob := CalculateAdaptiveBotProbability(tt.features)

			if prob < tt.minProb || prob > tt.maxProb {
				t.Errorf("Expected probability between %f and %f, got %f", tt.minProb, tt.maxProb, prob)
			}
		})
	}
}

func TestGenerateAdaptiveChallenge(t *testing.T) {
	tests := []struct {
		level          AdaptiveDifficultyLevel
		expectedPieces int
	}{
		{AdaptiveDifficultyLevel(DifficultyLevelEasy), 3},
		{AdaptiveDifficultyLevel(DifficultyLevelMedium), 5},
		{AdaptiveDifficultyLevel(DifficultyLevelHard), 7},
		{AdaptiveDifficultyLevel(DifficultyLevelExpert), 9},
	}

	for _, tt := range tests {
		challenge := GenerateAdaptiveChallenge(tt.level)

		if challenge.Difficulty != tt.level {
			t.Errorf("Expected difficulty %d, got %d", tt.level, challenge.Difficulty)
		}

		pieces, ok := challenge.Parameters["puzzle_pieces"].(int)
		if !ok || pieces != tt.expectedPieces {
			t.Errorf("Expected %d puzzle pieces for level %d, got %v", tt.expectedPieces, tt.level, pieces)
		}
	}
}

func TestAdaptiveEnsembleDetector(t *testing.T) {
	ensemble := NewAdaptiveEnsembleDetector()

	detector1 := &mockAdaptiveDetector{name: "detector1", score: 0.8}
	detector2 := &mockAdaptiveDetector{name: "detector2", score: 0.6}

	ensemble.AddDetector(detector1, 0.6)
	ensemble.AddDetector(detector2, 0.4)

	req := &AdaptiveVerificationRequest{
		UserID:       "ensemble_test",
		Success:      true,
		ResponseTime: 1000,
	}

	features := &AdaptiveBehaviorFeatures{
		AvgSpeed:             500,
		TrajectorySmoothness: 0.7,
	}

	score := ensemble.Detect(req, features)

	expectedScore := 0.8*0.6 + 0.6*0.4
	if math.Abs(score-expectedScore) > 0.001 {
		t.Errorf("Expected score %f, got %f", expectedScore, score)
	}
}

type mockAdaptiveDetector struct {
	name  string
	score float64
}

func (m *mockAdaptiveDetector) Detect(req *AdaptiveVerificationRequest, features *AdaptiveBehaviorFeatures) float64 {
	return m.score
}

func (m *mockAdaptiveDetector) GetName() string {
	return m.name
}

func TestAdaptiveService_ConcurrentAccess(t *testing.T) {
	service := NewAdaptiveService()

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			userID := fmt.Sprintf("concurrent_user_%d", id%20)

			for j := 0; j < 10; j++ {
				req := &AdaptiveVerificationRequest{
					UserID:       userID,
					Success:      j%2 == 0,
					ResponseTime: float64(1000 + j*100),
					Metadata:     make(map[string]interface{}),
				}
				service.ProcessVerification(req)
			}
		}(i)
	}

	wg.Wait()

	for i := 0; i < 20; i++ {
		profile := service.GetUserProfile(fmt.Sprintf("concurrent_user_%d", i))
		if profile == nil {
			t.Errorf("Expected profile for user %d", i)
		}
		if profile.TotalAttempts != 50 {
			t.Errorf("Expected 50 attempts for user %d, got %d", i, profile.TotalAttempts)
		}
	}
}

func TestAdaptiveService_ConfidenceCalculation(t *testing.T) {
	service := NewAdaptiveService()

	metrics := &AdaptiveUserMetrics{
		SuccessRate:       100,
		RecentDifficulty:  []float64{2, 2, 2, 2, 2, 2, 2, 2, 2, 2},
		LastResponseTime: 1500,
		AverageDifficulty: 2.0,
	}

	confidence := service.calculateConfidence(metrics)

	if confidence < 0.4 || confidence > 1.0 {
		t.Errorf("Expected confidence between 0.4 and 1.0, got %f", confidence)
	}
}

func TestAdaptiveService_ExtractFeatures(t *testing.T) {
	service := NewAdaptiveService()

	behaviorData := []models.BehaviorData{
		{DataType: "trajectory", Data: `{"x":100,"y":100,"timestamp":0,"event":"move"}`},
		{DataType: "trajectory", Data: `{"x":200,"y":200,"timestamp":100,"event":"move"}`},
		{DataType: "trajectory", Data: `{"x":300,"y":300,"timestamp":200,"event":"move"}`},
		{DataType: "trajectory", Data: `{"x":400,"y":400,"timestamp":300,"event":"click"}`},
	}

	features := service.extractFeatures(behaviorData)

	if features == nil {
		t.Fatal("Expected features to be extracted")
	}

	if features.AvgSpeed == 0 && features.MaxSpeed == 0 {
		t.Error("Expected speed features to be calculated")
	}
}

func TestAdaptiveService_DifficultyMetrics(t *testing.T) {
	service := NewAdaptiveService()

	for level := AdaptiveDifficultyLevel(DifficultyLevelEasy); level <= AdaptiveDifficultyLevel(DifficultyLevelExpert); level++ {
		metrics := service.GetDifficultyMetrics(level)
		if metrics != nil && metrics.Level != level {
			t.Errorf("Expected level %d, got %d", level, metrics.Level)
		}
	}

	allMetrics := service.GetAllDifficultyMetrics()
	if len(allMetrics) != 0 {
		t.Errorf("Expected 0 metrics initially, got %d", len(allMetrics))
	}
}

func TestAdaptiveService_ActiveAttackSignatures(t *testing.T) {
	service := NewAdaptiveService()

	sig1 := &AdaptiveAttackSignature{
		Type:       AttackTypeBatchAttack,
		IsActive:   true,
		PatternHash: "hash1",
		FirstSeen:  time.Now(),
		LastSeen:   time.Now(),
	}
	sig2 := &AdaptiveAttackSignature{
		Type:       AttackTypeSpeedAttack,
		IsActive:   false,
		PatternHash: "hash2",
		FirstSeen:  time.Now(),
		LastSeen:   time.Now().Add(-24 * time.Hour),
	}

	service.attackSignatures["sig1"] = sig1
	service.attackSignatures["sig2"] = sig2

	signatures := service.GetActiveAttackSignatures()

	if len(signatures) != 1 {
		t.Errorf("Expected 1 active signature, got %d", len(signatures))
	}

	if signatures[0].Type != AttackTypeBatchAttack {
		t.Errorf("Expected batch attack signature, got %s", signatures[0].Type)
	}
}

func TestAdaptiveService_NormalCDF(t *testing.T) {
	service := NewAdaptiveService()

	tests := []struct {
		input    float64
		expected float64
	}{
		{0, 0.5},
		{1.96, 0.975},
		{-1.96, 0.025},
		{2.58, 0.995},
	}

	for _, tt := range tests {
		result := service.normalCDF(tt.input)
		if math.Abs(result-tt.expected) > 0.01 {
			t.Errorf("normalCDF(%f) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}

func TestAdaptiveService_SignatureGeneration(t *testing.T) {
	service := NewAdaptiveService()

	result := &AdaptiveDetectionResult{
		AttackType:       AttackTypeBatchAttack,
		Severity:         3,
		SourceIdentifiers: []string{"source1", "source2"},
	}

	sigID := service.generateSignatureID(result)

	if sigID == "" {
		t.Error("Expected non-empty signature ID")
	}

	result2 := &AdaptiveDetectionResult{
		AttackType:       AttackTypeBatchAttack,
		Severity:         3,
		SourceIdentifiers: []string{"source1", "source2"},
	}

	sigID2 := service.generateSignatureID(result2)

	if sigID != sigID2 {
		t.Error("Expected same signature ID for same input")
	}
}

func TestAdaptiveService_ProfileUpdateLocking(t *testing.T) {
	_ = NewAdaptiveService()

	profile := &AdaptiveUserProfile{
		UserID:          "locking_test",
		SuccessHistory:  make([]bool, 0),
		AdaptiveMetrics: AdaptiveUserMetrics{
			RecentDifficulty: make([]float64, 0),
		},
	}

	var wg sync.WaitGroup
	numGoroutines := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				profile.mu.Lock()
				profile.TotalAttempts++
				profile.mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	expectedAttempts := numGoroutines * 10
	if profile.TotalAttempts != expectedAttempts {
		t.Errorf("Expected %d attempts, got %d", expectedAttempts, profile.TotalAttempts)
	}
}

func TestAdaptiveService_MultipleUserProfiles(t *testing.T) {
	service := NewAdaptiveService()

	numUsers := 50

	for i := 0; i < numUsers; i++ {
		userID := fmt.Sprintf("multi_user_%d", i)
		req := &AdaptiveVerificationRequest{
			UserID:       userID,
			Success:      i%2 == 0,
			ResponseTime: float64(1000 + i*100),
			Metadata:     make(map[string]interface{}),
		}
		service.ProcessVerification(req)
	}

	if len(service.userProfiles) != numUsers {
		t.Errorf("Expected %d profiles, got %d", numUsers, len(service.userProfiles))
	}

	for i := 0; i < numUsers; i++ {
		userID := fmt.Sprintf("multi_user_%d", i)
		profile := service.GetUserProfile(userID)

		if profile == nil {
			t.Errorf("Expected profile for user %s", userID)
		}

		if profile.TotalAttempts != 1 {
			t.Errorf("Expected 1 attempt for user %s, got %d", userID, profile.TotalAttempts)
		}
	}
}

func TestAdaptiveService_ConfigValidation(t *testing.T) {
	service := NewAdaptiveService()

	if service.config.DifficultyStep <= 0 {
		t.Error("Expected difficulty step > 0")
	}

	if service.config.AdjustmentWindow <= 0 {
		t.Error("Expected adjustment window > 0")
	}

	if service.config.CooldownPeriod <= 0 {
		t.Error("Expected cooldown period > 0")
	}

	if service.config.SuccessRateTarget <= 0 || service.config.SuccessRateTarget >= 1 {
		t.Error("Expected success rate target between 0 and 1")
	}
}

func TestAdaptiveService_MetricsAccuracy(t *testing.T) {
	service := NewAdaptiveService()

	profile := &AdaptiveUserProfile{
		UserID:           "accuracy_test",
		DifficultyLevel:  2,
		SuccessHistory:   make([]bool, 0),
		AdaptiveMetrics: AdaptiveUserMetrics{
			RecentDifficulty: make([]float64, 0),
		},
	}

	successes := 7
	total := 10

	for i := 0; i < total; i++ {
		success := i < successes
		profile.DifficultyLevel = 2
		service.updateMetrics(profile, success, 1000)
	}

	expectedRate := float64(successes) / float64(total)
	if math.Abs(profile.AdaptiveMetrics.SuccessRate-expectedRate) > 0.001 {
		t.Errorf("Expected success rate %f, got %f", expectedRate, profile.AdaptiveMetrics.SuccessRate)
	}
}

func TestAdaptiveService_ProfileInitialization(t *testing.T) {
	service := NewAdaptiveService()

	profile := service.getOrCreateUserProfile("init_test")

	if profile == nil {
		t.Fatal("Expected profile to be created")
	}

	if profile.SuccessHistory == nil {
		t.Error("Expected success history to be initialized")
	}

	if profile.SessionData == nil {
		t.Error("Expected session data to be initialized")
	}

	if profile.AdaptiveMetrics.RecentDifficulty == nil {
		t.Error("Expected recent difficulty to be initialized")
	}

	if profile.AdaptiveMetrics.AbilityEstimate != 0.5 {
		t.Errorf("Expected default ability estimate 0.5, got %f", profile.AdaptiveMetrics.AbilityEstimate)
	}

	if profile.DifficultyLevel != AdaptiveDifficultyLevel(DifficultyLevelMedium) {
		t.Errorf("Expected default difficulty Medium, got %d", profile.DifficultyLevel)
	}
}

func TestAdaptiveService_StreakTracking(t *testing.T) {
	service := NewAdaptiveService()

	profile := &AdaptiveUserProfile{
		UserID:          "streak_test",
		SuccessHistory:  make([]bool, 0),
		AdaptiveMetrics: AdaptiveUserMetrics{
			RecentDifficulty: make([]float64, 0),
		},
	}

	for i := 0; i < 5; i++ {
		metrics := service.updateMetrics(profile, true, 1000)
		if metrics.StreakCount != i+1 {
			t.Errorf("Expected streak %d, got %d", i+1, metrics.StreakCount)
		}
	}

	if profile.AdaptiveMetrics.MaxStreak < 5 {
		t.Errorf("Expected max streak >= 5, got %d", profile.AdaptiveMetrics.MaxStreak)
	}

	metrics := service.updateMetrics(profile, false, 1000)

	if metrics.StreakCount != 0 {
		t.Errorf("Expected streak 0 after failure, got %d", metrics.StreakCount)
	}
}

func TestAdaptiveService_LearningModelLocking(t *testing.T) {
	service := NewAdaptiveService()

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			features := &AdaptiveBehaviorFeatures{
				AvgSpeed:             500 + float64(i*10),
				TrajectorySmoothness: 0.7,
			}
			service.updateFeatureStats(features)

			model := service.GetLearningModel()
			if model == nil {
				t.Error("Expected model to be retrievable")
			}
		}()
	}

	wg.Wait()

	if len(service.learningModel.FeatureStats) == 0 {
		t.Error("Expected feature stats to be updated")
	}
}

func TestAdaptiveService_FeatureExtractionEdgeCases(t *testing.T) {
	service := NewAdaptiveService()

	tests := []struct {
		name         string
		behaviorData []models.BehaviorData
		expectEmpty  bool
	}{
		{
			name:         "Empty behavior data",
			behaviorData: []models.BehaviorData{},
			expectEmpty:  true,
		},
		{
			name: "Single point",
			behaviorData: []models.BehaviorData{
				{DataType: "trajectory", Data: `{"x":100,"y":100,"timestamp":0}`},
			},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := service.extractFeatures(tt.behaviorData)

			if tt.expectEmpty && features.AvgSpeed != 0 {
				t.Error("Expected empty features for edge case")
			}
		})
	}
}

func TestAdaptiveService_ConfidenceCalculationEdgeCases(t *testing.T) {
	service := NewAdaptiveService()

	tests := []struct {
		name    string
		metrics *AdaptiveUserMetrics
	}{
		{
			name: "Very few attempts",
			metrics: &AdaptiveUserMetrics{
				SuccessRate:       1,
				RecentDifficulty:  []float64{2},
			},
		},
		{
			name: "No recent difficulty",
			metrics: &AdaptiveUserMetrics{
				SuccessRate:       100,
				RecentDifficulty:  []float64{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := service.calculateConfidence(tt.metrics)

			if confidence < 0 || confidence > 1 {
				t.Errorf("Expected confidence between 0 and 1, got %f", confidence)
			}
		})
	}
}

func TestAdaptiveService_AdaptiveEventLimit(t *testing.T) {
	service := NewAdaptiveService()

	maxEvents := 10000

	for i := 0; i < maxEvents+100; i++ {
		req := &AdaptiveVerificationRequest{
			UserID:       fmt.Sprintf("user_%d", i%100),
			Success:      i%2 == 0,
			ResponseTime: float64(1000 + i),
			Metadata:     make(map[string]interface{}),
		}
		service.ProcessVerification(req)
	}

	if len(service.eventHistory) > maxEvents {
		t.Errorf("Expected event history to be limited to %d, got %d", maxEvents, len(service.eventHistory))
	}

	if len(service.eventHistory) < maxEvents/2 {
		t.Errorf("Expected event history to have at least %d events, got %d", maxEvents/2, len(service.eventHistory))
	}
}

func TestAdaptiveService_LearningRateApplication(t *testing.T) {
	service := NewAdaptiveService()

	initialAbility := 0.5
	metrics := &AdaptiveUserMetrics{
		AbilityEstimate:    initialAbility,
		AverageDifficulty: 2.0,
		LastResponseTime:   2000,
		RecentDifficulty:   make([]float64, 0),
	}

	abilityValues := make([]float64, 0, 100)

	for i := 0; i < 100; i++ {
		service.updateAbilityEstimate(metrics, true, 1000)
		abilityValues = append(abilityValues, metrics.AbilityEstimate)
	}

	sort.Slice(abilityValues, func(i, j int) bool {
		return abilityValues[i] < abilityValues[j]
	})

	if abilityValues[len(abilityValues)-1] <= initialAbility {
		t.Error("Expected ability to generally increase with successful attempts")
	}
}

func TestAdaptiveService_ProfileDataIsolation(t *testing.T) {
	service := NewAdaptiveService()

	users := []string{"user_a", "user_b", "user_c"}

	for _, userID := range users {
		for i := 0; i < 3; i++ {
			req := &AdaptiveVerificationRequest{
				UserID:       userID,
				Success:      true,
				ResponseTime: 1000,
				Metadata:     make(map[string]interface{}),
			}
			service.ProcessVerification(req)
		}
	}

	userAProfile := service.GetUserProfile("user_a")
	userBProfile := service.GetUserProfile("user_b")

	if userAProfile == nil || userBProfile == nil {
		t.Error("Expected both profiles to exist")
	}

	if len(service.userProfiles) != 3 {
		t.Errorf("Expected 3 user profiles, got %d", len(service.userProfiles))
	}
}

func TestAdaptiveService_ResponseTimeSmoothing(t *testing.T) {
	service := NewAdaptiveService()

	profile := &AdaptiveUserProfile{
		UserID:          "smoothing_test",
		SuccessHistory:  make([]bool, 0),
		AdaptiveMetrics: AdaptiveUserMetrics{
			RecentDifficulty: make([]float64, 0),
		},
	}

	metrics := service.updateMetrics(profile, true, 1000)

	if metrics.LastResponseTime != 1000 {
		t.Errorf("Expected first response time 1000, got %f", metrics.LastResponseTime)
	}

	metrics = service.updateMetrics(profile, true, 2000)

	expectedTime := 1000*0.9 + 2000*0.1
	if math.Abs(metrics.LastResponseTime-expectedTime) > 0.1 {
		t.Errorf("Expected smoothed time ~%f, got %f", expectedTime, metrics.LastResponseTime)
	}
}

func TestAdaptiveService_FeatureStatsUpdate(t *testing.T) {
	service := NewAdaptiveService()

	features1 := &AdaptiveBehaviorFeatures{
		AvgSpeed:             500,
		TrajectorySmoothness: 0.7,
		PathComplexity:       0.4,
	}

	features2 := &AdaptiveBehaviorFeatures{
		AvgSpeed:             600,
		TrajectorySmoothness: 0.75,
		PathComplexity:       0.45,
	}

	service.updateFeatureStats(features1)
	service.updateFeatureStats(features2)

	stats, exists := service.learningModel.FeatureStats["avg_speed"]
	if !exists {
		t.Fatal("Expected avg_speed stats to exist")
	}

	if stats.Count != 2 {
		t.Errorf("Expected count 2, got %d", stats.Count)
	}

	expectedMean := 550.0
	if math.Abs(stats.Mean-expectedMean) > 0.1 {
		t.Errorf("Expected mean ~%f, got %f", expectedMean, stats.Mean)
	}
}

func TestAdaptiveService_FindMatchingSignature(t *testing.T) {
	service := NewAdaptiveService()

	sig := &AdaptiveAttackSignature{
		Type:       AttackTypeBatchAttack,
		Confidence: 0.9,
		IsActive:   true,
		PatternHash: "test_hash",
	}

	service.attackSignatures["test_hash"] = sig

	result := &AdaptiveDetectionResult{
		AttackType: AttackTypeBatchAttack,
		Confidence: 0.85,
	}

	matched := service.findMatchingSignature(result)

	if matched == nil {
		t.Fatal("Expected to find matching signature")
	}

	if matched.PatternHash != "test_hash" {
		t.Errorf("Expected pattern hash 'test_hash', got '%s'", matched.PatternHash)
	}
}

func TestAdaptiveService_RecordAttackPattern(t *testing.T) {
	service := NewAdaptiveService()

	result := &AdaptiveDetectionResult{
		IsAttack:          true,
		AttackType:       AttackTypeSpeedAttack,
		Confidence:       0.92,
		SourceIdentifiers: []string{"source1", "source2"},
		Indicators: map[string]float64{
			"speed_score": 0.9,
		},
	}

	service.recordAttackPattern(result)

	if len(service.attackSignatures) != 1 {
		t.Errorf("Expected 1 attack signature, got %d", len(service.attackSignatures))
	}

	for _, sig := range service.attackSignatures {
		if sig.Type != AttackTypeSpeedAttack {
			t.Errorf("Expected type SpeedAttack, got %s", sig.Type)
		}

		if sig.Frequency != 1 {
			t.Errorf("Expected frequency 1, got %d", sig.Frequency)
		}

		if !sig.IsActive {
			t.Error("Expected signature to be active")
		}
	}
}

func TestAdaptiveService_AdaptiveEventRecording(t *testing.T) {
	service := NewAdaptiveService()

	userID := "event_recording_test"

	for i := 0; i < 5; i++ {
		req := &AdaptiveVerificationRequest{
			UserID:       userID,
			Success:      i%2 == 0,
			ResponseTime: float64(1000 + i*100),
			Metadata:     map[string]interface{}{"test": "data"},
		}
		service.ProcessVerification(req)
	}

	if len(service.eventHistory) != 5 {
		t.Errorf("Expected 5 events, got %d", len(service.eventHistory))
	}

	for _, event := range service.eventHistory {
		if event.UserID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, event.UserID)
		}

		if event.Timestamp.IsZero() {
			t.Error("Expected event timestamp to be set")
		}
	}
}

func TestAdaptiveService_MultipleExperiments(t *testing.T) {
	service := NewAdaptiveService()

	variants1 := []*AdaptiveABTestVariant{
		{ID: "v1_1", Name: "control", TrafficPercent: 50},
		{ID: "v1_2", Name: "variant_a", TrafficPercent: 50},
	}

	variants2 := []*AdaptiveABTestVariant{
		{ID: "v2_1", Name: "control", TrafficPercent: 33},
		{ID: "v2_2", Name: "variant_b", TrafficPercent: 33},
		{ID: "v2_3", Name: "variant_c", TrafficPercent: 34},
	}

	exp1, err := service.CreateExperiment("experiment_1", variants1)
	if err != nil {
		t.Fatalf("Failed to create experiment 1: %v", err)
	}

	exp2, err := service.CreateExperiment("experiment_2", variants2)
	if err != nil {
		t.Fatalf("Failed to create experiment 2: %v", err)
	}

	if exp1.ID == exp2.ID {
		t.Error("Expected different experiment IDs")
	}

	v1, _ := service.AssignVariant(exp1.ID, "user_1")
	v2, _ := service.AssignVariant(exp2.ID, "user_1")

	if v1.ID == v2.ID {
		t.Error("Expected different variant assignments")
	}
}

func TestAdaptiveService_ExperimentConversionTracking(t *testing.T) {
	service := NewAdaptiveService()

	variants := []*AdaptiveABTestVariant{
		{ID: "v1", Name: "control", TrafficPercent: 50},
		{ID: "v2", Name: "variant_a", TrafficPercent: 50},
	}

	exp, _ := service.CreateExperiment("conversion_test", variants)

	for i := 0; i < 100; i++ {
		success := i < 60
		service.RecordConversion(exp.ID, "v1", success)
		service.RecordConversion(exp.ID, "v2", i%2 == 0)
	}

	exp, _ = service.AnalyzeExperiment(exp.ID)

	result1 := exp.Results["v1"]
	result2 := exp.Results["v2"]

	if result1.SampleSize != 100 {
		t.Errorf("Expected sample size 100 for v1, got %d", result1.SampleSize)
	}

	if result1.Conversions != 60 {
		t.Errorf("Expected 60 conversions for v1, got %d", result1.Conversions)
	}

	if result1.ConversionRate != 0.6 {
		t.Errorf("Expected conversion rate 0.6 for v1, got %f", result1.ConversionRate)
	}

	if result2.Conversions != 50 {
		t.Errorf("Expected 50 conversions for v2, got %d", result2.Conversions)
	}

	if result2.Improvement == 0 {
		t.Error("Expected improvement to be calculated")
	}
}

func TestAdaptiveService_UserProfileHistoryLimit(t *testing.T) {
	service := NewAdaptiveService()

	profile := &AdaptiveUserProfile{
		UserID:          "history_limit_test",
		SuccessHistory:  make([]bool, 0),
		AdaptiveMetrics: AdaptiveUserMetrics{
			RecentDifficulty: make([]float64, 0),
		},
	}

	service.config.AdjustmentWindow = 10

	for i := 0; i < 20; i++ {
		service.updateMetrics(profile, i%2 == 0, float64(1000+i*100))
	}

	if len(profile.SuccessHistory) > 10 {
		t.Errorf("Expected success history to be limited to %d, got %d", 10, len(profile.SuccessHistory))
	}

	if len(profile.AdaptiveMetrics.RecentDifficulty) > 10 {
		t.Errorf("Expected recent difficulty to be limited to %d, got %d", 10, len(profile.AdaptiveMetrics.RecentDifficulty))
	}
}

func TestAdaptiveService_DifficultyLevelConstants(t *testing.T) {
	if DifficultyLevelEasy != 1 {
		t.Errorf("Expected DifficultyLevelEasy = 1, got %d", DifficultyLevelEasy)
	}
	if DifficultyLevelMedium != 2 {
		t.Errorf("Expected DifficultyLevelMedium = 2, got %d", DifficultyLevelMedium)
	}
	if DifficultyLevelHard != 3 {
		t.Errorf("Expected DifficultyLevelHard = 3, got %d", DifficultyLevelHard)
	}
	if DifficultyLevelExpert != 4 {
		t.Errorf("Expected DifficultyLevelExpert = 4, got %d", DifficultyLevelExpert)
	}
}

func TestAdaptiveService_AttackTypeConstants(t *testing.T) {
	expectedTypes := []string{
		AttackTypeNone,
		AttackTypeBruteForce,
		AttackTypeBatchAttack,
		AttackTypeDistributed,
		AttackTypeReplayAttack,
		AttackTypePatternAttack,
		AttackTypeSpeedAttack,
		AttackTypeCoordinated,
	}

	if len(expectedTypes) != 8 {
		t.Errorf("Expected 8 attack types, got %d", len(expectedTypes))
	}
}

func TestAdaptiveService_ChallengeGeneration(t *testing.T) {
	tests := []struct {
		level      AdaptiveDifficultyLevel
		expectType string
	}{
		{AdaptiveDifficultyLevel(DifficultyLevelEasy), "standard"},
		{AdaptiveDifficultyLevel(DifficultyLevelMedium), "standard"},
		{AdaptiveDifficultyLevel(DifficultyLevelHard), "standard"},
		{AdaptiveDifficultyLevel(DifficultyLevelExpert), "standard"},
	}

	for _, tt := range tests {
		challenge := GenerateAdaptiveChallenge(tt.level)

		if challenge.Type != tt.expectType {
			t.Errorf("Expected type %s for level %d, got %s", tt.expectType, tt.level, challenge.Type)
		}

		if challenge.Difficulty != tt.level {
			t.Errorf("Expected difficulty %d, got %d", tt.level, challenge.Difficulty)
		}

		if challenge.TimeLimit == 0 {
			t.Error("Expected time limit to be set")
		}

		if challenge.MaxAttempts == 0 {
			t.Error("Expected max attempts to be set")
		}
	}
}

func TestAdaptiveService_GetChallengeType(t *testing.T) {
	service := NewAdaptiveService()

	tests := []struct {
		attackType    string
		expectedType string
	}{
		{AttackTypeSpeedAttack, "behavior_analysis"},
		{AttackTypePatternAttack, "behavior_analysis"},
		{AttackTypeBatchAttack, "captcha"},
		{AttackTypeBruteForce, "captcha"},
		{AttackTypeDistributed, "advanced_captcha"},
		{AttackTypeCoordinated, "advanced_captcha"},
		{AttackTypeReplayAttack, "standard_captcha"},
		{"unknown", "standard_captcha"},
	}

	for _, tt := range tests {
		t.Run(tt.attackType, func(t *testing.T) {
			result := &AdaptiveDetectionResult{
				IsAttack:   true,
				AttackType: tt.attackType,
			}

			challengeType := service.getChallengeType(result)

			if challengeType != tt.expectedType {
				t.Errorf("For attack type %s, expected %s, got %s", tt.attackType, tt.expectedType, challengeType)
			}
		})
	}
}

func TestAdaptiveService_ExtractSourceIdentifiers(t *testing.T) {
	service := NewAdaptiveService()

	req := &AdaptiveVerificationRequest{
		UserID:    "source_test_user",
		SessionID: "session_123",
	}

	identifiers := service.extractSourceIdentifiers(req)

	if len(identifiers) != 2 {
		t.Errorf("Expected 2 identifiers, got %d", len(identifiers))
	}

	foundUserID := false
	foundSessionID := false
	for _, id := range identifiers {
		if id == "source_test_user" {
			foundUserID = true
		}
		if id == "session_123" {
			foundSessionID = true
		}
	}

	if !foundUserID {
		t.Error("Expected user ID in identifiers")
	}
	if !foundSessionID {
		t.Error("Expected session ID in identifiers")
	}
}

func TestAdaptiveService_ExtractAffectedResources(t *testing.T) {
	service := NewAdaptiveService()

	req := &AdaptiveVerificationRequest{
		UserID: "resource_test",
	}

	resources := service.extractAffectedResources(req)

	if len(resources) == 0 {
		t.Error("Expected affected resources to be returned")
	}
}

func TestAdaptiveService_StatisticalDetector(t *testing.T) {
	detector := NewAdaptiveStatisticalDetector("test_detector", 0.7)

	if detector.name != "test_detector" {
		t.Errorf("Expected name 'test_detector', got '%s'", detector.name)
	}

	if detector.threshold != 0.7 {
		t.Errorf("Expected threshold 0.7, got %f", detector.threshold)
	}

	req := &AdaptiveVerificationRequest{
		UserID:       "stat_test",
		Success:      false,
		ResponseTime: 5000,
	}

	features := &AdaptiveBehaviorFeatures{}

	score := detector.Detect(req, features)

	if score < 0.3 {
		t.Errorf("Expected score >= 0.3 for extreme response time, got %f", score)
	}
}

func TestAdaptiveService_EnsembleWithMultipleDetectors(t *testing.T) {
	ensemble := NewAdaptiveEnsembleDetector()

	detectors := []struct {
		name   string
		score  float64
		weight float64
	}{
		{"detector1", 0.9, 0.4},
		{"detector2", 0.7, 0.3},
		{"detector3", 0.5, 0.3},
	}

	for _, d := range detectors {
		ensemble.AddDetector(&mockAdaptiveDetector{name: d.name, score: d.score}, d.weight)
	}

	req := &AdaptiveVerificationRequest{
		UserID:       "ensemble_multi_test",
		Success:      false,
		ResponseTime: 100,
	}

	features := &AdaptiveBehaviorFeatures{
		AvgSpeed:             3000,
		TrajectorySmoothness: 0.98,
	}

	score := ensemble.Detect(req, features)

	expectedScore := 0.9*0.4 + 0.7*0.3 + 0.5*0.3
	if math.Abs(score-expectedScore) > 0.001 {
		t.Errorf("Expected score %f, got %f", expectedScore, score)
	}
}

func TestAdaptiveService_ModelVersioning(t *testing.T) {
	service := NewAdaptiveService()

	initialVersion := service.learningModel.Version

	updates := []*AdaptiveModelUpdate{
		{Type: "weight", FeatureName: "feature1", NewValue: 0.5},
		{Type: "threshold", FeatureName: "threshold1", NewValue: 0.7},
		{Type: "feature_stat", FeatureName: "avg_speed", NewValue: 500},
	}

	for _, update := range updates {
		service.UpdateLearningModel(update)
	}

	if service.learningModel.Version <= initialVersion {
		t.Error("Expected model version to be incremented after updates")
	}

	expectedVersion := initialVersion + len(updates)
	if service.learningModel.Version != expectedVersion {
		t.Errorf("Expected version %d, got %d", expectedVersion, service.learningModel.Version)
	}
}

func TestAdaptiveService_DifficultyBounds(t *testing.T) {
	service := NewAdaptiveService()

	profile := &AdaptiveUserProfile{
		UserID:           "bounds_test",
		DifficultyLevel:   AdaptiveDifficultyLevel(DifficultyLevelMedium),
		AdaptiveMetrics: AdaptiveUserMetrics{
			AbilityEstimate: 0.5,
			SuccessRate:    0.5,
		},
	}

	recommended := service.calculateRecommendedDifficulty(profile)

	if recommended < AdaptiveDifficultyLevel(DifficultyLevelEasy) || recommended > AdaptiveDifficultyLevel(DifficultyLevelExpert) {
		t.Errorf("Recommended difficulty should be between Easy and Expert, got %d", recommended)
	}

	profile.AdaptiveMetrics.AbilityEstimate = 0.0
	recommended = service.calculateRecommendedDifficulty(profile)

	if recommended != AdaptiveDifficultyLevel(DifficultyLevelEasy) {
		t.Errorf("Expected Easy for very low ability, got %d", recommended)
	}

	profile.AdaptiveMetrics.AbilityEstimate = 1.0
	profile.AdaptiveMetrics.SuccessRate = 1.0
	recommended = service.calculateRecommendedDifficulty(profile)

	if recommended < AdaptiveDifficultyLevel(DifficultyLevelMedium) || recommended > AdaptiveDifficultyLevel(DifficultyLevelHard) {
		t.Errorf("Expected Medium or Hard for very high ability, got %d", recommended)
	}
}

func TestAdaptiveService_CooldownPeriod(t *testing.T) {
	service := NewAdaptiveService()

	profile := &AdaptiveUserProfile{
		UserID:          "cooldown_test",
		DifficultyLevel: AdaptiveDifficultyLevel(DifficultyLevelMedium),
		LastAdjustment:  time.Now(),
	}

	recommended := AdaptiveDifficultyLevel(DifficultyLevelHard)

	adjusted := service.adjustDifficulty(profile, recommended, nil)

	if adjusted != AdaptiveDifficultyLevel(DifficultyLevelMedium) {
		t.Error("Expected difficulty to remain unchanged during cooldown period")
	}

	time.Sleep(35 * time.Second)

	adjusted = service.adjustDifficulty(profile, recommended, nil)

	if adjusted != AdaptiveDifficultyLevel(DifficultyLevelHard) {
		t.Errorf("Expected difficulty to change after cooldown, got %d", adjusted)
	}
}

func TestAdaptiveService_MetricsUpdate(t *testing.T) {
	service := NewAdaptiveService()

	profile := &AdaptiveUserProfile{
		UserID:           "metrics_test",
		DifficultyLevel:   2,
		SuccessHistory:    make([]bool, 0),
		AdaptiveMetrics: AdaptiveUserMetrics{
			RecentDifficulty: make([]float64, 0),
		},
	}

	profile.DifficultyLevel = 2
	metrics := service.updateMetrics(profile, true, 2000)

	if metrics.SuccessRate != 1.0 {
		t.Errorf("Expected success rate 1.0, got %f", metrics.SuccessRate)
	}

	if metrics.StreakCount != 1 {
		t.Errorf("Expected streak count 1, got %d", metrics.StreakCount)
	}

	profile.DifficultyLevel = 2
	metrics = service.updateMetrics(profile, false, 3000)

	if metrics.SuccessRate != 0.5 {
		t.Errorf("Expected success rate 0.5, got %f", metrics.SuccessRate)
	}

	if metrics.StreakCount != 0 {
		t.Errorf("Expected streak count 0 after failure, got %d", metrics.StreakCount)
	}
}

func TestAdaptiveService_AbilityEstimateBounds(t *testing.T) {
	service := NewAdaptiveService()

	metrics := &AdaptiveUserMetrics{
		AbilityEstimate:    0.5,
		AverageDifficulty: 2.0,
		LastResponseTime:  2000,
	}

	service.updateAbilityEstimate(metrics, true, 1000)

	if metrics.AbilityEstimate < 0 || metrics.AbilityEstimate > 1 {
		t.Errorf("Ability estimate should be between 0 and 1, got %f", metrics.AbilityEstimate)
	}

	metrics.AbilityEstimate = 0.9
	metrics.AverageDifficulty = 4.0
	service.updateAbilityEstimate(metrics, false, 100)

	if metrics.AbilityEstimate < 0 || metrics.AbilityEstimate > 1 {
		t.Errorf("Ability estimate should be between 0 and 1, got %f", metrics.AbilityEstimate)
	}
}

func TestProcessVerificationWithBehaviorData(t *testing.T) {
	service := NewAdaptiveService()

	behaviorData := []models.BehaviorData{}
	for i := 0; i < 20; i++ {
		x := 100 + i*10
		y := 100 + i*10
		timestamp := int64(i * 50)
		dp := BehaviorDataPoint{
			X:         x,
			Y:         y,
			Timestamp: timestamp,
			Event:     "move",
		}
		data, _ := json.Marshal(dp)
		behaviorData = append(behaviorData, models.BehaviorData{
			DataType: "trajectory",
			Data:     string(data),
		})
	}

	req := &AdaptiveVerificationRequest{
		UserID:       "behavior_test_user",
		BehaviorData: behaviorData,
		Success:      true,
		ResponseTime: 1500,
		Metadata:     make(map[string]interface{}),
	}

	resp := service.ProcessVerification(req)

	if resp == nil {
		t.Fatal("Expected response to not be nil")
	}

	profile := service.GetUserProfile("behavior_test_user")
	if profile == nil {
		t.Fatal("Expected user profile to be created")
	}

	if profile.BehaviorFeatures == nil {
		t.Error("Expected behavior features to be extracted")
	}
}

func BenchmarkProcessVerification(b *testing.B) {
	service := NewAdaptiveService()

	req := &AdaptiveVerificationRequest{
		UserID:       "benchmark_user",
		Success:      true,
		ResponseTime: 1500,
		Metadata:     make(map[string]interface{}),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.UserID = fmt.Sprintf("user_%d", i)
		service.ProcessVerification(req)
	}
}

func BenchmarkAttackDetection(b *testing.B) {
	service := NewAdaptiveService()

	for i := 0; i < 50; i++ {
		service.eventHistory = append(service.eventHistory, AdaptiveEvent{
			UserID:    fmt.Sprintf("user_%d", i%10),
			Success:   false,
			Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
		})
	}

	features := &AdaptiveBehaviorFeatures{
		AvgSpeed:             3000,
		TrajectorySmoothness: 0.99,
		PathComplexity:       0.05,
		MicroCorrections:     0,
		PauseCount:          0,
		BotScore:             0.9,
	}

	req := &AdaptiveVerificationRequest{
		UserID:       "attack_user",
		Success:      false,
		ResponseTime: 50,
		Metadata:     make(map[string]interface{}),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.detectAttack(req, features)
	}
}

func BenchmarkLearningModelUpdate(b *testing.B) {
	service := NewAdaptiveService()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		features := &AdaptiveBehaviorFeatures{
			AvgSpeed:             500 + float64(i),
			TrajectorySmoothness: 0.7,
		}
		service.updateFeatureStats(features)
	}
}

func BenchmarkCalculateAdaptiveBotProbability(b *testing.B) {
	features := &AdaptiveBehaviorFeatures{
		AvgSpeed:             750,
		TrajectorySmoothness: 0.85,
		Acceleration:         0.3,
		PathComplexity:       0.35,
		PathSimilarity:       0.5,
		MicroCorrections:     5,
		PauseCount:          3,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateAdaptiveBotProbability(features)
	}
}

func BenchmarkEnsembleDetection(b *testing.B) {
	ensemble := NewAdaptiveEnsembleDetector()

	for i := 0; i < 5; i++ {
		ensemble.AddDetector(&mockAdaptiveDetector{
			name:  fmt.Sprintf("detector_%d", i),
			score: 0.5 + float64(i)*0.1,
		}, 0.2)
	}

	req := &AdaptiveVerificationRequest{
		UserID:       "ensemble_bench",
		Success:      false,
		ResponseTime: 500,
	}

	features := &AdaptiveBehaviorFeatures{
		AvgSpeed:             1000,
		TrajectorySmoothness: 0.8,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ensemble.Detect(req, features)
	}
}

func TestIntegration_CompleteFlow(t *testing.T) {
	service := NewAdaptiveService()

	variants := []*AdaptiveABTestVariant{
		{ID: "v1", Name: "control", TrafficPercent: 50},
		{ID: "v2", Name: "variant_a", TrafficPercent: 50},
	}

	exp, err := service.CreateExperiment("integration_test", variants)
	if err != nil {
		t.Fatalf("Failed to create experiment: %v", err)
	}

	for i := 0; i < 1000; i++ {
		userID := fmt.Sprintf("user_%d", i)

		variant, _ := service.AssignVariant(exp.ID, userID)
		service.RecordConversion(exp.ID, variant.ID, i%2 == 0)

		req := &AdaptiveVerificationRequest{
			UserID:       userID,
			Success:      i%2 == 0,
			ResponseTime: float64(1000 + (i%10)*100),
			Metadata:     map[string]interface{}{"variant": variant.ID},
		}

		resp := service.ProcessVerification(req)

		if resp == nil {
			t.Fatal("Expected response from process verification")
		}

		if resp.AdjustedDifficulty == 0 {
			t.Error("Expected difficulty to be set")
		}
	}

	exp, _ = service.AnalyzeExperiment(exp.ID)

	for _, result := range exp.Results {
		if result.SampleSize != 500 {
			t.Errorf("Expected 500 samples per variant, got %d", result.SampleSize)
		}
	}

	signatures := service.GetActiveAttackSignatures()
	if len(signatures) != 0 {
		t.Logf("Detected %d attack signatures", len(signatures))
	}

	model := service.GetLearningModel()
	if model.Version < 1 {
		t.Error("Expected model version to be incremented")
	}
}

func TestAdaptiveService_AttackDetectionAccuracy(t *testing.T) {
	service := NewAdaptiveService()

	truePositives := 0
	falsePositives := 0
	trueNegatives := 0
	falseNegatives := 0

	for i := 0; i < 100; i++ {
		service.eventHistory = make([]AdaptiveEvent, 0)
		
		isAttack := i < 50
		userID := fmt.Sprintf("user_%d", i)
		
		if isAttack {
			for j := 0; j < 60; j++ {
				service.eventHistory = append(service.eventHistory, AdaptiveEvent{
					UserID:    userID,
					Success:   false,
					Timestamp: time.Now().Add(-time.Duration(j/10) * time.Minute),
				})
			}
		} else {
			for j := 0; j < 10; j++ {
				service.eventHistory = append(service.eventHistory, AdaptiveEvent{
					UserID:    userID,
					Success:   true,
					Timestamp: time.Now().Add(-time.Duration(j) * time.Hour),
				})
			}
		}
		
		req := &AdaptiveVerificationRequest{
			UserID:       userID,
			Success:      true,
			ResponseTime: 2000,
			Metadata:     make(map[string]interface{}),
		}
		
		if isAttack {
			score := service.detectBatchAttack(req)
			if score > 0.3 {
				truePositives++
			} else {
				falseNegatives++
			}
		} else {
			score := service.detectBatchAttack(req)
			if score > 0.3 {
				falsePositives++
			} else {
				trueNegatives++
			}
		}
	}
	
	accuracy := float64(truePositives+trueNegatives) / float64(truePositives+trueNegatives+falsePositives+falseNegatives)
	
	t.Logf("Attack Detection Accuracy Test Results:")
	t.Logf("True Positives: %d", truePositives)
	t.Logf("True Negatives: %d", trueNegatives)
	t.Logf("False Positives: %d", falsePositives)
	t.Logf("False Negatives: %d", falseNegatives)
	t.Logf("Accuracy: %.2f%%", accuracy*100)
	
	if accuracy < 0.99 {
		t.Errorf("Attack detection accuracy %.2f%% is below 99%% requirement", accuracy*100)
	}
}
