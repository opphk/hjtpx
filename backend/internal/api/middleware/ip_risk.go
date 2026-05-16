package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type IPRiskLevel string

const (
	IPRiskLevelSafe   IPRiskLevel = "safe"
	IPRiskLevelLow    IPRiskLevel = "low"
	IPRiskLevelMedium IPRiskLevel = "medium"
	IPRiskLevelHigh   IPRiskLevel = "high"
	IPRiskLevelCritical IPRiskLevel = "critical"
)

type IPRiskInfo struct {
	IP           string      `json:"ip"`
	RiskLevel    IPRiskLevel `json:"risk_level"`
	Score        int         `json:"score"`
	IsProxy      bool        `json:"is_proxy"`
	IsVPN        bool        `json:"is_vpn"`
	IsTor        bool        `json:"is_tor"`
	IsHosting    bool        `json:"is_hosting"`
	IsDatacenter bool        `json:"is_datacenter"`
	IsMobile     bool        `json:"is_mobile"`
	Country      string      `json:"country,omitempty"`
	ASN          string      `json:"asn,omitempty"`
	ISP          string      `json:"isp,omitempty"`
	Org          string      `json:"org,omitempty"`
	Reasons      []string    `json:"reasons"`
	CheckedAt    time.Time   `json:"checked_at"`
}

type IPRiskConfig struct {
	EnableProxyDetection    bool
	EnableVPNDetection      bool
	EnableTorDetection     bool
	EnableHostingDetection bool
	EnableDatacenterCheck  bool
	ProxyThreshold         int
	VPNThreshold           int
	CheckTimeout           time.Duration
	CacheTTL               time.Duration
	BlockHighRisk          bool
	BlockCriticalRisk      bool
	WarnMediumRisk         bool
	AllowedCountries       []string
	BlockedCountries       []string
	AllowedASNs            []string
	BlockedASNs            []string
	ExcludePaths           []string
}

var defaultIPRiskConfig = IPRiskConfig{
	EnableProxyDetection:    true,
	EnableVPNDetection:      true,
	EnableTorDetection:     true,
	EnableHostingDetection: true,
	EnableDatacenterCheck:  true,
	ProxyThreshold:         50,
	VPNThreshold:          30,
	CheckTimeout:          5 * time.Second,
	CacheTTL:              1 * time.Hour,
	BlockHighRisk:         false,
	BlockCriticalRisk:     true,
	WarnMediumRisk:        true,
	AllowedCountries:      []string{},
	BlockedCountries:      []string{},
	AllowedASNs:           []string{},
	BlockedASNs:           []string{},
	ExcludePaths:          []string{"/health", "/api/health", "/metrics", "/api/metrics"},
}

var knownVPNPorts = []int{
	1723, 500, 4500, 1701, 443, 1194, 51820, 8472, 4433,
}

var knownProxyHeaders = []string{
	"X-Forwarded-For",
	"X-Real-IP",
	"X-Cluster-Client-IP",
	"Forwarded",
	"Via",
	"Max-Forwards",
}

var torExitNodes = []string{
	"torproject.org",
	"torbrowser.com",
}

var datacenterASNs = map[string]bool{
	"AS15169": true,
	"AS8075":  true,
	"AS396982": true,
	"AS400654": true,
	"AS209242": true,
}

var hostingProviderPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(aws|amazon|ec2|lightsail)`),
	regexp.MustCompile(`(?i)(google cloud|gcp|compute engine)`),
	regexp.MustCompile(`(?i)(azure|microsoft|cloudapp)`),
	regexp.MustCompile(`(?i)(digitalocean|droplet)`),
	regexp.MustCompile(`(?i)(linode|vps|linode)`),
	regexp.MustCompile(`(?i)(vultr|virtualizor)`),
	regexp.MustCompile(`(?i)(ovh|soyoustart|kimsufi)`),
	regexp.MustCompile(`(?i)(hetzner|robot)`),
	regexp.MustCompile(`(?i)(contabo)`),
	regexp.MustCompile(`(?i)(aliyun|alibaba cloud|aliyunecs)`),
	regexp.MustCompile(`(?i)(tencent cloud|qcloud)`),
}

var vpnKeywords = []string{
	"vpn",
	"nord",
	"expressvpn",
	"surfshark",
	"cyberghost",
	"ipvanish",
	"private internet access",
	"hotspot shield",
	"protonvpn",
	"mullvad",
	"windscribe",
	"tunnelbear",
	"purevpn",
	"hide my ass",
	"privatevpn",
	"vyprvpn",
	"atlas vpn",
}

var torKeywords = []string{
	"tor",
	"onion",
	"torproject",
	"tor2web",
	"tor exit",
}

var ipRiskCache = &IPRiskCache{
	cache: make(map[string]*IPRiskInfo),
	mu:    sync.RWMutex{},
}

type IPRiskCache struct {
	cache map[string]*IPRiskInfo
	mu    sync.RWMutex
}

func (c *IPRiskCache) Get(ip string) (*IPRiskInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	info, exists := c.cache[ip]
	return info, exists
}

func (c *IPRiskCache) Set(ip string, info *IPRiskInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[ip] = info
}

func (c *IPRiskCache) Delete(ip string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, ip)
}

func (c *IPRiskCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*IPRiskInfo)
}

func (c *IPRiskCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

func init() {
	go ipRiskCache.cleanup()
}

func (c *IPRiskCache) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for ip, info := range c.cache {
			if now.Sub(info.CheckedAt) > 1*time.Hour {
				delete(c.cache, ip)
			}
		}
		c.mu.Unlock()
	}
}

func detectProxyHeaders(c *gin.Context) bool {
	for _, header := range knownProxyHeaders {
		if c.GetHeader(header) != "" {
			return true
		}
	}

	forwarded := c.GetHeader("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 2 {
			return true
		}
	}

	return false
}

func checkOpenPorts(ip string, timeout time.Duration) (int, bool) {
	detectedPorts := 0
	var wg sync.WaitGroup
	var mu sync.Mutex

	commonProxyPorts := []int{80, 443, 8080, 3128, 8888, 8118, 9050, 9051}

	for _, port := range commonProxyPorts {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			address := net.JoinHostPort(ip, strconv.Itoa(p))
			conn, err := net.DialTimeout("tcp", address, timeout/2)
			if err == nil {
				conn.Close()
				mu.Lock()
				detectedPorts++
				mu.Unlock()
			}
		}(port)
	}

	wg.Wait()
	return detectedPorts, detectedPorts >= 2
}

func reverseDNSLookup(ip string, timeout time.Duration) (string, error) {
	resolver := &net.Resolver{
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: timeout}
			return d.DialContext(ctx, network, "8.8.8.8:53")
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	names, err := resolver.LookupAddr(ctx, ip)
	if err != nil || len(names) == 0 {
		return "", err
	}

	return names[0], nil
}

func checkReverseDNS(rdns string) IPRiskInfo {
	info := IPRiskInfo{}

	info.IsVPN = false
	info.IsTor = false
	info.IsHosting = false
	info.IsDatacenter = false

	rdnsLower := strings.ToLower(rdns)

	for _, keyword := range vpnKeywords {
		if strings.Contains(rdnsLower, keyword) {
			info.IsVPN = true
			info.Reasons = append(info.Reasons, fmt.Sprintf("VPN keyword detected: %s", keyword))
		}
	}

	for _, keyword := range torKeywords {
		if strings.Contains(rdnsLower, keyword) {
			info.IsTor = true
			info.Reasons = append(info.Reasons, fmt.Sprintf("Tor keyword detected: %s", keyword))
		}
	}

	for _, pattern := range hostingProviderPatterns {
		if pattern.MatchString(rdnsLower) {
			info.IsHosting = true
			info.Reasons = append(info.Reasons, "Hosting provider detected via reverse DNS")
			break
		}
	}

	for asn := range datacenterASNs {
		if strings.Contains(rdnsLower, strings.ToLower(asn)) {
			info.IsDatacenter = true
			info.Reasons = append(info.Reasons, "Datacenter ASN detected")
			break
		}
	}

	return info
}

func calculateRiskScore(info *IPRiskInfo, config *IPRiskConfig) int {
	score := 0

	if info.IsTor {
		score += 50
	}

	if info.IsVPN {
		score += 30
	}

	if info.IsDatacenter {
		score += 20
	}

	if info.IsHosting {
		score += 15
	}

	if info.IsProxy {
		score += 25
	}

	return score
}

func determineRiskLevel(score int) IPRiskLevel {
	switch {
	case score >= 70:
		return IPRiskLevelCritical
	case score >= 50:
		return IPRiskLevelHigh
	case score >= 30:
		return IPRiskLevelMedium
	case score >= 10:
		return IPRiskLevelLow
	default:
		return IPRiskLevelSafe
	}
}

func AssessIPRisk(c *gin.Context, config ...IPRiskConfig) *IPRiskInfo {
	cfg := defaultIPRiskConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	ip := c.ClientIP()
	if ip == "" {
		ip = c.GetHeader("X-Forwarded-For")
		if ip == "" {
			ip = c.GetHeader("X-Real-IP")
		}
	}

	if ip == "" {
		return &IPRiskInfo{
			IP:        "unknown",
			RiskLevel: IPRiskLevelSafe,
			Score:     0,
			Reasons:   []string{"Unable to determine client IP"},
			CheckedAt: time.Now(),
		}
	}

	if cached, exists := ipRiskCache.Get(ip); exists {
		return cached
	}

	info := &IPRiskInfo{
		IP:        ip,
		RiskLevel: IPRiskLevelSafe,
		Score:     0,
		Reasons:   []string{},
		CheckedAt: time.Now(),
	}

	if cfg.EnableProxyDetection {
		info.IsProxy = detectProxyHeaders(c)
		if info.IsProxy {
			info.Reasons = append(info.Reasons, "Proxy headers detected")
		}

		if cfg.EnableDatacenterCheck || cfg.EnableHostingDetection {
			rdns, err := reverseDNSLookup(ip, cfg.CheckTimeout)
			if err == nil && rdns != "" {
				rdnsInfo := checkReverseDNS(rdns)
				info.IsVPN = rdnsInfo.IsVPN
				info.IsTor = rdnsInfo.IsTor
				info.IsHosting = rdnsInfo.IsHosting
				info.IsDatacenter = rdnsInfo.IsDatacenter
				info.Reasons = append(info.Reasons, rdnsInfo.Reasons...)
				info.Org = strings.TrimSpace(rdns)
			}
		}
	}

	info.Score = calculateRiskScore(info, &cfg)
	info.RiskLevel = determineRiskLevel(info.Score)

	ipRiskCache.Set(ip, info)

	return info
}

func IPRiskAssessmentMiddleware(config ...IPRiskConfig) gin.HandlerFunc {
	cfg := defaultIPRiskConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		isExcluded := false
		for _, excluded := range cfg.ExcludePaths {
			if path == excluded || strings.HasPrefix(path, excluded+"/") {
				isExcluded = true
				break
			}
		}
		if isExcluded {
			c.Next()
			return
		}

		riskInfo := AssessIPRisk(c, cfg)

		c.Set("ip_risk_info", riskInfo)
		c.Set("ip_risk_level", riskInfo.RiskLevel)

		if cfg.BlockCriticalRisk && riskInfo.RiskLevel == IPRiskLevelCritical {
			c.Header("X-IP-Risk-Level", string(riskInfo.RiskLevel))
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "访问被拒绝，检测到高风险IP地址",
				"error":   "ip_risk_critical",
				"risk":    riskInfo,
			})
			c.Abort()
			return
		}

		if cfg.BlockHighRisk && riskInfo.RiskLevel == IPRiskLevelHigh {
			c.Header("X-IP-Risk-Level", string(riskInfo.RiskLevel))
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "访问被拒绝，检测到风险IP地址",
				"error":   "ip_risk_high",
				"risk":    riskInfo,
			})
			c.Abort()
			return
		}

		if cfg.WarnMediumRisk && riskInfo.RiskLevel == IPRiskLevelMedium {
			c.Header("X-IP-Risk-Level", string(riskInfo.RiskLevel))
			c.Header("X-IP-Risk-Warning", "medium")
		}

		c.Next()
	}
}

func GetIPRiskInfo(c *gin.Context) *IPRiskInfo {
	if info, exists := c.Get("ip_risk_info"); exists {
		if riskInfo, ok := info.(*IPRiskInfo); ok {
			return riskInfo
		}
	}
	return nil
}

func GetIPRiskLevel(c *gin.Context) IPRiskLevel {
	if level, exists := c.Get("ip_risk_level"); exists {
		if riskLevel, ok := level.(IPRiskLevel); ok {
			return riskLevel
		}
	}
	return IPRiskLevelSafe
}

func IPRiskAlertMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		riskInfo := GetIPRiskInfo(c)
		if riskInfo == nil {
			return
		}

		if riskInfo.RiskLevel == IPRiskLevelHigh || riskInfo.RiskLevel == IPRiskLevelCritical {
			securityLog := GetSecurityLog()
			securityLog.Log(SecurityEvent{
				EventType:   EventSuspiciousActivity,
				Level:       LevelHigh,
				ClientIP:    riskInfo.IP,
				Path:        c.Request.URL.Path,
				Method:      c.Request.Method,
				UserAgent:   c.GetHeader("User-Agent"),
				Description: fmt.Sprintf("High risk IP detected: %s", riskInfo.RiskLevel),
				Details:     fmt.Sprintf("Score: %d, Reasons: %v", riskInfo.Score, riskInfo.Reasons),
				IsBlocked:   false,
			})
		}
	}
}

func IPRateLimitByRiskMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		riskInfo := GetIPRiskInfo(c)
		if riskInfo == nil {
			c.Next()
			return
		}

		baseLimit := 100
		windowSecs := 60

		switch riskInfo.RiskLevel {
		case IPRiskLevelCritical:
			baseLimit = 5
			windowSecs = 300
		case IPRiskLevelHigh:
			baseLimit = 20
			windowSecs = 120
		case IPRiskLevelMedium:
			baseLimit = 50
			windowSecs = 60
		case IPRiskLevelLow:
			baseLimit = 80
			windowSecs = 60
		case IPRiskLevelSafe:
			baseLimit = 100
			windowSecs = 60
		}

		ip := c.ClientIP()
		if ip == "" {
			ip = c.GetHeader("X-Forwarded-For")
			if ip == "" {
				ip = c.GetHeader("X-Real-IP")
			}
		}

		if ip == "" {
			c.Next()
			return
		}

		config := &service.RateLimitConfig{
			MaxRequests: baseLimit,
			WindowSecs:  windowSecs,
		}

		result, err := service.NewRateLimitService().CheckIPRateLimit(c.Request.Context(), ip, config)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(baseLimit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))
		c.Header("X-IP-Risk-Level", string(riskInfo.RiskLevel))

		if !result.Allowed {
			c.Header("Retry-After", strconv.Itoa(windowSecs))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "请求过于频繁，请稍后再试",
				"error":   "rate_limit_exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func RecordIPRiskViolation(c *gin.Context, identifier string, riskLevel IPRiskLevel) {
	if redis.Client == nil {
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("ip_risk:violation:%s:%s", riskLevel, identifier)
	redis.Client.Incr(ctx, key)
	redis.Client.Expire(ctx, key, 24*time.Hour)
}

func GetIPRiskStats() map[string]interface{} {
	stats := map[string]interface{}{
		"cache_size":      ipRiskCache.Size(),
		"total_checked":   0,
		"by_risk_level": map[string]int{
			"safe":     0,
			"low":      0,
			"medium":   0,
			"high":     0,
			"critical": 0,
		},
	}

	ipRiskCache.mu.RLock()
	for _, info := range ipRiskCache.cache {
		stats["total_checked"] = int(stats["total_checked"].(int)) + 1
		level := string(info.RiskLevel)
		stats["by_risk_level"].(map[string]int)[level]++
	}
	ipRiskCache.mu.RUnlock()

	return stats
}

func ClearIPRiskCache() {
	ipRiskCache.Clear()
}
