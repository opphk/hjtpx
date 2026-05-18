package service

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestEnhancedNetworkDetection_Init(t *testing.T) {
	nd := NewEnhancedNetworkDetection()
	if nd == nil {
		t.Fatal("NewEnhancedNetworkDetection() returned nil")
	}
	if len(nd.vpnProviders) == 0 {
		t.Error("Expected VPN providers to be initialized")
	}
	if len(nd.torExitNodes) == 0 {
		t.Error("Expected Tor exit nodes to be initialized")
	}
}

func TestEnhancedNetworkDetection_DetectNetwork_Basic(t *testing.T) {
	nd := NewEnhancedNetworkDetection()
	ctx := context.Background()
	headers := http.Header{}

	result, err := nd.DetectNetwork(ctx, "8.8.8.8", headers)
	if err != nil {
		t.Fatalf("DetectNetwork() returned error: %v", err)
	}
	if result == nil {
		t.Fatal("DetectNetwork() returned nil result")
	}
	if result.IPAddress != "8.8.8.8" {
		t.Errorf("Expected IPAddress to be '8.8.8.8', got '%s'", result.IPAddress)
	}
	if result.NetworkInfo.IPVersion != 4 {
		t.Errorf("Expected IPVersion to be 4, got %d", result.NetworkInfo.IPVersion)
	}
}

func TestEnhancedNetworkDetection_DetectNetwork_IPv6(t *testing.T) {
	nd := NewEnhancedNetworkDetection()
	ctx := context.Background()
	headers := http.Header{}

	result, err := nd.DetectNetwork(ctx, "2001:4860:4860::8888", headers)
	if err != nil {
		t.Fatalf("DetectNetwork() returned error: %v", err)
	}
	if result.NetworkInfo.IPVersion != 6 {
		t.Errorf("Expected IPVersion to be 6, got %d", result.NetworkInfo.IPVersion)
	}
}

func TestEnhancedNetworkDetection_DetectProxy_XForwardedFor(t *testing.T) {
	nd := NewEnhancedNetworkDetection()
	ctx := context.Background()
	headers := http.Header{}
	headers.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1, 172.16.0.1")

	result, err := nd.DetectNetwork(ctx, "8.8.8.8", headers)
	if err != nil {
		t.Fatalf("DetectNetwork() returned error: %v", err)
	}

	if !result.IsProxy {
		t.Error("Expected IsProxy to be true when X-Forwarded-For has multiple hops")
	}
	if result.ProxyDetails == nil {
		t.Fatal("Expected ProxyDetails to be set")
	}
	if result.ProxyDetails.HopCount != 3 {
		t.Errorf("Expected HopCount to be 3, got %d", result.ProxyDetails.HopCount)
	}
	if result.ProxyDetails.ProxyType != "multi-hop" {
		t.Errorf("Expected ProxyType to be 'multi-hop', got '%s'", result.ProxyDetails.ProxyType)
	}
	if result.ProxyDetails.AnonymityLevel != "high" {
		t.Errorf("Expected AnonymityLevel to be 'high', got '%s'", result.ProxyDetails.AnonymityLevel)
	}
}

func TestEnhancedNetworkDetection_DetectProxy_ViaHeader(t *testing.T) {
	testCases := []struct {
		name           string
		viaValue       string
		expectedProxy  bool
		expectedType   string
	}{
		{"nginx proxy", "1.1 nginx", true, "nginx"},
		{"cloudflare", "1.1 cloudflare", true, "cloudflare"},
		{"squid", "1.0 squid", true, "squid"},
		{"haproxy", "1.1 haproxy", true, "haproxy"},
		{"varnish", "1.1 varnish", true, "varnish"},
		{"unknown", "1.1 unknown-proxy", false, ""},
	}

	nd := NewEnhancedNetworkDetection()
	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			headers := http.Header{}
			headers.Set("Via", tc.viaValue)

			result, err := nd.DetectNetwork(ctx, "8.8.8.8", headers)
			if err != nil {
				t.Fatalf("DetectNetwork() returned error: %v", err)
			}

			if result.IsProxy != tc.expectedProxy {
				t.Errorf("Expected IsProxy to be %v, got %v", tc.expectedProxy, result.IsProxy)
			}
			if tc.expectedProxy && result.ProxyDetails.ProxyType != tc.expectedType {
				t.Errorf("Expected ProxyType to be '%s', got '%s'", tc.expectedType, result.ProxyDetails.ProxyType)
			}
		})
	}
}

