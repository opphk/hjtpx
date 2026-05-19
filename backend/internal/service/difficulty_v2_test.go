package service

import (
	"math"
	"testing"
	"time"
)

func TestNewAdaptiveDifficultyServiceV2(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()

	if service == nil {
		t.Fatal("Service should not be nil")
	}

	if service.profiles == nil {
		t.Error("Profiles map should be initialized")
	}

	if service.config == nil {
		t.Error("Config should be initialized")
	}

	if service.timeoutManager == nil {
		t.Error("TimeoutManager should be initialized")
	}

	if service.retryManager == nil {
		t.Error("RetryManager should be initialized")
	}

	if service.riskCalculator == nil {
		t.Error("RiskCalculator should be initialized")
	}
}

func TestGetOrCreateProfileV2(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()

	userID := "test_user_123"

	profile1 := service.GetOrCreateProfileV2(userID)
	if profile1 == nil {
		t.Fatal("Profile should not be nil")
	}

	if profile1.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, profile1.UserID)
	}

	if profile1.CompositeRisk == nil {
		t.Error("CompositeRisk should be initialized")
	}

	if profile1.SessionMetrics == nil {
		t.Error("SessionMetrics should be initialized")
	}

	profile2 := service.GetOrCreateProfileV2(userID)
	if profile1 != profile2 {
		t.Error("Should return the same profile for same user")
	}
}

func TestCalculateMultiDimensionalRiskScore(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_risk"

	riskScore := service.CalculateMultiDimensionalRiskScore(userID, nil)

	if riskScore == nil {
		t.Fatal("RiskScore should not be nil")
	}

	if riskScore.TotalScore < 0 || riskScore.TotalScore > 100 {
		t.Errorf("TotalScore should be between 0 and 100, got %f", riskScore.TotalScore)
	}

	if riskScore.Components == nil {
		t.Error("Components should not be nil")
	}

	if riskScore.Confidence < 0 || riskScore.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", riskScore.Confidence)
	}
}

func TestCalculateMultiDimensionalRiskScoreWithContext(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_context"

	context := &RiskContextV2{
		IsVPN:            true,
		IsProxy:          false,
		IsTor:            false,
		NewDevice:        true,
		NetworkQuality:    0.5,
		MechanicalBehavior: true,
	}

	riskScore := service.CalculateMultiDimensionalRiskScore(userID, context)

	if riskScore == nil {
		t.Fatal("RiskScore should not be nil")
	}

	if riskScore.Components.NetworkRisk <= 0 {
		t.Error("NetworkRisk should be positive when VPN is enabled")
	}

	if riskScore.Components.DeviceRisk <= 0 {
		t.Error("DeviceRisk should be positive for new device")
	}

	if riskScore.Components.BehavioralRisk <= 0 {
		t.Error("BehavioralRisk should be positive for mechanical behavior")
	}
}

func TestCalculateDeviceRisk(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()

	tests := []struct {
		name           string
		profile         *UserRiskProfileV2
		context        *RiskContextV2
		minExpected    float64
		maxExpected    float64
	}{
		{
			name:           "nil profile",
			profile:        nil,
			context:        nil,
			minExpected:    0,
			maxExpected:    50,
		},
		{
			name:           "known device",
			profile:        &UserRiskProfileV2{DeviceTrust: &DeviceTrust{IsKnownDevice: true, TrustScore: 80}},
			context:        nil,
			minExpected:    0,
			maxExpected:    20,
		},
		{
			name:           "new device",
			profile:        nil,
			context:        &RiskContextV2{NewDevice: true},
			minExpected:    20,
			maxExpected:    40,
		},
		{
			name:           "fingerprint mismatch",
			profile:        nil,
			context:        &RiskContextV2{FingerprintMismatch: true},
			minExpected:    25,
			maxExpected:    45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := service.calculateDeviceRisk(tt.profile, tt.context)
			if risk < tt.minExpected || risk > tt.maxExpected {
				t.Errorf("Expected risk between %f and %f, got %f", tt.minExpected, tt.maxExpected, risk)
			}
		})
	}
}

