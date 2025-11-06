package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

var (
	defaultLogger *slog.Logger
	logLevel      slog.Level = slog.LevelInfo
)

// Init initializes the logger with the specified configuration
func Init(level string, format string, output string) error {
	// Parse log level
	switch strings.ToLower(level) {
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

	// Set up output destination
	var writer io.Writer = os.Stdout
	if output != "" {
		file, err := os.OpenFile(output,
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		writer = file
	}

	// Set up handler based on format
	var handler slog.Handler
	switch strings.ToLower(format) {
	case "text":
		opts := &slog.HandlerOptions{
			Level:       logLevel,
			AddSource:   false,
			ReplaceAttr: replaceAttr,
		}
		handler = slog.NewTextHandler(writer, opts)
	case "json":
		fallthrough
	default:
		opts := &slog.HandlerOptions{
			Level:       logLevel,
			AddSource:   false,
			ReplaceAttr: replaceAttr,
		}
		handler = slog.NewJSONHandler(writer, opts)
	}

	defaultLogger = slog.New(handler)
	return nil
}

// replaceAttr replaces default attributes with more compact formats
// Changes "time" key to "t" and formats as HH:MM:SS.mmm
// Changes "level" key to "l" and formats as [LEVEL]
func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		a.Key = "t"
		if t, ok := a.Value.Any().(time.Time); ok {
			// Format as HH:MM:SS.mmm for compactness
			a.Value = slog.StringValue(t.Format("15:04:05.000"))
		}
	} else if a.Key == slog.LevelKey {
		a.Key = "l"
		if level, ok := a.Value.Any().(slog.Level); ok {
			// Format as [LEVEL] for compactness
			levelStr := level.String()
			if len(levelStr) > 4 {
				levelStr = levelStr[:4] // Truncate to 4 chars if needed
			}
			a.Value = slog.StringValue("[" + levelStr + "]")
		}
	}
	return a
}

// Debug logs a debug-level message
func Debug(msg string, fields ...any) {
	if defaultLogger == nil {
		slog.Debug(msg, fields...)
		return
	}
	defaultLogger.Debug(msg, fields...)
}

// Info logs an info-level message
func Info(msg string, fields ...any) {
	if defaultLogger == nil {
		slog.Info(msg, fields...)
		return
	}
	defaultLogger.Info(msg, fields...)
}

// Warn logs a warning-level message
func Warn(msg string, fields ...any) {
	if defaultLogger == nil {
		slog.Warn(msg, fields...)
		return
	}
	defaultLogger.Warn(msg, fields...)
}

// Error logs an error-level message
func Error(msg string, fields ...any) {
	if defaultLogger == nil {
		slog.Error(msg, fields...)
		return
	}
	defaultLogger.Error(msg, fields...)
}

// Fatal logs a fatal-level message and exits
func Fatal(msg string, fields ...any) {
	if defaultLogger == nil {
		slog.Error(msg, fields...)
		os.Exit(1)
		return
	}
	defaultLogger.Error(msg, fields...)
	os.Exit(1)
}

// WithFields returns a logger with the specified fields attached
func WithFields(fields ...any) *slog.Logger {
	if defaultLogger == nil {
		return slog.Default()
	}
	return defaultLogger.With(fields...)
}

// WithContext returns a logger with context attached
func WithContext(ctx context.Context) *slog.Logger {
	if defaultLogger == nil {
		return slog.Default()
	}
	return defaultLogger.With(toFields(ctx)...)
}

// toFields extracts fields from context
func toFields(ctx context.Context) []any {
	fields := []any{}
	if ctx == nil {
		return fields
	}

	// Extract common context values
	if userID, ok := ctx.Value("userID").(int); ok {
		fields = append(fields, "userID", userID)
	}
	if requestID, ok := ctx.Value("requestID").(string); ok {
		fields = append(fields, "requestID", requestID)
	}

	return fields
}

// GetLogger returns the default logger instance
func GetLogger() *slog.Logger {
	if defaultLogger == nil {
		return slog.Default()
	}
	return defaultLogger
}

// Writer returns an io.Writer that writes to the logger at Info level.
// This is useful for integrating with Fiber's logger middleware.
func Writer() io.Writer {
	return &loggerWriter{}
}

// loggerWriter implements io.Writer to write logs to slog
type loggerWriter struct{}

func (w *loggerWriter) Write(p []byte) (n int, err error) {
	// Remove trailing newline if present
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}

	if defaultLogger == nil {
		slog.Info(msg)
	} else {
		defaultLogger.Info(msg)
	}

	return len(p), nil
}
