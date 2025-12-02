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

	// Record more than defaultClipboardHistoryLimit entries
	for i := 0; i < defaultClipboardHistoryLimit+5; i++ {
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

	if len(history) != defaultClipboardHistoryLimit {
		t.Errorf("expected %d entries (max), got %d", defaultClipboardHistoryLimit, len(history))
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

// Test getHistoryEncryptionConfig with encryption enabled
func TestGetHistoryEncryptionConfigEnabled(t *testing.T) {
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

	// Create config with encryption enabled
	configDir := tmpDir + "/pipeboard"
	_ = os.MkdirAll(configDir, 0755)
	configContent := `version: 1
sync:
  backend: local
  encryption: aes256
  passphrase: testsecret
`
	_ = os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0600)

	enabled, passphrase := getHistoryEncryptionConfig()
	if !enabled {
		t.Error("encryption should be enabled")
	}
	if passphrase != "testsecret" {
		t.Errorf("passphrase should be 'testsecret', got %q", passphrase)
	}
}

// Test getHistoryEncryptionConfig with encryption disabled
func TestGetHistoryEncryptionConfigDisabled(t *testing.T) {
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

	// Create config without encryption
	configDir := tmpDir + "/pipeboard"
	_ = os.MkdirAll(configDir, 0755)
	configContent := `version: 1
sync:
  backend: local
`
	_ = os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0600)

	enabled, _ := getHistoryEncryptionConfig()
	if enabled {
		t.Error("encryption should be disabled")
	}
}

// Test getHistoryEncryptionConfig with no config file
func TestGetHistoryEncryptionConfigNoFile(t *testing.T) {
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

	// No config file exists
	enabled, _ := getHistoryEncryptionConfig()
	if enabled {
		t.Error("encryption should be disabled when no config exists")
	}
}

// Test cmdRecall with invalid (non-numeric) index
func TestCmdRecallInvalidIndexNonNumeric(t *testing.T) {
	err := cmdRecall([]string{"abc"})
	if err == nil {
		t.Error("cmdRecall should error on non-numeric index")
	}
	if !strings.Contains(err.Error(), "invalid index") {
		t.Errorf("error should mention invalid index: %v", err)
	}
}

// Test cmdRecall with no arguments
func TestCmdRecallNoArguments(t *testing.T) {
	err := cmdRecall([]string{})
	if err == nil {
		t.Error("cmdRecall should error when no args provided")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error should show usage: %v", err)
	}
}

// Test cmdRecall with zero index
func TestCmdRecallZeroIndexValue(t *testing.T) {
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

	err := cmdRecall([]string{"0"})
	if err == nil {
		t.Error("cmdRecall should error on zero index")
	}
	if !strings.Contains(err.Error(), ">= 1") {
		t.Errorf("error should mention index must be >= 1: %v", err)
	}
}

// Test cmdHistory with --fx filter
func TestCmdHistoryFxFilter(t *testing.T) {
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

	// Record some history entries
	recordHistory("fx:pretty-json", "", 100)
	recordHistory("push", "slot1", 200)

	err := cmdHistory([]string{"--fx"})
	if err != nil {
		t.Errorf("cmdHistory --fx should not error: %v", err)
	}
}

// Test cmdHistory with --slots filter
func TestCmdHistorySlotsFilter(t *testing.T) {
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
	recordHistory("pull", "slot2", 200)

	err := cmdHistory([]string{"--slots"})
	if err != nil {
		t.Errorf("cmdHistory --slots should not error: %v", err)
	}
}

// Test cmdHistory with --peer filter
func TestCmdHistoryPeerFilter(t *testing.T) {
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

	recordHistory("send", "dev", 100)
	recordHistory("recv", "mac", 200)

	err := cmdHistory([]string{"--peer"})
	if err != nil {
		t.Errorf("cmdHistory --peer should not error: %v", err)
	}
}

// Test cmdHistory with unknown flag
func TestCmdHistoryUnknownFlag(t *testing.T) {
	err := cmdHistory([]string{"--unknown"})
	if err == nil {
		t.Error("cmdHistory should error on unknown flag")
	}
	if !strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("error should mention unknown flag: %v", err)
	}
}

// Test cmdHistory with -s= short flag syntax
func TestCmdHistorySearchShortEqualsSyntax(t *testing.T) {
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

	recordClipboardHistory([]byte("equals syntax test"))

	err := cmdHistory([]string{"--local", "-s=equals"})
	if err != nil {
		t.Errorf("cmdHistory with -s= should not error: %v", err)
	}
}

