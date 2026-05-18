package redis

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type CacheKeyPrefix string

const (
	PrefixCaptcha       CacheKeyPrefix = "captcha"
	PrefixSession       CacheKeyPrefix = "session"
	PrefixBlacklist     CacheKeyPrefix = "blacklist"
	PrefixApplication   CacheKeyPrefix = "application"
	PrefixStats         CacheKeyPrefix = "stats"
	PrefixRateLimit     CacheKeyPrefix = "ratelimit"
	PrefixBehavior      CacheKeyPrefix = "behavior"
	PrefixConfig        CacheKeyPrefix = "config"
	PrefixUser          CacheKeyPrefix = "user"
	PrefixToken         CacheKeyPrefix = "token"
	PrefixLock          CacheKeyPrefix = "lock"
	PrefixWarmup        CacheKeyPrefix = "warmup"
	PrefixMetrics       CacheKeyPrefix = "metrics"
	PrefixVersion       CacheKeyPrefix = "version"
	PrefixTag           CacheKeyPrefix = "tag"
	PrefixMeta          CacheKeyPrefix = "meta"
	PrefixAnalytics     CacheKeyPrefix = "analytics"
	PrefixWhitelist     CacheKeyPrefix = "whitelist"
	PrefixAlert         CacheKeyPrefix = "alert"
)

type CacheKeyNamespace string

const (
	NamespaceGlobal   CacheKeyNamespace = "global"
	NamespaceApp      CacheKeyNamespace = "app"
	NamespaceUser     CacheKeyNamespace = "user"
	NamespaceSession  CacheKeyNamespace = "session"
	NamespaceAPI      CacheKeyNamespace = "api"
)

type CacheKeyConfig struct {
	Prefix    CacheKeyPrefix
	Namespace CacheKeyNamespace
	Version   int
	Timestamp time.Time
}

func NewCacheKeyConfig(prefix CacheKeyPrefix) *CacheKeyConfig {
	return &CacheKeyConfig{
		Prefix:    prefix,
		Namespace: NamespaceGlobal,
		Version:   1,
		Timestamp: time.Now(),
	}
}

func (ckc *CacheKeyConfig) WithNamespace(ns CacheKeyNamespace) *CacheKeyConfig {
	ckc.Namespace = ns
	return ckc
}

func (ckc *CacheKeyConfig) WithVersion(version int) *CacheKeyConfig {
	ckc.Version = version
	return ckc
}

type CacheKeyBuilder struct {
	prefix    CacheKeyPrefix
	namespace CacheKeyNamespace
	segments  []string
	version   int
	ttl       time.Duration
	tags      []string
}

func NewCacheKeyBuilder(prefix CacheKeyPrefix) *CacheKeyBuilder {
	return &CacheKeyBuilder{
		prefix:   prefix,
		segments: make([]string, 0),
		version:  1,
	}
}

func (ckb *CacheKeyBuilder) Namespace(ns CacheKeyNamespace) *CacheKeyBuilder {
	ckb.namespace = ns
	return ckb
}

func (ckb *CacheKeyBuilder) AddSegment(segment string) *CacheKeyBuilder {
	ckb.segments = append(ckb.segments, segment)
	return ckb
}

func (ckb *CacheKeyBuilder) AddSegments(segments ...string) *CacheKeyBuilder {
	ckb.segments = append(ckb.segments, segments...)
	return ckb
}

func (ckb *CacheKeyBuilder) Version(v int) *CacheKeyBuilder {
	ckb.version = v
	return ckb
}

func (ckb *CacheKeyBuilder) TTL(ttl time.Duration) *CacheKeyBuilder {
	ckb.ttl = ttl
	return ckb
}

func (ckb *CacheKeyBuilder) Tags(tags ...string) *CacheKeyBuilder {
	ckb.tags = append(ckb.tags, tags...)
	return ckb
}

func (ckb *CacheKeyBuilder) Build() string {
	result := string(ckb.prefix)

	if ckb.namespace != NamespaceGlobal {
		result += ":" + string(ckb.namespace)
	}

	for _, segment := range ckb.segments {
		result += ":" + segment
	}

	return result
}

