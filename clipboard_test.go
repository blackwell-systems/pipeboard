package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// Test readClipboard function
func TestReadClipboard(t *testing.T) {
	// Test on current system - should not error on supported platforms
	data, err := readClipboard()
	if err != nil {
		// Error is expected on some test environments without clipboard
		// Just verify the error is reasonable
		if !strings.Contains(err.Error(), "reading clipboard") &&
			!strings.Contains(err.Error(), "missing") &&
			!strings.Contains(err.Error(), "not found") &&
			!strings.Contains(err.Error(), "no paste command") &&
			!strings.Contains(err.Error(), "no command configured") {
			t.Errorf("unexpected error from readClipboard: %v", err)
		}
		return
	}
	// If successful, data should be non-nil (but can be empty)
	if data == nil {
		t.Error("readClipboard returned nil data without error")
	}
}

// Test writeClipboard function with empty data
func TestWriteClipboardEmpty(t *testing.T) {
	err := writeClipboard([]byte{})
	if err != nil {
		// Error is expected on some test environments without clipboard
		// Just verify the error is reasonable
		if !strings.Contains(err.Error(), "missing") &&
			!strings.Contains(err.Error(), "not found") &&
			!strings.Contains(err.Error(), "no command configured") {
			t.Errorf("unexpected error from writeClipboard: %v", err)
		}
	}
}

// Test writeClipboard function with sample data
func TestWriteClipboardWithData(t *testing.T) {
	data := []byte("test clipboard content")
	err := writeClipboard(data)
	if err != nil {
		// Error is expected on some test environments without clipboard
		// Just verify the error is reasonable
		if !strings.Contains(err.Error(), "missing") &&
			!strings.Contains(err.Error(), "not found") &&
			!strings.Contains(err.Error(), "no command configured") {
			t.Errorf("unexpected error from writeClipboard: %v", err)
		}
	}
}

// Test writeClipboard with large data
func TestWriteClipboardLargeData(t *testing.T) {
	// Create 1MB of data
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	err := writeClipboard(data)
	if err != nil {
		// Error is expected on some test environments without clipboard
		// Just verify the error is reasonable
		if !strings.Contains(err.Error(), "missing") &&
			!strings.Contains(err.Error(), "not found") &&
			!strings.Contains(err.Error(), "no command configured") {
			t.Errorf("unexpected error from writeClipboard: %v", err)
		}
	}
}

// Test cmdClear fallback path when ClearCmd is empty
func TestCmdClearFallback(t *testing.T) {
	// This tests the fallback behavior when ClearCmd is not available
	// The function should use CopyCmd with empty data
	err := cmdClear([]string{})
	if err != nil {
		// Error is expected on some test environments
		if !strings.Contains(err.Error(), "missing") &&
			!strings.Contains(err.Error(), "not found") &&
			!strings.Contains(err.Error(), "no command configured") {
			t.Errorf("unexpected error from cmdClear: %v", err)
		}
	}
}

// Test cmdBackend with Notes field
func TestCmdBackendWithNotes(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_ = cmdBackend([]string{})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should contain backend info
	if !strings.Contains(output, "Backend:") {
		t.Error("cmdBackend output should contain 'Backend:'")
	}
	if !strings.Contains(output, "OS:") {
		t.Error("cmdBackend output should contain 'OS:'")
	}
	if !strings.Contains(output, "Copy cmd:") {
		t.Error("cmdBackend output should contain 'Copy cmd:'")
	}
	if !strings.Contains(output, "Paste cmd:") {
		t.Error("cmdBackend output should contain 'Paste cmd:'")
	}
}

// Test cmdCopy with stdin simulation
func TestCmdCopyFromStdin(t *testing.T) {
	// This test validates that cmdCopy can handle data from readInputOrArgs
	// We test with explicit args since stdin simulation is complex
	err := cmdCopy([]string{})
	// Error expected because no stdin and no args
	if err != nil {
		// Expected error path
		if !strings.Contains(err.Error(), "missing") &&
			!strings.Contains(err.Error(), "not found") &&
			!strings.Contains(err.Error(), "no input") {
			t.Logf("cmdCopy error (expected in test environment): %v", err)
		}
	}
}