// Test cmdHistory with empty history but JSON output
func TestCmdHistoryEmptyJSON(t *testing.T) {
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
	historyPath := getHistoryPath()
	_ = os.MkdirAll(tmpDir+"/pipeboard", 0755)
	_ = os.WriteFile(historyPath, []byte("[]"), 0600)

	err := cmdHistory([]string{"--json"})
	if err != nil {
		t.Errorf("cmdHistory --json with empty history should not error: %v", err)
	}
}

// Test cmdHistory with filter that matches nothing
func TestCmdHistoryFilterNoMatches(t *testing.T) {
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

	// Record some history
	recordHistory("copy", "", 100)

	// Filter for fx (which we didn't record)
	err := cmdHistory([]string{"--fx"})
	if err != nil {
		t.Errorf("cmdHistory --fx with no matches should not error: %v", err)
	}
}

// Test getHistoryPath with XDG_CONFIG_HOME unset
func TestGetHistoryPathDefaultHome(t *testing.T) {
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Unsetenv("XDG_CONFIG_HOME")

	path := getHistoryPath()
	if path == "" {
		t.Error("getHistoryPath should return a valid path when HOME is set")
	}
	if !strings.Contains(path, ".config/pipeboard/history.json") {
		t.Errorf("getHistoryPath should use default config path, got %q", path)
	}
}

// Test getClipboardHistoryPath with XDG_CONFIG_HOME unset
func TestGetClipboardHistoryPathDefaultHome(t *testing.T) {
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Unsetenv("XDG_CONFIG_HOME")

	path := getClipboardHistoryPath()
	if path == "" {
		t.Error("getClipboardHistoryPath should return a valid path when HOME is set")
	}
	if !strings.Contains(path, ".config/pipeboard/clipboard_history.json") {
		t.Errorf("getClipboardHistoryPath should use default config path, got %q", path)
	}
}

// Test cmdRecall with encrypted entry but no passphrase configured
func TestCmdRecallEncryptedNoPassphrase(t *testing.T) {
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

	// Create a config without encryption
	configDir := tmpDir + "/pipeboard"
	_ = os.MkdirAll(configDir, 0755)
	configContent := `version: 1
sync:
  backend: local
`
	_ = os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0600)

	// Create history with an encrypted entry
	historyContent := `[{
		"timestamp": "2025-01-01T00:00:00Z",
		"hash": "abc123",
		"preview": "encrypted preview",
		"size": 100,
		"content": "ZW5jcnlwdGVkIGRhdGE=",
		"encrypted": true
	}]`
	_ = os.WriteFile(configDir+"/clipboard_history.json", []byte(historyContent), 0600)

	err := cmdRecall([]string{"1"})
	if err == nil {
		t.Error("cmdRecall should error when entry is encrypted but no passphrase")
	}
	if !strings.Contains(err.Error(), "encrypted") || !strings.Contains(err.Error(), "passphrase") {
		t.Errorf("error should mention encrypted and passphrase: %v", err)
	}
}

// Test cmdRecall with invalid JSON history
func TestCmdRecallInvalidJSON(t *testing.T) {
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

	// Create invalid history file
	historyPath := getClipboardHistoryPath()
	_ = os.MkdirAll(tmpDir+"/pipeboard", 0755)
	_ = os.WriteFile(historyPath, []byte("invalid json {"), 0600)

	err := cmdRecall([]string{"1"})
	if err == nil {
		t.Error("cmdRecall should error on invalid JSON")
	}
}

// Test cmdHistory with invalid JSON history
func TestCmdHistoryInvalidJSON(t *testing.T) {
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

	// Create invalid history file
	historyPath := getHistoryPath()
	_ = os.MkdirAll(tmpDir+"/pipeboard", 0755)
	_ = os.WriteFile(historyPath, []byte("invalid json {"), 0600)

	err := cmdHistory([]string{})
	if err == nil {
		t.Error("cmdHistory should error on invalid JSON")
	}
}

// Test cmdHistory --local with invalid JSON
func TestCmdHistoryLocalInvalidJSON(t *testing.T) {
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

	// Create invalid clipboard history file
	historyPath := getClipboardHistoryPath()
	_ = os.MkdirAll(tmpDir+"/pipeboard", 0755)
	_ = os.WriteFile(historyPath, []byte("not valid json"), 0600)

	err := cmdHistory([]string{"--local"})
	if err == nil {
		t.Error("cmdHistory --local should error on invalid JSON")
	}
}

// Test showClipboardHistory JSON with search results
func TestShowClipboardHistoryJSONSearch(t *testing.T) {
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

	// Search with JSON output
	err := showClipboardHistory(true, "hello")
	if err != nil {
		t.Errorf("showClipboardHistory JSON with search should not error: %v", err)
	}
}

