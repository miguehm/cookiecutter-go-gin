package configs

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

var db *gorm.DB
var dbInitErr error
var onceInitDBClient sync.Once

func InitDBClient() error {
	onceInitDBClient.Do(func() {
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			Env.DBHost, Env.DBPort, Env.DBUser, Env.DBPass, Env.DBName, Env.DBSslMode)

		var err error
		logLevel := glogger.LogLevel(Env.DBLogLevel)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger:                 glogger.Default.LogMode(logLevel),
			TranslateError:         true,
			SkipDefaultTransaction: true,
		})

		if err != nil {
			dbInitErr = fmt.Errorf("failed to connect database: %w", err)
			return
		}

		sqlDB, err := db.DB()
		if err != nil {
			dbInitErr = fmt.Errorf("failed to get database: %w", err)
			db = nil
			return
		}

		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(500)
		sqlDB.SetConnMaxLifetime(time.Hour)
	})
	return dbInitErr
}

func GetDB() (*gorm.DB, error) {
	if dbInitErr != nil {
		return nil, dbInitErr
	}
	if db == nil {
		return nil, fmt.Errorf("database client not initialized")
	}
	return db, nil
}

func MustGetDB() *gorm.DB {
	d, err := GetDB()
	if err != nil {
		panic(err)
	}
	return d
}
