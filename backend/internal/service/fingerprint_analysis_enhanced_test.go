package service

import (
	"math"
	"testing"
)

func TestEnhancedFingerprintAnalyzer_New(t *testing.T) {
	analyzer := NewEnhancedFingerprintAnalyzer()
	if analyzer == nil {
		t.Fatal("Expected analyzer to be created")
	}
	if analyzer.canvasAnalyzer == nil {
		t.Error("Expected canvasAnalyzer to be initialized")
	}
	if analyzer.webglAnalyzer == nil {
		t.Error("Expected webglAnalyzer to be initialized")
	}
	if analyzer.fontAnalyzer == nil {
		t.Error("Expected fontAnalyzer to be initialized")
	}
	if analyzer.audioAnalyzer == nil {
		t.Error("Expected audioAnalyzer to be initialized")
	}
	if analyzer.mediaAnalyzer == nil {
		t.Error("Expected mediaAnalyzer to be initialized")
	}
	if analyzer.browserMatcher == nil {
		t.Error("Expected browserMatcher to be initialized")
	}
}

func TestCanvasFingerprintAnalyzer_Analyze(t *testing.T) {
	analyzer := NewCanvasFingerprintAnalyzer()

	data := map[string]interface{}{
		"canvas_hash":         "test_hash_123",
		"canvas_rgba_histogram": []interface{}{100, 200, 150, 80},
		"canvas_noise_signature": 0.05,
	}

	result := analyzer.Analyze(data)

	if result.Hash != "test_hash_123" {
		t.Errorf("Expected hash 'test_hash_123', got '%s'", result.Hash)
	}
	if len(result.RGBAHistogram) != 4 {
		t.Errorf("Expected 4 histogram values, got %d", len(result.RGBAHistogram))
	}
	if result.NoiseSignature != 0.05 {
		t.Errorf("Expected noise signature 0.05, got %f", result.NoiseSignature)
	}
}

func TestCanvasFingerprintAnalyzer_CalculateSimilarity(t *testing.T) {
	analyzer := NewCanvasFingerprintAnalyzer()

	fp1 := &CanvasFingerprint{
		Hash:              "hash123",
		RGBAHistogram:     []int{100, 200, 150},
		NoiseSignature:   0.05,
		GradientPatterns: []string{"linear", "radial"},
		PathPatterns:     []string{"arc", "rect"},
	}

	fp2 := &CanvasFingerprint{
		Hash:              "hash123",
		RGBAHistogram:     []int{100, 200, 150},
		NoiseSignature:   0.06,
		GradientPatterns: []string{"linear", "radial"},
		PathPatterns:     []string{"arc", "rect"},
	}

	sim := analyzer.CalculateSimilarity(fp1, fp2)
	if sim < 10 {
		t.Errorf("Expected similarity >= 10%% for identical hashes, got %.2f%%", sim)
	}

	fp3 := &CanvasFingerprint{
		Hash:              "hash456",
		RGBAHistogram:     []int{50, 100, 75},
		NoiseSignature:   0.1,
		GradientPatterns: []string{"pattern"},
		PathPatterns:     []string{"line"},
	}

	sim2 := analyzer.CalculateSimilarity(fp1, fp3)
	if sim2 > sim {
		t.Errorf("Expected different fingerprints to have lower similarity")
	}
}

func TestCanvasFingerprintAnalyzer_CalculateSimilarity_Nil(t *testing.T) {
	analyzer := NewCanvasFingerprintAnalyzer()

	sim := analyzer.CalculateSimilarity(nil, nil)
	if sim != 0 {
		t.Errorf("Expected 0 similarity for nil fingerprints, got %.2f", sim)
	}

	fp := &CanvasFingerprint{Hash: "hash"}
	sim = analyzer.CalculateSimilarity(fp, nil)
	if sim != 0 {
		t.Errorf("Expected 0 similarity with nil, got %.2f", sim)
	}
}

func TestCanvasFingerprintAnalyzer_CalculateUniquenessScore(t *testing.T) {
	analyzer := NewCanvasFingerprintAnalyzer()

	fp := &CanvasFingerprint{
		RGBAHistogram:     make([]int, 100),
		NoiseSignature:    0.05,
		GradientPatterns:  []string{"grad1", "grad2", "grad3"},
		PathPatterns:      []string{"path1", "path2", "path3", "path4"},
		CompositeOps:      []string{"op1", "op2", "op3"},
	}

	score := analyzer.calculateUniquenessScore(fp)
	if score < 70 {
		t.Errorf("Expected uniqueness score > 70 for rich fingerprint, got %.2f", score)
	}
}

func TestCanvasFingerprintAnalyzer_ExtractRenderingFeatures(t *testing.T) {
	analyzer := NewCanvasFingerprintAnalyzer()

	data := map[string]interface{}{
		"shadow_blur":          float64(5),
		"shadow_offset_x":     float64(3),
		"shadow_offset_y":     float64(3),
		"global_alpha":        float64(0.8),
		"miter_limit":         float64(10),
		"line_width":          float64(2),
		"line_cap":            "round",
		"line_join":           "miter",
		"has_linear_gradient": true,
		"has_radial_gradient": true,
		"has_pattern_fill":    false,
	}

	features := analyzer.extractRenderingFeatures(data)

	if !features.HasShadow {
		t.Error("Expected HasShadow to be true")
	}
	if features.ShadowBlur != 5 {
		t.Errorf("Expected ShadowBlur 5, got %f", features.ShadowBlur)
	}
	if features.LineCap != "round" {
		t.Errorf("Expected LineCap 'round', got '%s'", features.LineCap)
	}
	if !features.HasLinearGradient {
		t.Error("Expected HasLinearGradient to be true")
	}
}

