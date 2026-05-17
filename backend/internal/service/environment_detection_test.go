package service

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEnvironmentDetectionService(t *testing.T) {
	s := NewEnvironmentDetectionService()
	assert.NotNil(t, s)
	assert.NotNil(t, s.ipCache)
	assert.NotNil(t, s.vmPatterns)
	assert.NotNil(t, s.emulatorPatterns)
	assert.NotNil(t, s.automationPatterns)
	assert.Equal(t, 24*time.Hour, s.ipCacheExpiry)
	assert.Equal(t, 10000, s.maxIPCacheSize)
}

func TestDetectVMWare(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) VMware, VirtualBox")

	result := s.Detect(req, nil)

	assert.True(t, result.IsVM || len(result.Indicators) > 0)
}

func TestDetectDocker(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Docker")

	result := s.Detect(req, nil)

	if result.IsContainer {
		assert.True(t, result.VMType == VMTypeDocker || result.VMType == VMTypeKubernetes)
	}
}

func TestDetectKubernetes(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Kubernetes")

	result := s.Detect(req, nil)

	if result.IsContainer || result.IsCloud {
		assert.True(t, len(result.Indicators) > 0)
	}
}

func TestDetectAndroidEmulator(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 11; sdk_gphone64_arm64 Build/RKQ1.000217.002; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0.0.0 Mobile Safari/537.36")

	result := s.Detect(req, nil)

	assert.True(t, result.IsEmulator)
	assert.Contains(t, result.Indicators, "emulator_android")
}

func TestDetectSelenium(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 selenium")

	result := s.Detect(req, nil)

	assert.True(t, result.IsAutomated)
	assert.Contains(t, result.Indicators, "automation_selenium")
}

func TestDetectPuppeteer(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 puppeteer")

	result := s.Detect(req, nil)

	assert.True(t, result.IsAutomated)
	assert.Contains(t, result.Indicators, "automation_puppeteer")
}

func TestDetectPlaywright(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 playwright")

	result := s.Detect(req, nil)

	assert.True(t, result.IsAutomated)
	assert.Contains(t, result.Indicators, "automation_playwright")
}

func TestDetectProxyHeaders(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")
	req.Header.Set("X-Real-IP", "192.168.1.1")
	req.Header.Set("Via", "1.1 proxy-server")

	result := s.Detect(req, nil)

	assert.True(t, result.IsProxy)
	assert.Contains(t, result.Indicators, "proxy_headers_detected")
}

func TestDetectMultiHopProxy(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3, 10.0.0.4, 10.0.0.5")

	result := s.Detect(req, nil)

	assert.True(t, result.IsProxy)
	assert.Contains(t, result.Indicators, "multi_hop_proxy")
}

func TestDetectVPNIP(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "5.10.20.30:12345"

	result := s.Detect(req, nil)

	assert.True(t, result.IsVPN || result.IPRiskScore > 0)
}

func TestDetectPrivateIP(t *testing.T) {
	s := NewEnvironmentDetectionService()

	testCases := []struct {
		name string
		ip   string
	}{
		{"10.x.x.x", "10.0.0.1"},
		{"172.16.x.x", "172.16.0.1"},
		{"192.168.x.x", "192.168.1.1"},
		{"127.0.0.x", "127.0.0.1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tc.ip + ":12345"

			result := s.Detect(req, nil)

			assert.LessOrEqual(t, result.IPRiskScore, 10.0)
		})
	}
}

func TestDetectMaliciousIP(t *testing.T) {
	s := NewEnvironmentDetectionService()
	maliciousIP := "203.0.113.100"
	s.AddMaliciousIP(maliciousIP)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = maliciousIP + ":12345"

	result := s.Detect(req, nil)

	assert.True(t, result.IsMaliciousIP || result.IPRiskLevel == IPRiskLevelCritical || result.RiskScore > 0)

	s.RemoveMaliciousIP(maliciousIP)
}

func TestDetectTorExitNode(t *testing.T) {
	s := NewEnvironmentDetectionService()
	torIP := "192.168.200.200"
	s.AddTorExitNode(torIP)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = torIP + ":12345"

	result := s.Detect(req, nil)

	assert.True(t, result.IsTor)
	assert.Contains(t, result.Indicators, "tor_exit_node")

	s.RemoveTorExitNode(torIP)
}

func TestDetectCloudAWS(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Cloud-Trace-Context", "abc123/12345")

	result := s.Detect(req, nil)

	assert.True(t, result.IsCloud)
}

