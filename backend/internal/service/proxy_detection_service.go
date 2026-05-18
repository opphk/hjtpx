package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type ProxyDetection struct {
	IP                string    `json:"ip"`
	IPAddress         string    `json:"ip_address"`
	RiskLevel         string    `json:"risk_level"`
	IsProxy           bool      `json:"is_proxy"`
	IsVPN             bool      `json:"is_vpn"`
	IsTor             bool      `json:"is_tor"`
	IsHosting         bool      `json:"is_hosting"`
	IsDataCenter      bool      `json:"is_data_center"`
	IsMobile          bool      `json:"is_mobile"`
	Hosting           bool      `json:"hosting"`
	Mobile            bool      `json:"mobile"`
	DetectionMethods  []string  `json:"detection_methods"`
	ProxyType         string    `json:"proxy_type"`
	ISP               string    `json:"isp"`
	Country           string    `json:"country"`
	City              string    `json:"city"`
	ASN               string    `json:"asn"`
	RiskScore         float64   `json:"risk_score"`
	Score             float64   `json:"score"`
	Confidence        float64   `json:"confidence"`
	LastChecked       time.Time `json:"last_checked"`
}

type ProxyDetectionService struct {
	cache         map[string]*ProxyDetection
	httpClient    *http.Client
	vpnRanges     []string
	torExitNodes  []string
	datacenterIPs []string
	vpnPatterns   []*regexp.Regexp
	mu            sync.RWMutex
	maxCacheSize  int
}

func NewProxyDetectionService() *ProxyDetectionService {
	p := &ProxyDetectionService{
		cache: make(map[string]*ProxyDetection),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: 5 * time.Second,
				}).DialContext,
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
		vpnRanges: []string{
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
		},
		torExitNodes: []string{
			"185.220.100.240/28",
			"185.220.101.0/28",
		},
		maxCacheSize: 10000,
	}
	p.initDataCenterRanges()
	p.initVPNPatterns()
	return p
}

func (s *ProxyDetectionService) initDataCenterRanges() {
	s.datacenterIPs = []string{
		"104.16.0.0/12",
		"172.70.0.0/15",
		"172.64.0.0/13",
	}
}

func (s *ProxyDetectionService) initVPNPatterns() {
	patterns := []string{
		`^vpn[\-\.]?\d*\.`,
		`^vpn\.`,
		`proxy\.`,
		`tor\.`,
		`exit\.tor\.`,
	}
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			s.vpnPatterns = append(s.vpnPatterns, re)
		}
	}
}

func (s *ProxyDetectionService) DetectProxy(ip string, headers map[string]string) (*ProxyDetection, error) {
	ctx := context.Background()
	return s.ProxyDetection(ctx, ip, headers)
}

func (s *ProxyDetectionService) ProxyDetection(ctx context.Context, ip string, headers map[string]string) (*ProxyDetection, error) {
	s.mu.RLock()
	cached, ok := s.cache[ip]
	s.mu.RUnlock()
	if ok && cached != nil {
		return cached, nil
	}

	result := &ProxyDetection{
		IP:               ip,
		IPAddress:        ip,
		DetectionMethods: []string{},
		LastChecked:     time.Now(),
	}

	s.checkVPN(ip, result)
	s.checkTor(ip, result)
	s.checkDataCenter(ip, result)
	s.checkHosting(ctx, ip, result)
	s.checkHeaders(headers, result)

	s.calculateRiskScore(result)

	s.mu.Lock()
	if len(s.cache) >= s.maxCacheSize {
		s.cleanupCache()
	}
	s.cache[ip] = result
	s.mu.Unlock()

	return result, nil
}

func (s *ProxyDetectionService) cleanupCache() {
	count := 0
	target := len(s.cache) / 10
	for k := range s.cache {
		delete(s.cache, k)
		count++
		if count >= target {
			break
		}
	}
}

func (s *ProxyDetectionService) checkVPN(ip string, result *ProxyDetection) {
	for _, cidr := range s.vpnRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		parsedIP := net.ParseIP(ip)
		if parsedIP != nil && network.Contains(parsedIP) {
			result.IsVPN = true
			result.ProxyType = "private_range"
			result.DetectionMethods = append(result.DetectionMethods, "vpn_range")
			return
		}
	}
}

func (s *ProxyDetectionService) checkTor(ip string, result *ProxyDetection) {
	for _, cidr := range s.torExitNodes {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		parsedIP := net.ParseIP(ip)
		if parsedIP != nil && network.Contains(parsedIP) {
			result.IsTor = true
			result.ProxyType = "tor"
			result.DetectionMethods = append(result.DetectionMethods, "tor_exit_node")
			return
		}
	}
}

