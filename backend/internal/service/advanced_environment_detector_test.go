package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAdvancedDetector(t *testing.T) {
	detector := NewAdvancedDetector()
	assert.NotNil(t, detector)
	assert.NotNil(t, detector.config)
	assert.NotNil(t, detector.features)
	assert.True(t, detector.config.EnableCloudDetection)
	assert.True(t, detector.config.EnableVMDetection)
	assert.True(t, detector.config.EnableContainerDetection)
	assert.True(t, detector.config.EnableBrowserEngineDetection)
}

func TestGenerateSession(t *testing.T) {
	detector := NewAdvancedDetector()
	sessionID := detector.GenerateSession()
	assert.NotEmpty(t, sessionID)
	assert.Contains(t, sessionID, "adv_")
}

func TestGenerateMultipleSessions(t *testing.T) {
	detector := NewAdvancedDetector()
	id1 := detector.GenerateSession()
	id2 := detector.GenerateSession()
	assert.NotEqual(t, id1, id2)
}

func TestGetSession(t *testing.T) {
	detector := NewAdvancedDetector()
	sessionID := detector.GenerateSession()
	session := detector.GetSession(sessionID)
	assert.NotNil(t, session)
	assert.Equal(t, sessionID, session.ID)
}

func TestGetNonexistentSession(t *testing.T) {
	detector := NewAdvancedDetector()
	session := detector.GetSession("nonexistent_session_id")
	assert.Nil(t, session)
}

func TestUpdateSession(t *testing.T) {
	detector := NewAdvancedDetector()
	sessionID := detector.GenerateSession()

	updatedData := &AdvancedEnvironmentData{
		ID:         sessionID,
		RiskScore:  50,
		RiskLevel:  "medium",
		Features:   make(map[string]interface{}),
	}

	detector.UpdateSession(sessionID, updatedData)

	session := detector.GetSession(sessionID)
	assert.NotNil(t, session)
	assert.Equal(t, 50.0, session.RiskScore)
}

func TestAnalyzeBrowserInfo(t *testing.T) {
	detector := NewAdvancedDetector()

	testCases := []struct {
		name           string
		data           map[string]interface{}
		expectedEngine BrowserEngine
	}{
		{
			name:           "Chrome",
			data:           map[string]interface{}{"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"},
			expectedEngine: EngineBlink,
		},
		{
			name:           "Firefox",
			data:           map[string]interface{}{"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0"},
			expectedEngine: EngineGecko,
		},
		{
			name:           "Safari",
			data:           map[string]interface{}{"user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15"},
			expectedEngine: EngineWebKit,
		},
		{
			name:           "Edge",
			data:           map[string]interface{}{"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0"},
			expectedEngine: EngineBlink,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.analyzeBrowserInfo(tc.data)
			assert.Equal(t, tc.expectedEngine, result.Engine)
		})
	}
}

