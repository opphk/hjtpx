package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"time"
)

type QuantumRandomGenerator struct {
	entropyPool     []byte
	poolMutex       sync.Mutex
	lastRefreshTime time.Time
	refreshInterval time.Duration
	noiseSource     *QuantumNoiseSource
}

type QuantumNoiseSource struct {
	seed       uint64
	phase      float64
	amplitude  float64
	frequency  float64
}

type QuantumCaptchaConfig struct {
	Length           int      `json:"length"`
	CharacterSet     string   `json:"character_set"`
	NoiseLevel       float64  `json:"noise_level"`
	AntiAI           bool     `json:"anti_ai"`
	QuantumEnhanced   bool     `json:"quantum_enhanced"`
	Unpredictability float64  `json:"unpredictability"`
}

type QuantumChallenge struct {
	Characters      []rune            `json:"characters"`
	NoisePattern    []float64        `json:"noise_pattern"`
	Entropy         float64          `json:"entropy"`
	QuantumBits     int              `json:"quantum_bits"`
	PredictionResistant bool         `json:"prediction_resistant"`
	Timestamp       time.Time        `json:"timestamp"`
	Signature       string           `json:"signature"`
}

func NewQuantumRandomGenerator() *QuantumRandomGenerator {
	return &QuantumRandomGenerator{
		entropyPool:     make([]byte, 256),
		lastRefreshTime: time.Now(),
		refreshInterval: 100 * time.Millisecond,
		noiseSource: &QuantumNoiseSource{
			seed:      uint64(time.Now().UnixNano()),
			phase:     0.0,
			amplitude: 1.0,
			frequency: 1.0,
		},
	}
}

func (q *QuantumRandomGenerator) Initialize() error {
	return q.refreshEntropyPool()
}

func (q *QuantumRandomGenerator) refreshEntropyPool() error {
	q.poolMutex.Lock()
	defer q.poolMutex.Unlock()

	randomBytes := make([]byte, len(q.entropyPool))
	_, err := rand.Read(randomBytes)
	if err != nil {
		return fmt.Errorf("failed to generate random bytes: %w", err)
	}

	quantumNoise := q.generateQuantumNoise(len(randomBytes))

	for i := 0; i < len(randomBytes); i++ {
		q.entropyPool[i] ^= quantumNoise[i]
	}

	thermalNoise := q.generateThermalNoise(len(randomBytes))
	for i := 0; i < len(randomBytes); i++ {
		combined := int(q.entropyPool[i]) + int(thermalNoise[i])
		if combined > 255 {
			combined = 255
		}
		if combined < 0 {
			combined = 0
		}
		q.entropyPool[i] = byte(combined)
	}

	q.lastRefreshTime = time.Now()

	return nil
}

func (q *QuantumRandomGenerator) generateQuantumNoise(length int) []byte {
	noise := make([]byte, length)

	for i := 0; i < length; i++ {
		timestamp := float64(time.Now().UnixNano())
		phase := math.Mod(q.noiseSource.phase+float64(i)*0.01, 2*math.Pi)
		
		sample := q.noiseSource.amplitude * math.Sin(q.noiseSource.frequency*timestamp+phase)
		
		shotNoise := (math.Pow(randFloat64()-0.5, 2) * 2) * math.Sqrt(math.Max(0, sample))
		
		noise[i] = byte(int((sample+shotNoise)*127.5 + 128.5) % 256)
	}

	return noise
}

func (q *QuantumRandomGenerator) generateThermalNoise(length int) []byte {
	noise := make([]byte, length)
	
	temperature := 300.0
	boltzmann := 1.38e-23
	resistance := 50.0
	bandwidth := 1e6
	
	thermalPower := 4 * boltzmann * temperature * bandwidth / resistance
	thermalAmplitude := math.Sqrt(thermalPower * 50)
	
	for i := 0; i < length; i++ {
		gaussianNoise := boxMullerTransform()
		noise[i] = byte(int((gaussianNoise*thermalAmplitude+0.5)*127.5 + 128.5) % 256)
	}
	
	return noise
}

func randFloat64() float64 {
	b := make([]byte, 8)
	rand.Read(b)
	return math.Float64frombits(binary.BigEndian.Uint64(b))
}

func boxMullerTransform() float64 {
	u1 := randFloat64()
	u2 := randFloat64()
	return math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
}

func (q *QuantumRandomGenerator) GenerateRandomBytes(length int) ([]byte, error) {
	if time.Since(q.lastRefreshTime) > q.refreshInterval {
		if err := q.refreshEntropyPool(); err != nil {
			return nil, err
		}
	}

	q.poolMutex.Lock()
	defer q.poolMutex.Unlock()

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		index := int(time.Now().UnixNano()) % len(q.entropyPool)
		result[i] = q.entropyPool[index]
		
		if i%8 == 0 {
			quantumSample := q.sampleQuantumBit()
			result[i] ^= quantumSample
		}
	}

	return result, nil
}

