package database

import (
	"context"
	"os"
	"testing"

	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/postgres"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDatabaseMigration(t *testing.T) {
	os.Setenv("POSTGRES_DB", "verification")
	os.Setenv("POSTGRES_USER", "postgres")
	os.Setenv("POSTGRES_PASSWORD", "postgres")
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")

	cfg := config.LoadConfig()

	err := InitDB(cfg)
	require.NoError(t, err, "InitDB should succeed")
	require.NotNil(t, DB, "DB should not be nil")

	tables := []string{
		"users",
		"admins",
		"applications",
		"api_key_histories",
		"verifications",
		"behavior_data",
		"verification_logs",
	}

	for _, table := range tables {
		t.Run("Table_"+table, func(t *testing.T) {
			exists := DB.Migrator().HasTable(table)
			assert.True(t, exists, "Table %s should exist after migration", table)
		})
	}

	t.Run("DefaultAdmin", func(t *testing.T) {
		var count int64
		DB.Model(&adminModel{}).Count(&count)
		assert.Greater(t, count, int64(0), "Default admin should exist")
	})

	t.Run("PoolStats", func(t *testing.T) {
		stats, err := GetPoolStats()
		require.NoError(t, err)
		assert.Greater(t, stats.MaxOpenConnections, 0)
	})

	t.Run("Ping", func(t *testing.T) {
		err := Ping(context.Background())
		require.NoError(t, err)
	})
}

type adminModel struct {
	gorm.Model
	Username     string
	PasswordHash string
	IsSuperAdmin bool
}

func (adminModel) TableName() string {
	return "admins"
}

func TestDatabaseConnectionFailure(t *testing.T) {
	os.Setenv("POSTGRES_DB", "nonexistent_db")
	os.Setenv("POSTGRES_USER", "postgres")
	os.Setenv("POSTGRES_PASSWORD", "postgres")
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")

	cfg := config.LoadConfig()

	err := InitDB(cfg)
	assert.Error(t, err, "InitDB with nonexistent database should fail")
}

func TestPostgresConnection(t *testing.T) {
	os.Setenv("POSTGRES_DB", "verification")
	os.Setenv("POSTGRES_USER", "postgres")
	os.Setenv("POSTGRES_PASSWORD", "postgres")
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")

	cfg := config.LoadConfig()

	err := postgres.Connect(&cfg.Postgres)
	require.NoError(t, err, "Postgres Connect should succeed")
	require.NotNil(t, postgres.GetDB(), "GetDB() should not return nil")

	err = postgres.Close()
	require.NoError(t, err, "Close should succeed")
}

func TestRedisConnection(t *testing.T) {
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "6379")

	cfg := config.LoadConfig()

	err := redis.ConnectRedis(&cfg.Redis)
	require.NoError(t, err, "Redis Connect should succeed")
	require.NotNil(t, redis.GetClient(), "GetClient() should not return nil")

	err = redis.CloseRedis()
	require.NoError(t, err, "CloseRedis should succeed")
}