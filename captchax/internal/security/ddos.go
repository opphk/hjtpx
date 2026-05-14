package security

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type DDoSProtectionConfig struct {
	Enabled             bool
	ConnectionLimit     int
	RequestsPerSecond   int
	BurstSize           int
	BlockDuration       time.Duration
	CleanupInterval     time.Duration
	EnableGeoBlocking   bool
	AllowedCountries    []string
	BlockedCountries    []string
	EnableTorDetection  bool
	MaxConcurrentPerIP  int
	ConnectionTimeout   time.Duration
}

var defaultDDoSConfig = &DDoSProtectionConfig{
	Enabled:             true,
	ConnectionLimit:     1000,
	RequestsPerSecond:   100,
	BurstSize:           20,
	BlockDuration:       15 * time.Minute,
	CleanupInterval:     1 * time.Minute,
	EnableGeoBlocking:   false,
	MaxConcurrentPerIP: 10,
	ConnectionTimeout:  30 * time.Second,
}

type IPTracker struct {
	Connections    int32
	Requests       []time.Time
	LastRequest    time.Time
	Blocked        bool
	BlockedUntil   time.Time
	Tokens         int
	LastCleanup    time.Time
}

type DDoSProtector struct {
	config    *DDoSProtectionConfig
	trackers  map[string]*IPTracker
	mu        sync.RWMutex
	stats     *DDoSStats
	stopChan  chan struct{}
}

type DDoSStats struct {
	TotalConnections    uint64
	BlockedConnections  uint64
	RateLimitedRequests uint64
	ActiveConnections   uint64
	BlockedIPs          uint64
	mu                  sync.RWMutex
}

func (s *DDoSStats) Copy() DDoSStatsCopy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return DDoSStatsCopy{
		TotalConnections:    s.TotalConnections,
		BlockedConnections:  s.BlockedConnections,
		RateLimitedRequests: s.RateLimitedRequests,
		ActiveConnections:   s.ActiveConnections,
		BlockedIPs:          s.BlockedIPs,
	}
}

type DDoSStatsCopy struct {
	TotalConnections    uint64
	BlockedConnections  uint64
	RateLimitedRequests uint64
	ActiveConnections   uint64
	BlockedIPs          uint64
}

func NewDDoSProtector(config *DDoSProtectionConfig) *DDoSProtector {
	if config == nil {
		config = defaultDDoSConfig
	}

	if config.CleanupInterval <= 0 {
		config.CleanupInterval = defaultDDoSConfig.CleanupInterval
	}

	protector := &DDoSProtector{
		config:   config,
		trackers: make(map[string]*IPTracker),
		stats: &DDoSStats{},
		stopChan: make(chan struct{}),
	}

	if config.Enabled {
		go protector.cleanup()
	}

	return protector
}

func (p *DDoSProtector) cleanup() {
	ticker := time.NewTicker(p.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.cleanupOldEntries()
		}
	}
}

func (p *DDoSProtector) cleanupOldEntries() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	window := time.Minute * 5

	for ip, tracker := range p.trackers {
		if tracker.Blocked && now.After(tracker.BlockedUntil) {
			tracker.Blocked = false
			p.stats.mu.Lock()
			p.stats.BlockedIPs--
			p.stats.mu.Unlock()
		}

		if !tracker.Blocked && tracker.Connections == 0 {
			lastActivity := tracker.LastRequest
			if len(tracker.Requests) > 0 {
				lastActivity = tracker.Requests[len(tracker.Requests)-1]
			}
			if now.Sub(lastActivity) > window {
				delete(p.trackers, ip)
			}
		}

		if now.Sub(tracker.LastCleanup) > time.Minute {
			var recent []time.Time
			for _, t := range tracker.Requests {
				if now.Sub(t) < window {
					recent = append(recent, t)
				}
			}
			tracker.Requests = recent
			tracker.LastCleanup = now
		}
	}
}

