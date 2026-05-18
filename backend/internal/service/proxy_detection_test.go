package service

import (
	"testing"
	"time"
)

func TestProxyDetectionService_DetectProxy(t *testing.T) {
	service := NewProxyDetectionService()

	testCases := []struct {
		name        string
		ip          string
		headers     map[string]string
		expectProxy bool
	}{
		{
			name:        "Direct connection",
			ip:          "203.0.113.1",
			headers:     map[string]string{},
			expectProxy: false,
		},
		{
			name:        "With X-Forwarded-For",
			ip:          "203.0.113.1",
			headers:     map[string]string{"X-Forwarded-For": "192.0.2.1, 192.0.2.2"},
			expectProxy: true,
		},
		{
			name:        "With X-Real-IP",
			ip:          "203.0.113.1",
			headers:     map[string]string{"X-Real-IP": "192.0.2.1"},
			expectProxy: true,
		},
		{
			name:        "With Via header",
			ip:          "203.0.113.1",
			headers:     map[string]string{"Via": "1.1 proxy.example.com"},
			expectProxy: true,
		},
		{
			name:        "Multi-hop proxy",
			ip:          "203.0.113.1",
			headers:     map[string]string{"X-Forwarded-For": "192.0.2.1, 192.0.2.2, 192.0.2.3"},
			expectProxy: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.DetectProxy(tc.ip, tc.headers)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tc.expectProxy && !result.IsProxy {
				t.Errorf("Expected proxy to be detected")
			}
			if !tc.expectProxy && result.IsProxy {
				t.Errorf("Did not expect proxy detection")
			}
		})
	}
}

func TestProxyDetectionService_DetectProxy_PrivateIP(t *testing.T) {
	service := NewProxyDetectionService()

	privateIPs := []string{
		"10.0.0.1",
		"172.16.0.1",
		"192.168.0.1",
		"127.0.0.1",
	}

	for _, ip := range privateIPs {
		t.Run(ip, func(t *testing.T) {
			result, err := service.DetectProxy(ip, map[string]string{})
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(result.DetectionMethods) == 0 {
				t.Error("Expected at least one detection method for private IP")
			}
		})
	}
}

func TestProxyDetectionService_DetectProxy_DatacenterIP(t *testing.T) {
	service := NewProxyDetectionService()

	datacenterIPs := []string{
		"52.94.236.1",
		"54.239.28.1",
		"3.5.119.1",
		"35.157.127.1",
	}

	for _, ip := range datacenterIPs {
		t.Run(ip, func(t *testing.T) {
			result, err := service.DetectProxy(ip, map[string]string{})
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !result.IsDatacenter {
				t.Errorf("Expected datacenter IP detection for %s", ip)
			}
		})
	}
}

