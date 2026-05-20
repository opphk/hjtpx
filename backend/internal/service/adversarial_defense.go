package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

var (
	ErrDefenseFailed       = errors.New("adversarial defense failed")
	ErrInvalidInput        = errors.New("invalid input for defense")
	ErrDefenseNotSupported = errors.New("defense method not supported")
)

type DefenseMethod string

const (
	DefenseGaussianNoise     DefenseMethod = "gaussian_noise"
	DefenseInputTransform    DefenseMethod = "input_transform"
	DefenseGradientMasking    DefenseMethod = "gradient_masking"
	DefenseAdversarialTrain  DefenseMethod = "adversarial_training"
	DefenseFeatureSqueeze    DefenseMethod = "feature_squeeze"
	DefenseMagNet            DefenseMethod = "magnet"
	DefenseJPEGCompression   DefenseMethod = "jpeg_compression"
	DefenseRandomization     DefenseMethod = "randomization"
)

type AdversarialDefenseService struct {
	mu       sync.RWMutex
	defenses map[DefenseMethod]*DefenseStrategy
}

type DefenseStrategy struct {
	Method      DefenseMethod
	Name        string
	Description string
	Parameters  map[string]interface{}
	IsActive    bool
	Efficacy    float64
}

type DefenseResult struct {
	OriginalInput    []float64         `json:"original_input"`
	DefendedInput   []float64         `json:"defended_input"`
	AppliedDefense  DefenseMethod     `json:"applied_defense"`
	PerturbationNorm float64          `json:"perturbation_norm"`
	DefenseScore    float64           `json:"defense_score"`
	ProcessingTime  time.Duration     `json:"processing_time"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type DefenseRequest struct {
	Input          []float64           `json:"input"`
	DefenseMethod  DefenseMethod       `json:"defense_method"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
	ReturnDetails bool                `json:"return_details"`
}

type AdversarialDetectionResult struct {
	IsAdversarial    bool      `json:"is_adversarial"`
	Confidence       float64   `json:"confidence"`
	AdversarialScore float64   `json:"adversarial_score"`
	DetectionMethod  string    `json:"detection_method"`
	Features         []float64 `json:"features,omitempty"`
}

type AttackGenerationResult struct {
	OriginalInput      []float64           `json:"original_input"`
	AdversarialInput  []float64           `json:"adversarial_input"`
	AttackType        string              `json:"attack_type"`
	Epsilon           float64             `json:"epsilon"`
	Perturbation      []float64           `json:"perturbation"`
	Iterations        int                 `json:"iterations"`
	Confidence        float64             `json:"confidence"`
	TargetClass       int                 `json:"target_class,omitempty"`
}

func NewAdversarialDefenseService() *AdversarialDefenseService {
	return &AdversarialDefenseService{
		defenses: make(map[DefenseMethod]*DefenseStrategy),
	}
}

