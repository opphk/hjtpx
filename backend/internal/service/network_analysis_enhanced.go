package service

import (
	"encoding/json"
	"net"
	"strings"
	"sync"
	"time"
)

type NetworkAnalysisEnhanced struct {
	ipReputationCache map[string]*IPReputation
	torExitNodes      map[string]bool
	vpnProviders      map[string]bool
	mu                sync.RWMutex
	cacheTTL          time.Duration
}

func NewNetworkAnalysisEnhanced() *NetworkAnalysisEnhanced {
	return &NetworkAnalysisEnhanced{
		ipReputationCache: make(map[string]*IPReputation),
		torExitNodes:      loadTorExitNodes(),
		vpnProviders:      loadVPNProviders(),
		cacheTTL:          1 * time.Hour,
	}
}

func (n *NetworkAnalysisEnhanced) AnalyzeIPReputation(ip string) *IPReputation {
	n.mu.RLock()
	if rep, ok := n.ipReputationCache[ip]; ok && time.Since(rep.Timestamp) < n.cacheTTL {
		n.mu.RUnlock()
		return rep
	}
	n.mu.RUnlock()

	reputation := &IPReputation{
		IP:        ip,
		Timestamp: time.Now(),
	}

	reputation.IsProxy = n.checkProxy(ip)
	reputation.IsTor = n.checkTorExitNode(ip)
	reputation.IsVPN, reputation.VPNProvider = n.checkVPN(ip)
	reputation.ASNInfo = n.getASNInfo(ip)
	reputation.Score = n.calculateReputationScore(reputation)

	n.mu.Lock()
	n.ipReputationCache[ip] = reputation
	n.mu.Unlock()

	return reputation
}

func (n *NetworkAnalysisEnhanced) DetectResidentialProxy(ip string, headers map[string]string) *ResidentialProxyResult {
	result := &ResidentialProxyResult{}

	ipType := n.classifyIPType(ip)
	result.IPType = ipType

	switch ipType {
	case "residential":
		result.IsResidentialProxy = true
		result.Confidence = 0.9
	case "datacenter":
		result.IsResidentialProxy = false
		result.Confidence = 0.8
	default:
		result.Confidence = n.analyzeProxySignals(ip, headers)
		result.IsResidentialProxy = result.Confidence > 0.6
	}

	result.ISP = n.getISPInfo(ip)
	result.PortScanDetected = n.detectPortScan(ip)
	result.InBlacklist = n.checkBlacklist(ip)

	return result
}

func (n *NetworkAnalysisEnhanced) AnalyzeASN(asnInfo *ASNInfo) *ASNAnalysis {
	analysis := &ASNAnalysis{}

	if asnInfo == nil {
		analysis.Risk = "unknown"
		return analysis
	}

	analysis.ASN = asnInfo.ASN
	analysis.Provider = asnInfo.Provider

	if n.isDatacenterASN(asnInfo.ASN) {
		analysis.Type = "datacenter"
		analysis.Risk = "medium"
	} else if n.isHostingASN(asnInfo.ASN) {
		analysis.Type = "hosting"
		analysis.Risk = "medium"
	} else {
		analysis.Type = "isp"
		analysis.Risk = "low"
	}

	analysis.HistoricalRisk = n.checkASNHistory(asnInfo.ASN)
	analysis.ReputationScore = n.evaluateASNReputation(asnInfo)

	return analysis
}

func (n *NetworkAnalysisEnhanced) DetectVPNConnection(ip string, timing *NetworkTimingAnalysis) *VPNDetection {
	result := &VPNDetection{}

	isVPN, provider := n.checkVPN(ip)
	if isVPN {
		result.IsVPN = true
		result.Provider = provider
		result.Confidence = 0.95
		return result
	}

	if n.checkTorExitNode(ip) {
		result.IsVPN = true
		result.Provider = "Tor"
		result.Confidence = 0.95
		return result
	}

	if timing != nil {
		result.TimeBasedDetection = n.analyzeTiming(timing)
		if result.TimeBasedDetection {
			result.Confidence += 0.3
		}
	}

	result.PortBasedDetection = n.analyzePorts(ip)
	if result.PortBasedDetection {
		result.Confidence += 0.2
	}

	result.DNSBasedDetection = n.analyzeDNS(ip)
	if result.DNSBasedDetection {
		result.Confidence += 0.3
	}

	result.IsVPN = result.Confidence > 0.7
	if result.Confidence > 0.95 {
		result.Confidence = 0.95
	}

	return result
}

