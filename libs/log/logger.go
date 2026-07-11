package log

import (
	"fmt"
	"io"
	"time"

	"cosmossdk.io/log"
	"github.com/rs/zerolog"

	"github.com/sentinel-official/sentinel-go-sdk/v2/types"
)

type Logger = log.Logger // An alias for the log.Logger type

var (
	// Global logger, default to a no-op logger.
	logger = log.NewNopLogger() //nolint:gochecknoglobals
)

// Info logs an informational message.
func Info(msg string, keyVals ...any) {
	logger.Info(msg, keyVals...)
}

// Warn logs a warning message.
func Warn(msg string, keyVals ...any) {
	logger.Warn(msg, keyVals...)
}

// Error logs an error message.
func Error(msg string, keyVals ...any) {
	logger.Error(msg, keyVals...)
}

// Debug logs a debug message.
func Debug(msg string, keyVals ...any) {
	logger.Debug(msg, keyVals...)
}

// With returns a logger with additional context.
func With(keyVals ...any) Logger {
	return logger.With(keyVals...)
}

// Impl returns the underlying implementation of the logger.
func Impl() any {
	return logger.Impl()
}

// SetLogger sets the global logger instance.
func SetLogger(l Logger) {
	if l != nil {
		logger = l
	}
}

// NewLogger creates a new logger instance with the specified output writer, format, and log level.
func NewLogger(w io.Writer, format, level string) (Logger, error) {
	// Check if the format is valid.
	validFormats := map[string]bool{
		types.FormatJSON: true,
		types.FormatText: true,
	}
	if !validFormats[format] {
		return nil, fmt.Errorf("unsupported log format %q (allowed: json, text)", format)
	}

	// Check if the level is valid.
	validLevels := map[string]bool{
		"debug": true,
		"error": true,
		"info":  true,
		"none":  true,
		"warn":  true,
	}
	if !validLevels[level] {
		return nil, fmt.Errorf("unsupported log level %q (allowed: debug, error, info, none, warn)", level)
	}

	// Return a no-op logger if logging is disabled.
	if level == "none" {
		return log.NewNopLogger(), nil
	}

	// Parse the log level from the string
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("parsing log level %q: %w", level, err)
	}

	// Prepare options for logger
	opts := []log.Option{
		log.LevelOption(logLevel),
		log.TimeFormatOption(time.RFC3339),
	}

	// Set log format based on the provided format string
	if format == types.FormatJSON {
		opts = append(opts, log.OutputJSONOption())
	}

	// Create and return the logger with the specified options
	return log.NewLogger(w, opts...), nil
}
