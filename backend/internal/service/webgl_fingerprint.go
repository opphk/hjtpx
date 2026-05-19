package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type WebGLFingerprintService struct {
	config        *model.WebGLEnhancementConfig
	mu            sync.RWMutex
	fingerprintDB map[string]*model.WebGLFingerprintStability
	stabilityMu   sync.RWMutex
}

func NewWebGLFingerprintService() *WebGLFingerprintService {
	return &WebGLFingerprintService{
		config: &model.WebGLEnhancementConfig{
			EnableParameterExtraction:   true,
			EnableExtensionAnalysis:     true,
			EnableRendererIdentification: true,
			EnableStabilityTrack:       true,
			EnableAnomalyDetection:     true,
			EnableAntiSpoof:           true,
			StabilityThreshold:         0.8,
			AnomalyThreshold:           0.3,
			SimilarityThreshold:        0.85,
			TrustedVendors: []string{
				"NVIDIA Corporation",
				"ATI Technologies Inc.",
				"Advanced Micro Devices, Inc.",
				"Intel Inc.",
				"Intel(R) Corporation",
				"Google Inc.",
				"Microsoft Corporation",
				"Apple Inc.",
				"ARM Ltd.",
				"Qualcomm",
			},
			SoftwareRenderers: []string{
				"SwiftShader",
				"llvmpipe",
				"Mesa",
				"Software",
				"Virtual",
			},
			VirtualGPUs: []string{
				"VirtualBox",
				"VMware",
				"QEMU",
				"KVM",
				"Parallels",
				"Hyper-V",
			},
		},
		fingerprintDB: make(map[string]*model.WebGLFingerprintStability),
	}
}

func NewWebGLFingerprintServiceWithConfig(config *model.WebGLEnhancementConfig) *WebGLFingerprintService {
	if config == nil {
		return NewWebGLFingerprintService()
	}

	if len(config.TrustedVendors) == 0 {
		config.TrustedVendors = []string{
			"NVIDIA Corporation",
			"Intel Inc.",
			"Google Inc.",
			"Apple Inc.",
		}
	}

	if len(config.SoftwareRenderers) == 0 {
		config.SoftwareRenderers = []string{
			"SwiftShader",
			"llvmpipe",
			"Mesa",
		}
	}

	if len(config.VirtualGPUs) == 0 {
		config.VirtualGPUs = []string{
			"VirtualBox",
			"VMware",
			"QEMU",
		}
	}

	if config.StabilityThreshold == 0 {
		config.StabilityThreshold = 0.8
	}
	if config.AnomalyThreshold == 0 {
		config.AnomalyThreshold = 0.3
	}
	if config.SimilarityThreshold == 0 {
		config.SimilarityThreshold = 0.85
	}

	return &WebGLFingerprintService{
		config:        config,
		fingerprintDB: make(map[string]*model.WebGLFingerprintStability),
	}
}

func (s *WebGLFingerprintService) GenerateEnhancedFingerprint(info *model.EnvInfo) *model.WebGLFingerprintResult {
	result := &model.WebGLFingerprintResult{
		Success:          true,
		EnhancedData:     &model.WebGLFingerprintData{},
		RiskLevel:        "low",
		RiskScore:        0.0,
		Confidence:       0.0,
		Warnings:         make([]string, 0),
		Metadata:         make(map[string]interface{}),
	}

	if info.WebGLRenderer == "" && info.WebGLVendor == "" {
		result.Success = false
		result.Error = "WebGL fingerprint data is empty"
		result.RiskLevel = "high"
		result.RiskScore = 50.0
		return result
	}

	gpuInfo := s.extractGPUInfo(info)
	result.EnhancedData.GPUInfo = gpuInfo

	if s.config.EnableParameterExtraction {
		paramLimits := s.extractParameterLimits(info)
		result.EnhancedData.ParameterLimits = paramLimits
	}

	if s.config.EnableExtensionAnalysis {
		extensions := s.analyzeExtensions(info)
		result.EnhancedData.Extensions = extensions
	}

	renderingFeatures := s.analyzeRenderingFeatures(info)
	result.EnhancedData.RenderingFeatures = renderingFeatures

	parameters := s.extractParameters(info)
	result.EnhancedData.Parameters = parameters

	fingerprint := s.computeFingerprint(gpuInfo, result.EnhancedData)
	result.Fingerprint = fingerprint
	result.Hash = fingerprint

	riskLevel, riskScore, confidence := s.analyzeWebGLRisk(result.EnhancedData, info)
	result.RiskLevel = riskLevel
	result.RiskScore = riskScore
	result.Confidence = confidence

	if s.config.EnableAnomalyDetection {
		anomalies := s.detectAnomalies(result.EnhancedData, info)
		result.Anomalies = anomalies
		if len(anomalies) > 0 {
			result.RiskLevel = "medium"
			result.RiskScore += float64(len(anomalies)) * 10.0
		}
	}

	if result.RiskScore > 100 {
		result.RiskScore = 100
	}

	return result
}

func (s *WebGLFingerprintService) extractGPUInfo(info *model.EnvInfo) model.WebGLGPUInfo {
	gpuInfo := model.WebGLGPUInfo{
		Vendor:         info.WebGLVendor,
		Renderer:       info.WebGLRenderer,
		UnmaskedVendor: info.WebGLVendor,
		UnmaskedRenderer: info.WebGLRenderer,
	}

	rendererLower := strings.ToLower(info.WebGLRenderer)

	gpuInfo.RendererType = s.identifyRendererType(rendererLower)

	if containsAny(rendererLower, s.config.TrustedVendors) {
		gpuInfo.RendererType = model.WebGLRendererTypeHardware
	} else if containsAny(rendererLower, s.config.SoftwareRenderers) {
		gpuInfo.RendererType = model.WebGLRendererTypeSoftware
	} else if containsAny(rendererLower, s.config.VirtualGPUs) {
		gpuInfo.RendererType = model.WebGLRendererTypeVirtual
	}

	gpuInfo.GPUFamily = s.extractGPUFamily(rendererLower)
	gpuInfo.GPUModel = s.extractGPUModel(rendererLower)
	gpuInfo.IsIntegrated = strings.Contains(rendererLower, "intel") && !strings.Contains(rendererLower, "discrete")
	gpuInfo.IsDiscrete = strings.Contains(rendererLower, "discrete") || strings.Contains(rendererLower, "geforce") || strings.Contains(rendererLower, "radeon")

	if strings.Contains(rendererLower, "nvidia") || strings.Contains(rendererLower, "geforce") {
		gpuInfo.UnmaskedVendor = "NVIDIA Corporation"
	} else if strings.Contains(rendererLower, "amd") || strings.Contains(rendererLower, "radeon") || strings.Contains(rendererLower, "ati") {
		gpuInfo.UnmaskedVendor = "Advanced Micro Devices, Inc."
	} else if strings.Contains(rendererLower, "intel") {
		gpuInfo.UnmaskedVendor = "Intel Inc."
	} else if strings.Contains(rendererLower, "apple") {
		gpuInfo.UnmaskedVendor = "Apple Inc."
	}

	return gpuInfo
}

