package privacy

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

type FederatedClientConfig struct {
	ClientID     string
	DataSize     int
	LocalEpochs  int
	BatchSize    int
	LearningRate float64
	UseDP        bool
	DPConfig     DPConfig
	ModelType    string
}

type FederatedClient struct {
	config       FederatedClientConfig
	localData    []DataPoint
	modelWeights map[string][]float64
	modelBiases  map[string][]float64
	gradients    *GradientBuffer
	privacyBudget float64
	mu           sync.RWMutex
	isTraining   bool
	lastSync    time.Time
}

type DataPoint struct {
	Features []float64
	Label    float64
}

type GradientBuffer struct {
	Weights map[string][]float64
	Biases  map[string][]float64
	mu      sync.Mutex
}

func NewFederatedClient(config FederatedClientConfig) *FederatedClient {
	return &FederatedClient{
		config:    config,
		localData: make([]DataPoint, 0),
		modelWeights: make(map[string][]float64),
		modelBiases:  make(map[string][]float64),
		gradients: &GradientBuffer{
			Weights: make(map[string][]float64),
			Biases:  make(map[string][]float64),
		},
		privacyBudget: config.DPConfig.Epsilon,
		isTraining:   false,
		lastSync:    time.Now(),
	}
}

func (fc *FederatedClient) Train(globalModel *ModelParameters, round int) *ClientUpdate {
	fc.mu.Lock()
	if fc.isTraining {
		fc.mu.Unlock()
		return nil
	}
	fc.isTraining = true
	fc.mu.Unlock()

	defer func() {
		fc.mu.Lock()
		fc.isTraining = false
		fc.mu.Unlock()
	}()

	fc.mu.Lock()
	fc.modelWeights = fc.copyWeights(globalModel.Weights)
	fc.modelBiases = fc.copyBiases(globalModel.Biases)
	fc.mu.Unlock()

	for epoch := 0; epoch < fc.config.LocalEpochs; epoch++ {
		fc.runLocalEpoch()
	}

	if fc.config.UseDP {
		fc.applyDifferentialPrivacy()
	}

	gradNorm := fc.computeGradientNorm()

	fc.mu.Lock()
	weights := fc.copyWeights(fc.modelWeights)
	biases := fc.copyBiases(fc.modelBiases)
	fc.mu.Unlock()

	fc.lastSync = time.Now()

	return &ClientUpdate{
		ClientID:     fc.config.ClientID,
		Weights:      weights,
		Biases:       biases,
		DataSize:     fc.config.DataSize,
		GradientNorm: gradNorm,
		Round:        round,
		Timestamp:    time.Now(),
	}
}

func (fc *FederatedClient) runLocalEpoch() {
	if len(fc.localData) == 0 {
		fc.generateSyntheticData()
	}

	batchSize := fc.config.BatchSize
	if batchSize <= 0 {
		batchSize = 32
	}

	for i := 0; i < len(fc.localData); i += batchSize {
		end := i + batchSize
		if end > len(fc.localData) {
			end = len(fc.localData)
		}

		batch := fc.localData[i:end]
		fc.computeGradients(batch)
		fc.applyGradients()
	}
}

func (fc *FederatedClient) generateSyntheticData() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	inputSize := 10
	if fc.config.ModelType == "linear" {
		inputSize = 5
	}

	fc.localData = make([]DataPoint, fc.config.DataSize)
	for i := 0; i < fc.config.DataSize; i++ {
		features := make([]float64, inputSize)
		for j := 0; j < inputSize; j++ {
			features[j] = (rand.Float64() - 0.5) * 2
		}
		label := 0.0
		for j := 0; j < inputSize; j++ {
			label += features[j] * 0.5
		}
		label += (rand.Float64() - 0.5) * 0.1

		fc.localData[i] = DataPoint{
			Features: features,
			Label:    label,
		}
	}
}

func (fc *FederatedClient) computeGradients(batch []DataPoint) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.gradients.mu.Lock()
	defer fc.gradients.mu.Unlock()

	for name, weights := range fc.modelWeights {
		if _, exists := fc.gradients.Weights[name]; !exists {
			fc.gradients.Weights[name] = make([]float64, len(weights))
		}
	}

	for name, biases := range fc.modelBiases {
		if _, exists := fc.gradients.Biases[name]; !exists {
			fc.gradients.Biases[name] = make([]float64, len(biases))
		}
	}

	for _, dp := range batch {
		prediction := fc.predict(dp.Features)

		for name, weights := range fc.modelWeights {
			for i := range weights {
				fc.gradients.Weights[name][i] += (prediction - dp.Label) * dp.Features[i%len(dp.Features)]
			}
		}

		for name, biases := range fc.modelBiases {
			for i := range biases {
				fc.gradients.Biases[name][i] += prediction - dp.Label
			}
		}
	}

	scale := 1.0 / float64(len(batch))
	for name := range fc.gradients.Weights {
		for i := range fc.gradients.Weights[name] {
			fc.gradients.Weights[name][i] *= scale
		}
	}

	for name := range fc.gradients.Biases {
		for i := range fc.gradients.Biases[name] {
			fc.gradients.Biases[name][i] *= scale
		}
	}
}

