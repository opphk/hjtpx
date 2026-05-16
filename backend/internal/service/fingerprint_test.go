package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDeviceFingerprintService(t *testing.T) {
	service := NewDeviceFingerprintService()
	assert.NotNil(t, service)
}

func TestGenerateFingerprintHash(t *testing.T) {
	service := NewDeviceFingerprintService()

	data := FingerprintData{
		UserAgent:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		ScreenWidth:         1920,
		ScreenHeight:        1080,
		ColorDepth:          24,
		Timezone:           "Asia/Shanghai",
		Language:           "zh-CN",
		Platform:           "Win32",
		HardwareConcurrency: 8,
		DeviceMemory:       8,
		TouchPoints:        0,
		WebGLVendor:        "Google Inc.",
		WebGLRenderer:      "ANGLE (Intel, Intel(R) UHD Graphics Direct3D11)",
		CanvasFingerprint:  "test-canvas-hash",
		AudioFingerprint:   "test-audio-hash",
		Fonts:              []string{"Arial", "Helvetica"},
		Plugins:            []string{"Plugin1", "Plugin2"},
	}

	hash := service.GenerateFingerprintHash(data)

	assert.NotEmpty(t, hash.UserAgentHash)
	assert.NotEmpty(t, hash.ScreenHash)
	assert.NotEmpty(t, hash.BrowserHash)
	assert.NotEmpty(t, hash.PlatformHash)
	assert.NotEmpty(t, hash.CanvasHash)
	assert.NotEmpty(t, hash.WebGLHash)
	assert.NotEmpty(t, hash.AudioHash)
}

func TestGenerateFingerprintHashEmptyCanvas(t *testing.T) {
	service := NewDeviceFingerprintService()

	data := FingerprintData{
		UserAgent:   "Mozilla/5.0",
		Platform:    "Win32",
		WebGLRenderer: "",
	}

	hash := service.GenerateFingerprintHash(data)

	assert.NotEmpty(t, hash.CanvasHash)
}

func TestGenerateFingerprintHashEmptyAudio(t *testing.T) {
	service := NewDeviceFingerprintService()

	data := FingerprintData{
		UserAgent:   "Mozilla/5.0",
		Platform:    "Win32",
		Language:   "en",
		DeviceMemory: 8,
	}

	hash := service.GenerateFingerprintHash(data)

	assert.NotEmpty(t, hash.AudioHash)
}

func TestCombineHashes(t *testing.T) {
	service := NewDeviceFingerprintService()

	hash := FingerprintHash{
		UserAgentHash: "abc123",
		ScreenHash:    "def456",
		BrowserHash:   "ghi789",
		PlatformHash:  "jkl012",
		CanvasHash:    "mno345",
		WebGLHash:     "pqr678",
		AudioHash:     "stu901",
	}

	combined := service.CombineHashes(hash)

	assert.NotEmpty(t, combined)
	assert.Len(t, combined, 64)
}

func TestHashString(t *testing.T) {
	service := NewDeviceFingerprintService()

	hash1 := service.hashString("test-input")
	hash2 := service.hashString("test-input")
	hash3 := service.hashString("different-input")

	assert.Equal(t, hash1, hash2)
	assert.NotEqual(t, hash1, hash3)
	assert.Len(t, hash1, 64)
}

