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

// Test fx config loading
func TestLoadConfigForFxMissingFile(t *testing.T) {
	orig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", orig)

	_ = os.Setenv("PIPEBOARD_CONFIG", "/nonexistent/path/config.yaml")

	// loadConfigForFx returns empty config for missing file (not an error)
	cfg, err := loadConfigForFx()
	if err != nil {
		t.Fatalf("loadConfigForFx() should not error on missing file: %v", err)
	}
	if cfg.Fx == nil {
		t.Error("Fx map should be initialized even for missing file")
	}
}

func TestLoadConfigForFxValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `fx:
  pretty-json:
    cmd: ["jq", "."]
    description: "Format JSON"
  strip-ansi:
    shell: "sed 's/\\x1b\\[[0-9;]*m//g'"
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	orig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", orig)

	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	cfg, err := loadConfigForFx()
	if err != nil {
		t.Fatalf("loadConfigForFx() error: %v", err)
	}

	if cfg.Fx == nil {
		t.Fatal("Fx map should not be nil")
	}
	if len(cfg.Fx) != 2 {
		t.Errorf("expected 2 fx transforms, got %d", len(cfg.Fx))
	}

	// Check pretty-json
	pj, ok := cfg.Fx["pretty-json"]
	if !ok {
		t.Fatal("expected 'pretty-json' transform to exist")
	}
	if len(pj.Cmd) != 2 || pj.Cmd[0] != "jq" {
		t.Errorf("unexpected Cmd for pretty-json: %v", pj.Cmd)
	}
	if pj.Description != "Format JSON" {
		t.Errorf("unexpected Description: %s", pj.Description)
	}

	// Check strip-ansi
	sa, ok := cfg.Fx["strip-ansi"]
	if !ok {
		t.Fatal("expected 'strip-ansi' transform to exist")
	}
	if sa.Shell == "" {
		t.Error("expected Shell to be set for strip-ansi")
	}
}

func TestGetFx(t *testing.T) {
	cfg := &Config{
		Fx: map[string]FxConfig{
			"pretty-json": {Cmd: []string{"jq", "."}, Description: "Format JSON"},
			"strip-ansi":  {Shell: "sed 's/foo/bar/g'"},
		},
	}

	t.Run("existing transform", func(t *testing.T) {
		fx, err := cfg.getFx("pretty-json")
		if err != nil {
			t.Fatalf("getFx() error: %v", err)
		}
		if len(fx.Cmd) != 2 {
			t.Errorf("expected Cmd length 2, got %d", len(fx.Cmd))
		}
	})

	t.Run("unknown transform", func(t *testing.T) {
		_, err := cfg.getFx("unknown")
		if err == nil {
			t.Error("expected error for unknown transform")
		}
	})

	t.Run("empty fx map", func(t *testing.T) {
		emptyCfg := &Config{}
		_, err := emptyCfg.getFx("any")
		if err == nil {
			t.Error("expected error for empty fx map")
		}
	})
}

func TestFxConfigGetCommand(t *testing.T) {
	t.Run("cmd syntax", func(t *testing.T) {
		fx := FxConfig{Cmd: []string{"jq", "."}}
		cmd := fx.getCommand()
		if len(cmd) != 2 || cmd[0] != "jq" {
			t.Errorf("unexpected command: %v", cmd)
		}
	})

	t.Run("shell syntax", func(t *testing.T) {
		fx := FxConfig{Shell: "sed 's/foo/bar/g'"}
		cmd := fx.getCommand()
		if len(cmd) != 3 {
			t.Fatalf("expected 3 args for shell command, got %d", len(cmd))
		}
		if cmd[0] != "sh" || cmd[1] != "-c" {
			t.Errorf("expected ['sh', '-c', ...], got %v", cmd)
		}
	})

	t.Run("shell takes precedence", func(t *testing.T) {
		// If both are set, shell should take precedence
		fx := FxConfig{
			Cmd:   []string{"jq", "."},
			Shell: "cat",
		}
		cmd := fx.getCommand()
		if cmd[0] != "sh" {
			t.Error("shell should take precedence over cmd")
		}
	})

	t.Run("empty config", func(t *testing.T) {
		fx := FxConfig{}
		cmd := fx.getCommand()
		if len(cmd) != 0 {
			t.Errorf("expected empty command, got %v", cmd)
		}
	})
}

func TestDefaultPeer(t *testing.T) {
	cfg := &Config{
		Defaults: &DefaultsConfig{
			Peer: "dev",
		},
		Peers: map[string]PeerConfig{
			"dev": {SSH: "devbox"},
		},
	}

	if cfg.Defaults.Peer != "dev" {
		t.Errorf("expected default peer 'dev', got %s", cfg.Defaults.Peer)
	}
}

func TestEncryptionConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `version: 1
sync:
  backend: s3
  encryption: aes256
  passphrase: test-secret
  ttl_days: 30
  s3:
    bucket: test-bucket
    region: us-west-2
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

	if cfg.Sync.Encryption != "aes256" {
		t.Errorf("expected encryption 'aes256', got %s", cfg.Sync.Encryption)
	}
	if cfg.Sync.Passphrase != "test-secret" {
		t.Errorf("expected passphrase 'test-secret', got %s", cfg.Sync.Passphrase)
	}
	if cfg.Sync.TTLDays != 30 {
		t.Errorf("expected TTLDays 30, got %d", cfg.Sync.TTLDays)
	}
}

