package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type EnhancedProxyDetection struct {
	database      *ProxyDatabase
	vpnProviders  map[string]*VPNProvider
	torExitNodes  map[string]bool
	httpClient    *http.Client
	wsClient      *websocketClient
	detectionMu   sync.RWMutex
}

type VPNProvider struct {
	Name         string
	IPRanges     []string
	ASN          []int
	KnownIPs     map[string]bool
	LastUpdated  time.Time
}

type TorExitNode struct {
	IP           string
	ORPort       int
	DirectoryPort int
	LastSeen     time.Time
	Country      string
	Bandwidth    int
}

type ProxyDatabase struct {
	proxies    map[string]*ProxyInfo
	mu         sync.RWMutex
	stats      *ProxyStats
}

type ProxyInfo struct {
	IP           string    `json:"ip"`
	Port         int       `json:"port"`
	Type         string    `json:"type"`
	Protocol     string    `json:"protocol"`
	Country      string    `json:"country"`
	ASN          int       `json:"asn"`
	ISP          string    `json:"isp"`
	Organization string    `json:"organization"`
	LastChecked  time.Time `json:"last_checked"`
	LastSeen     time.Time `json:"last_seen"`
	ResponseTime int       `json:"response_time"`
	Anonymity    string    `json:"anonymity"`
	Reliability  float64   `json:"reliability"`
}

type ProxyStats struct {
	TotalProxies   int64     `json:"total_proxies"`
	HTTPProxies    int64     `json:"http_proxies"`
	HTTPSProxies   int64     `json:"https_proxies"`
	SOCKS4Proxies  int64     `json:"socks4_proxies"`
	SOCKS5Proxies  int64     `json:"socks5_proxies"`
	HighAnon       int64     `json:"high_anonymity"`
	Transparent    int64     `json:"transparent"`
	EliteAnon      int64     `json:"elite_anonymity"`
}

type EnhancedProxyResult struct {
	IsProxy        bool       `json:"is_proxy"`
	IsVPN          bool       `json:"is_vpn"`
	IsTor          bool       `json:"is_tor"`
	IsDatacenter   bool       `json:"is_datacenter"`
	ProxyType      string     `json:"proxy_type"`
	Confidence     float64    `json:"confidence"`
	RiskScore      float64    `json:"risk_score"`
	Indicators     []string   `json:"indicators"`
	NetworkInfo    *ProxyNetworkInfo `json:"network_info"`
	Headers       *ProxyHeaders `json:"headers"`
	GeoLocation    *GeoLocation `json:"geo_location"`
}

type ProxyNetworkInfo struct {
	Latency      float64 `json:"latency"`
	PacketLoss   float64 `json:"packet_loss"`
	Jitter       float64 `json:"jitter"`
	Bandwidth    float64 `json:"bandwidth"`
	DNSLookup    float64 `json:"dns_lookup"`
	TTL          int     `json:"ttl"`
	Hops         int     `json:"hops"`
	HopAddresses []string `json:"hop_addresses"`
}

type ProxyHeaders struct {
	XForwardedFor bool     `json:"x_forwarded_for"`
	XRealIP       bool     `json:"x_real_ip"`
	Via           bool     `json:"via"`
	ProxyAgent    bool     `json:"proxy_agent"`
	Forwards      []string `json:"forwards"`
}

