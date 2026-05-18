package service

import (
	"context"
	"net/http"
	"sync"
	"testing"
)

func TestEnhancedVMDetector(t *testing.T) {
	detector := NewEnhancedVMDetector()

	t.Run("detect_vmware", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) VMware, VirtualBox",
			"webgl_renderer":  "VMware SVGA II Adapter",
			"cpu_cores":       float64(2),
			"device_memory":    float64(2),
		}

		result := detector.DetectVM(data)

		if len(result.Indicators) == 0 {
			t.Error("Expected VM indicators to be detected")
		}

		if result.RiskScore == 0 {
			t.Error("Expected non-zero risk score")
		}

		t.Logf("VM Detection - Type: %s, Score: %d, Indicators: %v", result.VMType, result.RiskScore, result.Indicators)
	})

	t.Run("detect_virtualbox", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Oracle VirtualBox",
			"webgl_renderer":  "VirtualBox Graphics Adapter",
			"cpu_cores":       float64(4),
			"device_memory":    float64(8),
		}

		result := detector.DetectVM(data)

		if result.RiskScore > 0 {
			t.Logf("VirtualBox detected with score: %d", result.RiskScore)
		}
	})

	t.Run("detect_headless_browser", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":      "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/91.0 HeadlessChrome",
			"webgl_renderer":  "SwiftShader",
			"cpu_cores":       float64(4),
			"device_memory":    float64(8),
			"headless_browser": true,
		}

		result := detector.DetectVM(data)

		if result.RiskScore > 0 {
			t.Logf("Headless browser detected with score: %d", result.RiskScore)
		}
	})

	t.Run("detect_low_core_count", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":   "Mozilla/5.0 Test Browser",
			"cpu_cores":    float64(1),
			"device_memory": float64(0.5),
		}

		result := detector.DetectVM(data)

		found := false
		for _, ind := range result.Indicators {
			if ind == "single_core_detected" || ind == "very_low_memory" {
				found = true
			}
		}

		if !found {
			t.Error("Expected low resource indicators")
		}
	})

	t.Run("detect_emulator_resolution", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":         "Mozilla/5.0 (Linux; Android 11; Pixel 5) AppleWebKit/537.36",
			"screen_resolution":  "375x667",
		}

		result := detector.DetectVM(data)

		width, ok := result.Details["screen_width"]
		if !ok {
			t.Error("Expected screen_width in details")
			return
		}
		switch v := width.(type) {
		case int:
			if v != 375 {
				t.Errorf("Expected screen width 375, got %v", v)
			}
		case float64:
			if int(v) != 375 {
				t.Errorf("Expected screen width 375, got %v", v)
			}
		default:
			t.Errorf("Unexpected type for screen_width: %T", width)
		}
	})

	t.Run("detect_hyperv", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Microsoft Hyper-V",
			"webgl_renderer":  "Microsoft Basic Render Driver",
			"cpu_cores":       float64(4),
			"device_memory":    float64(8),
		}

		result := detector.DetectVM(data)

		if result.RiskScore > 0 {
			t.Logf("Hyper-V detected with score: %d", result.RiskScore)
		}
	})

	t.Run("detect_qemu_kvm", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":      "Mozilla/5.0 (X11; Linux x86_64; QEMU) AppleWebKit/537.36",
			"webgl_renderer":  "llvmpipe (LLVM 12.0.0, 256 bits)",
			"cpu_cores":       float64(2),
			"device_memory":    float64(4),
		}

		result := detector.DetectVM(data)

		if result.RiskScore > 0 {
			t.Logf("QEMU/KVM detected with score: %d", result.RiskScore)
		}
	})

	t.Run("detect_automation_framework", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent":         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/91.0 Selenium",
			"automation_detected": true,
		}

		result := detector.DetectVM(data)

		found := false
		for _, ind := range result.Indicators {
			if ind == "automation_framework_detected" {
				found = true
			}
		}

		if !found {
			t.Error("Expected automation framework indicator")
		}
	})
}

func TestEnhancedContainerDetection(t *testing.T) {
	detector := NewEnhancedVMDetector()

	t.Run("detect_docker", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 (Linux; Docker) AppleWebKit/537.36",
			"hostname":   "container-abc12345",
			"environment_vars": map[string]string{
				"container": "docker",
				"HOME":      "/root",
			},
			"storage_info": map[string]interface{}{
				"quota": float64(0),
			},
		}

		result := detector.DetectContainer(data)

		if len(result.Indicators) >= 3 {
			t.Logf("Container detected with %d indicators: %v", len(result.Indicators), result.Indicators)
		}
	})

	t.Run("detect_kubernetes", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 (Linux; Kubernetes; GKE) AppleWebKit/537.36",
			"hostname":   "myapp-deployment-abc123",
			"environment_vars": map[string]string{
				"KUBERNETES_PORT": "tcp://10.0.0.1:443",
			},
		}

		result := detector.DetectContainer(data)

		if len(result.Indicators) > 0 {
			t.Logf("Kubernetes detected with %d indicators: %v", len(result.Indicators), result.Indicators)
		}
	})

	t.Run("detect_low_storage_quota", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 Test Browser",
			"storage_info": map[string]interface{}{
				"quota": float64(50000000),
			},
		}

		result := detector.DetectContainer(data)

		found := false
		for _, ind := range result.Indicators {
			if ind == "low_storage_quota" {
				found = true
			}
		}

		if !found {
			t.Error("Expected low storage quota indicator")
		}
	})

	t.Run("detect_lxc", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 (Linux; LXC) AppleWebKit/537.36",
			"hostname":   "container-lxc-xyz789",
			"environment_vars": map[string]string{
				"LXC_NAME": "my-container",
			},
		}

		result := detector.DetectContainer(data)

		if len(result.Indicators) > 0 {
			t.Logf("LXC detected with %d indicators", len(result.Indicators))
		}
	})
}

