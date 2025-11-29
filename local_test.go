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

// Test Push with TTL
func TestLocalBackendPushWithTTL(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 30) // 30 days TTL
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	if err := backend.Push("ttlslot", []byte("data with ttl"), nil); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Verify the slot was created with expiry
	data, err := os.ReadFile(filepath.Join(tmpDir, "ttlslot.pb"))
	if err != nil {
		t.Fatalf("failed to read slot: %v", err)
	}

	if !strings.Contains(string(data), "expires_at") {
		t.Error("slot should have expires_at field")
	}
}

// Test Push without hostname in meta (uses os.Hostname)
func TestLocalBackendPushNoHostname(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Push with nil meta
	if err := backend.Push("nohostslot", []byte("data"), nil); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Pull and check that hostname was set
	_, meta, err := backend.Pull("nohostslot")
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	if meta["hostname"] == "" {
		t.Error("hostname should be set from os.Hostname()")
	}
}

// Test Pull with corrupted JSON
func TestLocalBackendPullCorruptedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Write corrupted JSON
	slotFile := filepath.Join(tmpDir, "corrupted.pb")
	if err := os.WriteFile(slotFile, []byte("{invalid json"), 0600); err != nil {
		t.Fatalf("failed to write corrupted file: %v", err)
	}

	_, _, err = backend.Pull("corrupted")
	if err == nil {
		t.Error("expected error for corrupted JSON")
	}
	if !strings.Contains(err.Error(), "decoding payload") {
		t.Errorf("error should mention decoding: %v", err)
	}
}