func (q *QuantumRandomGenerator) sampleQuantumBit() byte {
	timestamp := float64(time.Now().UnixNano())
	phase := timestamp * 1e-9 * q.noiseSource.frequency
	
	quantumValue := (math.Sin(phase) + 1) / 2
	
	if quantumValue > 0.5 {
		return 0xFF
	}
	return 0x00
}

func (q *QuantumRandomGenerator) GenerateQuantumRandomString(config *QuantumCaptchaConfig) (*QuantumChallenge, error) {
	if config == nil {
		config = &QuantumCaptchaConfig{
			Length:           6,
			CharacterSet:     "ABCDEFGHJKLMNPQRSTUVWXYZ23456789",
			NoiseLevel:       0.3,
			AntiAI:           true,
			QuantumEnhanced:  true,
			Unpredictability: 0.95,
		}
	}

	if config.Length < 4 {
		config.Length = 4
	}
	if config.Length > 16 {
		config.Length = 16
	}

	charSet := []rune(config.CharacterSet)
	charCount := len(charSet)

	characters := make([]rune, config.Length)
	noisePattern := make([]float64, config.Length)

	quantumBits := 0

	for i := 0; i < config.Length; i++ {
		randomBytes, err := q.GenerateRandomBytes(8)
		if err != nil {
			return nil, err
		}

		value := binary.BigEndian.Uint64(randomBytes)
		
		quantumBit := q.sampleQuantumBit()
		if quantumBit > 0 {
			quantumBits++
		}

		charIndex := int(value % uint64(charCount))
		
		if config.AntiAI {
			chaosFactor := math.Sin(float64(value) * 0.1)
			charIndex = int(float64(charIndex) * (1 + chaosFactor*0.1))
			if charIndex >= charCount {
				charIndex = charCount - 1
			}
			if charIndex < 0 {
				charIndex = 0
			}
		}

		characters[i] = charSet[charIndex]

		noisePattern[i] = q.generateNoiseSample(config.NoiseLevel)
	}

	signature := q.generateSignature(characters)

	challenge := &QuantumChallenge{
		Characters:          characters,
		NoisePattern:        noisePattern,
		Entropy:             q.calculateEntropy(config),
		QuantumBits:         quantumBits,
		PredictionResistant: config.QuantumEnhanced,
		Timestamp:           time.Now(),
		Signature:            signature,
	}

	return challenge, nil
}

func (q *QuantumRandomGenerator) generateNoiseSample(level float64) float64 {
	noise := (randFloat64() - 0.5) * 2 * level
	
	quantumNoise := math.Sin(randFloat64() * math.Pi * 2) * level * 0.5
	
	return noise + quantumNoise
}

func (q *QuantumRandomGenerator) generateSignature(characters []rune) string {
	signatureBytes := make([]byte, 0, len(characters)*2)
	
	for i, char := range characters {
		highBits := byte((int(char) >> 8) & 0xFF)
		lowBits := byte(int(char) & 0xFF)
		
		signatureBytes = append(signatureBytes, highBits^byte(i))
		signatureBytes = append(signatureBytes, lowBits^byte(len(characters)-i))
	}

	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().UnixNano()))
	signatureBytes = append(signatureBytes, timestamp...)

	randomBytes := make([]byte, 16)
	rand.Read(randomBytes)
	signatureBytes = append(signatureBytes, randomBytes...)

	quantumSignature := q.generateQuantumNoise(len(signatureBytes))
	for i := 0; i < len(signatureBytes); i++ {
		signatureBytes[i] ^= quantumSignature[i]
	}

	return base64.StdEncoding.EncodeToString(signatureBytes)
}

func (q *QuantumRandomGenerator) calculateEntropy(config *QuantumCaptchaConfig) float64 {
	charSetSize := float64(len(config.CharacterSet))
	
	baseEntropy := math.Log2(charSetSize) * float64(config.Length)
	
	quantumEnhancement := 0.0
	if config.QuantumEnhanced {
		quantumEnhancement = float64(config.Length) * 0.1
	}
	
	noiseEntropy := config.NoiseLevel * float64(config.Length) * 0.5
	
	unpredictabilityEntropy := config.Unpredictability * float64(config.Length) * 0.3
	
	totalEntropy := baseEntropy + quantumEnhancement + noiseEntropy + unpredictabilityEntropy
	
	return math.Min(totalEntropy, 256.0)
}

