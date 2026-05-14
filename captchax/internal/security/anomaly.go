package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type AnomalyType string

const (
	AnomalyBruteForce    AnomalyType = "brute_force"
	AnomalyCredentialStuffing AnomalyType = "credential_stuffing"
	AnomalyFrequentFailures AnomalyType = "frequent_failures"
	AnomalySuspiciousIP   AnomalyType = "suspicious_ip"
	AnomalyBotLike        AnomalyType = "bot_like"
	AnomalyRapidRequests  AnomalyType = "rapid_requests"
	AnomalyUnusualPattern AnomalyType = "unusual_pattern"
)

type AnomalyEvent struct {
	ID           string
	Type         AnomalyType
	IPAddress    string
	UserID       string
	SessionID    string
	Timestamp    time.Time
	Severity     int
	Score        float64
	Description  string
	Details      map[string]interface{}
	Action       AnomalyAction
}

type AnomalyAction string

const (
	ActionAllow       AnomalyAction = "allow"
	ActionWarn         AnomalyAction = "warn"
	ActionBlock        AnomalyAction = "block"
	ActionCaptcha      AnomalyAction = "captcha"
	ActionThrottle     AnomalyAction = "throttle"
	ActionLockout      AnomalyAction = "lockout"
)

type AnomalyDetectorConfig struct {
	BruteForceThreshold     int
	BruteForceWindow       time.Duration
	FailureCountThreshold  int
	FailureWindow          time.Duration
	RequestRateThreshold   float64
	RequestRateWindow      time.Duration
	SuspiciousScoreThreshold float64
	BlockDuration          time.Duration
	AutoBlockEnabled       bool
	EnableLearning         bool
	MinSamplesForLearning  int
}

var defaultAnomalyConfig = &AnomalyDetectorConfig{
	BruteForceThreshold:      10,
	BruteForceWindow:         10 * time.Minute,
	FailureCountThreshold:     5,
	FailureWindow:             5 * time.Minute,
	RequestRateThreshold:      100,
	RequestRateWindow:         1 * time.Minute,
	SuspiciousScoreThreshold:  50,
	BlockDuration:             30 * time.Minute,
	AutoBlockEnabled:          true,
	EnableLearning:            true,
	MinSamplesForLearning:     100,
}

type IPAnomalyRecord struct {
	IPAddress       string
	LoginAttempts   []*AttemptRecord
	Failures        []*AttemptRecord
	Requests        []*RequestRecord
	SuspiciousCount int
	LastActivity    time.Time
	Blocked         bool
	BlockedUntil    time.Time
	RiskScore       float64
	Patterns        []string
}

type AttemptRecord struct {
	Timestamp   time.Time
	Success     bool
	UserID      string
	Action      string
	UserAgent   string
}

type RequestRecord struct {
	Timestamp    time.Time
	Path         string
	Method       string
	ResponseCode int
	Duration     time.Duration
}

type AnomalyDetector struct {
	config       *AnomalyDetectorConfig
	ipRecords    map[string]*IPAnomalyRecord
	mu           sync.RWMutex
	blockedIPs   map[string]time.Time
	knownPatterns []string
}

func NewAnomalyDetector(config *AnomalyDetectorConfig) *AnomalyDetector {
	if config == nil {
		config = defaultAnomalyConfig
	}
	return &AnomalyDetector{
		config:        config,
		ipRecords:    make(map[string]*IPAnomalyRecord),
		blockedIPs:   make(map[string]time.Time),
		knownPatterns: []string{
			"sql_injection",
			"xss_attempt",
			"path_traversal",
			"command_injection",
			"brute_force",
			"rapid_requests",
		},
	}
}

func (d *AnomalyDetector) RecordAttempt(ip, userID, action, userAgent string, success bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	record := d.getOrCreateRecord(ip)
	record.LastActivity = time.Now()

	attempt := &AttemptRecord{
		Timestamp: time.Now(),
		Success:   success,
		UserID:    userID,
		Action:    action,
		UserAgent: userAgent,
	}

	if success {
		record.LoginAttempts = append(record.LoginAttempts, attempt)
	} else {
		record.Failures = append(record.Failures, attempt)
	}

	d.cleanupOldRecords(record)
	d.calculateRiskScore(record)
}

