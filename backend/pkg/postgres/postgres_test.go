package postgres

import (
	"testing"

	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestConnect_Success(t *testing.T) {
	cfg := &config.PostgresConfig{
		Host:            "localhost",
		Port:            "5432",
		User:            "testuser",
		Password:        "testpass",
		DBName:          "testdb",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime:  300,
	}

	err := Connect(cfg)
	if err != nil {
		t.Skip("PostgreSQL not available, skipping integration test")
	}

	assert.NoError(t, err)
	assert.NotNil(t, DB)

	err = Close()
	assert.NoError(t, err)
}

func TestConnect_InvalidDSN(t *testing.T) {
	cfg := &config.PostgresConfig{
		Host:    "invalid-host",
		Port:    "5432",
		User:    "testuser",
		Password: "testpass",
		DBName:  "testdb",
		SSLMode: "disable",
	}

	err := Connect(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open database")
}

func TestClose_NilDB(t *testing.T) {
	DB = nil
	err := Close()
	assert.NoError(t, err)
}

func TestClose_WithDB(t *testing.T) {
	cfg := &config.PostgresConfig{
		Host:            "localhost",
		Port:            "5432",
		User:            "testuser",
		Password:        "testpass",
		DBName:          "testdb",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 300,
	}

	err := Connect(cfg)
	if err != nil {
		t.Skip("PostgreSQL not available, skipping integration test")
	}

	err = Close()
	assert.NoError(t, err)
	assert.Nil(t, DB)
}

func TestGetDB_Nil(t *testing.T) {
	DB = nil
	db := GetDB()
	assert.Nil(t, db)
}

func TestGetDB_WithConnection(t *testing.T) {
	cfg := &config.PostgresConfig{
		Host:            "localhost",
		Port:            "5432",
		User:            "testuser",
		Password:        "testpass",
		DBName:          "testdb",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 300,
	}

	err := Connect(cfg)
	if err != nil {
		t.Skip("PostgreSQL not available, skipping integration test")
	}

	db := GetDB()
	assert.NotNil(t, db)

	Close()
}

func TestConnect_ConnectionPool(t *testing.T) {
	cfg := &config.PostgresConfig{
		Host:            "localhost",
		Port:            "5432",
		User:            "testuser",
		Password:        "testpass",
		DBName:          "testdb",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 600,
	}

	err := Connect(cfg)
	if err != nil {
		t.Skip("PostgreSQL not available, skipping integration test")
	}

	db := GetDB()
	assert.NotNil(t, db)

	stats := db.Stats()
	assert.Equal(t, 25, stats.MaxOpenConnections)
	assert.Equal(t, 10, stats.MaxIdleConnections)

	Close()
}

func TestPostgresConfig_Structure(t *testing.T) {
	cfg := &config.PostgresConfig{
		Host:            "localhost",
		Port:            "5432",
		User:            "testuser",
		Password:        "testpass",
		DBName:          "testdb",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 300,
	}

	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, "5432", cfg.Port)
	assert.Equal(t, "testuser", cfg.User)
	assert.Equal(t, "testpass", cfg.Password)
	assert.Equal(t, "testdb", cfg.DBName)
	assert.Equal(t, "disable", cfg.SSLMode)
	assert.Equal(t, 10, cfg.MaxOpenConns)
	assert.Equal(t, 5, cfg.MaxIdleConns)
	assert.Equal(t, 300, cfg.ConnMaxLifetime)
}
