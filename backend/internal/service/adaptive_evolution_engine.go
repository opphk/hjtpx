package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

type AdaptiveEvolutionEngine struct {
	userProfiles     map[string]*UserBehaviorProfile
	attackHistory    map[string][]AttackPattern
	difficultyLevels map[string]*DifficultyConfig
	evolutionHistory map[string][]EvolutionEvent
	ecosystemMetrics *EcosystemMetrics
	mu               sync.RWMutex
}

type UserBehaviorProfile struct {
	UserID           string                 `json:"user_id"`
	SessionCount     int                    `json:"session_count"`
	SuccessRate      float64                `json:"success_rate"`
	AvgSolveTime     time.Duration          `json:"avg_solve_time"`
	FailedAttempts   int                    `json:"failed_attempts"`
	PreferredTypes   []string               `json:"preferred_types"`
	AbilityScores    map[string]float64     `json:"ability_scores"`
	LearningRate     float64                `json:"learning_rate"`
	AdaptationLevel  int                    `json:"adaptation_level"`
	LastUpdate       time.Time              `json:"last_update"`
	SuccessHistory   []bool                 `json:"success_history"`
	TimeHistory      []float64              `json:"time_history"`
	DifficultyHistory []int                `json:"difficulty_history"`
}

type AttackPattern struct {
	Type           string                 `json:"type"`
	Timestamp      time.Time              `json:"timestamp"`
	Success        bool                   `json:"success"`
	AttackVector   string                 `json:"attack_vector"`
	TargetType     string                 `json:"target_type"`
	AttemptCount   int                    `json:"attempt_count"`
	IPAddress      string                 `json:"ip_address"`
	UserAgent      string                 `json:"user_agent"`
	FeatureVector  map[string]float64     `json:"feature_vector"`
	Complexity     float64                `json:"complexity"`
	Signature      string                 `json:"signature"`
}

type DifficultyConfig struct {
	Level           int                     `json:"level"`
	VisualComplexity int                   `json:"visual_complexity"`
	AudioComplexity int                   `json:"audio_complexity"`
	TimeLimit       time.Duration          `json:"time_limit"`
	HintAvailability bool                  `json:"hint_availability"`
	AlternativeModes []string              `json:"alternative_modes"`
	RewardMultiplier float64               `json:"reward_multiplier"`
}

type EvolutionEvent struct {
	Timestamp     time.Time               `json:"timestamp"`
	EventType     string                  `json:"event_type"`
	UserID        string                  `json:"user_id"`
	Change        map[string]interface{}  `json:"change"`
	Reason        string                  `json:"reason"`
	Efficiency    float64                 `json:"efficiency"`
}

type EcosystemMetrics struct {
	TotalAttempts    int64                `json:"total_attempts"`
	SuccessRate      float64              `json:"success_rate"`
	AvgDifficulty    float64              `json:"avg_difficulty"`
	BotDetectionRate float64              `json:"bot_detection_rate"`
	HumanSuccessRate float64              `json:"human_success_rate"`
	AttackCount      int64                `json:"attack_count"`
	DefenseCount     int64                `json:"defense_count"`
	EvolutionCycles  int                  `json:"evolution_cycles"`
	StabilityScore   float64              `json:"stability_score"`
}

type CaptchaChallenge struct {
	Type            string                 `json:"type"`
	Difficulty      int                    `json:"difficulty"`
	Config          *DifficultyConfig      `json:"config"`
	AdaptiveHints   []string               `json:"adaptive_hints"`
	Multimodal      bool                   `json:"multimodal"`
	Personalized    bool                   `json:"personalized"`
	Metadata        map[string]interface{} `json:"metadata"`
}

func NewAdaptiveEvolutionEngine() *AdaptiveEvolutionEngine {
	engine := &AdaptiveEvolutionEngine{
		userProfiles:     make(map[string]*UserBehaviorProfile),
		attackHistory:    make(map[string][]AttackPattern),
		difficultyLevels: make(map[string]*DifficultyConfig),
		evolutionHistory: make(map[string][]EvolutionEvent),
		ecosystemMetrics: &EcosystemMetrics{
			TotalAttempts:   0,
			SuccessRate:     0.5,
			AvgDifficulty:   2.0,
			StabilityScore:  0.8,
		},
	}

	engine.initializeDifficultyLevels()

	return engine
}

