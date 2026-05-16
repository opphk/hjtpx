package service

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"net"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	EnvironmentVersion = "v4.0"
	MaxSessionAge     = 10 * time.Minute
	MinEntropy        = 0.5
)

type BrowserEngine string

const (
	EngineBlink     BrowserEngine = "blink"
	EngineGecko     BrowserEngine = "gecko"
	EngineWebKit    BrowserEngine = "webkit"
	EngineEdgeHTML  BrowserEngine = "edgehtml"
	EngineTrident   BrowserEngine = "trident"
	EngineUnknown   BrowserEngine = "unknown"
)

type AutomationFramework string

const (
	FrameworkNone       AutomationFramework = "none"
	FrameworkSelenium    AutomationFramework = "selenium"
	FrameworkPuppeteer  AutomationFramework = "puppeteer"
	FrameworkPlaywright  AutomationFramework = "playwright"
	FrameworkPhantomJS   AutomationFramework = "phantomjs"
	FrameworkCypress    AutomationFramework = "cypress"
	FrameworkSelenoid   AutomationFramework = "selenoid"
)

type EnvironmentType string

const (
	EnvTypeNormal        EnvironmentType = "normal"
	EnvTypeHeadless      EnvironmentType = "headless"
	EnvTypeVM            EnvironmentType = "vm"
	EnvTypeContainer     EnvironmentType = "container"
	EnvTypeCloudVM       EnvironmentType = "cloud_vm"
	EnvTypeEmulator      EnvironmentType = "emulator"
	EnvTypeTorBrowser    EnvironmentType = "tor"
)

type AdvancedEnvironmentData struct {
	ID               string                 `json:"id"`
	SessionID        string                 `json:"session_id"`
	Timestamp        int64                  `json:"timestamp"`
	BrowserEngine    BrowserEngine          `json:"browser_engine"`
	EngineVersion    string                 `json:"engine_version"`
	Automation       AutomationFramework    `json:"automation_framework"`
	EnvironmentType  EnvironmentType        `json:"environment_type"`
	CloudProvider    string                 `json:"cloud_provider,omitempty"`
	VMType           string                 `json:"vm_type,omitempty"`
	IsHeadless       bool                   `json:"is_headless"`
	IsVM             bool                   `json:"is_vm"`
	IsCloudVM        bool                   `json:"is_cloud_vm"`
	IsContainer      bool                   `json:"is_container"`
	RiskScore        float64                `json:"risk_score"`
	RiskLevel        string                 `json:"risk_level,omitempty"`
	Confidence       float64                `json:"confidence"`
	DetectionFlags   []string               `json:"detection_flags"`
	FingerprintHash  string                 `json:"fingerprint_hash"`
	Features         map[string]interface{} `json:"features"`
	RawData          map[string]interface{} `json:"raw_data,omitempty"`
}

type BrowserInfo struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	Engine          BrowserEngine `json:"engine"`
	EngineVersion   string `json:"engine_version"`
	OS              string `json:"os"`
	OSVersion       string `json:"os_version"`
	Arch            string `json:"arch"`
	Mobile          bool   `json:"mobile"`
	Tablet          bool   `json:"tablet"`
	Bot             bool   `json:"bot"`
}

type CanvasFingerprint struct {
	Hash          string   `json:"hash"`
	Renderer      string   `json:"renderer"`
	Vendor        string   `json:"vendor"`
	Version       string   `json:"version"`
	Anomalies     []string `json:"anomalies"`
	NoisePatterns []string `json:"noise_patterns"`
}

type WebGLFingerprint struct {
	Vendor         string   `json:"vendor"`
	Renderer       string   `json:"renderer"`
	Version        string   `json:"version"`
	ShadingVersion string   `json:"shading_version"`
	Extensions     []string `json:"extensions"`
	MaxTextureSize int      `json:"max_texture_size"`
	MaxViewport    []int    `json:"max_viewport"`
	Anisotropy     bool     `json:"anisotropy"`
	IsSoftware     bool     `json:"is_software"`
	SoftwareName   string   `json:"software_name,omitempty"`
}

type AudioFingerprint struct {
	SampleRate     int      `json:"sample_rate"`
	ChannelCount  int      `json:"channel_count"`
	LatencyMode   string   `json:"latency_mode"`
	State         string   `json:"state"`
	RenderTime    float64  `json:"render_time_ms"`
	SignalPatterns []string `json:"signal_patterns"`
	Anomalies     []string `json:"anomalies"`
}

type NetworkAnalysis struct {
	Protocol      string   `json:"protocol"`
	EffectiveType string   `json:"effective_type"`
	Downlink      float64  `json:"downlink"`
	RTT           int      `json:"rtt"`
	SaveData      bool     `json:"save_data"`
	ProxyHeaders  []string `json:"proxy_headers"`
	IPInfo        IPInfo   `json:"ip_info"`
	VPNDetected   bool     `json:"vpn_detected"`
	TorDetected   bool     `json:"tor_detected"`
	CloudIP       bool     `json:"cloud_ip"`
}

