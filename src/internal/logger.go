package internal

import (
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// GlobalLogger is the shared logger instance
	GlobalLogger zerolog.Logger
	once         sync.Once
)

// Component type can still be used for adding structured context
type Component string

const (
	ComponentNATS        Component = "NATS"
	ComponentStorage     Component = "Storage"
	ComponentConfig      Component = "Config"
	ComponentTransaction Component = "Trans"
	ComponentService     Component = "Service"
	ComponentGeneral     Component = "General"
	ComponentCLI         Component = "CLI"
	ComponentAPI         Component = "API" // Added for potential API logging
)

// InitGlobalLogger initializes the global zerolog logger.
// It defaults to Info level and console output.
// Call ConfigureLogger later to adjust based on config.
func InitGlobalLogger() {
	once.Do(func() {
		// Default to pretty console logging for development
		output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
		GlobalLogger = zerolog.New(output).Level(zerolog.InfoLevel).With().Timestamp().Logger()
		log.Logger = GlobalLogger // Set the global log package logger
	})
}

// ConfigureLogger sets the log level and output based on configuration.
// TODO: Integrate this with config loading (e.g., call this from LoadConfig).
func ConfigureLogger(logLevel string, logFile string) error {
	// Parse log level string
	level, err := zerolog.ParseLevel(strings.ToLower(logLevel))
	if err != nil {
		level = zerolog.InfoLevel // Default to Info on parse error
		log.Warn().Err(err).Str("providedLevel", logLevel).Msg("Invalid log level string, defaulting to INFO")
	}

	var writers []io.Writer

	// Console Writer (always enabled for now, could be configurable)
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	writers = append(writers, consoleWriter)

	// File Writer (if path provided)
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			// Log error to console but continue without file logging
			log.Error().Err(err).Str("path", logFile).Msg("Failed to open log file")
		} else {
			// Consider adding file rotation later
			writers = append(writers, file)
			// TODO: Add a mechanism to close the file on application shutdown
		}
	}

	multiWriter := io.MultiWriter(writers...)
	GlobalLogger = zerolog.New(multiWriter).Level(level).With().Timestamp().Logger()
	log.Logger = GlobalLogger // Update the global log package logger

	log.Info().Str("level", level.String()).Msg("Logger configured")
	return nil
}

// GetLogger returns the initialized global logger.
// It ensures InitGlobalLogger is called at least once.
func GetLogger() zerolog.Logger {
	InitGlobalLogger() // Ensure logger is initialized
	return GlobalLogger
}

// Example of adding component context:
// GetLogger().Info().Str("component", string(ComponentAPI)).Msg("API request received")