func TestCanvasFingerprintAnalyzer_ExtractTextMetrics(t *testing.T) {
	analyzer := NewCanvasFingerprintAnalyzer()

	data := map[string]interface{}{
		"width":                      float64(100.5),
		"actualBoundingBoxLeft":      float64(10.2),
		"actualBoundingBoxRight":     float64(90.3),
		"actualBoundingBoxAscent":   float64(80.1),
		"actualBoundingBoxDescent":  float64(20.5),
		"fontBoundingBoxAscent":     float64(100.0),
		"fontBoundingBoxDescent":    float64(25.0),
		"emHeightAscent":            float64(75.0),
		"emHeightDescent":           float64(25.0),
		"baseline":                  float64(0),
		"capHeight":                 float64(70.0),
	}

	metrics := analyzer.extractTextMetrics(data)

	if metrics.Width != 100.5 {
		t.Errorf("Expected Width 100.5, got %f", metrics.Width)
	}
	if metrics.ActualBoundingBoxLeft != 10.2 {
		t.Errorf("Expected ActualBoundingBoxLeft 10.2, got %f", metrics.ActualBoundingBoxLeft)
	}
	if metrics.CapHeight != 70.0 {
		t.Errorf("Expected CapHeight 70.0, got %f", metrics.CapHeight)
	}
}

func TestWebGLFingerprintAnalyzer_Analyze(t *testing.T) {
	analyzer := NewWebGLFingerprintAnalyzer()

	data := map[string]interface{}{
		"webgl_hash":          "wgl_hash_123",
		"webgl_vendor":        "Google Inc.",
		"webgl_renderer":      "ANGLE (Intel, Intel(R) UHD Graphics Direct3D11)",
		"webgl_max_texture_size": float64(16384),
		"webgl_extensions":    []interface{}{"OES_texture_float", "WEBGL_debug_renderer_info"},
	}

	result := analyzer.Analyze(data)

	if result.Hash != "wgl_hash_123" {
		t.Errorf("Expected hash 'wgl_hash_123', got '%s'", result.Hash)
	}
	if result.Vendor != "Google Inc." {
		t.Errorf("Expected vendor 'Google Inc.', got '%s'", result.Vendor)
	}
	if result.MaxTextureSize != 16384 {
		t.Errorf("Expected MaxTextureSize 16384, got %d", result.MaxTextureSize)
	}
	if result.ExtensionCount != 2 {
		t.Errorf("Expected ExtensionCount 2, got %d", result.ExtensionCount)
	}
}

func TestWebGLFingerprintAnalyzer_DetectSoftwareRenderer(t *testing.T) {
	analyzer := NewWebGLFingerprintAnalyzer()

	testCases := []struct {
		renderer string
		expected bool
	}{
		{"SwiftShader", true},
		{"llvmpipe", true},
		{"Mesa", true},
		{"Software", true},
		{"Intel UHD Graphics", false},
		{"NVIDIA GeForce RTX 3080", false},
	}

	for _, tc := range testCases {
		result := analyzer.detectSoftwareRenderer(tc.renderer)
		if result != tc.expected {
			t.Errorf("For renderer '%s': expected %v, got %v", tc.renderer, tc.expected, result)
		}
	}
}

func TestWebGLFingerprintAnalyzer_DetectVirtualGPU(t *testing.T) {
	analyzer := NewWebGLFingerprintAnalyzer()

	testCases := []struct {
		renderer string
		expected bool
	}{
		{"VMware SVGA", true},
		{"VirtualBox Graphics", true},
		{"Parallels Graphics", true},
		{"QEMU Virtual GPU", true},
		{"KVM", true},
		{"Intel UHD Graphics", false},
		{"AMD Radeon RX 580", false},
	}

	for _, tc := range testCases {
		result := analyzer.detectVirtualGPU(tc.renderer)
		if result != tc.expected {
			t.Errorf("For renderer '%s': expected %v, got %v", tc.renderer, tc.expected, result)
		}
	}
}