func (p *DDoSProtector) getOrCreateTracker(ip string) *IPTracker {
	p.mu.RLock()
	tracker, exists := p.trackers[ip]
	p.mu.RUnlock()

	if exists {
		return tracker
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if tracker, exists = p.trackers[ip]; exists {
		return tracker
	}

	tracker = &IPTracker{
		Requests:     make([]time.Time, 0),
		LastCleanup:  time.Now(),
		Tokens:       p.config.BurstSize,
	}
	p.trackers[ip] = tracker
	return tracker
}

func (p *DDoSProtector) CheckConnection(ip string) (bool, string) {
	if !p.config.Enabled {
		return true, ""
	}

	atomic.AddUint64(&p.stats.TotalConnections, 1)

	tracker := p.getOrCreateTracker(ip)

	if tracker.Blocked {
		if time.Now().Before(tracker.BlockedUntil) {
			atomic.AddUint64(&p.stats.BlockedConnections, 1)
			return false, fmt.Sprintf("IP %s is blocked until %v", ip, tracker.BlockedUntil)
		}
		tracker.Blocked = false
	}

	connCount := atomic.LoadInt32(&tracker.Connections)
	if connCount >= int32(p.config.MaxConcurrentPerIP) {
		p.blockIP(ip, "Too many concurrent connections")
		return false, fmt.Sprintf("Too many concurrent connections from %s", ip)
	}

	return true, ""
}

func (p *DDoSProtector) CheckRateLimit(ip string) (bool, string, int) {
	if !p.config.Enabled {
		return true, "", p.config.RequestsPerSecond
	}

	tracker := p.getOrCreateTracker(ip)

	now := time.Now()
	window := time.Second

	var recentRequests []time.Time
	for _, t := range tracker.Requests {
		if now.Sub(t) < window {
			recentRequests = append(recentRequests, t)
		}
	}

	currentRate := len(recentRequests)

	if tracker.Tokens > 0 {
		tracker.Tokens--
		tracker.Requests = append(tracker.Requests, now)
		atomic.AddUint64(&p.stats.ActiveConnections, 1)
		return true, "", tracker.Tokens
	}

	if currentRate >= p.config.RequestsPerSecond {
		atomic.AddUint64(&p.stats.RateLimitedRequests, 1)
		remaining := time.Until(recentRequests[0].Add(window))
		return false, fmt.Sprintf("Rate limit exceeded for %s, retry after %v", ip, remaining), 0
	}

	tracker.Tokens = p.config.BurstSize - 1
	tracker.Requests = append(tracker.Requests, now)

	return true, "", tracker.Tokens
}

func (p *DDoSProtector) blockIP(ip string, reason string) {
	tracker := p.getOrCreateTracker(ip)
	tracker.Blocked = true
	tracker.BlockedUntil = time.Now().Add(p.config.BlockDuration)
	atomic.AddUint64(&p.stats.BlockedIPs, 1)
}

func (p *DDoSProtector) IncrementConnections(ip string) {
	tracker := p.getOrCreateTracker(ip)
	atomic.AddInt32(&tracker.Connections, 1)
}

func (p *DDoSProtector) DecrementConnections(ip string) {
	tracker := p.getOrCreateTracker(ip)
	atomic.AddInt32(&tracker.Connections, -1)
}

func (p *DDoSProtector) IsIPBlocked(ip string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if tracker, exists := p.trackers[ip]; exists {
		return tracker.Blocked && time.Now().Before(tracker.BlockedUntil)
	}
	return false
}

func (p *DDoSProtector) BlockIP(ip string, duration time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	tracker := p.getOrCreateTracker(ip)
	tracker.Blocked = true
	tracker.BlockedUntil = time.Now().Add(duration)
	return nil
}

func (p *DDoSProtector) UnblockIP(ip string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if tracker, exists := p.trackers[ip]; exists {
		tracker.Blocked = false
	}
	return nil
}

func (p *DDoSProtector) GetStats() *DDoSStats {
	stats := p.stats.Copy()

	p.mu.RLock()
	defer p.mu.RUnlock()

	var activeConns int32
	var blocked int
	for _, tracker := range p.trackers {
		activeConns += tracker.Connections
		if tracker.Blocked {
			blocked++
		}
	}
	stats.ActiveConnections = uint64(activeConns)
	stats.BlockedIPs = uint64(blocked)

	return &DDoSStats{
		TotalConnections:    stats.TotalConnections,
		BlockedConnections:  stats.BlockedConnections,
		RateLimitedRequests: stats.RateLimitedRequests,
		ActiveConnections:   stats.ActiveConnections,
		BlockedIPs:          stats.BlockedIPs,
	}
}

func (p *DDoSProtector) Stop() {
	close(p.stopChan)
}

type DDoSMiddleware struct {
	protector *DDoSProtector
	logger    *AuditLoggerV2
	config    *DDoSProtectionConfig
}

func NewDDoSMiddleware(protector *DDoSProtector, logger *AuditLoggerV2, config *DDoSProtectionConfig) *DDoSMiddleware {
	if config == nil {
		config = defaultDDoSConfig
	}
	return &DDoSMiddleware{
		protector: protector,
		logger:    logger,
		config:    config,
	}
}

func (m *DDoSMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.config.Enabled {
			c.Next()
			return
		}

		ip := GetClientIP(c)

		allowed, reason := m.protector.CheckConnection(ip)
		if !allowed {
			LogSecurityViolation(m.logger, "ddos", ip, map[string]interface{}{
				"reason": reason,
				"type":   "connection_limit",
			})

			c.Header("X-Blocked-By", "DDoS-Protection")
			c.Header("Retry-After", fmt.Sprintf("%d", int(m.config.BlockDuration.Seconds())))
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "access_denied",
				"message": "Your IP has been blocked due to suspicious activity",
			})
			return
		}

		m.protector.IncrementConnections(ip)
		defer m.protector.DecrementConnections(ip)

		allowed, reason, remaining := m.protector.CheckRateLimit(ip)
		if !allowed {
			LogSecurityViolation(m.logger, "ddos", ip, map[string]interface{}{
				"reason": reason,
				"type":   "rate_limit",
			})

			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", "1")
			c.Header("X-Blocked-By", "DDoS-Protection")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests, please slow down",
			})
			return
		}

		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", m.config.RequestsPerSecond))

		c.Next()
	}
}