func (s *WebGLFingerprintService) identifyRendererType(rendererLower string) model.WebGLRendererType {
	if strings.Contains(rendererLower, "swiftshader") ||
		strings.Contains(rendererLower, "llvmpipe") ||
		strings.Contains(rendererLower, "mesa") ||
		strings.Contains(rendererLower, "software") {
		return model.WebGLRendererTypeSoftware
	}

	if strings.Contains(rendererLower, "virtualbox") ||
		strings.Contains(rendererLower, "vmware") ||
		strings.Contains(rendererLower, "qemu") ||
		strings.Contains(rendererLower, "kvm") ||
		strings.Contains(rendererLower, "parallels") ||
		strings.Contains(rendererLower, "hyper") {
		return model.WebGLRendererTypeVirtual
	}

	if strings.Contains(rendererLower, "angle") ||
		strings.Contains(rendererLower, "webkit") ||
		strings.Contains(rendererLower, "chromium") {
		return model.WebGLRendererTypeHardware
	}

	return model.WebGLRendererTypeHardware
}

func (s *WebGLFingerprintService) extractGPUFamily(rendererLower string) string {
	if strings.Contains(rendererLower, "nvidia") || strings.Contains(rendererLower, "geforce") {
		return "NVIDIA"
	}
	if strings.Contains(rendererLower, "amd") || strings.Contains(rendererLower, "radeon") || strings.Contains(rendererLower, "ati") {
		return "AMD"
	}
	if strings.Contains(rendererLower, "intel") {
		return "Intel"
	}
	if strings.Contains(rendererLower, "apple") {
		return "Apple"
	}
	if strings.Contains(rendererLower, "adreno") {
		return "Qualcomm Adreno"
	}
	if strings.Contains(rendererLower, "mali") {
		return "ARM Mali"
	}
	if strings.Contains(rendererLower, "powervr") {
		return "PowerVR"
	}
	return "Unknown"
}

func (s *WebGLFingerprintService) extractGPUModel(rendererLower string) string {
	nvidiaRegex := regexp.MustCompile(`(GTX?\s*\d+[A-Za-z]*)`)
	if match := nvidiaRegex.FindStringSubmatch(rendererLower); len(match) > 1 {
		return match[1]
	}

	amdRegex := regexp.MustCompile(`(RX\s*\d+[A-Za-z]*)`)
	if match := amdRegex.FindStringSubmatch(rendererLower); len(match) > 1 {
		return match[1]
	}

	intelRegex := regexp.MustCompile(`(HD\s*\d+[A-Za-z]*)`)
	if match := intelRegex.FindStringSubmatch(rendererLower); len(match) > 1 {
		return match[1]
	}

	appleRegex := regexp.MustCompile(`(Apple\s*M\d+)`)
	if match := appleRegex.FindStringSubmatch(rendererLower); len(match) > 1 {
		return match[1]
	}

	return ""
}

func (s *WebGLFingerprintService) extractParameterLimits(info *model.EnvInfo) model.WebGLParameterLimits {
	limits := model.WebGLParameterLimits{
		MaxTextureSize:              4096,
		MaxTextureSize3D:            256,
		MaxRenderbufferSize:         4096,
		MaxVertexAttribs:            16,
		MaxVertexUniformVectors:     1024,
		MaxFragmentUniformVectors:   1024,
		MaxVaryingVectors:           15,
		MaxViewportDims:             []int{4096, 4096},
		MaxCubeMapTextureSize:       4096,
		MaxTextureImageUnits:        16,
		MaxVertexTextureImageUnits:   16,
		MaxCombinedTextureImageUnits: 32,
		MaxSamples:                  4,
		MaxColorAttachments:         8,
		MaxDrawBuffers:              8,
	}

	metadata := info.Fingerprint
	if metadata == "" {
		return limits
	}

	if strings.Contains(metadata, "texture_size:") {
		regex := regexp.MustCompile(`texture_size:(\d+)`)
		if match := regex.FindStringSubmatch(metadata); len(match) > 1 {
			fmt.Sscanf(match[1], "%d", &limits.MaxTextureSize)
		}
	}

	if strings.Contains(metadata, "renderbuffer:") {
		regex := regexp.MustCompile(`renderbuffer:(\d+)`)
		if match := regex.FindStringSubmatch(metadata); len(match) > 1 {
			fmt.Sscanf(match[1], "%d", &limits.MaxRenderbufferSize)
		}
	}

	if strings.Contains(metadata, "vertex_attribs:") {
		regex := regexp.MustCompile(`vertex_attribs:(\d+)`)
		if match := regex.FindStringSubmatch(metadata); len(match) > 1 {
			fmt.Sscanf(match[1], "%d", &limits.MaxVertexAttribs)
		}
	}

	return limits
}