func TestWebGLFingerprintAnalyzer_CalculateSimilarity(t *testing.T) {
	analyzer := NewWebGLFingerprintAnalyzer()

	fp1 := &WebGLFingerprint{
		Hash:               "wgl_hash_123",
		UnmaskedRenderer:   "Intel UHD Graphics",
		RendererFingerprint: "renderer_fp_1",
		SupportedExtensions: []string{"ext1", "ext2", "ext3"},
		MaxTextureSize:    16384,
		MaxVertexAttribs:  16,
	}

	fp2 := &WebGLFingerprint{
		Hash:               "wgl_hash_123",
		UnmaskedRenderer:   "Intel UHD Graphics",
		RendererFingerprint: "renderer_fp_1",
		SupportedExtensions: []string{"ext1", "ext2", "ext3"},
		MaxTextureSize:    16384,
		MaxVertexAttribs:  16,
	}

	sim := analyzer.CalculateSimilarity(fp1, fp2)
	if sim < 10 {
		t.Errorf("Expected similarity >= 10%% for identical fingerprints, got %.2f%%", sim)
	}

	fp3 := &WebGLFingerprint{
		Hash:               "wgl_hash_456",
		UnmaskedRenderer:   "SwiftShader",
		RendererFingerprint: "renderer_fp_2",
		SupportedExtensions: []string{"ext1"},
		MaxTextureSize:    4096,
		MaxVertexAttribs:  8,
	}

	sim2 := analyzer.CalculateSimilarity(fp1, fp3)
	if sim2 > sim {
		t.Errorf("Expected different fingerprints to have lower similarity")
	}
}

func TestWebGLFingerprintAnalyzer_CalculateUniquenessScore(t *testing.T) {
	analyzer := NewWebGLFingerprintAnalyzer()

	fp := &WebGLFingerprint{
		IsSoftwareRenderer:    false,
		IsVirtualGPU:          false,
		ExtensionCount:        60,
		MaxTextureSize:        16384,
		MaxVertexAttribs:      16,
		UnmaskedRenderer:      "Intel UHD Graphics",
		PrecisionFormats:     []*PrecisionInfo{{PrecisionBits: 23}, {PrecisionBits: 23}},
		Max3DTextureSize:     2048,
		MaxCubeMapTextureSize: 8192,
	}

	score := analyzer.calculateUniquenessScore(fp)
	if score < 60 {
		t.Errorf("Expected uniqueness score > 60 for rich fingerprint, got %.2f", score)
	}
}

func TestFontFingerprintAnalyzer_Analyze(t *testing.T) {
	analyzer := NewFontFingerprintAnalyzer()

	data := map[string]interface{}{
		"font_hash":     "font_hash_123",
		"detected_fonts": []interface{}{"Arial", "Times New Roman", "Helvetica", "Roboto"},
		"font_metrics":  []interface{}{
			map[string]interface{}{"font_family": "Arial", "width": 100.5, "ascent": 80.0},
		},
	}

	result := analyzer.Analyze(data)

	if result.Hash != "font_hash_123" {
		t.Errorf("Expected hash 'font_hash_123', got '%s'", result.Hash)
	}
	if result.FontCount != 4 {
		t.Errorf("Expected FontCount 4, got %d", result.FontCount)
	}
	if len(result.DetectedFonts) != 4 {
		t.Errorf("Expected 4 detected fonts, got %d", len(result.DetectedFonts))
	}
	if result.UniquenessScore < 30 {
		t.Errorf("Expected uniqueness score >= 30, got %.2f", result.UniquenessScore)
	}
}

func TestFontFingerprintAnalyzer_DetectMissingFonts(t *testing.T) {
	analyzer := NewFontFingerprintAnalyzer()
	fp := &FontFingerprint{
		DetectedFonts: []string{"Arial", "Helvetica", "Roboto"},
	}

	analyzer.detectMissingFonts(fp)

	if len(fp.CommonFontsMissing) == 0 {
		t.Error("Expected some common fonts to be missing")
	}
}

func TestFontFingerprintAnalyzer_DetectRareFonts(t *testing.T) {
	analyzer := NewFontFingerprintAnalyzer()
	fp := &FontFingerprint{
		DetectedFonts: []string{"Arial", "Brush Script MT", "Papyrus", "Roboto"},
	}

	analyzer.detectRareFonts(fp)

	if len(fp.RareFonts) != 2 {
		t.Errorf("Expected 2 rare fonts, got %d", len(fp.RareFonts))
	}
}

func TestFontFingerprintAnalyzer_DetectSuspiciousFonts(t *testing.T) {
	analyzer := NewFontFingerprintAnalyzer()

	fp1 := &FontFingerprint{
		DetectedFonts: []string{"Arial", "Fake Font", "TestFont"},
	}
	analyzer.detectSuspiciousFonts(fp1)
	if !fp1.IsSuspiciousFontSet {
		t.Error("Expected suspicious font set detected")
	}

	fp2 := &FontFingerprint{
		DetectedFonts: []string{"Arial", "Times New Roman", "Roboto"},
	}
	analyzer.detectSuspiciousFonts(fp2)
	if fp2.IsSuspiciousFontSet {
		t.Error("Expected no suspicious fonts detected")
	}
}

func TestFontFingerprintAnalyzer_CalculateSimilarity(t *testing.T) {
	analyzer := NewFontFingerprintAnalyzer()

	fp1 := &FontFingerprint{
		Hash:           "font_hash_123",
		DetectedFonts: []string{"Arial", "Helvetica", "Roboto", "Times New Roman"},
		FontMetrics: []*FontMetricInfo{
			{FontFamily: "Arial", Width: 100, Ascent: 80},
		},
		MonospaceFonts: []string{"Courier"},
		SerifFonts:     []string{"Times New Roman"},
	}

	fp2 := &FontFingerprint{
		Hash:           "font_hash_123",
		DetectedFonts: []string{"Arial", "Helvetica", "Roboto", "Times New Roman"},
		FontMetrics: []*FontMetricInfo{
			{FontFamily: "Arial", Width: 100, Ascent: 80},
		},
		MonospaceFonts: []string{"Courier"},
		SerifFonts:     []string{"Times New Roman"},
	}

	sim := analyzer.CalculateSimilarity(fp1, fp2)
	if sim < 10 {
		t.Errorf("Expected similarity >= 10%% for identical fingerprints, got %.2f%%", sim)
	}
}