func TestDetectAutomationFramework(t *testing.T) {
	detector := NewAdvancedDetector()
	browser := &BrowserInfo{}

	testCases := []struct {
		name     string
		data     map[string]interface{}
		expected AutomationFramework
	}{
		{
			name:     "Selenium",
			data:     map[string]interface{}{"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Gecko/20100101 Firefox/120.0", "webdriver": "true"},
			expected: FrameworkSelenium,
		},
		{
			name:     "Headless Puppeteer",
			data:     map[string]interface{}{"user_agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/120.0.0.0 Safari/537.36 Puppeteer/21.0.0"},
			expected: FrameworkPuppeteer,
		},
		{
			name:     "PhantomJS",
			data:     map[string]interface{}{"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/534.34 PhantomJS/2.1.1 Safari/534.34"},
			expected: FrameworkPhantomJS,
		},
		{
			name:     "Normal Browser",
			data:     map[string]interface{}{"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"},
			expected: FrameworkNone,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.detectAutomationFramework(tc.data, browser)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAnalyzeWebGLFingerprint(t *testing.T) {
	detector := NewAdvancedDetector()

	testCases := []struct {
		name         string
		data         map[string]interface{}
		expectSoftware bool
	}{
		{
			name: "Normal GPU",
			data: map[string]interface{}{
				"webgl": "Google Inc.|ANGLE (Intel HD Graphics)",
			},
			expectSoftware: false,
		},
		{
			name: "SwiftShader",
			data: map[string]interface{}{
				"webgl": "Google Inc.|SwiftShader for Chrome",
			},
			expectSoftware: true,
		},
		{
			name: "LLVMpipe",
			data: map[string]interface{}{
				"webgl": "VMware, Inc.|llvmpipe (LLVM 12.0.0, 256 bits)",
			},
			expectSoftware: true,
		},
		{
			name: "VMware",
			data: map[string]interface{}{
				"webgl": "VMware, Inc.|VMware Virtual Platform Graphics Adapter",
			},
			expectSoftware: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.analyzeWebGLFingerprint(tc.data)
			assert.Equal(t, tc.expectSoftware, result.IsSoftware)
		})
	}
}

func TestAnalyzeFonts(t *testing.T) {
	detector := NewAdvancedDetector()

	testCases := []struct {
		name           string
		data           map[string]interface{}
		expectedCount  int
	}{
		{
			name: "Many fonts",
			data: map[string]interface{}{
				"fonts": "Arial,Helvetica,Times New Roman,Courier New,Verdana,Georgia,Comic Sans MS,Impact,Tahoma",
			},
			expectedCount: 9,
		},
		{
			name: "Few fonts",
			data: map[string]interface{}{
				"fonts": "Arial",
			},
			expectedCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.analyzeFonts(tc.data)
			assert.Equal(t, tc.expectedCount, result.Count)
		})
	}
}

func TestDetectVMEnvironment(t *testing.T) {
	detector := NewAdvancedDetector()

	testCases := []struct {
		name       string
		data       map[string]interface{}
		isVM       bool
	}{
		{
			name: "Normal environment",
			data: map[string]interface{}{
				"webgl": "Google Inc.|ANGLE (Intel)",
				"cpu_cores": "8",
			},
			isVM: false,
		},
		{
			name: "VirtualBox",
			data: map[string]interface{}{
				"webgl": "Google Inc.|VirtualBox Graphics Adapter",
				"cpu_cores": "4",
			},
			isVM: true,
		},
		{
			name: "Zero screen",
			data: map[string]interface{}{
				"screen": "0x0",
			},
			isVM: true,
		},
		{
			name: "Too many cores",
			data: map[string]interface{}{
				"cpu_cores": "128",
			},
			isVM: true,
		},
		{
			name: "Single core",
			data: map[string]interface{}{
				"cpu_cores": "1",
			},
			isVM: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.detectVMEnvironment(tc.data)
			assert.Equal(t, tc.isVM, result)
		})
	}
}

func TestDetectContainerEnvironment(t *testing.T) {
	detector := NewAdvancedDetector()

	testCases := []struct {
		name         string
		data         map[string]interface{}
		isContainer  bool
	}{
		{
			name: "Normal desktop",
			data: map[string]interface{}{
				"cpu_cores":     "8",
				"device_memory": "16",
			},
			isContainer: false,
		},
		{
			name: "Docker container",
			data: map[string]interface{}{
				"cpu_cores":     "2",
				"device_memory": "0.5",
			},
			isContainer: true,
		},
		{
			name: "Single core container",
			data: map[string]interface{}{
				"cpu_cores":     "1",
				"device_memory": "1",
			},
			isContainer: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.detectContainerEnvironment(tc.data)
			assert.Equal(t, tc.isContainer, result)
		})
	}
}

func TestDetectCloudProvider(t *testing.T) {
	detector := NewAdvancedDetector()

	testCases := []struct {
		name     string
		info     map[string]interface{}
		provider string
	}{
		{
			name: "AWS",
			info: map[string]interface{}{
				"ip": "54.123.45.67",
				"isp": "Amazon.com Inc.",
				"organization": "AWS EC2",
			},
			provider: "aws",
		},
		{
			name: "GCP",
			info: map[string]interface{}{
				"ip": "34.123.45.67",
				"isp": "Google LLC",
				"organization": "Google Cloud Platform",
			},
			provider: "gcp",
		},
		{
			name: "Azure",
			info: map[string]interface{}{
				"ip": "13.65.89.123",
				"isp": "Microsoft Corporation",
				"organization": "Microsoft Azure",
			},
			provider: "azure",
		},
		{
			name: "DigitalOcean",
			info: map[string]interface{}{
				"ip": "167.172.55.123",
				"isp": "DigitalOcean LLC",
			},
			provider: "digitalocean",
		},
		{
			name: "Unknown",
			info: map[string]interface{}{
				"ip": "203.0.113.42",
				"isp": "ISP Name",
			},
			provider: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.detectCloudProvider(tc.info)
			assert.Equal(t, tc.provider, result)
		})
	}
}