type IPInfo struct {
	IP            string `json:"ip"`
	Country       string `json:"country"`
	Region        string `json:"region"`
	City          string `json:"city"`
	ISP           string `json:"isp"`
	Organization  string `json:"organization"`
	ASN           string `json:"asn"`
	Type          string `json:"type"`
	CloudProvider string `json:"cloud_provider,omitempty"`
	Hosting       bool   `json:"hosting"`
	Proxy         bool   `json:"proxy"`
	Tor           bool   `json:"tor"`
}

type HardwareProfile struct {
	CPUCores            int     `json:"cpu_cores"`
	DeviceMemory        float64 `json:"device_memory_gb"`
	HardwareConcurrency int     `json:"hardware_concurrency"`
	Platform            string  `json:"platform"`
	TouchPoints         int     `json:"touch_points"`
	MaxTouchPoints      int     `json:"max_touch_points"`
	GPUVendor           string  `json:"gpu_vendor,omitempty"`
	GPUModel            string  `json:"gpu_model,omitempty"`
	GPUVerified         bool    `json:"gpu_verified"`
}

type FontAnalysis struct {
	Fonts         []string `json:"fonts"`
	Count         int      `json:"count"`
	CommonFonts   []string `json:"common_fonts"`
	RareFonts     []string `json:"rare_fonts"`
	MissingFonts  []string `json:"missing_fonts"`
	FakeFontRatio float64  `json:"fake_font_ratio"`
}

type AdvancedDetector struct {
	sessions    map[string]*AdvancedEnvironmentData
	sessionLock sync.RWMutex
	features    map[string]bool
	config      *DetectorConfig
}

type DetectorConfig struct {
	EnableCloudDetection     bool
	EnableVMDetection        bool
	EnableContainerDetection bool
	EnableBrowserEngineDetection bool
	MaxEntropyThreshold     float64
	ConfidenceThreshold     float64
	CloudProviderRanges     map[string]string
	VMIndicators           []string
}

var (
	defaultDetector *AdvancedDetector
	detectorOnce  sync.Once
)

func GetAdvancedDetector() *AdvancedDetector {
	detectorOnce.Do(func() {
		defaultDetector = NewAdvancedDetector()
	})
	return defaultDetector
}