func (e *AdaptiveEvolutionEngine) initializeDifficultyLevels() {
	levels := []DifficultyConfig{
		{
			Level:             1,
			VisualComplexity:  1,
			AudioComplexity:   1,
			TimeLimit:         60 * time.Second,
			HintAvailability:  true,
			AlternativeModes:  []string{"simplified", "text"},
			RewardMultiplier:  1.0,
		},
		{
			Level:             2,
			VisualComplexity:  2,
			AudioComplexity:   2,
			TimeLimit:         45 * time.Second,
			HintAvailability:  true,
			AlternativeModes:  []string{"simplified", "audio"},
			RewardMultiplier:  1.2,
		},
		{
			Level:             3,
			VisualComplexity:  3,
			AudioComplexity:   3,
			TimeLimit:         30 * time.Second,
			HintAvailability:  false,
			AlternativeModes:  []string{"multilingual"},
			RewardMultiplier:  1.5,
		},
		{
			Level:             4,
			VisualComplexity:  4,
			AudioComplexity:   4,
			TimeLimit:         20 * time.Second,
			HintAvailability:  false,
			AlternativeModes:  []string{"spatial"},
			RewardMultiplier:  2.0,
		},
		{
			Level:             5,
			VisualComplexity:  5,
			AudioComplexity:   5,
			TimeLimit:         15 * time.Second,
			HintAvailability:  false,
			AlternativeModes:  []string{"quantum"},
			RewardMultiplier:  3.0,
		},
	}

	for i := range levels {
		levelKey := fmt.Sprintf("level_%d", levels[i].Level)
		e.difficultyLevels[levelKey] = &levels[i]
	}
}

func (e *AdaptiveEvolutionEngine) AnalyzeUserAndGenerateChallenge(
	ctx context.Context,
	userID string,
	attemptCount int,
) (*CaptchaChallenge, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	profile := e.getOrCreateUserProfile(userID)

	e.ecosystemMetrics.TotalAttempts++

	recommendedDifficulty := e.calculateRecommendedDifficulty(profile, attemptCount)

	challenge := &CaptchaChallenge{
		Type:         e.selectChallengeType(profile),
		Difficulty:   recommendedDifficulty,
		Config:       e.getDifficultyConfig(recommendedDifficulty),
		Multimodal:   recommendedDifficulty >= 3,
		Personalized: true,
		Metadata:    make(map[string]interface{}),
	}

	if profile.AdaptationLevel > 1 {
		challenge.AdaptiveHints = e.generateAdaptiveHints(challenge)
	}

	challenge.Metadata["recommended_solve_time"] = challenge.Config.TimeLimit.Seconds()
	challenge.Metadata["user_success_rate"] = profile.SuccessRate
	challenge.Metadata["adaptation_score"] = profile.LearningRate

	e.recordEvolutionEvent(userID, "challenge_generated", map[string]interface{}{
		"difficulty":   recommendedDifficulty,
		"challenge_type": challenge.Type,
		"user_history": len(profile.SuccessHistory),
	})

	return challenge, nil
}

func (e *AdaptiveEvolutionEngine) getOrCreateUserProfile(userID string) *UserBehaviorProfile {
	profile, exists := e.userProfiles[userID]
	if !exists {
		profile = &UserBehaviorProfile{
			UserID:          userID,
			AbilityScores:  make(map[string]float64),
			SuccessHistory: make([]bool, 0),
			TimeHistory:    make([]float64, 0),
			DifficultyHistory: make([]int, 0),
			LearningRate:   0.5,
			AdaptationLevel: 1,
		}
		e.userProfiles[userID] = profile
	}
	return profile
}

