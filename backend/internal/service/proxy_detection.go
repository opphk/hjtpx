package service

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type ProxyDetection struct {
	IPAddress        string            `json:"ip_address"`
	IsProxy          bool              `json:"is_proxy"`
	IsVPN            bool              `json:"is_vpn"`
	IsTor            bool              `json:"is_tor"`
	IsDatacenter     bool              `json:"is_datacenter"`
	Confidence       float64           `json:"confidence"`
	DetectionMethods []string          `json:"detection_methods"`
	RiskLevel        string            `json:"risk_level"`
	Country          string            `json:"country"`
	ISP              string            `json:"isp"`
	ASN              string            `json:"asn"`
	Hosting          bool              `json:"hosting"`
	Mobile           bool              `json:"mobile"`
	Score            float64           `json:"score"`
	LastChecked      time.Time         `json:"last_checked"`
	ResponseTime     time.Duration     `json:"response_time"`
	Headers          map[string]string `json:"headers"`
}

type IPInfo struct {
	IP          string  `json:"ip"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Region      string  `json:"region"`
	City        string  `json:"city"`
	ISP         string  `json:"isp"`
	ASN         string  `json:"asn"`
	Org         string  `json:"org"`
	Hosting     bool    `json:"hosting"`
	Mobile      bool    `json:"mobile"`
	Proxy       bool    `json:"proxy"`
	VPN         bool    `json:"vpn"`
	Tor         bool    `json:"tor"`
	Risk        float64 `json:"risk"`
}

type ProxyDatabase struct {
	knownProxies     map[string]*ProxyDetection
	knownVPNs        map[string]*ProxyDetection
	knownTor         map[string]bool
	datacenterRanges []string
	blacklist        map[string]time.Time
	mu               sync.RWMutex
}

type ConnectionAnalysis struct {
	Latency        time.Duration `json:"latency"`
	Jitter         float64       `json:"jitter"`
	PacketLoss     float64       `json:"packet_loss"`
	Bandwidth      float64       `json:"bandwidth"`
	IsProxyPattern bool          `json:"is_proxy_pattern"`
	IsVPNPattern   bool          `json:"is_vpn_pattern"`
	AnomalyScore   float64       `json:"anomaly_score"`
}

type ProxyDetectionService struct {
	database       *ProxyDatabase
	httpClient     *http.Client
	ipapiEndpoint  string
	ipdataEndpoint string
	mu             sync.RWMutex
}

func NewProxyDetectionService() *ProxyDetectionService {
	return &ProxyDetectionService{
		database:       NewProxyDatabase(),
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		ipapiEndpoint:  "http://ip-api.com/json",
		ipdataEndpoint: "https://api.ipdata.co",
	}
}

func NewProxyDatabase() *ProxyDatabase {
	return &ProxyDatabase{
		knownProxies: make(map[string]*ProxyDetection),
		knownVPNs:    make(map[string]*ProxyDetection),
		knownTor:     make(map[string]bool),
		datacenterRanges: []string{
			"3.", "4.", "8.", "13.", "15.", "16.", "17.", "18.", "20.",
			"23.", "34.", "35.", "40.", "44.", "45.", "47.", "48.", "49.",
			"50.", "52.", "54.", "63.", "64.", "65.", "66.", "67.", "68.",
			"69.", "70.", "71.", "72.", "73.", "74.", "75.", "76.", "77.",
			"78.", "79.", "80.", "81.", "82.", "83.", "84.", "85.", "86.",
			"87.", "88.", "89.", "90.", "91.", "92.", "93.", "94.", "95.",
			"96.", "97.", "98.", "99.", "104.", "108.", "130.", "131.",
			"136.", "142.", "143.", "144.", "146.", "147.", "148.", "149.",
			"150.", "151.", "157.", "158.", "159.", "160.", "161.", "162.",
			"163.", "164.", "165.", "166.", "167.", "168.", "169.", "170.",
			"171.", "172.", "173.", "174.", "175.", "176.", "177.", "178.",
			"179.", "180.", "181.", "182.", "183.", "184.", "185.", "186.",
			"187.", "188.", "189.", "190.", "191.", "192.", "193.", "194.",
			"195.", "196.", "197.", "198.", "199.", "200.", "204.", "207.",
			"208.", "209.", "210.", "211.", "212.", "213.", "214.", "215.",
			"216.", "217.", "218.", "219.", "220.", "221.", "222.", "223.",
			"224.", "225.", "226.", "227.", "228.", "229.", "230.", "231.",
			"232.", "233.", "234.", "235.", "236.", "237.", "238.", "239.",
		},
		blacklist: make(map[string]time.Time),
	}
}

func (s *ProxyDetectionService) DetectProxy(ip string, headers map[string]string) (*ProxyDetection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	startTime := time.Now()

	detection := &ProxyDetection{
		IPAddress:    ip,
		Headers:      headers,
		LastChecked:  time.Now(),
		ResponseTime: time.Since(startTime),
	}

	detectionMethods := []string{}

	xff := headers["X-Forwarded-For"]
	xri := headers["X-Real-IP"]
	via := headers["Via"]
	proxyChain := headers["X-ProxyChain"]
	forwarded := headers["Forwarded"]

	if xff != "" || xri != "" || via != "" {
		detectionMethods = append(detectionMethods, "proxy_header")
		detection.Score += 25
		detection.IsProxy = true
	}

	if via != "" {
		proxyKeywords := []string{"proxy", "squid", "nginx", "apache", "varnish", "traefik"}
		for _, keyword := range proxyKeywords {
			if strings.Contains(strings.ToLower(via), keyword) {
				detectionMethods = append(detectionMethods, "via_header_keyword")
				detection.Score += 15
				break
			}
		}
	}

	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 2 {
			detectionMethods = append(detectionMethods, "multi_hop_proxy")
			detection.Score += 20
			detection.IsProxy = true
		}
	}

	if forwarded != "" {
		detectionMethods = append(detectionMethods, "forwarded_header")
		detection.Score += 10
	}

	if proxyChain != "" {
		detectionMethods = append(detectionMethods, "proxy_chain_header")
		detection.Score += 25
		detection.IsProxy = true
	}

	if parsedIP := net.ParseIP(ip); parsedIP != nil {
		if s.isPrivateIP(ip) {
			detectionMethods = append(detectionMethods, "private_ip")
			detection.Score += 15
		}

		if s.isDatacenterIP(ip) {
			detectionMethods = append(detectionMethods, "datacenter_ip")
			detection.IsDatacenter = true
			detection.Score += 20
		}

		if s.isTorExitIP(ip) {
			detectionMethods = append(detectionMethods, "tor_exit_node")
			detection.IsTor = true
			detection.Score += 30
		}
	}

	info, err := s.lookupIPInfo(ip)
	if err == nil && info != nil {
		if info.Proxy {
			detectionMethods = append(detectionMethods, "ip_api_proxy")
			detection.IsProxy = true
			detection.Score += 35
		}
		if info.VPN {
			detectionMethods = append(detectionMethods, "ip_api_vpn")
			detection.IsVPN = true
			detection.Score += 30
		}
		if info.Tor {
			detectionMethods = append(detectionMethods, "ip_api_tor")
			detection.IsTor = true
			detection.Score += 30
		}
		if info.Hosting {
			detectionMethods = append(detectionMethods, "hosting_provider")
			detection.Hosting = true
			detection.Score += 15
		}
		if info.Mobile {
			detectionMethods = append(detectionMethods, "mobile_network")
			detection.Mobile = true
		}

		detection.Country = info.Country
		detection.ISP = info.ISP
		detection.ASN = info.ASN
	}

	if detection.Score > 60 {
		detection.Confidence = 0.90
		detection.RiskLevel = "high"
	} else if detection.Score > 30 {
		detection.Confidence = 0.70
		detection.RiskLevel = "medium"
	} else if detection.Score > 10 {
		detection.Confidence = 0.50
		detection.RiskLevel = "low"
	} else {
		detection.Confidence = 0.10
		detection.RiskLevel = "minimal"
	}

	detection.DetectionMethods = detectionMethods
	detection.Score = math.Min(detection.Score, 100)
	detection.ResponseTime = time.Since(startTime)

	return detection, nil
}

func (s *ProxyDetectionService) lookupIPInfo(ip string) (*IPInfo, error) {
	url := fmt.Sprintf("%s/%s", s.ipapiEndpoint, ip)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ip-api returned status %d", resp.StatusCode)
	}

	var ipInfo struct {
		Status      string `json:"status"`
		Country     string `json:"country"`
		CountryCode string `json:"countryCode"`
		Region      string `json:"regionName"`
		City        string `json:"city"`
		ISP         string `json:"isp"`
		Org         string `json:"org"`
		AS          string `json:"as"`
		Proxy       bool   `json:"proxy"`
		Hosting     bool   `json:"hosting"`
		Mobile      bool   `json:"mobile"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ipInfo); err != nil {
		return nil, err
	}

	return &IPInfo{
		IP:          ip,
		Country:     ipInfo.Country,
		CountryCode: ipInfo.CountryCode,
		Region:      ipInfo.Region,
		City:        ipInfo.City,
		ISP:         ipInfo.ISP,
		ASN:         ipInfo.AS,
		Org:         ipInfo.Org,
		Hosting:     ipInfo.Hosting,
		Mobile:      ipInfo.Mobile,
		Proxy:       ipInfo.Proxy,
	}, nil
}

