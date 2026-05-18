package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type EnhancedVMDetector struct {
	mu sync.RWMutex
	db *VMDatabase
}

type VMDatabase struct {
	mu      sync.RWMutex
	entries map[string]*VMEntry
}

type VMEntry struct {
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	DetectedAt  time.Time `json:"detected_at"`
	Confidence  float64   `json:"confidence"`
	Indicators  []string  `json:"indicators"`
}

var vmPatterns = map[string]*VMDetectionPattern{
	"vmware": {
		Name:     "VMware",
		Patterns: []string{"vmware", "virtual platform", "vmware virtual platform", "vmware7,1"},
		MACPrefixes: []string{
			"00:05:69", "00:0C:29", "00:1C:14", "00:50:56",
		},
		CPUHints:        []string{"vmware", "virtual cpu"},
		BIOSVersions:    []string{"vmware", "virtualbox"},
		HardDiskSerial:  []string{"vmware", "virtual"},
		Score:           85,
	},
	"virtualbox": {
		Name:     "VirtualBox",
		Patterns: []string{"virtualbox", "vbox", "oracle vm virtualbox", "virtualbox guest"},
		MACPrefixes: []string{
			"08:00:27", "0A:00:27", "52:54:00",
		},
		CPUHints:        []string{"vbox"},
		BIOSVersions:    []string{"virtualbox", "vbox bios"},
		HardDiskSerial:  []string{"vbox", "vb-"},
		Score:           90,
	},
	"hyperv": {
		Name:     "Microsoft Hyper-V",
		Patterns: []string{"hyper-v", "microsoft corporation", "virtual machine", "hyperv"},
		MACPrefixes: []string{
			"00:03:FF", "00:0D:3A", "00:12:5A", "00:15:5D",
		},
		CPUHints:        []string{"hyperv", "microsoft hyper-v"},
		BIOSVersions:    []string{"hyper-v", "microsoft corporation"},
		HardDiskSerial:  []string{"hyper-v", "msft virtual"},
		Score:           88,
	},
	"qemu_kvm": {
		Name:     "QEMU/KVM",
		Patterns: []string{"qemu", "kvm", "bochs", "standard pc", "rhev", "oVirt"},
		MACPrefixes: []string{
			"52:54:00", "06:1C:9A", "0A:1B:2C", "52:B0:34",
		},
		CPUHints:        []string{"qemu", "kvm", "bochs"},
		BIOSVersions:    []string{"qemu", "sea bios", "coreboot"},
		HardDiskSerial:  []string{"qemu", "lvm", "dm-"},
		Score:           82,
	},
	"parallels": {
		Name:     "Parallels",
		Patterns: []string{"parallels", "parallels virtual platform", "parallels software"},
		MACPrefixes: []string{
			"00:1C:42", "AC:DE:AD",
		},
		CPUHints:        []string{"parallels"},
		BIOSVersions:    []string{"parallels"},
		HardDiskSerial:  []string{"parallels"},
		Score:           87,
	},
	"xen": {
		Name:     "Xen",
		Patterns: []string{"xen", "hvm domu", "xen hvm", "amazon ec2", "xen-3.0"},
		MACPrefixes: []string{
			"00:16:3E", "52:54:00",
		},
		CPUHints:        []string{"xen", "domu"},
		BIOSVersions:    []string{"xen", "bochs"},
		HardDiskSerial:  []string{"xen", "xvda", "xvdb"},
		Score:           80,
	},
	"genymotion": {
		Name:     "Genymotion",
		Patterns: []string{"genymotion", "genymobile", "custom phone", "generic google phone"},
		MACPrefixes: []string{
			"00:11:22", "AA:BB:CC",
		},
		CPUHints:        []string{"goldfish"},
		BIOSVersions:    []string{"android"},
		HardDiskSerial:  []string{"geno", "vbox"},
		Score:           92,
	},
	"bluestacks": {
		Name:     "BlueStacks",
		Patterns: []string{"bluestacks", "bluestacks pop", "android subsystem"},
		MACPrefixes: []string{
			"00:1E:C6", "00:1F:E1",
		},
		CPUHints:        []string{"bluestacks"},
		BIOSVersions:    []string{"android"},
		HardDiskSerial:  []string{"bst"},
		Score:           89,
	},
}

type VMDetectionPattern struct {
	Name            string
	Patterns        []string
	MACPrefixes     []string
	CPUHints        []string
	BIOSVersions    []string
	HardDiskSerial  []string
	Score           int
}

type VMDetectionResult struct {
	IsVM           bool                   `json:"is_vm"`
	VMType         string                 `json:"vm_type"`
	VMName         string                 `json:"vm_name"`
	Confidence     float64                `json:"confidence"`
	Indicators     []string               `json:"indicators"`
	RiskScore      int                    `json:"risk_score"`
	Details        map[string]interface{} `json:"details"`
}

type ContainerDetectionResult struct {
	IsContainer         bool                   `json:"is_container"`
	ContainerType       string                 `json:"container_type"`
	Confidence          float64                `json:"confidence"`
	Indicators          []string               `json:"indicators"`
	RiskScore           int                    `json:"risk_score"`
	CgroupVersion       int                    `json:"cgroup_version"`
	NamespaceInfo       map[string]bool        `json:"namespace_info"`
}

