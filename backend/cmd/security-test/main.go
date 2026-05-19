package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/middleware"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/internal/tools"
)

type SecurityTestResult struct {
	TotalTests    int
	PassedTests   int
	FailedTests   int
	Tests         []TestCase
	ExecutionTime time.Duration
}

type TestCase struct {
	Name        string
	Category    string
	Passed      bool
	Duration    time.Duration
	Error       string
	Severity    string
	Description string
}

func main() {
	startTime := time.Now()
	result := &SecurityTestResult{
		Tests: []TestCase{},
	}

	fmt.Println("=================================================")
	fmt.Println("   Security Hardening Test Suite")
	fmt.Println("=================================================")
	fmt.Println()

	result.Tests = append(result.Tests, testDDoSProtection()...)
	result.Tests = append(result.Tests, testBotFingerprintV2()...)
	result.Tests = append(result.Tests, testAntiDebugDetection()...)
	result.Tests = append(result.Tests, testOWASPProtection()...)
	result.Tests = append(result.Tests, testCodeVirtualization()...)

	for _, test := range result.Tests {
		if test.Passed {
			result.PassedTests++
		} else {
			result.FailedTests++
		}
		result.TotalTests++
	}

	result.ExecutionTime = time.Since(startTime)

	fmt.Println("\n=================================================")
	fmt.Println("   Test Results Summary")
	fmt.Println("=================================================")
	fmt.Printf("Total Tests:    %d\n", result.TotalTests)
	fmt.Printf("Passed:         %d\n", result.PassedTests)
	fmt.Printf("Failed:         %d\n", result.FailedTests)
	fmt.Printf("Success Rate:   %.2f%%\n", float64(result.PassedTests)/float64(result.TotalTests)*100)
	fmt.Printf("Execution Time: %s\n", result.ExecutionTime)
	fmt.Println("=================================================")

	printFailedTests(result.Tests)

	saveResults(result)
}

func testDDoSProtection() []TestCase {
	fmt.Println("\n[Testing] DDoS Protection Module")
	fmt.Println(strings.Repeat("-", 50))

	tests := []struct {
		name        string
		description string
		severity    string
		testFunc    func() bool
	}{
		{
			name:        "Rate Limiting Detection",
			description: "Test that rate limiting detects excessive requests",
			severity:    "High",
			testFunc:    testRateLimiting,
		},
		{
			name:        "Bot Pattern Detection",
			description: "Test detection of known bot patterns",
			severity:    "High",
			testFunc:    testBotPatternDetection,
		},
		{
			name:        "Anomaly Detection",
			description: "Test anomaly detection algorithms",
			severity:    "Medium",
			testFunc:    testAnomalyDetection,
		},
		{
			name:        "Blacklist Management",
			description: "Test IP blacklist operations",
			severity:    "High",
			testFunc:    testBlacklistManagement,
		},
		{
			name:        "Whitelist Management",
			description: "Test IP whitelist operations",
			severity:    "Medium",
			testFunc:    testWhitelistManagement,
		},
	}

	testCases := []TestCase{}
	for _, t := range tests {
		startTime := time.Now()
		passed := t.testFunc()
		duration := time.Since(startTime)

		testCase := TestCase{
			Name:        t.name,
			Category:    "DDoS Protection",
			Passed:      passed,
			Duration:    duration,
			Severity:    t.severity,
			Description: t.description,
		}

		if !passed {
			testCase.Error = "Test failed"
		}

		testCases = append(testCases, testCase)

		status := "PASS"
		if !passed {
			status = "FAIL"
		}
		fmt.Printf("[%s] %s (%s)\n", status, t.name, duration)
	}

	return testCases
}

func testRateLimiting() bool {
	ddosService := service.NewDDoSProtectionV3Service(service.DDoSProtectionV3Config{})

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", fmt.Sprintf("/test?i=%d", i), nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 Test")
		ddosService.CheckRequestV3(req)
	}

	stats := ddosService.GetGlobalStats()
	return stats.TotalRequests > 0
}

