package handler

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

type AdvancedDetectionRequest struct {
	SessionID    string                 `json:"session_id" binding:"required"`
	Fingerprint  string                 `json:"fingerprint"`
	Data         map[string]interface{} `json:"data" binding:"required"`
	Timestamp    int64                  `json:"timestamp"`
	UserAgent    string                 `json:"user_agent"`
	IPAddress    string                 `json:"ip_address"`
	RequestHeaders map[string]string     `json:"request_headers"`
}

type AdvancedDetectionResponse struct {
	Success        bool                          `json:"success"`
	SessionID      string                        `json:"session_id"`
	RiskScore      float64                       `json:"risk_score"`
	Confidence     float64                       `json:"confidence"`
	RiskLevel      string                        `json:"risk_level"`
	EnvironmentType string                       `json:"environment_type"`
	DetectionFlags []string                      `json:"detection_flags"`
	FingerprintHash string                       `json:"fingerprint_hash"`
	BrowserEngine  string                        `json:"browser_engine"`
	EngineVersion string                        `json:"engine_version"`
	IsHeadless    bool                          `json:"is_headless"`
	IsVM          bool                          `json:"is_vm"`
	IsCloudVM     bool                          `json:"is_cloud_vm"`
	IsContainer   bool                          `json:"is_container"`
	CloudProvider string                        `json:"cloud_provider,omitempty"`
	Features      map[string]interface{}         `json:"features"`
}

type AdvancedScriptResponse struct {
	Success bool   `json:"success"`
	Script string `json:"script"`
}

var (
	advancedDetector = service.GetAdvancedDetector()
	advSessionMutex  sync.RWMutex
	sessions         = make(map[string]*service.AdvancedEnvironmentData)
)

func init() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			advancedDetector.CleanupSessions()
		}
	}()
}

func GetAdvancedDetectionScript(c *gin.Context) {
	script := advancedDetector.GenerateDetectionScript()

	sessionID := advancedDetector.GenerateSession()

	scriptTemplate := fmt.Sprintf(`(function(){
var __sessionId="%s";
var __uuid=Math.random().toString(36).substr(2,9);
var __timestamp=Date.now();
%s
var __data={
session_id:__sessionId,
fingerprint:__fp,
data:__results,
timestamp:__timestamp,
user_agent:navigator.userAgent,
ip_address:"",
request_headers:{}
};
fetch("/api/v1/detect/advanced/submit",{
method:"POST",
headers:{"Content-Type":"application/json"},
body:JSON.stringify(__data)
}).then(function(r){return r.json()}).then(function(r){window.__advDetectionResult=r}).catch(function(e){console.error(e)});
})();`, sessionID, script)

	c.Header("Content-Type", "application/javascript")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.String(http.StatusOK, scriptTemplate)
}

func SubmitAdvancedDetection(c *gin.Context) {
	var req AdvancedDetectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	if req.Data == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "data field is required",
		})
		return
	}

	if req.RequestHeaders == nil {
		req.RequestHeaders = extractRequestHeaders(c)
	}

	if req.UserAgent == "" {
		req.UserAgent = c.GetHeader("User-Agent")
	}

	if req.IPAddress == "" {
		req.IPAddress = c.ClientIP()
	}

	if req.SessionID == "" {
		req.SessionID = advancedDetector.GenerateSession()
	}

	analysisData := req.Data
	analysisData["user_agent"] = req.UserAgent
	analysisData["ip_address"] = req.IPAddress
	analysisData["request_headers"] = req.RequestHeaders

	result := advancedDetector.AnalyzeEnvironment(analysisData)
	result.SessionID = req.SessionID

	advSessionMutex.Lock()
	sessions[req.SessionID] = result
	advSessionMutex.Unlock()

	advancedDetector.UpdateSession(result.ID, result)

	fpHash := md5.Sum([]byte(req.Fingerprint + result.FingerprintHash))
	result.FingerprintHash = hex.EncodeToString(fpHash[:])[:16]

	response := AdvancedDetectionResponse{
		Success:         true,
		SessionID:       result.SessionID,
		RiskScore:       result.RiskScore,
		Confidence:      result.Confidence,
		RiskLevel:       getRiskLevel(result.RiskScore),
		EnvironmentType: string(result.EnvironmentType),
		DetectionFlags:  result.DetectionFlags,
		FingerprintHash: result.FingerprintHash,
		BrowserEngine:   string(result.BrowserEngine),
		EngineVersion:   result.EngineVersion,
		IsHeadless:     result.IsHeadless,
		IsVM:           result.IsVM,
		IsCloudVM:      result.IsCloudVM,
		IsContainer:    result.IsContainer,
		Features:       result.Features,
	}

	if result.CloudProvider != "" {
		response.CloudProvider = result.CloudProvider
	}

	c.JSON(http.StatusOK, response)
}

