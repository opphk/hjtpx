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

type EnhancedNetworkDetection struct {
	vpnProviders    map[string]*VPNProvider
	torExitNodes    map[string]*TorExitNode
	httpClient      *http.Client
	mu              sync.RWMutex
	geoCache        map[string]*GeoLocation
	cacheExpiration time.Duration
}

type NetworkDetectionResult struct {
	IPAddress        string                `json:"ip_address"`
	IsProxy          bool                  `json:"is_proxy"`
	IsVPN            bool                  `json:"is_vpn"`
	IsTor            bool                  `json:"is_tor"`
	IsDatacenter     bool                  `json:"is_datacenter"`
	IsMobile         bool                  `json:"is_mobile"`
	RiskLevel        string                `json:"risk_level"`
	RiskScore        float64               `json:"risk_score"`
	Confidence       float64               `json:"confidence"`
	DetectionMethods []string              `json:"detection_methods"`
	GeoLocation      *GeoLocation          `json:"geo_location"`
	NetworkInfo      *NetworkInfo          `json:"network_info"`
	ProxyDetails     *ProxyDetails         `json:"proxy_details"`
	VPNDetails       *VPNDetails           `json:"vpn_details"`
	TorDetails       *TorDetails           `json:"tor_details"`
}

type NetworkInfo struct {
	ASN            int      `json:"asn"`
	ISP            string   `json:"isp"`
	Organization   string   `json:"organization"`
	ConnectionType string   `json:"connection_type"`
	Latency        float64  `json:"latency_ms"`
	IPVersion      int      `json:"ip_version"`
}

type ProxyDetails struct {
	ProxyType      string   `json:"proxy_type"`
	HopCount       int      `json:"hop_count"`
	AnonymityLevel string   `json:"anonymity_level"`
	ForwardedIPs   []string `json:"forwarded_ips"`
}

type VPNDetails struct {
	ProviderName string   `json:"provider_name"`
	ProviderASNs []int    `json:"provider_asns"`
	Confidence   float64  `json:"confidence"`
	Evidence     []string `json:"evidence"`
}

type TorDetails struct {
	ExitNodeIP string    `json:"exit_node_ip"`
	Country    string    `json:"country"`
	LastSeen   time.Time `json:"last_seen"`
	Bandwidth  int       `json:"bandwidth"`
	RelayCount int       `json:"relay_count"`
}

func NewEnhancedNetworkDetection() *EnhancedNetworkDetection {
	nd := &EnhancedNetworkDetection{
		vpnProviders:    make(map[string]*VPNProvider),
		torExitNodes:    make(map[string]*TorExitNode),
		geoCache:        make(map[string]*GeoLocation),
		cacheExpiration: 30 * time.Minute,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).Dial,
				TLSHandshakeTimeout: 10 * time.Second,
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}

	nd.initVPNProviders()
	nd.initTorExitNodes()

	return nd
}

