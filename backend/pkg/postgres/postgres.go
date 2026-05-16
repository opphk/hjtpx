package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/hjtpx/hjtpx/pkg/config"
)

var DB *sql.DB
var ctx = context.Background()

const (
	MaxOpenConnsDefault    = 100
	MaxIdleConnsDefault    = 10
	ConnMaxLifetimeDefault = 30
	ConnMaxIdleTimeDefault = 5
)

type DBStats struct {
	OpenConnections int
	InUse           int
	Idle            int
	WaitCount       int64
	WaitDuration    time.Duration
	MaxIdleClosed   int64
	MaxLifetimeClosed int64
}

func Connect(cfg *config.PostgresConfig) error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.DBName,
		cfg.SSLMode,
	)

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	maxOpenConns := cfg.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = MaxOpenConnsDefault
	}
	maxIdleConns := cfg.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = MaxIdleConnsDefault
	}
	connMaxLifetime := cfg.ConnMaxLifetime
	if connMaxLifetime <= 0 {
		connMaxLifetime = ConnMaxLifetimeDefault
	}

	DB.SetMaxOpenConns(maxOpenConns)
	DB.SetMaxIdleConns(maxIdleConns)
	DB.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Minute)
	DB.SetConnMaxIdleTime(time.Duration(ConnMaxIdleTimeDefault) * time.Minute)

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	if err := createIndexes(); err != nil {
		fmt.Printf("Warning: failed to create indexes: %v\n", err)
	}

	return nil
}

func createIndexes() error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_verification_logs_session_id ON verification_logs(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_verification_logs_application_id ON verification_logs(application_id)`,
		`CREATE INDEX IF NOT EXISTS idx_verification_logs_created_at ON verification_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_verification_logs_status ON verification_logs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_verification_logs_composite ON verification_logs(application_id, created_at, status)`,
		`CREATE INDEX IF NOT EXISTS idx_verifications_session_id ON verifications(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_verifications_application_id ON verifications(application_id)`,
		`CREATE INDEX IF NOT EXISTS idx_silent_verification_token ON silent_verifications(token)`,
		`CREATE INDEX IF NOT EXISTS idx_silent_verification_session_id ON silent_verifications(session_id)`,
	}

	for _, idx := range indexes {
		if _, err := DB.ExecContext(ctx, idx); err != nil {
			return err
		}
	}

	return nil
}

func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

func GetDB() *sql.DB {
	return DB
}

func GetDBStats() DBStats {
	stats := DB.Stats()
	return DBStats{
		OpenConnections: stats.OpenConnections,
		InUse:          stats.InUse,
		Idle:           stats.Idle,
		WaitCount:      stats.WaitCount,
		WaitDuration:   stats.WaitDuration,
		MaxIdleClosed:  stats.MaxIdleClosed,
		MaxLifetimeClosed: stats.MaxLifetimeClosed,
	}
}

func HealthCheck() error {
	if DB == nil {
		return fmt.Errorf("database connection is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return DB.PingContext(ctx)
}

func ExecWithRetry(query string, args ...interface{}) (sql.Result, error) {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		result, err := DB.ExecContext(ctx, query, args...)
		if err == nil {
			return result, nil
		}
		if i < maxRetries-1 {
			time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
		}
	}
	return nil, fmt.Errorf("exec failed after %d retries", maxRetries)
}

func QueryWithRetry(query string, args ...interface{}) (*sql.Rows, error) {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		rows, err := DB.QueryContext(ctx, query, args...)
		if err == nil {
			return rows, nil
		}
		if i < maxRetries-1 {
			time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
		}
	}
	return nil, fmt.Errorf("query failed after %d retries", maxRetries)
}