func TestEnhancedNetworkDetection_DetectProxy_OtherHeaders(t *testing.T) {
	nd := NewEnhancedNetworkDetection()
	ctx := context.Background()

	tests := []struct {
		name           string
		headers        http.Header
		expectedProxy  bool
	}{
		{"X-ProxyChain", func() http.Header { h := http.Header{}; h.Set("X-ProxyChain", "1"); return h }(), true},
		{"X-ProxyId", func() http.Header { h := http.Header{}; h.Set("X-ProxyId", "proxy123"); return h }(), true},
		{"CF-Connecting-IP", func() http.Header { h := http.Header{}; h.Set("CF-Connecting-IP", "192.168.1.1"); return h }(), false},
		{"X-Real-IP", func() http.Header { h := http.Header{}; h.Set("X-Real-IP", "192.168.1.1"); return h }(), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := nd.DetectNetwork(ctx, "8.8.8.8", tc.headers)
			if err != nil {
				t.Fatalf("DetectNetwork() returned error: %v", err)
			}
			if result.IsProxy != tc.expectedProxy {
				t.Errorf("Expected IsProxy to be %v, got %v", tc.expectedProxy, result.IsProxy)
			}
		})
	}
}

func TestEnhancedNetworkDetection_DetectVPN_IPRange(t *testing.T) {
	nd := NewEnhancedNetworkDetection()
	ctx := context.Background()
	headers := http.Header{}

	testCases := []struct {
		name        string
		ip          string
		expectedVPN bool
	}{
		{"NordVPN IP", "45.33.1.1", true},
		{"ExpressVPN IP", "104.154.1.1", true},
		{"Surfshark IP", "172.104.1.1", true},
		{"ProtonVPN IP", "185.195.1.1", true},
		{"Normal IP", "8.8.8.8", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := nd.DetectNetwork(ctx, tc.ip, headers)
			if err != nil {
				t.Fatalf("DetectNetwork() returned error: %v", err)
			}
			if result.IsVPN != tc.expectedVPN {
				t.Errorf("Expected IsVPN to be %v for IP %s, got %v", tc.expectedVPN, tc.ip, result.IsVPN)
			}
		})
	}
}

func TestEnhancedNetworkDetection_DetectVPN_ByASN(t *testing.T) {
	nd := NewEnhancedNetworkDetection()

	testCases := []struct {
		name          string
		asn           int
		expectedVPN   bool
		expectedName  string
	}{
		{"NordVPN ASN", 201229, true, "Private Internet Access / NordVPN"},
		{"CyberGhost ASN", 207083, true, "CyberGhost"},
		{"ProtonVPN ASN", 19168, true, "ProtonVPN"},
		{"Mullvad ASN", 39189, true, "Mullvad"},
		{"Google ASN", 15169, false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isVPN, provider := nd.DetectVPNByASN(tc.asn)
			if isVPN != tc.expectedVPN {
				t.Errorf("Expected DetectVPNByASN(%d) to return isVPN=%v, got %v", tc.asn, tc.expectedVPN, isVPN)
			}
			if isVPN && provider != tc.expectedName {
				t.Errorf("Expected provider to be '%s', got '%s'", tc.expectedName, provider)
			}
		})
	}
}