func NewEnhancedVMDetector() *EnhancedVMDetector {
	return &EnhancedVMDetector{
		db: &VMDatabase{
			entries: make(map[string]*VMEntry),
		},
	}
}

func (d *EnhancedVMDetector) DetectVM(data map[string]interface{}) *VMDetectionResult {
	result := &VMDetectionResult{
		IsVM:       false,
		Indicators: []string{},
		Details:    make(map[string]interface{}),
	}

	var maxScore int
	var detectedType, detectedName string

	if ua, ok := data["user_agent"].(string); ok {
		d.checkPatterns(ua, result)
	}

	if webgl, ok := data["webgl_renderer"].(string); ok {
		d.checkWebGLRenderer(webgl, result)
	}

	if canvas, ok := data["canvas_fingerprint"].(string); ok {
		d.checkCanvasFingerprint(canvas, result)
	}

	if cpuCores, ok := data["cpu_cores"].(float64); ok {
		d.checkCPUCores(int(cpuCores), result)
	}

	if memory, ok := data["device_memory"].(float64); ok {
		d.checkDeviceMemory(memory, result)
	}

	if screenRes, ok := data["screen_resolution"].(string); ok {
		d.checkScreenResolution(screenRes, result)
	}

	if audioHash, ok := data["audio_fingerprint"].(string); ok {
		d.checkAudioFingerprint(audioHash, result)
	}

	if _, ok := data["headless_browser"]; ok {
		result.Indicators = append(result.Indicators, "headless_browser_detected")
		result.RiskScore += 25
	}

	if _, ok := data["automation_detected"]; ok {
		result.Indicators = append(result.Indicators, "automation_framework_detected")
		result.RiskScore += 30
	}

	if timings, ok := data["timings"].(map[string]interface{}); ok {
		d.checkTimingAnomalies(timings, result)
	}

	for _, pattern := range vmPatterns {
		matchCount := 0
		for _, indicator := range result.Indicators {
			for _, p := range pattern.Patterns {
				if strings.Contains(strings.ToLower(indicator), strings.ToLower(p)) {
					matchCount++
				}
			}
		}

		if matchCount > 0 {
			score := pattern.Score * matchCount / len(pattern.Patterns)
			if score > maxScore {
				maxScore = score
				detectedType = pattern.Name
				detectedName = pattern.Name
			}
		}
	}

	if maxScore > 40 {
		result.IsVM = true
		result.VMType = detectedType
		result.VMName = detectedName
		result.Confidence = float64(maxScore) / 100.0
		result.RiskScore = maxScore
	}

	if result.RiskScore > 100 {
		result.RiskScore = 100
	}

	return result
}

func (d *EnhancedVMDetector) checkPatterns(text string, result *VMDetectionResult) {
	lower := strings.ToLower(text)

	for vmType, pattern := range vmPatterns {
		for _, p := range pattern.Patterns {
			if strings.Contains(lower, strings.ToLower(p)) {
				result.Indicators = append(result.Indicators, fmt.Sprintf("%s:%s", vmType, p))
				result.RiskScore += 20
			}
		}
	}

	if strings.Contains(lower, "headless") {
		result.Indicators = append(result.Indicators, "headless_detected")
		result.RiskScore += 30
	}

	if strings.Contains(lower, "phantom") {
		result.Indicators = append(result.Indicators, "phantomjs_detected")
		result.RiskScore += 35
	}

	if strings.Contains(lower, "puppeteer") || strings.Contains(lower, "playwright") {
		result.Indicators = append(result.Indicators, "automation_framework")
		result.RiskScore += 40
	}

	if strings.Contains(lower, "selenium") {
		result.Indicators = append(result.Indicators, "selenium_detected")
		result.RiskScore += 35
	}
}

func (d *EnhancedVMDetector) checkWebGLRenderer(renderer string, result *VMDetectionResult) {
	lower := strings.ToLower(renderer)

	if strings.Contains(lower, "swiftshader") || strings.Contains(lower, "llvmpipe") {
		result.Indicators = append(result.Indicators, "software_rendering")
		result.RiskScore += 25
	}

	if strings.Contains(lower, "vmware") || strings.Contains(lower, "virtualbox") {
		result.Indicators = append(result.Indicators, "vm_rendering:"+renderer)
		result.RiskScore += 35
	}

	if strings.Contains(lower, "microsoft basic") || strings.Contains(lower, "basic render") {
		result.Indicators = append(result.Indicators, "basic_renderer")
		result.RiskScore += 40
	}

	if strings.Contains(lower, "headless") {
		result.Indicators = append(result.Indicators, "headless_webgl")
		result.RiskScore += 30
	}
}

func (d *EnhancedVMDetector) checkCanvasFingerprint(fingerprint string, result *VMDetectionResult) {
	if len(fingerprint) == 0 {
		result.Indicators = append(result.Indicators, "missing_canvas_fingerprint")
		result.RiskScore += 20
	}

	if strings.HasPrefix(fingerprint, "0") || strings.HasPrefix(fingerprint, "00") {
		result.Indicators = append(result.Indicators, "suspicious_canvas_hash")
		result.RiskScore += 15
	}
}

func (d *EnhancedVMDetector) checkCPUCores(cores int, result *VMDetectionResult) {
	result.Details["cpu_cores"] = cores

	if cores <= 1 {
		result.Indicators = append(result.Indicators, "single_core_detected")
		result.RiskScore += 25
	} else if cores == 2 {
		result.Indicators = append(result.Indicators, "low_core_count_2")
		result.RiskScore += 15
	} else if cores > 64 {
		result.Indicators = append(result.Indicators, "unusually_high_core_count")
		result.RiskScore += 20
	}
}