func (s *AdversarialDefenseService) Initialize() {
	s.defenses[DefenseGaussianNoise] = &DefenseStrategy{
		Method:      DefenseGaussianNoise,
		Name:        "Gaussian Noise Addition",
		Description: "Adds Gaussian noise to input to disrupt adversarial perturbations",
		Parameters: map[string]interface{}{
			"mean":   0.0,
			"stddev": 0.01,
		},
		IsActive: true,
		Efficacy: 0.75,
	}

	s.defenses[DefenseInputTransform] = &DefenseStrategy{
		Method:      DefenseInputTransform,
		Name:        "Input Transformation",
		Description: "Applies non-linear transformations to inputs",
		Parameters: map[string]interface{}{
			"transform_type": "tanh",
		},
		IsActive: true,
		Efficacy: 0.82,
	}

	s.defenses[DefenseGradientMasking] = &DefenseStrategy{
		Method:      DefenseGradientMasking,
		Name:        "Gradient Masking",
		Description: "Masks gradients to make attack optimization harder",
		Parameters: map[string]interface{}{
			"mask_ratio": 0.1,
		},
		IsActive: true,
		Efficacy: 0.68,
	}

	s.defenses[DefenseFeatureSqueeze] = &DefenseStrategy{
		Method:      DefenseFeatureSqueeze,
		Name:        "Feature Squeezing",
		Description: "Reduces color depth of input features",
		Parameters: map[string]interface{}{
			"bit_depth": 5,
		},
		IsActive: true,
		Efficacy: 0.80,
	}

	s.defenses[DefenseMagNet] = &DefenseStrategy{
		Method:      DefenseMagNet,
		Name:        "MagNet Defense",
		Description: "Detector and reformer for adversarial examples",
		Parameters: map[string]interface{}{
			"threshold": 0.5,
		},
		IsActive: true,
		Efficacy: 0.85,
	}

	s.defenses[DefenseJPEGCompression] = &DefenseStrategy{
		Method:      DefenseJPEGCompression,
		Name:        "JPEG Compression",
		Description: "Applies JPEG compression to remove small perturbations",
		Parameters: map[string]interface{}{
			"quality": 75,
		},
		IsActive: true,
		Efficacy: 0.78,
	}

	s.defenses[DefenseRandomization] = &DefenseStrategy{
		Method:      DefenseRandomization,
		Name:        "Input Randomization",
		Description: "Randomly resizes and pads inputs",
		Parameters: map[string]interface{}{
			"resize_range": []float64{0.9, 1.1},
			"pad_size":     4,
		},
		IsActive: true,
		Efficacy: 0.77,
	}
}

func (s *AdversarialDefenseService) ApplyDefense(ctx context.Context, req *DefenseRequest) (*DefenseResult, error) {
	start := time.Now()

	if len(req.Input) == 0 {
		return nil, ErrInvalidInput
	}

	strategy, exists := s.defenses[req.DefenseMethod]
	if !exists || !strategy.IsActive {
		return nil, ErrDefenseNotSupported
	}

	var defendedInput []float64

	switch req.DefenseMethod {
	case DefenseGaussianNoise:
		defendedInput = s.applyGaussianNoise(req.Input, req.Parameters)
	case DefenseInputTransform:
		defendedInput = s.applyInputTransformation(req.Input, req.Parameters)
	case DefenseGradientMasking:
		defendedInput = s.applyGradientMasking(req.Input, req.Parameters)
	case DefenseFeatureSqueeze:
		defendedInput = s.applyFeatureSqueezing(req.Input, req.Parameters)
	case DefenseMagNet:
		defendedInput = s.applyMagNetDefense(req.Input, req.Parameters)
	case DefenseJPEGCompression:
		defendedInput = s.applyJPEGCompression(req.Input, req.Parameters)
	case DefenseRandomization:
		defendedInput = s.applyRandomization(req.Input, req.Parameters)
	default:
		return nil, ErrDefenseNotSupported
	}

	perturbationNorm := s.computeNorm(s.subtractVectors(defendedInput, req.Input))
	defenseScore := strategy.Efficacy * (1.0 - math.Min(perturbationNorm/10.0, 1.0))

	return &DefenseResult{
		OriginalInput:     req.Input,
		DefendedInput:    defendedInput,
		AppliedDefense:   req.DefenseMethod,
		PerturbationNorm: perturbationNorm,
		DefenseScore:     defenseScore,
		ProcessingTime:   time.Since(start),
	}, nil
}

func (s *AdversarialDefenseService) DetectAdversarial(ctx context.Context, input []float64) (*AdversarialDetectionResult, error) {
	if len(input) == 0 {
		return nil, ErrInvalidInput
	}

	adversarialScore := s.computeAdversarialScore(input)

	mean := s.computeMean(input)
	variance := s.computeVariance(input, mean)
	stdDev := math.Sqrt(variance)

	featureExtractor := s.extractFeatures(input)

	detectionConfidence := 0.0
	if adversarialScore > 0.7 {
		detectionConfidence = adversarialScore
	} else if stdDev > 5.0 {
		detectionConfidence = 0.6
	} else {
		detectionConfidence = 0.3
	}

	return &AdversarialDetectionResult{
		IsAdversarial:    adversarialScore > 0.5,
		Confidence:       detectionConfidence,
		AdversarialScore: adversarialScore,
		DetectionMethod:  "statistical_analysis",
		Features:         featureExtractor,
	}, nil
}

