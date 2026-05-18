package service

import (
	"context"
	"fmt"
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
	RiskScore         float64
	Confidence        float64
	DetectionMethods  []string
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
	AttackHistory []AttackEvent
}

type AttackEvent struct {
	Timestamp   time.Time
	AttackType  string
	Blocked     bool
	Score       float64
}

type DDoSTrafficData struct {
	RequestTimes []time.Time
	RequestSizes []int
	Methods      []string
	Paths        []string
	UserAgents   []string
	Referrers    []string
	StatusCodes  []int
}

type IPReputation struct {
	IP           string
	Score        int
	CountryCode  string
	ASN          string
	IsTorExit    bool
	IsVPN        bool
	IsProxy      bool
	IsDatacenter bool
	ThreatLevel  string
	LastUpdated  time.Time
	Source       string
}

type DDoSProtectionService struct {
	ipStats                      map[string]*IPStatistics
	trafficData                  map[string]*DDoSTrafficData
	blacklist                    map[string]time.Time
	whitelist                    map[string]bool
	ipReputations                map[string]*IPReputation
	globalRequestCount           int64
	globalLastReset              time.Time
	mu                           sync.RWMutex
	maxIPs                       int
	requestsPerMin               int
	cleanupPeriod                time.Duration
	enableAdvancedDetection      bool
	botPatterns                  []*regexp.Regexp
	suspiciousUA                 []string
	attackThreshold              float64
	ipReputationCache            map[string]*IPReputation
	networkDetection             *EnhancedNetworkDetection
	rateLimiters                 map[string]*TokenBucket
	requestWindow                time.Duration
	maxWindowRequests            int
	enableAdaptiveRateLimiting   bool
	enableIPReputation           bool
	enableBotDetection           bool
	enableBehaviorAnalysis       bool
	enableGlobalRateLimit        bool
	globalRateLimit              int
	attackModeActive             bool
	attackModeStartTime          time.Time
}

func NewTokenBucket(maxTokens int, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:       float64(maxTokens),
		maxTokens:    float64(maxTokens),
		capacity:     int64(maxTokens),
		refillRate:   refillRate,
		lastRefill:   time.Now(),
	}
}

func (tb *TokenBucket) Take(count int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens = math.Min(tb.maxTokens, tb.tokens+elapsed*tb.refillRate)
	tb.lastRefill = now

	if tb.tokens >= float64(count) {
		tb.tokens -= float64(count)
		return true
	}
	return false
}