func (nd *EnhancedNetworkDetection) initVPNProviders() {
	nd.vpnProviders["nordvpn"] = &VPNProvider{
		Name:    "NordVPN",
		IPRanges: []string{"45.33.", "45.45.", "45.67.", "45.89.", "103.252.", "104.224."},
		ASN:     []int{201229, 212502, 198354},
		KnownIPs: make(map[string]bool),
	}
	nd.vpnProviders["expressvpn"] = &VPNProvider{
		Name:    "ExpressVPN",
		IPRanges: []string{"23.", "104.154.", "132.252.", "144.217.", "154.72."},
		ASN:     []int{201229, 394165},
		KnownIPs: make(map[string]bool),
	}
	nd.vpnProviders["surfshark"] = &VPNProvider{
		Name:    "Surfshark",
		IPRanges: []string{"172.104.", "185.220.", "188.172.", "192.187."},
		ASN:     []int{212502, 393218},
		KnownIPs: make(map[string]bool),
	}
	nd.vpnProviders["cyberghost"] = &VPNProvider{
		Name:    "CyberGhost",
		IPRanges: []string{"37.", "82.", "85.", "89.", "91.", "176."},
		ASN:     []int{207083, 20454},
		KnownIPs: make(map[string]bool),
	}
	nd.vpnProviders["protonvpn"] = &VPNProvider{
		Name:    "ProtonVPN",
		IPRanges: []string{"185.195.", "185.220.", "193.122.", "188.166."},
		ASN:     []int{19168, 51087},
		KnownIPs: make(map[string]bool),
	}
	nd.vpnProviders["mullvad"] = &VPNProvider{
		Name:    "Mullvad",
		IPRanges: []string{"185.195.", "194.132.", "104.16."},
		ASN:     []int{39189, 14061},
		KnownIPs: make(map[string]bool),
	}
	nd.vpnProviders["windscribe"] = &VPNProvider{
		Name:    "Windscribe",
		IPRanges: []string{"35.182.", "45.33.", "104.236.", "185.233."},
		ASN:     []int{201229, 393218},
		KnownIPs: make(map[string]bool),
	}
	nd.vpnProviders["ivpn"] = &VPNProvider{
		Name:    "IVPN",
		IPRanges: []string{"185.183.", "198.50.", "45.33."},
		ASN:     []int{394242},
		KnownIPs: make(map[string]bool),
	}
	nd.vpnProviders["hide.me"] = &VPNProvider{
		Name:    "hide.me",
		IPRanges: []string{"103.231.", "103.245.", "104.244.", "185.176."},
		ASN:     []int{13213, 51468},
		KnownIPs: make(map[string]bool),
	}
	nd.vpnProviders["purevpn"] = &VPNProvider{
		Name:    "PureVPN",
		IPRanges: []string{"104.155.", "104.156.", "107.178.", "149.56."},
		ASN:     []int{36351, 19850},
		KnownIPs: make(map[string]bool),
	}
}

func (nd *EnhancedNetworkDetection) initTorExitNodes() {
	nd.torExitNodes["128.31.0.34"] = &TorExitNode{IP: "128.31.0.34", ORPort: 443, DirectoryPort: 9030, Country: "US", Bandwidth: 1000}
	nd.torExitNodes["128.93.34.5"] = &TorExitNode{IP: "128.93.34.5", ORPort: 443, DirectoryPort: 9030, Country: "US", Bandwidth: 800}
	nd.torExitNodes["131.188.40.189"] = &TorExitNode{IP: "131.188.40.189", ORPort: 443, DirectoryPort: 9030, Country: "DE", Bandwidth: 1200}
	nd.torExitNodes["154.35.22.11"] = &TorExitNode{IP: "154.35.22.11", ORPort: 443, DirectoryPort: 9030, Country: "NL", Bandwidth: 900}
	nd.torExitNodes["171.25.193.77"] = &TorExitNode{IP: "171.25.193.77", ORPort: 443, DirectoryPort: 9030, Country: "DE", Bandwidth: 1100}
	nd.torExitNodes["176.10.99.200"] = &TorExitNode{IP: "176.10.99.200", ORPort: 443, DirectoryPort: 9030, Country: "FR", Bandwidth: 750}
	nd.torExitNodes["185.220.101.1"] = &TorExitNode{IP: "185.220.101.1", ORPort: 443, DirectoryPort: 9030, Country: "NL", Bandwidth: 2000}
	nd.torExitNodes["185.220.101.2"] = &TorExitNode{IP: "185.220.101.2", ORPort: 443, DirectoryPort: 9030, Country: "NL", Bandwidth: 1800}
	nd.torExitNodes["185.220.101.3"] = &TorExitNode{IP: "185.220.101.3", ORPort: 443, DirectoryPort: 9030, Country: "NL", Bandwidth: 1500}
	nd.torExitNodes["185.220.101.4"] = &TorExitNode{IP: "185.220.101.4", ORPort: 443, DirectoryPort: 9030, Country: "NL", Bandwidth: 1600}
	nd.torExitNodes["185.220.101.5"] = &TorExitNode{IP: "185.220.101.5", ORPort: 443, DirectoryPort: 9030, Country: "NL", Bandwidth: 1400}
	nd.torExitNodes["192.95.30.12"] = &TorExitNode{IP: "192.95.30.12", ORPort: 443, DirectoryPort: 9030, Country: "CA", Bandwidth: 600}
	nd.torExitNodes["193.11.166.7"] = &TorExitNode{IP: "193.11.166.7", ORPort: 443, DirectoryPort: 9030, Country: "FR", Bandwidth: 850}
	nd.torExitNodes["199.249.230.1"] = &TorExitNode{IP: "199.249.230.1", ORPort: 443, DirectoryPort: 9030, Country: "US", Bandwidth: 950}
	nd.torExitNodes["204.13.164.53"] = &TorExitNode{IP: "204.13.164.53", ORPort: 443, DirectoryPort: 9030, Country: "US", Bandwidth: 700}
	nd.torExitNodes["209.141.60.1"] = &TorExitNode{IP: "209.141.60.1", ORPort: 443, DirectoryPort: 9030, Country: "US", Bandwidth: 650}
	nd.torExitNodes["23.129.64.1"] = &TorExitNode{IP: "23.129.64.1", ORPort: 443, DirectoryPort: 9030, Country: "US", Bandwidth: 1300}
	nd.torExitNodes["23.129.64.2"] = &TorExitNode{IP: "23.129.64.2", ORPort: 443, DirectoryPort: 9030, Country: "US", Bandwidth: 1250}
	nd.torExitNodes["45.154.34.1"] = &TorExitNode{IP: "45.154.34.1", ORPort: 443, DirectoryPort: 9030, Country: "NL", Bandwidth: 1700}
	nd.torExitNodes["62.210.105.116"] = &TorExitNode{IP: "62.210.105.116", ORPort: 443, DirectoryPort: 9030, Country: "FR", Bandwidth: 550}
	nd.torExitNodes["66.111.2.131"] = &TorExitNode{IP: "66.111.2.131", ORPort: 443, DirectoryPort: 9030, Country: "US", Bandwidth: 800}
	nd.torExitNodes["72.14.180.105"] = &TorExitNode{IP: "72.14.180.105", ORPort: 443, DirectoryPort: 9030, Country: "US", Bandwidth: 500}
	nd.torExitNodes["78.142.211.102"] = &TorExitNode{IP: "78.142.211.102", ORPort: 443, DirectoryPort: 9030, Country: "SE", Bandwidth: 700}
	nd.torExitNodes["86.59.21.38"] = &TorExitNode{IP: "86.59.21.38", ORPort: 443, DirectoryPort: 9030, Country: "GB", Bandwidth: 600}
	nd.torExitNodes["91.250.242.12"] = &TorExitNode{IP: "91.250.242.12", ORPort: 443, DirectoryPort: 9030, Country: "FI", Bandwidth: 900}
	nd.torExitNodes["94.140.8.48"] = &TorExitNode{IP: "94.140.8.48", ORPort: 443, DirectoryPort: 9030, Country: "NL", Bandwidth: 1100}
	nd.torExitNodes["95.211.138.97"] = &TorExitNode{IP: "95.211.138.97", ORPort: 443, DirectoryPort: 9030, Country: "FI", Bandwidth: 850}
}

