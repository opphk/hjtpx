package database

import (
	"fmt"
	"log"

	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/models"
	"golang.org/x/crypto/bcrypt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

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
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return err
	}

	log.Println("Database connection established successfully")

	if err := AutoMigrate(); err != nil {
		return err
	}

	if err := CreateDefaultAdmin(); err != nil {
		log.Printf("Failed to create default admin: %v", err)
	}

	return nil
}

func AutoMigrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.Admin{},
		&models.Application{},
		&models.Verification{},
		&models.BehaviorData{},
		&models.VerificationLog{},
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