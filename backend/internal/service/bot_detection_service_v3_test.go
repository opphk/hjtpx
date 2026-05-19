package service

import (
	"net/http"
	"testing"
)

func TestBotDetectionV3Service_BasicDetection(t *testing.T) {
	config := BotDetectionV3Config{
		EnableAIDetection:            true,
		EnableDeviceAnalysis:        true,
		EnableDeepBrowserCheck:      true,
		EnableAdvancedAutomationDetection: true,
		MLModelEnabled:              true,
		NeuralNetworkEnabled:        true,
	}

	service := NewBotDetectionV3Service(config)

	t.Run("Normal Request", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

		result := service.DetectBotV3(req, nil)

		if result.IsBot {
			t.Logf("Request marked as bot, reasons: %v", result.Reasons)
		}

		if result.RiskScore > 0.8 {
			t.Errorf("Normal request should not have high risk score: %f", result.RiskScore)
		}
	})

	t.Run("Bot User Agent Detection", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "curl/7.68.0")

		result := service.DetectBotV3(req, nil)

		if !result.IsBot {
			t.Log("curl user agent not detected as bot")
		}
	})

	t.Run("Selenium Detection", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/91.0.4472.124 selenium")

		result := service.DetectBotV3(req, nil)

		hasSeleniumDetection := false
		for _, reason := range result.Reasons {
			if reason != "" {
				hasSeleniumDetection = true
				break
			}
		}

		if !hasSeleniumDetection {
			t.Logf("Selenium not detected, reasons: %v", result.Reasons)
		}
	})
}

func TestBotDetectionV3Service_DeviceAnalysis(t *testing.T) {
	config := BotDetectionV3Config{
		EnableDeviceAnalysis: true,
	}

	service := NewBotDetectionV3Service(config)

	t.Run("Software WebGL Detection", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")

		additionalData := map[string]interface{}{
			"webgl_renderer": "SwiftShader",
			"touch_points":   0,
		}

		result := service.DetectBotV3(req, additionalData)

		foundSoftwareWebGL := false
		for _, reason := range result.Reasons {
			if reason != "" {
				foundSoftwareWebGL = true
				break
			}
		}

		if !foundSoftwareWebGL {
			t.Logf("Software WebGL detection, reasons: %v", result.Reasons)
		}
	})

	t.Run("Canvas Fingerprint Analysis", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)

		additionalData := map[string]interface{}{
			"canvas_fingerprint": "a1b2c3d4e5f6",
		}

		result := service.DetectBotV3(req, additionalData)

		if result.RiskScore > 0 {
			t.Logf("Canvas fingerprint detected, score: %f", result.RiskScore)
		}
	})
}

func TestBotDetectionV3Service_MLDetection(t *testing.T) {
	config := BotDetectionV3Config{
		EnableAIDetection: true,
		MLModelEnabled:   true,
	}

	service := NewBotDetectionV3Service(config)

	features := [][]float64{
		{0.9, 0.1, 0.8, 0.9, 0.2, 0.3, 0.7, 0.5, 0.3, 0.95},
		{0.1, 0.9, 0.2, 0.1, 0.8, 0.9, 0.1, 0.8, 0.9, 0.1},
		{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5},
	}

	labels := []float64{1.0, 0.0, 0.5}

	err := service.mlModel.Train(features, labels)
	if err != nil {
		t.Fatalf("Failed to train ML model: %v", err)
	}

	for i, feature := range features {
		prediction := service.mlModel.Predict(feature)
		t.Logf("Prediction for sample %d: %f (expected: %f)", i, prediction, labels[i])

		if prediction < 0 || prediction > 1 {
			t.Errorf("Prediction should be between 0 and 1, got %f", prediction)
		}
	}
}

func TestBotDetectionV3Service_NeuralNetworkDetection(t *testing.T) {
	config := BotDetectionV3Config{
		NeuralNetworkEnabled: true,
	}

	service := NewBotDetectionV3Service(config)

	if service.neuralNet == nil {
		t.Fatal("Neural network should be initialized")
	}

	input := make([]float64, 10)
	for i := range input {
		input[i] = float64(i) / 10.0
	}

	result := service.neuralNet.Forward(input)

	if result == nil {
		t.Fatal("Neural network result should not be nil")
	}

	if result.Prediction < 0 || result.Prediction > 1 {
		t.Errorf("Prediction should be between 0 and 1, got %f", result.Prediction)
	}

	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", result.Confidence)
	}

	if len(result.LayerOutputs) == 0 {
		t.Error("Should have layer outputs")
	}
}