func TestEnhancedNetworkDetection_DetectTor(t *testing.T) {
	nd := NewEnhancedNetworkDetection()
	ctx := context.Background()
	headers := http.Header{}

	testCases := []struct {
		name       string
		ip         string
		expected   bool
	}{
		{"Known Tor exit node", "128.31.0.34", true},
		{"Another Tor exit", "185.220.101.1", true},
		{"Normal IP", "8.8.8.8", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := nd.DetectNetwork(ctx, tc.ip, headers)
			if err != nil {
				t.Fatalf("DetectNetwork() returned error: %v", err)
			}
			if result.IsTor != tc.expected {
				t.Errorf("Expected IsTor to be %v for IP %s, got %v", tc.expected, tc.ip, result.IsTor)
			}
			if tc.expected && result.TorDetails == nil {
				t.Error("Expected TorDetails to be set for Tor exit node")
			}
			if tc.expected && result.IsProxy != true {
				t.Error("Expected IsProxy to be true for Tor exit node")
			}
		})
	}
}

func TestEnhancedNetworkDetection_IsTorExitNode(t *testing.T) {
	nd := NewEnhancedNetworkDetection()

	testCases := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"Known Tor exit", "128.31.0.34", true},
		{"Unknown IP", "192.168.1.1", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := nd.IsTorExitNode(tc.ip)
			if result != tc.expected {
				t.Errorf("IsTorExitNode(%s) = %v, want %v", tc.ip, result, tc.expected)
			}
		})
	}
}

func TestEnhancedNetworkDetection_DetectDatacenter(t *testing.T) {
	nd := NewEnhancedNetworkDetection()
	ctx := context.Background()
	headers := http.Header{}

	testCases := []struct {
		name         string
		ip           string
		expectedDC   bool
	}{
		{"AWS IP", "3.1.1.1", true},
		{"Google Cloud", "35.1.1.1", true},
		{"Azure IP", "13.107.1.1", true},
		{"DigitalOcean", "167.99.1.1", true},
		{"Linode", "45.79.1.1", true},
		{"Normal home IP", "192.168.1.100", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := nd.DetectNetwork(ctx, tc.ip, headers)
			if err != nil {
				t.Fatalf("DetectNetwork() returned error: %v", err)
			}
			if result.IsDatacenter != tc.expectedDC {
				t.Errorf("Expected IsDatacenter to be %v for IP %s, got %v", tc.expectedDC, tc.ip, result.IsDatacenter)
			}
		})
	}
}

func TestEnhancedNetworkDetection_CalculateRiskScore(t *testing.T) {
	nd := NewEnhancedNetworkDetection()

	testCases := []struct {
		name        string
		result      *NetworkDetectionResult
		expectedMin float64
		expectedMax float64
	}{
		{"Tor exit node", &NetworkDetectionResult{IsTor: true, GeoLocation: &GeoLocation{Country: "US"}}, 60, 60},
		{"VPN", &NetworkDetectionResult{IsVPN: true, GeoLocation: &GeoLocation{Country: "US"}}, 40, 40},
		{"Tor + VPN", &NetworkDetectionResult{IsTor: true, IsVPN: true, GeoLocation: &GeoLocation{Country: "US"}}, 100, 100},
		{"Proxy multi-hop", &NetworkDetectionResult{IsProxy: true, ProxyDetails: &ProxyDetails{ProxyType: "multi-hop"}, GeoLocation: &GeoLocation{Country: "US"}}, 35, 35},
		{"Proxy low", &NetworkDetectionResult{IsProxy: true, ProxyDetails: &ProxyDetails{ProxyType: "http"}, GeoLocation: &GeoLocation{Country: "US"}}, 25, 25},
		{"High risk country", &NetworkDetectionResult{GeoLocation: &GeoLocation{Country: "RU"}}, 20, 20},
		{"Medium risk country", &NetworkDetectionResult{GeoLocation: &GeoLocation{Country: "IN"}}, 10, 10},
		{"Datacenter", &NetworkDetectionResult{IsDatacenter: true, GeoLocation: &GeoLocation{Country: "US"}}, 20, 20},
		{"Mobile", &NetworkDetectionResult{IsMobile: true, GeoLocation: &GeoLocation{Country: "US"}}, 5, 5},
		{"Combined: Tor + VPN + High risk", &NetworkDetectionResult{IsTor: true, IsVPN: true, GeoLocation: &GeoLocation{Country: "RU"}}, 100, 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := nd.calculateRiskScore(tc.result)
			if score < tc.expectedMin || score > tc.expectedMax {
				t.Errorf("Expected risk score between %.2f and %.2f, got %.2f", tc.expectedMin, tc.expectedMax, score)
			}
		})
	}
}

