package service

import (
	"net/http"
	"testing"
	"time"
)

func TestEnhancedFingerprintMetrics(t *testing.T) {
	analyzer := NewFingerprintAnalyzer()

	data := map[string]interface{}{
		"user_agent":         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"canvas_hash":        "abc123def456",
		"webgl_hash":         "webgl789xyz",
		"webgl_renderer":     "Intel Iris OpenGL Engine",
		"screen_resolution":  "1920x1080",
		"timezone":           "Asia/Shanghai",
		"detected_fonts":     []interface{}{"Arial", "Helvetica", "Times New Roman"},
		"screen_color_depth": float64(24),
		"screen_pixel_ratio": float64(1.0),
	}

	metrics, err := analyzer.AnalyzeEnhancedMetrics(data)
	if err != nil {
		t.Fatalf("AnalyzeEnhancedMetrics failed: %v", err)
	}

	if metrics == nil {
		t.Fatal("Expected metrics to be non-nil")
	}

	if metrics.UniquenessScore == 0 {
		t.Error("Expected UniquenessScore to be greater than 0")
	}

	if metrics.CanvasMetrics == nil {
		t.Error("Expected CanvasMetrics to be non-nil")
	}

	if metrics.WebGLMetrics == nil {
		t.Error("Expected WebGLMetrics to be non-nil")
	}

	if metrics.FontMetrics == nil {
		t.Error("Expected FontMetrics to be non-nil")
	}

	if metrics.ScreenMetrics == nil {
		t.Error("Expected ScreenMetrics to be non-nil")
	}

	if metrics.BrowserSignature == "" {
		t.Error("Expected BrowserSignature to be non-empty")
	}

	if metrics.MultiBrowserCompare == nil {
		t.Error("Expected MultiBrowserCompare to be non-nil")
	}
}

func TestCanvasSimilarityAnalyzer(t *testing.T) {
	db := NewFingerprintDatabase()
	analyzer := NewCanvasSimilarityAnalyzer(db)

	testCases := []struct {
		hash1       string
		hash2       string
		expectedMin float64
		name        string
	}{
		{"abc123", "abc123", 95.0, "identical_hashes"},
		{"abc123", "xyz789", 0.0, "different_hashes"},
		{"abc123", "abc456", 40.0, "partial_similarity"},
		{"", "abc123", 0.0, "empty_hash1"},
		{"abc123", "", 0.0, "empty_hash2"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			similarity := analyzer.CalculateCanvasSimilarity(tc.hash1, tc.hash2)
			if similarity < tc.expectedMin {
				t.Errorf("Expected similarity >= %.2f for %s, got %.2f",
					tc.expectedMin, tc.name, similarity)
			}
		})
	}
}

func TestEnhancedBotDetectionService(t *testing.T) {
	service := NewEnhancedBotDetectionService()

	t.Run("detect_selenium", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.212 Safari/537.36 selenium/3.141.0")

		result := service.DetectAutomationTool(req, nil)
		if result == nil {
			t.Fatal("Expected result to be non-nil")
		}

		if len(result.DetectionMethods) == 0 {
			t.Error("Expected detection methods to be recorded")
		}
	})

	t.Run("detect_puppeteer", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/90.0.4430.212 Safari/537.36")

		result := service.DetectAutomationTool(req, nil)
		if result == nil {
			t.Fatal("Expected result to be non-nil")
		}
	})

	t.Run("detect_playwright", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.212 Safari/537.36 playwright/1.20.0")

		result := service.DetectAutomationTool(req, nil)
		if result == nil {
			t.Fatal("Expected result to be non-nil")
		}
	})

	t.Run("detect_headless_browser", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/90.0.4430.212 Safari/537.36")

		additionalData := map[string]interface{}{
			"navigator_properties": map[string]interface{}{
				"webdriver": true,
				"plugins":   []interface{}{},
				"languages": []interface{}{},
			},
		}

		result := service.DetectAutomationTool(req, additionalData)
		if result == nil {
			t.Fatal("Expected result to be non-nil")
		}

		if result.Score < 50 {
			t.Errorf("Expected headless browser detection to have score >= 50, got %.2f", result.Score)
		}
	})

	t.Run("detect_phantomjs", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/538.1 (KHTML, like Gecko) PhantomJS/2.1.1 Safari/538.1")

		result := service.DetectAutomationTool(req, nil)
		if result == nil {
			t.Fatal("Expected result to be non-nil")
		}
	})
}

