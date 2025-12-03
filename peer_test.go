package main

import (
	"os"
	"strings"
	"testing"
)

// Helper to set up test config environment
func setupPeerTestConfig(t *testing.T, configContent string) func() {
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

// Test cmdSend with no default peer and no args
func TestCmdSendNoDefaultPeerError(t *testing.T) {
	cleanup := setupPeerTestConfig(t, `version: 1
peers:
  dev:
    ssh: user@host
`)
	defer cleanup()

	err := cmdSend([]string{})
	if err == nil {
		t.Error("cmdSend should error when no default peer")
	}
	if !strings.Contains(err.Error(), "default peer") {
		t.Errorf("error should mention default peer: %v", err)
	}
}

// Test cmdSend with unknown peer
func TestCmdSendUnknownPeerError(t *testing.T) {
	cleanup := setupPeerTestConfig(t, `version: 1
peers:
  dev:
    ssh: user@host
`)
	defer cleanup()

	err := cmdSend([]string{"nonexistent"})
	if err == nil {
		t.Error("cmdSend should error for unknown peer")
	}
	if !strings.Contains(err.Error(), "unknown peer") {
		t.Errorf("error should mention unknown peer: %v", err)
	}
}

// Test cmdRecv with no default peer
func TestCmdRecvNoDefaultPeerError(t *testing.T) {
	cleanup := setupPeerTestConfig(t, `version: 1
peers:
  dev:
    ssh: user@host
`)
	defer cleanup()

	err := cmdRecv([]string{})
	if err == nil {
		t.Error("cmdRecv should error when no default peer")
	}
	if !strings.Contains(err.Error(), "default peer") {
		t.Errorf("error should mention default peer: %v", err)
	}
}

// Test cmdRecv with unknown peer
func TestCmdRecvUnknownPeerError(t *testing.T) {
	cleanup := setupPeerTestConfig(t, `version: 1
peers:
  dev:
    ssh: user@host
`)
	defer cleanup()

	err := cmdRecv([]string{"nonexistent"})
	if err == nil {
		t.Error("cmdRecv should error for unknown peer")
	}
	if !strings.Contains(err.Error(), "unknown peer") {
		t.Errorf("error should mention unknown peer: %v", err)
	}
}

// Test cmdPeek with no default peer
func TestCmdPeekNoDefaultPeerError(t *testing.T) {
	cleanup := setupPeerTestConfig(t, `version: 1
peers:
  dev:
    ssh: user@host
`)
	defer cleanup()

	err := cmdPeek([]string{})
	if err == nil {
		t.Error("cmdPeek should error when no default peer")
	}
	if !strings.Contains(err.Error(), "default peer") {
		t.Errorf("error should mention default peer: %v", err)
	}
}

// Test cmdPeek with unknown peer
func TestCmdPeekUnknownPeerError(t *testing.T) {
	cleanup := setupPeerTestConfig(t, `version: 1
peers:
  dev:
    ssh: user@host
`)
	defer cleanup()

	err := cmdPeek([]string{"nonexistent"})
	if err == nil {
		t.Error("cmdPeek should error for unknown peer")
	}
	if !strings.Contains(err.Error(), "unknown peer") {
		t.Errorf("error should mention unknown peer: %v", err)
	}
}

// Test cmdPeek with peer missing SSH
func TestCmdPeekPeerMissingSSHError(t *testing.T) {
	cleanup := setupPeerTestConfig(t, `version: 1
peers:
  broken:
    remote_cmd: pipeboard
`)
	defer cleanup()

	err := cmdPeek([]string{"broken"})
	if err == nil {
		t.Error("cmdPeek should error for peer missing SSH")
	}
	if !strings.Contains(err.Error(), "ssh") {
		t.Errorf("error should mention ssh: %v", err)
	}
}

// Test cmdSend with peer missing SSH
func TestCmdSendPeerMissingSSHError(t *testing.T) {
	cleanup := setupPeerTestConfig(t, `version: 1
peers:
  broken:
    remote_cmd: pipeboard
`)
	defer cleanup()

	err := cmdSend([]string{"broken"})
	if err == nil {
		t.Error("cmdSend should error for peer missing SSH")
	}
	if !strings.Contains(err.Error(), "ssh") {
		t.Errorf("error should mention ssh: %v", err)
	}
}

// Test cmdRecv with peer missing SSH
func TestCmdRecvPeerMissingSSHError(t *testing.T) {
	cleanup := setupPeerTestConfig(t, `version: 1
peers:
  broken:
    remote_cmd: pipeboard
`)
	defer cleanup()

	err := cmdRecv([]string{"broken"})
	if err == nil {
		t.Error("cmdRecv should error for peer missing SSH")
	}
	if !strings.Contains(err.Error(), "ssh") {
		t.Errorf("error should mention ssh: %v", err)
	}
}

// Test all peer commands with no peers section
func TestPeerCommandsNoPeersSection(t *testing.T) {
	cleanup := setupPeerTestConfig(t, `version: 1
sync:
  backend: local
`)
	defer cleanup()

	tests := []struct {
		name string
		fn   func([]string) error
	}{
		{"send", cmdSend},
		{"recv", cmdRecv},
		{"peek", cmdPeek},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn([]string{"anypeer"})
			if err == nil {
				t.Errorf("%s should error when no peers configured", tc.name)
			}
			if !strings.Contains(err.Error(), "no peers") {
				t.Errorf("%s error should mention no peers: %v", tc.name, err)
			}
		})
	}
}