func (s *WebGLFingerprintService) analyzeExtensions(info *model.EnvInfo) []model.WebGLSupportedExtension {
	extensions := make([]model.WebGLSupportedExtension, 0)

	standardExtensions := []string{
		"GL_EXT_blend_minmax",
		"GL_EXT_color_buffer_float",
		"GL_EXT_frag_depth",
		"GL_EXT_shader_texture_lod",
		"GL_EXT_sRGB",
		"GL_OES_standard_derivatives",
		"GL_OES_texture_float",
		"GL_OES_texture_float_linear",
		"GL_OES_texture_half_float",
		"GL_OES_texture_half_float_linear",
		"GL_OES_vertex_array_object",
		"GL_ANGLE_instanced_arrays",
		"GL_WEBGL_compressed_texture_s3tc",
		"GL_WEBGL_depth_texture",
		"GL_WEBGL_lose_context",
		"WEBGL_debug_renderer_info",
		"WEBGL_debug_shaders",
		"WEBGL_draw_buffers",
		"WEBGL_lose_context",
	}

	extensionCategories := map[string]string{
		"blend":            "Blending",
		"color_buffer":     "Color Buffer",
		"depth":            "Depth",
		"texture":          "Texture",
		"float":            "Floating Point",
		"half_float":        "Half Float",
		"vertex_array":     "Vertex Array",
		"instanced":        "Instancing",
		"compressed":       "Compression",
		"debug":            "Debug",
		"draw_buffers":     "Multiple Buffers",
		"derivatives":       "Derivatives",
		"los":              "Context Loss",
	}

	metadata := info.Fingerprint
	if metadata == "" {
		for _, ext := range standardExtensions[:5] {
			extensions = append(extensions, model.WebGLSupportedExtension{
				ExtensionName: ext,
				IsSupported:   true,
				Category:      "Standard",
			})
		}
		return extensions
	}

	for _, ext := range standardExtensions {
		isSupported := strings.Contains(strings.ToLower(metadata), strings.ToLower(ext))
		category := "Standard"

		extLower := strings.ToLower(ext)
		for pattern, cat := range extensionCategories {
			if strings.Contains(extLower, pattern) {
				category = cat
				break
			}
		}

		extensions = append(extensions, model.WebGLSupportedExtension{
			ExtensionName: ext,
			IsSupported:   isSupported,
			Category:      category,
		})
	}

	return extensions
}

func (s *WebGLFingerprintService) analyzeRenderingFeatures(info *model.EnvInfo) []model.WebGLRenderingFeature {
	features := []model.WebGLRenderingFeature{
		{FeatureName: "WebGL2", IsSupported: true, SupportLevel: "full", PerformanceHint: "optimal"},
		{FeatureName: "FloatTextures", IsSupported: true, SupportLevel: "full"},
		{FeatureName: "HalfFloatTextures", IsSupported: true, SupportLevel: "full"},
		{FeatureName: "DepthTextures", IsSupported: true, SupportLevel: "full"},
		{FeatureName: "InstancedDrawing", IsSupported: true, SupportLevel: "full"},
		{FeatureName: "MultipleBuffers", IsSupported: true, SupportLevel: "full"},
		{FeatureName: "3DTextures", IsSupported: true, SupportLevel: "full"},
		{FeatureName: "CompressedTextures", IsSupported: true, SupportLevel: "full"},
		{FeatureName: "Derivatives", IsSupported: true, SupportLevel: "full"},
		{FeatureName: "StandardDerivatives", IsSupported: true, SupportLevel: "full"},
	}

	metadata := info.Fingerprint
	if metadata == "" {
		return features
	}

	rendererLower := strings.ToLower(info.WebGLRenderer)

	if strings.Contains(rendererLower, "swiftshader") || strings.Contains(rendererLower, "llvmpipe") {
		for i := range features {
			if features[i].FeatureName == "FloatTextures" {
				features[i].IsSupported = false
				features[i].SupportLevel = "none"
				features[i].PerformanceHint = "software_limitation"
			}
		}
	}

	return features
}

func (s *WebGLFingerprintService) extractParameters(info *model.EnvInfo) []model.WebGLParameterInfo {
	parameters := []model.WebGLParameterInfo{
		{ParameterName: "ALIASED_LINE_WIDTH_RANGE", ParameterValue: []int{1, 1}, DataType: "Float32Array", IsConsistent: true},
		{ParameterName: "ALIASED_POINT_SIZE_RANGE", ParameterValue: []int{1, 1}, DataType: "Float32Array", IsConsistent: true},
		{ParameterName: "MAX_TEXTURE_SIZE", ParameterValue: 4096, DataType: "GLint", IsConsistent: true},
		{ParameterName: "MAX_VIEWPORT_DIMS", ParameterValue: []int{4096, 4096}, DataType: "Int32Array", IsConsistent: true},
		{ParameterName: "MAX_CUBE_MAP_TEXTURE_SIZE", ParameterValue: 4096, DataType: "GLint", IsConsistent: true},
		{ParameterName: "RENDERER", ParameterValue: info.WebGLRenderer, DataType: "DOMString", IsConsistent: true},
		{ParameterName: "VENDOR", ParameterValue: info.WebGLVendor, DataType: "DOMString", IsConsistent: true},
		{ParameterName: "VERSION", ParameterValue: "WebGL 2.0", DataType: "DOMString", IsConsistent: true},
	}

	metadata := info.Fingerprint
	if metadata == "" {
		return parameters
	}

	return parameters
}