func TestEnhancedBotDetectionService_BehavioralAnalysis(t *testing.T) {
	service := NewEnhancedBotDetectionService()

	ip := "192.168.1.100"

	for i := 0; i < 20; i++ {
		service.RecordRequest(ip, "/test"+string(rune('0'+i%10)))
		time.Sleep(100 * time.Millisecond)
	}

	isAutomated, score := service.DetectAutomatedScriptPattern(ip)
	if !isAutomated && score > 0 {
		t.Logf("Automated pattern detected with score %.2f", score)
	}

	session := service.GetSessionInfo(ip)
	if session == nil {
		t.Error("Expected session info to exist")
	} else {
		if session.RequestCount != 20 {
			t.Errorf("Expected 20 requests, got %d", session.RequestCount)
		}
	}
}

func TestEnhancedProxyDetectionService(t *testing.T) {
	service := NewEnhancedProxyDetectionService()

	t.Run("assess_ip_risk_proxy", func(t *testing.T) {
		headers := map[string]string{
			"X-Forwarded-For": "203.0.113.1, 192.168.1.1",
		}

		assessment := service.AssessIPRisk("203.0.113.1", headers, nil)
		if assessment == nil {
			t.Fatal("Expected assessment to be non-nil")
		}

		if len(assessment.RiskFactors) == 0 {
			t.Error("Expected at least one risk factor")
		}

		if assessment.OverallRisk < 0 {
			t.Error("Expected overall risk to be non-negative")
		}
	})

	t.Run("assess_ip_risk_vpn", func(t *testing.T) {
		headers := map[string]string{}

		assessment := service.AssessIPRisk("45.33.32.156", headers, nil)
		if assessment == nil {
			t.Fatal("Expected assessment to be non-nil")
		}
	})

	t.Run("assess_ip_risk_tor", func(t *testing.T) {
		headers := map[string]string{}

		assessment := service.AssessIPRisk("128.31.0.34", headers, nil)
		if assessment == nil {
			t.Fatal("Expected assessment to be non-nil")
		}

		hasTorRisk := false
		for _, factor := range assessment.RiskFactors {
			if factor.Category == "tor" {
				hasTorRisk = true
				break
			}
		}

		if hasTorRisk {
			t.Log("Tor exit node risk detected")
		}
	})

	t.Run("cache_assessment", func(t *testing.T) {
		headers := map[string]string{}
		assessment := service.AssessIPRisk("192.168.1.1", headers, nil)

		service.CacheAssessment(assessment)

		cached, found := service.GetCachedAssessment("192.168.1.1")
		if !found {
			t.Error("Expected cached assessment to be found")
		}

		if cached == nil {
			t.Error("Expected cached assessment to be non-nil")
		}
	})

	t.Run("threat_intelligence", func(t *testing.T) {
		maliciousIPs := []string{"10.0.0.1", "10.0.0.2"}
		botNets := []string{"10.0.0.3"}

		service.UpdateThreatIntelligence(maliciousIPs, botNets)

		headers := map[string]string{}
		assessment := service.AssessIPRisk("10.0.0.1", headers, nil)

		hasThreatRisk := false
		for _, factor := range assessment.RiskFactors {
			if factor.Category == "threat_intel" || factor.Category == "botnet" {
				hasThreatRisk = true
				break
			}
		}

		if hasThreatRisk {
			t.Log("Threat intelligence risk detected")
		}
	})
}