func (e *AdaptiveEvolutionEngine) calculateRecommendedDifficulty(profile *UserBehaviorProfile, attemptCount int) int {
	if len(profile.SuccessHistory) < 3 {
		return 1
	}

	baseDifficulty := 1
	if profile.SuccessRate > 0.9 {
		baseDifficulty = 5
	} else if profile.SuccessRate > 0.8 {
		baseDifficulty = 4
	} else if profile.SuccessRate > 0.7 {
		baseDifficulty = 3
	} else if profile.SuccessRate > 0.5 {
		baseDifficulty = 2
	}

	timeBonus := 0
	if len(profile.TimeHistory) >= 5 {
		avgTime := average(profile.TimeHistory)
		if avgTime < 10 {
			timeBonus = 1
		}
	}

	difficultyBonus := 0
	if len(profile.DifficultyHistory) >= 3 {
		avgDifficulty := float64(sum(profile.DifficultyHistory)) / float64(len(profile.DifficultyHistory))
		if avgDifficulty > 3 {
			difficultyBonus = 1
		}
	}

	recommendedDifficulty := baseDifficulty + timeBonus + difficultyBonus

	if recommendedDifficulty > 5 {
		recommendedDifficulty = 5
	}
	if recommendedDifficulty < 1 {
		recommendedDifficulty = 1
	}

	profile.DifficultyHistory = append(profile.DifficultyHistory, recommendedDifficulty)
	if len(profile.DifficultyHistory) > 20 {
		profile.DifficultyHistory = profile.DifficultyHistory[len(profile.DifficultyHistory)-20:]
	}

	return recommendedDifficulty
}

func (e *AdaptiveEvolutionEngine) selectChallengeType(profile *UserBehaviorProfile) string {
	if len(profile.PreferredTypes) == 0 {
		types := []string{"visual", "audio", "tactile", "spatial"}
		return types[int(time.Now().Unix())%len(types)]
	}

	typeFreq := make(map[string]int)
	for _, t := range profile.PreferredTypes {
		typeFreq[t]++
	}

	mostPreferred := ""
	maxFreq := 0
	for t, freq := range typeFreq {
		if freq > maxFreq {
			maxFreq = freq
			mostPreferred = t
		}
	}

	return mostPreferred
}

func (e *AdaptiveEvolutionEngine) getDifficultyConfig(level int) *DifficultyConfig {
	levelKey := fmt.Sprintf("level_%d", level)
	if config, ok := e.difficultyLevels[levelKey]; ok {
		return config
	}
	return e.difficultyLevels["level_1"]
}

func (e *AdaptiveEvolutionEngine) generateAdaptiveHints(challenge *CaptchaChallenge) []string {
	hints := []string{}

	switch challenge.Type {
	case "visual":
		hints = append(hints, "观察图像的颜色分布")
		if challenge.Difficulty < 3 {
			hints = append(hints, "注意图像中的圆形元素")
		}
	case "audio":
		hints = append(hints, "注意数字之间的停顿")
		if challenge.Difficulty < 3 {
			hints = append(hints, "语速较慢，请仔细聆听")
		}
	case "tactile":
		hints = append(hints, "短震动表示0-4，长震动表示5-9")
	}

	return hints
}

func (e *AdaptiveEvolutionEngine) RecordAttempt(
	ctx context.Context,
	userID string,
	success bool,
	solveTime time.Duration,
	difficulty int,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	profile := e.getOrCreateUserProfile(userID)
	profile.LastUpdate = time.Now()
	profile.SessionCount++

	profile.SuccessHistory = append(profile.SuccessHistory, success)
	if len(profile.SuccessHistory) > 50 {
		profile.SuccessHistory = profile.SuccessHistory[len(profile.SuccessHistory)-50:]
	}

	profile.TimeHistory = append(profile.TimeHistory, solveTime.Seconds())
	if len(profile.TimeHistory) > 50 {
		profile.TimeHistory = profile.TimeHistory[len(profile.TimeHistory)-50:]
	}

	profile.SuccessRate = e.calculateSuccessRate(profile.SuccessHistory)
	profile.AvgSolveTime = e.calculateAvgSolveTime(profile.TimeHistory)

	if success {
		profile.AdaptationLevel++
		if profile.AdaptationLevel > 10 {
			profile.AdaptationLevel = 10
		}
		profile.LearningRate = math.Min(1.0, profile.LearningRate+0.05)
	} else {
		profile.FailedAttempts++
		profile.AdaptationLevel = math.Max(1, profile.AdaptationLevel-1)
		profile.LearningRate = math.Max(0.1, profile.LearningRate-0.1)
	}

	e.ecosystemMetrics.SuccessRate = (e.ecosystemMetrics.SuccessRate*float64(e.ecosystemMetrics.TotalAttempts-1) +
		boolToFloat(success)) / float64(e.ecosystemMetrics.TotalAttempts)

	e.recordEvolutionEvent(userID, "attempt_recorded", map[string]interface{}{
		"success":     success,
		"solve_time":  solveTime.Seconds(),
		"difficulty":  difficulty,
		"new_rate":    profile.SuccessRate,
	})

	return nil
}

