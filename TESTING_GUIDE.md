# Quick Testing Guide

## Prerequisites

1. **Setup database** (choose one):
   ```yaml
   # For SQLite (easiest - no setup required)
   database:
     driver: sqlite
     sqlite:
       path: ./sensor_data.db
   
   # For MySQL (requires MySQL server)
   database:
     driver: mysql  
     mysql:
       host: localhost
       port: 3306
       user: your_user
       password: your_password
       dbname: sensor_data
   ```

2. **Run migrations**:
   ```bash
   ./sensor_data_import migrate
   ```

## Test Commands

### 1. Small Dataset Test (Recommended First Test)
```bash
# Test with just the basic sensor files (84 records total)
./sensor_data_import scan test_data
```

**Expected Output:**
```
Processing 6 CSV file(s)
✓ Completed temperature_sensors.csv: 15 records, 0 errors
✓ Completed humidity_sensors.csv: 12 records, 0 errors  
✓ Completed pressure_sensors.csv: 10 records, 0 errors
✓ Completed mixed_quality_data.csv: 5 records, 5 errors
✓ Completed large_sensor_data.csv: 42 records, 0 errors
✓ Completed empty_file.csv: 0 records, 0 errors

Total files processed: 6
Total records imported: 84
Total parsing errors: 5

Note: sensors/environmental_data.csv ignored (subdirectory)
```

### 2. Large Dataset Test
```bash  
# Test with generated large files (~13,000 records total)
./sensor_data_import scan test_data
```

### 3. Error Handling Test
```bash
# Focus on the file with intentional errors
./sensor_data_import scan test_data/mixed_quality_data.csv
```

**Expected Warnings in Log:**
```
WARN: Row 3 has invalid value: invalid_value
WARN: Row 4 has invalid timestamp format: invalid_timestamp
WARN: Row 5 has empty sensor name
WARN: Row 7 has insufficient columns (expected 3, got 2)
WARN: Row 9 has invalid timestamp format: 
```

### 4. Performance Test
```bash
# Test with the largest file (7,200 records)
./sensor_data_import scan test_data/vibration_sensors.csv
```

### 5. Subdirectory Test
```bash
# Test direct subdirectory scanning (only files in that specific directory)
./sensor_data_import scan test_data/sensors
```

## Validation Commands

### Check Migration Status
```bash
./sensor_data_import migrate:status
```

### Database Info (No Logging)
```bash
./sensor_data_import db:info
```

### Test Insert Sample Data
```bash
./sensor_data_import test:insert
```

## Log File Locations

- **Default**: `result.log` in current directory
- **Custom**: Edit `log_file` in `config.yaml`
- **View recent logs**: `Get-Content result.log -Tail 50` (Windows) or `tail -50 result.log` (Linux/Mac)

## File Size Reference

| File | Records | Size | Test Purpose |
|------|---------|------|--------------|
| temperature_sensors.csv | 15 | 764B | Basic header parsing |
| humidity_sensors.csv | 12 | 552B | No header parsing |
| pressure_sensors.csv | 10 | 509B | Multiple timestamp formats |
| mixed_quality_data.csv | 5 | 433B | Error handling |
| large_sensor_data.csv | 42 | 1.7KB | Medium batch |
| empty_file.csv | 0 | 0B | Empty file handling |
| sensors/environmental_data.csv | 15 | 699B | Subdirectory ignored test |
| **Generated Files** |
| temperature_hourly.csv | 1,152 | 48KB | Hourly temperature data |
| humidity_realtime.csv | 960 | 44KB | Real-time humidity |
| pressure_daily.csv | 72 | 3.4KB | Daily pressure readings |
| light_sensors.csv | 3,600 | 159KB | Light sensor array |
| vibration_sensors.csv | 7,200 | 331KB | High-frequency vibration |

## Common Issues & Solutions

### 1. Database Connection Errors
```
FATAL: Connection failed: dial tcp [::1]:3306: connectex: No connection could be made
```
**Solution**: 
- For MySQL: Start MySQL server or switch to SQLite
- For SQLite: No action needed, file will be created automatically

### 2. Permission Errors
```
ERROR: failed to open log file result.log: access denied
```
**Solution**: Run from a directory where you have write permissions

### 3. No CSV Files Found
```
No CSV files found in the directory
```
**Solution**: 
- Check the directory path
- Ensure files have `.csv` extension
- Use absolute paths if relative paths don't work

### 4. Import Validation

After successful import, verify with SQL queries (using any SQL client):

```sql
-- Total imported records
SELECT COUNT(*) FROM sensor_data;

-- Records per sensor  
SELECT sensor_name, COUNT(*) as record_count 
FROM sensor_data 
GROUP BY sensor_name 
ORDER BY record_count DESC;

-- Time range
SELECT 
    MIN(timestamp) as earliest_reading,
    MAX(timestamp) as latest_reading 
FROM sensor_data;

-- Sample data
SELECT * FROM sensor_data LIMIT 10;
```

## Expected Performance

- **Small files** (<1KB): < 100ms per file
- **Medium files** (1-50KB): 100ms - 1s per file  
- **Large files** (50KB+): 1-5s per file
- **Parallel processing**: Typically 2-8 workers depending on CPU cores
- **Batch insertion**: 1000 records per batch for optimal performance

## Success Indicators

✅ **All tests passing**:
- All CSV files processed without crashes
- Expected number of records imported
- Parsing errors logged for intentionally bad data
- Log file created with session details
- No duplicate key violations (due to composite primary key)

✅ **Log file contains**:
- Session start/end timestamps  
- Command execution details
- File processing progress
- Error details for problematic records
- Processing summary with statistics
