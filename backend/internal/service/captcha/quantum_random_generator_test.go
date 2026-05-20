package captcha

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQuantumRandomGenerator_Generate(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	config := &QuantumCaptchaConfig{
		Type:             "quantum_pattern",
		SeedLength:      32,
		NoiseIntensity:  0.5,
		PatternComplexity: 5,
	}

	data, err := generator.Generate(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.NotEmpty(t, data.SessionID)
	assert.Contains(t, data.SessionID, "q_")
	assert.NotEmpty(t, data.SeedData)
	assert.NotNil(t, data.NoisePattern)
	assert.NotNil(t, data.ChallengeData)
	assert.NotNil(t, data.VerificationData)
	assert.Greater(t, data.EntropyEstimate, 0.0)
}

func TestQuantumRandomGenerator_Generate_DefaultConfig(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	data, err := generator.Generate(context.Background(), &QuantumCaptchaConfig{})
	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Equal(t, 32, len(data.SeedData)/4*3)
}

func TestQuantumRandomGenerator_Generate_CustomSeedLength(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	config := &QuantumCaptchaConfig{
		SeedLength: 64,
	}

	data, err := generator.Generate(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Greater(t, len(data.SeedData), 32)
}

func TestQuantumRandomGenerator_Generate_ComplexPattern(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	config := &QuantumCaptchaConfig{
		PatternComplexity: 8,
	}

	data, err := generator.Generate(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, data)
}

func TestQuantumRandomGenerator_Generate_MultipleTimes(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	sessions := make(map[string]bool)

	for i := 0; i < 10; i++ {
		data, err := generator.Generate(context.Background(), &QuantumCaptchaConfig{})
		assert.NoError(t, err)
		assert.NotNil(t, data)
		assert.False(t, sessions[data.SessionID], "Session ID should be unique")
		sessions[data.SessionID] = true
	}
}

func TestQuantumRandomGenerator_GenerateQuantumSeed(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	seed1, err := generator.generateQuantumSeed(32)
	assert.NoError(t, err)
	assert.Len(t, seed1, 32)

	seed2, err := generator.generateQuantumSeed(32)
	assert.NoError(t, err)
	assert.Len(t, seed2, 32)

	assert.NotEqual(t, seed1, seed2, "Seeds should be different")
}

func TestQuantumRandomGenerator_GenerateQuantumSeed_LongLength(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	seed, err := generator.generateQuantumSeed(4096)
	assert.NoError(t, err)
	assert.Len(t, seed, 4096)
}

func TestQuantumRandomGenerator_ApplyQuantumEnhancement(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	seed := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	enhanced := generator.applyQuantumEnhancement(seed)

	assert.Len(t, enhanced, len(seed))
	assert.NotEqual(t, seed, enhanced, "Enhanced should differ from original")
}

func TestQuantumRandomGenerator_GenerateNoisePattern(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	pattern, err := generator.generateNoisePattern(256, 0.5)
	assert.NoError(t, err)
	assert.Len(t, pattern, 256)

	nonZero := 0
	for _, b := range pattern {
		if b > 0 {
			nonZero++
		}
	}
	assert.Greater(t, nonZero, 0, "Pattern should have non-zero values")
}

func TestQuantumRandomGenerator_GenerateNoisePattern_HighIntensity(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	pattern, err := generator.generateNoisePattern(256, 1.0)
	assert.NoError(t, err)
	assert.Len(t, pattern, 256)
}

func TestQuantumRandomGenerator_GenerateNoisePattern_ZeroIntensity(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	pattern, err := generator.generateNoisePattern(256, 0.0)
	assert.NoError(t, err)
	assert.Len(t, pattern, 256)
}

func TestQuantumRandomGenerator_GetQuantumByte(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	b1, err := generator.getQuantumByte()
	assert.NoError(t, err)

	b2, err := generator.getQuantumByte()
	assert.NoError(t, err)

	assert.NotEqual(t, b1, b2, "Bytes should be different")
}

func TestQuantumRandomGenerator_GenerateChallenge_Simple(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	seed := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	challenge, verification, err := generator.generateChallenge(seed, 2)
	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.NotNil(t, verification)
	assert.Equal(t, "simple_pattern", challenge["type"])
}

func TestQuantumRandomGenerator_GenerateChallenge_Sequence(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	seed := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	challenge, verification, err := generator.generateChallenge(seed, 4)
	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, "sequence", challenge["type"])
}

func TestQuantumRandomGenerator_GenerateChallenge_Complex(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	seed := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	challenge, verification, err := generator.generateChallenge(seed, 7)
	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, "complex_pattern", challenge["type"])
}

func TestQuantumRandomGenerator_GeneratePatternPositions(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	seed := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	positions := generator.generatePatternPositions(seed, 4)

	assert.Len(t, positions, 4)

	unique := make(map[int]bool)
	for _, p := range positions {
		assert.Less(t, p, 16)
		assert.GreaterOrEqual(t, p, 0)
		assert.False(t, unique[p], "Positions should be unique")
		unique[p] = true
	}
}