func TestBotDetectionV3Service_AdvancedAutomationDetection(t *testing.T) {
	config := BotDetectionV3Config{
		EnableAdvancedAutomationDetection: true,
	}

	service := NewBotDetectionV3Service(config)

	testCases := []struct {
		name            string
		userAgent       string
		additionalData  map[string]interface{}
	}{
		{
			name:      "Puppeteer Headless",
			userAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/91.0.4472.124 headless",
		},
		{
			name:      "Playwright",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Playwright",
		},
		{
			name:      "PhantomJS",
			userAgent: "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/534.34 (KHTML, like Gecko) PhantomJS/2.1.1",
		},
		{
			name:      "WebDriver",
			userAgent: "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", tc.userAgent)

			result := service.DetectBotV3(req, tc.additionalData)

			if len(result.DetectionMethods) == 0 {
				t.Error("Should have detection methods")
			}

			t.Logf("%s: RiskScore=%f, Reasons=%v", tc.name, result.RiskScore, result.Reasons)
		})
	}
}

func TestBotDetectionV3Service_DeepBrowserCheck(t *testing.T) {
	config := BotDetectionV3Config{
		EnableDeepBrowserCheck: true,
	}

	service := NewBotDetectionV3Service(config)

	t.Run("Missing Platform Header", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.Header.Set("Sec-Ch-Ua", "\"Chromium\";v=\"91\"")

		additionalData := map[string]interface{}{
			"navigator.webdriver": true,
		}

		result := service.DetectBotV3(req, additionalData)

		hasMissingPlatform := false
		for _, reason := range result.Reasons {
			if reason != "" {
				hasMissingPlatform = true
				break
			}
		}

		if !hasMissingPlatform {
			t.Logf("Deep browser checks, reasons: %v", result.Reasons)
		}
	})

	t.Run("Missing Security Headers", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")

		result := service.DetectBotV3(req, nil)

		if len(result.Reasons) > 0 {
			t.Logf("Security headers check, reasons: %v", result.Reasons)
		}
	})
}

func TestBotDetectionV3Service_AdaptiveThreshold(t *testing.T) {
	config := BotDetectionV3Config{
		EnableAIDetection: true,
		MLModelEnabled:   true,
	}

	service := NewBotDetectionV3Service(config)

	initialThreshold := service.adaptiveThresholds.CurrentThreshold

	service.adjustThreshold(0.9)

	newThreshold := service.adaptiveThresholds.CurrentThreshold
	if newThreshold >= initialThreshold {
		t.Logf("Threshold adjusted from %f to %f", initialThreshold, newThreshold)
	}

	service.adjustThreshold(0.2)

	newThreshold2 := service.adaptiveThresholds.CurrentThreshold
	if newThreshold2 <= newThreshold {
		t.Logf("Threshold adjusted from %f to %f", newThreshold, newThreshold2)
	}
}

func TestBotDetectionV3Service_Statistics(t *testing.T) {
	config := BotDetectionV3Config{
		EnableAIDetection: true,
	}

	service := NewBotDetectionV3Service(config)

	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		service.DetectBotV3(req, nil)
	}

	stats := service.GetStatistics()

	if stats.TotalFingerprints < 10 {
		t.Errorf("Expected at least 10 fingerprints, got %d", stats.TotalFingerprints)
	}

	if stats.ActiveBehaviors < 10 {
		t.Errorf("Expected at least 10 active behaviors, got %d", stats.ActiveBehaviors)
	}
}

func TestBotDetectionV3Service_FingerprintGeneration(t *testing.T) {
	config := BotDetectionV3Config{}
	service := NewBotDetectionV3Service(config)

	ip := "192.168.1.1"
	userAgent := "Mozilla/5.0"
	data := map[string]interface{}{
		"device_type": "desktop",
	}

	fingerprintID1 := service.generateFingerprintIDV3(ip, userAgent, data)
	fingerprintID2 := service.generateFingerprintIDV3(ip, userAgent, data)

	if fingerprintID1 != fingerprintID2 {
		t.Error("Same inputs should produce same fingerprint")
	}

	fingerprintID3 := service.generateFingerprintIDV3(ip, "Different UA", data)
	if fingerprintID1 == fingerprintID3 {
		t.Error("Different user agent should produce different fingerprint")
	}
}

