package security

import (
	"fmt"
	"sync"
	"time"
)

// AttemptRecord 尝试记录
type AttemptRecord struct {
	Count       int
	FirstAt     time.Time
	LastAt      time.Time
	LockedUntil time.Time
}

// LoginProtector 登录保护器
type LoginProtector struct {
	maxAttempts int
	lockoutTime time.Duration
	attempts    map[string]*AttemptRecord
	mu          sync.RWMutex
}

// NewLoginProtector 创建登录保护器
func NewLoginProtector(maxAttempts int, lockoutTime time.Duration) *LoginProtector {
	return &LoginProtector{
		maxAttempts: maxAttempts,
		lockoutTime: lockoutTime,
		attempts:    make(map[string]*AttemptRecord),
	}
}

// RecordAttempt 记录尝试
func (p *LoginProtector) RecordAttempt(identifier string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	record, exists := p.attempts[identifier]
	if !exists {
		record = &AttemptRecord{
			Count:   0,
			FirstAt: time.Now(),
		}
		p.attempts[identifier] = record
	}

	record.Count++
	record.LastAt = time.Now()

	if record.Count >= p.maxAttempts {
		record.LockedUntil = time.Now().Add(p.lockoutTime)
	}
}

// IsLocked 检查是否被锁定
func (p *LoginProtector) IsLocked(identifier string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	record, exists := p.attempts[identifier]
	if !exists {
		return false
	}

	if record.LockedUntil.IsZero() {
		return false
	}

	if time.Now().Before(record.LockedUntil) {
		return true
	}

	return false
}

// GetRemainingLockTime 获取剩余锁定时间
func (p *LoginProtector) GetRemainingLockTime(identifier string) time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()

	record, exists := p.attempts[identifier]
	if !exists {
		return 0
	}

	if record.LockedUntil.IsZero() {
		return 0
	}

	remaining := time.Until(record.LockedUntil)
	if remaining < 0 {
		return 0
	}

	return remaining
}

// GetRemainingAttempts 获取剩余尝试次数
func (p *LoginProtector) GetRemainingAttempts(identifier string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	record, exists := p.attempts[identifier]
	if !exists {
		return p.maxAttempts
	}

	remaining := p.maxAttempts - record.Count
	if remaining < 0 {
		return 0
	}

	return remaining
}

// ResetAttempts 重置尝试次数
func (p *LoginProtector) ResetAttempts(identifier string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.attempts, identifier)
}

// ClearExpiredLocks 清除过期的锁定
func (p *LoginProtector) ClearExpiredLocks() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for identifier, record := range p.attempts {
		if !record.LockedUntil.IsZero() && now.After(record.LockedUntil) {
			record.LockedUntil = time.Time{}
			record.Count = 0
		}

		if record.Count > 0 && now.Sub(record.LastAt) > 24*time.Hour {
			delete(p.attempts, identifier)
		}
	}
}

// GetLockStatus 获取锁定状态
func (p *LoginProtector) GetLockStatus(identifier string) map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	record, exists := p.attempts[identifier]
	if !exists {
		return map[string]interface{}{
			"locked":          false,
			"attempts":        0,
			"remaining":       p.maxAttempts,
			"remaining_lock":  0,
		}
	}

	return map[string]interface{}{
		"locked":         !record.LockedUntil.IsZero() && time.Now().Before(record.LockedUntil),
		"attempts":       record.Count,
		"remaining":      p.maxAttempts - record.Count,
		"remaining_lock":  p.GetRemainingLockTime(identifier).Seconds(),
		"locked_until":    record.LockedUntil,
	}
}

// BruteForceConfig 暴力破解防护配置
type BruteForceConfig struct {
	MaxAttempts     int
	LockoutTime    time.Duration
	ResetAfter     time.Duration
	MaxIdentifiers int
}

// DefaultBruteForceConfig 默认配置
var DefaultBruteForceConfig = &BruteForceConfig{
	MaxAttempts:     5,
	LockoutTime:    15 * time.Minute,
	ResetAfter:     30 * time.Minute,
	MaxIdentifiers: 10000,
}

// BruteForceProtector 暴力破解防护器
type BruteForceProtector struct {
	config  *BruteForceConfig
	records map[string]*AttemptRecord
	mu      sync.RWMutex
}

// NewBruteForceProtector 创建暴力破解防护器
func NewBruteForceProtector(config *BruteForceConfig) *BruteForceProtector {
	if config == nil {
		config = DefaultBruteForceConfig
	}

	protector := &BruteForceProtector{
		config:  config,
		records: make(map[string]*AttemptRecord),
	}

	go protector.cleanupRoutine()

	return protector
}

// RecordFailure 记录失败
func (p *BruteForceProtector) RecordFailure(identifier string) (bool, int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	record, exists := p.records[identifier]
	if !exists {
		record = &AttemptRecord{
			Count:   0,
			FirstAt: time.Now(),
		}
		p.records[identifier] = record
	}

	record.Count++
	record.LastAt = time.Now()

	if record.Count >= p.config.MaxAttempts {
		record.LockedUntil = time.Now().Add(p.config.LockoutTime)
		return true, 0
	}

	remaining := p.config.MaxAttempts - record.Count
	return false, remaining
}

// RecordSuccess 记录成功
func (p *BruteForceProtector) RecordSuccess(identifier string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.records, identifier)
}