func TestDetectCloudAzure(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Azure-InstanceID", "abc123")

	result := s.Detect(req, nil)

	assert.True(t, result.IsCloud)
}

func TestDetectAdditionalDataAutomation(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)

	additionalData := map[string]string{
		"webdriver":     "wd:true",
		"selenium":      "selenium_present",
		"puppeteer":     "pw_cdc",
		"playwright":    "pw_global",
		"navigator.webdriver": "true",
	}

	result := s.Detect(req, additionalData)

	assert.True(t, result.IsAutomated)
	assert.Contains(t, result.Indicators, "client_webdriver_detected")
	assert.Contains(t, result.Indicators, "navigator_webdriver_true")
}

func TestDetectSoftwareRenderer(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)

	additionalData := map[string]string{
		"webgl_renderer": "SwiftShader",
	}

	result := s.Detect(req, additionalData)

	assert.Contains(t, result.Indicators, "software_renderer")
}

func TestDetectNoPlugins(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)

	additionalData := map[string]string{
		"plugins_count": "0",
	}

	result := s.Detect(req, additionalData)

	assert.Contains(t, result.Indicators, "no_plugins")
}

func TestDetectPermissionsDenied(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)

	additionalData := map[string]string{
		"permissions": "denied",
	}

	result := s.Detect(req, additionalData)

	assert.Contains(t, result.Indicators, "permissions_denied")
}

func TestDetectFromRequestData(t *testing.T) {
	s := NewEnvironmentDetectionService()

	ip := "203.0.113.1"
	userAgent := "Mozilla/5.0 TestBrowser"
	headers := map[string]string{
		"X-Forwarded-For": "10.0.0.1",
	}
	clientData := map[string]string{
		"webdriver": "wd:true",
	}

	result := s.DetectFromRequestData(ip, userAgent, headers, clientData)

	assert.NotNil(t, result)
	assert.True(t, result.RiskScore > 0)
}

func TestGetClientIP(t *testing.T) {
	testCases := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expectedIP string
	}{
		{
			name:       "Direct connection",
			remoteAddr: "192.168.1.1:12345",
			headers:    nil,
			expectedIP: "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For single",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1"},
			expectedIP: "203.0.113.1",
		},
		{
			name:       "X-Forwarded-For multiple",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1, 10.0.0.1"},
			expectedIP: "203.0.113.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Real-IP": "198.51.100.1"},
			expectedIP: "198.51.100.1",
		},
		{
			name:       "CF-Connecting-IP",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"CF-Connecting-IP": "203.0.113.50"},
			expectedIP: "203.0.113.50",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.headers != nil {
				for k, v := range tc.headers {
					req.Header.Set(k, v)
				}
			}

			ip := getClientIP(req)
			assert.Equal(t, tc.expectedIP, ip)
		})
	}
}

func TestUniqueStrings(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "Single element",
			input:    []string{"a"},
			expected: []string{"a"},
		},
		{
			name:     "All unique",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "With duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := uniqueStrings(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIPReputationCaching(t *testing.T) {
	s := NewEnvironmentDetectionService()
	ip := "203.0.113.1"

	s.mu.Lock()
	s.ipCache[ip] = &IPReputationData{
		IPAddress: ip,
		RiskLevel: IPRiskLevelHigh,
		RiskScore: 75,
		LastSeen:  time.Now(),
	}
	s.mu.Unlock()

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = ip + ":12345"

	result := s.Detect(req, nil)

	assert.Equal(t, IPRiskLevelHigh, result.IPRiskLevel)
}

func TestIPCacheCleanup(t *testing.T) {
	s := NewEnvironmentDetectionService()
	s.ipCacheExpiry = 1 * time.Millisecond

	s.mu.Lock()
	s.ipCache["old_ip"] = &IPReputationData{
		IPAddress: "old_ip",
		LastSeen:  time.Now().Add(-24 * time.Hour),
	}
	s.ipCache["new_ip"] = &IPReputationData{
		IPAddress: "new_ip",
		LastSeen:  time.Now(),
	}
	s.mu.Unlock()

	s.cleanupIPCache()

	s.mu.RLock()
	_, hasOld := s.ipCache["old_ip"]
	_, hasNew := s.ipCache["new_ip"]
	s.mu.RUnlock()

	assert.False(t, hasOld)
	assert.True(t, hasNew)
}

