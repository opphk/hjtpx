package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

var (
	ErrHEInvalidParameters  = errors.New("invalid homomorphic encryption parameters")
	ErrHEKeyGenerationFailed = errors.New("HE key generation failed")
	ErrHEEncryptionFailed   = errors.New("HE encryption failed")
	ErrHEDecryptionFailed   = errors.New("HE decryption failed")
	ErrHEOperationFailed    = errors.New("HE operation failed")
	ErrHEInvalidCiphertext   = errors.New("invalid ciphertext")
	ErrHEInvalidPlaintext   = errors.New("invalid plaintext")
	ErrHEMaxDepthExceeded   = errors.New("maximum multiplication depth exceeded")
)

type HEAlgorithm string

const (
	HEAlgorithmPaillier HEAlgorithm = "paillier"
	HEAlgorithmBGV      HEAlgorithm = "bgv"
	HEAlgorithmCKKS     HEAlgorithm = "ckks"
	HEAlgorithmBFV      HEAlgorithm = "bfv"
)

type HEParameterSet struct {
	ParametersID       string
	Algorithm          HEAlgorithm
	SecurityLevel      int
	PlaintextModulus   *big.Int
	CiphertextModulus  *big.Int
	PolynomialDegree   int
	CoefModulusBits    []int
	PlaintextModulusBits int
	RequiredBECLevel   int
}

type HEPublicKey struct {
	N        *big.Int
	G        *big.Int
	NSquared *big.Int
}

type HEPrivateKey struct {
	Lambda    *big.Int
	P         *big.Int
	Q         *big.Int
	Mu        *big.Int
}

type HEEvaluationKey struct {
	PublicKey  *HEPublicKey
	KeySwitchingKey []byte
	RelinKey   []byte
	RotKey    []byte
}

type HECiphertext struct {
	C1        *big.Int
	C2        *big.Int
	Algorithm HEAlgorithm
	Depth     int
	Scale     float64
}

type HEPlaintext struct {
	M        *big.Int
	PlaintextModulus *big.Int
}

type HEContext struct {
	mu       sync.RWMutex
	params   *HEParameterSet
	publicKey *HEPublicKey
	privateKey *HEPrivateKey
	evalKey   *HEEvaluationKey
}

type HomomorphicEncryptionService struct {
	mu       sync.RWMutex
	contexts map[string]*HEContext
	params   map[HEAlgorithm]*HEParameterSet
}

func NewHomomorphicEncryptionService() *HomomorphicEncryptionService {
	service := &HomomorphicEncryptionService{
		contexts: make(map[string]*HEContext),
		params:   make(map[HEAlgorithm]*HEParameterSet),
	}

	service.initializeParameterSets()

	return service
}

func (s *HomomorphicEncryptionService) initializeParameterSets() {
	s.params[HEAlgorithmPaillier] = &HEParameterSet{
		ParametersID:     "paillier-2048",
		Algorithm:       HEAlgorithmPaillier,
		SecurityLevel:   128,
		PolynomialDegree: 0,
	}

	s.params[HEAlgorithmBGV] = &HEParameterSet{
		ParametersID:         "bgv-4096-1",
		Algorithm:            HEAlgorithmBGV,
		SecurityLevel:       128,
		PolynomialDegree:    4096,
		PlaintextModulusBits: 20,
		CiphertextModulus:    new(big.Int).Lsh(big.NewInt(1), 118),
	}

	s.params[HEAlgorithmCKKS] = &HEParameterSet{
		ParametersID:         "ckks-4096-1",
		Algorithm:            HEAlgorithmCKKS,
		SecurityLevel:       128,
		PolynomialDegree:    4096,
		PlaintextModulusBits: 0,
		CiphertextModulus:    new(big.Int).Lsh(big.NewInt(1), 218),
	}

	s.params[HEAlgorithmBFV] = &HEParameterSet{
		ParametersID:         "bfv-4096-1",
		Algorithm:            HEAlgorithmBFV,
		SecurityLevel:       128,
		PolynomialDegree:    4096,
		PlaintextModulusBits: 20,
		CiphertextModulus:    new(big.Int).Lsh(big.NewInt(1), 118),
	}
}

func (s *HomomorphicEncryptionService) CreateContext(ctx context.Context, contextID string, algorithm HEAlgorithm) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.contexts[contextID]; exists {
		return fmt.Errorf("context %s already exists", contextID)
	}

	params, exists := s.params[algorithm]
	if !exists {
		return ErrHEInvalidParameters
	}

	context := &HEContext{
		params:   params,
	}

	s.contexts[contextID] = context

	return nil
}

