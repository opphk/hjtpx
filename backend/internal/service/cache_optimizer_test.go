package service

import (
	"testing"
)

func TestNewCacheOptimizerService(t *testing.T) {
	optimizer := NewCacheOptimizerService()
	if optimizer == nil {
		t.Error("NewCacheOptimizerService 返回了 nil")
	}
}

func TestOptimizeCache(t *testing.T) {
	optimizer := NewCacheOptimizerService()
	
	err := optimizer.OptimizeCache()
	if err != nil {
		t.Errorf("优化缓存失败: %v", err)
	}
}

func TestAnalyzeCacheEfficiency(t *testing.T) {
	optimizer := NewCacheOptimizerService()
	
	analysis, err := optimizer.AnalyzeCacheEfficiency()
	if err != nil {
		t.Errorf("分析缓存效率失败: %v", err)
	}
	if analysis == nil {
		t.Error("缓存效率分析结果不应为 nil")
	}
}

func TestGetCacheHitRate(t *testing.T) {
	optimizer := NewCacheOptimizerService()
	
	hitRate, err := optimizer.GetCacheHitRate()
	if err != nil {
		t.Errorf("获取缓存命中率失败: %v", err)
	}
	if hitRate < 0 || hitRate > 100 {
		t.Error("缓存命中率应该在 0-100 之间")
	}
}

func TestSetCacheTTL(t *testing.T) {
	optimizer := NewCacheOptimizerService()
	
	err := optimizer.SetCacheTTL("user-session", 3600)
	if err != nil {
		t.Errorf("设置缓存 TTL 失败: %v", err)
	}
}

func TestGetCacheTTL(t *testing.T) {
	optimizer := NewCacheOptimizerService()
	
	ttl, err := optimizer.GetCacheTTL("user-session")
	if err != nil {
		t.Errorf("获取缓存 TTL 失败: %v", err)
	}
	if ttl < 0 {
		t.Error("缓存 TTL 不应为负数")
	}
}

func TestClearExpiredCache(t *testing.T) {
	optimizer := NewCacheOptimizerService()
	
	err := optimizer.ClearExpiredCache()
	if err != nil {
		t.Errorf("清空过期缓存失败: %v", err)
	}
}

func TestGetCacheSize(t *testing.T) {
	optimizer := NewCacheOptimizerService()
	
	size, err := optimizer.GetCacheSize()
	if err != nil {
		t.Errorf("获取缓存大小失败: %v", err)
	}
	if size < 0 {
		t.Error("缓存大小不应为负数")
	}
}

func TestSetCacheMaxSize(t *testing.T) {
	optimizer := NewCacheOptimizerService()
	
	err := optimizer.SetCacheMaxSize(1024 * 1024 * 100)
	if err != nil {
		t.Errorf("设置缓存最大大小失败: %v", err)
	}
}

func TestGetCacheStats(t *testing.T) {
	optimizer := NewCacheOptimizerService()
	
	stats, err := optimizer.GetCacheStats()
	if err != nil {
		t.Errorf("获取缓存统计失败: %v", err)
	}
	if stats == nil {
		t.Error("缓存统计不应为 nil")
	}
}

func TestPreloadCache(t *testing.T) {
	optimizer := NewCacheOptimizerService()
	
	keys := []string{"key1", "key2", "key3"}
	err := optimizer.PreloadCache(keys)
	if err != nil {
		t.Errorf("预加载缓存失败: %v", err)
	}
}

func TestExportCacheConfig(t *testing.T) {
	optimizer := NewCacheOptimizerService()
	
	config, err := optimizer.ExportCacheConfig()
	if err != nil {
		t.Errorf("导出缓存配置失败: %v", err)
	}
	if config == "" {
		t.Error("缓存配置不应为空")
	}
}

func TestImportCacheConfig(t *testing.T) {
	optimizer := NewCacheOptimizerService()
	
	configJSON := `{"max_size":1000000,"default_ttl":3600,"compression":true}`
	err := optimizer.ImportCacheConfig(configJSON)
	if err != nil {
		t.Errorf("导入缓存配置失败: %v", err)
	}
}
