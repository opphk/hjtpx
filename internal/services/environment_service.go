package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type EnvironmentService struct {
	config        *EnvironmentConfig
	redisClient   *redis.Client
	httpClient    *http.Client
	geoIPProvider GeoIPProvider
	proxyChecker  *ProxyChecker
}

type EnvironmentConfig struct {
	EnableGeoIP       bool
	EnableProxyDetect bool
	EnableVPNDetect   bool
	CacheDuration     time.Duration
	MaxRequestTimeout time.Duration
	APIKeys           map[string]string
}

var DefaultEnvironmentConfig = &EnvironmentConfig{
	EnableGeoIP:       true,
	EnableProxyDetect:  true,
	EnableVPNDetect:    true,
	CacheDuration:      24 * time.Hour,
	MaxRequestTimeout:  5 * time.Second,
}

type EnvironmentResult struct {
	IPAddress      string              `json:"ip_address"`
	Geolocation    *GeolocationInfo    `json:"geolocation,omitempty"`
	NetworkInfo    *NetworkInfo        `json:"network_info,omitempty"`
	RiskAssessment *RiskAssessment     `json:"risk_assessment"`
	ThreatIndicators []ThreatIndicator `json:"threat_indicators,omitempty"`
	Confidence     float64             `json:"confidence"`
	AnalysisTime   time.Duration       `json:"analysis_time"`
}

type GeolocationInfo struct {
	Country      string  `json:"country"`
	CountryCode  string  `json:"country_code"`
	Region       string  `json:"region,omitempty"`
	RegionName   string  `json:"region_name,omitempty"`
	City         string  `json:"city,omitempty"`
	ZipCode      string  `json:"zip_code,omitempty"`
	Latitude     float64 `json:"latitude,omitempty"`
	Longitude    float64 `json:"longitude,omitempty"`
	Timezone     string  `json:"timezone,omitempty"`
	ISP          string  `json:"isp,omitempty"`
	Organization string  `json:"organization,omitempty"`
	ASNumber     string  `json:"as_number,omitempty"`
	IsMobile     bool    `json:"is_mobile,omitempty"`
	IsProxy      bool    `json:"is_proxy"`
	IsVPN        bool    `json:"is_vpn"`
	IsTor        bool    `json:"is_tor"`
	IsHosting     bool    `json:"is_hosting"`
}

type NetworkInfo struct {
	ConnectionType  string `json:"connection_type"`
	ISP             string `json:"isp"`
	ASN             string `json:"asn,omitempty"`
	Organization    string `json:"organization,omitempty"`
	IsDataCenter    bool   `json:"is_data_center"`
	IsProxy         bool   `json:"is_proxy"`
	IsVPN           bool   `json:"is_vpn"`
	IsTor           bool   `json:"is_tor"`
	IsMobileNetwork bool   `json:"is_mobile_network"`
	IsResidential    bool   `json:"is_residential"`
	NetworkRisk     float64 `json:"network_risk"`
}

type RiskAssessment struct {
	OverallRisk   RiskLevel `json:"overall_risk"`
	RiskScore     float64   `json:"risk_score"`
	RiskFactors   []RiskFactor `json:"risk_factors"`
	Recommendations []string `json:"recommendations"`
	IsSafe        bool      `json:"is_safe"`
	ThreatLevel   string    `json:"threat_level"`
}

type RiskFactor struct {
	Factor   string  `json:"factor"`
	Severity float64 `json:"severity"`
	Details  string  `json:"details"`
}

type ThreatIndicator struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Severity    float64 `json:"severity"`
	Detected    bool    `json:"detected"`
}

type GeoIPProvider interface {
	GetGeolocation(ctx context.Context, ip string) (*GeolocationInfo, error)
}

type IPAPIProvider struct {
	APIKey    string
	BaseURL   string
	UseHTTPS  bool
}

type ProxyChecker struct {
	mu sync.RWMutex
}

type ProxyCheckResult struct {
	IsProxy     bool    `json:"is_proxy"`
	IsVPN       bool    `json:"is_vpn"`
	IsTor       bool    `json:"is_tor"`
	IsHosting   bool    `json:"is_hosting"`
	Confidence  float64 `json:"confidence"`
	Services    []string `json:"services_checked"`
	ChecksPassed int     `json:"checks_passed"`
}

