package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"
)

type DeviceType string

const (
	DeviceDesktop DeviceType = "desktop"
	DeviceMobile  DeviceType = "mobile"
	DeviceTablet  DeviceType = "tablet"
	DeviceTV      DeviceType = "tv"
	DeviceUnknown DeviceType = "unknown"
)

type EnvironmentType string

const (
	EnvReal      EnvironmentType = "real"
	EnvEmulator  EnvironmentType = "emulator"
	EnvSimulator EnvironmentType = "simulator"
	EnvVirtual   EnvironmentType = "virtual"
	EnvContainer EnvironmentType = "container"
	EnvMultiBox  EnvironmentType = "multi_box"
)

type DeviceDetectionResult struct {
	DeviceType        DeviceType      `json:"device_type"`
	EnvironmentType   EnvironmentType `json:"environment_type"`
	IsEmulator        bool            `json:"is_emulator"`
	IsVirtual         bool            `json:"is_virtual"`
	IsContainer       bool            `json:"is_container"`
	IsMultiBox        bool            `json:"is_multi_box"`
	Confidence        float64         `json:"confidence"`
	Score             float64         `json:"score"`
	Indicators        []string        `json:"indicators"`
	DetectionMethods  []string        `json:"detection_methods"`
	DeviceFingerprint string          `json:"device_fingerprint"`
}

type DeviceFingerprintData struct {
	FingerprintID   string        `json:"fingerprint_id"`
	HardwareInfo    *HardwareInfo `json:"hardware_info"`
	SoftwareInfo    *SoftwareInfo `json:"software_info"`
	NetworkInfo     *NetworkInfo  `json:"network_info"`
	StabilityScore  float64       `json:"stability_score"`
	FirstSeen       time.Time     `json:"first_seen"`
	LastSeen        time.Time     `json:"last_seen"`
	RequestCount    int           `json:"request_count"`
	IsKnownMultiBox bool          `json:"is_known_multi_box"`
}

type HardwareInfo struct {
	CPUInfo             string  `json:"cpu_info"`
	CPUCores            int     `json:"cpu_cores"`
	MemoryInfo          string  `json:"memory_info"`
	DeviceMemory        float64 `json:"device_memory"`
	GPUInfo             string  `json:"gpu_info"`
	ScreenWidth         int     `json:"screen_width"`
	ScreenHeight        int     `json:"screen_height"`
	ScreenColorDepth    int     `json:"screen_color_depth"`
	PixelRatio          float64 `json:"pixel_ratio"`
	HardwareConcurrency int     `json:"hardware_concurrency"`
	IsVirtualCPU        bool    `json:"is_virtual_cpu"`
	IsLimitedHardware   bool    `json:"is_limited_hardware"`
}

type SoftwareInfo struct {
	OS               string   `json:"os"`
	OSVersion        string   `json:"os_version"`
	Browser          string   `json:"browser"`
	BrowserVersion   string   `json:"browser_version"`
	Platform         string   `json:"platform"`
	UserAgent        string   `json:"user_agent"`
	Language         string   `json:"language"`
	Timezone         string   `json:"timezone"`
	InstalledFonts   []string `json:"installed_fonts"`
	InstalledPlugins []string `json:"installed_plugins"`
}

type NetworkInfo struct {
	ConnectionType string   `json:"connection_type"`
	EffectiveType  string   `json:"effective_type"`
	RTT            int      `json:"rtt"`
	Downlink       float64  `json:"downlink"`
	SaveData       bool     `json:"save_data"`
	WebRTCEnabled  bool     `json:"webrtc_enabled"`
	LocalIPs       []string `json:"local_ips"`
}

type DeviceDetectionService struct {
	deviceDatabase  map[string]*DeviceFingerprintData
	knownEmulators  map[string]*EmulatorSignature
	knownVMs        map[string]*VMSignature
	knownContainers map[string]*ContainerSignature
	mu              sync.RWMutex
}

type EmulatorSignature struct {
	Name       string
	Patterns   []*regexp.Regexp
	Indicators []string
	Weight     float64
}

type VMSignature struct {
	Name       string
	Patterns   []*regexp.Regexp
	Indicators []string
	Weight     float64
}

type ContainerSignature struct {
	Name       string
	Patterns   []*regexp.Regexp
	Indicators []string
	Weight     float64
}

