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
				os.Setenv("PIPEBOARD_CONFIG", orig)
			} else {
				os.Unsetenv("PIPEBOARD_CONFIG")
			}
		}()

		os.Setenv("PIPEBOARD_CONFIG", "/custom/path/config.yaml")
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
				os.Setenv("PIPEBOARD_CONFIG", origConfig)
			} else {
				os.Unsetenv("PIPEBOARD_CONFIG")
			}
			if origXDG != "" {
				os.Setenv("XDG_CONFIG_HOME", origXDG)
			} else {
				os.Unsetenv("XDG_CONFIG_HOME")
			}
		}()

		os.Unsetenv("PIPEBOARD_CONFIG")
		os.Setenv("XDG_CONFIG_HOME", "/xdg/config")

		expected := "/xdg/config/pipeboard/config.yaml"
		if p := configPath(); p != expected {
			t.Errorf("expected %s, got %s", expected, p)
		}
	})
}

func TestApplyEnvOverrides(t *testing.T) {
	origBucket := os.Getenv("PIPEBOARD_S3_BUCKET")
	origRegion := os.Getenv("PIPEBOARD_S3_REGION")
	origPrefix := os.Getenv("PIPEBOARD_S3_PREFIX")
	origProfile := os.Getenv("PIPEBOARD_S3_PROFILE")
	origSSE := os.Getenv("PIPEBOARD_S3_SSE")
	origBackend := os.Getenv("PIPEBOARD_BACKEND")

	defer func() {
		restoreEnv("PIPEBOARD_S3_BUCKET", origBucket)
		restoreEnv("PIPEBOARD_S3_REGION", origRegion)
		restoreEnv("PIPEBOARD_S3_PREFIX", origPrefix)
		restoreEnv("PIPEBOARD_S3_PROFILE", origProfile)
		restoreEnv("PIPEBOARD_S3_SSE", origSSE)
		restoreEnv("PIPEBOARD_BACKEND", origBackend)
	}()

	// Set all env vars
	os.Setenv("PIPEBOARD_BACKEND", "s3")
	os.Setenv("PIPEBOARD_S3_BUCKET", "test-bucket")
	os.Setenv("PIPEBOARD_S3_REGION", "us-east-1")
	os.Setenv("PIPEBOARD_S3_PREFIX", "test/prefix/")
	os.Setenv("PIPEBOARD_S3_PROFILE", "test-profile")
	os.Setenv("PIPEBOARD_S3_SSE", "AES256")

	cfg := &Config{}
	applyEnvOverrides(cfg)

	if cfg.Backend != "s3" {
		t.Errorf("expected backend s3, got %s", cfg.Backend)
	}
	if cfg.S3 == nil {
		t.Fatal("S3 config should not be nil")
	}
	if cfg.S3.Bucket != "test-bucket" {
		t.Errorf("expected bucket test-bucket, got %s", cfg.S3.Bucket)
	}
	if cfg.S3.Region != "us-east-1" {
		t.Errorf("expected region us-east-1, got %s", cfg.S3.Region)
	}
	if cfg.S3.Prefix != "test/prefix/" {
		t.Errorf("expected prefix test/prefix/, got %s", cfg.S3.Prefix)
	}
	if cfg.S3.Profile != "test-profile" {
		t.Errorf("expected profile test-profile, got %s", cfg.S3.Profile)
	}
	if cfg.S3.SSE != "AES256" {
		t.Errorf("expected SSE AES256, got %s", cfg.S3.SSE)
	}
}

func restoreEnv(key, value string) {
	if value != "" {
		os.Setenv(key, value)
	} else {
		os.Unsetenv(key)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "empty backend",
			cfg:     Config{},
			wantErr: true,
		},
		{
			name:    "unsupported backend",
			cfg:     Config{Backend: "gcs"},
			wantErr: true,
		},
		{
			name: "s3 without config",
			cfg: Config{
				Backend: "s3",
			},
			wantErr: true,
		},
		{
			name: "s3 without bucket",
			cfg: Config{
				Backend: "s3",
				S3:      &S3Config{Region: "us-west-2"},
			},
			wantErr: true,
		},
		{
			name: "s3 without region",
			cfg: Config{
				Backend: "s3",
				S3:      &S3Config{Bucket: "my-bucket"},
			},
			wantErr: true,
		},
		{
			name: "valid s3 config",
			cfg: Config{
				Backend: "s3",
				S3: &S3Config{
					Bucket: "my-bucket",
					Region: "us-west-2",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(&tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	// Point to a non-existent config file
	orig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", orig)

	os.Setenv("PIPEBOARD_CONFIG", "/nonexistent/path/config.yaml")

	_, err := loadConfig()
	if err == nil {
		t.Error("expected error for missing config file")
	}
}

func TestLoadConfigValidFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `backend: s3
s3:
  bucket: test-bucket
  region: us-west-2
  prefix: test/
  sse: AES256
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	orig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", orig)

	os.Setenv("PIPEBOARD_CONFIG", configFile)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error: %v", err)
	}

	if cfg.Backend != "s3" {
		t.Errorf("expected backend s3, got %s", cfg.Backend)
	}
	if cfg.S3.Bucket != "test-bucket" {
		t.Errorf("expected bucket test-bucket, got %s", cfg.S3.Bucket)
	}
	if cfg.S3.Region != "us-west-2" {
		t.Errorf("expected region us-west-2, got %s", cfg.S3.Region)
	}
	if cfg.S3.Prefix != "test/" {
		t.Errorf("expected prefix test/, got %s", cfg.S3.Prefix)
	}
	if cfg.S3.SSE != "AES256" {
		t.Errorf("expected SSE AES256, got %s", cfg.S3.SSE)
	}
}
