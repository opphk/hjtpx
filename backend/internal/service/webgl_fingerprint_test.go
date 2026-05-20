package service

import (
	"testing"

	github.com/hjtpx/hjtpx/internal/model"
)

func TestNewWebGLFingerprintService(t *testing.T) {
	service := NewWebGLFingerprintService()

	if service == nil {
		t.Fatal("NewWebGLFingerprintService returned nil")
	}

	if service.config == nil {
		t.Fatal("config should not be nil")
	}

	if !service.config.EnableParameterExtraction {
		t.Error("EnableParameterExtraction should be true by default")
	}

	if !service.config.EnableExtensionAnalysis {
		t.Error("EnableExtensionAnalysis should be true by default")
	}

	if len(service.config.TrustedVendors) == 0 {
		t.Error("TrustedVendors should not be empty")
	}

	if len(service.config.SoftwareRenderers) == 0 {
		t.Error("SoftwareRenderers should not be empty")
	}
}

func TestNewWebGLFingerprintServiceWithConfig(t *testing.T) {
	config := &model.WebGLEnhancementConfig{
		EnableParameterExtraction:   false,
		EnableExtensionAnalysis:     true,
		EnableRendererIdentification: true,
		StabilityThreshold:         0.9,
		AnomalyThreshold:           0.5,
		SimilarityThreshold:        0.9,
		TrustedVendors:             []string{"Test Vendor"},
		SoftwareRenderers:          []string{"Test Renderer"},
		VirtualGPUs:                []string{"Test Virtual"},
	}

	service := NewWebGLFingerprintServiceWithConfig(config)

	if service == nil {
		t.Fatal("NewWebGLFingerprintServiceWithConfig returned nil")
	}

	if service.config.StabilityThreshold != 0.9 {
		t.Errorf("Expected StabilityThreshold 0.9, got %f", service.config.StabilityThreshold)
	}

	if service.config.EnableParameterExtraction {
		t.Error("EnableParameterExtraction should be false when set in config")
	}
}

func TestNewWebGLFingerprintServiceWithNilConfig(t *testing.T) {
	service := NewWebGLFingerprintServiceWithConfig(nil)

	if service == nil {
		t.Fatal("Should return default service when config is nil")
	}

	if service.config == nil {
		t.Fatal("config should not be nil")
	}

	if service.config.StabilityThreshold != 0.8 {
		t.Error("Default StabilityThreshold should be 0.8")
	}
}

func TestGenerateEnhancedFingerprintWebGL(t *testing.T) {
	service := NewWebGLFingerprintService()

	tests := []struct {
		name    string
		info    *model.EnvInfo
		success bool
		riskLevel string
	}{
		{
			name: "Hardware GPU",
			info: &model.EnvInfo{
				WebGLVendor:   "NVIDIA Corporation",
				WebGLRenderer: "GeForce GTX 1080/PCIe/SSE2",
				Fingerprint:   "texture_size:4096",
			},
			success:   true,
			riskLevel: "low",
		},
		{
			name: "Software Renderer",
			info: &model.EnvInfo{
				WebGLVendor:   "Google Inc.",
				WebGLRenderer: "SwiftShader 4.0.0.1",
				Fingerprint:   "texture_size:4096",
			},
			success:   true,
			riskLevel: "medium",
		},
		{
			name: "Virtual GPU",
			info: &model.EnvInfo{
				WebGLVendor:   "VMware, Inc.",
				WebGLRenderer: "llvmpipe (LLVM 12.0.0, 256 bits)",
				Fingerprint:   "texture_size:8192",
			},
			success:   true,
			riskLevel: "medium",
		},
		{
			name: "Empty Data",
			info: &model.EnvInfo{
				WebGLVendor:   "",
				WebGLRenderer: "",
			},
			success:   false,
			riskLevel: "high",
		},
		{
			name: "Missing Vendor",
			info: &model.EnvInfo{
				WebGLVendor:   "",
				WebGLRenderer: "Test Renderer",
			},
			success:   true,
			riskLevel: "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.GenerateEnhancedFingerprint(tt.info)

			if result.Success != tt.success {
				t.Errorf("Expected Success=%v, got %v", tt.success, result.Success)
			}

			if result.RiskLevel != tt.riskLevel {
				t.Errorf("Expected RiskLevel=%s, got %s", tt.riskLevel, result.RiskLevel)
			}

			if result.Success && result.Fingerprint == "" {
				t.Error("Fingerprint should not be empty when success is true")
			}
		})
	}
}