func (d *AnomalyDetector) RecordRequest(ip, path, method string, responseCode int, duration time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()

	record := d.getOrCreateRecord(ip)
	record.LastActivity = time.Now()

	record.Requests = append(record.Requests, &RequestRecord{
		Timestamp:    time.Now(),
		Path:         path,
		Method:       method,
		ResponseCode: responseCode,
		Duration:     duration,
	})

	d.cleanupOldRecords(record)
	d.calculateRiskScore(record)
}

func (d *AnomalyDetector) getOrCreateRecord(ip string) *IPAnomalyRecord {
	record, exists := d.ipRecords[ip]
	if !exists {
		record = &IPAnomalyRecord{
			IPAddress:     ip,
			LoginAttempts: make([]*AttemptRecord, 0),
			Failures:      make([]*AttemptRecord, 0),
			Requests:      make([]*RequestRecord, 0),
		}
		d.ipRecords[ip] = record
	}
	return record
}

func (d *AnomalyDetector) cleanupOldRecords(record *IPAnomalyRecord) {
	now := time.Now()

	var recentLogins []*AttemptRecord
	for _, a := range record.LoginAttempts {
		if now.Sub(a.Timestamp) < d.config.BruteForceWindow {
			recentLogins = append(recentLogins, a)
		}
	}
	record.LoginAttempts = recentLogins

	var recentFailures []*AttemptRecord
	for _, f := range record.Failures {
		if now.Sub(f.Timestamp) < d.config.FailureWindow {
			recentFailures = append(recentFailures, f)
		}
	}
	record.Failures = recentFailures

	var recentRequests []*RequestRecord
	for _, r := range record.Requests {
		if now.Sub(r.Timestamp) < d.config.RequestRateWindow {
			recentRequests = append(recentRequests, r)
		}
	}
	record.Requests = recentRequests
}

func (d *AnomalyDetector) calculateRiskScore(record *IPAnomalyRecord) {
	var score float64

	now := time.Now()
	loginWindowStart := now.Add(-d.config.BruteForceWindow)
	loginAttempts := 0
	for _, a := range record.LoginAttempts {
		if a.Timestamp.After(loginWindowStart) {
			loginAttempts++
		}
	}
	if loginAttempts > 0 {
		score += float64(loginAttempts) / float64(d.config.BruteForceThreshold) * 30
	}

	failureWindowStart := now.Add(-d.config.FailureWindow)
	failureCount := 0
	uniqueUsers := make(map[string]bool)
	for _, f := range record.Failures {
		if f.Timestamp.After(failureWindowStart) {
			failureCount++
			if f.UserID != "" {
				uniqueUsers[f.UserID] = true
			}
		}
	}
	if failureCount > 0 {
		score += float64(failureCount) / float64(d.config.FailureCountThreshold) * 25
	}

	if len(uniqueUsers) > 3 {
		score += 20
	}

	requestWindowStart := now.Add(-d.config.RequestRateWindow)
	requestCount := 0
	for _, r := range record.Requests {
		if r.Timestamp.After(requestWindowStart) {
			requestCount++
		}
	}
	if requestCount > 0 {
		score += math.Min(float64(requestCount)/d.config.RequestRateThreshold*15, 15)
	}

	if record.Blocked {
		score += 10
	}

	score = math.Min(score, 100)
	record.RiskScore = score
}

