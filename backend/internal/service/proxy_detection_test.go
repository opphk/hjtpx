package service

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNewProxyDetectionService(t *testing.T) {
	service := NewProxyDetectionService()
	if service == nil {
		t.Fatal("NewProxyDetectionService() returned nil")
	}
	if service.database == nil {
		t.Error("service.database should not be nil")
	}
	if service.httpClient == nil {
		t.Error("service.httpClient should not be nil")
	}
	if service.detectionWeights == nil {
		t.Error("service.detectionWeights should not be nil")
	}
}

func TestDetectProxy_Headers(t *testing.T) {
	service := NewProxyDetectionService()

	tests := []struct {
		name        string
		ip          string
		headers     map[string]string
		expectProxy bool
	}{
		{
			name:        "No proxy headers",
			ip:          "8.8.8.8",
			headers:     map[string]string{},
			expectProxy: false,
		},
		{
			name:        "X-Forwarded-For header",
			ip:          "8.8.8.8",
			headers:     map[string]string{"X-Forwarded-For": "192.168.1.1"},
			expectProxy: true,
		},
		{
			name:        "X-Real-IP header",
			ip:          "8.8.8.8",
			headers:     map[string]string{"X-Real-IP": "192.168.1.1"},
			expectProxy: true,
		},
		{
			name:        "Via header",
			ip:          "8.8.8.8",
			headers:     map[string]string{"Via": "1.1 proxy.example.com"},
			expectProxy: true,
		},
		{
			name:        "Multi-hop proxy",
			ip:          "8.8.8.8",
			headers:     map[string]string{"X-Forwarded-For": "192.168.1.1, 192.168.1.2, 192.168.1.3"},
			expectProxy: true,
		},
		{
			name:        "Proxy chain header",
			ip:          "8.8.8.8",
			headers:     map[string]string{"X-ProxyChain": "tor"},
			expectProxy: true,
		},
		{
			name:        "Forwarded header",
			ip:          "8.8.8.8",
			headers:     map[string]string{"Forwarded": "for=192.168.1.1"},
			expectProxy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detection, err := service.DetectProxy(tt.ip, tt.headers)
			if err != nil {
				t.Fatalf("DetectProxy() error = %v", err)
			}
			if detection.IsProxy != tt.expectProxy {
				t.Errorf("DetectProxy() IsProxy = %v, want %v", detection.IsProxy, tt.expectProxy)
			}
		})
	}
}

