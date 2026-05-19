package service

import (
	"net/http"
	"sync"
	"time"
)

type DDoSProtectionService struct {
	maxIPs     int
	ipStats    map[string]*IPStats
	blacklist  map[string]time.Time
	whitelist  map[string]bool
	mu         sync.RWMutex
}

type IPStats struct {
	RequestCount    int
	FirstSeen       time.Time
	LastSeen        time.Time
	Reputation      int
	AnomalyScore    float64
}

type DDoSCheckResult struct {
	Allowed bool
	Reason  string
}

var botUserAgents = []string{
	"curl",
	"Wget",
	"python-requests",
	"Go-http-client",
	"Java/",
	"Apache-HttpClient",
}

func NewDDoSProtectionService() *DDoSProtectionService {
	return &DDoSProtectionService{
		maxIPs:     10000,
		ipStats:    make(map[string]*IPStats),
		blacklist:  make(map[string]time.Time),
		whitelist:  make(map[string]bool),
	}
}

func (s *DDoSProtectionService) CheckRequest(req *http.Request) *DDoSCheckResult {
	ip := extractIP(req.RemoteAddr)
	
	s.mu.RLock()
	if s.whitelist[ip] {
		s.mu.RUnlock()
		return &DDoSCheckResult{Allowed: true, Reason: "whitelisted"}
	}
	if expireTime, ok := s.blacklist[ip]; ok {
		if time.Now().Before(expireTime) {
			s.mu.RUnlock()
			return &DDoSCheckResult{Allowed: false, Reason: "blacklisted"}
		}
	}
	s.mu.RUnlock()

	if s.isBot(req) {
		return &DDoSCheckResult{Allowed: false, Reason: "bot_detected"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	stats := s.getOrCreateStats(ip)
	stats.RequestCount++
	stats.LastSeen = time.Now()

	rateLimit := s.getDynamicRateLimit(stats.Reputation, stats.AnomalyScore)
	if stats.RequestCount > rateLimit {
		return &DDoSCheckResult{Allowed: false, Reason: "rate_limit"}
	}

	return &DDoSCheckResult{Allowed: true, Reason: ""}
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

func (s *DDoSProtectionService) GetIPStats(ip string) *IPStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ipStats[ip]
}

func (s *DDoSProtectionService) GetGlobalStats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]int{
		"total_ips":       len(s.ipStats),
		"blacklist_count": len(s.blacklist),
		"whitelist_count": len(s.whitelist),
	}
}

func (s *DDoSProtectionService) getDynamicRateLimit(reputation int, anomalyScore float64) int {
	baseLimit := 60
	if reputation < 50 {
		baseLimit = 18
	}
	if anomalyScore > 0.5 {
		baseLimit /= 2
	}
	return baseLimit
}

func (s *DDoSProtectionService) countUnique(items []string) int {
	seen := make(map[string]bool)
	for _, item := range items {
		seen[item] = true
	}
	return len(seen)
}

func (s *DDoSProtectionService) isBot(req *http.Request) bool {
	ua := req.Header.Get("User-Agent")
	for _, botUA := range botUserAgents {
		if containsIgnoreCase(ua, botUA) {
			return true
		}
	}
	return false
}

func (s *DDoSProtectionService) getOrCreateStats(ip string) *IPStats {
	stats, ok := s.ipStats[ip]
	if !ok {
		stats = &IPStats{
			RequestCount: 0,
			FirstSeen:    time.Now(),
			LastSeen:     time.Now(),
			Reputation:   s.calculateReputation(ip),
			AnomalyScore: 0.0,
		}
		s.ipStats[ip] = stats
	}
	return stats
}

func (s *DDoSProtectionService) calculateReputation(ip string) int {
	if isPrivateIP(ip) {
		return 85
	}
	return 50 + int(time.Now().Unix())%30
}

func extractIP(remoteAddr string) string {
	for i := len(remoteAddr) - 1; i >= 0; i-- {
		if remoteAddr[i] == ':' {
			return remoteAddr[:i]
		}
	}
	return remoteAddr
}

func containsIgnoreCase(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			tc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if tc >= 'A' && tc <= 'Z' {
				tc += 32
			}
			if sc != tc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}


