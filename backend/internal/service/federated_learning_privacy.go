package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"
)

type FederatedLearningSystem struct {
	mu              sync.RWMutex
	participants    map[string]*FLParticipant
	globalModel     *GlobalModel
	featureExtractor *FederatedFeatureExtractor
	privacyEngine   *PrivacyProtectionEngine
	coordinationService *FLCoordinationService
	initialized     bool
}

type FLParticipant struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Platform     string                 `json:"platform"`
	DataType     string                 `json:"data_type"`
	LocalModel   *LocalModel
	LocalData    *LocalDataset
	Metrics      *ParticipantMetrics
	Status       string                 `json:"status"`
	LastSync     time.Time              `json:"last_sync"`
	TrustScore   float64               `json:"trust_score"`
	Contributions int                  `json:"contributions"`
}

type GlobalModel struct {
	ModelID       string                 `json:"model_id"`
	Version       string                 `json:"version"`
	Weights       []float64             `json:"weights"`
	Architecture  string                 `json:"architecture"`
	Performance   *ModelPerformance      `json:"performance"`
	PrivacyBudget float64               `json:"privacy_budget"`
	LastUpdate    time.Time              `json:"last_update"`
}

type LocalModel struct {
	Weights    []float64          `json:"weights"`
	Gradients  []float64          `json:"gradients"`
	UpdateCount int               `json:"update_count"`
	LastUpdate time.Time          `json:"last_update"`
}

type LocalDataset struct {
	SampleCount   int                  `json:"sample_count"`
	Features      [][]float64         `json:"features"`
	Labels       []float64           `json:"labels"`
	Metadata     map[string]interface{} `json:"metadata"`
	QualityScore float64             `json:"quality_score"`
}

type ParticipantMetrics struct {
	Accuracy      float64             `json:"accuracy"`
	Precision     float64             `json:"precision"`
	Recall        float64             `json:"recall"`
	F1Score      float64             `json:"f1_score"`
	Latency      time.Duration       `json:"latency"`
	DataQuality  float64             `json:"data_quality"`
}

type FederatedFeatureExtractor struct {
	mu           sync.RWMutex
	extractionRules map[string]*ExtractionRule
	aggregators   map[string]*FeatureAggregator
}

type ExtractionRule struct {
	RuleID       string   `json:"rule_id"`
	FeatureName  string   `json:"feature_name"`
	TransformType string  `json:"transform_type"`
	Parameters   map[string]float64 `json:"parameters"`
	PrivacyBudget float64 `json:"privacy_budget"`
}

type FeatureAggregator struct {
	AggregatorID string               `json:"aggregator_id"`
	Method      string                `json:"method"`
	Weights     map[string]float64   `json:"weights"`
	Threshold   float64               `json:"threshold"`
}

type PrivacyProtectionEngine struct {
	mu              sync.RWMutex
	mechanisms      map[string]PrivacyMechanism
	differentialPrivacy *DifferentialPrivacy
	secureAggregation *SecureAggregation
}

type PrivacyMechanism interface {
	Apply(data []float64) ([]float64, error)
	GetPrivacyBudget() float64
}

type DifferentialPrivacy struct {
	epsilon       float64
	delta         float64
	sensitivity   float64
	noiseType     string
}

type SecureAggregation struct {
	encryptionEnabled bool
	secureSumProtocol bool
	thresholdScheme   bool
}

type FLCoordinationService struct {
	mu            sync.RWMutex
	rounds        int
	currentRound  int
	minParticipants int
	aggregationStrategy string
	convergenceThreshold float64
}

type FederatedRoundResult struct {
	RoundNumber       int                   `json:"round_number"`
	ParticipatingNodes []string             `json:"participating_nodes"`
	AggregatedModel   *GlobalModel          `json:"aggregated_model"`
	PerformanceMetrics *RoundMetrics        `json:"performance_metrics"`
	PrivacyBudgetUsed float64               `json:"privacy_budget_used"`
	Duration          time.Duration         `json:"duration"`
	Converged         bool                  `json:"converged"`
}

type RoundMetrics struct {
	AvgAccuracy  float64            `json:"avg_accuracy"`
	AvgLoss      float64            `json:"avg_loss"`
	StdDeviation float64            `json:"std_deviation"`
	BestNode     string             `json:"best_node"`
	WorstNode    string             `json:"worst_node"`
}

