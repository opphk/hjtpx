// Package database provides database migration and connection management for CaptchaX
package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

var (
	// ErrMigrationsAlreadyApplied is returned when all migrations are already applied
	ErrMigrationsAlreadyApplied = errors.New("no migrations to apply")
	
	// ErrNoDirtyMigrations is returned when there are no dirty migrations
	ErrNoDirtyMigrations = errors.New("no dirty migrations")
)

// Config holds database configuration
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DefaultConfig returns the default database configuration
func DefaultConfig() *Config {
	return &Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "captcha_db"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

// ConnectionString builds the PostgreSQL connection string
func (c *Config) ConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.DBName,
		c.SSLMode,
	)
}

// Migrator handles database migrations
type Migrator struct {
	db      *sql.DB
	migrate *migrate.Migrate
	config  *Config
}

// NewMigrator creates a new database migrator
func NewMigrator(config *Config) (*Migrator, error) {
	if config == nil {
		config = DefaultConfig()
	}

	db, err := sql.Open("postgres", config.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create database driver: %w", err)
	}

	migrationsPath, err := getMigrationsPath()
	if err != nil {
		return nil, fmt.Errorf("failed to find migrations path: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}

	return &Migrator{
		db:      db,
		migrate: m,
		config:  config,
	}, nil
}

// Up applies all pending migrations
func (m *Migrator) Up() error {
	if err := m.migrate.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return ErrMigrationsAlreadyApplied
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	log.Println("Migrations applied successfully")
	return nil
}

// UpN applies N pending migrations
func (m *Migrator) UpN(n int) error {
	if err := m.migrate.Steps(n); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return ErrMigrationsAlreadyApplied
		}
		return fmt.Errorf("failed to apply %d migrations: %w", n, err)
	}
	log.Printf("Applied %d migrations successfully", n)
	return nil
}

// Down rolls back all migrations
func (m *Migrator) Down() error {
	if err := m.migrate.Down(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return ErrNoDirtyMigrations
		}
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}
	log.Println("All migrations rolled back successfully")
	return nil
}

// DownN rolls back N migrations
func (m *Migrator) DownN(n int) error {
	if err := m.migrate.Steps(-n); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return ErrNoDirtyMigrations
		}
		return fmt.Errorf("failed to rollback %d migrations: %w", n, err)
	}
	log.Printf("Rolled back %d migrations successfully", n)
	return nil
}

// Goto migrates to a specific version
func (m *Migrator) Goto(version uint) error {
	if err := m.migrate.Migrate(version); err != nil {
		return fmt.Errorf("failed to migrate to version %d: %w", version, err)
	}
	log.Printf("Migrated to version %d successfully", version)
	return nil
}

// Version returns the current migration version
func (m *Migrator) Version() (uint, bool, error) {
	version, dirty, err := m.migrate.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("failed to get version: %w", err)
	}
	return version, dirty, nil
}

// Force sets the version without running migrations
func (m *Migrator) Force(version int) error {
	if err := m.migrate.Force(version); err != nil {
		return fmt.Errorf("failed to force version %d: %w", version, err)
	}
	log.Printf("Forced version to %d successfully", version)
	return nil
}

// Drop drops everything in the database
func (m *Migrator) Drop() error {
	if err := m.migrate.Drop(); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}
	log.Println("Database dropped successfully")
	return nil
}

// Close closes the migrator and database connection
func (m *Migrator) Close() error {
	srcErr, dbErr := m.migrate.Close()
	if srcErr != nil {
		log.Printf("Warning: source close error: %v", srcErr)
	}
	if dbErr != nil {
		return fmt.Errorf("database close error: %w", dbErr)
	}
	return nil
}

// GetDB returns the underlying database connection
func (m *Migrator) GetDB() *sql.DB {
	return m.db
}

// getMigrationsPath finds the migrations directory
func getMigrationsPath() (string, error) {
	_, filename, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(filename)
	
	// Try to find migrations directory by traversing up
	searchPaths := []string{
		filepath.Join(currentDir, "..", "..", "migrations"),
		filepath.Join(currentDir, "..", "migrations"),
		filepath.Join(currentDir, "migrations"),
		"./migrations",
		"/workspace/captchax/migrations",
	}
	
	for _, path := range searchPaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return "", err
			}
			return absPath, nil
		}
	}
	
	return "", fmt.Errorf("migrations directory not found in search paths: %v", searchPaths)
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// MustNewMigrator creates a new migrator or panics on error
func MustNewMigrator(config *Config) *Migrator {
	m, err := NewMigrator(config)
	if err != nil {
		log.Fatalf("Failed to create migrator: %v", err)
	}
	return m
}