func (m *DDoSMiddleware) GetStats() *DDoSStats {
	return m.protector.GetStats()
}

type ConnectionLimiter struct {
	config      *ConnectionLimitConfig
	connections map[string]int
	mu          sync.RWMutex
}

type ConnectionLimitConfig struct {
	MaxConnections   int
	MaxConnectionsIP int
	CleanupTimeout   time.Duration
}

var defaultConnLimitConfig = &ConnectionLimitConfig{
	MaxConnections:   10000,
	MaxConnectionsIP:  100,
	CleanupTimeout:   5 * time.Minute,
}

func NewConnectionLimiter(config *ConnectionLimitConfig) *ConnectionLimiter {
	if config == nil {
		config = defaultConnLimitConfig
	}
	return &ConnectionLimiter{
		config:      config,
		connections: make(map[string]int),
	}
}

func (l *ConnectionLimiter) AllowConnection(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.connections[ip]
	if current >= l.config.MaxConnectionsIP {
		return false
	}

	l.connections[ip]++
	return true
}

func (l *ConnectionLimiter) CloseConnection(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if count, exists := l.connections[ip]; exists && count > 0 {
		l.connections[ip] = count - 1
	}
}

func (l *ConnectionLimiter) GetConnectionCount(ip string) int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.connections[ip]
}

func (l *ConnectionLimiter) GetTotalConnections() int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	total := 0
	for _, count := range l.connections {
		total += count
	}
	return total
}

func (l *ConnectionLimiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for ip, count := range l.connections {
		if count <= 0 {
			delete(l.connections, ip)
		}
	}
}

type SynFloodProtection struct {
	mu              sync.RWMutex
	synCount        map[string]int
	config          *SynFloodConfig
	lastCleanup     time.Time
}

type SynFloodConfig struct {
	Enabled        bool
	SynThreshold   int
	SynWindow      time.Duration
	BlockDuration  time.Duration
}

var defaultSynFloodConfig = &SynFloodConfig{
	Enabled:       true,
	SynThreshold:   100,
	SynWindow:      time.Second,
	BlockDuration:  30 * time.Second,
}

func NewSynFloodProtection(config *SynFloodConfig) *SynFloodProtection {
	if config == nil {
		config = defaultSynFloodConfig
	}
	return &SynFloodProtection{
		synCount:    make(map[string]int),
		config:      config,
		lastCleanup: time.Now(),
	}
}

func (s *SynFloodProtection) RecordSyn(ip string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.synCount[ip]++

	if time.Since(s.lastCleanup) > s.config.SynWindow {
		s.cleanup()
	}

	return s.synCount[ip] > s.config.SynThreshold
}

func (s *SynFloodProtection) cleanup() {
	s.synCount = make(map[string]int)
	s.lastCleanup = time.Now()
}

func (s *SynFloodProtection) IsBlocked(ip string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.synCount[ip] > s.config.SynThreshold
}

type DDoSProtectionManager struct {
	protector  *DDoSProtector
	limiter    *ConnectionLimiter
	synFlood   *SynFloodProtection
	anomaly    *AnomalyDetector
	logger     *AuditLoggerV2
	config     *DDoSProtectionConfig
	stopChan   chan struct{}
}

func NewDDoSProtectionManager(config *DDoSProtectionConfig, logger *AuditLoggerV2) *DDoSProtectionManager {
	if config == nil {
		config = defaultDDoSConfig
	}

	manager := &DDoSProtectionManager{
		protector: NewDDoSProtector(config),
		limiter:   NewConnectionLimiter(nil),
		synFlood:  NewSynFloodProtection(nil),
		logger:    logger,
		config:    config,
		stopChan:  make(chan struct{}),
	}

	return manager
}

func (m *DDoSProtectionManager) StartAnomalyDetection(ctx context.Context, anomaly *AnomalyDetector) {
	m.anomaly = anomaly
	StartAnomalyCleanup(ctx, anomaly, time.Minute)
}