func (s *ProxyDetectionService) isPrivateIP(ip string) bool {
	privateRanges := []string{
		"10.", "172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.", "172.24.",
		"172.25.", "172.26.", "172.27.", "172.28.", "172.29.",
		"172.30.", "172.31.", "192.168.", "127.", "169.254.",
	}

	for _, prefix := range privateRanges {
		if strings.HasPrefix(ip, prefix) {
			return true
		}
	}

	if parsedIP := net.ParseIP(ip); parsedIP != nil {
		return parsedIP.IsPrivate()
	}

	return false
}

func (s *ProxyDetectionService) isDatacenterIP(ip string) bool {
	for _, prefix := range s.database.datacenterRanges {
		if strings.HasPrefix(ip, prefix) {
			return true
		}
	}

	ispPatterns := []string{
		"amazon", "aws", "digitalocean", "linode", "vultr",
		"ovh", "hetzner", "cloudflare", "google cloud", "azure",
		"microsoft", "alibaba", "tencent", "oracle cloud",
	}

	for _, pattern := range ispPatterns {
		if strings.Contains(strings.ToLower(ip), pattern) {
			return true
		}
	}

	return false
}

func (s *ProxyDetectionService) isTorExitIP(ip string) bool {
	s.database.mu.RLock()
	defer s.database.mu.RUnlock()

	if s.database.knownTor[ip] {
		return true
	}

	knownTorExitNodes := []string{
		"128.31.0.34", "128.93.34.5", "131.188.40.189",
		"154.35.22.1", "171.25.193.77", "176.10.99.200",
		"185.220.100.240", "185.220.101.1", "185.220.102.1",
		"192.42.113.102", "192.42.113.109", "199.249.230.1",
		"199.249.230.3", "199.249.230.6", "199.249.230.7",
		"23.129.64.1", "23.129.64.2", "23.129.64.3",
		"45.154.255.1", "45.154.255.2", "45.66.33.1",
		"51.15.43.205", "51.15.80.145", "51.15.80.33",
		"51.222.13.74", "51.77.135.89", "52.10.128.136",
	}

	for _, torIP := range knownTorExitNodes {
		if ip == torIP || strings.HasPrefix(ip, torIP[:strings.LastIndex(torIP, ".")+1]) {
			return true
		}
	}

	return false
}

