package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/term"
)

var (
	defaultLogger *slog.Logger
	logLevel      slog.Level = slog.LevelInfo
	useColors     bool       = false
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
		// Determine if colors should be used (only for text format to terminal)
		useColors = isTerminal(writer)
		opts := &slog.HandlerOptions{
			Level:       logLevel,
			AddSource:   false,
			ReplaceAttr: replaceAttr,
		}
		if useColors {
			// Use a color writer that post-processes output
			colorW := &colorWriter{Writer: writer}
			handler = slog.NewTextHandler(colorW, opts)
		} else {
			handler = slog.NewTextHandler(writer, opts)
		}
	case "json":
		fallthrough
	default:
		// Never use colors for JSON format
		useColors = false
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
			// Map slog levels to our preferred format
			switch level {
			case slog.LevelDebug:
				levelStr = "DEBUG"
			case slog.LevelInfo:
				levelStr = "INFO"
			case slog.LevelWarn:
				levelStr = "WARN"
			case slog.LevelError:
				levelStr = "ERROR"
			default:
				if len(levelStr) > 4 {
					levelStr = levelStr[:4] // Truncate to 4 chars if needed
				}
			}
			a.Value = slog.StringValue("[" + levelStr + "]")
		}
	}
	return a
}

// Debug logs a debug-level message
func Debug(msg string, fields ...any) {
	defaultLogger.Debug(msg, fields...)
}

// Info logs an info-level message
func Info(msg string, fields ...any) {
	defaultLogger.Info(msg, fields...)
}

// Warn logs a warning-level message
func Warn(msg string, fields ...any) {
	defaultLogger.Warn(msg, fields...)
}

// Error logs an error-level message
func Error(msg string, fields ...any) {
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
	return defaultLogger.With(fields...)
}

// WithContext returns a logger with context attached
func WithContext(ctx context.Context) *slog.Logger {
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

	defaultLogger.Info(msg)

	return len(p), nil
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGreen  = "\033[32m"
	colorGray   = "\033[90m"
)

// isTerminal checks if the writer is a terminal (TTY)
func isTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

// colorWriter wraps an io.Writer and adds colors to log level indicators
type colorWriter struct {
	io.Writer
}

var (
	// Regex patterns to match log level indicators
	levelPatterns = map[string]*regexp.Regexp{
		"ERROR": regexp.MustCompile(`\bl="\[ERROR\]"`),
		"WARN":  regexp.MustCompile(`\bl="\[WARN\]"`),
		"INFO":  regexp.MustCompile(`\bl="\[INFO\]"`),
		"DEBUG": regexp.MustCompile(`\bl="\[DEBUG\]"`),
	}
)

func (w *colorWriter) Write(p []byte) (n int, err error) {
	output := string(p)

	// Colorize each level pattern - replace quoted versions with colored unquoted versions
	// This works because we're post-processing after the handler has formatted everything
	output = levelPatterns["ERROR"].ReplaceAllString(output, `l=`+colorRed+`[ERROR]`+colorReset)
	output = levelPatterns["WARN"].ReplaceAllString(output, `l=`+colorYellow+`[WARN]`+colorReset)
	output = levelPatterns["INFO"].ReplaceAllString(output, `l=`+colorBlue+`[INFO]`+colorReset)
	output = levelPatterns["DEBUG"].ReplaceAllString(output, `l=`+colorGreen+`[DEBUG]`+colorReset)

	// Also handle unquoted versions (in case handler doesn't quote)
	output = strings.ReplaceAll(output, ` l=[ERROR] `, ` l=`+colorRed+`[ERROR]`+colorReset+` `)
	output = strings.ReplaceAll(output, ` l=[WARN] `, ` l=`+colorYellow+`[WARN]`+colorReset+` `)
	output = strings.ReplaceAll(output, ` l=[INFO] `, ` l=`+colorBlue+`[INFO]`+colorReset+` `)
	output = strings.ReplaceAll(output, ` l=[DEBUG] `, ` l=`+colorGreen+`[DEBUG]`+colorReset+` `)

	_, err = w.Writer.Write([]byte(output))
	return len(p), err
}