func (d *EnhancedVMDetector) checkDeviceMemory(memoryGB float64, result *VMDetectionResult) {
	result.Details["device_memory"] = memoryGB

	if memoryGB < 1 {
		result.Indicators = append(result.Indicators, "very_low_memory")
		result.RiskScore += 25
	} else if memoryGB < 2 {
		result.Indicators = append(result.Indicators, "low_memory")
		result.RiskScore += 15
	} else if memoryGB > 128 {
		result.Indicators = append(result.Indicators, "unusually_high_memory")
		result.RiskScore += 10
	}
}

func (d *EnhancedVMDetector) checkScreenResolution(resolution string, result *VMDetectionResult) {
	parts := strings.Split(resolution, "x")
	if len(parts) != 2 {
		return
	}

	width, wErr := strconv.Atoi(parts[0])
	height, hErr := strconv.Atoi(parts[1])
	if wErr != nil || hErr != nil {
		return
	}

	result.Details["screen_width"] = width
	result.Details["screen_height"] = height

	emulatorResolutions := map[string][2]int{
		"iphone_3gs":     {320, 480},
		"iphone_se":      {320, 568},
		"iphone_8":       {375, 667},
		"iphone_x":       {375, 812},
		"iphone_xr":      {414, 896},
		"generic_android": {360, 640},
		"generic_tablet":  {600, 1024},
	}

	for name, dims := range emulatorResolutions {
		if width == dims[0] && height == dims[1] {
			result.Indicators = append(result.Indicators, fmt.Sprintf("emulator_resolution:%s", name))
			result.RiskScore += 20
		}
	}

	if width == height && width > 1000 {
		result.Indicators = append(result.Indicators, "square_screen_anomaly")
		result.RiskScore += 15
	}

	aspectRatio := float64(width) / float64(height)
	if aspectRatio < 0.5 || aspectRatio > 2.5 {
		result.Indicators = append(result.Indicators, "unusual_aspect_ratio")
		result.RiskScore += 10
	}
}

func (d *EnhancedVMDetector) checkAudioFingerprint(fingerprint string, result *VMDetectionResult) {
	if len(fingerprint) == 0 {
		result.Indicators = append(result.Indicators, "missing_audio_fingerprint")
		result.RiskScore += 15
	}
}

func (d *EnhancedVMDetector) checkTimingAnomalies(timings map[string]interface{}, result *VMDetectionResult) {
	if renderTime, ok := timings["dom_content_loaded"].(float64); ok {
		if renderTime < 100 {
			result.Indicators = append(result.Indicators, "extremely_fast_render")
			result.RiskScore += 20
		}
	}

	if loadTime, ok := timings["load_event"].(float64); ok {
		if loadTime < 200 {
			result.Indicators = append(result.Indicators, "extremely_fast_load")
			result.RiskScore += 15
		}
	}

	if responseTime, ok := timings["ttfb"].(float64); ok {
		if responseTime < 10 {
			result.Indicators = append(result.Indicators, "suspiciously_fast_ttfb")
			result.RiskScore += 10
		}
	}
}

func (d *EnhancedVMDetector) DetectContainer(data map[string]interface{}) *ContainerDetectionResult {
	result := &ContainerDetectionResult{
		IsContainer:   false,
		Indicators:    []string{},
		NamespaceInfo: make(map[string]bool),
	}

	result.CgroupVersion = d.detectCgroupVersion()

	if ua, ok := data["user_agent"].(string); ok {
		result.Indicators = append(result.Indicators, d.checkContainerPatterns(ua)...)
	}

	if hostname, ok := data["hostname"].(string); ok {
		result.Indicators = append(result.Indicators, d.checkHostnamePatterns(hostname)...)
	}

	if env, ok := data["environment_vars"].(map[string]string); ok {
		result.Indicators = append(result.Indicators, d.checkEnvironmentVariables(env)...)
	}

	if storage, ok := data["storage_info"].(map[string]interface{}); ok {
		d.checkStorageQuota(storage, result)
	}

	if perf, ok := data["performance_metrics"].(map[string]interface{}); ok {
		d.checkPerformancePatterns(perf, result)
	}

	result.RiskScore = len(result.Indicators) * 15
	if result.RiskScore > 100 {
		result.RiskScore = 100
	}

	if len(result.Indicators) >= 3 {
		result.IsContainer = true
		result.Confidence = float64(len(result.Indicators)) / 10.0
	}

	return result
}

func (d *EnhancedVMDetector) detectCgroupVersion() int {
	return 2
}

func (d *EnhancedVMDetector) checkContainerPatterns(ua string) []string {
	indicators := []string{}
	lower := strings.ToLower(ua)

	containerPatterns := map[string][]string{
		"docker":        {"docker", "containerd", "moby"},
		"kubernetes":    {"kubernetes", "k8s", "k3s", "gke", "eks"},
		"lxc":           {"lxc", "linux container"},
		"podman":        {"podman", "containers"},
		"cri-o":         {"cri-o"},
	}

	for containerType, patterns := range containerPatterns {
		for _, pattern := range patterns {
			if strings.Contains(lower, pattern) {
				indicators = append(indicators, fmt.Sprintf("container_ua:%s", containerType))
			}
		}
	}

	return indicators
}

