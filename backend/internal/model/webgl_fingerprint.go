package model

import "time"

type WebGLRendererType string

const (
	WebGLRendererTypeHardware WebGLRendererType = "hardware"
	WebGLRendererTypeSoftware WebGLRendererType = "software"
	WebGLRendererTypeVirtual  WebGLRendererType = "virtual"
	WebGLRendererTypeUnknown  WebGLRendererType = "unknown"
)

type WebGLParameterInfo struct {
	ParameterName  string `json:"parameter_name"`
	ParameterValue interface{} `json:"parameter_value"`
	DataType       string `json:"data_type"`
	IsConsistent   bool   `json:"is_consistent"`
}

type WebGLSupportedExtension struct {
	ExtensionName string   `json:"extension_name"`
	IsSupported    bool     `json:"is_supported"`
	Version        string   `json:"version,omitempty"`
	Category       string   `json:"category,omitempty"`
}

type WebGLGPUInfo struct {
	Vendor           string             `json:"vendor"`
	Renderer         string             `json:"renderer"`
	UnmaskedVendor   string             `json:"unmasked_vendor"`
	UnmaskedRenderer string            `json:"unmasked_renderer"`
	RendererType     WebGLRendererType  `json:"renderer_type"`
	GPUFamily        string             `json:"gpu_family,omitempty"`
	GPUModel         string             `json:"gpu_model,omitempty"`
	IsIntegrated     bool               `json:"is_integrated"`
	IsDiscrete       bool               `json:"is_discrete"`
	DriverVersion    string             `json:"driver_version,omitempty"`
}

type WebGLParameterLimits struct {
	MaxTextureSize              int `json:"max_texture_size"`
	MaxTextureSize3D            int `json:"max_texture_size_3d"`
	MaxRenderbufferSize         int `json:"max_renderbuffer_size"`
	MaxVertexAttribs            int `json:"max_vertex_attribs"`
	MaxVertexUniformVectors     int `json:"max_vertex_uniform_vectors"`
	MaxFragmentUniformVectors   int `json:"max_fragment_uniform_vectors"`
	MaxVaryingVectors           int `json:"max_varying_vectors"`
	MaxViewportDims             []int `json:"max_viewport_dims"`
	MaxCubeMapTextureSize      int `json:"max_cube_map_texture_size"`
	MaxTextureImageUnits        int `json:"max_texture_image_units"`
	MaxVertexTextureImageUnits  int `json:"max_vertex_texture_image_units"`
	MaxCombinedTextureImageUnits int `json:"max_combined_texture_image_units"`
	MaxSamples                  int `json:"max_samples"`
	MaxColorAttachments         int `json:"max_color_attachments"`
	MaxDrawBuffers              int `json:"max_draw_buffers"`
}

type WebGLRenderingFeature struct {
	FeatureName     string `json:"feature_name"`
	IsSupported     bool   `json:"is_supported"`
	SupportLevel    string `json:"support_level"`
	PerformanceHint string `json:"performance_hint,omitempty"`
}

type WebGLFingerprintData struct {
	GPUInfo          WebGLGPUInfo          `json:"gpu_info"`
	ParameterLimits  WebGLParameterLimits  `json:"parameter_limits"`
	Extensions       []WebGLSupportedExtension `json:"extensions"`
	RenderingFeatures []WebGLRenderingFeature `json:"rendering_features"`
	Parameters       []WebGLParameterInfo  `json:"parameters"`
	RawExtensions    []string               `json:"raw_extensions"`
	Version          string                 `json:"version"`
	VersionMajor     int                    `json:"version_major"`
	VersionMinor     int                    `json:"version_minor"`
	ContextType      string                 `json:"context_type"`
}

type WebGLFingerprintStability struct {
	FingerprintID    string    `json:"fingerprint_id"`
	SessionID        string    `json:"session_id"`
	FirstSeen        time.Time `json:"first_seen"`
	LastSeen         time.Time `json:"last_seen"`
	HitCount         int       `json:"hit_count"`
	StabilityScore   float64   `json:"stability_score"`
	Variations       []string  `json:"variations"`
	IsStable         bool      `json:"is_stable"`
	Confidence       float64   `json:"confidence"`
	RendererChanges  int       `json:"renderer_changes"`
	VendorChanges    int       `json:"vendor_changes"`
	ExtensionChanges int       `json:"extension_changes"`
}