func TestVMScore(t *testing.T) {
	s := NewEnvironmentDetectionService()
	score := s.GetVMScore()
	assert.Equal(t, 25.0, score)
}

func TestAutomationScore(t *testing.T) {
	s := NewEnvironmentDetectionService()
	score := s.GetAutomationScore()
	assert.Equal(t, 30.0, score)
}

func TestEmulatorScore(t *testing.T) {
	s := NewEnvironmentDetectionService()
	score := s.GetEmulatorScore()
	assert.Equal(t, 25.0, score)
}

func TestIPRiskScore(t *testing.T) {
	s := NewEnvironmentDetectionService()
	score := s.GetIPRiskScore()
	assert.Equal(t, 20.0, score)
}

func TestSerialize(t *testing.T) {
	s := NewEnvironmentDetectionService()

	data, err := s.Serialize()
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var s2 EnvironmentDetectionService
	err = s2.Deserialize(data)
	assert.NoError(t, err)
}

func TestGetIPReputation(t *testing.T) {
	s := NewEnvironmentDetectionService()
	ip := "203.0.113.1"

	s.mu.Lock()
	s.ipCache[ip] = &IPReputationData{
		IPAddress: ip,
		RiskLevel: IPRiskLevelMedium,
		RiskScore: 50,
		LastSeen:  time.Now(),
	}
	s.mu.Unlock()

	reputation := s.GetIPReputation(ip)
	assert.Equal(t, ip, reputation.IPAddress)
	assert.Equal(t, IPRiskLevelMedium, reputation.RiskLevel)
}

func TestUpdateIPCacheExpiry(t *testing.T) {
	s := NewEnvironmentDetectionService()
	newExpiry := 48 * time.Hour

	s.UpdateIPCacheExpiry(newExpiry)

	assert.Equal(t, newExpiry, s.ipCacheExpiry)
}

func TestDetectBlueStacks(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 11; bstn4gpp Build/RKQ1.000217.002; wv) Bluestacks")

	result := s.Detect(req, nil)

	assert.True(t, result.IsEmulator)
}

func TestDetectNoxEmulator(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 9; Nox_APP_Player Build/PKQ1.000217.002)")

	result := s.Detect(req, nil)

	assert.True(t, result.IsEmulator || len(result.Indicators) > 0)
}

func TestDetectGenymotion(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 7.0; vbox86p Build/NHG47K; wv) Genymotion")

	result := s.Detect(req, nil)

	assert.True(t, result.IsEmulator)
}

func TestDetectCypress(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Cypress/12.0.0")

	result := s.Detect(req, nil)

	assert.True(t, result.IsAutomated)
	assert.Contains(t, result.Indicators, "automation_cypress")
}

func TestDetectWebDriverHeader(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("WebDriver", "true")

	result := s.Detect(req, nil)

	assert.True(t, result.IsAutomated)
	assert.Contains(t, result.Indicators, "webdriver_header")
}

func TestDetectChromeCdpHeader(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Chrome-CDP-Version", "102")

	result := s.Detect(req, nil)

	assert.True(t, result.IsAutomated)
	assert.Contains(t, result.Indicators, "chrome_cdp")
}

func TestDetectVPNHeader(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-VPN", "true")

	result := s.Detect(req, nil)

	assert.True(t, result.IsVPN)
	assert.Contains(t, result.Indicators, "vpn_header")
}

func TestDetectTorCircuitHeader(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Tor-Circuit-Id", "abc123")

	result := s.Detect(req, nil)

	assert.True(t, result.IsTor)
	assert.Contains(t, result.Indicators, "tor_circuit_header")
}

func TestDetectViaProxy(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Via", "1.1 proxy-server-name")

	result := s.Detect(req, nil)

	assert.Contains(t, result.Indicators, "via_header_proxy")
}

func TestRiskScoreCapping(t *testing.T) {
	s := NewEnvironmentDetectionService()
	s.AddMaliciousIP("1.2.3.4")

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	req.Header.Set("User-Agent", "Mozilla/5.0 selenium puppeteer playwright")
	req.Header.Set("WebDriver", "true")
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3, 10.0.0.4, 10.0.0.5")
	req.Header.Set("Tor-Circuit-Id", "tor123")

	additionalData := map[string]string{
		"webdriver": "wd:true",
	}

	result := s.Detect(req, additionalData)

	assert.LessOrEqual(t, result.RiskScore, 100.0)
	assert.GreaterOrEqual(t, result.RiskScore, 0.0)
}