func TestCalculateBehavioralRisk(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()

	tests := []struct {
		name        string
		profile     *UserRiskProfileV2
		context     *RiskContextV2
		expectRisk  bool
	}{
		{
			name:       "nil profile",
			profile:    nil,
			context:    nil,
			expectRisk: false,
		},
		{
			name: "mechanical behavior",
			profile: &UserRiskProfileV2{
				BehaviorPattern: &BehaviorPattern{
					ClickIntervalStats: &IntervalStats{StdDev: 0.05},
				},
			},
			context:    &RiskContextV2{MechanicalBehavior: true},
			expectRisk: true,
		},
		{
			name: "unnatural speed",
			profile: &UserRiskProfileV2{
				BehaviorPattern: &BehaviorPattern{
					MouseSpeedStats: &SpeedStats{MaxSpeed: 2500},
				},
			},
			context:    &RiskContextV2{UnnaturalSpeed: true},
			expectRisk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := service.calculateBehavioralRisk(tt.profile, tt.context)
			if tt.expectRisk && risk < 20 {
				t.Errorf("Expected higher risk for %s, got %f", tt.name, risk)
			}
			if !tt.expectRisk && risk > 30 {
				t.Errorf("Expected lower risk for %s, got %f", tt.name, risk)
			}
		})
	}
}

func TestCalculateHistoricalRisk(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()

	tests := []struct {
		name          string
		profile       *UserRiskProfileV2
		expectedRange [2]float64
	}{
		{
			name:          "no history",
			profile:       nil,
			expectedRange: [2]float64{45, 55},
		},
		{
			name: "high success rate",
			profile: &UserRiskProfileV2{
				SuccessHistory:  []*VerificationResult{{Success: true}, {Success: true}, {Success: true}, {Success: true}},
				FailureHistory:  []*VerificationResult{},
				SessionMetrics:  &SessionMetrics{TotalAttempts: 4},
			},
			expectedRange: [2]float64{0, 25},
		},
		{
			name: "low success rate",
			profile: &UserRiskProfileV2{
				SuccessHistory:  []*VerificationResult{{Success: true}},
				FailureHistory:  []*VerificationResult{{Success: false}, {Success: false}, {Success: false}, {Success: false}},
				SessionMetrics:  &SessionMetrics{TotalAttempts: 5},
			},
			expectedRange: [2]float64{15, 45},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := service.calculateHistoricalRisk(tt.profile)
			if risk < tt.expectedRange[0] || risk > tt.expectedRange[1] {
				t.Errorf("Expected risk between %f and %f, got %f", tt.expectedRange[0], tt.expectedRange[1], risk)
			}
		})
	}
}

func TestRiskScoreToDifficulty(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()

	tests := []struct {
		riskScore  float64
		expected   DifficultyLevelV2
	}{
		{10, DifficultyV2Easy},
		{20, DifficultyV2Easy},
		{30, DifficultyV2Medium},
		{40, DifficultyV2Medium},
		{60, DifficultyV2Hard},
		{70, DifficultyV2Hard},
		{85, DifficultyV2Expert},
		{95, DifficultyV2Expert},
	}

	for _, tt := range tests {
		t.Run(string(tt.expected), func(t *testing.T) {
			result := service.riskScoreToDifficulty(tt.riskScore)
			if result != tt.expected {
				t.Errorf("Expected %s for risk score %f, got %s", tt.expected, tt.riskScore, result)
			}
		})
	}
}

func TestDifficultyToScore(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()

	tests := []struct {
		difficulty DifficultyLevelV2
		expected   float64
	}{
		{DifficultyV2Easy, 0},
		{DifficultyV2Medium, 1},
		{DifficultyV2Hard, 2},
		{DifficultyV2Expert, 3},
	}

	for _, tt := range tests {
		t.Run(string(tt.difficulty), func(t *testing.T) {
			result := service.difficultyToScore(tt.difficulty)
			if result != tt.expected {
				t.Errorf("Expected %f for difficulty %s, got %f", tt.expected, tt.difficulty, result)
			}
		})
	}
}