func (s *WebGLFingerprintService) computeFingerprint(gpuInfo model.WebGLGPUInfo, data *model.WebGLFingerprintData) string {
	components := []string{
		gpuInfo.Vendor,
		gpuInfo.Renderer,
		gpuInfo.UnmaskedVendor,
		gpuInfo.UnmaskedRenderer,
		string(gpuInfo.RendererType),
		fmt.Sprintf("%d", data.ParameterLimits.MaxTextureSize),
		fmt.Sprintf("%d", data.ParameterLimits.MaxRenderbufferSize),
		fmt.Sprintf("%d", data.ParameterLimits.MaxVertexAttribs),
		fmt.Sprintf("%d", len(data.Extensions)),
	}

	supportedExtCount := 0
	for _, ext := range data.Extensions {
		if ext.IsSupported {
			supportedExtCount++
			components = append(components, ext.ExtensionName)
		}
	}
	components = append(components, fmt.Sprintf("ext:%d", supportedExtCount))

	sort.Strings(components)
	combined := strings.Join(components, ":")

	h := sha256.New()
	h.Write([]byte(combined))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *WebGLFingerprintService) analyzeWebGLRisk(data *model.WebGLFingerprintData, info *model.EnvInfo) (string, float64, float64) {
	riskScore := 0.0
	confidence := 50.0

	if data.GPUInfo.Vendor == "" && data.GPUInfo.Renderer == "" {
		return "high", 100.0, 0.0
	}

	if data.GPUInfo.RendererType == model.WebGLRendererTypeSoftware {
		riskScore += 40.0
	}

	if data.GPUInfo.RendererType == model.WebGLRendererTypeVirtual {
		riskScore += 50.0
	}

	softwarePatterns := []string{"swiftshader", "llvmpipe", "software", "virtual", "mesa"}
	for _, pattern := range softwarePatterns {
		if strings.Contains(strings.ToLower(data.GPUInfo.Renderer), pattern) {
			riskScore += 15.0
		}
	}

	supportedExtCount := 0
	for _, ext := range data.Extensions {
		if ext.IsSupported {
			supportedExtCount++
		}
	}

	if supportedExtCount < 5 {
		riskScore += 20.0
		confidence -= 10.0
	}

	if data.ParameterLimits.MaxTextureSize < 2048 {
		riskScore += 15.0
	}

	if data.ParameterLimits.MaxVertexAttribs < 16 {
		riskScore += 10.0
	}

	if !s.isTrustedVendor(data.GPUInfo.Vendor) && !s.isTrustedVendor(data.GPUInfo.UnmaskedVendor) {
		riskScore += 10.0
		confidence -= 5.0
	}

	if supportedExtCount > 10 {
		confidence += 10.0
	}

	if data.GPUInfo.GPUFamily != "" && data.GPUInfo.GPUFamily != "Unknown" {
		confidence += 10.0
	}

	if riskScore < 20 {
		return "low", riskScore, confidence
	} else if riskScore < 50 {
		return "medium", riskScore, confidence
	}
	return "high", riskScore, confidence
}

func (s *WebGLFingerprintService) isTrustedVendor(vendor string) bool {
	vendorLower := strings.ToLower(vendor)
	for _, trusted := range s.config.TrustedVendors {
		if strings.Contains(strings.ToLower(trusted), vendorLower) || strings.Contains(vendorLower, strings.ToLower(trusted)) {
			return true
		}
	}
	return false
}

func (s *WebGLFingerprintService) detectAnomalies(data *model.WebGLFingerprintData, info *model.EnvInfo) []model.WebGLAnomaly {
	anomalies := make([]model.WebGLAnomaly, 0)

	if data.GPUInfo.Vendor == "" && data.GPUInfo.Renderer == "" {
		anomalies = append(anomalies, model.WebGLAnomaly{
			Type:        "missing_data",
			Severity:    "high",
			Description: "WebGL指纹数据完全缺失",
		})
	}

	if data.GPUInfo.Vendor == "" && data.GPUInfo.Renderer != "" {
		anomalies = append(anomalies, model.WebGLAnomaly{
			Type:        "incomplete_data",
			Severity:    "medium",
			Description: "缺少WebGL厂商信息",
			Field:       "vendor",
		})
	}

	if data.GPUInfo.Renderer == "" && data.GPUInfo.Vendor != "" {
		anomalies = append(anomalies, model.WebGLAnomaly{
			Type:        "incomplete_data",
			Severity:    "medium",
			Description: "缺少WebGL渲染器信息",
			Field:       "renderer",
		})
	}

	suspiciousPatterns := []string{"fake", "mock", "test", "spoof", "none", "unknown", "undefined", "null"}
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(strings.ToLower(data.GPUInfo.Vendor), pattern) {
			anomalies = append(anomalies, model.WebGLAnomaly{
				Type:        "suspicious_pattern",
				Severity:    "high",
				Description: fmt.Sprintf("厂商信息包含可疑模式: %s", pattern),
				Field:       "vendor",
				Expected:    "有效的GPU厂商名称",
				Actual:      data.GPUInfo.Vendor,
			})
		}

		if strings.Contains(strings.ToLower(data.GPUInfo.Renderer), pattern) {
			anomalies = append(anomalies, model.WebGLAnomaly{
				Type:        "suspicious_pattern",
				Severity:    "high",
				Description: fmt.Sprintf("渲染器信息包含可疑模式: %s", pattern),
				Field:       "renderer",
				Expected:    "有效的GPU渲染器名称",
				Actual:      data.GPUInfo.Renderer,
			})
		}
	}

	supportedExtCount := 0
	for _, ext := range data.Extensions {
		if ext.IsSupported {
			supportedExtCount++
		}
	}

	if supportedExtCount == 0 {
		anomalies = append(anomalies, model.WebGLAnomaly{
			Type:        "no_extensions",
			Severity:    "high",
			Description: "WebGL扩展数量为零，可能被完全禁用或伪造",
		})
	}

	if supportedExtCount > 0 && supportedExtCount < 3 {
		anomalies = append(anomalies, model.WebGLAnomaly{
			Type:        "few_extensions",
			Severity:    "medium",
			Description: fmt.Sprintf("WebGL扩展数量异常少: %d", supportedExtCount),
			Expected:    "至少10个标准扩展",
			Actual:      fmt.Sprintf("%d", supportedExtCount),
		})
	}

	if data.ParameterLimits.MaxTextureSize < 1024 {
		anomalies = append(anomalies, model.WebGLAnomaly{
			Type:        "limited_capability",
			Severity:    "medium",
			Description: fmt.Sprintf("纹理尺寸限制异常小: %d", data.ParameterLimits.MaxTextureSize),
			Expected:    "至少1024",
			Actual:      fmt.Sprintf("%d", data.ParameterLimits.MaxTextureSize),
		})
	}

	return anomalies
}

