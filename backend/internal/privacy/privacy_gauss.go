package privacy

import (
	"math"
	"math/rand"
	"sync"
)

type GaussianMechanism struct {
	sigma            float64
	sensitivity      float64
	epsilon          float64
	delta            float64
	rho              float64
	useConcentratedDifferential bool
	mu               float64
	truncated        bool
	lowerBound       float64
	upperBound       float64
}

type GaussianConfig struct {
	Epsilon          float64
	Delta            float64
	Sensitivity      float64
	UseConcentratedDifferential bool
	Truncated        bool
	LowerBound       float64
	UpperBound       float64
}

func NewGaussianMechanism(config GaussianConfig) *GaussianMechanism {
	gm := &GaussianMechanism{
		sensitivity:      config.Sensitivity,
		epsilon:          config.Epsilon,
		delta:            config.Delta,
		rho:              0.0,
		useConcentratedDifferential: config.UseConcentratedDifferential,
		truncated:        config.Truncated,
		lowerBound:       config.LowerBound,
		upperBound:       config.UpperBound,
	}

	if gm.delta == 0 {
		gm.delta = 1e-5
	}

	gm.sigma = gm.calculateSigma()
	return gm
}

func (gm *GaussianMechanism) calculateSigma() float64 {
	if gm.useConcentratedDifferential {
		return gm.calculateCDGSigma()
	}
	return gm.calculateClassicalSigma()
}

func (gm *GaussianMechanism) calculateClassicalSigma() float64 {
	c := math.Sqrt(2 * math.Log(1.25/gm.delta))
	return c * gm.sensitivity / gm.epsilon
}

func (gm *GaussianMechanism) calculateCDGSigma() float64 {
	if gm.rho == 0 {
		gm.rho = gm.epsilon / (2 * math.Sqrt(math.Log(1/gm.delta)))
	}
	return gm.sensitivity * math.Sqrt(1/gm.rho)
}

func (gm *GaussianMechanism) AddNoise(value float64) float64 {
	noise := gm.sampleGaussian()
	result := value + noise

	if gm.truncated {
		result = gm.truncate(result)
	}

	return result
}

func (gm *GaussianMechanism) sampleGaussian() float64 {
	return sampleNormal(0, gm.sigma)
}

func (gm *GaussianMechanism) truncate(value float64) float64 {
	if value < gm.lowerBound {
		return gm.lowerBound
	}
	if value > gm.upperBound {
		return gm.upperBound
	}
	return value
}

func sampleNormal(mean, stdDev float64) float64 {
	u1 := rand.Float64()
	u2 := rand.Float64()
	z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return mean + z*stdDev
}

func (gm *GaussianMechanism) AddNoiseToVector(values []float64) []float64 {
	result := make([]float64, len(values))
	for i, v := range values {
		result[i] = gm.AddNoise(v)
	}
	return result
}

func (gm *GaussianMechanism) AddNoiseToMatrix(matrix [][]float64) [][]float64 {
	result := make([][]float64, len(matrix))
	for i, row := range matrix {
		result[i] = gm.AddNoiseToVector(row)
	}
	return result
}

func (gm *GaussianMechanism) PrivacyUsage() (epsilon, delta float64) {
	return gm.epsilon, gm.delta
}

func (gm *GaussianMechanism) GetSigma() float64 {
	return gm.sigma
}

func (gm *GaussianMechanism) Compose(another *GaussianMechanism) *GaussianMechanism {
	composedEpsilon := gm.epsilon + another.epsilon
	composedDelta := gm.delta + another.delta

	return NewGaussianMechanism(GaussianConfig{
		Epsilon:     composedEpsilon,
		Delta:       composedDelta,
		Sensitivity: gm.sensitivity,
	})
}

type AdaptiveGaussianMechanism struct {
	baseConfig     GaussianConfig
	noiseMultipliers []float64
	currentRound   int
	mu             sync.RWMutex
	deltaEstimate  float64
	confidenceLevel float64
}