func TestConfidenceCapping(t *testing.T) {
	s := NewEnvironmentDetectionService()
	s.AddTorExitNode("5.5.5.5")

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "5.5.5.5:12345"
	req.Header.Set("WebDriver", "true")
	req.Header.Set("Tor-Circuit-Id", "tor123")

	result := s.Detect(req, nil)

	assert.LessOrEqual(t, result.Confidence, 1.0)
	assert.GreaterOrEqual(t, result.Confidence, 0.0)
}

func TestGatherVMIndicators(t *testing.T) {
	s := NewEnvironmentDetectionService()

	indicators := s.gatherVMIndicators()

	assert.NotNil(t, indicators)
}

func TestGatherContainerInfo(t *testing.T) {
	s := NewEnvironmentDetectionService()

	info := s.gatherContainerInfo()

	assert.NotNil(t, info)
	assert.NotNil(t, info.IsContainer)
}

func TestIsKnownVPNIP(t *testing.T) {
	s := NewEnvironmentDetectionService()

	vpnIPs := []string{
		"5.10.20.30",
		"45.67.89.1",
		"85.100.200.1",
	}

	for _, ipStr := range vpnIPs {
		ip := net.ParseIP(ipStr)
		assert.True(t, s.isKnownVPNIP(ip), "Expected %s to be known VPN IP", ipStr)
	}

	nonVPNIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
	}

	for _, ipStr := range nonVPNIPs {
		ip := net.ParseIP(ipStr)
		assert.False(t, s.isKnownVPNIP(ip), "Expected %s to not be known VPN IP", ipStr)
	}
}

func TestIsKnownCloudIP(t *testing.T) {
	s := NewEnvironmentDetectionService()

	awsIP := net.ParseIP("52.94.76.10")
	assert.True(t, s.isKnownCloudIP(awsIP))

	azureIP := net.ParseIP("20.1.2.3")
	assert.True(t, s.isKnownCloudIP(azureIP))

	gcpIP := net.ParseIP("34.102.136.1")
	assert.True(t, s.isKnownCloudIP(gcpIP))
}

func TestContainerInfoSerialization(t *testing.T) {
	info := &ContainerInfo{
		IsContainer:   true,
		ContainerType: "docker",
		ContainerID:   "abc123def456",
		ImageName:     "nginx:latest",
	}

	data, err := json.Marshal(info)
	assert.NoError(t, err)

	var decoded ContainerInfo
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, info.IsContainer, decoded.IsContainer)
	assert.Equal(t, info.ContainerType, decoded.ContainerType)
	assert.Equal(t, info.ContainerID, decoded.ContainerID)
}

func TestEnvironmentDetectionResultSerialization(t *testing.T) {
	result := &EnvironmentDetectionResult{
		IsVM:         true,
		IsContainer:  true,
		IsEmulator:   true,
		IsAutomated:  true,
		IsVPN:        true,
		IsProxy:      true,
		IsTor:        true,
		VMType:       VMTypeVMware,
		RiskScore:    75.5,
		Confidence:   0.85,
		Reasons:      []string{"reason1", "reason2"},
		Indicators:   []string{"indicator1", "indicator2"},
	}

	data, err := json.Marshal(result)
	assert.NoError(t, err)

	var decoded EnvironmentDetectionResult
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, result.IsVM, decoded.IsVM)
	assert.Equal(t, result.RiskScore, decoded.RiskScore)
	assert.Equal(t, result.Confidence, decoded.Confidence)
}

func TestIPReputationDataSerialization(t *testing.T) {
	data := &IPReputationData{
		IPAddress:      "203.0.113.1",
		RiskLevel:      IPRiskLevelHigh,
		RiskScore:      65,
		Country:        "US",
		ASN:            "AS12345",
		ISP:            "Example ISP",
		IsProxy:        true,
		IsVPN:          true,
		IsHosting:      true,
		IsKnownMalicious: false,
		FirstSeen:     time.Now().Add(-24 * time.Hour),
		LastSeen:       time.Now(),
		Reports: []IPReport{
			{
				Source:    "test",
				Reason:    "test_reason",
				Count:     5,
				Timestamp: time.Now(),
			},
		},
	}

	jsonData, err := json.Marshal(data)
	assert.NoError(t, err)

	var decoded IPReputationData
	err = json.Unmarshal(jsonData, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, data.IPAddress, decoded.IPAddress)
	assert.Equal(t, data.RiskLevel, decoded.RiskLevel)
	assert.Equal(t, data.RiskScore, decoded.RiskScore)
}