func GetAdvancedDetectionResult(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "session_id is required",
		})
		return
	}

	advSessionMutex.RLock()
	session, exists := sessions[sessionID]
	advSessionMutex.RUnlock()

	if !exists {
		session = advancedDetector.GetSession(sessionID)
	}

	if session == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "session not found",
		})
		return
	}

	response := AdvancedDetectionResponse{
		Success:         true,
		SessionID:       session.SessionID,
		RiskScore:       session.RiskScore,
		Confidence:      session.Confidence,
		RiskLevel:       getRiskLevel(session.RiskScore),
		EnvironmentType: string(session.EnvironmentType),
		DetectionFlags:  session.DetectionFlags,
		FingerprintHash: session.FingerprintHash,
		BrowserEngine:   string(session.BrowserEngine),
		EngineVersion:   session.EngineVersion,
		IsHeadless:     session.IsHeadless,
		IsVM:           session.IsVM,
		IsCloudVM:      session.IsCloudVM,
		IsContainer:    session.IsContainer,
		Features:       session.Features,
	}

	if session.CloudProvider != "" {
		response.CloudProvider = session.CloudProvider
	}

	c.JSON(http.StatusOK, response)
}

func extractRequestHeaders(c *gin.Context) map[string]string {
	headers := make(map[string]string)
	headerNames := []string{
		"User-Agent", "Accept", "Accept-Language", "Accept-Encoding",
		"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP", "Via",
		"X-Varnish", "True-Client-IP", "DNT", "Upgrade-Insecure-Requests",
		"Sec-Fetch-Dest", "Sec-Fetch-Mode", "Sec-Fetch-Site",
	}

	for _, name := range headerNames {
		val := c.GetHeader(name)
		if val != "" {
			headers[name] = val
		}
	}

	return headers
}

func getRiskLevel(score float64) string {
	if score >= 80 {
		return "critical"
	} else if score >= 60 {
		return "high"
	} else if score >= 40 {
		return "medium"
	} else if score >= 20 {
		return "low"
	}
	return "minimal"
}