func (s *WebGLFingerprintService) AnalyzeStability(fingerprintID string, sessionID string) (*model.WebGLStabilityResult, error) {
	s.stabilityMu.Lock()
	defer s.stabilityMu.Unlock()

	stored, exists := s.fingerprintDB[fingerprintID]
	now := time.Now()

	if !exists {
		stored = &model.WebGLFingerprintStability{
			FingerprintID: fingerprintID,
			SessionID:     sessionID,
			FirstSeen:     now,
			LastSeen:      now,
			HitCount:      1,
			StabilityScore: 0.0,
			Variations:    make([]string, 0),
			IsStable:      false,
			Confidence:    0.0,
		}
		s.fingerprintDB[fingerprintID] = stored
	} else {
		stored.LastSeen = now
		stored.HitCount++

		if stored.SessionID != sessionID {
			if !containsString(stored.Variations, sessionID) {
				stored.Variations = append(stored.Variations, sessionID)
				stored.RendererChanges++
			}
		}
	}

	stored.StabilityScore = s.calculateStabilityScore(stored)
	stored.IsStable = stored.StabilityScore >= s.config.StabilityThreshold

	if stored.HitCount > 0 && len(stored.Variations) >= 0 {
		stored.Confidence = math.Min(float64(stored.HitCount)*0.1, 1.0)
	}

	return &model.WebGLStabilityResult{
		DeviceID:           fingerprintID,
		SessionCount:       stored.HitCount,
		UniqueFingerprints: len(stored.Variations) + 1,
		StabilityScore:     stored.StabilityScore,
		IsTrusted:          stored.IsStable && stored.Confidence >= 0.8,
		RendererStability:  s.calculateRendererStability(stored),
		VendorStability:    s.calculateVendorStability(stored),
		ExtensionStability: s.calculateExtensionStability(stored),
		FirstSeen:          stored.FirstSeen,
		LastSeen:           stored.LastSeen,
	}, nil
}

func (s *WebGLFingerprintService) calculateStabilityScore(stored *model.WebGLFingerprintStability) float64 {
	baseScore := 1.0

	hitDecay := math.Max(0, 1.0-float64(stored.HitCount)*0.02)
	baseScore *= hitDecay

	if len(stored.Variations) > 0 {
		variationPenalty := float64(len(stored.Variations)) * 0.15
		baseScore *= math.Max(0.1, 1.0-variationPenalty)
	}

	rendererChangePenalty := float64(stored.RendererChanges) * 0.2
	baseScore *= math.Max(0.1, 1.0-rendererChangePenalty)

	age := time.Since(stored.FirstSeen)
	ageDays := age.Hours() / 24
	if ageDays > 7 {
		baseScore *= 1.2
		if baseScore > 1.0 {
			baseScore = 1.0
		}
	}

	return math.Max(0, math.Min(1.0, baseScore))
}

func (s *WebGLFingerprintService) calculateRendererStability(stored *model.WebGLFingerprintStability) float64 {
	if stored.HitCount == 0 {
		return 0.0
	}

	baseScore := 1.0
	rendererChangePenalty := float64(stored.RendererChanges) * 0.3
	baseScore = math.Max(0.1, 1.0-rendererChangePenalty)

	return baseScore
}

func (s *WebGLFingerprintService) calculateVendorStability(stored *model.WebGLFingerprintStability) float64 {
	if stored.HitCount == 0 {
		return 0.0
	}

	baseScore := 1.0
	vendorChangePenalty := float64(stored.VendorChanges) * 0.3
	baseScore = math.Max(0.1, 1.0-vendorChangePenalty)

	return baseScore
}

func (s *WebGLFingerprintService) calculateExtensionStability(stored *model.WebGLFingerprintStability) float64 {
	if stored.HitCount == 0 {
		return 0.0
	}

	baseScore := 1.0
	extensionChangePenalty := float64(stored.ExtensionChanges) * 0.25
	baseScore = math.Max(0.1, 1.0-extensionChangePenalty)

	return baseScore
}

func (s *WebGLFingerprintService) CompareFingerprints(hash1, hash2 string, data1, data2 *model.WebGLFingerprintData) *model.WebGLComparisonResult {
	result := &model.WebGLComparisonResult{
		Similarity:       s.calculateSimilarity(hash1, hash2),
		CommonFeatures:   make([]string, 0),
		DifferentFeatures: make([]string, 0),
		IsSameDevice:     false,
		Confidence:       0.0,
	}

	if data1 == nil || data2 == nil {
		return result
	}

	if data1.GPUInfo.Vendor == data2.GPUInfo.Vendor && data1.GPUInfo.Vendor != "" {
		result.VendorMatch = true
		result.CommonFeatures = append(result.CommonFeatures, "vendor")
	} else {
		result.DifferentFeatures = append(result.DifferentFeatures, "vendor")
	}

	if data1.GPUInfo.Renderer == data2.GPUInfo.Renderer && data1.GPUInfo.Renderer != "" {
		result.RendererMatch = true
		result.CommonFeatures = append(result.CommonFeatures, "renderer")
	} else {
		result.DifferentFeatures = append(result.DifferentFeatures, "renderer")
	}

	extMatchCount := 0
	totalExt := 0
	for _, ext1 := range data1.Extensions {
		for _, ext2 := range data2.Extensions {
			if ext1.ExtensionName == ext2.ExtensionName {
				totalExt++
				if ext1.IsSupported == ext2.IsSupported {
					extMatchCount++
				}
				break
			}
		}
	}
	if totalExt > 0 {
		result.ExtensionMatch = float64(extMatchCount) / float64(totalExt)
	}

	result.Similarity = (result.Similarity*0.4 +
		map[bool]float64{true: 1.0, false: 0.0}[result.VendorMatch]*0.2 +
		map[bool]float64{true: 1.0, false: 0.0}[result.RendererMatch]*0.2 +
		result.ExtensionMatch*0.2) * 100

	result.IsSameDevice = result.Similarity >= s.config.SimilarityThreshold*100
	result.Confidence = result.Similarity / 100.0

	return result
}

func (s *WebGLFingerprintService) calculateSimilarity(hash1, hash2 string) float64 {
	if hash1 == hash2 {
		return 100.0
	}

	if len(hash1) != len(hash2) {
		return 0.0
	}

	if len(hash1) == 0 {
		return 0.0
	}

	matches := 0
	for i := 0; i < len(hash1) && i < len(hash2); i++ {
		if hash1[i] == hash2[i] {
			matches++
		}
	}

	return float64(matches) / float64(len(hash1)) * 100.0
}

