package database

import (
	"fmt"
	"time"

	"sensor_data_import/config"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database instance
var DB *gorm.DB

// Connect establishes a database connection based on the provided configuration
func Connect(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	// Select the appropriate driver based on configuration
	switch cfg.Database.Driver {
	case "mysql":
		dsn := cfg.GetDSN()
		dialector = mysql.Open(dsn)
	case "postgres":
		dsn := cfg.GetDSN()
		dialector = postgres.Open(dsn)
	case "sqlite":
		dsn := cfg.GetDSN()
		dialector = sqlite.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Database.Driver)
	}

	// Configure GORM with logger
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	// Connect to database
	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	pool := cfg.Database.ConnectionPool
	sqlDB.SetMaxIdleConns(pool.MaxIdleConns)
	sqlDB.SetMaxOpenConns(pool.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(pool.ConnMaxLifetime) * time.Second)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set global DB instance
	DB = db

	return db, nil
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return fmt.Errorf("failed to get underlying sql.DB: %w", err)
		}
		return sqlDB.Close()
	}
	return nil
}

// GetDB returns the global database instance
func GetDB() *gorm.DB {
	return DB
}

// IsConnected checks if database is connected
func IsConnected() bool {
	if DB == nil {
		return false
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return false
	}

	if err := sqlDB.Ping(); err != nil {
		return false
	}

	return true
}

// GetDatabaseInfo returns information about the connected database
func GetDatabaseInfo(cfg *config.Config) map[string]interface{} {
	info := make(map[string]interface{})
	info["driver"] = cfg.Database.Driver
	info["connected"] = IsConnected()

	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			stats := sqlDB.Stats()
			info["max_open_connections"] = stats.MaxOpenConnections
			info["open_connections"] = stats.OpenConnections
			info["in_use"] = stats.InUse
			info["idle"] = stats.Idle
		}
	}

	switch cfg.Database.Driver {
	case "mysql":
		info["host"] = cfg.Database.MySQL.Host
		info["port"] = cfg.Database.MySQL.Port
		info["database"] = cfg.Database.MySQL.DBName
	case "postgres":
		info["host"] = cfg.Database.PostgreSQL.Host
		info["port"] = cfg.Database.PostgreSQL.Port
		info["database"] = cfg.Database.PostgreSQL.DBName
	case "sqlite":
		info["path"] = cfg.Database.SQLite.Path
	}

	return info
}
