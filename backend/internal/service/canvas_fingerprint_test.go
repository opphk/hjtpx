package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestNewCanvasFingerprintService(t *testing.T) {
	svc := NewCanvasFingerprintService()

	if svc == nil {
		t.Fatal("NewCanvasFingerprintService returned nil")
	}

	if svc.config == nil {
		t.Fatal("config is nil")
	}

	if !svc.config.EnableTextFingerprint {
		t.Error("EnableTextFingerprint should be true by default")
	}

	if !svc.config.EnableImageAnalysis {
		t.Error("EnableImageAnalysis should be true by default")
	}

	if !svc.config.EnableStabilityTrack {
		t.Error("EnableStabilityTrack should be true by default")
	}

	if len(svc.config.SampleTexts) == 0 {
		t.Error("SampleTexts should not be empty")
	}

	if svc.config.ImageWidth != 280 {
		t.Errorf("ImageWidth should be 280, got %d", svc.config.ImageWidth)
	}

	if svc.config.ImageHeight != 60 {
		t.Errorf("ImageHeight should be 60, got %d", svc.config.ImageHeight)
	}
}

func TestNewCanvasFingerprintServiceWithConfig(t *testing.T) {
	config := &model.CanvasEnhancementConfig{
		EnableTextFingerprint: false,
		EnableImageAnalysis:   true,
		EnableStabilityTrack: true,
		EnableAnomalyDetect:  true,
		SampleTexts:          []string{"test1", "test2"},
		ImageWidth:           300,
		ImageHeight:          100,
		StabilityThreshold:   0.9,
		AnomalyThreshold:     0.5,
	}

	svc := NewCanvasFingerprintServiceWithConfig(config)

	if svc.config.EnableTextFingerprint {
		t.Error("EnableTextFingerprint should be false")
	}

	if svc.config.ImageWidth != 300 {
		t.Errorf("ImageWidth should be 300, got %d", svc.config.ImageWidth)
	}

	if svc.config.StabilityThreshold != 0.9 {
		t.Errorf("StabilityThreshold should be 0.9, got %f", svc.config.StabilityThreshold)
	}
}

func TestGenerateEnhancedFingerprint(t *testing.T) {
	svc := NewCanvasFingerprintService()

	tests := []struct {
		name     string
		info     *model.EnvInfo
		wantErr  bool
		wantRisk string
	}{
		{
			name: "valid fingerprint",
			info: &model.EnvInfo{
				CanvasFingerprint: "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
				WebGLRenderer:     "NVIDIA GeForce GTX 1080",
				WebGLVendor:       "NVIDIA",
			},
			wantErr:  false,
			wantRisk: "medium",
		},
		{
			name: "empty fingerprint",
			info: &model.EnvInfo{
				CanvasFingerprint: "",
			},
			wantErr:  true,
			wantRisk: "high",
		},
		{
			name: "short fingerprint",
			info: &model.EnvInfo{
				CanvasFingerprint: "abc123",
				WebGLRenderer:     "Intel HD Graphics",
			},
			wantErr:  false,
			wantRisk: "medium",
		},
		{
			name: "fingerprint with software renderer",
			info: &model.EnvInfo{
				CanvasFingerprint: "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
				WebGLRenderer:     "SwiftShader",
			},
			wantErr:  false,
			wantRisk: "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.GenerateEnhancedFingerprint(tt.info)

			if tt.wantErr && result.Success {
				t.Error("expected error but got success")
			}

			if !tt.wantErr && !result.Success {
				t.Errorf("expected success but got error: %s", result.Error)
			}

			if result.RiskLevel != tt.wantRisk {
				t.Errorf("RiskLevel = %s, want %s", result.RiskLevel, tt.wantRisk)
			}

			if result.Fingerprint == "" && !tt.wantErr {
				t.Error("Fingerprint should not be empty for successful result")
			}
		})
	}
}