func TestExtractGPUInfo(t *testing.T) {
	service := NewWebGLFingerprintService()

	tests := []struct {
		name           string
		renderer       string
		vendor         string
		expectedType   model.WebGLRendererType
		expectedFamily string
	}{
		{
			name:           "NVIDIA GPU",
			renderer:       "GeForce GTX 1080/PCIe/SSE2",
			vendor:         "NVIDIA Corporation",
			expectedType:   model.WebGLRendererTypeHardware,
			expectedFamily: "NVIDIA",
		},
		{
			name:           "SwiftShader",
			renderer:       "SwiftShader 4.0.0.1",
			vendor:         "Google Inc.",
			expectedType:   model.WebGLRendererTypeSoftware,
			expectedFamily: "Unknown",
		},
		{
			name:           "LLVMpipe Virtual",
			renderer:       "llvmpipe (LLVM 12.0.0, 256 bits)",
			vendor:         "VMware, Inc.",
			expectedType:   model.WebGLRendererTypeSoftware,
			expectedFamily: "Unknown",
		},
		{
			name:           "VirtualBox",
			renderer:       "VirtualBox Graphics Adapter",
			vendor:         "VirtualBox",
			expectedType:   model.WebGLRendererTypeVirtual,
			expectedFamily: "Unknown",
		},
		{
			name:           "AMD GPU",
			renderer:       "Radeon RX 580 Series",
			vendor:         "Advanced Micro Devices, Inc.",
			expectedType:   model.WebGLRendererTypeHardware,
			expectedFamily: "AMD",
		},
		{
			name:           "Intel Integrated",
			renderer:       "Intel(R) HD Graphics 630",
			vendor:         "Intel Inc.",
			expectedType:   model.WebGLRendererTypeHardware,
			expectedFamily: "Intel",
		},
		{
			name:           "Apple M1",
			renderer:       "Apple M1",
			vendor:         "Apple Inc.",
			expectedType:   model.WebGLRendererTypeHardware,
			expectedFamily: "Apple",
		},
		{
			name:           "Adreno",
			renderer:       "Adreno 630",
			vendor:         "Qualcomm",
			expectedType:   model.WebGLRendererTypeHardware,
			expectedFamily: "Qualcomm Adreno",
		},
		{
			name:           "ARM Mali",
			renderer:       "Mali-G76",
			vendor:         "ARM Ltd.",
			expectedType:   model.WebGLRendererTypeHardware,
			expectedFamily: "ARM Mali",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &model.EnvInfo{
				WebGLVendor:   tt.vendor,
				WebGLRenderer: tt.renderer,
			}

			gpuInfo := service.extractGPUInfo(info)

			if gpuInfo.RendererType != tt.expectedType {
				t.Errorf("Expected RendererType=%s, got %s", tt.expectedType, gpuInfo.RendererType)
			}

			if gpuInfo.GPUFamily != tt.expectedFamily {
				t.Errorf("Expected GPUFamily=%s, got %s", tt.expectedFamily, gpuInfo.GPUFamily)
			}
		})
	}
}

func TestIdentifyRendererType(t *testing.T) {
	service := NewWebGLFingerprintService()

	tests := []struct {
		rendererLower string
		expectedType  model.WebGLRendererType
	}{
		{"swiftshader 4.0", model.WebGLRendererTypeSoftware},
		{"llvmpipe", model.WebGLRendererTypeSoftware},
		{"mesa", model.WebGLRendererTypeSoftware},
		{"software", model.WebGLRendererTypeSoftware},
		{"virtualbox", model.WebGLRendererTypeVirtual},
		{"vmware", model.WebGLRendererTypeVirtual},
		{"qemu", model.WebGLRendererTypeVirtual},
		{"kvm", model.WebGLRendererTypeVirtual},
		{"parallels", model.WebGLRendererTypeVirtual},
		{"hyper-v", model.WebGLRendererTypeVirtual},
		{"angle", model.WebGLRendererTypeHardware},
		{"webkit", model.WebGLRendererTypeHardware},
		{"chromium", model.WebGLRendererTypeHardware},
		{"geforce gtx 1080", model.WebGLRendererTypeHardware},
		{"radeon rx 580", model.WebGLRendererTypeHardware},
	}

	for _, tt := range tests {
		t.Run(tt.rendererLower, func(t *testing.T) {
			rendererType := service.identifyRendererType(tt.rendererLower)
			if rendererType != tt.expectedType {
				t.Errorf("Expected %s for '%s', got %s", tt.expectedType, tt.rendererLower, rendererType)
			}
		})
	}
}

