package database

import (
	"context"
	"fmt"
	"time"

	"hjtpx/internal/config"
	"hjtpx/internal/models"
	"hjtpx/internal/utils"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

type PostgresConfig struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

func InitPostgres(cfg config.DatabaseConfig, poolCfg PostgresConfig) (*gorm.DB, error) {
	dsn := cfg.DSN()

	gormLogger := logger.Default.LogMode(logger.Warn)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if poolCfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(poolCfg.MaxIdleConns)
	} else {
		sqlDB.SetMaxIdleConns(10)
	}

	if poolCfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(poolCfg.MaxOpenConns)
	} else {
		sqlDB.SetMaxOpenConns(100)
	}

	if poolCfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(poolCfg.ConnMaxLifetime)
	} else {
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	if poolCfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(poolCfg.ConnMaxIdleTime)
	} else {
		sqlDB.SetConnMaxIdleTime(30 * time.Minute)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	DB = db
	utils.Info("PostgreSQL connection established")
	utils.Info("Connection pool: max_idle=%d, max_open=%d, max_lifetime=%v",
		poolCfg.MaxIdleConns, poolCfg.MaxOpenConns, poolCfg.ConnMaxLifetime)

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	utils.Info("Starting database auto migration...")

	err := db.AutoMigrate(
		&models.User{},
		&models.App{},
		&models.Captcha{},
		&models.VerificationLog{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	if err := createIndexes(db); err != nil {
		utils.Warn("Failed to create custom indexes: %v", err)
	}

	utils.Info("Database migration completed")
	return nil
}

func createIndexes(db *gorm.DB) error {
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_captcha_token ON captchas(token)").Error; err != nil {
		utils.Warn("Failed to create idx_captcha_token: %v", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_captcha_expires_at ON captchas(expires_at)").Error; err != nil {
		utils.Warn("Failed to create idx_captcha_expires_at: %v", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_verification_logs_user_id ON verification_logs(user_id)").Error; err != nil {
		utils.Warn("Failed to create idx_verification_logs_user_id: %v", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_verification_logs_created_at ON verification_logs(created_at)").Error; err != nil {
		utils.Warn("Failed to create idx_verification_logs_created_at: %v", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)").Error; err != nil {
		utils.Warn("Failed to create idx_users_email: %v", err)
	}

	return nil
}

func ClosePostgres() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close PostgreSQL connection: %w", err)
	}

	utils.Info("PostgreSQL connection closed")
	return nil
}

func GetDB() *gorm.DB {
	return DB
}
