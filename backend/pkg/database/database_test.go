package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDBConnection(t *testing.T) {
	db := GetDB()
	assert.NotNil(t, db)
}

func TestDBOperations(t *testing.T) {
	db := GetDB()
	assert.NotNil(t, db)

	sqlDB, err := db.DB()
	if err != nil {
		t.Skip("Database not available, skipping database connection test")
	}

	err = sqlDB.Ping()
	if err != nil {
		t.Skip("Database not available for ping")
	}
}

func TestDBConfiguration(t *testing.T) {
	db := GetDB()
	assert.NotNil(t, db)

	sqlDB, err := db.DB()
	if err != nil {
		t.Skip("Database connection not available")
	}

	maxOpenConns := sqlDB.MaxOpenConns()
	maxIdleConns := sqlDB.MaxIdleConns()

	assert.GreaterOrEqual(t, maxOpenConns, 0)
	assert.GreaterOrEqual(t, maxIdleConns, 0)
}