func (s *ProxyDetectionService) AnalyzeConnection(measurements []time.Duration) *ConnectionAnalysis {
	analysis := &ConnectionAnalysis{}

	if len(measurements) == 0 {
		return analysis
	}

	var sum time.Duration
	for _, m := range measurements {
		sum += m
	}
	avgLatency := sum / time.Duration(len(measurements))
	analysis.Latency = avgLatency

	if len(measurements) > 1 {
		var jitterSum float64
		for i := 1; i < len(measurements); i++ {
			diff := measurements[i] - measurements[i-1]
			if diff < 0 {
				diff = -diff
			}
			jitterSum += float64(diff)
		}
		analysis.Jitter = jitterSum / float64(len(measurements)-1)
	}

	analysis.IsProxyPattern = analysis.Latency > 200*time.Millisecond && analysis.Jitter > 50
	analysis.IsVPNPattern = analysis.Latency > 100*time.Millisecond && analysis.Jitter > 30

	if analysis.IsProxyPattern {
		analysis.AnomalyScore += 40
	}
	if analysis.IsVPNPattern {
		analysis.AnomalyScore += 30
	}
	if analysis.Latency > 500*time.Millisecond {
		analysis.AnomalyScore += 20
	}
	if analysis.Jitter > 100 {
		analysis.AnomalyScore += 15
	}

	analysis.AnomalyScore = math.Min(analysis.AnomalyScore, 100)

	return analysis
}

func (s *ProxyDetectionService) CheckBlacklist(ip string) bool {
	s.database.mu.RLock()
	defer s.database.mu.RUnlock()

	if expiry, exists := s.database.blacklist[ip]; exists {
		if time.Now().Before(expiry) {
			return true
		}
	}

	return false
}

func (s *ProxyDetectionService) AddToBlacklist(ip string, duration time.Duration) {
	s.database.mu.Lock()
	defer s.database.mu.Unlock()

	s.database.blacklist[ip] = time.Now().Add(duration)
}

func (s *ProxyDetectionService) RemoveFromBlacklist(ip string) {
	s.database.mu.Lock()
	defer s.database.mu.Unlock()

	delete(s.database.blacklist, ip)
}

func (s *ProxyDetectionService) GetDatabase() *ProxyDatabase {
	return s.database
}

func (s *ProxyDetectionService) ClearExpiredBlacklist() int {
	s.database.mu.Lock()
	defer s.database.mu.Unlock()

	now := time.Now()
	removed := 0

	for ip, expiry := range s.database.blacklist {
		if now.After(expiry) {
			delete(s.database.blacklist, ip)
			removed++
		}
	}

	return removed
}

type RealtimeCheckRequest struct {
	IPAddress string            `json:"ip_address"`
	Headers   map[string]string `json:"headers"`
	UserAgent string            `json:"user_agent"`
}

type RealtimeCheckResponse struct {
	IPAddress       string              `json:"ip_address"`
	IsSuspicious    bool                `json:"is_suspicious"`
	RiskLevel       string              `json:"risk_level"`
	Score           float64             `json:"score"`
	Reasons         []string            `json:"reasons"`
	Indicators      []string            `json:"indicators"`
	Recommendations []string            `json:"recommendations"`
	ProxyResult     *ProxyDetection     `json:"proxy_detection"`
	Analysis        *ConnectionAnalysis `json:"connection_analysis"`
}

