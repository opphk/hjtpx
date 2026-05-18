package service

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type DRLPolicyType string

const (
	PolicyTypeQLearning  DRLPolicyType = "q_learning"
	PolicyTypeDQN        DRLPolicyType = "dqn"
	PolicyTypePolicyGrad DRLPolicyType = "policy_gradient"
)

type RiskState struct {
	DeviceScore     float64 `json:"device_score"`
	IPScore         float64 `json:"ip_score"`
	BehaviorScore   float64 `json:"behavior_score"`
	GeoScore        float64 `json:"geo_score"`
	HistoricalScore float64 `json:"historical_score"`
	TimeScore       float64 `json:"time_score"`
	SessionScore    float64 `json:"session_score"`
}

type RiskAction string

const (
	ActionAllow    RiskAction = "allow"
	ActionBlock    RiskAction = "block"
	ActionReview   RiskAction = "review"
	ActionCaptcha  RiskAction = "captcha"
	ActionChallenge RiskAction = "challenge"
)

type DRLPolicy struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	Name          string         `json:"name" gorm:"size:100"`
	PolicyType    DRLPolicyType  `json:"policy_type"`
	StateDim      int            `json:"state_dim"`
	ActionDim     int            `json:"action_dim"`
	LearningRate  float64        `json:"learning_rate"`
	DiscountFactor float64       `json:"discount_factor"`
	ExplorationRate float64      `json:"exploration_rate"`
	QLearningTable string        `json:"q_table" gorm:"type:text"`
	ModelWeights  string         `json:"model_weights" gorm:"type:text"`
	IsActive      bool           `json:"is_active"`
	TrainedAt     *time.Time     `json:"trained_at"`
	Performance   float64        `json:"performance"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type RiskTransition struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	State     string    `json:"state" gorm:"type:text"`
	Action    string    `json:"action"`
	Reward    float64   `json:"reward"`
	NextState string    `json:"next_state" gorm:"type:text"`
	Timestamp time.Time `json:"timestamp"`
}

type RiskExperience struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	StateHash string    `json:"state_hash" gorm:"size:64;index"`
	Action    string    `json:"action" gorm:"size:50"`
	Reward    float64   `json:"reward"`
	NextStateHash string `json:"next_state_hash" gorm:"size:64"`
	Terminal  bool      `json:"terminal"`
	Priority  float64   `json:"priority"`
	Timestamp time.Time `json:"timestamp" gorm:"index"`
}

type DeepRiskEngine struct {
	mu              sync.RWMutex
	policy          *DRLPolicy
	replayBuffer    []RiskExperience
	maxBufferSize   int
	learningRate    float64
	discountFactor  float64
	explorationRate float64
	qTable          map[string]map[string]float64
	recentRewards   []float64
	windowSize      int
}

var drlEngineInstance *DeepRiskEngine
var drlEngineOnce sync.Once

func NewDeepRiskEngine() *DeepRiskEngine {
	drlEngineOnce.Do(func() {
		drlEngineInstance = &DeepRiskEngine{
			maxBufferSize:    100000,
			learningRate:     0.001,
			discountFactor:   0.95,
			explorationRate: 0.1,
			qTable:          make(map[string]map[string]float64),
			replayBuffer:    make([]RiskExperience, 0, 100000),
			recentRewards:   make([]float64, 0, 1000),
			windowSize:      100,
		}
		drlEngineInstance.loadPolicy()
	})
	return drlEngineInstance
}

func (e *DeepRiskEngine) loadPolicy() {
	var policy DRLPolicy
	if err := database.DB.Where("is_active = ?", true).Order("updated_at DESC").First(&policy).Error; err == nil {
		e.mu.Lock()
		e.policy = &policy
		if policy.QLearningTable != "" {
			json.Unmarshal([]byte(policy.QLearningTable), &e.qTable)
		}
		e.mu.Unlock()
	}
}

func (e *DeepRiskEngine) GetStateDim() int {
	return 7
}

func (e *DeepRiskEngine) GetActionDim() int {
	return 5
}

func (e *DeepRiskEngine) ExtractState(ctx context.Context, deviceFingerprint string, ipAddress string, behaviorScore float64, geoScore float64, historicalScore float64, sessionInfo map[string]interface{}) *RiskState {
	state := &RiskState{
		DeviceScore:     100.0,
		IPScore:         100.0,
		BehaviorScore:   behaviorScore,
		GeoScore:        geoScore,
		HistoricalScore: historicalScore,
		TimeScore:       100.0,
		SessionScore:    100.0,
	}

	if deviceFingerprint != "" {
		if cached, err := redis.GetClient().Get(ctx, fmt.Sprintf("device_score:%s", deviceFingerprint)).Result(); err == nil {
			var score float64
			fmt.Sscanf(cached, "%f", &score)
			state.DeviceScore = score
		} else {
			state.DeviceScore = e.calculateDeviceScore(deviceFingerprint)
			redis.GetClient().Set(ctx, fmt.Sprintf("device_score:%s", deviceFingerprint), state.DeviceScore, 10*time.Minute)
		}
	}

	if ipAddress != "" {
		state.IPScore = e.calculateIPScore(ipAddress)
	}

	hour := time.Now().Hour()
	if hour < 2 || hour > 22 {
		state.TimeScore = 70.0
	} else if hour >= 8 && hour <= 18 {
		state.TimeScore = 100.0
	} else {
		state.TimeScore = 85.0
	}

	if sessionInfo != nil {
		if failCount, ok := sessionInfo["fail_count"].(int); ok {
			state.SessionScore = math.Max(0, 100.0-float64(failCount)*20)
		}
		if reqCount, ok := sessionInfo["request_count"].(int); ok {
			if reqCount > 100 {
				state.SessionScore *= 0.7
			}
		}
	}

	state.DeviceScore = math.Max(0, math.Min(100, state.DeviceScore))
	state.IPScore = math.Max(0, math.Min(100, state.IPScore))
	state.BehaviorScore = math.Max(0, math.Min(100, state.BehaviorScore))
	state.GeoScore = math.Max(0, math.Min(100, state.GeoScore))
	state.HistoricalScore = math.Max(0, math.Min(100, state.HistoricalScore))
	state.TimeScore = math.Max(0, math.Min(100, state.TimeScore))
	state.SessionScore = math.Max(0, math.Min(100, state.SessionScore))

	return state
}

func (e *DeepRiskEngine) calculateDeviceScore(fingerprint string) float64 {
	score := 100.0
	patterns := []struct {
		pattern  string
		penalty  float64
	}{
		{"headless", 40},
		{"phantom", 50},
		{"selenium", 45},
		{"puppeteer", 45},
		{"automation", 50},
	}

	for _, p := range patterns {
		if contains(fingerprint, p.pattern) {
			score -= p.penalty
		}
	}

	return score
}

func (e *DeepRiskEngine) calculateIPScore(ipAddress string) float64 {
	score := 100.0

	ipCacheKey := fmt.Sprintf("ip_score:%s", ipAddress)
	ctx := context.Background()
	if cached, err := redis.GetClient().Get(ctx, ipCacheKey).Result(); err == nil {
		fmt.Sscanf(cached, "%f", &score)
		return score
	}

	var blockCount int64
	database.DB.Model(&struct {
		tableName struct{} `gorm:"table:risk_events"`
	}{}).Where("ip_address = ? AND action = 'block' AND created_at > ?", ipAddress, time.Now().Add(-24*time.Hour)).Count(&blockCount)
	score -= float64(blockCount) * 10

	var recentCount int64
	database.DB.Model(&struct {
		tableName struct{} `gorm:"table:risk_events"`
	}{}).Where("ip_address = ? AND created_at > ?", ipAddress, time.Now().Add(-1*time.Hour)).Count(&recentCount)
	if recentCount > 50 {
		score -= 30
	}

	redis.GetClient().Set(ctx, ipCacheKey, score, 5*time.Minute)

	return math.Max(0, score)
}

func (e *DeepRiskEngine) StateToVector(state *RiskState) []float64 {
	return []float64{
		state.DeviceScore / 100.0,
		state.IPScore / 100.0,
		state.BehaviorScore / 100.0,
		state.GeoScore / 100.0,
		state.HistoricalScore / 100.0,
		state.TimeScore / 100.0,
		state.SessionScore / 100.0,
	}
}

func (e *DeepRiskEngine) StateToHash(state *RiskState) string {
	vector := e.StateToVector(state)
	data, _ := json.Marshal(vector)
	return fmt.Sprintf("%x", md5Hash(data))
}

func (e *DeepRiskEngine) SelectAction(state *RiskState) RiskAction {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stateHash := e.StateToHash(state)

	if rand.Float64() < e.explorationRate {
		return e.exploreAction()
	}

	if qValues, exists := e.qTable[stateHash]; exists {
		bestAction := ActionAllow
		bestValue := qValues[string(ActionAllow)]

		actions := []RiskAction{ActionBlock, ActionReview, ActionCaptcha, ActionChallenge}
		for _, action := range actions {
			if value := qValues[string(action)]; value > bestValue {
				bestValue = value
				bestAction = action
			}
		}
		return bestAction
	}

	return e.ruleBasedAction(state)
}

func (e *DeepRiskEngine) exploreAction() RiskAction {
	r := rand.Float64()
	if r < 0.4 {
		return ActionAllow
	} else if r < 0.6 {
		return ActionBlock
	} else if r < 0.8 {
		return ActionReview
	} else {
		return ActionCaptcha
	}
}

func (e *DeepRiskEngine) ruleBasedAction(state *RiskState) RiskAction {
	totalScore := (state.DeviceScore + state.IPScore + state.BehaviorScore + state.GeoScore + state.HistoricalScore + state.TimeScore + state.SessionScore) / 7.0

	switch {
	case totalScore >= 80:
		return ActionAllow
	case totalScore >= 60:
		return ActionCaptcha
	case totalScore >= 40:
		return ActionReview
	case totalScore >= 20:
		return ActionBlock
	default:
		return ActionChallenge
	}
}

func (e *DeepRiskEngine) UpdateQValue(stateHash string, action RiskAction, reward float64, nextStateHash string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.qTable[stateHash]; !exists {
		e.qTable[stateHash] = make(map[string]float64)
		for _, a := range []RiskAction{ActionAllow, ActionBlock, ActionReview, ActionCaptcha, ActionChallenge} {
			e.qTable[stateHash][string(a)] = 0.0
		}
	}

	maxNextQ := 0.0
	if nextQ, exists := e.qTable[nextStateHash]; exists {
		for _, q := range nextQ {
			if q > maxNextQ {
				maxNextQ = q
			}
		}
	}

	currentQ := e.qTable[stateHash][string(action)]
	newQ := currentQ + e.learningRate*(reward+e.discountFactor*maxNextQ-currentQ)
	e.qTable[stateHash][string(action)] = newQ

	e.recentRewards = append(e.recentRewards, reward)
	if len(e.recentRewards) > e.windowSize {
		e.recentRewards = e.recentRewards[1:]
	}
}

func (e *DeepRiskEngine) CalculateReward(state *RiskState, action RiskAction, actualOutcome string, isHuman bool) float64 {
	reward := 0.0

	if action == ActionAllow && actualOutcome == "success" {
		reward += 10.0
	} else if action == ActionAllow && actualOutcome == "failed" {
		reward -= 20.0
	} else if action == ActionBlock && actualOutcome == "blocked_attack" {
		reward += 15.0
	} else if action == ActionBlock && actualOutcome == "blocked_human" {
		reward -= 25.0
	} else if action == ActionCaptcha && isHuman {
		reward += 5.0
	} else if action == ActionCaptcha && !isHuman {
		reward += 8.0
	} else if action == ActionReview {
		reward += 2.0
	}

	reward -= (100 - state.DeviceScore) * 0.05
	reward -= (100 - state.IPScore) * 0.05
	reward -= (100 - state.BehaviorScore) * 0.1

	return reward
}

func (e *DeepRiskEngine) StoreExperience(state *RiskState, action RiskAction, reward float64, nextState *RiskState, terminal bool) {
	experience := RiskExperience{
		StateHash:     e.StateToHash(state),
		Action:        string(action),
		Reward:        reward,
		NextStateHash: e.StateToHash(nextState),
		Terminal:      terminal,
		Priority:      math.Abs(reward),
		Timestamp:     time.Now(),
	}

	e.mu.Lock()
	e.replayBuffer = append(e.replayBuffer, experience)
	if len(e.replayBuffer) > e.maxBufferSize {
		e.replayBuffer = e.replayBuffer[1:]
	}
	e.mu.Unlock()

	ctx := context.Background()
	expData, _ := json.Marshal(experience)
	redis.GetClient().LPush(ctx, "drp:experiences", expData)
	redis.GetClient().LTrim(ctx, "drp:experiences", 0, int64(e.maxBufferSize-1))
}

func (e *DeepRiskEngine) SampleExperiences(batchSize int) []RiskExperience {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.replayBuffer) <= batchSize {
		return e.replayBuffer
	}

	samples := make([]RiskExperience, batchSize)
	indices := rand.Perm(len(e.replayBuffer))
	for i := 0; i < batchSize; i++ {
		samples[i] = e.replayBuffer[indices[i]]
	}
	return samples
}

func (e *DeepRiskEngine) Train(batchSize int) error {
	experiences := e.SampleExperiences(batchSize)
	if len(experiences) == 0 {
		return nil
	}

	for _, exp := range experiences {
		action := RiskAction(exp.Action)
		e.UpdateQValue(exp.StateHash, action, exp.Reward, exp.NextStateHash)
	}

	e.mu.Lock()
	e.explorationRate *= 0.999
	if e.explorationRate < 0.01 {
		e.explorationRate = 0.01
	}
	e.mu.Unlock()

	return nil
}

func (e *DeepRiskEngine) SavePolicy() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	qTableJSON, err := json.Marshal(e.qTable)
	if err != nil {
		return err
	}

	policy := &DRLPolicy{
		Name:           "deep_risk_policy_v1",
		PolicyType:     PolicyTypeQLearning,
		StateDim:       e.GetStateDim(),
		ActionDim:      e.GetActionDim(),
		LearningRate:   e.learningRate,
		DiscountFactor: e.discountFactor,
		ExplorationRate: e.explorationRate,
		QLearningTable: string(qTableJSON),
		IsActive:       true,
		TrainedAt:      &time.Time{},
	}

	now := time.Now()
	policy.TrainedAt = &now

	if e.policy != nil {
		policy.ID = e.policy.ID
		return database.DB.Save(policy).Error
	}

	return database.DB.Create(policy).Error
}

func (e *DeepRiskEngine) GetPerformance() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.recentRewards) == 0 {
		return 0.0
	}

	var total float64
	for _, r := range e.recentRewards {
		total += r
	}
	return total / float64(len(e.recentRewards))
}

func (e *DeepRiskEngine) RecordOutcome(sessionID string, action RiskAction, success bool, latency time.Duration) {
	ctx := context.Background()

	outcome := map[string]interface{}{
		"session_id": sessionID,
		"action":     string(action),
		"success":    success,
		"latency_ms": latency.Milliseconds(),
		"timestamp":  time.Now().Unix(),
	}

	outcomeData, _ := json.Marshal(outcome)
	redis.GetClient().LPush(ctx, "drp:outcomes", outcomeData)
	redis.GetClient().LTrim(ctx, "drp:outcomes", 0, 9999)
}

func (e *DeepRiskEngine) GetOutcomesSummary() map[string]interface{} {
	ctx := context.Background()

	outcomes, _ := redis.GetClient().LRange(ctx, "drp:outcomes", 0, 999).Result()

	var total, success, fail int64
	var totalLatency int64
	actionCounts := make(map[string]int64)
	actionSuccessCounts := make(map[string]int64)

	for _, outcomeStr := range outcomes {
		var outcome map[string]interface{}
		if err := json.Unmarshal([]byte(outcomeStr), &outcome); err != nil {
			continue
		}

		total++
		if latency, ok := outcome["latency_ms"].(float64); ok {
			totalLatency += int64(latency)
		}

		action := fmt.Sprintf("%v", outcome["action"])
		actionCounts[action]++

		if outcome["success"] == true {
			success++
			actionSuccessCounts[action]++
		} else {
			fail++
		}
	}

	actionAccuracy := make(map[string]float64)
	for action, count := range actionCounts {
		if count > 0 {
			actionAccuracy[action] = float64(actionSuccessCounts[action]) / float64(count)
		}
	}

	avgLatency := float64(0)
	if total > 0 {
		avgLatency = float64(totalLatency) / float64(total)
	}

	return map[string]interface{}{
		"total_requests":    total,
		"successful":        success,
		"failed":            fail,
		"accuracy":          float64(success) / float64(total),
		"avg_latency_ms":    avgLatency,
		"action_counts":     actionCounts,
		"action_accuracy":   actionAccuracy,
		"current_exploration": e.explorationRate,
		"policy_performance": e.GetPerformance(),
	}
}

func (e *DeepRiskEngine) AnalyzeRiskProfile(ctx context.Context, fingerprint string, ipAddress string) map[string]interface{} {
	profile := make(map[string]interface{})

	deviceScore := e.calculateDeviceScore(fingerprint)
	ipScore := e.calculateIPScore(ipAddress)

	profile["device_score"] = deviceScore
	profile["ip_score"] = ipScore
	profile["device_risk_factors"] = e.getDeviceRiskFactors(fingerprint)
	profile["ip_risk_factors"] = e.getIPRiskFactors(ipAddress)

	var deviceHistory []map[string]interface{}
	if history, err := redis.GetClient().LRange(ctx, fmt.Sprintf("device:%s:history", fingerprint), 0, 99).Result(); err == nil {
		for _, h := range history {
			var item map[string]interface{}
			if json.Unmarshal([]byte(h), &item) == nil {
				deviceHistory = append(deviceHistory, item)
			}
		}
	}
	profile["device_history"] = deviceHistory

	var ipHistory []map[string]interface{}
	if history, err := redis.GetClient().LRange(ctx, fmt.Sprintf("ip:%s:history", ipAddress), 0, 99).Result(); err == nil {
		for _, h := range history {
			var item map[string]interface{}
			if json.Unmarshal([]byte(h), &item) == nil {
				ipHistory = append(ipHistory, item)
			}
		}
	}
	profile["ip_history"] = ipHistory

	return profile
}

func (e *DeepRiskEngine) getDeviceRiskFactors(fingerprint string) []string {
	var factors []string

	if contains(fingerprint, "headless") {
		factors = append(factors, "检测到无头浏览器特征")
	}
	if contains(fingerprint, "phantom") {
		factors = append(factors, "检测到PhantomJS特征")
	}
	if contains(fingerprint, "selenium") {
		factors = append(factors, "检测到Selenium自动化工具")
	}
	if contains(fingerprint, "puppeteer") {
		factors = append(factors, "检测到Puppeteer特征")
	}
	if contains(fingerprint, "automation") {
		factors = append(factors, "检测到自动化框架特征")
	}

	return factors
}

func (e *DeepRiskEngine) getIPRiskFactors(ipAddress string) []string {
	var factors []string

	ctx := context.Background()

	var blockCount int64
	database.DB.Model(&struct {
		tableName struct{} `gorm:"table:risk_events"`
	}{}).Where("ip_address = ? AND action = 'block' AND created_at > ?", ipAddress, time.Now().Add(-24*time.Hour)).Count(&blockCount)
	if blockCount > 0 {
		factors = append(factors, fmt.Sprintf("24小时内被拦截%d次", blockCount))
	}

	var recentCount int64
	database.DB.Model(&struct {
		tableName struct{} `gorm:"table:risk_events"`
	}{}).Where("ip_address = ? AND created_at > ?", ipAddress, time.Now().Add(-1*time.Hour)).Count(&recentCount)
	if recentCount > 50 {
		factors = append(factors, fmt.Sprintf("1小时内请求数异常：%d次", recentCount))
	}

	if cached, err := redis.GetClient().Get(ctx, fmt.Sprintf("ip:vpn:%s", ipAddress)).Result(); err == nil && cached == "1" {
		factors = append(factors, "检测到VPN使用")
	}

	if cached, err := redis.GetClient().Get(ctx, fmt.Sprintf("ip:tor:%s", ipAddress)).Result(); err == nil && cached == "1" {
		factors = append(factors, "检测到Tor网络出口节点")
	}

	if cached, err := redis.GetClient().Get(ctx, fmt.Sprintf("ip:proxy:%s", ipAddress)).Result(); err == nil && cached == "1" {
		factors = append(factors, "检测到代理服务器")
	}

	return factors
}

func (e *DeepRiskEngine) GetRealTimeRiskScore(ctx context.Context, fingerprint string, ipAddress string, sessionID string) float64 {
	deviceScore := e.calculateDeviceScore(fingerprint)
	ipScore := e.calculateIPScore(ipAddress)

	var behaviorScore float64 = 100.0
	var geoScore float64 = 100.0
	var historicalScore float64 = 100.0

	if sessionData, err := redis.GetClient().Get(ctx, fmt.Sprintf("session:%s", sessionID)).Result(); err == nil {
		var session map[string]interface{}
		if json.Unmarshal([]byte(sessionData), &session) == nil {
			if bs, ok := session["behavior_score"].(float64); ok {
				behaviorScore = bs
			}
		}
	}

	if cached, err := redis.GetClient().Get(ctx, fmt.Sprintf("geo:score:%s", ipAddress)).Result(); err == nil {
		fmt.Sscanf(cached, "%f", &geoScore)
	} else {
		geoScore = e.calculateGeoScore(ipAddress)
		redis.GetClient().Set(ctx, fmt.Sprintf("geo:score:%s", ipAddress), geoScore, 30*time.Minute)
	}

	var recentRiskEvents int64
	database.DB.Model(&struct {
		tableName struct{} `gorm:"table:risk_events"`
	}{}).Where("fingerprint = ? AND created_at > ?", fingerprint, time.Now().Add(-7*24*time.Hour)).Count(&recentRiskEvents)
	historicalScore = math.Max(0, 100.0-float64(recentRiskEvents)*5)

	totalScore := (deviceScore + ipScore + behaviorScore + geoScore + historicalScore) / 5.0

	return totalScore
}

func (e *DeepRiskEngine) calculateGeoScore(ipAddress string) float64 {
	score := 100.0

	ctx := context.Background()

	var suspiciousCountries = map[string]bool{
		"CN": true,
		"RU": true,
		"KR": true,
		"IR": true,
		"NG": true,
	}

	if country, err := redis.GetClient().Get(ctx, fmt.Sprintf("ip:country:%s", ipAddress)).Result(); err == nil {
		if suspiciousCountries[country] {
			score -= 10
		}
	}

	var asn int64
	database.DB.Raw("SELECT COUNT(DISTINCT ip_address) FROM risk_events WHERE ip_address = ? GROUP BY ip_address", ipAddress).Scan(&asn)
	if asn > 1000 {
		score -= 15
	}

	return math.Max(0, score)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func md5Hash(data []byte) []byte {
	h := md5.Sum(data)
	return h[:]
}
