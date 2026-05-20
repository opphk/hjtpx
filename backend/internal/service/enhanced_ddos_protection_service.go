package service

import (
	"container/list"
	"context"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/redis"
)

type EnhancedDDoSProtectionConfig struct {
	Enabled                    bool
	RequestsPerSecond          int
	RequestsPerMinute          int
	RequestsPerHour            int
	BurstSize                  int
	ConnectionLimitPerIP       int
	ConnectionTimeout          time.Duration
	BlacklistDuration         time.Duration
	WhitelistEnabled          bool
	TrafficAnomalyThreshold   float64
	EnableGeoIPBlock          bool
	BlockedCountries          []string
	MaxIPsInMemory            int
	CleanupInterval           time.Duration
	EnableBehavioralAnalysis  bool
	MinRequestInterval        time.Duration
}

type EnhancedDDoSCheckResult struct {
	Allowed              bool
	Reason               string
	Severity             string
	IPStats              *EnhancedIPStatistics
	RetryAfter           int
	ThreatLevel          float64
	RecommendedAction    string
}

type EnhancedIPStatistics struct {
	IP                  string
	RequestCount        int64
	RequestCountMinute  int64
	RequestCountHour    int64
	BlockedCount        int64
	ConnectionCount     int
	FirstSeen           time.Time
	LastSeen            time.Time
	RequestRate         float64
	AvgRequestInterval  time.Duration
	IsAnomaly           bool
	IsBlacklisted       bool
	ThreatScore         float64
	Country             string
	UserAgents          []string
	UniquePaths         int
	UniqueMethods       int
	ErrorRate           float64
	TotalErrors         int
	TotalRequests       int64
	ErrorCounts         map[int]int
	UniqueUserAgents    map[string]int
	UniquePathsSet      map[string]bool
}

type EnhancedDDoSTrafficData struct {
	RequestTimes      []time.Time
	RequestSizes      []int64
	Methods           []string
	Paths             []string
	UserAgents        []string
	StatusCodes       []int
	ErrorCounts       map[int]int
	UniqueUserAgents  map[string]int
	UniquePathsSet    map[string]bool
	mu                sync.RWMutex
}

type SlidingWindowRateLimiter struct {
	windowSize time.Duration
	maxRequests int64
	requests    *list.List
	mu          sync.Mutex
}

type TokenBucketDDoS struct {
	capacity    int64
	rate        float64
	tokens      float64
	lastRefill  time.Time
	mu          sync.Mutex
}

type EnhancedDDoSProtectionService struct {
	config            EnhancedDDoSProtectionConfig
	ipStats           map[string]*EnhancedIPStatistics
	trafficData       map[string]*EnhancedDDoSTrafficData
	blacklist         map[string]time.Time
	whitelist         map[string]bool
	connectionCounts   map[string]int
	slidingWindows    map[string]*SlidingWindowRateLimiter
	tokenBuckets      map[string]*TokenBucketDDoS
	globalRateLimiter *SlidingWindowRateLimiter
	mu                sync.RWMutex
	anomalyDetector   *DDOSAnomalyDetector
	patternMatcher    *AttackPatternMatcher
}

type DDOSAnomalyDetector struct {
	baselineMetrics map[string]*DDOSBaselineMetrics
	mu              sync.RWMutex
}

type DDOSBaselineMetrics struct {
	MeanInterval     float64
	StdDevInterval  float64
	MeanRequestRate  float64
	StdDevRequestRate float64
	SampleCount      int
	LastUpdated      time.Time
}

type AttackPatternMatcher struct {
	patterns []*AttackPatternV2
	mu       sync.RWMutex
}

type AttackPatternV2 struct {
	Name        string
	Pattern     *regexp.Regexp
	Severity    float64
	Description string
}

