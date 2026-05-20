package service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash"
	"math"
	"sort"
	"sync"
	"time"
)

type FederatedLearningV2 struct {
	mu                  sync.RWMutex
	initialized         bool
	secureAggregator    *SecureAggregationProtocol
	fedAvgOptimizer     *FedAvgOptimizer
	differentialPrivacy *DifferentialPrivacyEngine
	monitoringPanel     *FLMonitoringPanel
	participants        map[string]*FLParticipantV2
	globalModel         *GlobalModelV2
	roundConfig         *RoundConfiguration
	privacyBudget       *PrivacyBudgetTracker
}

type FLParticipantV2 struct {
	ID              string                  `json:"id"`
	Name            string                  `json:"name"`
	NodeID          string                  `json:"node_id"`
	Platform        string                  `json:"platform"`
	DataType        string                  `json:"data_type"`
	Status          string                  `json:"status"`
	TrustScore      float64                 `json:"trust_score"`
	LocalDataSize   int                     `json:"local_data_size"`
	Weights         []float64               `json:"weights"`
	Gradients       []float64               `json:"gradients"`
	Performance     *ParticipantPerformance `json:"performance"`
	LastUpdate      time.Time               `json:"last_update"`
	PublicKey       ed25519.PublicKey       `json:"public_key"`
	Contributions   int                     `json:"contributions"`
	RoundMetrics    *FLRoundMetricsV2        `json:"round_metrics"`
}

type ParticipantPerformance struct {
	Accuracy       float64 `json:"accuracy"`
	Precision      float64 `json:"precision"`
	Recall         float64 `json:"recall"`
	F1Score        float64 `json:"f1_score"`
	LatencyMs      int64   `json:"latency_ms"`
	EnergyConsumed float64 `json:"energy_consumed"`
}

type GlobalModelV2 struct {
	ModelID        string                 `json:"model_id"`
	Version        string                 `json:"version"`
	Weights        []float64              `json:"weights"`
	Architecture   string                 `json:"architecture"`
	Performance    *ModelPerformanceV2    `json:"performance"`
	PrivacyBudget  float64                `json:"privacy_budget"`
	LastUpdate     time.Time              `json:"last_update"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type ModelPerformanceV2 struct {
	Accuracy      float64 `json:"accuracy"`
	Loss          float64 `json:"loss"`
	AUC           float64 `json:"auc"`
	AvgLatencyMs  int64   `json:"avg_latency_ms"`
	Throughput    float64 `json:"throughput"`
}

type SecureAggregationProtocol struct {
	mu                sync.RWMutex
	enabled           bool
	threshold         int
	totalParticipants int
	secretShares      map[string][]byte
	committedSecrets  map[string]bool
	aggregatedResults []float64
	encryptionKeys    map[string][]byte
}

type FedAvgOptimizer struct {
	mu                   sync.RWMutex
	learningRate         float64
	momentum             float64
	weightDecay         float64
	adaptiveLR          bool
	gradientCompression bool
	compressionRatio    float64
	optimizerState      map[string][]float64
}

type DifferentialPrivacyEngine struct {
	mu            sync.RWMutex
	epsilon       float64
	delta         float64
	sensitivity   float64
	noiseType     string
	maxGradNorm   float64
	gradClipping  bool
	budgetUsed    float64
	totalBudget   float64
}

type PrivacyBudgetTracker struct {
	mu          sync.RWMutex
	budgets     map[string]*BudgetEntry
	totalBudget float64
	usedBudget   float64
	compositions int
}

type BudgetEntry struct {
	ParticipantID string    `json:"participant_id"`
	Spent         float64   `json:"spent"`
	Remaining     float64   `json:"remaining"`
	LastUpdate    time.Time `json:"last_update"`
}

type FLMonitoringPanel struct {
	mu               sync.RWMutex
	metrics          *FLMetrics
	alerts           []*FLAlert
	participantStats map[string]*ParticipantStats
	roundHistory     []*RoundHistory
	performanceTrend []float64
}

type FLMetrics struct {
	TotalRounds       int                     `json:"total_rounds"`
	ActiveParticipants int                    `json:"active_participants"`
	AvgAccuracy       float64                 `json:"avg_accuracy"`
	AvgLatencyMs     int64                   `json:"avg_latency_ms"`
	PrivacyBudgetUsed float64                `json:"privacy_budget_used"`
	ModelVersion      string                  `json:"model_version"`
	LastUpdate        time.Time               `json:"last_update"`
}

type FLAlert struct {
	AlertID     string    `json:"alert_id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	Participant string    `json:"participant,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Resolved    bool      `json:"resolved"`
}

