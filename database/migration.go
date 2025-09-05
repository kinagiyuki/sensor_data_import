package database

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"sensor_data_import/config"
	"sensor_data_import/logger"

	"gorm.io/gorm"
)

// Migration represents a database migration
type Migration struct {
	ID          uint   `gorm:"primaryKey"`
	Version     string `gorm:"unique;not null"`
	Name        string `gorm:"not null"`
	Applied     bool   `gorm:"default:false"`
	AppliedAt   *time.Time
	Description string
}

// MigrationFile represents a migration file
type MigrationFile struct {
	Version     string
	Name        string
	Description string
	FilePath    string
	Applied     bool
}

// MigrationRunner handles database migrations
type MigrationRunner struct {
	db             *gorm.DB
	migrationTable string
	migrationDir   string
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *gorm.DB, cfg *config.Config) *MigrationRunner {
	return &MigrationRunner{
		db:             db,
		migrationTable: cfg.Migration.MigrationTable,
		migrationDir:   "migrations",
	}
}

// InitializeMigrationTable creates the migration table if it doesn't exist
func (mr *MigrationRunner) InitializeMigrationTable() error {
	return mr.db.AutoMigrate(&Migration{})
}

// GetMigrationFiles returns all migration files from the migrations directory
func (mr *MigrationRunner) GetMigrationFiles() ([]MigrationFile, error) {
	var migrationFiles []MigrationFile

	// Check if migration directory exists
	if _, err := os.Stat(mr.migrationDir); os.IsNotExist(err) {
		return migrationFiles, nil // Return empty slice if directory doesn't exist
	}

	// Walk through migration directory
	err := filepath.WalkDir(mr.migrationDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-SQL files
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".sql") {
			return nil
		}

		// Parse migration filename (format: YYYYMMDD_HHMMSS_description.sql)
		filename := d.Name()
		parts := strings.SplitN(filename, "_", 3)
		if len(parts) < 3 {
			return fmt.Errorf("invalid migration filename format: %s (expected: YYYYMMDD_HHMMSS_description.sql)", filename)
		}

		version := parts[0] + "_" + parts[1]
		description := strings.TrimSuffix(parts[2], ".sql")
		name := strings.ReplaceAll(description, "_", " ")

		migrationFiles = append(migrationFiles, MigrationFile{
			Version:     version,
			Name:        name,
			Description: description,
			FilePath:    path,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %w", err)
	}

	// Sort migration files by version
	sort.Slice(migrationFiles, func(i, j int) bool {
		return migrationFiles[i].Version < migrationFiles[j].Version
	})

	return migrationFiles, nil
}

// GetAppliedMigrations returns all applied migrations from the database
func (mr *MigrationRunner) GetAppliedMigrations() ([]Migration, error) {
	var migrations []Migration

	// Initialize migration table if it doesn't exist
	if err := mr.InitializeMigrationTable(); err != nil {
		return nil, fmt.Errorf("failed to initialize migration table: %w", err)
	}

	result := mr.db.Where("applied = ?", true).Order("version ASC").Find(&migrations)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", result.Error)
	}

	return migrations, nil
}

// GetPendingMigrations returns migrations that haven't been applied yet
func (mr *MigrationRunner) GetPendingMigrations() ([]MigrationFile, error) {
	// Get all migration files
	allMigrations, err := mr.GetMigrationFiles()
	if err != nil {
		return nil, err
	}

	// Get applied migrations
	appliedMigrations, err := mr.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	// Create a map of applied migration versions
	appliedVersions := make(map[string]bool)
	for _, migration := range appliedMigrations {
		appliedVersions[migration.Version] = true
	}

	// Filter out applied migrations
	var pendingMigrations []MigrationFile
	for _, migration := range allMigrations {
		if !appliedVersions[migration.Version] {
			pendingMigrations = append(pendingMigrations, migration)
		}
	}

	return pendingMigrations, nil
}

// RunMigrations executes all pending migrations
func (mr *MigrationRunner) RunMigrations() error {
	pendingMigrations, err := mr.GetPendingMigrations()
	if err != nil {
		return fmt.Errorf("failed to get pending migrations: %w", err)
	}

	if len(pendingMigrations) == 0 {
		logger.Println("No pending migrations to run")
		return nil
	}

	logger.Printf("Running %d pending migration(s)...\n", len(pendingMigrations))

	for _, migration := range pendingMigrations {
		if err := mr.runSingleMigration(migration); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", migration.Version, err)
		}
	}

	logger.Println("All migrations completed successfully")
	return nil
}

// runSingleMigration executes a single migration
func (mr *MigrationRunner) runSingleMigration(migrationFile MigrationFile) error {
	logger.Printf("Running migration: %s - %s\n", migrationFile.Version, migrationFile.Name)

	// Read migration file content
	content, err := os.ReadFile(migrationFile.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Execute migration in a transaction
	return mr.db.Transaction(func(tx *gorm.DB) error {
		// Execute the SQL
		if err := tx.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("failed to execute migration SQL: %w", err)
		}

		// Record migration as applied
		now := time.Now()
		migration := Migration{
			Version:     migrationFile.Version,
			Name:        migrationFile.Name,
			Applied:     true,
			AppliedAt:   &now,
			Description: migrationFile.Description,
		}

		if err := tx.Create(&migration).Error; err != nil {
			return fmt.Errorf("failed to record migration: %w", err)
		}

		return nil
	})
}

// GetMigrationStatus returns the status of all migrations
func (mr *MigrationRunner) GetMigrationStatus() ([]MigrationFile, error) {
	// Get all migration files
	allMigrations, err := mr.GetMigrationFiles()
	if err != nil {
		return nil, err
	}

	// Get applied migrations
	appliedMigrations, err := mr.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	// Create a map of applied migration versions
	appliedVersions := make(map[string]bool)
	for _, migration := range appliedMigrations {
		appliedVersions[migration.Version] = true
	}

	// Mark applied status
	for i := range allMigrations {
		allMigrations[i].Applied = appliedVersions[allMigrations[i].Version]
	}

	return allMigrations, nil
}

// CreateMigration creates a new migration file with the given name
func (mr *MigrationRunner) CreateMigration(name string) (string, error) {
	// Ensure migrations directory exists
	if err := os.MkdirAll(mr.migrationDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Generate timestamp
	now := time.Now()
	version := now.Format("20060102_150405")

	// Clean up migration name
	cleanName := strings.ReplaceAll(strings.ToLower(name), " ", "_")
	filename := fmt.Sprintf("%s_%s.sql", version, cleanName)
	filePath := filepath.Join(mr.migrationDir, filename)

	// Create migration file with template
	template := fmt.Sprintf(`-- Migration: %s
-- Created: %s
-- Description: %s

-- Add your migration SQL here
-- Example:
-- CREATE TABLE example (
--     id INT AUTO_INCREMENT PRIMARY KEY,
--     name VARCHAR(255) NOT NULL,
--     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
-- );
`, name, now.Format("2006-01-02 15:04:05"), name)

	if err := os.WriteFile(filePath, []byte(template), 0644); err != nil {
		return "", fmt.Errorf("failed to create migration file: %w", err)
	}

	return filePath, nil
}