var defaultEnhancedDDoSConfig = EnhancedDDoSProtectionConfig{
	Enabled:                   true,
	RequestsPerSecond:         10,
	RequestsPerMinute:         100,
	RequestsPerHour:           1000,
	BurstSize:                15,
	ConnectionLimitPerIP:      10,
	ConnectionTimeout:        30 * time.Second,
	BlacklistDuration:        1 * time.Hour,
	WhitelistEnabled:         false,
	TrafficAnomalyThreshold:  0.7,
	EnableGeoIPBlock:         false,
	BlockedCountries:          []string{},
	MaxIPsInMemory:           100000,
	CleanupInterval:          10 * time.Minute,
	EnableBehavioralAnalysis: true,
	MinRequestInterval:       100 * time.Millisecond,
}

func NewEnhancedDDoSProtectionService(config ...EnhancedDDoSProtectionConfig) *EnhancedDDoSProtectionService {
	cfg := defaultEnhancedDDoSConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	service := &EnhancedDDoSProtectionService{
		config:            cfg,
		ipStats:           make(map[string]*EnhancedIPStatistics),
		trafficData:       make(map[string]*EnhancedDDoSTrafficData),
		blacklist:         make(map[string]time.Time),
		whitelist:         make(map[string]bool),
		connectionCounts:  make(map[string]int),
		slidingWindows:    make(map[string]*SlidingWindowRateLimiter),
		tokenBuckets:      make(map[string]*TokenBucketDDoS),
		anomalyDetector:   NewDDOSAnomalyDetector(),
		patternMatcher:    NewAttackPatternMatcher(),
	}

	service.globalRateLimiter = NewSlidingWindowRateLimiter(time.Minute, int64(cfg.RequestsPerMinute))

	go service.cleanupRoutine()
	go service.statsUpdateRoutine()

	return service
}

func NewSlidingWindowRateLimiter(windowSize time.Duration, maxRequests int64) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		requests:    list.New(),
	}
}

func (sw *SlidingWindowRateLimiter) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.windowSize)

	for sw.requests.Len() > 0 && sw.requests.Front().Value.(time.Time).Before(cutoff) {
		sw.requests.Remove(sw.requests.Front())
	}

	if int64(sw.requests.Len()) < sw.maxRequests {
		sw.requests.PushBack(now)
		return true
	}

	return false
}

func (sw *SlidingWindowRateLimiter) Count() int64 {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.windowSize)

	count := int64(0)
	for e := sw.requests.Front(); e != nil; e = e.Next() {
		if e.Value.(time.Time).After(cutoff) {
			count++
		}
	}

	return count
}

func NewTokenBucketDDoS(capacity int64, rate float64) *TokenBucketDDoS {
	return &TokenBucketDDoS{
		capacity:   capacity,
		rate:       rate,
		tokens:     float64(capacity),
		lastRefill: time.Now(),
	}
}

func (tb *TokenBucketDDoS) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}

	return false
}

func (tb *TokenBucketDDoS) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}
	tb.lastRefill = now
}

func (tb *TokenBucketDDoS) GetTokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	return tb.tokens
}

func NewDDOSAnomalyDetector() *DDOSAnomalyDetector {
	return &DDOSAnomalyDetector{
		baselineMetrics: make(map[string]*DDOSBaselineMetrics),
	}
}

func (ad *DDOSAnomalyDetector) UpdateBaseline(ip string, interval time.Duration, rate float64) {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	metrics, exists := ad.baselineMetrics[ip]
	if !exists {
		metrics = &DDOSBaselineMetrics{
			SampleCount: 0,
		}
		ad.baselineMetrics[ip] = metrics
	}

	if metrics.SampleCount == 0 {
		metrics.MeanInterval = float64(interval)
		metrics.MeanRequestRate = rate
		metrics.SampleCount = 1
	} else {
		n := float64(metrics.SampleCount)
		metrics.MeanInterval = (n*metrics.MeanInterval + float64(interval)) / (n + 1)
		metrics.MeanRequestRate = (n*metrics.MeanRequestRate + rate) / (n + 1)
		metrics.SampleCount++
	}

	metrics.LastUpdated = time.Now()
}

