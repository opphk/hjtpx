package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type AdvancedEnvDetectorService struct {
	webglAnalyzer    *WebGLFingerprintAnalyzer
	torDetector      *TorNetworkDetector
	vmDetector       *VMMultiDimensionDetector
	riskScorer       *RiskScorer
	cache            map[string]*AdvancedEnvResult
	cacheMu          sync.RWMutex
	cacheExpiration  time.Duration
}

type WebGLFingerprintAnalyzer struct {
	softwareRenderers []string
	vmRenderers       []string
	knownBotPatterns  []string
}

type TorNetworkDetector struct {
	torExitPatterns []string
	blacklistCache  map[string]bool
	blacklistMu     sync.RWMutex
	httpClient      *http.Client
}

type VMMultiDimensionDetector struct {
	cpuPatterns   []string
	gpuPatterns   []string
	biosPatterns  []string
	registryKeys  []string
	processNames  []string
	memoryThresholds VMMemoryThresholds
}

type VMMemoryThresholds struct {
	MinVM  float64
	MaxVM  float64
	Low    float64
	High   float64
}

type RiskScorer struct {
	weights         map[string]float64
	categoryWeights map[string]float64
}

type AdvancedEnvResult struct {
	DetectionID    string                    `json:"detection_id"`
	Timestamp      time.Time                 `json:"timestamp"`
	RiskScore      float64                   `json:"risk_score"`
	RiskLevel      string                    `json:"risk_level"`
	IsBot          bool                      `json:"is_bot"`
	IsVPN          bool                      `json:"is_vpn"`
	IsProxy        bool                      `json:"is_proxy"`
	IsTor          bool                      `json:"is_tor"`
	IsVM           bool                      `json:"is_vm"`
	IsDarkWeb      bool                      `json:"is_dark_web"`
	Confidence     float64                   `json:"confidence"`
	Indicators     []string                  `json:"indicators"`
	WebGLAnalysis  *EnvWebGLAnalysisResult   `json:"webgl_analysis,omitempty"`
	TorAnalysis    *TorAnalysisResult        `json:"tor_analysis,omitempty"`
	VMAnalysis     *VMAnalysisResult         `json:"vm_analysis,omitempty"`
	Recommendations []string                 `json:"recommendations"`
}

type EnvWebGLAnalysisResult struct {
	IsSoftwareRenderer bool     `json:"is_software_renderer"`
	IsVMRenderer       bool     `json:"is_vm_renderer"`
	IsAnonymized       bool     `json:"is_anonymized"`
	RendererInfo       string   `json:"renderer_info"`
	VendorInfo         string   `json:"vendor_info"`
	MaxTextureSize     int      `json:"max_texture_size"`
	MaxRenderbufferSize int     `json:"max_renderbuffer_size"`
	ExtensionsCount    int      `json:"extensions_count"`
	Anomalies          []string `json:"anomalies"`
	Score              float64  `json:"score"`
}

type TorAnalysisResult struct {
	IsTorNode        bool     `json:"is_tor_node"`
	IsTorExitNode    bool     `json:"is_tor_exit_node"`
	IsDarkWebAccess  bool     `json:"is_dark_web_access"`
	TorVersion       string   `json:"tor_version"`
	ExitNodeCountry  string   `json:"exit_node_country"`
	ExitNodeISP      string   `json:"exit_node_isp"`
	ExitNodeASN      string   `json:"exit_node_asn"`
	Indicators       []string `json:"indicators"`
	Score            float64  `json:"score"`
}

type VMAnalysisResult struct {
	IsVM             bool     `json:"is_vm"`
	VMType           string   `json:"vm_type"`
	CPUDetected      bool     `json:"cpu_detected"`
	GPUDetected      bool     `json:"gpu_detected"`
	MemoryDetected   bool     `json:"memory_detected"`
	ProcessDetected  bool     `json:"process_detected"`
	BiosDetected     bool     `json:"bios_detected"`
	RegistryDetected bool     `json:"registry_detected"`
	CPUCores         int      `json:"cpu_cores"`
	MemoryGB         float64  `json:"memory_gb"`
	Indicators       []string `json:"indicators"`
	Score            float64  `json:"score"`
}

