package service

import (
	"context"
	"net/http"
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
	enhanced *EnhancedProxyDetection
}

func NewProxyDetectionService() *ProxyDetectionService {
	return &ProxyDetectionService{
		enhanced: NewEnhancedProxyDetection(),
	}
}

func (s *ProxyDetectionService) DetectProxy(ip string, headers map[string]string) (*ProxyDetection, error) {
	httpHeaders := make(http.Header)
	for k, v := range headers {
		httpHeaders.Set(k, v)
	}

	ctx := context.Background()
	result, err := s.enhanced.DetectProxy(ctx, ip, httpHeaders)
	if err != nil {
		return nil, err
	}

	riskLevel := "low"
	if result.RiskScore > 70 {
		riskLevel = "high"
	} else if result.RiskScore > 40 {
		riskLevel = "medium"
	}

	country := ""
	asn := ""
	isp := ""
	if result.GeoLocation != nil {
		country = result.GeoLocation.Country
		isp = result.GeoLocation.ISP
		asn = result.GeoLocation.Organization
	}

	return &ProxyDetection{
		IPAddress:        ip,
		IsProxy:          result.IsProxy,
		IsVPN:            result.IsVPN,
		IsTor:            result.IsTor,
		IsDatacenter:     result.IsDatacenter,
		RiskLevel:        riskLevel,
		Score:            result.RiskScore,
		Confidence:       result.Confidence,
		Country:          country,
		ISP:              isp,
		ASN:              asn,
		DetectionMethods: result.Indicators,
		Hosting:          result.IsDatacenter,
		Mobile:           false,
		LastChecked:      time.Now(),
	}, nil
}

func (s *ProxyDetectionService) GetVPNPatterns() []*VPNProvider {
	return s.enhanced.GetVPNProviders()
}

func (s *ProxyDetectionService) ValidateHeaders(headers map[string]string) (bool, []string) {
	flagged := make([]string, 0)
	isFlagged := false

	suspiciousHeaders := []string{
		"X-Forwarded-For",
		"X-Real-IP",
		"Via",
		"X-ProxyChain",
		"Forwarded",
		"CF-Connecting-IP",
		"True-Client-IP",
		"X-Originating-IP",
		"X-Client-IP",
		"X-Forwarded",
	}

	for _, header := range suspiciousHeaders {
		if _, exists := headers[header]; exists {
			flagged = append(flagged, header)
			isFlagged = true
		}
	}

	return isFlagged, flagged
}