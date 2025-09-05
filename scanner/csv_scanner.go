package scanner

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"sensor_data_import/logger"
	"sensor_data_import/models"

	"gorm.io/gorm"
)

// CSVScanner handles scanning and processing CSV files
type CSVScanner struct {
	db          *gorm.DB
	workerCount int
}

// FileJob represents a CSV file to be processed
type FileJob struct {
	FilePath string
	FileName string
}

// ProcessResult contains the result of processing a CSV file
type ProcessResult struct {
	FilePath    string
	RecordCount int
	ErrorCount  int
	Duration    time.Duration
	Error       error
}

// NewCSVScanner creates a new CSV scanner
func NewCSVScanner(db *gorm.DB) *CSVScanner {
	// Default to number of CPU cores for parallel processing
	workerCount := runtime.NumCPU()
	if workerCount > 8 {
		workerCount = 8 // Limit to 8 workers to avoid overwhelming the database
	}

	return &CSVScanner{
		db:          db,
		workerCount: workerCount,
	}
}

// SetWorkerCount sets the number of parallel workers
func (cs *CSVScanner) SetWorkerCount(count int) {
	if count > 0 {
		cs.workerCount = count
	}
}

// ScanDirectory scans a directory for CSV files and processes them in parallel
func (cs *CSVScanner) ScanDirectory(directoryPath string) error {
	logger.Printf("Scanning directory: %s\n", directoryPath)

	// Check if directory exists
	if _, err := os.Stat(directoryPath); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", directoryPath)
	}

	// Find all CSV files
	csvFiles, err := cs.findCSVFiles(directoryPath)
	if err != nil {
		return fmt.Errorf("failed to find CSV files: %w", err)
	}

	if len(csvFiles) == 0 {
		logger.Println("No CSV files found in the directory")
		return nil
	}

	logger.Printf("Found %d CSV file(s) to process\n", len(csvFiles))
	logger.Printf("Processing with %d parallel workers\n", cs.workerCount)

	// Process files in parallel
	results := cs.processFilesParallel(csvFiles)

	// Display results summary
	cs.displaySummary(results)

	return nil
}

// findCSVFiles finds all CSV files in the specified directory (non-recursive)
func (cs *CSVScanner) findCSVFiles(directoryPath string) ([]FileJob, error) {
	var csvFiles []FileJob

	// Read directory contents
	entries, err := os.ReadDir(directoryPath)
	if err != nil {
		return nil, err
	}

	// Process each entry
	for _, entry := range entries {
		// Skip subdirectories
		if entry.IsDir() {
			continue
		}

		// Check if file has CSV extension
		if strings.ToLower(filepath.Ext(entry.Name())) == ".csv" {
			filePath := filepath.Join(directoryPath, entry.Name())
			csvFiles = append(csvFiles, FileJob{
				FilePath: filePath,
				FileName: entry.Name(),
			})
		}
	}

	return csvFiles, nil
}

// processFilesParallel processes CSV files in parallel using worker goroutines
func (cs *CSVScanner) processFilesParallel(files []FileJob) []ProcessResult {
	jobs := make(chan FileJob, len(files))
	results := make(chan ProcessResult, len(files))

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < cs.workerCount; i++ {
		wg.Add(1)
		go cs.worker(jobs, results, &wg)
	}

	// Send jobs
	go func() {
		for _, file := range files {
			jobs <- file
		}
		close(jobs)
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	var allResults []ProcessResult
	for result := range results {
		allResults = append(allResults, result)
	}

	return allResults
}

// worker processes CSV files from the job channel
func (cs *CSVScanner) worker(jobs <-chan FileJob, results chan<- ProcessResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		result := cs.processCSVFile(job)
		results <- result
	}
}

// processCSVFile processes a single CSV file
func (cs *CSVScanner) processCSVFile(job FileJob) ProcessResult {
	startTime := time.Now()
	result := ProcessResult{
		FilePath: job.FilePath,
	}

	logger.Printf("Processing file: %s\n", job.FileName)

	// Open CSV file
	file, err := os.Open(job.FilePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to open file: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}
	defer file.Close()

	// Create CSV reader
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		result.Error = fmt.Errorf("failed to read CSV: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	if len(records) == 0 {
		result.Error = fmt.Errorf("empty CSV file")
		result.Duration = time.Since(startTime)
		return result
	}

	// Process records (skip header if present)
	sensorData, errorCount := cs.parseCSVRecords(records, job.FileName)
	result.RecordCount = len(sensorData)
	result.ErrorCount = errorCount

	// Batch insert sensor data
	if len(sensorData) > 0 {
		if err := cs.batchInsertSensorData(sensorData); err != nil {
			result.Error = fmt.Errorf("failed to insert data: %w", err)
			result.Duration = time.Since(startTime)
			return result
		}
	}

	result.Duration = time.Since(startTime)
	logger.Printf("✓ Completed %s: %d records processed, %d errors in %v\n",
		job.FileName, result.RecordCount, result.ErrorCount, result.Duration)

	return result
}

