package logging

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel parses a string into a LogLevel
func ParseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}

// Logger represents a configurable logger with ASCII-only output and colorization
type Logger struct {
	level      LogLevel
	output     io.Writer
	colorized  bool
	timeFormat string
	prefix     string
}

// NewLogger creates a new logger with the specified configuration
func NewLogger(level LogLevel, output io.Writer, colorized bool) *Logger {
	return &Logger{
		level:      level,
		output:     output,
		colorized:  colorized,
		timeFormat: "2006-01-02 15:04:05",
		prefix:     "",
	}
}

// NewDefaultLogger creates a logger with default settings
func NewDefaultLogger() *Logger {
	return NewLogger(INFO, os.Stdout, true)
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// SetOutput sets the output writer
func (l *Logger) SetOutput(output io.Writer) {
	l.output = output
}

// SetColorized enables or disables colorized output
func (l *Logger) SetColorized(colorized bool) {
	l.colorized = colorized
}

// SetTimeFormat sets the time format for log messages
func (l *Logger) SetTimeFormat(format string) {
	l.timeFormat = format
}

// SetPrefix sets a prefix for all log messages
func (l *Logger) SetPrefix(prefix string) {
	l.prefix = prefix
}

// log writes a log message with the specified level
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	// Ensure all content is ASCII-only
	message := fmt.Sprintf(format, args...)
	message = toASCII(message)

	// Format timestamp
	timestamp := time.Now().Format(l.timeFormat)

	// Build log line
	var logLine string
	if l.prefix != "" {
		logLine = fmt.Sprintf("[%s] [%s] %s: %s", timestamp, l.prefix, level.String(), message)
	} else {
		logLine = fmt.Sprintf("[%s] %s: %s", timestamp, level.String(), message)
	}

	// Apply colorization if enabled
	if l.colorized {
		logLine = l.colorizeLogLine(level, logLine)
	}

	// Write to output
	fmt.Fprintln(l.output, logLine)

	// Exit on fatal errors
	if level == FATAL {
		os.Exit(1)
	}
}

// colorizeLogLine applies color to the log line based on level
func (l *Logger) colorizeLogLine(level LogLevel, line string) string {
	switch level {
	case DEBUG:
		return color.HiBlackString(line)
	case INFO:
		return color.CyanString(line)
	case WARN:
		return color.YellowString(line)
	case ERROR:
		return color.RedString(line)
	case FATAL:
		return color.New(color.FgRed, color.Bold).Sprint(line)
	default:
		return line
	}
}

// toASCII converts a string to ASCII-only characters
func toASCII(s string) string {
	var result strings.Builder
	for _, r := range s {
		if r <= 127 {
			result.WriteRune(r)
		} else {
			// Replace non-ASCII characters with '?'
			result.WriteRune('?')
		}
	}
	return result.String()
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Fatal logs a fatal message and exits the program
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
}

// Global logger instance
var defaultLogger = NewDefaultLogger()

// SetGlobalLevel sets the global logger level
func SetGlobalLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}

// SetGlobalColorized sets the global logger colorization
func SetGlobalColorized(colorized bool) {
	defaultLogger.SetColorized(colorized)
}

// SetGlobalPrefix sets the global logger prefix
func SetGlobalPrefix(prefix string) {
	defaultLogger.SetPrefix(prefix)
}

// Global logging functions
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}