func (nd *EnhancedNetworkDetection) DetectNetwork(ctx context.Context, ip string, headers http.Header) (*NetworkDetectionResult, error) {
	result := &NetworkDetectionResult{
		IPAddress:        ip,
		DetectionMethods: make([]string, 0),
		NetworkInfo:      &NetworkInfo{},
		GeoLocation:      &GeoLocation{},
	}

	nd.mu.Lock()
	if cached, exists := nd.geoCache[ip]; exists {
		result.GeoLocation = cached
		nd.mu.Unlock()
	} else {
		nd.mu.Unlock()
		geoInfo, err := nd.getGeoLocation(ctx, ip)
		if err == nil {
			result.GeoLocation = geoInfo
			nd.mu.Lock()
			nd.geoCache[ip] = geoInfo
			nd.mu.Unlock()
		}
	}

	result.NetworkInfo.IPVersion = nd.detectIPVersion(ip)

	if result.GeoLocation.ASN != 0 {
		result.NetworkInfo.ASN = result.GeoLocation.ASN
		result.NetworkInfo.ISP = result.GeoLocation.ISP
		result.NetworkInfo.Organization = result.GeoLocation.Organization
		result.DetectionMethods = append(result.DetectionMethods, fmt.Sprintf("asn:%d", result.GeoLocation.ASN))
	}

	nd.detectProxy(ip, headers, result)
	nd.detectVPN(ip, result)
	nd.detectTor(ip, result)
	nd.detectDatacenter(ip, result)
	nd.measureNetworkInfo(ctx, ip, result)

	result.Confidence = nd.calculateConfidence(result)
	result.RiskScore = nd.calculateRiskScore(result)
	result.RiskLevel = nd.determineRiskLevel(result.RiskScore)

	return result, nil
}