func (q *QuantumRandomGenerator) VerifyQuantumChallenge(challenge *QuantumChallenge, userInput string) (bool, float64, error) {
	if len(userInput) != len(challenge.Characters) {
		return false, 0.0, fmt.Errorf("input length mismatch")
	}

	matchingCount := 0
	for i := 0; i < len(challenge.Characters); i++ {
		if i < len(userInput) && rune(userInput[i]) == challenge.Characters[i] {
			matchingCount++
		}
	}

	matchRatio := float64(matchingCount) / float64(len(challenge.Characters))

	entropyThreshold := 0.8
	isQuantumValid := challenge.Entropy >= entropyThreshold && challenge.QuantumBits > len(challenge.Characters)/2

	signatureValid := q.validateSignature(challenge)

	if isQuantumValid && signatureValid && matchRatio >= 0.8 {
		return true, matchRatio, nil
	}

	if matchRatio >= 0.5 && isQuantumValid {
		return true, matchRatio, nil
	}

	return false, matchRatio, nil
}

func (q *QuantumRandomGenerator) validateSignature(challenge *QuantumChallenge) bool {
	expectedSignature := q.generateSignature(challenge.Characters)
	
	tolerance := 0.1
	similarity := q.calculateStringSimilarity(expectedSignature, challenge.Signature)
	
	return similarity >= (1.0 - tolerance)
}

func (q *QuantumRandomGenerator) calculateStringSimilarity(s1, s2 string) float64 {
	if len(s1) == 0 && len(s2) == 0 {
		return 1.0
	}
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	charMatches := 0
	minLen := len(s1)
	if len(s2) < minLen {
		minLen = len(s2)
	}

	for i := 0; i < minLen; i++ {
		if s1[i] == s2[i] {
			charMatches++
		}
	}

	return float64(charMatches) / float64(len(s1))
}

func (q *QuantumRandomGenerator) GenerateAntiAIPattern(length int) ([]byte, error) {
	if length <= 0 || length > 1000 {
		length = 100
	}

	pattern := make([]byte, length)

	for i := 0; i < length; i++ {
		quantumSeed, err := q.GenerateRandomBytes(8)
		if err != nil {
			return nil, err
		}

		baseValue := quantumSeed[0]
		
		chaosFactor := math.Sin(float64(i) * 0.1 * float64(baseValue))
		
		pattern[i] = byte(int(float64(baseValue)*(1+chaosFactor*0.5)) % 256)
	}

	return pattern, nil
}

func (q *QuantumRandomGenerator) GenerateVisualNoisePattern(width, height int) ([][]float64, error) {
	if width <= 0 || height <= 0 || width > 1000 || height > 1000 {
		return nil, fmt.Errorf("invalid dimensions")
	}

	pattern := make([][]float64, height)
	for i := range pattern {
		pattern[i] = make([]float64, width)
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			timestamp := float64(time.Now().UnixNano())
			phase := float64(x)*0.1 + float64(y)*0.1 + timestamp*1e-9
			
			quantumNoise := math.Sin(phase) * math.Cos(phase*1.5)
			
			gaussianNoise := boxMullerTransform() * 0.3
			
			shotNoise := (randFloat64() - 0.5) * 0.2
			
			pattern[y][x] = (quantumNoise + gaussianNoise + shotNoise) / 3.0
		}
	}

	return pattern, nil
}

func (q *QuantumRandomGenerator) GetEntropyPoolStatus() map[string]interface{} {
	q.poolMutex.Lock()
	defer q.poolMutex.Unlock()

	entropySum := 0
	for _, b := range q.entropyPool {
		entropySum += int(b)
	}
	avgEntropy := float64(entropySum) / float64(len(q.entropyPool))

	return map[string]interface{}{
		"pool_size":       len(q.entropyPool),
		"last_refresh":    q.lastRefreshTime,
		"refresh_interval": q.refreshInterval.String(),
		"average_entropy": avgEntropy,
		"quantum_source": map[string]interface{}{
			"phase":     q.noiseSource.phase,
			"amplitude": q.noiseSource.amplitude,
			"frequency": q.noiseSource.frequency,
		},
	}
}

func (q *QuantumRandomGenerator) EstimateBreakingComplexity(challenge *QuantumChallenge) float64 {
	baseComplexity := math.Pow(2, challenge.Entropy)
	
	quantumBonus := math.Pow(2, float64(challenge.QuantumBits)*0.1)
	
	predictionBonus := 1.0
	if challenge.PredictionResistant {
		predictionBonus = 10.0
	}
	
	totalComplexity := baseComplexity * quantumBonus * predictionBonus
	
	return math.Min(totalComplexity, 1e20)
}