func (d *EnhancedVMDetector) checkHostnamePatterns(hostname string) []string {
	indicators := []string{}
	lower := strings.ToLower(hostname)

	dockerPatterns := []string{"docker", "container", "kubernetes", "k8s", "svc", "cluster"}
	for _, pattern := range dockerPatterns {
		if strings.Contains(lower, pattern) {
			indicators = append(indicators, fmt.Sprintf("container_hostname:%s", pattern))
		}
	}

	if matched, _ := regexp.MatchString(`^[a-z0-9]{8,}$`, hostname); matched {
		indicators = append(indicators, "random_hostname")
	}

	return indicators
}

func (d *EnhancedVMDetector) checkEnvironmentVariables(env map[string]string) []string {
	indicators := []string{}

	containerVars := []string{
		"DOCKER_", "KUBERNETES_", "K8S_", "HOME=/root",
		"container", "kubernetes", "kube_", "KUBE_",
	}

	for key := range env {
		upperKey := strings.ToUpper(key)
		for _, pattern := range containerVars {
			if strings.HasPrefix(upperKey, pattern) || env[key] == pattern {
				indicators = append(indicators, fmt.Sprintf("container_env:%s", key))
			}
		}
	}

	if _, ok := env["HOSTNAME"]; ok {
		if _, ok := env["container"]; ok {
			indicators = append(indicators, "container_env_detected")
		}
	}

	return indicators
}

func (d *EnhancedVMDetector) checkStorageQuota(storage map[string]interface{}, result *ContainerDetectionResult) {
	if quota, ok := storage["quota"].(float64); ok {
		if quota == 0 {
			result.Indicators = append(result.Indicators, "zero_storage_quota")
			result.RiskScore += 25
		} else if quota < 100_000_000 {
			result.Indicators = append(result.Indicators, "low_storage_quota")
			result.RiskScore += 15
		}
	}
}

func (d *EnhancedVMDetector) checkPerformancePatterns(perf map[string]interface{}, result *ContainerDetectionResult) {
	if memoryAccessTime, ok := perf["memory_access_time"].(float64); ok {
		if memoryAccessTime < 10 {
			result.Indicators = append(result.Indicators, "very_fast_memory_access")
			result.RiskScore += 15
		}
	}

	if cpuScore, ok := perf["cpu_benchmark_score"].(float64); ok {
		if cpuScore > 90 {
			result.Indicators = append(result.Indicators, "unusually_high_cpu_benchmark")
			result.RiskScore += 20
		}
	}
}

type EnhancedProxyVPNDetector struct {
	mu             sync.RWMutex
	vpnDatabase    *VPNDatabase
	torDatabase    *TorDatabase
	proxyDatabase  *ProxyDatabase
	behaviorCache  *BehaviorCache
}

type VPNDatabase struct {
	mu       sync.RWMutex
	providers map[string]*VPNProviderInfo
	ranges    map[string]string
	asnMap    map[int]string
}

type VPNProviderInfo struct {
	Name          string
	Ranges        []string
	ASNs          []int
	KnownIPs      map[string]bool
	Confidence    float64
}

type TorDatabase struct {
	mu        sync.RWMutex
	exitNodes map[string]*TorExitNode
	relays    map[string]*TorRelay
}

type TorExitNode struct {
	IP        string
	Port      int
	Country   string
	ASN       string
	LastSeen  time.Time
	Bandwidth float64
}

type TorRelay struct {
	IP       string
	Type     string
	Country  string
	Nickname string
	Fingerprint string
}

type ProxyDatabase struct {
	mu      sync.RWMutex
	proxies map[string]*ProxyInfo
}

type ProxyInfo struct {
	IP          string
	Port        int
	Type        string
	Protocol    string
	Country     string
	Anonymity   string
	LastChecked time.Time
	LastSeen    time.Time
}

type BehaviorCache struct {
	mu       sync.RWMutex
	entries  map[string]*BehaviorEntry
}

type BehaviorEntry struct {
	IP           string
	RequestCount int
	Timestamps   []int64
	UserAgents   []string
	RiskPatterns []string
	LastSeen     time.Time
}

func NewEnhancedProxyVPNDetector() *EnhancedProxyVPNDetector {
	detector := &EnhancedProxyVPNDetector{
		vpnDatabase:   NewVPNDatabase(),
		torDatabase:   NewTorDatabase(),
		proxyDatabase: NewProxyDatabase(),
		behaviorCache: NewBehaviorCache(),
	}
	detector.initVPNProviders()
	detector.initTorExitNodes()
	return detector
}

func NewVPNDatabase() *VPNDatabase {
	return &VPNDatabase{
		providers: make(map[string]*VPNProviderInfo),
		ranges:    make(map[string]string),
		asnMap:    make(map[int]string),
	}
}

func NewTorDatabase() *TorDatabase {
	return &TorDatabase{
		exitNodes: make(map[string]*TorExitNode),
		relays:    make(map[string]*TorRelay),
	}
}

func NewProxyDatabase() *ProxyDatabase {
	return &ProxyDatabase{
		proxies: make(map[string]*ProxyInfo),
	}
}

func NewBehaviorCache() *BehaviorCache {
	return &BehaviorCache{
		entries: make(map[string]*BehaviorEntry),
	}
}