func (ad *DDOSAnomalyDetector) DetectAnomaly(ip string, currentInterval time.Duration, currentRate float64) (bool, float64) {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	metrics, exists := ad.baselineMetrics[ip]
	if !exists || metrics.SampleCount < 10 {
		return false, 0
	}

	intervalDeviation := math.Abs(float64(currentInterval) - metrics.MeanInterval) / (metrics.StdDevInterval + 1)
	rateDeviation := math.Abs(currentRate - metrics.MeanRequestRate) / (metrics.StdDevRequestRate + 1)

	totalDeviation := (intervalDeviation + rateDeviation) / 2
	anomalyScore := math.Min(totalDeviation/3.0, 1.0)

	return totalDeviation > 2.5, anomalyScore
}

func NewAttackPatternMatcher() *AttackPatternMatcher {
	matcher := &AttackPatternMatcher{
		patterns: make([]*AttackPatternV2, 0),
	}

	matcher.patterns = append(matcher.patterns, &AttackPatternV2{
		Name:        "SQL Injection",
		Pattern:     regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|exec|execute|script|--|;|/\*|\*/|declare|convert|xp_)`),
		Severity:    0.8,
		Description: "SQL injection attack pattern detected",
	})

	matcher.patterns = append(matcher.patterns, &AttackPatternV2{
		Name:        "XSS Attack",
		Pattern:     regexp.MustCompile(`(?i)(<script|javascript:|onerror|onload|onclick|alert\(|eval\(|document\.|window\.|<img|<svg|<iframe|<embed|<object)`),
		Severity:    0.7,
		Description: "XSS attack pattern detected",
	})

	matcher.patterns = append(matcher.patterns, &AttackPatternV2{
		Name:        "Path Traversal",
		Pattern:     regexp.MustCompile(`(?i)(\.\./|\.\.\\|%2e%2e|/etc/passwd|c:\\windows|root:|/etc/shadow|\.\.%2f)`),
		Severity:    0.75,
		Description: "Path traversal attack pattern detected",
	})

	matcher.patterns = append(matcher.patterns, &AttackPatternV2{
		Name:        "Command Injection",
		Pattern:     regexp.MustCompile(`(?i)(;|\|\||&&|` + "`" + `|\$\(|\\x|;.*(cat|ls|wget|curl|nc|bash|sh))`),
		Severity:    0.9,
		Description: "Command injection pattern detected",
	})

	matcher.patterns = append(matcher.patterns, &AttackPatternV2{
		Name:        "Scanner Activity",
		Pattern:     regexp.MustCompile(`(?i)(nikto|nmap|gobuster|dirbuster|sqlmap|burp|hydra|wpscan|acunetix|appscan|metasploit)`),
		Severity:    0.6,
		Description: "Security scanner activity detected",
	})

	matcher.patterns = append(matcher.patterns, &AttackPatternV2{
		Name:        "LDAP Injection",
		Pattern:     regexp.MustCompile(`(?i)(\(|\)|\*|%|,|;|&|\||=|\+|\\/)`),
		Severity:    0.75,
		Description: "LDAP injection pattern detected",
	})

	matcher.patterns = append(matcher.patterns, &AttackPatternV2{
		Name:        "XML Injection",
		Pattern:     regexp.MustCompile(`(?i)(<!DOCTYPE|<!ENTITY|<!ATTLIST|<!ELEMENT|<!NOTATION|<!%|<%|\%3c|\%3e)`),
		Severity:    0.8,
		Description: "XML injection pattern detected",
	})

	matcher.patterns = append(matcher.patterns, &AttackPatternV2{
		Name:        "Template Injection",
		Pattern:     regexp.MustCompile(`(?i)(\{\{|\}\}|\{%|%7b%7d|\$\{)`),
		Severity:    0.85,
		Description: "Template injection pattern detected",
	})

	return matcher
}

func (apm *AttackPatternMatcher) Match(content string) (bool, *AttackPatternV2) {
	apm.mu.RLock()
	defer apm.mu.RUnlock()

	for _, pattern := range apm.patterns {
		if pattern.Pattern.MatchString(content) {
			return true, pattern
		}
	}

	return false, nil
}

func (s *EnhancedDDoSProtectionService) CheckRequest(r *http.Request) *EnhancedDDoSCheckResult {
	if !s.config.Enabled {
		return &EnhancedDDoSCheckResult{Allowed: true}
	}

	ip := s.getClientIP(r)
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.config.WhitelistEnabled && s.whitelist[ip] {
		return &EnhancedDDoSCheckResult{Allowed: true, IPStats: s.getOrCreateStats(ip)}
	}

	if expiry, exists := s.blacklist[ip]; exists {
		if now.Before(expiry) {
			return &EnhancedDDoSCheckResult{
				Allowed:           false,
				Reason:            "ip_blacklisted",
				Severity:          "critical",
				RetryAfter:        int(time.Until(expiry).Seconds()),
				RecommendedAction: "wait",
			}
		}
		delete(s.blacklist, ip)
	}

	stats := s.getOrCreateStatsLocked(ip)

	if s.isRequestRateAnomalous(stats) {
		s.blacklist[ip] = now.Add(s.config.BlacklistDuration)
		stats.BlockedCount++
		stats.ThreatScore = 1.0
		return &EnhancedDDoSCheckResult{
			Allowed:           false,
			Reason:            "rate_anomaly_detected",
			Severity:          "critical",
			IPStats:           stats,
			ThreatLevel:       stats.ThreatScore,
			RetryAfter:        int(s.config.BlacklistDuration.Seconds()),
			RecommendedAction: "block",
		}
	}

	s.recordTrafficLocked(ip, r, now)

	if s.connectionCounts[ip] >= s.config.ConnectionLimitPerIP {
		return &EnhancedDDoSCheckResult{
			Allowed:           false,
			Reason:            "connection_limit_exceeded",
			Severity:          "high",
			IPStats:           stats,
			RecommendedAction: "reduce_connections",
		}
	}

	s.connectionCounts[ip]++

	if !s.globalRateLimiter.Allow() {
		stats.BlockedCount++
		return &EnhancedDDoSCheckResult{
			Allowed:           false,
			Reason:            "global_rate_limit_exceeded",
			Severity:           "medium",
			IPStats:           stats,
			RetryAfter:        60,
			RecommendedAction: "wait",
		}
	}

	slidingWindow := s.getOrCreateSlidingWindowLocked(ip)
	if !slidingWindow.Allow() {
		stats.BlockedCount++
		return &EnhancedDDoSCheckResult{
			Allowed:           false,
			Reason:            "rate_limit_exceeded",
			Severity:          "medium",
			IPStats:           stats,
			RetryAfter:        60,
			RecommendedAction: "wait",
		}
	}

	tokenBucket := s.getOrCreateTokenBucketLocked(ip)
	if !tokenBucket.Allow() {
		stats.BlockedCount++
		return &EnhancedDDoSCheckResult{
			Allowed:           false,
			Reason:            "burst_limit_exceeded",
			Severity:          "medium",
			IPStats:           stats,
			RetryAfter:        10,
			RecommendedAction: "slow_down",
		}
	}

	if s.config.EnableBehavioralAnalysis {
		anomalyResult := s.analyzeBehaviorLocked(ip, now)
		if anomalyResult.detected {
			stats.ThreatScore += anomalyResult.score * 0.3
			if stats.ThreatScore > 0.8 {
				s.blacklist[ip] = now.Add(s.config.BlacklistDuration)
				return &EnhancedDDoSCheckResult{
					Allowed:           false,
					Reason:            "behavioral_anomaly_detected",
					Severity:          "critical",
					IPStats:           stats,
					ThreatLevel:       stats.ThreatScore,
					RetryAfter:        int(s.config.BlacklistDuration.Seconds()),
					RecommendedAction: "block",
				}
			}
		}
	}

	s.checkAttackPatterns(r, stats)

	stats.RequestCount++
	stats.RequestCountMinute++
	stats.RequestCountHour++
	stats.LastSeen = now

	if stats.RequestRate == 0 {
		stats.RequestRate = 1
	} else {
		stats.RequestRate = float64(stats.RequestCountMinute) / 1.0
	}

	return &EnhancedDDoSCheckResult{
		Allowed:           true,
		IPStats:           stats,
		ThreatLevel:       stats.ThreatScore,
		RecommendedAction: "allow",
	}
}

func (s *EnhancedDDoSProtectionService) isRequestRateAnomalous(stats *EnhancedIPStatistics) bool {
	if stats.RequestCountMinute > int64(s.config.RequestsPerMinute)*5 {
		return true
	}

	if stats.RequestCount > 0 && stats.BlockedCount > 0 {
		blockRatio := float64(stats.BlockedCount) / float64(stats.RequestCount)
		if blockRatio > 0.5 {
			return true
		}
	}

	return false
}

func (s *EnhancedDDoSProtectionService) getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}

	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	ip = r.Header.Get("CF-Connecting-IP")
	if ip != "" {
		return ip
	}

	return r.RemoteAddr
}