func (s *WebGLFingerprintService) DetectSpoofing(originalHash, receivedHash string, originalData, receivedData *model.WebGLFingerprintData) *model.WebGLAntiSpoofResult {
	result := &model.WebGLAntiSpoofResult{
		IsSpoofed:          false,
		SpoofingIndicators: make([]string, 0),
		ConsistencyScore:   100.0,
		Recommendation:     "allow",
		SuspiciousPatterns: make([]string, 0),
		Confidence:        0.0,
	}

	if originalData == nil || receivedData == nil {
		result.SpoofingIndicators = append(result.SpoofingIndicators, "missing_fingerprint_data")
		result.ConsistencyScore -= 30.0
		return result
	}

	similarity := s.calculateSimilarity(originalHash, receivedHash)

	if similarity < 30.0 {
		result.IsSpoofed = true
		result.SpoofingIndicators = append(result.SpoofingIndicators, "指纹哈希相似度过低，可能来自不同设备或被伪造")
		result.ConsistencyScore -= 50.0
		result.SuspiciousPatterns = append(result.SuspiciousPatterns, "hash_mismatch")
	}

	if len(originalHash) != len(receivedHash) {
		result.SpoofingIndicators = append(result.SpoofingIndicators, "指纹长度不匹配")
		result.ConsistencyScore -= 20.0
		result.IsSpoofed = true
		result.SuspiciousPatterns = append(result.SuspiciousPatterns, "length_mismatch")
	}

	if originalData.GPUInfo.Vendor != receivedData.GPUInfo.Vendor &&
		originalData.GPUInfo.Vendor != "" && receivedData.GPUInfo.Vendor != "" {
		result.SpoofingIndicators = append(result.SpoofingIndicators, "WebGL厂商不匹配")
		result.ConsistencyScore -= 15.0
		result.SuspiciousPatterns = append(result.SuspiciousPatterns, "vendor_mismatch")
	}

	if originalData.GPUInfo.Renderer != receivedData.GPUInfo.Renderer &&
		originalData.GPUInfo.Renderer != "" && receivedData.GPUInfo.Renderer != "" {
		result.SpoofingIndicators = append(result.SpoofingIndicators, "WebGL渲染器不匹配")
		result.ConsistencyScore -= 20.0
		result.SuspiciousPatterns = append(result.SuspiciousPatterns, "renderer_mismatch")
	}

	if originalData.GPUInfo.RendererType != receivedData.GPUInfo.RendererType {
		result.SpoofingIndicators = append(result.SpoofingIndicators, "渲染器类型发生变化")
		result.ConsistencyScore -= 25.0
		result.SuspiciousPatterns = append(result.SuspiciousPatterns, "renderer_type_change")
	}

	originalExtCount := 0
	for _, ext := range originalData.Extensions {
		if ext.IsSupported {
			originalExtCount++
		}
	}
	receivedExtCount := 0
	for _, ext := range receivedData.Extensions {
		if ext.IsSupported {
			receivedExtCount++
		}
	}

	if originalExtCount > 0 && receivedExtCount > 0 {
		extDiff := math.Abs(float64(originalExtCount - receivedExtCount))
		if extDiff > float64(originalExtCount)*0.5 {
			result.SpoofingIndicators = append(result.SpoofingIndicators, "WebGL扩展数量差异过大")
			result.ConsistencyScore -= 15.0
			result.SuspiciousPatterns = append(result.SuspiciousPatterns, "extension_count_mismatch")
		}
	}

	softwareIndicators := []string{"swiftshader", "llvmpipe", "mesa", "software"}
	for _, indicator := range softwareIndicators {
		if strings.Contains(strings.ToLower(receivedData.GPUInfo.Renderer), indicator) {
			result.SuspiciousPatterns = append(result.SuspiciousPatterns, "software_renderer_detected")
			result.ConsistencyScore -= 10.0
		}
	}

	if result.ConsistencyScore < 50 {
		result.Recommendation = "block"
		result.IsSpoofed = true
	} else if result.ConsistencyScore < 70 {
		result.Recommendation = "review"
	}

	result.Confidence = math.Max(0, math.Min(1.0, result.ConsistencyScore/100.0))

	return result
}

func (s *WebGLFingerprintService) GenerateRenderAnalysis(webglHash string, enhancedData *model.WebGLFingerprintData) *model.WebGLRenderAnalysis {
	analysis := &model.WebGLRenderAnalysis{
		RenderHash:        webglHash,
		Features:          make([]string, 0),
		ComplexityScore:   0.0,
		UniquenessScore:   0.0,
		RenderingPatterns: make([]string, 0),
	}

	if enhancedData == nil {
		return analysis
	}

	analysis.Features = append(analysis.Features, fmt.Sprintf("vendor:%s", enhancedData.GPUInfo.Vendor))
	analysis.Features = append(analysis.Features, fmt.Sprintf("renderer:%s", enhancedData.GPUInfo.Renderer))
	analysis.Features = append(analysis.Features, fmt.Sprintf("renderer_type:%s", enhancedData.GPUInfo.RendererType))

	if enhancedData.GPUInfo.GPUFamily != "" {
		analysis.Features = append(analysis.Features, fmt.Sprintf("family:%s", enhancedData.GPUInfo.GPUFamily))
		analysis.ComplexityScore += 10.0
	}

	analysis.ComplexityScore += float64(len(enhancedData.Extensions))
	analysis.ComplexityScore += float64(enhancedData.ParameterLimits.MaxTextureSize) / 1000.0

	supportedExtCount := 0
	for _, ext := range enhancedData.Extensions {
		if ext.IsSupported {
			supportedExtCount++
		}
	}
	analysis.Features = append(analysis.Features, fmt.Sprintf("extensions:%d", supportedExtCount))
	analysis.UniquenessScore += float64(supportedExtCount) * 2.0

	if enhancedData.GPUInfo.RendererType == model.WebGLRendererTypeHardware {
		analysis.RenderingPatterns = append(analysis.RenderingPatterns, "hardware_accelerated")
		analysis.UniquenessScore += 20.0
	} else if enhancedData.GPUInfo.RendererType == model.WebGLRendererTypeSoftware {
		analysis.RenderingPatterns = append(analysis.RenderingPatterns, "software_rendering")
		analysis.UniquenessScore -= 10.0
	} else if enhancedData.GPUInfo.RendererType == model.WebGLRendererTypeVirtual {
		analysis.RenderingPatterns = append(analysis.RenderingPatterns, "virtual_gpu")
		analysis.UniquenessScore -= 15.0
	}

	if enhancedData.GPUInfo.IsDiscrete {
		analysis.RenderingPatterns = append(analysis.RenderingPatterns, "discrete_gpu")
		analysis.UniquenessScore += 10.0
	} else if enhancedData.GPUInfo.IsIntegrated {
		analysis.RenderingPatterns = append(analysis.RenderingPatterns, "integrated_gpu")
	}

	if analysis.ComplexityScore > 100 {
		analysis.ComplexityScore = 100
	}
	if analysis.UniquenessScore > 100 {
		analysis.UniquenessScore = 100
	}
	if analysis.UniquenessScore < 0 {
		analysis.UniquenessScore = 0
	}

	return analysis
}