func (fc *FederatedClient) predict(features []float64) float64 {
	prediction := 0.0
	weightIndex := 0
	for name, weights := range fc.modelWeights {
		if name == "output" {
			continue
		}
		for _, w := range weights {
			if weightIndex < len(features) {
				prediction += w * features[weightIndex]
				weightIndex++
			}
		}
	}

	return prediction
}

func (fc *FederatedClient) applyGradients() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.gradients.mu.Lock()
	defer fc.gradients.mu.Unlock()

	lr := fc.config.LearningRate

	for name, grads := range fc.gradients.Weights {
		if weights, exists := fc.modelWeights[name]; exists {
			for i := range weights {
				weights[i] -= lr * grads[i]
			}
		}
	}

	for name, grads := range fc.gradients.Biases {
		if biases, exists := fc.modelBiases[name]; exists {
			for i := range biases {
				biases[i] -= lr * grads[i]
			}
		}
	}
}

func (fc *FederatedClient) applyDifferentialPrivacy() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	clipNorm := fc.config.DPConfig.ClipNorm
	if clipNorm == 0 {
		clipNorm = 1.0
	}

	for _, weights := range fc.gradients.Weights {
		gradNorm := fc.computeVectorNorm(weights)
		clipFactor := math.Min(1.0, clipNorm/gradNorm)

		for i := range weights {
			weights[i] *= clipFactor
		}

		sigma := clipNorm * fc.config.DPConfig.MaxGradNorm / fc.privacyBudget
		for i := range weights {
			weights[i] += fc.sampleGaussian(sigma)
		}
	}
}

func (fc *FederatedClient) sampleGaussian(sigma float64) float64 {
	u1 := rand.Float64()
	u2 := rand.Float64()
	return sigma * math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
}

func (fc *FederatedClient) computeGradientNorm() float64 {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	fc.gradients.mu.Lock()
	defer fc.gradients.mu.Unlock()

	totalNorm := 0.0

	for _, grads := range fc.gradients.Weights {
		totalNorm += fc.computeVectorNorm(grads)
	}

	for _, grads := range fc.gradients.Biases {
		totalNorm += fc.computeVectorNorm(grads)
	}

	return math.Sqrt(totalNorm)
}

func (fc *FederatedClient) computeVectorNorm(v []float64) float64 {
	sum := 0.0
	for _, x := range v {
		sum += x * x
	}
	return math.Sqrt(sum)
}

func (fc *FederatedClient) computeVectorNormWithClip(v []float64, clipNorm float64) float64 {
	norm := fc.computeVectorNorm(v)
	if norm > clipNorm {
		return clipNorm
	}
	return norm
}

func (fc *FederatedClient) copyWeights(weights map[string][]float64) map[string][]float64 {
	result := make(map[string][]float64)
	for k, v := range weights {
		copied := make([]float64, len(v))
		copy(copied, v)
		result[k] = copied
	}
	return result
}

func (fc *FederatedClient) copyBiases(biases map[string][]float64) map[string][]float64 {
	result := make(map[string][]float64)
	for k, v := range biases {
		copied := make([]float64, len(v))
		copy(copied, v)
		result[k] = copied
	}
	return result
}

func (fc *FederatedClient) AddLocalData(data []DataPoint) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.localData = append(fc.localData, data...)
}

func (fc *FederatedClient) ClearLocalData() {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.localData = make([]DataPoint, 0)
}

func (fc *FederatedClient) GetLocalDataSize() int {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return len(fc.localData)
}

func (fc *FederatedClient) GetClientID() string {
	return fc.config.ClientID
}

func (fc *FederatedClient) GetLastSyncTime() time.Time {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.lastSync
}

func (fc *FederatedClient) GetPrivacyBudgetRemaining() float64 {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.privacyBudget
}

func (fc *FederatedClient) SpendPrivacyBudget(amount float64) bool {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if fc.privacyBudget >= amount {
		fc.privacyBudget -= amount
		return true
	}
	return false
}

func (fc *FederatedClient) IsTraining() bool {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.isTraining
}

func (fc *FederatedClient) GetLocalModel() (weights, biases map[string][]float64) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.copyWeights(fc.modelWeights), fc.copyBiases(fc.modelBiases)
}