func TestExtractGPUFamily(t *testing.T) {
	service := NewWebGLFingerprintService()

	tests := []struct {
		rendererLower string
		expected      string
	}{
		{"nvidia geforce gtx 1080", "NVIDIA"},
		{"amd radeon rx 580", "AMD"},
		{"intel hd graphics", "Intel"},
		{"apple m1", "Apple"},
		{"adreno 630", "Qualcomm Adreno"},
		{"mali-g76", "ARM Mali"},
		{"powervr", "PowerVR"},
		{"unknown renderer", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.rendererLower, func(t *testing.T) {
			family := service.extractGPUFamily(tt.rendererLower)
			if family != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, family)
			}
		})
	}
}

func TestExtractGPUModel(t *testing.T) {
	service := NewWebGLFingerprintService()

	tests := []struct {
		rendererLower string
		expected      string
	}{
		{"geforce gtx 1080 ti", "GTX 1080 Ti"},
		{"radeon rx 580 8gb", "RX 580"},
		{"intel hd graphics 630", "HD 630"},
		{"apple m1 pro", "Apple M1"},
		{"nvidia geforce gtx 1070", "GTX 1070"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.rendererLower, func(t *testing.T) {
			model := service.extractGPUModel(tt.rendererLower)
			if model != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, model)
			}
		})
	}
}

func TestExtractParameterLimits(t *testing.T) {
	service := NewWebGLFingerprintService()

	info := &model.EnvInfo{
		WebGLRenderer: "Test",
		Fingerprint:   "texture_size:16384 renderbuffer:8192 vertex_attribs:16",
	}

	limits := service.extractParameterLimits(info)

	if limits.MaxTextureSize == 0 {
		t.Error("MaxTextureSize should be extracted from fingerprint")
	}

	if limits.MaxRenderbufferSize == 0 {
		t.Error("MaxRenderbufferSize should be extracted from fingerprint")
	}

	info2 := &model.EnvInfo{
		WebGLRenderer: "Test",
		Fingerprint:   "",
	}

	limits2 := service.extractParameterLimits(info2)

	if limits2.MaxTextureSize != 4096 {
		t.Error("Should return default value when no fingerprint data")
	}
}

func TestAnalyzeExtensions(t *testing.T) {
	service := NewWebGLFingerprintService()

	info := &model.EnvInfo{
		WebGLRenderer: "Test Renderer",
		Fingerprint:   "GL_EXT_blend_minmax GL_EXT_color_buffer_float",
	}

	extensions := service.analyzeExtensions(info)

	if len(extensions) == 0 {
		t.Error("Should return extension list")
	}

	supportedCount := 0
	for _, ext := range extensions {
		if ext.IsSupported {
			supportedCount++
		}
	}

	if supportedCount < 2 {
		t.Errorf("Should have at least 2 supported extensions, got %d", supportedCount)
	}
}

func TestAnalyzeRenderingFeatures(t *testing.T) {
	service := NewWebGLFingerprintService()

	info := &model.EnvInfo{
		WebGLRenderer: "Test Renderer",
		Fingerprint:   "",
	}

	features := service.analyzeRenderingFeatures(info)

	if len(features) == 0 {
		t.Error("Should return rendering features")
	}

	info2 := &model.EnvInfo{
		WebGLRenderer: "SwiftShader",
		Fingerprint:   "",
	}

	features2 := service.analyzeRenderingFeatures(info2)

	for _, feature := range features2 {
		if feature.FeatureName == "FloatTextures" && feature.IsSupported {
			t.Error("SwiftShader should not support FloatTextures")
		}
	}
}

func TestComputeFingerprint(t *testing.T) {
	service := NewWebGLFingerprintService()

	gpuInfo := model.WebGLGPUInfo{
		Vendor:             "NVIDIA Corporation",
		Renderer:           "GeForce GTX 1080",
		UnmaskedVendor:     "NVIDIA Corporation",
		UnmaskedRenderer:   "GeForce GTX 1080",
		RendererType:       model.WebGLRendererTypeHardware,
		GPUFamily:         "NVIDIA",
	}

	data := &model.WebGLFingerprintData{
		GPUInfo: gpuInfo,
		ParameterLimits: model.WebGLParameterLimits{
			MaxTextureSize:      16384,
			MaxRenderbufferSize: 16384,
			MaxVertexAttribs:    16,
		},
		Extensions: []model.WebGLSupportedExtension{
			{ExtensionName: "GL_EXT_blend_minmax", IsSupported: true},
			{ExtensionName: "GL_EXT_color_buffer_float", IsSupported: true},
		},
	}

	fingerprint := service.computeFingerprint(gpuInfo, data)

	if fingerprint == "" {
		t.Error("Fingerprint should not be empty")
	}

	if len(fingerprint) != 64 {
		t.Errorf("Expected 64 character SHA256 hash, got %d characters", len(fingerprint))
	}

	fingerprint2 := service.computeFingerprint(gpuInfo, data)
	if fingerprint != fingerprint2 {
		t.Error("Fingerprint should be deterministic")
	}

	gpuInfo2 := gpuInfo
	gpuInfo2.Renderer = "AMD Radeon RX 580"
	fingerprint3 := service.computeFingerprint(gpuInfo2, data)
	if fingerprint == fingerprint3 {
		t.Error("Different GPU should produce different fingerprint")
	}
}