func (nd *EnhancedNetworkDetection) detectIPVersion(ip string) int {
	parsedIP := net.ParseIP(ip)
	if parsedIP != nil && parsedIP.To4() == nil {
		return 6
	}
	return 4
}

func (nd *EnhancedNetworkDetection) detectProxy(ip string, headers http.Header, result *NetworkDetectionResult) {
	proxyDetails := &ProxyDetails{ForwardedIPs: make([]string, 0)}

	if xff := headers.Get("X-Forwarded-For"); xff != "" {
		proxyDetails.ForwardedIPs = strings.Split(strings.ReplaceAll(xff, " ", ""), ",")
		proxyDetails.HopCount = len(proxyDetails.ForwardedIPs)
		result.DetectionMethods = append(result.DetectionMethods, fmt.Sprintf("xff_hops:%d", proxyDetails.HopCount))

		if proxyDetails.HopCount > 1 {
			result.IsProxy = true
			proxyDetails.ProxyType = "multi-hop"
			proxyDetails.AnonymityLevel = "high"
			result.DetectionMethods = append(result.DetectionMethods, "multi_hop_proxy")
		}
	}

	if headers.Get("X-Real-IP") != "" {
		result.DetectionMethods = append(result.DetectionMethods, "x_real_ip_present")
		proxyDetails.AnonymityLevel = "medium"
	}

	if via := headers.Get("Via"); via != "" {
		result.DetectionMethods = append(result.DetectionMethods, "via_header_present")
		viaLower := strings.ToLower(via)
		proxyKeywords := []string{"haproxy", "squid", "nginx", "apache", "varnish", "cloudflare", "cdn", "proxy"}
		for _, keyword := range proxyKeywords {
			if strings.Contains(viaLower, keyword) {
				result.IsProxy = true
				proxyDetails.ProxyType = keyword
				result.DetectionMethods = append(result.DetectionMethods, fmt.Sprintf("known_proxy_type:%s", keyword))
				break
			}
		}
	}

	if headers.Get("X-ProxyChain") != "" {
		result.IsProxy = true
		proxyDetails.AnonymityLevel = "high"
		result.DetectionMethods = append(result.DetectionMethods, "proxy_chain_detected")
	}

	if headers.Get("X-ProxyId") != "" {
		result.IsProxy = true
		result.DetectionMethods = append(result.DetectionMethods, "proxy_id_header")
	}

	if forwarded := headers.Get("Forwarded"); forwarded != "" {
		result.DetectionMethods = append(result.DetectionMethods, "forwarded_header_present")
		if strings.Contains(strings.ToLower(forwarded), "for=") {
			proxyDetails.HopCount++
		}
	}

	if headers.Get("CF-Connecting-IP") != "" {
		result.DetectionMethods = append(result.DetectionMethods, "cloudflare_proxy")
		proxyDetails.ProxyType = "cloudflare"
	}

	if result.IsProxy && proxyDetails.AnonymityLevel == "" {
		proxyDetails.AnonymityLevel = "low"
	}

	result.ProxyDetails = proxyDetails
}

func (nd *EnhancedNetworkDetection) detectVPN(ip string, result *NetworkDetectionResult) {
	vpnDetails := &VPNDetails{Evidence: make([]string, 0)}

	nd.mu.RLock()
	for providerName, provider := range nd.vpnProviders {
		for _, prefix := range provider.IPRanges {
			if strings.HasPrefix(ip, prefix) {
				result.IsVPN = true
				vpnDetails.ProviderName = providerName
				vpnDetails.ProviderASNs = provider.ASN
				vpnDetails.Confidence = 0.85
				vpnDetails.Evidence = append(vpnDetails.Evidence, fmt.Sprintf("IP range match: %s", prefix))
				result.DetectionMethods = append(result.DetectionMethods, fmt.Sprintf("vpn_provider:%s", providerName))
				break
			}
		}
		if result.IsVPN {
			break
		}
	}
	nd.mu.RUnlock()

	if result.NetworkInfo.ASN != 0 {
		if isVPN, provider := nd.DetectVPNByASN(result.NetworkInfo.ASN); isVPN {
			if !result.IsVPN {
				result.IsVPN = true
				vpnDetails.ProviderName = provider
				vpnDetails.Evidence = append(vpnDetails.Evidence, fmt.Sprintf("ASN match: AS%d", result.NetworkInfo.ASN))
				result.DetectionMethods = append(result.DetectionMethods, fmt.Sprintf("vpn_by_asn:%s", provider))
			}
			vpnDetails.Confidence = math.Max(vpnDetails.Confidence, 0.9)
		}
	}

	result.VPNDetails = vpnDetails
}

