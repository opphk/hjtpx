package service

import (
	"math"
	"net/http"
	"regexp"
	"sync"
	"time"
)

type DDoSProtectionTier string

const (
	DDoSTierNormal   DDoSProtectionTier = "normal"
	DDoSTierWarning  DDoSProtectionTier = "warning"
	DDoSTierCritical DDoSProtectionTier = "critical"
	DDoSTierBlocked DDoSProtectionTier = "blocked"
)

type TrafficCleanResult struct {
	Cleaned         bool
	DropReason      string
	DroppedBytes    int64
	ProcessedBytes  int64
	BandwidthSaved  int64
}

type CleanedTrafficStats struct {
	TotalRequests   int64
	CleanedRequests int64
	DroppedBytes    int64
	CleanRate       float64
}

type RateLimitStrategy struct {
	Name             string
	RequestsPerSec   int
	BurstSize        int
	WindowSize       time.Duration
	Adaptive         bool
	BlockDuration    time.Duration
}

type IPReputation struct {
	IP              string
	Score           float64
	Tier            DDoSProtectionTier
	LastUpdated     time.Time
	RequestCount    int64
	BlockCount      int64
	TotalTraffic    int64
	ReputationTags  []string
	TrustLevel      int
	IsWhitelisted   bool
	IsBlacklisted   bool
}

type DDoSProtectionConfig struct {
	EnableTrafficCleaning    bool
	EnableIPReputation       bool
	EnableAdaptiveLimits     bool
	EnableRateLimitStrategy  bool
	EnableBandwidthThrottle  bool
	MaxBandwidthGbps         float64
	RequestsPerSecond        int
	BurstSize                int
	WindowSize               time.Duration
	BlockDuration            time.Duration
	CleanupInterval          time.Duration
	MaxIPs                   int
}

type DDoSEnhancedProtectionService struct {
	config                  DDoSProtectionConfig

	ipStats                 map[string]*IPStatistics
	trafficData             map[string]*DDoSTrafficData
	blacklist               map[string]time.Time
	ipReputation            map[string]*IPReputation

	rateLimitStrategy       *RateLimitStrategy
	trafficCleaner          *TrafficCleanerImpl

	mu                      sync.RWMutex

	currentBandwidth        int64
	peakBandwidth           int64
	totalRequests           int64
	cleanedRequests         int64
	droppedBytes            int64
}

type TrafficCleanerImpl struct {
	mu               sync.RWMutex
	cleanStats       *CleanedTrafficStats
	dropPatterns     map[string]*DropPattern
}

type DropPattern struct {
	Pattern         string
	MatchCount      int64
	DroppedBytes    int64
	LastMatched     time.Time
}

func NewDDoSEnhancedProtectionService(config *DDoSProtectionConfig) *DDoSEnhancedProtectionService {
	if config == nil {
		config = &DDoSProtectionConfig{
			EnableTrafficCleaning:   true,
			EnableIPReputation:      true,
			EnableAdaptiveLimits:    true,
			EnableRateLimitStrategy: true,
			EnableBandwidthThrottle: true,
			MaxBandwidthGbps:        10.0,
			RequestsPerSecond:       100,
			BurstSize:              200,
			WindowSize:              time.Minute,
			BlockDuration:           5 * time.Minute,
			CleanupInterval:         30 * time.Minute,
			MaxIPs:                  10000,
		}
	}

	svc := &DDoSEnhancedProtectionService{
		config:           *config,
		ipStats:          make(map[string]*IPStatistics),
		trafficData:      make(map[string]*DDoSTrafficData),
		blacklist:        make(map[string]time.Time),
		ipReputation:     make(map[string]*IPReputation),
		rateLimitStrategy: &RateLimitStrategy{
			Name:           "adaptive",
			RequestsPerSec: config.RequestsPerSecond,
			BurstSize:      config.BurstSize,
			WindowSize:     config.WindowSize,
			Adaptive:       config.EnableAdaptiveLimits,
			BlockDuration:  config.BlockDuration,
		},
	}

	svc.trafficCleaner = &TrafficCleanerImpl{
		cleanStats: &CleanedTrafficStats{},
		dropPatterns: make(map[string]*DropPattern),
	}

	return svc
}

