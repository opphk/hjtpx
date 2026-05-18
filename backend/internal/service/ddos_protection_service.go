package service

import (
	"math"
	"net/http"
	"sync"
	"time"
)

type DDoSCheckResult struct {
	Allowed    bool
	Reason     string
	IPStats    *IPStatistics
	RetryAfter int
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
}

type DDoSTrafficData struct {
	RequestTimes []time.Time
	RequestSizes []int
	Methods      []string
	Paths        []string
}

type DDoSProtectionService struct {
	ipStats        map[string]*IPStatistics
	trafficData    map[string]*DDoSTrafficData
	blacklist      map[string]time.Time
	mu             sync.RWMutex
	maxIPs         int
	requestsPerMin int
	cleanupPeriod  time.Duration
}

func NewDDoSProtectionService() *DDoSProtectionService {
	service := &DDoSProtectionService{
		ipStats:        make(map[string]*IPStatistics),
		trafficData:    make(map[string]*DDoSTrafficData),
		blacklist:      make(map[string]time.Time),
		maxIPs:         10000,
		requestsPerMin: 100,
		cleanupPeriod:  1 * time.Hour,
	}
	go service.cleanupLoop()
	return service
}

func (s *DDoSProtectionService) CheckRequest(r *http.Request) *DDoSCheckResult {
	ip := getClientIP(r)
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	if expiry, exists := s.blacklist[ip]; exists {
		if now.Before(expiry) {
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
		}
		s.trafficData[ip] = traffic
	}

	traffic.RequestTimes = append(traffic.RequestTimes, now)
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

	stats.IsAnomaly = s.detectAnomaly(traffic)

	if stats.Rate > float64(s.requestsPerMin) {
		stats.BlockedCount++
		return &DDoSCheckResult{
			Allowed: false,
			Reason:  "rate_limit",
			IPStats: stats,
		}
	}

	if stats.IsAnomaly {
		return &DDoSCheckResult{
			Allowed: false,
			Reason:  "anomaly_detected",
			IPStats: stats,
		}
	}

	return &DDoSCheckResult{
		Allowed: true,
		IPStats: stats,
	}
}

func (s *DDoSProtectionService) detectAnomaly(traffic *DDoSTrafficData) bool {
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

	return false
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
