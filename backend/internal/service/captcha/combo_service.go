package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
)

type ComboService struct {
	sessionCache   *cache.SessionCache
	captchaRepo    *db.CaptchaRepository
	videoGenerator *VideoGeneratorService
	arGenerator    *ARGeneratorService
	inMemoryStore  map[string]*ComboCaptchaConfig
}

type ComboCaptchaConfig struct {
	SessionID       string              `json:"session_id"`
	Steps           []*ComboStep       `json:"steps"`
	Strategy        ComboStrategy      `json:"strategy"`
	TotalRequired   int                 `json:"total_required"`
	Status          string              `json:"status"`
	CurrentStep     int                 `json:"current_step"`
	VerifiedCount   int                 `json:"verified_count"`
	FailedCount     int                 `json:"failed_count"`
	TotalScore      float64             `json:"total_score"`
	RiskScore       float64             `json:"risk_score"`
	CreatedAt       time.Time           `json:"created_at"`
	ExpiredAt       time.Time           `json:"expired_at"`
	ClientIP        string              `json:"client_ip"`
	UserAgent       string              `json:"user_agent"`
	Fingerprint     string              `json:"fingerprint"`
}

type ComboStep struct {
	Index          int                    `json:"index"`
	Type           string                `json:"type"`
	SubType        string                `json:"sub_type,omitempty"`
	Difficulty     int                    `json:"difficulty"`
	Mandatory      bool                   `json:"mandatory"`
	Status         string                `json:"status"`
	Score          float64               `json:"score"`
	MaxScore       float64               `json:"max_score"`
	Attempts       int                    `json:"attempts"`
	MaxAttempts    int                    `json:"max_attempts"`
	SessionID      string                `json:"session_id"`
	Data           interface{}           `json:"data"`
	TimeLimit      int                    `json:"time_limit"`
	StartedAt      *time.Time           `json:"started_at,omitempty"`
	CompletedAt    *time.Time           `json:"completed_at,omitempty"`
}

type ComboStrategy string

const (
	ComboStrategyAll       ComboStrategy = "all"
	ComboStrategyAny       ComboStrategy = "any"
	ComboStrategyMajority  ComboStrategy = "majority"
	ComboStrategyWeighted  ComboStrategy = "weighted"
)

type CreateComboRequest struct {
	Types          []string              `json:"types"`
	Strategy       ComboStrategy        `json:"strategy"`
	Difficulty     int                   `json:"difficulty"`
	RiskScore      float64              `json:"risk_score"`
	MaxSteps       int                   `json:"max_steps"`
	ClientIP       string               `json:"client_ip"`
	UserAgent      string               `json:"user_agent"`
	Fingerprint    string               `json:"fingerprint"`
	PreferFast     bool                 `json:"prefer_fast"`
	PreferSecure   bool                 `json:"prefer_secure"`
}

type CreateComboResponse struct {
	SessionID     string              `json:"session_id"`
	Steps         []*ComboStep        `json:"steps"`
	Strategy      ComboStrategy       `json:"strategy"`
	TotalRequired int                 `json:"total_required"`
	ExpiresIn     int64               `json:"expires_in"`
	ExpiresAt     int64               `json:"expires_at"`
	EstimatedTime int                 `json:"estimated_time"`
}

type VerifyComboRequest struct {
	SessionID     string                 `json:"session_id"`
	StepResults   []StepResult          `json:"step_results"`
	BehaviorData  map[string]interface{} `json:"behavior_data"`
}

type StepResult struct {
	StepIndex     int                    `json:"step_index"`
	StepType      string                `json:"step_type"`
	StepSessionID string                `json:"step_session_id"`
	Success       bool                  `json:"success"`
	Score         float64               `json:"score"`
	Data          interface{}           `json:"data,omitempty"`
	TimeSpent     float64               `json:"time_spent"`
}