func TestDetectProxy_ViaHeaderKeywords(t *testing.T) {
	service := NewProxyDetectionService()

	keywords := []string{"proxy", "squid", "nginx", "apache", "varnish", "traefik", "haproxy", "envoy"}

	for _, keyword := range keywords {
		t.Run(keyword, func(t *testing.T) {
			headers := map[string]string{"Via": "1.1 " + keyword + ".example.com"}
			detection, err := service.DetectProxy("8.8.8.8", headers)
			if err != nil {
				t.Fatalf("DetectProxy() error = %v", err)
			}
			if len(detection.DetectionMethods) == 0 {
				t.Errorf("Expected detection method for keyword: %s", keyword)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	service := NewProxyDetectionService()

	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"10.x.x.x private", "10.0.0.1", true},
		{"172.16.x.x private", "172.16.0.1", true},
		{"172.31.x.x private", "172.31.0.1", true},
		{"192.168.x.x private", "192.168.0.1", true},
		{"127.0.0.1 loopback", "127.0.0.1", true},
		{"Public IP", "8.8.8.8", false},
		{"Public IP 2", "1.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isPrivateIP(tt.ip)
			if result != tt.expected {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsDatacenterIP(t *testing.T) {
	service := NewProxyDetectionService()

	tests := []struct {
		name      string
		ip        string
		shouldHit bool
	}{
		{"AWS IP", "52.84.0.1", true},
		{"Azure IP", "13.64.0.1", true},
		{"GCP IP", "104.154.0.1", true},
		{"Public IP not in DC", "8.8.8.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isDatacenterIP(tt.ip)
			if result != tt.shouldHit {
				t.Errorf("isDatacenterIP(%s) = %v, want %v", tt.ip, result, tt.shouldHit)
			}
		})
	}
}

func TestCheckDatacenterProvider(t *testing.T) {
	service := NewProxyDetectionService()

	tests := []struct {
		name              string
		ip                string
		expectedProvider  string
	}{
		{"AWS range", "18.132.45.67", "AWS"},
		{"Azure range", "20.45.67.89", "Azure"},
		{"GCP range", "35.192.45.67", "GCP"},
		{"Not datacenter", "8.8.8.8", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.checkDatacenterProvider(tt.ip)
			if result != tt.expectedProvider {
				t.Errorf("checkDatacenterProvider(%s) = %v, want %v", tt.ip, result, tt.expectedProvider)
			}
		})
	}
}

func TestCheckVPNProvider(t *testing.T) {
	service := NewProxyDetectionService()

	tests := []struct {
		name     string
		asn      string
		expected string
	}{
		{"NordVPN ASN", "AS45090", "NordVPN"},
		{"ExpressVPN ASN", "AS400052", "ExpressVPN"},
		{"ProtonVPN ASN", "AS42385", "ProtonVPN"},
		{"Unknown ASN", "AS12345", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.checkVPNProvider("", tt.asn)
			if result != tt.expected {
				t.Errorf("checkVPNProvider(, %s) = %v, want %v", tt.asn, result, tt.expected)
			}
		})
	}
}

func TestIsTorExitIP(t *testing.T) {
	service := NewProxyDetectionService()

	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"Known Tor exit", "185.220.100.240", true},
		{"Another Tor exit", "199.249.230.1", true},
		{"Public IP", "8.8.8.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isTorExitIP(tt.ip)
			if result != tt.expected {
				t.Errorf("isTorExitIP(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestAnalyzeConnection(t *testing.T) {
	service := NewProxyDetectionService()

	tests := []struct {
		name           string
		measurements   []time.Duration
		expectProxy    bool
		expectVPN      bool
	}{
		{
			name:           "Normal latency",
			measurements:   []time.Duration{50 * time.Millisecond, 55 * time.Millisecond, 52 * time.Millisecond},
			expectProxy:    false,
			expectVPN:      false,
		},
		{
			name:           "High latency VPN pattern",
			measurements:   []time.Duration{150 * time.Millisecond, 180 * time.Millisecond, 200 * time.Millisecond},
			expectProxy:    false,
			expectVPN:      true,
		},
		{
			name:           "Very high latency proxy pattern",
			measurements:   []time.Duration{250 * time.Millisecond, 320 * time.Millisecond, 400 * time.Millisecond},
			expectProxy:    true,
			expectVPN:      false,
		},
		{
			name:           "Empty measurements",
			measurements:   []time.Duration{},
			expectProxy:    false,
			expectVPN:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := service.AnalyzeConnection(tt.measurements)
			if analysis.IsProxyPattern != tt.expectProxy {
				t.Errorf("AnalyzeConnection() IsProxyPattern = %v, want %v", analysis.IsProxyPattern, tt.expectProxy)
			}
			if analysis.IsVPNPattern != tt.expectVPN {
				t.Errorf("AnalyzeConnection() IsVPNPattern = %v, want %v", analysis.IsVPNPattern, tt.expectVPN)
			}
		})
	}
}

func TestBlacklist(t *testing.T) {
	service := NewProxyDetectionService()

	ip := "192.168.1.100"

	if service.CheckBlacklist(ip) {
		t.Error("Newly added IP should not be in blacklist yet")
	}

	service.AddToBlacklist(ip, 1*time.Hour)

	if !service.CheckBlacklist(ip) {
		t.Error("IP should be in blacklist after AddToBlacklist")
	}

	service.RemoveFromBlacklist(ip)

	if service.CheckBlacklist(ip) {
		t.Error("IP should not be in blacklist after RemoveFromBlacklist")
	}
}

func TestRealtimeCheck(t *testing.T) {
	service := NewProxyDetectionService()

	req := &RealtimeCheckRequest{
		IPAddress: "8.8.8.8",
		Headers:  map[string]string{},
	}

	resp, err := service.RealtimeCheck(req)
	if err != nil {
		t.Fatalf("RealtimeCheck() error = %v", err)
	}

	if resp == nil {
		t.Fatal("RealtimeCheck() returned nil response")
	}

	if resp.IPAddress != req.IPAddress {
		t.Errorf("IPAddress = %v, want %v", resp.IPAddress, req.IPAddress)
	}
}

func TestRealtimeCheck_WithWebRTC(t *testing.T) {
	service := NewProxyDetectionService()

	req := &RealtimeCheckRequest{
		IPAddress: "8.8.8.8",
		Headers:   map[string]string{},
		WebRTCInfo: &WebRTCInfo{
			PublicIPs:     []string{"203.0.113.1"},
			RelayDetected: true,
		},
	}

	resp, err := service.RealtimeCheck(req)
	if err != nil {
		t.Fatalf("RealtimeCheck() error = %v", err)
	}

	if !resp.WebRTCLeakDetected {
		t.Error("WebRTC leak should be detected")
	}
}

func TestRealtimeCheck_WithTimezoneMismatch(t *testing.T) {
	service := NewProxyDetectionService()

	req := &RealtimeCheckRequest{
		IPAddress:  "8.8.8.8",
		Headers:   map[string]string{},
		TimezoneInfo: &TimezoneInfo{
			Timezone: "Asia/Tokyo",
		},
	}

	resp, err := service.RealtimeCheck(req)
	if err != nil {
		t.Fatalf("RealtimeCheck() error = %v", err)
	}

	if resp.TimezoneMismatch {
		t.Error("Timezone mismatch should not be detected for valid combination")
	}
}

func TestRealtimeCheck_UserAgentIndicators(t *testing.T) {
	service := NewProxyDetectionService()

	automationIndicators := []string{"headless", "phantom", "puppeteer", "playwright", "selenium", "webdriver"}

	for _, indicator := range automationIndicators {
		t.Run(indicator, func(t *testing.T) {
			req := &RealtimeCheckRequest{
				IPAddress: "8.8.8.8",
				Headers:   map[string]string{},
				UserAgent: "Mozilla/5.0 (" + indicator + ")",
			}

			resp, err := service.RealtimeCheck(req)
			if err != nil {
				t.Fatalf("RealtimeCheck() error = %v", err)
			}

			found := false
			for _, reason := range resp.Reasons {
				if strings.Contains(reason, indicator) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected automation indicator %s to be detected in reasons", indicator)
			}
		})
	}
}

func TestValidateHeaders(t *testing.T) {
	service := NewProxyDetectionService()

	tests := []struct {
		name           string
		headers        map[string]string
		expectFlagged  bool
	}{
		{
			name:          "Clean headers",
			headers:       map[string]string{"Content-Type": "application/json"},
			expectFlagged: false,
		},
		{
			name:          "Proxy keyword in value",
			headers:       map[string]string{"X-Custom": "via proxy"},
			expectFlagged: true,
		},
		{
			name:          "VPN keyword in value",
			headers:       map[string]string{"X-Custom": "vpn connection"},
			expectFlagged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagged, _ := service.ValidateHeaders(tt.headers)
			if flagged != tt.expectFlagged {
				t.Errorf("ValidateHeaders() flagged = %v, want %v", flagged, tt.expectFlagged)
			}
		})
	}
}

func TestGetVPNPatterns(t *testing.T) {
	service := NewProxyDetectionService()

	patterns := service.GetVPNPatterns()

	if len(patterns) == 0 {
		t.Error("GetVPNPatterns() should return at least one pattern")
	}

	for _, pattern := range patterns {
		if pattern.Name == "" {
			t.Error("Pattern should have a name")
		}
		if pattern.Weight < 0 || pattern.Weight > 1 {
			t.Errorf("Pattern weight should be between 0 and 1, got %v", pattern.Weight)
		}
	}
}

func TestGetIPReputation(t *testing.T) {
	service := NewProxyDetectionService()

	result, err := service.GetIPReputation("8.8.8.8")
	if err != nil {
		t.Fatalf("GetIPReputation() error = %v", err)
	}

	if result == nil {
		t.Fatal("GetIPReputation() returned nil")
	}

	if result["ip"] != "8.8.8.8" {
		t.Errorf("IP should be 8.8.8.8, got %v", result["ip"])
	}
}

func TestBatchCheck(t *testing.T) {
	service := NewProxyDetectionService()

	ips := []string{"8.8.8.8", "1.1.1.1", "208.67.222.222"}

	results, err := service.BatchCheck(ips)
	if err != nil {
		t.Fatalf("BatchCheck() error = %v", err)
	}

	if len(results) != len(ips) {
		t.Errorf("BatchCheck() returned %d results, want %d", len(results), len(ips))
	}
}

func TestValidateAndEnrichIP(t *testing.T) {
	service := NewProxyDetectionService()

	tests := []struct {
		name      string
		ip        string
		isValid   bool
		ipVersion int
	}{
		{"Valid IPv4", "8.8.8.8", true, 4},
		{"Valid IPv4 private", "192.168.1.1", true, 4},
		{"Valid IPv6", "2001:4860:4860::8888", true, 6},
		{"Invalid IP", "invalid", false, 0},
		{"Invalid IP 2", "256.256.256.256", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.ValidateAndEnrichIP(tt.ip)
			if result.IsValid != tt.isValid {
				t.Errorf("ValidateAndEnrichIP(%s) IsValid = %v, want %v", tt.ip, result.IsValid, tt.isValid)
			}
			if result.IPVersion != tt.ipVersion {
				t.Errorf("ValidateAndEnrichIP(%s) IPVersion = %v, want %v", tt.ip, result.IPVersion, tt.ipVersion)
			}
		})
	}
}

func TestEnhancedProxyDetectionServiceBasic(t *testing.T) {
	service := NewEnhancedProxyDetectionService()
	if service == nil {
		t.Fatal("NewEnhancedProxyDetectionService() returned nil")
	}

	assessment := service.AssessIPRisk("8.8.8.8", map[string]string{}, nil)
	if assessment == nil {
		t.Fatal("AssessIPRisk() returned nil")
	}

	if assessment.IPAddress != "8.8.8.8" {
		t.Errorf("IPAddress = %v, want 8.8.8.8", assessment.IPAddress)
	}
}

func TestEnhancedService_VPNDetection(t *testing.T) {
	service := NewEnhancedProxyDetectionService()

	isVPN, confidence, evidence := service.DetectVPN("8.8.8.8", map[string]string{})

	if len(evidence) == 0 && isVPN {
		t.Error("Should not detect VPN for public IP")
	}

	if confidence < 0 || confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %v", confidence)
	}
}

func TestEnhancedService_TorDetection(t *testing.T) {
	service := NewEnhancedProxyDetectionService()

	isTor, confidence, evidence := service.DetectTorNetwork("185.220.100.240")

	if !isTor {
		t.Error("Should detect known Tor exit node")
	}

	if confidence < 0.9 {
		t.Errorf("Confidence for known Tor exit should be high, got %v", confidence)
	}

	if len(evidence) == 0 {
		t.Error("Should have evidence for Tor detection")
	}
}

func TestEnhancedService_CDNDetection(t *testing.T) {
	service := NewEnhancedProxyDetectionService()

	tests := []struct {
		name          string
		ip            string
		expectedCDN   bool
		expectedName  string
	}{
		{"Cloudflare IP", "104.16.0.1", true, "Cloudflare"},
		{"Akamai IP", "23.0.0.1", true, "Akamai"},
		{"Fastly IP", "151.101.0.1", true, "Fastly"},
		{"Public IP", "8.8.8.8", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCDN, cdnName, confidence := service.DetectCDNOrigin(tt.ip)
			if isCDN != tt.expectedCDN {
				t.Errorf("DetectCDNOrigin(%s) = %v, want %v", tt.ip, isCDN, tt.expectedCDN)
			}
			if tt.expectedCDN && cdnName != tt.expectedName {
				t.Errorf("CDN name = %v, want %v", cdnName, tt.expectedName)
			}
			if isCDN && (confidence < 0 || confidence > 1) {
				t.Errorf("Confidence should be between 0 and 1, got %v", confidence)
			}
		})
	}
}