func TestAdjustDifficultyDynamically(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_adjust"

	profile := service.GetOrCreateProfileV2(userID)
	profile.SuccessHistory = append(profile.SuccessHistory, &VerificationResult{
		Difficulty: DifficultyV2Medium,
		Success:    true,
	})

	riskScore := &MultiDimensionalRiskScore{
		TotalScore: 30,
		Components: &RiskScoreComponent{},
	}

	difficulty, adjustment := service.AdjustDifficultyDynamically(userID, riskScore)

	if difficulty == "" {
		t.Error("Difficulty should not be empty")
	}

	t.Logf("Adjusted difficulty: %s, adjustment: %+v", difficulty, adjustment)
}

func TestAdjustDifficultyWithHighRisk(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_high_risk"

	riskScore := &MultiDimensionalRiskScore{
		TotalScore: 80,
		Components: &RiskScoreComponent{
			BehavioralRisk: 75,
			NetworkRisk:    70,
		},
		AnomalyIndicators: []string{"mechanical_behavior", "high_risk_network"},
	}

	difficulty, _ := service.AdjustDifficultyDynamically(userID, riskScore)

	if difficulty != DifficultyV2Hard && difficulty != DifficultyV2Expert {
		t.Errorf("Expected Hard or Expert for high risk, got %s", difficulty)
	}
}

func TestAdjustDifficultyWithLowRisk(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_low_risk"

	profile := service.GetOrCreateProfileV2(userID)
	profile.SessionMetrics.CurrentStreak = 7

	riskScore := &MultiDimensionalRiskScore{
		TotalScore: 15,
		Components: &RiskScoreComponent{},
	}

	difficulty, _ := service.AdjustDifficultyDynamically(userID, riskScore)

	if difficulty != DifficultyV2Easy {
		t.Errorf("Expected Easy for low risk with good streak, got %s", difficulty)
	}
}

func TestHandleTimeout(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_timeout"

	timeout1 := service.HandleTimeout(userID)

	if timeout1 == nil {
		t.Fatal("Timeout state should not be nil")
	}

	if !timeout1.IsActive {
		t.Error("Timeout should be active after first call")
	}

	if timeout1.Extensions != 0 {
		t.Errorf("Expected 0 extensions initially, got %d", timeout1.Extensions)
	}

	service.HandleTimeout(userID)
}

func TestHandleTimeoutWithExtensions(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_timeout_ext"

	profile := service.GetOrCreateProfileV2(userID)
	profile.TimeoutState.TimeoutDuration = 1 * time.Millisecond
	profile.TimeoutState.GracePeriod = 1 * time.Millisecond

	timeout1 := service.HandleTimeout(userID)
	if timeout1.Extensions != 0 {
		t.Errorf("Expected 0 extensions initially, got %d", timeout1.Extensions)
	}

	time.Sleep(5 * time.Millisecond)

	timeout2 := service.HandleTimeout(userID)
	if timeout2 == nil {
		t.Fatal("Timeout state should not be nil")
	}
}

func TestCheckTimeoutStatus(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_status"

	profile := service.GetOrCreateProfileV2(userID)
	profile.TimeoutState.IsActive = true
	profile.TimeoutState.StartTime = time.Now()
	profile.TimeoutState.TimeoutDuration = 60 * time.Second

	status := service.CheckTimeoutStatus(userID)

	if status == nil {
		t.Fatal("Status should not be nil")
	}

	if status.RemainingTime <= 0 {
		t.Error("Remaining time should be positive")
	}
}

func TestCancelTimeout(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_cancel"

	profile := service.GetOrCreateProfileV2(userID)
	profile.TimeoutState.IsActive = true

	result := service.CancelTimeout(userID)

	if !result {
		t.Error("Cancel should return true")
	}

	if profile.TimeoutState.IsActive {
		t.Error("Timeout should be inactive after cancel")
	}
}