type VerifyComboResponse struct {
	Success       bool                    `json:"success"`
	Score         float64                 `json:"score"`
	Message       string                  `json:"message"`
	PassedSteps   int                     `json:"passed_steps"`
	FailedSteps   int                     `json:"failed_steps"`
	StepResults   []StepResult            `json:"step_results"`
	CanRetry      bool                    `json:"can_retry"`
	RetrySteps    []int                   `json:"retry_steps,omitempty"`
}

type ComboSelector struct {
	weights       map[string]float64
	sceneTypes     []string
	difficultyCaps map[string]int
}

func NewComboService(
	sessionCache *cache.SessionCache,
	captchaRepo *db.CaptchaRepository,
	videoGenerator *VideoGeneratorService,
	arGenerator *ARGeneratorService,
) *ComboService {
	return &ComboService{
		sessionCache:   sessionCache,
		captchaRepo:    captchaRepo,
		videoGenerator: videoGenerator,
		arGenerator:    arGenerator,
		inMemoryStore:  make(map[string]*ComboCaptchaConfig),
	}
}

func (s *ComboService) Create(ctx context.Context, req *CreateComboRequest) (*CreateComboResponse, error) {
	sessionID := generateComboSessionIDV2()
	expiresAt := time.Now().Add(10 * time.Minute)

	strategy := req.Strategy
	if strategy == "" {
		strategy = ComboStrategyMajority
	}

	difficulty := req.Difficulty
	if difficulty <= 0 {
		difficulty = 2
	}
	if difficulty > 5 {
		difficulty = 5
	}

	maxSteps := req.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 3
	}
	if maxSteps > 5 {
		maxSteps = 5
	}

	steps := s.generateIntelligentSteps(ctx, req, difficulty, maxSteps)
	totalRequired := s.calculateRequiredSteps(strategy, len(steps))

	config := &ComboCaptchaConfig{
		SessionID:     sessionID,
		Steps:         steps,
		Strategy:      strategy,
		TotalRequired: totalRequired,
		Status:        "pending",
		CurrentStep:   0,
		VerifiedCount: 0,
		FailedCount:   0,
		TotalScore:    0,
		RiskScore:     req.RiskScore,
		CreatedAt:     time.Now(),
		ExpiredAt:     expiresAt,
		ClientIP:      req.ClientIP,
		UserAgent:     req.UserAgent,
		Fingerprint:   req.Fingerprint,
	}

	if err := s.saveConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to save combo config: %w", err)
	}

	estimatedTime := 0
	for _, step := range steps {
		estimatedTime += step.TimeLimit
	}

	return &CreateComboResponse{
		SessionID:     sessionID,
		Steps:         steps,
		Strategy:      strategy,
		TotalRequired: totalRequired,
		ExpiresIn:     int64(10 * time.Minute / time.Second),
		ExpiresAt:     expiresAt.Unix(),
		EstimatedTime: estimatedTime,
	}, nil
}

func (s *ComboService) generateIntelligentSteps(ctx context.Context, req *CreateComboRequest, difficulty, maxSteps int) []*ComboStep {
	availableTypes := s.getAvailableTypes(req)
	selector := s.createSmartSelector(req)

	selectedTypes := selector.SelectTypes(maxSteps, difficulty, req.RiskScore)
	filteredTypes := s.filterAvailableTypes(selectedTypes, availableTypes)

	// Ensure we have at least maxSteps types
	for len(filteredTypes) < maxSteps {
		for _, t := range availableTypes {
			if len(filteredTypes) >= maxSteps {
				break
			}
			// Check if we already have this type
			hasType := false
			for _, ft := range filteredTypes {
				if ft == t {
					hasType = true
					break
				}
			}
			if !hasType {
				filteredTypes = append(filteredTypes, t)
			}
		}
		// If we've exhausted all available types and still need more, reuse types
		if len(filteredTypes) < maxSteps && len(availableTypes) > 0 {
			for _, t := range availableTypes {
				if len(filteredTypes) >= maxSteps {
					break
				}
				filteredTypes = append(filteredTypes, t)
			}
		}
	}

	// Limit to maxSteps
	if len(filteredTypes) > maxSteps {
		filteredTypes = filteredTypes[:maxSteps]
	}

	steps := make([]*ComboStep, len(filteredTypes))
	for i, captchaType := range filteredTypes {
		stepDifficulty := s.calculateStepDifficulty(difficulty, i, len(filteredTypes))
		step := s.createStep(ctx, captchaType, i, stepDifficulty, req)
		steps[i] = step
	}

	return steps
}