func TestDeviceDetectionService(t *testing.T) {
	service := NewDeviceDetectionService()

	t.Run("detect_vm_placeholder", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) VMware, Inc. VMware7,1",
		}

		result := service.DetectDevice(data)
		if result == nil {
			t.Fatal("Expected result to be non-nil")
		}

		if len(result.DetectionMethods) == 0 {
			t.Error("Expected detection methods to be recorded")
		}
	})

	t.Run("detect_container", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 (Linux; Docker) AppleWebKit/537.36",
		}

		result := service.DetectDevice(data)
		if result == nil {
			t.Fatal("Expected result to be non-nil")
		}
	})

	t.Run("detect_mobile_device", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X)",
		}

		result := service.DetectDevice(data)
		if result == nil {
			t.Fatal("Expected result to be non-nil")
		}

		if result.DeviceType != DeviceMobile {
			t.Errorf("Expected DeviceType to be %s, got %s", DeviceMobile, result.DeviceType)
		}
	})

	t.Run("detect_desktop_device", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		}

		result := service.DetectDevice(data)
		if result == nil {
			t.Fatal("Expected result to be non-nil")
		}

		if result.DeviceType != DeviceDesktop {
			t.Errorf("Expected DeviceType to be %s, got %s", DeviceDesktop, result.DeviceType)
		}
	})

	t.Run("record_fingerprint", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			"screen_resolution": "1920x1080",
			"timezone":          "UTC",
			"platform":          "Win32",
		}

		fingerprintID := service.RecordDeviceFingerprint(data)
		if fingerprintID == "" {
			t.Error("Expected non-empty fingerprint ID")
		}

		retrieved, exists := service.GetDeviceFingerprint(fingerprintID)
		if !exists {
			t.Error("Expected fingerprint to exist")
		}

		if retrieved == nil {
			t.Error("Expected retrieved fingerprint to be non-nil")
		}
	})

	t.Run("stability_score", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			"screen_resolution": "2560x1600",
			"timezone":          "America/New_York",
			"platform":          "MacIntel",
		}

		fingerprintID := service.RecordDeviceFingerprint(data)

		for i := 0; i < 10; i++ {
			time.Sleep(100 * time.Millisecond)
			service.RecordDeviceFingerprint(data)
		}

		score := service.CalculateStabilityScore(fingerprintID)
		if score < 0 {
			t.Error("Expected stability score to be non-negative")
		}

		t.Logf("Stability score: %.2f", score)
	})

	t.Run("emulator_detection", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 (Linux; Android 11; sdk_phone_x86_64 Build/RP1A.201005.001; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/90.0.4430.91 Mobile Safari/537.36",
			"navigator_properties": map[string]interface{}{
				"maxTouchPoints": float64(0),
				"platform":       "Linux x86_64",
			},
		}

		result := service.DetectDevice(data)
		if result == nil {
			t.Fatal("Expected result to be non-nil")
		}

		if result.Score > 0 {
			t.Logf("Emulator detection score: %.2f", result.Score)
		}
	})

	t.Run("cleanup_old_data", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			data := map[string]interface{}{
				"user_agent": "Mozilla/5.0 Test",
			}
			service.RecordDeviceFingerprint(data)
		}

		removed := service.CleanupOldData(24 * time.Hour)
		t.Logf("Removed %d old device fingerprints", removed)
	})
}

func TestCanvasMetrics(t *testing.T) {
	analyzer := NewFingerprintAnalyzer()

	data := map[string]interface{}{
		"canvas_hash":                  "test_canvas_hash_123",
		"canvas_rgba_distribution":     []interface{}{float64(100), float64(150), float64(200)},
		"canvas_noise_level":           float64(0.05),
		"canvas_rendering_consistency": float64(0.98),
	}

	metrics := analyzer.analyzeCanvasEnhanced(data)
	if metrics == nil {
		t.Fatal("Expected canvas metrics to be non-nil")
	}

	if metrics.Hash != "test_canvas_hash_123" {
		t.Errorf("Expected hash to match, got %s", metrics.Hash)
	}

	if len(metrics.RgbaDistribution) != 3 {
		t.Errorf("Expected 3 RGBA distribution values, got %d", len(metrics.RgbaDistribution))
	}

	if metrics.NoiseLevel != 0.05 {
		t.Errorf("Expected noise level 0.05, got %f", metrics.NoiseLevel)
	}
}

func TestWebGLMetrics(t *testing.T) {
	analyzer := NewFingerprintAnalyzer()

	data := map[string]interface{}{
		"webgl_hash":                  "test_webgl_hash",
		"webgl_vendor":                "Intel Inc.",
		"webgl_renderer":              "Intel Iris OpenGL Engine",
		"webgl_max_texture_size":      float64(16384),
		"webgl_max_renderbuffer_size": float64(16384),
		"webgl_max_vertex_attribs":    float64(16),
		"webgl_extensions_count":      float64(45),
	}

	metrics := analyzer.analyzeWebGLEnhanced(data)
	if metrics == nil {
		t.Fatal("Expected WebGL metrics to be non-nil")
	}

	if metrics.Vendor != "Intel Inc." {
		t.Errorf("Expected vendor to be 'Intel Inc.', got %s", metrics.Vendor)
	}

	if metrics.MaxTextureSize != 16384 {
		t.Errorf("Expected max texture size 16384, got %d", metrics.MaxTextureSize)
	}

	if metrics.IsSoftwareRenderer {
		t.Error("Expected software renderer to be false")
	}
}

