package main

import (
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