func (e *AdaptiveEvolutionEngine) calculateSuccessRate(history []bool) float64 {
	if len(history) == 0 {
		return 0.5
	}
	successCount := 0
	for _, s := range history {
		if s {
			successCount++
		}
	}
	return float64(successCount) / float64(len(history))
}

func (e *AdaptiveEvolutionEngine) calculateAvgSolveTime(history []float64) time.Duration {
	if len(history) == 0 {
		return 0
	}
	sum := 0.0
	for _, t := range history {
		sum += t
	}
	return time.Duration(sum/float64(len(history))) * time.Second
}

func (e *AdaptiveEvolutionEngine) RecordAttack(ctx context.Context, pattern *AttackPattern) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	pattern.Timestamp = time.Now()

	key := pattern.AttackVector
	e.attackHistory[key] = append(e.attackHistory[key], *pattern)

	e.ecosystemMetrics.AttackCount++
	e.ecosystemMetrics.DefenseCount++

	e.adjustEcosystemForAttack(pattern)

	e.recordEvolutionEvent("system", "attack_detected", map[string]interface{}{
		"attack_type":  pattern.Type,
		"attack_vector": pattern.AttackVector,
		"success":      pattern.Success,
	})

	return nil
}

func (e *AdaptiveEvolutionEngine) adjustEcosystemForAttack(pattern *AttackPattern) {
	e.ecosystemMetrics.BotDetectionRate = (e.ecosystemMetrics.BotDetectionRate*float64(e.ecosystemMetrics.DefenseCount-1) +
		boolToFloat(!pattern.Success)) / float64(e.ecosystemMetrics.DefenseCount)

	baseDifficultyIncrease := 0.5
	if pattern.Success {
		baseDifficultyIncrease = 1.0
	}

	e.ecosystemMetrics.AvgDifficulty = math.Min(5.0, e.ecosystemMetrics.AvgDifficulty+baseDifficultyIncrease)
}

func (e *AdaptiveEvolutionEngine) OptimizeDifficulty() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.ecosystemMetrics.TotalAttempts < 10 {
		return nil
	}

	targetSuccessRate := 0.7
	currentSuccessRate := e.ecosystemMetrics.HumanSuccessRate

	if currentSuccessRate < targetSuccessRate-0.1 {
		e.decreaseOverallDifficulty()
	} else if currentSuccessRate > targetSuccessRate+0.1 {
		e.increaseOverallDifficulty()
	}

	e.ecosystemMetrics.EvolutionCycles++

	stability := 1.0 - math.Abs(currentSuccessRate-targetSuccessRate)
	e.ecosystemMetrics.StabilityScore = stability

	e.recordEvolutionEvent("system", "difficulty_optimized", map[string]interface{}{
		"new_avg_difficulty": e.ecosystemMetrics.AvgDifficulty,
		"stability_score":    e.ecosystemMetrics.StabilityScore,
	})

	return nil
}

func (e *AdaptiveEvolutionEngine) decreaseOverallDifficulty() {
	for key := range e.difficultyLevels {
		if e.difficultyLevels[key].Level > 1 {
			e.difficultyLevels[key].Level--
			e.difficultyLevels[key].TimeLimit += 5 * time.Second
		}
	}
	e.ecosystemMetrics.AvgDifficulty = math.Max(1.0, e.ecosystemMetrics.AvgDifficulty-0.5)
}

func (e *AdaptiveEvolutionEngine) increaseOverallDifficulty() {
	for key := range e.difficultyLevels {
		if e.difficultyLevels[key].Level < 5 {
			e.difficultyLevels[key].Level++
			e.difficultyLevels[key].TimeLimit -= 3 * time.Second
		}
	}
	e.ecosystemMetrics.AvgDifficulty = math.Min(5.0, e.ecosystemMetrics.AvgDifficulty+0.3)
}

func (e *AdaptiveEvolutionEngine) GetUserProfile(ctx context.Context, userID string) (*UserBehaviorProfile, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	profile, exists := e.userProfiles[userID]
	if !exists {
		return nil, fmt.Errorf("user profile not found")
	}

	return profile, nil
}

func (e *AdaptiveEvolutionEngine) GetEcosystemMetrics(ctx context.Context) (*EcosystemMetrics, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	metrics := *e.ecosystemMetrics
	return &metrics, nil
}

