package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type CCConsumptionType string

const (
	CCConsumeCreate    CCConsumptionType = "create"
	CCConsumeVerify    CCConsumptionType = "verify"
	CCConsumeFailed    CCConsumptionType = "failed"
)

type CCConsumptionRecord struct {
	Type        CCConsumptionType
	CaptchaID   string
	IPAddress   string
	UserID      string
	AppID       string
	Timestamp   time.Time
	Success     bool
	Duration    time.Duration
	FailReason  string
}

type CCProtectionConfig struct {
	Enabled                bool
	MaxConsumptionsPerIP   int
	MaxConsumptionsPerUser int
	MaxConsumptionsPerApp int
	ConsumptionWindow      time.Duration
	BlockDuration          time.Duration
	AlertThreshold         int
	EnablePerAppLimits     bool
	EnablePerUserLimits    bool
	EnablePerIPLimits      bool
	CleanupInterval        time.Duration
	MaxRecordAge           time.Duration
}

var defaultCCConfig = &CCProtectionConfig{
	Enabled:                true,
	MaxConsumptionsPerIP:   100,
	MaxConsumptionsPerUser: 50,
	MaxConsumptionsPerApp: 200,
	ConsumptionWindow:      1 * time.Minute,
	BlockDuration:          10 * time.Minute,
	AlertThreshold:         80,
	EnablePerAppLimits:     true,
	EnablePerUserLimits:    true,
	EnablePerIPLimits:      true,
	CleanupInterval:        1 * time.Minute,
	MaxRecordAge:           10 * time.Minute,
}

type CCProtector struct {
	config    *CCProtectionConfig
	records   []*CCConsumptionRecord
	ipIndex   map[string][]int
	userIndex map[string][]int
	appIndex  map[string][]int
	mu        sync.RWMutex
	blocked   map[string]time.Time
	stopChan  chan struct{}
}

type CCProtectionStats struct {
	TotalConsumptions uint64
	BlockedRequests   uint64
	Alerts            uint64
	ActiveBlocks      int
	ByType            map[CCConsumptionType]uint64
}

func NewCCProtector(config *CCProtectionConfig) *CCProtector {
	if config == nil {
		config = defaultCCConfig
	}

	if config.CleanupInterval <= 0 {
		config.CleanupInterval = defaultCCConfig.CleanupInterval
	}

	protector := &CCProtector{
		config:    config,
		records:   make([]*CCConsumptionRecord, 0),
		ipIndex:   make(map[string][]int),
		userIndex: make(map[string][]int),
		appIndex:  make(map[string][]int),
		blocked:   make(map[string]time.Time),
		stopChan:  make(chan struct{}),
	}

	if config.Enabled {
		go protector.cleanup()
	}

	return protector
}

func (p *CCProtector) cleanup() {
	ticker := time.NewTicker(p.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.cleanupOldRecords()
			p.cleanupExpiredBlocks()
		}
	}
}

func (p *CCProtector) cleanupOldRecords() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-p.config.MaxRecordAge)

	var validRecords []*CCConsumptionRecord
	newIPIndex := make(map[string][]int)
	newUserIndex := make(map[string][]int)
	newAppIndex := make(map[string][]int)

	for i, record := range p.records {
		if record.Timestamp.After(cutoff) {
			validRecords = append(validRecords, record)

			if record.IPAddress != "" {
				newIPIndex[record.IPAddress] = append(newIPIndex[record.IPAddress], i)
			}
			if record.UserID != "" {
				newUserIndex[record.UserID] = append(newUserIndex[record.UserID], i)
			}
			if record.AppID != "" {
				newAppIndex[record.AppID] = append(newAppIndex[record.AppID], i)
			}
		}
	}

	p.records = validRecords
	p.ipIndex = newIPIndex
	p.userIndex = newUserIndex
	p.appIndex = newAppIndex
}

func (p *CCProtector) cleanupExpiredBlocks() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for key, until := range p.blocked {
		if now.After(until) {
			delete(p.blocked, key)
		}
	}
}

