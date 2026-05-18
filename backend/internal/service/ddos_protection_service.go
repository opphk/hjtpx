package service

import (
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type DDoSCheckResult struct {
	Allowed           bool
	Reason            string
	IPStats           *IPStatistics
	RetryAfter        int
	AttackType        string
	SuggestedAction   string
}

type IPStatistics struct {
	IP            string
	RequestCount  int
	BlockedCount  int
	FirstSeen     time.Time
	LastSeen      time.Time
	Rate          float64
	IsAnomaly     bool
	IsBlacklisted bool
	Reputation    int
	Score         float64
	CountryCode   string
	ASN           string
}

type DDoSTrafficData struct {
	RequestTimes []time.Time
	RequestSizes []int
	Methods      []string
	Paths        []string
	UserAgents   []string
	Referrers    []string
}

type IPReputation struct {
	IP           string
	Score        int
	CountryCode  string
	ASN          string
	IsTorExit    bool
	IsVPN        bool
	IsProxy      bool
	ThreatLevel  string
	LastUpdated  time.Time
}

type DDoSProtectionService struct {
	ipStats          map[string]*IPStatistics
	trafficData      map[string]*DDoSTrafficData
	blacklist        map[string]time.Time
	whitelist        map[string]bool
	ipReputations    map[string]*IPReputation
	globalRequestCount int64
	globalLastReset    time.Time
	mu               sync.RWMutex
	maxIPs           int
	requestsPerMin   int
	cleanupPeriod    time.Duration
	enableAdvancedDetection bool
	botPatterns      []*regexp.Regexp
	suspiciousUA     []string
	attackThreshold  float64
}

func NewDDoSProtectionService() *DDoSProtectionService {
	service := &DDoSProtectionService{
		ipStats:             make(map[string]*IPStatistics),
		trafficData:         make(map[string]*DDoSTrafficData),
		blacklist:           make(map[string]time.Time),
		whitelist:           make(map[string]bool),
		ipReputations:       make(map[string]*IPReputation),
		maxIPs:              10000,
		requestsPerMin:      60,
		cleanupPeriod:       1 * time.Hour,
		enableAdvancedDetection: true,
		attackThreshold:     0.8,
		botPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)bot|crawler|spider|scraper|curl|wget|python-requests`),
			regexp.MustCompile(`(?i)Googlebot|Bingbot|Yahoo|Baidu|Sogou|Yandex`),
			regexp.MustCompile(`(?i)Ahrefs|Semrush|Moz|Majestic|Seznam`),
		},
		suspiciousUA: []string{
			"Mozilla/5.0 (compatible; MSIE 6.0; Windows NT 5.1)",
			"Mozilla/4.0 (compatible; MSIE 7.0; Windows NT 5.1)",
			"curl/7.",
			"Wget/",
			"python-requests/",
			"Go-http-client/",
			"Java/",
			"Jakarta Commons-HttpClient/",
		},
	}
	go service.cleanupLoop()
	go service.globalRateMonitor()
	return service
}

func (s *DDoSProtectionService) CheckRequest(r *http.Request) *DDoSCheckResult {
	ip := getClientIP(r)
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.whitelist[ip] {
		return &DDoSCheckResult{
			Allowed: true,
			Reason:  "whitelisted",
		}
	}

	if expiry, exists := s.blacklist[ip]; exists {
		if now.Before(expiry) {
			return &DDoSCheckResult{
				Allowed:         false,
				Reason:          "blacklisted",
				RetryAfter:      int(time.Until(expiry).Seconds()),
				AttackType:      "Blacklisted IP",
				SuggestedAction: "Review and consider permanent block",
			}
		}
		delete(s.blacklist, ip)
	}

	reputation := s.getIPReputation(ip)
	if reputation.ThreatLevel == "critical" {
		s.blacklist[ip] = now.Add(24 * time.Hour)
		return &DDoSCheckResult{
			Allowed:         false,
			Reason:          "high_risk_ip",
			AttackType:      "Malicious IP",
			SuggestedAction: "Permanent block recommended",
		}
	}

	if s.isBot(r.UserAgent()) {
		return &DDoSCheckResult{
			Allowed:         false,
			Reason:          "bot_detected",
			AttackType:      "Automated Bot",
			SuggestedAction: "Add to bot blacklist",
		}
	}

	stats, exists := s.ipStats[ip]
	if !exists {
		stats = &IPStatistics{
			IP:           ip,
			RequestCount: 0,
			BlockedCount: 0,
			FirstSeen:    now,
			LastSeen:     now,
			Reputation:   reputation.Score,
			CountryCode:  reputation.CountryCode,
			ASN:          reputation.ASN,
		}
		s.ipStats[ip] = stats
		if len(s.ipStats) > s.maxIPs {
			s.cleanupOldIPs()
		}
	}

	traffic, exists := s.trafficData[ip]
	if !exists {
		traffic = &DDoSTrafficData{
			RequestTimes: []time.Time{},
			RequestSizes: []int{},
			Methods:      []string{},
			Paths:        []string{},
			UserAgents:   []string{},
			Referrers:    []string{},
		}
		s.trafficData[ip] = traffic
	}

	traffic.RequestTimes = append(traffic.RequestTimes, now)
	traffic.UserAgents = append(traffic.UserAgents, r.UserAgent())
	traffic.Methods = append(traffic.Methods, r.Method)
	traffic.Paths = append(traffic.Paths, r.URL.Path)

	if len(traffic.UserAgents) > 500 {
		traffic.UserAgents = traffic.UserAgents[len(traffic.UserAgents)-500:]
	}
	if len(traffic.Methods) > 500 {
		traffic.Methods = traffic.Methods[len(traffic.Methods)-500:]
	}
	if len(traffic.Paths) > 500 {
		traffic.Paths = traffic.Paths[len(traffic.Paths)-500:]
	}

	stats.RequestCount++
	stats.LastSeen = now

	cutoff := now.Add(-1 * time.Minute)
	recentRequests := 0
	for _, t := range traffic.RequestTimes {
		if t.After(cutoff) {
			recentRequests++
		}
	}
	stats.Rate = float64(recentRequests)

	if len(traffic.RequestTimes) > 1000 {
		traffic.RequestTimes = traffic.RequestTimes[len(traffic.RequestTimes)-1000:]
	}

	anomalyResult := s.advancedAnomalyDetection(traffic, stats)
	stats.IsAnomaly = anomalyResult.IsAnomaly
	stats.Score = anomalyResult.Score

	adjustedLimit := s.getDynamicRateLimit(float64(reputation.Score), stats.Score)
	if stats.Rate > float64(adjustedLimit) {
		stats.BlockedCount++
		return &DDoSCheckResult{
			Allowed:         false,
			Reason:          "rate_limit",
			IPStats:         stats,
			AttackType:      "Rate Limit Exceeded",
			SuggestedAction: "Increase rate limit or investigate",
		}
	}

	if stats.IsAnomaly {
		stats.BlockedCount++
		return &DDoSCheckResult{
			Allowed:         false,
			Reason:          "anomaly_detected",
			IPStats:         stats,
			AttackType:      anomalyResult.AttackType,
			SuggestedAction: anomalyResult.SuggestedAction,
		}
	}

	return &DDoSCheckResult{
		Allowed: true,
		IPStats: stats,
	}
}

type DDoSAnomalyResult struct {
	IsAnomaly       bool
	Score           float64
	AttackType      string
	SuggestedAction string
}

func (s *DDoSProtectionService) advancedAnomalyDetection(traffic *DDoSTrafficData, stats *IPStatistics) DDoSAnomalyResult {
	result := DDoSAnomalyResult{
		IsAnomaly: false,
		Score:     0.0,
	}

	if len(traffic.RequestTimes) < 5 {
		return result
	}

	intervals := make([]float64, 0, len(traffic.RequestTimes)-1)
	for i := 1; i < len(traffic.RequestTimes); i++ {
		interval := traffic.RequestTimes[i].Sub(traffic.RequestTimes[i-1]).Milliseconds()
		intervals = append(intervals, float64(interval))
	}

	mean := 0.0
	for _, i := range intervals {
		mean += i
	}
	mean /= float64(len(intervals))

	variance := 0.0
	for _, i := range intervals {
		variance += math.Pow(i-mean, 2)
	}
	variance /= float64(len(intervals))
	stdDev := math.Sqrt(variance)

	cv := stdDev / mean

	if cv < 0.05 && mean < 500 {
		result.IsAnomaly = true
		result.Score += 0.4
		result.AttackType = "Automated Request Pattern"
	}

	if mean < 50 && len(traffic.RequestTimes) > 30 {
		result.IsAnomaly = true
		result.Score += 0.3
		if result.AttackType == "" {
			result.AttackType = "Fast Request Rate"
		}
	}

	uniqueUA := s.countUnique(traffic.UserAgents)
	if uniqueUA == 1 && len(traffic.UserAgents) > 20 {
		result.IsAnomaly = true
		result.Score += 0.2
		if result.AttackType == "" {
			result.AttackType = "Single User Agent Flood"
		}
	}

	uniquePaths := s.countUnique(traffic.Paths)
	if uniquePaths == 1 && len(traffic.Paths) > 20 {
		result.IsAnomaly = true
		result.Score += 0.2
		if result.AttackType == "" {
			result.AttackType = "Single Path Flood"
		}
	}

	if stats.BlockedCount > 3 {
		result.IsAnomaly = true
		result.Score += 0.3
		if result.AttackType == "" {
			result.AttackType = "Repeat Offender"
		}
	}

	if result.Score >= s.attackThreshold {
		result.IsAnomaly = true
		result.SuggestedAction = "Investigate and consider blocking"
	}

	return result
}

func (s *DDoSProtectionService) countUnique(items []string) int {
	seen := make(map[string]bool)
	for _, item := range items {
		seen[item] = true
	}
	return len(seen)
}

func (s *DDoSProtectionService) getIPReputation(ip string) *IPReputation {
	if rep, exists := s.ipReputations[ip]; exists {
		return rep
	}

	rep := &IPReputation{
		IP:          ip,
		Score:       50,
		CountryCode: "Unknown",
		ASN:         "Unknown",
		ThreatLevel: "low",
		LastUpdated: time.Now(),
	}

	if strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "192.168.") {
		rep.Score = 80
		rep.ThreatLevel = "low"
	} else if strings.HasPrefix(ip, "172.") {
		rep.Score = 75
		rep.ThreatLevel = "low"
	}

	s.ipReputations[ip] = rep
	return rep
}

func (s *DDoSProtectionService) isBot(userAgent string) bool {
	if userAgent == "" {
		return true
	}

	for _, pattern := range s.botPatterns {
		if pattern.MatchString(userAgent) {
			return true
		}
	}

	for _, suspicious := range s.suspiciousUA {
		if strings.Contains(userAgent, suspicious) {
			return true
		}
	}

	return false
}

func (s *DDoSProtectionService) getDynamicRateLimit(reputationScore, anomalyScore float64) int {
	baseLimit := s.requestsPerMin
	
	if reputationScore < 30 {
		baseLimit = int(float64(baseLimit) * 0.3)
	} else if reputationScore < 50 {
		baseLimit = int(float64(baseLimit) * 0.6)
	}

	if anomalyScore > 0.5 {
		baseLimit = int(float64(baseLimit) * 0.5)
	}

	return baseLimit
}

func (s *DDoSProtectionService) AddToWhitelist(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.whitelist[ip] = true
}

func (s *DDoSProtectionService) RemoveFromWhitelist(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.whitelist, ip)
}

func (s *DDoSProtectionService) globalRateMonitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		s.globalRequestCount = 0
		s.globalLastReset = time.Now()
		s.mu.Unlock()
	}
}

func (s *DDoSProtectionService) GetGlobalStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"total_ips":         len(s.ipStats),
		"blacklist_count":   len(s.blacklist),
		"whitelist_count":   len(s.whitelist),
		"active_since":      s.globalLastReset,
	}
}

func (s *DDoSProtectionService) AddToBlacklist(ip string, reason string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blacklist[ip] = time.Now().Add(duration)
}

func (s *DDoSProtectionService) RemoveFromBlacklist(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.blacklist, ip)
}

func (s *DDoSProtectionService) GetIPStats(ip string) *IPStatistics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ipStats[ip]
}

func (s *DDoSProtectionService) cleanupOldIPs() {
	cutoff := time.Now().Add(-24 * time.Hour)
	for ip, stats := range s.ipStats {
		if stats.LastSeen.Before(cutoff) {
			delete(s.ipStats, ip)
			delete(s.trafficData, ip)
		}
	}
}

func (s *DDoSProtectionService) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		s.cleanupOldIPs()
		now := time.Now()
		for ip, expiry := range s.blacklist {
			if now.After(expiry) {
				delete(s.blacklist, ip)
			}
		}
		s.mu.Unlock()
	}
}