func TestExtractTextFeatures(t *testing.T) {
	svc := NewCanvasFingerprintService()

	tests := []struct {
		name        string
		fingerprint string
		wantFeatures int
	}{
		{
			name:        "unicode fingerprint",
			fingerprint: "你好世界こんにちは",
			wantFeatures: 1,
		},
		{
			name:        "emoji fingerprint",
			fingerprint: "🏠🎉⭐🚀",
			wantFeatures: 1,
		},
		{
			name:        "hex fingerprint",
			fingerprint: "abcdef1234567890abcdef1234567890",
			wantFeatures: 1,
		},
		{
			name:        "mixed fingerprint",
			fingerprint: "Hello世界🌍123abc",
			wantFeatures: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.extractTextFeatures(tt.fingerprint)

			if result == nil {
				t.Fatal("extractTextFeatures returned nil")
			}

			if result.TextHash == "" {
				t.Error("TextHash should not be empty")
			}

			if len(result.Features) < tt.wantFeatures {
				t.Errorf("Features count = %d, want at least %d", len(result.Features), tt.wantFeatures)
			}
		})
	}
}

func TestExtractGradientFeatures(t *testing.T) {
	svc := NewCanvasFingerprintService()

	tests := []struct {
		name        string
		fingerprint string
		wantHasGrad bool
	}{
		{
			name:        "gradient fingerprint",
			fingerprint: "gradient_linear_color_transition",
			wantHasGrad: true,
		},
		{
			name:        "regular fingerprint",
			fingerprint: "a1b2c3d4e5f6",
			wantHasGrad: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.extractGradientFeatures(tt.fingerprint)

			if result == nil {
				t.Fatal("extractGradientFeatures returned nil")
			}

			hasGrad := result["has_gradient"] == "true"
			if hasGrad != tt.wantHasGrad {
				t.Errorf("has_gradient = %v, want %v", hasGrad, tt.wantHasGrad)
			}

			if result["gradient_hash"] == "" {
				t.Error("gradient_hash should not be empty")
			}
		})
	}
}

func TestComputeEnhancedHash(t *testing.T) {
	svc := NewCanvasFingerprintService()

	features := map[string]interface{}{
		"text_fingerprint": &model.CanvasTextFingerprint{
			TextHash: "abc123",
			Features: []string{"unicode"},
		},
		"gradient_features": map[string]string{
			"has_gradient": "true",
			"gradient_hash": "def456",
		},
	}

	hash := svc.computeEnhancedHash("baseHash123", features)

	if len(hash) != 64 {
		t.Errorf("SHA256 hash should be 64 chars, got %d", len(hash))
	}

	hashDifferent := svc.computeEnhancedHash("differentBase", features)

	if hash == hashDifferent {
		t.Error("Different base should produce different hash")
	}
}

func TestAnalyzeCanvasRisk(t *testing.T) {
	svc := NewCanvasFingerprintService()

	tests := []struct {
		name          string
		features      map[string]interface{}
		info          *model.EnvInfo
		wantRiskLevel string
		minRiskScore  float64
	}{
		{
			name:     "low risk",
			features: make(map[string]interface{}),
			info: &model.EnvInfo{
				CanvasFingerprint: "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
				WebGLRenderer:     "NVIDIA GeForce GTX 1080",
			},
			wantRiskLevel: "medium",
			minRiskScore:  0,
		},
		{
			name:     "empty fingerprint",
			features: make(map[string]interface{}),
			info: &model.EnvInfo{
				CanvasFingerprint: "",
			},
			wantRiskLevel: "high",
			minRiskScore:  100,
		},
		{
			name:     "short fingerprint",
			features: make(map[string]interface{}),
			info: &model.EnvInfo{
				CanvasFingerprint: "abc",
			},
			wantRiskLevel: "medium",
			minRiskScore:  20,
		},
		{
			name:     "software renderer",
			features: make(map[string]interface{}),
			info: &model.EnvInfo{
				CanvasFingerprint: "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
				WebGLRenderer:     "SwiftShader",
			},
			wantRiskLevel: "medium",
			minRiskScore:  15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			riskLevel, riskScore := svc.analyzeCanvasRisk(tt.features, tt.info)

			if riskLevel != tt.wantRiskLevel {
				t.Errorf("RiskLevel = %s, want %s", riskLevel, tt.wantRiskLevel)
			}

			if riskScore < tt.minRiskScore {
				t.Errorf("RiskScore = %f, want at least %f", riskScore, tt.minRiskScore)
			}
		})
	}
}