func (d *AnomalyDetector) Detect(ip string) *AnomalyEvent {
	d.mu.Lock()
	defer d.mu.Unlock()

	record, exists := d.ipRecords[ip]
	if !exists {
		return nil
	}

	if blockedUntil, blocked := d.blockedIPs[ip]; blocked && time.Now().Before(blockedUntil) {
		return &AnomalyEvent{
			ID:          generateEventID(),
			Type:        AnomalyBruteForce,
			IPAddress:   ip,
			Timestamp:   time.Now(),
			Severity:    10,
			Score:       100,
			Description: fmt.Sprintf("IP %s is blocked until %v", ip, blockedUntil),
			Action:      ActionBlock,
		}
	}

	var anomalies []*AnomalyEvent

	now := time.Now()
	loginWindowStart := now.Add(-d.config.BruteForceWindow)
	recentLogins := 0
	for _, a := range record.LoginAttempts {
		if a.Timestamp.After(loginWindowStart) {
			recentLogins++
		}
	}
	if recentLogins >= d.config.BruteForceThreshold {
		anomalies = append(anomalies, &AnomalyEvent{
			ID:          generateEventID(),
			Type:        AnomalyBruteForce,
			IPAddress:   ip,
			Timestamp:   time.Now(),
			Severity:    9,
			Score:       85,
			Description: fmt.Sprintf("Potential brute force attack detected: %d login attempts in %v", recentLogins, d.config.BruteForceWindow),
			Details: map[string]interface{}{
				"login_attempts": recentLogins,
				"window":         d.config.BruteForceWindow.String(),
			},
			Action: ActionBlock,
		})
	}

	failureWindowStart := now.Add(-d.config.FailureWindow)
	recentFailures := 0
	uniqueUsers := make(map[string]bool)
	for _, f := range record.Failures {
		if f.Timestamp.After(failureWindowStart) {
			recentFailures++
			if f.UserID != "" {
				uniqueUsers[f.UserID] = true
			}
		}
	}
	if recentFailures >= d.config.FailureCountThreshold {
		anomalies = append(anomalies, &AnomalyEvent{
			ID:          generateEventID(),
			Type:        AnomalyFrequentFailures,
			IPAddress:   ip,
			Timestamp:   time.Now(),
			Severity:    7,
			Score:       70,
			Description: fmt.Sprintf("Multiple failed attempts detected: %d failures in %v", recentFailures, d.config.FailureWindow),
			Details: map[string]interface{}{
				"failure_count":   recentFailures,
				"unique_users":    len(uniqueUsers),
				"window":          d.config.FailureWindow.String(),
			},
			Action: ActionCaptcha,
		})
	}

	if len(uniqueUsers) > 3 && recentFailures > 5 {
		anomalies = append(anomalies, &AnomalyEvent{
			ID:          generateEventID(),
			Type:        AnomalyCredentialStuffing,
			IPAddress:   ip,
			Timestamp:   time.Now(),
			Severity:    8,
			Score:       80,
			Description: fmt.Sprintf("Potential credential stuffing attack: %d unique users attempted", len(uniqueUsers)),
			Details: map[string]interface{}{
				"unique_users": len(uniqueUsers),
				"failure_count": recentFailures,
			},
			Action: ActionBlock,
		})
	}

	requestWindowStart := now.Add(-d.config.RequestRateWindow)
	requestCount := 0
	for _, r := range record.Requests {
		if r.Timestamp.After(requestWindowStart) {
			requestCount++
		}
	}
	if float64(requestCount) > d.config.RequestRateThreshold {
		anomalies = append(anomalies, &AnomalyEvent{
			ID:          generateEventID(),
			Type:        AnomalyRapidRequests,
			IPAddress:   ip,
			Timestamp:   time.Now(),
			Severity:    6,
			Score:       60,
			Description: fmt.Sprintf("High request rate detected: %d requests in %v", requestCount, d.config.RequestRateWindow),
			Details: map[string]interface{}{
				"request_count": requestCount,
				"window":        d.config.RequestRateWindow.String(),
				"threshold":     d.config.RequestRateThreshold,
			},
			Action: ActionThrottle,
		})
	}

	if len(record.Patterns) > 0 {
		anomalies = append(anomalies, &AnomalyEvent{
			ID:          generateEventID(),
			Type:        AnomalySuspiciousIP,
			IPAddress:   ip,
			Timestamp:   time.Now(),
			Severity:    5,
			Score:       50,
			Description: fmt.Sprintf("Suspicious patterns detected: %v", record.Patterns),
			Details: map[string]interface{}{
				"patterns": record.Patterns,
			},
			Action: ActionWarn,
		})
	}

	if record.RiskScore >= d.config.SuspiciousScoreThreshold {
		action := ActionCaptcha
		if record.RiskScore >= 80 {
			action = ActionBlock
		}
		anomalies = append(anomalies, &AnomalyEvent{
			ID:          generateEventID(),
			Type:        AnomalyUnusualPattern,
			IPAddress:   ip,
			Timestamp:   time.Now(),
			Severity:    int(record.RiskScore / 10),
			Score:       record.RiskScore,
			Description: fmt.Sprintf("Elevated risk score: %.2f", record.RiskScore),
			Details: map[string]interface{}{
				"risk_score": record.RiskScore,
			},
			Action: action,
		})
	}

	if len(anomalies) > 0 {
		highest := anomalies[0]
		for _, a := range anomalies[1:] {
			if a.Score > highest.Score {
				highest = a
			}
		}
		return highest
	}

	return nil
}

