package main

import (
	"os"
	"strings"
	"testing"
)

// Test generateConfigYAML with minimal config
func TestGenerateConfigYAMLMinimal(t *testing.T) {
	cfg := &Config{
		Sync: &SyncConfig{
			Backend: "local",
		},
	}

	yaml := generateConfigYAML(cfg)

	if !strings.Contains(yaml, "sync:") {
		t.Error("YAML should contain sync section")
	}
	if !strings.Contains(yaml, "backend: local") {
		t.Error("YAML should contain backend: local")
	}
}

// Test generateConfigYAML with S3 config
func TestGenerateConfigYAMLWithS3(t *testing.T) {
	cfg := &Config{
		Sync: &SyncConfig{
			Backend: "s3",
			S3: &S3Config{
				Bucket: "my-bucket",
				Region: "us-west-2",
				Prefix: "pipeboard/",
			},
			Encryption: "aes256",
			TTLDays:    30,
		},
	}

	yaml := generateConfigYAML(cfg)

	if !strings.Contains(yaml, "backend: s3") {
		t.Error("YAML should contain backend: s3")
	}
	if !strings.Contains(yaml, "bucket: my-bucket") {
		t.Error("YAML should contain bucket")
	}
	if !strings.Contains(yaml, "region: us-west-2") {
		t.Error("YAML should contain region")
	}
	if !strings.Contains(yaml, "prefix: pipeboard/") {
		t.Error("YAML should contain prefix")
	}
	if !strings.Contains(yaml, "encryption: aes256") {
		t.Error("YAML should contain encryption")
	}
	if !strings.Contains(yaml, "ttl_days: 30") {
		t.Error("YAML should contain ttl_days")
	}
}

// Test generateConfigYAML with peers
func TestGenerateConfigYAMLWithPeers(t *testing.T) {
	cfg := &Config{
		Sync: &SyncConfig{
			Backend: "local",
		},
		Peers: map[string]PeerConfig{
			"work": {
				SSH:       "user@workhost",
				RemoteCmd: "pipeboard",
			},
			"laptop": {
				SSH:       "user@laptop",
				RemoteCmd: "/custom/path/pipeboard",
			},
		},
		Defaults: &DefaultsConfig{
			Peer: "work",
		},
	}

	yaml := generateConfigYAML(cfg)

	if !strings.Contains(yaml, "peers:") {
		t.Error("YAML should contain peers section")
	}
	if !strings.Contains(yaml, "work:") {
		t.Error("YAML should contain work peer")
	}
	if !strings.Contains(yaml, "ssh: user@workhost") {
		t.Error("YAML should contain work SSH host")
	}
	if !strings.Contains(yaml, "laptop:") {
		t.Error("YAML should contain laptop peer")
	}
	if !strings.Contains(yaml, "remote_cmd: /custom/path/pipeboard") {
		t.Error("YAML should contain custom remote_cmd")
	}
	if !strings.Contains(yaml, "defaults:") {
		t.Error("YAML should contain defaults section")
	}
	if !strings.Contains(yaml, "peer: work") {
		t.Error("YAML should contain default peer")
	}
}

// Test generateConfigYAML with transforms (fx)
func TestGenerateConfigYAMLWithFx(t *testing.T) {
	cfg := &Config{
		Sync: &SyncConfig{
			Backend: "local",
		},
		Fx: map[string]FxConfig{
			"pretty-json": {
				Cmd:         []string{"jq", "."},
				Description: "Format JSON",
			},
			"sort-lines": {
				Shell:       "sort",
				Description: "Sort lines",
			},
		},
	}

	yaml := generateConfigYAML(cfg)

	if !strings.Contains(yaml, "fx:") {
		t.Error("YAML should contain fx section")
	}
	if !strings.Contains(yaml, "pretty-json:") {
		t.Error("YAML should contain pretty-json transform")
	}
	if !strings.Contains(yaml, `cmd: ["jq", "."]`) {
		t.Error("YAML should contain cmd array")
	}
	if !strings.Contains(yaml, "sort-lines:") {
		t.Error("YAML should contain sort-lines transform")
	}
	if !strings.Contains(yaml, `shell: "sort"`) {
		t.Error("YAML should contain shell command")
	}
	if !strings.Contains(yaml, `description: "Format JSON"`) {
		t.Error("YAML should contain description")
	}
}