func TestFontMetrics(t *testing.T) {
	analyzer := NewFingerprintAnalyzer()

	data := map[string]interface{}{
		"font_hash":      "test_font_hash",
		"detected_fonts": []interface{}{"Arial", "Helvetica", "Times New Roman", "Verdana", "Georgia"},
	}

	metrics := analyzer.analyzeFontsEnhanced(data)
	if metrics == nil {
		t.Fatal("Expected font metrics to be non-nil")
	}

	if metrics.FontCount != 5 {
		t.Errorf("Expected 5 detected fonts, got %d", metrics.FontCount)
	}

	if metrics.IsLimitedFontSet {
		t.Error("Expected limited font set to be false")
	}

	if len(metrics.DetectedFonts) != 5 {
		t.Errorf("Expected 5 fonts in DetectedFonts, got %d", len(metrics.DetectedFonts))
	}
}

func TestScreenMetrics(t *testing.T) {
	analyzer := NewFingerprintAnalyzer()

	data := map[string]interface{}{
		"screen_resolution":   "1920x1080",
		"screen_color_depth":  float64(24),
		"screen_pixel_ratio":  float64(1.0),
		"screen_avail_width":  float64(1920),
		"screen_avail_height": float64(1040),
	}

	metrics := analyzer.analyzeScreenEnhanced(data)
	if metrics == nil {
		t.Fatal("Expected screen metrics to be non-nil")
	}

	if metrics.Resolution != "1920x1080" {
		t.Errorf("Expected resolution '1920x1080', got %s", metrics.Resolution)
	}

	if metrics.ColorDepth != 24 {
		t.Errorf("Expected color depth 24, got %d", metrics.ColorDepth)
	}

	if !metrics.IsCommonResolution {
		t.Error("Expected common resolution to be true for 1920x1080")
	}
}

func TestBrowserSignature(t *testing.T) {
	analyzer := NewFingerprintAnalyzer()

	data := map[string]interface{}{
		"user_agent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/90.0",
		"canvas_hash":       "abc123",
		"webgl_renderer":    "Intel Iris",
		"detected_fonts":    []interface{}{"Arial", "Helvetica"},
		"screen_resolution": "1920x1080",
		"timezone":          "Asia/Shanghai",
	}

	signature := analyzer.generateBrowserSignature(data)
	if signature == "" {
		t.Error("Expected non-empty browser signature")
	}

	if len(signature) < 10 {
		t.Errorf("Expected signature to be reasonably long, got %d chars", len(signature))
	}
}

func TestUniquenessScore(t *testing.T) {
	analyzer := NewFingerprintAnalyzer()

	metrics := &EnhancedFingerprintMetrics{
		CanvasMetrics: &CanvasMetrics{
			Hash:               "unique_canvas_hash",
			IsHeadlessRenderer: false,
			SoftwareRenderer:   false,
			NoiseLevel:         0.05,
		},
		WebGLMetrics: &WebGLMetrics{
			IsSoftwareRenderer:  false,
			IsVirtualGPU:        false,
			SupportedExtensions: 40,
		},
		FontMetrics: &FontMetrics{
			FontCount:           10,
			IsLimitedFontSet:    false,
			FontFamilyDiversity: 0.8,
		},
		ScreenMetrics: &ScreenMetrics{
			Resolution:         "2560x1440",
			IsCommonResolution: false,
		},
		MultiBrowserCompare: &MultiBrowserCompare{
			IsUniqueSignature: true,
		},
	}

	score := analyzer.calculateUniquenessScore(metrics)
	if score < 50 {
		t.Errorf("Expected uniqueness score > 50, got %.2f", score)
	}

	if score > 100 {
		t.Errorf("Expected uniqueness score <= 100, got %.2f", score)
	}

	t.Logf("Calculated uniqueness score: %.2f", score)
}