func (s *ProxyDetectionService) RealtimeCheck(req *RealtimeCheckRequest) (*RealtimeCheckResponse, error) {
	response := &RealtimeCheckResponse{
		IPAddress:       req.IPAddress,
		Reasons:         make([]string, 0),
		Indicators:      make([]string, 0),
		Recommendations: make([]string, 0),
	}

	proxyResult, err := s.DetectProxy(req.IPAddress, req.Headers)
	if err == nil && proxyResult != nil {
		response.ProxyResult = proxyResult
		response.Score += proxyResult.Score

		if proxyResult.IsProxy {
			response.IsSuspicious = true
			response.Reasons = append(response.Reasons, "代理服务器检测")
			response.Indicators = append(response.Indicators, "proxy_detected")
			response.Recommendations = append(response.Recommendations, "建议进一步验证用户身份")
		}

		if proxyResult.IsVPN {
			response.IsSuspicious = true
			response.Reasons = append(response.Reasons, "VPN连接检测")
			response.Indicators = append(response.Indicators, "vpn_detected")
			response.Recommendations = append(response.Recommendations, "VPN可能用于隐私保护，需结合其他指标判断")
		}

		if proxyResult.IsTor {
			response.IsSuspicious = true
			response.Reasons = append(response.Reasons, "Tor网络检测")
			response.Indicators = append(response.Indicators, "tor_detected")
			response.Recommendations = append(response.Recommendations, "Tor出口节点存在被滥用的风险")
		}

		if proxyResult.IsDatacenter {
			response.Reasons = append(response.Reasons, "数据中心IP")
			response.Indicators = append(response.Indicators, "datacenter_ip")
		}
	}

	if req.UserAgent != "" {
		uaLower := strings.ToLower(req.UserAgent)
		automationIndicators := []string{"headless", "phantom", "puppeteer", "playwright", "selenium", "webdriver"}

		for _, indicator := range automationIndicators {
			if strings.Contains(uaLower, indicator) {
				response.IsSuspicious = true
				response.Score += 25
				response.Reasons = append(response.Reasons, fmt.Sprintf("自动化工具标识: %s", indicator))
				response.Indicators = append(response.Indicators, "automation:"+indicator)
			}
		}

		vpnIndicators := []string{"vpn", "proxy", "tor"}
		for _, indicator := range vpnIndicators {
			if strings.Contains(uaLower, indicator) {
				response.Score += 15
				response.Reasons = append(response.Reasons, fmt.Sprintf("UserAgent包含%s标识", indicator))
				response.Indicators = append(response.Indicators, "ua:"+indicator)
			}
		}
	}

	xff := req.Headers["X-Forwarded-For"]
	xri := req.Headers["X-Real-IP"]
	via := req.Headers["Via"]

	if xff != "" && req.IPAddress != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 2 {
			response.IsSuspicious = true
			response.Score += 20
			response.Reasons = append(response.Reasons, "多层代理链检测")
			response.Indicators = append(response.Indicators, "multi_hop_proxy")
		}

		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != req.IPAddress && !s.isPrivateIP(ip) {
				response.Score += 15
				response.Reasons = append(response.Reasons, fmt.Sprintf("X-Forwarded-For包含外部IP: %s", ip))
				response.Indicators = append(response.Indicators, "xff_external_ip")
			}
		}
	}

	if xri != "" && xri != req.IPAddress {
		response.Score += 10
		response.Reasons = append(response.Reasons, "X-Real-IP与连接IP不匹配")
		response.Indicators = append(response.Indicators, "xri_mismatch")
	}

	if via != "" {
		proxyKeywords := []string{"proxy", "squid", "nginx", "varnish", "vpn"}
		for _, keyword := range proxyKeywords {
			if strings.Contains(strings.ToLower(via), keyword) {
				response.IsSuspicious = true
				response.Score += 25
				response.Reasons = append(response.Reasons, fmt.Sprintf("Via头检测到代理标识: %s", keyword))
				response.Indicators = append(response.Indicators, "via_keyword")
				break
			}
		}
	}

	if response.Score > 70 {
		response.RiskLevel = "high"
	} else if response.Score > 40 {
		response.RiskLevel = "medium"
	} else if response.Score > 20 {
		response.RiskLevel = "low"
	} else {
		response.RiskLevel = "minimal"
	}

	if response.Score > 60 && len(response.Recommendations) == 0 {
		response.Recommendations = append(response.Recommendations, "建议启用增强验证")
	}

	return response, nil
}

func (s *ProxyDetectionService) GetIPReputation(ip string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	detection, err := s.DetectProxy(ip, make(map[string]string))
	if err != nil {
		return nil, err
	}

	result["ip"] = ip
	result["is_proxy"] = detection.IsProxy
	result["is_vpn"] = detection.IsVPN
	result["is_tor"] = detection.IsTor
	result["is_datacenter"] = detection.IsDatacenter
	result["confidence"] = detection.Confidence
	result["risk_level"] = detection.RiskLevel
	result["score"] = detection.Score
	result["detection_methods"] = detection.DetectionMethods
	result["country"] = detection.Country
	result["isp"] = detection.ISP
	result["asn"] = detection.ASN

	return result, nil
}