func TestClassifyIPType(t *testing.T) {
	detector := NewAdvancedDetector()

	testCases := []struct {
		name     string
		ip       string
		expected string
	}{
		{"Public", "8.8.8.8", "public"},
		{"Private 10.x", "10.0.0.1", "private"},
		{"Private 172.16.x", "172.16.0.1", "private"},
		{"Private 192.168.x", "192.168.1.1", "private"},
		{"Loopback", "127.0.0.1", "loopback"},
		{"Unspecified", "0.0.0.0", "unspecified"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.classifyIPType(tc.ip)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAnalyzeEnvironment(t *testing.T) {
	detector := NewAdvancedDetector()

	data := map[string]interface{}{
		"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"canvas": "test_canvas_data",
		"webgl": "Google Inc.|ANGLE (Intel)",
		"fonts": "Arial,Helvetica,Verdana",
		"cpu_cores": "8",
		"device_memory": "8",
	}

	result := detector.AnalyzeEnvironment(data)

	assert.NotNil(t, result)
	assert.NotNil(t, result.Features)
	assert.Greater(t, result.Confidence, 0.0)
	assert.LessOrEqual(t, result.RiskScore, 100.0)
}

func TestCalculateRiskScore(t *testing.T) {
	detector := NewAdvancedDetector()

	testCases := []struct {
		name           string
		result         *AdvancedEnvironmentData
		data           map[string]interface{}
		minScore       float64
	}{
		{
			name: "Normal browser",
			result: &AdvancedEnvironmentData{
				Automation:     FrameworkNone,
				EnvironmentType: EnvTypeNormal,
				Features:       make(map[string]interface{}),
				DetectionFlags: []string{},
			},
			data: map[string]interface{}{
				"canvas": "test",
				"webgl": "Google Inc.|ANGLE",
			},
			minScore: 0,
		},
		{
			name: "Selenium automation",
			result: &AdvancedEnvironmentData{
				Automation:     FrameworkSelenium,
				EnvironmentType: EnvTypeNormal,
				Features:       make(map[string]interface{}),
				DetectionFlags: []string{},
			},
			data:     map[string]interface{}{},
			minScore: 35,
		},
		{
			name: "Headless browser",
			result: &AdvancedEnvironmentData{
				Automation:     FrameworkNone,
				EnvironmentType: EnvTypeHeadless,
				Features:       make(map[string]interface{}),
				DetectionFlags: []string{},
			},
			data:     map[string]interface{}{},
			minScore: 20,
		},
		{
			name: "VM with Selenium",
			result: &AdvancedEnvironmentData{
				Automation:     FrameworkSelenium,
				EnvironmentType: EnvTypeVM,
				Features: map[string]interface{}{
					"webgl": &WebGLFingerprint{IsSoftware: true, SoftwareName: "swiftshader"},
				},
				DetectionFlags: []string{},
			},
			data:     map[string]interface{}{},
			minScore: 60,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := detector.calculateRiskScore(tc.result, tc.data)
			assert.GreaterOrEqual(t, score, tc.minScore)
			assert.LessOrEqual(t, score, 100.0)
		})
	}
}

func TestCalculateConfidence(t *testing.T) {
	detector := NewAdvancedDetector()

	testCases := []struct {
		name        string
		result      *AdvancedEnvironmentData
		data        map[string]interface{}
		minConf     float64
	}{
		{
			name: "No data",
			result: &AdvancedEnvironmentData{
				Automation:     FrameworkNone,
				EnvironmentType: EnvTypeNormal,
				DetectionFlags: []string{},
			},
			data:    map[string]interface{}{},
			minConf: 0.5,
		},
		{
			name: "With all checks",
			result: &AdvancedEnvironmentData{
				Automation:     FrameworkNone,
				EnvironmentType: EnvTypeNormal,
				DetectionFlags: []string{},
			},
			data: map[string]interface{}{
				"canvas":            "test",
				"webgl":            "test",
				"audio":            "test",
				"fonts":            "test",
				"webgl_extensions": "test",
			},
			minConf: 0.5,
		},
		{
			name: "With automation",
			result: &AdvancedEnvironmentData{
				Automation:     FrameworkSelenium,
				EnvironmentType: EnvTypeNormal,
				DetectionFlags: []string{},
			},
			data: map[string]interface{}{
				"canvas": "test",
			},
			minConf: 0.75,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conf := detector.calculateConfidence(tc.result, tc.data)
			assert.GreaterOrEqual(t, conf, tc.minConf)
			assert.LessOrEqual(t, conf, 1.0)
		})
	}
}

func TestGenerateFingerprintHash(t *testing.T) {
	detector := NewAdvancedDetector()

	result := &AdvancedEnvironmentData{
		BrowserEngine: EngineBlink,
		EngineVersion: "120",
		Features: map[string]interface{}{
			"webgl": &WebGLFingerprint{
				Vendor:   "Google Inc.",
				Renderer: "ANGLE (Intel)",
			},
			"canvas": &CanvasFingerprint{
				Hash: "test_hash",
			},
			"fonts": &FontAnalysis{
				Fonts: []string{"Arial", "Helvetica"},
			},
			"hardware": &HardwareProfile{
				HardwareConcurrency: 8,
				DeviceMemory:       8,
			},
		},
	}

	hash := detector.generateFingerprintHash(result)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 32)
}

