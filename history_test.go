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
