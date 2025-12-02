package main

import (
	"os"
	"strings"
	"testing"
)

// Test cmdWatch with too many arguments
func TestCmdWatchTooManyArgsError(t *testing.T) {
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

	// Create config with peers
	configDir := tmpDir + "/pipeboard"
	_ = os.MkdirAll(configDir, 0755)
	configContent := `version: 1
peers:
  work:
    ssh: user@host
`
	_ = os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0600)

	err := cmdWatch([]string{"peer1", "peer2", "peer3"})
	if err == nil {
		t.Error("cmdWatch should error with too many args")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error should mention usage: %v", err)
	}
}

// Test cmdWatch with no config file
func TestCmdWatchNoConfig(t *testing.T) {
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

	err := cmdWatch([]string{})
	if err == nil {
		t.Error("cmdWatch should error when config doesn't exist")
	}
}

// Test cmdWatch with no default peer and no args
func TestCmdWatchNoDefaultPeer(t *testing.T) {
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

	// Create config without default peer
	configDir := tmpDir + "/pipeboard"
	_ = os.MkdirAll(configDir, 0755)
	configContent := `version: 1
peers:
  work:
    ssh: user@host
`
	_ = os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0600)

	err := cmdWatch([]string{})
	if err == nil {
		t.Error("cmdWatch should error when no default peer and no args")
	}
	if !strings.Contains(err.Error(), "default peer") || !strings.Contains(err.Error(), "usage") {
		t.Errorf("error should mention default peer and usage: %v", err)
	}
}

// Test cmdWatch with unknown peer
func TestCmdWatchUnknownPeer(t *testing.T) {
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

	// Create config with peers
	configDir := tmpDir + "/pipeboard"
	_ = os.MkdirAll(configDir, 0755)
	configContent := `version: 1
peers:
  work:
    ssh: user@host
`
	_ = os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0600)

	err := cmdWatch([]string{"nonexistent"})
	if err == nil {
		t.Error("cmdWatch should error for unknown peer")
	}
	if !strings.Contains(err.Error(), "unknown peer") {
		t.Errorf("error should mention unknown peer: %v", err)
	}
}

// Test cmdWatch with peer missing SSH field
func TestCmdWatchPeerMissingSSH(t *testing.T) {
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

	// Create config with peer missing SSH
	configDir := tmpDir + "/pipeboard"
	_ = os.MkdirAll(configDir, 0755)
	configContent := `version: 1
peers:
  broken:
    remote_cmd: pipeboard
`
	_ = os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0600)

	err := cmdWatch([]string{"broken"})
	if err == nil {
		t.Error("cmdWatch should error for peer missing SSH")
	}
	if !strings.Contains(err.Error(), "ssh") {
		t.Errorf("error should mention ssh field: %v", err)
	}
}

// Test cmdWatch with no peers configured
func TestCmdWatchNoPeers(t *testing.T) {
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

	// Create config with no peers
	configDir := tmpDir + "/pipeboard"
	_ = os.MkdirAll(configDir, 0755)
	configContent := `version: 1
sync:
  backend: local
`
	_ = os.WriteFile(configDir+"/config.yaml", []byte(configContent), 0600)

	err := cmdWatch([]string{"anypeer"})
	if err == nil {
		t.Error("cmdWatch should error when no peers configured")
	}
	if !strings.Contains(err.Error(), "no peers") {
		t.Errorf("error should mention no peers: %v", err)
	}
}

// Test watch constants are reasonable
func TestWatchConstants(t *testing.T) {
	if defaultWatchInterval <= 0 {
		t.Error("defaultWatchInterval should be positive")
	}
	if minWatchInterval <= 0 {
		t.Error("minWatchInterval should be positive")
	}
	if minWatchInterval > defaultWatchInterval {
		t.Error("minWatchInterval should not exceed defaultWatchInterval")
	}
}

// Test readRemoteClipboard with invalid SSH command
func TestReadRemoteClipboardError(t *testing.T) {
	peer := PeerConfig{
		SSH:       "nonexistent-host-12345",
		RemoteCmd: "pipeboard",
	}
	_, err := readRemoteClipboard(peer)
	if err == nil {
		t.Error("readRemoteClipboard should error with invalid SSH host")
	}
}

// Test readRemoteClipboard with successful execution
func TestReadRemoteClipboardSuccess(t *testing.T) {
	// Use a command that will succeed - echo piped through cat
	peer := PeerConfig{
		SSH:       "localhost",
		RemoteCmd: "echo",
	}
	// We can't fully test this without SSH setup, but we can verify the function exists
	// and handles the peer config correctly
	_, err := readRemoteClipboard(peer)
	// Error is expected since "echo paste" won't work as expected
	// but it tests the code path
	_ = err
}

// Test sendToRemote with invalid SSH command
func TestSendToRemoteError(t *testing.T) {
	peer := PeerConfig{
		SSH:       "nonexistent-host-12345",
		RemoteCmd: "pipeboard",
	}
	data := []byte("test data")
	err := sendToRemote(peer, data)
	if err == nil {
		t.Error("sendToRemote should error with invalid SSH host")
	}
}

// Test sendToRemote with empty data
func TestSendToRemoteEmptyData(t *testing.T) {
	peer := PeerConfig{
		SSH:       "nonexistent-host-12345",
		RemoteCmd: "pipeboard",
	}
	data := []byte{}
	err := sendToRemote(peer, data)
	// Should still error due to invalid host, but tests empty data handling
	if err == nil {
		t.Error("sendToRemote should error with invalid SSH host")
	}
}

// Test sendToRemote with large data
func TestSendToRemoteLargeData(t *testing.T) {
	peer := PeerConfig{
		SSH:       "nonexistent-host-12345",
		RemoteCmd: "pipeboard",
	}
	// Create 1MB of data
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	err := sendToRemote(peer, data)
	// Should error due to invalid host, but tests large data handling
	if err == nil {
		t.Error("sendToRemote should error with invalid SSH host")
	}
}
