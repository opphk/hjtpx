package captcha

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"time"
)

type QuantumRandomGenerator struct {
	entropyPool   []byte
	poolMu        sync.Mutex
	poolSize      int
	refillThreshold float64
	sourceType    string
}

type QuantumCaptchaConfig struct {
	SessionID       string
	Type            string
	SeedLength      int
	NoiseIntensity  float64
	PatternComplexity int
	EntropyBits     float64
	Unpredictability float64
}

type QuantumCaptchaData struct {
	SessionID       string                 `json:"session_id"`
	Type            string                 `json:"type"`
	SeedData        string                 `json:"seed_data"`
	NoisePattern    []byte                 `json:"noise_pattern"`
	ChallengeData   interface{}            `json:"challenge_data"`
	VerificationData interface{}           `json:"verification_data"`
	EntropyEstimate float64               `json:"entropy_estimate"`
	Timestamp       int64                  `json:"timestamp"`
	ExpiresAt       int64                  `json:"expires_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type QuantumVerifyRequest struct {
	SessionID  string `json:"session_id"`
	Response   interface{} `json:"response"`
	ResponseTime int64 `json:"response_time"`
}

type QuantumVerifyResult struct {
	Success         bool    `json:"success"`
	Score           float64 `json:"score"`
	Message         string  `json:"message"`
	EntropyUsed     float64 `json:"entropy_used"`
	Unpredictability float64 `json:"unpredictability"`
}

func NewQuantumRandomGenerator() *QuantumRandomGenerator {
	return &QuantumRandomGenerator{
		entropyPool:     make([]byte, 4096),
		poolSize:        4096,
		refillThreshold: 0.25,
		sourceType:      "quantum_simulated",
	}
}

func (g *QuantumRandomGenerator) Generate(ctx context.Context, config *QuantumCaptchaConfig) (*QuantumCaptchaData, error) {
	sessionID := g.generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	g.poolMu.Lock()
	g.ensurePoolCapacity()
	g.poolMu.Unlock()

	seedLength := config.SeedLength
	if seedLength == 0 {
		seedLength = 32
	}

	seedData, err := g.generateQuantumSeed(seedLength)
	if err != nil {
		return nil, err
	}

	noiseIntensity := config.NoiseIntensity
	if noiseIntensity == 0 {
		noiseIntensity = 0.5
	}

	noisePattern, err := g.generateNoisePattern(256, noiseIntensity)
	if err != nil {
		return nil, err
	}

	patternComplexity := config.PatternComplexity
	if patternComplexity == 0 {
		patternComplexity = 5
	}

	challengeData, verificationData, err := g.generateChallenge(seedData, patternComplexity)
	if err != nil {
		return nil, err
	}

	entropyEstimate := g.estimateEntropy(seedData, noisePattern)
	unpredictability := g.calculateUnpredictability(entropyEstimate, seedLength)

	return &QuantumCaptchaData{
		SessionID:        sessionID,
		Type:             config.Type,
		SeedData:         base64.StdEncoding.EncodeToString(seedData),
		NoisePattern:     noisePattern,
		ChallengeData:    challengeData,
		VerificationData: verificationData,
		EntropyEstimate:  entropyEstimate,
		Timestamp:        time.Now().Unix(),
		ExpiresAt:        expiresAt.Unix(),
		Metadata: map[string]interface{}{
			"pool_size":      g.poolSize,
			"source_type":    g.sourceType,
			"noise_intensity": noiseIntensity,
			"complexity":     patternComplexity,
		},
	}, nil
}

func (g *QuantumRandomGenerator) generateQuantumSeed(length int) ([]byte, error) {
	g.poolMu.Lock()
	defer g.poolMu.Unlock()

	if len(g.entropyPool) < length {
		g.refillPool()
	}

	seed := make([]byte, length)
	copy(seed, g.entropyPool[:length])

	g.entropyPool = g.entropyPool[length:]
	if len(g.entropyPool) < int(float64(g.poolSize)*g.refillThreshold) {
		g.refillPoolUnsafe()
	}

	quantumEnhanced := g.applyQuantumEnhancement(seed)

	return quantumEnhanced, nil
}

func (g *QuantumRandomGenerator) ensurePoolCapacity() {
	if len(g.entropyPool) < int(float64(g.poolSize)*g.refillThreshold) {
		g.refillPoolUnsafe()
	}
}

func (g *QuantumRandomGenerator) refillPool() {
	g.refillPoolUnsafe()
}

func (g *QuantumRandomGenerator) refillPoolUnsafe() {
	randomBytes := make([]byte, g.poolSize)
	_, err := rand.Read(randomBytes)
	if err != nil {
		for i := range randomBytes {
			randomBytes[i] = byte(time.Now().UnixNano() % 256)
		}
	}

	g.entropyPool = append(g.entropyPool, randomBytes...)
}

func (g *QuantumRandomGenerator) applyQuantumEnhancement(seed []byte) []byte {
	enhanced := make([]byte, len(seed))

	for i := range seed {
		timestamp := time.Now().UnixNano()
		phase := float64(timestamp%1000) / 1000.0 * 2 * math.Pi

		quantumBit := float64(seed[i])
		quantumFactor := math.Sin(phase) * 0.5
		enhancedByte := quantumBit ^ byte(int(quantumFactor*256)%256)

		timeSeed := byte((timestamp >> (i % 8)) & 0xFF)
		enhanced[i] = enhancedByte ^ timeSeed
	}

	return enhanced
}

func (g *QuantumRandomGenerator) generateNoisePattern(length int, intensity float64) ([]byte, error) {
	pattern := make([]byte, length)

	for i := 0; i < length; i++ {
		randomValue, err := g.getQuantumByte()
		if err != nil {
			randomValue = byte(time.Now().UnixNano() % 256)
		}

		noiseValue := float64(randomValue) * intensity
		pattern[i] = byte(noiseValue)
	}

	return pattern, nil
}

func (g *QuantumRandomGenerator) getQuantumByte() (byte, error) {
	g.poolMu.Lock()
	defer g.poolMu.Unlock()

	if len(g.entropyPool) < 1 {
		g.refillPoolUnsafe()
	}

	b := g.entropyPool[0]
	g.entropyPool = g.entropyPool[1:]

	return b, nil
}

func (g *QuantumRandomGenerator) generateChallenge(seed []byte, complexity int) (map[string]interface{}, map[string]interface{}, error) {
	challenge := make(map[string]interface{})
	verification := make(map[string]interface{})

	switch complexity {
	case 1, 2:
		challenge["type"] = "simple_pattern"
		challenge["positions"] = g.generatePatternPositions(seed, 4)
		verification["correct_positions"] = challenge["positions"]

	case 3, 4:
		challenge["type"] = "sequence"
		challenge["sequence"] = g.generateSequence(seed, 6)
		verification["correct_sequence"] = challenge["sequence"]

	default:
		challenge["type"] = "complex_pattern"
		challenge["positions"] = g.generatePatternPositions(seed, complexity)
		challenge["sequence"] = g.generateSequence(seed, complexity+2)
		verification["correct_positions"] = challenge["positions"]
		verification["correct_sequence"] = challenge["sequence"]
	}

	return challenge, verification, nil
}

func (g *QuantumRandomGenerator) generatePatternPositions(seed []byte, count int) []int {
	positions := make([]int, count)
	used := make(map[int]bool)

	for i := 0; i < count; i++ {
		var pos int
		for {
			seedIndex := i % len(seed)
			pos = int(seed[seedIndex]) % 16
			if !used[pos] {
				used[pos] = true
				break
			}
			pos = (pos + 1) % 16
		}
		positions[i] = pos
	}

	return positions
}

func (g *QuantumRandomGenerator) generateSequence(seed []byte, length int) []int {
	sequence := make([]int, length)

	for i := 0; i < length; i++ {
		seedIndex := (i * 3) % len(seed)
		value := int(seed[seedIndex]) % 10
		sequence[i] = value
	}

	return sequence
}

func (g *QuantumRandomGenerator) estimateEntropy(seed []byte, noise []byte) float64 {
	seedEntropy := float64(len(seed)) * 7.5

	var noiseBytes int
	for _, b := range noise {
		if b > 0 {
			noiseBytes++
		}
	}
	noiseEntropy := float64(noiseBytes) * 0.5

	totalEntropy := seedEntropy + noiseEntropy

	return math.Min(totalEntropy, float64(len(seed))*8.0)
}

func (g *QuantumRandomGenerator) calculateUnpredictability(entropy float64, seedLength int) float64 {
	maxEntropy := float64(seedLength) * 8.0
	unpredictability := entropy / maxEntropy

	return math.Min(1.0, unpredictability)
}

func (g *QuantumRandomGenerator) Verify(ctx context.Context, req *QuantumVerifyRequest, challenge *QuantumCaptchaData) (*QuantumVerifyResult, error) {
	if time.Now().Unix() > challenge.ExpiresAt {
		return &QuantumVerifyResult{
			Success:          false,
			Score:            0.0,
			Message:          "验证码已过期",
			EntropyUsed:      0.0,
			Unpredictability: 0.0,
		}, nil
	}

	verificationData, ok := challenge.VerificationData.(map[string]interface{})
	if !ok {
		return &QuantumVerifyResult{
			Success:          false,
			Score:            0.0,
			Message:          "验证数据无效",
			EntropyUsed:      0.0,
			Unpredictability: 0.0,
		}, nil
	}

	challengeType := "unknown"
	if ct, ok := challenge.ChallengeData.(map[string]interface{}); ok {
		if t, ok := ct["type"].(string); ok {
			challengeType = t
		}
	}

	correct := g.checkResponse(req.Response, verificationData, challengeType)

	result := &QuantumVerifyResult{
		Success:          correct,
		EntropyUsed:       challenge.EntropyEstimate,
		Unpredictability: challenge.EntropyEstimate / (float64(len(challenge.SeedData)) * 8.0),
	}

	if correct {
		result.Score = 1.0
		result.Message = "验证成功"
	} else {
		result.Score = 0.0
		result.Message = "验证失败"
	}

	return result, nil
}

func (g *QuantumRandomGenerator) checkResponse(response interface{}, verification map[string]interface{}, challengeType string) bool {
	if response == nil {
		return false
	}

	switch challengeType {
	case "simple_pattern":
		if positions, ok := response.([]int); ok {
			if correct, ok := verification["correct_positions"].([]int); ok {
				return g.compareSlices(positions, correct)
			}
		}

	case "sequence":
		if sequence, ok := response.([]int); ok {
			if correct, ok := verification["correct_sequence"].([]int); ok {
				return g.compareSlices(sequence, correct)
			}
		}

	case "complex_pattern":
		if respMap, ok := response.(map[string]interface{}); ok {
			if positions, ok := respMap["positions"].([]int); ok {
				if correct, ok := verification["correct_positions"].([]int); ok {
					if !g.compareSlices(positions, correct) {
						return false
					}
				}
			}
			if sequence, ok := respMap["sequence"].([]int); ok {
				if correct, ok := verification["correct_sequence"].([]int); ok {
					return g.compareSlices(sequence, correct)
				}
			}
		}
	}

	return false
}

func (g *QuantumRandomGenerator) compareSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (g *QuantumRandomGenerator) generateSessionID() string {
	seed, _ := g.generateQuantumSeed(16)
	timestamp := time.Now().UnixNano()

	idBytes := make([]byte, 24)
	copy(idBytes, seed)
	binary.BigEndian.PutUint64(idBytes[16:24], uint64(timestamp))

	return fmt.Sprintf("q_%s", base64.RawURLEncoding.EncodeToString(idBytes))
}

func (g *QuantumRandomGenerator) GetPoolStatus() (size int, available int, sourceType string) {
	g.poolMu.Lock()
	defer g.poolMu.Unlock()
	return g.poolSize, len(g.entropyPool), g.sourceType
}

func (g *QuantumRandomGenerator) RefillPool() {
	g.poolMu.Lock()
	defer g.poolMu.Unlock()
	g.refillPoolUnsafe()
}

type QuantumNoiseGenerator struct {
	baseGenerator *QuantumRandomGenerator
}

func NewQuantumNoiseGenerator() *QuantumNoiseGenerator {
	return &QuantumNoiseGenerator{
		baseGenerator: NewQuantumRandomGenerator(),
	}
}

func (g *QuantumNoiseGenerator) Generate2DNoise(width, height int, intensity float64) ([][]float64, error) {
	noise := make([][]float64, height)

	for y := 0; y < height; y++ {
		noise[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			seedByte, _ := g.baseGenerator.getQuantumByte()
			phase := float64(seedByte) / 256.0 * 2 * math.Pi

			frequency := 0.1 + float64(seedByte)/512.0
			amplitude := intensity * (0.5 + float64(time.Now().UnixNano()%1000)/2000.0)

			value := amplitude * math.Sin(2*math.Pi*frequency*float64(x) + phase)
			value += amplitude * 0.5 * math.Sin(4*math.Pi*frequency*float64(y) + phase*1.5)

			noise[y][x] = value
		}
	}

	return noise, nil
}

func (g *QuantumNoiseGenerator) Generate3DNoise(depth, width, height int, intensity float64) ([][][]float64, error) {
	noise := make([][][]float64, depth)

	for z := 0; z < depth; z++ {
		noise[z] = make([][]float64, width)
		for x := 0; x < width; x++ {
			noise[z][x] = make([]float64, height)
			for y := 0; y < height; y++ {
				seedByte, _ := g.baseGenerator.getQuantumByte()
				phase := float64(seedByte) / 256.0 * 2 * math.Pi

				frequency := 0.1 + float64(seedByte)/512.0
				amplitude := intensity * 0.5

				value := amplitude * math.Sin(2*math.Pi*frequency*float64(x) + phase)
				value += amplitude * math.Sin(2*math.Pi*frequency*float64(y) + phase*1.3)
				value += amplitude * math.Sin(2*math.Pi*frequency*float64(z) + phase*1.7)

				noise[z][x][y] = value
			}
		}
	}

	return noise, nil
}

func (g *QuantumNoiseGenerator) ApplyNoiseToImage(pixels [][]byte, intensity float64) ([][]byte, error) {
	height := len(pixels)
	if height == 0 {
		return pixels, nil
	}
	width := len(pixels[0])

	noisyPixels := make([][]byte, height)
	for y := 0; y < height; y++ {
		noisyPixels[y] = make([]byte, width)
		for x := 0; x < width; x++ {
			noiseByte, _ := g.baseGenerator.getQuantumByte()
			noise := float64(noiseByte) * intensity - (intensity * 128)

			original := float64(pixels[y][x])
			noisy := original + noise

			if noisy < 0 {
				noisy = 0
			} else if noisy > 255 {
				noisy = 255
			}

			noisyPixels[y][x] = byte(noisy)
		}
	}

	return noisyPixels, nil
}

func (g *QuantumNoiseGenerator) GenerateRandomWalk(steps int, stepSize float64) ([][2]float64, error) {
	walk := make([][2]float64, steps+1)
	walk[0] = [2]float64{0, 0}

	for i := 1; i <= steps; i++ {
		seedByte1, _ := g.baseGenerator.getQuantumByte()
		seedByte2, _ := g.baseGenerator.getQuantumByte()

		angle := float64(seedByte1)/256.0 * 2 * math.Pi

		dx := stepSize * math.Cos(angle)
		dy := stepSize * math.Sin(angle)

		walk[i][0] = walk[i-1][0] + dx
		walk[i][1] = walk[i-1][1] + dy
	}

	return walk, nil
}
