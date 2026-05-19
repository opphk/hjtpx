package service

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

type FingerprintRequest struct {
	UserAgent string
	Headers   map[string]string
	IP        string
	IPAddress string
}

type BotFingerprintV2 struct {
	patterns map[string]bool
}

func NewBotFingerprintV2() *BotFingerprintV2 {
	return &BotFingerprintV2{
		patterns: map[string]bool{
			"curl":         true,
			"python":       true,
			"wget":         true,
			"httpie":       true,
			"go-http-client": true,
			"java":         true,
			"ruby":         true,
			"node-fetch":   true,
		},
	}
}

func (bf *BotFingerprintV2) CheckRequest(req *FingerprintRequest) bool {
	if req == nil {
		return false
	}
	
	ua := req.UserAgent
	for pattern := range bf.patterns {
		if containsIgnoreCase(ua, pattern) {
			return true
		}
	}
	
	return false
}

func (bf *BotFingerprintV2) AnalyzeRequest(req *FingerprintRequest) *FingerprintResult {
	result := &FingerprintResult{
		Fingerprint: fmt.Sprintf("fp_%d", time.Now().UnixNano()),
		IsBot:       bf.CheckRequest(req),
		Confidence:  0.8,
	}
	return result
}

type FingerprintResult struct {
	Fingerprint string
	IsBot       bool
	Confidence  float64
	Features    *BotFeatures
}

type BotFeatures struct {
	Webdriver     bool
	Headless      bool
	Automation    bool
	Selenium      bool
	PhantomJS     bool
	Puppeteer     bool
}

func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

type DDoSProtectionService struct {
	rateLimiter  *RateLimiter
	blacklist    map[string]time.Time
	whitelist    map[string]bool
	attackCount  map[string]int
}

func NewDDoSProtectionService() *DDoSProtectionService {
	return &DDoSProtectionService{
		rateLimiter:  NewRateLimiter(100, 10*time.Second),
		blacklist:    make(map[string]time.Time),
		whitelist:    make(map[string]bool),
		attackCount:  make(map[string]int),
	}
}

func (d *DDoSProtectionService) CheckRequest(req *http.Request) *DDoSCheckResult {
	ip := getDDoSClientIP(req)
	
	if d.whitelist[ip] {
		return &DDoSCheckResult{Allowed: true, Reason: "whitelisted"}
	}
	
	if expiry, exists := d.blacklist[ip]; exists && time.Now().Before(expiry) {
		return &DDoSCheckResult{Allowed: false, Reason: "blacklisted"}
	}
	
	if d.rateLimiter.Allow(ip) {
		return &DDoSCheckResult{Allowed: true}
	}
	return &DDoSCheckResult{Allowed: false, Reason: "rate_limited"}
}

func (d *DDoSProtectionService) GetGlobalStats() map[string]interface{} {
	return map[string]interface{}{
		"total_ips":    len(d.attackCount),
		"blacklist_size": len(d.blacklist),
		"whitelist_size": len(d.whitelist),
	}
}

func (d *DDoSProtectionService) SetAttackThreshold(ip string, threshold int) {
	d.attackCount[ip] = threshold
}

func (d *DDoSProtectionService) AddToBlacklist(ip string, reason string, duration time.Duration) {
	d.blacklist[ip] = time.Now().Add(duration)
}

func (d *DDoSProtectionService) RemoveFromBlacklist(ip string) {
	delete(d.blacklist, ip)
}

func (d *DDoSProtectionService) AddToWhitelist(ip string) {
	d.whitelist[ip] = true
}

func (d *DDoSProtectionService) RemoveFromWhitelist(ip string) {
	delete(d.whitelist, ip)
}

type DDoSCheckResult struct {
	Allowed bool
	Reason  string
}

type RateLimiter struct {
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	now := time.Now()
	requests := rl.requests[ip]
	
	var validRequests []time.Time
	for _, t := range requests {
		if now.Sub(t) < rl.window {
			validRequests = append(validRequests, t)
		}
	}
	
	if len(validRequests) >= rl.limit {
		rl.requests[ip] = validRequests
		return false
	}
	
	validRequests = append(validRequests, now)
	rl.requests[ip] = validRequests
	return true
}

func getDDoSClientIP(req *http.Request) string {
	xff := req.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}
	ip := req.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}
	return req.RemoteAddr
}
