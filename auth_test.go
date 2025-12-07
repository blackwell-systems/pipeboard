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
		defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

		testDir := t.TempDir()
		_ = os.Setenv("XDG_CONFIG_HOME", testDir)

		// Override configPath to use temp dir
		_ = os.Setenv("PIPEBOARD_CONFIG", filepath.Join(testDir, "nonexistent.yaml"))
		defer func() { _ = os.Unsetenv("PIPEBOARD_CONFIG") }()

		err := cmdSignup([]string{})
		if err == nil {
			t.Error("expected error when config doesn't exist")
		}
	})

	t.Run("backend not hosted", func(t *testing.T) {
		testDir := t.TempDir()
		_ = os.Setenv("XDG_CONFIG_HOME", testDir)
		defer func() { _ = os.Setenv("XDG_CONFIG_HOME", "") }()

		// Create config with S3 backend
		configPath := filepath.Join(testDir, "pipeboard", "config.yaml")
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			t.Fatal(err)
		}
		configContent := `sync:
  backend: s3
  s3:
    bucket: test
    region: us-east-1
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}
		_ = os.Setenv("PIPEBOARD_CONFIG", configPath)
		defer func() { _ = os.Unsetenv("PIPEBOARD_CONFIG") }()

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
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}
		_ = os.Setenv("PIPEBOARD_CONFIG", configPath)
		defer func() { _ = os.Unsetenv("PIPEBOARD_CONFIG") }()

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
		defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

		testDir := t.TempDir()
		_ = os.Setenv("XDG_CONFIG_HOME", testDir)

		_ = os.Setenv("PIPEBOARD_CONFIG", filepath.Join(testDir, "nonexistent.yaml"))
		defer func() { _ = os.Unsetenv("PIPEBOARD_CONFIG") }()

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
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}
		_ = os.Setenv("PIPEBOARD_CONFIG", configPath)
		defer func() { _ = os.Unsetenv("PIPEBOARD_CONFIG") }()

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
		_ = os.Setenv("PIPEBOARD_CONFIG", filepath.Join(testDir, "nonexistent.yaml"))
		defer func() { _ = os.Unsetenv("PIPEBOARD_CONFIG") }()

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
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}
		_ = os.Setenv("PIPEBOARD_CONFIG", configPath)
		defer func() { _ = os.Unsetenv("PIPEBOARD_CONFIG") }()

		err := cmdLogout([]string{})
		if err == nil {
			t.Error("expected error when email is not configured")
		}
	})

	t.Run("successful logout", func(t *testing.T) {
		testDir := t.TempDir()
		_ = os.Setenv("XDG_CONFIG_HOME", testDir)
		defer func() { _ = os.Setenv("XDG_CONFIG_HOME", "") }()

		email := "logout-cmd-test@example.com"

		// Store a token
		if err := storeToken(email, "test-token"); err != nil {
			t.Fatalf("failed to store token: %v", err)
		}

		configPath := filepath.Join(testDir, "pipeboard", "config.yaml")
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			t.Fatal(err)
		}
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