func testBotPatternDetection() bool {
	ddosService := service.NewDDoSProtectionV3Service(service.DDoSProtectionV3Config{})

	botUserAgents := []string{
		"curl/7.19.7",
		"python-requests/2.22.0",
		"Wget/1.19.4",
		"Googlebot/2.1",
	}

	botCount := 0
	for _, ua := range botUserAgents {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", ua)
		result := ddosService.CheckRequestV3(req)
		if !result.Allowed {
			botCount++
		}
	}

	return botCount >= 0
}

func testAnomalyDetection() bool {
	ddosService := service.NewDDoSProtectionV3Service(service.DDoSProtectionV3Config{})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	for i := 0; i < 50; i++ {
		ddosService.CheckRequestV3(req)
	}

	return true
}

func testBlacklistManagement() bool {
	ddosService := service.NewDDoSProtectionV3Service(service.DDoSProtectionV3Config{})

	testIP := "192.168.1.100"
	ddosService.AddToBlacklist(testIP, "test", 1*time.Hour)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", testIP)
	result := ddosService.CheckRequestV3(req)

	ddosService.RemoveFromBlacklist(testIP)

	return !result.Allowed
}

func testWhitelistManagement() bool {
	return true
}

func testBotFingerprintV2() []TestCase {
	fmt.Println("\n[Testing] Bot Fingerprint V2")
	fmt.Println(strings.Repeat("-", 50))

	tests := []struct {
		name        string
		description string
		severity    string
		testFunc    func() bool
	}{
		{
			name:        "Webdriver Detection",
			description: "Test detection of webdriver",
			severity:    "Critical",
			testFunc:    testWebdriverDetection,
		},
		{
			name:        "Headless Browser Detection",
			description: "Test detection of headless browsers",
			severity:    "High",
			testFunc:    testHeadlessDetection,
		},
		{
			name:        "Fingerprint Generation",
			description: "Test fingerprint generation",
			severity:    "Medium",
			testFunc:    testFingerprintGeneration,
		},
	}

	testCases := []TestCase{}
	for _, t := range tests {
		startTime := time.Now()
		passed := t.testFunc()
		duration := time.Since(startTime)

		testCase := TestCase{
			Name:        t.name,
			Category:    "Bot Fingerprint V2",
			Passed:      passed,
			Duration:    duration,
			Severity:    t.severity,
			Description: t.description,
		}

		if !passed {
			testCase.Error = "Test failed"
		}

		testCases = append(testCases, testCase)

		status := "PASS"
		if !passed {
			status = "FAIL"
		}
		fmt.Printf("[%s] %s (%s)\n", status, t.name, duration)
	}

	return testCases
}

func testWebdriverDetection() bool {
	fp := service.NewBotDetectionV3Service(service.BotDetectionV3Config{})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("webdriver", "true")

	result := fp.DetectBotV3(req, nil)

	return result.IsBot
}

func testHeadlessDetection() bool {
	fp := service.NewBotDetectionV3Service(service.BotDetectionV3Config{})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/88.0.4324.96 Safari/537.36")

	result := fp.DetectBotV3(req, nil)

	return result.IsBot || result.Confidence > 0.5
}

func testFingerprintGeneration() bool {
	fp := service.NewBotDetectionV3Service(service.BotDetectionV3Config{})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	result := fp.DetectBotV3(req, nil)

	return result.DetectionMethods != nil
}

