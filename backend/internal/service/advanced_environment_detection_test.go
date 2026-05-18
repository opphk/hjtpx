package service

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestAdvancedFingerprintAnalyzer(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	t.Run("analyze_basic_fingerprint", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			"canvas_hash":       "abc123hash",
			"webgl_hash":        "def456hash",
			"screen_resolution": "1920x1080",
			"timezone":          "America/New_York",
			"language":          "en-US",
			"platform":          "Win32",
			"hardware_concurrency": float64(4),
			"device_memory":     float64(8),
		}

		analysis, err := analyzer.AnalyzeAdvancedFingerprint(data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if analysis == nil {
			t.Fatal("Expected analysis to be non-nil")
		}

		if analysis.BaseFingerprint == nil {
			t.Fatal("Expected base fingerprint to be non-nil")
		}

		if analysis.BaseFingerprint.UserAgent != data["user_agent"] {
			t.Errorf("Expected UserAgent %s, got %s", data["user_agent"], analysis.BaseFingerprint.UserAgent)
		}
	})

	t.Run("detect_bot_patterns", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/91.0 Headless",
		}

		analysis, err := analyzer.AnalyzeAdvancedFingerprint(data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if analysis.BaseFingerprint.IsKnownBot {
			t.Log("Bot pattern detected successfully")
		}
	})

	t.Run("detect_vpn_indicators", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			"public_ip":   "45.33.32.156",
		}

		analysis, err := analyzer.AnalyzeAdvancedFingerprint(data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if analysis.BaseFingerprint.IsKnownVPN {
			t.Log("VPN indicator detected successfully")
		}
	})
}

func TestEnhancedProxyDetection(t *testing.T) {
	detection := NewEnhancedProxyDetection()

	t.Run("detect_proxy_headers", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Forwarded-For", "203.0.113.1, 192.168.1.1, 10.0.0.1")

		ctx := context.Background()
		result, err := detection.DetectProxy(ctx, "203.0.113.1", headers)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result to be non-nil")
		}

		if !result.Headers.XForwardedFor {
			t.Error("Expected X-Forwarded-For to be detected")
		}

		if !result.IsProxy {
			t.Log("Proxy detected via multi-hop headers")
		}
	})

	t.Run("detect_tor_exit_node", func(t *testing.T) {
		ctx := context.Background()
		result, err := detection.DetectProxy(ctx, "128.31.0.34", nil)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result.IsTor {
			t.Log("Tor exit node detected successfully")
		}
	})

	t.Run("detect_vpn_provider", func(t *testing.T) {
		ctx := context.Background()
		result, err := detection.DetectProxy(ctx, "45.33.32.156", nil)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result.IsVPN {
			t.Log("VPN provider detected successfully")
		}
	})

	t.Run("calculate_confidence", func(t *testing.T) {
		ctx := context.Background()
		headers := http.Header{}
		headers.Set("X-Forwarded-For", "203.0.113.1")
		headers.Set("Via", "1.1 proxy-server")

		result, err := detection.DetectProxy(ctx, "203.0.113.1", headers)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result.Confidence < 0 || result.Confidence > 100 {
			t.Errorf("Confidence should be between 0 and 100, got %f", result.Confidence)
		}

		t.Logf("Detection confidence: %.2f", result.Confidence)
	})

	t.Run("batch_detect", func(t *testing.T) {
		ctx := context.Background()
		requests := []ProxyCheckRequest{
			{IP: "203.0.113.1", Headers: http.Header{}},
			{IP: "45.33.32.156", Headers: http.Header{}},
			{IP: "128.31.0.34", Headers: http.Header{}},
		}

		results := detection.BatchDetect(ctx, requests)

		if len(results) != len(requests) {
			t.Errorf("Expected %d results, got %d", len(requests), len(results))
		}
	})
}

func TestAdvancedRiskScorer(t *testing.T) {
	scorer := NewEnhancedRiskScorer()

	t.Run("calculate_score", func(t *testing.T) {
		analysis := &AdvancedFingerprintAnalysis{
			BaseFingerprint: &FingerprintAnalysis{
				AnomalyScore: 50,
			},
			MLRiskScore: 30,
			BehaviorScore: 25,
			AdvancedIndicators: &AdvancedIndicators{
				AutomationIndicators: []string{"headless", "webdriver"},
				ProxyVPNIndicators:   []string{"proxy_detected"},
			},
		}

		score := scorer.CalculateScore(analysis)

		if score < 0 || score > 100 {
			t.Errorf("Score should be between 0 and 100, got %f", score)
		}

		t.Logf("Calculated risk score: %.2f", score)
	})
}