// Test peer commands with invalid YAML config
func TestPeerCommandsInvalidYAML(t *testing.T) {
	cleanup := setupPeerTestConfig(t, `invalid yaml: [`)
	defer cleanup()

	tests := []struct {
		name string
		fn   func([]string) error
		args []string
	}{
		{"send", cmdSend, []string{"test"}},
		{"recv", cmdRecv, []string{"test"}},
		{"peek", cmdPeek, []string{"test"}},
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

// Test cmdSend with too many arguments
func TestCmdSendTooManyArgsError(t *testing.T) {
	cleanup := setupPeerTestConfig(t, `version: 1
peers:
  dev:
    ssh: user@host
`)
	defer cleanup()

	err := cmdSend([]string{"peer1", "peer2"})
	if err == nil {
		t.Error("cmdSend should error with too many args")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error should show usage: %v", err)
	}
}

// Test cmdRecv with too many arguments
func TestCmdRecvTooManyArgsError(t *testing.T) {
	cleanup := setupPeerTestConfig(t, `version: 1
peers:
  dev:
    ssh: user@host
`)
	defer cleanup()

	err := cmdRecv([]string{"peer1", "peer2"})
	if err == nil {
		t.Error("cmdRecv should error with too many args")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error should show usage: %v", err)
	}
}

// Test cmdPeek with too many arguments
func TestCmdPeekTooManyArgsError(t *testing.T) {
	cleanup := setupPeerTestConfig(t, `version: 1
peers:
  dev:
    ssh: user@host
`)
	defer cleanup()

	err := cmdPeek([]string{"peer1", "peer2"})
	if err == nil {
		t.Error("cmdPeek should error with too many args")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("error should show usage: %v", err)
	}
}

// Test cmdSend with no config file
func TestCmdSendNoConfigFile(t *testing.T) {
	cleanup := setupPeerTestConfig(t, "")
	defer cleanup()

	err := cmdSend([]string{"anypeer"})
	if err == nil {
		t.Error("cmdSend should error when no config file exists")
	}
}

// Test cmdRecv with no config file
func TestCmdRecvNoConfigFile(t *testing.T) {
	cleanup := setupPeerTestConfig(t, "")
	defer cleanup()

	err := cmdRecv([]string{"anypeer"})
	if err == nil {
		t.Error("cmdRecv should error when no config file exists")
	}
}

// Test cmdPeek with no config file
func TestCmdPeekNoConfigFile(t *testing.T) {
	cleanup := setupPeerTestConfig(t, "")
	defer cleanup()

	err := cmdPeek([]string{"anypeer"})
	if err == nil {
		t.Error("cmdPeek should error when no config file exists")
	}
}