func (s *WebGLFingerprintService) ExportFingerprintData(fingerprintID string) (string, error) {
	s.stabilityMu.RLock()
	defer s.stabilityMu.RUnlock()

	data, exists := s.fingerprintDB[fingerprintID]
	if !exists {
		return "", fmt.Errorf("fingerprint not found")
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

func (s *WebGLFingerprintService) ImportFingerprintData(fingerprintID string, jsonData string) error {
	s.stabilityMu.Lock()
	defer s.stabilityMu.Unlock()

	var data model.WebGLFingerprintStability
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return err
	}

	data.FingerprintID = fingerprintID
	s.fingerprintDB[fingerprintID] = &data

	return nil
}

func (s *WebGLFingerprintService) ClearExpiredData(expiration time.Duration) int {
	s.stabilityMu.Lock()
	defer s.stabilityMu.Unlock()

	cutoff := time.Now().Add(-expiration)
	removed := 0

	for id, data := range s.fingerprintDB {
		if data.LastSeen.Before(cutoff) {
			delete(s.fingerprintDB, id)
			removed++
		}
	}

	return removed
}

func (s *WebGLFingerprintService) GetStatistics() map[string]interface{} {
	s.stabilityMu.RLock()
	defer s.stabilityMu.RUnlock()

	stats := make(map[string]interface{})

	stats["total_fingerprints"] = len(s.fingerprintDB)

	totalHits := 0
	totalVariations := 0
	stableCount := 0
	softwareRendererCount := 0
	virtualGPUCount := 0

	for _, data := range s.fingerprintDB {
		totalHits += data.HitCount
		totalVariations += len(data.Variations)
		if data.IsStable {
			stableCount++
		}
	}

	stats["total_hits"] = totalHits
	stats["total_variations"] = totalVariations
	stats["stable_count"] = stableCount
	stats["software_renderer_count"] = softwareRendererCount
	stats["virtual_gpu_count"] = virtualGPUCount

	if len(s.fingerprintDB) > 0 {
		stats["avg_hits_per_fp"] = float64(totalHits) / float64(len(s.fingerprintDB))
		stats["stability_rate"] = float64(stableCount) / float64(len(s.fingerprintDB))
	} else {
		stats["avg_hits_per_fp"] = 0.0
		stats["stability_rate"] = 0.0
	}

	return stats
}

func (s *WebGLFingerprintService) GetConfig() *model.WebGLEnhancementConfig {
	return s.config
}

func (s *WebGLFingerprintService) UpdateConfig(config *model.WebGLEnhancementConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config.EnableParameterExtraction {
		s.config.EnableParameterExtraction = config.EnableParameterExtraction
	}
	if config.EnableExtensionAnalysis {
		s.config.EnableExtensionAnalysis = config.EnableExtensionAnalysis
	}
	if config.EnableRendererIdentification {
		s.config.EnableRendererIdentification = config.EnableRendererIdentification
	}
	if config.EnableStabilityTrack {
		s.config.EnableStabilityTrack = config.EnableStabilityTrack
	}
	if config.EnableAnomalyDetection {
		s.config.EnableAnomalyDetection = config.EnableAnomalyDetection
	}
	if config.EnableAntiSpoof {
		s.config.EnableAntiSpoof = config.EnableAntiSpoof
	}
	if config.StabilityThreshold > 0 {
		s.config.StabilityThreshold = config.StabilityThreshold
	}
	if config.AnomalyThreshold > 0 {
		s.config.AnomalyThreshold = config.AnomalyThreshold
	}
	if config.SimilarityThreshold > 0 {
		s.config.SimilarityThreshold = config.SimilarityThreshold
	}
	if len(config.TrustedVendors) > 0 {
		s.config.TrustedVendors = config.TrustedVendors
	}
	if len(config.SoftwareRenderers) > 0 {
		s.config.SoftwareRenderers = config.SoftwareRenderers
	}
	if len(config.VirtualGPUs) > 0 {
		s.config.VirtualGPUs = config.VirtualGPUs
	}
}

func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func containsAny(str string, list []string) bool {
	strLower := strings.ToLower(str)
	for _, item := range list {
		if strings.Contains(strLower, strings.ToLower(item)) {
			return true
		}
	}
	return false
}

type SimulatedWebGLRenderer struct {
	vendor   string
	renderer string
	version  string
	limits   model.WebGLParameterLimits
	extensions []string
}

func NewSimulatedWebGLRenderer(rendererType string) *SimulatedWebGLRenderer {
	sim := &SimulatedWebGLRenderer{
		vendor:      "Google Inc.",
		renderer:    "Simulated Renderer",
		version:     "WebGL 2.0",
		extensions: make([]string, 0),
	}

	switch rendererType {
	case "hardware":
		sim.vendor = "NVIDIA Corporation"
		sim.renderer = "GeForce GTX 1080/PCIe/SSE2"
		sim.limits = model.WebGLParameterLimits{
			MaxTextureSize:              32768,
			MaxTextureSize3D:            2048,
			MaxRenderbufferSize:         32768,
			MaxVertexAttribs:            16,
			MaxVertexUniformVectors:     4096,
			MaxFragmentUniformVectors:   4096,
			MaxVaryingVectors:           15,
			MaxViewportDims:             []int{32768, 32768},
			MaxCubeMapTextureSize:       32768,
			MaxTextureImageUnits:        16,
			MaxVertexTextureImageUnits:  16,
			MaxCombinedTextureImageUnits: 32,
			MaxSamples:                  8,
			MaxColorAttachments:         8,
			MaxDrawBuffers:              8,
		}
		sim.extensions = []string{
			"GL_EXT_blend_minmax",
			"GL_EXT_color_buffer_float",
			"GL_EXT_frag_depth",
			"GL_OES_standard_derivatives",
			"GL_OES_texture_float",
			"GL_ANGLE_instanced_arrays",
			"GL_WEBGL_compressed_texture_s3tc",
		}
	case "software":
		sim.vendor = "Google Inc."
		sim.renderer = "SwiftShader 4.0.0.1"
		sim.limits = model.WebGLParameterLimits{
			MaxTextureSize:              4096,
			MaxTextureSize3D:            256,
			MaxRenderbufferSize:         4096,
			MaxVertexAttribs:            16,
			MaxVertexUniformVectors:     1024,
			MaxFragmentUniformVectors:   1024,
			MaxVaryingVectors:           15,
			MaxViewportDims:             []int{4096, 4096},
			MaxCubeMapTextureSize:       4096,
			MaxTextureImageUnits:        16,
			MaxVertexTextureImageUnits:  0,
			MaxCombinedTextureImageUnits: 16,
			MaxSamples:                  0,
			MaxColorAttachments:         1,
			MaxDrawBuffers:              1,
		}
		sim.extensions = []string{
			"GL_EXT_blend_minmax",
			"GL_OES_standard_derivatives",
		}
	case "virtual":
		sim.vendor = "VMware, Inc."
		sim.renderer = "llvmpipe (LLVM 12.0.0, 256 bits)"
		sim.limits = model.WebGLParameterLimits{
			MaxTextureSize:              8192,
			MaxTextureSize3D:            512,
			MaxRenderbufferSize:         8192,
			MaxVertexAttribs:            16,
			MaxVertexUniformVectors:     1024,
			MaxFragmentUniformVectors:   1024,
			MaxVaryingVectors:           15,
			MaxViewportDims:             []int{8192, 8192},
			MaxCubeMapTextureSize:       8192,
			MaxTextureImageUnits:        16,
			MaxVertexTextureImageUnits:  0,
			MaxCombinedTextureImageUnits: 16,
			MaxSamples:                  4,
			MaxColorAttachments:         8,
			MaxDrawBuffers:              8,
		}
		sim.extensions = []string{
			"GL_EXT_blend_minmax",
			"GL_EXT_color_buffer_float",
			"GL_OES_standard_derivatives",
		}
	default:
		sim.renderer = "Unknown Renderer"
	}

	return sim
}

func (r *SimulatedWebGLRenderer) GetFingerprint() *model.WebGLFingerprintData {
	extensions := make([]model.WebGLSupportedExtension, 0)
	for _, ext := range r.extensions {
		extensions = append(extensions, model.WebGLSupportedExtension{
			ExtensionName: ext,
			IsSupported:   true,
		})
	}

	return &model.WebGLFingerprintData{
		GPUInfo: model.WebGLGPUInfo{
			Vendor:             r.vendor,
			Renderer:           r.renderer,
			UnmaskedVendor:    r.vendor,
			UnmaskedRenderer:  r.renderer,
			RendererType:      r.identifyType(),
		},
		ParameterLimits: r.limits,
		Extensions:      extensions,
		RawExtensions:   r.extensions,
		Version:         r.version,
	}
}

func (r *SimulatedWebGLRenderer) identifyType() model.WebGLRendererType {
	rendererLower := strings.ToLower(r.renderer)
	if strings.Contains(rendererLower, "swiftshader") || strings.Contains(rendererLower, "llvmpipe") || strings.Contains(rendererLower, "mesa") {
		return model.WebGLRendererTypeSoftware
	}
	if strings.Contains(rendererLower, "virtualbox") || strings.Contains(rendererLower, "vmware") || strings.Contains(rendererLower, "qemu") {
		return model.WebGLRendererTypeVirtual
	}
	return model.WebGLRendererTypeHardware
}

func GenerateSimulatedWebGLFingerprint(userAgent string, rendererType string) (*model.WebGLFingerprintData, string) {
	renderer := NewSimulatedWebGLRenderer(rendererType)
	data := renderer.GetFingerprint()

	service := NewWebGLFingerprintService()
	info := &model.EnvInfo{
		WebGLVendor:   data.GPUInfo.Vendor,
		WebGLRenderer: data.GPUInfo.Renderer,
		Fingerprint:   "",
	}

	result := service.GenerateEnhancedFingerprint(info)

	return result.EnhancedData, result.Fingerprint
}

func SimulateWebGLContextLoss() *model.EnvInfo {
	return &model.EnvInfo{
		WebGLVendor:   "",
		WebGLRenderer: "",
		WebGLSupport: false,
		Fingerprint:  "",
	}
}

func SimulateWebGLEmptyExtensions() *model.EnvInfo {
	return &model.EnvInfo{
		WebGLVendor:   "Test Vendor",
		WebGLRenderer: "Test Renderer",
		WebGLSupport: true,
		Fingerprint:  "no_extensions",
	}
}

func SimulateWebGLSoftwareRenderer() *model.EnvInfo {
	return &model.EnvInfo{
		WebGLVendor:   "Google Inc.",
		WebGLRenderer: "SwiftShader 4.0.0.1",
		WebGLSupport: true,
		Fingerprint:  "texture_size:4096",
	}
}

func SimulateWebGLVirtualGPU() *model.EnvInfo {
	return &model.EnvInfo{
		WebGLVendor:   "VMware, Inc.",
		WebGLRenderer: "llvmpipe (LLVM 12.0.0, 256 bits)",
		WebGLSupport: true,
		Fingerprint:  "texture_size:8192",
	}
}
