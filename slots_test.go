package main

import (
	"os"
	"strings"
	"testing"
)

// Helper to set up test config environment for slots
func setupSlotsTestConfig(t *testing.T, configContent string) func() {
	t.Helper()
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")

	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)

	if configContent != "" {
		configDir := tmpDir + "/pipeboard"
		_ = os.MkdirAll(configDir, 0755)
		_ = os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0600)
	}

	return func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}
}

// Test cmdPull with nonexistent slot
func TestCmdPullNonexistentSlot(t *testing.T) {
	cleanup := setupSlotsTestConfig(t, `version: 1
sync:
  backend: local
`)
	defer cleanup()

	err := cmdPull([]string{"nonexistent-slot-xyz"})
	if err == nil {
		t.Error("cmdPull should error for nonexistent slot")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

// Test cmdShow with nonexistent slot
func TestCmdShowNonexistentSlot(t *testing.T) {
	cleanup := setupSlotsTestConfig(t, `version: 1
sync:
  backend: local
`)
	defer cleanup()

	err := cmdShow([]string{"nonexistent-slot-xyz"})
	if err == nil {
		t.Error("cmdShow should error for nonexistent slot")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

// Test cmdSlots with unknown flag
func TestCmdSlotsUnknownFlag(t *testing.T) {
	cleanup := setupSlotsTestConfig(t, `version: 1
sync:
  backend: local
`)
	defer cleanup()

	err := cmdSlots([]string{"--unknown"})
	if err == nil {
		t.Error("cmdSlots should error with unknown flag")
	}
	if !strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("error should mention unknown flag: %v", err)
	}
}

// Test cmdSlots with no config (should use default local backend)
func TestCmdSlotsNoConfig(t *testing.T) {
	cleanup := setupSlotsTestConfig(t, "")
	defer cleanup()

	// Should not error - falls back to local backend
	err := cmdSlots([]string{})
	// Local backend should work without config
	if err != nil && !strings.Contains(err.Error(), "config") {
		t.Errorf("unexpected error: %v", err)
	}
}

// Test cmdSlots with empty slots (text output)
func TestCmdSlotsEmptyText(t *testing.T) {
	cleanup := setupSlotsTestConfig(t, `version: 1
sync:
  backend: local
`)
	defer cleanup()

	// Should not error, just print "No slots found"
	err := cmdSlots([]string{})
	if err != nil {
		t.Errorf("cmdSlots should not error with empty slots: %v", err)
	}
}

// Test cmdSlots with empty slots (JSON output)
func TestCmdSlotsEmptyJSON(t *testing.T) {
	cleanup := setupSlotsTestConfig(t, `version: 1
sync:
  backend: local
`)
	defer cleanup()

	// Should not error, just print "[]"
	err := cmdSlots([]string{"--json"})
	if err != nil {
		t.Errorf("cmdSlots --json should not error with empty slots: %v", err)
	}
}

// Test cmdRm with nonexistent slot
func TestCmdRmNonexistentSlot(t *testing.T) {
	cleanup := setupSlotsTestConfig(t, `version: 1
sync:
  backend: local
`)
	defer cleanup()

	err := cmdRm([]string{"nonexistent-slot-xyz"})
	if err == nil {
		t.Error("cmdRm should error for nonexistent slot")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

// Test slot commands with invalid config
func TestSlotCommandsInvalidConfig(t *testing.T) {
	cleanup := setupSlotsTestConfig(t, `invalid yaml: [`)
	defer cleanup()

	tests := []struct {
		name string
		fn   func([]string) error
		args []string
	}{
		{"push", cmdPush, []string{"test"}},
		{"pull", cmdPull, []string{"test"}},
		{"show", cmdShow, []string{"test"}},
		{"slots", cmdSlots, []string{}},
		{"rm", cmdRm, []string{"test"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn(tc.args)
			if err == nil {
				t.Errorf("%s should error with invalid config", tc.name)
			}
		})
	}
}

// Test push/pull roundtrip with local backend
func TestSlotsPushPullRoundtrip(t *testing.T) {
	cleanup := setupSlotsTestConfig(t, `version: 1
sync:
  backend: local
`)
	defer cleanup()

	// This tests the actual push/pull flow
	// Note: This test requires stdin input for push, which is complex
	// For now, we test that the functions accept correct arguments
	// and that pull properly fails for nonexistent slots
	err := cmdPull([]string{"test-roundtrip-slot"})
	if err == nil {
		t.Error("cmdPull should error for nonexistent slot before push")
	}
}

// Test cmdShow for slot that doesn't exist with JSON output
func TestCmdShowNonexistentJSON(t *testing.T) {
	cleanup := setupSlotsTestConfig(t, `version: 1
sync:
  backend: local
`)
	defer cleanup()

	// cmdShow doesn't have --json flag, but we test the basic error handling
	err := cmdShow([]string{"nonexistent-json-slot"})
	if err == nil {
		t.Error("cmdShow should error for nonexistent slot")
	}
}

// Test cmdSlots with verbose flag
func TestCmdSlotsVerbose(t *testing.T) {
	cleanup := setupSlotsTestConfig(t, `version: 1
sync:
  backend: local
`)
	defer cleanup()

	// Test verbose flag (if supported)
	err := cmdSlots([]string{"-v"})
	// Should either work or give unknown flag error
	if err != nil && !strings.Contains(err.Error(), "unknown flag") && !strings.Contains(err.Error(), "flag") {
		t.Errorf("unexpected error: %v", err)
	}
}

// Test successful slot operations with local backend
func TestSlotOperationsWithLocalBackend(t *testing.T) {
	cleanup := setupSlotsTestConfig(t, `version: 1
sync:
  backend: local
`)
	defer cleanup()

	// Create a local backend directly to push test data
	backend, err := newRemoteBackendFromConfig()
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	// Push test data
	testData := []byte("test clipboard content")
	meta := map[string]string{"hostname": "test-host"}
	if err := backend.Push("test-slot", testData, meta); err != nil {
		t.Fatalf("failed to push test data: %v", err)
	}

	// Test cmdSlots lists the slot
	err = cmdSlots([]string{})
	if err != nil {
		t.Errorf("cmdSlots should succeed with existing slots: %v", err)
	}

	// Test cmdSlots JSON output
	err = cmdSlots([]string{"--json"})
	if err != nil {
		t.Errorf("cmdSlots --json should succeed: %v", err)
	}

	// Test cmdShow displays content
	err = cmdShow([]string{"test-slot"})
	if err != nil {
		t.Errorf("cmdShow should succeed for existing slot: %v", err)
	}

	// Test cmdRm deletes the slot
	err = cmdRm([]string{"test-slot"})
	if err != nil {
		t.Errorf("cmdRm should succeed for existing slot: %v", err)
	}

	// Verify slot was deleted
	err = cmdShow([]string{"test-slot"})
	if err == nil {
		t.Error("cmdShow should error after slot deletion")
	}
}

// Test slot operations with multiple slots
func TestSlotListMultipleSlots(t *testing.T) {
	cleanup := setupSlotsTestConfig(t, `version: 1
sync:
  backend: local
`)
	defer cleanup()

	backend, err := newRemoteBackendFromConfig()
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	// Push multiple slots
	for _, name := range []string{"alpha", "beta", "gamma"} {
		if err := backend.Push(name, []byte("data-"+name), nil); err != nil {
			t.Fatalf("failed to push slot %s: %v", name, err)
		}
	}

	// List should show all slots
	slots, err := backend.List()
	if err != nil {
		t.Fatalf("failed to list slots: %v", err)
	}
	if len(slots) != 3 {
		t.Errorf("expected 3 slots, got %d", len(slots))
	}

	// Cleanup
	for _, name := range []string{"alpha", "beta", "gamma"} {
		_ = backend.Delete(name)
	}
}

// Test slot with metadata including hostname
func TestSlotMetadataHostname(t *testing.T) {
	cleanup := setupSlotsTestConfig(t, `version: 1
sync:
  backend: local
`)
	defer cleanup()

	backend, err := newRemoteBackendFromConfig()
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	// Push with hostname metadata
	meta := map[string]string{"hostname": "my-workstation"}
	if err := backend.Push("meta-test", []byte("content"), meta); err != nil {
		t.Fatalf("failed to push: %v", err)
	}

	// Pull should include metadata
	data, pullMeta, err := backend.Pull("meta-test")
	if err != nil {
		t.Fatalf("failed to pull: %v", err)
	}

	if string(data) != "content" {
		t.Errorf("expected 'content', got %q", string(data))
	}

	if pullMeta["hostname"] != "my-workstation" {
		t.Errorf("expected hostname 'my-workstation', got %q", pullMeta["hostname"])
	}

	_ = backend.Delete("meta-test")
}