func NewEnvironmentService(redisClient *redis.Client, config *EnvironmentConfig) *EnvironmentService {
	if config == nil {
		config = DefaultEnvironmentConfig
	}

	service := &EnvironmentService{
		config:       config,
		redisClient:  redisClient,
		proxyChecker: NewProxyChecker(),
	}

	service.httpClient = &http.Client{
		Timeout: config.MaxRequestTimeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   3 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	if config.EnableGeoIP {
		service.geoIPProvider = NewIPAPIProvider(config.APIKeys["ipapi"])
	}

	return service
}

func (s *EnvironmentService) AnalyzeEnvironment(ctx context.Context, req *EnvironmentRequest) (*EnvironmentResult, error) {
	startTime := time.Now()

	result := &EnvironmentResult{
		ThreatIndicators: make([]ThreatIndicator, 0),
		RiskAssessment: &RiskAssessment{
			RiskFactors: make([]RiskFactor, 0),
			Recommendations: make([]string, 0),
		},
	}

	ip := s.extractIPAddress(req)
	result.IPAddress = ip

	if ip == "" || ip == "unknown" {
		result.RiskAssessment.OverallRisk = RiskLevelMedium
		result.RiskAssessment.RiskScore = 0.5
		result.Confidence = 0.3
		result.RiskAssessment.Recommendations = append(result.RiskAssessment.Recommendations, "Unable to determine IP address")
		return result, nil
	}

	cached, err := s.getCachedResult(ctx, ip)
	if err == nil && cached != nil {
		result.Geolocation = cached.Geolocation
		result.NetworkInfo = cached.NetworkInfo
		result.RiskAssessment = cached.RiskAssessment
		result.ThreatIndicators = cached.ThreatIndicators
		result.Confidence = cached.Confidence
		result.AnalysisTime = time.Since(startTime)
		return result, nil
	}

	if s.config.EnableGeoIP && s.geoIPProvider != nil {
		geoInfo, err := s.geoIPProvider.GetGeolocation(ctx, ip)
		if err == nil && geoInfo != nil {
			result.Geolocation = geoInfo

			result.ThreatIndicators = append(result.ThreatIndicators, ThreatIndicator{
				Type:        "proxy",
				Description: "Proxy/VPN detected",
				Severity:    0.7,
				Detected:   geoInfo.IsProxy,
			})

			result.ThreatIndicators = append(result.ThreatIndicators, ThreatIndicator{
				Type:        "vpn",
				Description: "VPN detected",
				Severity:    0.5,
				Detected:   geoInfo.IsVPN,
			})

			result.ThreatIndicators = append(result.ThreatIndicators, ThreatIndicator{
				Type:        "tor",
				Description: "Tor exit node detected",
				Severity:    0.8,
				Detected:   geoInfo.IsTor,
			})

			result.ThreatIndicators = append(result.ThreatIndicators, ThreatIndicator{
				Type:        "hosting",
				Description: "Hosting/Data center",
				Severity:    0.6,
				Detected:   geoInfo.IsHosting,
			})

			result.NetworkInfo = &NetworkInfo{
				ISP:          geoInfo.ISP,
				ASN:          geoInfo.ASNumber,
				Organization: geoInfo.Organization,
				IsProxy:      geoInfo.IsProxy,
				IsVPN:        geoInfo.IsVPN,
				IsTor:        geoInfo.IsTor,
				IsDataCenter: geoInfo.IsHosting,
				IsMobileNetwork: geoInfo.IsMobile,
			}
		}
	}

	if s.config.EnableProxyDetect {
		proxyCheck := s.checkProxy(ctx, ip, req)
		if proxyCheck.IsProxy || proxyCheck.IsVPN || proxyCheck.IsTor || proxyCheck.IsHosting {
			result.ThreatIndicators = append(result.ThreatIndicators, ThreatIndicator{
				Type:        "proxy_check",
				Description: fmt.Sprintf("Proxy check positive: %v", proxyCheck.Services),
				Severity:    0.8,
				Detected:    true,
			})
		}
	}

	result.RiskAssessment = s.assessRisk(result)

	if result.RiskAssessment.OverallRisk == RiskLevelCritical || result.RiskAssessment.OverallRisk == RiskLevelHigh {
		result.RiskAssessment.Recommendations = append(result.RiskAssessment.Recommendations, "Block or require additional verification")
	} else if result.RiskAssessment.OverallRisk == RiskLevelMedium {
		result.RiskAssessment.Recommendations = append(result.RiskAssessment.Recommendations, "Monitor closely and require additional verification")
	}

	result.Confidence = s.calculateConfidence(result)

	s.cacheResult(ctx, ip, result)

	result.AnalysisTime = time.Since(startTime)

	return result, nil
}

func (s *EnvironmentService) extractIPAddress(req *EnvironmentRequest) string {
	if req.IPAddress != "" && req.IPAddress != "unknown" {
		return req.IPAddress
	}

	if req.ForwardedFor != "" {
		ips := strings.Split(req.ForwardedFor, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if s.isValidIP(ip) {
				return ip
			}
		}
	}

	if req.RealIP != "" && s.isValidIP(req.RealIP) {
		return req.RealIP
	}

	if req.Headers != nil {
		headers := []string{
			"X-Forwarded-For",
			"X-Real-IP",
			"CF-Connecting-IP",
			"True-Client-IP",
			"X-Cluster-Client-IP",
		}

		for _, header := range headers {
			if value, ok := req.Headers[header]; ok {
				ip := strings.TrimSpace(strings.Split(value, ",")[0])
				if s.isValidIP(ip) {
					return ip
				}
			}
		}
	}

	return "unknown"
}

func (s *EnvironmentService) isValidIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil
}