func (n *NetworkAnalysisEnhanced) AssessNetworkRisk(ip string, data *NetworkRiskData) *NetworkRiskAssessment {
	assessment := &NetworkRiskAssessment{}

	assessment.IPReputation = n.AnalyzeIPReputation(ip)

	if data.Headers != nil {
		assessment.ResidentialProxy = n.DetectResidentialProxy(ip, data.Headers)
	}

	if data.Timing != nil {
		assessment.VPN = n.DetectVPNConnection(ip, data.Timing)
	}

	if assessment.IPReputation != nil && assessment.IPReputation.ASNInfo != nil {
		assessment.ASNAnalysis = n.AnalyzeASN(assessment.IPReputation.ASNInfo)
	}

	assessment.RiskLevel = n.calculateOverallRisk(assessment)
	assessment.RecommendedAction = n.getRecommendedAction(assessment.RiskLevel)

	return assessment
}

func (n *NetworkAnalysisEnhanced) checkProxy(ip string) bool {
	proxyPorts := []int{80, 8080, 3128, 8888, 1080, 8118, 8123}

	for _, port := range proxyPorts {
		addr := net.JoinHostPort(ip, string(rune(port)))
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			return true
		}
	}

	return false
}

func (n *NetworkAnalysisEnhanced) checkTorExitNode(ip string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.torExitNodes[ip]
}

func (n *NetworkAnalysisEnhanced) checkVPN(ip string) (bool, string) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for provider := range n.vpnProviders {
		if n.matchVPNRange(ip, provider) {
			return true, provider
		}
	}
	return false, ""
}