// IsLocked 检查是否被锁定
func (p *BruteForceProtector) IsLocked(identifier string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	record, exists := p.records[identifier]
	if !exists {
		return false
	}

	if record.LockedUntil.IsZero() {
		return false
	}

	return time.Now().Before(record.LockedUntil)
}

// GetStatus 获取状态
func (p *BruteForceProtector) GetStatus(identifier string) *AttemptRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()

	record, exists := p.records[identifier]
	if !exists {
		return nil
	}

	return &AttemptRecord{
		Count:       record.Count,
		FirstAt:     record.FirstAt,
		LastAt:      record.LastAt,
		LockedUntil: record.LockedUntil,
	}
}

// Unlock 解除锁定
func (p *BruteForceProtector) Unlock(identifier string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.records, identifier)
}

// UnlockAll 解除所有锁定
func (p *BruteForceProtector) UnlockAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.records = make(map[string]*AttemptRecord)
}

// GetLockedCount 获取被锁定数量
func (p *BruteForceProtector) GetLockedCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	now := time.Now()
	for _, record := range p.records {
		if !record.LockedUntil.IsZero() && now.Before(record.LockedUntil) {
			count++
		}
	}

	return count
}

// GetTotalAttempts 获取总尝试次数
func (p *BruteForceProtector) GetTotalAttempts() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	total := 0
	for _, record := range p.records {
		total += record.Count
	}

	return total
}

// cleanupRoutine 清理例程
func (p *BruteForceProtector) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()

		now := time.Now()
		resetThreshold := now.Add(-p.config.ResetAfter)

		for identifier, record := range p.records {
			if !record.LockedUntil.IsZero() && now.After(record.LockedUntil) {
				delete(p.records, identifier)
				continue
			}

			if record.LastAt.Before(resetThreshold) {
				delete(p.records, identifier)
			}
		}

		if len(p.records) > p.config.MaxIdentifiers {
			oldest := time.Now()
			var oldestKey string

			for identifier, record := range p.records {
				if record.FirstAt.Before(oldest) {
					oldest = record.FirstAt
					oldestKey = identifier
				}
			}

			if oldestKey != "" {
				delete(p.records, oldestKey)
			}
		}

		p.mu.Unlock()
	}
}

// LoginAttempt 登录尝试
type LoginAttempt struct {
	Identifier string
	IP         string
	Success    bool
	Timestamp  time.Time
}

// LoginAttemptLogger 登录尝试记录器
type LoginAttemptLogger struct {
	attempts []LoginAttempt
	maxSize  int
	mu       sync.RWMutex
}

// NewLoginAttemptLogger 创建登录尝试记录器
func NewLoginAttemptLogger(maxSize int) *LoginAttemptLogger {
	if maxSize <= 0 {
		maxSize = 1000
	}

	return &LoginAttemptLogger{
		attempts: make([]LoginAttempt, 0, maxSize),
		maxSize:  maxSize,
	}
}

// Log 记录尝试
func (l *LoginAttemptLogger) Log(attempt LoginAttempt) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.attempts = append(l.attempts, attempt)

	if len(l.attempts) > l.maxSize {
		l.attempts = l.attempts[1:]
	}
}

// GetRecentAttempts 获取最近的尝试
func (l *LoginAttemptLogger) GetRecentAttempts(identifier string, duration time.Duration) []LoginAttempt {
	l.mu.RLock()
	defer l.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	result := make([]LoginAttempt, 0)

	for i := len(l.attempts) - 1; i >= 0; i-- {
		attempt := l.attempts[i]
		if attempt.Identifier == identifier && attempt.Timestamp.After(cutoff) {
			result = append(result, attempt)
		}
	}

	return result
}

// GetFailedAttempts 获取失败的尝试
func (l *LoginAttemptLogger) GetFailedAttempts(identifier string, duration time.Duration) []LoginAttempt {
	l.mu.RLock()
	defer l.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	result := make([]LoginAttempt, 0)

	for i := len(l.attempts) - 1; i >= 0; i-- {
		attempt := l.attempts[i]
		if attempt.Identifier == identifier && !attempt.Success && attempt.Timestamp.After(cutoff) {
			result = append(result, attempt)
		}
	}

	return result
}

// GetIPAttempts 获取IP的尝试
func (l *LoginAttemptLogger) GetIPAttempts(ip string, duration time.Duration) []LoginAttempt {
	l.mu.RLock()
	defer l.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	result := make([]LoginAttempt, 0)

	for i := len(l.attempts) - 1; i >= 0; i-- {
		attempt := l.attempts[i]
		if attempt.IP == ip && attempt.Timestamp.After(cutoff) {
			result = append(result, attempt)
		}
	}

	return result
}

// GetSuspiciousIPs 获取可疑IP
func (l *LoginAttemptLogger) GetSuspiciousIPs(threshold int, duration time.Duration) []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	ipCounts := make(map[string]int)

	for _, attempt := range l.attempts {
		if attempt.Timestamp.After(cutoff) && !attempt.Success {
			ipCounts[attempt.IP]++
		}
	}

	result := make([]string, 0)
	for ip, count := range ipCounts {
		if count >= threshold {
			result = append(result, fmt.Sprintf("%s (%d attempts)", ip, count))
		}
	}

	return result
}

// GlobalLoginProtector 全局登录保护器
var GlobalLoginProtector *BruteForceProtector

// InitGlobalLoginProtector 初始化全局登录保护器
func InitGlobalLoginProtector(config *BruteForceConfig) {
	GlobalLoginProtector = NewBruteForceProtector(config)
}