func AnalyzeBrowserEngine(c *gin.Context) {
	var req struct {
		UserAgent string `json:"user_agent" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "user_agent is required",
		})
		return
	}

	result := analyzeBrowserEngineDetails(req.UserAgent)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func analyzeBrowserEngineDetails(ua string) map[string]interface{} {
	result := make(map[string]interface{})

	uaLower := strings.ToLower(ua)

	result["raw"] = ua

	engine := "unknown"
	engineVersion := ""
	browser := "unknown"
	browserVersion := ""

	edgeChromium := regexp.MustCompile(`edg[e]?\/(\d+[\.\d]*)`)
	if matches := edgeChromium.FindStringSubmatch(ua); len(matches) > 1 {
		engine = "blink"
		browser = "edge"
		browserVersion = matches[1]
		engineVersion = extractBlinkVersion(ua)
	}

	if engine == "unknown" {
		chrome := regexp.MustCompile(`chrome\/(\d+[\.\d]*)`)
		if matches := chrome.FindStringSubmatch(ua); len(matches) > 1 {
			if strings.Contains(uaLower, "edg/") {
				engine = "blink"
				browser = "edge"
			} else if strings.Contains(uaLower, "opr/") {
				engine = "blink"
				browser = "opera"
				browserVersion = matches[1]
				engineVersion = extractBlinkVersion(ua)
			} else if strings.Contains(uaLower, "brave") {
				engine = "blink"
				browser = "brave"
				browserVersion = matches[1]
				engineVersion = extractBlinkVersion(ua)
			} else {
				engine = "blink"
				browser = "chrome"
				browserVersion = matches[1]
				engineVersion = extractBlinkVersion(ua)
			}
		}
	}

	if engine == "unknown" {
		firefox := regexp.MustCompile(`firefox\/(\d+[\.\d]*)`)
		if matches := firefox.FindStringSubmatch(ua); len(matches) > 1 {
			engine = "gecko"
			browser = "firefox"
			browserVersion = matches[1]
			rv := regexp.MustCompile(`rv:(\d+[\.\d]*)`)
			if rvMatches := rv.FindStringSubmatch(ua); len(rvMatches) > 1 {
				engineVersion = rvMatches[1]
			}
		}
	}

	if engine == "unknown" {
		webkit := regexp.MustCompile(`applewebkit\/(\d+[\.\d]*)`)
		if matches := webkit.FindStringSubmatch(ua); len(matches) > 1 {
			engine = "webkit"
			engineVersion = matches[1]

			safari := regexp.MustCompile(`version\/(\d+[\.\d]*)`)
			if matches := safari.FindStringSubmatch(ua); len(matches) > 1 {
				browser = "safari"
				browserVersion = matches[1]
			}
		}
	}

	if engine == "unknown" {
		trident := regexp.MustCompile(`trident\/(\d+[\.\d]*)`)
		if matches := trident.FindStringSubmatch(ua); len(matches) > 1 {
			engine = "trident"
			browser = "ie"
			browserVersion = matches[1]
			engineVersion = matches[1]
		}
	}

	if engine == "unknown" && strings.Contains(uaLower, "phantom") {
		engine = "webkit"
		browser = "phantomjs"
	}

	if engine == "unknown" && strings.Contains(uaLower, "playwright") {
		engine = "blink"
		browser = "playwright"
	}

	if engine == "unknown" && strings.Contains(uaLower, "puppeteer") {
		engine = "blink"
		browser = "puppeteer"
	}

	result["engine"] = engine
	result["engine_version"] = engineVersion
	result["browser"] = browser
	result["browser_version"] = browserVersion

	os := extractOS(ua)
	result["os"] = os.name
	result["os_version"] = os.version
	result["os_family"] = os.family

	result["mobile"] = strings.Contains(uaLower, "mobile") || strings.Contains(uaLower, "android")
	result["tablet"] = strings.Contains(uaLower, "tablet") || strings.Contains(uaLower, "ipad")
	result["bot"] = isBot(uaLower)

	return result
}

func extractBlinkVersion(ua string) string {
	chromeMatch := regexp.MustCompile(`chrome\/(\d+[\.\d]*)`).FindStringSubmatch(ua)
	if len(chromeMatch) > 1 {
		version := chromeMatch[1]
		parts := strings.Split(version, ".")
		if len(parts) > 0 {
			major, _ := fmt.Sscanf(parts[0], "%d", new(int))
			if major > 0 {
				return parts[0]
			}
		}
	}
	return ""
}

func extractOS(ua string) struct {
	name    string
	version string
	family  string
} {
	uaLower := strings.ToLower(ua)

	if strings.Contains(uaLower, "windows nt 10") || strings.Contains(uaLower, "windows nt 11") {
		return struct {
			name    string
			version string
			family  string
		}{"Windows", "10/11", "Windows"}
	}
	if strings.Contains(uaLower, "windows nt 6.3") {
		return struct {
			name    string
			version string
			family  string
		}{"Windows", "8.1", "Windows"}
	}
	if strings.Contains(uaLower, "windows nt 6.2") {
		return struct {
			name    string
			version string
			family  string
		}{"Windows", "8", "Windows"}
	}
	if strings.Contains(uaLower, "windows nt 6.1") {
		return struct {
			name    string
			version string
			family  string
		}{"Windows", "7", "Windows"}
	}
	if strings.Contains(uaLower, "windows") {
		return struct {
			name    string
			version string
			family  string
		}{"Windows", "Unknown", "Windows"}
	}

	if strings.Contains(uaLower, "mac os x") {
		re := regexp.MustCompile(`mac os x (\d+[_\.\d]*)`)
		if matches := re.FindStringSubmatch(ua); len(matches) > 1 {
			version := strings.ReplaceAll(matches[1], "_", ".")
			return struct {
				name    string
				version string
				family  string
			}{"macOS", version, "macOS"}
		}
		return struct {
			name    string
			version string
			family  string
		}{"macOS", "Unknown", "macOS"}
	}

	if strings.Contains(uaLower, "iphone os") || strings.Contains(uaLower, "ipad") {
		re := regexp.MustCompile(`os (\d+[_\d]*)`)
		if matches := re.FindStringSubmatch(ua); len(matches) > 1 {
			version := strings.ReplaceAll(matches[1], "_", ".")
			if strings.Contains(uaLower, "ipad") {
				return struct {
					name    string
					version string
					family  string
				}{"iPadOS", version, "iOS/iPadOS"}
			}
			return struct {
				name    string
				version string
				family  string
			}{"iOS", version, "iOS/iPadOS"}
		}
	}

	if strings.Contains(uaLower, "android") {
		re := regexp.MustCompile(`android (\d+[\.\d]*)`)
		if matches := re.FindStringSubmatch(ua); len(matches) > 1 {
			return struct {
				name    string
				version string
				family  string
			}{"Android", matches[1], "Android"}
		}
		return struct {
			name    string
			version string
			family  string
		}{"Android", "Unknown", "Android"}
	}

	if strings.Contains(uaLower, "linux") {
		return struct {
			name    string
			version string
			family  string
		}{"Linux", "Unknown", "Linux"}
	}

	if strings.Contains(uaLower, "cros") {
		return struct {
			name    string
			version string
			family  string
		}{"Chrome OS", "Unknown", "Chrome OS"}
	}

	return struct {
		name    string
		version string
		family  string
	}{"Unknown", "Unknown", "Unknown"}
}

func isBot(uaLower string) bool {
	botIndicators := []string{"bot", "crawl", "spider", "slurp", "mediapartners", "googlebot", "bingbot", "yandex", "baiduspider", "facebookexternalhit", "twitterbot", "linkedinbot", "whatsapp", "telegram"}
	for _, indicator := range botIndicators {
		if strings.Contains(uaLower, indicator) {
			return true
		}
	}
	return false
}

type detectVMReq struct {
	WebGLRenderer string
	ScreenSize    string
	CPUCores      int
	DeviceMemory  float64
	UserAgent     string
}

type detectCloudReq struct {
	IPAddress    string
	UserAgent    string
	ISP          string
	Organization string
}

type detectContainerReq struct {
	CPUCores     int
	DeviceMemory float64
	UserAgent    string
	Platform     string
}

type detectHeadlessReq struct {
	UserAgent         string
	NavigatorWebdriver bool
	ChromeRuntime     bool
	PluginsCount      int
	Languages         []string
	Permissions       map[string]string
}

type analyzeWebGLReq struct {
	WebGLData      string
	Extensions     []string
	MaxTextureSize int
	Renderer       string
	Vendor         string
}

func DetectVMEnvironment(c *gin.Context) {
	var req struct {
		WebGLRenderer string `json:"webgl_renderer"`
		ScreenSize   string `json:"screen_size"`
		CPUCores     int    `json:"cpu_cores"`
		DeviceMemory float64 `json:"device_memory"`
		UserAgent    string `json:"user_agent"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request",
		})
		return
	}

	result := detectVMIndicators(detectVMReq{
		WebGLRenderer: req.WebGLRenderer,
		ScreenSize:    req.ScreenSize,
		CPUCores:      req.CPUCores,
		DeviceMemory:  req.DeviceMemory,
		UserAgent:     req.UserAgent,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func detectVMIndicators(req struct {
	WebGLRenderer string
	ScreenSize    string
	CPUCores      int
	DeviceMemory  float64
	UserAgent     string
}) map[string]interface{} {
	result := make(map[string]interface{})
	result["is_vm"] = false
	result["vm_type"] = ""
	result["indicators"] = []string{}

	indicators := []string{}

	rendererLower := strings.ToLower(req.WebGLRenderer)
	vmKeywords := []string{
		"virtualbox", "vmware", "qemu", "kvm", "xen",
		"hyper-v", "parallels", "bochs", "virtual machine",
		"vmware virtual platform", "virtualbox graphics adapter",
		"microsoft corporation virtual",
	}

	for _, keyword := range vmKeywords {
		if strings.Contains(rendererLower, keyword) {
			indicators = append(indicators, "vmware_detected")
			result["is_vm"] = true
			result["vm_type"] = "vmware"
			break
		}
	}

	if !result["is_vm"].(bool) {
		for _, keyword := range []string{"swiftshader", "llvmpipe", "mesa", "softpipe", "software"} {
			if strings.Contains(rendererLower, keyword) {
				indicators = append(indicators, "software_renderer")
				result["is_vm"] = true
				result["vm_type"] = "software_rendering"
				break
			}
		}
	}

	if strings.Contains(strings.ToLower(req.ScreenSize), "0x0") || strings.Contains(strings.ToLower(req.ScreenSize), "1x1") {
		indicators = append(indicators, "zero_screen_size")
		result["is_vm"] = true
	}

	if req.CPUCores > 64 || req.CPUCores == 1 {
		indicators = append(indicators, "unusual_cpu_cores")
		result["is_vm"] = true
	}

	if req.DeviceMemory > 64 || req.DeviceMemory <= 0.25 {
		indicators = append(indicators, "unusual_memory")
		result["is_vm"] = true
	}

	result["indicators"] = indicators
	return result
}

func DetectCloudEnvironment(c *gin.Context) {
	var req struct {
		IPAddress    string `json:"ip_address"`
		UserAgent    string `json:"user_agent"`
		ISP          string `json:"isp"`
		Organization string `json:"organization"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request",
		})
		return
	}

	result := detectCloudIndicators(detectCloudReq{
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		ISP:          req.ISP,
		Organization: req.Organization,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func detectCloudIndicators(req struct {
	IPAddress    string
	UserAgent    string
	ISP          string
	Organization string
}) map[string]interface{} {
	result := make(map[string]interface{})
	result["is_cloud"] = false
	result["provider"] = ""
	result["is_datacenter"] = false
	result["indicators"] = []string{}

	indicators := []string{}
	combined := strings.ToLower(req.IPAddress + " " + req.ISP + " " + req.Organization)

	cloudProviders := map[string][]string{
		"aws":     {"amazon", "aws", "amazon web services", "amazon.com", "ec2", "3.", "52.", "54.", "35."},
		"gcp":     {"google", "google cloud", "gcp", "google llc", "34.", "35.", "104."},
		"azure":   {"microsoft", "azure", "msft", "windows azure", "20.", "40.", "13.", "52."},
		"digitalocean": {"digitalocean", "digital ocean", "104.", "167.", "159."},
		"linode":  {"linode", "linode llc", "45.", "50.", "96.", "172.104."},
		"vultr":   {"vultr", "vultr holdings", "45.", "104.", "149.", "167."},
		"oracle":  {"oracle", "oracle cloud", "oci", "oracle infrastructure", "144."},
		"ibm":     {"ibm", "softlayer", "bluemix", "watson", "75."},
		"alibaba": {"alibaba", "aliyun", "alicdn", "168.", "106.", "47."},
		"tencent": {"tencent", "tencent cloud", "cloud.tencent", "43.", "119.", "125."},
		"huawei":  {"huawei", "huawei cloud", "hwclouds", "114.", "159."},
	}

	for provider, keywords := range cloudProviders {
		for _, keyword := range keywords {
			if strings.Contains(combined, keyword) {
				indicators = append(indicators, provider+"_detected")
				result["is_cloud"] = true
				result["provider"] = provider
				break
			}
		}
		if result["is_cloud"].(bool) {
			break
		}
	}

	datacenterKeywords := []string{"datacenter", "data center", "hosting", "colocation", "server farm"}
	for _, keyword := range datacenterKeywords {
		if strings.Contains(combined, keyword) {
			indicators = append(indicators, "datacenter_detected")
			result["is_datacenter"] = true
			break
		}
	}

	result["indicators"] = indicators
	return result
}

func DetectContainerEnvironment(c *gin.Context) {
	var req struct {
		CPUCores     int     `json:"cpu_cores"`
		DeviceMemory float64 `json:"device_memory"`
		UserAgent    string  `json:"user_agent"`
		Platform     string  `json:"platform"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request",
		})
		return
	}

	result := detectContainerIndicators(detectContainerReq{
		CPUCores:     req.CPUCores,
		DeviceMemory: req.DeviceMemory,
		UserAgent:    req.UserAgent,
		Platform:     req.Platform,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func detectContainerIndicators(req struct {
	CPUCores     int
	DeviceMemory float64
	UserAgent    string
	Platform     string
}) map[string]interface{} {
	result := make(map[string]interface{})
	result["is_container"] = false
	result["container_type"] = ""
	result["indicators"] = []string{}

	indicators := []string{}

	if req.CPUCores == 1 || req.CPUCores == 2 {
		indicators = append(indicators, "low_cpu_cores")
	}

	if req.DeviceMemory <= 0.5 {
		indicators = append(indicators, "low_memory")
	}

	uaLower := strings.ToLower(req.UserAgent)
	if strings.Contains(uaLower, "docker") || strings.Contains(uaLower, "container") {
		indicators = append(indicators, "container_in_ua")
		result["is_container"] = true
		result["container_type"] = "docker"
	}

	platformLower := strings.ToLower(req.Platform)
	if strings.Contains(platformLower, "docker") || strings.Contains(platformLower, "container") {
		indicators = append(indicators, "container_in_platform")
		result["is_container"] = true
		result["container_type"] = "docker"
	}

	if len(indicators) >= 2 {
		result["is_container"] = true
		if result["container_type"] == "" {
			result["container_type"] = "likely_docker"
		}
	}

	result["indicators"] = indicators
	return result
}

func DetectHeadlessBrowser(c *gin.Context) {
	var req struct {
		UserAgent     string `json:"user_agent"`
		NavigatorWebdriver bool `json:"navigator_webdriver"`
		ChromeRuntime bool   `json:"chrome_runtime"`
		PluginsCount  int    `json:"plugins_count"`
		Languages     []string `json:"languages"`
		Permissions   map[string]string `json:"permissions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request",
		})
		return
	}

	result := detectHeadlessIndicators(detectHeadlessReq{
		UserAgent:         req.UserAgent,
		NavigatorWebdriver: req.NavigatorWebdriver,
		ChromeRuntime:     req.ChromeRuntime,
		PluginsCount:      req.PluginsCount,
		Languages:         req.Languages,
		Permissions:       req.Permissions,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func detectHeadlessIndicators(req struct {
	UserAgent      string
	NavigatorWebdriver bool
	ChromeRuntime  bool
	PluginsCount   int
	Languages      []string
	Permissions    map[string]string
}) map[string]interface{} {
	result := make(map[string]interface{})
	result["is_headless"] = false
	result["confidence"] = 0.0
	result["indicators"] = []string{}

	indicators := []string{}
	score := 0.0

	uaLower := strings.ToLower(req.UserAgent)
	if strings.Contains(uaLower, "headless") {
		indicators = append(indicators, "headless_in_ua")
		score += 40
	}
	if strings.Contains(uaLower, "phantom") {
		indicators = append(indicators, "phantomjs_in_ua")
		score += 50
	}
	if strings.Contains(uaLower, "puppeteer") {
		indicators = append(indicators, "puppeteer_in_ua")
		score += 30
	}
	if strings.Contains(uaLower, "playwright") {
		indicators = append(indicators, "playwright_in_ua")
		score += 30
	}

	if req.NavigatorWebdriver {
		indicators = append(indicators, "navigator_webdriver_true")
		score += 40
	}

	if !req.ChromeRuntime {
		indicators = append(indicators, "chrome_runtime_missing")
		score += 20
	}

	if req.PluginsCount == 0 {
		indicators = append(indicators, "no_plugins")
		score += 15
	}

	if len(req.Languages) == 0 {
		indicators = append(indicators, "no_languages")
		score += 15
	}

	deniedCount := 0
	for _, perm := range req.Permissions {
		if perm == "denied" {
			deniedCount++
		}
	}
	if deniedCount >= 3 {
		indicators = append(indicators, "all_permissions_denied")
		score += 20
	}

	result["is_headless"] = score >= 30
	result["confidence"] = math.Min(score/100, 1.0)
	result["indicators"] = indicators

	return result
}

func EnhancedCanvasFingerprint(c *gin.Context) {
	var req struct {
		CanvasData string `json:"canvas_data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "canvas_data is required",
		})
		return
	}

	result := analyzeEnhancedCanvas(req.CanvasData)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func analyzeEnhancedCanvas(canvasData string) map[string]interface{} {
	result := make(map[string]interface{})

	result["length"] = len(canvasData)

	hash := md5.Sum([]byte(canvasData))
	result["md5_hash"] = hex.EncodeToString(hash[:])

	shaHash := sha256Hash(canvasData)
	result["sha256_hash"] = shaHash[:32]

	result["has_webgl"] = strings.Contains(canvasData, "webgl") || len(canvasData) > 500
	result["has_2d_context"] = len(canvasData) > 100

	anomalies := []string{}
	if len(canvasData) < 100 {
		anomalies = append(anomalies, "too_small")
	}
	if len(canvasData) > 10000 {
		anomalies = append(anomalies, "unusually_large")
	}

	result["anomalies"] = anomalies

	return result
}

func sha256Hash(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

func EnhancedWebGLFingerprint(c *gin.Context) {
	var req struct {
		WebGLData string `json:"webgl_data"`
		Extensions []string `json:"extensions"`
		MaxTextureSize int `json:"max_texture_size"`
		Renderer string `json:"renderer"`
		Vendor string `json:"vendor"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request",
		})
		return
	}

	result := analyzeEnhancedWebGL(analyzeWebGLReq{
		WebGLData:      req.WebGLData,
		Extensions:     req.Extensions,
		MaxTextureSize: req.MaxTextureSize,
		Renderer:       req.Renderer,
		Vendor:         req.Vendor,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func analyzeEnhancedWebGL(req struct {
	WebGLData      string
	Extensions     []string
	MaxTextureSize int
	Renderer       string
	Vendor         string
}) map[string]interface{} {
	result := make(map[string]interface{})

	result["vendor"] = req.Vendor
	result["renderer"] = req.Renderer
	result["extension_count"] = len(req.Extensions)
	result["max_texture_size"] = req.MaxTextureSize

	rendererLower := strings.ToLower(req.Renderer)
	softwareIndicators := map[string]string{
		"swiftshader": "SwiftShader",
		"llvmpipe": "LLVMpipe",
		"mesa": "Mesa",
		"softpipe": "Softpipe",
		"software": "Software Rendering",
		"virtualbox": "VirtualBox",
		"vmware": "VMware",
	}

	result["is_software"] = false
	result["software_name"] = ""

	for indicator, name := range softwareIndicators {
		if strings.Contains(rendererLower, indicator) {
			result["is_software"] = true
			result["software_name"] = name
			break
		}
	}

	anomalies := []string{}
	if req.MaxTextureSize < 2048 {
		anomalies = append(anomalies, "low_max_texture_size")
	}
	if len(req.Extensions) < 10 {
		anomalies = append(anomalies, "few_extensions")
	}
	if req.Vendor == "" || req.Renderer == "" {
		anomalies = append(anomalies, "missing_vendor_or_renderer")
	}

	result["anomalies"] = anomalies

	return result
}

func BatchDetection(c *gin.Context) {
	var req struct {
		Sessions []string `json:"session_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "session_ids is required",
		})
		return
	}

	results := make([]AdvancedDetectionResponse, 0)

	advSessionMutex.RLock()
	defer advSessionMutex.RUnlock()

	for _, sessionID := range req.Sessions {
		if session, exists := sessions[sessionID]; exists {
			results = append(results, AdvancedDetectionResponse{
				Success:          true,
				SessionID:        session.SessionID,
				RiskScore:        session.RiskScore,
				Confidence:       session.Confidence,
				RiskLevel:        getRiskLevel(session.RiskScore),
				EnvironmentType:  string(session.EnvironmentType),
				DetectionFlags:   session.DetectionFlags,
				FingerprintHash: session.FingerprintHash,
				BrowserEngine:   string(session.BrowserEngine),
				EngineVersion:    session.EngineVersion,
				IsHeadless:      session.IsHeadless,
				IsVM:            session.IsVM,
				IsCloudVM:       session.IsCloudVM,
				IsContainer:     session.IsContainer,
				Features:        session.Features,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"results":  results,
		"count":    len(results),
	})
}
