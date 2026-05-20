package service

import (
	"math"
	"testing"
	"time"
)

func TestNewAdaptiveDifficultyService(t *testing.T) {
	service := NewAdaptiveDifficultyService()
	if service == nil {
		t.Error("NewAdaptiveDifficultyService returned nil")
	}
	if service.profiles == nil {
		t.Error("profiles map should be initialized")
	}
	if service.config == nil {
		t.Error("config should be initialized")
	}
}

func TestAdaptiveDifficultyService_GetOrCreateProfile(t *testing.T) {
	service := NewAdaptiveDifficultyService()

	profile := service.GetOrCreateProfile("user1")
	if profile == nil {
		t.Error("GetOrCreateProfile returned nil")
	}
	if profile.UserID != "user1" {
		t.Errorf("UserID mismatch: expected user1, got %s", profile.UserID)
	}
	if profile.RiskScore != 50.0 {
		t.Errorf("Initial RiskScore should be 50.0, got %f", profile.RiskScore)
	}
	if profile.SuccessRate != 0.8 {
		t.Errorf("Initial SuccessRate should be 0.8, got %f", profile.SuccessRate)
	}

	existingProfile := service.GetOrCreateProfile("user1")
	if existingProfile != profile {
		t.Error("GetOrCreateProfile should return same profile for same user")
	}
}

func TestAdaptiveDifficultyService_UpdateProfile_Success(t *testing.T) {
	service := NewAdaptiveDifficultyService()

	initialProfile := service.GetOrCreateProfile("user1")
	initialScore := initialProfile.RiskScore

	service.UpdateProfile("user1", true, 5*time.Second)

	updatedProfile := service.GetOrCreateProfile("user1")
	if updatedProfile.RiskScore >= initialScore {
		t.Error("Successful verification should decrease risk score")
	}
	if updatedProfile.FailureCount != 0 {
		t.Error("Failure count should be reset after success")
	}
	if updatedProfile.SuccessRate <= initialProfile.SuccessRate {
		t.Error("Success rate should increase after successful verification")
	}
}

func TestAdaptiveDifficultyService_UpdateProfile_Failure(t *testing.T) {
	service := NewAdaptiveDifficultyService()

	initialProfile := service.GetOrCreateProfile("user1")
	initialScore := initialProfile.RiskScore

	service.UpdateProfile("user1", false, 5*time.Second)

	updatedProfile := service.GetOrCreateProfile("user1")
	if updatedProfile.RiskScore <= initialScore {
		t.Error("Failed verification should increase risk score")
	}
	if updatedProfile.FailureCount != 1 {
		t.Error("Failure count should be 1 after one failure")
	}
}

func TestAdaptiveDifficultyService_UpdateProfile_MultipleFailures(t *testing.T) {
	service := NewAdaptiveDifficultyService()

	profile := service.GetOrCreateProfile("user1")

	for i := 0; i < 3; i++ {
		service.UpdateProfile("user1", false, 5*time.Second)
	}

	updatedProfile := service.GetOrCreateProfile("user1")
	if updatedProfile.FailureCount != 3 {
		t.Errorf("Failure count should be 3, got %d", updatedProfile.FailureCount)
	}

	expectedMinScore := profile.RiskScore + 15.0*3.0
	if updatedProfile.RiskScore < expectedMinScore {
		t.Errorf("Risk score should increase significantly with multiple failures")
	}
}

func TestAdaptiveDifficultyService_UpdateProfile_TimePenalty(t *testing.T) {
	service := NewAdaptiveDifficultyService()

	profile := service.GetOrCreateProfile("user1")
	initialScore := profile.RiskScore

	service.UpdateProfile("user1", true, 500*time.Millisecond)

	updatedProfile := service.GetOrCreateProfile("user1")
	if updatedProfile.RiskScore <= initialScore {
		t.Error("Fast verification should add time penalty")
	}

	service.UpdateProfile("user1", true, 35*time.Second)

	updatedProfile = service.GetOrCreateProfile("user1")
	if updatedProfile.RiskScore <= initialScore {
		t.Error("Slow verification should add time penalty")
	}
}