func TestFontFingerprintAnalyzer_CalculateFontListSimilarity(t *testing.T) {
	analyzer := NewFontFingerprintAnalyzer()

	fonts1 := []string{"Arial", "Helvetica", "Roboto"}
	fonts2 := []string{"Arial", "Helvetica", "Times New Roman"}

	sim := analyzer.calculateFontListSimilarity(fonts1, fonts2)
	expected := 2.0 / 4.0
	if math.Abs(sim-expected) > 0.01 {
		t.Errorf("Expected similarity ~%.2f, got %.2f", expected, sim)
	}
}

func TestAudioFingerprintAnalyzer_Analyze(t *testing.T) {
	analyzer := NewAudioFingerprintAnalyzer()

	data := map[string]interface{}{
		"audio_hash":                    "audio_hash_123",
		"audio_renders_count":           float64(3),
		"audio_variance":                0.05,
		"audio_non_zero_ratio":          0.8,
		"audio_peak_amplitude":          0.95,
		"audio_rms_level":               0.5,
		"audio_dynamic_range":          60.0,
		"audio_zero_crossing_rate":      0.01,
		"audio_spectral_centroid":      0.3,
		"audio_is_identical_across_renders": false,
		"audio_channels": []interface{}{
			map[string]interface{}{
				"channel_index": 0,
				"sample_count":  44100,
				"min_value":      -1.0,
				"max_value":      1.0,
				"mean_value":     0.0,
				"variance":       0.05,
			},
		},
	}

	result := analyzer.Analyze(data)

	if result.Hash != "audio_hash_123" {
		t.Errorf("Expected hash 'audio_hash_123', got '%s'", result.Hash)
	}
	if result.RendersCount != 3 {
		t.Errorf("Expected RendersCount 3, got %d", result.RendersCount)
	}
	if result.Variance != 0.05 {
		t.Errorf("Expected Variance 0.05, got %f", result.Variance)
	}
	if len(result.ChannelData) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(result.ChannelData))
	}
}

func TestAudioFingerprintAnalyzer_CalculateSimilarity(t *testing.T) {
	analyzer := NewAudioFingerprintAnalyzer()

	fp1 := &AudioFingerprint{
		Hash:             "audio_hash_123",
		Variance:        0.05,
		NonZeroRatio:    0.8,
		RMSLevel:        0.5,
		DynamicRange:    60.0,
		ChannelData: []*AudioChannelInfo{
			{ChannelIndex: 0, SampleCount: 44100, Variance: 0.05},
		},
	}

	fp2 := &AudioFingerprint{
		Hash:             "audio_hash_123",
		Variance:        0.051,
		NonZeroRatio:    0.79,
		RMSLevel:        0.49,
		DynamicRange:    59.5,
		ChannelData: []*AudioChannelInfo{
			{ChannelIndex: 0, SampleCount: 44100, Variance: 0.051},
		},
	}

	sim := analyzer.CalculateSimilarity(fp1, fp2)
	if sim < 20 {
		t.Errorf("Expected similarity >= 20%% for similar fingerprints, got %.2f%%", sim)
	}

	fp3 := &AudioFingerprint{
		Hash:          "audio_hash_456",
		Variance:     0.001,
		NonZeroRatio: 0.1,
		RMSLevel:     0.05,
		ChannelData:  []*AudioChannelInfo{},
	}

	sim2 := analyzer.CalculateSimilarity(fp1, fp3)
	if sim2 > sim {
		t.Errorf("Expected different fingerprints to have lower similarity")
	}
}

func TestAudioFingerprintAnalyzer_CalculateUniquenessScore(t *testing.T) {
	analyzer := NewAudioFingerprintAnalyzer()

	fp := &AudioFingerprint{
		IsIdenticalAcrossRenders: false,
		IsSuspiciouslySilent:     false,
		Variance:                0.05,
		NonZeroRatio:            0.8,
		DynamicRange:            60.0,
		ChannelData: []*AudioChannelInfo{
			{ChannelIndex: 0}, {ChannelIndex: 1},
		},
	}

	score := analyzer.calculateUniquenessScore(fp)
	if score < 60 {
		t.Errorf("Expected uniqueness score > 60 for rich audio fingerprint, got %.2f", score)
	}
}