func NewAdvancedDetector() *AdvancedDetector {
	d := &AdvancedDetector{
		sessions: make(map[string]*AdvancedEnvironmentData),
		features: make(map[string]bool),
		config: &DetectorConfig{
			EnableCloudDetection:     true,
			EnableVMDetection:       true,
			EnableContainerDetection: true,
			EnableBrowserEngineDetection: true,
			MaxEntropyThreshold:      0.7,
			ConfidenceThreshold:      0.85,
			CloudProviderRanges: map[string]string{
				"aws":          "3.54.52.35.18.",
				"gcp":          "34.35.104.35.192.",
				"azure":        "20.40.13.52.",
				"digitalocean": "104.167.159.",
				"linode":       "45.50.96.172.104.",
				"vultr":        "45.104.149.167.",
			},
			VMIndicators: []string{
				"VirtualBox", "VMware", "QEMU", "KVM", "Xen",
				"Hyper-V", "Parallels", "Bochs", "Virtual",
			},
		},
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	return d
}

func (d *AdvancedDetector) GenerateSession() string {
	d.sessionLock.Lock()
	defer d.sessionLock.Unlock()

	id := fmt.Sprintf("adv_%s_%d", uuid.New().String()[:8], time.Now().UnixNano())
	session := &AdvancedEnvironmentData{
		ID:          id,
		SessionID:   id,
		Timestamp:   time.Now().UnixMilli(),
		Features:    make(map[string]interface{}),
		RawData:     make(map[string]interface{}),
		DetectionFlags: make([]string, 0),
	}
	d.sessions[id] = session
	return id
}

func (d *AdvancedDetector) GetSession(id string) *AdvancedEnvironmentData {
	d.sessionLock.RLock()
	defer d.sessionLock.RUnlock()
	return d.sessions[id]
}

func (d *AdvancedDetector) UpdateSession(id string, data *AdvancedEnvironmentData) {
	d.sessionLock.Lock()
	defer d.sessionLock.Unlock()
	if existing, ok := d.sessions[id]; ok {
		data.ID = existing.ID
		data.SessionID = existing.SessionID
	}
	d.sessions[id] = data
}

func (d *AdvancedDetector) CleanupSessions() {
	d.sessionLock.Lock()
	defer d.sessionLock.Unlock()
	now := time.Now()
	for id, session := range d.sessions {
		if now.UnixMilli()-session.Timestamp > int64(MaxSessionAge) {
			delete(d.sessions, id)
		}
	}
}

func (d *AdvancedDetector) AnalyzeEnvironment(rawData map[string]interface{}) *AdvancedEnvironmentData {
	result := &AdvancedEnvironmentData{
		Timestamp:      time.Now().UnixMilli(),
		Features:       make(map[string]interface{}),
		RawData:        rawData,
		DetectionFlags: make([]string, 0),
		Confidence:     0.5,
	}

	browserInfo := d.analyzeBrowserInfo(rawData)
	result.BrowserEngine = browserInfo.Engine
	result.EngineVersion = browserInfo.EngineVersion

	automation := d.detectAutomationFramework(rawData, browserInfo)
	result.Automation = automation
	result.Features["browser"] = browserInfo
	result.Features["automation"] = automation

	canvasFP := d.analyzeCanvasFingerprint(rawData)
	result.Features["canvas"] = canvasFP

	webglFP := d.analyzeWebGLFingerprint(rawData)
	result.Features["webgl"] = webglFP

	audioFP := d.analyzeAudioFingerprint(rawData)
	result.Features["audio"] = audioFP

	fonts := d.analyzeFonts(rawData)
	result.Features["fonts"] = fonts

	hardware := d.analyzeHardware(rawData, webglFP)
	result.Features["hardware"] = hardware

	network := d.analyzeNetwork(rawData)
	result.Features["network"] = network

	envType := d.detectEnvironmentType(rawData, browserInfo, webglFP, network)
	result.EnvironmentType = envType

	switch envType {
	case EnvTypeHeadless:
		result.IsHeadless = true
	case EnvTypeVM:
		result.IsVM = true
	case EnvTypeCloudVM:
		result.IsCloudVM = true
	case EnvTypeContainer:
		result.IsContainer = true
	}

	if automation != FrameworkNone {
		result.DetectionFlags = append(result.DetectionFlags, fmt.Sprintf("automation:%s", automation))
	}

	result.RiskScore = d.calculateRiskScore(result, rawData)
	result.FingerprintHash = d.generateFingerprintHash(result)

	return result
}

func (d *AdvancedDetector) analyzeBrowserInfo(data map[string]interface{}) *BrowserInfo {
	info := &BrowserInfo{
		Engine: EngineUnknown,
	}

	if ua, ok := data["user_agent"].(string); ok {
		info.parseUserAgent(ua)
	}

	if engine, ok := data["browser_engine"].(string); ok {
		info.Engine = BrowserEngine(strings.ToLower(engine))
	}

	if engineVersion, ok := data["engine_version"].(string); ok {
		info.EngineVersion = engineVersion
	} else {
		info.EngineVersion = info.extractEngineVersion(data)
	}

	if platform, ok := data["platform"].(string); ok {
		info.OS = d.extractOS(platform)
	}

	return info
}

func (b *BrowserInfo) parseUserAgent(ua string) {
	uaLower := strings.ToLower(ua)

	if strings.Contains(uaLower, "edge") || strings.Contains(uaLower, "edg/") {
		b.Name = "Edge"
		b.Engine = EngineBlink
		if matches := regexp.MustCompile(`edg[e]?\/(\d+[\.\d]*)`).FindStringSubmatch(ua); len(matches) > 1 {
			b.Version = matches[1]
		}
	} else if strings.Contains(uaLower, "chrome") && !strings.Contains(uaLower, "chromium") {
		b.Name = "Chrome"
		b.Engine = EngineBlink
		if matches := regexp.MustCompile(`chrome\/(\d+[\.\d]*)`).FindStringSubmatch(ua); len(matches) > 1 {
			b.Version = matches[1]
		}
	} else if strings.Contains(uaLower, "firefox") {
		b.Name = "Firefox"
		b.Engine = EngineGecko
		if matches := regexp.MustCompile(`firefox\/(\d+[\.\d]*)`).FindStringSubmatch(ua); len(matches) > 1 {
			b.Version = matches[1]
		}
	} else if strings.Contains(uaLower, "safari") && !strings.Contains(uaLower, "chrome") {
		b.Name = "Safari"
		b.Engine = EngineWebKit
		if matches := regexp.MustCompile(`version\/(\d+[\.\d]*)`).FindStringSubmatch(ua); len(matches) > 1 {
			b.Version = matches[1]
		}
	} else if strings.Contains(uaLower, "chromium") {
		b.Name = "Chromium"
		b.Engine = EngineBlink
		if matches := regexp.MustCompile(`chromium\/(\d+[\.\d]*)`).FindStringSubmatch(ua); len(matches) > 1 {
			b.Version = matches[1]
		}
	} else if strings.Contains(uaLower, "trident") || strings.Contains(uaLower, "msie") {
		b.Name = "IE"
		b.Engine = EngineTrident
		if matches := regexp.MustCompile(`trident\/(\d+[\.\d]*)`).FindStringSubmatch(ua); len(matches) > 1 {
			b.Version = matches[1]
		}
	}

	if strings.Contains(uaLower, "windows") {
		b.OS = "Windows"
		if matches := regexp.MustCompile(`windows nt (\d+[\.\d]*)`).FindStringSubmatch(ua); len(matches) > 1 {
			switch matches[1] {
			case "10.0", "11":
				b.OSVersion = "10/11"
			case "6.3":
				b.OSVersion = "8.1"
			case "6.2":
				b.OSVersion = "8"
			case "6.1":
				b.OSVersion = "7"
			}
		}
	} else if strings.Contains(uaLower, "mac os x") {
		b.OS = "macOS"
		if matches := regexp.MustCompile(`mac os x (\d+[_\.\d]*)`).FindStringSubmatch(ua); len(matches) > 1 {
			b.OSVersion = strings.ReplaceAll(matches[1], "_", ".")
		}
	} else if strings.Contains(uaLower, "linux") {
		b.OS = "Linux"
	} else if strings.Contains(uaLower, "android") {
		b.OS = "Android"
		b.Mobile = true
	} else if strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "ipad") {
		b.OS = "iOS"
		b.Mobile = true
		if strings.Contains(uaLower, "ipad") {
			b.Tablet = true
		}
	}

	if strings.Contains(uaLower, "bot") || strings.Contains(uaLower, "crawl") || strings.Contains(uaLower, "spider") {
		b.Bot = true
	}
}