func (s *ProxyDetectionService) checkDataCenter(ip string, result *ProxyDetection) {
	for _, cidr := range s.datacenterIPs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		parsedIP := net.ParseIP(ip)
		if parsedIP != nil && network.Contains(parsedIP) {
			result.IsDataCenter = true
			result.IsHosting = true
			result.Hosting = true
			if result.ProxyType == "" {
				result.ProxyType = "datacenter"
			}
			result.DetectionMethods = append(result.DetectionMethods, "datacenter_ip_range")
			return
		}
	}
}

func (s *ProxyDetectionService) checkHosting(ctx context.Context, ip string, result *ProxyDetection) {
	reverseDNS, err := net.LookupAddr(ip)
	if err == nil && len(reverseDNS) > 0 {
		host := strings.ToLower(reverseDNS[0])
		hostingKeywords := []string{"hosting", "vps", "vmware", "digitalocean", "aws", "azure", "gce", "linode", "vultr", "contabo", "hetzner"}
		for _, keyword := range hostingKeywords {
			if strings.Contains(host, keyword) {
				result.IsHosting = true
				result.Hosting = true
				result.IsDataCenter = true
				if result.ProxyType == "" {
					result.ProxyType = "hosting"
				}
				result.DetectionMethods = append(result.DetectionMethods, "hosting_provider_dns")
				return
			}
		}
	}
}

func (s *ProxyDetectionService) checkHeaders(headers map[string]string, result *ProxyDetection) {
	if headers == nil {
		return
	}
	suspiciousHeaders := []string{
		"x-forwarded-for",
		"x-real-ip",
		"x-proxyid",
		"via",
		"forwarded",
	}
	for key := range headers {
		lowerKey := strings.ToLower(key)
		for _, suspicious := range suspiciousHeaders {
			if strings.Contains(lowerKey, suspicious) {
				result.IsProxy = true
				if result.ProxyType == "" {
					result.ProxyType = "http_header"
				}
				result.DetectionMethods = append(result.DetectionMethods, "proxy_header")
				return
			}
		}
	}
}

func (s *ProxyDetectionService) calculateRiskScore(result *ProxyDetection) {
	score := 0.0
	confidence := 0.8

	if result.IsVPN {
		score += 30
	}
	if result.IsTor {
		score += 50
	}
	if result.IsProxy {
		score += 70
	}
	if result.IsHosting {
		score += 40
	}
	if result.IsDataCenter {
		score += 25
	}
	if result.IsMobile {
		score -= 10
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	result.RiskScore = score
	result.Score = score
	result.Confidence = confidence

	if score >= 70 {
		result.RiskLevel = "high"
	} else if score >= 40 {
		result.RiskLevel = "medium"
	} else {
		result.RiskLevel = "low"
	}
}

func (s *ProxyDetectionService) DetectProxyType(ctx context.Context, ip string) (string, error) {
	result, err := s.ProxyDetection(ctx, ip, nil)
	if err != nil {
		return "", err
	}
	return result.ProxyType, nil
}

func (s *ProxyDetectionService) BatchDetection(ctx context.Context, ips []string) ([]*ProxyDetection, error) {
	results := make([]*ProxyDetection, 0, len(ips))
	for _, ip := range ips {
		result, err := s.ProxyDetection(ctx, ip, nil)
		if err != nil {
			continue
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *ProxyDetectionService) ValidateIP(ip string) error {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	return nil
}

func (s *ProxyDetectionService) GetVPNPatterns() []string {
	return []string{
		"vpn",
		"proxy",
		"tor",
		"exit.tor",
		"vpn-",
		"proxy-",
	}
}

func (s *ProxyDetectionService) ValidateHeaders(headers map[string]string) (bool, string) {
	if headers == nil {
		return false, ""
	}
	suspiciousHeaders := []string{
		"x-forwarded-for",
		"x-real-ip",
		"x-proxyid",
		"via",
		"forwarded",
	}

	for key := range headers {
		lowerKey := strings.ToLower(key)
		for _, suspicious := range suspiciousHeaders {
			if strings.Contains(lowerKey, suspicious) {
				return true, fmt.Sprintf("suspicious header: %s", key)
			}
		}
	}

	return false, ""
}

func (s *ProxyDetectionService) GetCacheStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]interface{}{
		"cache_size": len(s.cache),
		"vpn_ranges": len(s.vpnRanges),
		"tor_nodes":  len(s.torExitNodes),
	}
}

func (s *ProxyDetectionService) ExportCache() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cacheData := make(map[string]*ProxyDetection)
	for k, v := range s.cache {
		cacheData[k] = v
	}

	return json.MarshalIndent(cacheData, "", "  ")
}