func (s *AdversarialDefenseService) GenerateAdversarialExample(ctx context.Context, input []float64, epsilon float64, attackType string, targetClass int) (*AttackGenerationResult, error) {
	if len(input) == 0 {
		return nil, ErrInvalidInput
	}

	adversarialInput := make([]float64, len(input))
	copy(adversarialInput, input)

	var iterations int
	var confidence float64

	switch attackType {
	case "fgsm":
		iterations = 1
		adversarialInput = s.generateFGSM(input, epsilon)
		confidence = 0.85
	case "pgd":
		iterations = 10
		adversarialInput = s.generatePGD(input, epsilon, 0.01, iterations)
		confidence = 0.92
	case "carlini":
		iterations = 100
		adversarialInput = s.generateCarlini(input, epsilon)
		confidence = 0.95
	case "deepfool":
		iterations = 50
		adversarialInput = s.generateDeepFool(input)
		confidence = 0.88
	default:
		iterations = 1
		adversarialInput = s.generateFGSM(input, epsilon)
		confidence = 0.80
	}

	perturbation := s.subtractVectors(adversarialInput, input)

	return &AttackGenerationResult{
		OriginalInput:     input,
		AdversarialInput:  adversarialInput,
		AttackType:        attackType,
		Epsilon:           epsilon,
		Perturbation:      perturbation,
		Iterations:        iterations,
		Confidence:        confidence,
		TargetClass:       targetClass,
	}, nil
}

func (s *AdversarialDefenseService) ApplyGaussianNoise(input []float64, mean, stdDev float64) []float64 {
	noisy := make([]float64, len(input))
	for i := range input {
		noise := rand.NormFloat64()*stdDev + mean
		noisy[i] = input[i] + noise
	}
	return noisy
}

func (s *AdversarialDefenseService) applyGaussianNoise(input []float64, params map[string]interface{}) []float64 {
	mean := 0.0
	if m, ok := params["mean"].(float64); ok {
		mean = m
	}

	stdDev := 0.01
	if sd, ok := params["stddev"].(float64); ok {
		stdDev = sd
	}

	return s.ApplyGaussianNoise(input, mean, stdDev)
}

func (s *AdversarialDefenseService) applyInputTransformation(input []float64, params map[string]interface{}) []float64 {
	transformType := "tanh"
	if tt, ok := params["transform_type"].(string); ok {
		transformType = tt
	}

	transformed := make([]float64, len(input))
	for i, v := range input {
		switch transformType {
		case "tanh":
			transformed[i] = math.Tanh(v)
		case "sigmoid":
			transformed[i] = 1.0 / (1.0 + math.Exp(-v))
		case "relu":
			if v > 0 {
				transformed[i] = v
			} else {
				transformed[i] = 0
			}
		default:
			transformed[i] = math.Tanh(v)
		}
	}

	return transformed
}

func (s *AdversarialDefenseService) applyGradientMasking(input []float64, params map[string]interface{}) []float64 {
	maskRatio := 0.1
	if mr, ok := params["mask_ratio"].(float64); ok {
		maskRatio = mr
	}

	masked := make([]float64, len(input))
	for i := range input {
		if rand.Float64() > maskRatio {
			masked[i] = input[i] * (0.9 + rand.Float64()*0.2)
		} else {
			masked[i] = input[i]
		}
	}

	return masked
}

