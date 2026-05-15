// Package main provides a command-line interface for database migrations
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"captchax/internal/database"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	migrator, err := database.NewMigrator(nil)
	if err != nil {
		log.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	switch command {
	case "up":
		if len(os.Args) > 2 {
			n, err := strconv.Atoi(os.Args[2])
			if err != nil {
				log.Fatalf("Invalid number: %v", err)
			}
			if err := migrator.UpN(n); err != nil {
				if err == database.ErrMigrationsAlreadyApplied {
					fmt.Println("No migrations to apply")
				} else {
					log.Fatalf("Failed to apply migrations: %v", err)
				}
			}
		} else {
			if err := migrator.Up(); err != nil {
				if err == database.ErrMigrationsAlreadyApplied {
					fmt.Println("No migrations to apply")
				} else {
					log.Fatalf("Failed to apply migrations: %v", err)
				}
			}
		}

	case "down":
		if len(os.Args) > 2 {
			n, err := strconv.Atoi(os.Args[2])
			if err != nil {
				log.Fatalf("Invalid number: %v", err)
			}
			if err := migrator.DownN(n); err != nil {
				if err == database.ErrNoDirtyMigrations {
					fmt.Println("No migrations to rollback")
				} else {
					log.Fatalf("Failed to rollback migrations: %v", err)
				}
			}
		} else {
			if err := migrator.Down(); err != nil {
				if err == database.ErrNoDirtyMigrations {
					fmt.Println("No migrations to rollback")
				} else {
					log.Fatalf("Failed to rollback migrations: %v", err)
				}
			}
		}

	case "goto":
		if len(os.Args) < 3 {
			log.Fatal("Version number required for goto command")
		}
		version, err := strconv.ParseUint(os.Args[2], 10, 32)
		if err != nil {
			log.Fatalf("Invalid version: %v", err)
		}
		if err := migrator.Goto(uint(version)); err != nil {
			log.Fatalf("Failed to migrate to version %d: %v", version, err)
		}

	case "version":
		version, dirty, err := migrator.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		fmt.Printf("Current version: %d\n", version)
		if dirty {
			fmt.Println("Database is dirty!")
		}

	case "force":
		if len(os.Args) < 3 {
			log.Fatal("Version number required for force command")
		}
		version, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("Invalid version: %v", err)
		}
		if err := migrator.Force(version); err != nil {
			log.Fatalf("Failed to force version: %v", err)
		}

	case "drop":
		fmt.Print("WARNING: This will drop ALL tables and data! Are you sure? (yes/no): ")
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) == "yes" {
			if err := migrator.Drop(); err != nil {
				log.Fatalf("Failed to drop database: %v", err)
			}
		} else {
			fmt.Println("Operation cancelled")
		}

	case "status":
		version, dirty, err := migrator.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		fmt.Println("Migration Status:")
		fmt.Printf("  Current Version: %d\n", version)
		fmt.Printf("  Dirty: %v\n", dirty)

	case "help", "--help", "-h":
		printUsage()

	default:
		log.Fatalf("Unknown command: %s", command)
	}
}

func printUsage() {
	fmt.Println("CaptchaX Database Migration Tool")
	fmt.Println()
	fmt.Println("Usage: migrate [command] [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  up [N]          Apply all or N up migrations")
	fmt.Println("  down [N]        Rollback all or N down migrations")
	fmt.Println("  goto V          Migrate to version V")
	fmt.Println("  version         Print current migration version")
	fmt.Println("  force V         Set version V without running migrations")
	fmt.Println("  drop            Drop everything in the database")
	fmt.Println("  status          Show migration status")
	fmt.Println("  help            Show this help message")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  DB_HOST         Database host (default: localhost)")
	fmt.Println("  DB_PORT         Database port (default: 5432)")
	fmt.Println("  DB_NAME         Database name (default: captcha_db)")
	fmt.Println("  DB_USER         Database user (default: postgres)")
	fmt.Println("  DB_PASSWORD     Database password (default: postgres)")
	fmt.Println("  DB_SSLMODE      SSL mode (default: disable)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  migrate up                    # Apply all migrations")
	fmt.Println("  migrate up 1                  # Apply 1 migration")
	fmt.Println("  migrate down                  # Rollback all migrations")
	fmt.Println("  migrate down 1                # Rollback 1 migration")
	fmt.Println("  migrate goto 3                # Migrate to version 3")
	fmt.Println("  migrate version               # Show current version")
}