func (s *HomomorphicEncryptionService) GenerateKeyPair(ctx context.Context, contextID string) (*HEPublicKey, *HEPrivateKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	context, exists := s.contexts[contextID]
	if !exists {
		return nil, nil, fmt.Errorf("context %s not found", contextID)
	}

	switch context.params.Algorithm {
	case HEAlgorithmPaillier:
		return s.generatePaillierKeyPair(context)
	default:
		return s.generatePaillierKeyPair(context)
	}
}

func (s *HomomorphicEncryptionService) generatePaillierKeyPair(context *HEContext) (*HEPublicKey, *HEPrivateKey, error) {
	p, err := rand.Prime(rand.Reader, 1024)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: failed to generate p: %v", ErrHEKeyGenerationFailed, err)
	}

	q, err := rand.Prime(rand.Reader, 1024)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: failed to generate q: %v", ErrHEKeyGenerationFailed, err)
	}

	n := new(big.Int).Mul(p, q)
	g := new(big.Int).Add(n, big.NewInt(1))
	lambda := new(big.Int).Mul(
		new(big.Int).Sub(p, big.NewInt(1)),
		new(big.Int).Sub(q, big.NewInt(1)),
	)

	nSquared := new(big.Int).Mul(n, n)

	mu := new(big.Int).ModInverse(lambda, n)

	context.publicKey = &HEPublicKey{
		N:        n,
		G:        g,
		NSquared: nSquared,
	}

	context.privateKey = &HEPrivateKey{
		Lambda: lambda,
		P:      p,
		Q:      q,
		Mu:     mu,
	}

	return context.publicKey, context.privateKey, nil
}

func (s *HomomorphicEncryptionService) Encrypt(ctx context.Context, contextID string, plaintext *big.Int) (*HECiphertext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	context, exists := s.contexts[contextID]
	if !exists {
		return nil, fmt.Errorf("context %s not found", contextID)
	}

	if context.publicKey == nil {
		return nil, ErrHEKeyGenerationFailed
	}

	if plaintext.Cmp(big.NewInt(0)) < 0 || plaintext.Cmp(context.publicKey.N) >= 0 {
		return nil, ErrHEInvalidPlaintext
	}

	n := context.publicKey.N
	g := context.publicKey.NSquared

	r := new(big.Int)
	for r.BitLen() < n.BitLen()/2 {
		r, _ = rand.Int(rand.Reader, n)
	}

	c1 := new(big.Int).Exp(g, plaintext, nSquared)
	rN := new(big.Int).Exp(r, n, nSquared)
	c1 = new(big.Int).Mod(
		new(big.Int).Mul(c1, rN),
		nSquared,
	)

	c2 := new(big.Int).Exp(g, big.NewInt(1), nSquared)

	return &HECiphertext{
		C1:        c1,
		C2:        c2,
		Algorithm: context.params.Algorithm,
		Depth:     1,
		Scale:     1.0,
	}, nil
}

func (s *HomomorphicEncryptionService) Decrypt(ctx context.Context, contextID string, ciphertext *HECiphertext) (*big.Int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	context, exists := s.contexts[contextID]
	if !exists {
		return nil, fmt.Errorf("context %s not found", contextID)
	}

	if context.privateKey == nil {
		return nil, ErrHEKeyGenerationFailed
	}

	if ciphertext == nil || ciphertext.C1 == nil {
		return nil, ErrHEInvalidCiphertext
	}

	n := new(big.Int).Mul(context.privateKey.P, context.privateKey.Q)
	mu := context.privateKey.Mu

	power := new(big.Int).Sub(ciphertext.C1, big.NewInt(1))
	power.Div(power, n)

	m := new(big.Int).Mod(power, n)
	m.Mul(m, mu)
	m.Mod(m, n)

	return m, nil
}

func (s *HomomorphicEncryptionService) Add(ctx context.Context, contextID string, ct1, ct2 *HECiphertext) (*HECiphertext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	context, exists := s.contexts[contextID]
	if !exists {
		return nil, fmt.Errorf("context %s not found", contextID)
	}

	if context.publicKey == nil {
		return nil, ErrHEKeyGenerationFailed
	}

	result := new(big.Int).Mod(
		new(big.Int).Mul(ct1.C1, ct2.C1),
		context.publicKey.NSquared,
	)

	maxDepth := ct1.Depth
	if ct2.Depth > maxDepth {
		maxDepth = ct2.Depth
	}

	return &HECiphertext{
		C1:        result,
		C2:        ct1.C2,
		Algorithm: context.params.Algorithm,
		Depth:     maxDepth + 1,
		Scale:     ct1.Scale,
	}, nil
}