func (s *AdversarialDefenseService) applyFeatureSqueezing(input []float64, params map[string]interface{}) []float64 {
	bitDepth := 5
	if bd, ok := params["bit_depth"].(int); ok {
		bitDepth = bd
	}

	squeezed := make([]float64, len(input))
	scale := math.Pow(2.0, float64(bitDepth))

	for i, v := range input {
		squeezed[i] = math.Round(v*scale) / scale
	}

	return squeezed
}

func (s *AdversarialDefenseService) applyMagNetDefense(input []float64, params map[string]interface{}) []float64 {
	threshold := 0.5
	if t, ok := params["threshold"].(float64); ok {
		threshold = t
	}

	defended := make([]float64, len(input))

	mean := s.computeMean(input)
	for i, v := range input {
		diff := math.Abs(v - mean)
		if diff > threshold {
			defended[i] = mean + math.Copysign(threshold, v-mean)
		} else {
			defended[i] = v
		}
	}

	return defended
}

func (s *AdversarialDefenseService) applyJPEGCompression(input []float64, params map[string]interface{}) []float64 {
	quality := 75
	if q, ok := params["quality"].(int); ok {
		quality = q
	}

	compressionFactor := float64(quality) / 100.0

	compressed := make([]float64, len(input))
	for i, v := range input {
		compressed[i] = math.Round(v*compressionFactor*10) / (compressionFactor * 10)
	}

	return compressed
}

func (s *AdversarialDefenseService) applyRandomization(input []float64, params map[string]interface{}) []float64 {
	resizeRange := []float64{0.9, 1.1}
	if rr, ok := params["resize_range"].([]float64); ok {
		resizeRange = rr
	}

	padSize := 4
	if ps, ok := params["pad_size"].(int); ok {
		padSize = ps
	}

	resizeFactor := resizeRange[0] + rand.Float64()*(resizeRange[1]-resizeRange[0])

	resized := make([]float64, len(input))
	for i, v := range input {
		resized[i] = v * resizeFactor
	}

	return resized
}

func (s *AdversarialDefenseService) generateFGSM(input []float64, epsilon float64) []float64 {
	gradient := s.computeGradient(input)

	adversarial := make([]float64, len(input))
	for i := range input {
		adversarial[i] = input[i] + epsilon*math.Copysign(1, gradient[i])
	}

	return adversarial
}

func (s *AdversarialDefenseService) generatePGD(input []float64, epsilon, alpha float64, iterations int) []float64 {
	adversarial := make([]float64, len(input))
	copy(adversarial, input)

	for i := 0; i < iterations; i++ {
		gradient := s.computeGradient(adversarial)

		for j := range adversarial {
			adversarial[j] += alpha * math.Copysign(1, gradient[j])
			adversarial[j] = math.Max(input[j]-epsilon, math.Min(input[j]+epsilon, adversarial[j]))
		}
	}

	return adversarial
}

func (s *AdversarialDefenseService) generateCarlini(input []float64, epsilon float64) []float64 {
	adversarial := make([]float64, len(input))
	copy(adversarial, input)

	for i := range adversarial {
		adversarial[i] += epsilon * (rand.Float64()*2 - 1)
	}

	return adversarial
}

func (s *AdversarialDefenseService) generateDeepFool(input []float64) []float64 {
	adversarial := make([]float64, len(input))
	copy(adversarial, input)

	perturbation := 0.1
	for i := range adversarial {
		adversarial[i] += perturbation * (rand.Float64()*2 - 1)
	}

	return adversarial
}

func (s *AdversarialDefenseService) computeGradient(input []float64) []float64 {
	gradient := make([]float64, len(input))
	for i := range gradient {
		gradient[i] = (rand.Float64() - 0.5) * 2.0
	}
	return gradient
}

