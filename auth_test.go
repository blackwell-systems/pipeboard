package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCmdSignupValidation tests the configuration validation in cmdSignup.
// Note: Full testing of cmdSignup is limited because it uses interactive password prompts.
func TestCmdSignupValidation(t *testing.T) {
	// Save and restore config path
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	t.Run("no config file", func(t *testing.T) {
		err := cmdSignup(nil)
		if err == nil {
			t.Error("expected error when no config file exists")
		}
	})

	t.Run("wrong backend type", func(t *testing.T) {
		// Create config with s3 backend instead of hosted
		configDir := filepath.Join(tmpDir, "pipeboard")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}

		configContent := `sync:
  backend: s3
  s3:
    bucket: test-bucket
    region: us-east-1
`
		if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := cmdSignup(nil)
		if err == nil {
			t.Error("expected error for non-hosted backend")
		}
	})

	t.Run("missing hosted url", func(t *testing.T) {
		configDir := filepath.Join(tmpDir, "pipeboard")
		configContent := `sync:
  backend: hosted
  hosted:
    email: test@example.com
`
		if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := cmdSignup(nil)
		if err == nil {
			t.Error("expected error when hosted.url is missing")
		}
	})
}

// TestCmdLoginValidation tests the configuration validation in cmdLogin.
func TestCmdLoginValidation(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	t.Run("no config file", func(t *testing.T) {
		err := cmdLogin(nil)
		if err == nil {
			t.Error("expected error when no config file exists")
		}
	})

	t.Run("wrong backend type", func(t *testing.T) {
		configDir := filepath.Join(tmpDir, "pipeboard")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}

		configContent := `sync:
  backend: s3
`
		if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := cmdLogin(nil)
		if err == nil {
			t.Error("expected error for non-hosted backend")
		}
	})

	t.Run("missing hosted url", func(t *testing.T) {
		configDir := filepath.Join(tmpDir, "pipeboard")
		configContent := `sync:
  backend: hosted
  hosted:
    email: test@example.com
`
		if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := cmdLogin(nil)
		if err == nil {
			t.Error("expected error when hosted.url is missing")
		}
	})
}

// TestCmdLogoutValidation tests the configuration validation in cmdLogout.
func TestCmdLogoutValidation(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	t.Run("no config file", func(t *testing.T) {
		err := cmdLogout(nil)
		if err == nil {
			t.Error("expected error when no config file exists")
		}
	})

	t.Run("no email configured", func(t *testing.T) {
		configDir := filepath.Join(tmpDir, "pipeboard")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}

		configContent := `sync:
  backend: hosted
  hosted:
    url: https://example.com
`
		if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := cmdLogout(nil)
		if err == nil {
			t.Error("expected error when email is not configured")
		}
	})

	t.Run("successful logout", func(t *testing.T) {
		configDir := filepath.Join(tmpDir, "pipeboard")
		configContent := `sync:
  backend: hosted
  hosted:
    url: https://example.com
    email: test@example.com
`
		if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Pre-store a token to clear
		if err := storeToken("test@example.com", "token-to-clear"); err != nil {
			t.Fatal(err)
		}

		err := cmdLogout(nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify token was cleared
		_, err = getStoredToken("test@example.com")
		if err == nil {
			t.Error("token should be cleared after logout")
		}
	})
}

// TestCmdLogoutNoSync tests logout when sync config is nil.
func TestCmdLogoutNoSync(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	configDir := filepath.Join(tmpDir, "pipeboard")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Config with no sync section
	configContent := `# empty config
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := cmdLogout(nil)
	if err == nil {
		t.Error("expected error when sync config is nil")
	}
}

// TestCmdSignupNoSync tests signup when sync config is nil.
func TestCmdSignupNoSync(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	configDir := filepath.Join(tmpDir, "pipeboard")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Config with no sync section
	configContent := `# empty config
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := cmdSignup(nil)
	if err == nil {
		t.Error("expected error when sync config is nil")
	}
}

// TestCmdLoginNoSync tests login when sync config is nil.
func TestCmdLoginNoSync(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	configDir := filepath.Join(tmpDir, "pipeboard")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Config with no sync section
	configContent := `# empty config
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := cmdLogin(nil)
	if err == nil {
		t.Error("expected error when sync config is nil")
	}
}