func (d *EnhancedProxyVPNDetector) initVPNProviders() {
	providers := []*VPNProviderInfo{
		{Name: "NordVPN", Ranges: []string{"45.33.", "45.45.", "45.67.", "45.89.", "185.195.", "104.238."}, ASNs: []int{201229, 212502}, Confidence: 0.95},
		{Name: "ExpressVPN", Ranges: []string{"23.", "104.", "132.", "185.220."}, ASNs: []int{201229}, Confidence: 0.92},
		{Name: "Surfshark", Ranges: []string{"172.104.", "185.220.", "188.172.", "45.33."}, ASNs: []int{212502}, Confidence: 0.93},
		{Name: "CyberGhost", Ranges: []string{"37.", "82.", "85.", "89.", "185.220."}, ASNs: []int{207083}, Confidence: 0.90},
		{Name: "ProtonVPN", Ranges: []string{"185.195.", "185.220.", "185.118.", "194.132."}, Confidence: 0.91},
		{Name: "Mullvad", Ranges: []string{"185.195.", "194.132.", "185.118."}, Confidence: 0.94},
		{Name: "Private Internet Access", Ranges: []string{"104.238.", "107.170.", "172.104."}, ASNs: []int{201229}, Confidence: 0.88},
		{Name: "Windscribe", Ranges: []string{"35.182.", "45.33.", "104."}, ASNs: []int{201229}, Confidence: 0.85},
		{Name: "IPVanish", Ranges: []string{"107.170.", "173.245."}, ASNs: []int{201229}, Confidence: 0.87},
		{Name: "Hotspot Shield", Ranges: []string{"104.238.", "45.33.", "52."}, ASNs: []int{201229}, Confidence: 0.86},
		{Name: "TunnelBear", Ranges: []string{"108.60.", "198."}, ASNs: []int{201229}, Confidence: 0.83},
		{Name: "Hide My Ass", Ranges: []string{"85.", "89.", "95."}, ASNs: []int{207083}, Confidence: 0.82},
		{Name: "PureVPN", Ranges: []string{"103.", "109.", "213.", "37."}, ASNs: []int{207083}, Confidence: 0.84},
		{Name: "Buffered VPN", Ranges: []string{"37.", "85.", "185."}, Confidence: 0.80},
		{Name: "Perfect Privacy", Ranges: []string{"185.220."}, Confidence: 0.96},
		{Name: "AirVPN", Ranges: []string{"5.", "185.220."}, Confidence: 0.93},
		{Name: "VPNArea", Ranges: []string{"185.220."}, Confidence: 0.88},
		{Name: "BlackSpects", Ranges: []string{"185.195."}, Confidence: 0.92},
		{Name: "AzireVPN", Ranges: []string{"193."}, Confidence: 0.91},
		{Name: "IVPN", Ranges: []string{"185.195."}, Confidence: 0.90},
	}

	for _, p := range providers {
		p.KnownIPs = make(map[string]bool)
		d.vpnDatabase.providers[p.Name] = p
		for _, range_ := range p.Ranges {
			d.vpnDatabase.ranges[range_] = p.Name
		}
		for _, asn := range p.ASNs {
			d.vpnDatabase.asnMap[asn] = p.Name
		}
	}
}

func (d *EnhancedProxyVPNDetector) initTorExitNodes() {
	torExitIPs := []string{
		"128.31.0.34", "128.31.0.39", "128.93.34.5", "128.119.245.12",
		"131.188.40.101", "134.209.85.100", "137.74.19.240",
		"138.197.152.195", "139.162.1.105", "139.162.3.201",
		"141.105.65.113", "144.217.80.80", "149.56.85.18",
		"158.69.60.133", "159.65.3.181", "162.247.72.27",
		"163.172.25.23", "163.172.27.97", "164.90.219.85",
		"168.119.118.35", "171.25.193.77", "172.104.11.4",
		"176.10.99.200", "178.17.170.116", "178.32.45.50",
		"179.43.175.2", "179.43.177.242", "18.27.197.252",
		"185.100.84.84", "185.107.96.5", "185.117.73.171",
		"185.129.148.216", "185.13.57.103", "185.149.65.59",
		"185.156.174.149", "185.163.157.150", "185.165.168.229",
		"185.166.212.183", "185.167.96.6", "185.181.117.71",
		"185.19.107.172", "185.193.126.65", "185.194.108.27",
		"185.195.27.109", "185.197.75.71", "185.200.190.153",
		"185.203.116.116", "185.210.217.134", "185.220.100.240",
		"185.220.100.241", "185.220.100.242", "185.220.100.243",
		"185.220.100.244", "185.220.100.245", "185.220.100.246",
		"185.220.101.1", "185.220.101.2", "185.220.101.3",
	}

	for _, ip := range torExitIPs {
		d.torDatabase.exitNodes[ip] = &TorExitNode{
			IP:       ip,
			LastSeen: time.Now(),
		}
	}
}

func (d *EnhancedProxyVPNDetector) DetectProxyVPN(ctx context.Context, ip string, headers http.Header, data map[string]interface{}) *ProxyVPNResult {
	result := &ProxyVPNResult{
		IP:         ip,
		IsProxy:   false,
		IsVPN:     false,
		IsTor:     false,
		IsDatacenter: false,
		Confidence: 0,
		Indicators: []string{},
		Score:      0,
	}

	d.checkProxyHeaders(headers, result)
	d.checkVPNByIP(ip, result)
	d.checkTorByIP(ip, result)
	d.checkVPNByASN(data, result)
	d.checkDatacenterIP(ip, result)
	d.checkWebRTCLeaks(data, result)
	d.checkConnectionType(data, result)
	d.checkNetworkLatency(ctx, result)
	d.analyzeBehavior(ip, data, result)

	result.RiskLevel = d.calculateRiskLevel(result)
	result.Recommendations = d.getRecommendations(result)

	return result
}

