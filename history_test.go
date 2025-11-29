package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// Tests for new features: deduplication, max entries, search, encryption

func TestRecordClipboardHistoryDeduplication(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Record same content twice
	recordClipboardHistory([]byte("duplicate"))
	recordClipboardHistory([]byte("duplicate"))

	path := getClipboardHistoryPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read clipboard history: %v", err)
	}

	var history []ClipboardHistoryEntry
	if err := json.Unmarshal(data, &history); err != nil {
		t.Fatalf("failed to parse clipboard history: %v", err)
	}

	// Should only have 1 entry due to deduplication
	if len(history) != 1 {
		t.Errorf("expected 1 entry after deduplication, got %d", len(history))
	}
}

func TestRecordClipboardHistoryMaxEntries(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Record more than maxClipboardHistory entries
	for i := 0; i < maxClipboardHistory+5; i++ {
		recordClipboardHistory([]byte(string(rune('a' + i))))
	}

	path := getClipboardHistoryPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read clipboard history: %v", err)
	}

	var history []ClipboardHistoryEntry
	if err := json.Unmarshal(data, &history); err != nil {
		t.Fatalf("failed to parse clipboard history: %v", err)
	}

	if len(history) != maxClipboardHistory {
		t.Errorf("expected %d entries (max), got %d", maxClipboardHistory, len(history))
	}
}

func TestClipboardHistoryEntryStruct(t *testing.T) {
	entry := ClipboardHistoryEntry{
		Timestamp: time.Now(),
		Hash:      "abc123",
		Preview:   "test preview",
		Size:      100,
		Content:   []byte("test content"),
		Encrypted: true,
	}

	if entry.Hash != "abc123" {
		t.Error("Hash field not set correctly")
	}
	if !entry.Encrypted {
		t.Error("Encrypted field should be true")
	}
}

func TestHistoryEntryStruct(t *testing.T) {
	entry := HistoryEntry{
		Timestamp: time.Now(),
		Command:   "copy",
		Target:    "clipboard",
		Size:      50,
	}

	if entry.Command != "copy" {
		t.Error("Command field not set correctly")
	}
	if entry.Size != 50 {
		t.Error("Size field not set correctly")
	}
}

func TestRecordHistoryMaxEntries(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Record more than maxHistoryEntries
	for i := 0; i < maxHistoryEntries+10; i++ {
		recordHistory("cmd", "target", int64(i))
	}

	path := getHistoryPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read history: %v", err)
	}

	var history []HistoryEntry
	if err := json.Unmarshal(data, &history); err != nil {
		t.Fatalf("failed to parse history: %v", err)
	}

	if len(history) != maxHistoryEntries {
		t.Errorf("expected %d entries (max), got %d", maxHistoryEntries, len(history))
	}
}

func TestClipboardHistoryPreviewGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Record content with newlines
	content := "line1\nline2\nline3"
	recordClipboardHistory([]byte(content))

	path := getClipboardHistoryPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read clipboard history: %v", err)
	}

	var history []ClipboardHistoryEntry
	if err := json.Unmarshal(data, &history); err != nil {
		t.Fatalf("failed to parse clipboard history: %v", err)
	}

	// Preview should have newlines escaped
	if strings.Contains(history[0].Preview, "\n") {
		t.Error("preview should have newlines escaped")
	}
	if !strings.Contains(history[0].Preview, "\\n") {
		t.Error("preview should contain escaped newlines")
	}
}

func TestClipboardHistoryLongContent(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Record very long content
	longContent := strings.Repeat("a", 500)
	recordClipboardHistory([]byte(longContent))

	path := getClipboardHistoryPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read clipboard history: %v", err)
	}

	var history []ClipboardHistoryEntry
	if err := json.Unmarshal(data, &history); err != nil {
		t.Fatalf("failed to parse clipboard history: %v", err)
	}

	// Preview should be truncated
	if len(history[0].Preview) > previewLength+10 { // +10 for "..." and potential encoding
		t.Errorf("preview too long: %d chars", len(history[0].Preview))
	}

	// Full content should be preserved
	if len(history[0].Content) != 500 {
		t.Errorf("full content should be preserved: got %d bytes", len(history[0].Content))
	}
}

// Test showClipboardHistory with empty history
func TestShowClipboardHistoryEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create empty history file
	historyPath := getClipboardHistoryPath()
	_ = os.MkdirAll(tmpDir+"/pipeboard", 0755)
	_ = os.WriteFile(historyPath, []byte("[]"), 0600)

	err := showClipboardHistory(false, "")
	if err != nil {
		t.Errorf("showClipboardHistory should not error on empty history: %v", err)
	}
}