func (s *ComboService) filterAvailableTypes(selectedTypes []string, availableTypes []string) []string {
	if len(selectedTypes) == 0 {
		return selectedTypes
	}

	availableSet := make(map[string]bool)
	for _, t := range availableTypes {
		availableSet[t] = true
	}

	filtered := make([]string, 0, len(selectedTypes))
	for _, t := range selectedTypes {
		if availableSet[t] {
			filtered = append(filtered, t)
		}
	}

	return filtered
}

func (s *ComboService) getAvailableTypes(req *CreateComboRequest) []string {
	baseTypes := []string{
		"slider",
		"click",
		"gesture",
		"video",
		"ar",
		"3d",
		"emoji",
		"semantic",
	}

	if len(req.Types) > 0 {
		available := make([]string, 0)
		for _, t := range req.Types {
			if s.isTypeAvailable(t) {
				available = append(available, t)
			}
		}
		if len(available) > 0 {
			return available
		}
	}

	result := make([]string, 0)
	for _, t := range baseTypes {
		if s.isTypeAvailable(t) {
			result = append(result, t)
		}
	}

	return result
}

func (s *ComboService) isTypeAvailable(captchaType string) bool {
	switch captchaType {
	case "video":
		return s.videoGenerator != nil
	case "ar":
		return s.arGenerator != nil
	case "slider", "click", "gesture", "3d", "emoji", "semantic":
		return true
	default:
		return false
	}
}

func (s *ComboService) createSmartSelector(req *CreateComboRequest) *ComboSelector {
	selector := &ComboSelector{
		weights:       make(map[string]float64),
		sceneTypes:    []string{},
		difficultyCaps: make(map[string]int),
	}

	baseWeights := map[string]float64{
		"slider":    0.8,
		"click":     0.7,
		"gesture":   0.75,
		"video":     0.6,
		"ar":        0.5,
		"3d":        0.55,
		"emoji":     0.7,
		"semantic":  0.65,
	}

	for k, v := range baseWeights {
		selector.weights[k] = v
	}

	selector.difficultyCaps = map[string]int{
		"slider":   5,
		"click":    4,
		"gesture":  4,
		"video":    3,
		"ar":       3,
		"3d":       4,
		"emoji":    3,
		"semantic": 3,
	}

	if req.PreferFast {
		selector.weights["slider"] = 0.95
		selector.weights["click"] = 0.9
		selector.weights["gesture"] = 0.85
	}

	if req.PreferSecure {
		selector.weights["video"] = 0.95
		selector.weights["ar"] = 0.9
		selector.weights["semantic"] = 0.85
	}

	riskMultiplier := 1.0 + req.RiskScore*0.5
	selector.weights["video"] *= riskMultiplier
	selector.weights["ar"] *= riskMultiplier
	selector.weights["semantic"] *= riskMultiplier

	return selector
}