type CrossPlatformAnalysis struct {
	mu            sync.RWMutex
	platforms     map[string]*PlatformData
	correlations  map[string]float64
	anomalies     []AnomalyDetection
}

type PlatformData struct {
	PlatformID   string              `json:"platform_id"`
	Features     map[string]float64  `json:"features"`
	Behaviors    []BehaviorPattern   `json:"behaviors"`
	TrustScore   float64            `json:"trust_score"`
	LastUpdate   time.Time           `json:"last_update"`
}

type BehaviorPattern struct {
	PatternID    string                 `json:"pattern_id"`
	Type        string                 `json:"type"`
	Features     []float64             `json:"features"`
	Frequency   float64               `json:"frequency"`
	RiskScore   float64               `json:"risk_score"`
}

type AnomalyDetection struct {
	AnomalyID    string                 `json:"anomaly_id"`
	Type        string                 `json:"type"`
	Severity    float64               `json:"severity"`
	Description string                 `json:"description"`
	AffectedPlatforms []string         `json:"affected_platforms"`
	Timestamp   time.Time             `json:"timestamp"`
}

type FederatedTrainingRequest struct {
	TaskType       string                `json:"task_type"`
	Rounds         int                   `json:"rounds"`
	MinParticipants int                  `json:"min_participants"`
	LearningRate   float64              `json:"learning_rate"`
	PrivacyBudget  float64              `json:"privacy_budget"`
}

type FederatedTrainingResponse struct {
	Success       bool                  `json:"success"`
	Result        *FederatedRoundResult `json:"result"`
	GlobalModelID string                `json:"global_model_id"`
}

type FeatureExtractionRequest struct {
	ParticipantID string                  `json:"participant_id"`
	DataType     string                  `json:"data_type"`
	Features     []string                `json:"features"`
	PrivacyLevel string                  `json:"privacy_level"`
}

type FeatureExtractionResponse struct {
	Success       bool                  `json:"success"`
	ExtractedFeatures []ExtractedFeature `json:"extracted_features"`
	PrivacyBudgetUsed float64           `json:"privacy_budget_used"`
}

type ExtractedFeature struct {
	Name        string                 `json:"name"`
	Value       float64               `json:"value"`
	PrivacyNoise float64              `json:"privacy_noise"`
	Quality     float64               `json:"quality"`
}

func NewFederatedLearningSystem() *FederatedLearningSystem {
	return &FederatedLearningSystem{
		participants:    make(map[string]*FLParticipant),
		featureExtractor: NewFederatedFeatureExtractor(),
		privacyEngine:   NewPrivacyProtectionEngine(),
		coordinationService: NewFLCoordinationService(),
	}
}

func (s *FederatedLearningSystem) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return nil
	}

	if err := s.featureExtractor.Initialize(ctx); err != nil {
		return err
	}

	if err := s.privacyEngine.Initialize(ctx); err != nil {
		return err
	}

	if err := s.coordinationService.Initialize(ctx); err != nil {
		return err
	}

	s.globalModel = &GlobalModel{
		ModelID:      fmt.Sprintf("global_model_%d", time.Now().Unix()),
		Version:      "v1.0",
		Weights:      make([]float64, 128),
		Architecture: "federated_neural_network",
		Performance: &ModelPerformance{
			Accuracy: 0.0,
			Loss: 1.0,
		},
		PrivacyBudget: 10.0,
		LastUpdate:    time.Now(),
	}

	for i := range s.globalModel.Weights {
		s.globalModel.Weights[i] = 0.0
	}

	s.initialized = true
	return nil
}

func (s *FederatedLearningSystem) RegisterParticipant(ctx context.Context, participant *FLParticipant) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.participants[participant.ID]; exists {
		return fmt.Errorf("participant %s already registered", participant.ID)
	}

	participant.LocalModel = &LocalModel{
		Weights:    make([]float64, 128),
		Gradients: make([]float64, 128),
		UpdateCount: 0,
		LastUpdate: time.Now(),
	}

	participant.Metrics = &ParticipantMetrics{
		Accuracy:     0.0,
		Precision:    0.0,
		Recall:       0.0,
		F1Score:     0.0,
		DataQuality:  0.8,
	}

	participant.Status = "registered"

	s.participants[participant.ID] = participant

	return nil
}