func TestEnhancedProxyVPNDetector(t *testing.T) {
	detector := NewEnhancedProxyVPNDetector()

	t.Run("detect_vpn_by_ip_range", func(t *testing.T) {
		ctx := context.Background()
		result := detector.DetectProxyVPN(ctx, "45.33.32.156", nil, nil)

		if result.IsVPN {
			t.Logf("VPN detected: %v, Provider: %s", result.IsVPN, result.VPNProvider)
		} else {
			t.Log("VPN not detected (IP may not be in known ranges)")
		}
	})

	t.Run("detect_tor_exit_node", func(t *testing.T) {
		ctx := context.Background()
		result := detector.DetectProxyVPN(ctx, "128.31.0.34", nil, nil)

		if result.IsTor {
			t.Logf("Tor exit node detected with confidence: %.2f", result.Confidence)
		} else {
			t.Error("Expected Tor exit node to be detected")
		}
	})

	t.Run("detect_proxy_headers", func(t *testing.T) {
		ctx := context.Background()
		headers := http.Header{}
		headers.Set("X-Forwarded-For", "203.0.113.1, 192.168.1.1, 10.0.0.1")
		headers.Set("Via", "1.1 proxy-server")

		result := detector.DetectProxyVPN(ctx, "203.0.113.1", headers, nil)

		if result.IsProxy {
			t.Logf("Proxy detected via headers")
		}

		if len(result.Indicators) > 0 {
			t.Logf("Indicators: %v", result.Indicators)
		}
	})

	t.Run("detect_vpn_by_asn", func(t *testing.T) {
		ctx := context.Background()
		data := map[string]interface{}{
			"asn": float64(201229),
		}

		result := detector.DetectProxyVPN(ctx, "192.168.1.1", nil, data)

		if result.IsVPN {
			t.Logf("VPN detected via ASN: %s", result.VPNProvider)
		}
	})

	t.Run("detect_datacenter_ip", func(t *testing.T) {
		ctx := context.Background()
		result := detector.DetectProxyVPN(ctx, "45.67.89.10", nil, nil)

		if result.IsDatacenter {
			t.Log("Datacenter IP detected")
		}
	})

	t.Run("detect_webrtc_leak", func(t *testing.T) {
		ctx := context.Background()
		data := map[string]interface{}{
			"webrtc_ips": []string{"192.168.1.100", "203.0.113.50"},
		}

		result := detector.DetectProxyVPN(ctx, "192.168.1.100", nil, data)

		found := false
		for _, ind := range result.Indicators {
			if ind == "webrtc_ip_mismatch" {
				found = true
			}
		}

		if found {
			t.Log("WebRTC leak detected")
		}
	})

	t.Run("calculate_risk_level", func(t *testing.T) {
		ctx := context.Background()
		result := detector.DetectProxyVPN(ctx, "128.31.0.34", nil, nil)

		if result.RiskLevel != "" {
			t.Logf("Risk level: %s, Score: %d", result.RiskLevel, result.Score)
		}

		if len(result.Recommendations) > 0 {
			t.Logf("Recommendations: %v", result.Recommendations)
		}
	})

	t.Run("batch_detect", func(t *testing.T) {
		ctx := context.Background()
		requests := []ProxyCheckRequestEnv{
			{IP: "45.33.32.156", Headers: nil, Data: nil},
			{IP: "128.31.0.34", Headers: nil, Data: nil},
			{IP: "203.0.113.1", Headers: http.Header{}, Data: nil},
		}

		results := detector.BatchDetect(ctx, requests)

		if len(results) != len(requests) {
			t.Errorf("Expected %d results, got %d", len(requests), len(results))
		}

		for i, result := range results {
			t.Logf("Request %d - VPN: %v, Tor: %v, Proxy: %v, Score: %d",
				i, result.IsVPN, result.IsTor, result.IsProxy, result.Score)
		}
	})

	t.Run("connection_type_vpn", func(t *testing.T) {
		ctx := context.Background()
		data := map[string]interface{}{
			"connection_type": "vpn",
		}

		result := detector.DetectProxyVPN(ctx, "192.168.1.1", nil, data)

		if result.IsVPN {
			t.Logf("VPN detected via connection type")
		}
	})

	t.Run("connection_type_proxy", func(t *testing.T) {
		ctx := context.Background()
		data := map[string]interface{}{
			"connection_type": "socks",
		}

		result := detector.DetectProxyVPN(ctx, "192.168.1.1", nil, data)

		if result.IsProxy {
			t.Logf("Proxy detected via connection type")
		}
	})
}

