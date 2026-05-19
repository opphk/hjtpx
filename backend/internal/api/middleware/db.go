package middleware

import (
	"sync"

	"github.com/hjtpx/hjtpx/pkg/database"
	"gorm.io/gorm"
)

var (
	dbOnce sync.Once
	dbInstance *gorm.DB
)

func GetDB() *gorm.DB {
	dbOnce.Do(func() {
		dbInstance = database.GetDB()
	})
	return dbInstance
}

func SetDB(db *gorm.DB) {
	dbOnce.Do(func() {
		dbInstance = db
	})
}