func NewAdaptiveGaussianMechanism(baseEpsilon, baseDelta, sensitivity float64) *AdaptiveGaussianMechanism {
	return &AdaptiveGaussianMechanism{
		baseConfig: GaussianConfig{
			Epsilon:     baseEpsilon,
			Delta:       baseDelta,
			Sensitivity: sensitivity,
		},
		noiseMultipliers:  []float64{1.0},
		currentRound:     0,
		deltaEstimate:     baseDelta,
		confidenceLevel:  0.99,
	}
}

func (agm *AdaptiveGaussianMechanism) AddNoise(value float64) float64 {
	agm.mu.RLock()
	multiplier := agm.noiseMultipliers[agm.currentRound%len(agm.noiseMultipliers)]
	agm.mu.RUnlock()

	sigma := agm.baseConfig.Sensitivity / agm.baseConfig.Epsilon * multiplier * math.Sqrt(2*math.Log(1.25/agm.baseConfig.Delta))
	noise := sampleNormal(0, sigma)

	return value + noise
}

func (agm *AdaptiveGaussianMechanism) UpdateNoiseMultiplier(multiplier float64) {
	agm.mu.Lock()
	defer agm.mu.Unlock()
	agm.noiseMultipliers = append(agm.noiseMultipliers, multiplier)
}

func (agm *AdaptiveGaussianMechanism) GetCurrentRound() int {
	agm.mu.RLock()
	defer agm.mu.RUnlock()
	return agm.currentRound
}

func (agm *AdaptiveGaussianMechanism) IncrementRound() {
	agm.mu.Lock()
	defer agm.mu.Unlock()
	agm.currentRound++
}

func (gm *GaussianMechanism) PrivacyAmplificationBySampling(sampleProbability float64, mechanism string) float64 {
	epsilonPrime := 2 * math.Log(1.25/gm.delta) * sampleProbability
	return epsilonPrime
}

func (gm *GaussianMechanism) PrivacyAmplificationBySubsampling(numClients, sampleSize int, epsilon, delta float64) (float64, float64) {
	sampleProb := float64(sampleSize) / float64(numClients)
	rho := epsilon / (2 * math.Sqrt(math.Log(1/delta)))
	amplifiedEpsilon := 2 * rho * math.Sqrt(float64(sampleSize)) * sampleProb
	amplifiedDelta := delta * math.Exp(epsilon)
	return amplifiedEpsilon, amplifiedDelta
}

type GaussianState struct {
	sum   float64
	count int
	mu    sync.Mutex
}

func NewGaussianState() *GaussianState {
	return &GaussianState{}
}

func (gs *GaussianState) Add(value float64) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.sum += value
	gs.count++
}

func (gs *GaussianState) Mean() float64 {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if gs.count == 0 {
		return 0
	}
	return gs.sum / float64(gs.count)
}

func (gs *GaussianState) Reset() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.sum = 0
	gs.count = 0
}

type SparseGaussianMechanism struct {
	gaussianMechanism *GaussianMechanism
	sparsityParameter int
	threshold         float64
}

func NewSparseGaussianMechanism(config GaussianConfig, sparsity int) *SparseGaussianMechanism {
	return &SparseGaussianMechanism{
		gaussianMechanism: NewGaussianMechanism(config),
		sparsityParameter: sparsity,
		threshold:         0.0,
	}
}

func (sgm *SparseGaussianMechanism) AddNoise(value float64, index int) float64 {
	if index >= sgm.sparsityParameter {
		return value
	}
	return sgm.gaussianMechanism.AddNoise(value)
}

func (sgm *SparseGaussianMechanism) AddNoiseToSparseVector(values []float64) []float64 {
	result := make([]float64, len(values))
	for i, v := range values {
		result[i] = sgm.AddNoise(v, i)
	}
	return result
}
