package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/models"
	"golang.org/x/crypto/bcrypt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

type DBPoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	WaitForPool     bool
	PoolStats       func() interface{}
}

var poolConfig = &DBPoolConfig{
	MaxOpenConns:    100,
	MaxIdleConns:    20,
	ConnMaxLifetime: 30 * time.Minute,
	ConnMaxIdleTime: 10 * time.Minute,
}

func InitDB(cfg *config.Config) error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.DBName,
		cfg.Postgres.SSLMode,
	)

	var err error
	DB, err = gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Printf("Warning: Failed to initialize database connection: %v", err)
		log.Println("Server will start without database - some features may not work")
		DB = nil
		return nil // Don't fail startup
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Printf("Warning: Failed to get underlying sql.DB: %v", err)
		DB = nil
		return nil
	}

	maxOpenConns := cfg.Postgres.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = poolConfig.MaxOpenConns
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)

	maxIdleConns := cfg.Postgres.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = poolConfig.MaxIdleConns
	}
	sqlDB.SetMaxIdleConns(maxIdleConns)

	connMaxLifetime := time.Duration(cfg.Postgres.ConnMaxLifetime) * time.Minute
	if connMaxLifetime <= 0 {
		connMaxLifetime = poolConfig.ConnMaxLifetime
	}
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	sqlDB.SetConnMaxIdleTime(poolConfig.ConnMaxIdleTime)

	if err := sqlDB.Ping(); err != nil {
		log.Printf("Warning: Failed to ping database: %v", err)
		log.Println("Server will start without database - some features may not work")
		DB = nil
		return nil // Don't fail startup
	}

	log.Println("Database connection established successfully")

	if err := AutoMigrate(); err != nil {
		log.Printf("Warning: Failed to run auto migration: %v", err)
	}

	if err := CreateDefaultAdmin(); err != nil {
		log.Printf("Failed to create default admin: %v", err)
	}

	if err := InitializeDatabaseFeatures(cfg); err != nil {
		log.Printf("Warning: Failed to initialize database features: %v", err)
	}

	return nil
}

func InitializeDatabaseFeatures(cfg *config.Config) error {
	InitPerformanceMonitor(cfg)
	InitConnectionPool(cfg)
	InitOptimizedQueryCache(cfg)
	InitOptimizedQueryAnalyzer(DB, cfg.Database.SlowQueryThresholdMs)
	InitReadWriteSeparation(cfg)

	GormQueryCallback(DB)

	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			InitOptimizedConnectionPool(sqlDB, cfg)
		}

		InitPerformanceOptimizer(DB, cfg)
		if optimizer := GetPerformanceOptimizer(); optimizer != nil {
			if err := optimizer.OptimizeAll(); err != nil {
				log.Printf("Warning: Database optimization failed: %v", err)
			} else {
				log.Println("Database optimization completed successfully")
			}
		}

		indexAnalyzer := NewIndexAnalyzer(DB)
		if err := indexAnalyzer.AnalyzeAndCreateMissingIndexes(); err != nil {
			log.Printf("Warning: Index analysis failed: %v", err)
		}

		queryOptimizer := NewAdvancedQueryOptimizer(DB, 50*time.Millisecond)
		if err := queryOptimizer.OptimizeAll(); err != nil {
			log.Printf("Warning: Query optimization failed: %v", err)
		} else {
			log.Println("Query optimization completed successfully")
		}
	}
	return nil
}

func AutoMigrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.Admin{},
		&models.AdminLoginLog{},
		&models.Application{},
		&models.APIKeyHistory{},
		&models.Verification{},
		&models.BehaviorData{},
		&models.Blacklist{},
		&models.VerificationLog{},
		&models.DeviceFingerprint{},
		&models.AlertChannel{},
		&models.AlertRule{},
		&models.AlertRecord{},
		&models.AlertHistory{},
		&models.TraceRecord{},
		&models.Config{},
	)
}

func CreateDefaultAdmin() error {
	var count int64
	DB.Model(&models.Admin{}).Count(&count)
	if count > 0 {
		log.Println("Admin account already exists, skipping default admin creation")
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	admin := &models.Admin{
		Username:     "admin",
		PasswordHash: string(hashedPassword),
		IsSuperAdmin: true,
	}

	if err := DB.Create(admin).Error; err != nil {
		return fmt.Errorf("failed to create default admin: %w", err)
	}

	log.Println("Default admin account created successfully (username: admin, password: admin123)")
	return nil
}

func GetDB() *gorm.DB {
	return DB
}

type PoolStats struct {
	MaxOpenConnections int           `json:"max_open_connections"`
	OpenConnections    int           `json:"open_connections"`
	InUse              int           `json:"in_use"`
	Idle               int           `json:"idle"`
	WaitCount          int64         `json:"wait_count"`
	WaitDuration       time.Duration `json:"wait_duration"`
	MaxIdleClosed      int64         `json:"max_idle_closed"`
	MaxLifetimeClosed  int64         `json:"max_lifetime_closed"`
}

func GetPoolStats() (*PoolStats, error) {
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	stats := sqlDB.Stats()
	return &PoolStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}, nil
}

func Close() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func Ping(ctx context.Context) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func SetPoolMaxOpenConns(n int) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(n)
	return nil
}

func SetPoolMaxIdleConns(n int) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(n)
	return nil
}

func SetPoolConnMaxLifetime(d time.Duration) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	sqlDB.SetConnMaxLifetime(d)
	return nil
}

func ConfigurePool(cfg *DBPoolConfig) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	return nil
}