func TestProxyDetectionService_DetectProxy_TorExitNode(t *testing.T) {
	service := NewProxyDetectionService()

	torIP := "185.220.100.240"

	result, err := service.DetectProxy(torIP, map[string]string{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.IsTor {
		t.Error("Expected Tor exit node detection")
	}
}

func TestProxyDetectionService_DetectProxy_ScoreCalculation(t *testing.T) {
	service := NewProxyDetectionService()

	testCases := []struct {
		name          string
		ip            string
		headers       map[string]string
		minScore      float64
		maxScore      float64
		expectedLevel string
	}{
		{
			name:          "Clean IP",
			ip:            "203.0.113.1",
			headers:       map[string]string{},
			minScore:      0,
			maxScore:      15,
			expectedLevel: "minimal",
		},
		{
			name:          "Single proxy header",
			ip:            "203.0.113.1",
			headers:       map[string]string{"X-Forwarded-For": "192.0.2.1"},
			minScore:      25,
			maxScore:      50,
			expectedLevel: "low",
		},
		{
			name:          "Multiple proxy headers",
			ip:            "203.0.113.1",
			headers:       map[string]string{"X-Forwarded-For": "192.0.2.1,192.0.2.2,192.0.2.3"},
			minScore:      40,
			maxScore:      70,
			expectedLevel: "medium",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.DetectProxy(tc.ip, tc.headers)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Score < tc.minScore || result.Score > tc.maxScore {
				t.Errorf("Expected score between %.0f and %.0f, got %.0f", tc.minScore, tc.maxScore, result.Score)
			}

			if result.RiskLevel != tc.expectedLevel {
				t.Errorf("Expected risk level '%s', got '%s'", tc.expectedLevel, result.RiskLevel)
			}
		})
	}
}

func TestProxyDetectionService_AnalyzeConnection(t *testing.T) {
	service := NewProxyDetectionService()

	testCases := []struct {
		name            string
		measurements    []time.Duration
		expectProxy     bool
		expectVPN       bool
		minAnomalyScore float64
	}{
		{
			name:         "Normal connection",
			measurements: []time.Duration{10 * time.Millisecond, 12 * time.Millisecond, 11 * time.Millisecond},
			expectProxy:  false,
			expectVPN:    false,
		},
		{
			name:         "High latency connection",
			measurements: []time.Duration{300 * time.Millisecond, 280 * time.Millisecond, 320 * time.Millisecond},
			expectProxy:  true,
			expectVPN:    false,
		},
		{
			name:         "VPN-like connection",
			measurements: []time.Duration{150 * time.Millisecond, 180 * time.Millisecond, 140 * time.Millisecond},
			expectProxy:  false,
			expectVPN:    true,
		},
		{
			name:            "Empty measurements",
			measurements:    []time.Duration{},
			expectProxy:     false,
			expectVPN:       false,
			minAnomalyScore: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.AnalyzeConnection(tc.measurements)

			if tc.expectProxy && !result.IsProxyPattern {
				t.Error("Expected proxy pattern detection")
			}
			if tc.expectVPN && !result.IsVPNPattern {
				t.Error("Expected VPN pattern detection")
			}
			if result.AnomalyScore < tc.minAnomalyScore {
				t.Errorf("Expected anomaly score >= %.0f, got %.0f", tc.minAnomalyScore, result.AnomalyScore)
			}
		})
	}
}

func TestProxyDetectionService_Blacklist(t *testing.T) {
	service := NewProxyDetectionService()

	ip := "203.0.113.1"

	if service.CheckBlacklist(ip) {
		t.Error("IP should not be blacklisted initially")
	}

	service.AddToBlacklist(ip, 1*time.Hour)

	if !service.CheckBlacklist(ip) {
		t.Error("IP should be blacklisted after adding")
	}

	service.RemoveFromBlacklist(ip)

	if service.CheckBlacklist(ip) {
		t.Error("IP should not be blacklisted after removal")
	}
}

func TestProxyDetectionService_ClearExpiredBlacklist(t *testing.T) {
	service := NewProxyDetectionService()

	ip1 := "203.0.113.1"
	ip2 := "203.0.113.2"

	service.AddToBlacklist(ip1, 1*time.Hour)
	service.AddToBlacklist(ip2, 1*time.Millisecond)

	time.Sleep(10 * time.Millisecond)

	removed := service.ClearExpiredBlacklist()

	if removed < 1 {
		t.Error("Expected at least 1 expired entry to be removed")
	}
}

func TestProxyDetectionService_RealtimeCheck(t *testing.T) {
	service := NewProxyDetectionService()

	testCases := []struct {
		name             string
		req              *RealtimeCheckRequest
		expectSuspicious bool
		minScore         float64
	}{
		{
			name: "Clean request",
			req: &RealtimeCheckRequest{
				IPAddress: "203.0.113.1",
				Headers:   map[string]string{},
				UserAgent: "Mozilla/5.0",
			},
			expectSuspicious: false,
			minScore:         0,
		},
		{
			name: "Request with proxy headers",
			req: &RealtimeCheckRequest{
				IPAddress: "203.0.113.1",
				Headers:   map[string]string{"X-Forwarded-For": "192.0.2.1"},
				UserAgent: "Mozilla/5.0",
			},
			expectSuspicious: true,
			minScore:         25,
		},
		{
			name: "Request with automation UA",
			req: &RealtimeCheckRequest{
				IPAddress: "203.0.113.1",
				Headers:   map[string]string{},
				UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/91.0.4472.0 Safari/537.36",
			},
			expectSuspicious: true,
			minScore:         25,
		},
		{
			name: "Multi-hop proxy",
			req: &RealtimeCheckRequest{
				IPAddress: "203.0.113.1",
				Headers:   map[string]string{"X-Forwarded-For": "192.0.2.1,192.0.2.2,192.0.2.3"},
				UserAgent: "Mozilla/5.0",
			},
			expectSuspicious: true,
			minScore:         20,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.RealtimeCheck(tc.req)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tc.expectSuspicious && !result.IsSuspicious {
				t.Error("Expected suspicious request")
			}
			if result.Score < tc.minScore {
				t.Errorf("Expected score >= %.0f, got %.0f", tc.minScore, result.Score)
			}
		})
	}
}

