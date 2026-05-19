package service

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type cacheItem struct {
	value      interface{}
	expireTime time.Time
}

type CacheService struct {
	cache map[string]cacheItem
	mu    sync.RWMutex
}

func NewCacheService() *CacheService {
	return &CacheService{
		cache: make(map[string]cacheItem),
	}
}

func (c *CacheService) Set(key string, value interface{}, ttlSeconds int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	expireTime := time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	c.cache[key] = cacheItem{
		value:      value,
		expireTime: expireTime,
	}
	
	return nil
}

func (c *CacheService) Get(key string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	item, ok := c.cache[key]
	if !ok {
		return nil, errors.New("key not found")
	}
	
	if time.Now().After(item.expireTime) {
		return nil, errors.New("key expired")
	}
	
	return item.value, nil
}

func (c *CacheService) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.cache, key)
	return nil
}

func (c *CacheService) Exists(key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	item, ok := c.cache[key]
	if !ok {
		return false, nil
	}
	
	if time.Now().After(item.expireTime) {
		return false, nil
	}
	
	return true, nil
}

func (c *CacheService) Expire(key string, ttlSeconds int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	item, ok := c.cache[key]
	if !ok {
		return errors.New("key not found")
	}
	
	item.expireTime = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	c.cache[key] = item
	
	return nil
}

func (c *CacheService) TTL(key string) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	item, ok := c.cache[key]
	if !ok {
		return -1, errors.New("key not found")
	}
	
	remaining := item.expireTime.Sub(time.Now())
	if remaining <= 0 {
		return 0, nil
	}
	
	return int64(remaining.Seconds()), nil
}

func (c *CacheService) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache = make(map[string]cacheItem)
	return nil
}

func (c *CacheService) Increment(key string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	item, ok := c.cache[key]
	if !ok {
		return 0, errors.New("key not found")
	}
	
	if time.Now().After(item.expireTime) {
		return 0, errors.New("key expired")
	}
	
	value, ok := item.value.(string)
	if !ok {
		return 0, errors.New("value is not a string")
	}
	
	var intValue int64
	if _, err := fmt.Sscanf(value, "%d", &intValue); err != nil {
		return 0, errors.New("value is not a valid integer")
	}
	
	intValue++
	item.value = fmt.Sprintf("%d", intValue)
	c.cache[key] = item
	
	return intValue, nil
}

func (c *CacheService) Decrement(key string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	item, ok := c.cache[key]
	if !ok {
		return 0, errors.New("key not found")
	}
	
	if time.Now().After(item.expireTime) {
		return 0, errors.New("key expired")
	}
	
	value, ok := item.value.(string)
	if !ok {
		return 0, errors.New("value is not a string")
	}
	
	var intValue int64
	if _, err := fmt.Sscanf(value, "%d", &intValue); err != nil {
		return 0, errors.New("value is not a valid integer")
	}
	
	intValue--
	item.value = fmt.Sprintf("%d", intValue)
	c.cache[key] = item
	
	return intValue, nil
}