// parseCSVRecords parses CSV records into SensorData structs
func (cs *CSVScanner) parseCSVRecords(records [][]string, fileName string) ([]models.SensorData, int) {
	var sensorData []models.SensorData
	var errorCount int

	// Detect if first row is header
	startRow := 0
	if len(records) > 0 && cs.isHeaderRow(records[0]) {
		startRow = 1
	}

	for i := startRow; i < len(records); i++ {
		record := records[i]

		// Skip empty rows
		if len(record) == 0 || (len(record) == 1 && strings.TrimSpace(record[0]) == "") {
			continue
		}

		// Expect at least 3 columns: timestamp, sensor_name, value
		if len(record) < 3 {
			errorCount++
			logger.Warnf("Row %d in %s has insufficient columns (expected 3, got %d)\n",
				i+1, fileName, len(record))
			continue
		}

		// Parse timestamp
		timestampStr := strings.TrimSpace(record[0])
		timestamp, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			// Try alternative formats
			if timestamp, err = time.Parse("2006-01-02T15:04:05", timestampStr); err != nil {
				if timestamp, err = time.Parse("2006-01-02 15:04:05", timestampStr); err != nil {
					errorCount++
					logger.Warnf("Row %d in %s has invalid timestamp format: %s\n",
						i+1, fileName, timestampStr)
					continue
				}
			}
		}

		// Parse sensor name
		sensorName := strings.TrimSpace(record[1])
		if sensorName == "" {
			errorCount++
			logger.Warnf("Row %d in %s has empty sensor name\n", i+1, fileName)
			continue
		}

		// Parse value
		valueStr := strings.TrimSpace(record[2])
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			errorCount++
			logger.Warnf("Row %d in %s has invalid value: %s\n", i+1, fileName, valueStr)
			continue
		}

		// Create sensor data entry
		sensorData = append(sensorData, models.SensorData{
			Timestamp:  timestamp.UTC(),
			SensorName: sensorName,
			Value:      value,
		})
	}

	return sensorData, errorCount
}

// isHeaderRow checks if the first row is likely a header
func (cs *CSVScanner) isHeaderRow(row []string) bool {
	if len(row) < 3 {
		return false
	}

	// Check if first column looks like a timestamp or contains header words
	firstCol := strings.ToLower(strings.TrimSpace(row[0]))
	headerWords := []string{"timestamp", "time", "date", "datetime"}

	for _, word := range headerWords {
		if strings.Contains(firstCol, word) {
			return true
		}
	}

	// Try to parse as timestamp - if it fails, it's likely a header
	_, err := time.Parse(time.RFC3339, strings.TrimSpace(row[0]))
	return err != nil
}

// batchInsertSensorData inserts sensor data in batches to improve performance
func (cs *CSVScanner) batchInsertSensorData(data []models.SensorData) error {
	const batchSize = 1000

	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}

		batch := data[i:end]

		// Use GORM's CreateInBatches for efficient batch insertion
		if err := cs.db.CreateInBatches(batch, batchSize).Error; err != nil {
			// If batch insert fails, try individual inserts to identify problematic records
			return cs.individualInsert(batch)
		}
	}

	return nil
}

// individualInsert attempts to insert records individually when batch insert fails
func (cs *CSVScanner) individualInsert(data []models.SensorData) error {
	var lastError error
	successCount := 0

	for _, record := range data {
		if err := cs.db.Create(&record).Error; err != nil {
			lastError = err
			// Log the error but continue with other records
			logger.Warnf("Failed to insert record %s at %s: %v\n",
				record.SensorName, record.Timestamp.Format(time.RFC3339), err)
		} else {
			successCount++
		}
	}

	if successCount == 0 && lastError != nil {
		return fmt.Errorf("failed to insert any records: %w", lastError)
	}

	if lastError != nil {
		logger.Printf("Inserted %d out of %d records with some errors\n", successCount, len(data))
	}

	return nil
}

// displaySummary displays a summary of the processing results
func (cs *CSVScanner) displaySummary(results []ProcessResult) {
	logger.Println("\n" + strings.Repeat("=", 60))
	logger.Println("PROCESSING SUMMARY")
	logger.Println(strings.Repeat("=", 60))

	totalFiles := len(results)
	totalRecords := 0
	totalErrors := 0
	successfulFiles := 0
	failedFiles := 0
	totalDuration := time.Duration(0)

	for _, result := range results {
		if result.Error != nil {
			failedFiles++
			logger.Printf("❌ %s: FAILED - %v\n", filepath.Base(result.FilePath), result.Error)
		} else {
			successfulFiles++
			totalRecords += result.RecordCount
			totalErrors += result.ErrorCount
			logger.Printf("✅ %s: %d records, %d errors (%v)\n",
				filepath.Base(result.FilePath), result.RecordCount, result.ErrorCount, result.Duration)
		}
		totalDuration += result.Duration
	}

	logger.Println(strings.Repeat("-", 60))
	logger.Printf("Total files processed: %d\n", totalFiles)
	logger.Printf("Successful: %d\n", successfulFiles)
	logger.Printf("Failed: %d\n", failedFiles)
	logger.Printf("Total records imported: %d\n", totalRecords)
	logger.Printf("Total parsing errors: %d\n", totalErrors)
	logger.Printf("Total processing time: %v\n", totalDuration)
	logger.Println(strings.Repeat("=", 60))
}