func TestDetectCanvasAnomalies(t *testing.T) {
	svc := NewCanvasFingerprintService()

	tests := []struct {
		name     string
		fp       string
		features map[string]interface{}
		wantMin  int
	}{
		{
			name:     "normal fingerprint",
			fp:       "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
			features: map[string]interface{}{"text": "test"},
			wantMin:  0,
		},
		{
			name:     "short fingerprint",
			fp:       "abc",
			features: make(map[string]interface{}),
			wantMin:  1,
		},
		{
			name:     "hex only fingerprint",
			fp:       "abcd1234567890abcd1234567890abcd1234567890abcd1234567890abcd1234567890",
			features: make(map[string]interface{}),
			wantMin:  1,
		},
		{
			name:     "empty features",
			fp:       "a1b2c3d4e5f6",
			features: make(map[string]interface{}),
			wantMin:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anomalies := svc.detectCanvasAnomalies(tt.fp, tt.features)

			if len(anomalies) < tt.wantMin {
				t.Errorf("Anomalies count = %d, want at least %d", len(anomalies), tt.wantMin)
			}
		})
	}
}

func TestExtractTextFingerprint(t *testing.T) {
	svc := NewCanvasFingerprintService()

	sampleTexts := []string{"Hello World", "Test Text", "Canvas Fingerprint"}

	results := svc.ExtractTextFingerprint(sampleTexts)

	if len(results) != len(sampleTexts) {
		t.Errorf("Results count = %d, want %d", len(results), len(sampleTexts))
	}

	for i, result := range results {
		if result.TextHash == "" {
			t.Errorf("Result %d: TextHash should not be empty", i)
		}

		if result.TextContent != sampleTexts[i] {
			t.Errorf("Result %d: TextContent = %s, want %s", i, result.TextContent, sampleTexts[i])
		}

		if result.FontFamily == "" {
			t.Errorf("Result %d: FontFamily should not be empty", i)
		}

		if result.FontSize == 0 {
			t.Errorf("Result %d: FontSize should not be zero", i)
		}
	}
}

func TestCalculateImageEntropy(t *testing.T) {
	svc := NewCanvasFingerprintService()

	tests := []struct {
		name      string
		histogram []int
		wantMin   float64
		wantMax   float64
	}{
		{
			name:      "uniform histogram",
			histogram: make([]int, 256),
			wantMin:   0.0,
			wantMax:   0.0,
		},
		{
			name:      "single peak",
			histogram: func() []int {
				h := make([]int, 256)
				h[128] = 1000
				return h
			}(),
			wantMin: 0.0,
			wantMax: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entropy := svc.calculateImageEntropy(tt.histogram)

			if entropy < tt.wantMin {
				t.Errorf("Entropy = %f, want at least %f", entropy, tt.wantMin)
			}

			if entropy > tt.wantMax {
				t.Errorf("Entropy = %f, want at most %f", entropy, tt.wantMax)
			}
		})
	}
}