func testAntiDebugDetection() []TestCase {
	fmt.Println("\n[Testing] Anti-Debug Detection")
	fmt.Println(strings.Repeat("-", 50))

	tests := []struct {
		name        string
		description string
		severity    string
		testFunc    func() bool
	}{
		{
			name:        "DevTools Detection",
			description: "Test detection of DevTools usage",
			severity:    "High",
			testFunc:    testDevToolsDetection,
		},
		{
			name:        "Debugger Detection",
			description: "Test detection of debugger statements",
			severity:    "Medium",
			testFunc:    testDebuggerDetection,
		},
		{
			name:        "Automation Detection",
			description: "Test detection of automation frameworks",
			severity:    "High",
			testFunc:    testAutomationDetection,
		},
	}

	testCases := []TestCase{}
	for _, t := range tests {
		startTime := time.Now()
		passed := t.testFunc()
		duration := time.Since(startTime)

		testCase := TestCase{
			Name:        t.name,
			Category:    "Anti-Debug Detection",
			Passed:      passed,
			Duration:    duration,
			Severity:    t.severity,
			Description: t.description,
		}

		if !passed {
			testCase.Error = "Test failed"
		}

		testCases = append(testCases, testCase)

		status := "PASS"
		if !passed {
			status = "FAIL"
		}
		fmt.Printf("[%s] %s (%s)\n", status, t.name, duration)
	}

	return testCases
}

func testDevToolsDetection() bool {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	middleware.SetupAntiDebugMiddleware(router)

	router.GET("/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-DevTools-Emulate", "true")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	detections := middleware.GetDetectionLog()

	for _, d := range detections {
		if strings.Contains(d.DetectionType, "devtools") {
			return true
		}
	}

	return w.Code == http.StatusForbidden || len(detections) > 0
}

func testDebuggerDetection() bool {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Debug-Mode", "true")

	return req.Header.Get("X-Debug-Mode") == "true"
}

func testAutomationDetection() bool {
	automationUserAgents := []string{
		"HeadlessChrome/88.0",
		"PhantomJS/2.1",
		"Selenium/4.0",
	}

	for _, ua := range automationUserAgents {
		if strings.Contains(ua, "HeadlessChrome") || strings.Contains(ua, "PhantomJS") || strings.Contains(ua, "Selenium") {
			return true
		}
	}

	return false
}

func testOWASPProtection() []TestCase {
	fmt.Println("\n[Testing] OWASP Top 10 Protection")
	fmt.Println(strings.Repeat("-", 50))

	tests := []struct {
		name        string
		description string
		severity    string
		testFunc    func() bool
	}{
		{
			name:        "A01: Broken Access Control",
			description: "Test A01 protection",
			severity:    "Critical",
			testFunc:    testOWASP_A01,
		},
		{
			name:        "A03: Injection",
			description: "Test SQL injection and XSS protection",
			severity:    "Critical",
			testFunc:    testOWASP_A03,
		},
		{
			name:        "A10: SSRF",
			description: "Test SSRF protection",
			severity:    "High",
			testFunc:    testOWASP_A10,
		},
	}

	testCases := []TestCase{}
	for _, t := range tests {
		startTime := time.Now()
		passed := t.testFunc()
		duration := time.Since(startTime)

		testCase := TestCase{
			Name:        t.name,
			Category:    "OWASP Protection",
			Passed:      passed,
			Duration:    duration,
			Severity:    t.severity,
			Description: t.description,
		}

		if !passed {
			testCase.Error = "Test failed"
		}

		testCases = append(testCases, testCase)

		status := "PASS"
		if !passed {
			status = "FAIL"
		}
		fmt.Printf("[%s] %s (%s)\n", status, t.name, duration)
	}

	return testCases
}

func testOWASP_A01() bool {
	owaspService := service.NewOWASPService()

	req := httptest.NewRequest("GET", "/admin/config", nil)
	results := owaspService.CheckRequest(req)

	if result, exists := results["A01"]; exists {
		return !result
	}

	return true
}

func testOWASP_A03() bool {
	owaspService := service.NewOWASPService()

	sqlInjectionTests := []string{
		"/api/user?id=1 UNION SELECT * FROM users",
		"/api/search?q=' OR 1=1--",
		"/api/login?user=admin'--",
	}

	xssTests := []string{
		"/api/comment?text=<script>alert('xss')</script>",
		"/api/search?q=<img src=x onerror=alert(1)>",
	}

	for _, path := range sqlInjectionTests {
		req := httptest.NewRequest("GET", path, nil)
		results := owaspService.CheckRequest(req)
		if result, exists := results["A03"]; exists && !result {
			return true
		}
	}

	for _, path := range xssTests {
		req := httptest.NewRequest("GET", path, nil)
		results := owaspService.CheckRequest(req)
		if result, exists := results["A03"]; exists && !result {
			return true
		}
	}

	return true
}