func (s *EnhancedDDoSProtectionService) getOrCreateStats(ip string) *EnhancedIPStatistics {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getOrCreateStatsLocked(ip)
}

func (s *EnhancedDDoSProtectionService) getOrCreateStatsLocked(ip string) *EnhancedIPStatistics {
	stats, exists := s.ipStats[ip]
	if !exists {
		stats = &EnhancedIPStatistics{
			IP:                  ip,
			FirstSeen:          time.Now(),
			LastSeen:           time.Now(),
			ErrorCounts:        make(map[int]int),
			UniqueUserAgents:   make(map[string]int),
			UniquePathsSet:     make(map[string]bool),
		}
		s.ipStats[ip] = stats

		if len(s.ipStats) > s.config.MaxIPsInMemory {
			s.cleanupOldIPsLocked()
		}
	}
	return stats
}

func (s *EnhancedDDoSProtectionService) recordTrafficLocked(ip string, r *http.Request, now time.Time) {
	traffic, exists := s.trafficData[ip]
	if !exists {
		traffic = &EnhancedDDoSTrafficData{
			RequestTimes:     make([]time.Time, 0, 1000),
			RequestSizes:     make([]int64, 0, 1000),
			Methods:          make([]string, 0, 100),
			Paths:            make([]string, 0, 100),
			UserAgents:       make([]string, 0, 100),
			StatusCodes:      make([]int, 0, 100),
			ErrorCounts:      make(map[int]int),
			UniqueUserAgents: make(map[string]int),
			UniquePathsSet:   make(map[string]bool),
		}
		s.trafficData[ip] = traffic
	}

	traffic.mu.Lock()
	defer traffic.mu.Unlock()

	traffic.RequestTimes = append(traffic.RequestTimes, now)
	if len(traffic.RequestTimes) > 1000 {
		traffic.RequestTimes = traffic.RequestTimes[len(traffic.RequestTimes)-1000:]
	}

	if r.ContentLength > 0 {
		traffic.RequestSizes = append(traffic.RequestSizes, r.ContentLength)
	}

	traffic.Methods = append(traffic.Methods, r.Method)
	if len(traffic.Methods) > 100 {
		traffic.Methods = traffic.Methods[len(traffic.Methods)-100:]
	}

	path := r.URL.Path
	traffic.Paths = append(traffic.Paths, path)
	traffic.UniquePathsSet[path] = true

	userAgent := r.Header.Get("User-Agent")
	if userAgent != "" {
		traffic.UserAgents = append(traffic.UserAgents, userAgent)
		traffic.UniqueUserAgents[userAgent]++
	}
	if len(traffic.UserAgents) > 100 {
		traffic.UserAgents = traffic.UserAgents[len(traffic.UserAgents)-100:]
	}
}

