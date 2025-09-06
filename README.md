# Sensor Data import

A Go application for importing sensor data from CSV files into various databases using GORM as the ORM. The project supports MySQL (default), PostgreSQL, and SQLite databases with parallel processing for efficient data import.

## Features

- **Multi-database support**: MySQL, PostgreSQL, SQLite
- **Configurable database connections**: Easy configuration via YAML file
- **Migration system**: Database schema management with SQL migration files
- **Parallel CSV processing**: Process multiple CSV files simultaneously
- **Batch insertion**: Efficient bulk data insertion with automatic batching
- **Error handling**: Robust error handling with detailed logging
- **Composite primary key**: Uses timestamp + sensor_name as composite primary key
- **Configurable logging**: All operations logged to file with configurable log filename and level

## Project Structure

```
sensor_data_import/
├── config/                 # Configuration management
│   └── config.go
├── database/              # Database connection and migrations
│   ├── database.go
│   └── migration.go
├── migrations/            # SQL migration files
│   └── *.sql
├── models/               # Data models
│   └── sensor_data.go
├── scanner/              # CSV file processing
│   └── csv_scanner.go
├── config.yaml           # Configuration file
├── go.mod               # Go module file
├── main.go              # Main application entry point
└── README.md            # This file
```

## Data Model

The application uses a simple `SensorData` model:

```go
type SensorData struct {
ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
Timestamp  time.Time `gorm:"uniqueIndex:idx_timestamp_sensor;not null" json:"timestamp"`
SensorName string    `gorm:"uniqueIndex:idx_timestamp_sensor;not null;size:255" json:"sensor_name"`
Value      float64   `gorm:"not null" json:"value"`
CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}
```

## Configuration

Edit `config.yaml` to configure your database connection:

```yaml
database:
  # Supported drivers: mysql, postgres, sqlite
  driver: mysql
  
  # MySQL configuration (default)
  mysql:
    host: localhost
    port: 3306
    user: root
    password: ""
    dbname: sensor_data
    charset: utf8mb4
    parse_time: true
    loc: UTC
    
  # PostgreSQL configuration
  postgres:
    host: localhost
    port: 5432
    user: postgres
    password: ""
    dbname: sensor_data
    sslmode: disable
    timezone: UTC
    
  # SQLite configuration
  sqlite:
    path: ./sensor_data.db
    
# Logging settings
logging:
  log_file: result.log  # Log filename (default: result.log)
  log_to_console: true  # Also output to console
  log_level: info       # Log level: debug, info, warn, error
```

## Installation and Setup

1. **Clone or create the project directory**:
   ```bash
   cd /path/to/your/projects
   ```
2. **Copy `config-example.yaml` into `config.yaml` and config the setting as you need**:
   - You can choose to use `MySQL`, `PostgreSQL` or `SQLite`. Please remember to fill the connection information
     - *Remember to use `host.docker.internal` instead of `localhost` if you are inside the docker container*
   - It is recommended to set `logging.log_to_console` to `false` when you are processing large volume file


> If you are going to use docker to run instead of local `go` executable, please run the following commands before step 2:
> 1. You need to change the host from `localhost` to `host.docker.internal` in `config.yaml` file
> 2. Also, please mount your target directory into the docker container below `/app` directory
> - For Windows or macOS environment
>   ```bash
>   docker run -it --name golang -v "$(pwd):/app" -v "/path/to/target/directory:/app/target" -w /app --add-host=host.docker.internal:host-gateway golang:1.24-alpine sh
>   ```
> - For Linux environment
>   ```bash
>   docker run -it --name golang -v "$(pwd):/app -v "/path/to/target/directory:/app/target" -w /app --network host golang:1.24-alpine sh
>   ```

3. **Install dependencies**:
   ```bash
   go mod tidy
   ```

4. **Run database migrations**:
   ```bash
   go run main.go migrate
   ```

5. **Scan the sensor data**:
   ```bash
   # If you are using docker container and mounted the target directory into the container
   go run main.go scan target
   # If you are running the program locally
   go run main.go scan /path/to/target/directory
   ```

> If you are using docker to run the program, please remember to remove the container after things done
> ```bash
> docker rm -f golang
> ```

## Usage

### Available Commands

```bash
# Test database connection
go run main.go connect

# Run database migrations
go run main.go migrate

# Check migration status
go run main.go migrate:status

# Create a new migration
go run main.go migrate:create "add_new_table"

# Show database information
go run main.go db:info

# Scan directory for CSV files and import data
go run main.go scan /path/to/csv/directory

# Insert sample test data
go run main.go test:insert

# Show help
go run main.go help
```

### CSV File Format

The application expects CSV files with the following format:

