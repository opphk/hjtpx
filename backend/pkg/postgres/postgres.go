package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	_ "github.com/lib/pq"
)

var DB *sql.DB

// Connect 建立PostgreSQL连接
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

	// 配置连接池
	DB.SetMaxOpenConns(cfg.MaxOpenConns)
	DB.SetMaxIdleConns(cfg.MaxIdleConns)
	DB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Minute)

	// 测试连接
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// Close 关闭PostgreSQL连接
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// GetDB 获取数据库连接
func GetDB() *sql.DB {
	return DB
}