func TestThreatIntelligenceUpdate(t *testing.T) {
	service := NewEnhancedProxyDetectionService()

	maliciousIPs := []string{"1.2.3.4", "5.6.7.8"}
	botNets := []string{"9.9.9.9"}

	service.UpdateThreatIntelligence(maliciousIPs, botNets)

	assessment := service.AssessIPRisk("1.2.3.4", map[string]string{}, nil)

	found := false
	for _, factor := range assessment.RiskFactors {
		if factor.Category == "threat_intel" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Should detect malicious IP from threat intelligence")
	}
}

func TestCacheAssessment(t *testing.T) {
	service := NewEnhancedProxyDetectionService()

	assessment := service.AssessIPRisk("8.8.8.8", map[string]string{}, nil)
	service.CacheAssessment(assessment)

	cached, found := service.GetCachedAssessment("8.8.8.8")
	if !found {
		t.Error("Should find cached assessment")
	}

	if cached == nil {
		t.Error("Cached assessment should not be nil")
	}
}

func TestTorExitNodeManagement(t *testing.T) {
	service := NewProxyDetectionService()

	initialCount := service.GetTorExitNodeCount()

	testIP := "10.0.0.1"
	service.AddTorExitNode(testIP)

	if !service.isTorExitIP(testIP) {
		t.Error("Added Tor exit node should be detected")
	}

	newCount := service.GetTorExitNodeCount()
	if newCount != initialCount+1 {
		t.Errorf("Tor exit node count should increase by 1, got %d", newCount-initialCount)
	}

	service.RemoveTorExitNode(testIP)

	if service.isTorExitIP(testIP) {
		t.Error("Removed Tor exit node should not be detected")
	}
}