func (nd *EnhancedNetworkDetection) detectTor(ip string, result *NetworkDetectionResult) {
	nd.mu.RLock()
	node, exists := nd.torExitNodes[ip]
	nd.mu.RUnlock()

	if exists {
		result.IsTor = true
		result.IsProxy = true
		result.DetectionMethods = append(result.DetectionMethods, "known_tor_exit_node")

		result.TorDetails = &TorDetails{
			ExitNodeIP: node.IP,
			Country:    node.Country,
			LastSeen:   node.LastSeen,
			Bandwidth:  node.Bandwidth,
			RelayCount: 3,
		}
	}
}

func (nd *EnhancedNetworkDetection) detectDatacenter(ip string, result *NetworkDetectionResult) {
	datacenterPrefixes := []string{
		"3.", "4.", "8.", "13.", "15.", "16.", "17.", "18.", "20.",
		"23.", "34.", "35.", "40.", "44.", "45.", "47.", "48.", "49.",
		"50.", "52.", "54.", "63.", "64.", "65.", "66.", "67.", "68.",
		"70.", "72.", "74.", "75.", "76.", "77.", "78.", "79.", "80.",
		"81.", "82.", "83.", "84.", "85.", "87.", "88.", "89.", "90.",
		"91.", "92.", "93.", "94.", "95.", "96.", "97.", "98.", "99.",
		"100.", "103.", "104.", "107.", "108.", "109.", "115.", "116.",
		"140.", "141.", "142.", "143.", "144.", "147.", "148.", "149.",
		"150.", "151.", "152.", "155.", "156.", "162.", "165.", "168.",
		"172.", "185.", "186.", "188.", "192.", "198.", "199.", "203.",
		"204.", "205.", "206.", "207.", "208.", "209.", "212.", "213.",
	}

	for _, prefix := range datacenterPrefixes {
		if strings.HasPrefix(ip, prefix) {
			result.IsDatacenter = true
			result.DetectionMethods = append(result.DetectionMethods, "datacenter_ip_range")
			break
		}
	}

	if result.GeoLocation.Organization != "" {
		datacenterOrgs := []string{"amazon", "aws", "google", "cloud", "microsoft", "azure", "digitalocean", "linode", "ovh", "hetzner", "vultr"}
		orgLower := strings.ToLower(result.GeoLocation.Organization)
		for _, org := range datacenterOrgs {
			if strings.Contains(orgLower, org) {
				result.IsDatacenter = true
				result.DetectionMethods = append(result.DetectionMethods, fmt.Sprintf("datacenter_org:%s", org))
				break
			}
		}
	}
}

