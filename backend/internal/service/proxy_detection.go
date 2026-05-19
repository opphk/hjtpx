package service

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type ProxyDetection struct {
	IPAddress        string    `json:"ip_address"`
	IsProxy          bool      `json:"is_proxy"`
	IsVPN            bool      `json:"is_vpn"`
	IsTor            bool      `json:"is_tor"`
	IsDatacenter     bool      `json:"is_datacenter"`
	RiskLevel        string    `json:"risk_level"`
	Score            float64   `json:"score"`
	Confidence       float64   `json:"confidence"`
	Country          string    `json:"country"`
	ISP              string    `json:"isp"`
	ASN              string    `json:"asn"`
	DetectionMethods []string  `json:"detection_methods"`
	Hosting          bool      `json:"hosting"`
	Mobile           bool      `json:"mobile"`
	LastChecked      time.Time `json:"last_checked"`
}

type ProxyDetectionService struct {
	enhanced     *EnhancedProxyDetection
	cache        map[string]*ProxyDetection
	cacheMutex   sync.RWMutex
	cacheTTL     time.Duration
}

func NewProxyDetectionService() *ProxyDetectionService {
	return &ProxyDetectionService{
		enhanced: NewEnhancedProxyDetection(),
		cache:    make(map[string]*ProxyDetection),
		cacheTTL: 1 * time.Hour,
	}
}

func (s *ProxyDetectionService) DetectProxy(ip string, headers map[string]string) (*ProxyDetection, error) {
	s.cacheMutex.RLock()
	if cached, exists := s.cache[ip]; exists {
		if time.Since(cached.LastChecked) < s.cacheTTL {
			s.cacheMutex.RUnlock()
			return cached, nil
		}
	}
	s.cacheMutex.RUnlock()

	httpHeaders := make(http.Header)
	for k, v := range headers {
		httpHeaders.Set(k, v)
	}

	ctx := context.Background()
	enhancedResult, err := s.enhanced.DetectProxy(ctx, ip, httpHeaders)
	if err != nil {
		return &ProxyDetection{
			IPAddress:   ip,
			IsProxy:     false,
			IsVPN:       false,
			IsTor:       false,
			RiskLevel:   "unknown",
			Score:       0,
			Confidence:  0,
			LastChecked: time.Now(),
		}, nil
	}

	result := &ProxyDetection{
		IPAddress:        ip,
		IsProxy:          enhancedResult.IsProxy,
		IsVPN:            enhancedResult.IsVPN,
		IsTor:            enhancedResult.IsTor,
		IsDatacenter:     enhancedResult.IsDatacenter,
		RiskLevel:        "low",
		Score:            enhancedResult.RiskScore,
		Confidence:       enhancedResult.Confidence,
		DetectionMethods: enhancedResult.Indicators,
		LastChecked:      time.Now(),
	}

	if enhancedResult.GeoLocation != nil {
		result.Country = enhancedResult.GeoLocation.Country
		result.ISP = enhancedResult.GeoLocation.ISP
		result.ASN = strconv.Itoa(enhancedResult.GeoLocation.ASN)
	}

	if result.Score >= 80 {
		result.RiskLevel = "critical"
	} else if result.Score >= 60 {
		result.RiskLevel = "high"
	} else if result.Score >= 40 {
		result.RiskLevel = "medium"
	}

	s.cacheMutex.Lock()
	s.cache[ip] = result
	s.cacheMutex.Unlock()

	return result, nil
}

func (s *ProxyDetectionService) GetVPNPatterns() []string {
	providers := s.enhanced.GetVPNProviders()
	patterns := make([]string, 0, len(providers))
	for _, p := range providers {
		patterns = append(patterns, p.Name)
	}
	return patterns
}

func (s *ProxyDetectionService) ValidateHeaders(headers map[string]string) (bool, []string) {
	httpHeaders := make(http.Header)
	for k, v := range headers {
		httpHeaders.Set(k, v)
	}

	result := &EnhancedProxyResult{
		Headers: &ProxyHeaders{},
	}
	s.enhanced.AnalyzeHeaders(result, httpHeaders)

	flagged := []string{}
	if result.Headers.XForwardedFor {
		flagged = append(flagged, "X-Forwarded-For")
	}
	if result.Headers.Via {
		flagged = append(flagged, "Via")
	}
	if result.Headers.XRealIP {
		flagged = append(flagged, "X-Real-IP")
	}

	return len(flagged) > 0, flagged
}

func (s *ProxyDetectionService) DetectProxyWithHeaders(ip string, headers map[string]string) (*ProxyDetection, error) {
	return s.DetectProxy(ip, headers)
}

func (s *ProxyDetectionService) IsProxy(ip string) (bool, error) {
	result, err := s.DetectProxy(ip, nil)
	if err != nil {
		return false, err
	}
	return result.IsProxy, nil
}

func (s *ProxyDetectionService) GetProxyInfo(ip string) (*ProxyInfo, error) {
	result, err := s.DetectProxy(ip, nil)
	if err != nil {
		return nil, err
	}
	
	proxyType := ""
	if result.IsVPN {
		proxyType = "VPN"
	} else if result.IsTor {
		proxyType = "Tor"
	} else if result.IsProxy {
		proxyType = "Proxy"
	}
	
	return &ProxyInfo{
		IP:          ip,
		Type:        proxyType,
		Country:     result.Country,
		ISP:         result.ISP,
		LastChecked: result.LastChecked,
	}, nil
}

func (s *ProxyDetectionService) CheckVPN(ip string) (bool, error) {
	result, err := s.DetectProxy(ip, nil)
	if err != nil {
		return false, err
	}
	return result.IsVPN, nil
}

func (s *ProxyDetectionService) CheckTor(ip string) (bool, error) {
	result, err := s.DetectProxy(ip, nil)
	if err != nil {
		return false, err
	}
	return result.IsTor, nil
}

func (s *ProxyDetectionService) CheckHosting(ip string) (bool, error) {
	result, err := s.DetectProxy(ip, nil)
	if err != nil {
		return false, err
	}
	return result.Hosting, nil
}

func (s *ProxyDetectionService) GetIPReputation(ip string) (map[string]interface{}, error) {
	result, err := s.DetectProxy(ip, nil)
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"ip":          ip,
		"risk_score":  result.Score,
		"risk_level":  result.RiskLevel,
		"confidence":  result.Confidence,
		"is_proxy":    result.IsProxy,
		"is_vpn":      result.IsVPN,
		"is_tor":      result.IsTor,
		"country":     result.Country,
		"isp":         result.ISP,
	}, nil
}

func (s *ProxyDetectionService) UpdateIPCache(info *ProxyInfo) error {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	
	isProxy := info.Type != ""
	
	s.cache[info.IP] = &ProxyDetection{
		IPAddress:   info.IP,
		IsProxy:     isProxy,
		LastChecked: time.Now(),
	}
	return nil
}

func (s *ProxyDetectionService) ClearIPCache() error {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	
	s.cache = make(map[string]*ProxyDetection)
	return nil
}

func (s *ProxyDetectionService) GetDetectionStats() (map[string]interface{}, error) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()
	
	return map[string]interface{}{
		"cache_size":    len(s.cache),
		"cache_ttl":     s.cacheTTL.String(),
		"detection_count": 0,
	}, nil
}