func TestAnalyzeWebGLRisk(t *testing.T) {
	service := NewWebGLFingerprintService()

	tests := []struct {
		name           string
		gpuInfo        model.WebGLGPUInfo
		limits         model.WebGLParameterLimits
		extCount       int
		expectedRisk   string
		minExpectedScore float64
	}{
		{
			name: "Hardware GPU Low Risk",
			gpuInfo: model.WebGLGPUInfo{
				Vendor:       "NVIDIA Corporation",
				Renderer:     "GeForce GTX 1080",
				RendererType: model.WebGLRendererTypeHardware,
				GPUFamily:   "NVIDIA",
			},
			limits: model.WebGLParameterLimits{
				MaxTextureSize:   16384,
				MaxVertexAttribs: 16,
			},
			extCount:         15,
			expectedRisk:     "low",
			minExpectedScore: 0,
		},
		{
			name: "Software Renderer Medium Risk",
			gpuInfo: model.WebGLGPUInfo{
				Vendor:       "Google Inc.",
				Renderer:     "SwiftShader 4.0",
				RendererType: model.WebGLRendererTypeSoftware,
				GPUFamily:   "Unknown",
			},
			limits: model.WebGLParameterLimits{
				MaxTextureSize:   4096,
				MaxVertexAttribs: 8,
			},
			extCount:         3,
			expectedRisk:     "medium",
			minExpectedScore: 40,
		},
		{
			name: "Virtual GPU High Risk",
			gpuInfo: model.WebGLGPUInfo{
				Vendor:       "VMware, Inc.",
				Renderer:     "VirtualBox",
				RendererType: model.WebGLRendererTypeVirtual,
				GPUFamily:   "Unknown",
			},
			limits: model.WebGLParameterLimits{
				MaxTextureSize:   4096,
				MaxVertexAttribs: 8,
			},
			extCount:         2,
			expectedRisk:     "high",
			minExpectedScore: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &model.EnvInfo{}

			data := &model.WebGLFingerprintData{
				GPUInfo: tt.gpuInfo,
				ParameterLimits: tt.limits,
				Extensions: make([]model.WebGLSupportedExtension, tt.extCount),
			}

			for i := 0; i < tt.extCount; i++ {
				data.Extensions[i] = model.WebGLSupportedExtension{
					ExtensionName: fmt.Sprintf("EXT_%d", i),
					IsSupported:    true,
				}
			}

			riskLevel, riskScore, confidence := service.analyzeWebGLRisk(data, info)

			if riskLevel != tt.expectedRisk {
				t.Errorf("Expected risk level %s, got %s", tt.expectedRisk, riskLevel)
			}

			if riskScore < tt.minExpectedScore {
				t.Errorf("Expected risk score >= %f, got %f", tt.minExpectedScore, riskScore)
			}
		})
	}
}

