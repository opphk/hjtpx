package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

type FederatedLearningV20 struct {
	mu                sync.RWMutex
	initialized       bool
	privacyEngine     *EnhancedPrivacyEngine
	aggregationEngine *FederatedAggregationEngine
	monitor           *FLMonitoringPanel
	participants      map[string]*FLParticipantV20
	globalModel       *GlobalModelV20
	secureComms       *SecureCommunicationLayer
}

type FLParticipantV20 struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Platform        string                 `json:"platform"`
	DataType        string                 `json:"data_type"`
	LocalModel      *LocalModelV20
	TrustScore      float64                `json:"trust_score"`
	Contributions   int                    `json:"contributions"`
	Status          string                 `json:"status"`
	LastSync        time.Time              `json:"last_sync"`
	PrivacyBudget   float64                `json:"privacy_budget"`
	Metrics         *ParticipantMetricsV20 `json:"metrics"`
	CommunicationKey []byte               `json:"-"`
}

type LocalModelV20 struct {
	Weights         []float64          `json:"weights"`
	Gradients       []float64          `json:"gradients"`
	UpdateCount     int                `json:"update_count"`
	LastUpdate      time.Time          `json:"last_update"`
	PrunedWeights   []bool             `json:"pruned_weights"`
	QuantizedWeights []int8            `json:"quantized_weights"`
}

type GlobalModelV20 struct {
	ModelID        string                 `json:"model_id"`
	Version        string                 `json:"version"`
	Weights        []float64             `json:"weights"`
	Architecture   string                 `json:"architecture"`
	Performance    *ModelPerformanceV20   `json:"performance"`
	PrivacyBudget  float64               `json:"privacy_budget"`
	LastUpdate     time.Time              `json:"last_update"`
	QuantizationLevel int                `json:"quantization_level"`
	PruningRate    float64               `json:"pruning_rate"`
}

type ParticipantMetricsV20 struct {
	Accuracy       float64            `json:"accuracy"`
	Precision      float64            `json:"precision"`
	Recall         float64            `json:"recall"`
	F1Score       float64            `json:"f1_score"`
	Latency        time.Duration      `json:"latency"`
	DataQuality   float64            `json:"data_quality"`
	PrivacyUsed   float64            `json:"privacy_used"`
	RoundNumber   int                `json:"round_number"`
}

type ModelPerformanceV20 struct {
	Accuracy      float64 `json:"accuracy"`
	Precision     float64 `json:"precision"`
	Recall        float64 `json:"recall"`
	F1Score       float64 `json:"f1_score"`
	Loss          float64 `json:"loss"`
	LatencyMs     float64 `json:"latency_ms"`
	Throughput    float64 `json:"throughput"`
}

type EnhancedPrivacyEngine struct {
	mu               sync.RWMutex
	mechanisms        map[string]PrivacyMechanismV20
	privacyAccountant *PrivacyAccountant
	clippingBounds    float64
	noiseMultiplier   float64
	totalBudget       float64
	spentBudget       float64
}

type PrivacyMechanismV20 interface {
	Apply(data []float64, epsilon, delta float64) ([]float64, error)
	GetPrivacySpend() (float64, float64)
}

type GaussianMechanismV20 struct {
	epsilon     float64
	delta       float64
	sensitivity float64
	spend       float64
}

type LaplaceMechanismV20 struct {
	epsilon     float64
	sensitivity float64
	spend       float64
}

type ExponentialMechanismV20 struct {
	epsilon     float64
	sensitivity float64
	spend       float64
}

type PrivacyAccountant struct {
	mu          sync.RWMutex
	epsilon     float64
	delta       float64
	orders      []float64
	RDP         float64
	composedEPS float64
}

type FederatedAggregationEngine struct {
	mu                  sync.RWMutex
	strategies          map[string]AggregationStrategy
	currentStrategy     string
	convergenceThreshold float64
	adaptiveWeights     bool
	momentumBuffer     []float64
}