func TestAdaptiveDifficultyService_GetDifficulty(t *testing.T) {
	service := NewAdaptiveDifficultyService()

	tests := []struct {
		name        string
		userID      string
		score       float64
		expected    DifficultyLevel
	}{
		{"Easy user", "easy_user", 10.0, DifficultyEasy},
		{"Medium user", "medium_user", 30.0, DifficultyMedium},
		{"Hard user", "hard_user", 50.0, DifficultyHard},
		{"Expert user", "expert_user", 90.0, DifficultyExpert},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := service.GetOrCreateProfile(tt.userID)
			profile.RiskScore = tt.score

			difficulty := service.GetDifficulty(tt.userID)
			if difficulty != tt.expected {
				t.Errorf("GetDifficulty() = %s, want %s", difficulty, tt.expected)
			}
		})
	}
}

func TestAdaptiveDifficultyService_GetDifficultyForCaptcha(t *testing.T) {
	service := NewAdaptiveDifficultyService()

	profile := service.GetOrCreateProfile("user1")
	profile.RiskScore = 30.0

	baseDifficulty := service.GetDifficulty("user1")

	abTestDifficulty := service.GetDifficultyForCaptcha("user1", true)

	if abTestDifficulty != baseDifficulty {
		t.Logf("A/B test may have modified difficulty (this is expected behavior)")
	}

	noAbTestDifficulty := service.GetDifficultyForCaptcha("user1", false)
	if noAbTestDifficulty != baseDifficulty {
		t.Error("Without A/B test, difficulty should match base difficulty")
	}
}

func TestAdaptiveDifficultyService_UpdateConfig(t *testing.T) {
	service := NewAdaptiveDifficultyService()

	newConfig := &DifficultyConfig{
		EasyThreshold:   30.0,
		MediumThreshold: 50.0,
		HardThreshold:   70.0,
		ExpertThreshold: 90.0,
		FailureWeight:   20.0,
		SuccessWeight:   -10.0,
		TimePenalty:     5.0,
	}

	service.UpdateConfig(newConfig)

	retrievedConfig := service.GetConfig()
	if retrievedConfig.EasyThreshold != 30.0 {
		t.Errorf("EasyThreshold not updated correctly: got %f, want 30.0", retrievedConfig.EasyThreshold)
	}
	if retrievedConfig.FailureWeight != 20.0 {
		t.Errorf("FailureWeight not updated correctly: got %f, want 20.0", retrievedConfig.FailureWeight)
	}
}

func TestAdaptiveDifficultyService_GetConfig(t *testing.T) {
	service := NewAdaptiveDifficultyService()

	config := service.GetConfig()
	if config == nil {
		t.Error("GetConfig returned nil")
	}

	if config.EasyThreshold != 20.0 {
		t.Errorf("Default EasyThreshold should be 20.0, got %f", config.EasyThreshold)
	}
	if config.MediumThreshold != 40.0 {
		t.Errorf("Default MediumThreshold should be 40.0, got %f", config.MediumThreshold)
	}
	if config.HardThreshold != 60.0 {
		t.Errorf("Default HardThreshold should be 60.0, got %f", config.HardThreshold)
	}
	if config.ExpertThreshold != 80.0 {
		t.Errorf("Default ExpertThreshold should be 80.0, got %f", config.ExpertThreshold)
	}
}

func TestAdaptiveDifficultyService_GetAllProfiles(t *testing.T) {
	service := NewAdaptiveDifficultyService()

	service.GetOrCreateProfile("user1")
	service.GetOrCreateProfile("user2")
	service.GetOrCreateProfile("user3")

	profiles := service.GetAllProfiles()
	if len(profiles) != 3 {
		t.Errorf("GetAllProfiles count mismatch: expected 3, got %d", len(profiles))
	}
}