func TestMediaFingerprintAnalyzer_Analyze(t *testing.T) {
	analyzer := NewMediaFingerprintAnalyzer()

	data := map[string]interface{}{
		"media_hash": "media_hash_123",
		"video_codecs": []interface{}{"video/webm", "video/mp4"},
		"audio_codecs": []interface{}{"audio/webm", "audio/mpeg"},
		"media_devices": []interface{}{
			map[string]interface{}{
				"device_id": "device_1",
				"label": "Built-in Camera",
				"kind": "videoinput",
				"group_id": "group_1",
			},
			map[string]interface{}{
				"device_id": "device_2",
				"label": "Built-in Microphone",
				"kind": "audioinput",
				"group_id": "group_2",
			},
		},
		"supported_formats": []interface{}{
			map[string]interface{}{
				"mime_type": "video/mp4",
				"has_video": true,
				"has_audio": false,
				"is_supported": true,
			},
		},
	}

	result := analyzer.Analyze(data)

	if result.Hash != "media_hash_123" {
		t.Errorf("Expected hash 'media_hash_123', got '%s'", result.Hash)
	}
	if result.DeviceCount != 2 {
		t.Errorf("Expected DeviceCount 2, got %d", result.DeviceCount)
	}
	if !result.HasCamera {
		t.Error("Expected HasCamera to be true")
	}
	if !result.HasMicrophone {
		t.Error("Expected HasMicrophone to be true")
	}
	if len(result.VideoCodecs) != 2 {
		t.Errorf("Expected 2 video codecs, got %d", len(result.VideoCodecs))
	}
}

func TestMediaFingerprintAnalyzer_CalculateSimilarity(t *testing.T) {
	analyzer := NewMediaFingerprintAnalyzer()

	fp1 := &MediaFingerprint{
		Hash: "media_hash_123",
		MediaDevices: []*MediaDeviceInfo{
			{Kind: "videoinput"},
			{Kind: "audioinput"},
		},
		VideoCodecs: []string{"video/webm", "video/mp4"},
		AudioCodecs: []string{"audio/webm", "audio/mpeg"},
		SupportedFormats: []*MediaFormatInfo{
			{MimeType: "video/mp4", IsSupported: true},
		},
	}

	fp2 := &MediaFingerprint{
		Hash: "media_hash_123",
		MediaDevices: []*MediaDeviceInfo{
			{Kind: "videoinput"},
			{Kind: "audioinput"},
		},
		VideoCodecs: []string{"video/webm", "video/mp4"},
		AudioCodecs: []string{"audio/webm", "audio/mpeg"},
		SupportedFormats: []*MediaFormatInfo{
			{MimeType: "video/mp4", IsSupported: true},
		},
	}

	sim := analyzer.CalculateSimilarity(fp1, fp2)
	if sim < 10 {
		t.Errorf("Expected similarity >= 10%% for identical fingerprints, got %.2f%%", sim)
	}

	fp3 := &MediaFingerprint{
		Hash: "media_hash_456",
		MediaDevices: []*MediaDeviceInfo{
			{Kind: "videoinput"},
		},
		VideoCodecs: []string{"video/webm"},
		AudioCodecs: []string{"audio/webm"},
	}

	sim2 := analyzer.CalculateSimilarity(fp1, fp3)
	if sim2 > sim {
		t.Errorf("Expected different fingerprints to have lower similarity")
	}
}

func TestMediaFingerprintAnalyzer_CalculateUniquenessScore(t *testing.T) {
	analyzer := NewMediaFingerprintAnalyzer()

	fp := &MediaFingerprint{
		DeviceCount:   4,
		HasCamera:     true,
		HasMicrophone: true,
		HasSpeaker:    true,
		VideoCodecs:   []string{"vp8", "vp9", "h264", "av1"},
		AudioCodecs:   []string{"opus", "vorbis", "mp4a"},
		VideoCapabilities: &VideoCapabilities{
			MaxWidth: 3840,
			MaxHeight: 2160,
		},
		AudioCapabilities: &AudioCapabilities{
			MaxChannelCount: 6,
		},
		DeviceLabels: []string{"Camera", "Microphone"},
	}

	score := analyzer.calculateUniquenessScore(fp)
	if score < 60 {
		t.Errorf("Expected uniqueness score > 60 for rich media fingerprint, got %.2f", score)
	}
}

func TestBrowserFingerprintMatcher_New(t *testing.T) {
	matcher := NewBrowserFingerprintMatcher()
	if matcher == nil {
		t.Fatal("Expected matcher to be created")
	}
	if len(matcher.knownBrowsers) == 0 {
		t.Error("Expected known browsers to be initialized")
	}
}

func TestBrowserFingerprintMatcher_Match(t *testing.T) {
	matcher := NewBrowserFingerprintMatcher()

	data := map[string]interface{}{
		"user_agent":   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"canvas_hash":  "chrome_canvas_hash",
		"webgl_renderer": "ANGLE (Intel, Intel(R) UHD Graphics Direct3D11)",
		"detected_fonts": []interface{}{"Segoe UI", "Arial", "Roboto"},
	}

	result := matcher.Match(data)

	if result.BestMatch == "" {
		t.Error("Expected a browser match")
	}
	if result.Confidence <= 0 {
		t.Error("Expected positive confidence")
	}
	if len(result.MatchedSignatures) == 0 {
		t.Error("Expected at least one matched signature")
	}
}

func TestBrowserFingerprintMatcher_MatchHeadless(t *testing.T) {
	matcher := NewBrowserFingerprintMatcher()

	data := map[string]interface{}{
		"user_agent":    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/120.0.0.0 Safari/537.36",
		"canvas_hash":   "swiftshader_canvas",
		"webgl_renderer": "SwiftShader",
		"detected_fonts": []interface{}{},
	}

	result := matcher.Match(data)

	if len(result.MatchedSignatures) == 0 {
		t.Error("Expected matches for headless browser")
	}
}

