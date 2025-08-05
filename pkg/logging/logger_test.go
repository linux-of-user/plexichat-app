package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, test := range tests {
		if got := test.level.String(); got != test.expected {
			t.Errorf("LogLevel.String() = %v, want %v", got, test.expected)
		}
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"DEBUG", DEBUG},
		{"debug", DEBUG},
		{"INFO", INFO},
		{"info", INFO},
		{"WARN", WARN},
		{"warn", WARN},
		{"WARNING", WARN},
		{"ERROR", ERROR},
		{"error", ERROR},
		{"FATAL", FATAL},
		{"fatal", FATAL},
		{"invalid", INFO}, // default
	}

	for _, test := range tests {
		if got := ParseLogLevel(test.input); got != test.expected {
			t.Errorf("ParseLogLevel(%q) = %v, want %v", test.input, got, test.expected)
		}
	}
}

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(DEBUG, &buf, false)

	if logger.level != DEBUG {
		t.Errorf("Expected level DEBUG, got %v", logger.level)
	}
	if logger.output != &buf {
		t.Errorf("Expected output to be set to buffer")
	}
	if logger.colorized {
		t.Errorf("Expected colorized to be false")
	}
}

func TestLogger_SetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(INFO, &buf, false)

	// Test that DEBUG messages are filtered out at INFO level
	logger.Debug("debug message")
	if buf.Len() > 0 {
		t.Errorf("DEBUG message should be filtered at INFO level")
	}

	// Test that INFO messages are logged at INFO level
	logger.Info("info message")
	if buf.Len() == 0 {
		t.Errorf("INFO message should be logged at INFO level")
	}

	// Reset buffer and change level to DEBUG
	buf.Reset()
	logger.SetLevel(DEBUG)

	// Test that DEBUG messages are now logged
	logger.Debug("debug message")
	if buf.Len() == 0 {
		t.Errorf("DEBUG message should be logged at DEBUG level")
	}
}

func TestLogger_LogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(DEBUG, &buf, false)

	tests := []struct {
		logFunc func(string, ...interface{})
		level   string
		message string
	}{
		{logger.Debug, "DEBUG", "debug message"},
		{logger.Info, "INFO", "info message"},
		{logger.Warn, "WARN", "warn message"},
		{logger.Error, "ERROR", "error message"},
	}

	for _, test := range tests {
		buf.Reset()
		test.logFunc(test.message)

		output := buf.String()
		if !strings.Contains(output, test.level) {
			t.Errorf("Expected log output to contain level %s, got: %s", test.level, output)
		}
		if !strings.Contains(output, test.message) {
			t.Errorf("Expected log output to contain message %s, got: %s", test.message, output)
		}
	}
}

func TestLogger_Prefix(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(INFO, &buf, false)
	logger.SetPrefix("TEST")

	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "[TEST]") {
		t.Errorf("Expected log output to contain prefix [TEST], got: %s", output)
	}
}

func TestToASCII(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"hello world", "hello world"},
		{"hello 世界", "hello ??"},
		{"café", "caf?"},
		{"", ""},
		{"123!@#", "123!@#"},
	}

	for _, test := range tests {
		if got := toASCII(test.input); got != test.expected {
			t.Errorf("toASCII(%q) = %q, want %q", test.input, got, test.expected)
		}
	}
}

func TestLogger_ASCIIOnly(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(INFO, &buf, false)

	// Test with non-ASCII characters
	logger.Info("Message with unicode: 世界")

	output := buf.String()
	// Should contain ASCII replacement characters
	if !strings.Contains(output, "??") {
		t.Errorf("Expected non-ASCII characters to be replaced with ?, got: %s", output)
	}
}

func TestLogger_Colorization(t *testing.T) {
	var buf bytes.Buffer

	// Test with colorization disabled
	logger := NewLogger(INFO, &buf, false)
	logger.Info("test message")
	outputNoColor := buf.String()

	// Test with colorization enabled
	buf.Reset()
	logger.SetColorized(true)
	logger.Info("test message")
	outputWithColor := buf.String()

	// In test environments, color might be disabled automatically
	// So we just check that both outputs contain the message
	if !strings.Contains(outputNoColor, "test message") {
		t.Errorf("Expected non-colorized output to contain message")
	}
	if !strings.Contains(outputWithColor, "test message") {
		t.Errorf("Expected colorized output to contain message")
	}
}

func TestGlobalLoggerFunctions(t *testing.T) {
	// Test that global functions don't panic
	SetGlobalLevel(DEBUG)
	SetGlobalColorized(false)
	SetGlobalPrefix("GLOBAL")

	// These should not panic
	Debug("debug test")
	Info("info test")
	Warn("warn test")
	Error("error test")
	// Note: We don't test Fatal() as it would exit the program
}