// Test generateConfigYAML with empty config
func TestGenerateConfigYAMLEmpty(t *testing.T) {
	cfg := &Config{}

	yaml := generateConfigYAML(cfg)

	// Should at least have the header comments
	if !strings.Contains(yaml, "# pipeboard configuration") {
		t.Error("YAML should contain header comment")
	}
}

// Test generateConfigYAML with local backend and custom path
func TestGenerateConfigYAMLLocalWithPath(t *testing.T) {
	cfg := &Config{
		Sync: &SyncConfig{
			Backend: "local",
			Local: &LocalConfig{
				Path: "/custom/slots/path",
			},
		},
	}

	yaml := generateConfigYAML(cfg)

	if !strings.Contains(yaml, "local:") {
		t.Error("YAML should contain local section")
	}
	if !strings.Contains(yaml, "path: /custom/slots/path") {
		t.Error("YAML should contain custom path")
	}
}

// Test generateConfigYAML with S3 without prefix
func TestGenerateConfigYAMLS3NoPrefix(t *testing.T) {
	cfg := &Config{
		Sync: &SyncConfig{
			Backend: "s3",
			S3: &S3Config{
				Bucket: "bucket",
				Region: "us-east-1",
				Prefix: "", // No prefix
			},
		},
	}

	yaml := generateConfigYAML(cfg)

	if !strings.Contains(yaml, "bucket: bucket") {
		t.Error("YAML should contain bucket")
	}
	// Prefix should not appear when empty
	if strings.Contains(yaml, "prefix:") {
		t.Error("YAML should not contain prefix when empty")
	}
}

// Test generateConfigYAML with peer using default remote_cmd
func TestGenerateConfigYAMLPeerDefaultCmd(t *testing.T) {
	cfg := &Config{
		Sync: &SyncConfig{Backend: "local"},
		Peers: map[string]PeerConfig{
			"default-peer": {
				SSH:       "host",
				RemoteCmd: "pipeboard", // Default value
			},
		},
	}

	yaml := generateConfigYAML(cfg)

	if !strings.Contains(yaml, "default-peer:") {
		t.Error("YAML should contain peer")
	}
	// Default remote_cmd should not be written
	if strings.Contains(yaml, "remote_cmd: pipeboard") {
		t.Error("YAML should not contain default remote_cmd")
	}
}

// Test generateConfigYAML produces valid structure
func TestGenerateConfigYAMLStructure(t *testing.T) {
	cfg := &Config{
		Sync: &SyncConfig{
			Backend: "s3",
			S3: &S3Config{
				Bucket: "test",
				Region: "us-east-1",
			},
			Encryption: "aes256",
			TTLDays:    7,
		},
		Peers: map[string]PeerConfig{
			"test": {SSH: "host"},
		},
		Defaults: &DefaultsConfig{Peer: "test"},
		Fx: map[string]FxConfig{
			"test": {Shell: "cat"},
		},
	}

	yaml := generateConfigYAML(cfg)

	// Check section order (sync should come before peers)
	syncIdx := strings.Index(yaml, "sync:")
	peersIdx := strings.Index(yaml, "peers:")
	defaultsIdx := strings.Index(yaml, "defaults:")
	fxIdx := strings.Index(yaml, "fx:")

	if syncIdx == -1 || peersIdx == -1 || defaultsIdx == -1 || fxIdx == -1 {
		t.Error("YAML missing sections")
	}

	if syncIdx > peersIdx {
		t.Error("sync should come before peers")
	}
}

// Helper to mock stdin with given input
func mockStdin(t *testing.T, input string) func() {
	t.Helper()
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	go func() {
		defer w.Close()
		w.WriteString(input)
	}()
	return func() {
		os.Stdin = oldStdin
	}
}

// Test promptString with default value
func TestPromptStringDefault(t *testing.T) {
	restore := mockStdin(t, "\n")
	defer restore()

	result := promptString("Test prompt", "default-value")
	if result != "default-value" {
		t.Errorf("expected 'default-value', got %q", result)
	}
}

// Test promptString with custom input
func TestPromptStringCustom(t *testing.T) {
	restore := mockStdin(t, "custom-input\n")
	defer restore()

	result := promptString("Test prompt", "default")
	if result != "custom-input" {
		t.Errorf("expected 'custom-input', got %q", result)
	}
}

// Test promptString with empty default
func TestPromptStringNoDefault(t *testing.T) {
	restore := mockStdin(t, "user-input\n")
	defer restore()

	result := promptString("Test prompt", "")
	if result != "user-input" {
		t.Errorf("expected 'user-input', got %q", result)
	}
}