func (s *FederatedLearningSystem) StartFederatedTraining(ctx context.Context, request *FederatedTrainingRequest) (*FederatedTrainingResponse, error) {
	if !s.initialized {
		return nil, fmt.Errorf("system not initialized")
	}

	s.coordinationService.mu.Lock()
	s.coordinationService.minParticipants = request.MinParticipants
	s.coordinationService.aggregationStrategy = "fedavg"
	s.coordinationService.mu.Unlock()

	result := &FederatedRoundResult{
		RoundNumber:        s.coordinationService.currentRound + 1,
		ParticipatingNodes: make([]string, 0),
		AggregatedModel:    &GlobalModel{},
		PerformanceMetrics: &RoundMetrics{},
	}

	start := time.Now()

	selectedParticipants := s.selectParticipants(request.MinParticipants)
	result.ParticipatingNodes = selectedParticipants

	localModels := make(map[string]*LocalModel)
	for _, participantID := range selectedParticipants {
		participant := s.participants[participantID]
		if participant != nil {
			localModels[participantID] = participant.LocalModel
		}
	}

	aggregatedWeights := s.aggregateModels(localModels, selectedParticipants)

	result.AggregatedModel = &GlobalModel{
		ModelID:      s.globalModel.ModelID,
		Version:      fmt.Sprintf("v%d.%d", s.coordinationService.currentRound/10+1, s.coordinationService.currentRound%10),
		Weights:      aggregatedWeights,
		Architecture: s.globalModel.Architecture,
		Performance:  s.calculatePerformance(aggregatedWeights),
		LastUpdate:   time.Now(),
	}

	s.globalModel.Weights = aggregatedWeights
	s.globalModel.Version = result.AggregatedModel.Version
	s.globalModel.LastUpdate = time.Now()

	result.PerformanceMetrics = s.calculateRoundMetrics(localModels, selectedParticipants)
	result.Duration = time.Since(start)
	result.Converged = s.checkConvergence(result.PerformanceMetrics)

	result.PrivacyBudgetUsed = request.PrivacyBudget * float64(len(selectedParticipants)) / float64(len(s.participants))

	s.coordinationService.currentRound++

	return &FederatedTrainingResponse{
		Success:       true,
		Result:        result,
		GlobalModelID: s.globalModel.ModelID,
	}, nil
}

func (s *FederatedLearningSystem) selectParticipants(minCount int) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	selected := make([]string, 0)

	available := make([]string, 0)
	for id, p := range s.participants {
		if p.Status == "registered" || p.Status == "active" {
			available = append(available, id)
		}
	}

	if len(available) <= minCount {
		return available
	}

	for i := 0; i < minCount && i < len(available); i++ {
		idx := i + int(math.Mod(float64(time.Now().UnixNano()), float64(len(available)-i)))
		selected = append(selected, available[idx])
		available = append(available[:idx], available[idx+1:]...)
	}

	return selected
}

func (s *FederatedLearningSystem) aggregateModels(localModels map[string]*LocalModel, participantIDs []string) []float64 {
	weights := make([]float64, 128)

	if len(participantIDs) == 0 {
		return weights
	}

	totalSamples := 0
	for _, id := range participantIDs {
		if model, exists := localModels[id]; exists {
			if model.UpdateCount > 0 {
				totalSamples += model.UpdateCount
			} else {
				totalSamples += 100
			}
		}
	}

	for _, id := range participantIDs {
		if model, exists := localModels[id]; exists {
			weight := 100.0
			if model.UpdateCount > 0 {
				weight = float64(model.UpdateCount)
			}

			normalizedWeight := weight / float64(totalSamples)

			for i := range weights {
				weights[i] += model.Weights[i] * normalizedWeight
			}
		}
	}

	return weights
}

func (s *FederatedLearningSystem) calculatePerformance(weights []float64) *ModelPerformance {
	loss := 0.0
	for _, w := range weights {
		loss += w * w
	}
	loss = loss / float64(len(weights))

	accuracy := 1.0 / (1.0 + loss)

	return &ModelPerformance{
		Accuracy: accuracy,
		Loss:     loss,
	}
}