// Test showClipboardHistory JSON with no search results
func TestShowClipboardHistoryJSONSearchNoMatch(t *testing.T) {
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

	recordClipboardHistory([]byte("hello world"))

	// Search for non-existent content with JSON
	err := showClipboardHistory(true, "notfound")
	if err != nil {
		t.Errorf("showClipboardHistory JSON with no match should not error: %v", err)
	}
}

// Test cmdHistory with size display
func TestCmdHistoryWithSize(t *testing.T) {
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

	// Record with size
	recordHistory("push", "slot1", 1024)
	recordHistory("pull", "slot2", 0) // Zero size

	err := cmdHistory([]string{})
	if err != nil {
		t.Errorf("cmdHistory should not error: %v", err)
	}
}

// Test cmdHistory --slots --json combination
func TestCmdHistorySlotsJSON(t *testing.T) {
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

	err := cmdHistory([]string{"--slots", "--json"})
	if err != nil {
		t.Errorf("cmdHistory --slots --json should not error: %v", err)
	}
}

// Test cmdHistory with no matching filters JSON output
func TestCmdHistoryFilterNoMatchesJSON(t *testing.T) {
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

	// Record some history that won't match --fx filter
	recordHistory("push", "slot1", 100)

	err := cmdHistory([]string{"--fx", "--json"})
	if err != nil {
		t.Errorf("cmdHistory --fx --json with no matches should not error: %v", err)
	}
}

// Test truncateString with exact maxLen
func TestTruncateStringExactMaxLen(t *testing.T) {
	s := "exactly10!"
	result := truncateString(s, 10)
	if result != s {
		t.Errorf("string of exact max length should not be truncated: got %q", result)
	}
}

// Test truncateString with string shorter than maxLen
func TestTruncateStringShorterThanMax(t *testing.T) {
	s := "short"
	result := truncateString(s, 20)
	if result != s {
		t.Errorf("short string should not be truncated: got %q", result)
	}
}

// Test truncateString with string longer than maxLen
func TestTruncateStringLongerThanMax(t *testing.T) {
	s := "this is a very long string that exceeds the maximum length"
	result := truncateString(s, 20)
	if len(result) != 20 {
		t.Errorf("truncated string should be exactly maxLen: got %d chars", len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Error("truncated string should end with '...'")
	}
}

// Test isSlotCommand with all slot commands
func TestIsSlotCommandAllCommands(t *testing.T) {
	slotCommands := []string{"push", "pull", "show", "rm"}
	for _, cmd := range slotCommands {
		if !isSlotCommand(cmd) {
			t.Errorf("isSlotCommand should return true for %q", cmd)
		}
	}
}

// Test isSlotCommand with non-slot commands
func TestIsSlotCommandNonSlotCommands(t *testing.T) {
	nonSlotCommands := []string{"send", "recv", "copy", "paste", "fx:test"}
	for _, cmd := range nonSlotCommands {
		if isSlotCommand(cmd) {
			t.Errorf("isSlotCommand should return false for %q", cmd)
		}
	}
}

// Test isPeerCommand with all peer commands
func TestIsPeerCommandAllCommands(t *testing.T) {
	peerCommands := []string{"send", "recv", "peek", "watch:send", "watch:recv"}
	for _, cmd := range peerCommands {
		if !isPeerCommand(cmd) {
			t.Errorf("isPeerCommand should return true for %q", cmd)
		}
	}
}

// Test isPeerCommand with non-peer commands
func TestIsPeerCommandNonPeerCommands(t *testing.T) {
	nonPeerCommands := []string{"push", "pull", "copy", "paste", "fx:test"}
	for _, cmd := range nonPeerCommands {
		if isPeerCommand(cmd) {
			t.Errorf("isPeerCommand should return false for %q", cmd)
		}
	}
}

// Test getClipboardHistoryLimit with custom limit in config
func TestGetClipboardHistoryLimitCustom(t *testing.T) {
	// Clean state - unset any existing config env vars
	origConfig := os.Getenv("PIPEBOARD_CONFIG")
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		restoreEnv("PIPEBOARD_CONFIG", origConfig)
		restoreEnv("XDG_CONFIG_HOME", origXDG)
	}()

	// Ensure clean start
	_ = os.Unsetenv("PIPEBOARD_CONFIG")
	_ = os.Unsetenv("XDG_CONFIG_HOME")

	tmpDir := t.TempDir()
	configFile := tmpDir + "/config.yaml"
	configContent := `version: 1
sync:
  backend: local
history:
  limit: 50
`
	if err := os.WriteFile(configFile, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	limit := getClipboardHistoryLimit()
	if limit != 50 {
		t.Errorf("expected limit 50, got %d", limit)
	}
}

// Test getClipboardHistoryLimit with zero limit (should use default)
func TestGetClipboardHistoryLimitZero(t *testing.T) {
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

	// Create config with zero limit
	configDir := tmpDir + "/pipeboard"
	_ = os.MkdirAll(configDir, 0755)
	configContent := `version: 1
history:
  limit: 0
`
	_ = os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0600)

	limit := getClipboardHistoryLimit()
	if limit != defaultClipboardHistoryLimit {
		t.Errorf("zero limit should use default %d, got %d", defaultClipboardHistoryLimit, limit)
	}
}

