package service

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

type VMType string

const (
	VMTypeNone          VMType = "none"
	VMTypeVMware        VMType = "vmware"
	VMTypeVirtualBox    VMType = "virtualbox"
	VMTypeHyperV        VMType = "hyperv"
	VMTypeKVM           VMType = "kvm"
	VMTypeQEMU          VMType = "qemu"
	VMTypeXen           VMType = "xen"
	VMTypeParallels     VMType = "parallels"
	VMTypeDocker        VMType = "docker"
	VMTypeKubernetes    VMType = "kubernetes"
	VMTypeAWS           VMType = "aws"
	VMTypeAzure         VMType = "azure"
	VMTypeGCP           VMType = "gcp"
	VMTypeAlibabaCloud  VMType = "alibaba_cloud"
	VMTypeTencentCloud  VMType = "tencent_cloud"
)

type EmulatorType string

const (
	EmulatorTypeNone     EmulatorType = "none"
	EmulatorTypeAndroid   EmulatorType = "android"
	EmulatorTypeIOS       EmulatorType = "ios"
	EmulatorTypeBlueStacks EmulatorType = "bluestacks"
	EmulatorTypeNox       EmulatorType = "nox"
	EmulatorTypeLDPlayer  EmulatorType = "ldplayer"
	EmulatorTypeMEmu     EmulatorType = "memu"
	EmulatorTypeGenymotion EmulatorType = "genymotion"
)

type AutomationType string

const (
	AutomationTypeNone       AutomationType = "none"
	AutomationTypeSelenium    AutomationType = "selenium"
	AutomationTypePuppeteer   AutomationType = "puppeteer"
	AutomationTypePlaywright  AutomationType = "playwright"
	AutomationTypeCypress    AutomationType = "cypress"
	AutomationTypePhantomJS  AutomationType = "phantomjs"
	AutomationTypeNightmare  AutomationType = "nightmare"
)

type IPRiskLevel string

const (
	IPRiskLevelLow      IPRiskLevel = "low"
	IPRiskLevelMedium   IPRiskLevel = "medium"
	IPRiskLevelHigh     IPRiskLevel = "high"
	IPRiskLevelCritical IPRiskLevel = "critical"
)

type EnvironmentDetectionResult struct {
	IsVM           bool               `json:"is_vm"`
	IsContainer    bool               `json:"is_container"`
	IsCloud        bool               `json:"is_cloud"`
	IsEmulator     bool               `json:"is_emulator"`
	IsAutomated    bool               `json:"is_automated"`
	IsVPN          bool               `json:"is_vpn"`
	IsProxy        bool               `json:"is_proxy"`
	IsTor          bool               `json:"is_tor"`
	IsHosting      bool               `json:"is_hosting"`

	VMType         VMType             `json:"vm_type"`
	EmulatorType   EmulatorType      `json:"emulator_type"`
	AutomationType AutomationType     `json:"automation_type"`
	CloudProvider  VMType             `json:"cloud_provider"`

	IPRiskLevel    IPRiskLevel       `json:"ip_risk_level"`
	IPRiskScore    float64            `json:"ip_risk_score"`
	IPCountry      string             `json:"ip_country"`
	IPASN          string             `json:"ip_asn"`
	IsMaliciousIP  bool               `json:"is_malicious_ip"`

	RiskScore      float64            `json:"risk_score"`
	Confidence     float64            `json:"confidence"`
	Reasons        []string           `json:"reasons"`
	Indicators     []string           `json:"indicators"`
	Metadata       map[string]string  `json:"metadata,omitempty"`
}

type VMDetectionIndicators struct {
	CPUIDFeatures  []string
	MACAddresses   []string
	BIOSInfo       string
	SystemProduct  string
	DiskDevices    []string
	MountedDevices []string
	ProcFiles      []string
	DMIDecode      string
	LSHWOutput     string
	VirtWhat       string
}

type IPReputationData struct {
	IPAddress       string
	RiskLevel       IPRiskLevel
	RiskScore       float64
	Country         string
	ASN             string
	ISP             string
	IsProxy         bool
	IsVPN           bool
	IsTor           bool
	IsHosting       bool
	IsCloud         bool
	IsDatacenter    bool
	IsMobileCarrier bool
	IsResidential   bool
	IsKnownMalicious bool
	FirstSeen       time.Time
	LastSeen        time.Time
	Reports         []IPReport
}

type IPReport struct {
	Source    string    `json:"source"`
	Reason    string    `json:"reason"`
	Count     int       `json:"count"`
	Timestamp time.Time `json:"timestamp"`
}

type ContainerInfo struct {
	IsContainer    bool
	ContainerType  string
	ContainerID    string
	ImageName      string
	PodName        string
	Namespace      string
	CloudProvider  string
}

type EnvironmentDetectionService struct {
	mu           sync.RWMutex
	ipCache      map[string]*IPReputationData
	ipCacheExpiry time.Duration

	knownVPNRanges    []*net.IPNet
	knownProxyPorts   []int
	torExitNodes      map[string]bool
	knownMaliciousIPs map[string]bool

	vmPatterns       map[VMType][]*regexp.Regexp
	emulatorPatterns map[EmulatorType][]*regexp.Regexp
	automationPatterns map[AutomationType][]*regexp.Regexp

	maxIPCacheSize   int
	containerChecked  bool
	containerInfo    *ContainerInfo
}

