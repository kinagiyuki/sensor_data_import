package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"sensor_data_import/config"
	"sensor_data_import/database"
	"sensor_data_import/logger"
	"sensor_data_import/models"
	"sensor_data_import/scanner"
)

func main() {
	if len(os.Args) < 2 {
		showHelp()
		return
	}

	command := os.Args[1]

	// Initialize logging only for commands that need it
	if needsLogging(command) {
		cfg := loadConfig()
		if err := logger.Init(cfg); err != nil {
			log.Fatalf("Failed to initialize logging: %v", err)
		}
		defer func() {
			err := logger.Close()
			if err != nil {
				log.Fatalf("Failed to close logging: %v", err)
			}
		}()
		logger.LogCommand(os.Args[0], os.Args)
	}

	switch command {
	case "connect":
		connectCommand()
	case "migrate":
		migrateCommand()
	case "migrate:create":
		if len(os.Args) < 3 {
			fmt.Println("Error: migration name required")
			fmt.Println("Usage: go run main.go migrate:create <migration_name>")
			return
		}
		createMigrationCommand(os.Args[2])
	case "migrate:status":
		migrationStatusCommand()
	case "db:info":
		dbInfoCommand()
	case "scan":
		if len(os.Args) < 3 {
			fmt.Println("Error: directory path required")
			fmt.Println("Usage: go run main.go scan <directory_path>")
			return
		}
		scanCommand(os.Args[2])
	case "test:insert":
		testInsertCommand()
	case "help":
		showHelp()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		showHelp()
	}
}

// needsLogging determines which commands need logging
func needsLogging(command string) bool {
	loggingCommands := map[string]bool{
		"migrate":        true,
		"migrate:create": true,
		"migrate:status": true,
		"scan":           true,
		"connect":        true,
		"test:insert":    true,
	}
	return loggingCommands[command]
}

func showHelp() {
	fmt.Println("Sensor Data import - Database Management Tool")
	fmt.Println("")
	fmt.Println("Usage: go run main.go <command> [arguments]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  connect              Test database connection")
	fmt.Println("  migrate              Run pending migrations")
	fmt.Println("  migrate:create <name> Create a new migration file")
	fmt.Println("  migrate:status       Show migration status")
	fmt.Println("  db:info              Show database information")
	fmt.Println("  scan <directory>     Scan directory for CSV files and import sensor data (non-recursive)")
	fmt.Println("  test:insert          Insert sample sensor data")
	fmt.Println("  help                 Show this help message")
	fmt.Println("")
	fmt.Println("Configuration:")
	fmt.Println("  Edit config.yaml to configure database settings")
	fmt.Println("")
	fmt.Println("CSV File Format:")
	fmt.Println("  Expected columns: timestamp,sensor_name,value")
	fmt.Println("  Timestamp format: ISO8601 (e.g., 2025-09-05T12:30:45Z)")
}

func loadConfig() *config.Config {
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	return cfg
}

func connectDatabase() (*config.Config, error) {
	cfg := loadConfig()

	_, err := database.Connect(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return cfg, nil
}

func connectCommand() {
	logger.Println("Testing database connection...")

	cfg, err := connectDatabase()
	if err != nil {
		logger.Fatalf("Connection failed: %v", err)
	}

	logger.Printf("✓ Successfully connected to %s database\n", cfg.Database.Driver)

	// Show connection info
	info := database.GetDatabaseInfo(cfg)
	infoJSON, _ := json.MarshalIndent(info, "", "  ")
	logger.Printf("Connection info: %s\n", infoJSON)
}

func migrateCommand() {
	logger.Println("Running database migrations...")

	cfg, err := connectDatabase()
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}

	runner := database.NewMigrationRunner(database.GetDB(), cfg)

	if err := runner.RunMigrations(); err != nil {
		logger.Fatalf("Migration failed: %v", err)
	}
}

func createMigrationCommand(name string) {
	logger.Printf("Creating migration: %s\n", name)

	cfg := loadConfig()
	runner := database.NewMigrationRunner(nil, cfg) // Don't need DB connection to create files

	filePath, err := runner.CreateMigration(name)
	if err != nil {
		logger.Fatalf("Failed to create migration: %v", err)
	}

	logger.Printf("✓ Migration created: %s\n", filePath)
}

func migrationStatusCommand() {
	logger.Println("Checking migration status...")

	cfg, err := connectDatabase()
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}

	runner := database.NewMigrationRunner(database.GetDB(), cfg)

	migrations, err := runner.GetMigrationStatus()
	if err != nil {
		logger.Fatalf("Failed to get migration status: %v", err)
	}

	if len(migrations) == 0 {
		logger.Println("No migrations found")
		return
	}

	logger.Printf("%-20s %-40s %s\n", "Version", "Name", "Status")
	logger.Println("-------------------------------------------------------------------")

	for _, migration := range migrations {
		status := "Pending"
		if migration.Applied {
			status = "Applied"
		}
		logger.Printf("%-20s %-40s %s\n", migration.Version, migration.Name, status)
	}
}