func TestDatacenterRangesUpdate(t *testing.T) {
	service := NewProxyDetectionService()

	providers := service.GetSupportedDatacenterProviders()
	if len(providers) == 0 {
		t.Error("Should have at least one datacenter provider")
	}

	newRanges := []string{"1.0.0.0/8", "2.0.0.0/8"}
	service.UpdateDatacenterRanges("TestProvider", newRanges)

	updatedProviders := service.GetSupportedDatacenterProviders()
	found := false
	for _, p := range updatedProviders {
		if p == "TestProvider" {
			found = true
			break
		}
	}

	if !found {
		t.Error("New datacenter provider should be in the list")
	}
}

func TestVPNProviders(t *testing.T) {
	service := NewProxyDetectionService()

	providers := service.GetSupportedVPNProviders()
	if len(providers) == 0 {
		t.Error("Should have at least one VPN provider")
	}

	expectedProviders := []string{"NordVPN", "ExpressVPN", "ProtonVPN", "Mullvad"}
	for _, expected := range expectedProviders {
		found := false
		for _, p := range providers {
			if p == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected VPN provider %s not found", expected)
		}
	}
}

func TestDetectionWeights(t *testing.T) {
	service := NewProxyDetectionService()

	weights := service.GetDetectionWeights()
	if len(weights) == 0 {
		t.Error("Should have detection weights")
	}

	newWeights := map[string]float64{
		"proxy_header": 30.0,
		"vpn_provider": 40.0,
	}
	service.SetDetectionWeights(newWeights)

	updatedWeights := service.GetDetectionWeights()
	if updatedWeights["proxy_header"] != 30.0 {
		t.Error("Detection weights should be updated")
	}
}