// Test promptYesNo with default yes
func TestPromptYesNoDefaultYes(t *testing.T) {
	restore := mockStdin(t, "\n")
	defer restore()

	result := promptYesNo("Continue?", true)
	if !result {
		t.Error("expected true for default yes")
	}
}

// Test promptYesNo with default no
func TestPromptYesNoDefaultNo(t *testing.T) {
	restore := mockStdin(t, "\n")
	defer restore()

	result := promptYesNo("Continue?", false)
	if result {
		t.Error("expected false for default no")
	}
}

// Test promptYesNo with explicit yes
func TestPromptYesNoExplicitYes(t *testing.T) {
	restore := mockStdin(t, "y\n")
	defer restore()

	result := promptYesNo("Continue?", false)
	if !result {
		t.Error("expected true for 'y' input")
	}
}

// Test promptYesNo with explicit yes (full word)
func TestPromptYesNoExplicitYesFull(t *testing.T) {
	restore := mockStdin(t, "yes\n")
	defer restore()

	result := promptYesNo("Continue?", false)
	if !result {
		t.Error("expected true for 'yes' input")
	}
}

// Test promptYesNo with explicit no
func TestPromptYesNoExplicitNo(t *testing.T) {
	restore := mockStdin(t, "n\n")
	defer restore()

	result := promptYesNo("Continue?", true)
	if result {
		t.Error("expected false for 'n' input")
	}
}

// Test promptChoice with default
func TestPromptChoiceDefault(t *testing.T) {
	restore := mockStdin(t, "\n")
	defer restore()

	result := promptChoice("Choose", []string{"a", "b", "c"}, "b")
	if result != "b" {
		t.Errorf("expected 'b', got %q", result)
	}
}

// Test promptChoice with valid selection
func TestPromptChoiceValid(t *testing.T) {
	restore := mockStdin(t, "c\n")
	defer restore()

	result := promptChoice("Choose", []string{"a", "b", "c"}, "a")
	if result != "c" {
		t.Errorf("expected 'c', got %q", result)
	}
}

// Test promptChoice with invalid selection (falls back to default)
func TestPromptChoiceInvalid(t *testing.T) {
	restore := mockStdin(t, "invalid\n")
	defer restore()

	result := promptChoice("Choose", []string{"a", "b", "c"}, "a")
	if result != "a" {
		t.Errorf("expected 'a' (default), got %q", result)
	}
}

// Test cmdInit when config already exists and user declines overwrite
func TestCmdInitExistsDecline(t *testing.T) {
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

	// Create existing config
	configDir := tmpDir + "/pipeboard"
	_ = os.MkdirAll(configDir, 0755)
	_ = os.WriteFile(configDir+"/config.yaml", []byte("existing: config"), 0600)

	// Mock stdin to decline overwrite
	restore := mockStdin(t, "n\n")
	defer restore()

	err := cmdInit([]string{})
	if err != nil {
		t.Errorf("cmdInit should not error when user declines: %v", err)
	}
}

// Test cmdInit creates new config with local backend
func TestCmdInitLocalBackend(t *testing.T) {
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

	// Mock stdin: choose local backend, no peers, yes to example transforms
	restore := mockStdin(t, "local\nn\ny\n")
	defer restore()

	err := cmdInit([]string{})
	if err != nil {
		t.Errorf("cmdInit should not error: %v", err)
	}

	// Verify config was created
	configPath := tmpDir + "/pipeboard/config.yaml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file should exist: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "backend: local") {
		t.Error("config should contain local backend")
	}
	if !strings.Contains(content, "fx:") {
		t.Error("config should contain fx section (example transforms)")
	}
}

// Test cmdInit with none backend
func TestCmdInitNoneBackend(t *testing.T) {
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

	// Mock stdin: choose none backend, no peers, no transforms
	restore := mockStdin(t, "none\nn\nn\n")
	defer restore()

	err := cmdInit([]string{})
	if err != nil {
		t.Errorf("cmdInit should not error: %v", err)
	}

	configPath := tmpDir + "/pipeboard/config.yaml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file should exist: %v", err)
	}

	if !strings.Contains(string(data), "backend: none") {
		t.Error("config should contain none backend")
	}
}

// Note: Multi-prompt cmdInit tests are skipped because the init.go prompt functions
// each create their own bufio.Reader, which causes buffering issues with piped input.
// The individual prompt functions are tested above with single-input tests.
