package privacy

import (
	"math"
	"math/rand"
	"sync"
)

type LaplaceMechanism struct {
	scale       float64
	sensitivity float64
	epsilon     float64
	bounded     bool
	lowerBound  float64
	upperBound  float64
}

type LaplaceConfig struct {
	Epsilon     float64
	Sensitivity float64
	Bounded     bool
	LowerBound  float64
	UpperBound  float64
}

func NewLaplaceMechanism(config LaplaceConfig) *LaplaceMechanism {
	lm := &LaplaceMechanism{
		sensitivity: config.Sensitivity,
		epsilon:     config.Epsilon,
		bounded:     config.Bounded,
		lowerBound:  config.LowerBound,
		upperBound:  config.UpperBound,
	}

	if lm.epsilon == 0 {
		lm.epsilon = 1.0
	}

	lm.scale = config.Sensitivity / config.Epsilon
	return lm
}

func (lm *LaplaceMechanism) AddNoise(value float64) float64 {
	noise := lm.sampleLaplace()
	result := value + noise

	if lm.bounded {
		result = lm.boundValue(result)
	}

	return result
}

func (lm *LaplaceMechanism) sampleLaplace() float64 {
	u := rand.Float64() - 0.5
	if u < 0 {
		return lm.scale * math.Log(1+2*u)
	}
	return -lm.scale * math.Log(1-2*u)
}

func (lm *LaplaceMechanism) boundValue(value float64) float64 {
	if value < lm.lowerBound {
		return lm.lowerBound
	}
	if value > lm.upperBound {
		return lm.upperBound
	}
	return value
}

func (lm *LaplaceMechanism) AddNoiseToVector(values []float64) []float64 {
	result := make([]float64, len(values))
	for i, v := range values {
		result[i] = lm.AddNoise(v)
	}
	return result
}

func (lm *LaplaceMechanism) PrivacyUsage() float64 {
	return lm.epsilon
}

func (lm *LaplaceMechanism) GetScale() float64 {
	return lm.scale
}

func (lm *LaplaceMechanism) Compose(another *LaplaceMechanism) *LaplaceMechanism {
	composedEpsilon := lm.epsilon + another.epsilon

	return NewLaplaceMechanism(LaplaceConfig{
		Epsilon:     composedEpsilon,
		Sensitivity: lm.sensitivity,
	})
}

type GeometricMechanism struct {
	sensitivity float64
	epsilon     float64
}

func NewGeometricMechanism(epsilon, sensitivity float64) *GeometricMechanism {
	return &GeometricMechanism{
		sensitivity: sensitivity,
		epsilon:     epsilon,
	}
}

func (gm *GeometricMechanism) AddNoise(value int) int {
	p := math.Exp(-gm.epsilon)
	noise := gm.sampleGeometric(p)
	sign := 1
	if rand.Float64() < 0.5 {
		sign = -1
	}
	return value + sign*noise
}

func (gm *GeometricMechanism) sampleGeometric(p float64) int {
	n := 0
	for rand.Float64() > p {
		n++
	}
	return n
}

func (gm *GeometricMechanism) PrivacyUsage() float64 {
	return gm.epsilon
}

type DiscreteLaplaceMechanism struct {
	scale       float64
	sensitivity float64
	epsilon     float64
}

func NewDiscreteLaplaceMechanism(epsilon, sensitivity float64) *DiscreteLaplaceMechanism {
	return &DiscreteLaplaceMechanism{
		sensitivity: sensitivity,
		epsilon:     epsilon,
		scale:       sensitivity / epsilon,
	}
}

func (dlm *DiscreteLaplaceMechanism) AddNoise(value int) int {
	noise := dlm.sampleDiscreteLaplace()
	return value + noise
}

func (dlm *DiscreteLaplaceMechanism) sampleDiscreteLaplace() int {
	u := rand.Float64()
	k := int(math.Floor(dlm.scale * math.Log(1/u)))
	if k > 0 && rand.Float64() < 0.5 {
		k = -k
	}
	return k
}

func (dlm *DiscreteLaplaceMechanism) PrivacyUsage() float64 {
	return dlm.epsilon
}