func TestDetectAnomaliesWebGL(t *testing.T) {
	service := NewWebGLFingerprintService()

	tests := []struct {
		name             string
		gpuInfo          model.WebGLGPUInfo
		extensions       int
		maxTextureSize   int
		expectedAnomalies int
	}{
		{
			name: "No Data",
			gpuInfo: model.WebGLGPUInfo{
				Vendor:   "",
				Renderer: "",
			},
			extensions:       0,
			maxTextureSize:   0,
			expectedAnomalies: 1,
		},
		{
			name: "Missing Vendor",
			gpuInfo: model.WebGLGPUInfo{
				Vendor:   "",
				Renderer: "Test Renderer",
			},
			extensions:       5,
			maxTextureSize:   4096,
			expectedAnomalies: 1,
		},
		{
			name: "No Extensions",
			gpuInfo: model.WebGLGPUInfo{
				Vendor:   "Test Vendor",
				Renderer: "Test Renderer",
			},
			extensions:       0,
			maxTextureSize:   4096,
			expectedAnomalies: 1,
		},
		{
			name: "Few Extensions",
			gpuInfo: model.WebGLGPUInfo{
				Vendor:   "Test Vendor",
				Renderer: "Test Renderer",
			},
			extensions:       2,
			maxTextureSize:   4096,
			expectedAnomalies: 1,
		},
		{
			name: "Limited Texture Size",
			gpuInfo: model.WebGLGPUInfo{
				Vendor:   "Test Vendor",
				Renderer: "Test Renderer",
			},
			extensions:       10,
			maxTextureSize:   512,
			expectedAnomalies: 1,
		},
		{
			name: "Suspicious Pattern",
			gpuInfo: model.WebGLGPUInfo{
				Vendor:   "Fake Vendor",
				Renderer: "Mock Renderer",
			},
			extensions:       5,
			maxTextureSize:   4096,
			expectedAnomalies: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &model.EnvInfo{}

			extensions := make([]model.WebGLSupportedExtension, tt.extensions)
			for i := 0; i < tt.extensions; i++ {
				extensions[i] = model.WebGLSupportedExtension{
					ExtensionName: fmt.Sprintf("EXT_%d", i),
					IsSupported:    true,
				}
			}

			data := &model.WebGLFingerprintData{
				GPUInfo: tt.gpuInfo,
				ParameterLimits: model.WebGLParameterLimits{
					MaxTextureSize: tt.maxTextureSize,
				},
				Extensions: extensions,
			}

			anomalies := service.detectAnomalies(data, info)

			if len(anomalies) < tt.expectedAnomalies {
				t.Errorf("Expected at least %d anomalies, got %d", tt.expectedAnomalies, len(anomalies))
			}
		})
	}
}

func TestAnalyzeStabilityWebGL(t *testing.T) {
	service := NewWebGLFingerprintService()

	result, err := service.AnalyzeStability("test-fp-1", "session-1")

	if err != nil {
		t.Fatalf("AnalyzeStability returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.DeviceID != "test-fp-1" {
		t.Errorf("Expected DeviceID 'test-fp-1', got '%s'", result.DeviceID)
	}

	if result.SessionCount != 1 {
		t.Errorf("Expected SessionCount 1, got %d", result.SessionCount)
	}

	result2, _ := service.AnalyzeStability("test-fp-1", "session-1")
	if result2.SessionCount != 2 {
		t.Errorf("Expected SessionCount 2 on second call, got %d", result2.SessionCount)
	}

	result3, _ := service.AnalyzeStability("test-fp-1", "session-2")
	if result3.UniqueFingerprints != 2 {
		t.Errorf("Expected UniqueFingerprints 2, got %d", result3.UniqueFingerprints)
	}
}

func TestCalculateStabilityScore(t *testing.T) {
	service := NewWebGLFingerprintService()

	tests := []struct {
		name            string
		hitCount        int
		variations      int
		rendererChanges int
		minScore        float64
		maxScore        float64
	}{
		{
			name:            "First Hit",
			hitCount:        1,
			variations:      0,
			rendererChanges: 0,
			minScore:        0.9,
			maxScore:        1.0,
		},
		{
			name:            "Multiple Hits with Variations",
			hitCount:        10,
			variations:      3,
			rendererChanges: 1,
			minScore:        0.3,
			maxScore:        0.6,
		},
		{
			name:            "Many Variations",
			hitCount:        20,
			variations:      10,
			rendererChanges: 5,
			minScore:        0.0,
			maxScore:        0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stored := &model.WebGLFingerprintStability{
				HitCount:         tt.hitCount,
				Variations:       make([]string, tt.variations),
				RendererChanges:  tt.rendererChanges,
				FirstSeen:        time.Now().Add(-24 * time.Hour),
			}

			for i := 0; i < tt.variations; i++ {
				stored.Variations[i] = fmt.Sprintf("variation-%d", i)
			}

			score := service.calculateStabilityScore(stored)

			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("Expected score between %f and %f, got %f", tt.minScore, tt.maxScore, score)
			}
		})
	}
}