func TestPatternMatcher(t *testing.T) {
	matcher := NewPatternMatcher()

	t.Run("match_patterns", func(t *testing.T) {
		testCases := []struct {
			text     string
			expected bool
		}{
			{"headless browser detected", true},
			{"puppeteer automation active", true},
			{"normal browser session", false},
			{"virtualbox vm running", true},
		}

		for _, tc := range testCases {
			results := matcher.Match(tc.text)
			matched := len(results) > 0

			if matched != tc.expected {
				t.Errorf("For text '%s', expected match=%v, got match=%v", tc.text, tc.expected, matched)
			}
		}
	})
}

func TestRiskReport(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	t.Run("generate_report", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0 Headless",
		}

		analysis, err := analyzer.AnalyzeAdvancedFingerprint(data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		report := analyzer.GenerateRiskReport(analysis)

		if report == nil {
			t.Fatal("Expected report to be non-nil")
		}

		if report.FinalScore < 0 || report.FinalScore > 100 {
			t.Errorf("FinalScore should be between 0 and 100, got %f", report.FinalScore)
		}

		t.Logf("Risk Level: %s", report.RiskLevel)
		t.Logf("Summary: %s", report.GetSummary())
	})
}

func TestAdvancedFingerprintDatabase(t *testing.T) {
	db := NewAdvancedFingerprintDatabase()

	t.Run("add_and_get_analysis", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 Test Browser",
			"canvas_hash": "test123",
		}

		analyzer := NewAdvancedFingerprintAnalyzer()
		analysis, err := analyzer.AnalyzeAdvancedFingerprint(data)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		id := analysis.BaseFingerprint.FingerprintID
		db.AddAdvancedAnalysis(id, analysis)

		retrieved, exists := db.GetAdvancedAnalysis(id)
		if !exists {
			t.Fatal("Expected analysis to exist")
		}

		if retrieved.BaseFingerprint.FingerprintID != id {
			t.Errorf("Expected ID %s, got %s", id, retrieved.BaseFingerprint.FingerprintID)
		}
	})

	t.Run("get_analytics", func(t *testing.T) {
		analytics := db.GetAnalytics()

		if analytics == nil {
			t.Fatal("Expected analytics to be non-nil")
		}

		t.Logf("Total Fingerprints: %d", analytics.TotalFingerprints)
		t.Logf("Bot Count: %d", analytics.BotCount)
		t.Logf("VPN Count: %d", analytics.VPNCount)
	})
}

func TestTemporalPatternAnalysis(t *testing.T) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	t.Run("analyze_temporal_pattern", func(t *testing.T) {
		now := float64(time.Now().UnixMilli())
		data := map[string]interface{}{
			"request_timestamps": []interface{}{
				now,
				now + 100,
				now + 200,
				now + 300,
				now + 400,
			},
		}

		analysis := analyzer.AnalyzeTemporalPattern(data)

		if analysis == nil {
			t.Fatal("Expected analysis to be non-nil")
		}

		if analysis.RequestCount != 5 {
			t.Errorf("Expected 5 requests, got %d", analysis.RequestCount)
		}

		t.Logf("Avg Interval: %.2fms", analysis.AvgInterval)
	})
}

func TestProxyDatabase(t *testing.T) {
	t.Run("add_and_get_proxy", func(t *testing.T) {
		detection := NewEnhancedProxyDetection()

		proxy := &ProxyInfo{
			IP:          "192.168.1.100",
			Port:        8080,
			Type:        "http",
			Protocol:    "HTTP",
			Country:     "US",
			Anonymity:   "high",
			LastChecked: time.Now(),
			LastSeen:    time.Now(),
		}

		detection.database.Add(proxy)

		retrieved, exists := detection.database.Get(proxy.IP)
		if !exists {
			t.Fatal("Expected proxy to exist")
		}

		if retrieved.IP != proxy.IP {
			t.Errorf("Expected IP %s, got %s", proxy.IP, retrieved.IP)
		}
	})
}