func (s *EnvironmentService) checkProxy(ctx context.Context, ip string, req *EnvironmentRequest) *ProxyCheckResult {
	result := &ProxyCheckResult{
		Services: make([]string, 0),
	}

	if s.isPrivateIP(ip) {
		return result
	}

	portChecks := []int{80, 8080, 3128, 8888, 1080}
	for _, port := range portChecks {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), 2*time.Second)
		if err == nil {
			conn.Close()
			result.IsProxy = true
			result.Services = append(result.Services, fmt.Sprintf("open_port:%d", port))
			result.ChecksPassed++
		}
	}

	knownProxyPorts := map[int]string{
		1080: "SOCKS",
		3128: "HTTP Proxy",
		8080: "HTTP Proxy",
		8888: "HTTP Proxy",
		8118: "Privoxy",
		9050: "Tor",
	}

	if openPorts, _ := s.checkCommonPorts(ip); len(openPorts) > 3 {
		result.IsProxy = true
		result.Confidence = 0.7
		for _, port := range openPorts {
			if name, ok := knownProxyPorts[port]; ok {
				result.Services = append(result.Services, name)
			}
		}
	}

	return result
}

func (s *EnvironmentService) checkCommonPorts(ip string) ([]int, error) {
	ports := []int{21, 22, 23, 25, 80, 110, 143, 443, 993, 995, 3306, 3389, 5432, 6379, 8080, 8443}
	var openPorts []int

	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), 1*time.Second)
		if err == nil {
			openPorts = append(openPorts, port)
			conn.Close()
		}
	}

	return openPorts, nil
}

func (s *EnvironmentService) isPrivateIP(ip string) bool {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"localhost",
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(parsedIP) {
			return true
		}
	}

	return false
}

func (s *EnvironmentService) assessRisk(result *EnvironmentResult) *RiskAssessment {
	assessment := &RiskAssessment{
		RiskFactors:   make([]RiskFactor, 0),
		Recommendations: make([]string, 0),
	}

	riskScore := 0.0

	for _, indicator := range result.ThreatIndicators {
		if indicator.Detected {
			riskScore += indicator.Severity

			assessment.RiskFactors = append(assessment.RiskFactors, RiskFactor{
				Factor:   indicator.Type,
				Severity: indicator.Severity,
				Details:  indicator.Description,
			})
		}
	}

	if result.Geolocation != nil {
		highRiskCountries := map[string]bool{
			"CN": true, "RU": true, "KP": true, "IR": true, "BY": true,
			"NK": true, "SY": true, "CU": true, "VE": true,
		}

		if highRiskCountries[result.Geolocation.CountryCode] {
			riskScore += 0.1
			assessment.RiskFactors = append(assessment.RiskFactors, RiskFactor{
				Factor:   "high_risk_country",
				Severity: 0.1,
				Details:  fmt.Sprintf("IP from high-risk country: %s", result.Geolocation.Country),
			})
		}

		if result.Geolocation.IsProxy {
			riskScore += 0.3
		}

		if result.Geolocation.IsVPN {
			riskScore += 0.2
		}

		if result.Geolocation.IsTor {
			riskScore += 0.4
		}

		if result.Geolocation.IsHosting {
			riskScore += 0.2
		}
	}

	if result.NetworkInfo != nil {
		if result.NetworkInfo.IsDataCenter {
			riskScore += 0.2
			assessment.RiskFactors = append(assessment.RiskFactors, RiskFactor{
				Factor:   "data_center",
				Severity: 0.2,
				Details:  "Traffic from data center or hosting provider",
			})
		}

		if result.NetworkInfo.IsMobileNetwork {
			riskScore -= 0.1
		}

		if result.NetworkInfo.IsResidential {
			riskScore -= 0.1
		}
	}

	if riskScore > 1.0 {
		riskScore = 1.0
	}
	if riskScore < 0 {
		riskScore = 0
	}

	assessment.RiskScore = riskScore

	if riskScore >= 0.8 {
		assessment.OverallRisk = RiskLevelCritical
		assessment.ThreatLevel = "critical"
		assessment.IsSafe = false
	} else if riskScore >= 0.6 {
		assessment.OverallRisk = RiskLevelHigh
		assessment.ThreatLevel = "high"
		assessment.IsSafe = false
	} else if riskScore >= 0.4 {
		assessment.OverallRisk = RiskLevelMedium
		assessment.ThreatLevel = "medium"
		assessment.IsSafe = false
	} else if riskScore >= 0.2 {
		assessment.OverallRisk = RiskLevelLow
		assessment.ThreatLevel = "low"
		assessment.IsSafe = true
	} else {
		assessment.OverallRisk = RiskLevelSafe
		assessment.ThreatLevel = "safe"
		assessment.IsSafe = true
	}

	return assessment
}