func TestShouldAllowRetry(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_retry"

	allowed, retryState := service.ShouldAllowRetry(userID)

	if !allowed {
		t.Error("Should allow retry for new user")
	}

	if retryState == nil {
		t.Error("RetryState should not be nil")
	}
}

func TestShouldAllowRetryAfterMaxRetries(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_max_retry"

	profile := service.GetOrCreateProfileV2(userID)
	profile.RetryState.CurrentRetry = 3
	profile.RetryState.MaxRetries = 3
	profile.RetryState.LastRetryTime = time.Now()
	profile.RetryState.BackoffStrategy.CurrentDelay = 60 * time.Second

	allowed, _ := service.ShouldAllowRetry(userID)

	if allowed {
		t.Error("Should not allow retry after max retries within backoff period")
	}
}

func TestRecordRetryAttempt(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_record_retry"

	retryState1 := service.RecordRetryAttempt(userID)

	if retryState1.CurrentRetry != 1 {
		t.Errorf("Expected retry count 1, got %d", retryState1.CurrentRetry)
	}

	if retryState1.TotalRetries != 1 {
		t.Errorf("Expected total retries 1, got %d", retryState1.TotalRetries)
	}

	retryState2 := service.RecordRetryAttempt(userID)
	if retryState2.CurrentRetry != 2 {
		t.Errorf("Expected retry count 2, got %d", retryState2.CurrentRetry)
	}
}

func TestResetRetryState(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_reset"

	profile := service.GetOrCreateProfileV2(userID)
	profile.RetryState.CurrentRetry = 2
	profile.RetryState.TotalRetries = 2

	service.ResetRetryState(userID)

	if profile.RetryState.CurrentRetry != 0 {
		t.Errorf("Expected retry count 0 after reset, got %d", profile.RetryState.CurrentRetry)
	}
}

func TestRecordVerificationResult(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_result"

	result := &VerificationResult{
		Difficulty:   DifficultyV2Medium,
		Success:      true,
		ResponseTime: 5 * time.Second,
		MethodUsed:   "slider",
	}

	service.RecordVerificationResult(userID, result)

	profile := service.GetOrCreateProfileV2(userID)

	if len(profile.SuccessHistory) != 1 {
		t.Errorf("Expected 1 success history, got %d", len(profile.SuccessHistory))
	}

	if profile.SessionMetrics.TotalAttempts != 1 {
		t.Errorf("Expected 1 total attempt, got %d", profile.SessionMetrics.TotalAttempts)
	}

	if profile.SessionMetrics.CurrentStreak != 1 {
		t.Errorf("Expected current streak 1, got %d", profile.SessionMetrics.CurrentStreak)
	}
}

func TestRecordVerificationResultWithFailure(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_failure"

	successResult := &VerificationResult{
		Difficulty:   DifficultyV2Medium,
		Success:      true,
		ResponseTime: 5 * time.Second,
	}
	service.RecordVerificationResult(userID, successResult)

	failureResult := &VerificationResult{
		Difficulty:   DifficultyV2Medium,
		Success:      false,
		ResponseTime: 2 * time.Second,
		FailureReason: "wrong_answer",
	}
	service.RecordVerificationResult(userID, failureResult)

	profile := service.GetOrCreateProfileV2(userID)

	if len(profile.FailureHistory) != 1 {
		t.Errorf("Expected 1 failure history, got %d", len(profile.FailureHistory))
	}

	if profile.SessionMetrics.CurrentStreak != 0 {
		t.Errorf("Expected current streak 0 after failure, got %d", profile.SessionMetrics.CurrentStreak)
	}
}

func TestGetDifficultyRecommendation(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_recommend"

	context := &RiskContextV2{
		IsVPN:         false,
		NewDevice:     false,
		NetworkQuality: 0.8,
	}

	difficulty, riskScore := service.GetDifficultyRecommendation(userID, context)

	if difficulty == "" {
		t.Error("Difficulty should not be empty")
	}

	if riskScore < 0 || riskScore > 100 {
		t.Errorf("Risk score should be between 0 and 100, got %f", riskScore)
	}
}