type anomalyResult struct {
	detected bool
	score    float64
	reason   string
}

func (s *EnhancedDDoSProtectionService) analyzeBehaviorLocked(ip string, now time.Time) anomalyResult {
	traffic, exists := s.trafficData[ip]
	if !exists {
		return anomalyResult{detected: false}
	}

	traffic.mu.RLock()
	defer traffic.mu.RUnlock()

	if len(traffic.RequestTimes) < 10 {
		return anomalyResult{detected: false}
	}

	intervals := make([]float64, 0, len(traffic.RequestTimes)-1)
	for i := 1; i < len(traffic.RequestTimes); i++ {
		interval := traffic.RequestTimes[i].Sub(traffic.RequestTimes[i-1]).Seconds() * 1000
		intervals = append(intervals, interval)
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

	cv := stdDev / (mean + 1)
	if cv < 0.05 && mean < 500 {
		return anomalyResult{
			detected: true,
			score:    0.9,
			reason:   "uniform_request_interval",
		}
	}

	if mean < 50 && len(traffic.RequestTimes) > 30 {
		return anomalyResult{
			detected: true,
			score:    0.85,
			reason:   "extremely_high_request_rate",
		}
	}

	uniqueUserAgents := len(traffic.UniqueUserAgents)
	totalRequests := len(traffic.RequestTimes)
	if totalRequests > 20 && uniqueUserAgents == 1 {
		ua := ""
		for k := range traffic.UniqueUserAgents {
			ua = k
			break
		}
		if strings.Contains(strings.ToLower(ua), "python") ||
			strings.Contains(strings.ToLower(ua), "curl") ||
			strings.Contains(strings.ToLower(ua), "wget") ||
			strings.Contains(strings.ToLower(ua), "go-http") ||
			strings.Contains(strings.ToLower(ua), "java/") ||
			strings.Contains(strings.ToLower(ua), "okhttp") {
			return anomalyResult{
				detected: true,
				score:    0.7,
				reason:   "automated_tool_detected",
			}
		}
	}

	uniquePaths := len(traffic.UniquePathsSet)
	if uniquePaths > 50 && float64(uniquePaths)/float64(totalRequests) > 0.9 {
		return anomalyResult{
			detected: true,
			score:    0.75,
			reason:   "high_path_diversity_scanning",
		}
	}

	if mean < 100 && stdDev < 20 {
		return anomalyResult{
			detected: true,
			score:    0.8,
			reason:   "mechanical_request_pattern",
		}
	}

	stats := s.ipStats[ip]
	if stats != nil {
		currentRate := stats.RequestRate
		isAnomaly, anomalyScore := s.anomalyDetector.DetectAnomaly(ip, time.Duration(mean)*time.Millisecond, currentRate)
		if isAnomaly {
			return anomalyResult{
				detected: true,
				score:    anomalyScore,
				reason:   "baseline_deviation",
			}
		}

		if stats.ErrorRate > 0.5 && stats.TotalErrors > 10 {
			return anomalyResult{
				detected: true,
				score:    0.6,
				reason:   "high_error_rate",
			}
		}
	}

	return anomalyResult{detected: false}
}

func (s *EnhancedDDoSProtectionService) checkAttackPatterns(r *http.Request, stats *EnhancedIPStatistics) {
	url := r.URL.String()

	matched, pattern := s.patternMatcher.Match(url)
	if matched {
		stats.ThreatScore += pattern.Severity * 0.2
		stats.TotalErrors++
	}

	for name, values := range r.Header {
		headerValue := name + ": " + strings.Join(values, ", ")
		if matched, pattern := s.patternMatcher.Match(headerValue); matched {
			stats.ThreatScore += pattern.Severity * 0.15
		}
	}
}

func (s *EnhancedDDoSProtectionService) getOrCreateSlidingWindowLocked(ip string) *SlidingWindowRateLimiter {
	sw, exists := s.slidingWindows[ip]
	if !exists {
		sw = NewSlidingWindowRateLimiter(time.Minute, int64(s.config.RequestsPerMinute))
		s.slidingWindows[ip] = sw
	}
	return sw
}

func (s *EnhancedDDoSProtectionService) getOrCreateTokenBucketLocked(ip string) *TokenBucketDDoS {
	tb, exists := s.tokenBuckets[ip]
	if !exists {
		tb = NewTokenBucketDDoS(int64(s.config.BurstSize), float64(s.config.RequestsPerSecond))
		s.tokenBuckets[ip] = tb
	}
	return tb
}

func (s *EnhancedDDoSProtectionService) cleanupOldIPsLocked() {
	cutoff := time.Now().Add(-30 * time.Minute)
	for ip, stats := range s.ipStats {
		if stats.LastSeen.Before(cutoff) {
			delete(s.ipStats, ip)
			delete(s.trafficData, ip)
			delete(s.slidingWindows, ip)
			delete(s.tokenBuckets, ip)
			delete(s.connectionCounts, ip)
		}
	}
}

func (s *EnhancedDDoSProtectionService) cleanupRoutine() {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		s.cleanupOldIPsLocked()

		now := time.Now()
		for ip, expiry := range s.blacklist {
			if now.After(expiry) {
				delete(s.blacklist, ip)
			}
		}
		s.mu.Unlock()
	}
}