func (s *HomomorphicEncryptionService) AddPlaintext(ctx context.Context, contextID string, ciphertext *HECiphertext, plaintext *big.Int) (*HECiphertext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	context, exists := s.contexts[contextID]
	if !exists {
		return nil, fmt.Errorf("context %s not found", contextID)
	}

	if context.publicKey == nil {
		return nil, ErrHEKeyGenerationFailed
	}

	m := plaintext.Mod(plaintext, context.publicKey.N)

	plaintextCipher := new(big.Int).Exp(context.publicKey.G, m, context.publicKey.NSquared)

	result := new(big.Int).Mod(
		new(big.Int).Mul(ciphertext.C1, plaintextCipher),
		context.publicKey.NSquared,
	)

	return &HECiphertext{
		C1:        result,
		C2:        ciphertext.C2,
		Algorithm: context.params.Algorithm,
		Depth:     ciphertext.Depth + 1,
		Scale:     ciphertext.Scale,
	}, nil
}

func (s *HomomorphicEncryptionService) Multiply(ctx context.Context, contextID string, ct1, ct2 *HECiphertext) (*HECiphertext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	context, exists := s.contexts[contextID]
	if !exists {
		return nil, fmt.Errorf("context %s not found", contextID)
	}

	if context.publicKey == nil {
		return nil, ErrHEKeyGenerationFailed
	}

	result := new(big.Int).Mod(
		new(big.Int).Exp(ct1.C1, ct2.C1, context.publicKey.NSquared),
		context.publicKey.NSquared,
	)

	newDepth := ct1.Depth + ct2.Depth

	if newDepth > 10 {
		return nil, ErrHEMaxDepthExceeded
	}

	return &HECiphertext{
		C1:        result,
		C2:        ct1.C2,
		Algorithm: context.params.Algorithm,
		Depth:     newDepth,
		Scale:     ct1.Scale * ct2.Scale,
	}, nil
}

func (s *HomomorphicEncryptionService) ScalarMultiply(ctx context.Context, contextID string, ciphertext *HECiphertext, scalar *big.Int) (*HECiphertext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	context, exists := s.contexts[contextID]
	if !exists {
		return nil, fmt.Errorf("context %s not found", contextID)
	}

	if context.publicKey == nil {
		return nil, ErrHEKeyGenerationFailed
	}

	result := new(big.Int).Mod(
		new(big.Int).Exp(ciphertext.C1, scalar, context.publicKey.NSquared),
		context.publicKey.NSquared,
	)

	return &HECiphertext{
		C1:        result,
		C2:        ciphertext.C2,
		Algorithm: context.params.Algorithm,
		Depth:     ciphertext.Depth + 1,
		Scale:     ciphertext.Scale * float64(scalar.Int64()),
	}, nil
}

func (s *HomomorphicEncryptionService) Negate(ctx context.Context, contextID string, ciphertext *HECiphertext) (*HECiphertext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	context, exists := s.contexts[contextID]
	if !exists {
		return nil, fmt.Errorf("context %s not found", contextID)
	}

	if context.publicKey == nil {
		return nil, ErrHEKeyGenerationFailed
	}

	nMinusC1 := new(big.Int).Sub(context.publicKey.NSquared, ciphertext.C1)

	result := new(big.Int).Mod(nMinusC1, context.publicKey.NSquared)

	return &HECiphertext{
		C1:        result,
		C2:        ciphertext.C2,
		Algorithm: context.params.Algorithm,
		Depth:     ciphertext.Depth + 1,
		Scale:     ciphertext.Scale,
	}, nil
}

func (s *HomomorphicEncryptionService) Subtract(ctx context.Context, contextID string, ct1, ct2 *HECiphertext) (*HECiphertext, error) {
	negCt2, err := s.Negate(ctx, contextID, ct2)
	if err != nil {
		return nil, err
	}

	return s.Add(ctx, contextID, ct1, negCt2)
}

func (s *HomomorphicEncryptionService) Relinearize(ctx context.Context, contextID string, ciphertext *HECiphertext) (*HECiphertext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ciphertext.Depth <= 2 {
		return ciphertext, nil
	}

	return &HECiphertext{
		C1:        ciphertext.C1,
		C2:        ciphertext.C2,
		Algorithm: ciphertext.Algorithm,
		Depth:     2,
		Scale:     ciphertext.Scale,
	}, nil
}

func (s *HomomorphicEncryptionService) GetPublicKey(ctx context.Context, contextID string) (*HEPublicKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	context, exists := s.contexts[contextID]
	if !exists {
		return nil, fmt.Errorf("context %s not found", contextID)
	}

	if context.publicKey == nil {
		return nil, ErrHEKeyGenerationFailed
	}

	return context.publicKey, nil
}