type ProxyVPNResult struct {
	IP            string   `json:"ip"`
	IsProxy       bool     `json:"is_proxy"`
	IsVPN         bool     `json:"is_vpn"`
	IsTor         bool     `json:"is_tor"`
	IsDatacenter  bool     `json:"is_datacenter"`
	VPNProvider   string   `json:"vpn_provider,omitempty"`
	Confidence    float64  `json:"confidence"`
	Score         int      `json:"score"`
	Indicators    []string `json:"indicators"`
	RiskLevel     string   `json:"risk_level"`
	Recommendations []string `json:"recommendations"`
}

func (d *EnhancedProxyVPNDetector) checkProxyHeaders(headers http.Header, result *ProxyVPNResult) {
	if headers == nil {
		return
	}

	if xff := headers.Get("X-Forwarded-For"); xff != "" {
		result.Indicators = append(result.Indicators, "x_forwarded_for_header")
		result.Score += 25
		result.IsProxy = true

		ips := strings.Split(xff, ",")
		if len(ips) > 2 {
			result.Indicators = append(result.Indicators, "multiple_proxy_hops")
			result.Score += 20
		}
	}

	if via := headers.Get("Via"); via != "" {
		result.Indicators = append(result.Indicators, "via_header:"+via)
		result.Score += 20
		result.IsProxy = true
	}

	if xRealIP := headers.Get("X-Real-IP"); xRealIP != "" {
		result.Indicators = append(result.Indicators, "x_real_ip_header")
		result.Score += 20
		result.IsProxy = true
	}

	if forwarded := headers.Get("Forwarded"); forwarded != "" {
		result.Indicators = append(result.Indicators, "forwarded_header")
		result.Score += 15
	}

	if xProxyID := headers.Get("X-ProxyId"); xProxyID != "" {
		result.Indicators = append(result.Indicators, "x_proxy_id_header")
		result.Score += 25
		result.IsProxy = true
	}

	if trueClientIP := headers.Get("True-Client-IP"); trueClientIP != "" {
		result.Indicators = append(result.Indicators, "true_client_ip_header")
		result.Score += 20
	}
}

func (d *EnhancedProxyVPNDetector) checkVPNByIP(ip string, result *ProxyVPNResult) {
	if ip == "" {
		return
	}

	for prefix, provider := range d.vpnDatabase.ranges {
		if strings.HasPrefix(ip, prefix) {
			result.IsVPN = true
			result.VPNProvider = provider
			result.Indicators = append(result.Indicators, fmt.Sprintf("vpn_ip_range:%s", provider))
			result.Score += 35
			result.Confidence += 0.9

			if providerInfo, ok := d.vpnDatabase.providers[provider]; ok {
				result.Confidence = providerInfo.Confidence
			}
			return
		}
	}
}

func (d *EnhancedProxyVPNDetector) checkTorByIP(ip string, result *ProxyVPNResult) {
	if ip == "" {
		return
	}

	if _, exists := d.torDatabase.exitNodes[ip]; exists {
		result.IsTor = true
		result.Indicators = append(result.Indicators, "known_tor_exit_node")
		result.Score += 50
		result.Confidence = 1.0
	}
}

func (d *EnhancedProxyVPNDetector) checkVPNByASN(data map[string]interface{}, result *ProxyVPNResult) {
	if asn, ok := data["asn"].(float64); ok {
		asnInt := int(asn)
		if provider, exists := d.vpnDatabase.asnMap[asnInt]; exists {
			result.IsVPN = true
			result.VPNProvider = provider
			result.Indicators = append(result.Indicators, fmt.Sprintf("vpn_asn:%s", provider))
			result.Score += 30
			result.Confidence += 0.8
		}
	}

	if asnOrg, ok := data["asn_org"].(string); ok {
		lower := strings.ToLower(asnOrg)
		vpnKeywords := []string{"vpn", "virtual private network", "proxy", "tunnel", "privacy"}

		for _, keyword := range vpnKeywords {
			if strings.Contains(lower, keyword) {
				result.Indicators = append(result.Indicators, fmt.Sprintf("vpn_asn_org:%s", asnOrg))
				result.Score += 25
				result.Confidence += 0.7
				break
			}
		}
	}
}