func TestClearExpiredBlacklist(t *testing.T) {
	service := NewProxyDetectionService()

	service.AddToBlacklist("192.168.1.1", 1*time.Second)

	time.Sleep(1100 * time.Millisecond)

	removed := service.ClearExpiredBlacklist()
	if removed == 0 {
		t.Error("Should have removed at least one expired blacklist entry")
	}
}

func TestGetCountryTimezone(t *testing.T) {
	service := NewProxyDetectionService()

	tests := []struct {
		country   string
		expected  string
	}{
		{"CN", "Asia/Shanghai"},
		{"US", "America/New_York"},
		{"JP", "Asia/Tokyo"},
		{"GB", "Europe/London"},
		{"XX", ""},
	}

	for _, tt := range tests {
		t.Run(tt.country, func(t *testing.T) {
			result := service.getCountryTimezone(tt.country)
			if result != tt.expected {
				t.Errorf("getCountryTimezone(%s) = %v, want %v", tt.country, result, tt.expected)
			}
		})
	}
}

func TestGetTimezoneOffset(t *testing.T) {
	service := NewProxyDetectionService()

	tests := []struct {
		timezone string
		expected int
	}{
		{"Asia/Shanghai", 480},
		{"America/New_York", -300},
		{"Europe/London", 0},
		{"Unknown/Zone", 0},
	}

	for _, tt := range tests {
		t.Run(tt.timezone, func(t *testing.T) {
			result := service.getTimezoneOffset(tt.timezone)
			if result != tt.expected {
				t.Errorf("getTimezoneOffset(%s) = %v, want %v", tt.timezone, result, tt.expected)
			}
		})
	}
}

