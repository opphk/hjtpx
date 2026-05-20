package privacy

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"sync"
)

type SecureAggregator struct {
	threshold      int
	totalClients   int
	secretShares   map[string][][]byte
	masks          map[string]map[string][]byte
	receivedMasks  map[int]map[string]bool
	mu             sync.RWMutex
	useSecuredAggregation bool
}

type AggregationConfig struct {
	Threshold            int
	TotalClients         int
	UseSecureAggregation bool
}

func NewSecureAggregator(config AggregationConfig) *SecureAggregator {
	return &SecureAggregator{
		threshold:            config.Threshold,
		totalClients:         config.TotalClients,
		secretShares:         make(map[string][][]byte),
		masks:                make(map[string]map[string][]byte),
		receivedMasks:        make(map[int]map[string]bool),
		useSecuredAggregation: config.UseSecureAggregation,
	}
}

func (sa *SecureAggregator) GenerateSecretShares(clientID string) (shares [][]byte, secret []byte) {
	secret = make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, nil
	}

	shares = make([][]byte, sa.totalClients)
	for i := 0; i < sa.totalClients; i++ {
		shares[i] = sa.generatePolynomialShare(secret, i, sa.threshold)
	}

	sa.mu.Lock()
	sa.secretShares[clientID] = shares
	sa.mu.Unlock()

	return shares, secret
}

func (sa *SecureAggregator) generatePolynomialShare(secret []byte, index, degree int) []byte {
	coefficients := make([][]byte, degree+1)
	coefficients[0] = secret

	for i := 1; i <= degree; i++ {
		coefficients[i] = make([]byte, 32)
		if _, err := rand.Read(coefficients[i]); err != nil {
			continue
		}
	}

	x := make([]byte, 4)
	binary.BigEndian.PutUint32(x, uint32(index+1))

	share := make([]byte, 32)
	for i, coeff := range coefficients {
		xPower := sa.powBytes(x, i)
		for j := range share {
			share[j] ^= coeff[j] & xPower[j%len(xPower)]
		}
	}

	return share
}

func (sa *SecureAggregator) powBytes(x []byte, power int) []byte {
	if power == 0 {
		return []byte{1}
	}

	result := make([]byte, len(x))
	copy(result, x)

	for i := 1; i < power; i++ {
		newResult := make([]byte, len(x))
		for j := range x {
			newResult[j] = result[j] * x[j]
		}
		copy(result, newResult)
	}

	return result
}

func (sa *SecureAggregator) ReceiveShare(clientID string, shareIndex int, share []byte) {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	if _, exists := sa.receivedMasks[shareIndex]; !exists {
		sa.receivedMasks[shareIndex] = make(map[string]bool)
	}
	sa.receivedMasks[shareIndex][clientID] = true
}

func (sa *SecureAggregator) GenerateMasks(clientID string, numClients int) map[string][]byte {
	masks := make(map[string][]byte)

	for i := 0; i < numClients; i++ {
		mask := make([]byte, 32)
		if _, err := rand.Read(mask); err != nil {
			continue
		}
		masks[string(rune('a'+i))] = mask
	}

	sa.mu.Lock()
	sa.masks[clientID] = masks
	sa.mu.Unlock()

	return masks
}

func (sa *SecureAggregator) AggregateWithMasks(updates []*ClientUpdate, masks map[string]map[string][]byte) *ModelParameters {
	if len(updates) == 0 {
		return nil
	}

	aggregatedWeights := make(map[string][]float64)
	aggregatedBiases := make(map[string][]float64)

	for _, update := range updates {
		for name, weights := range update.Weights {
			if _, exists := aggregatedWeights[name]; !exists {
				aggregatedWeights[name] = make([]float64, len(weights))
			}
			for i := range weights {
				aggregatedWeights[name][i] += weights[i]
			}
		}

		for name, biases := range update.Biases {
			if _, exists := aggregatedBiases[name]; !exists {
				aggregatedBiases[name] = make([]float64, len(biases))
			}
			for i := range biases {
				aggregatedBiases[name][i] += biases[i]
			}
		}
	}

	scale := 1.0 / float64(len(updates))
	for name := range aggregatedWeights {
		for i := range aggregatedWeights[name] {
			aggregatedWeights[name][i] *= scale
		}
	}

	for name := range aggregatedBiases {
		for i := range aggregatedBiases[name] {
			aggregatedBiases[name][i] *= scale
		}
	}

	return &ModelParameters{
		Weights: aggregatedWeights,
		Biases:  aggregatedBiases,
		Version: updates[0].Round,
		Round:   updates[0].Round,
	}
}

