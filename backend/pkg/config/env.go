package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type EnvManager struct {
	mu         sync.RWMutex
	envCache   map[string]string
	loaded     bool
	envFiles   []string
}

var (
	envManager     *EnvManager
	envManagerOnce sync.Once
	envManagerMu   sync.RWMutex
)

func GetEnvManager() *EnvManager {
	envManagerMu.Lock()
	defer envManagerMu.Unlock()

	envManagerOnce.Do(func() {
		envManager = &EnvManager{
			envCache: make(map[string]string),
			envFiles: []string{".env", ".env.local", ".env.production", ".env.development"},
		}
	})

	return envManager
}

func (m *EnvManager) LoadEnvFiles() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.loaded {
		return nil
	}

	for _, envFile := range m.envFiles {
		if err := m.loadEnvFile(envFile); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to load %s: %w", envFile, err)
			}
		}
	}

	m.loaded = true
	return nil
}

func (m *EnvManager) loadEnvFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"'")

		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
		m.envCache[key] = value
	}

	return nil
}

func (m *EnvManager) Get(key string, defaultValue string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	if value, exists := m.envCache[key]; exists {
		return value
	}
	return defaultValue
}

func (m *EnvManager) GetInt(key string, defaultValue int) int {
	valueStr := m.Get(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func (m *EnvManager) GetInt64(key string, defaultValue int64) int64 {
	valueStr := m.Get(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return defaultValue
	}
	return value
}

func (m *EnvManager) GetBool(key string, defaultValue bool) bool {
	valueStr := m.Get(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func (m *EnvManager) GetDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := m.Get(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func (m *EnvManager) GetSlice(key string, defaultValue []string, sep string) []string {
	valueStr := m.Get(key, "")
	if valueStr == "" {
		return defaultValue
	}

	return strings.Split(valueStr, sep)
}

func (m *EnvManager) Set(key string, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	os.Setenv(key, value)
	m.envCache[key] = value
}

func (m *EnvManager) MustGet(key string) (string, error) {
	value := m.Get(key, "")
	if value == "" {
		return "", fmt.Errorf("environment variable %s is required", key)
	}
	return value, nil
}

func (m *EnvManager) MustGetInt(key string) (int, error) {
	valueStr, err := m.MustGet(key)
	if err != nil {
		return 0, err
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("environment variable %s must be a valid integer", key)
	}
	return value, nil
}

func (m *EnvManager) GetAll() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string)
	for key, value := range m.envCache {
		result[key] = value
	}
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func (m *EnvManager) IsProduction() bool {
	return m.Get("GIN_MODE", "debug") == "release" ||
		m.Get("APP_ENV", "development") == "production"
}

func (m *EnvManager) IsDevelopment() bool {
	return !m.IsProduction()
}

func (m *EnvManager) ClearCache() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.envCache = make(map[string]string)
	m.loaded = false
}
