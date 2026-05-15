package config

import "time"

type OptimizationConfig struct {
	ImageCache    ImageCacheConfig       `yaml:"image_cache"`
	Redis         RedisOptimizationConfig `yaml:"redis"`
	Database      DatabaseOptimizationConfig `yaml:"database"`
	RateLimit     RateLimitConfig       `yaml:"rate_limit"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
}

type ImageCacheConfig struct {
	MaxSize int           `yaml:"max_size"`
	TTL     time.Duration `yaml:"ttl"`
	Enabled bool          `yaml:"enabled"`
}

type RedisOptimizationConfig struct {
	MaxRetries    int           `yaml:"max_retries"`
	PoolSize      int           `yaml:"pool_size"`
	MinIdleConns  int           `yaml:"min_idle_conns"`
	DialTimeout   time.Duration `yaml:"dial_timeout"`
	ReadTimeout   time.Duration `yaml:"read_timeout"`
	WriteTimeout  time.Duration `yaml:"write_timeout"`
	LocalCacheTTL time.Duration `yaml:"local_cache_ttl"`
}

type DatabaseOptimizationConfig struct {
	MaxOpenConns       int           `yaml:"max_open_conns"`
	MaxIdleConns       int           `yaml:"max_idle_conns"`
	ConnMaxLifetime    time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime    time.Duration `yaml:"conn_max_idle_time"`
	SlowQueryLog       bool          `yaml:"slow_query_log"`
	SlowQueryThreshold time.Duration `yaml:"slow_query_threshold"`
}

type RateLimitConfig struct {
	Enabled        bool  `yaml:"enabled"`
	RequestsPerSec int64 `yaml:"requests_per_sec"`
	BurstSize      int64 `yaml:"burst_size"`
}

type CircuitBreakerConfig struct {
	FailureThreshold int           `yaml:"failure_threshold"`
	SuccessThreshold int           `yaml:"success_threshold"`
	Timeout          time.Duration `yaml:"timeout"`
}

func DefaultOptimizationConfig() *OptimizationConfig {
	return &OptimizationConfig{
		ImageCache: ImageCacheConfig{
			MaxSize: 1000,
			TTL:     10 * time.Minute,
			Enabled: true,
		},
		Redis: RedisOptimizationConfig{
			MaxRetries:    3,
			PoolSize:      100,
			MinIdleConns:  10,
			DialTimeout:   5 * time.Second,
			ReadTimeout:   3 * time.Second,
			WriteTimeout:  3 * time.Second,
			LocalCacheTTL: 10 * time.Minute,
		},
		Database: DatabaseOptimizationConfig{
			MaxOpenConns:       100,
			MaxIdleConns:       10,
			ConnMaxLifetime:    1 * time.Hour,
			ConnMaxIdleTime:    30 * time.Minute,
			SlowQueryLog:       true,
			SlowQueryThreshold: 100 * time.Millisecond,
		},
		RateLimit: RateLimitConfig{
			Enabled:        true,
			RequestsPerSec: 100,
			BurstSize:      200,
		},
		CircuitBreaker: CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          30 * time.Second,
		},
	}
}
