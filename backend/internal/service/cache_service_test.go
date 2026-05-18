package service

import (
	"testing"
)

func TestNewCacheService(t *testing.T) {
	cacheService := NewCacheService()
	if cacheService == nil {
		t.Error("NewCacheService 返回了 nil")
	}
}

func TestSetAndGet(t *testing.T) {
	cacheService := NewCacheService()
	
	err := cacheService.Set("test-key", "test-value", 300)
	if err != nil {
		t.Errorf("设置缓存失败: %v", err)
	}
	
	value, err := cacheService.Get("test-key")
	if err != nil {
		t.Errorf("获取缓存失败: %v", err)
	}
	if value != "test-value" {
		t.Errorf("缓存值不匹配: 期望 test-value, 实际 %v", value)
	}
}

func TestGet_NonExistent(t *testing.T) {
	cacheService := NewCacheService()
	
	value, err := cacheService.Get("nonexistent-key")
	if err == nil {
		t.Error("不存在的键应该返回错误")
	}
	if value != nil {
		t.Error("不存在的键应该返回 nil 值")
	}
}

func TestDelete(t *testing.T) {
	cacheService := NewCacheService()
	
	err := cacheService.Set("test-key-delete", "test-value", 300)
	if err != nil {
		t.Skipf("无法设置缓存，跳过测试: %v", err)
	}
	
	err = cacheService.Delete("test-key-delete")
	if err != nil {
		t.Errorf("删除缓存失败: %v", err)
	}
	
	value, err := cacheService.Get("test-key-delete")
	if err == nil && value != nil {
		t.Error("键应该已被删除")
	}
}

func TestExists(t *testing.T) {
	cacheService := NewCacheService()
	
	err := cacheService.Set("test-key-exists", "test-value", 300)
	if err != nil {
		t.Skipf("无法设置缓存，跳过测试: %v", err)
	}
	
	exists, err := cacheService.Exists("test-key-exists")
	if err != nil {
		t.Errorf("检查存在性失败: %v", err)
	}
	if !exists {
		t.Error("键应该存在")
	}
	
	exists, err = cacheService.Exists("nonexistent-key")
	if err != nil {
		t.Errorf("检查存在性失败: %v", err)
	}
	if exists {
		t.Error("不存在的键不应该存在")
	}
}

func TestExpire(t *testing.T) {
	cacheService := NewCacheService()
	
	err := cacheService.Set("test-key-expire", "test-value", 300)
	if err != nil {
		t.Skipf("无法设置缓存，跳过测试: %v", err)
	}
	
	err = cacheService.Expire("test-key-expire", 600)
	if err != nil {
		t.Errorf("设置过期时间失败: %v", err)
	}
}

func TestTTL(t *testing.T) {
	cacheService := NewCacheService()
	
	err := cacheService.Set("test-key-ttl", "test-value", 300)
	if err != nil {
		t.Skipf("无法设置缓存，跳过测试: %v", err)
	}
	
	ttl, err := cacheService.TTL("test-key-ttl")
	if err != nil {
		t.Errorf("获取 TTL 失败: %v", err)
	}
	if ttl <= 0 {
		t.Error("TTL 应该大于 0")
	}
}

func TestFlush(t *testing.T) {
	cacheService := NewCacheService()
	
	err := cacheService.Set("test-key-flush1", "value1", 300)
	if err != nil {
		t.Skipf("无法设置缓存，跳过测试: %v", err)
	}
	
	err = cacheService.Set("test-key-flush2", "value2", 300)
	if err != nil {
		t.Skipf("无法设置缓存，跳过测试: %v", err)
	}
	
	err = cacheService.Flush()
	if err != nil {
		t.Errorf("清空缓存失败: %v", err)
	}
}

func TestIncrement(t *testing.T) {
	cacheService := NewCacheService()
	
	err := cacheService.Set("test-counter", "10", 300)
	if err != nil {
		t.Skipf("无法设置缓存，跳过测试: %v", err)
	}
	
	newValue, err := cacheService.Increment("test-counter")
	if err != nil {
		t.Errorf("递增失败: %v", err)
	}
	if newValue != 11 {
		t.Errorf("递增结果不正确: 期望 11, 实际 %d", newValue)
	}
}

func TestDecrement(t *testing.T) {
	cacheService := NewCacheService()
	
	err := cacheService.Set("test-counter-dec", "10", 300)
	if err != nil {
		t.Skipf("无法设置缓存，跳过测试: %v", err)
	}
	
	newValue, err := cacheService.Decrement("test-counter-dec")
	if err != nil {
		t.Errorf("递减失败: %v", err)
	}
	if newValue != 9 {
		t.Errorf("递减结果不正确: 期望 9, 实际 %d", newValue)
	}
}