func TestEnhancedNetworkDetection_DetermineRiskLevel(t *testing.T) {
	nd := NewEnhancedNetworkDetection()

	testCases := []struct {
		name     string
		score    float64
		expected string
	}{
		{"Critical", 90, "critical"},
		{"High", 70, "high"},
		{"Medium", 50, "medium"},
		{"Low", 25, "low"},
		{"None", 10, "none"},
		{"Exact critical", 80, "critical"},
		{"Exact high", 60, "high"},
		{"Exact medium", 40, "medium"},
		{"Exact low", 20, "low"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			level := nd.determineRiskLevel(tc.score)
			if level != tc.expected {
				t.Errorf("determineRiskLevel(%.2f) = %s, want %s", tc.score, level, tc.expected)
			}
		})
	}
}

func TestEnhancedNetworkDetection_TorExitNodeManagement(t *testing.T) {
	nd := NewEnhancedNetworkDetection()

	newNode := &TorExitNode{
		IP:           "10.10.10.10",
		ORPort:       443,
		DirectoryPort: 9030,
		Country:      "US",
		Bandwidth:    1000,
	}

	nd.AddTorExitNode(newNode)
	if !nd.IsTorExitNode("10.10.10.10") {
		t.Error("Expected added node to be detected")
	}

	node, exists := nd.GetTorExitNodeInfo("10.10.10.10")
	if !exists {
		t.Error("Expected node to exist")
	}
	if node.IP != "10.10.10.10" {
		t.Errorf("Expected IP to be '10.10.10.10', got '%s'", node.IP)
	}

	nd.RemoveTorExitNode("10.10.10.10")
	if nd.IsTorExitNode("10.10.10.10") {
		t.Error("Expected node to be removed")
	}
}

func TestEnhancedNetworkDetection_VPNProviderManagement(t *testing.T) {
	nd := NewEnhancedNetworkDetection()

	providers := nd.GetVPNProviders()
	if len(providers) == 0 {
		t.Error("Expected VPN providers to be returned")
	}

	newProvider := &VPNProvider{
		Name:    "TestVPN",
		IPRanges: []string{"192.168.0."},
		ASN:     []int{99999},
		KnownIPs: make(map[string]bool),
	}

	nd.UpdateVPNProvider("testvpn", newProvider)

	ctx := context.Background()
	headers := http.Header{}

	result, err := nd.DetectNetwork(ctx, "192.168.0.1", headers)
	if err != nil {
		t.Fatalf("DetectNetwork() returned error: %v", err)
	}
	if !result.IsVPN {
		t.Error("Expected updated VPN provider to be detected")
	}
}

func TestEnhancedNetworkDetection_BatchDetect(t *testing.T) {
	nd := NewEnhancedNetworkDetection()
	ctx := context.Background()
	headers := http.Header{}

	ips := []string{"8.8.8.8", "128.31.0.34", "45.33.1.1", "192.168.1.1"}

	results := nd.BatchDetect(ctx, ips, headers)

	if len(results) != len(ips) {
		t.Errorf("Expected %d results, got %d", len(ips), len(results))
	}

	for i, ip := range ips {
		if results[i].IPAddress != ip {
			t.Errorf("Expected result[%d].IPAddress to be '%s', got '%s'", i, ip, results[i].IPAddress)
		}
	}

	if !results[1].IsTor {
		t.Error("Expected Tor detection in batch results")
	}

	if !results[2].IsVPN {
		t.Error("Expected VPN detection in batch results")
	}
}