var (
	vmwarePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)vmware`),
		regexp.MustCompile(`(?i)vmware[_\s]?virtual[_\s]?platform`),
		regexp.MustCompile(`(?i)vmware[_\s]?virtual[_\s]?disk`),
	}

	virtualBoxPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)virtualbox`),
		regexp.MustCompile(`(?i)vbox`),
		regexp.MustCompile(`(?i)oracle[_\s]?virtualbox`),
	}

	hyperVPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)hyper[_\s]?v`),
		regexp.MustCompile(`(?i)microsoft[_\s]?corporation[_\s]?virtual`),
		regexp.MustCompile(`(?i)virtual[_\s]?machine`),
	}

	kvmPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)kvm`),
		regexp.MustCompile(`(?i)kernel[_\s]?based[_\s]?virtual`),
		regexp.MustCompile(`(?i)qemu[_\s]?kvm`),
	}

	dockerPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)docker`),
		regexp.MustCompile(`(?i)containerd`),
		regexp.MustCompile(`(?i)moby`),
		regexp.MustCompile(`\.docker\.internal$`),
	}

	kubernetesPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)kubernetes`),
		regexp.MustCompile(`(?i)k8s`),
		regexp.MustCompile(`\.kubernetes\.local$`),
	}

	cloudPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)amazon[_\s]?ec2`),
		regexp.MustCompile(`(?i)aws`),
		regexp.MustCompile(`\.aws\.`),
		regexp.MustCompile(`(?i)azure`),
		regexp.MustCompile(`\.cloudapp\.net$`),
		regexp.MustCompile(`(?i)google[_\s]?cloud`),
		regexp.MustCompile(`(?i)gcp`),
		regexp.MustCompile(`\.compute\.internal$`),
		regexp.MustCompile(`(?i)alibaba[_\s]?cloud`),
		regexp.MustCompile(`\.aliyuncs\.com$`),
		regexp.MustCompile(`(?i)tencent`),
		regexp.MustCompile(`\.tencent\.com$`),
	}

	androidEmulatorPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)android[_\s]?emulator`),
		regexp.MustCompile(`(?i)goldfish`),
		regexp.MustCompile(`(?i)ranchu`),
		regexp.MustCompile(`(?i)sdk[_\s]?phone`),
		regexp.MustCompile(`(?i)sdk[_\s]?gphone`),
		regexp.MustCompile(`(?i)generic[_\s]?arm64`),
		regexp.MustCompile(`(?i)generic[_\s]?x86`),
		regexp.MustCompile(`(?i)phone[_\s]?emulator`),
		regexp.MustCompile(`(?i)screen[_\s]?width[_\s]?height`),
	}

	iosEmulatorPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)iphone[_\s]?simulator`),
		regexp.MustCompile(`(?i)ipad[_\s]?simulator`),
		regexp.MustCompile(`(?i)ios[_\s]?simulator`),
		regexp.MustCompile(`(?i)x86[_\s]?64`),
	}

	bluestacksPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)bluestacks`),
		regexp.MustCompile(`(?i)bst[_\s]?instance`),
	}

	noxPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)nox[_\s]?app[_\s]?player`),
		regexp.MustCompile(`(?i)nox[_\s]?emulator`),
	}

	genymotionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)genymotion`),
		regexp.MustCompile(`(?i)geny[_\s]?motion`),
		regexp.MustCompile(`(?i)vbox86p`),
	}

	seleniumPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)selenium`),
		regexp.MustCompile(`(?i)webdriver`),
		regexp.MustCompile(`__selenium_evaluate`),
		regexp.MustCompile(`__webdriver_evaluate`),
		regexp.MustCompile(`__driver_evaluate`),
		regexp.MustCompile(`__fxdriver_evaluate`),
		regexp.MustCompile(`__webdriver_unwrapped`),
		regexp.MustCompile(`__lastWatirAlert`),
		regexp.MustCompile(`__$webdriverAsyncExecutor`),
		regexp.MustCompile(`callSelenium`),
		regexp.MustCompile(`Selenium`),
	}

	puppeteerPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)puppeteer`),
		regexp.MustCompile(`\$cdc_asdjflasutopfhvcZLmcfl_`),
		regexp.MustCompile(`\$chrome_asyncScriptInfo`),
		regexp.MustCompile(`_puppeteer_globals`),
		regexp.MustCompile(`(?i)_?puppeteer_`),
	}

	playwrightPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)playwright`),
		regexp.MustCompile(`__playwright__`),
		regexp.MustCompile(`__pw_tags`),
		regexp.MustCompile(`__pw_resume__`),
		regexp.MustCompile(`__pw_bound__`),
		regexp.MustCompile(`__playwright__execution`),
	}

	cypressPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)cypress`),
		regexp.MustCompile(`__cypress`),
		regexp.MustCompile(`cypress[_\s]?agent`),
		regexp.MustCompile(`__cypress[_\s]?`),
	}

	privateIPRanges = []*net.IPNet{
		parseCIDR("10.0.0.0/8"),
		parseCIDR("172.16.0.0/12"),
		parseCIDR("192.168.0.0/16"),
		parseCIDR("127.0.0.0/8"),
	}

	linkLocalRange = parseCIDR("169.254.0.0/16")

	knownVPNSubnets = []*net.IPNet{
		parseCIDR("5.0.0.0/8"),
		parseCIDR("45.0.0.0/8"),
		parseCIDR("85.0.0.0/8"),
		parseCIDR("91.0.0.0/8"),
		parseCIDR("92.0.0.0/8"),
		parseCIDR("93.0.0.0/8"),
		parseCIDR("185.0.0.0/8"),
		parseCIDR("188.0.0.0/8"),
		parseCIDR("193.0.0.0/8"),
		parseCIDR("195.0.0.0/8"),
		parseCIDR("212.0.0.0/8"),
		parseCIDR("213.0.0.0/8"),
		parseCIDR("217.0.0.0/8"),
		parseCIDR("37.0.0.0/8"),
		parseCIDR("46.0.0.0/8"),
		parseCIDR("94.0.0.0/8"),
		parseCIDR("109.0.0.0/8"),
		parseCIDR("176.0.0.0/8"),
		parseCIDR("178.0.0.0/8"),
		parseCIDR("185.0.0.0/8"),
	}

	cloudProviderCIDRs = map[VMType][]*net.IPNet{
		VMTypeAWS: {
			parseCIDR("3.0.0.0/8"),
			parseCIDR("18.0.0.0/8"),
			parseCIDR("52.0.0.0/8"),
			parseCIDR("54.0.0.0/8"),
			parseCIDR("204.246.0.0/8"),
			parseCIDR("205.251.192.0/19"),
		},
		VMTypeAzure: {
			parseCIDR("13.64.0.0/11"),
			parseCIDR("20.0.0.0/8"),
			parseCIDR("40.0.0.0/8"),
			parseCIDR("52.0.0.0/8"),
			parseCIDR("104.0.0.0/8"),
			parseCIDR("137.0.0.0/8"),
			parseCIDR("138.0.0.0/8"),
		},
		VMTypeGCP: {
			parseCIDR("8.0.0.0/8"),
			parseCIDR("23.0.0.0/8"),
			parseCIDR("34.0.0.0/8"),
			parseCIDR("35.0.0.0/8"),
			parseCIDR("104.0.0.0/8"),
			parseCIDR("107.0.0.0/8"),
			parseCIDR("108.0.0.0/8"),
			parseCIDR("142.0.0.0/8"),
			parseCIDR("172.0.0.0/8"),
		},
	}

	hostingProviderPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)digitalocean`),
		regexp.MustCompile(`(?i)droplet`),
		regexp.MustCompile(`(?i)linode`),
		regexp.MustCompile(`(?i)vultr`),
		regexp.MustCompile(`(?i)ovh`),
		regexp.MustCompile(`(?i)hetzner`),
		regexp.MustCompile(`(?i)contabo`),
		regexp.MustCompile(`(?i)dedicated`),
		regexp.MustCompile(`(?i)colocated`),
		regexp.MustCompile(`(?i)hosting`),
	}
)

func parseCIDR(cidr string) *net.IPNet {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return &net.IPNet{}
	}
	return ipNet
}

func NewEnvironmentDetectionService() *EnvironmentDetectionService {
	s := &EnvironmentDetectionService{
		ipCache:          make(map[string]*IPReputationData),
		ipCacheExpiry:    24 * time.Hour,
		torExitNodes:     make(map[string]bool),
		knownMaliciousIPs: make(map[string]bool),
		vmPatterns:       make(map[VMType][]*regexp.Regexp),
		emulatorPatterns: make(map[EmulatorType][]*regexp.Regexp),
		automationPatterns: make(map[AutomationType][]*regexp.Regexp),
		maxIPCacheSize:   10000,
	}

	s.vmPatterns[VMTypeVMware] = vmwarePatterns
	s.vmPatterns[VMTypeVirtualBox] = virtualBoxPatterns
	s.vmPatterns[VMTypeHyperV] = hyperVPatterns
	s.vmPatterns[VMTypeKVM] = kvmPatterns
	s.vmPatterns[VMTypeDocker] = dockerPatterns
	s.vmPatterns[VMTypeKubernetes] = kubernetesPatterns
	s.vmPatterns[VMTypeAWS] = cloudPatterns
	s.vmPatterns[VMTypeAzure] = cloudPatterns
	s.vmPatterns[VMTypeGCP] = cloudPatterns

	s.emulatorPatterns[EmulatorTypeAndroid] = androidEmulatorPatterns
	s.emulatorPatterns[EmulatorTypeIOS] = iosEmulatorPatterns
	s.emulatorPatterns[EmulatorTypeBlueStacks] = bluestacksPatterns
	s.emulatorPatterns[EmulatorTypeNox] = noxPatterns
	s.emulatorPatterns[EmulatorTypeGenymotion] = genymotionPatterns

	s.automationPatterns[AutomationTypeSelenium] = seleniumPatterns
	s.automationPatterns[AutomationTypePuppeteer] = puppeteerPatterns
	s.automationPatterns[AutomationTypePlaywright] = playwrightPatterns
	s.automationPatterns[AutomationTypeCypress] = cypressPatterns

	s.knownVPNRanges = knownVPNSubnets

	s.knownProxyPorts = []int{80, 8080, 3128, 8888, 1080, 9050, 9051, 8118}

	return s
}

func (s *EnvironmentDetectionService) Detect(r *http.Request, additionalData map[string]string) *EnvironmentDetectionResult {
	result := &EnvironmentDetectionResult{
		RiskScore:  0,
		Confidence: 0,
		Reasons:    []string{},
		Indicators: []string{},
		Metadata:   make(map[string]string),
	}

	s.detectVM(r, result)
	s.detectContainer(result)
	s.detectCloud(r, result)
	s.detectEmulator(r, result)
	s.detectAutomation(r, additionalData, result)
	s.detectNetworkAnonymity(r, result)

	ip := getClientIP(r)
	s.checkIPReputation(ip, result)

	s.calculateOverallRisk(result)

	return result
}

func (s *EnvironmentDetectionService) detectVM(r *http.Request, result *EnvironmentDetectionResult) {
	vmIndicators := s.gatherVMIndicators()

	s.checkVMType(vmIndicators, VMTypeVMware, vmwarePatterns, result)
	s.checkVMType(vmIndicators, VMTypeVirtualBox, virtualBoxPatterns, result)
	s.checkVMType(vmIndicators, VMTypeHyperV, hyperVPatterns, result)
	s.checkVMType(vmIndicators, VMTypeKVM, kvmPatterns, result)
	s.checkVMType(vmIndicators, VMTypeDocker, dockerPatterns, result)
	s.checkVMType(vmIndicators, VMTypeKubernetes, kubernetesPatterns, result)

	if s.checkForVMFiles() {
		result.IsVM = true
		result.RiskScore += 15
		result.Confidence += 0.3
		result.Indicators = append(result.Indicators, "vm_files_detected")
		result.Reasons = append(result.Reasons, "VM-specific files detected on system")
	}

	if s.checkCPUIDVMFlags() {
		result.IsVM = true
		result.RiskScore += 25
		result.Confidence += 0.5
		result.Indicators = append(result.Indicators, "cpuid_vm_flags")
		result.Reasons = append(result.Reasons, "CPUID flags indicate virtual machine")
	}

	s.checkDMIInfo(vmIndicators, result)

	if s.checkVirtualDevices() {
		result.IsVM = true
		result.RiskScore += 20
		result.Confidence += 0.4
		result.Indicators = append(result.Indicators, "virtual_devices")
		result.Reasons = append(result.Reasons, "Virtual disk or network devices detected")
	}
}

func (s *EnvironmentDetectionService) gatherVMIndicators() VMDetectionIndicators {
	indicators := VMDetectionIndicators{}

	readFile := func(path string) string {
		data, err := os.ReadFile(path)
		if err != nil {
			return ""
		}
		return string(data)
	}

	indicators.BIOSInfo = readFile("/sys/class/dmi/id/bios_vendor")
	indicators.SystemProduct = readFile("/sys/class/dmi/id/product_name")
	indicators.DMIDecode = readFile("/sys/class/dmi/id/sys_vendor")
	indicators.DMIDecode += readFile("/sys/class/dmi/id/product_family")

	sysClassDmi := "/sys/class/dmi/id"
	if entries, err := os.ReadDir(sysClassDmi); err == nil {
		for _, entry := range entries {
			if content := readFile(sysClassDmi + "/" + entry.Name()); content != "" {
				indicators.ProcFiles = append(indicators.ProcFiles, content)
			}
		}
	}

	if data, err := os.ReadFile("/proc/scsi/scsi"); err == nil {
		indicators.DiskDevices = append(indicators.DiskDevices, string(data))
	}

	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		indicators.ProcFiles = append(indicators.ProcFiles, string(data))
	}

	if data, err := os.ReadFile("/proc/self/status"); err == nil {
		indicators.ProcFiles = append(indicators.ProcFiles, string(data))
	}

	if entries, err := os.ReadDir("/dev/disk/by-id"); err == nil {
		for _, entry := range entries {
			indicators.DiskDevices = append(indicators.DiskDevices, entry.Name())
		}
	}

	if entries, err := os.ReadDir("/sys/class/net"); err == nil {
		for _, entry := range entries {
			macPath := "/sys/class/net/" + entry.Name() + "/address"
			if data, err := os.ReadFile(macPath); err == nil {
				mac := strings.TrimSpace(string(data))
				indicators.MACAddresses = append(indicators.MACAddresses, mac)
			}
		}
	}

	return indicators
}

func (s *EnvironmentDetectionService) checkVMType(indicators VMDetectionIndicators, vmType VMType, patterns []*regexp.Regexp, result *EnvironmentDetectionResult) {
	checkStrings := []string{
		indicators.BIOSInfo,
		indicators.SystemProduct,
		indicators.DMIDecode,
		indicators.LSHWOutput,
		indicators.VirtWhat,
	}
	checkStrings = append(checkStrings, indicators.ProcFiles...)
	checkStrings = append(checkStrings, indicators.DiskDevices...)
	checkStrings = append(checkStrings, indicators.MountedDevices...)

	for _, text := range checkStrings {
		for _, pattern := range patterns {
			if pattern.MatchString(text) {
				result.IsVM = true
				result.VMType = vmType
				result.RiskScore += 20
				result.Confidence += 0.4
				result.Indicators = append(result.Indicators, fmt.Sprintf("vm_type_%s", vmType))
				result.Reasons = append(result.Reasons, fmt.Sprintf("Detected %s virtual machine", vmType))
				return
			}
		}
	}
}

func (s *EnvironmentDetectionService) checkForVMFiles() bool {
	vmFiles := []string{
		"/usr/bin/vmware-toolbox-cmd",
		"/usr/bin/vmware-user",
		"/usr/lib/vmware-tools",
		"/etc/vmware-tools",
		"/usr/lib/virtualbox",
		"/etc/init.d/vboxadd",
		"/opt/VBoxGuestAdditions",
		"/usr/src/linux-headers/include/linux/hyperv",
		"/sys/firmware/acpi/tables/HYPERV",
	}

	for _, file := range vmFiles {
		if _, err := os.Stat(file); err == nil {
			return true
		}
	}
	return false
}

func (s *EnvironmentDetectionService) checkCPUIDVMFlags() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	cpuidPath := "/proc/cpuinfo"
	data, err := os.ReadFile(cpuidPath)
	if err != nil {
		return false
	}

	content := strings.ToLower(string(data))
	vmFlags := []string{
		"vmware",
		"virtualbox",
		"qemu",
		"kvm",
		"hyperv",
		"xen",
		"parallels",
	}

	for _, flag := range vmFlags {
		if strings.Contains(content, flag) {
			return true
		}
	}

	return false
}

func (s *EnvironmentDetectionService) checkDMIInfo(indicators VMDetectionIndicators, result *EnvironmentDetectionResult) {
	vmDMIVendors := map[VMType][]string{
		VMTypeVMware:    {"vmware", "vmware, inc."},
		VMTypeVirtualBox: {"virtualbox", "innotek gmbh", "oracle corporation"},
		VMTypeHyperV:    {"microsoft corporation"},
		VMTypeKVM:       {"kvm", "qemu"},
		VMTypeXen:       {"xen"},
		VMTypeParallels: {"parallels"},
	}

	for vmType, vendors := range vmDMIVendors {
		for _, vendor := range vendors {
			if strings.Contains(strings.ToLower(indicators.BIOSInfo), vendor) ||
				strings.Contains(strings.ToLower(indicators.SystemProduct), vendor) ||
				strings.Contains(strings.ToLower(indicators.DMIDecode), vendor) {

				result.IsVM = true
				result.VMType = vmType
				result.RiskScore += 25
				result.Confidence += 0.6
				result.Indicators = append(result.Indicators, "dmi_vm_vendor")
				result.Reasons = append(result.Reasons, fmt.Sprintf("DMI vendor indicates %s", vmType))
				return
			}
		}
	}
}

func (s *EnvironmentDetectionService) checkVirtualDevices() bool {
	virtualDevices := []string{
		"/dev/vda",
		"/dev/vdb",
		"/dev/vdc",
		"/dev/sr0",
		"/sys/block/vda",
		"/sys/block/vdb",
	}

	for _, device := range virtualDevices {
		if _, err := os.Stat(device); err == nil {
			return true
		}
	}

	return false
}

func (s *EnvironmentDetectionService) detectContainer(result *EnvironmentDetectionResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.containerChecked {
		if s.containerInfo != nil && s.containerInfo.IsContainer {
			s.applyContainerResults(result)
		}
		return
	}

	s.containerChecked = true
	s.containerInfo = s.gatherContainerInfo()

	if s.containerInfo != nil && s.containerInfo.IsContainer {
		s.applyContainerResults(result)
	}
}

func (s *EnvironmentDetectionService) gatherContainerInfo() *ContainerInfo {
	info := &ContainerInfo{}

	if _, err := os.Stat("/.dockerenv"); err == nil {
		info.IsContainer = true
		info.ContainerType = "docker"
	}

	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "containerd") {
			info.IsContainer = true
			if info.ContainerType == "" {
				info.ContainerType = "docker"
			}
			if matches := regexp.MustCompile(`([a-f0-9]{64})`).FindStringSubmatch(content); len(matches) > 0 {
				info.ContainerID = matches[1][:12]
			}
		}
		if strings.Contains(content, "kubepods") || strings.Contains(content, "kubernetes") {
			info.IsContainer = true
			info.ContainerType = "kubernetes"
		}
	}

	if data, err := os.ReadFile("/proc/self/mountinfo"); err == nil {
		content := string(data)
		if strings.Contains(content, "overlay") && !strings.Contains(content, "/var/lib/docker") {
			info.IsContainer = true
			if info.ContainerType == "" {
				info.ContainerType = "containerd"
			}
		}
	}

	envVars := []string{
		"DOCKER_CONTAINER",
		"DOCKER_IMAGE",
		"CONTAINER",
		"KUBERNETES_SERVICE_PORT",
	}

	for _, env := range envVars {
		if val := os.Getenv(env); val != "" {
			info.IsContainer = true
			if env == "KUBERNETES_SERVICE_PORT" {
				info.ContainerType = "kubernetes"
			} else if info.ContainerType == "" {
				info.ContainerType = "container"
			}
		}
	}

	if host, err := os.Hostname(); err == nil {
		hostnameLower := strings.ToLower(host)
		if regexp.MustCompile(`^[a-f0-9]{12}$`).MatchString(hostnameLower) {
			info.ContainerID = hostnameLower
			info.IsContainer = true
			if info.ContainerType == "" {
				info.ContainerType = "docker"
			}
		}
	}

	if data, err := os.ReadFile("/etc/hostname"); err == nil {
		hostnameStr := strings.TrimSpace(strings.ToLower(string(data)))
		if regexp.MustCompile(`^[a-f0-9]{12}$`).MatchString(hostnameStr) {
			info.ContainerID = hostnameStr
		}
	}

	return info
}

func (s *EnvironmentDetectionService) applyContainerResults(result *EnvironmentDetectionResult) {
	result.IsContainer = true
	result.RiskScore += 10
	result.Confidence += 0.5

	switch s.containerInfo.ContainerType {
	case "docker":
		result.VMType = VMTypeDocker
		result.Indicators = append(result.Indicators, "docker_detected")
		result.Reasons = append(result.Reasons, "Docker container environment detected")
	case "kubernetes":
		result.VMType = VMTypeKubernetes
		result.IsCloud = true
		result.CloudProvider = VMTypeKubernetes
		result.Indicators = append(result.Indicators, "kubernetes_detected")
		result.Reasons = append(result.Reasons, "Kubernetes pod environment detected")
	default:
		result.Indicators = append(result.Indicators, "container_detected")
		result.Reasons = append(result.Reasons, "Container environment detected")
	}

	if s.containerInfo.ContainerID != "" {
		result.Metadata["container_id"] = s.containerInfo.ContainerID
	}
	if s.containerInfo.ImageName != "" {
		result.Metadata["image_name"] = s.containerInfo.ImageName
	}
}

func (s *EnvironmentDetectionService) detectCloud(r *http.Request, result *EnvironmentDetectionResult) {
	headers := []string{
		"X-Azure-InstanceID",
		"X-Azure-SLBID",
		"X-Azure-RequestInfo",
		"X-Cloud-Trace-Context",
		"X-Gcloud-Resource-Priority",
		"X-Aws-Cloudfront-Is-Desktop-Viewer",
		"X-Aws-Cloudfront-Is-Mobile-Viewer",
		"X-Aliyun-Region",
		"X-Aliyun-Request-ID",
	}

	for _, header := range headers {
		if val := r.Header.Get(header); val != "" {
			result.IsCloud = true
			result.RiskScore += 15
			result.Confidence += 0.5

			if strings.Contains(header, "Azure") {
				result.CloudProvider = VMTypeAzure
				result.Indicators = append(result.Indicators, "azure_detected")
				result.Reasons = append(result.Reasons, "Microsoft Azure environment detected")
			} else if strings.Contains(header, "Cloud-Trace") || strings.Contains(header, "Gcloud") {
				result.CloudProvider = VMTypeGCP
				result.Indicators = append(result.Indicators, "gcp_detected")
				result.Reasons = append(result.Reasons, "Google Cloud Platform detected")
			} else if strings.Contains(header, "Aws") || strings.Contains(header, "Cloudfront") {
				result.CloudProvider = VMTypeAWS
				result.Indicators = append(result.Indicators, "aws_detected")
				result.Reasons = append(result.Reasons, "Amazon Web Services detected")
			} else if strings.Contains(header, "Aliyun") {
				result.CloudProvider = VMTypeAlibabaCloud
				result.Indicators = append(result.Indicators, "aliyun_detected")
				result.Reasons = append(result.Reasons, "Alibaba Cloud detected")
			}

			break
		}
	}

	clientIP := getClientIP(r)
	if ip := net.ParseIP(clientIP); ip != nil {
		for provider, cidrs := range cloudProviderCIDRs {
			for _, cidr := range cidrs {
				if cidr.Contains(ip) {
					result.IsCloud = true
					result.CloudProvider = provider
					result.RiskScore += 10
					result.Confidence += 0.4

					switch provider {
					case VMTypeAWS:
						result.Indicators = append(result.Indicators, "aws_ip_range")
						result.Reasons = append(result.Reasons, "IP address belongs to AWS IP range")
					case VMTypeAzure:
						result.Indicators = append(result.Indicators, "azure_ip_range")
						result.Reasons = append(result.Reasons, "IP address belongs to Azure IP range")
					case VMTypeGCP:
						result.Indicators = append(result.Indicators, "gcp_ip_range")
						result.Reasons = append(result.Reasons, "IP address belongs to GCP IP range")
					}
					return
				}
			}
		}
	}
}

func (s *EnvironmentDetectionService) detectEmulator(r *http.Request, result *EnvironmentDetectionResult) {
	userAgent := strings.ToLower(r.UserAgent())
	platform := r.Header.Get("Sec-Ch-Ua-Platform")
	platformVersion := r.Header.Get("Sec-Ch-Ua-Platform-Version")

	if platform != "" {
		platform = strings.ToLower(platform)
	}

	checkEmulatorPatterns := func(patterns []*regexp.Regexp, emulatorType EmulatorType) bool {
		for _, pattern := range patterns {
			if pattern.MatchString(userAgent) {
				result.IsEmulator = true
				result.EmulatorType = emulatorType
				result.RiskScore += 25
				result.Confidence += 0.5
				result.Indicators = append(result.Indicators, fmt.Sprintf("emulator_%s", emulatorType))
				result.Reasons = append(result.Reasons, fmt.Sprintf("%s emulator detected", emulatorType))
				return true
			}
		}
		return false
	}

	checkEmulatorPatterns(androidEmulatorPatterns, EmulatorTypeAndroid)
	checkEmulatorPatterns(iosEmulatorPatterns, EmulatorTypeIOS)
	checkEmulatorPatterns(bluestacksPatterns, EmulatorTypeBlueStacks)
	checkEmulatorPatterns(noxPatterns, EmulatorTypeNox)
	checkEmulatorPatterns(genymotionPatterns, EmulatorTypeGenymotion)

	if platform == "android" {
		androidUAIndicators := []string{
			"android sdk",
			"android emulator",
			"sdk_phone",
			"sdk_gphone",
			"goldfish",
			"generic",
		}
		for _, indicator := range androidUAIndicators {
			if strings.Contains(userAgent, indicator) {
				result.IsEmulator = true
				if result.EmulatorType == EmulatorTypeNone {
					result.EmulatorType = EmulatorTypeAndroid
				}
				result.RiskScore += 20
				result.Confidence += 0.4
				result.Indicators = append(result.Indicators, "android_emulator_ua")
				result.Reasons = append(result.Reasons, "Android emulator user agent detected")
				break
			}
		}
	}

	if platformVersion != "" && platform == "android" {
		if strings.Contains(platformVersion, ".") {
			parts := strings.Split(platformVersion, ".")
			if len(parts) >= 2 {
				major, minor := parts[0], parts[1]
				if major == "0" || minor == "0" {
					result.IsEmulator = true
					result.EmulatorType = EmulatorTypeAndroid
					result.RiskScore += 15
					result.Confidence += 0.3
					result.Indicators = append(result.Indicators, "emulator_platform_version")
					result.Reasons = append(result.Reasons, "Emulator-like platform version detected")
				}
			}
		}
	}

	if s.checkEmulatorFiles() {
		result.IsEmulator = true
		result.RiskScore += 20
		result.Confidence += 0.4
		result.Indicators = append(result.Indicators, "emulator_files")
		result.Reasons = append(result.Reasons, "Android emulator files detected on system")
	}
}

func (s *EnvironmentDetectionService) checkEmulatorFiles() bool {
	emulatorFiles := []string{
		"/usr/bin/emulator",
		"/opt/android-sdk/emulator",
		"/Users/Shared/Android/emulator",
		"/home/*/Android/Sdk/emulator",
		"/system/app/Superuser",
		"/system/xbin/su",
		"/system/bin/su",
	}

	for _, pattern := range emulatorFiles {
		if strings.Contains(pattern, "*") {
			continue
		}
		if _, err := os.Stat(pattern); err == nil {
			return true
		}
	}

	return false
}

func (s *EnvironmentDetectionService) detectAutomation(r *http.Request, additionalData map[string]string, result *EnvironmentDetectionResult) {
	userAgent := r.UserAgent()

	for automationType, patterns := range s.automationPatterns {
		for _, pattern := range patterns {
			if pattern.MatchString(userAgent) {
				result.IsAutomated = true
				result.AutomationType = automationType
				result.RiskScore += 30
				result.Confidence += 0.6
				result.Indicators = append(result.Indicators, fmt.Sprintf("automation_%s", automationType))
				result.Reasons = append(result.Reasons, fmt.Sprintf("%s automation detected in user agent", automationType))
			}
		}
	}

	for _, key := range []string{"webdriver", "selenium", "puppeteer", "playwright", "chrome_runtime"} {
		if val, ok := additionalData[key]; ok {
			valLower := strings.ToLower(val)
			if !strings.Contains(valLower, "no_") && !strings.Contains(valLower, "missing") && val != "" {
				result.IsAutomated = true
				result.RiskScore += 25
				result.Confidence += 0.5
				result.Indicators = append(result.Indicators, fmt.Sprintf("client_%s_detected", key))
				result.Reasons = append(result.Reasons, fmt.Sprintf("%s indicator detected in client data", key))
			}
		}
	}

	automationHeaders := map[string]string{
		"WebDriver":            "webdriver_header",
		"Selenium":             "selenium_header",
		"Chrome-Cdp-Version":   "chrome_cdp",
		"Playwright":          "playwright_header",
	}

	for header, indicator := range automationHeaders {
		if r.Header.Get(header) != "" {
			result.IsAutomated = true
			result.RiskScore += 35
			result.Confidence += 0.7
			result.Indicators = append(result.Indicators, indicator)
			result.Reasons = append(result.Reasons, fmt.Sprintf("Automation header %s detected", header))
		}
	}

	s.checkBrowserFingerprintAnomalies(additionalData, result)
}

func (s *EnvironmentDetectionService) checkBrowserFingerprintAnomalies(data map[string]string, result *EnvironmentDetectionResult) {
	if val, ok := data["navigator.webdriver"]; ok && val == "true" {
		result.IsAutomated = true
		result.RiskScore += 30
		result.Confidence += 0.6
		result.Indicators = append(result.Indicators, "navigator_webdriver_true")
		result.Reasons = append(result.Reasons, "navigator.webdriver is true")
	}

	if val, ok := data["permissions"]; ok {
		if strings.Contains(strings.ToLower(val), "denied") || strings.Contains(strings.ToLower(val), "prompt") {
			result.RiskScore += 10
			result.Indicators = append(result.Indicators, "permissions_denied")
			result.Reasons = append(result.Reasons, "Browser permissions appear restricted (common in automation)")
		}
	}

	if val, ok := data["plugins_count"]; ok {
		if val == "0" || val == "" {
			result.RiskScore += 15
			result.Indicators = append(result.Indicators, "no_plugins")
			result.Reasons = append(result.Reasons, "No browser plugins detected (unusual for regular browser)")
		}
	}

	if val, ok := data["webgl_renderer"]; ok {
		rendererLower := strings.ToLower(val)
		softwareRenderers := []string{"swiftshader", "llvmpipe", "mesa", "software", "virtual"}
		for _, sr := range softwareRenderers {
			if strings.Contains(rendererLower, sr) {
				result.RiskScore += 20
				result.Indicators = append(result.Indicators, "software_renderer")
				result.Reasons = append(result.Reasons, "Software WebGL renderer detected (common in VMs/headless)")
				break
			}
		}
	}

	if val, ok := data["languages"]; ok {
		if val == "" || val == "[]" {
			result.RiskScore += 10
			result.Indicators = append(result.Indicators, "no_languages")
			result.Reasons = append(result.Reasons, "No browser languages detected")
		}
	}
}

func (s *EnvironmentDetectionService) detectNetworkAnonymity(r *http.Request, result *EnvironmentDetectionResult) {
	s.detectProxy(r, result)
	s.detectVPN(r, result)
	s.detectTor(r, result)
	s.detectHostingProvider(r, result)
}

func (s *EnvironmentDetectionService) detectProxy(r *http.Request, result *EnvironmentDetectionResult) {
	proxyHeaders := []string{
		"X-Forwarded-For",
		"X-Real-IP",
		"X-ProxyUser-Ip",
		"X-Originating-IP",
		"X-Remote-IP",
		"X-Client-IP",
		"CF-Connecting-IP",
		"True-Client-IP",
		"X-Cluster-Client-IP",
	}

	proxyCount := 0
	var proxyChain []string

	for _, header := range proxyHeaders {
		if val := r.Header.Get(header); val != "" {
			proxyCount++
			proxyChain = append(proxyChain, fmt.Sprintf("%s:%s", header, val))
		}
	}

	if proxyCount > 0 {
		result.IsProxy = true
		result.RiskScore += float64(proxyCount) * 5
		result.Confidence += 0.3
		result.Indicators = append(result.Indicators, "proxy_headers_detected")
		result.Reasons = append(result.Reasons, fmt.Sprintf("%d proxy-related headers detected", proxyCount))
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 2 {
			result.RiskScore += 15
			result.Confidence += 0.4
			result.Indicators = append(result.Indicators, "multi_hop_proxy")
			result.Reasons = append(result.Reasons, "Multi-hop proxy chain detected (3+ proxies)")
		}
	}

	if via := r.Header.Get("Via"); via != "" {
		if regexp.MustCompile(`(?i)(proxy|vpn|tor|cache)`).MatchString(via) {
			result.IsProxy = true
			result.RiskScore += 20
			result.Confidence += 0.5
			result.Indicators = append(result.Indicators, "via_header_proxy")
			result.Reasons = append(result.Reasons, "Proxy indicator in Via header")
		}
	}
}

func (s *EnvironmentDetectionService) detectVPN(r *http.Request, result *EnvironmentDetectionResult) {
	clientIP := getClientIP(r)
	if ip := net.ParseIP(clientIP); ip != nil {
		for _, cidr := range s.knownVPNRanges {
			if cidr.Contains(ip) {
				result.IsVPN = true
				result.RiskScore += 25
				result.Confidence += 0.5
				result.Indicators = append(result.Indicators, "vpn_ip_range")
				result.Reasons = append(result.Reasons, "IP address belongs to known VPN provider range")
				return
			}
		}
	}

	if vpnHeaders := r.Header.Get("X-VPN"); vpnHeaders != "" {
		result.IsVPN = true
		result.RiskScore += 30
		result.Confidence += 0.6
		result.Indicators = append(result.Indicators, "vpn_header")
		result.Reasons = append(result.Reasons, "VPN header detected")
	}

	asNum := r.Header.Get("X-AS-Num")
	if asNum != "" {
		vpnASNs := []string{"AS9009", "AS12876", "AS206728", "AS212238"}
		for _, vpnAS := range vpnASNs {
			if strings.Contains(asNum, vpnAS) {
				result.IsVPN = true
				result.RiskScore += 25
				result.Confidence += 0.5
				result.Indicators = append(result.Indicators, "vpn_asn")
				result.Reasons = append(result.Reasons, "Traffic from known VPN ASN")
				return
			}
		}
	}
}

func (s *EnvironmentDetectionService) detectTor(r *http.Request, result *EnvironmentDetectionResult) {
	clientIP := getClientIP(r)

	s.mu.RLock()
	isTor, exists := s.torExitNodes[clientIP]
	s.mu.RUnlock()

	if exists && isTor {
		result.IsTor = true
		result.RiskScore += 40
		result.Confidence += 0.8
		result.Indicators = append(result.Indicators, "tor_exit_node")
		result.Reasons = append(result.Reasons, "IP address is a known Tor exit node")
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if torHeader := r.Header.Get("Tor-Circuit-Id"); torHeader != "" {
		result.IsTor = true
		result.RiskScore += 35
		result.Confidence += 0.6
		result.Indicators = append(result.Indicators, "tor_circuit_header")
		result.Reasons = append(result.Reasons, "Tor circuit header detected")
	}
}

func (s *EnvironmentDetectionService) detectHostingProvider(r *http.Request, result *EnvironmentDetectionResult) {
	clientIP := getClientIP(r)

	if ip := net.ParseIP(clientIP); ip != nil {
		for _, cidr := range privateIPRanges {
			if cidr.Contains(ip) {
				return
			}
		}

		isHosting := false
		if isHostingCIDR(ip) {
			isHosting = true
		}
		if isHosting {
			result.IsHosting = true
			result.RiskScore += 15
			result.Confidence += 0.4
			result.Indicators = append(result.Indicators, "hosting_provider_ip")
			result.Reasons = append(result.Reasons, "IP address belongs to known hosting provider range")
		}
	}
}

func isHostingCIDR(ip net.IP) bool {
	hostingCIDRs := []*net.IPNet{
		parseCIDR("45.33.0.0/16"),
		parseCIDR("104.238.0.0/16"),
		parseCIDR("107.170.0.0/16"),
		parseCIDR("159.89.0.0/16"),
		parseCIDR("167.99.0.0/16"),
		parseCIDR("188.166.0.0/16"),
		parseCIDR("198.199.0.0/16"),
		parseCIDR("206.189.0.0/16"),
		parseCIDR("209.141.0.0/16"),
		parseCIDR("217.79.0.0/16"),
		parseCIDR("5.2.0.0/16"),
		parseCIDR("185.220.0.0/16"),
	}

	for _, cidr := range hostingCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
}

func (s *EnvironmentDetectionService) checkIPReputation(ipAddress string, result *EnvironmentDetectionResult) {
	s.mu.RLock()
	if cached, exists := s.ipCache[ipAddress]; exists {
		if time.Since(cached.LastSeen) < s.ipCacheExpiry {
			s.applyIPReputation(cached, result)
			s.mu.RUnlock()
			return
		}
	}
	s.mu.RUnlock()

	reputationData := s.lookupIPReputation(ipAddress)

	s.mu.Lock()
	if len(s.ipCache) > s.maxIPCacheSize {
		s.cleanupIPCache()
	}
	s.ipCache[ipAddress] = reputationData
	s.mu.Unlock()

	s.applyIPReputation(reputationData, result)
}

func (s *EnvironmentDetectionService) lookupIPReputation(ipAddress string) *IPReputationData {
	data := &IPReputationData{
		IPAddress: ipAddress,
		LastSeen:  time.Now(),
	}

	ip := net.ParseIP(ipAddress)
	if ip == nil {
		data.RiskLevel = IPRiskLevelLow
		return data
	}

	for _, cidr := range privateIPRanges {
		if cidr.Contains(ip) {
			data.RiskLevel = IPRiskLevelLow
			data.RiskScore = 0
			return data
		}
	}

	s.mu.RLock()
	if s.knownMaliciousIPs[ipAddress] {
		data.RiskLevel = IPRiskLevelCritical
		data.RiskScore = 100
		data.IsKnownMalicious = true
		data.Reports = append(data.Reports, IPReport{
			Source:    "local_blacklist",
			Reason:    "IP in known malicious blacklist",
			Timestamp: time.Now(),
		})
	}
	s.mu.RUnlock()

	if s.isKnownVPNIP(ip) {
		data.RiskLevel = IPRiskLevelMedium
		data.RiskScore = 50
		data.IsVPN = true
	}

	if s.isKnownHostingIP(ip) {
		data.RiskLevel = IPRiskLevelMedium
		data.RiskScore = math.Max(data.RiskScore, 40)
		data.IsHosting = true
	}

	if s.isKnownCloudIP(ip) {
		data.RiskLevel = IPRiskLevelLow
		data.RiskScore = math.Max(data.RiskScore, 20)
		data.IsCloud = true
		data.IsDatacenter = true
	}

	s.mu.RLock()
	if s.torExitNodes[ipAddress] {
		data.RiskLevel = IPRiskLevelCritical
		data.RiskScore = 100
		data.IsTor = true
	}
	s.mu.RUnlock()

	if data.RiskScore == 0 {
		data.RiskLevel = IPRiskLevelLow
		data.RiskScore = 10
	}

	return data
}

func (s *EnvironmentDetectionService) applyIPReputation(data *IPReputationData, result *EnvironmentDetectionResult) {
	result.IPRiskLevel = data.RiskLevel
	result.IPRiskScore = data.RiskScore
	result.IPCountry = data.Country
	result.IPASN = data.ASN
	result.IsMaliciousIP = data.IsKnownMalicious

	switch data.RiskLevel {
	case IPRiskLevelCritical:
		result.RiskScore += 50
		result.Confidence += 0.8
		result.Indicators = append(result.Indicators, "critical_ip_reputation")
		result.Reasons = append(result.Reasons, "IP has critical reputation score")
	case IPRiskLevelHigh:
		result.RiskScore += 30
		result.Confidence += 0.5
		result.Indicators = append(result.Indicators, "high_ip_reputation")
		result.Reasons = append(result.Reasons, "IP has high risk reputation")
	case IPRiskLevelMedium:
		result.RiskScore += 15
		result.Confidence += 0.3
		result.Indicators = append(result.Indicators, "medium_ip_reputation")
		result.Reasons = append(result.Reasons, "IP has moderate risk factors")
	}

	if data.IsVPN {
		result.IsVPN = true
		result.Indicators = append(result.Indicators, "ip_vpn_detected")
	}
	if data.IsProxy {
		result.IsProxy = true
		result.Indicators = append(result.Indicators, "ip_proxy_detected")
	}
	if data.IsTor {
		result.IsTor = true
		result.Indicators = append(result.Indicators, "ip_tor_detected")
	}
}

func (s *EnvironmentDetectionService) isKnownVPNIP(ip net.IP) bool {
	for _, cidr := range s.knownVPNRanges {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func (s *EnvironmentDetectionService) isKnownHostingIP(ip net.IP) bool {
	hostingCIDRs := []*net.IPNet{
		parseCIDR("45.33.0.0/16"),
		parseCIDR("104.238.0.0/16"),
		parseCIDR("198.199.0.0/16"),
		parseCIDR("217.79.0.0/16"),
	}

	for _, cidr := range hostingCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func (s *EnvironmentDetectionService) isKnownCloudIP(ip net.IP) bool {
	for _, cidrs := range cloudProviderCIDRs {
		for _, cidr := range cidrs {
			if cidr.Contains(ip) {
				return true
			}
		}
	}
	return false
}

func (s *EnvironmentDetectionService) cleanupIPCache() {
	cutoff := time.Now().Add(-s.ipCacheExpiry)
	for ip, data := range s.ipCache {
		if data.LastSeen.Before(cutoff) {
			delete(s.ipCache, ip)
		}
	}
}

func (s *EnvironmentDetectionService) calculateOverallRisk(result *EnvironmentDetectionResult) {
	result.RiskScore = math.Min(math.Max(result.RiskScore, 0), 100)
	result.Confidence = math.Min(math.Max(result.Confidence, 0), 1)

	if result.IsVM {
		result.RiskScore += 15
	}
	if result.IsContainer {
		result.RiskScore += 10
	}
	if result.IsEmulator {
		result.RiskScore += 20
	}
	if result.IsAutomated {
		result.RiskScore += 30
	}
	if result.IsVPN {
		result.RiskScore += 15
	}
	if result.IsProxy {
		result.RiskScore += 10
	}
	if result.IsTor {
		result.RiskScore += 40
	}
	if result.IsMaliciousIP {
		result.RiskScore += 50
	}

	result.RiskScore = math.Min(result.RiskScore, 100)

	result.Reasons = uniqueStrings(result.Reasons)
	result.Indicators = uniqueStrings(result.Indicators)
}

func (s *EnvironmentDetectionService) AddMaliciousIP(ipAddress string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.knownMaliciousIPs[ipAddress] = true
}

func (s *EnvironmentDetectionService) RemoveMaliciousIP(ipAddress string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.knownMaliciousIPs, ipAddress)
}

func (s *EnvironmentDetectionService) AddTorExitNode(ipAddress string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.torExitNodes[ipAddress] = true
}

func (s *EnvironmentDetectionService) RemoveTorExitNode(ipAddress string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.torExitNodes, ipAddress)
}

func (s *EnvironmentDetectionService) GetIPReputation(ipAddress string) *IPReputationData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if data, exists := s.ipCache[ipAddress]; exists {
		return data
	}

	return s.lookupIPReputation(ipAddress)
}

func (s *EnvironmentDetectionService) UpdateIPCacheExpiry(expiry time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ipCacheExpiry = expiry
}

func getClientIP(r *http.Request) string {
	headersToCheck := []string{
		"X-Forwarded-For",
		"X-Real-IP",
		"CF-Connecting-IP",
		"True-Client-IP",
		"X-Client-IP",
	}

	for _, header := range headersToCheck {
		if ip := r.Header.Get(header); ip != "" {
			if strings.Contains(ip, ",") {
				ip = strings.Split(ip, ",")[0]
			}
			ip = strings.TrimSpace(ip)
			if parsedIP := net.ParseIP(ip); parsedIP != nil {
				return ip
			}
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return ip
	}

	return r.RemoteAddr
}

func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

func (s *EnvironmentDetectionService) DetectFromRequestData(ip string, userAgent string, headers map[string]string, clientData map[string]string) *EnvironmentDetectionResult {
	req := &http.Request{}
	req.Header = http.Header{}
	req.RemoteAddr = ip + ":0"

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	return s.Detect(req, clientData)
}

func (s *EnvironmentDetectionService) GetVMScore() float64 {
	return 25.0
}

func (s *EnvironmentDetectionService) GetAutomationScore() float64 {
	return 30.0
}

func (s *EnvironmentDetectionService) GetEmulatorScore() float64 {
	return 25.0
}

func (s *EnvironmentDetectionService) GetIPRiskScore() float64 {
	return 20.0
}

func (s *EnvironmentDetectionService) Serialize() ([]byte, error) {
	return json.Marshal(s)
}

func (s *EnvironmentDetectionService) Deserialize(data []byte) error {
	return json.Unmarshal(data, s)
}