func TestBrowserFingerprintMatcher_MatchPuppeteer(t *testing.T) {
	matcher := NewBrowserFingerprintMatcher()

	data := map[string]interface{}{
		"user_agent":    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36 puppeteer",
		"canvas_hash":   "swiftshader_canvas",
		"webgl_renderer": "SwiftShader",
		"detected_fonts": []interface{}{},
	}

	result := matcher.Match(data)

	hasPuppeteer := false
	for _, sig := range result.MatchedSignatures {
		if sig.SignatureName == "Puppeteer" {
			hasPuppeteer = true
			break
		}
	}
	if !hasPuppeteer {
		t.Error("Expected Puppeteer signature match")
	}
}

func TestEnhancedFingerprintAnalyzer_AnalyzeEnhanced(t *testing.T) {
	analyzer := NewEnhancedFingerprintAnalyzer()

	data := map[string]interface{}{
		"canvas_hash":         "canvas_hash_123",
		"webgl_hash":          "webgl_hash_123",
		"webgl_vendor":        "Google Inc.",
		"webgl_renderer":      "ANGLE (Intel, Intel(R) UHD Graphics Direct3D11)",
		"font_hash":           "font_hash_123",
		"detected_fonts":      []interface{}{"Arial", "Helvetica", "Roboto"},
		"audio_hash":          "audio_hash_123",
		"media_hash":          "media_hash_123",
		"user_agent":          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0",
	}

	result := analyzer.AnalyzeEnhanced(data)

	if result.Timestamp == 0 {
		t.Error("Expected timestamp to be set")
	}
	if result.CanvasFingerprint == nil {
		t.Error("Expected CanvasFingerprint to be set")
	}
	if result.WebGLFingerprint == nil {
		t.Error("Expected WebGLFingerprint to be set")
	}
	if result.FontFingerprint == nil {
		t.Error("Expected FontFingerprint to be set")
	}
	if result.AudioFingerprint == nil {
		t.Error("Expected AudioFingerprint to be set")
	}
	if result.MediaFingerprint == nil {
		t.Error("Expected MediaFingerprint to be set")
	}
	if result.BrowserMatch == nil {
		t.Error("Expected BrowserMatch to be set")
	}
	if result.Fingerprint == "" {
		t.Error("Expected Fingerprint to be generated")
	}
	if result.CombinedUniquenessScore < 0 {
		t.Error("Expected non-negative CombinedUniquenessScore")
	}
	if result.EntropyBits < 0 {
		t.Error("Expected non-negative EntropyBits")
	}
}

func TestEnhancedFingerprintAnalyzer_CalculateFingerprintSimilarity(t *testing.T) {
	analyzer := NewEnhancedFingerprintAnalyzer()

	data1 := map[string]interface{}{
		"canvas_hash":         "canvas_hash_123",
		"webgl_hash":          "webgl_hash_123",
		"webgl_vendor":        "Google Inc.",
		"webgl_renderer":      "ANGLE (Intel, Intel(R) UHD Graphics Direct3D11)",
		"font_hash":           "font_hash_123",
		"detected_fonts":      []interface{}{"Arial", "Helvetica", "Roboto"},
		"audio_hash":          "audio_hash_123",
		"media_hash":          "media_hash_123",
	}

	data2 := map[string]interface{}{
		"canvas_hash":         "canvas_hash_123",
		"webgl_hash":          "webgl_hash_123",
		"webgl_vendor":        "Google Inc.",
		"webgl_renderer":      "ANGLE (Intel, Intel(R) UHD Graphics Direct3D11)",
		"font_hash":           "font_hash_123",
		"detected_fonts":      []interface{}{"Arial", "Helvetica", "Roboto"},
		"audio_hash":          "audio_hash_123",
		"media_hash":          "media_hash_123",
	}

	result := analyzer.CalculateFingerprintSimilarity(data1, data2)

	if result.OverallSimilarity < 10 {
		t.Errorf("Expected similarity >= 10%% for identical fingerprints, got %.2f%%", result.OverallSimilarity)
	}
	if len(result.Components) != 5 {
		t.Errorf("Expected 5 components, got %d", len(result.Components))
	}

	data3 := map[string]interface{}{
		"canvas_hash": "canvas_hash_456",
		"webgl_hash":  "webgl_hash_456",
		"font_hash":   "font_hash_456",
		"audio_hash":  "audio_hash_456",
		"media_hash":  "media_hash_456",
	}

	result2 := analyzer.CalculateFingerprintSimilarity(data1, data3)
	if result2.OverallSimilarity > result.OverallSimilarity {
		t.Errorf("Expected different fingerprints to have lower similarity")
	}
}