func (s *FederatedLearningSystem) calculateRoundMetrics(localModels map[string]*LocalModel, participantIDs []string) *RoundMetrics {
	if len(participantIDs) == 0 {
		return &RoundMetrics{}
	}

	accuracies := make([]float64, 0)
	losses := make([]float64, 0)

	for _, id := range participantIDs {
		if model, exists := localModels[id]; exists {
			loss := 0.0
			for _, w := range model.Weights {
				loss += w * w
			}
			loss = loss / float64(len(model.Weights))

			accuracies = append(accuracies, 1.0/(1.0+loss))
			losses = append(losses, loss)
		}
	}

	avgAccuracy := 0.0
	avgLoss := 0.0
	for _, acc := range accuracies {
		avgAccuracy += acc
	}
	for _, loss := range losses {
		avgLoss += loss
	}
	avgAccuracy /= float64(len(accuracies))
	avgLoss /= float64(len(losses))

	stdDev := 0.0
	for _, acc := range accuracies {
		diff := acc - avgAccuracy
		stdDev += diff * diff
	}
	stdDev = math.Sqrt(stdDev / float64(len(accuracies)))

	return &RoundMetrics{
		AvgAccuracy:  avgAccuracy,
		AvgLoss:     avgLoss,
		StdDeviation: stdDev,
		BestNode:    participantIDs[0],
		WorstNode:   participantIDs[len(participantIDs)-1],
	}
}

func (s *FederatedLearningSystem) checkConvergence(metrics *RoundMetrics) bool {
	if metrics.AvgLoss < s.coordinationService.convergenceThreshold {
		return true
	}

	if metrics.StdDeviation < 0.01 {
		return true
	}

	return false
}

func (s *FederatedLearningSystem) SubmitLocalUpdate(ctx context.Context, participantID string, localModel *LocalModel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	participant, exists := s.participants[participantID]
	if !exists {
		return fmt.Errorf("participant %s not found", participantID)
	}

	if localModel != nil && len(localModel.Weights) > 0 {
		participant.LocalModel = localModel
		participant.LocalModel.UpdateCount++
		participant.LocalModel.LastUpdate = time.Now()
	}

	participant.LastSync = time.Now()
	participant.Contributions++

	return nil
}

func NewFederatedFeatureExtractor() *FederatedFeatureExtractor {
	return &FederatedFeatureExtractor{
		extractionRules: make(map[string]*ExtractionRule),
		aggregators:     make(map[string]*FeatureAggregator),
	}
}

func (e *FederatedFeatureExtractor) Initialize(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.extractionRules["mouse_velocity"] = &ExtractionRule{
		RuleID:        "rule_mouse_velocity",
		FeatureName:  "mouse_velocity",
		TransformType: "statistical",
		Parameters:   map[string]float64{"window_size": 10.0, "threshold": 5.0},
		PrivacyBudget: 0.5,
	}

	e.extractionRules["click_timing"] = &ExtractionRule{
		RuleID:        "rule_click_timing",
		FeatureName:  "click_timing",
		TransformType: "temporal",
		Parameters:   map[string]float64{"granularity": 100.0},
		PrivacyBudget: 0.3,
	}

	e.extractionRules["scroll_pattern"] = &ExtractionRule{
		RuleID:        "rule_scroll_pattern",
		FeatureName:  "scroll_pattern",
		TransformType: "sequence",
		Parameters:   map[string]float64{"sequence_length": 20.0},
		PrivacyBudget: 0.4,
	}

	e.aggregators["weighted_mean"] = &FeatureAggregator{
		AggregatorID: "agg_weighted_mean",
		Method:      "weighted_mean",
		Weights:     map[string]float64{"default": 1.0},
		Threshold:   0.5,
	}

	return nil
}

func (e *FederatedFeatureExtractor) ExtractFeatures(ctx context.Context, request *FeatureExtractionRequest) (*FeatureExtractionResponse, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	extractedFeatures := make([]ExtractedFeature, 0)
	totalPrivacyBudget := 0.0

	for _, featureName := range request.Features {
		rule, exists := e.extractionRules[featureName]
		if !exists {
			continue
		}

		noise := 0.0
		switch request.PrivacyLevel {
		case "high":
			noise = rule.PrivacyBudget * 2.0
		case "medium":
			noise = rule.PrivacyBudget * 1.0
		case "low":
			noise = rule.PrivacyBudget * 0.5
		default:
			noise = rule.PrivacyBudget
		}

		feature := ExtractedFeature{
			Name:         featureName,
			Value:        math.Sin(float64(time.Now().UnixNano())) * 0.5,
			PrivacyNoise: noise,
			Quality:      1.0 - noise,
		}

		extractedFeatures = append(extractedFeatures, feature)
		totalPrivacyBudget += noise
	}

	return &FeatureExtractionResponse{
		Success:           true,
		ExtractedFeatures: extractedFeatures,
		PrivacyBudgetUsed: totalPrivacyBudget,
	}, nil
}