func TestAnalyzeStability(t *testing.T) {
	svc := NewCanvasFingerprintService()

	fingerprintID := "test-fp-123"
	sessionID := "session-1"

	result1, err := svc.AnalyzeStability(fingerprintID, sessionID)
	if err != nil {
		t.Fatalf("AnalyzeStability failed: %v", err)
	}

	if result1.DeviceID != fingerprintID {
		t.Errorf("DeviceID = %s, want %s", result1.DeviceID, fingerprintID)
	}

	if result1.SessionCount != 1 {
		t.Errorf("SessionCount = %d, want 1", result1.SessionCount)
	}

	if result1.StabilityScore <= 0 {
		t.Error("StabilityScore should be positive")
	}

	result2, err := svc.AnalyzeStability(fingerprintID, sessionID)
	if err != nil {
		t.Fatalf("AnalyzeStability second call failed: %v", err)
	}

	if result2.SessionCount != 2 {
		t.Errorf("SessionCount = %d, want 2", result2.SessionCount)
	}

	result3, err := svc.AnalyzeStability(fingerprintID, "session-2")
	if err != nil {
		t.Fatalf("AnalyzeStability third call failed: %v", err)
	}

	if result3.UniqueFingerprints < 2 {
		t.Errorf("UniqueFingerprints = %d, want at least 2", result3.UniqueFingerprints)
	}
}

func TestCompareFingerprints(t *testing.T) {
	svc := NewCanvasFingerprintService()

	tests := []struct {
		name     string
		hash1    string
		hash2    string
		wantSim  float64
		wantSame bool
	}{
		{
			name:     "identical hashes",
			hash1:    "a1b2c3d4e5f6",
			hash2:    "a1b2c3d4e5f6",
			wantSim:  100.0,
			wantSame: true,
		},
		{
			name:     "completely different hashes",
			hash1:    "a1b2c3d4e5f6",
			hash2:    "1234567890ab",
			wantSim:  0.0,
			wantSame: false,
		},
		{
			name:     "partial match",
			hash1:    "a1b2c3d4e5f6",
			hash2:    "a1b2c3d4xxxx",
			wantSim:  66.67,
			wantSame: false,
		},
		{
			name:     "different length",
			hash1:    "a1b2c3",
			hash2:    "a1b2c3d4e5f6",
			wantSim:  0.0,
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.CompareFingerprints(tt.hash1, tt.hash2)

			if math.Abs(result.Similarity-tt.wantSim) > 0.01 {
				t.Errorf("Similarity = %f, want %f", result.Similarity, tt.wantSim)
			}

			if result.IsSameDevice != tt.wantSame {
				t.Errorf("IsSameDevice = %v, want %v", result.IsSameDevice, tt.wantSame)
			}
		})
	}
}

func TestDetectSpoofing(t *testing.T) {
	svc := NewCanvasFingerprintService()

	tests := []struct {
		name           string
		originalHash   string
		receivedHash   string
		features       map[string]interface{}
		wantSpoofed    bool
	}{
		{
			name:         "identical hashes",
			originalHash: "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
			receivedHash: "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
			features:     map[string]interface{}{"text": "test"},
			wantSpoofed:  false,
		},
		{
			name:         "completely different hashes",
			originalHash: "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
			receivedHash: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcd",
			features:     map[string]interface{}{"text": "test"},
			wantSpoofed:  true,
		},
		{
			name:         "different length",
			originalHash: "a1b2c3d4e5f6",
			receivedHash: "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
			features:     map[string]interface{}{"text": "test"},
			wantSpoofed:  true,
		},
		{
			name:         "missing features",
			originalHash: "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
			receivedHash: "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
			features:     make(map[string]interface{}),
			wantSpoofed:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.DetectSpoofing(tt.originalHash, tt.receivedHash, tt.features)

			if result.IsSpoofed != tt.wantSpoofed {
				t.Errorf("IsSpoofed = %v, want %v", result.IsSpoofed, tt.wantSpoofed)
			}
		})
	}
}