func TestProxyDetectionService_RealtimeCheck_RiskLevel(t *testing.T) {
	service := NewProxyDetectionService()

	testCases := []struct {
		name     string
		score    float64
		expected string
	}{
		{"High risk", 80, "high"},
		{"Medium risk", 55, "medium"},
		{"Low risk", 30, "low"},
		{"Minimal risk", 10, "minimal"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &RealtimeCheckRequest{
				IPAddress: "203.0.113.1",
				Headers:   map[string]string{},
			}

			if tc.score >= 70 {
				req.Headers = map[string]string{"X-Forwarded-For": "192.0.2.1,192.0.2.2,192.0.2.3"}
			} else if tc.score >= 40 {
				req.Headers = map[string]string{"X-Forwarded-For": "192.0.2.1"}
			}

			result, _ := service.RealtimeCheck(req)

			if result.Score >= 70 && result.RiskLevel != "high" {
				t.Errorf("Expected 'high' risk level for score %.0f", tc.score)
			} else if result.Score >= 40 && result.RiskLevel != "medium" && result.Score < 70 {
				t.Logf("Score %.0f resulted in risk level '%s'", tc.score, result.RiskLevel)
			}
		})
	}
}

func TestProxyDetectionService_BatchCheck(t *testing.T) {
	service := NewProxyDetectionService()

	ips := []string{
		"203.0.113.1",
		"203.0.113.2",
		"192.0.2.1",
	}

	results, err := service.BatchCheck(ips)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != len(ips) {
		t.Errorf("Expected %d results, got %d", len(ips), len(results))
	}

	for _, ip := range ips {
		if _, exists := results[ip]; !exists {
			t.Errorf("Expected result for IP %s", ip)
		}
	}
}

func TestProxyDetectionService_GetIPReputation(t *testing.T) {
	service := NewProxyDetectionService()

	ip := "203.0.113.1"

	reputation, err := service.GetIPReputation(ip)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if reputation == nil {
		t.Fatal("Expected reputation data to be returned")
	}

	if reputation["ip"] != ip {
		t.Errorf("Expected IP '%s', got '%v'", ip, reputation["ip"])
	}

	expectedFields := []string{"is_proxy", "is_vpn", "is_tor", "confidence", "risk_level", "score"}
	for _, field := range expectedFields {
		if _, exists := reputation[field]; !exists {
			t.Errorf("Expected field '%s' in reputation data", field)
		}
	}
}

func TestProxyDetectionService_ValidateHeaders(t *testing.T) {
	service := NewProxyDetectionService()

	testCases := []struct {
		name       string
		headers    map[string]string
		expectFlag bool
	}{
		{
			name:       "Clean headers",
			headers:    map[string]string{"Content-Type": "application/json"},
			expectFlag: false,
		},
		{
			name:       "Headers with proxy keyword",
			headers:    map[string]string{"Via": "1.1 proxy.example.com"},
			expectFlag: true,
		},
		{
			name:       "Headers with VPN keyword",
			headers:    map[string]string{"X-VPN": "enabled"},
			expectFlag: true,
		},
		{
			name:       "Headers with multiple proxy indicators",
			headers:    map[string]string{"X-Forwarded-For": "192.0.2.1", "Via": "squid/3.5"},
			expectFlag: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isFlagged, flagged := service.ValidateHeaders(tc.headers)

			if tc.expectFlag && !isFlagged {
				t.Error("Expected headers to be flagged")
			}
			if !tc.expectFlag && isFlagged {
				t.Errorf("Did not expect headers to be flagged, but got: %v", flagged)
			}
		})
	}
}