func (s *DDoSEnhancedProtectionService) CheckRequest(r *http.Request) *DDoSCheckResult {
	ip := getClientIP(r)
	now := time.Now()

	s.mu.Lock()
	s.totalRequests++
	s.mu.Unlock()

	if s.config.EnableIPReputation {
		s.mu.RLock()
		reputation, exists := s.ipReputation[ip]
		if exists && (reputation.IsBlacklisted || reputation.Tier == DDoSTierBlocked) {
			s.mu.RUnlock()
			return &DDoSCheckResult{
				Allowed:    false,
				Reason:     "ip_blacklisted",
				RetryAfter: int(reputation.Score * 3600),
			}
		}
		s.mu.RUnlock()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if expiry, exists := s.blacklist[ip]; exists {
		if now.Before(expiry) {
			s.updateIPReputationUnsafe(ip, -0.1)
			return &DDoSCheckResult{
				Allowed:    false,
				Reason:     "blacklisted",
				RetryAfter: int(time.Until(expiry).Seconds()),
			}
		}
		delete(s.blacklist, ip)
	}

	stats, exists := s.ipStats[ip]
	if !exists {
		stats = &IPStatistics{
			IP:           ip,
			RequestCount: 0,
			BlockedCount: 0,
			FirstSeen:    now,
			LastSeen:     now,
		}
		s.ipStats[ip] = stats
	}

	traffic, exists := s.trafficData[ip]
	if !exists {
		traffic = &DDoSTrafficData{
			RequestTimes: []time.Time{},
			RequestSizes: []int{},
			Methods:      []string{},
			Paths:        []string{},
		}
		s.trafficData[ip] = traffic
	}

	requestSize := getRequestSize(r)
	traffic.RequestTimes = append(traffic.RequestTimes, now)
	traffic.RequestSizes = append(traffic.RequestSizes, requestSize)
	traffic.Methods = append(traffic.Methods, r.Method)
	traffic.Paths = append(traffic.Paths, r.URL.Path)

	stats.RequestCount++
	stats.LastSeen = now

	cutoff := now.Add(-s.rateLimitStrategy.WindowSize)
	recentRequests := 0
	for _, t := range traffic.RequestTimes {
		if t.After(cutoff) {
			recentRequests++
		}
	}
	stats.Rate = float64(recentRequests)

	if len(traffic.RequestTimes) > 1000 {
		traffic.RequestTimes = traffic.RequestTimes[len(traffic.RequestTimes)-1000:]
		traffic.RequestSizes = traffic.RequestSizes[len(traffic.RequestSizes)-1000:]
		traffic.Methods = traffic.Methods[len(traffic.Methods)-1000:]
		traffic.Paths = traffic.Paths[len(traffic.Paths)-1000:]
	}

	stats.IsAnomaly = s.detectAnomaly(traffic)

	isBlocked := s.checkIfIPBlocked(ip)
	stats.IsBlacklisted = isBlocked

	limit := s.getAdaptiveLimitUnsafe()

	if stats.Rate > float64(limit) {
		stats.BlockedCount++
		s.updateIPReputationUnsafe(ip, -0.2)
		s.blacklist[ip] = now.Add(s.rateLimitStrategy.BlockDuration)
		return &DDoSCheckResult{
			Allowed:    false,
			Reason:     "rate_limit_exceeded",
			IPStats:    stats,
			RetryAfter: int(s.rateLimitStrategy.BlockDuration.Seconds()),
		}
	}

	if stats.IsAnomaly {
		s.updateIPReputationUnsafe(ip, -0.15)
		return &DDoSCheckResult{
			Allowed: false,
			Reason:  "anomaly_detected",
			IPStats: stats,
		}
	}

	s.updateIPReputationUnsafe(ip, 0.01)

	return &DDoSCheckResult{
		Allowed: true,
		IPStats: stats,
	}
}

func (s *DDoSEnhancedProtectionService) CleanTraffic(ip string, r *http.Request) *TrafficCleanResult {
	if !s.config.EnableTrafficCleaning {
		return &TrafficCleanResult{Cleaned: false}
	}

	result := &TrafficCleanResult{
		ProcessedBytes: int64(getRequestSize(r)),
	}

	maliciousPatterns := s.detectMaliciousPatterns(r)
	if maliciousPatterns.matched {
		result.Cleaned = true
		result.DropReason = maliciousPatterns.reason
		result.DroppedBytes = result.ProcessedBytes

		s.mu.Lock()
		s.cleanedRequests++
		s.droppedBytes += result.DroppedBytes
		s.mu.Unlock()

		s.trafficCleaner.mu.Lock()
		s.trafficCleaner.cleanStats.CleanedRequests++
		s.trafficCleaner.cleanStats.DroppedBytes += result.DroppedBytes

		pattern := s.trafficCleaner.dropPatterns[maliciousPatterns.reason]
		if pattern == nil {
			pattern = &DropPattern{
				Pattern:      maliciousPatterns.reason,
				DroppedBytes: 0,
			}
			s.trafficCleaner.dropPatterns[maliciousPatterns.reason] = pattern
		}
		pattern.MatchCount++
		pattern.DroppedBytes += result.DroppedBytes
		pattern.LastMatched = time.Now()
		s.trafficCleaner.mu.Unlock()

		return result
	}

	s.mu.Lock()
	s.currentBandwidth += result.ProcessedBytes
	s.mu.Unlock()

	return result
}

type maliciousPatternResult struct {
	matched bool
	reason  string
}

func (s *DDoSEnhancedProtectionService) detectMaliciousPatterns(r *http.Request) *maliciousPatternResult {
	path := r.URL.Path
	query := r.URL.RawQuery

	dangerousPatterns := map[string]string{
		`\.\./`:                              "directory_traversal",
		`\x00|\n|\r`:                         "null_byte_injection",
		`(<script|javascript:|on\w+=|<iframe`: "xss_attempt",
		`(union|select|insert|update|delete)`: "sql_injection_attempt",
		`\{.*\$`:                             "template_injection",
	}

	for pattern, reason := range dangerousPatterns {
		matched, _ := regexp.MatchString(pattern, path+query)
		if matched {
			return &maliciousPatternResult{matched: true, reason: reason}
		}
	}

	if len(path) > 2048 {
		return &maliciousPatternResult{matched: true, reason: "path_too_long"}
	}

	userAgent := r.UserAgent()
	if len(userAgent) == 0 || userAgent == "-" {
		return &maliciousPatternResult{matched: true, reason: "missing_user_agent"}
	}

	return &maliciousPatternResult{matched: false}
}

func (s *DDoSEnhancedProtectionService) GetIPReputation(ip string) *IPReputation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if reputation, exists := s.ipReputation[ip]; exists {
		return reputation
	}

	return &IPReputation{
		IP:           ip,
		Score:        0.5,
		Tier:         DDoSTierNormal,
		LastUpdated:  time.Now(),
		TrustLevel:   5,
	}
}