func TestGetUserAnalyticsV2(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_analytics"

	for i := 0; i < 5; i++ {
		service.RecordVerificationResult(userID, &VerificationResult{
			Difficulty:   DifficultyV2Medium,
			Success:      i%2 == 0,
			ResponseTime: time.Duration(3+i) * time.Second,
		})
	}

	analytics := service.GetUserAnalyticsV2(userID)

	if analytics == nil {
		t.Fatal("Analytics should not be nil")
	}

	if analytics.TotalAttempts != 5 {
		t.Errorf("Expected 5 total attempts, got %d", analytics.TotalAttempts)
	}

	if analytics.SuccessCount != 3 {
		t.Errorf("Expected 3 success count, got %d", analytics.SuccessCount)
	}

	if analytics.SuccessRate == 0 {
		t.Error("Success rate should not be 0")
	}
}

func TestAnalyzeTrend(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_trend"

	profile := service.GetOrCreateProfileV2(userID)

	trend := service.analyzeTrend(profile)
	if trend != "insufficient_data" {
		t.Errorf("Expected 'insufficient_data' for new user, got %s", trend)
	}

	for i := 0; i < 10; i++ {
		profile.SuccessHistory = append(profile.SuccessHistory, &VerificationResult{
			Success: true,
		})
	}

	trend = service.analyzeTrend(profile)
	if trend == "" {
		t.Error("Trend should not be empty")
	}
}

func TestBackoffStrategy(t *testing.T) {
	bs := &BackoffStrategy{
		InitialDelay: 5 * time.Second,
		MaxDelay:     60 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.2,
	}

	delays := make([]time.Duration, 5)
	for i := 0; i < 5; i++ {
		bs.CurrentRetry = i
		delay := calculateNextDelayForRetry(bs)
		delays[i] = delay
	}

	for i := 1; i < len(delays); i++ {
		if delays[i] < delays[i-1] {
			t.Error("Backoff delay should increase over time")
		}
	}

	lastDelay := delays[len(delays)-1]
	maxDelay := bs.MaxDelay
	if lastDelay > maxDelay {
		t.Errorf("Delay should not exceed max delay, got %v, max %v", lastDelay, maxDelay)
	}
}

func TestRetryStateProgress(t *testing.T) {
	rs := &RetryState{
		CurrentRetry: 2,
		MaxRetries:  5,
	}

	current, total, percentage := rs.GetRetryProgress()

	if current != 2 {
		t.Errorf("Expected current 2, got %d", current)
	}

	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}

	expectedPercentage := float64(2) / float64(5) * 100
	if math.Abs(percentage-expectedPercentage) > 0.01 {
		t.Errorf("Expected percentage %f, got %f", expectedPercentage, percentage)
	}
}

func TestDetectAnomalyIndicators(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()

	tests := []struct {
		name            string
		components      *RiskScoreComponent
		expectedCount   int
		expectedContains []string
	}{
		{
			name: "no anomalies",
			components: &RiskScoreComponent{
				DeviceRisk:       10,
				BehavioralRisk:   20,
				HistoricalRisk:   30,
				ContextualRisk:   20,
				NetworkRisk:      10,
				GeolocationRisk:  10,
				TimePatternRisk:  10,
			},
			expectedCount:   0,
			expectedContains: nil,
		},
		{
			name: "high device risk",
			components: &RiskScoreComponent{
				DeviceRisk:       75,
				BehavioralRisk:   20,
				HistoricalRisk:   30,
				ContextualRisk:   20,
				NetworkRisk:      10,
				GeolocationRisk:  10,
				TimePatternRisk:  10,
			},
			expectedCount:   1,
			expectedContains: []string{"high_device_risk"},
		},
		{
			name: "multiple anomalies",
			components: &RiskScoreComponent{
				DeviceRisk:       80,
				BehavioralRisk:   80,
				HistoricalRisk:   70,
				ContextualRisk:   70,
				NetworkRisk:      85,
				GeolocationRisk:  75,
				TimePatternRisk:  65,
			},
			expectedCount:   6,
			expectedContains: []string{"high_device_risk", "mechanical_behavior_detected"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indicators := service.detectAnomalyIndicators(tt.components)

			if len(indicators) != tt.expectedCount {
				t.Errorf("Expected %d indicators, got %d", tt.expectedCount, len(indicators))
			}

			for _, expected := range tt.expectedContains {
				found := false
				for _, indicator := range indicators {
					if indicator == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find %s in indicators", expected)
				}
			}
		})
	}
}