func TestCompareFingerprintsWebGL(t *testing.T) {
	service := NewWebGLFingerprintService()

	data1 := &model.WebGLFingerprintData{
		GPUInfo: model.WebGLGPUInfo{
			Vendor:   "NVIDIA Corporation",
			Renderer: "GeForce GTX 1080",
		},
		Extensions: []model.WebGLSupportedExtension{
			{ExtensionName: "EXT_1", IsSupported: true},
			{ExtensionName: "EXT_2", IsSupported: true},
		},
	}

	data2 := &model.WebGLFingerprintData{
		GPUInfo: model.WebGLGPUInfo{
			Vendor:   "NVIDIA Corporation",
			Renderer: "GeForce GTX 1080",
		},
		Extensions: []model.WebGLSupportedExtension{
			{ExtensionName: "EXT_1", IsSupported: true},
			{ExtensionName: "EXT_2", IsSupported: true},
		},
	}

	data3 := &model.WebGLFingerprintData{
		GPUInfo: model.WebGLGPUInfo{
			Vendor:   "AMD",
			Renderer: "Radeon RX 580",
		},
		Extensions: []model.WebGLSupportedExtension{
			{ExtensionName: "EXT_1", IsSupported: true},
			{ExtensionName: "EXT_3", IsSupported: false},
		},
	}

	hash1 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	hash2 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	hash3 := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	result1 := service.CompareFingerprints(hash1, hash2, data1, data2)
	if !result1.IsSameDevice {
		t.Error("Same device should be identified as same")
	}
	if !result1.VendorMatch {
		t.Error("Same vendor should match")
	}
	if !result1.RendererMatch {
		t.Error("Same renderer should match")
	}

	result2 := service.CompareFingerprints(hash1, hash3, data1, data3)
	if result2.VendorMatch {
		t.Error("Different vendors should not match")
	}
	if result2.RendererMatch {
		t.Error("Different renderers should not match")
	}
}

func TestDetectSpoofingWebGL(t *testing.T) {
	service := NewWebGLFingerprintService()

	originalData := &model.WebGLFingerprintData{
		GPUInfo: model.WebGLGPUInfo{
			Vendor:       "NVIDIA Corporation",
			Renderer:     "GeForce GTX 1080",
			RendererType: model.WebGLRendererTypeHardware,
		},
		Extensions: []model.WebGLSupportedExtension{
			{ExtensionName: "EXT_1", IsSupported: true},
			{ExtensionName: "EXT_2", IsSupported: true},
		},
	}

	receivedData1 := &model.WebGLFingerprintData{
		GPUInfo: model.WebGLGPUInfo{
			Vendor:       "NVIDIA Corporation",
			Renderer:     "GeForce GTX 1080",
			RendererType: model.WebGLRendererTypeHardware,
		},
		Extensions: []model.WebGLSupportedExtension{
			{ExtensionName: "EXT_1", IsSupported: true},
			{ExtensionName: "EXT_2", IsSupported: true},
		},
	}

	result1 := service.DetectSpoofing("hash1", "hash1", originalData, receivedData1)
	if result1.IsSpoofed {
		t.Error("Original data should not be flagged as spoofed")
	}
	if result1.ConsistencyScore < 90 {
		t.Errorf("Expected high consistency score, got %f", result1.ConsistencyScore)
	}

	receivedData2 := &model.WebGLFingerprintData{
		GPUInfo: model.WebGLGPUInfo{
			Vendor:       "AMD",
			Renderer:     "Radeon RX 580",
			RendererType: model.WebGLRendererTypeHardware,
		},
		Extensions: []model.WebGLSupportedExtension{
			{ExtensionName: "EXT_1", IsSupported: true},
		},
	}

	result2 := service.DetectSpoofing("hash1", "hash2", originalData, receivedData2)
	if !result2.IsSpoofed {
		t.Error("Completely different data should be flagged as spoofed")
	}

	receivedData3 := &model.WebGLFingerprintData{
		GPUInfo: model.WebGLGPUInfo{
			Vendor:       "Google Inc.",
			Renderer:     "SwiftShader 4.0",
			RendererType: model.WebGLRendererTypeSoftware,
		},
		Extensions: []model.WebGLSupportedExtension{
			{ExtensionName: "EXT_1", IsSupported: false},
		},
	}

	result3 := service.DetectSpoofing("hash1", "hash3", originalData, receivedData3)
	if len(result3.SuspiciousPatterns) == 0 {
		t.Error("Software renderer should be detected as suspicious")
	}
}

