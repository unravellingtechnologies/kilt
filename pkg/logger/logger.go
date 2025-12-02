// Package logger provides structured logging with colour support for Kilt.
// It respects the --no-colour flag and supports different log levels (debug, info, warn, error).
package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Level represents the log level
type Level int

// Log level constants define the severity of log messages.
const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// colours for terminal output
const (
	colourReset  = "\033[0m"
	colourRed    = "\033[31m"
	colourGreen  = "\033[32m"
	colourYellow = "\033[33m"
	colourBlue   = "\033[34m"
	colourCyan   = "\033[36m"
	colourGray   = "\033[90m"
)

// Logger provides structured logging with colour support
type Logger struct {
	level      Level
	output     io.Writer
	colourized bool
	prefix     string
}

// NewLogger creates a new logger instance
func NewLogger(level Level, colourized bool) *Logger {
	return &Logger{
		level:      level,
		output:     os.Stderr,
		colourized: colourized && isTerminal(os.Stderr),
		prefix:     "",
	}
}

// WithPrefix returns a new logger with a prefix
func (l *Logger) WithPrefix(prefix string) *Logger {
	return &Logger{
		level:      l.level,
		output:     l.output,
		colourized: l.colourized,
		prefix:     prefix,
	}
}

// isTerminal checks if the writer is a terminal
func isTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isTerminalFile(file)
}

// isTerminalFile checks if a file is a terminal
func isTerminalFile(f *os.File) bool {
	// Check if file is a terminal by attempting to get file info
	// and checking if it's a character device (terminal)
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	// On Unix systems, terminals are character devices
	// Mode & os.ModeCharDevice checks if it's a character device
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// colourize applies colour to a string if colourization is enabled
func (l *Logger) colourize(colour, text string) string {
	if !l.colourized {
		return text
	}
	return colour + text + colourReset
}

// formatMessage formats a log message with timestamp and level
func (l *Logger) formatMessage(level Level, msg string, fields ...interface{}) string {
	var parts []string

	// Timestamp (only in verbose mode)
	if l.level <= LevelDebug {
		timestamp := time.Now().Format("15:04:05")
		parts = append(parts, l.colourize(colourGray, timestamp))
	}

	// Level
	var levelcolour string
	var levelText string
	switch level {
	case LevelDebug:
		levelcolour = colourGray
		levelText = "DEBUG"
	case LevelInfo:
		levelcolour = colourBlue
		levelText = "INFO"
	case LevelWarn:
		levelcolour = colourYellow
		levelText = "WARN"
	case LevelError:
		levelcolour = colourRed
		levelText = "ERROR"
	}
	parts = append(parts, l.colourize(levelcolour, levelText))

	// Prefix
	if l.prefix != "" {
		parts = append(parts, l.colourize(colourCyan, "["+l.prefix+"]"))
	}

	// Message
	parts = append(parts, msg)

	// Fields
	if len(fields) > 0 {
		var fieldParts []string
		for i := 0; i < len(fields); i += 2 {
			if i+1 < len(fields) {
				key := fmt.Sprintf("%v", fields[i])
				value := fields[i+1]
				fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", key, value))
			}
		}
		if len(fieldParts) > 0 {
			parts = append(parts, l.colourize(colourGray, "("+strings.Join(fieldParts, ", ")+")"))
		}
	}

	return strings.Join(parts, " ")
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...interface{}) {
	if l.level <= LevelDebug {
		_, _ = fmt.Fprintln(l.output, l.formatMessage(LevelDebug, msg, fields...)) //nolint:errcheck // Writing to stderr rarely fails and is non-recoverable
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...interface{}) {
	if l.level <= LevelInfo {
		_, _ = fmt.Fprintln(l.output, l.formatMessage(LevelInfo, msg, fields...)) //nolint:errcheck // Writing to stderr rarely fails and is non-recoverable
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...interface{}) {
	if l.level <= LevelWarn {
		_, _ = fmt.Fprintln(l.output, l.formatMessage(LevelWarn, msg, fields...)) //nolint:errcheck // Writing to stderr rarely fails and is non-recoverable
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...interface{}) {
	if l.level <= LevelError {
		_, _ = fmt.Fprintln(l.output, l.formatMessage(LevelError, msg, fields...)) //nolint:errcheck // Writing to stderr rarely fails and is non-recoverable
	}
}

// Success logs a success message (info level with green colour)
func (l *Logger) Success(msg string, fields ...interface{}) {
	formatted := l.colourize(colourGreen, "✓ "+msg)
	if len(fields) > 0 {
		var fieldParts []string
		for i := 0; i < len(fields); i += 2 {
			if i+1 < len(fields) {
				key := fmt.Sprintf("%v", fields[i])
				value := fields[i+1]
				fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", key, value))
			}
		}
		if len(fieldParts) > 0 {
			formatted += " " + l.colourize(colourGray, "("+strings.Join(fieldParts, ", ")+")")
		}
	}
	_, _ = fmt.Fprintln(l.output, formatted) //nolint:errcheck // Writing to stderr rarely fails and is non-recoverable
}

// Printf formats and prints a message (no level prefix)
func (l *Logger) Printf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(l.output, format, args...) //nolint:errcheck // Writing to stderr rarely fails and is non-recoverable
}

// Println prints a message (no level prefix)
func (l *Logger) Println(args ...interface{}) {
	_, _ = fmt.Fprintln(l.output, args...) //nolint:errcheck // Writing to stderr rarely fails and is non-recoverable
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// Setcolourized sets whether output should be colourized
func (l *Logger) Setcolourized(colourized bool) {
	l.colourized = colourized && isTerminal(l.output)
}

// LevelFromString converts a string to a log level
func LevelFromString(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}