type LaplaceCalibrator struct {
	delta       float64
	confidence  float64
	samples     int
	targetEpsilon float64
	mu          sync.RWMutex
}

func NewLaplaceCalibrator(delta, confidence float64) *LaplaceCalibrator {
	return &LaplaceCalibrator{
		delta:      delta,
		confidence: confidence,
		samples:    1000,
	}
}

func (lc *LaplaceCalibrator) Calibrate(lowerBound, upperBound float64) float64 {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	sensitivity := upperBound - lowerBound
	lower := 0.0
	upper := 10.0

	for i := 0; i < 50; i++ {
		mid := (lower + upper) / 2
		sigma := mid * sensitivity / math.Sqrt(2*math.Log(2/lc.delta))
		probability := lc.computeSuccessProbability(sigma, sensitivity)

		if probability > lc.confidence {
			upper = mid
		} else {
			lower = mid
		}
	}

	return lc.targetEpsilon
}

func (lc *LaplaceCalibrator) computeSuccessProbability(sigma, sensitivity float64) float64 {
	_ = sensitivity / sigma
	count := 0
	for i := 0; i < lc.samples; i++ {
		noise := sampleNormal(0, sigma)
		if math.Abs(noise) < sensitivity {
			count++
		}
	}
	return float64(count) / float64(lc.samples)
}

func (lc *LaplaceCalibrator) SetTargetEpsilon(epsilon float64) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.targetEpsilon = epsilon
}

func (lc *LaplaceCalibrator) GetTargetEpsilon() float64 {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	return lc.targetEpsilon
}

type LaplaceAccountant struct {
	mu         sync.Mutex
	totalEpsilon float64
	spentEpsilon float64
	queries     []float64
}

func NewLaplaceAccountant(totalEpsilon float64) *LaplaceAccountant {
	return &LaplaceAccountant{
		totalEpsilon: totalEpsilon,
		queries:      []float64{},
	}
}

func (la *LaplaceAccountant) Account(epsilon float64) error {
	la.mu.Lock()
	defer la.mu.Unlock()

	if la.spentEpsilon+epsilon > la.totalEpsilon {
		return ErrBudgetExceeded
	}

	la.spentEpsilon += epsilon
	la.queries = append(la.queries, epsilon)
	return nil
}

func (la *LaplaceAccountant) GetRemainingBudget() float64 {
	la.mu.Lock()
	defer la.mu.Unlock()
	return la.totalEpsilon - la.spentEpsilon
}

func (la *LaplaceAccountant) GetSpentBudget() float64 {
	la.mu.Lock()
	defer la.mu.Unlock()
	return la.spentEpsilon
}

func (la *LaplaceAccountant) ComputeComposedEpsilon() float64 {
	la.mu.Lock()
	defer la.mu.Unlock()

	total := 0.0
	for _, e := range la.queries {
		total += e
	}
	return total
}

type LaplaceFactory struct {
	defaultEpsilon float64
	defaultSensitivity float64
	bounded       bool
	lowerBound    float64
	upperBound    float64
}

func NewLaplaceFactory(epsilon, sensitivity float64) *LaplaceFactory {
	return &LaplaceFactory{
		defaultEpsilon: epsilon,
		defaultSensitivity: sensitivity,
		bounded: false,
	}
}

func (lf *LaplaceFactory) Create() *LaplaceMechanism {
	return NewLaplaceMechanism(LaplaceConfig{
		Epsilon:     lf.defaultEpsilon,
		Sensitivity: lf.defaultSensitivity,
		Bounded:     lf.bounded,
		LowerBound:  lf.lowerBound,
		UpperBound:  lf.upperBound,
	})
}

func (lf *LaplaceFactory) CreateWithEpsilon(epsilon float64) *LaplaceMechanism {
	return NewLaplaceMechanism(LaplaceConfig{
		Epsilon:     epsilon,
		Sensitivity: lf.defaultSensitivity,
		Bounded:     lf.bounded,
		LowerBound:  lf.lowerBound,
		UpperBound:  lf.upperBound,
	})
}

func (lf *LaplaceFactory) SetBounds(lower, upper float64) {
	lf.bounded = true
	lf.lowerBound = lower
	lf.upperBound = upper
}