func (e *AdaptiveEvolutionEngine) recordEvolutionEvent(userID, eventType string, change map[string]interface{}) {
	event := EvolutionEvent{
		Timestamp:  time.Now(),
		EventType: eventType,
		UserID:    userID,
		Change:    change,
		Efficiency: e.ecosystemMetrics.StabilityScore,
	}

	e.evolutionHistory[userID] = append(e.evolutionHistory[userID], event)

	if len(e.evolutionHistory[userID]) > 100 {
		e.evolutionHistory[userID] = e.evolutionHistory[userID][len(e.evolutionHistory[userID])-100:]
	}
}

func (e *AdaptiveEvolutionEngine) LearnFromFeedback(ctx context.Context, feedback *models.Feedback) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if feedback.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	profile := e.getOrCreateUserProfile(feedback.UserID)

	profile.PreferredTypes = append(profile.PreferredTypes, feedback.Type)

	for capability, score := range feedback.Capabilities {
		if existingScore, ok := profile.AbilityScores[capability]; ok {
			profile.AbilityScores[capability] = existingScore*0.7 + score*0.3
		} else {
			profile.AbilityScores[capability] = score
		}
	}

	e.recordEvolutionEvent(feedback.UserID, "feedback_learning", map[string]interface{}{
		"feedback_type": feedback.Type,
		"rating":        feedback.Rating,
	})

	return nil
}

func (e *AdaptiveEvolutionEngine) PredictUserBehavior(userID string, futureAttempts int) (float64, error) {
	e.mu.RLock()
	profile, exists := e.userProfiles[userID]
	e.mu.RUnlock()

	if !exists || len(profile.SuccessHistory) < 5 {
		return 0.5, nil
	}

	successRate := profile.SuccessRate
	learningRate := profile.LearningRate

	for i := 0; i < futureAttempts; i++ {
		predictedSuccess := successRate + learningRate*0.05
		if predictedSuccess > 1.0 {
			predictedSuccess = 1.0
		}
		successRate = predictedSuccess
	}

	return math.Min(1.0, successRate), nil
}

func (e *AdaptiveEvolutionEngine) GenerateEcosystemReport() string {
	report := "自适应生态系统报告\n"
	report += "====================\n\n"
	report += fmt.Sprintf("总尝试次数: %d\n", e.ecosystemMetrics.TotalAttempts)
	report += fmt.Sprintf("整体成功率: %.2f%%\n", e.ecosystemMetrics.SuccessRate*100)
	report += fmt.Sprintf("人类用户成功率: %.2f%%\n", e.ecosystemMetrics.HumanSuccessRate*100)
	report += fmt.Sprintf("平均难度等级: %.1f\n", e.ecosystemMetrics.AvgDifficulty)
	report += fmt.Sprintf("机器人检测率: %.2f%%\n", e.ecosystemMetrics.BotDetectionRate*100)
	report += fmt.Sprintf("攻击次数: %d\n", e.ecosystemMetrics.AttackCount)
	report += fmt.Sprintf("防御次数: %d\n", e.ecosystemMetrics.DefenseCount)
	report += fmt.Sprintf("进化周期: %d\n", e.ecosystemMetrics.EvolutionCycles)
	report += fmt.Sprintf("稳定性评分: %.2f\n", e.ecosystemMetrics.StabilityScore)
	report += "\n难度等级分布:\n"

	for key, config := range e.difficultyLevels {
		report += fmt.Sprintf("  %s: 视觉复杂度=%d, 音频复杂度=%d, 时间限制=%.1fs\n",
			key, config.VisualComplexity, config.AudioComplexity, config.TimeLimit.Seconds())
	}

	report += "\n用户画像统计:\n"
	report += fmt.Sprintf("  注册用户数: %d\n", len(e.userProfiles))

	return report
}

func (e *AdaptiveEvolutionEngine) ResetUserProfile(userID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.userProfiles[userID]; exists {
		delete(e.userProfiles, userID)
	}

	if history, exists := e.evolutionHistory[userID]; exists {
		delete(e.evolutionHistory, userID)
		_ = history
	}

	return nil
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func sum(values []int) int {
	s := 0
	for _, v := range values {
		s += v
	}
	return s
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func (e *AdaptiveEvolutionEngine) ExportMetricsJSON() (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	data := map[string]interface{}{
		"ecosystem_metrics": e.ecosystemMetrics,
		"difficulty_levels":  e.difficultyLevels,
		"user_count":        len(e.userProfiles),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}