func NewDeviceDetectionService() *DeviceDetectionService {
	service := &DeviceDetectionService{
		deviceDatabase:  make(map[string]*DeviceFingerprintData),
		knownEmulators:  make(map[string]*EmulatorSignature),
		knownVMs:        make(map[string]*VMSignature),
		knownContainers: make(map[string]*ContainerSignature),
	}

	service.initializeSignatures()
	return service
}

func (s *DeviceDetectionService) initializeSignatures() {
	s.knownEmulators["Android_Emulator"] = &EmulatorSignature{
		Name: "Android Emulator",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)android.*emulator`),
			regexp.MustCompile(`(?i)goldfish`),
			regexp.MustCompile(`(?i)ranchu`),
		},
		Indicators: []string{
			"GenericAndroid",
			"sdk_phone_x86",
		},
		Weight: 0.90,
	}

	s.knownEmulators["iOS_Simulator"] = &EmulatorSignature{
		Name: "iOS Simulator",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)iphonesimulator`),
			regexp.MustCompile(`(?i)ipadsimulator`),
		},
		Indicators: []string{
			"iOS Simulator",
			"CFNetwork",
		},
		Weight: 0.85,
	}

	s.knownEmulators["BlueStacks"] = &EmulatorSignature{
		Name: "BlueStacks",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)bluestacks`),
			regexp.MustCompile(`(?i)bst.*(?:special|split|instance)`),
			regexp.MustCompile(`(?i)android.*bluestacks`),
		},
		Indicators: []string{
			"BlueStacks",
			"bst-helper",
			"bstsdfsdksandbox",
			"Android on x86",
		},
		Weight: 0.95,
	}

	s.knownEmulators["Nox"] = &EmulatorSignature{
		Name: "Nox",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)nox`),
			regexp.MustCompile(`(?i)noxplayer`),
			regexp.MustCompile(`(?i)noxsandbox`),
		},
		Indicators: []string{
			"Nox",
			"NoxPlayer",
			"NoxApp",
			"android-virtualbox",
		},
		Weight: 0.93,
	}

	s.knownEmulators["Memu"] = &EmulatorSignature{
		Name: "Memu",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)memu`),
			regexp.MustCompile(`(?i)memuplay`),
			regexp.MustCompile(`(?i)memuplayer`),
		},
		Indicators: []string{
			"Memu",
			"MemuPlayer",
			"Microvirt",
			"Android on MEmu",
		},
		Weight: 0.92,
	}

	s.knownEmulators["LDPlayer"] = &EmulatorSignature{
		Name: "LDPlayer",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)ldplayer`),
			regexp.MustCompile(`(?i)ld-play`),
			regexp.MustCompile(`(?i)leidian`),
		},
		Indicators: []string{
			"LDPlayer",
			"LeiDroid",
			"ldlib",
			"ldvbox",
		},
		Weight: 0.91,
	}

	s.knownEmulators["Mumu"] = &EmulatorSignature{
		Name: "Mumu",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)mumu`),
			regexp.MustCompile(`(?i)mumux`),
			regexp.MustCompile(`(?i)xiaomu`),
		},
		Indicators: []string{
			"MuMu",
			"mumu模拟器",
			"Netease MuMu",
			"mumu_x86",
		},
		Weight: 0.88,
	}

	s.knownEmulators["Genymotion"] = &EmulatorSignature{
		Name: "Genymotion",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)genymotion`),
			regexp.MustCompile(`(?i)genyotion`),
		},
		Indicators: []string{
			"Genymotion",
			"vbox86p",
			"vbox86t",
			"Genymobile",
		},
		Weight: 0.94,
	}

	s.knownEmulators["Gameloop"] = &EmulatorSignature{
		Name: "Gameloop",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)gameloop`),
			regexp.MustCompile(`(?i)tencentgameloop`),
			regexp.MustCompile(`(?i)txgame`),
		},
		Indicators: []string{
			"GameLoop",
			"Tencent Gaming Buddy",
			"android-gameloop",
		},
		Weight: 0.90,
	}

	s.knownEmulators["SmartGaGa"] = &EmulatorSignature{
		Name: "SmartGaGa",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)smartgaga`),
			regexp.MustCompile(`(?i)smartga`),
		},
		Indicators: []string{
			"SmartGaGa",
			"SmartGaga",
			"windows-sandbox",
		},
		Weight: 0.87,
	}

	s.knownEmulators["WindRoy"] = &EmulatorSignature{
		Name: "WindRoy",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)windroy`),
			regexp.MustCompile(`(?i)windroye`),
		},
		Indicators: []string{
			"WindRoy",
			"WindRoye",
			"Microvirt",
		},
		Weight: 0.85,
	}

	s.knownEmulators["Droid4X"] = &EmulatorSignature{
		Name: "Droid4X",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)droid4x`),
			regexp.MustCompile(`(?i)macsigner`),
		},
		Indicators: []string{
			"Droid4X",
			"droid4x-system",
			"com.hypergryph.arknights",
		},
		Weight: 0.84,
	}

	s.knownEmulators["YouWei"] = &EmulatorSignature{
		Name: "YouWei",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)youwei`),
			regexp.MustCompile(`(?i)youwin`),
		},
		Indicators: []string{
			"YouWei",
			"YouWin",
			"android-wechat",
		},
		Weight: 0.82,
	}

	s.knownEmulators["雷电云"] = &EmulatorSignature{
		Name: "雷电云",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)ldyun`),
			regexp.MustCompile(`(?i)ldy`),
			regexp.MustCompile(`(?i)lei(dian)?.*cloud`),
		},
		Indicators: []string{
			"LDYun",
			"ldy_api",
			"android-cloud-ldy",
			"雷电云手机",
		},
		Weight: 0.96,
	}

	s.knownEmulators["多多云"] = &EmulatorSignature{
		Name: "多多云",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)ddyun`),
			regexp.MustCompile(`(?i)ddy`),
			regexp.MustCompile(`(?i)duoduo.*cloud`),
		},
		Indicators: []string{
			"DDYun",
			"ddy_api",
			"android-cloud-ddy",
			"多多云手机",
		},
		Weight: 0.95,
	}

	s.knownEmulators["红警"] = &EmulatorSignature{
		Name: "红警云手机",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)hongji`),
			regexp.MustCompile(`(?i)redalert`),
			regexp.MustCompile(`(?i)red.?alert`),
		},
		Indicators: []string{
			"HongJi",
			"hongjicloud",
			"android-cloud-hj",
			"红警云",
		},
		Weight: 0.94,
	}

	s.knownEmulators["双子云"] = &EmulatorSignature{
		Name: "双子云手机",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)shuangzi`),
			regexp.MustCompile(`(?i)gemini.*cloud`),
		},
		Indicators: []string{
			"ShuangZi",
			"szcloud",
			"android-cloud-sz",
		},
		Weight: 0.90,
	}

	s.knownEmulators["蜂窝云"] = &EmulatorSignature{
		Name: "蜂窝云手机",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)fengwo`),
			regexp.MustCompile(`(?i)fwcloud`),
			regexp.MustCompile(`(?i)beecow`),
		},
		Indicators: []string{
			"FengWo",
			"FWCloud",
			"android-cloud-fw",
			"蜂窝云",
		},
		Weight: 0.89,
	}

	s.knownEmulators["云帅云"] = &EmulatorSignature{
		Name: "云帅云手机",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)yunshuai`),
			regexp.MustCompile(`(?i)yscloud`),
		},
		Indicators: []string{
			"YunShuai",
			"YSCloud",
			"android-cloud-ys",
		},
		Weight: 0.88,
	}

	s.knownEmulators["蓝光云"] = &EmulatorSignature{
		Name: "蓝光云手机",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)languang`),
			regexp.MustCompile(`(?i)lgcloud`),
			regexp.MustCompile(`(?i)bluelight`),
		},
		Indicators: []string{
			"LanGuang",
			"LGCloud",
			"android-cloud-lg",
		},
		Weight: 0.87,
	}

	s.knownEmulators["山寨云"] = &EmulatorSignature{
		Name: "山寨云手机",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)shanzhai`),
			regexp.MustCompile(`(?i)szcloud`),
		},
		Indicators: []string{
			"ShanZhai",
			"SZCloud",
			"android-cloud-sz",
		},
		Weight: 0.86,
	}

	s.knownVMs["VMware"] = &VMSignature{
		Name: "VMware",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)vmware`),
			regexp.MustCompile(`(?i)vmware.*virtual`),
			regexp.MustCompile(`(?i)vmware[_-]? Tools`),
		},
		Indicators: []string{
			"VMware7,1",
			"VMware Virtual Platform",
			"VMware Vista",
			"VMware7",
		},
		Weight: 0.95,
	}

	s.knownVMs["VirtualBox"] = &VMSignature{
		Name: "VirtualBox",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)virtualbox`),
			regexp.MustCompile(`(?i)vbox`),
			regexp.MustCompile(`(?i)vboxclient`),
		},
		Indicators: []string{
			"VirtualBox",
			"VBOX",
			"vbox86p",
			"vbox86t",
			"VBoxSharedFolders",
		},
		Weight: 0.92,
	}

	s.knownVMs["QEMU_KVM"] = &VMSignature{
		Name: "QEMU/KVM",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)qemu`),
			regexp.MustCompile(`(?i)kvm`),
			regexp.MustCompile(`(?i)tcg`),
		},
		Indicators: []string{
			"QEMU Virtual CPU",
			"KVM",
			"Standard PC (Q35 + ICH9",
			"TCG",
		},
		Weight: 0.88,
	}

	s.knownVMs["HyperV"] = &VMSignature{
		Name: "Hyper-V",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)hyper[_-]?v`),
			regexp.MustCompile(`(?i)microsoft.*virtual`),
		},
		Indicators: []string{
			"Microsoft Hyper-V",
			"Virtual Machine",
			"HYPER-V",
			"Microsoft Corporation",
		},
		Weight: 0.90,
	}

	s.knownVMs["Parallels"] = &VMSignature{
		Name: "Parallels",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)parallels`),
			regexp.MustCompile(`(?i)prl_`),
		},
		Indicators: []string{
			"Parallels",
			"Parallels Virtual Platform",
			"prl_vm_app",
		},
		Weight: 0.89,
	}

	s.knownVMs["Xen"] = &VMSignature{
		Name: "Xen",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)xen`),
		},
		Indicators: []string{
			"Xen",
			"HVM domU",
			"XenSource",
		},
		Weight: 0.87,
	}

	s.knownContainers["Docker"] = &ContainerSignature{
		Name: "Docker",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)docker`),
		},
		Indicators: []string{
			"/.dockerenv",
			"container=docker",
			"docker-init",
			"docker-cluster",
		},
		Weight: 0.85,
	}

	s.knownContainers["Kubernetes"] = &ContainerSignature{
		Name: "Kubernetes",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)kubernetes`),
			regexp.MustCompile(`(?i)k8s`),
		},
		Indicators: []string{
			"KUBERNETES_SERVICE_PORT",
			"kubernetes.io",
			"kube-cluster",
		},
		Weight: 0.82,
	}

	s.knownContainers["LXC"] = &ContainerSignature{
		Name: "LXC",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)lxc`),
		},
		Indicators: []string{
			"/dev/lxd/sock",
			"lxc/",
			"machine-id",
		},
		Weight: 0.80,
	}

	s.knownContainers["cgroup"] = &ContainerSignature{
		Name: "Container (cgroup)",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)container`),
		},
		Indicators: []string{
			"1:freezer:/",
			"1:name=systemd:",
			"/sys/fs/cgroup/",
		},
		Weight: 0.75,
	}
}

func (s *DeviceDetectionService) DetectDevice(data map[string]interface{}) *DeviceDetectionResult {
	result := &DeviceDetectionResult{
		Indicators:       make([]string, 0),
		DetectionMethods: make([]string, 0),
	}

	s.detectEmulator(data, result)
	s.detectVirtualMachine(data, result)
	s.detectContainer(data, result)
	s.detectMultiBox(data, result)
	s.detectDeviceType(data, result)
	s.calculateFinalScore(result)

	return result
}

func (s *DeviceDetectionService) detectEmulator(data map[string]interface{}, result *DeviceDetectionResult) {
	method := "emulator_detection"
	result.DetectionMethods = append(result.DetectionMethods, method)

	userAgent := getStringFromData(data, "user_agent")

	for _, signature := range s.knownEmulators {
		for _, pattern := range signature.Patterns {
			if pattern.MatchString(userAgent) {
				result.Indicators = append(result.Indicators, fmt.Sprintf("emulator_match:%s", signature.Name))
				result.Score += signature.Weight * 80
			}
		}

		for _, indicator := range signature.Indicators {
			if strings.Contains(userAgent, indicator) {
				result.Indicators = append(result.Indicators, fmt.Sprintf("emulator_indicator:%s", indicator))
				result.Score += signature.Weight * 70
			}
		}
	}

	if navigatorProps, ok := data["navigator_properties"].(map[string]interface{}); ok {
		if maxTouchPoints, ok := navigatorProps["maxTouchPoints"].(float64); ok && maxTouchPoints == 0 {
			if isMobileUA(userAgent) {
				result.Indicators = append(result.Indicators, "no_touch_mobile_emulator")
				result.Score += 40
			}
		}

		if platform, ok := navigatorProps["platform"].(string); ok {
			if strings.Contains(strings.ToLower(platform), "linux") && isMobileUA(userAgent) {
				result.Indicators = append(result.Indicators, "linux_mobile_emulator")
				result.Score += 50
			}
		}
	}

	if webglRenderer, ok := data["webgl_renderer"].(string); ok {
		if strings.Contains(strings.ToLower(webglRenderer), "swiftshader") ||
			strings.Contains(strings.ToLower(webglRenderer), "llvmpipe") {
			result.Indicators = append(result.Indicators, "software_renderer_emulator")
			result.Score += 45
		}
	}

	result.IsEmulator = result.Score >= 60
}

func (s *DeviceDetectionService) detectVirtualMachine(data map[string]interface{}, result *DeviceDetectionResult) {
	method := "vm_detection"
	result.DetectionMethods = append(result.DetectionMethods, method)

	userAgent := getStringFromData(data, "user_agent")
	webglRenderer := getStringFromData(data, "webgl_renderer")

	for _, signature := range s.knownVMs {
		for _, pattern := range signature.Patterns {
			if pattern.MatchString(userAgent) || pattern.MatchString(webglRenderer) {
				result.Indicators = append(result.Indicators, fmt.Sprintf("vm_match:%s", signature.Name))
				result.Score += signature.Weight * 85
			}
		}

		for _, indicator := range signature.Indicators {
			if strings.Contains(userAgent, indicator) || strings.Contains(webglRenderer, indicator) {
				result.Indicators = append(result.Indicators, fmt.Sprintf("vm_indicator:%s", indicator))
				result.Score += signature.Weight * 75
			}
		}
	}

	if hardwareInfo, ok := data["hardware_info"].(map[string]interface{}); ok {
		if cpuCores, ok := hardwareInfo["cpu_cores"].(float64); ok {
			if cpuCores == 1 || cpuCores > 32 {
				result.Indicators = append(result.Indicators, "unusual_cpu_cores")
				result.Score += 30
			}
		}

		if memory, ok := hardwareInfo["device_memory"].(float64); ok {
			if memory < 0.5 || memory > 128 {
				result.Indicators = append(result.Indicators, "unusual_memory")
				result.Score += 25
			}
		}
	}

	screenWidth := getIntFromData(data, "screen_width")
	screenHeight := getIntFromData(data, "screen_height")
	if screenWidth > 0 && screenHeight > 0 {
		if isMobileUA(userAgent) && (screenWidth > 1920 || screenHeight > 1080) {
			result.Indicators = append(result.Indicators, "mobile_ua_desktop_res")
			result.Score += 35
		}
	}

	result.IsVirtual = result.Score >= 70
}

func (s *DeviceDetectionService) detectContainer(data map[string]interface{}, result *DeviceDetectionResult) {
	method := "container_detection"
	result.DetectionMethods = append(result.DetectionMethods, method)

	userAgent := getStringFromData(data, "user_agent")

	for _, signature := range s.knownContainers {
		for _, pattern := range signature.Patterns {
			if pattern.MatchString(userAgent) {
				result.Indicators = append(result.Indicators, fmt.Sprintf("container_match:%s", signature.Name))
				result.Score += signature.Weight * 80
			}
		}

		for _, indicator := range signature.Indicators {
			if strings.Contains(userAgent, indicator) {
				result.Indicators = append(result.Indicators, fmt.Sprintf("container_indicator:%s", indicator))
				result.Score += signature.Weight * 70
			}
		}
	}

	if navigatorProps, ok := data["navigator_properties"].(map[string]interface{}); ok {
		if storageEstimate, ok := navigatorProps["storage_estimate"].(map[string]interface{}); ok {
			if quota, ok := storageEstimate["quota"].(float64); ok {
				if quota == 0 {
					result.Indicators = append(result.Indicators, "zero_storage_container")
					result.Score += 50
				}
			}
		}

		if cookies, ok := navigatorProps["cookieEnabled"].(bool); ok && !cookies {
			result.Indicators = append(result.Indicators, "cookies_disabled_container")
			result.Score += 40
		}
	}

	result.IsContainer = result.Score >= 65
}

func (s *DeviceDetectionService) detectMultiBox(data map[string]interface{}, result *DeviceDetectionResult) {
	method := "multi_box_detection"
	result.DetectionMethods = append(result.DetectionMethods, method)

	if deviceFP, ok := data["device_fingerprint"].(string); ok && deviceFP != "" {
		fingerprintID := s.generateDeviceFingerprintID(deviceFP)

		s.mu.Lock()
		defer s.mu.Unlock()

		if existingData, exists := s.deviceDatabase[fingerprintID]; exists {
			if time.Since(existingData.LastSeen) < 5*time.Minute && existingData.RequestCount > 10 {
				result.Indicators = append(result.Indicators, "rapid_requests_same_device")
				result.Score += 60
				result.IsMultiBox = true
			}
		}
	}

	if sessionData, ok := data["session_data"].(map[string]interface{}); ok {
		if _, ok := sessionData["session_id"].(string); ok {
			s.mu.Lock()
			sessionCount := 0
			for _, deviceData := range s.deviceDatabase {
				if deviceData.RequestCount > 5 {
					sessionCount++
				}
			}
			s.mu.Unlock()

			if sessionCount > 5 {
				result.Indicators = append(result.Indicators, "multiple_sessions_device")
				result.Score += 50
				result.IsMultiBox = true
			}
		}
	}

	if ipData, ok := data["ip_data"].(map[string]interface{}); ok {
		if concurrentConnections, ok := ipData["concurrent_connections"].(float64); ok && concurrentConnections > 3 {
			result.Indicators = append(result.Indicators, "multiple_concurrent_connections")
			result.Score += 45
			result.IsMultiBox = true
		}
	}
}

func (s *DeviceDetectionService) detectDeviceType(data map[string]interface{}, result *DeviceDetectionResult) {
	method := "device_type_detection"
	result.DetectionMethods = append(result.DetectionMethods, method)

	userAgent := getStringFromData(data, "user_agent")

	if regexp.MustCompile(`(?i)mobile|android|iphone|ipad`).MatchString(userAgent) {
		result.DeviceType = DeviceMobile
	} else if regexp.MustCompile(`(?i)tablet|ipad`).MatchString(userAgent) {
		result.DeviceType = DeviceTablet
	} else if regexp.MustCompile(`(?i)tv|smarttv|googletv`).MatchString(userAgent) {
		result.DeviceType = DeviceTV
	} else {
		result.DeviceType = DeviceDesktop
	}
}

func (s *DeviceDetectionService) calculateFinalScore(result *DeviceDetectionResult) {
	result.Score = math.Min(result.Score, 100)
	result.Confidence = result.Score / 100.0

	if result.IsEmulator {
		result.EnvironmentType = EnvEmulator
	} else if result.IsVirtual {
		result.EnvironmentType = EnvVirtual
	} else if result.IsContainer {
		result.EnvironmentType = EnvContainer
	} else if result.IsMultiBox {
		result.EnvironmentType = EnvMultiBox
	} else {
		result.EnvironmentType = EnvReal
	}
}

func (s *DeviceDetectionService) RecordDeviceFingerprint(data map[string]interface{}) string {
	fingerprintID := s.generateDeviceFingerprintIDFromData(data)

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if existingData, exists := s.deviceDatabase[fingerprintID]; exists {
		existingData.LastSeen = now
		existingData.RequestCount++
	} else {
		hardwareInfo := s.extractHardwareInfo(data)
		softwareInfo := s.extractSoftwareInfo(data)
		networkInfo := s.extractNetworkInfo(data)

		s.deviceDatabase[fingerprintID] = &DeviceFingerprintData{
			FingerprintID:  fingerprintID,
			HardwareInfo:   hardwareInfo,
			SoftwareInfo:   softwareInfo,
			NetworkInfo:    networkInfo,
			StabilityScore: 100.0,
			FirstSeen:      now,
			LastSeen:       now,
			RequestCount:   1,
		}
	}

	return fingerprintID
}

func (s *DeviceDetectionService) generateDeviceFingerprintID(deviceFP string) string {
	hash := sha256.Sum256([]byte(deviceFP))
	return hex.EncodeToString(hash[:16])
}

func (s *DeviceDetectionService) generateDeviceFingerprintIDFromData(data map[string]interface{}) string {
	components := make([]string, 0)

	if ua, ok := data["user_agent"].(string); ok {
		components = append(components, ua)
	}

	if screen, ok := data["screen_resolution"].(string); ok {
		components = append(components, screen)
	}

	if tz, ok := data["timezone"].(string); ok {
		components = append(components, tz)
	}

	if platform, ok := data["platform"].(string); ok {
		components = append(components, platform)
	}

	combined := strings.Join(components, "|")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:16])
}

func (s *DeviceDetectionService) extractHardwareInfo(data map[string]interface{}) *HardwareInfo {
	info := &HardwareInfo{}

	if hw, ok := data["hardware_info"].(map[string]interface{}); ok {
		if cpu, ok := hw["cpu_info"].(string); ok {
			info.CPUInfo = cpu
		}
		if cores, ok := hw["cpu_cores"].(float64); ok {
			info.CPUCores = int(cores)
		}
		if mem, ok := hw["device_memory"].(float64); ok {
			info.DeviceMemory = mem
		}
		if gpu, ok := hw["gpu_info"].(string); ok {
			info.GPUInfo = gpu
		}
	}

	if screen, ok := data["screen_info"].(map[string]interface{}); ok {
		if width, ok := screen["width"].(float64); ok {
			info.ScreenWidth = int(width)
		}
		if height, ok := screen["height"].(float64); ok {
			info.ScreenHeight = int(height)
		}
		if depth, ok := screen["color_depth"].(float64); ok {
			info.ScreenColorDepth = int(depth)
		}
		if ratio, ok := screen["pixel_ratio"].(float64); ok {
			info.PixelRatio = ratio
		}
	}

	return info
}

func (s *DeviceDetectionService) extractSoftwareInfo(data map[string]interface{}) *SoftwareInfo {
	info := &SoftwareInfo{}

	if ua, ok := data["user_agent"].(string); ok {
		info.UserAgent = ua
		parts := strings.Split(ua, " ")
		if len(parts) > 0 {
			info.Browser = parts[0]
		}
	}

	if tz, ok := data["timezone"].(string); ok {
		info.Timezone = tz
	}

	if lang, ok := data["language"].(string); ok {
		info.Language = lang
	}

	return info
}

func (s *DeviceDetectionService) extractNetworkInfo(data map[string]interface{}) *NetworkInfo {
	info := &NetworkInfo{}

	if conn, ok := data["connection_info"].(map[string]interface{}); ok {
		if connType, ok := conn["type"].(string); ok {
			info.ConnectionType = connType
		}
		if effType, ok := conn["effective_type"].(string); ok {
			info.EffectiveType = effType
		}
		if rtt, ok := conn["rtt"].(float64); ok {
			info.RTT = int(rtt)
		}
		if dl, ok := conn["downlink"].(float64); ok {
			info.Downlink = dl
		}
	}

	return info
}

func (s *DeviceDetectionService) GetDeviceFingerprint(fingerprintID string) (*DeviceFingerprintData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.deviceDatabase[fingerprintID]
	return data, exists
}

func (s *DeviceDetectionService) CalculateStabilityScore(fingerprintID string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.deviceDatabase[fingerprintID]
	if !exists {
		return 0
	}

	score := 100.0

	if data.RequestCount < 5 {
		score -= (5 - float64(data.RequestCount)) * 10
	}

	timeSinceFirst := time.Since(data.FirstSeen)
	if timeSinceFirst < 24*time.Hour {
		score -= 20
	}

	return math.Max(score, 0)
}

func (s *DeviceDetectionService) CleanupOldData(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, data := range s.deviceDatabase {
		if data.LastSeen.Before(cutoff) && data.RequestCount < 10 {
			delete(s.deviceDatabase, id)
			removed++
		}
	}

	return removed
}

func isMobileUA(ua string) bool {
	return regexp.MustCompile(`(?i)mobile|android|iphone|ipad`).MatchString(ua)
}

func getStringFromData(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getIntFromData(data map[string]interface{}, key string) int {
	if val, ok := data[key].(float64); ok {
		return int(val)
	}
	return 0
}