type AggregationStrategy interface {
	Aggregate(updates []ModelUpdate, globalWeights []float64) []float64
	GetName() string
}

type FedAvgStrategy struct{}

func (s *FedAvgStrategy) GetName() string {
	return "FedAvg"
}

type FedProxStrategy struct {
	proximalTerm float64
}

func (s *FedProxStrategy) GetName() string {
	return "FedProx"
}

type ScaffoldStrategy struct {
	controlVariates map[string][]float64
}

func (s *ScaffoldStrategy) GetName() string {
	return "SCAFFOLD"
}

type ModelUpdate struct {
	ParticipantID  string
	Weights        []float64
	Gradients      []float64
	SampleCount    int
	TrustScore     float64
	PrivacyBudget  float64
	RoundNumber    int
	Timestamp      time.Time
}

type FLMonitoringPanel struct {
	mu              sync.RWMutex
	metrics         *FLMetrics
	roundsHistory   []RoundMetricsV20
	alerts          []FLAlert
	participantStats map[string]*ParticipantStats
}

type FLMetrics struct {
	TotalRounds        int                    `json:"total_rounds"`
	ActiveParticipants  int                    `json:"active_participants"`
	TotalContributions  int                    `json:"total_contributions"`
	AvgTrustScore      float64                `json:"avg_trust_score"`
	PrivacyBudgetUsed  float64                `json:"privacy_budget_used"`
	ModelAccuracy      float64                `json:"model_accuracy"`
	ModelLoss          float64                `json:"model_loss"`
	AvgLatency         time.Duration          `json:"avg_latency"`
	Throughput         float64                `json:"throughput"`
	LastUpdateTime     time.Time              `json:"last_update_time"`
}

type RoundMetricsV20 struct {
	RoundNumber       int                   `json:"round_number"`
	Timestamp         time.Time             `json:"timestamp"`
	ParticipantsCount  int                  `json:"participants_count"`
	Accuracy          float64               `json:"accuracy"`
	Loss              float64               `json:"loss"`
	PrivacySpend      float64               `json:"privacy_spend"`
	AvgLatency        time.Duration         `json:"avg_latency"`
	Converged         bool                  `json:"converged"`
}