func (p *CCProtector) RecordConsumption(record *CCConsumptionRecord) error {
	if !p.config.Enabled {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	record.Timestamp = time.Now()
	index := len(p.records)
	p.records = append(p.records, record)

	if record.IPAddress != "" {
		p.ipIndex[record.IPAddress] = append(p.ipIndex[record.IPAddress], index)
	}
	if record.UserID != "" {
		p.userIndex[record.UserID] = append(p.userIndex[record.UserID], index)
	}
	if record.AppID != "" {
		p.appIndex[record.AppID] = append(p.appIndex[record.AppID], index)
	}

	return nil
}

func (p *CCProtector) CheckLimit(ip, userID, appID string) (bool, string, int) {
	if !p.config.Enabled {
		return true, "", 0
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	now := time.Now()
	windowStart := now.Add(-p.config.ConsumptionWindow)

	var reasons []string

	if p.config.EnablePerIPLimits && ip != "" {
		count := p.countInIndex(p.ipIndex[ip], windowStart)
		remaining := p.config.MaxConsumptionsPerIP - count
		if count >= p.config.MaxConsumptionsPerIP {
			return false, fmt.Sprintf("IP %s has exceeded consumption limit (%d/%d)", ip, count, p.config.MaxConsumptionsPerIP), remaining
		}
		if remaining <= p.config.AlertThreshold {
			reasons = append(reasons, fmt.Sprintf("IP limit warning: %d remaining", remaining))
		}
	}

	if p.config.EnablePerUserLimits && userID != "" {
		count := p.countInIndex(p.userIndex[userID], windowStart)
		remaining := p.config.MaxConsumptionsPerUser - count
		if count >= p.config.MaxConsumptionsPerUser {
			return false, fmt.Sprintf("User %s has exceeded consumption limit (%d/%d)", userID, count, p.config.MaxConsumptionsPerUser), remaining
		}
		if remaining <= p.config.AlertThreshold {
			reasons = append(reasons, fmt.Sprintf("User limit warning: %d remaining", remaining))
		}
	}

	if p.config.EnablePerAppLimits && appID != "" {
		count := p.countInIndex(p.appIndex[appID], windowStart)
		remaining := p.config.MaxConsumptionsPerApp - count
		if count >= p.config.MaxConsumptionsPerApp {
			return false, fmt.Sprintf("App %s has exceeded consumption limit (%d/%d)", appID, count, p.config.MaxConsumptionsPerApp), remaining
		}
		if remaining <= p.config.AlertThreshold {
			reasons = append(reasons, fmt.Sprintf("App limit warning: %d remaining", remaining))
		}
	}

	_ = p.getTotalCount(ip, userID, appID, windowStart)
	remaining := p.getMinRemaining(ip, userID, appID)

	if len(reasons) > 0 && remaining <= p.config.AlertThreshold {
		return true, fmt.Sprintf("Warning: %v", reasons), remaining
	}

	return true, "", remaining
}

func (p *CCProtector) countInIndex(indices []int, windowStart time.Time) int {
	count := 0
	for _, idx := range indices {
		if idx >= 0 && idx < len(p.records) {
			if p.records[idx].Timestamp.After(windowStart) {
				count++
			}
		}
	}
	return count
}

func (p *CCProtector) getTotalCount(ip, userID, appID string, windowStart time.Time) int {
	count := 0

	if ip != "" {
		count += p.countInIndex(p.ipIndex[ip], windowStart)
	}
	if userID != "" {
		count += p.countInIndex(p.userIndex[userID], windowStart)
	}
	if appID != "" {
		count += p.countInIndex(p.appIndex[appID], windowStart)
	}

	return count
}

func (p *CCProtector) getMinRemaining(ip, userID, appID string) int {
	remaining := p.config.MaxConsumptionsPerIP

	if ip != "" {
		p.mu.RLock()
		count := p.countInIndex(p.ipIndex[ip], time.Now().Add(-p.config.ConsumptionWindow))
		p.mu.RUnlock()
		if p.config.MaxConsumptionsPerIP-count < remaining {
			remaining = p.config.MaxConsumptionsPerIP - count
		}
	}

	if userID != "" {
		p.mu.RLock()
		count := p.countInIndex(p.userIndex[userID], time.Now().Add(-p.config.ConsumptionWindow))
		p.mu.RUnlock()
		if p.config.MaxConsumptionsPerUser-count < remaining {
			remaining = p.config.MaxConsumptionsPerUser - count
		}
	}

	if appID != "" {
		p.mu.RLock()
		count := p.countInIndex(p.appIndex[appID], time.Now().Add(-p.config.ConsumptionWindow))
		p.mu.RUnlock()
		if p.config.MaxConsumptionsPerApp-count < remaining {
			remaining = p.config.MaxConsumptionsPerApp - count
		}
	}

	return remaining
}

func (p *CCProtector) Block(ip, userID, appID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := p.makeBlockKey(ip, userID, appID)
	p.blocked[key] = time.Now().Add(p.config.BlockDuration)

	return nil
}

func (p *CCProtector) Unblock(ip, userID, appID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := p.makeBlockKey(ip, userID, appID)
	delete(p.blocked, key)

	return nil
}

func (p *CCProtector) makeBlockKey(ip, userID, appID string) string {
	h := sha256.New()
	if ip != "" {
		h.Write([]byte("ip:" + ip))
	}
	if userID != "" {
		h.Write([]byte("user:" + userID))
	}
	if appID != "" {
		h.Write([]byte("app:" + appID))
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func (p *CCProtector) IsBlocked(ip, userID, appID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	now := time.Now()

	keys := []string{}
	if ip != "" {
		keys = append(keys, p.makeBlockKey(ip, "", ""))
	}
	if userID != "" {
		keys = append(keys, p.makeBlockKey("", userID, ""))
	}
	if appID != "" {
		keys = append(keys, p.makeBlockKey("", "", appID))
	}

	for _, key := range keys {
		if until, blocked := p.blocked[key]; blocked && now.Before(until) {
			return true
		}
	}

	return false
}

func (p *CCProtector) GetStats() *CCProtectionStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := &CCProtectionStats{
		ByType: make(map[CCConsumptionType]uint64),
	}

	now := time.Now()
	windowStart := now.Add(-p.config.ConsumptionWindow)

	for _, record := range p.records {
		if record.Timestamp.After(windowStart) {
			stats.TotalConsumptions++
			stats.ByType[record.Type]++
		}
	}

	stats.ActiveBlocks = len(p.blocked)

	return stats
}

func (p *CCProtector) Stop() {
	close(p.stopChan)
}

type CCMiddleware struct {
	protector *CCProtector
	logger    *AuditLoggerV2
	config    *CCProtectionConfig
}

func NewCCMiddleware(protector *CCProtector, logger *AuditLoggerV2, config *CCProtectionConfig) *CCMiddleware {
	if config == nil {
		config = defaultCCConfig
	}
	return &CCMiddleware{
		protector: protector,
		logger:    logger,
		config:    config,
	}
}

func (m *CCMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.config.Enabled {
			c.Next()
			return
		}

		ip := GetClientIP(c)
		userID := ""
		if v, exists := c.Get("user_id"); exists {
			userID = fmt.Sprintf("%v", v)
		}
		appID := c.Query("app_id")
		if appID == "" {
			appID = c.GetHeader("X-App-ID")
		}

		if m.protector.IsBlocked(ip, userID, appID) {
			LogSecurityViolation(m.logger, "cc_attack", ip, map[string]interface{}{
				"user_id": userID,
				"app_id":  appID,
				"reason":  "CC protection block",
			})

			c.AbortWithStatusJSON(429, gin.H{
				"error":   "cc_protection",
				"message": "Too many captcha requests, please try again later",
			})
			return
		}

		allowed, reason, remaining := m.protector.CheckLimit(ip, userID, appID)
		if !allowed {
			m.protector.Block(ip, userID, appID)

			LogSecurityViolation(m.logger, "cc_attack", ip, map[string]interface{}{
				"user_id": userID,
				"app_id":  appID,
				"reason":  reason,
			})

			c.AbortWithStatusJSON(429, gin.H{
				"error":   "cc_protection",
				"message": "Too many captcha requests, please try again later",
			})
			return
		}

		c.Header("X-CC-Limit-Remaining", fmt.Sprintf("%d", remaining))

		c.Next()

		record := &CCConsumptionRecord{
			IPAddress: ip,
			UserID:    userID,
			AppID:     appID,
			Success:   c.Writer.Status() < 400,
		}

		if strings.HasSuffix(c.Request.URL.Path, "/verify") {
			record.Type = CCConsumeVerify
			if !record.Success {
				record.Type = CCConsumeFailed
				record.FailReason = fmt.Sprintf("status: %d", c.Writer.Status())
			}
		} else if c.Request.Method == http.MethodPost {
			record.Type = CCConsumeCreate
		}

		m.protector.RecordConsumption(record)

		if reason != "" {
			c.Header("X-CC-Warning", reason)
		}
	}
}

func (m *CCMiddleware) GetStats() *CCProtectionStats {
	return m.protector.GetStats()
}

type CaptchaConsumptionLimiter struct {
	protector *CCProtector
	logger    *AuditLoggerV2
	config    *CCProtectionConfig
}

func NewCaptchaConsumptionLimiter(logger *AuditLoggerV2, config *CCProtectionConfig) *CaptchaConsumptionLimiter {
	if config == nil {
		config = defaultCCConfig
	}
	return &CaptchaConsumptionLimiter{
		protector: NewCCProtector(config),
		logger:    logger,
		config:    config,
	}
}

func (l *CaptchaConsumptionLimiter) CheckAndRecord(ctx context.Context, ip, userID, appID string, consumeType CCConsumptionType) (bool, error) {
	if l.protector.IsBlocked(ip, userID, appID) {
		return false, fmt.Errorf("blocked by CC protection")
	}

	allowed, reason, _ := l.protector.CheckLimit(ip, userID, appID)
	if !allowed {
		l.protector.Block(ip, userID, appID)

		LogSecurityViolation(l.logger, "cc_attack", ip, map[string]interface{}{
			"user_id": userID,
			"app_id":  appID,
			"type":    consumeType,
			"reason":  reason,
		})

		return false, fmt.Errorf("consumption limit exceeded")
	}

	record := &CCConsumptionRecord{
		Type:      consumeType,
		IPAddress: ip,
		UserID:    userID,
		AppID:     appID,
		Success:   true,
	}

	if err := l.protector.RecordConsumption(record); err != nil {
		return false, err
	}

	return true, nil
}

func (l *CaptchaConsumptionLimiter) RecordSuccess(ctx context.Context, ip, userID, appID, captchaID string, duration time.Duration) error {
	record := &CCConsumptionRecord{
		Type:       CCConsumeVerify,
		CaptchaID:  captchaID,
		IPAddress:  ip,
		UserID:     userID,
		AppID:      appID,
		Success:    true,
		Duration:   duration,
	}
	return l.protector.RecordConsumption(record)
}

func (l *CaptchaConsumptionLimiter) RecordFailure(ctx context.Context, ip, userID, appID, captchaID, reason string) error {
	record := &CCConsumptionRecord{
		Type:       CCConsumeFailed,
		CaptchaID:  captchaID,
		IPAddress:  ip,
		UserID:     userID,
		AppID:      appID,
		Success:    false,
		FailReason: reason,
	}
	return l.protector.RecordConsumption(record)
}

func (l *CaptchaConsumptionLimiter) GetStats() *CCProtectionStats {
	return l.protector.GetStats()
}

func (l *CaptchaConsumptionLimiter) GetConsumptionHistory(ctx context.Context, ip, userID, appID string, window time.Duration) []*CCConsumptionRecord {
	return nil
}

func (l *CaptchaConsumptionLimiter) IsBlocked(ip, userID, appID string) bool {
	return l.protector.IsBlocked(ip, userID, appID)
}

func (l *CaptchaConsumptionLimiter) Block(ip, userID, appID string) error {
	return l.protector.Block(ip, userID, appID)
}

func (l *CaptchaConsumptionLimiter) Unblock(ip, userID, appID string) error {
	return l.protector.Unblock(ip, userID, appID)
}

type CCProtectionResponseConfig struct {
	ErrorCode     string
	Message       string
	RetryAfter    int
	LimitType     string
	CurrentUsage  int
	Limit         int
	ResetAt       time.Time
}

func (m *CCMiddleware) GenerateErrorResponse(ip, userID, appID string) *CCProtectionResponseConfig {
	remaining := m.protector.getMinRemaining(ip, userID, appID)

	var limitType string
	var limit int

	if userID != "" {
		limitType = "user"
		limit = m.config.MaxConsumptionsPerUser
	} else if appID != "" {
		limitType = "app"
		limit = m.config.MaxConsumptionsPerApp
	} else {
		limitType = "ip"
		limit = m.config.MaxConsumptionsPerIP
	}

	return &CCProtectionResponseConfig{
		ErrorCode:    "CC_PROTECTION",
		Message:      "Too many captcha requests",
		RetryAfter:   int(m.config.BlockDuration.Seconds()),
		LimitType:    limitType,
		CurrentUsage: limit - remaining,
		Limit:        limit,
		ResetAt:      time.Now().Add(m.config.ConsumptionWindow),
	}
}

type CCProtectionAlert struct {
	ID          string
	Timestamp   time.Time
	Type        string
	IPAddress   string
	UserID      string
	AppID       string
	Consumption int
	Threshold   int
	Severity    string
}

type CCAlertManager struct {
	protector  *CCProtector
	alerts     []*CCProtectionAlert
	mu         sync.RWMutex
	maxAlerts  int
	threshold  int
	alertLevel string
}

func NewCCAlertManager(threshold int) *CCAlertManager {
	return &CCAlertManager{
		protector:  nil,
		alerts:     make([]*CCProtectionAlert, 0),
		maxAlerts:  100,
		threshold:  threshold,
		alertLevel: "warning",
	}
}

func (m *CCAlertManager) SetProtector(protector *CCProtector) {
	m.protector = protector
}

func (m *CCAlertManager) CheckAndAlert(ip, userID, appID string) {
	if m.protector == nil {
		return
	}

	remaining := m.protector.getMinRemaining(ip, userID, appID)

	var limit int
	var limitType string

	if userID != "" {
		limit = m.protector.config.MaxConsumptionsPerUser
		limitType = "user"
	} else if appID != "" {
		limit = m.protector.config.MaxConsumptionsPerApp
		limitType = "app"
	} else {
		limit = m.protector.config.MaxConsumptionsPerIP
		limitType = "ip"
	}

	consumption := limit - remaining

	if consumption >= m.threshold || remaining <= m.threshold/10 {
		m.addAlert(ip, userID, appID, consumption, limit, limitType)
	}
}

func (m *CCAlertManager) addAlert(ip, userID, appID string, consumption, threshold int, limitType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	alert := &CCProtectionAlert{
		ID:          generateEventID(),
		Timestamp:   time.Now(),
		Type:        "consumption_threshold",
		IPAddress:   ip,
		UserID:      userID,
		AppID:       appID,
		Consumption: consumption,
		Threshold:   threshold,
		Severity:    m.alertLevel,
	}

	m.alerts = append(m.alerts, alert)
	if len(m.alerts) > m.maxAlerts {
		m.alerts = m.alerts[len(m.alerts)-m.maxAlerts:]
	}
}

func (m *CCAlertManager) GetAlerts() []*CCProtectionAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	alerts := make([]*CCProtectionAlert, len(m.alerts))
	copy(alerts, m.alerts)
	return alerts
}

func (m *CCAlertManager) GetRecentAlerts(duration time.Duration) []*CCProtectionAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	var recent []*CCProtectionAlert

	for _, alert := range m.alerts {
		if alert.Timestamp.After(cutoff) {
			recent = append(recent, alert)
		}
	}

	return recent
}

func StartCCProtectionCleanup(ctx context.Context, protector *CCProtector, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				protector.cleanupOldRecords()
				protector.cleanupExpiredBlocks()
			}
		}
	}()
}