func (d *AnomalyDetector) BlockIP(ip string, duration time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.blockedIPs[ip] = time.Now().Add(duration)

	if record, exists := d.ipRecords[ip]; exists {
		record.Blocked = true
		record.BlockedUntil = time.Now().Add(duration)
	}

	return nil
}

func (d *AnomalyDetector) UnblockIP(ip string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.blockedIPs, ip)

	if record, exists := d.ipRecords[ip]; exists {
		record.Blocked = false
	}

	return nil
}

func (d *AnomalyDetector) IsBlocked(ip string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if blockedUntil, blocked := d.blockedIPs[ip]; blocked && time.Now().Before(blockedUntil) {
		return true
	}
	return false
}

func (d *AnomalyDetector) GetRiskScore(ip string) float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if record, exists := d.ipRecords[ip]; exists {
		return record.RiskScore
	}
	return 0
}

func (d *AnomalyDetector) GetBlockedIPs() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var blocked []string
	now := time.Now()
	for ip, until := range d.blockedIPs {
		if now.Before(until) {
			blocked = append(blocked, ip)
		}
	}
	return blocked
}

func (d *AnomalyDetector) Cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	for ip, until := range d.blockedIPs {
		if now.After(until) {
			delete(d.blockedIPs, ip)
			if record, exists := d.ipRecords[ip]; exists {
				record.Blocked = false
			}
		}
	}

	for ip, record := range d.ipRecords {
		if now.Sub(record.LastActivity) > 24*time.Hour && !record.Blocked {
			delete(d.ipRecords, ip)
		}
	}
}

type AnomalyDetectorMiddleware struct {
	detector *AnomalyDetector
	logger   *AuditLoggerV2
	config   *AnomalyDetectorConfig
}

func NewAnomalyDetectorMiddleware(detector *AnomalyDetector, logger *AuditLoggerV2, config *AnomalyDetectorConfig) *AnomalyDetectorMiddleware {
	if config == nil {
		config = defaultAnomalyConfig
	}
	return &AnomalyDetectorMiddleware{
		detector: detector,
		logger:   logger,
		config:   config,
	}
}

func (m *AnomalyDetectorMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := GetClientIP(c)

		if m.detector.IsBlocked(ip) {
			LogSecurityViolation(m.logger, "brute_force", ip, map[string]interface{}{
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			})

			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "access_denied",
				"message": "Your IP has been temporarily blocked due to suspicious activity",
			})
			return
		}

		start := time.Now()

		c.Next()

		duration := time.Since(start)

		m.detector.RecordRequest(ip, c.Request.URL.Path, c.Request.Method, c.Writer.Status(), duration)

		if c.Writer.Status() >= 400 {
			m.detector.RecordAttempt(ip, "", c.Request.URL.Path, c.GetHeader("User-Agent"), false)
		}

		if event := m.detector.Detect(ip); event != nil {
			m.handleAnomaly(c, event)
		}
	}
}