func (s *ProxyDetectionService) BatchCheck(ips []string) (map[string]*ProxyDetection, error) {
	results := make(map[string]*ProxyDetection)

	for _, ip := range ips {
		detection, err := s.DetectProxy(ip, make(map[string]string))
		if err != nil {
			continue
		}
		results[ip] = detection
	}

	return results, nil
}

type VPNDetectionPattern struct {
	Name        string   `json:"name"`
	Patterns    []string `json:"patterns"`
	Weight      float64  `json:"weight"`
	Description string   `json:"description"`
}

func (s *ProxyDetectionService) GetVPNPatterns() []VPNDetectionPattern {
	return []VPNDetectionPattern{
		{
			Name:        "header_analysis",
			Patterns:    []string{"X-Forwarded-For", "X-Real-IP", "Via", "X-ProxyChain"},
			Weight:      0.3,
			Description: "分析HTTP代理头部",
		},
		{
			Name:        "ip_range_check",
			Patterns:    s.database.datacenterRanges,
			Weight:      0.25,
			Description: "检查IP是否属于已知数据中心范围",
		},
		{
			Name:        "tor_exit_node",
			Patterns:    []string{"tor exit node"},
			Weight:      0.35,
			Description: "检查IP是否为Tor出口节点",
		},
		{
			Name:        "isp_analysis",
			Patterns:    []string{"VPN", "Proxy", "Hosting", "Cloud"},
			Weight:      0.1,
			Description: "分析ISP类型",
		},
	}
}

var vpnHeaderRegex = regexp.MustCompile(`(?i)(proxy|vpn|tor|exitnode|anonymizer|squid|nginx)`)

func (s *ProxyDetectionService) ValidateHeaders(headers map[string]string) (bool, []string) {
	flagged := make([]string, 0)

	for key, value := range headers {
		if vpnHeaderRegex.MatchString(value) {
			flagged = append(flagged, fmt.Sprintf("%s: %s", key, value))
		}
	}

	return len(flagged) > 0, flagged
}

type EnhancedIPRiskAssessment struct {
	IPAddress         string       `json:"ip_address"`
	OverallRisk       float64      `json:"overall_risk"`
	RiskLevel         string       `json:"risk_level"`
	RiskFactors       []RiskFactor `json:"risk_factors"`
	Confidence        float64      `json:"confidence"`
	AssessmentMethods []string     `json:"assessment_methods"`
	LastAssessed      time.Time    `json:"last_assessed"`
}

type RiskFactor struct {
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Score       float64  `json:"score"`
	Evidence    []string `json:"evidence"`
	Severity    string   `json:"severity"`
}

type EnhancedProxyDetectionService struct {
	*ProxyDetectionService
	ipRiskCache        map[string]*EnhancedIPRiskAssessment
	knownVPNProviders  map[string]*VPNProviderInfo
	knownCDNProviders  map[string]*CDNProviderInfo
	threatIntelligence *ThreatIntelligence
	assessmentMethods  []string
	riskFactors        []RiskFactor
	overallRisk         float64
	riskLevel          string
	confidence         float64
}

type VPNProviderInfo struct {
	Name            string   `json:"name"`
	IPRanges        []string `json:"ip_ranges"`
	ASNPatterns     []string `json:"asn_patterns"`
	DetectionWeight float64  `json:"detection_weight"`
}

type CDNProviderInfo struct {
	Name            string   `json:"name"`
	IPRanges        []string `json:"ip_ranges"`
	HostingPatterns []string `json:"hosting_patterns"`
	IsDatacenter    bool     `json:"is_datacenter"`
}

type ThreatIntelligence struct {
	KnownMaliciousIPs map[string]bool
	KnownBotNets      map[string]bool
	LastUpdated       time.Time
}

func NewEnhancedProxyDetectionService() *EnhancedProxyDetectionService {
	service := &EnhancedProxyDetectionService{
		ProxyDetectionService: NewProxyDetectionService(),
		ipRiskCache:           make(map[string]*EnhancedIPRiskAssessment),
		knownVPNProviders:     make(map[string]*VPNProviderInfo),
		knownCDNProviders:     make(map[string]*CDNProviderInfo),
		threatIntelligence: &ThreatIntelligence{
			KnownMaliciousIPs: make(map[string]bool),
			KnownBotNets:      make(map[string]bool),
			LastUpdated:       time.Now(),
		},
	}

	service.initializeKnownProviders()
	return service
}