func TestAdaptiveDifficultyService_AddBehaviorFlag(t *testing.T) {
	service := NewAdaptiveDifficultyService()

	profile := service.GetOrCreateProfile("user1")
	initialScore := profile.RiskScore

	service.AddBehaviorFlag("user1", "suspicious_pattern")

	updatedProfile := service.GetOrCreateProfile("user1")
	if len(updatedProfile.BehaviorFlags) != 1 {
		t.Error("Behavior flag should be added")
	}
	if updatedProfile.BehaviorFlags[0] != "suspicious_pattern" {
		t.Error("Behavior flag content mismatch")
	}
	if updatedProfile.RiskScore <= initialScore {
		t.Error("Adding behavior flag should increase risk score")
	}

	service.AddBehaviorFlag("user1", "suspicious_pattern")

	updatedProfile = service.GetOrCreateProfile("user1")
	if len(updatedProfile.BehaviorFlags) != 1 {
		t.Error("Duplicate behavior flag should not be added")
	}
}

func TestAdaptiveDifficultyService_RiskScoreBounds(t *testing.T) {
	service := NewAdaptiveDifficultyService()

	profile := service.GetOrCreateProfile("user1")
	profile.RiskScore = 95.0

	service.UpdateProfile("user1", false, 5*time.Second)

	updatedProfile := service.GetOrCreateProfile("user1")
	if updatedProfile.RiskScore > 100.0 {
		t.Error("Risk score should not exceed 100.0")
	}

	profile.RiskScore = 5.0
	service.UpdateProfile("user1", true, 5*time.Second)

	updatedProfile = service.GetOrCreateProfile("user1")
	if updatedProfile.RiskScore < 0.0 {
		t.Error("Risk score should not go below 0.0")
	}
}

func TestAdaptiveDifficultyService_DifficultyTransitions(t *testing.T) {
	service := NewAdaptiveDifficultyService()

	profile := service.GetOrCreateProfile("user1")
	profile.RiskScore = 25.0

	if service.GetDifficulty("user1") != DifficultyMedium {
		t.Error("User should be at Medium difficulty at score 25")
	}

	for i := 0; i < 5; i++ {
		service.UpdateProfile("user1", false, 5*time.Second)
	}

	if service.GetDifficulty("user1") != DifficultyHard && service.GetDifficulty("user1") != DifficultyExpert {
		t.Error("User should move to higher difficulty after multiple failures")
	}
}

func TestAdaptiveDifficultyService_AvgTimeUpdate(t *testing.T) {
	service := NewAdaptiveDifficultyService()

	profile := service.GetOrCreateProfile("user1")

	service.UpdateProfile("user1", true, 10*time.Second)
	updatedProfile := service.GetOrCreateProfile("user1")

	if math.Abs(updatedProfile.AvgTime-10.0) > 0.1 {
		t.Errorf("AvgTime should be close to 10.0 after first update, got %f", updatedProfile.AvgTime)
	}

	service.UpdateProfile("user1", true, 10*time.Second)
	updatedProfile = service.GetOrCreateProfile("user1")

	if math.Abs(updatedProfile.AvgTime-10.0) > 0.1 {
		t.Error("AvgTime should remain stable with consistent updates")
	}
}

func TestAdaptiveDifficultyService_ConcurrentProfileAccess(t *testing.T) {
	service := NewAdaptiveDifficultyService()

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(userID string) {
			for j := 0; j < 100; j++ {
				service.GetOrCreateProfile(userID)
				service.UpdateProfile(userID, j%2 == 0, 5*time.Second)
			}
			done <- true
		}(string(rune('0' + i)))
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestDifficultyLevelConstants(t *testing.T) {
	if DifficultyEasy != "Easy" {
		t.Errorf("DifficultyEasy should be 'Easy', got %s", DifficultyEasy)
	}
	if DifficultyMedium != "Medium" {
		t.Errorf("DifficultyMedium should be 'Medium', got %s", DifficultyMedium)
	}
	if DifficultyHard != "Hard" {
		t.Errorf("DifficultyHard should be 'Hard', got %s", DifficultyHard)
	}
	if DifficultyExpert != "Expert" {
		t.Errorf("DifficultyExpert should be 'Expert', got %s", DifficultyExpert)
	}
}