func TestGetDefaultPeer(t *testing.T) {
	t.Run("with default set", func(t *testing.T) {
		cfg := &Config{
			Defaults: &DefaultsConfig{
				Peer: "dev",
			},
			Peers: map[string]PeerConfig{
				"dev": {SSH: "devbox"},
			},
		}

		peerName, err := cfg.getDefaultPeer()
		if err != nil {
			t.Fatalf("getDefaultPeer() error: %v", err)
		}
		if peerName != "dev" {
			t.Errorf("expected peer name 'dev', got %s", peerName)
		}
	})

	t.Run("no default set", func(t *testing.T) {
		cfg := &Config{
			Peers: map[string]PeerConfig{
				"dev": {SSH: "devbox"},
			},
		}

		_, err := cfg.getDefaultPeer()
		if err == nil {
			t.Error("expected error when no default peer is set")
		}
	})

	t.Run("empty default peer", func(t *testing.T) {
		cfg := &Config{
			Defaults: &DefaultsConfig{
				Peer: "",
			},
			Peers: map[string]PeerConfig{
				"dev": {SSH: "devbox"},
			},
		}

		_, err := cfg.getDefaultPeer()
		if err == nil {
			t.Error("expected error when default peer is empty")
		}
	})

	t.Run("nil defaults", func(t *testing.T) {
		cfg := &Config{
			Defaults: nil,
			Peers: map[string]PeerConfig{
				"dev": {SSH: "devbox"},
			},
		}

		_, err := cfg.getDefaultPeer()
		if err == nil {
			t.Error("expected error when defaults is nil")
		}
	})
}

func TestEnsureSyncS3(t *testing.T) {
	t.Run("nil sync creates both", func(t *testing.T) {
		cfg := &Config{}
		ensureSyncS3(cfg)
		if cfg.Sync == nil {
			t.Error("ensureSyncS3 should create Sync")
		}
		if cfg.Sync.S3 == nil {
			t.Error("ensureSyncS3 should create S3 config")
		}
	})

	t.Run("sync with nil S3", func(t *testing.T) {
		cfg := &Config{
			Sync: &SyncConfig{
				Backend: "s3",
			},
		}
		ensureSyncS3(cfg)
		if cfg.Sync.S3 == nil {
			t.Error("ensureSyncS3 should create S3 config")
		}
	})

	t.Run("sync with existing S3", func(t *testing.T) {
		cfg := &Config{
			Sync: &SyncConfig{
				Backend: "s3",
				S3: &S3Config{
					Bucket: "existing-bucket",
				},
			},
		}
		ensureSyncS3(cfg)
		if cfg.Sync.S3.Bucket != "existing-bucket" {
			t.Error("ensureSyncS3 should not overwrite existing S3 config")
		}
	})
}

func TestLoadConfigWithEnvOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `version: 1
sync:
  backend: s3
  s3:
    bucket: config-bucket
    region: us-west-2
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Save original env vars
	origConfig := os.Getenv("PIPEBOARD_CONFIG")
	origBucket := os.Getenv("PIPEBOARD_S3_BUCKET")
	origRegion := os.Getenv("PIPEBOARD_S3_REGION")
	defer func() {
		restoreEnv("PIPEBOARD_CONFIG", origConfig)
		restoreEnv("PIPEBOARD_S3_BUCKET", origBucket)
		restoreEnv("PIPEBOARD_S3_REGION", origRegion)
	}()

	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)
	_ = os.Setenv("PIPEBOARD_S3_BUCKET", "env-bucket")
	_ = os.Setenv("PIPEBOARD_S3_REGION", "eu-west-1")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error: %v", err)
	}

	// Env vars should override config file
	if cfg.Sync.S3.Bucket != "env-bucket" {
		t.Errorf("expected bucket 'env-bucket', got %s", cfg.Sync.S3.Bucket)
	}
	if cfg.Sync.S3.Region != "eu-west-1" {
		t.Errorf("expected region 'eu-west-1', got %s", cfg.Sync.S3.Region)
	}
}

func TestApplyBackendEnv(t *testing.T) {
	t.Run("sets backend when env var is set", func(t *testing.T) {
		orig := os.Getenv("PIPEBOARD_BACKEND")
		defer restoreEnv("PIPEBOARD_BACKEND", orig)

		_ = os.Setenv("PIPEBOARD_BACKEND", "s3")

		cfg := &Config{}
		applyBackendEnv(cfg)

		if cfg.Sync == nil {
			t.Fatal("Sync should be created")
		}
		if cfg.Sync.Backend != "s3" {
			t.Errorf("expected backend 's3', got %s", cfg.Sync.Backend)
		}
	})

	t.Run("does nothing when env var not set", func(t *testing.T) {
		orig := os.Getenv("PIPEBOARD_BACKEND")
		defer restoreEnv("PIPEBOARD_BACKEND", orig)

		_ = os.Unsetenv("PIPEBOARD_BACKEND")

		cfg := &Config{}
		applyBackendEnv(cfg)

		if cfg.Sync != nil {
			t.Error("Sync should remain nil when env not set")
		}
	})

	t.Run("uses existing sync when present", func(t *testing.T) {
		orig := os.Getenv("PIPEBOARD_BACKEND")
		defer restoreEnv("PIPEBOARD_BACKEND", orig)

		_ = os.Setenv("PIPEBOARD_BACKEND", "s3")

		cfg := &Config{
			Sync: &SyncConfig{
				Passphrase: "existing-passphrase",
			},
		}
		applyBackendEnv(cfg)

		if cfg.Sync.Backend != "s3" {
			t.Errorf("expected backend 's3', got %s", cfg.Sync.Backend)
		}
		if cfg.Sync.Passphrase != "existing-passphrase" {
			t.Error("existing config should be preserved")
		}
	})
}

func TestApplyS3EnvAllVars(t *testing.T) {
	// Save all original env vars
	envVars := []string{
		"PIPEBOARD_S3_BUCKET",
		"PIPEBOARD_S3_REGION",
		"PIPEBOARD_S3_PREFIX",
		"PIPEBOARD_S3_PROFILE",
		"PIPEBOARD_S3_SSE",
	}
	origVals := make(map[string]string)
	for _, v := range envVars {
		origVals[v] = os.Getenv(v)
	}
	defer func() {
		for k, v := range origVals {
			restoreEnv(k, v)
		}
	}()

	// Set all env vars
	_ = os.Setenv("PIPEBOARD_S3_BUCKET", "test-bucket")
	_ = os.Setenv("PIPEBOARD_S3_REGION", "us-east-1")
	_ = os.Setenv("PIPEBOARD_S3_PREFIX", "prefix/")
	_ = os.Setenv("PIPEBOARD_S3_PROFILE", "test-profile")
	_ = os.Setenv("PIPEBOARD_S3_SSE", "AES256")

	cfg := &Config{}
	applyS3Env(cfg)

	if cfg.Sync == nil || cfg.Sync.S3 == nil {
		t.Fatal("Sync and S3 should be created")
	}
	if cfg.Sync.S3.Bucket != "test-bucket" {
		t.Errorf("expected bucket 'test-bucket', got %s", cfg.Sync.S3.Bucket)
	}
	if cfg.Sync.S3.Region != "us-east-1" {
		t.Errorf("expected region 'us-east-1', got %s", cfg.Sync.S3.Region)
	}
	if cfg.Sync.S3.Prefix != "prefix/" {
		t.Errorf("expected prefix 'prefix/', got %s", cfg.Sync.S3.Prefix)
	}
	if cfg.Sync.S3.Profile != "test-profile" {
		t.Errorf("expected profile 'test-profile', got %s", cfg.Sync.S3.Profile)
	}
	if cfg.Sync.S3.SSE != "AES256" {
		t.Errorf("expected SSE 'AES256', got %s", cfg.Sync.S3.SSE)
	}
	if cfg.Sync.Backend != "s3" {
		t.Errorf("expected backend 's3', got %s", cfg.Sync.Backend)
	}
}

func TestGetPeerProd(t *testing.T) {
	cfg := &Config{
		Peers: map[string]PeerConfig{
			"dev":  {SSH: "devbox"},
			"prod": {SSH: "prod-server"},
		},
	}

	peer, err := cfg.getPeer("prod")
	if err != nil {
		t.Fatalf("getPeer() error: %v", err)
	}
	if peer.SSH != "prod-server" {
		t.Errorf("expected SSH 'prod-server', got %s", peer.SSH)
	}
}

func TestValidateSyncConfigNilS3(t *testing.T) {
	cfg := &Config{
		Sync: &SyncConfig{
			Backend: "s3",
			S3:      nil,
		},
	}
	err := validateSyncConfig(cfg)
	if err == nil {
		t.Error("expected error for nil S3 config with s3 backend")
	}
}

// TestGetPeerMissingSSH tests getPeer when SSH field is empty
func TestGetPeerMissingSSH(t *testing.T) {
	cfg := &Config{
		Peers: map[string]PeerConfig{
			"broken": {SSH: "", RemoteCmd: "pipeboard"},
		},
	}

	_, err := cfg.getPeer("broken")
	if err == nil {
		t.Error("expected error for peer with missing SSH")
	}
	if err.Error() != "peer \"broken\" is missing 'ssh' field" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestGetFxMissingCmdAndShell tests getFx when transform has neither cmd nor shell
func TestGetFxMissingCmdAndShell(t *testing.T) {
	cfg := &Config{
		Fx: map[string]FxConfig{
			"broken": {Description: "No command defined"},
		},
	}

	_, err := cfg.getFx("broken")
	if err == nil {
		t.Error("expected error for transform with no cmd or shell")
	}
	if err.Error() != "transform \"broken\" has no 'cmd' or 'shell' defined" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestLoadConfigForFxInvalidYAML tests loadConfigForFx with invalid YAML
func TestLoadConfigForFxInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Write invalid YAML
	if err := os.WriteFile(configFile, []byte("invalid: [yaml"), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	orig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", orig)

	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	_, err := loadConfigForFx()
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

// TestLoadConfigForPeersInvalidYAML tests loadConfigForPeers with invalid YAML
func TestLoadConfigForPeersInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Write invalid YAML
	if err := os.WriteFile(configFile, []byte("invalid: [yaml"), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	orig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", orig)

	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	_, err := loadConfigForPeers()
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

// TestLoadConfigInvalidYAMLParsing tests loadConfig with invalid YAML
func TestLoadConfigInvalidYAMLParsing(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Write invalid YAML
	if err := os.WriteFile(configFile, []byte("invalid: [yaml"), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	orig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", orig)

	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	_, err := loadConfig()
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

// TestValidateSyncConfigLocalBackend tests local backend validation
func TestValidateSyncConfigLocalBackend(t *testing.T) {
	cfg := &Config{
		Sync: &SyncConfig{
			Backend: "local",
		},
	}

	err := validateSyncConfig(cfg)
	if err != nil {
		t.Errorf("local backend should be valid: %v", err)
	}

	// Should auto-create Local config
	if cfg.Sync.Local == nil {
		t.Error("local backend should auto-create Local config")
	}
}

// TestResolveAlias tests the alias resolution functionality
func TestResolveAlias(t *testing.T) {
	t.Run("resolves existing alias", func(t *testing.T) {
		cfg := &Config{
			Aliases: map[string]string{
				"k":   "kube-config",
				"p":   "prod-secrets",
				"aws": "aws-credentials",
			},
		}

		if got := cfg.resolveAlias("k"); got != "kube-config" {
			t.Errorf("resolveAlias(k) = %q, want %q", got, "kube-config")
		}
		if got := cfg.resolveAlias("p"); got != "prod-secrets" {
			t.Errorf("resolveAlias(p) = %q, want %q", got, "prod-secrets")
		}
		if got := cfg.resolveAlias("aws"); got != "aws-credentials" {
			t.Errorf("resolveAlias(aws) = %q, want %q", got, "aws-credentials")
		}
	})

	t.Run("returns original for non-existent alias", func(t *testing.T) {
		cfg := &Config{
			Aliases: map[string]string{
				"k": "kube-config",
			},
		}

		if got := cfg.resolveAlias("other"); got != "other" {
			t.Errorf("resolveAlias(other) = %q, want %q", got, "other")
		}
	})

	t.Run("returns original when aliases is nil", func(t *testing.T) {
		cfg := &Config{
			Aliases: nil,
		}

		if got := cfg.resolveAlias("anything"); got != "anything" {
			t.Errorf("resolveAlias(anything) = %q, want %q", got, "anything")
		}
	})

	t.Run("returns original when aliases is empty", func(t *testing.T) {
		cfg := &Config{
			Aliases: map[string]string{},
		}

		if got := cfg.resolveAlias("test"); got != "test" {
			t.Errorf("resolveAlias(test) = %q, want %q", got, "test")
		}
	})
}

// TestLoadConfigForAliases tests loading config for alias resolution
func TestLoadConfigForAliases(t *testing.T) {
	t.Run("loads aliases from config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")

		configContent := `aliases:
  k: kube-config
  p: prod-secrets
  db: database-creds
