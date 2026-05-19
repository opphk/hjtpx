package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository"
	goredis "github.com/redis/go-redis/v9"
)

type ConfigService struct {
	repo    *repository.ConfigRepo
	cache   *ConfigCache
	inMemory map[string]interface{}
	mu       sync.RWMutex
}

type ConfigCache struct {
	client *goredis.Client
	ctx    context.Context
}

func NewConfigCache(client *goredis.Client) *ConfigCache {
	return &ConfigCache{
		client: client,
		ctx:    context.Background(),
	}
}

func NewConfigService(repo *repository.ConfigRepo, cache *ConfigCache) *ConfigService {
	return &ConfigService{
		repo:      repo,
		cache:     cache,
		inMemory:  make(map[string]interface{}),
	}
}

func NewConfigServiceForTest() *ConfigService {
	return &ConfigService{
		repo:      nil,
		cache:     nil,
		inMemory:  make(map[string]interface{}),
	}
}

func (c *ConfigCache) SetAll(configs map[string]string) error {
	if c.client == nil {
		return nil
	}
	data, err := json.Marshal(configs)
	if err != nil {
		return fmt.Errorf("failed to marshal configs: %w", err)
	}
	return c.client.Set(c.ctx, "config:all", data, 10*time.Minute).Err()
}

func (c *ConfigCache) GetAll() (map[string]string, error) {
	if c.client == nil {
		return nil, errors.New("redis client is nil")
	}
	data, err := c.client.Get(c.ctx, "config:all").Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, errors.New("cache miss")
		}
		return nil, err
	}
	var configs map[string]string
	err = json.Unmarshal(data, &configs)
	return configs, err
}

func (c *ConfigCache) Clear() error {
	if c.client == nil {
		return nil
	}
	return c.client.Del(c.ctx, "config:all").Err()
}

func (s *ConfigService) GetAll() (map[string]string, error) {
	cached, err := s.cache.GetAll()
	if err == nil && cached != nil {
		return cached, nil
	}

	configs, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, config := range configs {
		result[config.Key] = config.Value
	}

	_ = s.cache.SetAll(result)

	return result, nil
}

func (s *ConfigService) GetByKey(key string) (string, error) {
	configs, err := s.GetAll()
	if err != nil {
		return "", err
	}
	value, exists := configs[key]
	if !exists {
		return "", errors.New("config key not found")
	}
	return value, nil
}

func (s *ConfigService) Update(config map[string]string) error {
	for key, value := range config {
		if err := s.Validate(key, value); err != nil {
			return err
		}
		if err := s.repo.Upsert(key, value); err != nil {
			return fmt.Errorf("failed to update config %s: %w", key, err)
		}
	}

	if err := s.cache.Clear(); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	return nil
}

func (s *ConfigService) Validate(key, value string) error {
	switch key {
	case "captcha.difficulty":
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 5 {
			return errors.New("难度等级必须在1-5之间")
		}
	case "captcha.timeout":
		n, err := strconv.Atoi(value)
		if err != nil || n < 60 || n > 600 {
			return errors.New("有效期必须在60-600秒之间")
		}
	case "captcha.max_attempts":
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 10 {
			return errors.New("最大尝试次数必须在1-10之间")
		}
	case "risk.threshold_pass":
		n, err := strconv.Atoi(value)
		if err != nil || n < 60 || n > 100 {
			return errors.New("通过阈值必须在60-100之间")
		}
	case "risk.threshold_review":
		n, err := strconv.Atoi(value)
		if err != nil || n < 40 || n > 80 {
			return errors.New("审查阈值必须在40-80之间")
		}
	case "rate_limit.max_per_ip":
		n, err := strconv.Atoi(value)
		if err != nil || n < 10 || n > 1000 {
			return errors.New("频率限制必须在10-1000之间")
		}
	case "session.timeout":
		n, err := strconv.Atoi(value)
		if err != nil || n < 5 || n > 1440 {
			return errors.New("会话超时必须在5-1440分钟之间")
		}
	}
	return nil
}

func (s *ConfigService) InitializeDefaults() error {
	defaults := map[string]string{
		"captcha.difficulty":                "3",
		"captcha.timeout":                   "120",
		"captcha.max_attempts":              "3",
		"risk.threshold_pass":               "75",
		"risk.threshold_review":             "60",
		"risk.enable_env_check":             "true",
		"rate_limit.enabled":                "true",
		"rate_limit.max_per_ip":             "100",
		"session.timeout":                   "30",
		"session.storage":                   "redis",
		"security.enable_csrf":              "true",
		"security.enable_captcha":           "true",
		"security.enable_replay_protection": "true",
	}

	existingConfigs, err := s.GetAll()
	if err != nil {
		return err
	}

	for key, value := range defaults {
		if _, exists := existingConfigs[key]; !exists {
			if err := s.repo.Upsert(key, value); err != nil {
				return fmt.Errorf("failed to initialize default config %s: %w", key, err)
			}
		}
	}

	return s.cache.Clear()
}

func (s *ConfigService) GetConfig() (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]interface{})
	for k, v := range s.inMemory {
		result[k] = v
	}
	return result, nil
}

func (s *ConfigService) UpdateConfig(config map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range config {
		s.inMemory[k] = v
	}
	return nil
}

func (s *ConfigService) GetConfigValue(key string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inMemory[key]
}

func (s *ConfigService) SetConfigValue(key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inMemory[key] = value
	return nil
}

func (s *ConfigService) ReloadConfig() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inMemory = make(map[string]interface{})
	return nil
}

func (s *ConfigService) ValidateConfig(config map[string]interface{}) error {
	for key, value := range config {
		switch key {
		case "app.port":
			if _, ok := value.(int); !ok {
				return errors.New("app.port must be an integer")
			}
		}
	}
	return nil
}

func (s *ConfigService) ExportConfig() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := json.Marshal(s.inMemory)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *ConfigService) ImportConfig(configJSON string) error {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range config {
		s.inMemory[k] = v
	}
	return nil
}