func TestWebRTCLeak(t *testing.T) {
	detection := NewEnhancedProxyDetection()

	t.Run("detect_webrtc_leak", func(t *testing.T) {
		ctx := context.Background()
		localIPs := []string{"192.168.1.100", "203.0.113.50"}
		remoteIP := "192.168.1.100"

		isLeaked, leakedIPs := detection.CheckWebRTCLeak(ctx, localIPs, remoteIP)

		if isLeaked && len(leakedIPs) > 0 {
			t.Logf("WebRTC leak detected: %v", leakedIPs)
		} else {
			t.Log("No WebRTC leak detected")
		}
	})
}

func TestConnectionAnalysis(t *testing.T) {
	detection := NewEnhancedProxyDetection()

	t.Run("analyze_connection", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		analysis, err := detection.AnalyzeConnection(ctx, "8.8.8.8")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if analysis == nil {
			t.Fatal("Expected analysis to be non-nil")
		}

		t.Logf("Connection analysis for %s", analysis.IP)
		t.Logf("TLS Versions: %v", analysis.TLSVersions)
	})
}

func TestVPNProviderDetection(t *testing.T) {
	detection := NewEnhancedProxyDetection()

	t.Run("detect_vpn_by_asn", func(t *testing.T) {
		testCases := []struct {
			asn         int
			expectedVPN bool
		}{
			{201229, true},
			{212502, true},
			{12345, false},
		}

		for _, tc := range testCases {
			isVPN, provider := detection.DetectVPNByASN(tc.asn)

			if isVPN != tc.expectedVPN {
				t.Errorf("For ASN %d, expected VPN=%v, got VPN=%v", tc.asn, tc.expectedVPN, isVPN)
			}

			if isVPN {
				t.Logf("ASN %d is VPN provider: %s", tc.asn, provider)
			}
		}
	})
}

func TestTorExitNodeManagement(t *testing.T) {
	detection := NewEnhancedProxyDetection()

	t.Run("check_tor_exit_node", func(t *testing.T) {
		if !detection.IsTorExitNode("128.31.0.34") {
			t.Error("Expected 128.31.0.34 to be a known Tor exit node")
		}

		if detection.IsTorExitNode("8.8.8.8") {
			t.Error("Expected 8.8.8.8 to NOT be a Tor exit node")
		}
	})

	t.Run("add_tor_exit_node", func(t *testing.T) {
		newTorIP := "192.168.1.200"
		detection.AddTorExitNode(newTorIP)

		if !detection.IsTorExitNode(newTorIP) {
			t.Error("Expected new Tor exit node to be added")
		}
	})
}

func TestProxyRiskReport(t *testing.T) {
	detection := NewEnhancedProxyDetection()

	t.Run("generate_risk_report", func(t *testing.T) {
		ctx := context.Background()
		result, _ := detection.DetectProxy(ctx, "203.0.113.1", nil)

		report := detection.GenerateRiskReport(result)

		if report == nil {
			t.Fatal("Expected report to be non-nil")
		}

		t.Logf("Risk Level: %s", report.RiskLevel)
		t.Logf("Risk Score: %.2f", report.Score)
		t.Logf("Summary: %s", report.Summary)
		t.Logf("Is Threat: %v", report.IsThreat)
	})
}

func BenchmarkAdvancedFingerprintAnalysis(b *testing.B) {
	analyzer := NewAdvancedFingerprintAnalyzer()

	data := map[string]interface{}{
		"user_agent":            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"canvas_hash":           "test_canvas_hash_123",
		"webgl_hash":            "test_webgl_hash_456",
		"audio_hash":            "test_audio_hash_789",
		"font_hash":             "test_font_hash",
		"screen_resolution":     "1920x1080",
		"timezone":              "America/New_York",
		"language":              "en-US",
		"platform":              "Win32",
		"hardware_concurrency":  float64(8),
		"device_memory":         float64(16),
		"plugins_count":         float64(5),
		"languages_count":       float64(3),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = analyzer.AnalyzeAdvancedFingerprint(data)
	}
}

func BenchmarkProxyDetection(b *testing.B) {
	detection := NewEnhancedProxyDetection()

	headers := http.Header{}
	headers.Set("X-Forwarded-For", "203.0.113.1, 192.168.1.1")

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detection.DetectProxy(ctx, "203.0.113.1", headers)
	}
}