func (ckb *CacheKeyBuilder) BuildWithVersion() string {
	result := string(ckb.prefix)

	if ckb.namespace != NamespaceGlobal {
		result += ":" + string(ckb.namespace)
	}

	for _, segment := range ckb.segments {
		result += ":" + segment
	}

	if ckb.version > 1 {
		result += ":" + fmt.Sprintf("v%d", ckb.version)
	}

	return result
}

func (ckb *CacheKeyBuilder) BuildWithTimestamp() string {
	key := ckb.Build()
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s:%d", key, timestamp)
}

func (ckb *CacheKeyBuilder) BuildPattern() string {
	pattern := ckb.Build()
	return pattern + ":*"
}

func (ckb *CacheKeyBuilder) BuildWithKey(key string) string {
	return fmt.Sprintf("%s:%s", ckb.Build(), key)
}

type CacheKeyManager struct {
	namespace CacheKeyNamespace
	version   int
	separator string
}

func NewCacheKeyManager(namespace CacheKeyNamespace) *CacheKeyManager {
	return &CacheKeyManager{
		namespace: namespace,
		version:   1,
		separator: ":",
	}
}

func (ckm *CacheKeyManager) SetVersion(version int) {
	ckm.version = version
}

func (ckm *CacheKeyManager) SetNamespace(ns CacheKeyNamespace) {
	ckm.namespace = ns
}

func (ckm *CacheKeyManager) BuildKey(prefix CacheKeyPrefix, segments ...string) string {
	parts := []string{string(prefix)}

	if ckm.namespace != NamespaceGlobal {
		parts = append(parts, string(ckm.namespace))
	}

	if len(segments) > 0 {
		parts = append(parts, segments...)
	}

	if ckm.version > 1 {
		parts = append(parts, fmt.Sprintf("v%d", ckm.version))
	}

	return strings.Join(parts, ckm.separator)
}

func (ckm *CacheKeyManager) BuildCaptchaKey(captchaID string) string {
	return ckm.BuildKey(PrefixCaptcha, captchaID)
}

func (ckm *CacheKeyManager) BuildSessionKey(token string) string {
	return ckm.BuildKey(PrefixSession, token)
}

func (ckm *CacheKeyManager) BuildUserKey(userID string) string {
	return ckm.BuildKey(PrefixUser, userID)
}

func (ckm *CacheKeyManager) BuildApplicationKey(apiKey string) string {
	return ckm.BuildKey(PrefixApplication, apiKey)
}

func (ckm *CacheKeyManager) BuildBlacklistKey(targetType, target string) string {
	return ckm.BuildKey(PrefixBlacklist, targetType, target)
}

func (ckm *CacheKeyManager) BuildStatsKey(metric string) string {
	return ckm.BuildKey(PrefixStats, metric)
}

func (ckm *CacheKeyManager) BuildRateLimitKey(identifier string, window int) string {
	return ckm.BuildKey(PrefixRateLimit, identifier, fmt.Sprintf("w%d", window))
}

func (ckm *CacheKeyManager) BuildBehaviorKey(sessionID string) string {
	return ckm.BuildKey(PrefixBehavior, sessionID)
}

func (ckm *CacheKeyManager) BuildConfigKey(configType, configID string) string {
	return ckm.BuildKey(PrefixConfig, configType, configID)
}

func (ckm *CacheKeyManager) BuildLockKey(resource string) string {
	return ckm.BuildKey(PrefixLock, resource)
}

func (ckm *CacheKeyManager) BuildTokenKey(token string) string {
	return ckm.BuildKey(PrefixToken, token)
}

func (ckm *CacheKeyManager) BuildVersionKey(key string) string {
	return fmt.Sprintf("%s:%s:%s", PrefixVersion, ckm.namespace, key)
}

func (ckm *CacheKeyManager) BuildTagKey(tag string) string {
	return fmt.Sprintf("%s:%s:%s", PrefixTag, ckm.namespace, tag)
}

func (ckm *CacheKeyManager) BuildMetaKey(key string) string {
	return fmt.Sprintf("%s:%s:%s", PrefixMeta, ckm.namespace, key)
}

func (ckm *CacheKeyManager) BuildAnalyticsKey(metricType, period string) string {
	return ckm.BuildKey(PrefixAnalytics, metricType, period)
}

func (ckm *CacheKeyManager) BuildWhitelistKey(whitelistType, target string) string {
	return ckm.BuildKey(PrefixWhitelist, whitelistType, target)
}