func TestEnhancedNetworkDetection_CacheManagement(t *testing.T) {
	nd := NewEnhancedNetworkDetection()
	ctx := context.Background()
	headers := http.Header{}

	_, _ = nd.DetectNetwork(ctx, "8.8.8.8", headers)

	nd.mu.RLock()
	cacheSize := len(nd.geoCache)
	nd.mu.RUnlock()

	if cacheSize == 0 {
		t.Error("Expected cache to have entries")
	}

	nd.CleanupCache()

	nd.mu.RLock()
	cacheSize = len(nd.geoCache)
	nd.mu.RUnlock()

	if cacheSize != 0 {
		t.Error("Expected cache to be empty after cleanup")
	}
}

func TestEnhancedNetworkDetection_GenerateRiskReport(t *testing.T) {
	nd := NewEnhancedNetworkDetection()

	testCases := []struct {
		name            string
		riskLevel       string
		riskScore       float64
		expectedIsThreat bool
	}{
		{"Critical", "critical", 90, true},
		{"High", "high", 70, true},
		{"Medium", "medium", 50, false},
		{"Low", "low", 25, false},
		{"None", "none", 10, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := &NetworkDetectionResult{
				IPAddress: "8.8.8.8",
				RiskLevel: tc.riskLevel,
				RiskScore: tc.riskScore,
				GeoLocation: &GeoLocation{
					City:    "Mountain View",
					Country: "US",
				},
			}

			report := nd.GenerateRiskReport(result)

			if report == nil {
				t.Fatal("Expected report to be generated")
			}
			if report.IPAddress != "8.8.8.8" {
				t.Errorf("Expected IPAddress to be '8.8.8.8', got '%s'", report.IPAddress)
			}
			if report.RiskLevel != tc.riskLevel {
				t.Errorf("Expected RiskLevel to be '%s', got '%s'", tc.riskLevel, report.RiskLevel)
			}
			if report.IsThreat != tc.expectedIsThreat {
				t.Errorf("Expected IsThreat to be %v, got %v", tc.expectedIsThreat, report.IsThreat)
			}
			if len(report.Recommendations) == 0 {
				t.Error("Expected recommendations to be present")
			}
		})
	}
}

func TestEnhancedNetworkDetection_CalculateConfidence(t *testing.T) {
	nd := NewEnhancedNetworkDetection()

	testCases := []struct {
		name        string
		result      *NetworkDetectionResult
		expectedMin float64
		expectedMax float64
	}{
		{"All flags set", &NetworkDetectionResult{
			IsProxy:      true,
			IsVPN:        true,
			IsTor:        true,
			IsDatacenter: true,
			DetectionMethods: []string{"method1", "method2", "method3"},
			GeoLocation:      &GeoLocation{Country: "US"},
			NetworkInfo:      &NetworkInfo{Latency: 30},
		}, 100, 100},
		{"No flags", &NetworkDetectionResult{
			DetectionMethods: []string{},
			GeoLocation:      &GeoLocation{},
			NetworkInfo:      &NetworkInfo{},
		}, 0, 0},
		{"Only proxy", &NetworkDetectionResult{
			IsProxy:          true,
			DetectionMethods: []string{"proxy"},
			GeoLocation:      &GeoLocation{Country: "US"},
		}, 48, 48},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			confidence := nd.calculateConfidence(tc.result)
			if confidence < tc.expectedMin || confidence > tc.expectedMax {
				t.Errorf("Expected confidence between %.2f and %.2f, got %.2f", tc.expectedMin, tc.expectedMax, confidence)
			}
		})
	}
}

func TestEnhancedNetworkDetection_ConcurrentAccess(t *testing.T) {
	nd := NewEnhancedNetworkDetection()
	ctx := context.Background()
	headers := http.Header{}

	var wg sync.WaitGroup
	numGoroutines := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = nd.DetectNetwork(ctx, "8.8.8.8", headers)
				_ = nd.IsTorExitNode("128.31.0.34")
				_ = nd.GetVPNProviders()
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("Test timed out, possible deadlock")
	}
}