func dbInfoCommand() {
	fmt.Println("Database Information:")
	fmt.Println(strings.Repeat("=", 50))

	cfg, err := connectDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	info := database.GetDatabaseInfo(cfg)

	// Display basic database info
	fmt.Printf("Database Type:     %v\n", info["driver"])
	fmt.Printf("Connection Status: %v\n", getConnectionStatusText(info["connected"]))

	// Display database-specific connection details
	switch cfg.Database.Driver {
	case "mysql":
		fmt.Printf("Host:              %v\n", info["host"])
		fmt.Printf("Port:              %v\n", info["port"])
		fmt.Printf("Database:          %v\n", info["database"])
	case "postgres":
		fmt.Printf("Host:              %v\n", info["host"])
		fmt.Printf("Port:              %v\n", info["port"])
		fmt.Printf("Database:          %v\n", info["database"])
	case "sqlite":
		fmt.Printf("File Path:         %v\n", info["path"])
	}

	// Display connection pool information if available
	if info["connected"] == true {
		fmt.Println("\nConnection Pool:")
		fmt.Printf("  Max Connections: %v\n", info["max_open_connections"])
		fmt.Printf("  Open Connections:%v\n", info["open_connections"])
		fmt.Printf("  In Use:          %v\n", info["in_use"])
		fmt.Printf("  Idle:            %v\n", info["idle"])

		// Get table information
		db := database.GetDB()
		var count int64
		db.Model(&models.SensorData{}).Count(&count)
		fmt.Println("\nData Information:")
		fmt.Printf("  Total Records:   %d\n", count)

		// Get sensor count
		var sensorCount int64
		db.Model(&models.SensorData{}).Distinct("sensor_name").Count(&sensorCount)
		fmt.Printf("  Unique Sensors:  %d\n", sensorCount)

		// Get date range if data exists
		if count > 0 {
			var earliest, latest time.Time
			db.Model(&models.SensorData{}).Select("MIN(timestamp)").Scan(&earliest)
			db.Model(&models.SensorData{}).Select("MAX(timestamp)").Scan(&latest)
			fmt.Printf("  Date Range:      %s to %s\n",
				earliest.Format("2006-01-02 15:04:05"),
				latest.Format("2006-01-02 15:04:05"))
		}
	} else {
		fmt.Println("\nConnection failed - unable to retrieve detailed information")
	}

	fmt.Println(strings.Repeat("=", 50))
}

func getConnectionStatusText(connected interface{}) string {
	if conn, ok := connected.(bool); ok && conn {
		return "✓ Connected"
	}
	return "✗ Disconnected"
}

func scanCommand(directoryPath string) {
	logger.Printf("Scanning directory: %s\n", directoryPath)

	_, err := connectDatabase()
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}

	db := database.GetDB()
	csvScanner := scanner.NewCSVScanner(db)

	if err := csvScanner.ScanDirectory(directoryPath); err != nil {
		logger.Fatalf("Scan failed: %v", err)
	}

	logger.Println("✓ Directory scan completed successfully")
}

func testInsertCommand() {
	logger.Println("Inserting sample sensor data...")

	_, err := connectDatabase()
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}

	db := database.GetDB()

	// Insert sample data
	sampleData := []models.SensorData{
		{
			Timestamp:  time.Now().UTC(),
			SensorName: "temperature_sensor_01",
			Value:      23.5,
		},
		{
			Timestamp:  time.Now().UTC().Add(1 * time.Minute),
			SensorName: "humidity_sensor_01",
			Value:      65.2,
		},
		{
			Timestamp:  time.Now().UTC().Add(2 * time.Minute),
			SensorName: "pressure_sensor_01",
			Value:      1013.25,
		},
	}

	for _, data := range sampleData {
		result := db.Create(&data)
		if result.Error != nil {
			logger.Errorf("Failed to insert data for %s: %v", data.SensorName, result.Error)
		} else {
			logger.Printf("✓ Inserted data: %s = %.2f at %s\n",
				data.SensorName, data.Value, data.Timestamp.Format(time.RFC3339))
		}
	}

	// Query and display all data
	logger.Println("\nAll sensor data:")
	var allData []models.SensorData
	db.Order("timestamp ASC").Find(&allData)

	for _, data := range allData {
		logger.Printf("  %s: %s = %.2f\n",
			data.Timestamp.Format(time.RFC3339), data.SensorName, data.Value)
	}
}