func TestDetectionPatternMatcher(t *testing.T) {
	matcher := NewDetectionPatternMatcher()

	t.Run("match_headless_browser", func(t *testing.T) {
		testCases := []struct {
			text     string
			expected bool
		}{
			{"headless browser detected", true},
			{"phantomjs automation", true},
			{"puppeteer is running", true},
			{"playwright framework", true},
			{"selenium webdriver", true},
			{"normal browser session", false},
		}

		for _, tc := range testCases {
			results := matcher.Match(tc.text)
			matched := len(results) > 0

			if matched != tc.expected {
				t.Errorf("For text '%s', expected match=%v, got match=%v", tc.text, tc.expected, matched)
			}
		}
	})

	t.Run("match_virtual_machine", func(t *testing.T) {
		testCases := []struct {
			text     string
			expected bool
		}{
			{"vmware virtual platform", true},
			{"virtualbox guest", true},
			{"hyper-v detected", true},
			{"qemu kvm running", true},
			{"physical machine", false},
		}

		for _, tc := range testCases {
			results := matcher.Match(tc.text)
			matched := len(results) > 0

			if matched != tc.expected {
				t.Errorf("For text '%s', expected match=%v, got match=%v", tc.text, tc.expected, matched)
			}
		}
	})

	t.Run("match_tor", func(t *testing.T) {
		testCases := []struct {
			text     string
			expected bool
		}{
			{"torbrowser bundle", true},
			{"tor exit node", true},
			{"torproject relay", true},
			{"onion routing", true},
		}

		for _, tc := range testCases {
			results := matcher.Match(tc.text)
			matched := len(results) > 0

			if matched != tc.expected {
				t.Errorf("For text '%s', expected match=%v, got match=%v", tc.text, tc.expected, matched)
			}
		}
	})

	t.Run("match_vpn", func(t *testing.T) {
		testCases := []struct {
			text     string
			expected bool
		}{
			{"nordvpn connected", true},
			{"expressvpn active", true},
			{"surfshark tunnel", true},
			{"protonvpn enabled", true},
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

func TestCalculateFingerprintHash(t *testing.T) {
	t.Run("hash_consistency", func(t *testing.T) {
		data := map[string]interface{}{
			"user_agent": "Mozilla/5.0 Test",
			"canvas":     "test123",
			"webgl":      "test456",
		}

		hash1 := CalculateFingerprintHash(data)
		hash2 := CalculateFingerprintHash(data)

		if hash1 != hash2 {
			t.Error("Hash should be consistent for same data")
		}
	})

	t.Run("hash_uniqueness", func(t *testing.T) {
		data1 := map[string]interface{}{
			"user_agent": "Mozilla/5.0 Test 1",
		}

		data2 := map[string]interface{}{
			"user_agent": "Mozilla/5.0 Test 2",
		}

		hash1 := CalculateFingerprintHash(data1)
		hash2 := CalculateFingerprintHash(data2)

		if hash1 == hash2 {
			t.Error("Hash should be different for different data")
		}
	})

	t.Run("hash_length", func(t *testing.T) {
		data := map[string]interface{}{
			"test": "data",
		}

		hash := CalculateFingerprintHash(data)

		if len(hash) != 16 {
			t.Errorf("Expected hash length 16, got %d", len(hash))
		}
	})
}

func BenchmarkEnhancedVMDetection(b *testing.B) {
	detector := NewEnhancedVMDetector()

	data := map[string]interface{}{
		"user_agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/91.0",
		"webgl_renderer":  "NVIDIA GeForce GTX 1080",
		"cpu_cores":       float64(8),
		"device_memory":    float64(16),
		"screen_resolution": "1920x1080",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.DetectVM(data)
	}
}

func BenchmarkEnhancedProxyVPNDetection(b *testing.B) {
	detector := NewEnhancedProxyVPNDetector()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.DetectProxyVPN(ctx, "45.33.32.156", nil, nil)
	}
}

func BenchmarkDetectionPatternMatching(b *testing.B) {
	matcher := NewDetectionPatternMatcher()
	text := "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/91.0 Headless PhantomJS detected"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.Match(text)
	}
}

func BenchmarkConcurrentProxyDetection(b *testing.B) {
	detector := NewEnhancedProxyVPNDetector()
	ctx := context.Background()

	requests := make([]ProxyCheckRequestEnv, 100)
	for i := 0; i < 100; i++ {
		requests[i] = ProxyCheckRequestEnv{
			IP:      "192.168.1.1",
			Headers: nil,
			Data:    nil,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for _, req := range requests {
			wg.Add(1)
			go func(r ProxyCheckRequestEnv) {
				defer wg.Done()
				detector.DetectProxyVPN(ctx, r.IP, r.Headers, r.Data)
			}(req)
		}
		wg.Wait()
	}
}
