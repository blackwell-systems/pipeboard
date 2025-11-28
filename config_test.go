package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigPath(t *testing.T) {
	// Test with PIPEBOARD_CONFIG set
	t.Run("env override", func(t *testing.T) {
		orig := os.Getenv("PIPEBOARD_CONFIG")
		defer func() {
			if orig != "" {
				_ = os.Setenv("PIPEBOARD_CONFIG", orig)
			} else {
				_ = os.Unsetenv("PIPEBOARD_CONFIG")
			}
		}()

		_ = os.Setenv("PIPEBOARD_CONFIG", "/custom/path/config.yaml")
		if p := configPath(); p != "/custom/path/config.yaml" {
			t.Errorf("expected custom path, got %s", p)
		}
	})

	// Test with XDG_CONFIG_HOME set
	t.Run("XDG_CONFIG_HOME", func(t *testing.T) {
		origConfig := os.Getenv("PIPEBOARD_CONFIG")
		origXDG := os.Getenv("XDG_CONFIG_HOME")
		defer func() {
			if origConfig != "" {
				_ = os.Setenv("PIPEBOARD_CONFIG", origConfig)
			} else {
				_ = os.Unsetenv("PIPEBOARD_CONFIG")
			}
			if origXDG != "" {
				_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
			} else {
				_ = os.Unsetenv("XDG_CONFIG_HOME")
			}
		}()

		_ = os.Unsetenv("PIPEBOARD_CONFIG")
		_ = os.Setenv("XDG_CONFIG_HOME", "/xdg/config")

		expected := "/xdg/config/pipeboard/config.yaml"
		if p := configPath(); p != expected {
			t.Errorf("expected %s, got %s", expected, p)
		}
	})
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.Peers == nil {
		t.Error("Peers should not be nil after defaults")
	}
}

func TestApplyDefaultsPeerRemoteCmd(t *testing.T) {
	cfg := &Config{
		Peers: map[string]PeerConfig{
			"dev": {SSH: "devbox"},
		},
	}
	applyDefaults(cfg)

	peer := cfg.Peers["dev"]
	if peer.RemoteCmd != "pipeboard" {
		t.Errorf("expected RemoteCmd 'pipeboard', got %s", peer.RemoteCmd)
	}
}

func restoreEnv(key, value string) {
	if value != "" {
		_ = os.Setenv(key, value)
	} else {
		_ = os.Unsetenv(key)
	}
}

func TestValidateSyncConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "nil sync",
			cfg:     Config{},
			wantErr: true,
		},
		{
			name: "empty backend",
			cfg: Config{
				Sync: &SyncConfig{},
			},
			wantErr: true,
		},
		{
			name: "backend none",
			cfg: Config{
				Sync: &SyncConfig{Backend: "none"},
			},
			wantErr: true,
		},
		{
			name: "unsupported backend",
			cfg: Config{
				Sync: &SyncConfig{Backend: "gcs"},
			},
			wantErr: true,
		},
		{
			name: "s3 without config",
			cfg: Config{
				Sync: &SyncConfig{Backend: "s3"},
			},
			wantErr: true,
		},
		{
			name: "s3 without bucket",
			cfg: Config{
				Sync: &SyncConfig{
					Backend: "s3",
					S3:      &S3Config{Region: "us-west-2"},
				},
			},
			wantErr: true,
		},
		{
			name: "s3 without region",
			cfg: Config{
				Sync: &SyncConfig{
					Backend: "s3",
					S3:      &S3Config{Bucket: "my-bucket"},
				},
			},
			wantErr: true,
		},
		{
			name: "valid s3 config",
			cfg: Config{
				Sync: &SyncConfig{
					Backend: "s3",
					S3: &S3Config{
						Bucket: "my-bucket",
						Region: "us-west-2",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSyncConfig(&tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSyncConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	// Point to a non-existent config file
	orig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", orig)

	_ = os.Setenv("PIPEBOARD_CONFIG", "/nonexistent/path/config.yaml")

	_, err := loadConfig()
	if err == nil {
		t.Error("expected error for missing config file")
	}
}

func TestLoadConfigValidFile(t *testing.T) {
	// Create a temporary config file with new format
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `version: 1
sync:
  backend: s3
  s3:
    bucket: test-bucket
    region: us-west-2
    prefix: test/
    sse: AES256
peers:
  dev:
    ssh: devbox
    remote_cmd: pb
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	orig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", orig)

	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error: %v", err)
	}

	if cfg.Sync.Backend != "s3" {
		t.Errorf("expected backend s3, got %s", cfg.Sync.Backend)
	}
	if cfg.Sync.S3.Bucket != "test-bucket" {
		t.Errorf("expected bucket test-bucket, got %s", cfg.Sync.S3.Bucket)
	}
	if cfg.Sync.S3.Region != "us-west-2" {
		t.Errorf("expected region us-west-2, got %s", cfg.Sync.S3.Region)
	}
	if cfg.Sync.S3.Prefix != "test/" {
		t.Errorf("expected prefix test/, got %s", cfg.Sync.S3.Prefix)
	}
	if cfg.Sync.S3.SSE != "AES256" {
		t.Errorf("expected SSE AES256, got %s", cfg.Sync.S3.SSE)
	}

	// Check peers
	peer, ok := cfg.Peers["dev"]
	if !ok {
		t.Fatal("expected peer 'dev' to exist")
	}
	if peer.SSH != "devbox" {
		t.Errorf("expected SSH 'devbox', got %s", peer.SSH)
	}
	if peer.RemoteCmd != "pb" {
		t.Errorf("expected RemoteCmd 'pb', got %s", peer.RemoteCmd)
	}
}

func TestLoadConfigLegacyFormat(t *testing.T) {
	// Create a temporary config file with legacy format
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `backend: s3
s3:
  bucket: legacy-bucket
  region: us-east-1
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	orig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", orig)

	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error: %v", err)
	}

	// Legacy format should be migrated to new format
	if cfg.Sync == nil {
		t.Fatal("Sync config should not be nil")
	}
	if cfg.Sync.Backend != "s3" {
		t.Errorf("expected backend s3, got %s", cfg.Sync.Backend)
	}
	if cfg.Sync.S3.Bucket != "legacy-bucket" {
		t.Errorf("expected bucket legacy-bucket, got %s", cfg.Sync.S3.Bucket)
	}
}

func TestGetPeer(t *testing.T) {
	cfg := &Config{
		Peers: map[string]PeerConfig{
			"dev": {SSH: "devbox", RemoteCmd: "pipeboard"},
			"mac": {SSH: "mac.local"},
		},
	}

	t.Run("existing peer", func(t *testing.T) {
		peer, err := cfg.getPeer("dev")
		if err != nil {
			t.Fatalf("getPeer() error: %v", err)
		}
		if peer.SSH != "devbox" {
			t.Errorf("expected SSH 'devbox', got %s", peer.SSH)
		}
		if peer.RemoteCmd != "pipeboard" {
			t.Errorf("expected RemoteCmd 'pipeboard', got %s", peer.RemoteCmd)
		}
	})

	t.Run("peer with default remote_cmd", func(t *testing.T) {
		peer, err := cfg.getPeer("mac")
		if err != nil {
			t.Fatalf("getPeer() error: %v", err)
		}
		if peer.RemoteCmd != "pipeboard" {
			t.Errorf("expected default RemoteCmd 'pipeboard', got %s", peer.RemoteCmd)
		}
	})

	t.Run("unknown peer", func(t *testing.T) {
		_, err := cfg.getPeer("unknown")
		if err == nil {
			t.Error("expected error for unknown peer")
		}
	})

	t.Run("empty peers", func(t *testing.T) {
		emptyCfg := &Config{}
		_, err := emptyCfg.getPeer("dev")
		if err == nil {
			t.Error("expected error for empty peers")
		}
	})
}

func TestLoadConfigForPeersMissingFile(t *testing.T) {
	orig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", orig)

	_ = os.Setenv("PIPEBOARD_CONFIG", "/nonexistent/path/config.yaml")

	_, err := loadConfigForPeers()
	if err == nil {
		t.Error("expected error for missing config file")
	}
}

func TestLoadConfigForPeersValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `peers:
  dev:
    ssh: devbox
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	orig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", orig)

	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	cfg, err := loadConfigForPeers()
	if err != nil {
		t.Fatalf("loadConfigForPeers() error: %v", err)
	}

	peer, ok := cfg.Peers["dev"]
	if !ok {
		t.Fatal("expected peer 'dev' to exist")
	}
	if peer.SSH != "devbox" {
		t.Errorf("expected SSH 'devbox', got %s", peer.SSH)
	}
}