func TestDecreaseDifficulty(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()

	tests := []struct {
		input    DifficultyLevelV2
		expected DifficultyLevelV2
	}{
		{DifficultyV2Expert, DifficultyV2Hard},
		{DifficultyV2Hard, DifficultyV2Medium},
		{DifficultyV2Medium, DifficultyV2Easy},
		{DifficultyV2Easy, DifficultyV2Easy},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := service.decreaseDifficulty(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestIncreaseDifficulty(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()

	tests := []struct {
		input    DifficultyLevelV2
		expected DifficultyLevelV2
	}{
		{DifficultyV2Easy, DifficultyV2Medium},
		{DifficultyV2Medium, DifficultyV2Hard},
		{DifficultyV2Hard, DifficultyV2Expert},
		{DifficultyV2Expert, DifficultyV2Expert},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := service.increaseDifficulty(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestApplySessionBasedAdjustment(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()

	tests := []struct {
		name        string
		metrics     *SessionMetrics
		base        DifficultyLevelV2
		expected    DifficultyLevelV2
	}{
		{
			name:     "good streak",
			metrics:  &SessionMetrics{CurrentStreak: 7},
			base:     DifficultyV2Hard,
			expected: DifficultyV2Medium,
		},
		{
			name: "poor performance",
			metrics: &SessionMetrics{
				CurrentStreak:    0,
				FailedAttempts:   3,
				SuccessfulAttempts: 1,
				TotalAttempts:    4,
			},
			base:     DifficultyV2Medium,
			expected: DifficultyV2Easy,
		},
		{
			name: "timeout attempts",
			metrics: &SessionMetrics{
				TimeoutAttempts: 2,
			},
			base:     DifficultyV2Medium,
			expected: DifficultyV2Easy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.applySessionBasedAdjustment(tt.metrics, tt.base)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestApplyAnomalyAdjustment(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()

	tests := []struct {
		name      string
		riskScore *MultiDimensionalRiskScore
		base      DifficultyLevelV2
		expected  DifficultyLevelV2
	}{
		{
			name: "high behavioral risk",
			riskScore: &MultiDimensionalRiskScore{
				Components: &RiskScoreComponent{
					BehavioralRisk: 75,
				},
				AnomalyIndicators: []string{},
			},
			base:     DifficultyV2Medium,
			expected: DifficultyV2Hard,
		},
		{
			name: "many anomalies",
			riskScore: &MultiDimensionalRiskScore{
				Components: &RiskScoreComponent{},
				AnomalyIndicators: []string{"a", "b", "c", "d"},
			},
			base:     DifficultyV2Medium,
			expected: DifficultyV2Hard,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.applyAnomalyAdjustment(tt.riskScore, tt.base)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestNewTimeoutManager(t *testing.T) {
	tm := NewTimeoutManager()

	if tm == nil {
		t.Fatal("TimeoutManager should not be nil")
	}

	if tm.activeTimeouts == nil {
		t.Error("activeTimeouts should be initialized")
	}

	if tm.defaultTimeout == 0 {
		t.Error("defaultTimeout should have a value")
	}
}

func TestNewRetryManager(t *testing.T) {
	rm := NewRetryManager()

	if rm == nil {
		t.Fatal("RetryManager should not be nil")
	}

	if rm.retryStates == nil {
		t.Error("retryStates should be initialized")
	}

	if rm.backoffConfig == nil {
		t.Error("backoffConfig should be initialized")
	}

	if rm.defaultMaxRetry != 3 {
		t.Errorf("Expected default max retry 3, got %d", rm.defaultMaxRetry)
	}
}

func TestNewMultiDimensionalRiskCalculator(t *testing.T) {
	calc := NewMultiDimensionalRiskCalculator()

	if calc == nil {
		t.Fatal("MultiDimensionalRiskCalculator should not be nil")
	}

	if calc.componentWeights == nil {
		t.Error("componentWeights should be initialized")
	}

	if len(calc.componentWeights) == 0 {
		t.Error("componentWeights should have entries")
	}

	if calc.anomalyThresholds == nil {
		t.Error("anomalyThresholds should be initialized")
	}
}

func TestNewDifficultyAdjustmentEngine(t *testing.T) {
	engine := NewDifficultyAdjustmentEngine()

	if engine == nil {
		t.Fatal("DifficultyAdjustmentEngine should not be nil")
	}

	if engine.adjustmentHistory == nil {
		t.Error("adjustmentHistory should be initialized")
	}

	if engine.trendAnalyzer == nil {
		t.Error("trendAnalyzer should be initialized")
	}
}

func TestRecordAdjustment(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()
	userID := "test_user_record_adj"

	adjustment := &DifficultyAdjustment{
		FromLevel: DifficultyV2Medium,
		ToLevel:   DifficultyV2Hard,
	}

	service.recordAdjustment(userID, adjustment)
	service.recordAdjustment(userID, adjustment)

	history := service.difficultyEngine.adjustmentHistory[userID]

	if len(history) != 2 {
		t.Errorf("Expected 2 adjustments in history, got %d", len(history))
	}
}

func TestCalculateDataSufficiency(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()

	tests := []struct {
		name     string
		profile  *UserRiskProfileV2
		minValue float64
	}{
		{
			name:     "nil profile",
			profile:  nil,
			minValue: 0.1,
		},
		{
			name: "known device with history",
			profile: &UserRiskProfileV2{
				SuccessHistory: make([]*VerificationResult, 25),
				DeviceTrust:    &DeviceTrust{IsKnownDevice: true},
				BehaviorPattern: &BehaviorPattern{
					ResponseTimeTrend: make([]float64, 15),
				},
				SessionMetrics: &SessionMetrics{TotalAttempts: 15},
			},
			minValue: 0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateDataSufficiency(tt.profile)
			if result < tt.minValue {
				t.Errorf("Expected sufficiency >= %f, got %f", tt.minValue, result)
			}
		})
	}
}

func TestGenerateAdjustmentReason(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()

	tests := []struct {
		name      string
		riskScore *MultiDimensionalRiskScore
		nonEmpty  bool
	}{
		{
			name:      "nil risk score",
			riskScore: nil,
			nonEmpty:  true,
		},
		{
			name: "with anomalies",
			riskScore: &MultiDimensionalRiskScore{
				AnomalyIndicators: []string{"test_anomaly"},
			},
			nonEmpty: true,
		},
		{
			name: "high behavioral risk",
			riskScore: &MultiDimensionalRiskScore{
				Components: &RiskScoreComponent{
					BehavioralRisk: 70,
				},
			},
			nonEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := service.generateAdjustmentReason(tt.riskScore)
			if tt.nonEmpty && reason == "" {
				t.Error("Expected non-empty reason")
			}
		})
	}
}

func TestCalculateAdjustmentFactor(t *testing.T) {
	service := NewAdaptiveDifficultyServiceV2()

	tests := []struct {
		from     DifficultyLevelV2
		to       DifficultyLevelV2
		expected float64
	}{
		{DifficultyV2Easy, DifficultyV2Hard, 1.0},
		{DifficultyV2Medium, DifficultyV2Expert, 1.0},
		{DifficultyV2Easy, DifficultyV2Medium, 0.5},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"_to_"+string(tt.to), func(t *testing.T) {
			result := service.calculateAdjustmentFactor(tt.from, tt.to)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}