func (s *EnhancedProxyDetectionService) initializeKnownProviders() {
	s.knownVPNProviders["ExpressVPN"] = &VPNProviderInfo{
		Name:            "ExpressVPN",
		ASNPatterns:     []string{"AS400052", "AS400053", "AS400054"},
		IPRanges:        []string{},
		DetectionWeight: 0.9,
	}

	s.knownVPNProviders["NordVPN"] = &VPNProviderInfo{
		Name:            "NordVPN",
		ASNPatterns:     []string{"AS45090", "AS42366"},
		IPRanges:        []string{},
		DetectionWeight: 0.9,
	}

	s.knownVPNProviders["Surfshark"] = &VPNProviderInfo{
		Name:            "Surfshark",
		ASNPatterns:     []string{"AS400065", "AS400066"},
		IPRanges:        []string{},
		DetectionWeight: 0.85,
	}

	s.knownVPNProviders["PrivateInternetAccess"] = &VPNProviderInfo{
		Name:            "Private Internet Access",
		ASNPatterns:     []string{"AS393398", "AS393399"},
		IPRanges:        []string{},
		DetectionWeight: 0.9,
	}

	s.knownVPNProviders["ProtonVPN"] = &VPNProviderInfo{
		Name:            "ProtonVPN",
		ASNPatterns:     []string{"AS42385", "AS42386"},
		IPRanges:        []string{},
		DetectionWeight: 0.85,
	}

	s.knownCDNProviders["Cloudflare"] = &CDNProviderInfo{
		Name:            "Cloudflare",
		IPRanges:        []string{"104.16.0.0/12", "172.64.0.0/13", "198.41.128.0/17"},
		HostingPatterns: []string{"cloudflare", "cloudflare.com"},
		IsDatacenter:    true,
	}

	s.knownCDNProviders["Akamai"] = &CDNProviderInfo{
		Name:            "Akamai",
		IPRanges:        []string{"23.0.0.0/8", "104.64.0.0/10"},
		HostingPatterns: []string{"akamaitechnologies", "akamai.com"},
		IsDatacenter:    true,
	}

	s.knownCDNProviders["Fastly"] = &CDNProviderInfo{
		Name:            "Fastly",
		IPRanges:        []string{"151.101.0.0/16", "199.232.0.0/16"},
		HostingPatterns: []string{"fastly", "fastly.com"},
		IsDatacenter:    true,
	}

	s.knownCDNProviders["AWS_CloudFront"] = &CDNProviderInfo{
		Name:            "AWS CloudFront",
		IPRanges:        []string{"52.84.0.0/15", "54.230.0.0/16", "99.84.0.0/16"},
		HostingPatterns: []string{"amazon", "aws", "cloudfront"},
		IsDatacenter:    true,
	}
}

func (s *EnhancedProxyDetectionService) AssessIPRisk(ip string, headers map[string]string, additionalData map[string]interface{}) *EnhancedIPRiskAssessment {
	s.assessmentMethods = make([]string, 0)
	s.riskFactors = make([]RiskFactor, 0)

	assessment := &EnhancedIPRiskAssessment{
		IPAddress:         ip,
		RiskFactors:       make([]RiskFactor, 0),
		AssessmentMethods: make([]string, 0),
		LastAssessed:      time.Now(),
	}

	s.assessProxyRisk(ip, headers)
	s.assessVPNRisk(ip, headers)
	s.assessTorRisk(ip)
	s.assessCDNRisk(ip)
	s.assessHostingRisk(ip)
	s.assessThreatIntelligence(ip)
	s.assessBehavioralRisk(additionalData)
	s.calculateOverallRisk()

	assessment.AssessmentMethods = s.assessmentMethods
	assessment.RiskFactors = s.riskFactors

	return assessment
}

func (s *EnhancedProxyDetectionService) assessProxyRisk(ip string, headers map[string]string) {
	method := "proxy_assessment"
	s.assessmentMethods = append(s.assessmentMethods, method)

	detection, err := s.ProxyDetectionService.DetectProxy(ip, headers)
	if err != nil {
		return
	}

	if detection.IsProxy {
		riskFactor := RiskFactor{
			Category:    "proxy",
			Description: "代理服务器检测",
			Score:       0.85,
			Evidence:    detection.DetectionMethods,
			Severity:    "high",
		}
		s.riskFactors = append(s.riskFactors, riskFactor)
	}

	if len(headers) > 0 {
		for headerName := range headers {
			if strings.Contains(strings.ToLower(headerName), "forwarded") ||
				strings.Contains(strings.ToLower(headerName), "proxy") {
				riskFactor := RiskFactor{
					Category:    "proxy_header",
					Description: fmt.Sprintf("检测到代理相关头部: %s", headerName),
					Score:       0.6,
					Evidence:    []string{headerName + ": " + headers[headerName]},
					Severity:    "medium",
				}
				s.riskFactors = append(s.riskFactors, riskFactor)
			}
		}
	}
}

func (s *EnhancedProxyDetectionService) assessVPNRisk(ip string, headers map[string]string) {
	method := "vpn_assessment"
	s.assessmentMethods = append(s.assessmentMethods, method)

	detection, err := s.ProxyDetectionService.DetectProxy(ip, headers)
	if err != nil {
		return
	}

	if detection.IsVPN {
		riskFactor := RiskFactor{
			Category:    "vpn",
			Description: "VPN连接检测",
			Score:       0.75,
			Evidence:    []string{fmt.Sprintf("ISP: %s", detection.ISP)},
			Severity:    "high",
		}
		s.riskFactors = append(s.riskFactors, riskFactor)
	}

	for providerName, provider := range s.knownVPNProviders {
		for _, asnPattern := range provider.ASNPatterns {
			if strings.Contains(detection.ASN, asnPattern) {
				riskFactor := RiskFactor{
					Category:    "vpn_provider",
					Description: fmt.Sprintf("检测到已知VPN提供商: %s", providerName),
					Score:       provider.DetectionWeight,
					Evidence:    []string{fmt.Sprintf("ASN: %s", detection.ASN)},
					Severity:    "medium",
				}
				s.riskFactors = append(s.riskFactors, riskFactor)
				break
			}
		}
	}
}

