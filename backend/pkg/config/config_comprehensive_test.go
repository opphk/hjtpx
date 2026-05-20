package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Structure(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port:         8080,
			Mode:         "release",
			ReadTimeout:  60,
			WriteTimeout: 60,
			IdleTimeout:  120,
		},
		Database: DatabaseConfig{
			Type:        "postgres",
			Host:        "localhost",
			Port:        5432,
			User:        "testuser",
			Password:    "testpass",
			DBName:      "testdb",
			SSLMode:     "disable",
			MaxOpenConns: 25,
			MaxIdleConns: 10,
			ConnMaxLifetime: 300,
		},
		Redis: RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "redispass",
			DB:       0,
			PoolSize: 100,
		},
		JWT: JWTConfig{
			Secret:        "jwt-secret-key",
			ExpireHours:   24,
			RefreshExpireDays: 7,
		},
		RateLimit: RateLimitConfig{
			Enabled:    true,
			MaxRequests: 100,
			Window:     60,
		},
	}

	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "release", cfg.Server.Mode)
	assert.Equal(t, "postgres", cfg.Database.Type)
	assert.Equal(t, "localhost", cfg.Redis.Host)
	assert.Equal(t, 6379, cfg.Redis.Port)
	assert.Equal(t, "jwt-secret-key", cfg.JWT.Secret)
	assert.True(t, cfg.RateLimit.Enabled)
}

func TestServerConfig_Structure(t *testing.T) {
	cfg := &ServerConfig{
		Port:         9090,
		Mode:         "debug",
		ReadTimeout:  30,
		WriteTimeout: 30,
		IdleTimeout:  60,
	}

	assert.Equal(t, 9090, cfg.Port)
	assert.Equal(t, "debug", cfg.Mode)
	assert.Equal(t, 30, cfg.ReadTimeout)
	assert.Equal(t, 30, cfg.WriteTimeout)
	assert.Equal(t, 60, cfg.IdleTimeout)
}

func TestDatabaseConfig_Structure(t *testing.T) {
	cfg := &DatabaseConfig{
		Type:          "mysql",
		Host:          "db.example.com",
		Port:          3306,
		User:          "admin",
		Password:      "adminpass",
		DBName:        "production",
		SSLMode:       "require",
		MaxOpenConns:  50,
		MaxIdleConns:  20,
		ConnMaxLifetime: 600,
	}

	assert.Equal(t, "mysql", cfg.Type)
	assert.Equal(t, "db.example.com", cfg.Host)
	assert.Equal(t, 3306, cfg.Port)
	assert.Equal(t, "admin", cfg.User)
	assert.Equal(t, "adminpass", cfg.Password)
	assert.Equal(t, "production", cfg.DBName)
	assert.Equal(t, "require", cfg.SSLMode)
	assert.Equal(t, 50, cfg.MaxOpenConns)
	assert.Equal(t, 20, cfg.MaxIdleConns)
	assert.Equal(t, 600, cfg.ConnMaxLifetime)
}

func TestRedisConfig_Structure(t *testing.T) {
	cfg := &RedisConfig{
		Host:     "redis.example.com",
		Port:     6380,
		Password: "redispassword",
		DB:       1,
		PoolSize: 200,
	}

	assert.Equal(t, "redis.example.com", cfg.Host)
	assert.Equal(t, 6380, cfg.Port)
	assert.Equal(t, "redispassword", cfg.Password)
	assert.Equal(t, 1, cfg.DB)
	assert.Equal(t, 200, cfg.PoolSize)
}

func TestJWTConfig_Structure(t *testing.T) {
	cfg := &JWTConfig{
		Secret:          "super-secret-key-12345",
		ExpireHours:     48,
		RefreshExpireDays: 14,
	}

	assert.Equal(t, "super-secret-key-12345", cfg.Secret)
	assert.Equal(t, 48, cfg.ExpireHours)
	assert.Equal(t, 14, cfg.RefreshExpireDays)
}

func TestRateLimitConfig_Structure(t *testing.T) {
	cfg := &RateLimitConfig{
		Enabled:     true,
		MaxRequests: 200,
		Window:      60,
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 200, cfg.MaxRequests)
	assert.Equal(t, 60, cfg.Window)
}

func TestSecurityConfig_Structure(t *testing.T) {
	cfg := &SecurityConfig{
		EnableCORS:           true,
		CORSAllowedOrigins:   []string{"https://example.com"},
		CSRFEnabled:          true,
		HTTPSEnabled:         true,
		HSTSEnabled:          true,
		ContentSecurityPolicy: "default-src 'self'",
		XFrameOptions:         "DENY",
		XContentTypeOptions:   "nosniff",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
	}

	assert.True(t, cfg.EnableCORS)
	assert.Contains(t, cfg.CORSAllowedOrigins, "https://example.com")
	assert.True(t, cfg.CSRFEnabled)
	assert.True(t, cfg.HTTPSEnabled)
	assert.True(t, cfg.HSTSEnabled)
	assert.Contains(t, cfg.ContentSecurityPolicy, "default-src 'self'")
	assert.Equal(t, "DENY", cfg.XFrameOptions)
}

func TestMicroserviceConfig_Structure(t *testing.T) {
	cfg := &MicroserviceConfig{
		Enabled:      true,
		RegistryAddr: "consul://localhost:8500",
		HealthCheckInterval: 10,
		DeregisterAfter:     30,
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, "consul://localhost:8500", cfg.RegistryAddr)
	assert.Equal(t, 10, cfg.HealthCheckInterval)
	assert.Equal(t, 30, cfg.DeregisterAfter)
}

func TestPostgresConfig_Structure(t *testing.T) {
	cfg := &PostgresConfig{
		Host:            "pg.example.com",
		Port:            "5432",
		User:            "pguser",
		Password:        "pgpass",
		DBName:          "hjtpx",
		SSLMode:         "disable",
		MaxOpenConns:    30,
		MaxIdleConns:    15,
		ConnMaxLifetime: 300,
	}

	assert.Equal(t, "pg.example.com", cfg.Host)
	assert.Equal(t, "5432", cfg.Port)
	assert.Equal(t, "pguser", cfg.User)
	assert.Equal(t, "pgpass", cfg.Password)
	assert.Equal(t, "hjtpx", cfg.DBName)
	assert.Equal(t, "disable", cfg.SSLMode)
	assert.Equal(t, 30, cfg.MaxOpenConns)
	assert.Equal(t, 15, cfg.MaxIdleConns)
	assert.Equal(t, 300, cfg.ConnMaxLifetime)
}