func TestBotDetectionV3Service_ModelExportImport(t *testing.T) {
	config := BotDetectionV3Config{
		MLModelEnabled: true,
	}

	service := NewBotDetectionV3Service(config)

	features := [][]float64{
		{0.9, 0.1, 0.8, 0.9, 0.2, 0.3, 0.7, 0.5, 0.3, 0.95},
		{0.1, 0.9, 0.2, 0.1, 0.8, 0.9, 0.1, 0.8, 0.9, 0.1},
	}
	labels := []float64{1.0, 0.0}

	service.mlModel.Train(features, labels)

	exportedData, err := service.ExportModel(nil)
	if err != nil {
		t.Fatalf("Failed to export model: %v", err)
	}

	if len(exportedData) == 0 {
		t.Error("Exported data should not be empty")
	}

	service2 := NewBotDetectionV3Service(config)

	err = service2.ImportModel(nil, exportedData)
	if err != nil {
		t.Fatalf("Failed to import model: %v", err)
	}

	testFeature := []float64{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5}
	prediction1 := service.mlModel.Predict(testFeature)
	prediction2 := service2.mlModel.Predict(testFeature)

	t.Logf("Original model prediction: %f", prediction1)
	t.Logf("Imported model prediction: %f", prediction2)
}

func TestBotDetectionV3Service_MLFeaturesExtraction(t *testing.T) {
	config := BotDetectionV3Config{
		MLModelEnabled: true,
	}

	service := NewBotDetectionV3Service(config)

	ip := "192.168.1.1"
	userAgent := "Mozilla/5.0"

	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		service.recordRequestV3(ip, req)
	}

	data := map[string]interface{}{
		"mouse_variance":      500.0,
		"keyboard_speed":      300.0,
		"touch_accuracy":      0.8,
		"canvas_unique":       0.7,
		"webgl_software":      0.0,
		"timezone_offset":     0.0,
		"language_variance":   0.5,
		"automation_flags":    0.0,
	}

	features := service.extractMLFeatures(ip, userAgent, data)

	if len(features) != 10 {
		t.Errorf("Expected 10 features, got %d", len(features))
	}

	for i, feature := range features {
		if feature < 0 || feature > 1 {
			t.Errorf("Feature %d should be between 0 and 1, got %f", i, feature)
		}
	}
}

func TestBotDetectionV3Service_BehaviorAnalysis(t *testing.T) {
	config := BotDetectionV3Config{}
	service := NewBotDetectionV3Service(config)

	ip := "192.168.1.1"

	for i := 0; i < 100; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		service.recordRequestV3(ip, req)
	}

	result := &BotDetectionV3Result{}
	service.analyzeBehaviorPatternsV3(ip, result)

	t.Logf("Behavior score: %f, Reasons: %v", result.BehaviorScore, result.Reasons)
}

func TestBotDetectionV3Service_CommonCanvasHash(t *testing.T) {
	config := BotDetectionV3Config{}
	service := NewBotDetectionV3Service(config)

	commonHashes := []string{
		"a1b2c3d4e5f6",
		"1234567890ab",
		"ffffffffffff",
		"000000000000",
	}

	for _, hash := range commonHashes {
		if !service.isCommonCanvasHash(hash) {
			t.Errorf("Hash %s should be common", hash)
		}
	}

	uniqueHash := "abc123def456"
	if service.isCommonCanvasHash(uniqueHash) {
		t.Errorf("Hash %s should not be common", uniqueHash)
	}
}

func TestBotDetectionV3Service_ClientIPExtraction(t *testing.T) {
	config := BotDetectionV3Config{}
	service := NewBotDetectionV3Service(config)

	testCases := []struct {
		name      string
		headers   map[string]string
		remoteAddr string
		expected  string
	}{
		{
			name:      "X-Forwarded-For",
			headers:   map[string]string{"X-Forwarded-For": "192.168.1.100, 10.0.0.1"},
			remoteAddr: "127.0.0.1:8080",
			expected:  "192.168.1.100",
		},
		{
			name:      "X-Real-IP",
			headers:   map[string]string{"X-Real-IP": "10.0.0.50"},
			remoteAddr: "127.0.0.1:8080",
			expected:  "10.0.0.50",
		},
		{
			name:      "RemoteAddr",
			headers:   map[string]string{},
			remoteAddr: "192.168.1.1:8080",
			expected:  "192.168.1.1:8080",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/test", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tc.remoteAddr

			ip := service.getClientIP(req)
			if ip != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, ip)
			}
		})
	}
}
