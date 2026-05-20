package privacy

import (
	"math"
	"math/rand"
	"sync"
)

type DifferentialPrivacy struct {
	epsilon       float64
	delta         float64
	sensitivity   float64
	noiseType     NoiseType
	mechanism     Mechanism
	budgetManager *PrivacyBudgetManager
}

type NoiseType int

const (
	GaussianNoise NoiseType = iota
	LaplaceNoise
)

type Mechanism int

const (
	CountingMechanism Mechanism = iota
	SumMechanism
	MeanMechanism
	HistogramMechanism
)

type PrivacyBudgetManager struct {
	mu          sync.RWMutex
	spentBudget float64
	totalBudget float64
	tau         float64
}

func NewPrivacyBudgetManager(totalBudget float64) *PrivacyBudgetManager {
	return &PrivacyBudgetManager{
		totalBudget: totalBudget,
		tau:         1.0,
	}
}

func (pbm *PrivacyBudgetManager) Spend(budget float64) bool {
	pbm.mu.Lock()
	defer pbm.mu.Unlock()
	if pbm.spentBudget+budget <= pbm.totalBudget {
		pbm.spentBudget += budget
		return true
	}
	return false
}

func (pbm *PrivacyBudgetManager) GetRemainingBudget() float64 {
	pbm.mu.RLock()
	defer pbm.mu.RUnlock()
	return pbm.totalBudget - pbm.spentBudget
}

func (pbm *PrivacyBudgetManager) Reset() {
	pbm.mu.Lock()
	defer pbm.mu.Unlock()
	pbm.spentBudget = 0
}

func NewDifferentialPrivacy(epsilon, delta, sensitivity float64, noiseType NoiseType, mechanism Mechanism) *DifferentialPrivacy {
	return &DifferentialPrivacy{
		epsilon:     epsilon,
		delta:       delta,
		sensitivity: sensitivity,
		noiseType:   noiseType,
		mechanism:   mechanism,
		budgetManager: &PrivacyBudgetManager{
			totalBudget: epsilon,
		},
	}
}

func (dp *DifferentialPrivacy) AddNoise(value float64) float64 {
	if !dp.budgetManager.Spend(dp.epsilon) {
		return value
	}

	switch dp.noiseType {
	case GaussianNoise:
		sigma := dp.calculateGaussianSigma()
		noise := generateGaussianNoise(sigma)
		return value + noise
	case LaplaceNoise:
		b := dp.sensitivity / dp.epsilon
		noise := generateLaplaceNoise(b)
		return value + noise
	default:
		return value
	}
}

func (dp *DifferentialPrivacy) calculateGaussianSigma() float64 {
	if dp.delta == 0 {
		dp.delta = 1e-5
	}
	c := math.Sqrt(2 * math.Log(1.25/dp.delta))
	return c * dp.sensitivity / dp.epsilon
}

func (dp *DifferentialPrivacy) SetEpsilon(epsilon float64) {
	dp.epsilon = epsilon
}

func (dp *DifferentialPrivacy) SetDelta(delta float64) {
	dp.delta = delta
}

func (dp *DifferentialPrivacy) GetPrivacyParameters() (epsilon, delta, sensitivity float64) {
	return dp.epsilon, dp.delta, dp.sensitivity
}

func (dp *DifferentialPrivacy) PrivacyLoss() float64 {
	return dp.epsilon
}

func (dp *DifferentialPrivacy) Compose(another *DifferentialPrivacy) *DifferentialPrivacy {
	newEpsilon := dp.epsilon + another.epsilon
	newDelta := dp.delta + another.delta
	return NewDifferentialPrivacy(newEpsilon, newDelta, dp.sensitivity, dp.noiseType, dp.mechanism)
}

func (dp *DifferentialPrivacy) PostProcess(result float64, boundMin, boundMax float64) float64 {
	return math.Max(boundMin, math.Min(boundMax, result))
}

func generateGaussianNoise(sigma float64) float64 {
	u1 := rand.Float64()
	u2 := rand.Float64()
	z0 := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
	return z0 * sigma
}

func generateLaplaceNoise(b float64) float64 {
	u := rand.Float64() - 0.5
	if u < 0 {
		return b * math.Log(1.0-2.0*u)
	}
	return -b * math.Log(-2.0*u+1.0)
}

type PrivacyAccountant struct {
	mu           sync.Mutex
	spentEpsilon float64
	spentDelta   float64
	totalEpsilon float64
	totalDelta   float64
}

func NewPrivacyAccountant(totalEpsilon, totalDelta float64) *PrivacyAccountant {
	return &PrivacyAccountant{
		totalEpsilon: totalEpsilon,
		totalDelta:   totalDelta,
	}
}

func (pa *PrivacyAccountant) Account(epsilon, delta float64) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	if pa.spentEpsilon+epsilon > pa.totalEpsilon {
		return ErrBudgetExceeded
	}
	if pa.spentDelta+delta > pa.totalDelta {
		return ErrBudgetExceeded
	}
	pa.spentEpsilon += epsilon
	pa.spentDelta += delta
	return nil
}

func (pa *PrivacyAccountant) GetSpentBudget() (epsilon, delta float64) {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	return pa.spentEpsilon, pa.spentDelta
}

func (pa *PrivacyAccountant) GetRemainingBudget() (epsilon, delta float64) {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	return pa.totalEpsilon - pa.spentEpsilon, pa.totalDelta - pa.spentDelta
}

var ErrBudgetExceeded = &PrivacyBudgetError{message: "privacy budget exceeded"}

type PrivacyBudgetError struct {
	message string
}

func (e *PrivacyBudgetError) Error() string {
	return e.message
}

type PrivateQuery struct {
	QueryType    string
	Sensitivity  float64
	Bounds       [2]float64
	NoiseType    NoiseType
	Epsilon      float64
	Delta        float64
}

type PrivateQueryExecutor struct {
	queries    map[string]*PrivateQuery
	accountant *PrivacyAccountant
	mu         sync.RWMutex
}

func NewPrivateQueryExecutor(totalEpsilon, totalDelta float64) *PrivateQueryExecutor {
	return &PrivateQueryExecutor{
		queries:    make(map[string]*PrivateQuery),
		accountant: NewPrivacyAccountant(totalEpsilon, totalDelta),
	}
}

func (pqe *PrivateQueryExecutor) RegisterQuery(id string, query *PrivateQuery) {
	pqe.mu.Lock()
	defer pqe.mu.Unlock()
	pqe.queries[id] = query
}

func (pqe *PrivateQueryExecutor) ExecuteQuery(id string, value float64) (float64, error) {
	pqe.mu.RLock()
	query, exists := pqe.queries[id]
	pqe.mu.RUnlock()

	if !exists {
		return 0, ErrQueryNotFound
	}

	if err := pqe.accountant.Account(query.Epsilon, query.Delta); err != nil {
		return 0, err
	}

	dp := NewDifferentialPrivacy(query.Epsilon, query.Delta, query.Sensitivity, query.NoiseType, CountingMechanism)
	noisyValue := dp.AddNoise(value)

	return dp.PostProcess(noisyValue, query.Bounds[0], query.Bounds[1]), nil
}

func (pqe *PrivateQueryExecutor) GetAccountant() *PrivacyAccountant {
	return pqe.accountant
}

var ErrQueryNotFound = &QueryError{message: "query not found"}

type QueryError struct {
	message string
}

func (e *QueryError) Error() string {
	return e.message
}