func TestGenerateRenderAnalysisWebGL(t *testing.T) {
	service := NewWebGLFingerprintService()

	data := &model.WebGLFingerprintData{
		GPUInfo: model.WebGLGPUInfo{
			Vendor:       "NVIDIA Corporation",
			Renderer:     "GeForce GTX 1080",
			RendererType: model.WebGLRendererTypeHardware,
			GPUFamily:   "NVIDIA",
			IsDiscrete:   true,
		},
		ParameterLimits: model.WebGLParameterLimits{
			MaxTextureSize: 16384,
		},
		Extensions: []model.WebGLSupportedExtension{
			{ExtensionName: "EXT_1", IsSupported: true},
			{ExtensionName: "EXT_2", IsSupported: true},
			{ExtensionName: "EXT_3", IsSupported: true},
		},
	}

	analysis := service.GenerateRenderAnalysis("test-hash", data)

	if analysis.RenderHash != "test-hash" {
		t.Errorf("Expected RenderHash 'test-hash', got '%s'", analysis.RenderHash)
	}

	if len(analysis.Features) == 0 {
		t.Error("Should have features")
	}

	if analysis.ComplexityScore == 0 {
		t.Error("ComplexityScore should not be zero")
	}

	if len(analysis.RenderingPatterns) == 0 {
		t.Error("Should have rendering patterns")
	}

	data2 := &model.WebGLFingerprintData{
		GPUInfo: model.WebGLGPUInfo{
			RendererType: model.WebGLRendererTypeSoftware,
		},
		Extensions: make([]model.WebGLSupportedExtension, 2),
	}

	analysis2 := service.GenerateRenderAnalysis("test-hash-2", data2)

	if analysis2.UniquenessScore > analysis.UniquenessScore {
		t.Error("Software renderer should have lower uniqueness score")
	}
}

func TestExportFingerprintDataWebGL(t *testing.T) {
	service := NewWebGLFingerprintService()

	service.AnalyzeStability("test-fp-export", "session-1")

	jsonData, err := service.ExportFingerprintData("test-fp-export")

	if err != nil {
		t.Fatalf("ExportFingerprintData returned error: %v", err)
	}

	if jsonData == "" {
		t.Error("Exported data should not be empty")
	}

	_, err = service.ExportFingerprintData("non-existent")
	if err == nil {
		t.Error("Should return error for non-existent fingerprint")
	}
}

func TestImportFingerprintDataWebGL(t *testing.T) {
	service := NewWebGLFingerprintService()

	jsonData := `{
		"fingerprint_id": "imported-fp",
		"session_id": "imported-session",
		"hit_count": 5,
		"stability_score": 0.85,
		"is_stable": true
	}`

	err := service.ImportFingerprintData("imported-fp", jsonData)

	if err != nil {
		t.Fatalf("ImportFingerprintData returned error: %v", err)
	}

	result, _ := service.AnalyzeStability("imported-fp", "session-2")

	if result.SessionCount != 6 {
		t.Errorf("Expected SessionCount 6, got %d", result.SessionCount)
	}
}

func TestClearExpiredDataWebGL(t *testing.T) {
	service := NewWebGLFingerprintService()

	service.AnalyzeStability("recent-fp", "session-1")

	removed := service.ClearExpiredData(24 * time.Hour)

	if removed != 0 {
		t.Errorf("Recent data should not be removed, got %d", removed)
	}

	removed2 := service.ClearExpiredData(0)
	if removed2 == 0 {
		t.Error("Should remove data with zero expiration")
	}
}

func TestGetStatisticsWebGL(t *testing.T) {
	service := NewWebGLFingerprintService()

	service.AnalyzeStability("fp-stat-1", "session-1")
	service.AnalyzeStability("fp-stat-2", "session-2")

	stats := service.GetStatistics()

	if stats["total_fingerprints"].(int) != 2 {
		t.Errorf("Expected 2 total fingerprints, got %v", stats["total_fingerprints"])
	}

	if stats["stable_count"] == nil {
		t.Error("stable_count should be present")
	}
}

func TestGetConfigWebGL(t *testing.T) {
	service := NewWebGLFingerprintService()

	config := service.GetConfig()

	if config == nil {
		t.Fatal("GetConfig should not return nil")
	}

	if config.StabilityThreshold == 0 {
		t.Error("StabilityThreshold should be set")
	}
}

func TestUpdateConfigWebGL(t *testing.T) {
	service := NewWebGLFingerprintService()

	originalThreshold := service.config.StabilityThreshold

	newConfig := &model.WebGLEnhancementConfig{
		StabilityThreshold: 0.95,
		EnableAnomalyDetection: false,
	}

	service.UpdateConfig(newConfig)

	if service.config.StabilityThreshold == originalThreshold {
		t.Error("StabilityThreshold should be updated")
	}

	if service.config.EnableAnomalyDetection {
		t.Error("EnableAnomalyDetection should be updated to false")
	}
}