type FLAlert struct {
	AlertID     string    `json:"alert_id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Resolved    bool      `json:"resolved"`
}

type ParticipantStats struct {
	ParticipantID     string          `json:"participant_id"`
	TotalContributions int            `json:"total_contributions"`
	AvgLatency        time.Duration   `json:"avg_latency"`
	SuccessRate       float64         `json:"success_rate"`
	LastContribution  time.Time       `json:"last_contribution"`
	TrustScoreHistory []float64       `json:"trust_score_history"`
}

type SecureCommunicationLayer struct {
	mu              sync.RWMutex
	encryptionKey   []byte
	secureChannels  map[string]*SecureChannel
	authenticatedNodes map[string]bool
}

type SecureChannel struct {
	NodeID      string
	SessionKey  []byte
	Established bool
	LastActive  time.Time
}

func NewFederatedLearningV20() *FederatedLearningV20 {
	return &FederatedLearningV20{
		privacyEngine:     NewEnhancedPrivacyEngine(),
		aggregationEngine: NewFederatedAggregationEngine(),
		monitor:           NewFLMonitoringPanel(),
		participants:      make(map[string]*FLParticipantV20),
		secureComms:       NewSecureCommunicationLayer(),
	}
}

func (s *FederatedLearningV20) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return nil
	}

	if err := s.privacyEngine.Initialize(ctx); err != nil {
		return err
	}

	if err := s.aggregationEngine.Initialize(ctx); err != nil {
		return err
	}

	s.globalModel = &GlobalModelV20{
		ModelID:           fmt.Sprintf("fl_model_%d", time.Now().Unix()),
		Version:           "v1.0",
		Weights:           make([]float64, 256),
		Architecture:      "federated_v20",
		Performance:       &ModelPerformanceV20{},
		PrivacyBudget:     10.0,
		LastUpdate:        time.Now(),
		QuantizationLevel: 32,
		PruningRate:       0.0,
	}

	for i := range s.globalModel.Weights {
		s.globalModel.Weights[i] = 0.0
	}

	s.initialized = true
	return nil
}

func NewEnhancedPrivacyEngine() *EnhancedPrivacyEngine {
	return &EnhancedPrivacyEngine{
		mechanisms:        make(map[string]PrivacyMechanismV20),
		privacyAccountant: NewPrivacyAccountant(),
		clippingBounds:    1.0,
		noiseMultiplier:   1.0,
		totalBudget:       10.0,
		spentBudget:       0.0,
	}
}

func (p *EnhancedPrivacyEngine) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.mechanisms["gaussian"] = &GaussianMechanismV20{epsilon: 1.0, delta: 1e-5, sensitivity: 1.0}
	p.mechanisms["laplace"] = &LaplaceMechanismV20{epsilon: 1.0, sensitivity: 1.0}
	p.mechanisms["exponential"] = &ExponentialMechanismV20{epsilon: 1.0, sensitivity: 1.0}

	return nil
}

func (p *EnhancedPrivacyEngine) ApplyDifferentialPrivacy(ctx context.Context, data []float64, epsilon, delta float64) ([]float64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(data) == 0 {
		return data, nil
	}

	clipped := p.clipGradients(data)

	mechanism, exists := p.mechanisms["gaussian"]
	if !exists {
		return clipped, nil
	}

	noisy, err := mechanism.Apply(clipped, epsilon, delta)
	if err != nil {
		return clipped, err
	}

	eps, del := mechanism.GetPrivacySpend()
	p.spentBudget += eps
	p.privacyAccountant.Account(int(len(data)), eps, del)

	return noisy, nil
}

func (p *EnhancedPrivacyEngine) clipGradients(gradients []float64) []float64 {
	clipped := make([]float64, len(gradients))
	norm := 0.0

	for _, g := range gradients {
		norm += g * g
	}
	norm = math.Sqrt(norm)

	if norm > p.clippingBounds {
		scale := p.clippingBounds / norm
		for i, g := range gradients {
			clipped[i] = g * scale
		}
	} else {
		copy(clipped, gradients)
	}

	return clipped
}

func (p *EnhancedPrivacyEngine) GetPrivacyBudget() (float64, float64) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.totalBudget, p.spentBudget
}

func (m *GaussianMechanismV20) Apply(data []float64, epsilon, delta float64) ([]float64, error) {
	m.epsilon = epsilon
	m.delta = delta
	sigma := math.Sqrt(2*math.Log(1.25/delta)) * (m.sensitivity / epsilon)

	result := make([]float64, len(data))
	for i := range data {
		noise := gaussianRandomV20(0, sigma)
		result[i] = data[i] + noise
	}

	m.spend = epsilon
	return result, nil
}

func (m *GaussianMechanismV20) GetPrivacySpend() (float64, float64) {
	return m.epsilon, m.delta
}

func (m *LaplaceMechanismV20) Apply(data []float64, epsilon, delta float64) ([]float64, error) {
	m.epsilon = epsilon
	b := m.sensitivity / epsilon

	result := make([]float64, len(data))
	for i := range data {
		noise := laplaceRandomV20(0, b)
		result[i] = data[i] + noise
	}

	m.spend = epsilon
	return result, nil
}

func (m *LaplaceMechanismV20) GetPrivacySpend() (float64, float64) {
	return m.epsilon, 0
}

func (m *ExponentialMechanismV20) Apply(data []float64, epsilon, delta float64) ([]float64, error) {
	m.epsilon = epsilon
	beta := 2 * m.sensitivity / epsilon

	result := make([]float64, len(data))
	for i := range data {
		noise := exponentialRandom(beta)
		prob := math.Exp(epsilon * data[i] / (2 * m.sensitivity))
		if rand.Float64() < prob {
			result[i] = data[i] + noise
		} else {
			result[i] = data[i] - noise
		}
	}

	m.spend = epsilon
	return result, nil
}

func (m *ExponentialMechanismV20) GetPrivacySpend() (float64, float64) {
	return m.epsilon, 0
}

func gaussianRandomV20(mean, stddev float64) float64 {
	u1 := rand.Float64()
	u2 := rand.Float64()
	z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return mean + stddev*z
}

func laplaceRandomV20(mean, b float64) float64 {
	u := rand.Float64() - 0.5
	return mean - b*math.Copysign(1, u)*math.Log(1-2*math.Abs(u))
}

func exponentialRandom(lambda float64) float64 {
	u := rand.Float64()
	return -math.Log(1-u) / lambda
}

func NewPrivacyAccountant() *PrivacyAccountant {
	return &PrivacyAccountant{
		epsilon:     0,
		delta:       1e-5,
		orders:      []float64{1.5, 2, 2.5, 3, 4, 8, 16, 32, 64, 128, 256},
		RDP:         0,
		composedEPS: 0,
	}
}

func (a *PrivacyAccountant) Account(numRecords int, epsilon, delta float64) {
	alpha := 1.0
	for _, order := range a.orders {
		rdp := alpha * (math.Exp(order*epsilon) - 1) / (order - 1)
		a.RDP += rdp
	}
	a.composedEPS += epsilon
	a.epsilon = a.composedEPS
}

func (a *PrivacyAccountant) GetPrivacySpend() (float64, float64) {
	return a.epsilon, a.delta
}

func NewFederatedAggregationEngine() *FederatedAggregationEngine {
	return &FederatedAggregationEngine{
		strategies:          make(map[string]AggregationStrategy),
		currentStrategy:     "fedavg",
		convergenceThreshold: 0.01,
		adaptiveWeights:     true,
		momentumBuffer:      make([]float64, 256),
	}
}

func (e *FederatedAggregationEngine) Initialize(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.strategies["fedavg"] = &FedAvgStrategy{}
	e.strategies["fedprox"] = &FedProxStrategy{proximalTerm: 0.01}
	e.strategies["scaffold"] = &ScaffoldStrategy{
		controlVariates: make(map[string][]float64),
	}

	return nil
}

func (e *FederatedAggregationEngine) Aggregate(ctx context.Context, updates []ModelUpdate, globalWeights []float64, strategy string) ([]float64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	aggStrategy, exists := e.strategies[strategy]
	if !exists {
		aggStrategy = e.strategies["fedavg"]
	}

	aggregatedWeights := aggStrategy.Aggregate(updates, globalWeights)

	if e.adaptiveWeights {
		aggregatedWeights = e.applyMomentum(aggregatedWeights)
	}

	return aggregatedWeights, nil
}

func (e *FederatedAggregationEngine) applyMomentum(weights []float64) []float64 {
	momentum := 0.9
	result := make([]float64, len(weights))

	for i := range weights {
		e.momentumBuffer[i] = momentum*e.momentumBuffer[i] + (1-momentum)*weights[i]
		result[i] = e.momentumBuffer[i]
	}

	return result
}

func (s *FedAvgStrategy) Aggregate(updates []ModelUpdate, globalWeights []float64) []float64 {
	if len(updates) == 0 {
		return globalWeights
	}

	totalSamples := 0
	for _, update := range updates {
		if update.SampleCount > 0 {
			totalSamples += update.SampleCount
		} else {
			totalSamples += 100
		}
	}

	result := make([]float64, len(globalWeights))
	for _, update := range updates {
		weight := 100.0
		if update.SampleCount > 0 {
			weight = float64(update.SampleCount)
		}
		normalizedWeight := weight / float64(totalSamples)

		for i := range result {
			if i < len(update.Weights) {
				result[i] += update.Weights[i] * normalizedWeight
			}
		}
	}

	return result
}

func (s *FedProxStrategy) Aggregate(updates []ModelUpdate, globalWeights []float64) []float64 {
	result := make([]float64, len(globalWeights))

	totalSamples := 0
	for _, update := range updates {
		if update.SampleCount > 0 {
			totalSamples += update.SampleCount
		} else {
			totalSamples += 100
		}
	}

	for _, update := range updates {
		weight := 100.0
		if update.SampleCount > 0 {
			weight = float64(update.SampleCount)
		}
		normalizedWeight := weight / float64(totalSamples)

		for i := range result {
			if i < len(update.Weights) {
				diff := update.Weights[i] - globalWeights[i]
				proximalCorrection := s.proximalTerm * diff
				result[i] += (update.Weights[i] - proximalCorrection) * normalizedWeight
			}
		}
	}

	return result
}

func (s *ScaffoldStrategy) Aggregate(updates []ModelUpdate, globalWeights []float64) []float64 {
	result := make([]float64, len(globalWeights))

	if len(updates) == 0 {
		return globalWeights
	}

	totalSamples := 0
	for _, update := range updates {
		if update.SampleCount > 0 {
			totalSamples += update.SampleCount
		} else {
			totalSamples += 100
		}
	}

	for _, update := range updates {
		weight := 100.0
		if update.SampleCount > 0 {
			weight = float64(update.SampleCount)
		}
		normalizedWeight := weight / float64(totalSamples)

		controlVariate := s.controlVariates[update.ParticipantID]
		if controlVariate == nil {
			controlVariate = make([]float64, len(globalWeights))
		}

		for i := range result {
			if i < len(update.Weights) {
				grad := update.Weights[i] - globalWeights[i]
				correction := controlVariate[i]
				result[i] += (grad + correction) * normalizedWeight
			}
		}
	}

	return result
}

func NewFLMonitoringPanel() *FLMonitoringPanel {
	return &FLMonitoringPanel{
		metrics:          &FLMetrics{},
		roundsHistory:    make([]RoundMetricsV20, 0),
		alerts:           make([]FLAlert, 0),
		participantStats: make(map[string]*ParticipantStats),
	}
}

func (m *FLMonitoringPanel) RecordRound(ctx context.Context, round int, metrics *RoundMetricsV20) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics.TotalRounds = round
	m.metrics.LastUpdateTime = time.Now()

	m.roundsHistory = append(m.roundsHistory, *metrics)

	if len(m.roundsHistory) > 1000 {
		m.roundsHistory = m.roundsHistory[len(m.roundsHistory)-1000:]
	}

	if metrics.PrivacySpend > 0.5 {
		m.alerts = append(m.alerts, FLAlert{
			AlertID:   fmt.Sprintf("privacy_%d", round),
			Type:      "privacy_budget",
			Severity:  "warning",
			Message:   fmt.Sprintf("High privacy spend in round %d: %.2f", round, metrics.PrivacySpend),
			Timestamp: time.Now(),
			Resolved:  false,
		})
	}
}

func (m *FLMonitoringPanel) GetMetrics(ctx context.Context) *FLMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.metrics
}

func (m *FLMonitoringPanel) GetRoundsHistory(ctx context.Context, limit int) []RoundMetricsV20 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit > len(m.roundsHistory) {
		limit = len(m.roundsHistory)
	}

	return m.roundsHistory[len(m.roundsHistory)-limit:]
}

func (m *FLMonitoringPanel) GetAlerts(ctx context.Context, unresolvedOnly bool) []FLAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []FLAlert
	for _, alert := range m.alerts {
		if !unresolvedOnly || !alert.Resolved {
			result = append(result, alert)
		}
	}

	return result
}

func (m *FLMonitoringPanel) ResolveAlert(ctx context.Context, alertID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.alerts {
		if m.alerts[i].AlertID == alertID {
			m.alerts[i].Resolved = true
			return nil
		}
	}

	return fmt.Errorf("alert %s not found", alertID)
}

func NewSecureCommunicationLayer() *SecureCommunicationLayer {
	return &SecureCommunicationLayer{
		secureChannels:    make(map[string]*SecureChannel),
		authenticatedNodes: make(map[string]bool),
		encryptionKey:      make([]byte, 32),
	}
}

func (s *SecureCommunicationLayer) EstablishChannel(ctx context.Context, nodeID string) (*SecureChannel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.secureChannels[nodeID]; exists {
		return s.secureChannels[nodeID], nil
	}

	channel := &SecureChannel{
		NodeID:      nodeID,
		SessionKey:  generateSessionKey(),
		Established: true,
		LastActive:  time.Now(),
	}

	s.secureChannels[nodeID] = channel
	s.authenticatedNodes[nodeID] = true

	return channel, nil
}

func (s *SecureCommunicationLayer) IsAuthenticated(nodeID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.authenticatedNodes[nodeID]
}

func (s *SecureCommunicationLayer) RevokeAccess(ctx context.Context, nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.secureChannels, nodeID)
	delete(s.authenticatedNodes, nodeID)

	return nil
}

func generateSessionKey() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(rand.Intn(256))
	}
	return key
}

func (s *FederatedLearningV20) RegisterParticipant(ctx context.Context, participant *FLParticipantV20) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.participants[participant.ID]; exists {
		return fmt.Errorf("participant %s already registered", participant.ID)
	}

	participant.LocalModel = &LocalModelV20{
		Weights:      make([]float64, 256),
		Gradients:    make([]float64, 256),
		UpdateCount:  0,
		LastUpdate:   time.Now(),
		PrunedWeights: make([]bool, 256),
		QuantizedWeights: make([]int8, 256),
	}

	for i := range participant.LocalModel.PrunedWeights {
		participant.LocalModel.PrunedWeights[i] = false
	}

	participant.Metrics = &ParticipantMetricsV20{
		Accuracy:    0.0,
		Precision:   0.0,
		Recall:      0.0,
		F1Score:    0.0,
		DataQuality: 0.8,
	}

	participant.Status = "registered"

	channel, _ := s.secureComms.EstablishChannel(ctx, participant.ID)
	participant.CommunicationKey = channel.SessionKey

	s.participants[participant.ID] = participant

	return nil
}

func (s *FederatedLearningV20) PerformFederatedRound(ctx context.Context, strategy string) (*FederatedRoundResultV20, error) {
	if !s.initialized {
		return nil, fmt.Errorf("system not initialized")
	}

	s.mu.Lock()
	selectedParticipants := s.selectParticipants(3)
	s.mu.Unlock()

	updates := make([]ModelUpdate, 0)
	for _, participantID := range selectedParticipants {
		participant := s.participants[participantID]
		if participant != nil && participant.Status == "registered" {
			update := ModelUpdate{
				ParticipantID: participantID,
				Weights:       participant.LocalModel.Weights,
				Gradients:     participant.LocalModel.Gradients,
				SampleCount:   100,
				TrustScore:   participant.TrustScore,
				PrivacyBudget: participant.PrivacyBudget,
				RoundNumber:   len(s.monitor.roundsHistory) + 1,
				Timestamp:     time.Now(),
			}
			updates = append(updates, update)
		}
	}

	aggregatedWeights, err := s.aggregationEngine.Aggregate(ctx, updates, s.globalModel.Weights, strategy)
	if err != nil {
		return nil, err
	}

	noisyWeights, err := s.privacyEngine.ApplyDifferentialPrivacy(ctx, aggregatedWeights, 1.0, 1e-5)
	if err != nil {
		noisyWeights = aggregatedWeights
	}

	s.globalModel.Weights = noisyWeights
	s.globalModel.LastUpdate = time.Now()

	performance := s.calculatePerformance(noisyWeights)

	roundMetrics := &RoundMetricsV20{
		RoundNumber:       len(s.monitor.roundsHistory) + 1,
		Timestamp:         time.Now(),
		ParticipantsCount: len(selectedParticipants),
		Accuracy:          performance.Accuracy,
		Loss:              performance.Loss,
		PrivacySpend:      1.0,
		AvgLatency:        10 * time.Millisecond,
		Converged:         performance.Loss < s.aggregationEngine.convergenceThreshold,
	}

	s.monitor.RecordRound(ctx, roundMetrics.RoundNumber, roundMetrics)

	return &FederatedRoundResultV20{
		Success:         true,
		RoundNumber:     roundMetrics.RoundNumber,
		GlobalModel:     s.globalModel,
		Performance:     performance,
		Participants:    selectedParticipants,
		PrivacyBudgetUsed: 1.0,
	}, nil
}

func (s *FederatedLearningV20) selectParticipants(minCount int) []string {
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
		idx := rand.Intn(len(available))
		selected = append(selected, available[idx])
		available = append(available[:idx], available[idx+1:]...)
	}

	return selected
}

func (s *FederatedLearningV20) calculatePerformance(weights []float64) *ModelPerformanceV20 {
	loss := 0.0
	for _, w := range weights {
		loss += w * w
	}
	loss = loss / float64(len(weights))

	accuracy := 1.0 / (1.0 + loss)
	precision := accuracy * 0.95
	recall := accuracy * 0.93
	f1Score := 2 * (precision * recall) / (precision + recall)

	return &ModelPerformanceV20{
		Accuracy:   accuracy,
		Precision:  precision,
		Recall:     recall,
		F1Score:    f1Score,
		Loss:       loss,
		LatencyMs:  10.0,
		Throughput: 1000.0,
	}
}

func (s *FederatedLearningV20) PruneModel(ctx context.Context, pruningRate float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.globalModel.PruningRate = pruningRate

	for i := range s.globalModel.Weights {
		if math.Abs(s.globalModel.Weights[i]) < pruningRate {
			s.globalModel.Weights[i] = 0
		}
	}

	return nil
}

func (s *FederatedLearningV20) QuantizeModel(ctx context.Context, bits int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.globalModel.QuantizationLevel = bits

	scale := math.Pow(2, float64(bits)) - 1
	maxVal := 0.0

	for _, w := range s.globalModel.Weights {
		if math.Abs(w) > maxVal {
			maxVal = math.Abs(w)
		}
	}

	if maxVal == 0 {
		maxVal = 1.0
	}

	for i, w := range s.globalModel.Weights {
		normalized := (w + maxVal) / (2 * maxVal)
		quantized := int8(math.Round(normalized * scale))
		s.globalModel.Weights[i] = float64(quantized) / scale * 2 * maxVal
	}

	return nil
}

func (s *FederatedLearningV20) GetMonitoringData(ctx context.Context) *FLMonitoringData {
	metrics := s.monitor.GetMetrics(ctx)
	rounds := s.monitor.GetRoundsHistory(ctx, 100)
	alerts := s.monitor.GetAlerts(ctx, false)

	return &FLMonitoringData{
		Metrics:    metrics,
		Rounds:     rounds,
		Alerts:     alerts,
		GeneratedAt: time.Now(),
	}
}

type FederatedRoundResultV20 struct {
	Success          bool              `json:"success"`
	RoundNumber      int               `json:"round_number"`
	GlobalModel      *GlobalModelV20   `json:"global_model"`
	Performance      *ModelPerformanceV20 `json:"performance"`
	Participants     []string          `json:"participants"`
	PrivacyBudgetUsed float64          `json:"privacy_budget_used"`
}

type FLMonitoringData struct {
	Metrics      *FLMetrics       `json:"metrics"`
	Rounds       []RoundMetricsV20 `json:"rounds"`
	Alerts       []FLAlert         `json:"alerts"`
	GeneratedAt  time.Time         `json:"generated_at"`
}

type FederatedTrainingRequest struct {
	TaskType       string                `json:"task_type"`
	Rounds         int                   `json:"rounds"`
	MinParticipants int                  `json:"min_participants"`
	LearningRate   float64              `json:"learning_rate"`
	PrivacyBudget  float64              `json:"privacy_budget"`
}

func ParseFLV20Request(data string) (*FederatedTrainingRequest, error) {
	var req FederatedTrainingRequest
	err := json.Unmarshal([]byte(data), &req)
	return &req, err
}