// Test Pull with invalid base64
func TestLocalBackendPullInvalidBase64(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Write valid JSON with invalid base64
	payload := `{"version":1,"created_at":"2025-01-01T00:00:00Z","hostname":"test","os":"linux","len":4,"mime":"text/plain","data_b64":"!!!invalid base64!!!"}`
	slotFile := filepath.Join(tmpDir, "badbase64.pb")
	if err := os.WriteFile(slotFile, []byte(payload), 0600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, _, err = backend.Pull("badbase64")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
	if !strings.Contains(err.Error(), "decoding base64") {
		t.Errorf("error should mention base64: %v", err)
	}
}

// Test Pull encrypted slot without passphrase
func TestLocalBackendPullEncryptedNoPassphrase(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	// Create backend with encryption
	backendEnc, err := newLocalBackend(cfg, "aes256", "secret", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Push encrypted data
	if err := backendEnc.Push("encrypted", []byte("secret data"), nil); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Try to pull without passphrase
	backendNoPass, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	_, _, err = backendNoPass.Pull("encrypted")
	if err == nil {
		t.Error("expected error when pulling encrypted slot without passphrase")
	}
	if !strings.Contains(err.Error(), "no passphrase configured") {
		t.Errorf("error should mention no passphrase: %v", err)
	}
}

// Test Pull expired slot
func TestLocalBackendPullExpired(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Write a slot that's already expired
	payload := `{"version":1,"created_at":"2020-01-01T00:00:00Z","expires_at":"2020-01-02T00:00:00Z","hostname":"test","os":"linux","len":4,"mime":"text/plain","data_b64":"dGVzdA=="}`
	slotFile := filepath.Join(tmpDir, "expired.pb")
	if err := os.WriteFile(slotFile, []byte(payload), 0600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, _, err = backend.Pull("expired")
	if err == nil {
		t.Error("expected error for expired slot")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("error should mention expired: %v", err)
	}

	// Slot should be auto-deleted
	if _, err := os.Stat(slotFile); !os.IsNotExist(err) {
		t.Error("expired slot should be auto-deleted")
	}
}

// Test List with directories and non-.pb files
func TestLocalBackendListFiltersCorrectly(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Create a valid slot
	_ = backend.Push("valid", []byte("data"), nil)

	// Create a directory (should be skipped)
	if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Create a non-.pb file (should be skipped)
	if err := os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("text"), 0644); err != nil {
		t.Fatalf("failed to create other file: %v", err)
	}

	slots, err := backend.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(slots) != 1 {
		t.Errorf("expected 1 slot, got %d", len(slots))
	}

	if len(slots) > 0 && slots[0].Name != "valid" {
		t.Errorf("expected slot name 'valid', got %q", slots[0].Name)
	}
}

// Test newLocalBackend with HOME fallback (no XDG_CONFIG_HOME)
func TestLocalBackendDefaultPathHOME(t *testing.T) {
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

	_ = os.Unsetenv("XDG_CONFIG_HOME")
	_ = os.Setenv("HOME", tmpDir)

	cfg := &LocalConfig{} // empty path
	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Should use HOME/.config/pipeboard/slots
	expectedDir := filepath.Join(tmpDir, ".config", "pipeboard", "slots")
	if backend.path != expectedDir {
		t.Errorf("expected path %q, got %q", expectedDir, backend.path)
	}
}

// Test slotPath helper
func TestLocalBackendSlotPath(t *testing.T) {
	backend := &LocalBackend{path: "/tmp/slots"}
	path := backend.slotPath("myslot")
	expected := "/tmp/slots/myslot.pb"
	if path != expected {
		t.Errorf("slotPath mismatch: got %q, want %q", path, expected)
	}
}

// Test Pull with wrong passphrase (decrypt error)
func TestLocalBackendPullWrongPassphrase(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	// Create backend with encryption
	backendEnc, err := newLocalBackend(cfg, "aes256", "correct-pass", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Push encrypted data
	if err := backendEnc.Push("encrypted", []byte("secret data"), nil); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Try to pull with wrong passphrase
	backendWrong, err := newLocalBackend(cfg, "aes256", "wrong-pass", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	_, _, err = backendWrong.Pull("encrypted")
	if err == nil {
		t.Error("expected error when pulling with wrong passphrase")
	}
	if !strings.Contains(err.Error(), "decrypting") {
		t.Errorf("error should mention decrypting: %v", err)
	}
}

// Test Push to read-only directory (write error)
func TestLocalBackendPushWriteError(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Make directory read-only
	if err := os.Chmod(tmpDir, 0555); err != nil {
		t.Skip("cannot change permissions")
	}
	defer func() { _ = os.Chmod(tmpDir, 0755) }()

	err = backend.Push("readonly", []byte("data"), nil)
	// In some environments (e.g., running as root), this may succeed
	if err != nil && !strings.Contains(err.Error(), "writing slot file") {
		t.Errorf("error should mention writing: %v", err)
	}
}

// Test List on nonexistent directory (returns empty, not error)
func TestLocalBackendListNonexistentDir(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistent := filepath.Join(tmpDir, "does-not-exist")

	backend := &LocalBackend{path: nonexistent}

	slots, err := backend.List()
	if err != nil {
		t.Fatalf("List should not error for nonexistent dir: %v", err)
	}

	if len(slots) != 0 {
		t.Errorf("expected 0 slots, got %d", len(slots))
	}
}

// Test Pull with read permission error
func TestLocalBackendPullReadError(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Push a slot
	if err := backend.Push("testslot", []byte("data"), nil); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Make file unreadable
	slotFile := filepath.Join(tmpDir, "testslot.pb")
	if err := os.Chmod(slotFile, 0000); err != nil {
		t.Skip("cannot change permissions")
	}
	defer func() { _ = os.Chmod(slotFile, 0644) }()

	_, _, err = backend.Pull("testslot")
	// In some environments (e.g., running as root), this may succeed
	if err != nil && !strings.Contains(err.Error(), "reading slot file") {
		t.Errorf("error should mention reading: %v", err)
	}
}

// Test newLocalBackend with path that can't be created
func TestLocalBackendMkdirError(t *testing.T) {
	// Try to create a directory inside a file (impossible)
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "afile")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Try to create slots dir inside the file
	cfg := &LocalConfig{Path: filepath.Join(filePath, "slots")}
	_, err := newLocalBackend(cfg, "", "", 0)
	if err == nil {
		t.Error("expected error when creating dir inside file")
	}
	if !strings.Contains(err.Error(), "creating slots directory") {
		t.Errorf("error should mention creating directory: %v", err)
	}
}

// Test Push with compression for large data
func TestLocalBackendPushWithCompression(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Create data > 1KB to trigger compression
	largeData := make([]byte, 2000)
	for i := range largeData {
		largeData[i] = byte('a' + (i % 26))
	}

	if err := backend.Push("compressed", largeData, nil); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Pull and verify data is correctly decompressed
	pulled, meta, err := backend.Pull("compressed")
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	if len(pulled) != len(largeData) {
		t.Errorf("data length mismatch: got %d, want %d", len(pulled), len(largeData))
	}

	if string(pulled) != string(largeData) {
		t.Error("pulled data does not match original")
	}

	// Verify MIME type was detected
	if meta["mime"] == "" {
		t.Error("MIME type should be set")
	}
}

// Test Push with small data (no compression)
func TestLocalBackendPushNoCompression(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Small data (< 1KB) should not be compressed
	smallData := []byte("small data")

	if err := backend.Push("small", smallData, nil); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Pull and verify
	pulled, _, err := backend.Pull("small")
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	if string(pulled) != string(smallData) {
		t.Errorf("data mismatch: got %q, want %q", pulled, smallData)
	}
}

// Test Push/Pull with compression and encryption
func TestLocalBackendCompressionAndEncryption(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "aes256", "test-passphrase", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Create data > 1KB
	largeData := make([]byte, 3000)
	for i := range largeData {
		largeData[i] = byte('x')
	}

	if err := backend.Push("compressed-encrypted", largeData, nil); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Pull and verify
	pulled, _, err := backend.Pull("compressed-encrypted")
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	if string(pulled) != string(largeData) {
		t.Error("data mismatch after compression+encryption roundtrip")
	}
}

// Test Pull of compressed slot without compression flag (legacy data)
func TestLocalBackendPullLegacyUncompressed(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	// Write a slot without compression flag (legacy format)
	payload := `{"version":1,"created_at":"2025-01-01T00:00:00Z","hostname":"test","os":"linux","len":11,"mime":"text/plain","data_b64":"aGVsbG8gd29ybGQ="}`
	slotFile := filepath.Join(tmpDir, "legacy.pb")
	if err := os.WriteFile(slotFile, []byte(payload), 0600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	pulled, _, err := backend.Pull("legacy")
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	if string(pulled) != "hello world" {
		t.Errorf("expected 'hello world', got %q", pulled)
	}
}

// Test MIME type detection for different content types
func TestLocalBackendMIMEDetection(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &LocalConfig{Path: tmpDir}

	backend, err := newLocalBackend(cfg, "", "", 0)
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	tests := []struct {
		name         string
		data         []byte
		expectedMIME string
	}{
		{"plain text", []byte("hello world"), "text/plain"},
		{"HTML", []byte("<html><body>test</body></html>"), "text/html"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slotName := "mime-" + tt.name
			if err := backend.Push(slotName, tt.data, nil); err != nil {
				t.Fatalf("Push failed: %v", err)
			}

			_, meta, err := backend.Pull(slotName)
			if err != nil {
				t.Fatalf("Pull failed: %v", err)
			}

			if !strings.HasPrefix(meta["mime"], tt.expectedMIME) {
				t.Errorf("expected MIME to start with %q, got %q", tt.expectedMIME, meta["mime"])
			}
		})
	}
}
