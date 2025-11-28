// Package logger provides structured logging with color support for Kilt.
// It respects the --no-color flag and supports different log levels (debug, info, warn, error).
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

// Colors for terminal output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// Logger provides structured logging with color support
type Logger struct {
	level     Level
	output    io.Writer
	colorized bool
	prefix    string
}

// NewLogger creates a new logger instance
func NewLogger(level Level, colorized bool) *Logger {
	return &Logger{
		level:     level,
		output:    os.Stderr,
		colorized: colorized && isTerminal(os.Stderr),
		prefix:    "",
	}
}

// WithPrefix returns a new logger with a prefix
func (l *Logger) WithPrefix(prefix string) *Logger {
	return &Logger{
		level:     l.level,
		output:    l.output,
		colorized: l.colorized,
		prefix:    prefix,
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

// colorize applies color to a string if colorization is enabled
func (l *Logger) colorize(color, text string) string {
	if !l.colorized {
		return text
	}
	return color + text + colorReset
}

// formatMessage formats a log message with timestamp and level
func (l *Logger) formatMessage(level Level, msg string, fields ...interface{}) string {
	var parts []string

	// Timestamp (only in verbose mode)
	if l.level <= LevelDebug {
		timestamp := time.Now().Format("15:04:05")
		parts = append(parts, l.colorize(colorGray, timestamp))
	}

	// Level
	var levelColor string
	var levelText string
	switch level {
	case LevelDebug:
		levelColor = colorGray
		levelText = "DEBUG"
	case LevelInfo:
		levelColor = colorBlue
		levelText = "INFO"
	case LevelWarn:
		levelColor = colorYellow
		levelText = "WARN"
	case LevelError:
		levelColor = colorRed
		levelText = "ERROR"
	}
	parts = append(parts, l.colorize(levelColor, levelText))

	// Prefix
	if l.prefix != "" {
		parts = append(parts, l.colorize(colorCyan, "["+l.prefix+"]"))
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
			parts = append(parts, l.colorize(colorGray, "("+strings.Join(fieldParts, ", ")+")"))
		}
	}

	return strings.Join(parts, " ")
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...interface{}) {
	if l.level <= LevelDebug {
		fmt.Fprintln(l.output, l.formatMessage(LevelDebug, msg, fields...))
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...interface{}) {
	if l.level <= LevelInfo {
		fmt.Fprintln(l.output, l.formatMessage(LevelInfo, msg, fields...))
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...interface{}) {
	if l.level <= LevelWarn {
		fmt.Fprintln(l.output, l.formatMessage(LevelWarn, msg, fields...))
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...interface{}) {
	if l.level <= LevelError {
		fmt.Fprintln(l.output, l.formatMessage(LevelError, msg, fields...))
	}
}

// Success logs a success message (info level with green color)
func (l *Logger) Success(msg string, fields ...interface{}) {
	formatted := l.colorize(colorGreen, "✓ "+msg)
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
			formatted += " " + l.colorize(colorGray, "("+strings.Join(fieldParts, ", ")+")")
		}
	}
	fmt.Fprintln(l.output, formatted)
}

// Printf formats and prints a message (no level prefix)
func (l *Logger) Printf(format string, args ...interface{}) {
	fmt.Fprintf(l.output, format, args...)
}

// Println prints a message (no level prefix)
func (l *Logger) Println(args ...interface{}) {
	fmt.Fprintln(l.output, args...)
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// SetColorized sets whether output should be colorized
func (l *Logger) SetColorized(colorized bool) {
	l.colorized = colorized && isTerminal(l.output)
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

