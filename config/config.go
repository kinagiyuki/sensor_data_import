package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DatabaseConfig holds all database configuration
type DatabaseConfig struct {
	Driver         string         `yaml:"driver"`
	MySQL          MySQLConfig    `yaml:"mysql"`
	PostgreSQL     PostgresConfig `yaml:"postgres"`
	SQLite         SQLiteConfig   `yaml:"sqlite"`
	ConnectionPool PoolConfig     `yaml:"connection_pool"`
}

// MySQLConfig holds MySQL specific configuration
type MySQLConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	User      string `yaml:"user"`
	Password  string `yaml:"password"`
	DBName    string `yaml:"dbname"`
	Charset   string `yaml:"charset"`
	ParseTime bool   `yaml:"parse_time"`
	Loc       string `yaml:"loc"`
}

// PostgresConfig holds PostgreSQL specific configuration
type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
	TimeZone string `yaml:"timezone"`
}

// SQLiteConfig holds SQLite specific configuration
type SQLiteConfig struct {
	Path string `yaml:"path"`
}

// PoolConfig holds connection pool configuration
type PoolConfig struct {
	MaxIdleConns    int `yaml:"max_idle_conns"`
	MaxOpenConns    int `yaml:"max_open_conns"`
	ConnMaxLifetime int `yaml:"conn_max_lifetime"`
}

// MigrationConfig holds migration specific configuration
type MigrationConfig struct {
	AutoMigrate    bool   `yaml:"auto_migrate"`
	MigrationTable string `yaml:"migration_table"`
}

// LoggingConfig holds logging specific configuration
type LoggingConfig struct {
	LogFile      string `yaml:"log_file"`
	LogToConsole bool   `yaml:"log_to_console"`
	LogLevel     string `yaml:"log_level"`
}

// Config holds the complete application configuration
type Config struct {
	Database  DatabaseConfig  `yaml:"database"`
	Migration MigrationConfig `yaml:"migration"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// Load loads configuration from the specified YAML file
func Load(configPath string) (*Config, error) {
	// Set default config path if not provided
	if configPath == "" {
		configPath = "config.yaml"
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set default values for logging if not specified
	if config.Logging.LogFile == "" {
		config.Logging.LogFile = "result.log"
	}
	if config.Logging.LogLevel == "" {
		config.Logging.LogLevel = "info"
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	switch c.Database.Driver {
	case "mysql":
		if c.Database.MySQL.Host == "" {
			return fmt.Errorf("mysql host is required")
		}
		if c.Database.MySQL.User == "" {
			return fmt.Errorf("mysql user is required")
		}
		if c.Database.MySQL.DBName == "" {
			return fmt.Errorf("mysql database name is required")
		}
	case "postgres":
		if c.Database.PostgreSQL.Host == "" {
			return fmt.Errorf("postgres host is required")
		}
		if c.Database.PostgreSQL.User == "" {
			return fmt.Errorf("postgres user is required")
		}
		if c.Database.PostgreSQL.DBName == "" {
			return fmt.Errorf("postgres database name is required")
		}
	case "sqlite":
		if c.Database.SQLite.Path == "" {
			return fmt.Errorf("sqlite path is required")
		}
	default:
		return fmt.Errorf("unsupported database driver: %s", c.Database.Driver)
	}

	return nil
}

// GetDSN returns the database connection string based on the configured driver
func (c *Config) GetDSN() string {
	switch c.Database.Driver {
	case "mysql":
		mysql := c.Database.MySQL
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
			mysql.User, mysql.Password, mysql.Host, mysql.Port, mysql.DBName,
			mysql.Charset, mysql.ParseTime, mysql.Loc)
		return dsn
	case "postgres":
		pg := c.Database.PostgreSQL
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
			pg.Host, pg.Port, pg.User, pg.Password, pg.DBName, pg.SSLMode, pg.TimeZone)
		return dsn
	case "sqlite":
		return c.Database.SQLite.Path
	default:
		return ""
	}
}