```csv
timestamp,sensor_name,value
2025-09-05T12:30:45Z,temperature_sensor_01,23.5
2025-09-05T12:31:45Z,humidity_sensor_01,65.2
2025-09-05T12:32:45Z,pressure_sensor_01,1013.25
```

**Requirements:**
- **timestamp**: ISO8601 format (e.g., `2025-09-05T12:30:45Z`)
- **sensor_name**: String identifier for the sensor
- **value**: Numeric sensor reading

**Supported timestamp formats:**
- `2025-09-05T12:30:45Z` (RFC3339)
- `2025-09-05T12:30:45` (without timezone)
- `2025-09-05 12:30:45` (space separator)

### Scanning CSV Files

The `scan` command processes all CSV files in a directory in parallel:

```bash
# Scan a directory for CSV files
go run main.go scan /path/to/csv/files

# Example output:
Scanning directory: /path/to/csv/files
Found 5 CSV file(s) to process
Processing with 8 parallel workers
Processing file: sensor_data_001.csv
Processing file: sensor_data_002.csv
✓ Completed sensor_data_001.csv: 1000 records processed, 0 errors in 1.2s
✓ Completed sensor_data_002.csv: 1500 records processed, 2 errors in 1.8s

============================================================
PROCESSING SUMMARY
============================================================
✅ sensor_data_001.csv: 1000 records, 0 errors (1.2s)
✅ sensor_data_002.csv: 1500 records, 2 errors (1.8s)
------------------------------------------------------------
Total files processed: 2
Successful: 2
Failed: 0
Total records imported: 2500
Total parsing errors: 2
Total processing time: 3s
============================================================
```

## Performance Features

- **Parallel Processing**: Processes multiple CSV files simultaneously using configurable worker goroutines
- **Batch Insertion**: Inserts data in batches of 1000 records for optimal database performance
- **Connection Pooling**: Configurable database connection pool settings
- **Error Recovery**: If batch insertion fails, falls back to individual record insertion
- **Memory Efficient**: Processes large CSV files without loading everything into memory at once

## Logging System

The application includes a comprehensive logging system that outputs to both console and a configurable log file:

### Logging Configuration

```yaml
logging:
  log_file: result.log  # Custom log filename (default: result.log)
  log_to_console: true  # Output to console as well as file
  log_level: info       # Log level: debug, info, warn, error
```

### Log Behavior

- **Commands with logging**: `scan`, `migrate`, `migrate:create`, `migrate:status`, `connect`, `test:insert`
- **Commands without logging**: `help`, `db:info` (only console output)
- **Log location**: Same directory where the command is executed
- **Session tracking**: Each session is logged with start/end timestamps
- **Parallel processing**: All CSV processing results are logged with detailed progress

### Log Levels

- **debug**: Detailed debugging information
- **info**: General information messages (default)
- **warn**: Warning messages (parsing errors, etc.)
- **error**: Error messages (always logged regardless of level)

## Database Support

### MySQL (Default)
```yaml
database:
  driver: mysql
  mysql:
    host: localhost
    port: 3306
    user: root
    password: "your_password"
    dbname: sensor_data
```

### PostgreSQL
```yaml
database:
  driver: postgres
  postgres:
    host: localhost
    port: 5432
    user: postgres
    password: "your_password"
    dbname: sensor_data
    sslmode: disable
```

### SQLite
```yaml
database:
  driver: sqlite
  sqlite:
    path: ./sensor_data.db
```

## Migration System

The project includes a built-in migration system:

- **Create migrations**: `go run main.go migrate:create "migration_name"`
- **Run migrations**: `go run main.go migrate`
- **Check status**: `go run main.go migrate:status`

Migration files are stored in the `migrations/` directory with the naming convention:
`YYYYMMDD_HHMMSS_description.sql`

## Error Handling

The application provides comprehensive error handling:

- **File-level errors**: Invalid CSV format, missing files, permission issues
- **Record-level errors**: Invalid timestamps, missing fields, invalid numeric values
- **Database errors**: Connection issues, constraint violations, insertion failures
- **Detailed logging**: All errors are logged with specific details about the problematic data

## Building for Production

```bash
# Build executable
go build -o sensor_data_import main.go

# Run the executable
./sensor_data_import scan /path/to/csv/files
```

## Testing with Sample Data

The project includes comprehensive test data in the `test_data/` directory:

- **Small files** (tracked in git): Basic test cases with various scenarios
- **Large files** (git-ignored): Generated performance test data

To generate large test files for performance testing:
```bash
go run generate_test_data.go test_data/large_files
```

See `TESTING_GUIDE.md` and `test_data/README.md` for detailed testing instructions.