func (sel *ComboSelector) SelectTypes(count, difficulty int, riskScore float64) []string {
	type weightedType struct {
		Type   string
		Weight float64
	}

	types := make([]weightedType, 0)
	for t, w := range sel.weights {
		cap := sel.difficultyCaps[t]
		if difficulty <= cap {
			types = append(types, weightedType{Type: t, Weight: w})
		}
	}

	sort.Slice(types, func(i, j int) bool {
		return types[i].Weight > types[j].Weight
	})

	selected := make([]string, 0, count)
	used := make(map[string]bool)

	for len(selected) < count && len(types) > 0 {
		for _, wt := range types {
			if !used[wt.Type] && len(selected) < count {
				selected = append(selected, wt.Type)
				used[wt.Type] = true
				break
			}
		}

		if len(selected) < count {
			for t := range sel.weights {
				if !used[t] && len(selected) < count {
					selected = append(selected, t)
					used[t] = true
					break
				}
			}
		}
	}

	rand.Shuffle(len(selected), func(i, j int) {
		selected[i], selected[j] = selected[j], selected[i]
	})

	return selected
}

func (s *ComboService) calculateStepDifficulty(baseDifficulty, stepIndex, totalSteps int) int {
	progression := float64(stepIndex) / float64(totalSteps)
	difficulty := int(float64(baseDifficulty) * (1.0 + progression*0.5))

	if difficulty > 5 {
		difficulty = 5
	}

	return difficulty
}

func (s *ComboService) createStep(ctx context.Context, captchaType string, index, difficulty int, req *CreateComboRequest) *ComboStep {
	step := &ComboStep{
		Index:        index,
		Type:         captchaType,
		Difficulty:   difficulty,
		Mandatory:    true,
		Status:       "pending",
		Score:        0,
		MaxScore:     100,
		Attempts:     0,
		MaxAttempts:  3,
		TimeLimit:    60 + difficulty*20,
	}

	switch captchaType {
	case "video":
		step.SubType = "object_count"
		step.Data = s.generateVideoStepData(ctx, difficulty)
		step.SessionID = step.Data.(map[string]interface{})["session_id"].(string)

	case "ar":
		step.SubType = "object_placement"
		step.Data = s.generateARStepData(ctx, difficulty)
		step.SessionID = step.Data.(map[string]interface{})["session_id"].(string)

	case "slider":
		step.SubType = "position"
		step.SessionID = fmt.Sprintf("slider_%d_%d", time.Now().UnixNano(), index)

	case "click":
		step.SubType = "sequence"
		step.SessionID = fmt.Sprintf("click_%d_%d", time.Now().UnixNano(), index)

	case "gesture":
		step.SubType = "pattern"
		step.SessionID = fmt.Sprintf("gesture_%d_%d", time.Now().UnixNano(), index)

	case "3d":
		step.SubType = "rotation"
		step.SessionID = fmt.Sprintf("3d_%d_%d", time.Now().UnixNano(), index)

	case "emoji":
		step.SubType = "selection"
		step.SessionID = fmt.Sprintf("emoji_%d_%d", time.Now().UnixNano(), index)

	case "semantic":
		step.SubType = "understanding"
		step.SessionID = fmt.Sprintf("semantic_%d_%d", time.Now().UnixNano(), index)
	}

	return step
}

func (s *ComboService) generateVideoStepData(ctx context.Context, difficulty int) map[string]interface{} {
	if s.videoGenerator != nil {
		req := &VideoCaptchaRequest{
			Difficulty: difficulty,
		}
		resp, err := s.videoGenerator.Generate(ctx, req)
		if err == nil && resp != nil {
			return map[string]interface{}{
				"session_id":  resp.SessionID,
				"scene_type":  resp.SceneType,
				"question":    resp.Question,
				"options":     resp.Options,
				"difficulty":  resp.Difficulty,
				"video_url":   resp.VideoURL,
			}
		}
	}

	return map[string]interface{}{
		"session_id": fmt.Sprintf("video_%d", time.Now().UnixNano()),
		"scene_type": "object_count",
		"question":   "视频中有多少个物体？",
		"options":    []string{"3", "4", "5", "6"},
		"difficulty": difficulty,
	}
}

