package privacy

import (
	"math"
	"math/rand"
	"sync"
)

type FederatedServerConfig struct {
	AggregationMethod string
	UseDP            bool
	DPConfig         DPConfig
}

type FederatedServer struct {
	config         FederatedServerConfig
	globalModel    *ModelParameters
	updateHistory  []*ClientUpdate
	aggregator     AggregationStrategy
	privacyBudget  float64
	usedBudget     float64
	mu             sync.RWMutex
	clientWeights  map[string]float64
}

type AggregationStrategy interface {
	Aggregate(updates []*ClientUpdate, weights map[string]float64) (*ModelParameters, error)
}

type FedAvgStrategy struct{}

func (f *FedAvgStrategy) Aggregate(updates []*ClientUpdate, weights map[string]float64) (*ModelParameters, error) {
	if len(updates) == 0 {
		return nil, ErrNoUpdates
	}

	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}

	if totalWeight == 0 {
		totalWeight = float64(len(updates))
		for i := range updates {
			weights[updates[i].ClientID] = 1.0
		}
	}

	aggregatedWeights := make(map[string][]float64)
	aggregatedBiases := make(map[string][]float64)

	for _, update := range updates {
		weight := weights[update.ClientID] / totalWeight

		for name, w := range update.Weights {
			if _, exists := aggregatedWeights[name]; !exists {
				aggregatedWeights[name] = make([]float64, len(w))
			}
			for i := range w {
				aggregatedWeights[name][i] += w[i] * weight
			}
		}

		for name, b := range update.Biases {
			if _, exists := aggregatedBiases[name]; !exists {
				aggregatedBiases[name] = make([]float64, len(b))
			}
			for i := range b {
				aggregatedBiases[name][i] += b[i] * weight
			}
		}
	}

	return &ModelParameters{
		Weights: aggregatedWeights,
		Biases:  aggregatedBiases,
		Version: updates[0].Round,
		Round:   updates[0].Round,
	}, nil
}

type FedProxStrategy struct {
	serverProxyTerm float64
}

func NewFedProxStrategy(proxyTerm float64) *FedProxStrategy {
	return &FedProxStrategy{
		serverProxyTerm: proxyTerm,
	}
}

func (f *FedProxStrategy) Aggregate(updates []*ClientUpdate, weights map[string]float64) (*ModelParameters, error) {
	if len(updates) == 0 {
		return nil, ErrNoUpdates
	}

	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}

	if totalWeight == 0 {
		totalWeight = float64(len(updates))
	}

	aggregatedWeights := make(map[string][]float64)
	aggregatedBiases := make(map[string][]float64)

	for _, update := range updates {
		weight := weights[update.ClientID] / totalWeight

		for name, w := range update.Weights {
			if _, exists := aggregatedWeights[name]; !exists {
				aggregatedWeights[name] = make([]float64, len(w))
			}
			for i := range w {
				aggregatedWeights[name][i] += w[i] * weight
			}
		}

		for name, b := range update.Biases {
			if _, exists := aggregatedBiases[name]; !exists {
				aggregatedBiases[name] = make([]float64, len(b))
			}
			for i := range b {
				aggregatedBiases[name][i] += b[i] * weight
			}
		}
	}

	return &ModelParameters{
		Weights: aggregatedWeights,
		Biases:  aggregatedBiases,
		Version: updates[0].Round,
		Round:   updates[0].Round,
	}, nil
}

func NewFederatedServer(config FederatedServerConfig) *FederatedServer {
	fs := &FederatedServer{
		config:        config,
		globalModel:   NewEmptyModel(),
		updateHistory: make([]*ClientUpdate, 0),
		clientWeights: make(map[string]float64),
		privacyBudget: config.DPConfig.Epsilon,
		usedBudget:    0,
	}

	switch config.AggregationMethod {
	case "fedavg":
		fs.aggregator = &FedAvgStrategy{}
	case "fedprox":
		fs.aggregator = NewFedProxStrategy(0.01)
	default:
		fs.aggregator = &FedAvgStrategy{}
	}

	return fs
}

func NewEmptyModel() *ModelParameters {
	return &ModelParameters{
		Weights: make(map[string][]float64),
		Biases:  make(map[string][]float64),
		Version: 0,
		Round:   0,
	}
}

func (fs *FederatedServer) AggregateUpdates(updates []*ClientUpdate) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if len(updates) == 0 {
		return
	}

	for _, update := range updates {
		fs.updateHistory = append(fs.updateHistory, update)
		fs.clientWeights[update.ClientID] = float64(update.DataSize)
	}

	weights := make(map[string]float64)
	for _, update := range updates {
		weights[update.ClientID] = float64(update.DataSize)
	}

	if fs.config.UseDP {
		updates = fs.applyDifferentialPrivacyToUpdates(updates)
	}

	aggregated, err := fs.aggregator.Aggregate(updates, weights)
	if err != nil {
		return
	}

	fs.globalModel = aggregated
}