func (s *DDoSEnhancedProtectionService) updateIPReputation(ip string, delta float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateIPReputationUnsafe(ip, delta)
}

func (s *DDoSEnhancedProtectionService) updateIPReputationUnsafe(ip string, delta float64) {
	reputation, exists := s.ipReputation[ip]
	if !exists {
		reputation = &IPReputation{
			IP:           ip,
			Score:        0.5,
			Tier:         DDoSTierNormal,
			LastUpdated:  time.Now(),
			TrustLevel:   5,
		}
		s.ipReputation[ip] = reputation
	}

	reputation.Score = math.Max(0, math.Min(1, reputation.Score+delta))
	reputation.LastUpdated = time.Now()

	if reputation.IsWhitelisted {
		reputation.Tier = DDoSTierNormal
	} else if reputation.IsBlacklisted {
		reputation.Tier = DDoSTierBlocked
	} else if reputation.Score >= 0.8 {
		reputation.Tier = DDoSTierCritical
	} else if reputation.Score >= 0.6 {
		reputation.Tier = DDoSTierWarning
	} else {
		reputation.Tier = DDoSTierNormal
	}

	if reputation.Score >= 0.9 {
		reputation.IsBlacklisted = true
	}
}

func (s *DDoSEnhancedProtectionService) SetIPWhitelist(ip string, whitelisted bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reputation, exists := s.ipReputation[ip]
	if !exists {
		reputation = &IPReputation{
			IP:           ip,
			Score:        0.5,
			Tier:         DDoSTierNormal,
			LastUpdated:  time.Now(),
			TrustLevel:   5,
		}
		s.ipReputation[ip] = reputation
	}

	reputation.IsWhitelisted = whitelisted
	if whitelisted {
		reputation.Tier = DDoSTierNormal
		reputation.IsBlacklisted = false
		reputation.Score = 0.0
	}
}

func (s *DDoSEnhancedProtectionService) SetIPBlacklist(ip string, blacklisted bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reputation, exists := s.ipReputation[ip]
	if !exists {
		reputation = &IPReputation{
			IP:           ip,
			Score:        0.5,
			Tier:         DDoSTierNormal,
			LastUpdated:  time.Now(),
			TrustLevel:   5,
		}
		s.ipReputation[ip] = reputation
	}

	reputation.IsBlacklisted = blacklisted
	if blacklisted {
		reputation.Tier = DDoSTierBlocked
		reputation.Score = 1.0
	}
}