func (b *BrowserInfo) extractEngineVersion(data map[string]interface{}) string {
	ua, _ := data["user_agent"].(string)
	switch b.Engine {
	case EngineBlink:
		if matches := regexp.MustCompile(`chrome\/(\d+[\.\d]*)`).FindStringSubmatch(ua); len(matches) > 1 {
			return matches[1]
		}
	case EngineGecko:
		if matches := regexp.MustCompile(`rv:(\d+[\.\d]*)`).FindStringSubmatch(ua); len(matches) > 1 {
			return matches[1]
		}
	case EngineWebKit:
		if matches := regexp.MustCompile(`applewebkit\/(\d+[\.\d]*)`).FindStringSubmatch(ua); len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

func (d *AdvancedDetector) extractOS(platform string) string {
	platformLower := strings.ToLower(platform)
	if strings.Contains(platformLower, "win") {
		return "Windows"
	} else if strings.Contains(platformLower, "mac") {
		return "macOS"
	} else if strings.Contains(platformLower, "linux") {
		return "Linux"
	} else if strings.Contains(platformLower, "android") {
		return "Android"
	} else if strings.Contains(platformLower, "ios") {
		return "iOS"
	}
	return platform
}

func (d *AdvancedDetector) detectAutomationFramework(data map[string]interface{}, browser *BrowserInfo) AutomationFramework {
	ua, _ := data["user_agent"].(string)
	uaLower := strings.ToLower(ua)

	if strings.Contains(uaLower, "headless") || strings.Contains(uaLower, "phantom") {
		if strings.Contains(uaLower, "phantom") {
			return FrameworkPhantomJS
		}
		if strings.Contains(uaLower, "puppeteer") || strings.Contains(uaLower, "chrome-lambda") {
			return FrameworkPuppeteer
		}
		if strings.Contains(uaLower, "playwright") {
			return FrameworkPlaywright
		}
		return FrameworkSelenium
	}

	if _, ok := data["webdriver"]; ok {
		wd, _ := data["webdriver"].(string)
		if strings.Contains(strings.ToLower(wd), "true") {
			if strings.Contains(uaLower, "selenium") {
				return FrameworkSelenium
			}
			if strings.Contains(uaLower, "puppeteer") {
				return FrameworkPuppeteer
			}
			if strings.Contains(uaLower, "playwright") {
				return FrameworkPlaywright
			}
			return FrameworkSelenium
		}
	}

	if _, ok := data["selenium"]; ok {
		return FrameworkSelenium
	}

	if _, ok := data["puppeteer"]; ok {
		return FrameworkPuppeteer
	}

	if _, ok := data["playwright"]; ok {
		return FrameworkPlaywright
	}

	if strings.Contains(uaLower, "cypress") {
		return FrameworkCypress
	}

	if strings.Contains(uaLower, "selenoid") {
		return FrameworkSelenoid
	}

	return FrameworkNone
}

func (d *AdvancedDetector) analyzeCanvasFingerprint(data map[string]interface{}) *CanvasFingerprint {
	fp := &CanvasFingerprint{
		Anomalies: make([]string, 0),
	}

	if canvasData, ok := data["canvas"].(string); ok {
		fp.Hash = d.hashString(canvasData)

		if len(canvasData) < 100 {
			fp.Anomalies = append(fp.Anomalies, "canvas_too_short")
		}

		_, hasWebGL := data["webgl"]
		if !hasWebGL && len(canvasData) > 1000 {
			fp.Anomalies = append(fp.Anomalies, "canvas_without_webgl")
		}
	}

	if webglData, ok := data["webgl"].(string); ok {
		parts := strings.Split(webglData, "|")
		if len(parts) >= 2 {
			fp.Vendor = parts[0]
			fp.Renderer = parts[1]
		}
		if len(parts) >= 3 {
			fp.Version = parts[2]
		}
	}

	return fp
}

func (d *AdvancedDetector) analyzeWebGLFingerprint(data map[string]interface{}) *WebGLFingerprint {
	fp := &WebGLFingerprint{
		Extensions: make([]string, 0),
	}

	if webglData, ok := data["webgl"].(string); ok {
		parts := strings.Split(webglData, "|")
		if len(parts) >= 2 {
			fp.Vendor = parts[0]
			fp.Renderer = parts[1]
		}

		softwareIndicators := []string{"swiftshader", "llvmpipe", "mesa", "softpipe", "software", "virtualbox", "vmware", "parallels"}
		rendererLower := strings.ToLower(fp.Renderer)
		for _, indicator := range softwareIndicators {
			if strings.Contains(rendererLower, indicator) {
				fp.IsSoftware = true
				fp.SoftwareName = indicator
				break
			}
		}
	}

	if extensions, ok := data["webgl_extensions"].(string); ok {
		fp.Extensions = strings.Split(extensions, ",")
	}

	if maxTex, ok := data["max_texture_size"].(string); ok {
		fmt.Sscanf(maxTex, "%d", &fp.MaxTextureSize)
	}

	return fp
}

func (d *AdvancedDetector) analyzeAudioFingerprint(data map[string]interface{}) *AudioFingerprint {
	fp := &AudioFingerprint{
		Anomalies:      make([]string, 0),
		SignalPatterns: make([]string, 0),
	}

	if audioData, ok := data["audio"].(string); ok {
		parts := strings.Split(audioData, ":")
		if len(parts) >= 2 {
			var sumAbs, sumSq float64
			fmt.Sscanf(parts[0], "%f", &sumAbs)
			fmt.Sscanf(parts[1], "%f", &sumSq)

			if sumAbs == 0 && sumSq == 0 {
				fp.Anomalies = append(fp.Anomalies, "silent_audio")
			}

			if sumAbs > 0 && sumAbs < 0.01 {
				fp.Anomalies = append(fp.Anomalies, "very_quiet_audio")
			}
		}
	}

	if renderTime, ok := data["audio_render_time"]; ok {
		switch t := renderTime.(type) {
		case float64:
			fp.RenderTime = t
			if t < 5 {
				fp.Anomalies = append(fp.Anomalies, "render_too_fast")
			}
		}
	}

	return fp
}

func (d *AdvancedDetector) analyzeFonts(data map[string]interface{}) *FontAnalysis {
	analysis := &FontAnalysis{
		Fonts:        make([]string, 0),
		CommonFonts:  make([]string, 0),
		RareFonts:    make([]string, 0),
		MissingFonts: make([]string, 0),
	}

	commonFonts := []string{"arial", "helvetica", "times new roman", "courier new", "verdana", "georgia",
		"comic sans ms", "impact", "tahoma", "trebuchet ms"}

	rareFonts := []string{"lucida console", "palatino", "garamond", "bookman", "futura", "optima",
		"candara", "calibri", "corbel", "jetbrains mono", "sf pro"}

	if fontsData, ok := data["fonts"].(string); ok {
		fonts := strings.Split(strings.ToLower(fontsData), ",")
		analysis.Fonts = fonts
		analysis.Count = len(fonts)

		for _, font := range fonts {
			font = strings.TrimSpace(font)
			isCommon := false
			for _, common := range commonFonts {
				if strings.Contains(font, common) {
					isCommon = true
					break
				}
			}
			if isCommon {
				analysis.CommonFonts = append(analysis.CommonFonts, font)
			} else {
				for _, rare := range rareFonts {
					if strings.Contains(font, rare) {
						analysis.RareFonts = append(analysis.RareFonts, font)
						break
					}
				}
			}
		}

		if analysis.Count < 3 {
			analysis.MissingFonts = append(analysis.MissingFonts, "too_few_fonts")
		}
	}

	if analysis.Count > 0 {
		analysis.FakeFontRatio = 1.0 - float64(len(analysis.CommonFonts))/float64(analysis.Count)
	}

	return analysis
}

func (d *AdvancedDetector) analyzeHardware(data map[string]interface{}, webgl *WebGLFingerprint) *HardwareProfile {
	profile := &HardwareProfile{
		GPUVerified: webgl != nil && webgl.Renderer != "",
	}

	if cpu, ok := data["cpu_cores"].(string); ok {
		fmt.Sscanf(cpu, "%d", &profile.HardwareConcurrency)
		profile.CPUCores = profile.HardwareConcurrency
	}

	if mem, ok := data["device_memory"].(string); ok {
		fmt.Sscanf(mem, "%f", &profile.DeviceMemory)
	}

	if platform, ok := data["platform"].(string); ok {
		profile.Platform = platform
	}

	if touch, ok := data["touch_points"]; ok {
		switch t := touch.(type) {
		case float64:
			profile.TouchPoints = int(t)
			profile.MaxTouchPoints = int(t)
		case int:
			profile.TouchPoints = t
			profile.MaxTouchPoints = t
		}
	}

	if webgl != nil {
		profile.GPUVendor = webgl.Vendor
		profile.GPUModel = webgl.Renderer
	}

	return profile
}

func (d *AdvancedDetector) analyzeNetwork(data map[string]interface{}) *NetworkAnalysis {
	analysis := &NetworkAnalysis{
		ProxyHeaders: make([]string, 0),
	}

	proxyHeaders := []string{
		"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP",
		"Via", "X-Varnish", "True-Client-IP",
	}

	if headers, ok := data["request_headers"].(map[string]interface{}); ok {
		for _, header := range proxyHeaders {
			if val, exists := headers[header]; exists {
				if strVal, ok := val.(string); ok && strVal != "" {
					analysis.ProxyHeaders = append(analysis.ProxyHeaders, header+":"+strVal)
				}
			}
		}
	}

	if len(analysis.ProxyHeaders) > 0 {
		analysis.VPNDetected = true
	}

	if connData, ok := data["connection"].(string); ok {
		parts := strings.Split(connData, "|")
		if len(parts) >= 1 {
			analysis.EffectiveType = parts[0]
		}
		if len(parts) >= 2 {
			fmt.Sscanf(parts[1], "%f", &analysis.Downlink)
		}
		if len(parts) >= 3 {
			fmt.Sscanf(parts[2], "%d", &analysis.RTT)
		}
	}

	if ipInfo, ok := data["ip_info"].(map[string]interface{}); ok {
		analysis.IPInfo = d.parseIPInfo(ipInfo)
		analysis.CloudIP = analysis.IPInfo.Hosting
		if analysis.IPInfo.CloudProvider != "" {
			analysis.CloudIP = true
		}
	}

	return analysis
}

func (d *AdvancedDetector) parseIPInfo(info map[string]interface{}) IPInfo {
	ipInfo := IPInfo{}

	if ip, ok := info["ip"].(string); ok {
		ipInfo.IP = ip
		ipInfo.Type = d.classifyIPType(ip)
	}

	if country, ok := info["country"].(string); ok {
		ipInfo.Country = country
	}

	if isp, ok := info["isp"].(string); ok {
		ipInfo.ISP = isp
		ispLower := strings.ToLower(isp)
		torIndicators := []string{"tor", "onion", "torproject"}
		for _, indicator := range torIndicators {
			if strings.Contains(ispLower, indicator) {
				ipInfo.Tor = true
				break
			}
		}
	}

	if org, ok := info["organization"].(string); ok {
		ipInfo.Organization = org
		orgLower := strings.ToLower(org)
		proxyIndicators := []string{"proxy", "vpn", "hosting", "datacenter", "cloud"}
		for _, indicator := range proxyIndicators {
			if strings.Contains(orgLower, indicator) {
				ipInfo.Proxy = true
				break
			}
		}
	}

	ipInfo.CloudProvider = d.detectCloudProvider(info)

	return ipInfo
}

func (d *AdvancedDetector) classifyIPType(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "unknown"
	}

	if ip.IsLoopback() {
		return "loopback"
	}

	if ip.IsUnspecified() {
		return "unspecified"
	}

	privateRanges := []string{
		"10.", "172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.", "172.24.",
		"172.25.", "172.26.", "172.27.", "172.28.", "172.29.",
		"172.30.", "172.31.", "192.168.", "fc00:", "fe80:",
	}

	for _, prefix := range privateRanges {
		if strings.HasPrefix(ipStr, prefix) {
			return "private"
		}
	}

	return "public"
}

func (d *AdvancedDetector) detectCloudProvider(info map[string]interface{}) string {
	ip, _ := info["ip"].(string)
	isp, _ := info["isp"].(string)
	org, _ := info["organization"].(string)

	combined := strings.ToLower(ip + " " + isp + " " + org)

	providers := map[string][]string{
		"aws":     {"amazon", "aws", "amazon web services", "amazon.com", "ec2"},
		"gcp":     {"google", "google cloud", "gcp", "google llc"},
		"azure":   {"microsoft", "azure", "msft", "windows azure"},
		"digitalocean": {"digitalocean", "digital ocean"},
		"linode":  {"linode", "linode llc"},
		"vultr":   {"vultr", "vultr holdings"},
		"oracle":  {"oracle", "oracle cloud", "oci"},
		"ibm":     {"ibm", "softlayer", "bluemix"},
		"alibaba": {"alibaba", "aliyun", "alicdn"},
		"tencent": {"tencent", "tencent cloud", "cloud.tencent"},
		"huawei":  {"huawei", "huawei cloud", "hwclouds"},
	}

	for provider, keywords := range providers {
		for _, keyword := range keywords {
			if strings.Contains(combined, keyword) {
				return provider
			}
		}
	}

	return ""
}

func (d *AdvancedDetector) detectEnvironmentType(data map[string]interface{}, browser *BrowserInfo, webgl *WebGLFingerprint, network *NetworkAnalysis) EnvironmentType {
	ua, _ := data["user_agent"].(string)
	uaLower := strings.ToLower(ua)

	if strings.Contains(uaLower, "tor") {
		return EnvTypeTorBrowser
	}

	if webgl != nil && webgl.IsSoftware {
		return EnvTypeVM
	}

	if network != nil && network.TorDetected {
		return EnvTypeTorBrowser
	}

	if network != nil && network.CloudIP {
		return EnvTypeCloudVM
	}

	if d.detectVMEnvironment(data) {
		return EnvTypeVM
	}

	if d.detectContainerEnvironment(data) {
		return EnvTypeContainer
	}

	if strings.Contains(uaLower, "headless") {
		return EnvTypeHeadless
	}

	if browser != nil && browser.Bot {
		return EnvTypeEmulator
	}

	return EnvTypeNormal
}

func (d *AdvancedDetector) detectVMEnvironment(data map[string]interface{}) bool {
	if webgl, ok := data["webgl"].(string); ok {
		rendererLower := strings.ToLower(webgl)
		for _, indicator := range d.config.VMIndicators {
			if strings.Contains(rendererLower, strings.ToLower(indicator)) {
				return true
			}
		}
	}

	if screen, ok := data["screen"].(string); ok {
		if strings.Contains(screen, "0x0") || strings.Contains(screen, "1x1") {
			return true
		}
	}

	if cpuCores, ok := data["cpu_cores"].(string); ok {
		var cores int
		fmt.Sscanf(cpuCores, "%d", &cores)
		if cores > 64 || cores == 1 {
			return true
		}
	}

	return false
}

func (d *AdvancedDetector) detectContainerEnvironment(data map[string]interface{}) bool {
	if _, ok := data["container_indicator"]; ok {
		return true
	}

	if cpuCores, ok := data["cpu_cores"].(string); ok {
		var cores int
		fmt.Sscanf(cpuCores, "%d", &cores)
		if cores == 1 || cores == 2 {
			return true
		}
	}

	if mem, ok := data["device_memory"].(string); ok {
		var memory float64
		fmt.Sscanf(mem, "%f", &memory)
		if memory <= 0.5 {
			return true
		}
	}

	return false
}

func (d *AdvancedDetector) calculateRiskScore(result *AdvancedEnvironmentData, data map[string]interface{}) float64 {
	score := 0.0

	switch result.Automation {
	case FrameworkSelenium:
		score += 40
		result.DetectionFlags = append(result.DetectionFlags, "selenium_detected")
	case FrameworkPuppeteer:
		score += 45
		result.DetectionFlags = append(result.DetectionFlags, "puppeteer_detected")
	case FrameworkPlaywright:
		score += 45
		result.DetectionFlags = append(result.DetectionFlags, "playwright_detected")
	case FrameworkPhantomJS:
		score += 50
		result.DetectionFlags = append(result.DetectionFlags, "phantomjs_detected")
	case FrameworkCypress:
		score += 35
		result.DetectionFlags = append(result.DetectionFlags, "cypress_detected")
	}

	switch result.EnvironmentType {
	case EnvTypeHeadless:
		score += 25
		result.DetectionFlags = append(result.DetectionFlags, "headless_mode")
	case EnvTypeVM:
		score += 35
		result.DetectionFlags = append(result.DetectionFlags, "virtual_machine")
	case EnvTypeCloudVM:
		score += 30
		result.DetectionFlags = append(result.DetectionFlags, "cloud_vm")
		result.DetectionFlags = append(result.DetectionFlags, "cloud_provider:"+result.CloudProvider)
	case EnvTypeContainer:
		score += 20
		result.DetectionFlags = append(result.DetectionFlags, "container")
	case EnvTypeTorBrowser:
		score += 15
		result.DetectionFlags = append(result.DetectionFlags, "tor_browser")
	}

	if webgl, ok := result.Features["webgl"].(*WebGLFingerprint); ok && webgl.IsSoftware {
		score += 25
		result.DetectionFlags = append(result.DetectionFlags, "software_renderer:"+webgl.SoftwareName)
	}

	if network, ok := result.Features["network"].(*NetworkAnalysis); ok {
		if network.VPNDetected {
			score += 15
			result.DetectionFlags = append(result.DetectionFlags, "vpn_detected")
		}
		if network.IPInfo.Proxy {
			score += 20
			result.DetectionFlags = append(result.DetectionFlags, "proxy_detected")
		}
		if network.IPInfo.Tor {
			score += 25
			result.DetectionFlags = append(result.DetectionFlags, "tor_detected")
		}
	}

	if fonts, ok := result.Features["fonts"].(*FontAnalysis); ok {
		if fonts.Count < 3 {
			score += 15
			result.DetectionFlags = append(result.DetectionFlags, "minimal_fonts")
		}
		if fonts.FakeFontRatio > 0.8 {
			score += 10
			result.DetectionFlags = append(result.DetectionFlags, "fake_fonts")
		}
	}

	if audio, ok := result.Features["audio"].(*AudioFingerprint); ok {
		if len(audio.Anomalies) > 0 {
			score += 10
			result.DetectionFlags = append(result.DetectionFlags, "audio_anomalies")
		}
	}

	if canvas, ok := result.Features["canvas"].(*CanvasFingerprint); ok {
		if len(canvas.Anomalies) > 0 {
			score += 10
			result.DetectionFlags = append(result.DetectionFlags, "canvas_anomalies")
		}
	}

	result.Confidence = d.calculateConfidence(result, data)

	score = math.Min(math.Max(score, 0), 100)
	result.RiskLevel = getRiskLevel(score)

	return score
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

func (d *AdvancedDetector) calculateConfidence(result *AdvancedEnvironmentData, data map[string]interface{}) float64 {
	confidence := 0.5

	checks := 0
	passed := 0

	if _, ok := data["canvas"]; ok {
		checks++
		passed++
	}

	if _, ok := data["webgl"]; ok {
		checks++
		passed++
	}

	if _, ok := data["audio"]; ok {
		checks++
		passed++
	}

	if _, ok := data["fonts"]; ok {
		checks++
		passed++
	}

	if _, ok := data["webgl_extensions"]; ok {
		checks++
		passed++
	}

	if checks > 0 {
		confidence = float64(passed) / float64(checks) * 0.5
	}

	if result.Automation != FrameworkNone {
		confidence += 0.3
	}

	if result.EnvironmentType != EnvTypeNormal {
		confidence += 0.2
	}

	return math.Min(confidence, 1.0)
}

func (d *AdvancedDetector) generateFingerprintHash(result *AdvancedEnvironmentData) string {
	var parts []string

	if webgl, ok := result.Features["webgl"].(*WebGLFingerprint); ok {
		parts = append(parts, webgl.Renderer, webgl.Vendor)
	}

	if canvas, ok := result.Features["canvas"].(*CanvasFingerprint); ok {
		parts = append(parts, canvas.Hash)
	}

	if fonts, ok := result.Features["fonts"].(*FontAnalysis); ok {
		sorted := make([]string, len(fonts.Fonts))
		copy(sorted, fonts.Fonts)
		sort.Strings(sorted)
		parts = append(parts, strings.Join(sorted, ","))
	}

	if hardware, ok := result.Features["hardware"].(*HardwareProfile); ok {
		parts = append(parts, fmt.Sprintf("%d", hardware.HardwareConcurrency))
		parts = append(parts, fmt.Sprintf("%.1f", hardware.DeviceMemory))
	}

	parts = append(parts, string(result.BrowserEngine))
	parts = append(parts, result.EngineVersion)

	combined := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])[:32]
}

func (d *AdvancedDetector) hashString(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

func (d *AdvancedDetector) GenerateDetectionScript() string {
	methods := []string{
		"browserEngine",
		"canvasFingerprint",
		"webGLFingerprint",
		"audioFingerprint",
		"fontDetection",
		"hardwareProfile",
		"networkAnalysis",
		"automationDetection",
		"vmDetection",
		"cloudDetection",
		"containerDetection",
		"headlessDetection",
	}

	shuffled := make([]string, len(methods))
	for i, idx := range rand.Perm(len(methods)) {
		shuffled[i] = methods[idx]
	}

	script := `(function(){`
	script += `var __detectionId="det_${uuid}_${timestamp}";`
	script += `var __results={};`
	script += `var __startTime=Date.now();`

	for _, method := range shuffled {
		script += fmt.Sprintf(`try{__results.%s=%s();}catch(e){__results.%s={error:e.message};}`, method, method, method)
	}

	script += `var __endTime=Date.now()-__startTime;`
	script += `__results.metadata={detectionId:__detectionId,duration:__endTime,timestamp:Date.now()};`
	script += `return __results;`
	script += `})()`

	return script
}