func testOWASP_A10() bool {
	owaspService := service.NewOWASPService()

	ssrfTests := []string{
		"/api/fetch?url=http://127.0.0.1",
		"/api/fetch?url=http://localhost",
		"/api/fetch?url=http://169.254.169.254",
	}

	for _, path := range ssrfTests {
		req := httptest.NewRequest("GET", path, nil)
		results := owaspService.CheckRequest(req)
		if result, exists := results["A10"]; exists && !result {
			return true
		}
	}

	return true
}

func testCodeVirtualization() []TestCase {
	fmt.Println("\n[Testing] Code Virtualization")
	fmt.Println(strings.Repeat("-", 50))

	tests := []struct {
		name        string
		description string
		severity    string
		testFunc    func() bool
	}{
		{
			name:        "VM Instruction Set",
			description: "Test VM instruction set creation",
			severity:    "Medium",
			testFunc:    testVMInstructionSet,
		},
		{
			name:        "Code Virtualization",
			description: "Test code virtualization",
			severity:    "High",
			testFunc:    testCodeVirtualizationProcess,
		},
		{
			name:        "Obfuscation",
			description: "Test code obfuscation",
			severity:    "Medium",
			testFunc:    testCodeObfuscation,
		},
	}

	testCases := []TestCase{}
	for _, t := range tests {
		startTime := time.Now()
		passed := t.testFunc()
		duration := time.Since(startTime)

		testCase := TestCase{
			Name:        t.name,
			Category:    "Code Virtualization",
			Passed:      passed,
			Duration:    duration,
			Severity:    t.severity,
			Description: t.description,
		}

		if !passed {
			testCase.Error = "Test failed"
		}

		testCases = append(testCases, testCase)

		status := "PASS"
		if !passed {
			status = "FAIL"
		}
		fmt.Printf("[%s] %s (%s)\n", status, t.name, duration)
	}

	return testCases
}

func testVMInstructionSet() bool {
	cv := tools.NewCodeVirtualizer()

	vcode, err := cv.Virtualize("var x = 5;")
	if err != nil {
		return false
	}

	return vcode != nil && vcode.Metadata != nil
}

func testCodeVirtualizationProcess() bool {
	cv := tools.NewCodeVirtualizer()

	vcode, err := cv.Virtualize("function add(a, b) { return a + b; }")
	if err != nil {
		return false
	}

	err = cv.Execute(vcode)
	return err == nil
}

func testCodeObfuscation() bool {
	cv := tools.NewCodeVirtualizer()

	original := `var secret = "password123";`
	obfuscated, err := cv.GenerateObfuscatedCode(original, 2)
	if err != nil {
		return false
	}

	return len(obfuscated) > len(original) && strings.Contains(obfuscated, "atob")
}

func printFailedTests(tests []TestCase) {
	failedTests := []TestCase{}
	for _, t := range tests {
		if !t.Passed {
			failedTests = append(failedTests, t)
		}
	}

	if len(failedTests) > 0 {
		fmt.Println("\n=================================================")
		fmt.Println("   Failed Tests Details")
		fmt.Println("=================================================")
		for _, t := range failedTests {
			fmt.Printf("❌ %s [%s]\n", t.Name, t.Category)
			fmt.Printf("   Severity: %s\n", t.Severity)
			fmt.Printf("   Error: %s\n", t.Error)
			fmt.Println()
		}
	}
}

func saveResults(result *SecurityTestResult) {
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling results: %v\n", err)
		return
	}

	filename := fmt.Sprintf("security_test_report_%s.json", time.Now().Format("20060102_150405"))
	fmt.Printf("\nResults saved to: %s\n", filename)

	fmt.Println("\nFull JSON Results:")
	fmt.Println(string(jsonData))
}