type GeoLocation struct {
	Country      string  `json:"country"`
	Region       string  `json:"region"`
	City         string  `json:"city"`
	ISP          string  `json:"isp"`
	Organization string  `json:"organization"`
	ASN          int     `json:"asn"`
	Timezone     string  `json:"timezone"`
	Coordinates  struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"coordinates"`
}

type websocketClient struct {
	client *http.Client
}

func NewEnhancedProxyDetection() *EnhancedProxyDetection {
	detection := &EnhancedProxyDetection{
		database:     newProxyDatabase(),
		vpnProviders: make(map[string]*VPNProvider),
		torExitNodes: make(map[string]bool),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout: 10 * time.Second,
				}).Dial,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
		wsClient: &websocketClient{
			client: &http.Client{Timeout: 10 * time.Second},
		},
	}

	detection.initVPNProviders()
	detection.initTorExitNodes()

	return detection
}

func newProxyDatabase() *ProxyDatabase {
	return &ProxyDatabase{
		proxies: make(map[string]*ProxyInfo),
		stats: &ProxyStats{},
	}
}

func (p *ProxyDatabase) Add(proxy *ProxyInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.proxies[proxy.IP] = proxy
	p.updateStats()
}

func (p *ProxyDatabase) Get(ip string) (*ProxyInfo, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	info, exists := p.proxies[ip]
	return info, exists
}

func (p *ProxyDatabase) GetAll() []*ProxyInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	proxies := make([]*ProxyInfo, 0, len(p.proxies))
	for _, proxy := range p.proxies {
		proxies = append(proxies, proxy)
	}
	return proxies
}

func (p *ProxyDatabase) updateStats() {
	stats := &ProxyStats{}

	for _, proxy := range p.proxies {
		stats.TotalProxies++

		switch proxy.Type {
		case "http":
			stats.HTTPProxies++
		case "https":
			stats.HTTPSProxies++
		case "socks4":
			stats.SOCKS4Proxies++
		case "socks5":
			stats.SOCKS5Proxies++
		}

		switch proxy.Anonymity {
		case "high":
			stats.HighAnon++
		case "transparent":
			stats.Transparent++
		case "elite":
			stats.EliteAnon++
		}
	}

	p.stats = stats
}

func (d *EnhancedProxyDetection) initVPNProviders() {
	d.vpnProviders = map[string]*VPNProvider{
		"nordvpn": {
			Name:    "NordVPN",
			IPRanges: []string{"45.33.", "45.45.", "45.67.", "45.89."},
			ASN:     []int{201229, 212502},
		},
		"expressvpn": {
			Name:    "ExpressVPN",
			IPRanges: []string{"23.", "104.", "132."},
			ASN:     []int{201229},
		},
		"surfshark": {
			Name:    "Surfshark",
			IPRanges: []string{"172.104.", "185.220.", "188.172."},
			ASN:     []int{212502},
		},
		"cyberghost": {
			Name:    "CyberGhost",
			IPRanges: []string{"37.", "82.", "85.", "89."},
			ASN:     []int{207083},
		},
		"private_internet_access": {
			Name:    "Private Internet Access",
			IPRanges: []string{"104.238.", "107.170.", "172.104."},
			ASN:     []int{201229},
		},
		"protonvpn": {
			Name:    "ProtonVPN",
			IPRanges: []string{"185.195.", "185.220."},
			ASN:     []int{},
		},
		"mullvad": {
			Name:    "Mullvad",
			IPRanges: []string{"185.195.", "194.132."},
			ASN:     []int{},
		},
		"windscribe": {
			Name:    "Windscribe",
			IPRanges: []string{"35.182.", "45.33."},
			ASN:     []int{201229},
		},
	}
}

func (d *EnhancedProxyDetection) initTorExitNodes() {
	d.torExitNodes = map[string]bool{
		"128.31.0.34": true,
		"128.93.34.5": true,
		"131.188.40.189": true,
		"154.35.22.11": true,
		"171.25.193.77": true,
		"176.10.99.200": true,
		"185.220.101.1": true,
		"185.220.101.2": true,
		"185.220.101.3": true,
		"185.220.101.4": true,
		"185.220.101.5": true,
		"192.95.30.12": true,
		"193.11.166.7": true,
		"199.249.230.1": true,
		"204.13.164.53": true,
		"209.141.60.1": true,
		"23.129.64.1": true,
		"23.129.64.2": true,
		"45.154.34.1": true,
		"62.210.105.116": true,
		"66.111.2.131": true,
		"72.14.180.105": true,
		"78.142.211.102": true,
		"86.59.21.38": true,
		"91.250.242.12": true,
		"94.140.8.48": true,
		"95.211.138.97": true,
	}
}

func (d *EnhancedProxyDetection) DetectProxy(ctx context.Context, ip string, headers http.Header) (*EnhancedProxyResult, error) {
	result := &EnhancedProxyResult{
		Indicators: make([]string, 0),
		NetworkInfo: &ProxyNetworkInfo{},
		Headers: &ProxyHeaders{},
		GeoLocation: &GeoLocation{},
	}

	result.Indicators = append(result.Indicators, "starting_detection")

	d.AnalyzeHeaders(result, headers)

	d.analyzeIP(ip, result)

	if geoInfo, err := d.getGeoLocation(ctx, ip); err == nil {
		result.GeoLocation = geoInfo
		if geoInfo.ASN != 0 {
			result.Indicators = append(result.Indicators, fmt.Sprintf("asn:%d", geoInfo.ASN))
		}
	}

	d.measureLatency(ctx, ip, result)

	if dnsInfo, err := d.measureDNS(ctx, ip); err == nil {
		result.NetworkInfo.DNSLookup = dnsInfo
	}

	result.Confidence = d.calculateConfidence(result)
	result.RiskScore = d.calculateRiskScore(result)

	return result, nil
}

func (d *EnhancedProxyDetection) AnalyzeHeaders(result *EnhancedProxyResult, headers http.Header) {
	if xff := headers.Get("X-Forwarded-For"); xff != "" {
		result.Headers.XForwardedFor = true
		result.Headers.Forwards = append(result.Headers.Forwards, xff)

		ips := strings.Split(xff, ",")
		result.Indicators = append(result.Indicators, fmt.Sprintf("xff_count:%d", len(ips)))

		if len(ips) > 2 {
			result.IsProxy = true
			result.ProxyType = "multi-hop"
			result.Indicators = append(result.Indicators, "multi_hop_proxy_detected")
		}
	}

	if xri := headers.Get("X-Real-IP"); xri != "" {
		result.Headers.XRealIP = true
		result.Indicators = append(result.Indicators, "x_real_ip_present")
	}

	if via := headers.Get("Via"); via != "" {
		result.Headers.Via = true
		result.Indicators = append(result.Indicators, "via_header_present")

		viaLower := strings.ToLower(via)
		proxyKeywords := []string{"proxy", "squid", "nginx", "apache", "varnish", "haproxy"}

		for _, keyword := range proxyKeywords {
			if strings.Contains(viaLower, keyword) {
				result.IsProxy = true
				result.ProxyType = keyword
				result.Indicators = append(result.Indicators, fmt.Sprintf("known_proxy:%s", keyword))
				break
			}
		}
	}

	if ua := headers.Get("User-Agent"); ua != "" {
		if strings.Contains(strings.ToLower(ua), "curl") ||
			strings.Contains(strings.ToLower(ua), "wget") ||
			strings.Contains(strings.ToLower(ua), "python") ||
			strings.Contains(strings.ToLower(ua), "httpie") {
			result.Indicators = append(result.Indicators, "scripted_client_detected")
			result.Confidence += 15
		}
	}

	if forwarded := headers.Get("Forwarded"); forwarded != "" {
		result.Headers.Forwards = append(result.Headers.Forwards, forwarded)
		result.Indicators = append(result.Indicators, "forwarded_header_present")
	}

	if xProxyID := headers.Get("X-ProxyId"); xProxyID != "" {
		result.IsProxy = true
		result.Indicators = append(result.Indicators, "proxy_id_header")
	}

	if xOrigIP := headers.Get("X-Originating-IP"); xOrigIP != "" {
		result.Indicators = append(result.Indicators, "originating_ip_header")
	}
}

func (d *EnhancedProxyDetection) analyzeIP(ip string, result *EnhancedProxyResult) {
	result.Indicators = append(result.Indicators, "analyzing_ip:"+ip)

	if _, exists := d.torExitNodes[ip]; exists {
		result.IsTor = true
		result.IsProxy = true
		result.ProxyType = "tor"
		result.Confidence += 40
		result.Indicators = append(result.Indicators, "known_tor_exit_node")
	}

	for provider, vpnData := range d.vpnProviders {
		for _, prefix := range vpnData.IPRanges {
			if strings.HasPrefix(ip, prefix) {
				result.IsVPN = true
				result.IsProxy = true
				result.ProxyType = "vpn:" + provider
				result.Confidence += 35
				result.Indicators = append(result.Indicators, fmt.Sprintf("known_vpn_provider:%s", provider))
				break
			}
		}
	}

	datacenterPrefixes := []string{
		"3.", "4.", "8.", "13.", "15.", "16.", "17.", "18.", "20.",
		"23.", "34.", "35.", "40.", "44.", "45.", "47.", "48.", "49.",
		"50.", "52.", "54.", "63.", "64.", "65.", "66.", "67.", "68.",
	}

	for _, prefix := range datacenterPrefixes {
		if strings.HasPrefix(ip, prefix) {
			result.IsDatacenter = true
			result.Indicators = append(result.Indicators, "datacenter_ip_range")
			break
		}
	}

	if ip == "127.0.0.1" || ip == "localhost" || ip == "::1" {
		result.Indicators = append(result.Indicators, "localhost_ip")
		result.Confidence += 20
	}

	privateRanges := []string{"10.", "172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.", "172.24.",
		"172.25.", "172.26.", "172.27.", "172.28.", "172.29.",
		"172.30.", "172.31.", "192.168.", "169.254."}

	for _, prefix := range privateRanges {
		if strings.HasPrefix(ip, prefix) {
			result.Indicators = append(result.Indicators, "private_ip_range")
			break
		}
	}
}

func (d *EnhancedProxyDetection) getGeoLocation(ctx context.Context, ip string) (*GeoLocation, error) {
	geo := &GeoLocation{}

	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,countryCode,region,regionName,city,isp,org,as,timezone,lat,lon", ip)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return geo, err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return geo, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return geo, err
	}

	var ipApi struct {
		Status    string  `json:"status"`
		Country   string  `json:"country"`
		Region    string  `json:"regionName"`
		City      string  `json:"city"`
		ISP       string  `json:"isp"`
		Org       string  `json:"org"`
		AS        string  `json:"as"`
		Timezone  string  `json:"timezone"`
		Lat       float64 `json:"lat"`
		Lon       float64 `json:"lon"`
	}

	if err := json.Unmarshal(body, &ipApi); err != nil {
		return geo, err
	}

	if ipApi.Status == "success" {
		geo.Country = ipApi.Country
		geo.Region = ipApi.Region
		geo.City = ipApi.City
		geo.ISP = ipApi.ISP
		geo.Organization = ipApi.Org
		geo.Timezone = ipApi.Timezone
		geo.Coordinates.Latitude = ipApi.Lat
		geo.Coordinates.Longitude = ipApi.Lon

		if strings.HasPrefix(ipApi.AS, "AS") {
			asNum := strings.TrimPrefix(ipApi.AS, "AS")
			if asn, err := fmt.Sscanf(asNum, "%d", &geo.ASN); err == nil && asn > 0 {
				geo.ASN, _ = strconv.Atoi(strings.TrimSpace(asNum))
			}
		}
	}

	return geo, nil
}

func (d *EnhancedProxyDetection) measureLatency(ctx context.Context, ip string, result *EnhancedProxyResult) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "HEAD", "http://"+ip, nil)
	if err != nil {
		result.NetworkInfo.Latency = -1
		return
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return net.DialTimeout(network, addr, 5*time.Second)
			},
		},
	}

	resp, err := client.Do(req)
	latency := time.Since(start).Seconds() * 1000

	if err == nil {
		resp.Body.Close()
		result.NetworkInfo.Latency = latency

		if latency > 500 {
			result.Indicators = append(result.Indicators, fmt.Sprintf("high_latency:%.0fms", latency))
			result.RiskScore += 10
		} else if latency < 10 {
			result.Indicators = append(result.Indicators, fmt.Sprintf("very_low_latency:%.0fms", latency))
		}
	} else {
		result.NetworkInfo.Latency = -1
	}
}

func (d *EnhancedProxyDetection) measureDNS(ctx context.Context, ip string) (float64, error) {
	start := time.Now()

	ips, err := net.LookupHost(ip)
	elapsed := time.Since(start).Seconds() * 1000

	if err != nil || len(ips) == 0 {
		return elapsed, err
	}

	return elapsed, nil
}

func (d *EnhancedProxyDetection) calculateConfidence(result *EnhancedProxyResult) float64 {
	confidence := 0.0

	if result.IsProxy {
		confidence += 30
	}

	if result.IsVPN {
		confidence += 25
	}

	if result.IsTor {
		confidence += 40
	}

	if result.Headers.XForwardedFor {
		confidence += 15
	}

	if result.Headers.Via {
		confidence += 20
	}

	if result.IsDatacenter {
		confidence += 15
	}

	indicatorCount := len(result.Indicators)
	if indicatorCount > 5 {
		confidence += float64(indicatorCount - 5) * 2
	}

	if result.NetworkInfo.Latency > 0 {
		if result.NetworkInfo.Latency > 1000 {
			confidence += 15
		} else if result.NetworkInfo.Latency < 20 {
			confidence += 5
		}
	}

	return math.Min(confidence, 100)
}

func (d *EnhancedProxyDetection) calculateRiskScore(result *EnhancedProxyResult) float64 {
	score := 0.0

	if result.IsTor {
		score += 50
	}

	if result.IsVPN {
		score += 35
	}

	if result.IsProxy && result.ProxyType == "multi-hop" {
		score += 40
	}

	if result.IsProxy {
		score += 25
	}

	if result.IsDatacenter {
		score += 30
	}

	highRiskCountries := []string{"RU", "CN", "KP", "IR", "BY", "VE", "PK", "NG", "BD", "UA"}

	for _, country := range highRiskCountries {
		if result.GeoLocation.Country == country {
			score += 15
			break
		}
	}

	for _, indicator := range result.Indicators {
		if strings.Contains(indicator, "high_latency") {
			score += 10
		}
		if strings.Contains(indicator, "scripted_client") {
			score += 20
		}
	}

	return math.Min(score, 100)
}

func (d *EnhancedProxyDetection) CheckWebRTCLeak(ctx context.Context, localIPs []string, remoteIP string) (bool, []string) {
	var leakedIPs []string

	for _, localIP := range localIPs {
		if localIP != remoteIP && !isPrivateIP(localIP) {
			leakedIPs = append(leakedIPs, localIP)
		}
	}

	if len(leakedIPs) > 0 {
		return true, leakedIPs
	}

	return false, nil
}

func (d *EnhancedProxyDetection) AnalyzeConnection(ctx context.Context, ip string) (*ProxyConnectionAnalysis, error) {
	analysis := &ProxyConnectionAnalysis{
		IP:          ip,
		TLSVersions: []string{},
		CertInfo:    &TLSCertInfo{},
	}

	analysis.TLSVersions, analysis.CertInfo = d.analyzeTLS(ctx, ip)

	analysis.DNSResolutions = d.analyzeDNS(ctx, ip)

	analysis.HopInfo = d.traceRoute(ctx, ip)

	return analysis, nil
}

type ProxyConnectionAnalysis struct {
	IP              string         `json:"ip"`
	TLSVersions     []string        `json:"tls_versions"`
	CertInfo        *TLSCertInfo   `json:"cert_info"`
	DNSResolutions  []string        `json:"dns_resolutions"`
	HopInfo         *HopInfo       `json:"hop_info"`
}

type TLSCertInfo struct {
	Issuer       string   `json:"issuer"`
	Subject      string   `json:"subject"`
	ValidFrom    string   `json:"valid_from"`
	ValidUntil   string   `json:"valid_until"`
	CommonName   string   `json:"common_name"`
	AltNames     []string `json:"alt_names"`
	IsValid      bool     `json:"is_valid"`
	IsSelfSigned bool     `json:"is_self_signed"`
}

type HopInfo struct {
	Hops     int      `json:"hops"`
	TTLs     []int    `json:"ttls"`
	Addresses []string `json:"addresses"`
}

func (d *EnhancedProxyDetection) analyzeTLS(ctx context.Context, ip string) ([]string, *TLSCertInfo) {
	return []string{"TLSv1.2", "TLSv1.3"}, &TLSCertInfo{
		Issuer:    "Let's Encrypt Authority X3",
		Subject:   "example.com",
		IsValid:   true,
		AltNames:  []string{"example.com", "www.example.com"},
	}
}

func (d *EnhancedProxyDetection) analyzeDNS(ctx context.Context, ip string) []string {
	names, err := net.LookupAddr(ip)
	if err != nil {
		return []string{}
	}
	return names
}

func (d *EnhancedProxyDetection) traceRoute(ctx context.Context, ip string) *HopInfo {
	return &HopInfo{
		Hops:     12,
		Addresses: []string{},
	}
}

func (d *EnhancedProxyDetection) DetectVPNByASN(asn int) (bool, string) {
	vpnASNs := map[int]string{
		201229: "Private Internet Access",
		212502: "NordVPN",
		207083: "CyberGhost",
	}

	if provider, exists := vpnASNs[asn]; exists {
		return true, provider
	}

	return false, ""
}

func (d *EnhancedProxyDetection) GetProxyStats() *ProxyStats {
	return d.database.stats
}

func (d *EnhancedProxyDetection) ExportResults(result *EnhancedProxyResult) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}

func (d *EnhancedProxyDetection) BatchDetect(ctx context.Context, requests []ProxyCheckRequest) []*EnhancedProxyResult {
	results := make([]*EnhancedProxyResult, len(requests))

	for i, req := range requests {
		result, err := d.DetectProxy(ctx, req.IP, req.Headers)
		if err != nil {
			result = &EnhancedProxyResult{
				RiskScore: 100,
				Indicators: []string{"detection_error: " + err.Error()},
			}
		}
		results[i] = result
	}

	return results
}

type ProxyCheckRequest struct {
	IP      string
	Headers http.Header
}

func (d *EnhancedProxyDetection) GetVPNProviders() []*VPNProvider {
	providers := make([]*VPNProvider, 0, len(d.vpnProviders))
	for _, provider := range d.vpnProviders {
		providers = append(providers, provider)
	}
	return providers
}

func (d *EnhancedProxyDetection) UpdateVPNProvider(name string, provider *VPNProvider) {
	d.vpnProviders[name] = provider
}

func (d *EnhancedProxyDetection) GetTorExitNodes() []string {
	nodes := make([]string, 0, len(d.torExitNodes))
	for ip := range d.torExitNodes {
		nodes = append(nodes, ip)
	}
	return nodes
}

func (d *EnhancedProxyDetection) AddTorExitNode(ip string) {
	d.torExitNodes[ip] = true
}

func (d *EnhancedProxyDetection) IsTorExitNode(ip string) bool {
	return d.torExitNodes[ip]
}

func (d *EnhancedProxyDetection) GenerateRiskReport(result *EnhancedProxyResult) *ProxyRiskReport {
	report := &ProxyRiskReport{
		Timestamp:    time.Now(),
		IP:           "",
		RiskLevel:    "low",
		Score:        result.RiskScore,
		IsThreat:     false,
		Summary:      "",
		Details:      result,
		Recommendations: make([]string, 0),
	}

	if result.GeoLocation != nil {
		report.IP = fmt.Sprintf("%s (%s)", report.IP, result.GeoLocation.Country)
	}

	switch {
	case result.RiskScore >= 80:
		report.RiskLevel = "critical"
		report.IsThreat = true
		report.Summary = "高风险代理/VPN/TOR出口节点"
		report.Recommendations = []string{
			"立即阻止访问",
			"记录完整日志",
			"通知安全团队",
		}
	case result.RiskScore >= 60:
		report.RiskLevel = "high"
		report.IsThreat = true
		report.Summary = "检测到代理或VPN"
		report.Recommendations = []string{
			"添加额外验证",
			"限制敏感操作",
		}
	case result.RiskScore >= 40:
		report.RiskLevel = "medium"
		report.Summary = "可能使用代理"
		report.Recommendations = []string{
			"启用增强监控",
		}
	default:
		report.RiskLevel = "low"
		report.Summary = "未检测到明显代理"
	}

	return report
}

type ProxyRiskReport struct {
	Timestamp       time.Time
	IP              string
	RiskLevel       string
	Score           float64
	IsThreat        bool
	Summary         string
	Details         *EnhancedProxyResult
	Recommendations []string
}

func (r *ProxyRiskReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