func TestGenerateRenderAnalysis(t *testing.T) {
	svc := NewCanvasFingerprintService()

	canvasHash := "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456"

	features := map[string]interface{}{
		"text_features":     map[string]string{"key": "value"},
		"gradient_features": map[string]string{"key": "value"},
		"text_fingerprint": &model.CanvasTextFingerprint{
			TextHash: "test123",
			Features: []string{"unicode"},
		},
	}

	result := svc.GenerateRenderAnalysis(canvasHash, features)

	if result.CanvasHash != canvasHash {
		t.Errorf("CanvasHash = %s, want %s", result.CanvasHash, canvasHash)
	}

	if len(result.Features) == 0 {
		t.Error("Features should not be empty")
	}

	if result.ComplexityScore <= 0 {
		t.Error("ComplexityScore should be positive")
	}

	if result.UniquenessScore <= 0 {
		t.Error("UniquenessScore should be positive")
	}

	if result.TextFingerprint == nil {
		t.Error("TextFingerprint should not be nil")
	}
}

func TestCalculateStringEntropy(t *testing.T) {
	svc := NewCanvasFingerprintService()

	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{
			name:  "empty string",
			input: "",
			want:  0.0,
		},
		{
			name:  "single char repeated",
			input: "aaaaaaaaaa",
			want:  0.0,
		},
		{
			name:  "all unique chars",
			input: "abcdefghij",
			want:  3.321928094887362,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entropy := svc.calculateStringEntropy(tt.input)

			if math.Abs(entropy-tt.want) > 0.01 {
				t.Errorf("Entropy = %f, want %f", entropy, tt.want)
			}
		})
	}
}

func TestSimulatedCanvasRenderer(t *testing.T) {
	renderer := NewSimulatedCanvasRenderer(280, 60)

	if renderer.width != 280 {
		t.Errorf("width = %d, want 280", renderer.width)
	}

	if renderer.height != 60 {
		t.Errorf("height = %d, want 60", renderer.height)
	}

	renderer.FillText("Test", 10, 20, "Arial", 14)

	renderer.DrawGradient(0, 0, 280, 60, [4]byte{0, 100, 200, 255}, [4]byte{200, 100, 0, 255})

	renderer.DrawBezierCurve([2]int{10, 30}, [2]int{50, 10}, [2]int{100, 50}, [2]int{140, 30})

	renderer.DrawArc(200, 30, 20, 0, math.Pi*2)

	renderer.ApplyShadow(2, 2, 5, [4]byte{0, 0, 0, 100})

	renderer.Composite("multiply")

	data := renderer.GetImageData()
	if len(data) == 0 {
		t.Error("GetImageData returned empty slice")
	}

	hash := renderer.GetHash()
	if len(hash) != 64 {
		t.Errorf("GetHash length = %d, want 64", len(hash))
	}
}

func TestGenerateSimulatedCanvasFingerprint(t *testing.T) {
	fp := GenerateSimulatedCanvasFingerprint("Mozilla/5.0", 1920, 1080)

	if fp == "" {
		t.Error("GenerateSimulatedCanvasFingerprint returned empty string")
	}

	if len(fp) != 64 {
		t.Errorf("Fingerprint length = %d, want 64", len(fp))
	}
}