func TestCalculateSimilarity(t *testing.T) {
	service := NewDeviceFingerprintService()

	tests := []struct {
		name     string
		hash1    string
		hash2    string
		expected float64
	}{
		{
			name:     "identical hashes",
			hash1:    "abcdef1234567890",
			hash2:    "abcdef1234567890",
			expected: 1.0,
		},
		{
			name:     "completely different hashes",
			hash1:    "0000000000000000",
			hash2:    "ffffffffffffffff",
			expected: 0.0,
		},
		{
			name:     "partial match",
			hash1:    "abcdef1234567890",
			hash2:    "abcdef0000000000",
			expected: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.CalculateSimilarity(tt.hash1, tt.hash2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateSimilarityDifferentLengths(t *testing.T) {
	service := NewDeviceFingerprintService()

	result := service.CalculateSimilarity("short", "muchlongerhash")
	assert.Equal(t, 0.0, result)
}

func TestFingerprintWeights(t *testing.T) {
	expectedWeights := map[string]float64{
		"user_agent": 1.5,
		"screen":     1.2,
		"browser":    1.3,
		"platform":   1.0,
		"canvas":     2.0,
		"webgl":      1.8,
		"audio":      1.5,
	}

	for key, expected := range expectedWeights {
		actual, exists := fingerprintWeights[key]
		assert.True(t, exists, "Weight for %s should exist", key)
		assert.Equal(t, expected, actual)
	}
}

func TestFingerprintDataStructure(t *testing.T) {
	data := FingerprintData{
		UserAgent:           "Mozilla/5.0",
		ScreenWidth:         1920,
		ScreenHeight:        1080,
		ColorDepth:          24,
		Timezone:           "UTC+8",
		Language:           "en-US",
		Platform:           "MacIntel",
		HardwareConcurrency: 4,
		DeviceMemory:       8,
		TouchPoints:        0,
		WebGLVendor:        "Apple Inc.",
		WebGLRenderer:      "Apple GPU",
		CanvasFingerprint:  "canvas123",
		AudioFingerprint:   "audio456",
		Fonts:              []string{"Arial"},
		Plugins:            []string{"PDF Viewer"},
		DoNotTrack:         true,
		CookiesEnabled:     true,
		LocalStorage:       true,
		SessionStorage:     true,
	}

	assert.Equal(t, "Mozilla/5.0", data.UserAgent)
	assert.Equal(t, 1920, data.ScreenWidth)
	assert.Equal(t, 1080, data.ScreenHeight)
	assert.True(t, data.DoNotTrack)
	assert.True(t, data.CookiesEnabled)
}

func TestFingerprintHashStructure(t *testing.T) {
	hash := FingerprintHash{
		UserAgentHash: "ua-hash",
		ScreenHash:    "screen-hash",
		BrowserHash:   "browser-hash",
		PlatformHash:  "platform-hash",
		CanvasHash:    "canvas-hash",
		WebGLHash:     "webgl-hash",
		AudioHash:     "audio-hash",
	}

	assert.NotEmpty(t, hash.UserAgentHash)
	assert.NotEmpty(t, hash.ScreenHash)
	assert.NotEmpty(t, hash.BrowserHash)
	assert.NotEmpty(t, hash.PlatformHash)
	assert.NotEmpty(t, hash.CanvasHash)
	assert.NotEmpty(t, hash.WebGLHash)
	assert.NotEmpty(t, hash.AudioHash)
}

func TestRiskAssessmentStructure(t *testing.T) {
	assessment := RiskAssessment{
		Score:          45.5,
		Level:          "medium",
		Factors:        []string{"factor1", "factor2"},
		IsNewDevice:    true,
		IsSharedDevice: false,
		Similarity:     0.85,
	}

	assert.Equal(t, 45.5, assessment.Score)
	assert.Equal(t, "medium", assessment.Level)
	assert.Len(t, assessment.Factors, 2)
	assert.True(t, assessment.IsNewDevice)
	assert.False(t, assessment.IsSharedDevice)
	assert.Equal(t, 0.85, assessment.Similarity)
}

func TestCollectedFingerprintStructure(t *testing.T) {
	fingerprint := CollectedFingerprint{
		FingerprintID: 123,
		Hash:         "hash-value",
		RiskLevel:    "low",
	}

	assert.Equal(t, uint(123), fingerprint.FingerprintID)
	assert.Equal(t, "hash-value", fingerprint.Hash)
	assert.Equal(t, "low", fingerprint.RiskLevel)
}

func TestDeviceInfoStructure(t *testing.T) {
	info := DeviceInfo{
		ID:           456,
		Hash:         "device-hash",
		UserAgent:    "Mozilla/5.0",
		ScreenInfo:   "1920x1080",
		BrowserInfo:  "Chrome",
		PlatformInfo: "Windows",
		VisitCount:   10,
		IsTrusted:    true,
		RiskLevel:    "low",
	}

	assert.Equal(t, uint(456), info.ID)
	assert.Equal(t, "device-hash", info.Hash)
	assert.Equal(t, 10, info.VisitCount)
	assert.True(t, info.IsTrusted)
}

func TestSimilarDeviceStructure(t *testing.T) {
	similar := SimilarDevice{
		DeviceID:   789,
		Similarity: 0.92,
		VisitCount: 5,
	}

	assert.Equal(t, uint(789), similar.DeviceID)
	assert.Equal(t, 0.92, similar.Similarity)
	assert.Equal(t, 5, similar.VisitCount)
}

func TestRiskLevels(t *testing.T) {
	levels := []string{"low", "medium", "high"}

	for _, level := range levels {
		assert.Contains(t, []string{"low", "medium", "high"}, level)
	}
}

func TestFingerprintHashConsistency(t *testing.T) {
	service := NewDeviceFingerprintService()

	data := FingerprintData{
		UserAgent:     "TestAgent",
		Platform:      "TestPlatform",
		WebGLRenderer: "TestRenderer",
	}

	hash1 := service.GenerateFingerprintHash(data)
	hash2 := service.GenerateFingerprintHash(data)

	assert.Equal(t, hash1.UserAgentHash, hash2.UserAgentHash)
	assert.Equal(t, hash1.ScreenHash, hash2.ScreenHash)
	assert.Equal(t, hash1.BrowserHash, hash2.BrowserHash)
	assert.Equal(t, hash1.PlatformHash, hash2.PlatformHash)
	assert.Equal(t, hash1.CanvasHash, hash2.CanvasHash)
	assert.Equal(t, hash1.WebGLHash, hash2.WebGLHash)
	assert.Equal(t, hash1.AudioHash, hash2.AudioHash)
}

func TestFingerprintHashUniqueness(t *testing.T) {
	service := NewDeviceFingerprintService()

	data1 := FingerprintData{
		UserAgent:     "Mozilla/5.0 Chrome",
		Platform:      "Win32",
		WebGLRenderer: "Renderer1",
		ScreenWidth:   1920,
		ScreenHeight:  1080,
	}

	data2 := FingerprintData{
		UserAgent:     "Mozilla/5.0 Firefox",
		Platform:      "Win32",
		WebGLRenderer: "Renderer2",
		ScreenWidth:   1920,
		ScreenHeight:  1080,
	}

	hash1 := service.GenerateFingerprintHash(data1)
	hash2 := service.GenerateFingerprintHash(data2)

	assert.NotEqual(t, hash1.UserAgentHash, hash2.UserAgentHash)
	assert.NotEqual(t, hash1.BrowserHash, hash2.BrowserHash)
	assert.NotEqual(t, hash1.WebGLHash, hash2.WebGLHash)
}