func TestHashString(t *testing.T) {
	detector := NewAdvancedDetector()

	hash1 := detector.hashString("test_string")
	hash2 := detector.hashString("test_string")
	hash3 := detector.hashString("different_string")

	assert.Equal(t, hash1, hash2)
	assert.NotEqual(t, hash1, hash3)
	assert.Len(t, hash1, 32)
}

func TestGenerateDetectionScript(t *testing.T) {
	detector := NewAdvancedDetector()

	script := detector.GenerateDetectionScript()
	assert.NotEmpty(t, script)
	assert.Contains(t, script, "(function()")
	assert.Contains(t, script, "__results")
	assert.Contains(t, script, "browserEngine")
	assert.Contains(t, script, "canvasFingerprint")
	assert.Contains(t, script, "webGLFingerprint")
}

func TestDetectEnvironmentType(t *testing.T) {
	detector := NewAdvancedDetector()

	testCases := []struct {
		name           string
		data           map[string]interface{}
		browser        *BrowserInfo
		webgl          *WebGLFingerprint
		network        *NetworkAnalysis
		expected       EnvironmentType
	}{
		{
			name:     "Normal browser",
			data:     map[string]interface{}{"user_agent": "Mozilla/5.0 Chrome"},
			browser:  &BrowserInfo{Engine: EngineBlink},
			webgl:    &WebGLFingerprint{IsSoftware: false},
			network:  &NetworkAnalysis{CloudIP: false},
			expected: EnvTypeNormal,
		},
		{
			name:     "Headless browser",
			data:     map[string]interface{}{"user_agent": "Mozilla/5.0 HeadlessChrome"},
			browser:  &BrowserInfo{Engine: EngineBlink},
			webgl:    &WebGLFingerprint{IsSoftware: false},
			network:  &NetworkAnalysis{},
			expected: EnvTypeHeadless,
		},
		{
			name:     "VM with SwiftShader",
			data:     map[string]interface{}{"user_agent": "Mozilla/5.0 Chrome"},
			browser:  &BrowserInfo{Engine: EngineBlink},
			webgl:    &WebGLFingerprint{IsSoftware: true, SoftwareName: "swiftshader"},
			network:  &NetworkAnalysis{},
			expected: EnvTypeVM,
		},
		{
			name:     "Cloud VM",
			data:     map[string]interface{}{"user_agent": "Mozilla/5.0 Chrome"},
			browser:  &BrowserInfo{Engine: EngineBlink},
			webgl:    &WebGLFingerprint{IsSoftware: false},
			network:  &NetworkAnalysis{CloudIP: true},
			expected: EnvTypeCloudVM,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.detectEnvironmentType(tc.data, tc.browser, tc.webgl, tc.network)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAdvancedDetectorSingleton(t *testing.T) {
	detector1 := GetAdvancedDetector()
	detector2 := GetAdvancedDetector()
	assert.Same(t, detector1, detector2)
}