func TestCheckTimezoneMismatch(t *testing.T) {
	service := NewProxyDetectionService()

	tests := []struct {
		name     string
		tzInfo   *TimezoneInfo
		country  string
		expected bool
	}{
		{
			name:     "Matching timezone and country",
			tzInfo:   &TimezoneInfo{Timezone: "Asia/Shanghai"},
			country:  "CN",
			expected: false,
		},
		{
			name:     "Mismatched timezone and country",
			tzInfo:   &TimezoneInfo{Timezone: "Asia/Tokyo"},
			country:  "US",
			expected: true,
		},
		{
			name:     "Nil timezone info",
			tzInfo:   nil,
			country:  "CN",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.checkTimezoneMismatch(tt.tzInfo, tt.country)
			if result != tt.expected {
				t.Errorf("checkTimezoneMismatch() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateDetectionAccuracy(t *testing.T) {
	service := NewProxyDetectionService()

	accuracy := service.CalculateDetectionAccuracy()
	if accuracy == nil {
		t.Fatal("CalculateDetectionAccuracy() returned nil")
	}

	if accuracy.TotalTests == 0 {
		t.Error("TotalTests should not be 0")
	}

	if accuracy.Accuracy < 0 || accuracy.Accuracy > 100 {
		t.Errorf("Accuracy should be between 0 and 100, got %v", accuracy.Accuracy)
	}

	if accuracy.Precision < 0 || accuracy.Precision > 100 {
		t.Errorf("Precision should be between 0 and 100, got %v", accuracy.Precision)
	}

	if accuracy.Recall < 0 || accuracy.Recall > 100 {
		t.Errorf("Recall should be between 0 and 100, got %v", accuracy.Recall)
	}
}

func TestDetectWebRTCLeak(t *testing.T) {
	service := NewProxyDetectionService()

	ctx := context.Background()
	info, err := service.DetectWebRTCLeak(ctx, "8.8.8.8")
	if err != nil {
		t.Fatalf("DetectWebRTCLeak() error = %v", err)
	}

	if info == nil {
		t.Fatal("DetectWebRTCLeak() returned nil")
	}
}

func TestAnalyzeTimezone(t *testing.T) {
	service := NewProxyDetectionService()

	info, err := service.AnalyzeTimezone("8.8.8.8", "America/New_York")
	if err != nil {
		t.Fatalf("AnalyzeTimezone() error = %v", err)
	}

	if info == nil {
		t.Fatal("AnalyzeTimezone() returned nil")
	}

	if info.Timezone != "America/New_York" {
		t.Errorf("Timezone = %v, want America/New_York", info.Timezone)
	}
}

func TestIPValidation(t *testing.T) {
	service := NewProxyDetectionService()

	tests := []struct {
		name         string
		ip           string
		expectValid  bool
		expectPrivate bool
	}{
		{"Valid public", "8.8.8.8", true, false},
		{"Valid private", "192.168.1.1", true, true},
		{"Invalid", "invalid-ip", false, false},
		{"IPv6", "2001:db8::1", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.ValidateAndEnrichIP(tt.ip)
			if result.IsValid != tt.expectValid {
				t.Errorf("IsValid = %v, want %v", result.IsValid, tt.expectValid)
			}
			if result.IsPrivate != tt.expectPrivate {
				t.Errorf("IsPrivate = %v, want %v", result.IsPrivate, tt.expectPrivate)
			}
		})
	}
}

func TestConnectionAnalysis_MultipleMeasurements(t *testing.T) {
	service := NewProxyDetectionService()

	measurements := []time.Duration{
		100 * time.Millisecond,
		110 * time.Millisecond,
		105 * time.Millisecond,
		115 * time.Millisecond,
		108 * time.Millisecond,
	}

	analysis := service.AnalyzeConnection(measurements)

	if analysis.Latency == 0 {
		t.Error("Latency should be calculated")
	}

	if len(measurements) > 1 && analysis.Jitter == 0 {
		t.Error("Jitter should be calculated for multiple measurements")
	}
}

func TestRiskAssessment_EmptyData(t *testing.T) {
	service := NewEnhancedProxyDetectionService()

	assessment := service.AssessIPRisk("8.8.8.8", map[string]string{}, nil)

	if assessment == nil {
		t.Fatal("Assessment should not be nil")
	}

	if assessment.OverallRisk < 0 || assessment.OverallRisk > 100 {
		t.Errorf("OverallRisk should be between 0 and 100, got %v", assessment.OverallRisk)
	}

	if assessment.Confidence < 0 || assessment.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %v", assessment.Confidence)
	}
}

func TestProxyDatabaseInitialization(t *testing.T) {
	db := NewProxyDatabase()

	if db.knownProxies == nil {
		t.Error("knownProxies should be initialized")
	}

	if db.knownVPNs == nil {
		t.Error("knownVPNs should be initialized")
	}

	if db.blacklist == nil {
		t.Error("blacklist should be initialized")
	}

	if db.vpnProviderRanges == nil {
		t.Error("vpnProviderRanges should be initialized")
	}

	if db.datacenterIPRanges == nil {
		t.Error("datacenterIPRanges should be initialized")
	}
}

func TestIPInfoParsing(t *testing.T) {
	info := &IPInfo{
		IP:          "8.8.8.8",
		Country:     "United States",
		CountryCode: "US",
		ISP:         "Google LLC",
		ASN:         "AS15169",
		Hosting:     true,
		Mobile:      false,
		Proxy:       false,
		VPN:         false,
		Tor:         false,
	}

	if info.IP != "8.8.8.8" {
		t.Errorf("IP = %v, want 8.8.8.8", info.IP)
	}

	if info.CountryCode != "US" {
		t.Errorf("CountryCode = %v, want US", info.CountryCode)
	}
}

func TestWebRTCInfo(t *testing.T) {
	info := &WebRTCInfo{
		LocalIPs:       []string{"192.168.1.100"},
		PublicIPs:      []string{"203.0.113.1"},
		RelayDetected:  true,
		LeakDetected:   true,
		InterfaceCount: 2,
	}

	if len(info.PublicIPs) != 1 {
		t.Errorf("PublicIPs length = %d, want 1", len(info.PublicIPs))
	}

	if !info.RelayDetected {
		t.Error("RelayDetected should be true")
	}
}

func TestTimezoneInfo(t *testing.T) {
	info := &TimezoneInfo{
		Timezone:      "Asia/Shanghai",
		OffsetMinutes: 480,
		OffsetString:  "GMT+8",
	}

	if info.Timezone != "Asia/Shanghai" {
		t.Errorf("Timezone = %v, want Asia/Shanghai", info.Timezone)
	}

	if info.OffsetMinutes != 480 {
		t.Errorf("OffsetMinutes = %d, want 480", info.OffsetMinutes)
	}
}

func TestConnectionAnalysis_Structure(t *testing.T) {
	analysis := &ConnectionAnalysis{
		Latency:         100 * time.Millisecond,
		Jitter:          15.5,
		PacketLoss:      0.5,
		Bandwidth:       100.5,
		IsProxyPattern:  false,
		IsVPNPattern:    true,
		AnomalyScore:    35.0,
		WebRTCLeakScore: 20.0,
	}

	if analysis.Latency != 100*time.Millisecond {
		t.Errorf("Latency = %v, want 100ms", analysis.Latency)
	}

	if analysis.AnomalyScore != 35.0 {
		t.Errorf("AnomalyScore = %v, want 35.0", analysis.AnomalyScore)
	}
}

func TestProxyDetectionResults(t *testing.T) {
	now := time.Now()
	detection := &ProxyDetection{
		IPAddress:         "8.8.8.8",
		IsProxy:           false,
		IsVPN:             true,
		IsTor:             false,
		IsDatacenter:      true,
		Confidence:        0.85,
		DetectionMethods:  []string{"datacenter_ip", "vpn_provider"},
		RiskLevel:         "high",
		Country:           "United States",
		ISP:               "Google LLC",
		ASN:               "AS15169",
		Hosting:           true,
		Mobile:            false,
		Score:             75.0,
		LastChecked:       now,
		ResponseTime:      50 * time.Millisecond,
		Headers:           map[string]string{},
		WebRTCLeakDetected: false,
		TimezoneMismatch:   false,
		VPNProvider:        "TestVPN",
		DatacenterProvider: "AWS",
	}

	if detection.IPAddress != "8.8.8.8" {
		t.Errorf("IPAddress = %v, want 8.8.8.8", detection.IPAddress)
	}

	if detection.IsVPN != true {
		t.Error("IsVPN should be true")
	}

	if detection.Confidence != 0.85 {
		t.Errorf("Confidence = %v, want 0.85", detection.Confidence)
	}

	if len(detection.DetectionMethods) != 2 {
		t.Errorf("DetectionMethods length = %d, want 2", len(detection.DetectionMethods))
	}
}

func TestRealtimeCheckResponse(t *testing.T) {
	resp := &RealtimeCheckResponse{
		IPAddress:          "8.8.8.8",
		IsSuspicious:      true,
		RiskLevel:         "medium",
		Score:             55.0,
		Reasons:           []string{"VPN detected", "Datacenter IP"},
		Indicators:        []string{"vpn_detected", "datacenter_ip"},
		Recommendations:   []string{"Enable enhanced verification"},
		WebRTCLeakDetected: true,
		TimezoneMismatch:   false,
	}

	if !resp.IsSuspicious {
		t.Error("IsSuspicious should be true")
	}

	if resp.Score != 55.0 {
		t.Errorf("Score = %v, want 55.0", resp.Score)
	}

	if len(resp.Reasons) != 2 {
		t.Errorf("Reasons length = %d, want 2", len(resp.Reasons))
	}
}

func TestVPNDetectionPattern(t *testing.T) {
	pattern := &VPNDetectionPattern{
		Name:        "test_pattern",
		Patterns:    []string{"pattern1", "pattern2"},
		Weight:      0.75,
		Description: "Test pattern description",
	}

	if pattern.Name != "test_pattern" {
		t.Errorf("Name = %v, want test_pattern", pattern.Name)
	}

	if pattern.Weight != 0.75 {
		t.Errorf("Weight = %v, want 0.75", pattern.Weight)
	}
}

func TestRiskFactor(t *testing.T) {
	factor := &RiskFactor{
		Category:    "vpn",
		Description: "VPN connection detected",
		Score:       0.85,
		Evidence:    []string{"ISP matches VPN provider"},
		Severity:    "high",
	}

	if factor.Category != "vpn" {
		t.Errorf("Category = %v, want vpn", factor.Category)
	}

	if factor.Score != 0.85 {
		t.Errorf("Score = %v, want 0.85", factor.Score)
	}

	if factor.Severity != "high" {
		t.Errorf("Severity = %v, want high", factor.Severity)
	}
}

func TestEnhancedIPRiskAssessment(t *testing.T) {
	now := time.Now()
	assessment := &EnhancedIPRiskAssessment{
		IPAddress:         "8.8.8.8",
		OverallRisk:       75.0,
		RiskLevel:         "high",
		RiskFactors:       []RiskFactor{},
		Confidence:        0.85,
		AssessmentMethods: []string{"proxy_assessment", "vpn_assessment"},
		LastAssessed:      now,
	}

	if assessment.IPAddress != "8.8.8.8" {
		t.Errorf("IPAddress = %v, want 8.8.8.8", assessment.IPAddress)
	}

	if assessment.OverallRisk != 75.0 {
		t.Errorf("OverallRisk = %v, want 75.0", assessment.OverallRisk)
	}

	if len(assessment.AssessmentMethods) != 2 {
		t.Errorf("AssessmentMethods length = %d, want 2", len(assessment.AssessmentMethods))
	}
}

func TestVPNProviderInfo(t *testing.T) {
	info := &VPNProviderInfo{
		Name:            "TestVPN",
		IPRanges:        []string{"1.0.0.0/8"},
		ASNPatterns:     []string{"AS12345"},
		DetectionWeight: 0.9,
	}

	if info.Name != "TestVPN" {
		t.Errorf("Name = %v, want TestVPN", info.Name)
	}

	if info.DetectionWeight != 0.9 {
		t.Errorf("DetectionWeight = %v, want 0.9", info.DetectionWeight)
	}
}

func TestCDNProviderInfo(t *testing.T) {
	info := &CDNProviderInfo{
		Name:            "TestCDN",
		IPRanges:        []string{"2.0.0.0/8"},
		HostingPatterns: []string{"testcdn.com"},
		IsDatacenter:    true,
	}

	if info.Name != "TestCDN" {
		t.Errorf("Name = %v, want TestCDN", info.Name)
	}

	if !info.IsDatacenter {
		t.Error("IsDatacenter should be true")
	}
}

func TestThreatIntelligence(t *testing.T) {
	ti := &ThreatIntelligence{
		KnownMaliciousIPs: map[string]bool{"1.2.3.4": true},
		KnownBotNets:      map[string]bool{"5.6.7.8": true},
		LastUpdated:       time.Now(),
	}

	if len(ti.KnownMaliciousIPs) != 1 {
		t.Errorf("KnownMaliciousIPs length = %d, want 1", len(ti.KnownMaliciousIPs))
	}

	if len(ti.KnownBotNets) != 1 {
		t.Errorf("KnownBotNets length = %d, want 1", len(ti.KnownBotNets))
	}
}

func TestIPValidationResult(t *testing.T) {
	result := &IPValidationResult{
		IsValid:         true,
		IP:              "8.8.8.8",
		IPVersion:       4,
		Error:           "",
		IsPrivate:       false,
		IsLoopback:      false,
		IsMulticast:     false,
		IsReserved:     false,
		NormalizedForm:  "8.8.8.8",
		DetectionMethods: []string{},
	}

	if !result.IsValid {
		t.Error("IsValid should be true")
	}

	if result.IPVersion != 4 {
		t.Errorf("IPVersion = %d, want 4", result.IPVersion)
	}
}

func TestDetectionAccuracy(t *testing.T) {
	acc := &DetectionAccuracy{
		TotalTests:        1000,
		CorrectDetections: 850,
		FalsePositives:    75,
		FalseNegatives:    75,
		Accuracy:          85.0,
		Precision:         91.89,
		Recall:            91.89,
	}

	if acc.TotalTests != 1000 {
		t.Errorf("TotalTests = %d, want 1000", acc.TotalTests)
	}

	if acc.Accuracy != 85.0 {
		t.Errorf("Accuracy = %v, want 85.0", acc.Accuracy)
	}
}

func TestIPParsing(t *testing.T) {
	testIPs := []string{
		"8.8.8.8",
		"192.168.1.1",
		"10.0.0.1",
		"172.16.0.1",
		"2001:4860:4860::8888",
		"::1",
	}

	for _, ipStr := range testIPs {
		parsedIP := net.ParseIP(ipStr)
		if parsedIP == nil {
			t.Errorf("Failed to parse IP: %s", ipStr)
		}
	}
}

func TestCIDRMatching(t *testing.T) {
	_ = NewProxyDetectionService()

	_, ipnet1, _ := net.ParseCIDR("52.84.0.0/15")
	testIP1 := net.ParseIP("52.84.0.1")
	if !ipnet1.Contains(testIP1) {
		t.Error("IP should be in CIDR range")
	}

	_, ipnet2, _ := net.ParseCIDR("104.16.0.0/12")
	testIP2 := net.ParseIP("104.20.0.1")
	if !ipnet2.Contains(testIP2) {
		t.Error("IP should be in CIDR range")
	}
}