func (d *EnhancedProxyVPNDetector) checkDatacenterIP(ip string, result *ProxyVPNResult) {
	if ip == "" {
		return
	}

	datacenterRanges := []string{
		"23.", "35.", "45.", "50.", "52.", "54.", "63.",
		"64.", "65.", "66.", "67.", "68.", "69.", "70.",
		"72.", "73.", "74.", "75.", "76.", "77.", "78.",
		"79.", "80.", "81.", "82.", "83.", "84.", "85.",
		"86.", "87.", "88.", "89.", "90.", "91.", "92.",
		"93.", "94.", "95.", "96.", "97.", "98.", "99.",
		"104.", "108.", "130.", "131.", "132.", "133.",
		"134.", "135.", "136.", "137.", "138.", "139.",
		"140.", "141.", "142.", "143.", "144.", "145.",
		"146.", "147.", "148.", "149.", "150.", "151.",
		"152.", "153.", "154.", "155.", "156.", "157.",
		"158.", "159.", "160.", "161.", "162.", "163.",
		"164.", "165.", "166.", "167.", "168.", "169.",
		"170.", "172.", "173.", "174.", "175.", "176.",
		"177.", "178.", "179.", "180.", "181.", "182.",
		"183.", "184.", "185.", "186.", "187.", "188.",
		"189.", "190.", "191.", "192.", "193.", "194.",
		"195.", "196.", "197.", "198.", "199.", "200.",
		"204.", "205.", "206.", "207.", "208.", "209.",
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP != nil {
		ipStr := parsedIP.String()
		for _, prefix := range datacenterRanges {
			if strings.HasPrefix(ipStr, prefix) {
				result.IsDatacenter = true
				result.Indicators = append(result.Indicators, "datacenter_ip_range")
				result.Score += 10
				break
			}
		}
	}
}

func (d *EnhancedProxyVPNDetector) checkWebRTCLeaks(data map[string]interface{}, result *ProxyVPNResult) {
	if webrtcIPs, ok := data["webrtc_ips"].([]string); ok {
		if len(webrtcIPs) > 1 {
			privateIPs := 0
			publicIPs := 0

			for _, ipStr := range webrtcIPs {
				parsedIP := net.ParseIP(ipStr)
				if parsedIP != nil {
					if parsedIP.IsPrivate() || parsedIP.IsLoopback() {
						privateIPs++
					} else {
						publicIPs++
					}
				}
			}

			if privateIPs > 0 && publicIPs > 0 {
				result.Indicators = append(result.Indicators, "webrtc_ip_mismatch")
				result.Score += 25
				result.IsVPN = true
				result.Confidence += 0.6
			}
		}
	}
}

func (d *EnhancedProxyVPNDetector) checkConnectionType(data map[string]interface{}, result *ProxyVPNResult) {
	if connType, ok := data["connection_type"].(string); ok {
		lower := strings.ToLower(connType)

		if lower == "vpn" || lower == "pptp" || lower == "tunnel" {
			result.IsVPN = true
			result.Indicators = append(result.Indicators, "vpn_connection_type:"+connType)
			result.Score += 30
			result.Confidence += 0.7
		}

		if lower == "proxy" || lower == "socks" {
			result.IsProxy = true
			result.Indicators = append(result.Indicators, "proxy_connection_type:"+connType)
			result.Score += 35
			result.Confidence += 0.8
		}
	}
}

func (d *EnhancedProxyVPNDetector) checkNetworkLatency(ctx context.Context, result *ProxyVPNResult) {
	if latency, ok := ctx.Value("latency").(float64); ok {
		if latency > 3000 {
			result.Indicators = append(result.Indicators, fmt.Sprintf("high_latency:%.0fms", latency))
			result.Score += 25
		} else if latency > 1000 {
			result.Indicators = append(result.Indicators, fmt.Sprintf("moderate_latency:%.0fms", latency))
			result.Score += 10
		}
	}
}

func (d *EnhancedProxyVPNDetector) analyzeBehavior(ip string, data map[string]interface{}, result *ProxyVPNResult) {
	if ip == "" {
		return
	}

	d.behaviorCache.mu.Lock()
	defer d.behaviorCache.mu.Unlock()

	entry, exists := d.behaviorCache.entries[ip]
	if !exists {
		entry = &BehaviorEntry{
			IP:           ip,
			RequestCount: 0,
			Timestamps:   []int64{},
			UserAgents:   []string{},
			RiskPatterns: []string{},
		}
		d.behaviorCache.entries[ip] = entry
	}

	entry.RequestCount++
	entry.LastSeen = time.Now()

	if ua, ok := data["user_agent"].(string); ok {
		entry.UserAgents = append(entry.UserAgents, ua)
		if len(entry.UserAgents) > 1 && entry.UserAgents[len(entry.UserAgents)-1] != entry.UserAgents[len(entry.UserAgents)-2] {
			result.Indicators = append(result.Indicators, "changing_user_agents")
			result.Score += 20
		}
	}

	if len(entry.Timestamps) > 1 {
		var totalInterval int64
		for i := 1; i < len(entry.Timestamps); i++ {
			totalInterval += entry.Timestamps[i] - entry.Timestamps[i-1]
		}
		avgInterval := totalInterval / int64(len(entry.Timestamps)-1)

		if avgInterval < 100 {
			result.Indicators = append(result.Indicators, "very_rapid_requests")
			result.Score += 30
		} else if avgInterval < 500 {
			result.Indicators = append(result.Indicators, "rapid_requests")
			result.Score += 15
		}
	}

	if entry.RequestCount > 100 {
		result.Indicators = append(result.Indicators, fmt.Sprintf("high_request_volume:%d", entry.RequestCount))
		result.Score += 25
	}
}

func (d *EnhancedProxyVPNDetector) calculateRiskLevel(result *ProxyVPNResult) string {
	score := result.Score

	if result.IsTor {
		score += 30
	}

	if result.IsVPN && result.VPNProvider != "" {
		score += 20
	}

	if result.IsProxy {
		score += 25
	}

	if result.IsDatacenter {
		score += 10
	}

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

func (d *EnhancedProxyVPNDetector) getRecommendations(result *ProxyVPNResult) []string {
	recommendations := []string{}

	if result.RiskLevel == "critical" || result.RiskLevel == "high" {
		recommendations = append(recommendations, "block_or_restrict_access")
		recommendations = append(recommendations, "require_additional_verification")
		recommendations = append(recommendations, "enhanced_logging")
	} else if result.RiskLevel == "medium" {
		recommendations = append(recommendations, "require_captcha")
		recommendations = append(recommendations, "rate_limiting")
	} else if result.RiskLevel == "low" {
		recommendations = append(recommendations, "standard_monitoring")
	} else {
		recommendations = append(recommendations, "standard_access")
	}

	if result.IsTor {
		recommendations = append(recommendations, "review_tor_exit_node_policy")
	}

	if result.IsVPN {
		recommendations = append(recommendations, "check_vpn_usage_policy")
	}

	return recommendations
}

func (d *EnhancedProxyVPNDetector) BatchDetect(ctx context.Context, requests []ProxyCheckRequestEnv) []*ProxyVPNResult {
	results := make([]*ProxyVPNResult, len(requests))

	var wg sync.WaitGroup
	for i, req := range requests {
		wg.Add(1)
		go func(idx int, r ProxyCheckRequestEnv) {
			defer wg.Done()
			results[idx] = d.DetectProxyVPN(ctx, r.IP, r.Headers, r.Data)
		}(i, req)
	}
	wg.Wait()

	return results
}

type ProxyCheckRequestEnv struct {
	IP       string
	Headers  http.Header
	Data     map[string]interface{}
}

type AdvancedRiskScorer struct {
	mu sync.RWMutex
}

func NewAdvancedRiskScorer() *AdvancedRiskScorer {
	return &AdvancedRiskScorer{}
}

func (s *AdvancedRiskScorer) CalculateScore(analysis *AdvancedFingerprintAnalysis) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	var totalScore float64
	var weightSum float64

	weights := map[string]float64{
		"vm_detection":      0.30,
		"container_detection": 0.20,
		"proxy_vpn_detection": 0.25,
		"tor_detection":       0.15,
		"behavior_analysis":   0.10,
	}

	if analysis.BaseFingerprint != nil {
		totalScore += analysis.BaseFingerprint.AnomalyScore * weights["vm_detection"]
		weightSum += weights["vm_detection"]
	}

	if analysis.AdvancedIndicators != nil {
		proxyWeight := weights["proxy_vpn_detection"]
		torWeight := weights["tor_detection"]

		if len(analysis.AdvancedIndicators.ProxyVPNIndicators) > 0 {
			totalScore += float64(len(analysis.AdvancedIndicators.ProxyVPNIndicators)*10) * proxyWeight
		}

		if len(analysis.AdvancedIndicators.NetworkIndicators) > 0 {
			totalScore += float64(len(analysis.AdvancedIndicators.NetworkIndicators)*15) * torWeight
		}

		weightSum += proxyWeight + torWeight
	}

	if analysis.MLRiskScore > 0 {
		totalScore += analysis.MLRiskScore * weights["behavior_analysis"]
		weightSum += weights["behavior_analysis"]
	}

	if weightSum > 0 {
		return math.Min(100, totalScore/weightSum)
	}

	return 0
}

type DetectionPatternMatcher struct {
	mu       sync.RWMutex
	patterns map[string]*regexp.Regexp
}

func NewDetectionPatternMatcher() *DetectionPatternMatcher {
	pm := &DetectionPatternMatcher{
		patterns: make(map[string]*regexp.Regexp),
	}
	pm.initPatterns()
	return pm
}

func (pm *DetectionPatternMatcher) initPatterns() {
	patterns := map[string]string{
		"headless_browser":      `(?i)(headless|phantom|puppeteer|playwright|selenium|chromium\("|chrome\-headless)`,
		"automation":             `(?i)(automation|selenium|webdriver|phantomjs|casperjs|nightmare)`,
		"virtual_machine":       `(?i)(vmware|virtualbox|hyper[- ]?v|qemu|kvm|xen|parallels)`,
		"emulator":               `(?i)(android.?emulator|genymotion|bluestacks|nox|memu|ldplayer|ios.?simulator)`,
		"tor":                    `(?i)(torbrowser|torproject|onion|tor.?exit|tordnsel)`,
		"vpn":                    `(?i)(nordvpn|expressvpn|surfshark|protonvpn|mullvad|cyberghost|pia\.com|private.?internet)`,
		"proxy":                  `(?i)(proxy|socks|rotating.?ip|residential.?proxy|datacenter.?proxy)`,
		"suspicious_timing":      `(timing|timeing|timng)`,
	}

	for name, pattern := range patterns {
		if re, err := regexp.Compile(pattern); err == nil {
			pm.patterns[name] = re
		}
	}
}

func (pm *DetectionPatternMatcher) Match(text string) []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	matches := []string{}
	for name, pattern := range pm.patterns {
		if pattern.MatchString(text) {
			matches = append(matches, name)
		}
	}
	return matches
}

func CalculateFingerprintHash(data map[string]interface{}) string {
	hasher := sha256.New()

	jsonData, _ := json.Marshal(data)
	hasher.Write(jsonData)

	return hex.EncodeToString(hasher.Sum(nil))[:16]
}
