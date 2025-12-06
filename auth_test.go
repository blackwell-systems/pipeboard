package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCmdSignup tests the signup command
func TestCmdSignup(t *testing.T) {
	t.Run("missing config", func(t *testing.T) {
		// Use a temp dir with no config
		originalXDG := os.Getenv("XDG_CONFIG_HOME")
		defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

		testDir := t.TempDir()
		os.Setenv("XDG_CONFIG_HOME", testDir)

		// Override configPath to use temp dir
		os.Setenv("PIPEBOARD_CONFIG", filepath.Join(testDir, "nonexistent.yaml"))
		defer os.Unsetenv("PIPEBOARD_CONFIG")

		err := cmdSignup([]string{})
		if err == nil {
			t.Error("expected error when config doesn't exist")
		}
	})

	t.Run("backend not hosted", func(t *testing.T) {
		testDir := t.TempDir()
		os.Setenv("XDG_CONFIG_HOME", testDir)
		defer os.Setenv("XDG_CONFIG_HOME", "")

		// Create config with S3 backend
		configPath := filepath.Join(testDir, "pipeboard", "config.yaml")
		os.MkdirAll(filepath.Dir(configPath), 0755)
		configContent := `sync:
  backend: s3
  s3:
    bucket: test
    region: us-east-1
`
		os.WriteFile(configPath, []byte(configContent), 0644)
		os.Setenv("PIPEBOARD_CONFIG", configPath)
		defer os.Unsetenv("PIPEBOARD_CONFIG")

		err := cmdSignup([]string{})
		if err == nil {
			t.Error("expected error when backend is not hosted")
		}
	})

	t.Run("hosted URL missing", func(t *testing.T) {
		testDir := t.TempDir()
		configPath := filepath.Join(testDir, "config.yaml")

		configContent := `sync:
  backend: hosted
  hosted:
    email: test@example.com
`
		os.WriteFile(configPath, []byte(configContent), 0644)
		os.Setenv("PIPEBOARD_CONFIG", configPath)
		defer os.Unsetenv("PIPEBOARD_CONFIG")

		err := cmdSignup([]string{})
		if err == nil {
			t.Error("expected error when hosted.url is missing")
		}
	})
}

// TestCmdLogin tests the login command
func TestCmdLogin(t *testing.T) {
	t.Run("missing config", func(t *testing.T) {
		originalXDG := os.Getenv("XDG_CONFIG_HOME")
		defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

		testDir := t.TempDir()
		os.Setenv("XDG_CONFIG_HOME", testDir)

		os.Setenv("PIPEBOARD_CONFIG", filepath.Join(testDir, "nonexistent.yaml"))
		defer os.Unsetenv("PIPEBOARD_CONFIG")

		err := cmdLogin([]string{})
		if err == nil {
			t.Error("expected error when config doesn't exist")
		}
	})

	t.Run("backend not hosted", func(t *testing.T) {
		testDir := t.TempDir()
		configPath := filepath.Join(testDir, "config.yaml")

		configContent := `sync:
  backend: local
`
		os.WriteFile(configPath, []byte(configContent), 0644)
		os.Setenv("PIPEBOARD_CONFIG", configPath)
		defer os.Unsetenv("PIPEBOARD_CONFIG")

		err := cmdLogin([]string{})
		if err == nil {
			t.Error("expected error when backend is not hosted")
		}
	})
}

// TestCmdLogout tests the logout command
func TestCmdLogout(t *testing.T) {
	t.Run("missing config", func(t *testing.T) {
		testDir := t.TempDir()
		os.Setenv("PIPEBOARD_CONFIG", filepath.Join(testDir, "nonexistent.yaml"))
		defer os.Unsetenv("PIPEBOARD_CONFIG")

		err := cmdLogout([]string{})
		if err == nil {
			t.Error("expected error when config doesn't exist")
		}
	})

	t.Run("email not configured", func(t *testing.T) {
		testDir := t.TempDir()
		configPath := filepath.Join(testDir, "config.yaml")

		// Config with no email
		configContent := `sync:
  backend: hosted
  hosted:
    url: https://example.com
`
		os.WriteFile(configPath, []byte(configContent), 0644)
		os.Setenv("PIPEBOARD_CONFIG", configPath)
		defer os.Unsetenv("PIPEBOARD_CONFIG")

		err := cmdLogout([]string{})
		if err == nil {
			t.Error("expected error when email is not configured")
		}
	})

	t.Run("successful logout", func(t *testing.T) {
		testDir := t.TempDir()
		os.Setenv("XDG_CONFIG_HOME", testDir)
		defer os.Setenv("XDG_CONFIG_HOME", "")

		email := "logout-cmd-test@example.com"

		// Store a token
		if err := storeToken(email, "test-token"); err != nil {
			t.Fatalf("failed to store token: %v", err)
		}

		configPath := filepath.Join(testDir, "pipeboard", "config.yaml")
		os.MkdirAll(filepath.Dir(configPath), 0755)
		configContent := `sync:
  backend: hosted
  hosted:
    url: https://example.com
    email: logout-cmd-test@example.com
`
		os.WriteFile(configPath, []byte(configContent), 0644)
		os.Setenv("PIPEBOARD_CONFIG", configPath)
		defer os.Unsetenv("PIPEBOARD_CONFIG")

		err := cmdLogout([]string{})
		if err != nil {
			t.Errorf("cmdLogout failed: %v", err)
		}

		// Verify token was cleared
		_, err = getStoredToken(email)
		if err == nil {
			t.Error("token should be cleared after logout")
		}
	})
}