func (s *HomomorphicEncryptionService) GetContextInfo(ctx context.Context, contextID string) (*HEParameterSet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	context, exists := s.contexts[contextID]
	if !exists {
		return nil, fmt.Errorf("context %s not found", contextID)
	}

	return context.params, nil
}

func (s *HomomorphicEncryptionService) DeleteContext(ctx context.Context, contextID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.contexts[contextID]; !exists {
		return fmt.Errorf("context %s not found", contextID)
	}

	delete(s.contexts, contextID)

	return nil
}

func (s *HomomorphicEncryptionService) GetAvailableAlgorithms() []HEAlgorithm {
	algorithms := make([]HEAlgorithm, 0, len(s.params))
	for algo := range s.params {
		algorithms = append(algorithms, algo)
	}
	return algorithms
}

type HEBatchOperation struct {
	Operation   string
	Ciphertexts []*HECiphertext
	Plaintexts  []*big.Int
	Scalar      *big.Int
}

type HEBatchResult struct {
	Results    []*HECiphertext
	Success    bool
	Errors     []string
	Duration   time.Duration
}

func (s *HomomorphicEncryptionService) BatchAdd(ctx context.Context, contextID string, ciphertexts []*HECiphertext) (*HEBatchResult, error) {
	start := time.Now()
	results := make([]*HECiphertext, 0)
	errors := make([]string, 0)

	if len(ciphertexts) == 0 {
		return &HEBatchResult{
			Success:  false,
			Errors:   []string{"no ciphertexts provided"},
			Duration: time.Since(start),
		}, nil
	}

	result := ciphertexts[0]
	for i := 1; i < len(ciphertexts); i++ {
		sum, err := s.Add(ctx, contextID, result, ciphertexts[i])
		if err != nil {
			errors = append(errors, fmt.Sprintf("addition failed at index %d: %v", i, err))
			continue
		}
		result = sum
	}

	results = append(results, result)

	return &HEBatchResult{
		Results:  results,
		Success:  len(errors) == 0,
		Errors:   errors,
		Duration: time.Since(start),
	}, nil
}

func (s *HomomorphicEncryptionService) BatchMultiply(ctx context.Context, contextID string, ciphertexts []*HECiphertext) (*HEBatchResult, error) {
	start := time.Now()
	results := make([]*HECiphertext, 0)
	errors := make([]string, 0)

	if len(ciphertexts) == 0 {
		return &HEBatchResult{
			Success:  false,
			Errors:   []string{"no ciphertexts provided"},
			Duration: time.Since(start),
		}, nil
	}

	result := ciphertexts[0]
	for i := 1; i < len(ciphertexts); i++ {
		product, err := s.Multiply(ctx, contextID, result, ciphertexts[i])
		if err != nil {
			errors = append(errors, fmt.Sprintf("multiplication failed at index %d: %v", i, err))
			continue
		}
		result = product
	}

	results = append(results, result)

	return &HEBatchResult{
		Results:  results,
		Success:  len(errors) == 0,
		Errors:   errors,
		Duration: time.Since(start),
	}, nil
}

func (s *HomomorphicEncryptionService) ComputeSum(ctx context.Context, contextID string, ciphertexts []*HECiphertext) (*HECiphertext, error) {
	result, err := s.BatchAdd(ctx, contextID, ciphertexts)
	if err != nil {
		return nil, err
	}

	if len(result.Results) == 0 {
		return nil, errors.New("no results from batch addition")
	}

	return result.Results[0], nil
}

func (s *HomomorphicEncryptionService) ComputeMean(ctx context.Context, contextID string, ciphertexts []*HECiphertext) (*HECiphertext, error) {
	sum, err := s.ComputeSum(ctx, contextID, ciphertexts)
	if err != nil {
		return nil, err
	}

	count := big.NewInt(int64(len(ciphertexts)))
	invCount := new(big.Int).ModInverse(count, contexts[contextID].publicKey.N)

	if invCount == nil {
		return nil, errors.New("failed to compute inverse")
	}

	return s.ScalarMultiply(ctx, contextID, sum, invCount)
}

func (s *HomomorphicEncryptionService) ComputeDotProduct(ctx context.Context, contextID string, ct1, ct2 []*HECiphertext) (*HECiphertext, error) {
	if len(ct1) != len(ct2) {
		return nil, errors.New("vectors must have the same length")
	}

	products := make([]*HECiphertext, len(ct1))
	for i := 0; i < len(ct1); i++ {
		product, err := s.Multiply(ctx, contextID, ct1[i], ct2[i])
		if err != nil {
			return nil, fmt.Errorf("multiplication failed at index %d: %w", i, err)
		}
		products[i] = product
	}

	return s.ComputeSum(ctx, contextID, products)
}