func NewPrivacyProtectionEngine() *PrivacyProtectionEngine {
	return &PrivacyProtectionEngine{
		mechanisms:          make(map[string]PrivacyMechanism),
		differentialPrivacy: &DifferentialPrivacy{epsilon: 1.0, delta: 1e-5, sensitivity: 1.0},
		secureAggregation:   &SecureAggregation{encryptionEnabled: true, secureSumProtocol: true},
	}
}

func (p *PrivacyProtectionEngine) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.mechanisms["gaussian"] = &GaussianMechanism{epsilon: 1.0, delta: 1e-5, sensitivity: 1.0}
	p.mechanisms["laplace"] = &LaplaceMechanism{epsilon: 1.0, sensitivity: 1.0}

	return nil
}

func (p *PrivacyProtectionEngine) ApplyDifferentialPrivacy(data []float64, epsilon, delta float64) ([]float64, error) {
	return applyGaussianNoise(data, epsilon, delta)
}

func (p *PrivacyProtectionEngine) SecureAggregate(updates map[string][]float64) ([]float64, error) {
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates to aggregate")
	}

	result := make([]float64, 128)

	for _, weights := range updates {
		for i := range weights {
			if i < len(result) {
				result[i] += weights[i]
			}
		}
	}

	for i := range result {
		result[i] /= float64(len(updates))
	}

	return result, nil
}

type GaussianMechanism struct {
	epsilon     float64
	delta       float64
	sensitivity float64
}

func (m *GaussianMechanism) Apply(data []float64) ([]float64, error) {
	return applyGaussianNoise(data, m.epsilon, m.delta)
}

func (m *GaussianMechanism) GetPrivacyBudget() float64 {
	return m.epsilon
}

type LaplaceMechanism struct {
	epsilon     float64
	sensitivity float64
}

func (m *LaplaceMechanism) Apply(data []float64) ([]float64, error) {
	return applyLaplaceNoise(data, m.epsilon)
}

func (m *LaplaceMechanism) GetPrivacyBudget() float64 {
	return m.epsilon
}

func applyGaussianNoise(data []float64, epsilon, delta float64) ([]float64, error) {
	sigma := math.Sqrt(2 * math.Log(1.25/delta)) * (1.0 / epsilon)

	result := make([]float64, len(data))
	for i := range data {
		noise := gaussianRandom(0, sigma)
		result[i] = data[i] + noise
	}

	return result, nil
}

func applyLaplaceNoise(data []float64, epsilon float64) ([]float64, error) {
	b := 1.0 / epsilon

	result := make([]float64, len(data))
	for i := range data {
		noise := laplaceRandom(0, b)
		result[i] = data[i] + noise
	}

	return result, nil
}

func gaussianRandom(mean, stddev float64) float64 {
	u1 := math.Random()
	u2 := math.Random()
	z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return mean + stddev*z
}

func laplaceRandom(mean, b float64) float64 {
	u := math.Random() - 0.5
	return mean - b*math.Sign(u)*math.Log(1-2*math.Abs(u))
}

func NewFLCoordinationService() *FLCoordinationService {
	return &FLCoordinationService{
		rounds:                 100,
		currentRound:           0,
		minParticipants:        3,
		aggregationStrategy:    "fedavg",
		convergenceThreshold:   0.01,
	}
}

func (c *FLCoordinationService) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return nil
}

