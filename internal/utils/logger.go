package utils

import (
	"log/slog"
	"os"
)

// Logger wraps slog.Logger to provide structured logging
type Logger struct {
	*slog.Logger
}

// NewLogger creates a new logger instance with the specified level
func NewLogger(level string) *Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &Logger{
		Logger: logger,
	}
}

// WithFields adds structured fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	var args []any
	for k, v := range fields {
		args = append(args, k, v)
	}
	
	return &Logger{
		Logger: l.Logger.With(args...),
	}
} 