func (fs *FederatedServer) applyDifferentialPrivacyToUpdates(updates []*ClientUpdate) []*ClientUpdate {
	if fs.usedBudget >= fs.privacyBudget {
		return updates
	}

	clipNorm := fs.config.DPConfig.ClipNorm
	if clipNorm == 0 {
		clipNorm = 1.0
	}

	noiseScale := clipNorm * fs.config.DPConfig.MaxGradNorm / (fs.privacyBudget - fs.usedBudget)

	processedUpdates := make([]*ClientUpdate, len(updates))
	for i, update := range updates {
		processedUpdate := &ClientUpdate{
			ClientID:     update.ClientID,
			Weights:      make(map[string][]float64),
			Biases:       make(map[string][]float64),
			DataSize:     update.DataSize,
			GradientNorm: update.GradientNorm,
			Round:        update.Round,
			Timestamp:    update.Timestamp,
		}

		for name, weights := range update.Weights {
			processedUpdate.Weights[name] = fs.clipAndNoise(weights, clipNorm, noiseScale)
		}

		for name, biases := range update.Biases {
			processedUpdate.Biases[name] = fs.clipAndNoise(biases, clipNorm, noiseScale)
		}

		processedUpdates[i] = processedUpdate
	}

	fs.usedBudget += fs.config.DPConfig.Epsilon

	return processedUpdates
}

func (fs *FederatedServer) clipAndNoise(gradients []float64, clipNorm, noiseScale float64) []float64 {
	result := make([]float64, len(gradients))
	norm := fs.computeNorm(gradients)
	clipFactor := math.Min(1.0, clipNorm/norm)

	for i, g := range gradients {
		clipped := g * clipFactor
		noise := fs.sampleGaussian(noiseScale)
		result[i] = clipped + noise
	}

	return result
}

func (fs *FederatedServer) computeNorm(v []float64) float64 {
	sum := 0.0
	for _, x := range v {
		sum += x * x
	}
	return math.Sqrt(sum)
}

func (fs *FederatedServer) sampleGaussian(sigma float64) float64 {
	u1 := rand.Float64()
	u2 := rand.Float64()
	return sigma * math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
}

func (fs *FederatedServer) GetGlobalModel() *ModelParameters {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	return &ModelParameters{
		Weights: fs.copyWeights(fs.globalModel.Weights),
		Biases:  fs.copyBiases(fs.globalModel.Biases),
		Version: fs.globalModel.Version,
		Round:   fs.globalModel.Round,
	}
}

func (fs *FederatedServer) copyWeights(weights map[string][]float64) map[string][]float64 {
	result := make(map[string][]float64)
	for k, v := range weights {
		copied := make([]float64, len(v))
		copy(copied, v)
		result[k] = copied
	}
	return result
}

func (fs *FederatedServer) copyBiases(biases map[string][]float64) map[string][]float64 {
	result := make(map[string][]float64)
	for k, v := range biases {
		copied := make([]float64, len(v))
		copy(copied, v)
		result[k] = copied
	}
	return result
}

func (fs *FederatedServer) SetGlobalModel(model *ModelParameters) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.globalModel = model
}

func (fs *FederatedServer) GetUpdateHistory() []*ClientUpdate {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return append([]*ClientUpdate{}, fs.updateHistory...)
}

func (fs *FederatedServer) GetClientWeights() map[string]float64 {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	result := make(map[string]float64)
	for k, v := range fs.clientWeights {
		result[k] = v
	}
	return result
}

func (fs *FederatedServer) GetPrivacyBudgetRemaining() float64 {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.privacyBudget - fs.usedBudget
}

func (fs *FederatedServer) GetPrivacyBudgetUsed() float64 {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.usedBudget
}

func (fs *FederatedServer) ResetPrivacyBudget() {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.usedBudget = 0
}

func (fs *FederatedServer) GetAggregator() AggregationStrategy {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.aggregator
}

func (fs *FederatedServer) SetAggregator(strategy AggregationStrategy) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.aggregator = strategy
}

func (fs *FederatedServer) GetModelVersion() int {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.globalModel.Version
}

func (fs *FederatedServer) GetModelRound() int {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.globalModel.Round
}

var ErrNoUpdates = &ServerError{message: "no updates to aggregate"}

type ServerError struct {
	message string
}

func (e *ServerError) Error() string {
	return e.message
}

type ServerMetrics struct {
	TotalUpdates     int
	AverageGradNorm  float64
	ModelVersion     int
	Round            int
	PrivacyUsed      float64
	ActiveClients    int
}

func (fs *FederatedServer) GetMetrics() ServerMetrics {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	avgGradNorm := 0.0
	if len(fs.updateHistory) > 0 {
		totalNorm := 0.0
		for _, update := range fs.updateHistory {
			totalNorm += update.GradientNorm
		}
		avgGradNorm = totalNorm / float64(len(fs.updateHistory))
	}

	return ServerMetrics{
		TotalUpdates:    len(fs.updateHistory),
		AverageGradNorm: avgGradNorm,
		ModelVersion:    fs.globalModel.Version,
		Round:           fs.globalModel.Round,
		PrivacyUsed:     fs.usedBudget,
		ActiveClients:   len(fs.clientWeights),
	}
}