func (nd *EnhancedNetworkDetection) measureNetworkInfo(ctx context.Context, ip string, result *NetworkDetectionResult) {
	if result.NetworkInfo.ISP != "" {
		if strings.Contains(strings.ToLower(result.NetworkInfo.ISP), "mobile") ||
			strings.Contains(strings.ToLower(result.NetworkInfo.ISP), "cell") ||
			strings.Contains(strings.ToLower(result.NetworkInfo.ISP), "wireless") {
			result.IsMobile = true
			result.NetworkInfo.ConnectionType = "mobile"
		} else {
			result.NetworkInfo.ConnectionType = "fixed"
		}
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", ip+":80", 1*time.Second)
	if err == nil {
		conn.Close()
		result.NetworkInfo.Latency = time.Since(start).Seconds() * 1000
	} else {
		result.NetworkInfo.Latency = -1
	}
}

func (nd *EnhancedNetworkDetection) getGeoLocation(ctx context.Context, ip string) (*GeoLocation, error) {
	geo := &GeoLocation{}

	if net.ParseIP(ip) == nil {
		return geo, fmt.Errorf("invalid IP address")
	}

	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,countryCode,region,regionName,city,isp,org,as,timezone,lat,lon,mobile,proxy,hosting", ip)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return geo, err
	}

	resp, err := nd.httpClient.Do(req)
	if err != nil {
		return geo, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return geo, err
	}

	var ipApi struct {
		Status      string  `json:"status"`
		Country     string  `json:"country"`
		CountryCode string  `json:"countryCode"`
		Region      string  `json:"regionName"`
		City        string  `json:"city"`
		ISP         string  `json:"isp"`
		Org         string  `json:"org"`
		AS          string  `json:"as"`
		Timezone    string  `json:"timezone"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
	}

	if err := json.Unmarshal(body, &ipApi); err != nil {
		return geo, err
	}

	if ipApi.Status == "success" {
		geo.Country = ipApi.CountryCode
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

func (nd *EnhancedNetworkDetection) calculateConfidence(result *NetworkDetectionResult) float64 {
	confidence := 0.0

	if result.IsProxy {
		confidence += 30
	}
	if result.IsVPN {
		confidence += 35
	}
	if result.IsTor {
		confidence += 45
	}
	if result.IsDatacenter {
		confidence += 20
	}

	if len(result.DetectionMethods) > 0 {
		confidence += float64(len(result.DetectionMethods)) * 3
	}

	if result.NetworkInfo != nil && result.NetworkInfo.Latency > 0 {
		if result.NetworkInfo.Latency > 500 {
			confidence += 10
		} else if result.NetworkInfo.Latency < 50 {
			confidence += 5
		}
	}

	if result.GeoLocation != nil && result.GeoLocation.Country != "" {
		confidence += 15
	}

	return math.Min(confidence, 100)
}

func (nd *EnhancedNetworkDetection) calculateRiskScore(result *NetworkDetectionResult) float64 {
	score := 0.0

	if result.IsTor {
		score += 60
	}
	if result.IsVPN {
		score += 40
	}
	if result.IsProxy {
		if result.ProxyDetails != nil && result.ProxyDetails.ProxyType == "multi-hop" {
			score += 35
		} else {
			score += 25
		}
	}
	if result.IsDatacenter {
		score += 20
	}

	if result.GeoLocation != nil {
		highRiskCountries := []string{"RU", "CN", "KP", "IR", "BY", "VE", "PK", "NG", "BD", "UA", "SY", "AF"}
		for _, country := range highRiskCountries {
			if result.GeoLocation.Country == country {
				score += 20
				break
			}
		}

		mediumRiskCountries := []string{"IN", "BR", "MX", "TR", "ID", "EG", "TH", "VN", "MY", "PH"}
		for _, country := range mediumRiskCountries {
			if result.GeoLocation.Country == country {
				score += 10
				break
			}
		}
	}

	if result.NetworkInfo != nil && result.NetworkInfo.Latency > 0 && result.NetworkInfo.Latency > 1000 {
		score += 10
	}

	if result.IsMobile {
		score += 5
	}

	return math.Min(score, 100)
}

func (nd *EnhancedNetworkDetection) determineRiskLevel(score float64) string {
	switch {
	case score >= 80:
		return "critical"
	case score >= 60:
		return "high"
	case score >= 40:
		return "medium"
	case score >= 20:
		return "low"
	default:
		return "none"
	}
}

func (nd *EnhancedNetworkDetection) DetectVPNByASN(asn int) (bool, string) {
	vpnASNs := map[int]string{
		201229: "Private Internet Access / NordVPN",
		212502: "NordVPN / Surfshark",
		207083: "CyberGhost",
		393218: "Surfshark",
		19168:  "ProtonVPN",
		394242: "IVPN",
		36351:  "PureVPN",
		35988:  "StrongVPN",
		13213:  "hide.me",
		51468:  "hide.me",
		394165: "ExpressVPN",
		198354: "NordVPN",
		14061:  "Mullvad",
		39189:  "Mullvad",
		51087:  "ProtonVPN",
	}

	if provider, exists := vpnASNs[asn]; exists {
		return true, provider
	}
	return false, ""
}

func (nd *EnhancedNetworkDetection) IsTorExitNode(ip string) bool {
	nd.mu.RLock()
	defer nd.mu.RUnlock()
	_, exists := nd.torExitNodes[ip]
	return exists
}

func (nd *EnhancedNetworkDetection) GetTorExitNodeInfo(ip string) (*TorExitNode, bool) {
	nd.mu.RLock()
	defer nd.mu.RUnlock()
	node, exists := nd.torExitNodes[ip]
	return node, exists
}

func (nd *EnhancedNetworkDetection) AddTorExitNode(node *TorExitNode) {
	nd.mu.Lock()
	defer nd.mu.Unlock()
	nd.torExitNodes[node.IP] = node
}

func (nd *EnhancedNetworkDetection) RemoveTorExitNode(ip string) {
	nd.mu.Lock()
	defer nd.mu.Unlock()
	delete(nd.torExitNodes, ip)
}

func (nd *EnhancedNetworkDetection) GetVPNProviders() []*VPNProvider {
	nd.mu.RLock()
	defer nd.mu.RUnlock()
	providers := make([]*VPNProvider, 0, len(nd.vpnProviders))
	for _, provider := range nd.vpnProviders {
		providers = append(providers, provider)
	}
	return providers
}

func (nd *EnhancedNetworkDetection) UpdateVPNProvider(name string, provider *VPNProvider) {
	nd.mu.Lock()
	defer nd.mu.Unlock()
	nd.vpnProviders[name] = provider
}

func (nd *EnhancedNetworkDetection) BatchDetect(ctx context.Context, ips []string, headers http.Header) []*NetworkDetectionResult {
	results := make([]*NetworkDetectionResult, len(ips))
	var wg sync.WaitGroup

	for i, ip := range ips {
		wg.Add(1)
		go func(idx int, ipAddr string) {
			defer wg.Done()
			result, err := nd.DetectNetwork(ctx, ipAddr, headers)
			if err != nil {
				results[idx] = &NetworkDetectionResult{
					IPAddress: ipAddr,
					RiskScore: 100,
					RiskLevel: "critical",
					DetectionMethods: []string{"detection_error: " + err.Error()},
				}
			} else {
				results[idx] = result
			}
		}(i, ip)
	}

	wg.Wait()
	return results
}

func (nd *EnhancedNetworkDetection) GenerateRiskReport(result *NetworkDetectionResult) *NetworkRiskReport {
	report := &NetworkRiskReport{
		Timestamp:       time.Now(),
		IPAddress:       result.IPAddress,
		RiskLevel:       result.RiskLevel,
		RiskScore:       result.RiskScore,
		Confidence:      result.Confidence,
		IsThreat:        result.RiskLevel == "critical" || result.RiskLevel == "high",
		DetectionMethods: result.DetectionMethods,
		Recommendations: make([]string, 0),
	}

	if result.GeoLocation != nil && result.GeoLocation.City != "" {
		report.GeoLocation = fmt.Sprintf("%s, %s", result.GeoLocation.City, result.GeoLocation.Country)
	}

	switch result.RiskLevel {
	case "critical":
		report.Summary = "检测到高风险网络环境（Tor/VPN/多层代理）"
		report.Recommendations = []string{"立即阻止访问", "记录完整日志", "通知安全团队", "考虑添加额外验证"}
	case "high":
		report.Summary = "检测到代理或VPN连接"
		report.Recommendations = []string{"添加额外验证步骤", "限制敏感操作", "增强监控"}
	case "medium":
		report.Summary = "可能使用代理或位于数据中心"
		report.Recommendations = []string{"启用增强监控", "考虑添加验证码"}
	case "low":
		report.Summary = "低风险网络环境"
		report.Recommendations = []string{"正常处理请求", "持续监控"}
	default:
		report.Summary = "未检测到明显风险"
		report.Recommendations = []string{"正常处理请求"}
	}

	return report
}

type NetworkRiskReport struct {
	Timestamp        time.Time `json:"timestamp"`
	IPAddress        string    `json:"ip_address"`
	RiskLevel        string    `json:"risk_level"`
	RiskScore        float64   `json:"risk_score"`
	Confidence       float64   `json:"confidence"`
	IsThreat         bool      `json:"is_threat"`
	Summary          string    `json:"summary"`
	GeoLocation      string    `json:"geo_location,omitempty"`
	DetectionMethods []string  `json:"detection_methods"`
	Recommendations  []string  `json:"recommendations"`
}

func (nd *EnhancedNetworkDetection) CleanupCache() {
	nd.mu.Lock()
	defer nd.mu.Unlock()
	for ip := range nd.geoCache {
		delete(nd.geoCache, ip)
	}
}