`
		if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		orig := os.Getenv("PIPEBOARD_CONFIG")
		defer restoreEnv("PIPEBOARD_CONFIG", orig)

		_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

		cfg, err := loadConfigForAliases()
		if err != nil {
			t.Fatalf("loadConfigForAliases() error: %v", err)
		}

		if cfg.Aliases == nil {
			t.Fatal("Aliases should not be nil")
		}
		if len(cfg.Aliases) != 3 {
			t.Errorf("expected 3 aliases, got %d", len(cfg.Aliases))
		}
		if cfg.Aliases["k"] != "kube-config" {
			t.Errorf("expected alias k=kube-config, got %s", cfg.Aliases["k"])
		}
	})

	t.Run("returns empty config for missing file", func(t *testing.T) {
		orig := os.Getenv("PIPEBOARD_CONFIG")
		defer restoreEnv("PIPEBOARD_CONFIG", orig)

		_ = os.Setenv("PIPEBOARD_CONFIG", "/nonexistent/path/config.yaml")

		cfg, err := loadConfigForAliases()
		if err != nil {
			t.Fatalf("loadConfigForAliases() should not error on missing file: %v", err)
		}
		if cfg == nil {
			t.Error("Config should not be nil")
		}
	})

	t.Run("errors on invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")

		if err := os.WriteFile(configFile, []byte("invalid: [yaml"), 0644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		orig := os.Getenv("PIPEBOARD_CONFIG")
		defer restoreEnv("PIPEBOARD_CONFIG", orig)

		_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

		_, err := loadConfigForAliases()
		if err == nil {
			t.Error("expected error for invalid YAML")
		}
	})
}

// TestLoadConfigWithAliases tests full config loading includes aliases
func TestLoadConfigWithAliases(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `version: 1
sync:
  backend: local
aliases:
  k: kube-config
  p: prod-secrets
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

	if cfg.Aliases == nil {
		t.Fatal("Aliases should not be nil")
	}
	if cfg.Aliases["k"] != "kube-config" {
		t.Errorf("expected alias k=kube-config, got %s", cfg.Aliases["k"])
	}
	if cfg.Aliases["p"] != "prod-secrets" {
		t.Errorf("expected alias p=prod-secrets, got %s", cfg.Aliases["p"])
	}
}