func TestContainsStringWebGL(t *testing.T) {
	tests := []struct {
		slice   []string
		str     string
		matches bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{[]string{"test"}, "test", true},
	}

	for _, tt := range tests {
		result := containsString(tt.slice, tt.str)
		if result != tt.matches {
			t.Errorf("containsString(%v, %s) = %v, expected %v", tt.slice, tt.str, result, tt.matches)
		}
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		str     string
		list    []string
		matches bool
	}{
		{"hello world", []string{"world", "test"}, true},
		{"hello world", []string{"foo", "bar"}, false},
		{"SWIFTSHADER test", []string{"swiftshader", "mesa"}, true},
		{"", []string{"test"}, false},
	}

	for _, tt := range tests {
		result := containsAny(tt.str, tt.list)
		if result != tt.matches {
			t.Errorf("containsAny(%s, %v) = %v, expected %v", tt.str, tt.list, result, tt.matches)
		}
	}
}

func TestSimulatedWebGLRenderer(t *testing.T) {
	tests := []struct {
		rendererType string
		expectedType model.WebGLRendererType
	}{
		{"hardware", model.WebGLRendererTypeHardware},
		{"software", model.WebGLRendererTypeSoftware},
		{"virtual", model.WebGLRendererTypeVirtual},
	}

	for _, tt := range tests {
		t.Run(tt.rendererType, func(t *testing.T) {
			renderer := NewSimulatedWebGLRenderer(tt.rendererType)

			if renderer == nil {
				t.Fatal("NewSimulatedWebGLRenderer returned nil")
			}

			data := renderer.GetFingerprint()

			if data == nil {
				t.Fatal("GetFingerprint returned nil")
			}

			if data.GPUInfo.RendererType != tt.expectedType {
				t.Errorf("Expected renderer type %s, got %s", tt.expectedType, data.GPUInfo.RendererType)
			}
		})
	}
}

func TestGenerateSimulatedWebGLFingerprint(t *testing.T) {
	data, fingerprint := GenerateSimulatedWebGLFingerprint("test-ua", "hardware")

	if data == nil {
		t.Fatal("GenerateSimulatedWebGLFingerprint returned nil data")
	}

	if fingerprint == "" {
		t.Error("Fingerprint should not be empty")
	}

	data2, fingerprint2 := GenerateSimulatedWebGLFingerprint("test-ua", "software")

	if fingerprint == fingerprint2 {
		t.Error("Different renderer types should produce different fingerprints")
	}
}

func TestSimulateWebGLContextLoss(t *testing.T) {
	info := SimulateWebGLContextLoss()

	if info == nil {
		t.Fatal("SimulateWebGLContextLoss returned nil")
	}

	if info.WebGLVendor != "" || info.WebGLRenderer != "" {
		t.Error("Context loss simulation should have empty WebGL data")
	}
}

func TestSimulateWebGLEmptyExtensions(t *testing.T) {
	info := SimulateWebGLEmptyExtensions()

	if info == nil {
		t.Fatal("SimulateWebGLEmptyExtensions returned nil")
	}

	if info.WebGLVendor == "" || info.WebGLRenderer == "" {
		t.Error("Should have WebGL vendor and renderer")
	}
}

func TestSimulateWebGLSoftwareRenderer(t *testing.T) {
	info := SimulateWebGLSoftwareRenderer()

	if info == nil {
		t.Fatal("SimulateWebGLSoftwareRenderer returned nil")
	}

	if info.WebGLRenderer != "SwiftShader 4.0.0.1" {
		t.Errorf("Expected SwiftShader renderer, got %s", info.WebGLRenderer)
	}
}

func TestSimulateWebGLVirtualGPU(t *testing.T) {
	info := SimulateWebGLVirtualGPU()

	if info == nil {
		t.Fatal("SimulateWebGLVirtualGPU returned nil")
	}

	if info.WebGLRenderer == "" {
		t.Error("Should have renderer info")
	}
}

func TestIsTrustedVendor(t *testing.T) {
	service := NewWebGLFingerprintService()

	tests := []struct {
		vendor   string
		expected bool
	}{
		{"NVIDIA Corporation", true},
		{"Intel Inc.", true},
		{"Google Inc.", true},
		{"Apple Inc.", true},
		{"Unknown Vendor", false},
		{"Fake Company", false},
	}

	for _, tt := range tests {
		result := service.isTrustedVendor(tt.vendor)
		if result != tt.expected {
			t.Errorf("isTrustedVendor(%s) = %v, expected %v", tt.vendor, result, tt.expected)
		}
	}
}