func (ckm *CacheKeyManager) BuildAlertKey(alertType, identifier string) string {
	return ckm.BuildKey(PrefixAlert, alertType, identifier)
}

func (ckm *CacheKeyManager) BuildWarmupKey(taskName string) string {
	return ckm.BuildKey(PrefixWarmup, taskName)
}

func (ckm *CacheKeyManager) BuildMetricsKey(metricType string) string {
	return ckm.BuildKey(PrefixMetrics, metricType)
}

func (ckm *CacheKeyManager) GetPattern(prefix CacheKeyPrefix) string {
	return ckm.BuildKey(prefix) + ":*"
}

func (ckm *CacheKeyManager) GetPrefixes() []CacheKeyPrefix {
	return []CacheKeyPrefix{
		PrefixCaptcha,
		PrefixSession,
		PrefixUser,
		PrefixApplication,
		PrefixBlacklist,
		PrefixStats,
		PrefixRateLimit,
		PrefixBehavior,
		PrefixConfig,
		PrefixToken,
		PrefixLock,
		PrefixWarmup,
		PrefixMetrics,
		PrefixAnalytics,
		PrefixWhitelist,
		PrefixAlert,
	}
}

func (ckm *CacheKeyManager) BuildAllPatterns() []string {
	patterns := make([]string, len(ckm.GetPrefixes()))
	for i, prefix := range ckm.GetPrefixes() {
		patterns[i] = ckm.GetPattern(prefix)
	}
	return patterns
}

var (
	globalCacheKeyManager *CacheKeyManager
	globalKeyManagerOnce  sync.Once
)

func InitCacheKeyManager(namespace CacheKeyNamespace) {
	globalKeyManagerOnce.Do(func() {
		globalCacheKeyManager = NewCacheKeyManager(namespace)
	})
}

func GetCacheKeyManager() *CacheKeyManager {
	if globalCacheKeyManager == nil {
		InitCacheKeyManager(NamespaceGlobal)
	}
	return globalCacheKeyManager
}

func BuildCacheKey(prefix CacheKeyPrefix, segments ...string) string {
	return GetCacheKeyManager().BuildKey(prefix, segments...)
}

func BuildCaptchaKey(captchaID string) string {
	return GetCacheKeyManager().BuildCaptchaKey(captchaID)
}

func BuildSessionKey(token string) string {
	return GetCacheKeyManager().BuildSessionKey(token)
}

func BuildUserKey(userID string) string {
	return GetCacheKeyManager().BuildUserKey(userID)
}

func BuildApplicationKey(apiKey string) string {
	return GetCacheKeyManager().BuildApplicationKey(apiKey)
}

func BuildBlacklistKey(targetType, target string) string {
	return GetCacheKeyManager().BuildBlacklistKey(targetType, target)
}

func BuildStatsKey(metric string) string {
	return GetCacheKeyManager().BuildStatsKey(metric)
}

func BuildRateLimitKey(identifier string, window int) string {
	return GetCacheKeyManager().BuildRateLimitKey(identifier, window)
}

func BuildBehaviorKey(sessionID string) string {
	return GetCacheKeyManager().BuildBehaviorKey(sessionID)
}

func BuildConfigKey(configType, configID string) string {
	return GetCacheKeyManager().BuildConfigKey(configType, configID)
}

func BuildLockKey(resource string) string {
	return GetCacheKeyManager().BuildLockKey(resource)
}

func BuildTokenKey(token string) string {
	return GetCacheKeyManager().BuildTokenKey(token)
}

func BuildWarmupKey(taskName string) string {
	return GetCacheKeyManager().BuildWarmupKey(taskName)
}

func BuildMetricsKey(metricType string) string {
	return GetCacheKeyManager().BuildMetricsKey(metricType)
}

func BuildAnalyticsKey(metricType, period string) string {
	return GetCacheKeyManager().BuildAnalyticsKey(metricType, period)
}

func BuildWhitelistKey(whitelistType, target string) string {
	return GetCacheKeyManager().BuildWhitelistKey(whitelistType, target)
}

func BuildAlertKey(alertType, identifier string) string {
	return GetCacheKeyManager().BuildAlertKey(alertType, identifier)
}