func (s *EnvironmentService) calculateConfidence(result *EnvironmentResult) float64 {
	confidence := 0.3

	if result.Geolocation != nil {
		if result.Geolocation.Country != "" {
			confidence += 0.2
		}
		if result.Geolocation.City != "" {
			confidence += 0.1
		}
		if result.Geolocation.Latitude != 0 && result.Geolocation.Longitude != 0 {
			confidence += 0.1
		}
	}

	if result.NetworkInfo != nil {
		if result.NetworkInfo.ISP != "" {
			confidence += 0.1
		}
		if result.NetworkInfo.ASN != "" {
			confidence += 0.1
		}
	}

	detectedIndicators := 0
	for _, indicator := range result.ThreatIndicators {
		if indicator.Detected {
			detectedIndicators++
		}
	}

	if detectedIndicators > 0 {
		confidence += 0.1
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

func (s *EnvironmentService) getCachedResult(ctx context.Context, ip string) (*EnvironmentResult, error) {
	if s.redisClient == nil {
		return nil, errors.New("redis client not available")
	}

	key := fmt.Sprintf("env:analysis:%s", ip)
	data, err := s.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var result EnvironmentResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *EnvironmentService) cacheResult(ctx context.Context, ip string, result *EnvironmentResult) error {
	if s.redisClient == nil {
		return nil
	}

	key := fmt.Sprintf("env:analysis:%s", ip)
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return s.redisClient.Set(ctx, key, data, s.config.CacheDuration).Err()
}

func NewIPAPIProvider(apiKey string) *IPAPIProvider {
	provider := &IPAPIProvider{
		APIKey:   apiKey,
		UseHTTPS: true,
	}

	if apiKey != "" {
		provider.BaseURL = "https://ipapi.co"
	} else {
		provider.BaseURL = "http://ip-api.com"
	}

	return provider
}

func (p *IPAPIProvider) GetGeolocation(ctx context.Context, ip string) (*GeolocationInfo, error) {
	var url string

	if p.BaseURL == "https://ipapi.co" && p.APIKey != "" {
		url = fmt.Sprintf("%s/%s/json/?key=%s", p.BaseURL, ip, p.APIKey)
	} else if p.BaseURL == "http://ip-api.com" {
		url = fmt.Sprintf("%s/json/%s?fields=status,message,country,countryCode,region,regionName,city,zip,lat,lon,timezone,isp,org,as,mobile,proxy,vpn,tor,hosting", p.BaseURL, ip)
	} else {
		url = fmt.Sprintf("%s/%s/json/", p.BaseURL, ip)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; EnvironmentService/1.0)")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var geoInfo GeolocationInfo

	if strings.Contains(url, "ip-api.com") {
		var ipAPIResp struct {
			Status    string  `json:"status"`
			Country   string  `json:"country"`
			CountryCode string `json:"countryCode"`
			Region    string  `json:"region"`
			RegionName string `json:"regionName"`
			City      string  `json:"city"`
			Zip       string  `json:"zip"`
			Lat       float64 `json:"lat"`
			Lon       float64 `json:"lon"`
			Timezone  string  `json:"timezone"`
			ISP       string  `json:"isp"`
			Org       string  `json:"org"`
			AS        string  `json:"as"`
			Mobile    bool    `json:"mobile"`
			Proxy     bool    `json:"proxy"`
			VPN       bool    `json:"vpn"`
			Tor       bool    `json:"tor"`
			Hosting   bool    `json:"hosting"`
		}

		if err := json.Unmarshal(body, &ipAPIResp); err != nil {
			return nil, err
		}

		if ipAPIResp.Status != "success" {
			return nil, errors.New("IP geolocation failed")
		}

		geoInfo = GeolocationInfo{
			Country:      ipAPIResp.Country,
			CountryCode:  ipAPIResp.CountryCode,
			Region:       ipAPIResp.Region,
			RegionName:   ipAPIResp.RegionName,
			City:         ipAPIResp.City,
			ZipCode:      ipAPIResp.Zip,
			Latitude:     ipAPIResp.Lat,
			Longitude:    ipAPIResp.Lon,
			Timezone:     ipAPIResp.Timezone,
			ISP:          ipAPIResp.ISP,
			Organization: ipAPIResp.Org,
			ASNumber:     ipAPIResp.AS,
			IsMobile:     ipAPIResp.Mobile,
			IsProxy:      ipAPIResp.Proxy,
			IsVPN:        ipAPIResp.VPN,
			IsTor:        ipAPIResp.Tor,
			IsHosting:    ipAPIResp.Hosting,
		}
	} else {
		var ipapiResp struct {
			IP         string  `json:"ip"`
			City       string  `json:"city"`
			Region     string  `json:"region"`
			RegionCode string  `json:"region_code"`
			Country    string  `json:"country"`
			CountryCode string `json:"country_code"`
			Postal     string  `json:"postal"`
			Latitude   float64 `json:"latitude"`
			Longitude  float64 `json:"longitude"`
			Timezone   string  `json:"timezone"`
			ASN        string  `json:"asn"`
			ISP        string  `json:"isp"`
			Org        string  `json:"org"`
			Hosting    bool    `json:"hosting"`
			Proxy      bool    `json:"proxy"`
			Mobile     bool    `json:"mobile"`
		}

		if err := json.Unmarshal(body, &ipapiResp); err != nil {
			return nil, err
		}

		geoInfo = GeolocationInfo{
			Country:      ipapiResp.Country,
			CountryCode:  ipapiResp.CountryCode,
			Region:       ipapiResp.Region,
			RegionName:   ipapiResp.Region,
			City:         ipapiResp.City,
			ZipCode:      ipapiResp.Postal,
			Latitude:     ipapiResp.Latitude,
			Longitude:    ipapiResp.Longitude,
			Timezone:     ipapiResp.Timezone,
			ISP:          ipapiResp.ISP,
			Organization: ipapiResp.Org,
			ASNumber:     ipapiResp.ASN,
			IsMobile:     ipapiResp.Mobile,
			IsProxy:      ipapiResp.Proxy,
			IsHosting:    ipapiResp.Hosting,
		}
	}

	return &geoInfo, nil
}

func NewProxyChecker() *ProxyChecker {
	return &ProxyChecker{}
}

func (c *ProxyChecker) Check(ip string) *ProxyCheckResult {
	result := &ProxyCheckResult{
		Services: make([]string, 0),
	}

	return result
}

type EnvironmentRequest struct {
	IPAddress    string
	ForwardedFor string
	RealIP       string
	Headers      map[string]string
	UserAgent    string
	RequestURI   string
}

func (s *EnvironmentService) DetectBotByNetwork(ctx context.Context, req *EnvironmentRequest) (bool, string) {
	if req.IPAddress == "" {
		return false, ""
	}

	if s.isPrivateIP(req.IPAddress) {
		return false, ""
	}

	indicators := []struct {
		pattern *regexp.Regexp
		botType string
	}{
		{regexp.MustCompile(`(?i)(Googlebot|bingbot|YandexBot|Applebot|Twitterbot)`), "crawler"},
		{regexp.MustCompile(`(?i)(curl|wget|python|java|go-http|libwww)`), "script"},
		{regexp.MustCompile(`(?i)(phantomjs|selenium|webdriver)`), "automation"},
	}

	if req.UserAgent != "" {
		for _, indicator := range indicators {
			if indicator.pattern.MatchString(req.UserAgent) {
				return true, indicator.botType
			}
		}
	}

	if req.ForwardedFor != "" {
		forwardedIPs := strings.Split(req.ForwardedFor, ",")
		if len(forwardedIPs) > 3 {
			return true, "suspicious_forwarding"
		}
	}

	return false, ""
}

func (s *EnvironmentService) GetIPFromRequest(req *EnvironmentRequest) string {
	return s.extractIPAddress(req)
}