func (s *ComboService) generateARStepData(ctx context.Context, difficulty int) map[string]interface{} {
	if s.arGenerator != nil {
		req := &ARCaptchaRequest{
			SceneType:  "object_placement",
			Difficulty: difficulty,
		}
		resp, err := s.arGenerator.Generate(ctx, req)
		if err == nil && resp != nil {
			return map[string]interface{}{
				"session_id":    resp.SessionID,
				"scene_type":    resp.SceneType,
				"scene_config":  resp.SceneConfig,
				"instructions":  resp.Instructions,
				"difficulty":    resp.Difficulty,
				"target_actions": resp.TargetActions,
			}
		}
	}

	return map[string]interface{}{
		"session_id":   fmt.Sprintf("ar_%d", time.Now().UnixNano()),
		"scene_type":   "object_placement",
		"instructions": "请将物体拖动到目标位置",
		"difficulty":   difficulty,
	}
}

func (s *ComboService) calculateRequiredSteps(strategy ComboStrategy, totalSteps int) int {
	switch strategy {
	case ComboStrategyAll:
		return totalSteps
	case ComboStrategyAny:
		return 1
	case ComboStrategyMajority:
		return totalSteps/2 + 1
	case ComboStrategyWeighted:
		return int(math.Ceil(float64(totalSteps) * 0.6))
	default:
		return totalSteps / 2 + 1
	}
}

func (s *ComboService) Verify(ctx context.Context, req *VerifyComboRequest) (*VerifyComboResponse, error) {
	config, err := s.GetConfig(ctx, req.SessionID)
	if err != nil {
		return &VerifyComboResponse{
			Success:  false,
			Score:    0,
			Message:  "会话不存在或已过期",
			CanRetry: false,
		}, nil
	}

	if time.Now().After(config.ExpiredAt) {
		return &VerifyComboResponse{
			Success:  false,
			Score:    0,
			Message:  "会话已过期",
			CanRetry: false,
		}, nil
	}

	passedCount := 0
	failedCount := 0
	totalScore := 0.0
	stepResults := make([]StepResult, 0, len(req.StepResults))
	retrySteps := make([]int, 0)

	for _, result := range req.StepResults {
		if result.StepIndex >= len(config.Steps) {
			continue
		}

		step := config.Steps[result.StepIndex]
		step.Attempts++

		if result.Success {
			passedCount++
			step.Status = "passed"
			step.Score = result.Score
			totalScore += result.Score
		} else {
			step.Status = "failed"
			step.Score = 0
			failedCount++

			if step.Attempts < step.MaxAttempts && step.Mandatory {
				retrySteps = append(retrySteps, result.StepIndex)
			}
		}

		now := time.Now()
		step.CompletedAt = &now
		config.Steps[result.StepIndex] = step
		stepResults = append(stepResults, result)
	}

	config.VerifiedCount = passedCount
	config.FailedCount = failedCount
	config.TotalScore = totalScore

	canRetry := len(retrySteps) > 0
	// For 'all' strategy, if there are failed steps, don't allow retry
	if config.Strategy == ComboStrategyAll && failedCount > 0 {
		canRetry = false
		retrySteps = nil
	}

	success := s.evaluateSuccess(config, passedCount, failedCount)

	if success {
		config.Status = "verified"
		canRetry = false
		retrySteps = nil
	} else if failedCount >= len(config.Steps) {
		config.Status = "failed"
		canRetry = false
		retrySteps = nil
	}

	s.saveConfig(ctx, config)

	message := s.generateResultMessage(config, success, passedCount, failedCount)

	avgScore := 0.0
	if len(stepResults) > 0 {
		avgScore = totalScore / float64(len(stepResults))
	}

	return &VerifyComboResponse{
		Success:     success,
		Score:       avgScore,
		Message:     message,
		PassedSteps: passedCount,
		FailedSteps: failedCount,
		StepResults: stepResults,
		CanRetry:    canRetry,
		RetrySteps:  retrySteps,
	}, nil
}