func (m *AnomalyDetectorMiddleware) handleAnomaly(c *gin.Context, event *AnomalyEvent) {
	ip := event.IPAddress

	switch event.Action {
	case ActionBlock:
		if m.config.AutoBlockEnabled {
			m.detector.BlockIP(ip, m.config.BlockDuration)
			LogSecurityViolation(m.logger, "brute_force", ip, map[string]interface{}{
				"event_type": event.Type,
				"severity":   event.Severity,
				"score":      event.Score,
				"details":    event.Details,
			})
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":   "access_denied",
			"message": "Your request has been blocked due to suspicious activity",
		})

	case ActionThrottle:
		c.Header("X-RateLimit-Remaining", "0")
		c.Header("Retry-After", "60")
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error":   "rate_limit_exceeded",
			"message": "Too many requests, please slow down",
		})

	case ActionCaptcha:
		c.Header("X-Captcha-Required", "true")
		c.Header("X-Risk-Score", fmt.Sprintf("%.2f", event.Score))
		c.Set("captcha_required", true)
		c.Set("risk_score", event.Score)

	case ActionWarn:
		c.Set("suspicious_activity", true)
		c.Set("risk_score", event.Score)
	}
}

func GetClientIP(c *gin.Context) string {
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	xri := c.GetHeader("X-Real-IP")
	if xri != "" {
		return xri
	}
	ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	if ip != "" {
		return ip
	}
	return c.ClientIP()
}

func HashIP(ip string) string {
	h := sha256.New()
	h.Write([]byte(ip))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

type AnomalyReporter struct {
	detector *AnomalyDetector
	mu       sync.RWMutex
	history  []*AnomalyEvent
	maxSize  int
}

func NewAnomalyReporter(detector *AnomalyDetector) *AnomalyReporter {
	return &AnomalyReporter{
		detector: detector,
		history:  make([]*AnomalyEvent, 0),
		maxSize:  1000,
	}
}

func (r *AnomalyReporter) Report(event *AnomalyEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.history = append(r.history, event)
	if len(r.history) > r.maxSize {
		r.history = r.history[len(r.history)-r.maxSize:]
	}
}

func (r *AnomalyReporter) GetHistory(start, end time.Time) []*AnomalyEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*AnomalyEvent
	for _, e := range r.history {
		if e.Timestamp.After(start) && e.Timestamp.Before(end) {
			results = append(results, e)
		}
	}
	return results
}

func (r *AnomalyReporter) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := map[string]interface{}{
		"total_events": len(r.history),
		"by_type":      make(map[string]int),
		"by_action":   make(map[string]int),
	}

	typeCounts := make(map[string]int)
	actionCounts := make(map[string]int)

	for _, e := range r.history {
		typeCounts[string(e.Type)]++
		actionCounts[string(e.Action)]++
	}

	stats["by_type"] = typeCounts
	stats["by_action"] = actionCounts

	return stats
}

type ThreatIntel struct {
	blockedRanges []*net.IPNet
	mu            sync.RWMutex
}

func NewThreatIntel() *ThreatIntel {
	return &ThreatIntel{
		blockedRanges: make([]*net.IPNet, 0),
	}
}

func (t *ThreatIntel) BlockRange(cidr string) error {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.blockedRanges = append(t.blockedRanges, ipnet)
	return nil
}

func (t *ThreatIntel) IsIPBlocked(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, ipnet := range t.blockedRanges {
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

func (t *ThreatIntel) UnblockRange(cidr string) error {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	var newRanges []*net.IPNet
	for _, n := range t.blockedRanges {
		if n.String() != ipnet.String() {
			newRanges = append(newRanges, n)
		}
	}
	t.blockedRanges = newRanges
	return nil
}

func StartAnomalyCleanup(ctx context.Context, detector *AnomalyDetector, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				detector.Cleanup()
			}
		}
	}()
}
