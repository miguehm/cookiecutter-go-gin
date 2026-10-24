package configs

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// NewDB opens a Postgres connection. The caller owns the returned handle and
// must close it with CloseDB.
func NewDB(cfg *Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName, cfg.DBSslMode)

	logLevel := glogger.LogLevel(cfg.DBLogLevel)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 glogger.Default.LogMode(logLevel),
		TranslateError:         true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(500)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// CloseDB closes the underlying connection pool.
func CloseDB(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	return nil
}

// ProvideDB is the fx aware variant of NewDB: the connection is closed on
// application shutdown.
func ProvideDB(lc fx.Lifecycle, cfg *Config) (*gorm.DB, error) {
	db, err := NewDB(cfg)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return CloseDB(db)
		},
	})
	return db, nil
}
