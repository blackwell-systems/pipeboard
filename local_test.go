package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalBackendPushPull(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Push data
	testData := []byte("hello world")
	meta := map[string]string{"hostname": "testhost"}
	if err := backend.Push("testslot", testData, meta); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Verify file was created
	slotFile := filepath.Join(tmpDir, "testslot.pb")
	if _, err := os.Stat(slotFile); os.IsNotExist(err) {
		t.Error("slot file should exist")
	}

	// Pull data
	pulled, pulledMeta, err := backend.Pull("testslot")
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	if string(pulled) != string(testData) {
		t.Errorf("pulled data mismatch: got %q, want %q", pulled, testData)
	}

	if pulledMeta["hostname"] != "testhost" {
		t.Errorf("hostname mismatch: got %q, want %q", pulledMeta["hostname"], "testhost")
	}
}

func TestLocalBackendList(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Push multiple slots
	_ = backend.Push("slot1", []byte("data1"), nil)
	_ = backend.Push("slot2", []byte("data2"), nil)
	_ = backend.Push("slot3", []byte("data3"), nil)

	// List slots
	slots, err := backend.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(slots) != 3 {
		t.Errorf("expected 3 slots, got %d", len(slots))
	}

	// Check slot names
	names := make(map[string]bool)
	for _, s := range slots {
		names[s.Name] = true
	}

	for _, expected := range []string{"slot1", "slot2", "slot3"} {
		if !names[expected] {
			t.Errorf("missing slot: %s", expected)
		}
	}
}

func TestLocalBackendDelete(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Push and delete
	_ = backend.Push("todelete", []byte("delete me"), nil)

	if err := backend.Delete("todelete"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify file was deleted
	slotFile := filepath.Join(tmpDir, "todelete.pb")
	if _, err := os.Stat(slotFile); !os.IsNotExist(err) {
		t.Error("slot file should be deleted")
	}

	// Pull should fail
	_, _, err = backend.Pull("todelete")
	if err == nil {
		t.Error("Pull should fail after delete")
	}
}

func TestLocalBackendDeleteNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	err = backend.Delete("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent slot")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestLocalBackendPullNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	_, _, err = backend.Pull("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent slot")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestLocalBackendWithEncryption(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "aes256", "test-passphrase", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Push encrypted data
	testData := []byte("secret data")
	if err := backend.Push("encrypted", testData, nil); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Pull and verify decryption
	pulled, _, err := backend.Pull("encrypted")
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	if string(pulled) != string(testData) {
		t.Errorf("decrypted data mismatch: got %q, want %q", pulled, testData)
	}
}

func TestLocalBackendEncryptionMissingPassphrase(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	_, err := newLocalBackend(cfg, "aes256", "", 0)
	if err == nil {
		t.Error("expected error for missing passphrase")
	}
	if !strings.Contains(err.Error(), "passphrase required") {
		t.Errorf("error should mention passphrase: %v", err)
	}
}

func TestLocalBackendDefaultPath(t *testing.T) {
	// Test with empty config (uses default path)
	tmpDir := t.TempDir()

	origHome := os.Getenv("HOME")
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		_ = os.Setenv("HOME", origHome)
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	_ = os.Unsetenv("HOME")

	cfg := &LocalConfig{} // empty path
	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Should use XDG_CONFIG_HOME/pipeboard/slots
	expectedDir := filepath.Join(tmpDir, "pipeboard", "slots")
	if backend.path != expectedDir {
		t.Errorf("expected path %q, got %q", expectedDir, backend.path)
	}
}

func TestLocalBackendListEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	slots, err := backend.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(slots) != 0 {
		t.Errorf("expected 0 slots, got %d", len(slots))
	}
}

func TestValidateSyncConfigLocal(t *testing.T) {
	cfg := &Config{
		Sync: &SyncConfig{
			Backend: "local",
		},
	}

	err := validateSyncConfig(cfg)
	if err != nil {
		t.Errorf("local backend should be valid: %v", err)
	}

	// Should have created default LocalConfig
	if cfg.Sync.Local == nil {
		t.Error("LocalConfig should be created")
	}
}