func (s *EnhancedDDoSProtectionService) statsUpdateRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()

		for ip, stats := range s.ipStats {
			s.anomalyDetector.UpdateBaseline(ip, stats.AvgRequestInterval, stats.RequestRate)

			if stats.RequestCountMinute > int64(s.config.RequestsPerMinute) {
				stats.ThreatScore += 0.1
			}
		}
		s.mu.Unlock()
	}
}

func (s *EnhancedDDoSProtectionService) AddToBlacklist(ip string, reason string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.blacklist[ip] = time.Now().Add(duration)

	if s.ipStats[ip] != nil {
		s.ipStats[ip].IsBlacklisted = true
	}
}

func (s *EnhancedDDoSProtectionService) RemoveFromBlacklist(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.blacklist, ip)

	if s.ipStats[ip] != nil {
		s.ipStats[ip].IsBlacklisted = false
	}
}

func (s *EnhancedDDoSProtectionService) AddToWhitelist(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.whitelist[ip] = true
}

func (s *EnhancedDDoSProtectionService) RemoveFromWhitelist(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.whitelist, ip)
}

func (s *EnhancedDDoSProtectionService) GetIPStats(ip string) *EnhancedIPStatistics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ipStats[ip]
}

func (s *EnhancedDDoSProtectionService) GetAllStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalRequests := int64(0)
	totalBlocked := int64(0)
	blacklistedIPs := len(s.blacklist)

	for _, stats := range s.ipStats {
		totalRequests += stats.RequestCount
		totalBlocked += stats.BlockedCount
	}

	return map[string]interface{}{
		"total_ips":        len(s.ipStats),
		"total_requests":   totalRequests,
		"total_blocked":    totalBlocked,
		"blacklisted_ips":  blacklistedIPs,
		"whitelisted_ips":  len(s.whitelist),
		"config":           s.config,
	}
}