type AdvancedEnvDetectionRequest struct {
	DetectionID   string                 `json:"detection_id"`
	RiskScore     float64                `json:"risk_score"`
	RiskLevel     string                 `json:"risk_level"`
	AllDetections []string               `json:"all_detections"`
	Timestamp     int64                  `json:"timestamp"`
	ClientResults map[string]interface{} `json:"client_results"`
	Summary       *DetectionSummary      `json:"summary"`
	Fingerprint   string                 `json:"fingerprint"`
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
}

type DetectionSummary struct {
	TotalChecks      int                       `json:"total_checks"`
	HighRiskChecks   int                       `json:"high_risk_checks"`
	MediumRiskChecks int                       `json:"medium_risk_checks"`
	LowRiskChecks    int                       `json:"low_risk_checks"`
	Categories       map[string]CategoryResult `json:"categories"`
}

type CategoryResult struct {
	Score      float64  `json:"score"`
	Detections []string `json:"detections"`
}

type TorCheckResponse struct {
	Success       bool     `json:"success"`
	IsTor         bool     `json:"is_tor"`
	IsTorExitNode bool     `json:"is_tor_exit_node"`
	IPAddress     string   `json:"ip_address"`
	Country       string   `json:"country"`
	ISP           string   `json:"isp"`
	ASN           string   `json:"asn"`
	Hosting       bool     `json:"hosting"`
	Proxy         bool     `json:"proxy"`
	RiskLevel     string   `json:"risk_level"`
	Score         float64  `json:"score"`
	Confidence    float64  `json:"confidence"`
	Indicators    []string `json:"indicators"`
	CheckedAt     time.Time `json:"checked_at"`
}

func NewAdvancedEnvDetectorService() *AdvancedEnvDetectorService {
	return &AdvancedEnvDetectorService{
		webglAnalyzer:   NewWebGLFingerprintAnalyzer(),
		torDetector:     NewTorNetworkDetector(),
		vmDetector:      NewVMMultiDimensionDetector(),
		riskScorer:      NewRiskScorer(),
		cache:          make(map[string]*AdvancedEnvResult),
		cacheExpiration: 5 * time.Minute,
	}
}

func NewWebGLFingerprintAnalyzer() *WebGLFingerprintAnalyzer {
	return &WebGLFingerprintAnalyzer{
		softwareRenderers: []string{
			"swiftshader", "llvmpipe", "mesa", "software", "emulated",
			"virtual", "google swiftshader", "angle", "disabled",
		},
		vmRenderers: []string{
			"vmware", "virtualbox", "parallels", "qemu", "kvm", "hyperv",
			"xen", "bochs", "bhyve", "openvz", "lxc",
		},
		knownBotPatterns: []string{
			"headless", "phantom", "selenium", "puppeteer", "playwright",
			"automation", "bot", "crawler", "scraper",
		},
	}
}