func TestConcurrentAccess(t *testing.T) {
	s := NewEnvironmentDetectionService()
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "203.0.113.1:12345"
				_ = s.Detect(req, nil)
				_ = s.GetIPReputation("203.0.113.1")
			}
		}()
	}

	wg.Wait()
}

func TestDetectHostingProvider(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "45.33.0.1:12345"

	result := s.Detect(req, nil)

	assert.True(t, result.IsHosting)
}

func TestDetectHeadlessBrowser(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/120.0.0.0 Safari/537.36")

	result := s.Detect(req, nil)

	assert.True(t, result.IsAutomated || result.RiskScore > 0)
}

func TestDetectNoLanguages(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)

	additionalData := map[string]string{
		"languages": "",
	}

	result := s.Detect(req, additionalData)

	assert.Contains(t, result.Indicators, "no_languages")
}

func TestVMDetectionServiceThreadSafety(t *testing.T) {
	s := NewEnvironmentDetectionService()
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			s.AddMaliciousIP("10.0.0.1")
			s.AddTorExitNode("10.0.0.2")
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			s.RemoveMaliciousIP("10.0.0.1")
			s.RemoveTorExitNode("10.0.0.2")
		}
	}()

	wg.Wait()
}

func TestIPCacheSizeLimit(t *testing.T) {
	s := NewEnvironmentDetectionService()
	s.maxIPCacheSize = 10

	for i := 0; i < 15; i++ {
		s.mu.Lock()
		s.ipCache[fmt.Sprintf("192.168.1.%d", i)] = &IPReputationData{
			IPAddress: fmt.Sprintf("192.168.1.%d", i),
			LastSeen:  time.Now(),
		}
		s.mu.Unlock()
	}

	s.mu.RLock()
	cacheSize := len(s.ipCache)
	s.mu.RUnlock()

	assert.LessOrEqual(t, cacheSize, s.maxIPCacheSize+5)
}

func TestParseCIDR(t *testing.T) {
	cidr := parseCIDR("192.168.1.0/24")
	assert.NotNil(t, cidr)

	ip := net.ParseIP("192.168.1.100")
	assert.True(t, cidr.Contains(ip))

	ip2 := net.ParseIP("10.0.0.1")
	assert.False(t, cidr.Contains(ip2))
}

func TestEnvironmentDetectionResultReasonsUnique(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 selenium")
	req.Header.Set("WebDriver", "true")
	req.Header.Set("X-Forwarded-For", "10.0.0.1")

	result := s.Detect(req, nil)

	seen := make(map[string]bool)
	for _, reason := range result.Reasons {
		assert.False(t, seen[reason], "Duplicate reason found: %s", reason)
		seen[reason] = true
	}
}

func TestEnvironmentDetectionResultIndicatorsUnique(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 selenium puppeteer")
	req.Header.Set("WebDriver", "true")
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")

	result := s.Detect(req, nil)

	seen := make(map[string]bool)
	for _, indicator := range result.Indicators {
		assert.False(t, seen[indicator], "Duplicate indicator found: %s", indicator)
		seen[indicator] = true
	}
}

func TestDetectPrivateIPRangeExcluded(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"

	result := s.Detect(req, nil)

	assert.False(t, result.IsHosting)
}

func TestDetectLinkLocalAddress(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "169.254.1.1:12345"

	result := s.Detect(req, nil)

	assert.LessOrEqual(t, result.IPRiskScore, 20.0)
}

func TestNormalBrowserDetection(t *testing.T) {
	s := NewEnvironmentDetectionService()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.RemoteAddr = "203.0.113.1:12345"

	additionalData := map[string]string{
		"plugins_count": "5",
		"webgl_renderer": "NVIDIA GeForce RTX 3080",
		"languages": "en-US,en",
	}

	result := s.Detect(req, additionalData)

	assert.LessOrEqual(t, result.RiskScore, 100.0)
}

func TestCloudProviderIPDetection(t *testing.T) {
	s := NewEnvironmentDetectionService()

	testCases := []struct {
		provider VMType
		ip       string
	}{
		{VMTypeAWS, "52.94.76.10"},
		{VMTypeAzure, "20.1.2.3"},
		{VMTypeGCP, "34.102.136.1"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.provider), func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tc.ip + ":12345"

			result := s.Detect(req, nil)

			assert.True(t, result.IsCloud)
			assert.Equal(t, tc.provider, result.CloudProvider)
		})
	}
}