func (s *FederatedLearningSystem) PerformCrossPlatformAnalysis(ctx context.Context) (*CrossPlatformAnalysis, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	analysis := &CrossPlatformAnalysis{
		platforms:    make(map[string]*PlatformData),
		correlations: make(map[string]float64),
		anomalies:    make([]AnomalyDetection, 0),
	}

	for id, participant := range s.participants {
		platform := &PlatformData{
			PlatformID: id,
			Features:   make(map[string]float64),
			Behaviors:  make([]BehaviorPattern, 0),
			TrustScore: participant.TrustScore,
			LastUpdate: participant.LastSync,
		}

		if participant.LocalData != nil {
			platform.Features["sample_count"] = float64(participant.LocalData.SampleCount)
			platform.Features["quality_score"] = participant.LocalData.QualityScore
		}

		platform.Features["contribution_rate"] = float64(participant.Contributions) / float64(s.coordinationService.currentRound+1)

		analysis.platforms[id] = platform
	}

	if len(analysis.platforms) >= 2 {
		platformIDs := make([]string, 0, len(analysis.platforms))
		for id := range analysis.platforms {
			platformIDs = append(platformIDs, id)
		}

		for i := 0; i < len(platformIDs); i++ {
			for j := i + 1; j < len(platformIDs); j++ {
				correlation := s.calculatePlatformCorrelation(analysis.platforms[platformIDs[i]], analysis.platforms[platformIDs[j]])
				key := fmt.Sprintf("%s_%s", platformIDs[i], platformIDs[j])
				analysis.correlations[key] = correlation

				if math.Abs(correlation) > 0.8 {
					anomaly := AnomalyDetection{
						AnomalyID:           fmt.Sprintf("anomaly_%d", len(analysis.anomalies)),
						Type:                "high_correlation",
						Severity:            math.Abs(correlation),
						Description:         fmt.Sprintf("平台 %s 和 %s 之间存在高相关性: %.2f", platformIDs[i], platformIDs[j], correlation),
						AffectedPlatforms:   []string{platformIDs[i], platformIDs[j]},
						Timestamp:           time.Now(),
					}
					analysis.anomalies = append(analysis.anomalies, anomaly)
				}
			}
		}
	}

	return analysis, nil
}

func (s *FederatedLearningSystem) calculatePlatformCorrelation(p1, p2 *PlatformData) float64 {
	commonFeatures := make([]float64, 0)
	commonFeatures2 := make([]float64, 0)

	for key, val1 := range p1.Features {
		if val2, exists := p2.Features[key]; exists {
			commonFeatures = append(commonFeatures, val1)
			commonFeatures2 = append(commonFeatures2, val2)
		}
	}

	if len(commonFeatures) < 2 {
		return 0.0
	}

	mean1 := 0.0
	mean2 := 0.0
	for i := range commonFeatures {
		mean1 += commonFeatures[i]
		mean2 += commonFeatures2[i]
	}
	mean1 /= float64(len(commonFeatures))
	mean2 /= float64(len(commonFeatures))

	covariance := 0.0
	var1 := 0.0
	var2 := 0.0

	for i := range commonFeatures {
		diff1 := commonFeatures[i] - mean1
		diff2 := commonFeatures2[i] - mean2
		covariance += diff1 * diff2
		var1 += diff1 * diff1
		var2 += diff2 * diff2
	}

	if var1 == 0 || var2 == 0 {
		return 0.0
	}

	return covariance / (math.Sqrt(var1) * math.Sqrt(var2))
}

func (s *FederatedLearningSystem) GetGlobalModel(ctx context.Context) (*GlobalModel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.globalModel == nil {
		return nil, fmt.Errorf("global model not initialized")
	}

	return s.globalModel, nil
}

func (s *FederatedLearningSystem) GetParticipantStats(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_participants": len(s.participants),
		"active_participants": 0,
		"total_contributions": 0,
		"avg_trust_score":     0.0,
		"current_round":       s.coordinationService.currentRound,
		"model_version":       s.globalModel.Version,
	}

	var totalTrust float64
	activeCount := 0

	for _, p := range s.participants {
		if p.Status == "active" {
			activeCount++
		}
		stats["total_contributions"] = stats["total_contributions"].(int) + p.Contributions
		totalTrust += p.TrustScore
	}

	stats["active_participants"] = activeCount
	if len(s.participants) > 0 {
		stats["avg_trust_score"] = totalTrust / float64(len(s.participants))
	}

	return stats, nil
}

type ModelPerformance struct {
	Accuracy float64 `json:"accuracy"`
	Loss     float64 `json:"loss"`
}

func ParseFLTrainingRequest(data string) (*FederatedTrainingRequest, error) {
	var req FederatedTrainingRequest
	err := json.Unmarshal([]byte(data), &req)
	return &req, err
}

func ParseFeatureExtractionRequest(data string) (*FeatureExtractionRequest, error) {
	var req FeatureExtractionRequest
	err := json.Unmarshal([]byte(data), &req)
	return &req, err
}