func (m *DDoSProtectionManager) Middleware() gin.HandlerFunc {
	return m.CombineMiddleware()
}

func (m *DDoSProtectionManager) CombineMiddleware() gin.HandlerFunc {
	dDoSMiddleware := NewDDoSMiddleware(m.protector, m.logger, m.config)

	return func(c *gin.Context) {
		ip := GetClientIP(c)

		if m.protector.IsIPBlocked(ip) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "blocked",
				"message": "Your IP has been blocked",
			})
			return
		}

		if !m.limiter.AllowConnection(ip) {
			LogSecurityViolation(m.logger, "ddos", ip, map[string]interface{}{
				"reason": "Too many connections",
				"type":   "connection_limit",
			})

			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error":   "service_unavailable",
				"message": "Server is experiencing high load",
			})
			return
		}
		defer m.limiter.CloseConnection(ip)

		dDoSMiddleware.Handler()(c)
	}
}

func (m *DDoSProtectionManager) GetStats() map[string]interface{} {
	stats := m.protector.GetStats()

	return map[string]interface{}{
		"total_connections":    stats.TotalConnections,
		"blocked_connections":   stats.BlockedConnections,
		"rate_limited_requests": stats.RateLimitedRequests,
		"active_connections":   stats.ActiveConnections,
		"blocked_ips":          m.protector.GetStats().BlockedIPs,
	}
}

func (m *DDoSProtectionManager) BlockIP(ip string, duration time.Duration) error {
	m.protector.BlockIP(ip, duration)
	m.limiter.connections[ip] = 0

	LogSecurityViolation(m.logger, "ddos", ip, map[string]interface{}{
		"reason":    "Manual block",
		"duration":  duration.String(),
		"blocked_by": "admin",
	})

	return nil
}

func (m *DDoSProtectionManager) UnblockIP(ip string) error {
	m.protector.UnblockIP(ip)
	m.limiter.connections[ip] = 0
	return nil
}

func (m *DDoSProtectionManager) GetBlockedIPs() []string {
	m.protector.mu.RLock()
	defer m.protector.mu.RUnlock()

	var blocked []string
	for ip, tracker := range m.protector.trackers {
		if tracker.Blocked {
			blocked = append(blocked, ip)
		}
	}
	return blocked
}

func (m *DDoSProtectionManager) Stop() {
	m.protector.Stop()
	close(m.stopChan)
}

type IPWhitelist struct {
	mu        sync.RWMutex
	whitelist map[string]bool
}

func NewIPWhitelist() *IPWhitelist {
	return &IPWhitelist{
		whitelist: make(map[string]bool),
	}
}

func (w *IPWhitelist) Add(ip string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.whitelist[ip] = true
}

func (w *IPWhitelist) Remove(ip string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.whitelist, ip)
}

func (w *IPWhitelist) IsWhitelisted(ip string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.whitelist[ip]
}

func (w *IPWhitelist) AddCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.whitelist[cidr] = true
	return nil
}

func (w *IPWhitelist) IsInWhitelist(ipStr string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.whitelist[ipStr] {
		return true
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	for cidr := range w.whitelist {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if ipnet.Contains(ip) {
			return true
		}
	}

	return false
}

type DDoSMiddlewareBuilder struct {
	config   *DDoSProtectionConfig
	logger   *AuditLoggerV2
	whitelist *IPWhitelist
}

func NewDDoSMiddlewareBuilder() *DDoSMiddlewareBuilder {
	return &DDoSMiddlewareBuilder{
		config:    defaultDDoSConfig,
		whitelist: NewIPWhitelist(),
	}
}

func (b *DDoSMiddlewareBuilder) WithConfig(config *DDoSProtectionConfig) *DDoSMiddlewareBuilder {
	b.config = config
	return b
}

func (b *DDoSMiddlewareBuilder) WithLogger(logger *AuditLoggerV2) *DDoSMiddlewareBuilder {
	b.logger = logger
	return b
}

func (b *DDoSMiddlewareBuilder) WithWhitelist(ip string) *DDoSMiddlewareBuilder {
	b.whitelist.Add(ip)
	return b
}

func (b *DDoSMiddlewareBuilder) WithWhitelistCIDR(cidr string) *DDoSMiddlewareBuilder {
	b.whitelist.AddCIDR(cidr)
	return b
}

func (b *DDoSMiddlewareBuilder) Build() gin.HandlerFunc {
	manager := NewDDoSProtectionManager(b.config, b.logger)

	return func(c *gin.Context) {
		ip := GetClientIP(c)

		if b.whitelist.IsInWhitelist(ip) {
			c.Next()
			return
		}

		manager.Middleware()(c)
	}
}