func TestEnhancedFingerprintAnalyzer_GenerateEnhancedFingerprint(t *testing.T) {
	analyzer := NewEnhancedFingerprintAnalyzer()

	result := &EnhancedFingerprintResult{
		CanvasFingerprint: &CanvasFingerprint{
			Hash: "canvas_test_hash_12345678",
		},
		WebGLFingerprint: &WebGLFingerprint{
			UnmaskedRenderer: "Intel UHD Graphics",
		},
		FontFingerprint: &FontFingerprint{
			DetectedFonts: []string{"Arial", "Helvetica"},
		},
		AudioFingerprint: &AudioFingerprint{
			Hash: "audio_test_hash_12345678",
		},
		MediaFingerprint: &MediaFingerprint{
			MediaDevices: []*MediaDeviceInfo{
				{Kind: "videoinput"},
				{Kind: "audioinput"},
			},
		},
	}

	fingerprint := analyzer.generateEnhancedFingerprint(result)
	if fingerprint == "" {
		t.Error("Expected fingerprint to be generated")
	}
	if len(fingerprint) < 16 {
		t.Errorf("Expected fingerprint length >= 16, got %d", len(fingerprint))
	}
}

func TestEnhancedFingerprintAnalyzer_CalculateCombinedUniquenessScore(t *testing.T) {
	analyzer := NewEnhancedFingerprintAnalyzer()

	result := &EnhancedFingerprintResult{
		CanvasFingerprint: &CanvasFingerprint{
			UniquenessScore: 70,
		},
		WebGLFingerprint: &WebGLFingerprint{
			UniquenessScore: 80,
		},
		FontFingerprint: &FontFingerprint{
			UniquenessScore: 60,
		},
		AudioFingerprint: &AudioFingerprint{
			UniquenessScore: 75,
		},
		MediaFingerprint: &MediaFingerprint{
			UniquenessScore: 65,
		},
		BrowserMatch: &BrowserMatchResult{
			BestMatch:  "Chrome on Windows",
			Confidence: 0.9,
		},
	}

	score := analyzer.calculateCombinedUniquenessScore(result)
	if score < 50 {
		t.Errorf("Expected combined uniqueness score > 50, got %.2f", score)
	}

	result2 := &EnhancedFingerprintResult{
		CanvasFingerprint:  &CanvasFingerprint{UniquenessScore: 70},
		WebGLFingerprint:   &WebGLFingerprint{UniquenessScore: 80},
		FontFingerprint:   &FontFingerprint{UniquenessScore: 60},
		AudioFingerprint:  &AudioFingerprint{UniquenessScore: 75},
		MediaFingerprint:  &MediaFingerprint{UniquenessScore: 65},
		BrowserMatch: &BrowserMatchResult{
			BestMatch:  "Headless Chrome",
			Confidence: 0.9,
		},
	}

	score2 := analyzer.calculateCombinedUniquenessScore(result2)
	if score2 >= score {
		t.Errorf("Expected headless browser to have lower score, got %.2f vs %.2f", score2, score)
	}
}

func TestEnhancedFingerprintAnalyzer_CalculateEntropyBits(t *testing.T) {
	analyzer := NewEnhancedFingerprintAnalyzer()

	result := &EnhancedFingerprintResult{
		CanvasFingerprint: &CanvasFingerprint{
			Hash: "canvas_hash_123",
		},
		WebGLFingerprint: &WebGLFingerprint{
			UniquenessScore: 60,
		},
		FontFingerprint: &FontFingerprint{
			FontCount: 15,
		},
		AudioFingerprint: &AudioFingerprint{
			IsIdenticalAcrossRenders: false,
		},
		MediaFingerprint: &MediaFingerprint{
			DeviceCount: 4,
		},
	}

	bits := analyzer.calculateEntropyBits(result)
	if bits < 40 {
		t.Errorf("Expected entropy bits > 40, got %.2f", bits)
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("toFloat64", func(t *testing.T) {
		val, ok := toFloat64(float64(1.5))
		if !ok || val != 1.5 {
			t.Errorf("Expected 1.5, got %f, ok=%v", val, ok)
		}

		val, ok = toFloat64(int(10))
		if !ok || val != 10 {
			t.Errorf("Expected 10, got %f, ok=%v", val, ok)
		}

		_, ok = toFloat64("invalid")
		if ok {
			t.Error("Expected toFloat64 to fail for string")
		}
	})

	t.Run("toInt", func(t *testing.T) {
		val, ok := toInt(float64(10))
		if !ok || val != 10 {
			t.Errorf("Expected 10, got %d, ok=%v", val, ok)
		}

		val, ok = toInt(int64(20))
		if !ok || val != 20 {
			t.Errorf("Expected 20, got %d, ok=%v", val, ok)
		}

		_, ok = toInt("invalid")
		if ok {
			t.Error("Expected toInt to fail for string")
		}
	})

	t.Run("average", func(t *testing.T) {
		avg := average([]float64{1, 2, 3, 4, 5})
		if avg != 3 {
			t.Errorf("Expected 3, got %f", avg)
		}

		avg = average([]float64{})
		if avg != 0 {
			t.Errorf("Expected 0 for empty slice, got %f", avg)
		}
	})

	t.Run("getStringValue", func(t *testing.T) {
		data := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		}

		val := getStringValue(data, "key1")
		if val != "value1" {
			t.Errorf("Expected 'value1', got '%s'", val)
		}

		val = getStringValue(data, "key2")
		if val != "" {
			t.Errorf("Expected empty string for non-string value, got '%s'", val)
		}

		val = getStringValue(data, "key3")
		if val != "" {
			t.Errorf("Expected empty string for missing key, got '%s'", val)
		}
	})

	t.Run("getStringArrayValue", func(t *testing.T) {
		data := map[string]interface{}{
			"key1": []interface{}{"a", "b", "c"},
			"key2": []interface{}{1, 2, 3},
			"key3": "not an array",
		}

		val := getStringArrayValue(data, "key1")
		if len(val) != 3 {
			t.Errorf("Expected 3 elements, got %d", len(val))
		}

		val = getStringArrayValue(data, "key2")
		if len(val) != 0 {
			t.Errorf("Expected 0 elements for non-string array, got %d", len(val))
		}

		val = getStringArrayValue(data, "key3")
		if len(val) != 0 {
			t.Errorf("Expected 0 elements for string, got %d", len(val))
		}
	})
}