func NewDDoSProtectionService() *DDoSProtectionService {
	service := &DDoSProtectionService{
		ipStats:                make(map[string]*IPStatistics),
		trafficData:            make(map[string]*DDoSTrafficData),
		blacklist:              make(map[string]time.Time),
		whitelist:              make(map[string]bool),
		ipReputations:          make(map[string]*IPReputation),
		ipReputationCache:      make(map[string]*IPReputation),
		maxIPs:                 100000,
		requestsPerMin:         60,
		cleanupPeriod:          1 * time.Hour,
		enableAdvancedDetection: true,
		attackThreshold:        0.75,
		botPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)bot|crawler|spider|scraper|curl|wget|python-requests`),
			regexp.MustCompile(`(?i)Googlebot|Bingbot|Yahoo|Baidu|Sogou|Yandex`),
			regexp.MustCompile(`(?i)Ahrefs|Semrush|Moz|Majestic|Seznam`),
			regexp.MustCompile(`(?i)selenium|phantomjs|puppeteer|playwright|headless`),
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
			"node-fetch/",
			"axios/",
			"Scrapy/",
		},
		rateLimiters:               make(map[string]*TokenBucket),
		requestWindow:              1 * time.Minute,
		maxWindowRequests:          100,
		enableAdaptiveRateLimiting: true,
		enableIPReputation:         true,
		enableBotDetection:         true,
		enableBehaviorAnalysis:     true,
		enableGlobalRateLimit:      true,
		globalRateLimit:            10000,
		networkDetection:           NewEnhancedNetworkDetection(),
	}

	go service.cleanupLoop()
	go service.globalRateMonitor()
	go service.reputationUpdateLoop()
	go service.attackModeMonitor()

	return service
}

func (s *DDoSProtectionService) CheckRequest(r *http.Request) *DDoSCheckResult {
	ip := getClientIP(r)
	now := time.Now()
	result := &DDoSCheckResult{
		DetectionMethods: make([]string, 0),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.whitelist[ip] {
		result.Allowed = true
		result.Reason = "whitelisted"
		return result
	}

	if expiry, exists := s.blacklist[ip]; exists {
		if now.Before(expiry) {
			result.Allowed = false
			result.Reason = "blacklisted"
			result.RetryAfter = int(time.Until(expiry).Seconds())
			result.AttackType = "Blacklisted IP"
			result.SuggestedAction = "Review and consider permanent block"
			result.DetectionMethods = append(result.DetectionMethods, "ip_blacklist")
			return result
		}
		delete(s.blacklist, ip)
	}

	if s.enableIPReputation {
		reputation := s.getIPReputation(ip, r)
		result.DetectionMethods = append(result.DetectionMethods, "ip_reputation")

		if reputation.ThreatLevel == "critical" {
			s.blacklist[ip] = now.Add(24 * time.Hour)
			result.Allowed = false
			result.Reason = "high_risk_ip"
			result.AttackType = "Malicious IP"
			result.SuggestedAction = "Permanent block recommended"
			result.RiskScore = 95.0
			result.Confidence = 0.95
			return result
		}

		if reputation.ThreatLevel == "high" {
			result.DetectionMethods = append(result.DetectionMethods, "high_risk")
			result.RiskScore += 40
			result.Confidence += 0.3
		}
	}

	if s.enableBotDetection && s.isBot(r.UserAgent()) {
		result.Allowed = false
		result.Reason = "bot_detected"
		result.AttackType = "Automated Bot"
		result.SuggestedAction = "Add to bot blacklist"
		result.DetectionMethods = append(result.DetectionMethods, "bot_detection")
		result.RiskScore = 85.0
		result.Confidence = 0.85
		return result
	}

	if s.enableGlobalRateLimit && !s.checkGlobalRateLimit() {
		result.Allowed = false
		result.Reason = "global_rate_limit"
		result.AttackType = "Global Rate Limit Exceeded"
		result.SuggestedAction = "Reduce request rate"
		result.DetectionMethods = append(result.DetectionMethods, "global_rate_limit")
		return result
	}

	stats, exists := s.ipStats[ip]
	if !exists {
		stats = &IPStatistics{
			IP:           ip,
			RequestCount: 0,
			BlockedCount: 0,
			FirstSeen:    now,
			LastSeen:     now,
			AttackHistory: make([]AttackEvent, 0),
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
			StatusCodes:  []int{},
		}
		s.trafficData[ip] = traffic
	}

	traffic.RequestTimes = append(traffic.RequestTimes, now)
	traffic.UserAgents = append(traffic.UserAgents, r.UserAgent())
	traffic.Methods = append(traffic.Methods, r.Method)
	traffic.Paths = append(traffic.Paths, r.URL.Path)

	s.trimTrafficData(traffic)

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

	if s.enableAdaptiveRateLimiting {
		if !s.checkAdaptiveRateLimit(ip, stats, traffic) {
			stats.BlockedCount++
			stats.AttackHistory = append(stats.AttackHistory, AttackEvent{
				Timestamp:  now,
				AttackType: "Rate Limit",
				Blocked:    true,
				Score:      0.7,
			})
			result.Allowed = false
			result.Reason = "rate_limit"
			result.IPStats = stats
			result.AttackType = "Rate Limit Exceeded"
			result.SuggestedAction = "Increase rate limit or investigate"
			result.DetectionMethods = append(result.DetectionMethods, "adaptive_rate_limit")
			return result
		}
	}

	if s.enableBehaviorAnalysis {
		anomalyResult := s.advancedAnomalyDetection(traffic, stats, ip)
		stats.IsAnomaly = anomalyResult.IsAnomaly
		stats.Score = anomalyResult.Score

		if stats.IsAnomaly {
			stats.BlockedCount++
			stats.AttackHistory = append(stats.AttackHistory, AttackEvent{
				Timestamp:  now,
				AttackType: anomalyResult.AttackType,
				Blocked:    true,
				Score:      anomalyResult.Score,
			})
			result.Allowed = false
			result.Reason = "anomaly_detected"
			result.IPStats = stats
			result.AttackType = anomalyResult.AttackType
			result.SuggestedAction = anomalyResult.SuggestedAction
			result.RiskScore = anomalyResult.Score * 100
			result.Confidence = 0.8 + (anomalyResult.Score * 0.2)
			result.DetectionMethods = append(result.DetectionMethods, "behavior_anomaly")
			return result
		}
	}

	result.Allowed = true
	result.IPStats = stats
	result.RiskScore = stats.Score * 100
	result.Confidence = 0.3

	return result
}

type DDoSAnomalyResult struct {
	IsAnomaly       bool
	Score           float64
	AttackType      string
	SuggestedAction string
}

func (s *DDoSProtectionService) advancedAnomalyDetection(traffic *DDoSTrafficData, stats *IPStatistics, ip string) DDoSAnomalyResult {
	result := DDoSAnomalyResult{
		IsAnomaly: false,
		Score:     0.0,
	}

	if len(traffic.RequestTimes) < 10 {
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

	if cv < 0.03 && mean < 300 {
		result.IsAnomaly = true
		result.Score += 0.45
		result.AttackType = "Automated Request Pattern (Highly Regular)"
	}

	if mean < 30 && len(traffic.RequestTimes) > 50 {
		result.IsAnomaly = true
		result.Score += 0.35
		if result.AttackType == "" {
			result.AttackType = "Fast Request Rate"
		}
	}

	uniqueUA := s.countUnique(traffic.UserAgents)
	if uniqueUA == 1 && len(traffic.UserAgents) > 30 {
		result.IsAnomaly = true
		result.Score += 0.25
		if result.AttackType == "" {
			result.AttackType = "Single User Agent Flood"
		}
	}

	uniquePaths := s.countUnique(traffic.Paths)
	if uniquePaths == 1 && len(traffic.Paths) > 30 {
		result.IsAnomaly = true
		result.Score += 0.25
		if result.AttackType == "" {
			result.AttackType = "Single Path Flood"
		}
	}

	uniqueMethods := s.countUnique(traffic.Methods)
	if uniqueMethods == 1 && len(traffic.Methods) > 20 {
		result.Score += 0.15
	}

	if stats.BlockedCount > 5 {
		result.IsAnomaly = true
		result.Score += 0.4
		if result.AttackType == "" {
			result.AttackType = "Repeat Offender"
		}
	}

	if len(traffic.StatusCodes) > 20 {
		errorRate := s.calculateErrorRate(traffic.StatusCodes)
		if errorRate > 0.5 {
			result.Score += 0.3
			if result.AttackType == "" {
				result.AttackType = "High Error Rate Attack"
			}
		}
	}

	if result.Score >= s.attackThreshold {
		result.IsAnomaly = true
		result.SuggestedAction = "Investigate and consider blocking"
	}

	return result
}

func (s *DDoSProtectionService) calculateErrorRate(statusCodes []int) float64 {
	if len(statusCodes) == 0 {
		return 0.0
	}
	errorCount := 0
	for _, code := range statusCodes {
		if code >= 400 {
			errorCount++
		}
	}
	return float64(errorCount) / float64(len(statusCodes))
}

func (s *DDoSProtectionService) countUnique(items []string) int {
	seen := make(map[string]bool)
	for _, item := range items {
		seen[item] = true
	}
	return len(seen)
}

func (s *DDoSProtectionService) getIPReputation(ip string, r *http.Request) *IPReputation {
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
		Source:      "internal",
	}

	if s.networkDetection != nil {
		ctx := context.Background()
		networkResult, _ := s.networkDetection.DetectNetwork(ctx, ip, r.Header)
		if networkResult != nil {
			rep.IsTorExit = networkResult.IsTor
			rep.IsVPN = networkResult.IsVPN
			rep.IsProxy = networkResult.IsProxy
			rep.IsDatacenter = networkResult.IsDatacenter
			rep.CountryCode = networkResult.GeoLocation.Country
			if networkResult.NetworkInfo != nil {
				rep.ASN = fmt.Sprintf("%d", networkResult.NetworkInfo.ASN)
			}
			rep.Score = int(100 - networkResult.RiskScore)
			rep.ThreatLevel = networkResult.RiskLevel
			rep.Source = "network_detection"
		}
	} else {
		if strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "192.168.") {
			rep.Score = 80
			rep.ThreatLevel = "low"
		} else if strings.HasPrefix(ip, "172.") {
			rep.Score = 75
			rep.ThreatLevel = "low"
		}
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

func (s *DDoSProtectionService) checkAdaptiveRateLimit(ip string, stats *IPStatistics, traffic *DDoSTrafficData) bool {
	baseLimit := float64(s.requestsPerMin)

	if stats.Reputation > 0 && stats.Reputation < 40 {
		baseLimit *= 0.2
	} else if stats.Reputation > 0 && stats.Reputation < 60 {
		baseLimit *= 0.5
	}

	if stats.Score > 0.4 {
		baseLimit *= 0.6
	}

	recentRequests := 0
	cutoff := time.Now().Add(-1 * time.Minute)
	for _, t := range traffic.RequestTimes {
		if t.After(cutoff) {
			recentRequests++
		}
	}

	return float64(recentRequests) <= baseLimit
}

func (s *DDoSProtectionService) checkGlobalRateLimit() bool {
	s.globalRequestCount++
	return s.globalRequestCount <= int64(s.globalRateLimit)
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

func (s *DDoSProtectionService) trimTrafficData(traffic *DDoSTrafficData) {
	maxEntries := 500
	if len(traffic.UserAgents) > maxEntries {
		traffic.UserAgents = traffic.UserAgents[len(traffic.UserAgents)-maxEntries:]
	}
	if len(traffic.Methods) > maxEntries {
		traffic.Methods = traffic.Methods[len(traffic.Methods)-maxEntries:]
	}
	if len(traffic.Paths) > maxEntries {
		traffic.Paths = traffic.Paths[len(traffic.Paths)-maxEntries:]
	}
	if len(traffic.RequestTimes) > 1000 {
		traffic.RequestTimes = traffic.RequestTimes[len(traffic.RequestTimes)-1000:]
	}
	if len(traffic.StatusCodes) > 500 {
		traffic.StatusCodes = traffic.StatusCodes[len(traffic.StatusCodes)-500:]
	}
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
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		s.globalRequestCount = 0
		s.globalLastReset = time.Now()
		s.mu.Unlock()
	}
}

func (s *DDoSProtectionService) reputationUpdateLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for ip, rep := range s.ipReputations {
			if now.Sub(rep.LastUpdated) > 1*time.Hour {
				delete(s.ipReputations, ip)
			}
		}
		s.mu.Unlock()
	}
}

func (s *DDoSProtectionService) attackModeMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		reqCount := s.globalRequestCount
		s.mu.RUnlock()

		if reqCount > int64(s.globalRateLimit)*2 {
			s.mu.Lock()
			s.attackModeActive = true
			s.attackModeStartTime = time.Now()
			s.mu.Unlock()
		} else {
			s.mu.Lock()
			if s.attackModeActive && time.Since(s.attackModeStartTime) > 5*time.Minute {
				s.attackModeActive = false
			}
			s.mu.Unlock()
		}
	}
}

func (s *DDoSProtectionService) IsAttackModeActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.attackModeActive
}

func (s *DDoSProtectionService) GetGlobalStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"total_ips":         len(s.ipStats),
		"blacklist_count":   len(s.blacklist),
		"whitelist_count":   len(s.whitelist),
		"active_since":      s.globalLastReset,
		"attack_mode":       s.attackModeActive,
		"global_requests":   s.globalRequestCount,
		"reputation_cache": len(s.ipReputations),
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

func (s *DDoSProtectionService) GetIPReputation(ip string) *IPReputation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ipReputations[ip]
}

func (s *DDoSProtectionService) UpdateIPReputation(ip string, reputation *IPReputation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	reputation.LastUpdated = time.Now()
	s.ipReputations[ip] = reputation
}

func (s *DDoSProtectionService) BatchCheckIPs(ips []string) []*IPReputation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]*IPReputation, len(ips))
	for i, ip := range ips {
		if rep, exists := s.ipReputations[ip]; exists {
			results[i] = rep
		} else {
			results[i] = &IPReputation{
				IP:          ip,
				Score:       50,
				ThreatLevel: "unknown",
			}
		}
	}
	return results
}

func (s *DDoSProtectionService) SetCustomRateLimit(ip string, requestsPerMinute int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rateLimiters[ip] = NewTokenBucket(requestsPerMinute, float64(requestsPerMinute)/60)
}

func (s *DDoSProtectionService) GetCustomRateLimit(ip string) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	limiter, exists := s.rateLimiters[ip]
	if !exists {
		return 0, false
	}
	return int(limiter.maxTokens), true
}

func (s *DDoSProtectionService) SimulateAttackDetection() {
	s.mu.Lock()
	defer s.mu.Unlock()

	testIP := "192.168.1.100"
	traffic := s.trafficData[testIP]
	if traffic == nil {
		traffic = &DDoSTrafficData{
			RequestTimes: []time.Time{},
			UserAgents:   []string{},
			Methods:      []string{},
			Paths:        []string{},
		}
		s.trafficData[testIP] = traffic
	}

	now := time.Now()
	for i := 0; i < 200; i++ {
		traffic.RequestTimes = append(traffic.RequestTimes, now.Add(time.Duration(i*50)*time.Millisecond))
		traffic.UserAgents = append(traffic.UserAgents, "python-requests/2.28.0")
		traffic.Methods = append(traffic.Methods, "GET")
		traffic.Paths = append(traffic.Paths, "/api/v1/data")
	}

	stats := s.ipStats[testIP]
	if stats == nil {
		stats = &IPStatistics{
			IP:           testIP,
			RequestCount: 200,
			FirstSeen:    now,
			LastSeen:     now,
		}
		s.ipStats[testIP] = stats
	} else {
		stats.RequestCount += 200
	}
}