func (s *EnhancedDDoSProtectionService) ReleaseConnection(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if count, exists := s.connectionCounts[ip]; exists && count > 0 {
		s.connectionCounts[ip] = count - 1
	}
}

func (s *EnhancedDDoSProtectionService) UpdateConfig(config EnhancedDDoSProtectionConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

func (s *EnhancedDDoSProtectionService) GetConfig() EnhancedDDoSProtectionConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *EnhancedDDoSProtectionService) RecordError(ip string, statusCode int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stats := s.ipStats[ip]
	if stats != nil {
		stats.TotalErrors++
		stats.ErrorCounts[statusCode]++
		stats.TotalRequests = stats.RequestCount
		if stats.TotalRequests > 0 {
			stats.ErrorRate = float64(stats.TotalErrors) / float64(stats.TotalRequests)
		}
	}
}

func (s *EnhancedDDoSProtectionService) ShouldBlockByCountry(country string) bool {
	if !s.config.EnableGeoIPBlock {
		return false
	}

	for _, blocked := range s.config.BlockedCountries {
		if strings.EqualFold(country, blocked) {
			return true
		}
	}

	return false
}

func (s *EnhancedDDoSProtectionService) GetTopThreatIPs(limit int) []*EnhancedIPStatistics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type threatPair struct {
		ip    string
		stats *EnhancedIPStatistics
	}

	pairs := make([]threatPair, 0, len(s.ipStats))
	for ip, stats := range s.ipStats {
		pairs = append(pairs, threatPair{ip: ip, stats: stats})
	}

	for i := 0; i < len(pairs)-1; i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].stats.ThreatScore > pairs[i].stats.ThreatScore {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	result := make([]*EnhancedIPStatistics, 0, limit)
	for i := 0; i < limit && i < len(pairs); i++ {
		result = append(result, pairs[i].stats)
	}

	return result
}

