package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"sensor_data_import/config"
)

var (
	// Global logger instances
	InfoLogger   *log.Logger
	ErrorLogger  *log.Logger
	DebugLogger  *log.Logger
	WarnLogger   *log.Logger
	logFile      *os.File
	logLevel     string
	logToConsole bool
)

// LogLevel constants
const (
	DEBUG = "debug"
	INFO  = "info"
	WARN  = "warn"
	ERROR = "error"
)

// Init initializes the logging system using configuration
func Init(cfg *config.Config) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Set global variables from config
	logToConsole = cfg.Logging.LogToConsole
	logLevel = cfg.Logging.LogLevel

	// Create log file path
	logPath := filepath.Join(cwd, cfg.Logging.LogFile)

	// Create or open log file
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", logPath, err)
	}

	// Create writers based on configuration
	var infoWriter, errorWriter, debugWriter, warnWriter io.Writer

	if logToConsole {
		// Write to both console and file
		infoWriter = io.MultiWriter(os.Stdout, logFile)
		errorWriter = io.MultiWriter(os.Stderr, logFile)
		debugWriter = io.MultiWriter(os.Stdout, logFile)
		warnWriter = io.MultiWriter(os.Stdout, logFile)
	} else {
		// Write only to file
		infoWriter = logFile
		errorWriter = logFile
		debugWriter = logFile
		warnWriter = logFile
	}

	// Create loggers with no prefix for clean output
	InfoLogger = log.New(infoWriter, "", 0)
	ErrorLogger = log.New(errorWriter, "", 0)
	DebugLogger = log.New(debugWriter, "", 0)
	WarnLogger = log.New(warnWriter, "", 0)

	// Log session start
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	InfoLogger.Printf("=== Session started at %s ===\n", timestamp)
	InfoLogger.Printf("Log file: %s\n", logPath)
	InfoLogger.Printf("Log level: %s\n", logLevel)
	InfoLogger.Printf("Log to console: %t\n", logToConsole)
	LogDivider()

	return nil
}

// Close closes the log file
func Close() error {
	if logFile != nil {
		// Log session end
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		LogDivider()
		InfoLogger.Printf("=== Session ended at %s ===\n\n", timestamp)
		return logFile.Close()
	}
	return nil
}

// shouldLog determines if a message should be logged based on log level
func shouldLog(messageLevel string) bool {
	levels := map[string]int{
		DEBUG: 0,
		INFO:  1,
		WARN:  2,
		ERROR: 3,
	}

	currentLevel, exists := levels[logLevel]
	if !exists {
		currentLevel = levels[INFO] // Default to INFO if invalid level
	}

	messageLogLevel, exists := levels[messageLevel]
	if !exists {
		return true // Log unknown levels
	}

	return messageLogLevel >= currentLevel
}

// Printf prints formatted text to log (respects log level)
func Printf(format string, v ...interface{}) {
	if InfoLogger != nil && shouldLog(INFO) {
		InfoLogger.Printf(format, v...)
	} else if shouldLog(INFO) {
		fmt.Printf(format, v...)
	}
}

// Println prints a line to log (respects log level)
func Println(v ...interface{}) {
	if InfoLogger != nil && shouldLog(INFO) {
		InfoLogger.Println(v...)
	} else if shouldLog(INFO) {
		fmt.Println(v...)
	}
}

// Print prints to log (respects log level)
func Print(v ...interface{}) {
	if InfoLogger != nil && shouldLog(INFO) {
		InfoLogger.Print(v...)
	} else if shouldLog(INFO) {
		fmt.Print(v...)
	}
}

// Debugf prints formatted debug text
func Debugf(format string, v ...interface{}) {
	if DebugLogger != nil && shouldLog(DEBUG) {
		DebugLogger.Printf("DEBUG: "+format, v...)
	} else if shouldLog(DEBUG) {
		fmt.Printf("DEBUG: "+format, v...)
	}
}

// Debugln prints debug line
func Debugln(v ...interface{}) {
	if DebugLogger != nil && shouldLog(DEBUG) {
		DebugLogger.Print("DEBUG: ")
		DebugLogger.Println(v...)
	} else if shouldLog(DEBUG) {
		fmt.Print("DEBUG: ")
		fmt.Println(v...)
	}
}

// Warnf prints formatted warning text
func Warnf(format string, v ...interface{}) {
	if WarnLogger != nil && shouldLog(WARN) {
		WarnLogger.Printf("WARN: "+format, v...)
	} else if shouldLog(WARN) {
		fmt.Printf("WARN: "+format, v...)
	}
}

// Warnln prints warning line
func Warnln(v ...interface{}) {
	if WarnLogger != nil && shouldLog(WARN) {
		WarnLogger.Print("WARN: ")
		WarnLogger.Println(v...)
	} else if shouldLog(WARN) {
		fmt.Print("WARN: ")
		fmt.Println(v...)
	}
}

// Errorf prints formatted error text (always logged regardless of level)
func Errorf(format string, v ...interface{}) {
	if ErrorLogger != nil {
		ErrorLogger.Printf("ERROR: "+format, v...)
	} else {
		fmt.Fprintf(os.Stderr, "ERROR: "+format, v...)
	}
}

// Errorln prints error line (always logged regardless of level)
func Errorln(v ...interface{}) {
	if ErrorLogger != nil {
		ErrorLogger.Print("ERROR: ")
		ErrorLogger.Println(v...)
	} else {
		fmt.Fprint(os.Stderr, "ERROR: ")
		fmt.Fprintln(os.Stderr, v...)
	}
}

// Fatalf prints formatted fatal error and exits (always logged)
func Fatalf(format string, v ...interface{}) {
	if ErrorLogger != nil {
		ErrorLogger.Printf("FATAL: "+format, v...)
	} else {
		fmt.Fprintf(os.Stderr, "FATAL: "+format, v...)
	}
	Close()
	os.Exit(1)
}

// LogCommand logs the command being executed
func LogCommand(command string, args []string) {
	Printf("Command executed: %s", command)
	if len(args) > 1 {
		Printf(" %v", args[1:])
	}
	Println("")
}

// LogDivider prints a divider line for better log organization
func LogDivider() {
	Println("------------------------------------------------------------")
}

// LogResult logs a result with status
func LogResult(operation string, success bool, details string) {
	if success {
		Printf("✅ %s: SUCCESS", operation)
	} else {
		Printf("❌ %s: FAILED", operation)
	}

	if details != "" {
		Printf(" - %s", details)
	}
	Println("")
}

// LogProgress logs progress information
func LogProgress(current, total int, item string) {
	Printf("Progress: [%d/%d] %s\n", current, total, item)
}

// GetLogFileName returns the current log file name
func GetLogFileName() string {
	if logFile != nil {
		return logFile.Name()
	}
	return "result.log"
}