func (n *NetworkAnalysisEnhanced) matchVPNRange(ip, provider string) bool {
	vpnRanges := map[string][]string{
		"NordVPN":      {"104.248.0.0/16", "185.180.0.0/16"},
		"ExpressVPN":   {"45.33.0.0/16", "96.126.0.0/16"},
		"Surfshark":    {"212.80.0.0/16", "86.49.0.0/16"},
		"CyberGhost":   {"185.234.0.0/16", "94.130.0.0/16"},
		"PrivateInternetAccess": {"104.238.0.0/16", "209.95.0.0/16"},
	}

	ranges, ok := vpnRanges[provider]
	if !ok {
		return false
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, cidr := range ranges {
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

func (n *NetworkAnalysisEnhanced) getASNInfo(ip string) *ASNInfo {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil
	}

	asnData := n.queryASNDatabase(ip)
	if asnData != nil {
		return asnData
	}

	return &ASNInfo{
		ASN:     0,
		Provider: "Unknown",
		Country:  "Unknown",
	}
}

func (n *NetworkAnalysisEnhanced) queryASNDatabase(ip string) *ASNInfo {
	datacenterASNs := map[int]struct {
		provider string
		country  string
	}{
		12345: {"Example DC", "US"},
		67890: {"Another DC", "DE"},
		15169: {"Google LLC", "US"},
		396982: {"Google Cloud", "US"},
	}

	for asn, info := range datacenterASNs {
		if n.isIPInASN(ip, asn) {
			return &ASNInfo{
				ASN:     asn,
				Provider: info.provider,
				Country:  info.country,
			}
		}
	}

	return nil
}

func (n *NetworkAnalysisEnhanced) isIPInASN(ip string, asn int) bool {
	asnRanges := map[int][]string{
		15169:   {"8.8.8.0/24", "8.8.4.0/24", "8.34.208.0/20"},
		396982:  {"34.64.0.0/10", "104.196.0.0/14"},
		12345:   {"203.0.113.0/24", "198.51.100.0/24"},
		67890:   {"192.0.2.0/24", "198.51.100.0/24"},
	}

	ranges, ok := asnRanges[asn]
	if !ok {
		return false
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, cidr := range ranges {
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

func (n *NetworkAnalysisEnhanced) calculateReputationScore(rep *IPReputation) float64 {
	score := 100.0

	if rep.IsProxy {
		score -= 50
	}
	if rep.IsTor {
		score -= 40
	}
	if rep.IsVPN {
		score -= 30
	}

	if rep.ASNInfo != nil {
		if n.isDatacenterASN(rep.ASNInfo.ASN) {
			score -= 10
		}
		if n.checkBlacklist(rep.IP) {
			score -= 20
		}
	}

	if score < 0 {
		score = 0
	}
	return score
}

func (n *NetworkAnalysisEnhanced) classifyIPType(ip string) string {
	if n.isDatacenterASNByIP(ip) {
		return "datacenter"
	}
	if n.isMobileIP(ip) {
		return "mobile"
	}
	if n.isHostingASNByIP(ip) {
		return "hosting"
	}

	return "unknown"
}

func (n *NetworkAnalysisEnhanced) isDatacenterASNByIP(ip string) bool {
	return n.isIPInASN(ip, 12345) || n.isIPInASN(ip, 67890) || n.isIPInASN(ip, 15169) || n.isIPInASN(ip, 396982)
}

func (n *NetworkAnalysisEnhanced) isHostingASNByIP(ip string) bool {
	hostingASNs := []int{20001, 20002, 20003}
	for _, asn := range hostingASNs {
		if n.isIPInASN(ip, asn) {
			return true
		}
	}
	return false
}

func (n *NetworkAnalysisEnhanced) isMobileIP(ip string) bool {
	mobileRanges := []string{
		"64.233.160.0/19",
		"66.249.80.0/20",
		"72.14.199.0/24",
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, cidr := range mobileRanges {
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

func (n *NetworkAnalysisEnhanced) getISPInfo(ip string) string {
	if n.isDatacenterASNByIP(ip) {
		return "Cloud Provider"
	}
	if n.isHostingASNByIP(ip) {
		return "Hosting Provider"
	}
	if n.isMobileIP(ip) {
		return "Mobile Carrier"
	}

	return "Unknown ISP"
}

func (n *NetworkAnalysisEnhanced) detectPortScan(ip string) bool {
	return false
}

func (n *NetworkAnalysisEnhanced) checkBlacklist(ip string) bool {
	blacklist := map[string]bool{
		"192.0.2.1":  true,
		"198.51.100.1": true,
		"203.0.113.1": true,
	}

	n.mu.RLock()
	defer n.mu.RUnlock()
	return blacklist[ip]
}

func (n *NetworkAnalysisEnhanced) analyzeProxySignals(ip string, headers map[string]string) float64 {
	confidence := 0.0

	if headers == nil {
		return confidence
	}

	if via, ok := headers["via"]; ok && via != "" {
		if strings.Contains(strings.ToLower(via), "proxy") {
			confidence += 0.3
		}
	}

	if xff, ok := headers["x-forwarded-for"]; ok && xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 1 {
			confidence += 0.2
		}
	}

	if forwarded, ok := headers["forwarded"]; ok && forwarded != "" {
		confidence += 0.2
	}

	if xrealip, ok := headers["x-real-ip"]; ok && xrealip != "" {
		confidence += 0.15
	}

	if cfconnectingip, ok := headers["cf-connecting-ip"]; ok && cfconnectingip != "" {
		confidence += 0.1
	}

	if confidence > 1.0 {
		confidence = 1.0
	}
	return confidence
}

func (n *NetworkAnalysisEnhanced) isDatacenterASN(asn int) bool {
	datacenterASNs := map[int]bool{
		12345:   true,
		67890:   true,
		15169:   true,
		396982:  true,
		22822:   true,
		63949:   true,
		14061:   true,
		14618:   true,
		16276:   true,
		20473:   true,
		45090:   true,
	}
	return datacenterASNs[asn]
}

func (n *NetworkAnalysisEnhanced) isHostingASN(asn int) bool {
	hostingASNs := map[int]bool{
		20001: true,
		20002: true,
		20003: true,
		20004: true,
		20005: true,
	}
	return hostingASNs[asn]
}

func (n *NetworkAnalysisEnhanced) checkASNHistory(asn int) string {
	highRiskASNs := map[int]bool{
		99999: true,
	}
	if highRiskASNs[asn] {
		return "high_risk"
	}
	return "clean"
}

func (n *NetworkAnalysisEnhanced) evaluateASNReputation(asnInfo *ASNInfo) float64 {
	if asnInfo == nil {
		return 0.5
	}

	score := 0.8

	if n.isDatacenterASN(asnInfo.ASN) {
		score -= 0.2
	}
	if n.isHostingASN(asnInfo.ASN) {
		score -= 0.1
	}
	if n.checkASNHistory(asnInfo.ASN) == "high_risk" {
		score -= 0.4
	}

	if score < 0 {
		score = 0
	}
	return score
}

func (n *NetworkAnalysisEnhanced) analyzeTiming(timing *NetworkTimingAnalysis) bool {
	if timing == nil {
		return false
	}

	if timing.RTTVariance > 0.5 {
		return true
	}

	if timing.RTT > 500 {
		return true
	}

	return false
}

func (n *NetworkAnalysisEnhanced) analyzePorts(ip string) bool {
	commonVPNPorts := map[int]bool{
		1194:  true,
		1723:  true,
		500:   true,
		4500:  true,
		1701:  true,
		443:   true,
		51820: true,
	}

	for port := range commonVPNPorts {
		addr := net.JoinHostPort(ip, string(rune(port)))
		conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
		if err == nil {
			conn.Close()
			return true
		}
	}

	return false
}

func (n *NetworkAnalysisEnhanced) analyzeDNS(ip string) bool {
	return false
}

func (n *NetworkAnalysisEnhanced) calculateOverallRisk(assessment *NetworkRiskAssessment) string {
	risk := 0

	if assessment.IPReputation != nil && assessment.IPReputation.Score < 50 {
		risk += 2
	}
	if assessment.IPReputation != nil && assessment.IPReputation.Score < 30 {
		risk += 1
	}

	if assessment.ResidentialProxy != nil && assessment.ResidentialProxy.IsResidentialProxy {
		risk += 3
	}

	if assessment.VPN != nil && assessment.VPN.IsVPN {
		risk += 2
	}

	if assessment.ASNAnalysis != nil {
		switch assessment.ASNAnalysis.Risk {
		case "high":
			risk += 2
		case "medium":
			risk += 1
		}
	}

	if risk >= 5 {
		return "critical"
	} else if risk >= 3 {
		return "high"
	} else if risk >= 2 {
		return "medium"
	}
	return "low"
}

func (n *NetworkAnalysisEnhanced) getRecommendedAction(riskLevel string) string {
	switch riskLevel {
	case "critical":
		return "block"
	case "high":
		return "challenge"
	case "medium":
		return "additional_verification"
	default:
		return "allow"
	}
}

func loadTorExitNodes() map[string]bool {
	torExitNodes := make(map[string]bool)

	torExitNodes["23.129.64.0/24"] = true
	torExitNodes["45.33.32.0/24"] = true
	torExitNodes["128.31.0.0/24"] = true
	torExitNodes["131.188.40.0/24"] = true
	torExitNodes["193.218.244.0/24"] = true
	torExitNodes["199.249.230.0/24"] = true
	torExitNodes["204.13.164.0/24"] = true
	torExitNodes["209.141.32.0/24"] = true
	torExitNodes["212.51.134.0/24"] = true
	torExitNodes["216.218.134.0/24"] = true

	return torExitNodes
}

func loadVPNProviders() map[string]bool {
	return map[string]bool{
		"NordVPN":                true,
		"ExpressVPN":             true,
		"Surfshark":              true,
		"CyberGhost":             true,
		"PrivateInternetAccess":  true,
		"ProtonVPN":              true,
		"HotspotShield":          true,
		"IPVanish":               true,
		"VyprVPN":                true,
		"Mullvad":                true,
		"Windscribe":             true,
		"TunnelBear":             true,
		"Browsec":                true,
		"HolaVPN":                true,
	}
}

type IPReputation struct {
	IP          string
	IsProxy     bool
	IsTor       bool
	IsVPN       bool
	VPNProvider string
	ASNInfo     *ASNInfo
	Score       float64
	Timestamp   time.Time
}

type ASNInfo struct {
	ASN     int
	Provider string
	Country  string
}

type ResidentialProxyResult struct {
	IsResidentialProxy bool
	Confidence         float64
	IPType            string
	ISP               string
	PortScanDetected  bool
	InBlacklist       bool
}

type ASNAnalysis struct {
	ASN             int
	Provider        string
	Type            string
	Risk            string
	HistoricalRisk  string
	ReputationScore float64
}

type VPNDetection struct {
	IsVPN              bool
	Provider           string
	Confidence         float64
	TimeBasedDetection bool
	PortBasedDetection bool
	DNSBasedDetection  bool
}

type NetworkTimingAnalysis struct {
	RTT         float64
	RTTVariance float64
}

type NetworkRiskData struct {
	IP      string
	Headers map[string]string
	Timing  *NetworkTimingAnalysis
}

type NetworkRiskAssessment struct {
	IPReputation      *IPReputation
	ResidentialProxy  *ResidentialProxyResult
	VPN               *VPNDetection
	ASNAnalysis       *ASNAnalysis
	RiskLevel         string
	RecommendedAction string
}

func (n *NetworkAnalysisEnhanced) ExportReputationData(ip string) ([]byte, error) {
	reputation := n.AnalyzeIPReputation(ip)
	return json.MarshalIndent(reputation, "", "  ")
}

func (n *NetworkAnalysisEnhanced) ExportRiskAssessment(ip string, data *NetworkRiskData) ([]byte, error) {
	assessment := n.AssessNetworkRisk(ip, data)
	return json.MarshalIndent(assessment, "", "  ")
}

func (n *NetworkAnalysisEnhanced) ClearCache() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.ipReputationCache = make(map[string]*IPReputation)
}

func (n *NetworkAnalysisEnhanced) GetCacheSize() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.ipReputationCache)
}

func (n *NetworkAnalysisEnhanced) SetCacheTTL(ttl time.Duration) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.cacheTTL = ttl
}
