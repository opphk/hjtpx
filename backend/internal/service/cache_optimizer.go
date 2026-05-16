package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type CacheOptimizer struct {
	cacheService     *CacheService
	enhancedCache    *redis.EnhancedCache
	cacheWarmer      *redis.CacheWarmer
	adaptiveRefresher *redis.AdaptiveRefresher
	hotKeys          map[string]int64
	hotKeysMu        sync.RWMutex
	initialized      bool
}

var (
	globalCacheOptimizer *CacheOptimizer
	optimizerOnce        sync.Once
)

const (
	BlacklistCachePrefix   = "blacklist:"
	ApplicationCachePrefix = "application:"
	CaptchaCachePrefix     = "captcha:"
	SessionCachePrefix     = "session:"
	StatsCachePrefix       = "stats:"
	HotKeyThreshold        = 100
)

func GetCacheOptimizer() *CacheOptimizer {
	optimizerOnce.Do(func() {
		globalCacheOptimizer = &CacheOptimizer{
			cacheService:  NewCacheService(),
			enhancedCache: redis.GetEnhancedCache(),
			cacheWarmer:   redis.GetGlobalWarmer(),
			hotKeys:       make(map[string]int64),
		}
	})
	return globalCacheOptimizer
}

func (co *CacheOptimizer) Initialize(ctx context.Context) error {
	if co.initialized {
		return nil
	}

	co.registerWarmupTasks()
	co.cacheWarmer.Start()
	co.initialized = true

	if err := co.WarmupCriticalData(ctx); err != nil {
		return fmt.Errorf("failed to warmup critical data: %w", err)
	}

	return nil
}

func (co *CacheOptimizer) registerWarmupTasks() {
	tasks := []*redis.WarmupTask{
		{
			Name:      "blacklist-active",
			Key:       "warmup:blacklist:active",
			TTL:       5 * time.Minute,
			Frequency: 1 * time.Minute,
			Loader:    co.loadActiveBlacklist,
			Enabled:   true,
		},
		{
			Name:      "stats-summary",
			Key:       "warmup:stats:summary",
			TTL:       10 * time.Minute,
			Frequency: 5 * time.Minute,
			Loader:    co.loadStatsSummary,
			Enabled:   true,
		},
	}

	for _, task := range tasks {
		co.cacheWarmer.AddTask(task)
	}
}

