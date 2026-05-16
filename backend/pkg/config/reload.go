package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ConfigReloader struct {
	config        *Config
	configPath    string
	watchPath     string
	mu            sync.RWMutex
	lastModified  time.Time
	reloadCallbacks []func() error
	watcher       *ConfigWatcher
}

type ConfigWatcher struct {
	configPath string
	pollInterval time.Duration
	stopCh      chan struct{}
	callback    func() error
}

var globalReloader *ConfigReloader

func NewConfigReloader(configPath string) (*ConfigReloader, error) {
	cfg := LoadConfig()

	reloader := &ConfigReloader{
		config:        cfg,
		configPath:    configPath,
		watchPath:    filepath.Dir(configPath),
		reloadCallbacks: make([]func() error, 0),
	}

	if err := reloader.loadFromFile(configPath); err != nil {
		return nil, err
	}

	return reloader, nil
}

func InitConfigReloader(configPath string) error {
	var err error
	globalReloader, err = NewConfigReloader(configPath)
	if err != nil {
		return err
	}

	globalReloader.StartWatching()
	return nil
}

func GetConfigReloader() *ConfigReloader {
	return globalReloader
}

func (r *ConfigReloader) GetConfig() *Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

func (r *ConfigReloader) loadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var fileConfig Config
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	envConfig := r.loadFromEnv()
	r.mergeConfigs(&fileConfig, envConfig)

	r.mu.Lock()
	r.config = &fileConfig
	r.mu.Unlock()

	for _, callback := range r.reloadCallbacks {
		if err := callback(); err != nil {
			return fmt.Errorf("callback failed during reload: %w", err)
		}
	}

	info, err := os.Stat(path)
	if err == nil {
		r.lastModified = info.ModTime()
	}

	return nil
}

func (r *ConfigReloader) loadFromEnv() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Mode: getEnv("GIN_MODE", "release"),
		},
		Postgres: PostgresConfig{
			Host:            getEnv("POSTGRES_HOST", "localhost"),
			Port:            getEnv("POSTGRES_PORT", "5432"),
			User:            getEnv("POSTGRES_USER", "postgres"),
			Password:        getEnv("POSTGRES_PASSWORD", ""),
			DBName:          getEnv("POSTGRES_DB", "hjtpx_db"),
			SSLMode:         getEnv("POSTGRES_SSLMODE", "disable"),
			MaxOpenConns:    25,
			MaxIdleConns:    10,
			ConnMaxLifetime: 5,
		},
		Redis: RedisConfig{
			Host:       getEnv("REDIS_HOST", "localhost"),
			Port:       getEnv("REDIS_PORT", "6379"),
			Password:   getEnv("REDIS_PASSWORD", ""),
			DB:         0,
			MaxRetries: 3,
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", ""),
			ExpireHours: getEnvAsInt("JWT_EXPIRE_HOURS", 24),
		},
	}
}

func (r *ConfigReloader) mergeConfigs(file, env *Config) {
	if env.Server.Port != "" && env.Server.Port != "8080" {
		file.Server.Port = env.Server.Port
	}
	if env.Server.Mode != "" {
		file.Server.Mode = env.Server.Mode
	}

	if env.Postgres.Host != "" {
		file.Postgres.Host = env.Postgres.Host
	}
	if env.Postgres.Port != "" {
		file.Postgres.Port = env.Postgres.Port
	}
	if env.Postgres.User != "" {
		file.Postgres.User = env.Postgres.User
	}
	if env.Postgres.Password != "" {
		file.Postgres.Password = env.Postgres.Password
	}
	if env.Postgres.DBName != "" {
		file.Postgres.DBName = env.Postgres.DBName
	}

	if env.Redis.Host != "" {
		file.Redis.Host = env.Redis.Host
	}
	if env.Redis.Port != "" {
		file.Redis.Port = env.Redis.Port
	}
	if env.Redis.Password != "" {
		file.Redis.Password = env.Redis.Password
	}

	if env.JWT.Secret != "" {
		file.JWT.Secret = env.JWT.Secret
	}
	if env.JWT.ExpireHours > 0 {
		file.JWT.ExpireHours = env.JWT.ExpireHours
	}
}

func (r *ConfigReloader) Reload() error {
	return r.loadFromFile(r.configPath)
}

func (r *ConfigReloader) RegisterReloadCallback(callback func() error) {
	r.reloadCallbacks = append(r.reloadCallbacks, callback)
}

func (r *ConfigReloader) StartWatching() {
	r.watcher = &ConfigWatcher{
		configPath:   r.configPath,
		pollInterval: 5 * time.Second,
		stopCh:       make(chan struct{}),
		callback:     r.Reload,
	}
	go r.watcher.watch()
}

func (r *ConfigReloader) StopWatching() {
	if r.watcher != nil {
		close(r.watcher.stopCh)
		r.watcher = nil
	}
}

func (w *ConfigWatcher) watch() {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.checkAndReload()
		}
	}
}

func (w *ConfigWatcher) checkAndReload() {
	info, err := os.Stat(w.configPath)
	if err != nil {
		return
	}

	if info.ModTime().After(globalReloader.lastModified) {
		if err := globalReloader.loadFromFile(w.configPath); err != nil {
			fmt.Printf("Failed to reload config: %v\n", err)
		} else {
			fmt.Printf("Config reloaded successfully at %s\n", time.Now().Format(time.RFC3339))
		}
	}
}

func ReloadConfig() error {
	if globalReloader == nil {
		return errors.New("config reloader not initialized")
	}
	return globalReloader.Reload()
}

func GetCurrentConfig() *Config {
	if globalReloader == nil {
		return LoadConfig()
	}
	return globalReloader.GetConfig()
}

type ConfigFile struct {
	Server     ServerConfig     `json:"server"`
	Postgres   PostgresConfig   `json:"postgres"`
	Redis      RedisConfig      `json:"redis"`
	JWT        JWTConfig        `json:"jwt"`
	Security   SecurityConfigV2 `json:"security"`
	Logging    LoggingConfig    `json:"logging"`
	RateLimit  RateLimitConfigV2 `json:"rate_limit"`
}

type SecurityConfigV2 struct {
	EnableCSRF          bool     `json:"enable_csrf"`
	EnableXSS           bool     `json:"enable_xss"`
	EnableSignature     bool     `json:"enable_signature"`
	SignatureExpireSecs int      `json:"signature_expire_secs"`
	SignatureKey        string   `json:"signature_key"`
	WhitelistIPs        []string `json:"whitelist_ips"`
}

type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	OutputPath string `json:"output_path"`
	MaxSizeMB  int    `json:"max_size_mb"`
	MaxBackups int    `json:"max_backups"`
	MaxAgeDays int    `json:"max_age_days"`
	Compress   bool   `json:"compress"`
}

type RateLimitConfigV2 struct {
	Enabled        bool `json:"enabled"`
	DefaultLimit   int  `json:"default_limit"`
	WindowSecs     int  `json:"window_secs"`
	BurstLimit     int  `json:"burst_limit"`
	CleanupIntvlMs int  `json:"cleanup_interval_ms"`
}

func (r *ConfigReloader) ExportConfig(path string) error {
	cfg := r.GetConfig()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	if cfg.Server.Port == "" {
		return errors.New("server port is required")
	}

	if cfg.JWT.Secret == "" {
		return errors.New("JWT secret is required")
	}

	if cfg.Postgres.Host == "" {
		return errors.New("postgres host is required")
	}

	if cfg.Redis.Host == "" {
		return errors.New("redis host is required")
	}

	return nil
}