func (s *EnhancedProxyDetectionService) assessTorRisk(ip string) {
	method := "tor_assessment"
	s.assessmentMethods = append(s.assessmentMethods, method)

	isTor := s.isTorExitIP(ip)
	if isTor {
		riskFactor := RiskFactor{
			Category:    "tor",
			Description: "Tor出口节点检测",
			Score:       0.95,
			Evidence:    []string{"IP匹配已知Tor出口节点"},
			Severity:    "critical",
		}
		s.riskFactors = append(s.riskFactors, riskFactor)
	}

	torRelatedPatterns := []string{"tor", "onion", "torproject"}
	for _, pattern := range torRelatedPatterns {
		if strings.Contains(strings.ToLower(ip), pattern) {
			riskFactor := RiskFactor{
				Category:    "tor_related",
				Description: "IP地址与Tor网络相关",
				Score:       0.8,
				Evidence:    []string{fmt.Sprintf("模式匹配: %s", pattern)},
				Severity:    "high",
			}
			s.riskFactors = append(s.riskFactors, riskFactor)
			break
		}
	}
}

func (s *EnhancedProxyDetectionService) assessCDNRisk(ip string) {
	method := "cdn_assessment"
	s.assessmentMethods = append(s.assessmentMethods, method)

	isCDNIP := false
	cdnName := ""

	for providerName, provider := range s.knownCDNProviders {
		for _, ipRange := range provider.IPRanges {
			if strings.HasPrefix(ip, ipRange[:strings.LastIndex(ipRange, "/")]) {
				isCDNIP = true
				cdnName = providerName
				break
			}
		}
		if isCDNIP {
			break
		}
	}

	if isCDNIP {
		riskFactor := RiskFactor{
			Category:    "cdn_origin",
			Description: fmt.Sprintf("检测到CDN提供商IP地址: %s", cdnName),
			Score:       0.7,
			Evidence:    []string{fmt.Sprintf("IP: %s", ip)},
			Severity:    "medium",
		}
		s.riskFactors = append(s.riskFactors, riskFactor)
	}
}

func (s *EnhancedProxyDetectionService) assessHostingRisk(ip string) {
	method := "hosting_assessment"
	s.assessmentMethods = append(s.assessmentMethods, method)

	detection, _ := s.ProxyDetectionService.DetectProxy(ip, make(map[string]string))

	if detection.IsDatacenter {
		riskFactor := RiskFactor{
			Category:    "datacenter",
			Description: "数据中心IP地址检测",
			Score:       0.6,
			Evidence:    []string{fmt.Sprintf("ISP: %s", detection.ISP)},
			Severity:    "medium",
		}
		s.riskFactors = append(s.riskFactors, riskFactor)
	}

	if detection.Hosting {
		riskFactor := RiskFactor{
			Category:    "hosting_provider",
			Description: "托管服务提供商检测",
			Score:       0.55,
			Evidence:    []string{fmt.Sprintf("ISP: %s", detection.ISP)},
			Severity:    "low",
		}
		s.riskFactors = append(s.riskFactors, riskFactor)
	}
}

func (s *EnhancedProxyDetectionService) assessThreatIntelligence(ip string) {
	method := "threat_intelligence"
	s.assessmentMethods = append(s.assessmentMethods, method)

	if s.threatIntelligence.KnownMaliciousIPs[ip] {
		riskFactor := RiskFactor{
			Category:    "threat_intel",
			Description: "IP地址在已知恶意IP列表中",
			Score:       1.0,
			Evidence:    []string{"威胁情报匹配"},
			Severity:    "critical",
		}
		s.riskFactors = append(s.riskFactors, riskFactor)
	}

	if s.threatIntelligence.KnownBotNets[ip] {
		riskFactor := RiskFactor{
			Category:    "botnet",
			Description: "IP地址与已知僵尸网络相关",
			Score:       1.0,
			Evidence:    []string{"僵尸网络情报匹配"},
			Severity:    "critical",
		}
		s.riskFactors = append(s.riskFactors, riskFactor)
	}
}

func (s *EnhancedProxyDetectionService) assessBehavioralRisk(data map[string]interface{}) {
	if data == nil {
		return
	}

	method := "behavioral_assessment"
	s.assessmentMethods = append(s.assessmentMethods, method)

	if requestPattern, ok := data["request_pattern"].(string); ok {
		if strings.Contains(strings.ToLower(requestPattern), "automated") {
			riskFactor := RiskFactor{
				Category:    "behavior",
				Description: "检测到自动化请求模式",
				Score:       0.8,
				Evidence:    []string{"请求模式分析"},
				Severity:    "high",
			}
			s.riskFactors = append(s.riskFactors, riskFactor)
		}
	}

	if frequency, ok := data["request_frequency"].(float64); ok {
		if frequency > 100 {
			riskFactor := RiskFactor{
				Category:    "frequency",
				Description: "异常高频请求",
				Score:       0.7,
				Evidence:    []string{fmt.Sprintf("频率: %.2f req/min", frequency)},
				Severity:    "medium",
			}
			s.riskFactors = append(s.riskFactors, riskFactor)
		}
	}
}