// Test showClipboardHistory JSON output
func TestShowClipboardHistoryJSON(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Record some content
	recordClipboardHistory([]byte("test content"))

	err := showClipboardHistory(true, "")
	if err != nil {
		t.Errorf("showClipboardHistory with JSON should not error: %v", err)
	}
}

// Test showClipboardHistory with search
func TestShowClipboardHistorySearch(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Record some content
	recordClipboardHistory([]byte("hello world"))
	recordClipboardHistory([]byte("foo bar"))
	recordClipboardHistory([]byte("hello again"))

	// Search for "hello"
	err := showClipboardHistory(false, "hello")
	if err != nil {
		t.Errorf("showClipboardHistory with search should not error: %v", err)
	}
}

// Test showClipboardHistory search with no matches
func TestShowClipboardHistorySearchNoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Record some content
	recordClipboardHistory([]byte("hello world"))

	// Search for something not present
	err := showClipboardHistory(false, "xyz123notfound")
	if err != nil {
		t.Errorf("showClipboardHistory with no match should not error: %v", err)
	}
}

// Test cmdRecall with index out of range
func TestCmdRecallOutOfRange(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Record one entry
	recordClipboardHistory([]byte("test"))

	// Try to recall index 100
	err := cmdRecall([]string{"100"})
	if err == nil {
		t.Error("cmdRecall should error on out of range index")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("error should mention out of range: %v", err)
	}
}

// Test cmdHistory with search flag
func TestCmdHistorySearchFlag(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Record some content
	recordClipboardHistory([]byte("searchable content"))

	err := cmdHistory([]string{"--local", "--search", "searchable"})
	if err != nil {
		t.Errorf("cmdHistory with --search should not error: %v", err)
	}
}

// Test cmdHistory with --search= syntax
func TestCmdHistorySearchEqualsSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	recordClipboardHistory([]byte("test data"))

	err := cmdHistory([]string{"--local", "--search=test"})
	if err != nil {
		t.Errorf("cmdHistory with --search= should not error: %v", err)
	}
}

// Test cmdHistory with -s short flag
func TestCmdHistorySearchShortFlag(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	recordClipboardHistory([]byte("short flag test"))

	err := cmdHistory([]string{"--local", "-s", "short"})
	if err != nil {
		t.Errorf("cmdHistory with -s should not error: %v", err)
	}
}

// Test cmdHistory --search without query
func TestCmdHistorySearchMissingQuery(t *testing.T) {
	err := cmdHistory([]string{"--local", "--search"})
	if err == nil {
		t.Error("cmdHistory should error when --search has no query")
	}
	if !strings.Contains(err.Error(), "requires a query") {
		t.Errorf("error should mention requires query: %v", err)
	}
}

// Test cmdHistory with JSON output
func TestCmdHistoryJSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	recordHistory("push", "slot1", 100)

	err := cmdHistory([]string{"--json"})
	if err != nil {
		t.Errorf("cmdHistory with --json should not error: %v", err)
	}
}

// Test cmdHistory --local with JSON and search
func TestCmdHistoryLocalJSONWithSearch(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	recordClipboardHistory([]byte("json search test"))

	err := cmdHistory([]string{"--local", "--json", "--search", "json"})
	if err != nil {
		t.Errorf("cmdHistory --local --json --search should not error: %v", err)
	}
}

// Test showClipboardHistory when file doesn't exist
func TestShowClipboardHistoryNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Don't create the file
	err := showClipboardHistory(false, "")
	if err != nil {
		t.Errorf("showClipboardHistory should not error when file doesn't exist: %v", err)
	}
}

// Test showClipboardHistory JSON when file doesn't exist
func TestShowClipboardHistoryNoFileJSON(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Don't create the file
	err := showClipboardHistory(true, "")
	if err != nil {
		t.Errorf("showClipboardHistory JSON should not error when file doesn't exist: %v", err)
	}
}

// Test cmdRecall when history file doesn't exist
func TestCmdRecallNoHistoryFile(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	err := cmdRecall([]string{"1"})
	if err == nil {
		t.Error("cmdRecall should error when history file doesn't exist")
	}
	if !strings.Contains(err.Error(), "no clipboard history") {
		t.Errorf("error should mention no history: %v", err)
	}
}

// Test cmdRecall with empty history
func TestCmdRecallEmptyHistory(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create empty history
	historyPath := getClipboardHistoryPath()
	_ = os.MkdirAll(tmpDir+"/pipeboard", 0755)
	_ = os.WriteFile(historyPath, []byte("[]"), 0600)

	err := cmdRecall([]string{"1"})
	if err == nil {
		t.Error("cmdRecall should error when history is empty")
	}
	if !strings.Contains(err.Error(), "no clipboard history") {
		t.Errorf("error should mention no history: %v", err)
	}
}