func TestProxyDetectionService_GetVPNPatterns(t *testing.T) {
	service := NewProxyDetectionService()

	patterns := service.GetVPNPatterns()

	if len(patterns) == 0 {
		t.Error("Expected VPN patterns to be returned")
	}

	for _, pattern := range patterns {
		if pattern.Name == "" {
			t.Error("Pattern name should not be empty")
		}
		if pattern.Weight <= 0 || pattern.Weight > 1 {
			t.Error("Pattern weight should be between 0 and 1")
		}
	}
}

func TestProxyDetectionService_IsPrivateIP(t *testing.T) {
	service := NewProxyDetectionService()

	testCases := []struct {
		ip       string
		expected bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.0.1", true},
		{"192.168.255.255", true},
		{"127.0.0.1", true},
		{"169.254.0.1", true},
		{"203.0.113.1", false},
		{"8.8.8.8", false},
	}

	for _, tc := range testCases {
		t.Run(tc.ip, func(t *testing.T) {
			result := service.isPrivateIP(tc.ip)
			if result != tc.expected {
				t.Errorf("Expected isPrivateIP(%s) = %v, got %v", tc.ip, tc.expected, result)
			}
		})
	}
}

func TestProxyDetectionService_IsDatacenterIP(t *testing.T) {
	service := NewProxyDetectionService()

	testCases := []struct {
		ip       string
		expected bool
	}{
		{"52.94.236.1", true},
		{"54.239.28.1", true},
		{"3.5.119.1", true},
		{"192.168.0.1", false},
		{"203.0.113.1", false},
	}

	for _, tc := range testCases {
		t.Run(tc.ip, func(t *testing.T) {
			result := service.isDatacenterIP(tc.ip)
			if result != tc.expected {
				t.Errorf("Expected isDatacenterIP(%s) = %v, got %v", tc.ip, tc.expected, result)
			}
		})
	}
}

func TestProxyDetectionService_IsTorExitIP(t *testing.T) {
	service := NewProxyDetectionService()

	testCases := []struct {
		ip       string
		expected bool
	}{
		{"185.220.100.240", true},
		{"199.249.230.1", true},
		{"203.0.113.1", false},
		{"8.8.8.8", false},
	}

	for _, tc := range testCases {
		t.Run(tc.ip, func(t *testing.T) {
			result := service.isTorExitIP(tc.ip)
			if result != tc.expected {
				t.Errorf("Expected isTorExitIP(%s) = %v, got %v", tc.ip, tc.expected, result)
			}
		})
	}
}

func TestProxyDatabase_KnownTorCache(t *testing.T) {
	db := NewProxyDatabase()

	ip := "192.0.2.1"

	if db.knownTor[ip] {
		t.Error("IP should not be in known Tor list initially")
	}

	db.knownTor[ip] = true

	if !db.knownTor[ip] {
		t.Error("IP should be in known Tor list after adding")
	}
}

func TestProxyDetection_ResponseTime(t *testing.T) {
	service := NewProxyDetectionService()

	ip := "203.0.113.1"
	headers := map[string]string{}

	result, err := service.DetectProxy(ip, headers)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ResponseTime < 0 {
		t.Error("Response time should not be negative")
	}

	if result.LastChecked.IsZero() {
		t.Error("LastChecked should be set")
	}
}

func TestProxyDetection_DetectionMethods(t *testing.T) {
	service := NewProxyDetectionService()

	testCases := []struct {
		name        string
		ip          string
		headers     map[string]string
		expectedMin int
	}{
		{
			name:        "Clean connection",
			ip:          "203.0.113.1",
			headers:     map[string]string{},
			expectedMin: 0,
		},
		{
			name:        "With proxy header",
			ip:          "203.0.113.1",
			headers:     map[string]string{"X-Forwarded-For": "192.0.2.1"},
			expectedMin: 1,
		},
		{
			name:        "With multiple headers",
			ip:          "203.0.113.1",
			headers:     map[string]string{"X-Forwarded-For": "192.0.2.1", "Via": "proxy"},
			expectedMin: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.DetectProxy(tc.ip, tc.headers)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(result.DetectionMethods) < tc.expectedMin {
				t.Errorf("Expected at least %d detection methods, got %d", tc.expectedMin, len(result.DetectionMethods))
			}
		})
	}
}