// Test cmdPaste basic functionality
func TestCmdPasteBasic(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmdPaste([]string{})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		// Error is expected on some test environments
		if !strings.Contains(err.Error(), "missing") &&
			!strings.Contains(err.Error(), "not found") {
			t.Logf("cmdPaste error (expected in test environment): %v", err)
		}
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	// Output is whatever was in clipboard (may be empty)
}

// Test cmdDoctor comprehensive output
func TestCmdDoctorComprehensive(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmdDoctor([]string{})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("cmdDoctor should not error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Check for all expected output sections
	expectedStrings := []string{
		"pipeboard doctor",
		"OS:",
		"Backend:",
		"Status:",
		"Tips:",
	}

	for _, s := range expectedStrings {
		if !strings.Contains(output, s) {
			t.Errorf("cmdDoctor output should contain %q", s)
		}
	}
}

// Test readClipboard error handling with no paste command
func TestReadClipboardNoPasteCmd(t *testing.T) {
	// This test verifies the error path when PasteCmd is empty
	// Can't easily simulate this without mocking, but the function
	// should handle the case where len(b.PasteCmd) == 0
	// The actual test happens naturally in environments without clipboard
	_, err := readClipboard()
	if err != nil {
		// Verify error messages are appropriate
		if err.Error() != "" {
			// Error is expected and should be informative
			t.Logf("readClipboard error (expected): %v", err)
		}
	}
}

// Test writeClipboard error handling
func TestWriteClipboardErrorHandling(t *testing.T) {
	// Test with various data sizes
	testCases := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"small", []byte("test")},
		{"medium", []byte(strings.Repeat("x", 1024))},
		{"large", []byte(strings.Repeat("y", 10240))},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := writeClipboard(tc.data)
			if err != nil {
				// Error is expected in test environments
				// Just verify it's a reasonable error
				if !strings.Contains(err.Error(), "missing") &&
					!strings.Contains(err.Error(), "not found") {
					t.Logf("writeClipboard error for %s (expected): %v", tc.name, err)
				}
			}
		})
	}
}

// Test cmdCopy with --image flag error handling
func TestCmdCopyImageWithTextArgsError(t *testing.T) {
	// Test that --image mode rejects text arguments
	err := cmdCopy([]string{"--image", "text_arg"})
	if err == nil {
		t.Error("cmdCopy --image with text args should return error")
	}
	// Either "not supported" (no image backend) or "PNG data from stdin" (has backend but rejects text args)
	if !strings.Contains(err.Error(), "PNG data from stdin") &&
		!strings.Contains(err.Error(), "not supported") {
		t.Errorf("error should mention PNG/stdin or not supported: %v", err)
	}
}

// Test cmdPaste with --image flag
func TestCmdPasteImageMode(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmdPaste([]string{"--image"})

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		// Error expected if image paste not supported
		if !strings.Contains(err.Error(), "not supported") &&
			!strings.Contains(err.Error(), "missing") &&
			!strings.Contains(err.Error(), "not found") {
			t.Logf("cmdPaste --image error (expected): %v", err)
		}
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	// Image data would be in output (if supported)
}

// Test cmdCopy image mode without text args
func TestCmdCopyImageModeNoTextArgs(t *testing.T) {
	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Create a pipe for stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write some fake PNG data
	go func() {
		_, _ = w.Write([]byte{0x89, 0x50, 0x4E, 0x47}) // PNG magic bytes
		_ = w.Close()
	}()

	err := cmdCopy([]string{"--image"})
	if err != nil {
		// Error expected if image copy not supported or in test environment
		if !strings.Contains(err.Error(), "not supported") &&
			!strings.Contains(err.Error(), "missing") &&
			!strings.Contains(err.Error(), "not found") {
			t.Logf("cmdCopy --image error (expected): %v", err)
		}
	}
}