type WebGLAnomaly struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Field       string `json:"field,omitempty"`
	Expected    string `json:"expected,omitempty"`
	Actual      string `json:"actual,omitempty"`
}

type WebGLFingerprintResult struct {
	Success          bool                   `json:"success"`
	Fingerprint      string                 `json:"fingerprint"`
	Hash             string                 `json:"hash"`
	EnhancedData     *WebGLFingerprintData `json:"enhanced_data,omitempty"`
	StabilityAnalysis *WebGLStabilityResult  `json:"stability_analysis,omitempty"`
	Anomalies        []WebGLAnomaly         `json:"anomalies,omitempty"`
	RiskLevel        string                 `json:"risk_level"`
	RiskScore        float64                `json:"risk_score"`
	Confidence       float64                `json:"confidence"`
	Error            string                 `json:"error,omitempty"`
	Warnings         []string               `json:"warnings,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

type WebGLStabilityResult struct {
	DeviceID            string    `json:"device_id"`
	SessionCount        int       `json:"session_count"`
	UniqueFingerprints  int       `json:"unique_fingerprints"`
	StabilityScore      float64   `json:"stability_score"`
	IsTrusted           bool      `json:"is_trusted"`
	RendererStability   float64   `json:"renderer_stability"`
	VendorStability     float64   `json:"vendor_stability"`
	ExtensionStability  float64   `json:"extension_stability"`
	FirstSeen           time.Time `json:"first_seen"`
	LastSeen            time.Time `json:"last_seen"`
	ChangesDetected     []string  `json:"changes_detected,omitempty"`
}

type WebGLComparisonResult struct {
	Similarity      float64  `json:"similarity"`
	CommonFeatures   []string `json:"common_features"`
	DifferentFeatures []string `json:"different_features"`
	IsSameDevice     bool     `json:"is_same_device"`
	Confidence       float64  `json:"confidence"`
	VendorMatch      bool     `json:"vendor_match"`
	RendererMatch    bool     `json:"renderer_match"`
	ExtensionMatch   float64  `json:"extension_match"`
}

type WebGLAntiSpoofResult struct {
	IsSpoofed           bool     `json:"is_spoofed"`
	SpoofingIndicators  []string `json:"spoofing_indicators"`
	ConsistencyScore    float64  `json:"consistency_score"`
	Recommendation      string   `json:"recommendation"`
	SuspiciousPatterns  []string `json:"suspicious_patterns,omitempty"`
	Confidence          float64  `json:"confidence"`
}

type WebGLRenderAnalysis struct {
	RenderHash        string   `json:"render_hash"`
	Features          []string `json:"features"`
	ComplexityScore   float64  `json:"complexity_score"`
	UniquenessScore   float64  `json:"uniqueness_score"`
	RenderingPatterns []string `json:"rendering_patterns"`
	Anomalies         []WebGLAnomaly `json:"anomalies,omitempty"`
}

type WebGLEnhancementConfig struct {
	EnableParameterExtraction  bool     `json:"enable_parameter_extraction"`
	EnableExtensionAnalysis    bool     `json:"enable_extension_analysis"`
	EnableRendererIdentification bool   `json:"enable_renderer_identification"`
	EnableStabilityTrack       bool     `json:"enable_stability_track"`
	EnableAnomalyDetection     bool     `json:"enable_anomaly_detection"`
	EnableAntiSpoof            bool     `json:"enable_anti_spoof"`
	StabilityThreshold         float64  `json:"stability_threshold"`
	AnomalyThreshold           float64  `json:"anomaly_threshold"`
	SimilarityThreshold        float64  `json:"similarity_threshold"`
	TrustedVendors             []string `json:"trusted_vendors"`
	SoftwareRenderers          []string `json:"software_renderers"`
	VirtualGPUs                []string `json:"virtual_gpus"`
}

type WebGLFeatureVector struct {
	HashComponents   []string  `json:"hash_components"`
	NumericFeatures  []float64 `json:"numeric_features"`
	TextFeatures     []string  `json:"text_features"`
	RenderFeatures   []string  `json:"render_features"`
	FinalHash        string    `json:"final_hash"`
}