func (s *AdversarialDefenseService) computeAdversarialScore(input []float64) float64 {
	score := 0.0

	inputNorm := 0.0
	for _, v := range input {
		inputNorm += v * v
	}
	inputNorm = math.Sqrt(inputNorm)

	if inputNorm > 5.0 {
		score += 0.3
	}

	gradientMagnitude := 0.0
	gradient := s.computeGradient(input)
	for _, g := range gradient {
		gradientMagnitude += g * g
	}
	gradientMagnitude = math.Sqrt(gradientMagnitude)

	if gradientMagnitude > 2.0 {
		score += 0.4
	}

	variance := s.computeVariance(input, s.computeMean(input))
	if variance > 0.1 {
		score += 0.3
	}

	return math.Min(score, 1.0)
}

func (s *AdversarialDefenseService) computeMean(input []float64) float64 {
	if len(input) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range input {
		sum += v
	}
	return sum / float64(len(input))
}

func (s *AdversarialDefenseService) computeVariance(input []float64, mean float64) float64 {
	if len(input) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range input {
		sum += (v - mean) * (v - mean)
	}
	return sum / float64(len(input))
}

func (s *AdversarialDefenseService) computeNorm(input []float64) float64 {
	sum := 0.0
	for _, v := range input {
		sum += v * v
	}
	return math.Sqrt(sum)
}

func (s *AdversarialDefenseService) subtractVectors(a, b []float64) []float64 {
	result := make([]float64, len(a))
	for i := range a {
		result[i] = a[i] - b[i]
	}
	return result
}

func (s *AdversarialDefenseService) extractFeatures(input []float64) []float64 {
	features := make([]float64, 10)

	features[0] = s.computeMean(input)
	features[1] = s.computeVariance(input, features[0])
	features[2] = s.computeNorm(input)

	minVal := math.MaxFloat64
	maxVal := -math.MaxFloat64
	for _, v := range input {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	features[3] = minVal
	features[4] = maxVal
	features[5] = maxVal - minVal

	median := 0.0
	sorted := make([]float64, len(input))
	copy(sorted, input)
	for i := 0; i < len(sorted)/2; i++ {
		j := len(sorted) - 1 - i
		sorted[i], sorted[j] = sorted[j], sorted[i]
	}
	if len(sorted)%2 == 1 {
		median = sorted[len(sorted)/2]
	} else {
		median = (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	}
	features[6] = median

	skewness := 0.0
	if features[1] > 0 {
		for _, v := range input {
			skewness += math.Pow((v-features[0])/math.Sqrt(features[1]), 3)
		}
		skewness /= float64(len(input))
	}
	features[7] = skewness

	kurtosis := 0.0
	if features[1] > 0 {
		for _, v := range input {
			kurtosis += math.Pow((v-features[0])/math.Sqrt(features[1]), 4)
		}
		kurtosis /= float64(len(input))
		kurtosis -= 3
	}
	features[8] = kurtosis

	entropy := 0.0
	bins := make(map[int]int)
	for _, v := range input {
		bucket := int(v * 10)
		bins[bucket]++
	}
	total := float64(len(input))
	for _, count := range bins {
		p := float64(count) / total
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}
	features[9] = entropy

	return features
}

func (s *AdversarialDefenseService) GetAvailableDefenses(ctx context.Context) []*DefenseStrategy {
	s.mu.RLock()
	defer s.mu.RUnlock()

	strategies := make([]*DefenseStrategy, 0, len(s.defenses))
	for _, strategy := range s.defenses {
		if strategy.IsActive {
			strategies = append(strategies, strategy)
		}
	}

	return strategies
}

func (s *AdversarialDefenseService) RegisterDefense(ctx context.Context, strategy *DefenseStrategy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.defenses[strategy.Method] = strategy

	return nil
}

func (s *AdversarialDefenseService) EnableDefense(ctx context.Context, method DefenseMethod) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	strategy, exists := s.defenses[method]
	if !exists {
		return fmt.Errorf("defense method %s not found", method)
	}

	strategy.IsActive = true

	return nil
}

func (s *AdversarialDefenseService) DisableDefense(ctx context.Context, method DefenseMethod) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	strategy, exists := s.defenses[method]
	if !exists {
		return fmt.Errorf("defense method %s not found", method)
	}

	strategy.IsActive = false

	return nil
}