func (s *EnhancedProxyDetectionService) calculateOverallRisk() {
	var totalScore float64
	var weightSum float64

	severityWeights := map[string]float64{
		"critical": 1.5,
		"high":     1.2,
		"medium":   1.0,
		"low":      0.7,
	}

	for _, factor := range s.riskFactors {
		weight := severityWeights[factor.Severity]
		totalScore += factor.Score * weight
		weightSum += weight
	}

	if weightSum > 0 {
		s.overallRisk = math.Min(totalScore/weightSum*100, 100)
	}

	if s.overallRisk >= 70 {
		s.riskLevel = "high"
		s.confidence = 0.85
	} else if s.overallRisk >= 40 {
		s.riskLevel = "medium"
		s.confidence = 0.75
	} else if s.overallRisk >= 20 {
		s.riskLevel = "low"
		s.confidence = 0.60
	} else {
		s.riskLevel = "minimal"
		s.confidence = 0.50
	}
}

func (s *EnhancedProxyDetectionService) GetCachedAssessment(ip string) (*EnhancedIPRiskAssessment, bool) {
	if assessment, exists := s.ipRiskCache[ip]; exists {
		if time.Since(assessment.LastAssessed) < 1*time.Hour {
			return assessment, true
		}
	}
	return nil, false
}

func (s *EnhancedProxyDetectionService) CacheAssessment(assessment *EnhancedIPRiskAssessment) {
	s.ipRiskCache[assessment.IPAddress] = assessment

	if len(s.ipRiskCache) > 10000 {
		s.cleanupRiskCache()
	}
}

func (s *EnhancedProxyDetectionService) cleanupRiskCache() {
	cutoff := time.Now().Add(-1 * time.Hour)
	for ip, assessment := range s.ipRiskCache {
		if assessment.LastAssessed.Before(cutoff) {
			delete(s.ipRiskCache, ip)
		}
	}
}

func (s *EnhancedProxyDetectionService) UpdateThreatIntelligence(maliciousIPs []string, botNets []string) {
	for _, ip := range maliciousIPs {
		s.threatIntelligence.KnownMaliciousIPs[ip] = true
	}

	for _, ip := range botNets {
		s.threatIntelligence.KnownBotNets[ip] = true
	}

	s.threatIntelligence.LastUpdated = time.Now()
}

func (s *EnhancedProxyDetectionService) DetectVPN(ip string, headers map[string]string) (bool, float64, []string) {
	method := "vpn_detection"
	s.assessmentMethods = append(s.assessmentMethods, method)

	isVPN := false
	confidence := 0.0
	evidence := make([]string, 0)

	detection, err := s.ProxyDetectionService.DetectProxy(ip, headers)
	if err == nil && detection.IsVPN {
		isVPN = true
		confidence = detection.Confidence
		evidence = append(evidence, fmt.Sprintf("VPN检测: ISP=%s, ASN=%s", detection.ISP, detection.ASN))
	}

	for providerName, provider := range s.knownVPNProviders {
		for _, asnPattern := range provider.ASNPatterns {
			if strings.Contains(detection.ASN, asnPattern) {
				isVPN = true
				confidence = math.Max(confidence, provider.DetectionWeight)
				evidence = append(evidence, fmt.Sprintf("VPN提供商: %s", providerName))
			}
		}
	}

	vpnHeaders := map[string]bool{
		"X-VPN-Connection": true,
		"X-VPN-Type":       true,
		"X-ProxyVPN":       true,
	}

	for headerName := range headers {
		if vpnHeaders[headerName] {
			isVPN = true
			confidence = math.Max(confidence, 0.95)
			evidence = append(evidence, fmt.Sprintf("VPN头部: %s", headerName))
		}
	}

	return isVPN, confidence, evidence
}

func (s *EnhancedProxyDetectionService) DetectTorNetwork(ip string) (bool, float64, []string) {
	isTor := s.isTorExitIP(ip)
	confidence := 0.0
	evidence := make([]string, 0)

	if isTor {
		confidence = 0.95
		evidence = append(evidence, "Tor出口节点匹配")
	}

	torIndicators := []string{"tor", "exit", "node"}
	for _, indicator := range torIndicators {
		if strings.Contains(strings.ToLower(ip), indicator) {
			isTor = true
			confidence = math.Max(confidence, 0.85)
			evidence = append(evidence, fmt.Sprintf("Tor指标: %s", indicator))
		}
	}

	return isTor, confidence, evidence
}

func (s *EnhancedProxyDetectionService) DetectCDNOrigin(ip string) (bool, string, float64) {
	for providerName, provider := range s.knownCDNProviders {
		for _, ipRange := range provider.IPRanges {
			prefix := ipRange[:strings.Index(ipRange, "/")]
			if strings.HasPrefix(ip, prefix) {
				return true, providerName, 0.9
			}
		}
	}

	return false, "", 0.0
}