func TestQuantumRandomGenerator_GenerateSequence(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	seed := []byte{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	sequence := generator.generateSequence(seed, 6)

	assert.Len(t, sequence, 6)
	for _, s := range sequence {
		assert.GreaterOrEqual(t, s, 0)
		assert.Less(t, s, 10)
	}
}

func TestQuantumRandomGenerator_EstimateEntropy(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	seed := make([]byte, 32)
	for i := range seed {
		seed[i] = byte(i)
	}
	noise := make([]byte, 256)

	entropy := generator.estimateEntropy(seed, noise)
	assert.Greater(t, entropy, 0.0)
	assert.LessOrEqual(t, entropy, float64(len(seed))*8.0)
}

func TestQuantumRandomGenerator_CalculateUnpredictability(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	unpredictability := generator.calculateUnpredictability(200, 32)
	assert.Greater(t, unpredictability, 0.0)
	assert.LessOrEqual(t, unpredictability, 1.0)

	perfectUnpredictability := generator.calculateUnpredictability(256, 32)
	assert.Equal(t, 1.0, perfectUnpredictability)
}

func TestQuantumRandomGenerator_Verify_CorrectResponse(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	captchaData, _ := generator.Generate(context.Background(), &QuantumCaptchaConfig{
		PatternComplexity: 2,
	})

	verification := captchaData.VerificationData.(map[string]interface{})
	challengeType := captchaData.ChallengeData.(map[string]interface{})["type"].(string)

	var response interface{}
	if challengeType == "simple_pattern" {
		response = verification["correct_positions"]
	}

	result, err := generator.Verify(context.Background(), &QuantumVerifyRequest{
		SessionID: captchaData.SessionID,
		Response:  response,
	}, captchaData)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestQuantumRandomGenerator_Verify_ExpiredSession(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	captchaData, _ := generator.Generate(context.Background(), &QuantumCaptchaConfig{})

	captchaData.ExpiresAt = time.Now().Add(-1 * time.Hour).Unix()

	result, err := generator.Verify(context.Background(), &QuantumVerifyRequest{
		SessionID: captchaData.SessionID,
		Response:  "test",
	}, captchaData)

	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "验证码已过期", result.Message)
}

func TestQuantumRandomGenerator_Verify_InvalidData(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	captchaData := &QuantumCaptchaData{
		SessionID:     "test-session",
		ExpiresAt:     time.Now().Add(5 * time.Minute).Unix(),
		VerificationData: "invalid",
	}

	result, err := generator.Verify(context.Background(), &QuantumVerifyRequest{
		SessionID: captchaData.SessionID,
		Response:  "test",
	}, captchaData)

	assert.NoError(t, err)
	assert.False(t, result.Success)
}

func TestQuantumRandomGenerator_CompareSlices(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	tests := []struct {
		name     string
		a        []int
		b        []int
		expected bool
	}{
		{"equal", []int{1, 2, 3}, []int{1, 2, 3}, true},
		{"not equal", []int{1, 2, 3}, []int{1, 2, 4}, false},
		{"different length", []int{1, 2}, []int{1, 2, 3}, false},
		{"empty", []int{}, []int{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.compareSlices(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQuantumRandomGenerator_GetPoolStatus(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	size, available, sourceType := generator.GetPoolStatus()
	assert.Greater(t, size, 0)
	assert.Greater(t, available, 0)
	assert.NotEmpty(t, sourceType)
}

func TestQuantumRandomGenerator_RefillPool(t *testing.T) {
	generator := NewQuantumRandomGenerator()

	generator.RefillPool()

	size, available, _ := generator.GetPoolStatus()
	assert.Equal(t, size, available)
}

func TestQuantumNoiseGenerator_Generate2DNoise(t *testing.T) {
	generator := NewQuantumNoiseGenerator()

	noise, err := generator.Generate2DNoise(100, 100, 0.5)
	assert.NoError(t, err)
	assert.Len(t, noise, 100)
	assert.Len(t, noise[0], 100)
}

func TestQuantumNoiseGenerator_Generate2DNoise_SmallSize(t *testing.T) {
	generator := NewQuantumNoiseGenerator()

	noise, err := generator.Generate2DNoise(10, 10, 0.3)
	assert.NoError(t, err)
	assert.Len(t, noise, 10)
}

func TestQuantumNoiseGenerator_Generate2DNoise_LargeSize(t *testing.T) {
	generator := NewQuantumNoiseGenerator()

	noise, err := generator.Generate2DNoise(500, 500, 0.8)
	assert.NoError(t, err)
	assert.Len(t, noise, 500)
}

func TestQuantumNoiseGenerator_Generate3DNoise(t *testing.T) {
	generator := NewQuantumNoiseGenerator()

	noise, err := generator.Generate3DNoise(10, 10, 10, 0.5)
	assert.NoError(t, err)
	assert.Len(t, noise, 10)
	assert.Len(t, noise[0], 10)
	assert.Len(t, noise[0][0], 10)
}

func TestQuantumNoiseGenerator_ApplyNoiseToImage(t *testing.T) {
	generator := NewQuantumNoiseGenerator()

	pixels := make([][]byte, 100)
	for i := range pixels {
		pixels[i] = make([]byte, 100)
		for j := range pixels[i] {
			pixels[i][j] = 128
		}
	}

	noisyPixels, err := generator.ApplyNoiseToImage(pixels, 0.3)
	assert.NoError(t, err)
	assert.Len(t, noisyPixels, 100)
}

func TestQuantumNoiseGenerator_ApplyNoiseToImage_Empty(t *testing.T) {
	generator := NewQuantumNoiseGenerator()

	pixels := [][]byte{}
	noisyPixels, err := generator.ApplyNoiseToImage(pixels, 0.5)
	assert.NoError(t, err)
	assert.Len(t, noisyPixels, 0)
}

func TestQuantumNoiseGenerator_GenerateRandomWalk(t *testing.T) {
	generator := NewQuantumNoiseGenerator()

	walk, err := generator.GenerateRandomWalk(100, 1.0)
	assert.NoError(t, err)
	assert.Len(t, walk, 101)
	assert.Equal(t, 0.0, walk[0][0])
	assert.Equal(t, 0.0, walk[0][1])
}

func TestQuantumNoiseGenerator_GenerateRandomWalk_Small(t *testing.T) {
	generator := NewQuantumNoiseGenerator()

	walk, err := generator.GenerateRandomWalk(10, 0.5)
	assert.NoError(t, err)
	assert.Len(t, walk, 11)
}

func TestQuantumNoiseGenerator_GenerateRandomWalk_Large(t *testing.T) {
	generator := NewQuantumNoiseGenerator()

	walk, err := generator.GenerateRandomWalk(10000, 2.0)
	assert.NoError(t, err)
	assert.Len(t, walk, 10001)
}

func TestQuantumCaptchaData_Fields(t *testing.T) {
	data := &QuantumCaptchaData{
		SessionID:        "test-session",
		Type:             "quantum_pattern",
		SeedData:         "base64encoded",
		NoisePattern:     []byte{1, 2, 3},
		ChallengeData:    map[string]interface{}{"type": "simple"},
		VerificationData: map[string]interface{}{"correct": true},
		EntropyEstimate:  200.5,
		Timestamp:        1234567890,
		ExpiresAt:        1234567890,
		Metadata:         map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, "test-session", data.SessionID)
	assert.Equal(t, "quantum_pattern", data.Type)
	assert.Equal(t, 200.5, data.EntropyEstimate)
}

func TestQuantumCaptchaConfig_Fields(t *testing.T) {
	config := &QuantumCaptchaConfig{
		SessionID:        "config-session",
		Type:             "test_type",
		SeedLength:       64,
		NoiseIntensity:    0.7,
		PatternComplexity: 8,
		EntropyBits:      512.0,
		Unpredictability:  0.95,
	}

	assert.Equal(t, "config-session", config.SessionID)
	assert.Equal(t, 64, config.SeedLength)
	assert.Equal(t, 0.7, config.NoiseIntensity)
	assert.Equal(t, 8, config.PatternComplexity)
}

func TestQuantumVerifyRequest_Fields(t *testing.T) {
	req := &QuantumVerifyRequest{
		SessionID:   "verify-session",
		Response:    []int{1, 2, 3},
		ResponseTime: 5000,
	}

	assert.Equal(t, "verify-session", req.SessionID)
	assert.Equal(t, 5000, req.ResponseTime)
}

func TestQuantumVerifyResult_Fields(t *testing.T) {
	result := &QuantumVerifyResult{
		Success:          true,
		Score:            1.0,
		Message:          "验证成功",
		EntropyUsed:      200.0,
		Unpredictability: 0.95,
	}

	assert.True(t, result.Success)
	assert.Equal(t, 1.0, result.Score)
	assert.Equal(t, "验证成功", result.Message)
	assert.Equal(t, 200.0, result.EntropyUsed)
	assert.Equal(t, 0.95, result.Unpredictability)
}

func TestNewQuantumRandomGenerator(t *testing.T) {
	generator := NewQuantumRandomGenerator()
	assert.NotNil(t, generator)
	assert.NotNil(t, generator.entropyPool)
	assert.Greater(t, len(generator.entropyPool), 0)
}

func TestNewQuantumNoiseGenerator(t *testing.T) {
	generator := NewQuantumNoiseGenerator()
	assert.NotNil(t, generator)
	assert.NotNil(t, generator.baseGenerator)
}

func BenchmarkQuantumRandomGenerator_Generate(b *testing.B) {
	generator := NewQuantumRandomGenerator()
	config := &QuantumCaptchaConfig{
		SeedLength:      32,
		NoiseIntensity:  0.5,
		PatternComplexity: 5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator.Generate(context.Background(), config)
	}
}

func BenchmarkQuantumRandomGenerator_GenerateSeed(b *testing.B) {
	generator := NewQuantumRandomGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator.generateQuantumSeed(32)
	}
}

func BenchmarkQuantumNoiseGenerator_Generate2DNoise(b *testing.B) {
	generator := NewQuantumNoiseGenerator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator.Generate2DNoise(100, 100, 0.5)
	}
}
