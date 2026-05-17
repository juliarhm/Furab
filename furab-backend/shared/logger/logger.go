// Package logger provides structured logging for all Furab microservices.
// Uses Go's built-in log/slog package (Go 1.21+).
package logger

import (
	"context"
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with service-specific context.
type Logger struct {
	*slog.Logger
}

// New creates a new Logger instance for a specific service.
// In production (environment != "development"), it uses JSON format.
// In development, it uses human-readable text format.
func New(serviceName, environment string) *Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	if environment == "production" || environment == "staging" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler).With(
		slog.String("service", serviceName),
	)

	return &Logger{Logger: logger}
}

// WithRequestID adds a request ID to the logger context.
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger: l.With(slog.String("request_id", requestID)),
	}
}

// WithUserID adds a user ID to the logger context.
func (l *Logger) WithUserID(userID string) *Logger {
	return &Logger{
		Logger: l.With(slog.String("user_id", userID)),
	}
}

// WithError adds an error to the logger context.
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger: l.With(slog.String("error", err.Error())),
	}
}

// WithContext extracts request metadata from context and adds to logger.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	logger := l.Logger
	if requestID, ok := ctx.Value("requestID").(string); ok {
		logger = logger.With(slog.String("request_id", requestID))
	}
	if userID, ok := ctx.Value("userID").(string); ok {
		logger = logger.With(slog.String("user_id", userID))
	}
	return &Logger{Logger: logger}
}

// Infof logs an info message with formatted string.
func (l *Logger) Infof(msg string, args ...interface{}) {
	l.Info(msg, args...)
}

// Errorf logs an error message with formatted string.
func (l *Logger) Errorf(msg string, args ...interface{}) {
	l.Error(msg, args...)
}

// Debugf logs a debug message with formatted string.
func (l *Logger) Debugf(msg string, args ...interface{}) {
	l.Debug(msg, args...)
}

// Warnf logs a warning message with formatted string.
func (l *Logger) Warnf(msg string, args ...interface{}) {
	l.Warn(msg, args...)
}