// Test getClipboardHistoryLimit with negative limit (should use default)
func TestGetClipboardHistoryLimitNegative(t *testing.T) {
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

	// Create config with negative limit
	configDir := tmpDir + "/pipeboard"
	_ = os.MkdirAll(configDir, 0755)
	configContent := `version: 1
history:
  limit: -5
`
	_ = os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0600)

	limit := getClipboardHistoryLimit()
	if limit != defaultClipboardHistoryLimit {
		t.Errorf("negative limit should use default %d, got %d", defaultClipboardHistoryLimit, limit)
	}
}

// Test getClipboardHistoryLimit with no history config (should use default)
func TestGetClipboardHistoryLimitNoHistoryConfig(t *testing.T) {
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

	// Create config without history section
	configDir := tmpDir + "/pipeboard"
	_ = os.MkdirAll(configDir, 0755)
	configContent := `version: 1
sync:
  backend: local
`
	_ = os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0600)

	limit := getClipboardHistoryLimit()
	if limit != defaultClipboardHistoryLimit {
		t.Errorf("no history config should use default %d, got %d", defaultClipboardHistoryLimit, limit)
	}
}

// Test getClipboardHistoryLimit with no config file (should use default)
func TestGetClipboardHistoryLimitNoConfig(t *testing.T) {
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

	// No config file
	limit := getClipboardHistoryLimit()
	if limit != defaultClipboardHistoryLimit {
		t.Errorf("no config should use default %d, got %d", defaultClipboardHistoryLimit, limit)
	}
}

// Test recordClipboardHistory with encrypted data and preview encryption success
func TestRecordClipboardHistoryEncryptionWithPreview(t *testing.T) {
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

	// Create config with encryption
	configDir := tmpDir + "/pipeboard"
	_ = os.MkdirAll(configDir, 0755)
	configContent := `version: 1
sync:
  backend: local
  encryption: aes256
  passphrase: testpassphrase123
`
	_ = os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0600)

	// Record encrypted content
	recordClipboardHistory([]byte("sensitive data that should be encrypted"))

	// Verify encryption worked
	path := getClipboardHistoryPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read clipboard history: %v", err)
	}

	var history []ClipboardHistoryEntry
	if err := json.Unmarshal(data, &history); err != nil {
		t.Fatalf("failed to parse clipboard history: %v", err)
	}

	if len(history) == 0 {
		t.Fatal("expected at least one history entry")
	}

	entry := history[len(history)-1]
	if !entry.Encrypted {
		t.Error("entry should be marked as encrypted")
	}

	// Content should be encrypted (not plaintext)
	if string(entry.Content) == "sensitive data that should be encrypted" {
		t.Error("content should be encrypted, not plaintext")
	}

	// Preview should be encrypted (hex-encoded)
	if strings.Contains(entry.Preview, "sensitive") {
		t.Error("preview should be encrypted, not plaintext")
	}
}

// Test showClipboardHistory with encrypted content and decryption success
func TestShowClipboardHistoryDecryption(t *testing.T) {
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

	// Create config with encryption
	configDir := tmpDir + "/pipeboard"
	_ = os.MkdirAll(configDir, 0755)
	configContent := `version: 1
sync:
  backend: local
  encryption: aes256
  passphrase: decrypttestpass
`
	_ = os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0600)

	// Record encrypted content
	recordClipboardHistory([]byte("encrypted test data"))

	// Show history (should decrypt)
	err := showClipboardHistory(false, "")
	if err != nil {
		t.Errorf("showClipboardHistory should not error with encryption: %v", err)
	}
}

// Test showClipboardHistory with search on encrypted content
func TestShowClipboardHistorySearchEncrypted(t *testing.T) {
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

	// Create config with encryption
	configDir := tmpDir + "/pipeboard"
	_ = os.MkdirAll(configDir, 0755)
	configContent := `version: 1
sync:
  backend: local
  encryption: aes256
  passphrase: searchencrypted
`
	_ = os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0600)

	// Record encrypted searchable content
	recordClipboardHistory([]byte("searchable encrypted data"))

	// Search in encrypted history (should decrypt and search)
	err := showClipboardHistory(false, "searchable")
	if err != nil {
		t.Errorf("search on encrypted history should not error: %v", err)
	}
}