func TestSimulateTextFingerprintRendering(t *testing.T) {
	hash, err := SimulateTextFingerprintRendering()
	if err != nil {
		t.Fatalf("SimulateTextFingerprintRendering failed: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if len(hash) != 64 {
		t.Errorf("Hash length = %d, want 64", len(hash))
	}
}

func TestSimulateImageDataExtraction(t *testing.T) {
	data, err := SimulateImageDataExtraction()
	if err != nil {
		t.Fatalf("SimulateImageDataExtraction failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Data should not be empty")
	}
}

func TestGetConfig(t *testing.T) {
	svc := NewCanvasFingerprintService()

	config := svc.GetConfig()

	if config == nil {
		t.Fatal("GetConfig returned nil")
	}

	if config.EnableTextFingerprint != svc.config.EnableTextFingerprint {
		t.Error("Config mismatch")
	}
}

func TestUpdateConfig(t *testing.T) {
	svc := NewCanvasFingerprintService()

	newConfig := &model.CanvasEnhancementConfig{
		EnableTextFingerprint: false,
		ImageWidth:            400,
		StabilityThreshold:    0.95,
	}

	svc.UpdateConfig(newConfig)

	if svc.config.EnableTextFingerprint {
		t.Error("EnableTextFingerprint should be false after update")
	}

	if svc.config.ImageWidth != 400 {
		t.Errorf("ImageWidth = %d, want 400", svc.config.ImageWidth)
	}

	if svc.config.StabilityThreshold != 0.95 {
		t.Errorf("StabilityThreshold = %f, want 0.95", svc.config.StabilityThreshold)
	}
}

func TestExportFingerprintData(t *testing.T) {
	svc := NewCanvasFingerprintService()

	svc.AnalyzeStability("export-test-fp", "session-1")

	data, err := svc.ExportFingerprintData("export-test-fp")
	if err != nil {
		t.Fatalf("ExportFingerprintData failed: %v", err)
	}

	if data == "" {
		t.Error("Exported data should not be empty")
	}

	var fp model.CanvasFingerprintStability
	if err := json.Unmarshal([]byte(data), &fp); err != nil {
		t.Fatalf("Failed to unmarshal exported data: %v", err)
	}

	if fp.FingerprintID != "export-test-fp" {
		t.Errorf("FingerprintID = %s, want export-test-fp", fp.FingerprintID)
	}
}

func TestImportFingerprintData(t *testing.T) {
	svc := NewCanvasFingerprintService()

	jsonData := `{
		"fingerprint_id": "import-test-fp",
		"session_id": "session-1",
		"hit_count": 5,
		"stability_score": 0.8,
		"is_stable": true,
		"variations": ["session-2", "session-3"]
	}`

	err := svc.ImportFingerprintData("import-test-fp", jsonData)
	if err != nil {
		t.Fatalf("ImportFingerprintData failed: %v", err)
	}

	result, err := svc.AnalyzeStability("import-test-fp", "session-1")
	if err != nil {
		t.Fatalf("AnalyzeStability after import failed: %v", err)
	}

	if result.SessionCount < 5 {
		t.Errorf("SessionCount = %d, want at least 5", result.SessionCount)
	}
}

func TestClearExpiredData(t *testing.T) {
	svc := NewCanvasFingerprintService()

	svc.AnalyzeStability("expire-test-fp", "session-1")

	time.Sleep(10 * time.Millisecond)

	removed := svc.ClearExpiredData(1 * time.Millisecond)

	if removed < 1 {
		t.Errorf("Removed = %d, want at least 1", removed)
	}
}

func TestGetStatistics(t *testing.T) {
	svc := NewCanvasFingerprintService()

	for i := 0; i < 5; i++ {
		svc.AnalyzeStability(fmt.Sprintf("stat-test-fp-%d", i), fmt.Sprintf("session-%d", i))
	}

	stats := svc.GetStatistics()

	if totalFPs, ok := stats["total_fingerprints"].(int); !ok || totalFPs != 5 {
		t.Errorf("total_fingerprints = %v, want 5", stats["total_fingerprints"])
	}

	if totalHits, ok := stats["total_hits"].(int); !ok || totalHits != 5 {
		t.Errorf("total_hits = %v, want 5", stats["total_hits"])
	}

	if avgHits, ok := stats["avg_hits_per_fp"].(float64); !ok || avgHits != 1.0 {
		t.Errorf("avg_hits_per_fp = %v, want 1.0", avgHits)
	}
}

func TestExtractArcFeatures(t *testing.T) {
	svc := NewCanvasFingerprintService()

	result := svc.extractArcFeatures("arc_circle_ellipse_pie")

	if result["has_arc"] != "true" {
		t.Error("has_arc should be true")
	}

	if result["arc_hash"] == "" {
		t.Error("arc_hash should not be empty")
	}
}

func TestExtractShadowFeatures(t *testing.T) {
	svc := NewCanvasFingerprintService()

	result := svc.extractShadowFeatures("shadow_blur_offset_x_offset_y")

	if result["has_shadow"] != "true" {
		t.Error("has_shadow should be true")
	}

	if result["shadow_hash"] == "" {
		t.Error("shadow_hash should not be empty")
	}
}

func TestExtractCompositeFeatures(t *testing.T) {
	svc := NewCanvasFingerprintService()

	result := svc.extractCompositeFeatures("source_over_multiply_screen_overlay")

	if result["has_composite"] != "true" {
		t.Error("has_composite should be true")
	}

	compositeCount := result["composite_count"]
	if compositeCount == "" || compositeCount == "0" {
		t.Error("composite_count should be greater than 0")
	}

	if result["composite_types"] == "" {
		t.Error("composite_types should not be empty")
	}
}

func TestHashString(t *testing.T) {
	svc := NewCanvasFingerprintService()

	hash := svc.hashString("test string")

	if len(hash) != 64 {
		t.Errorf("SHA256 hash should be 64 chars, got %d", len(hash))
	}

	hashDifferent := svc.hashString("different string")

	if hash == hashDifferent {
		t.Error("Different input should produce different hash")
	}
}

func TestContainsString(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !containsString(slice, "b") {
		t.Error("containsString should return true for existing element")
	}

	if containsString(slice, "d") {
		t.Error("containsString should return false for non-existing element")
	}

	if containsString(nil, "a") {
		t.Error("containsString should return false for nil slice")
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{-100, 100},
	}

	for _, tt := range tests {
		result := abs(tt.input)
		if result != tt.want {
			t.Errorf("abs(%d) = %d, want %d", tt.input, result, tt.want)
		}
	}
}

func TestCanvasFingerprintResultSerialization(t *testing.T) {
	svc := NewCanvasFingerprintService()

	info := &model.EnvInfo{
		CanvasFingerprint: "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
		WebGLRenderer:    "NVIDIA GeForce GTX 1080",
	}

	result := svc.GenerateEnhancedFingerprint(info)

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	if !strings.Contains(string(jsonData), "fingerprint") {
		t.Error("JSON should contain fingerprint field")
	}

	if !strings.Contains(string(jsonData), "enhanced_features") {
		t.Error("JSON should contain enhanced_features field")
	}

	if !strings.Contains(string(jsonData), "risk_level") {
		t.Error("JSON should contain risk_level field")
	}
}

func TestCanvasStabilityResultSerialization(t *testing.T) {
	svc := NewCanvasFingerprintService()

	result, err := svc.AnalyzeStability("serialize-test-fp", "session-1")
	if err != nil {
		t.Fatalf("AnalyzeStability failed: %v", err)
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	if !strings.Contains(string(jsonData), "device_id") {
		t.Error("JSON should contain device_id field")
	}

	if !strings.Contains(string(jsonData), "stability_score") {
		t.Error("JSON should contain stability_score field")
	}

	if !strings.Contains(string(jsonData), "is_trusted") {
		t.Error("JSON should contain is_trusted field")
	}
}

func TestCanvasRenderAnalysisSerialization(t *testing.T) {
	svc := NewCanvasFingerprintService()

	features := map[string]interface{}{
		"text_features":     map[string]string{"key": "value"},
		"gradient_features": map[string]string{"key": "value"},
	}

	result := svc.GenerateRenderAnalysis("test-hash-1234567890123456789012345678901234567890123456", features)

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	if !strings.Contains(string(jsonData), "canvas_hash") {
		t.Error("JSON should contain canvas_hash field")
	}

	if !strings.Contains(string(jsonData), "features") {
		t.Error("JSON should contain features field")
	}

	if !strings.Contains(string(jsonData), "complexity_score") {
		t.Error("JSON should contain complexity_score field")
	}

	if !strings.Contains(string(jsonData), "uniqueness_score") {
		t.Error("JSON should contain uniqueness_score field")
	}
}