func TestFingerprintSimilarityResult_CalculateOverallSimilarity(t *testing.T) {
	result := &FingerprintSimilarityResult{
		Components: []*ComponentSimilarity{
			{Name: "canvas", Score: 90, Weight: 0.25},
			{Name: "webgl", Score: 85, Weight: 0.25},
			{Name: "fonts", Score: 80, Weight: 0.20},
			{Name: "audio", Score: 95, Weight: 0.15},
			{Name: "media", Score: 70, Weight: 0.15},
		},
	}

	overall := result.calculateOverallSimilarity()
	expected := (90*0.25 + 85*0.25 + 80*0.20 + 95*0.15 + 70*0.15) / 1.0
	if math.Abs(overall-expected) > 0.01 {
		t.Errorf("Expected overall similarity ~%.2f, got %.2f", expected, overall)
	}

	result2 := &FingerprintSimilarityResult{
		Components: []*ComponentSimilarity{},
	}

	overall2 := result2.calculateOverallSimilarity()
	if overall2 != 0 {
		t.Errorf("Expected 0 for empty components, got %.2f", overall2)
	}
}

func TestPrecisionFormatExtraction(t *testing.T) {
	data := map[string]interface{}{
		"precision_formats": []interface{}{
			map[string]interface{}{
				"format":        "FLOAT",
				"type":          "HIGH_FLOAT",
				"range_min":     float64(127),
				"range_max":     float64(127),
				"precision_bits": float64(23),
			},
			map[string]interface{}{
				"format":        "FLOAT",
				"type":          "MEDIUM_FLOAT",
				"range_min":     float64(127),
				"range_max":     float64(127),
				"precision_bits": float64(10),
			},
		},
	}

	analyzer := NewWebGLFingerprintAnalyzer()
	fp := analyzer.Analyze(data)

	if len(fp.PrecisionFormats) != 2 {
		t.Errorf("Expected 2 precision formats, got %d", len(fp.PrecisionFormats))
	}

	if fp.PrecisionFormats[0].PrecisionBits != 23 {
		t.Errorf("Expected precision bits 23, got %d", fp.PrecisionFormats[0].PrecisionBits)
	}
}

func TestUniquenessScoreEdgeCases(t *testing.T) {
	t.Run("Canvas with empty data", func(t *testing.T) {
		analyzer := NewCanvasFingerprintAnalyzer()
		fp := &CanvasFingerprint{}

		score := analyzer.calculateUniquenessScore(fp)
		if score < 50 {
			t.Errorf("Expected base score >= 50, got %.2f", score)
		}
	})

	t.Run("WebGL with empty data", func(t *testing.T) {
		analyzer := NewWebGLFingerprintAnalyzer()
		fp := &WebGLFingerprint{}

		score := analyzer.calculateUniquenessScore(fp)
		if score < 30 {
			t.Errorf("Expected base score >= 30, got %.2f", score)
		}
	})

	t.Run("Font with empty data", func(t *testing.T) {
		analyzer := NewFontFingerprintAnalyzer()
		fp := &FontFingerprint{}

		score := analyzer.calculateUniquenessScore(fp)
		if score < 30 {
			t.Errorf("Expected base score >= 30, got %.2f", score)
		}
	})

	t.Run("Audio with empty data", func(t *testing.T) {
		analyzer := NewAudioFingerprintAnalyzer()
		fp := &AudioFingerprint{}

		score := analyzer.calculateUniquenessScore(fp)
		if score < 50 {
			t.Errorf("Expected base score >= 50, got %.2f", score)
		}
	})

	t.Run("Media with empty data", func(t *testing.T) {
		analyzer := NewMediaFingerprintAnalyzer()
		fp := &MediaFingerprint{}

		score := analyzer.calculateUniquenessScore(fp)
		if score < 30 {
			t.Errorf("Expected base score >= 30, got %.2f", score)
		}
	})
}

func TestCombinedUniquenessScoreWithNilComponents(t *testing.T) {
	analyzer := NewEnhancedFingerprintAnalyzer()

	result := &EnhancedFingerprintResult{}

	score := analyzer.calculateCombinedUniquenessScore(result)
	if score != 0 {
		t.Errorf("Expected 0 for empty result, got %.2f", score)
	}

	result2 := &EnhancedFingerprintResult{
		CanvasFingerprint: &CanvasFingerprint{},
	}

	score2 := analyzer.calculateCombinedUniquenessScore(result2)
	if score2 < 30 {
		t.Errorf("Expected score >= 30, got %.2f", score2)
	}
}