// Test cmdRecall with encrypted entry and successful decryption
func TestCmdRecallEncryptedSuccess(t *testing.T) {
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

	// Create config with encryption
	configDir := tmpDir + "/pipeboard"
	_ = os.MkdirAll(configDir, 0755)
	configContent := `version: 1
sync:
  backend: local
  encryption: aes256
  passphrase: recallencrypted
`
	_ = os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0600)

	// Record encrypted content
	recordClipboardHistory([]byte("recall encrypted test"))

	// Recall should decrypt and restore (may fail on writeClipboard in test env)
	err := cmdRecall([]string{"1"})
	// Error from writeClipboard is acceptable in test environment
	if err != nil && !strings.Contains(err.Error(), "missing") &&
		!strings.Contains(err.Error(), "not found") &&
		!strings.Contains(err.Error(), "no command configured") {
		t.Errorf("cmdRecall unexpected error: %v", err)
	}
}

// Test cmdHistory when history file is missing (should show empty message)
func TestCmdHistoryNoFile(t *testing.T) {
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

	// No history file exists
	err := cmdHistory([]string{})
	if err != nil {
		t.Errorf("cmdHistory should not error when file doesn't exist: %v", err)
	}
}

// Test cmdHistory --json when history file is missing
func TestCmdHistoryNoFileJSON(t *testing.T) {
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

	// No history file exists
	err := cmdHistory([]string{"--json"})
	if err != nil {
		t.Errorf("cmdHistory --json should not error when file doesn't exist: %v", err)
	}
}

// Test recordHistory silently handles directory creation errors
func TestRecordHistoryDirCreationHandling(t *testing.T) {
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

	// This should not panic even if there are issues
	recordHistory("test", "target", 100)
	// Success means no panic
}

// Test recordClipboardHistory with very long content (tests preview truncation)
func TestRecordClipboardHistoryVeryLongContent(t *testing.T) {
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

	// Record very long content (10KB)
	longContent := strings.Repeat("a", 10240)
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

	if len(history) == 0 {
		t.Fatal("expected at least one history entry")
	}

	entry := history[len(history)-1]
	// Preview should be truncated to previewLength + "..."
	if len(entry.Preview) > previewLength+10 {
		t.Errorf("preview should be truncated, got %d chars", len(entry.Preview))
	}

	// Full content should be preserved
	if len(entry.Content) != 10240 {
		t.Errorf("full content should be preserved: expected 10240, got %d bytes", len(entry.Content))
	}
}

// Test cmdRecall with multiple args (should error)
func TestCmdRecallMultipleArgs(t *testing.T) {
	err := cmdRecall([]string{"1", "2"})
	if err == nil {
		t.Error("cmdRecall should error with multiple arguments")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error should show usage: %v", err)
	}
}

// Test cmdRecall with negative index
func TestCmdRecallNegativeIndex(t *testing.T) {
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

	// Record content
	recordClipboardHistory([]byte("test"))

	err := cmdRecall([]string{"-1"})
	if err == nil {
		t.Error("cmdRecall should error on negative index")
	}
	if !strings.Contains(err.Error(), ">= 1") {
		t.Errorf("error should mention index must be >= 1: %v", err)
	}
}

// Test showClipboardHistory with invalid JSON structure
func TestShowClipboardHistoryInvalidJSONStructure(t *testing.T) {
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

	// Create clipboard history with valid JSON but wrong structure
	historyPath := getClipboardHistoryPath()
	_ = os.MkdirAll(tmpDir+"/pipeboard", 0755)
	_ = os.WriteFile(historyPath, []byte(`{"wrong": "structure"}`), 0600)

	err := showClipboardHistory(false, "")
	if err == nil {
		t.Error("showClipboardHistory should error on wrong JSON structure")
	}
}

// Test cmdHistory with all three filters combined
func TestCmdHistoryMultipleFilters(t *testing.T) {
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

	// Record mixed history
	recordHistory("push", "slot1", 100)
	recordHistory("send", "dev", 200)
	recordHistory("fx:pretty-json", "", 50)

	// Use multiple filters (should work even if they're mutually exclusive)
	err := cmdHistory([]string{"--fx", "--slots", "--peer"})
	if err != nil {
		t.Errorf("cmdHistory with multiple filters should not error: %v", err)
	}
}