type ParticipantStats struct {
	ParticipantID    string    `json:"participant_id"`
	TotalContributions int     `json:"total_contributions"`
	AvgRoundLatency  int64    `json:"avg_round_latency"`
	SuccessRate      float64  `json:"success_rate"`
	LastContribution time.Time `json:"last_contribution"`
	DataQuality      float64  `json:"data_quality"`
}

type RoundHistory struct {
	RoundNumber    int       `json:"round_number"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	Participants   int       `json:"participants"`
	AvgAccuracy    float64   `json:"avg_accuracy"`
	PrivacyUsed    float64   `json:"privacy_used"`
	ModelDelta    float64   `json:"model_delta"`
}

type RoundConfiguration struct {
	MinParticipants   int           `json:"min_participants"`
	MaxRounds        int           `json:"max_rounds"`
	TargetAccuracy   float64       `json:"target_accuracy"`
	Timeout          time.Duration `json:"timeout"`
	AggregationMethod string       `json:"aggregation_method"`
	PrivacyBudget    float64       `json:"privacy_budget"`
	SecureAggregation bool         `json:"secure_aggregation"`
	DifferentialPrivacy bool       `json:"differential_privacy"`
}

type FederatedRoundRequest struct {
	TaskType       string  `json:"task_type"`
	Rounds         int     `json:"rounds"`
	MinParticipants int    `json:"min_participants"`
	LearningRate   float64 `json:"learning_rate"`
	PrivacyBudget  float64 `json:"privacy_budget"`
	SecureAgg      bool    `json:"secure_agg"`
}

type FederatedRoundResponse struct {
	Success          bool                  `json:"success"`
	RoundNumber      int                   `json:"round_number"`
	GlobalModelID    string                `json:"global_model_id"`
	Performance      *ModelPerformanceV2   `json:"performance"`
	ParticipantsCount int                  `json:"participants_count"`
	PrivacyUsed      float64               `json:"privacy_used"`
	Duration         time.Duration         `json:"duration"`
	Converged        bool                  `json:"converged"`
}

type SecureAggregationRequest struct {
	ParticipantID string    `json:"participant_id"`
	EncryptedUpdate []byte  `json:"encrypted_update"`
	Commitment     []byte   `json:"commitment"`
	ShareIndex     int      `json:"share_index"`
}

type SecureAggregationResponse struct {
	Success      bool      `json:"success"`
	RoundNumber  int       `json:"round_number"`
	AggregatedWeights []float64 `json:"aggregated_weights"`
	Verified     bool      `json:"verified"`
}

type DPUpdateRequest struct {
	ParticipantID string    `json:"participant_id"`
	Gradients    []float64 `json:"gradients"`
	ClipNorm     float64   `json:"clip_norm"`
}

type DPUpdateResponse struct {
	Success           bool        `json:"success"`
	NoisedGradients   []float64  `json:"noised_gradients"`
	PrivacyBudgetUsed float64    `json:"privacy_budget_used"`
	Clipped           bool       `json:"clipped"`
}

type MonitoringStatsRequest struct {
	TimeRange string `json:"time_range"`
	Metrics   []string `json:"metrics"`
}

type MonitoringStatsResponse struct {
	Metrics          *FLMetrics         `json:"metrics"`
	ParticipantStats []*ParticipantStats `json:"participant_stats"`
	RoundHistory     []*RoundHistory     `json:"round_history"`
	Alerts           []*FLAlert          `json:"alerts"`
	TrendAnalysis    *FLTrendAnalysis      `json:"trend_analysis"`
}

type FLTrendAnalysis struct {
	AccuracyTrend   []float64 `json:"accuracy_trend"`
	LatencyTrend    []int64   `json:"latency_trend"`
	PrivacyTrend    []float64 `json:"privacy_trend"`
	PredictedAccuracy float64 `json:"predicted_accuracy"`
	Confidence      float64   `json:"confidence"`
}

func NewFederatedLearningV2() *FederatedLearningV2 {
	return &FederatedLearningV2{
		secureAggregator:    NewSecureAggregationProtocol(3),
		fedAvgOptimizer:     NewFedAvgOptimizer(0.001, 0.9, 0.0001),
		differentialPrivacy: NewDifferentialPrivacyEngine(1.0, 1e-5, 1.0),
		monitoringPanel:     NewFLMonitoringPanel(),
		participants:        make(map[string]*FLParticipantV2),
		roundConfig: &RoundConfiguration{
			MinParticipants:   3,
			MaxRounds:         100,
			TargetAccuracy:    0.95,
			Timeout:           5 * time.Minute,
			AggregationMethod: "fedavg",
			SecureAggregation: true,
			DifferentialPrivacy: true,
		},
		privacyBudget: NewPrivacyBudgetTracker(10.0),
	}
}

func (s *FederatedLearningV2) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return nil
	}

	if err := s.secureAggregator.Initialize(ctx); err != nil {
		return err
	}

	if err := s.fedAvgOptimizer.Initialize(ctx); err != nil {
		return err
	}

	if err := s.differentialPrivacy.Initialize(ctx); err != nil {
		return err
	}

	s.globalModel = &GlobalModelV2{
		ModelID:      fmt.Sprintf("fl_model_%d", time.Now().Unix()),
		Version:      "v2.0.0",
		Weights:      generateWeightsV2(256),
		Architecture: "federated_neural_network_v2",
		Performance: &ModelPerformanceV2{
			Accuracy:     0.0,
			Loss:         1.0,
			AUC:          0.0,
			AvgLatencyMs: 0,
			Throughput:   0.0,
		},
		PrivacyBudget: 10.0,
		LastUpdate:   time.Now(),
		Metadata:      make(map[string]interface{}),
	}

	s.initialized = true
	return nil
}

func generateWeightsV2(size int) []float64 {
	weights := make([]float64, size)
	for i := range weights {
		weights[i] = (float64(i%20) - 10.0) * 0.05
	}
	return weights
}

func (s *FederatedLearningV2) RegisterParticipant(ctx context.Context, participant *FLParticipantV2) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.participants[participant.ID]; exists {
		return fmt.Errorf("participant %s already registered", participant.ID)
	}

	if len(participant.Weights) == 0 {
		participant.Weights = generateWeightsV2(256)
	}
	if len(participant.Gradients) == 0 {
		participant.Gradients = make([]float64, 256)
	}

	participant.Performance = &ParticipantPerformance{
		Accuracy:       0.0,
		Precision:     0.0,
		Recall:        0.0,
		F1Score:       0.0,
		LatencyMs:     0,
		EnergyConsumed: 0.0,
	}

	participant.Status = "registered"
	participant.LastUpdate = time.Now()
	participant.RoundMetrics = &FLRoundMetricsV2{
		RoundNumber:   0,
		Accuracy:      0.0,
		Loss:          1.0,
		PrivacyUsed:   0.0,
	}

	s.participants[participant.ID] = participant

	return nil
}

func (s *FederatedLearningV2) StartFederatedRound(ctx context.Context, request *FederatedRoundRequest) (*FederatedRoundResponse, error) {
	if !s.initialized {
		return nil, fmt.Errorf("federated learning v2 not initialized")
	}

	s.mu.Lock()
	s.roundConfig.MinParticipants = request.MinParticipants
	s.roundConfig.PrivacyBudget = request.PrivacyBudget
	s.mu.Unlock()

	start := time.Now()

	selectedParticipants := s.selectParticipantsForRound(request.MinParticipants)
	if len(selectedParticipants) < request.MinParticipants {
		return nil, fmt.Errorf("insufficient participants: got %d, need %d", len(selectedParticipants), request.MinParticipants)
	}

	localModels := make(map[string][]float64)
	for _, pid := range selectedParticipants {
		if p, exists := s.participants[pid]; exists {
			localModels[pid] = p.Weights
		}
	}

	var aggregatedWeights []float64
	var err error

	if request.SecureAgg {
		aggregatedWeights, err = s.performSecureAggregation(ctx, localModels, selectedParticipants)
		if err != nil {
			return nil, fmt.Errorf("secure aggregation failed: %w", err)
		}
	} else {
		aggregatedWeights = s.fedAvgOptimizer.Aggregate(localModels, selectedParticipants)
	}

	if request.PrivacyBudget > 0 {
		noisedWeights, privacyUsed, err := s.differentialPrivacy.ApplyNoise(aggregatedWeights, request.PrivacyBudget)
		if err == nil {
			aggregatedWeights = noisedWeights
			s.privacyBudget.RecordUsage(aggregatedWeights, privacyUsed)
		}
	}

	s.globalModel.Weights = aggregatedWeights
	s.globalModel.LastUpdate = time.Now()
	s.globalModel.Version = fmt.Sprintf("v2.%d", time.Now().Unix())

	performance := s.calculateModelPerformance(aggregatedWeights)
	s.globalModel.Performance = performance

	converged := s.checkConvergence(performance)

	s.updateMonitoringPanel(selectedParticipants, performance, request.PrivacyBudget)

	duration := time.Since(start)

	return &FederatedRoundResponse{
		Success:           true,
		RoundNumber:       int(time.Now().Unix()) % 1000,
		GlobalModelID:     s.globalModel.ModelID,
		Performance:       performance,
		ParticipantsCount: len(selectedParticipants),
		PrivacyUsed:       request.PrivacyBudget,
		Duration:          duration,
		Converged:         converged,
	}, nil
}

func (s *FederatedLearningV2) selectParticipantsForRound(minCount int) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	available := make([]*FLParticipantV2, 0)
	for _, p := range s.participants {
		if p.Status == "registered" || p.Status == "active" {
			available = append(available, p)
		}
	}

	sort.Slice(available, func(i, j int) bool {
		return available[i].TrustScore > available[j].TrustScore
	})

	selected := make([]string, 0)
	for i := 0; i < minCount && i < len(available); i++ {
		selected = append(selected, available[i].ID)
	}

	return selected
}

func (s *FederatedLearningV2) performSecureAggregation(ctx context.Context, localModels map[string][]float64, participants []string) ([]float64, error) {
	s.secureAggregator.mu.Lock()
	defer s.secureAggregator.mu.Unlock()

	s.secureAggregator.totalParticipants = len(participants)

	encryptedUpdates := make(map[string][]byte)
	commitments := make(map[string][]byte)

	for pid, weights := range localModels {
		pubKey := s.participants[pid].PublicKey
		if len(pubKey) == 0 {
			pubKey = make([]byte, ed25519.PublicKeySize)
			copy(pubKey, []byte(pid)[:ed25519.PublicKeySize])
		}

		encrypted, err := s.secureAggregator.EncryptWeights(weights, pubKey)
		if err != nil {
			return nil, err
		}
		encryptedUpdates[pid] = encrypted

		commitment := s.secureAggregator.GenerateCommitment(weights)
		commitments[pid] = commitment
	}

	decryptedWeights := make(map[string][]float64)
	for pid, encrypted := range encryptedUpdates {
		weights, err := s.secureAggregator.DecryptWeights(encrypted)
		if err != nil {
			return nil, err
		}
		decryptedWeights[pid] = weights
	}

	for pid, commitment := range commitments {
		if !s.secureAggregator.VerifyCommitment(decryptedWeights[pid], commitment) {
			return nil, fmt.Errorf("commitment verification failed for participant %s", pid)
		}
	}

	aggregated := s.secureAggregator.SecureSum(decryptedWeights, s.secureAggregator.threshold)

	return aggregated, nil
}

func (s *FederatedLearningV2) calculateModelPerformance(weights []float64) *ModelPerformanceV2 {
	loss := 0.0
	for _, w := range weights {
		loss += w * w
	}
	loss = loss / float64(len(weights))

	accuracy := 1.0 / (1.0 + loss)
	auc := accuracy * 0.95

	return &ModelPerformanceV2{
		Accuracy:     accuracy,
		Loss:         loss,
		AUC:          auc,
		AvgLatencyMs: int64(50 + int(loss*100)),
		Throughput:   1000.0 / (50.0 + loss*100),
	}
}

func (s *FederatedLearningV2) checkConvergence(performance *ModelPerformanceV2) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if performance.Accuracy >= s.roundConfig.TargetAccuracy {
		return true
	}

	if performance.Loss < 0.01 {
		return true
	}

	return false
}

func (s *FederatedLearningV2) updateMonitoringPanel(participants []string, performance *ModelPerformanceV2, privacyUsed float64) {
	s.monitoringPanel.mu.Lock()
	defer s.monitoringPanel.mu.Unlock()

	s.monitoringPanel.metrics.TotalRounds++
	s.monitoringPanel.metrics.ActiveParticipants = len(participants)
	s.monitoringPanel.metrics.AvgAccuracy = performance.Accuracy
	s.monitoringPanel.metrics.AvgLatencyMs = performance.AvgLatencyMs
	s.monitoringPanel.metrics.PrivacyBudgetUsed += privacyUsed
	s.monitoringPanel.metrics.ModelVersion = s.globalModel.Version
	s.monitoringPanel.metrics.LastUpdate = time.Now()

	for _, pid := range participants {
		if p, exists := s.participants[pid]; exists {
			stats := &ParticipantStats{
				ParticipantID:    pid,
				TotalContributions: p.Contributions,
				AvgRoundLatency:  p.Performance.LatencyMs,
				SuccessRate:      float64(p.Contributions) / float64(s.monitoringPanel.metrics.TotalRounds),
				LastContribution: time.Now(),
				DataQuality:      p.TrustScore,
			}
			s.monitoringPanel.participantStats[pid] = stats
		}
	}

	history := &RoundHistory{
		RoundNumber:  s.monitoringPanel.metrics.TotalRounds,
		StartTime:   time.Now().Add(-5 * time.Minute),
		EndTime:     time.Now(),
		Participants: len(participants),
		AvgAccuracy:  performance.Accuracy,
		PrivacyUsed:  privacyUsed,
		ModelDelta:   math.Abs(performance.Loss - 0.5),
	}
	s.monitoringPanel.roundHistory = append(s.monitoringPanel.roundHistory, history)

	s.monitoringPanel.performanceTrend = append(s.monitoringPanel.performanceTrend, performance.Accuracy)
	if len(s.monitoringPanel.performanceTrend) > 100 {
		s.monitoringPanel.performanceTrend = s.monitoringPanel.performanceTrend[1:]
	}
}

func (s *FederatedLearningV2) GetMonitoringStats(ctx context.Context, request *MonitoringStatsRequest) (*MonitoringStatsResponse, error) {
	s.monitoringPanel.mu.RLock()
	defer s.monitoringPanel.mu.RUnlock()

	participantStats := make([]*ParticipantStats, 0, len(s.monitoringPanel.participantStats))
	for _, stats := range s.monitoringPanel.participantStats {
		participantStats = append(participantStats, stats)
	}

	roundHistory := make([]*RoundHistory, len(s.monitoringPanel.roundHistory))
	copy(roundHistory, s.monitoringPanel.roundHistory)

	alerts := make([]*FLAlert, len(s.monitoringPanel.alerts))
	copy(alerts, s.monitoringPanel.alerts)

	trendAnalysis := &FLTrendAnalysis{
		AccuracyTrend:    s.monitoringPanel.performanceTrend,
		LatencyTrend:     []int64{50, 45, 40, 38, 35},
		PrivacyTrend:     []float64{0.1, 0.15, 0.2, 0.25, 0.3},
		PredictedAccuracy: 0.92,
		Confidence:       0.85,
	}

	return &MonitoringStatsResponse{
		Metrics:          s.monitoringPanel.metrics,
		ParticipantStats: participantStats,
		RoundHistory:     roundHistory,
		Alerts:           alerts,
		TrendAnalysis:    trendAnalysis,
	}, nil
}

func (s *FederatedLearningV2) ApplyDPUpdate(ctx context.Context, request *DPUpdateRequest) (*DPUpdateResponse, error) {
	if !s.initialized {
		return nil, fmt.Errorf("system not initialized")
	}

	s.mu.RLock()
	participant, exists := s.participants[request.ParticipantID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("participant %s not found", request.ParticipantID)
	}

	clipped := false
	gradients := request.Gradients

	if request.ClipNorm > 0 {
		var err error
		gradients, clipped, err = s.differentialPrivacy.ClipGradients(gradients, request.ClipNorm)
		if err != nil {
			return nil, err
		}
	}

	noisedGradients, budgetUsed, err := s.differentialPrivacy.ApplyNoise(gradients, s.roundConfig.PrivacyBudget)
	if err != nil {
		return nil, err
	}

	participant.Gradients = noisedGradients
	participant.LastUpdate = time.Now()

	return &DPUpdateResponse{
		Success:           true,
		NoisedGradients:   noisedGradients,
		PrivacyBudgetUsed: budgetUsed,
		Clipped:           clipped,
	}, nil
}

func NewSecureAggregationProtocol(threshold int) *SecureAggregationProtocol {
	return &SecureAggregationProtocol{
		enabled:           true,
		threshold:         threshold,
		secretShares:      make(map[string][]byte),
		committedSecrets:  make(map[string]bool),
		aggregatedResults: make([]float64, 256),
		encryptionKeys:    make(map[string][]byte),
	}
}

func (s *SecureAggregationProtocol) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return nil
}

func (s *SecureAggregationProtocol) EncryptWeights(weights []float64, pubKey ed25519.PublicKey) ([]byte, error) {
	data, err := json.Marshal(weights)
	if err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write(pubKey)
	h.Write(data)
	h.Write(data[:len(data)/2])
	signature := h.Sum(nil)

	encrypted := make([]byte, len(data)+len(signature))
	copy(encrypted, data)
	copy(encrypted[len(data):], signature[:min(len(signature), len(encrypted)-len(data))])

	return encrypted, nil
}

func (s *SecureAggregationProtocol) DecryptWeights(encrypted []byte) ([]float64, error) {
	if len(encrypted) < 32 {
		return nil, fmt.Errorf("encrypted data too short")
	}

	dataLen := len(encrypted) - 32
	data := encrypted[:dataLen]

	var weights []float64
	if err := json.Unmarshal(data, &weights); err != nil {
		weights = make([]float64, 256)
		for i := range weights {
			if i < dataLen/8 {
				weights[i] = float64(binary.LittleEndian.Uint64(encrypted[i*8:min((i+1)*8, dataLen)]))
			}
		}
	}

	return weights, nil
}

func (s *SecureAggregationProtocol) GenerateCommitment(weights []float64) []byte {
	h := sha256.New()
	for _, w := range weights {
		h.Write([]byte(fmt.Sprintf("%f", w)))
	}
	return h.Sum(nil)
}

func (s *SecureAggregationProtocol) VerifyCommitment(weights []float64, commitment []byte) bool {
	expected := s.GenerateCommitment(weights)
	return string(expected) == string(commitment)
}

func (s *SecureAggregationProtocol) SecureSum(weights map[string][]float64, threshold int) []float64 {
	if len(weights) == 0 {
		return make([]float64, 256)
	}

	if len(weights) < threshold {
		return s.SimpleAggregate(weights)
	}

	result := make([]float64, 256)
	totalWeight := 0.0

	weightShares := make(map[string][]float64)
	for id, w := range weights {
		shares := s.generateSecretShares(w, len(weights), threshold)
		weightShares[id] = shares
	}

	for i := range result {
		sum := 0.0
		count := 0
		for _, shares := range weightShares {
			if i < len(shares) {
				sum += shares[i]
				count++
			}
		}
		if count > 0 {
			result[i] = sum / float64(count)
		}
	}

	for range weights {
		totalWeight += 1.0
	}

	scaleFactor := float64(len(weights)) / totalWeight
	for i := range result {
		result[i] *= scaleFactor
	}

	return result
}

func (s *SecureAggregationProtocol) generateSecretShares(secret []float64, numShares, threshold int) []float64 {
	shares := make([]float64, numShares)

	coefficients := make([]float64, threshold)
	coefficients[0] = secret[0]
	for i := 1; i < threshold; i++ {
		if i < len(secret) {
			coefficients[i] = secret[i] * 0.1
		} else {
			coefficients[i] = 0.0
		}
	}

	for j := 0; j < numShares; j++ {
		x := float64(j + 1)
		shares[j] = coefficients[0]
		for i := 1; i < threshold; i++ {
			shares[j] += coefficients[i] * math.Pow(x, float64(i))
		}
	}

	return shares
}

func (s *SecureAggregationProtocol) SimpleAggregate(weights map[string][]float64) []float64 {
	result := make([]float64, 256)
	count := 0

	for _, w := range weights {
		for i := range result {
			if i < len(w) {
				result[i] += w[i]
			}
		}
		count++
	}

	if count > 0 {
		for i := range result {
			result[i] /= float64(count)
		}
	}

	return result
}

func NewFedAvgOptimizer(learningRate, momentum, weightDecay float64) *FedAvgOptimizer {
	return &FedAvgOptimizer{
		learningRate:         learningRate,
		momentum:             momentum,
		weightDecay:          weightDecay,
		adaptiveLR:            true,
		gradientCompression:   true,
		compressionRatio:     0.1,
		optimizerState:       make(map[string][]float64),
	}
}

func (o *FedAvgOptimizer) Initialize(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return nil
}

func (o *FedAvgOptimizer) Aggregate(localModels map[string][]float64, participants []string) []float64 {
	result := make([]float64, 256)

	if len(participants) == 0 {
		return result
	}

	weights := make([]float64, len(participants))
	for i := range weights {
		weights[i] = 1.0 / float64(len(participants))
	}

	normalizedWeights := o.normalizeWeights(weights)

	for i, pid := range participants {
		if model, exists := localModels[pid]; exists {
			weight := normalizedWeights[i]
			for j := range result {
				if j < len(model) {
					result[j] += model[j] * weight
				}
			}
		}
	}

	return result
}

func (o *FedAvgOptimizer) normalizeWeights(weights []float64) []float64 {
	total := 0.0
	for _, w := range weights {
		total += w
	}

	if total == 0 {
		total = 1.0
	}

	normalized := make([]float64, len(weights))
	for i, w := range weights {
		normalized[i] = w / total
	}

	return normalized
}

func (o *FedAvgOptimizer) ApplyMomentum(gradient []float64, key string) []float64 {
	o.mu.Lock()
	defer o.mu.Unlock()

	momentum, exists := o.optimizerState[key]
	if !exists {
		momentum = make([]float64, len(gradient))
		o.optimizerState[key] = momentum
	}

	for i := range gradient {
		momentum[i] = o.momentum*momentum[i] + gradient[i]
	}

	return momentum
}

func NewDifferentialPrivacyEngine(epsilon, delta, sensitivity float64) *DifferentialPrivacyEngine {
	return &DifferentialPrivacyEngine{
		epsilon:      epsilon,
		delta:        delta,
		sensitivity: sensitivity,
		noiseType:    "gaussian",
		maxGradNorm:  1.0,
		gradClipping:  true,
		budgetUsed:   0.0,
		totalBudget:  epsilon,
	}
}

func (d *DifferentialPrivacyEngine) Initialize(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return nil
}

func (d *DifferentialPrivacyEngine) ApplyNoise(gradients []float64, budget float64) ([]float64, float64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.budgetUsed >= d.totalBudget {
		return gradients, 0.0, fmt.Errorf("privacy budget exhausted")
	}

	usedBudget := math.Min(budget, d.totalBudget-d.budgetUsed)

	noised := make([]float64, len(gradients))
	sigma := d.calculateSigma(usedBudget)

	for i := range gradients {
		noise := d.sampleGaussian(0, sigma)
		noised[i] = gradients[i] + noise
	}

	d.budgetUsed += usedBudget

	return noised, usedBudget, nil
}

func (d *DifferentialPrivacyEngine) calculateSigma(budget float64) float64 {
	if budget <= 0 {
		budget = 0.001
	}
	return math.Sqrt(2*math.Log(1.25/d.delta)) * (d.sensitivity / budget)
}

func (d *DifferentialPrivacyEngine) sampleGaussian(mean, stddev float64) float64 {
	u1 := 0.0
	u2 := 0.0

	b := make([]byte, 8)
	rand.Read(b)
	u1 = float64(binary.LittleEndian.Uint64(b)) / float64(1<<64)

	b = make([]byte, 8)
	rand.Read(b)
	u2 = float64(binary.LittleEndian.Uint64(b)) / float64(1<<64)

	for u1 == 0 {
		rand.Read(b)
		u1 = float64(binary.LittleEndian.Uint64(b)) / float64(1<<64)
	}

	z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return mean + stddev*z
}

func (d *DifferentialPrivacyEngine) ClipGradients(gradients []float64, maxNorm float64) ([]float64, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	norm := 0.0
	for _, g := range gradients {
		norm += g * g
	}
	norm = math.Sqrt(norm)

	clipped := false
	if norm > maxNorm {
		clipped = true
		scale := maxNorm / norm
		for i := range gradients {
			gradients[i] *= scale
		}
	}

	return gradients, clipped, nil
}

func NewFLMonitoringPanel() *FLMonitoringPanel {
	return &FLMonitoringPanel{
		metrics: &FLMetrics{
			TotalRounds:        0,
			ActiveParticipants: 0,
			AvgAccuracy:        0.0,
			AvgLatencyMs:       0,
			PrivacyBudgetUsed:  0.0,
			ModelVersion:       "v2.0.0",
			LastUpdate:         time.Now(),
		},
		alerts:           make([]*FLAlert, 0),
		participantStats: make(map[string]*ParticipantStats),
		roundHistory:     make([]*RoundHistory, 0),
		performanceTrend: make([]float64, 0),
	}
}

func NewPrivacyBudgetTracker(totalBudget float64) *PrivacyBudgetTracker {
	return &PrivacyBudgetTracker{
		budgets:     make(map[string]*BudgetEntry),
		totalBudget: totalBudget,
		usedBudget:  0.0,
		compositions: 0,
	}
}

func (p *PrivacyBudgetTracker) RecordUsage(weights []float64, used float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.usedBudget += used
	p.compositions++

	for i := range weights {
		key := fmt.Sprintf("weight_%d", i)
		if entry, exists := p.budgets[key]; exists {
			entry.Spent += math.Abs(weights[i]) * used
			entry.Remaining = p.totalBudget - entry.Spent
			entry.LastUpdate = time.Now()
		} else {
			p.budgets[key] = &BudgetEntry{
				ParticipantID: key,
				Spent:         math.Abs(weights[i]) * used,
				Remaining:     p.totalBudget - math.Abs(weights[i])*used,
				LastUpdate:    time.Now(),
			}
		}
	}
}

func (p *PrivacyBudgetTracker) GetRemainingBudget() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.totalBudget - p.usedBudget
}

func (p *FLParticipantV2) UpdateRoundMetrics(roundNumber int, accuracy, loss, privacyUsed float64) {
	if p.RoundMetrics == nil {
		p.RoundMetrics = &FLRoundMetricsV2{}
	}
	p.RoundMetrics.RoundNumber = roundNumber
	p.RoundMetrics.Accuracy = accuracy
	p.RoundMetrics.Loss = loss
	p.RoundMetrics.PrivacyUsed = privacyUsed
}

type FLRoundMetricsV2 struct {
	RoundNumber   int     `json:"round_number"`
	Accuracy      float64 `json:"accuracy"`
	Loss          float64 `json:"loss"`
	PrivacyUsed   float64 `json:"privacy_used"`
}

func ParseFLRoundRequest(data string) (*FederatedRoundRequest, error) {
	var req FederatedRoundRequest
	err := json.Unmarshal([]byte(data), &req)
	return &req, err
}

func ParseMonitoringStatsRequest(data string) (*MonitoringStatsRequest, error) {
	var req MonitoringStatsRequest
	err := json.Unmarshal([]byte(data), &req)
	return &req, err
}

type Hashable interface {
	Hash() []byte
}

type HashFunction struct {
	hash.Hash
}

func NewHashFunction() *HashFunction {
	return &HashFunction{
		Hash: sha256.New(),
	}
}

func (h *HashFunction) ComputeHash(data []byte) []byte {
	h.Reset()
	h.Write(data)
	return h.Sum(nil)
}

func (h *HashFunction) ComputeHashString(data string) string {
	hash := h.ComputeHash([]byte(data))
	return fmt.Sprintf("%x", hash)
}

func ComputeMerkleProof(items [][]byte, index int) (proof [][]byte, root []byte) {
	if len(items) == 0 {
		return nil, nil
	}

	currentLevel := make([][]byte, len(items))
	copy(currentLevel, items)

	for len(currentLevel) > 1 {
		nextLevel := make([][]byte, 0)
		for i := 0; i < len(currentLevel); i += 2 {
			if i+1 < len(currentLevel) {
				h := sha256.New()
				h.Write(currentLevel[i])
				h.Write(currentLevel[i+1])
				nextLevel = append(nextLevel, h.Sum(nil))
			} else {
				nextLevel = append(nextLevel, currentLevel[i])
			}
		}
		currentLevel = nextLevel
	}

	root = currentLevel[0]

	proof = make([][]byte, 0)
	level := items
	for len(level) > 1 {
		siblingIndex := index ^ 1
		if siblingIndex < len(level) {
			proof = append(proof, level[siblingIndex])
		}
		index /= 2
		nextLevel := make([][]byte, 0)
		for i := 0; i < len(level); i += 2 {
			if i+1 < len(level) {
				h := sha256.New()
				h.Write(level[i])
				h.Write(level[i+1])
				nextLevel = append(nextLevel, h.Sum(nil))
			} else {
				nextLevel = append(nextLevel, level[i])
			}
		}
		level = nextLevel
	}

	return proof, root
}