func NewTorNetworkDetector() *TorNetworkDetector {
	return &TorNetworkDetector{
		torExitPatterns: []string{
			"torproject", "tornode", "torservers", "exitnode", "exitnode",
			"onion", "anonymizer", "darkweb",
		},
		blacklistCache: make(map[string]bool),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func NewVMMultiDimensionDetector() *VMMultiDimensionDetector {
	return &VMMultiDimensionDetector{
		cpuPatterns: []string{
			"virtual", "vmware", "virtualbox", "qemu", "kvm", "hyperv",
			"xen", "parallels",
		},
		gpuPatterns: []string{
			"vmware", "svga", "virtualbox", "vbox", "parallels",
			"qemu", "virtio", "hyper-v", "microsoft basic",
		},
		biosPatterns: []string{
			"virtualbox", "vmware", "parallels", "qemu", "kvm",
			"hyperv", "xen", "bochs", "seawolf",
		},
		registryKeys: []string{
			"vmware", "virtualbox", "vbox", "parallels",
			"qemu", "kvm", "hyperv",
		},
		processNames: []string{
			"vboxservice", "vboxtray", "vmtoolsd", "vmusrvc",
			"VMwareUser", "parallels", "qemu", "kvm", "xenhpet",
			"winxp", "virtio", "vboxclient",
		},
		memoryThresholds: VMMemoryThresholds{
			MinVM:  0.25,
			MaxVM:  8.0,
			Low:    0.5,
			High:   64.0,
		},
	}
}

func NewRiskScorer() *RiskScorer {
	return &RiskScorer{
		weights: map[string]float64{
			"webgl_anomaly":           15.0,
			"webgl_rendering":         12.0,
			"canvas_timing":           14.0,
			"canvas_entropy":          10.0,
			"plugin_count":            6.0,
			"plugin_types":            8.0,
			"vm_cpu":                  12.0,
			"vm_memory":               10.0,
			"vm_process":              14.0,
			"vm_gpu":                  15.0,
			"tor_detected":            20.0,
			"tor_exit_node":           18.0,
			"dark_web":                16.0,
		},
		categoryWeights: map[string]float64{
			"webgl":    1.2,
			"canvas":   1.1,
			"plugins":  0.8,
			"vm":       1.3,
			"tor":      1.5,
		},
	}
}

func (s *AdvancedEnvDetectorService) DetectEnvironment(ctx context.Context, req *AdvancedEnvDetectionRequest) (*AdvancedEnvResult, error) {
	result := &AdvancedEnvResult{
		DetectionID: req.DetectionID,
		Timestamp:  time.Now(),
	}

	if req.DetectionID == "" {
		result.DetectionID = fmt.Sprintf("adv_%d_%s", time.Now().Unix(), generateRandomID())
	}

	result.Indicators = append(result.Indicators, req.AllDetections...)

	webglAnalysis := s.analyzeWebGLFromClientResults(req.ClientResults)
	result.WebGLAnalysis = webglAnalysis
	if webglAnalysis != nil {
		result.Indicators = append(result.Indicators, webglAnalysis.Anomalies...)
	}

	torAnalysis, err := s.detectTorFromIP(ctx, req.IPAddress)
	if err == nil && torAnalysis != nil {
		result.TorAnalysis = torAnalysis
		result.Indicators = append(result.Indicators, torAnalysis.Indicators...)
	}

	vmAnalysis := s.detectVMFromResults(req.ClientResults)
	result.VMAnalysis = vmAnalysis
	if vmAnalysis != nil {
		result.Indicators = append(result.Indicators, vmAnalysis.Indicators...)
	}

	result.RiskScore = s.calculateRiskScore(req, webglAnalysis, torAnalysis, vmAnalysis)
	result.RiskLevel = s.getRiskLevel(result.RiskScore)
	result.Confidence = s.calculateConfidence(req, webglAnalysis, torAnalysis, vmAnalysis)

	s.determineFlags(result)

	result.Recommendations = s.generateRecommendations(result)

	s.cacheResult(result)

	return result, nil
}

func (s *AdvancedEnvDetectorService) analyzeWebGLFromClientResults(results map[string]interface{}) *EnvWebGLAnalysisResult {
	if results == nil {
		return nil
	}

	analysis := &EnvWebGLAnalysisResult{
		Anomalies: []string{},
	}

	if webglAnomaly, ok := results["webgl_anomaly"].(map[string]interface{}); ok {
		if detections, ok := webglAnomaly["detections"].([]interface{}); ok {
			for _, d := range detections {
				if det, ok := d.(string); ok {
					analysis.Anomalies = append(analysis.Anomalies, det)
					
					if strings.Contains(det, "software_renderer") {
						analysis.IsSoftwareRenderer = true
					}
					if strings.Contains(det, "vm_") {
						analysis.IsVMRenderer = true
					}
					if strings.Contains(det, "anonymized") {
						analysis.IsAnonymized = true
					}
				}
			}
		}
		if score, ok := webglAnomaly["score"].(float64); ok {
			analysis.Score = score
		}
	}

	if webglRendering, ok := results["webgl_rendering"].(map[string]interface{}); ok {
		if detections, ok := webglRendering["detections"].([]interface{}); ok {
			for _, d := range detections {
				if det, ok := d.(string); ok {
					analysis.Anomalies = append(analysis.Anomalies, det)
					
					if strings.Contains(det, "software") {
						analysis.IsSoftwareRenderer = true
					}
				}
			}
		}
	}

	if len(analysis.Anomalies) == 0 && analysis.Score == 0 {
		return nil
	}

	return analysis
}

func (s *AdvancedEnvDetectorService) detectTorFromIP(ctx context.Context, ip string) (*TorAnalysisResult, error) {
	result := &TorAnalysisResult{
		Indicators: []string{},
	}

	if ip == "" {
		return nil, fmt.Errorf("no IP address provided")
	}

	torAPIURL := fmt.Sprintf("https://ipinfo.io/%s/json", ip)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, torAPIURL, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := s.torDetector.httpClient.Do(req)
	if err != nil {
		return s.fallbackTorDetection(ip)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return s.fallbackTorDetection(ip)
	}

	var ipInfo struct {
		IP      string `json:"ip"`
		Org     string `json:"org"`
		ISP     string `json:"isp"`
		ASN     string `json:"asn"`
		Country string `json:"country"`
		Hosting bool   `json:"hosting"`
		Proxy   bool   `json:"proxy"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&ipInfo); err != nil {
		return s.fallbackTorDetection(ip)
	}

	orgLower := strings.ToLower(ipInfo.Org)
	ispLower := strings.ToLower(ipInfo.ISP)

	for _, pattern := range s.torDetector.torExitPatterns {
		if strings.Contains(orgLower, pattern) || strings.Contains(ispLower, pattern) {
			result.IsTorNode = true
			result.Indicators = append(result.Indicators, fmt.Sprintf("tor_pattern:%s", pattern))
		}
	}

	if ipInfo.Hosting {
		result.IsTorExitNode = true
		result.Indicators = append(result.Indicators, "hosting_detected")
	}

	if ipInfo.Proxy {
		result.Indicators = append(result.Indicators, "proxy_detected")
	}

	result.ExitNodeCountry = ipInfo.Country
	result.ExitNodeISP = ipInfo.ISP
	result.ExitNodeASN = ipInfo.ASN

	if result.IsTorNode {
		result.Score = 80.0
	} else if result.IsTorExitNode {
		result.Score = 70.0
	} else if len(result.Indicators) > 0 {
		result.Score = 40.0
	}

	return result, nil
}

func (s *AdvancedEnvDetectorService) fallbackTorDetection(ip string) (*TorAnalysisResult, error) {
	result := &TorAnalysisResult{
		Indicators: []string{},
	}

	result.Score = 0
	return result, nil
}

func (s *AdvancedEnvDetectorService) detectVMFromResults(results map[string]interface{}) *VMAnalysisResult {
	if results == nil {
		return nil
	}

	analysis := &VMAnalysisResult{
		Indicators: []string{},
	}

	vmCategories := []string{"vm_cpu", "vm_gpu", "vm_process", "vm_memory"}
	
	for _, category := range vmCategories {
		if catResult, ok := results[category].(map[string]interface{}); ok {
			if detections, ok := catResult["detections"].([]interface{}); ok {
				for _, d := range detections {
					if det, ok := d.(string); ok {
						analysis.Indicators = append(analysis.Indicators, det)
						
						switch category {
						case "vm_cpu":
							if strings.Contains(det, "cpu") || strings.Contains(det, "core") {
								analysis.CPUDetected = true
							}
						case "vm_gpu":
							if strings.Contains(det, "gpu") || strings.Contains(det, "vmware") || 
							   strings.Contains(det, "virtualbox") || strings.Contains(det, "software") {
								analysis.GPUDetected = true
							}
						case "vm_memory":
							if strings.Contains(det, "memory") {
								analysis.MemoryDetected = true
							}
						case "vm_process":
							if strings.Contains(det, "process") || strings.Contains(det, "vbox") || 
							   strings.Contains(det, "vmware") || strings.Contains(det, "qemu") {
								analysis.ProcessDetected = true
							}
						}
						
						if strings.Contains(det, "vmware") {
							analysis.VMType = "vmware"
						} else if strings.Contains(det, "virtualbox") || strings.Contains(det, "vbox") {
							analysis.VMType = "virtualbox"
						} else if strings.Contains(det, "qemu") || strings.Contains(det, "kvm") {
							analysis.VMType = "qemu/kvm"
						}
					}
				}
				
				if score, ok := catResult["score"].(float64); ok {
					analysis.Score += score
				}
			}
		}
	}

	detectedDimensions := 0
	if analysis.CPUDetected {
		detectedDimensions++
	}
	if analysis.GPUDetected {
		detectedDimensions++
	}
	if analysis.MemoryDetected {
		detectedDimensions++
	}
	if analysis.ProcessDetected {
		detectedDimensions++
	}

	if detectedDimensions >= 2 {
		analysis.IsVM = true
		analysis.Score = math.Min(analysis.Score, 100)
	} else if len(analysis.Indicators) > 0 {
		analysis.Score = math.Min(analysis.Score*0.7, 100)
	}

	if len(analysis.Indicators) == 0 && analysis.Score == 0 {
		return nil
	}

	return analysis
}

func (s *AdvancedEnvDetectorService) calculateRiskScore(req *AdvancedEnvDetectionRequest, webgl *EnvWebGLAnalysisResult, tor *TorAnalysisResult, vm *VMAnalysisResult) float64 {
	score := req.RiskScore * 0.3

	if webgl != nil {
		score += float64(len(webgl.Anomalies)) * s.riskScorer.categoryWeights["webgl"] * 0.2
	}

	if tor != nil {
		score += tor.Score * s.riskScorer.categoryWeights["tor"] * 0.25
		
		if tor.IsTorExitNode {
			score += 25
		}
	}

	if vm != nil {
		score += vm.Score * s.riskScorer.categoryWeights["vm"] * 0.25
		
		if vm.IsVM && vm.VMType != "" {
			score += 15
		}
	}

	if req.Summary != nil {
		if req.Summary.HighRiskChecks >= 3 {
			score += 20
		} else if req.Summary.HighRiskChecks >= 1 {
			score += 10
		}
	}

	return math.Round(math.Min(math.Max(score, 0), 100)*100) / 100
}

func (s *AdvancedEnvDetectorService) calculateConfidence(req *AdvancedEnvDetectionRequest, webgl *EnvWebGLAnalysisResult, tor *TorAnalysisResult, vm *VMAnalysisResult) float64 {
	confidence := 0.5

	if webgl != nil && len(webgl.Anomalies) > 0 {
		confidence += 0.15
	}

	if tor != nil {
		if tor.IsTorNode || tor.IsTorExitNode {
			confidence += 0.25
		} else if len(tor.Indicators) > 0 {
			confidence += 0.1
		}
	}

	if vm != nil {
		if vm.IsVM {
			confidence += 0.2
		} else if len(vm.Indicators) > 0 {
			confidence += 0.1
		}
	}

	if req.Summary != nil {
		totalChecks := float64(req.Summary.TotalChecks)
		if totalChecks > 0 {
			checksRatio := float64(req.Summary.HighRiskChecks+req.Summary.MediumRiskChecks) / totalChecks
			confidence += checksRatio * 0.1
		}
	}

	return math.Round(math.Min(confidence, 1.0)*100) / 100
}

func (s *AdvancedEnvDetectorService) determineFlags(result *AdvancedEnvResult) {
	result.IsBot = result.RiskScore > 70
	result.IsVPN = false
	result.IsProxy = false
	result.IsTor = result.TorAnalysis != nil && (result.TorAnalysis.IsTorNode || result.TorAnalysis.IsTorExitNode)
	result.IsVM = result.VMAnalysis != nil && result.VMAnalysis.IsVM
	result.IsDarkWeb = result.TorAnalysis != nil && result.TorAnalysis.IsDarkWebAccess

	if result.WebGLAnalysis != nil {
		if result.WebGLAnalysis.IsSoftwareRenderer {
			result.Indicators = append(result.Indicators, "software_rendering_detected")
		}
	}
}

func (s *AdvancedEnvDetectorService) getRiskLevel(score float64) string {
	if score >= 70 {
		return "high"
	}
	if score >= 40 {
		return "medium"
	}
	return "low"
}

func (s *AdvancedEnvDetectorService) generateRecommendations(result *AdvancedEnvResult) []string {
	recommendations := []string{}

	switch result.RiskLevel {
	case "high":
		recommendations = append(recommendations, "高风险环境，建议阻止访问或要求额外验证")
	case "medium":
		recommendations = append(recommendations, "中等风险，建议启用验证码或限制操作")
	default:
		recommendations = append(recommendations, "低风险环境，允许正常访问")
	}

	if result.IsTor {
		recommendations = append(recommendations, "检测到Tor网络连接，Tor常被用于绕过限制，请谨慎处理")
	}

	if result.IsVM {
		vmType := "未知虚拟机"
		if result.VMAnalysis != nil && result.VMAnalysis.VMType != "" {
			vmType = result.VMAnalysis.VMType
		}
		recommendations = append(recommendations, fmt.Sprintf("检测到%s环境，可能用于自动化操作", vmType))
	}

	if result.WebGLAnalysis != nil && result.WebGLAnalysis.IsSoftwareRenderer {
		recommendations = append(recommendations, "检测到软件渲染，浏览器可能被修改或处于虚拟环境")
	}

	if result.IsBot {
		recommendations = append(recommendations, "检测到自动化工具特征，建议阻止访问")
	}

	return recommendations
}

func (s *AdvancedEnvDetectorService) CheckTorNetwork(ctx context.Context, ip string) (*TorCheckResponse, error) {
	response := &TorCheckResponse{
		Success:    true,
		Indicators: []string{},
		CheckedAt:  time.Now(),
	}

	if ip == "" {
		response.Success = false
		response.RiskLevel = "unknown"
		return response, fmt.Errorf("no IP address provided")
	}

	response.IPAddress = ip

	s.torDetector.blacklistMu.RLock()
	if isKnown, ok := s.torDetector.blacklistCache[ip]; ok {
		s.torDetector.blacklistMu.RUnlock()
		response.IsTor = isKnown
		response.Score = 50
		response.RiskLevel = "cached"
		return response, nil
	}
	s.torDetector.blacklistMu.RUnlock()

	torAPIURL := fmt.Sprintf("https://check.torproject.org/api/ip")
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, torAPIURL, nil)
	if err != nil {
		return s.fallbackTorCheck(ctx, ip)
	}
	
	resp, err := s.torDetector.httpClient.Do(req)
	if err != nil {
		return s.fallbackTorCheck(ctx, ip)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var torResult struct {
			IsTor string `json:"IsTor"`
			IP    string `json:"IP"`
		}
		
		if err := json.NewDecoder(resp.Body).Decode(&torResult); err == nil {
			if torResult.IsTor == "Yes" || torResult.IsTor == "true" {
				response.IsTor = true
				response.Indicators = append(response.Indicators, "tor_project_api_confirmed")
			}
		}
	}

	ipInfoURL := fmt.Sprintf("https://ipinfo.io/%s/json", ip)
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, ipInfoURL, nil)
	if err == nil {
		resp2, err := s.torDetector.httpClient.Do(req2)
		if err == nil && resp2.StatusCode == http.StatusOK {
			defer resp2.Body.Close()
			
			var ipInfo struct {
				IP      string `json:"ip"`
				Org     string `json:"org"`
				ISP     string `json:"isp"`
				ASN     string `json:"asn"`
				Country string `json:"country"`
				Hosting bool   `json:"hosting"`
				Proxy   bool   `json:"proxy"`
			}
			
			if err := json.NewDecoder(resp2.Body).Decode(&ipInfo); err == nil {
				response.IPAddress = ipInfo.IP
				response.Country = ipInfo.Country
				response.ISP = ipInfo.ISP
				response.ASN = ipInfo.ASN
				response.Hosting = ipInfo.Hosting
				response.Proxy = ipInfo.Proxy
				
				if ipInfo.Hosting {
					response.IsTorExitNode = true
					response.IsTor = true
					response.Indicators = append(response.Indicators, "hosting_provider_tor")
				}
				
				orgLower := strings.ToLower(ipInfo.Org)
				ispLower := strings.ToLower(ipInfo.ISP)
				
				torPatterns := []string{"tor", "onion", "exitnode", "tornode", "anonymizer"}
				for _, pattern := range torPatterns {
					if strings.Contains(orgLower, pattern) || strings.Contains(ispLower, pattern) {
						response.IsTor = true
						response.Indicators = append(response.Indicators, fmt.Sprintf("tor_indicator:%s", pattern))
					}
				}
			}
		}
	}

	if response.IsTor {
		response.Score = 80.0
		response.Confidence = 0.9
		response.RiskLevel = "high"
		
		s.torDetector.blacklistMu.Lock()
		s.torDetector.blacklistCache[ip] = true
		s.torDetector.blacklistMu.Unlock()
	} else if response.Hosting || response.Proxy {
		response.Score = 40.0
		response.Confidence = 0.7
		response.RiskLevel = "medium"
	} else {
		response.Score = 10.0
		response.Confidence = 0.6
		response.RiskLevel = "low"
		
		s.torDetector.blacklistMu.Lock()
		s.torDetector.blacklistCache[ip] = false
		s.torDetector.blacklistMu.Unlock()
	}

	return response, nil
}

func (s *AdvancedEnvDetectorService) fallbackTorCheck(ctx context.Context, ip string) (*TorCheckResponse, error) {
	response := &TorCheckResponse{
		Success:    true,
		Indicators: []string{},
		CheckedAt:  time.Now(),
		IPAddress:  ip,
		Score:      20,
		Confidence: 0.5,
		RiskLevel:  "low",
	}

	return response, nil
}

func (s *AdvancedEnvDetectorService) cacheResult(result *AdvancedEnvResult) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cache[result.DetectionID] = result
}

func (s *AdvancedEnvDetectorService) GetCachedResult(detectionID string) (*AdvancedEnvResult, bool) {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	
	result, ok := s.cache[detectionID]
	return result, ok
}

func (s *AdvancedEnvDetectorService) CleanupExpiredCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	
	now := time.Now()
	for id, result := range s.cache {
		if now.Sub(result.Timestamp) > s.cacheExpiration {
			delete(s.cache, id)
		}
	}
}

func (s *WebGLFingerprintAnalyzer) AnalyzeRenderer(renderer, vendor string) *EnvWebGLAnalysisResult {
	result := &EnvWebGLAnalysisResult{
		RendererInfo: renderer,
		VendorInfo:   vendor,
		Anomalies:    []string{},
	}

	rendererLower := strings.ToLower(renderer)
	vendorLower := strings.ToLower(vendor)

	for _, sw := range s.softwareRenderers {
		if strings.Contains(rendererLower, sw) || strings.Contains(vendorLower, sw) {
			result.IsSoftwareRenderer = true
			result.Anomalies = append(result.Anomalies, fmt.Sprintf("software_renderer:%s", sw))
			result.Score += 35
		}
	}

	for _, vm := range s.vmRenderers {
		if strings.Contains(rendererLower, vm) || strings.Contains(vendorLower, vm) {
			result.IsVMRenderer = true
			result.Anomalies = append(result.Anomalies, fmt.Sprintf("vm_renderer:%s", vm))
			result.Score += 40
		}
	}

	if renderer == "unknown" || renderer == "" || vendor == "unknown" || vendor == "" {
		result.IsAnonymized = true
		result.Anomalies = append(result.Anomalies, "anonymized_renderer")
		result.Score += 20
	}

	if strings.Contains(rendererLower, "generic") || strings.Contains(rendererLower, "default") {
		result.IsAnonymized = true
		result.Anomalies = append(result.Anomalies, "generic_renderer")
		result.Score += 15
	}

	result.Score = math.Min(result.Score, 100)
	return result
}

func generateRandomID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, 8)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

func (s *VMMultiDimensionDetector) AnalyzeCPU(hardwareConcurrency int, userAgent string) (bool, float64, []string) {
	detected := false
	score := 0.0
	indicators := []string{}

	if hardwareConcurrency <= 0 {
		score += 10
		indicators = append(indicators, "cpu_count_unavailable")
	} else if hardwareConcurrency == 1 {
		detected = true
		score += 35
		indicators = append(indicators, "single_core_vm")
	} else if hardwareConcurrency == 2 {
		detected = true
		score += 20
		indicators = append(indicators, "dual_core_vm")
	} else if hardwareConcurrency >= 4 && hardwareConcurrency <= 8 {
		score += 15
		indicators = append(indicators, fmt.Sprintf("typical_vm_cpu:%d", hardwareConcurrency))
	}

	uaLower := strings.ToLower(userAgent)
	for _, pattern := range s.cpuPatterns {
		if strings.Contains(uaLower, pattern) {
			detected = true
			score += 40
			indicators = append(indicators, fmt.Sprintf("cpu_pattern:%s", pattern))
		}
	}

	return detected, math.Min(score, 100), indicators
}

func (s *VMMultiDimensionDetector) AnalyzeMemory(deviceMemory float64) (bool, float64, []string) {
	detected := false
	score := 0.0
	indicators := []string{}

	if deviceMemory <= 0 {
		score += 10
		indicators = append(indicators, "memory_unavailable")
	} else if deviceMemory <= s.memoryThresholds.MinVM {
		detected = true
		score += 35
		indicators = append(indicators, fmt.Sprintf("minimal_memory:%.2f", deviceMemory))
	} else if deviceMemory <= s.memoryThresholds.Low {
		detected = true
		score += 25
		indicators = append(indicators, fmt.Sprintf("low_memory:%.2f", deviceMemory))
	} else if deviceMemory >= s.memoryThresholds.High {
		score += 10
		indicators = append(indicators, fmt.Sprintf("unusually_high_memory:%.2f", deviceMemory))
	} else if deviceMemory >= s.memoryThresholds.MinVM && deviceMemory <= s.memoryThresholds.MaxVM {
		detected = true
		score += 15
		indicators = append(indicators, fmt.Sprintf("typical_vm_memory:%.2f", deviceMemory))
	}

	return detected, math.Min(score, 100), indicators
}

func (s *VMMultiDimensionDetector) AnalyzeGPU(renderer, vendor string) (bool, float64, []string) {
	detected := false
	score := 0.0
	indicators := []string{}

	rendererLower := strings.ToLower(renderer)
	vendorLower := strings.ToLower(vendor)

	for _, pattern := range s.gpuPatterns {
		if strings.Contains(rendererLower, pattern) || strings.Contains(vendorLower, pattern) {
			detected = true
			score += 45
			indicators = append(indicators, fmt.Sprintf("gpu_pattern:%s", pattern))
		}
	}

	softwareRenderers := []string{"swiftshader", "llvmpipe", "mesa", "software"}
	for _, sw := range softwareRenderers {
		if strings.Contains(rendererLower, sw) {
			detected = true
			score += 40
			indicators = append(indicators, fmt.Sprintf("software_rendering:%s", sw))
		}
	}

	return detected, math.Min(score, 100), indicators
}

func (s *VMMultiDimensionDetector) AnalyzeBiosAndRegistry(userAgent, platform string) (bool, float64, []string) {
	detected := false
	score := 0.0
	indicators := []string{}

	combined := strings.ToLower(userAgent + " " + platform)

	for _, pattern := range s.biosPatterns {
		if strings.Contains(combined, pattern) {
			detected = true
			score += 35
			indicators = append(indicators, fmt.Sprintf("bios_pattern:%s", pattern))
		}
	}

	for _, key := range s.registryKeys {
		if strings.Contains(combined, key) {
			detected = true
			score += 30
			indicators = append(indicators, fmt.Sprintf("registry_pattern:%s", key))
		}
	}

	return detected, math.Min(score, 100), indicators
}

func isTorIP(ip string) bool {
	torIPPatterns := []string{
		`^tor\d+\.`, `^tor\d*\.exit\.`, `^exit\d*\.tor`,
	}
	
	for _, pattern := range torIPPatterns {
		matched, _ := regexp.MatchString(pattern, ip)
		if matched {
			return true
		}
	}
	
	return false
}