func (co *CacheOptimizer) WarmupCriticalData(ctx context.Context) error {
	db := database.GetDB()
	if db == nil {
		return nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 4)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := co.warmupBlacklist(ctx); err != nil {
			errChan <- fmt.Errorf("blacklist warmup failed: %w", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := co.warmupApplications(ctx); err != nil {
			errChan <- fmt.Errorf("applications warmup failed: %w", err)
		}
	}()

	wg.Wait()
	close(errChan)

	for err := range errChan {
		return err
	}

	return nil
}

func (co *CacheOptimizer) warmupBlacklist(ctx context.Context) error {
	db := database.GetDB()
	if db == nil {
		return nil
	}

	var items []models.Blacklist
	if err := db.Where("status = ?", "active").Find(&items).Error; err != nil {
		return err
	}

	for _, item := range items {
		key := fmt.Sprintf("%s%s:%s", BlacklistCachePrefix, item.Type, item.Target)
		data, _ := json.Marshal(item)
		co.cacheService.SetWithTTL(ctx, key, data, 1*time.Hour)
	}

	return nil
}

func (co *CacheOptimizer) warmupApplications(ctx context.Context) error {
	db := database.GetDB()
	if db == nil {
		return nil
	}

	var apps []models.Application
	if err := db.Where("is_active = ?", true).Find(&apps).Error; err != nil {
		return err
	}

	for _, app := range apps {
		key := fmt.Sprintf("%s%s", ApplicationCachePrefix, app.APIKey)
		data, _ := json.Marshal(app)
		co.cacheService.SetWithTTL(ctx, key, data, 30*time.Minute)
	}

	return nil
}

func (co *CacheOptimizer) loadActiveBlacklist(ctx context.Context) ([]byte, error) {
	db := database.GetDB()
	if db == nil {
		return nil, nil
	}

	var items []models.Blacklist
	if err := db.Where("status = ?", "active").Find(&items).Error; err != nil {
		return nil, err
	}

	return json.Marshal(items)
}

func (co *CacheOptimizer) loadStatsSummary(ctx context.Context) ([]byte, error) {
	db := database.GetDB()
	if db == nil {
		return nil, nil
	}

	stats := make(map[string]interface{})
	
	var userCount int64
	db.Model(&models.User{}).Count(&userCount)
	stats["total_users"] = userCount

	var appCount int64
	db.Model(&models.Application{}).Where("is_active = ?", true).Count(&appCount)
	stats["active_applications"] = appCount

	var todayVerifications int64
	today := time.Now().Truncate(24 * time.Hour)
	db.Model(&models.Verification{}).Where("created_at >= ?", today).Count(&todayVerifications)
	stats["today_verifications"] = todayVerifications

	return json.Marshal(stats)
}

func (co *CacheOptimizer) GetBlacklistItem(ctx context.Context, targetType, target string) (*models.Blacklist, error) {
	key := fmt.Sprintf("%s%s:%s", BlacklistCachePrefix, targetType, target)
	
	var item models.Blacklist
	if err := co.cacheService.GetJSON(ctx, key, &item); err == nil {
		co.recordHotKey(key)
		return &item, nil
	}

	db := database.GetDB()
	if db == nil {
		return nil, nil
	}

	if err := db.Where("target = ? AND type = ? AND status = ?", target, targetType, "active").First(&item).Error; err != nil {
		return nil, err
	}

	co.cacheService.SetWithTTL(ctx, key, &item, 1*time.Hour)
	return &item, nil
}

func (co *CacheOptimizer) GetApplicationByAPIKey(ctx context.Context, apiKey string) (*models.Application, error) {
	key := fmt.Sprintf("%s%s", ApplicationCachePrefix, apiKey)
	
	var app models.Application
	if err := co.cacheService.GetJSON(ctx, key, &app); err == nil {
		co.recordHotKey(key)
		return &app, nil
	}

	db := database.GetDB()
	if db == nil {
		return nil, nil
	}

	if err := db.Where("api_key = ? AND is_active = ?", apiKey, true).First(&app).Error; err != nil {
		return nil, err
	}

	co.cacheService.SetWithTTL(ctx, key, &app, 30*time.Minute)
	return &app, nil
}

func (co *CacheOptimizer) InvalidateApplicationCache(ctx context.Context, apiKey string) {
	key := fmt.Sprintf("%s%s", ApplicationCachePrefix, apiKey)
	co.cacheService.Delete(ctx, key)
}

func (co *CacheOptimizer) InvalidateBlacklistCache(ctx context.Context, targetType, target string) {
	key := fmt.Sprintf("%s%s:%s", BlacklistCachePrefix, targetType, target)
	co.cacheService.Delete(ctx, key)
}

func (co *CacheOptimizer) recordHotKey(key string) {
	co.hotKeysMu.Lock()
	defer co.hotKeysMu.Unlock()
	co.hotKeys[key]++
}

func (co *CacheOptimizer) GetHotKeys() []string {
	co.hotKeysMu.RLock()
	defer co.hotKeysMu.RUnlock()

	var hotKeys []string
	for key, count := range co.hotKeys {
		if count >= HotKeyThreshold {
			hotKeys = append(hotKeys, key)
		}
	}
	return hotKeys
}

func (co *CacheOptimizer) ResetHotKeys() {
	co.hotKeysMu.Lock()
	defer co.hotKeysMu.Unlock()
	co.hotKeys = make(map[string]int64)
}

type PerformanceCacheStats struct {
	Hits          int64         `json:"hits"`
	Misses        int64         `json:"misses"`
	HitRate       float64       `json:"hit_rate"`
	HotKeys       []string      `json:"hot_keys"`
	MemoryUsage   int64         `json:"memory_usage,omitempty"`
}

func (co *CacheOptimizer) GetStats(ctx context.Context) *PerformanceCacheStats {
	enhancedStats := co.enhancedCache.GetStats()
	
	hitRate := 0.0
	total := enhancedStats.Hits + enhancedStats.Misses
	if total > 0 {
		hitRate = float64(enhancedStats.Hits) / float64(total) * 100
	}

	return &PerformanceCacheStats{
		Hits:    enhancedStats.Hits,
		Misses:  enhancedStats.Misses,
		HitRate: hitRate,
		HotKeys: co.GetHotKeys(),
	}
}

func (co *CacheOptimizer) BatchSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	pipeItems := make(map[string][]byte)
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			continue
		}
		pipeItems[key] = data
	}

	if len(pipeItems) > 0 {
		for key, value := range pipeItems {
			co.cacheService.SetWithTTL(ctx, key, value, ttl)
		}
	}

	return nil
}

func (co *CacheOptimizer) Shutdown(ctx context.Context) {
	if co.cacheWarmer != nil {
		co.cacheWarmer.Stop()
	}
}