func (s *ComboService) evaluateSuccess(config *ComboCaptchaConfig, passed, failed int) bool {
	switch config.Strategy {
	case ComboStrategyAll:
		return passed == len(config.Steps)
	case ComboStrategyAny:
		return passed >= 1
	case ComboStrategyMajority:
		return passed >= config.TotalRequired
	case ComboStrategyWeighted:
		required := 0
		for _, step := range config.Steps {
			if step.Mandatory {
				required++
			}
		}
		return passed >= required && config.TotalScore/float64(len(config.Steps)) >= 60
	default:
		return passed >= config.TotalRequired
	}
}

func (s *ComboService) generateResultMessage(config *ComboCaptchaConfig, success bool, passed, failed int) string {
	if success {
		return fmt.Sprintf("验证成功！通过 %d/%d 步", passed, len(config.Steps))
	}

	switch config.Strategy {
	case ComboStrategyAll:
		return fmt.Sprintf("需要全部验证通过，当前通过 %d/%d", passed, len(config.Steps))
	case ComboStrategyAny:
		return "至少需要一项验证通过"
	case ComboStrategyMajority:
		return fmt.Sprintf("需要多数验证通过，当前通过 %d/%d", passed, config.TotalRequired)
	default:
		return fmt.Sprintf("验证失败，通过 %d/%d", passed, len(config.Steps))
	}
}

func (s *ComboService) GetConfig(ctx context.Context, sessionID string) (*ComboCaptchaConfig, error) {
	// First check in-memory store
	if config, ok := s.inMemoryStore[sessionID]; ok {
		if time.Now().Before(config.ExpiredAt) {
			return config, nil
		}
		delete(s.inMemoryStore, sessionID)
	}

	if s.sessionCache != nil {
		data, err := s.sessionCache.GetRaw(ctx, sessionID)
		if err == nil && data != "" {
			var config ComboCaptchaConfig
			if err := json.Unmarshal([]byte(data), &config); err == nil {
				return &config, nil
			}
		}
	}

	return nil, fmt.Errorf("config not found: %s", sessionID)
}

func (s *ComboService) saveConfig(ctx context.Context, config *ComboCaptchaConfig) error {
	// Always save to in-memory store first
	s.inMemoryStore[config.SessionID] = config

	if s.sessionCache != nil {
		data, err := json.Marshal(config)
		if err != nil {
			return err
		}
		remainingTime := time.Until(config.ExpiredAt)
		if remainingTime <= 0 {
			return fmt.Errorf("config expired")
		}
		return s.sessionCache.SetRaw(ctx, config.SessionID, string(data), remainingTime)
	}
	return nil
}

func (s *ComboService) GetStep(ctx context.Context, sessionID string, stepIndex int) (*ComboStep, error) {
	config, err := s.GetConfig(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if stepIndex >= len(config.Steps) {
		return nil, fmt.Errorf("step index out of range")
	}

	step := config.Steps[stepIndex]
	if step.Status == "pending" && step.StartedAt == nil {
		now := time.Now()
		step.StartedAt = &now
		config.Steps[stepIndex] = step
		s.saveConfig(ctx, config)
	}

	return step, nil
}

func (s *ComboService) GetStatus(ctx context.Context, sessionID string) (map[string]interface{}, error) {
	config, err := s.GetConfig(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	status := map[string]interface{}{
		"session_id":    config.SessionID,
		"status":        config.Status,
		"strategy":      config.Strategy,
		"total_steps":   len(config.Steps),
		"current_step":  config.CurrentStep,
		"passed_count":  config.VerifiedCount,
		"failed_count":  config.FailedCount,
		"total_score":   config.TotalScore,
		"remaining_time": time.Until(config.ExpiredAt).Seconds(),
	}

	return status, nil
}

func generateComboSessionIDV2() string {
	return fmt.Sprintf("combo_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}