func (s *EnhancedDDoSProtectionService) ResetStats(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.ipStats, ip)
	delete(s.trafficData, ip)
	delete(s.slidingWindows, ip)
	delete(s.tokenBuckets, ip)
	delete(s.connectionCounts, ip)
}

func (s *EnhancedDDoSProtectionService) ResetAllStats() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ipStats = make(map[string]*EnhancedIPStatistics)
	s.trafficData = make(map[string]*EnhancedDDoSTrafficData)
	s.slidingWindows = make(map[string]*SlidingWindowRateLimiter)
	s.tokenBuckets = make(map[string]*TokenBucketDDoS)
	s.connectionCounts = make(map[string]int)
	s.blacklist = make(map[string]time.Time)
}

type DistributedDDoSProtectionService struct {
	localService *EnhancedDDoSProtectionService
	redisEnabled bool
}

func NewDistributedDDoSProtectionService(config ...EnhancedDDoSProtectionConfig) *DistributedDDoSProtectionService {
	service := &DistributedDDoSProtectionService{
		localService: NewEnhancedDDoSProtectionService(config...),
		redisEnabled: redis.Client != nil,
	}

	if service.redisEnabled {
		go service.syncToRedis()
		go service.syncFromRedis()
	}

	return service
}

func (s *DistributedDDoSProtectionService) CheckRequest(r *http.Request) *EnhancedDDoSCheckResult {
	return s.localService.CheckRequest(r)
}

func (s *DistributedDDoSProtectionService) syncToRedis() {
	if !s.redisEnabled || redis.Client == nil {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := s.localService.GetAllStats()
		ctx := context.Background()

		data := fmt.Sprintf("%v", stats)
		redis.Client.Set(ctx, "ddos:global_stats", data, 5*time.Minute)
	}
}

func (s *DistributedDDoSProtectionService) syncFromRedis() {
	if !s.redisEnabled || redis.Client == nil {
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()

		blacklistJSON, err := redis.Client.Get(ctx, "ddos:blacklist").Result()
		if err == nil && blacklistJSON != "" {
			_ = blacklistJSON
		}

		whitelistJSON, err := redis.Client.Get(ctx, "ddos:whitelist").Result()
		if err == nil && whitelistJSON != "" {
			_ = whitelistJSON
		}
	}
}

func (s *DistributedDDoSProtectionService) AddToBlacklist(ip string, reason string, duration time.Duration) {
	s.localService.AddToBlacklist(ip, reason, duration)

	if s.redisEnabled && redis.Client != nil {
		ctx := context.Background()
		key := fmt.Sprintf("ddos:blacklist:%s", ip)
		redis.Client.Set(ctx, key, fmt.Sprintf("%s:%s", reason, time.Now().Add(duration).Format(time.RFC3339)), duration)
	}
}

func (s *DistributedDDoSProtectionService) GetStats() map[string]interface{} {
	return s.localService.GetAllStats()
}