func (s *DDoSEnhancedProtectionService) getAdaptiveLimit(ip string) int {
	if !s.config.EnableAdaptiveLimits {
		return s.config.RequestsPerSecond
	}

	s.mu.RLock()
	reputation, exists := s.ipReputation[ip]
	if !exists {
		s.mu.RUnlock()
		return s.config.RequestsPerSecond
	}
	s.mu.RUnlock()

	baseLimit := s.config.RequestsPerSecond

	switch reputation.Tier {
	case DDoSTierNormal:
		return int(float64(baseLimit) * 1.0)
	case DDoSTierWarning:
		return int(float64(baseLimit) * 0.5)
	case DDoSTierCritical:
		return int(float64(baseLimit) * 0.2)
	case DDoSTierBlocked:
		return 0
	}

	return baseLimit
}

func (s *DDoSEnhancedProtectionService) getAdaptiveLimitUnsafe() int {
	if !s.config.EnableAdaptiveLimits {
		return s.config.RequestsPerSecond
	}

	return s.config.RequestsPerSecond
}

func (s *DDoSEnhancedProtectionService) isIPBlacklisted(ip string) bool {
	s.mu.RLock()
	reputation, exists := s.ipReputation[ip]
	if exists {
		isBlacklisted := reputation.IsBlacklisted
		s.mu.RUnlock()
		return isBlacklisted
	}
	s.mu.RUnlock()
	return false
}

func (s *DDoSEnhancedProtectionService) checkIfIPBlocked(ip string) bool {
	reputation, exists := s.ipReputation[ip]
	if exists {
		return reputation.IsBlacklisted
	}
	return false
}

func (s *DDoSEnhancedProtectionService) detectAnomaly(traffic *DDoSTrafficData) bool {
	if len(traffic.RequestTimes) < 20 {
		return false
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
	if cv < 0.1 && mean < 500 {
		return true
	}

	if len(traffic.Paths) > 10 {
		pathCounts := make(map[string]int)
		for _, path := range traffic.Paths {
			pathCounts[path]++
		}
		for _, count := range pathCounts {
			if count > len(traffic.Paths)/2 {
				return true
			}
		}
	}

	return false
}

func (s *DDoSEnhancedProtectionService) GetProtectionStats() *DDoSProtectionStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &DDoSProtectionStats{
		TotalRequests:       s.totalRequests,
		CleanedRequests:     s.cleanedRequests,
		DroppedBytes:       s.droppedBytes,
		PeakBandwidth:      s.peakBandwidth,
		ActiveIPs:           len(s.ipStats),
		BlacklistedIPs:      len(s.blacklist),
		TrackedReputations: len(s.ipReputation),
		CleanRate:          float64(s.cleanedRequests) / math.Max(1, float64(s.totalRequests)),
	}
}

func (s *DDoSEnhancedProtectionService) GetRateLimitStrategy() *RateLimitStrategy {
	return s.rateLimitStrategy
}

func (s *DDoSEnhancedProtectionService) SetRateLimitStrategy(strategy *RateLimitStrategy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rateLimitStrategy = strategy
}

func (s *DDoSEnhancedProtectionService) GetTrafficCleaner() *TrafficCleanerImpl {
	return s.trafficCleaner
}

func (s *DDoSEnhancedProtectionService) GetCleanedStats() *CleanedTrafficStats {
	s.trafficCleaner.mu.RLock()
	defer s.trafficCleaner.mu.RUnlock()

	s.mu.RLock()
	totalRequests := s.totalRequests
	s.mu.RUnlock()

	stats := &CleanedTrafficStats{
		TotalRequests:   totalRequests,
		CleanedRequests: s.trafficCleaner.cleanStats.CleanedRequests,
		DroppedBytes:    s.trafficCleaner.cleanStats.DroppedBytes,
		CleanRate:       float64(s.trafficCleaner.cleanStats.CleanedRequests) / math.Max(1, float64(totalRequests)),
	}

	return stats
}

func getRequestSize(r *http.Request) int {
	if r.ContentLength > 0 {
		return int(r.ContentLength)
	}

	if r.Body != nil {
		return 512
	}

	return 0
}

type DDoSProtectionStats struct {
	TotalRequests       int64
	CleanedRequests     int64
	DroppedBytes        int64
	PeakBandwidth       int64
	ActiveIPs           int
	BlacklistedIPs     int
	TrackedReputations int
	CleanRate          float64
}
