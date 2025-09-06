# Test Data for Sensor Data import

This directory contains various CSV test files to validate the scanning and import functionality.

## Test Files Overview

### Basic Sensor Data Files

#### `temperature_sensors.csv`
- **Format**: With header row
- **Records**: 15 temperature readings 
- **Sensors**: 3 temperature sensors (temperature_sensor_01, 02, 03)
- **Time range**: 2025-09-05 08:00:00Z to 08:20:00Z (5-minute intervals)
- **Values**: 21.8°C to 26.7°C
- **Purpose**: Test basic CSV parsing with headers

#### `humidity_sensors.csv`
- **Format**: Without header row
- **Records**: 12 humidity readings
- **Sensors**: 2 humidity sensors (humidity_sensor_01, 02) 
- **Time range**: 2025-09-05 08:00:00Z to 08:25:00Z (5-minute intervals)
- **Values**: 58.7% to 65.2% relative humidity
- **Purpose**: Test CSV parsing without headers

#### `pressure_sensors.csv`
- **Format**: With header, mixed timestamp formats
- **Records**: 10 pressure readings
- **Sensors**: 2 pressure sensors (pressure_sensor_01, 02)
- **Time formats**: 
  - ISO8601 without timezone: `2025-09-05T08:00:00`
  - Space-separated: `2025-09-05 08:00:00`
- **Values**: 1012.97 to 1013.41 hPa
- **Purpose**: Test multiple timestamp format parsing

### Error Testing Files

#### `mixed_quality_data.csv`
- **Format**: With header
- **Records**: 10 rows with various data quality issues
- **Sensors**: Light sensors (light_sensor_01, 02, 03)
- **Error types**:
  - Invalid numeric values (`invalid_value`)
  - Invalid timestamps (`invalid_timestamp`)
  - Empty sensor names
  - Missing values (insufficient columns)
  - Empty timestamp fields
- **Purpose**: Test error handling and data validation

#### `empty_file.csv`
- **Format**: Completely empty file
- **Purpose**: Test edge case of empty files

### Performance Testing Files

#### `large_sensor_data.csv`
- **Format**: With header
- **Records**: 42 motion sensor readings
- **Sensors**: 2 motion sensors (motion_sensor_01, 02)
- **Values**: Binary (0 or 1) representing motion detection
- **Time range**: 2025-09-05 10:00:00Z to 10:05:00Z (15-second intervals)
- **Purpose**: Test batch processing and performance

### Additional Test Files

#### `sensors/environmental_data.csv` (In subdirectory - will be ignored)
- **Format**: With header
- **Records**: 15 environmental readings
- **Sensors**: CO2, air quality, and noise sensors
- **Values**: 
  - CO2: 410.5 to 425.6 ppm
  - Air quality: 82.7 to 91.3 AQI
  - Noise: 45.8 to 52.1 dB
- **Purpose**: Test that subdirectories are ignored (this file should NOT be processed)

### Non-CSV Files

#### `readme.txt`
- **Purpose**: Test that non-CSV files are ignored

## Test Scenarios

### 1. Basic Import Test
```bash
./sensor_data_import.exe scan test_data
```
**Expected**: Process all CSV files in test_data directory, ignore .txt files and subdirectories

### 2. Single File Test
```bash
./sensor_data_import.exe scan test_data/temperature_sensors.csv
```
**Expected**: Process only the temperature sensors file

### 3. Subdirectory Ignored Test
```bash
./sensor_data_import.exe scan test_data/sensors
```
**Expected**: Process only CSV files directly in the sensors directory (environmental_data.csv)

### 4. Error Handling Test
Focus on `mixed_quality_data.csv` results:
- **Expected errors**: 5 parsing errors
- **Expected success**: 5 valid records imported
- **Log should show**: Detailed warning messages for each error

### 5. Empty File Test
Check handling of `empty_file.csv`:
- **Expected**: File processed but no records imported
- **Log should show**: Appropriate handling of empty file

### 6. Performance Test
Process `large_sensor_data.csv`:
- **Expected**: All 42 records imported efficiently
- **Log should show**: Batch processing statistics

## Generating Additional Test Data

Use the included data generator for larger test sets:

```bash
# Generate additional test files
go run generate_test_data.go test_data/large_files
```

## Expected Results Summary

When running the full test suite (`./sensor_data_import.exe scan test_data`):

| File | Records | Errors | Notes |
|------|---------|--------|---------|
| temperature_sensors.csv | 15 | 0 | Perfect data with headers |
| humidity_sensors.csv | 12 | 0 | Perfect data without headers |
| pressure_sensors.csv | 10 | 0 | Multiple timestamp formats |
| mixed_quality_data.csv | 5 | 5 | Various data quality issues |
| large_sensor_data.csv | 42 | 0 | Binary sensor data |
| empty_file.csv | 0 | 0 | Empty file handling |
| **Total** | **84** | **5** | **6 files processed** |

**Note**: `sensors/environmental_data.csv` is NOT processed as subdirectories are ignored.

## Validation Queries

After importing, you can validate the data with SQL queries:

```sql
-- Count total records
SELECT COUNT(*) FROM sensor_data;

-- Count by sensor type
SELECT sensor_name, COUNT(*) FROM sensor_data GROUP BY sensor_name;

-- Check time range
SELECT MIN(timestamp), MAX(timestamp) FROM sensor_data;

-- Find duplicate keys (should be 0 due to composite primary key)
SELECT timestamp, sensor_name, COUNT(*) 
FROM sensor_data 
GROUP BY timestamp, sensor_name 
HAVING COUNT(*) > 1;
```

## Git Tracking

### Files Tracked in Git (Small Test Files)
These files are committed to version control for immediate testing:

**Root directory files (processed by scanner)**:
- `temperature_sensors.csv` (764B)
- `humidity_sensors.csv` (552B) 
- `pressure_sensors.csv` (509B)
- `mixed_quality_data.csv` (433B)
- `large_sensor_data.csv` (1.7KB)
- `empty_file.csv` (0B)
- `readme.txt` (176B)

**Subdirectory files (ignored by scanner)**:
- `sensors/environmental_data.csv` (699B) - Used to test that subdirectories are ignored

### Files Ignored by Git (Large Generated Files)
These are generated by `generate_test_data.go` and excluded from version control:
- `temperature_hourly.csv` (~88MB)
- `humidity_realtime.csv` (~50MB)
- `pressure_daily.csv` (~6.3MB)
- `light_sensors.csv` (~289MB)
- `vibration_sensors.csv` (~604MB)

**To generate large files**: `go run generate_test_data.go test_data/large_files`

## Notes

- All timestamps are in UTC timezone
- The composite primary key (timestamp + sensor_name) prevents duplicate entries
- Files use different timestamp formats to test parser flexibility
- Error data is intentionally included to validate robust error handling
- File sizes range from empty to several thousand records for performance testing
- Large files are excluded from git to keep repository size manageable