func (sa *SecureAggregator) ReconstructSecret(shares map[int][]byte) []byte {
	if len(shares) < sa.threshold {
		return nil
	}

	reconstructed := make([]byte, 32)
	indices := make([]int, 0, len(shares))
	for idx := range shares {
		indices = append(indices, idx)
	}

	for _, idx1 := range indices {
		lagrangeCoeff := sa.computeLagrangeCoeff(idx1, indices)
		for j := range reconstructed {
			reconstructed[j] ^= shares[idx1][j] * byte(lagrangeCoeff)
		}
	}

	return reconstructed
}

func (sa *SecureAggregator) computeLagrangeCoeff(targetIdx int, allIndices []int) float64 {
	result := 1.0
	for _, idx := range allIndices {
		if idx != targetIdx {
			result *= float64(idx) / float64(idx-targetIdx)
		}
	}
	return result
}

func (sa *SecureAggregator) AddNoiseToAggregation(value float64, sigma float64) float64 {
	noise := sa.sampleGaussian(sigma)
	return value + noise
}

func (sa *SecureAggregator) sampleGaussian(sigma float64) float64 {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	u1 := math.Abs(float64(bytes[0])/256.0) + 0.001
	rand.Read(bytes)
	u2 := math.Abs(float64(bytes[0])/256.0) + 0.001
	return sigma * math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
}

func (sa *SecureAggregator) GetThreshold() int {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.threshold
}

func (sa *SecureAggregator) GetTotalClients() int {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.totalClients
}

func (sa *SecureAggregator) VerifyThreshold() bool {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	for idx, received := range sa.receivedMasks {
		if len(received) < sa.threshold {
			return false
		}
		_ = idx
	}

	return true
}

type AggregationProtocol struct {
	phase          int
	participants   map[string]*ParticipantState
	aggregatedResult *ModelParameters
	mu             sync.RWMutex
}

type ParticipantState struct {
	ID           string
	Shares       [][]byte
	Masks        map[string][]byte
	HasSubmitted bool
}

func NewAggregationProtocol() *AggregationProtocol {
	return &AggregationProtocol{
		phase:        0,
		participants: make(map[string]*ParticipantState),
	}
}

func (ap *AggregationProtocol) RegisterParticipant(id string) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.participants[id] = &ParticipantState{
		ID:           id,
		Shares:       make([][]byte, 0),
		Masks:        make(map[string][]byte),
		HasSubmitted: false,
	}
}

func (ap *AggregationProtocol) SubmitUpdate(id string, shares [][]byte, masks map[string][]byte) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if participant, exists := ap.participants[id]; exists {
		participant.Shares = shares
		participant.Masks = masks
		participant.HasSubmitted = true
	}
}

func (ap *AggregationProtocol) GetSubmittedCount() int {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	count := 0
	for _, p := range ap.participants {
		if p.HasSubmitted {
			count++
		}
	}
	return count
}

func (ap *AggregationProtocol) SetPhase(phase int) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.phase = phase
}

func (ap *AggregationProtocol) GetPhase() int {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.phase
}

func (ap *AggregationProtocol) SetResult(result *ModelParameters) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.aggregatedResult = result
}

func (ap *AggregationProtocol) GetResult() *ModelParameters {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.aggregatedResult
}

type VectorCommitment struct {
	commitments map[string][]byte
	opening     map[string][]byte
	mu          sync.RWMutex
}

func NewVectorCommitment() *VectorCommitment {
	return &VectorCommitment{
		commitments: make(map[string][]byte),
		opening:     make(map[string][]byte),
	}
}

func (vc *VectorCommitment) Commit(id string, vector []float64) []byte {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	data := make([]byte, 0)
	for _, v := range vector {
		bytes := make([]byte, 8)
		binary.BigEndian.PutUint64(bytes, math.Float64bits(v))
		data = append(data, bytes...)
	}

	hash := sha256.Sum256(data)
	vc.commitments[id] = hash[:]

	return hash[:]
}

func (vc *VectorCommitment) Open(id string, vector []float64) bool {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	if _, exists := vc.commitments[id]; !exists {
		return false
	}

	data := make([]byte, 0)
	for _, v := range vector {
		bytes := make([]byte, 8)
		binary.BigEndian.PutUint64(bytes, math.Float64bits(v))
		data = append(data, bytes...)
	}

	hash := sha256.Sum256(data)

	for i := range hash {
		if hash[i] != vc.commitments[id][i] {
			return false
		}
	}

	return true
}

func (vc *VectorCommitment) Verify(id string, vector []float64) bool {
	return vc.Open(id, vector)
}
