package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

// Test debugLog when debug mode is disabled
func TestDebugLogDisabled(t *testing.T) {
	origDebug := debugMode
	defer func() { debugMode = origDebug }()

	debugMode = false
	// Should not panic or produce output
	debugLog("test message %s", "arg")
}

// Test debugLog when debug mode is enabled
func TestDebugLogEnabled(t *testing.T) {
	origDebug := debugMode
	origNoColor := os.Getenv("NO_COLOR")
	defer func() {
		debugMode = origDebug
		if origNoColor != "" {
			_ = os.Setenv("NO_COLOR", origNoColor)
		} else {
			_ = os.Unsetenv("NO_COLOR")
		}
	}()

	debugMode = true
	_ = os.Setenv("NO_COLOR", "1") // Disable color for predictable output

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	debugLog("test message %s", "value")

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "[debug]") {
		t.Errorf("debugLog output should contain [debug], got: %s", output)
	}
	if !strings.Contains(output, "test message value") {
		t.Errorf("debugLog output should contain message, got: %s", output)
	}
}

// Test debugLog with color enabled
func TestDebugLogWithColor(t *testing.T) {
	origDebug := debugMode
	origNoColor := os.Getenv("NO_COLOR")
	origTerm := os.Getenv("TERM")
	defer func() {
		debugMode = origDebug
		if origNoColor != "" {
			_ = os.Setenv("NO_COLOR", origNoColor)
		} else {
			_ = os.Unsetenv("NO_COLOR")
		}
		if origTerm != "" {
			_ = os.Setenv("TERM", origTerm)
		} else {
			_ = os.Unsetenv("TERM")
		}
	}()

	debugMode = true
	_ = os.Unsetenv("NO_COLOR")
	_ = os.Setenv("TERM", "xterm-256color")

	// Just verify it doesn't panic - color detection depends on terminal
	debugLog("test with color %d", 42)
}

// Test printInfo when quiet mode is enabled
func TestPrintInfoQuietMode(t *testing.T) {
	origQuiet := quietMode
	defer func() { quietMode = origQuiet }()

	quietMode = true

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printInfo("test message %s", "value")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if output != "" {
		t.Errorf("printInfo should produce no output in quiet mode, got: %s", output)
	}
}

// Test printInfo when quiet mode is disabled
func TestPrintInfoNormalMode(t *testing.T) {
	origQuiet := quietMode
	defer func() { quietMode = origQuiet }()

	quietMode = false

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printInfo("test message %s", "value")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "test message value") {
		t.Errorf("printInfo should produce output, got: %s", output)
	}
}

// Test printError without color
func TestPrintErrorNoColor(t *testing.T) {
	origNoColor := os.Getenv("NO_COLOR")
	defer func() {
		if origNoColor != "" {
			_ = os.Setenv("NO_COLOR", origNoColor)
		} else {
			_ = os.Unsetenv("NO_COLOR")
		}
	}()

	_ = os.Setenv("NO_COLOR", "1")

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	printError(errors.New("test error"))

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "pipeboard: test error") {
		t.Errorf("printError should contain error message, got: %s", output)
	}
	if strings.Contains(output, "\033[") {
		t.Errorf("printError should not contain ANSI codes when NO_COLOR is set")
	}
}

// Test stdinHasData (basic test - can't fully simulate pipe)
func TestStdinHasDataNotPipe(t *testing.T) {
	// When running tests, stdin is typically not a pipe
	// This tests the "not a pipe" path
	result := stdinHasData()
	// In test environment, stdin is usually not a pipe
	// Just verify it doesn't panic and returns a boolean
	_ = result
}

// Test printCommandHelp for known commands
func TestPrintCommandHelpKnown(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printCommandHelp("copy")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Usage:") {
		t.Error("printCommandHelp should show usage for known command")
	}
	if !strings.Contains(output, "copy") {
		t.Error("printCommandHelp should mention the command")
	}
}

// Test commandHelp map has expected commands
func TestCommandHelpMapContents(t *testing.T) {
	expectedCommands := []string{
		"copy", "paste", "clear", "backend", "doctor",
		"push", "pull", "show", "slots", "rm",
		"send", "recv", "peek", "watch",
		"history", "fx", "init", "completion", "recall",
	}

	for _, cmd := range expectedCommands {
		if _, ok := commandHelp[cmd]; !ok {
			t.Errorf("commandHelp should contain help for %q", cmd)
		}
	}
}

// Test useColor with empty NO_COLOR (not set vs empty string)
func TestUseColorEmptyNoColor(t *testing.T) {
	origNoColor := os.Getenv("NO_COLOR")
	defer func() {
		if origNoColor != "" {
			_ = os.Setenv("NO_COLOR", origNoColor)
		} else {
			_ = os.Unsetenv("NO_COLOR")
		}
	}()

	// Empty string should still disable color per no-color.org spec
	_ = os.Setenv("NO_COLOR", "")
	// Should return false because NO_COLOR is set (even if empty)
	// Actually, Go's os.Getenv returns "" for both unset and empty
	// So we need to check using os.LookupEnv behavior
	// For this test, just verify behavior when explicitly set
}

// Test printInfo with format specifiers
func TestPrintInfoFormatSpecifiers(t *testing.T) {
	origQuiet := quietMode
	defer func() { quietMode = origQuiet }()

	quietMode = false

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printInfo("int: %d, string: %s, float: %.2f", 42, "hello", 3.14)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "42") {
		t.Error("printInfo should format integer")
	}
	if !strings.Contains(output, "hello") {
		t.Error("printInfo should format string")
	}
	if !strings.Contains(output, "3.14") {
		t.Error("printInfo should format float")
	}
}

// Test debugLog with format specifiers
func TestDebugLogFormatSpecifiers(t *testing.T) {
	origDebug := debugMode
	origNoColor := os.Getenv("NO_COLOR")
	defer func() {
		debugMode = origDebug
		if origNoColor != "" {
			_ = os.Setenv("NO_COLOR", origNoColor)
		} else {
			_ = os.Unsetenv("NO_COLOR")
		}
	}()

	debugMode = true
	_ = os.Setenv("NO_COLOR", "1")

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	debugLog("values: %d, %s, %v", 123, "test", true)

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "123") {
		t.Error("debugLog should format integer")
	}
	if !strings.Contains(output, "test") {
		t.Error("debugLog should format string")
	}
	if !strings.Contains(output, "true") {
		t.Error("debugLog should format boolean")
	}
}

// Test printError with wrapped error
func TestPrintErrorWrappedError(t *testing.T) {
	origNoColor := os.Getenv("NO_COLOR")
	defer func() {
		if origNoColor != "" {
			_ = os.Setenv("NO_COLOR", origNoColor)
		} else {
			_ = os.Unsetenv("NO_COLOR")
		}
	}()

	_ = os.Setenv("NO_COLOR", "1")

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	innerErr := errors.New("inner error")
	wrappedErr := errors.New("outer: " + innerErr.Error())
	printError(wrappedErr)

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "outer") {
		t.Error("printError should show outer error")
	}
	if !strings.Contains(output, "inner") {
		t.Error("printError should show inner error")
	}
}
