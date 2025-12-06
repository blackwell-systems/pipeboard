package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetMachineKey(t *testing.T) {
	key, err := getMachineKey()
	if err != nil {
		t.Fatalf("getMachineKey failed: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("expected 32-byte key, got %d bytes", len(key))
	}

	// Should be deterministic
	key2, err := getMachineKey()
	if err != nil {
		t.Fatalf("getMachineKey second call failed: %v", err)
	}
	if string(key) != string(key2) {
		t.Error("getMachineKey should return consistent key")
	}
}

func TestEncryptDecryptToken(t *testing.T) {
	key, err := getMachineKey()
	if err != nil {
		t.Fatalf("getMachineKey failed: %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{"simple token", "test-token-123"},
		{"jwt-like", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"},
		{"empty token", ""},
		{"unicode", "token-🔐-secret"},
		{"long token", string(make([]byte, 1000))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := encryptToken(tt.token, key)
			if err != nil {
				t.Fatalf("encryptToken failed: %v", err)
			}

			// Encrypted should be different from original
			if encrypted == tt.token && tt.token != "" {
				t.Error("encrypted token should differ from original")
			}

			decrypted, err := decryptToken(encrypted, key)
			if err != nil {
				t.Fatalf("decryptToken failed: %v", err)
			}

			if decrypted != tt.token {
				t.Errorf("decrypted token mismatch: got %q, want %q", decrypted, tt.token)
			}
		})
	}
}

func TestDecryptTokenInvalidData(t *testing.T) {
	key, _ := getMachineKey()

	tests := []struct {
		name      string
		encrypted string
	}{
		{"invalid base64", "not-valid-base64!!!"},
		{"too short", "YWJj"}, // "abc" in base64, too short for nonce
		{"tampered ciphertext", "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3ODk="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decryptToken(tt.encrypted, key)
			if err == nil {
				t.Error("expected error for invalid encrypted data")
			}
		})
	}
}

func TestEncryptTokenWithDifferentKeys(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 1 // Different key

	token := "test-token"

	encrypted, err := encryptToken(token, key1)
	if err != nil {
		t.Fatalf("encryptToken failed: %v", err)
	}

	// Should fail to decrypt with different key
	_, err = decryptToken(encrypted, key2)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestTokenStoreFileOperations(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	email := "test@example.com"
	token := "test-jwt-token-12345"

	// Store token
	err := storeTokenFile(email, token)
	if err != nil {
		t.Fatalf("storeTokenFile failed: %v", err)
	}

	// Verify file exists
	tokenPath := filepath.Join(tmpDir, "pipeboard", ".tokens")
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		t.Error("token file should exist after store")
	}

	// Retrieve token
	retrieved, err := getTokenFile(email)
	if err != nil {
		t.Fatalf("getTokenFile failed: %v", err)
	}
	if retrieved != token {
		t.Errorf("retrieved token mismatch: got %q, want %q", retrieved, token)
	}

	// Clear token
	err = clearTokenFile(email)
	if err != nil {
		t.Fatalf("clearTokenFile failed: %v", err)
	}

	// Should not find cleared token
	_, err = getTokenFile(email)
	if err == nil {
		t.Error("expected error after clearing token")
	}
}

func TestTokenStoreMultipleEmails(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	emails := map[string]string{
		"user1@example.com": "token-for-user1",
		"user2@example.com": "token-for-user2",
		"user3@example.com": "token-for-user3",
	}

	// Store all tokens
	for email, token := range emails {
		if err := storeTokenFile(email, token); err != nil {
			t.Fatalf("storeTokenFile failed for %s: %v", email, err)
		}
	}

	// Verify all tokens
	for email, expectedToken := range emails {
		retrieved, err := getTokenFile(email)
		if err != nil {
			t.Errorf("getTokenFile failed for %s: %v", email, err)
			continue
		}
		if retrieved != expectedToken {
			t.Errorf("token mismatch for %s: got %q, want %q", email, retrieved, expectedToken)
		}
	}

	// Clear one token
	if err := clearTokenFile("user2@example.com"); err != nil {
		t.Fatalf("clearTokenFile failed: %v", err)
	}

	// user2 should be gone, others should remain
	if _, err := getTokenFile("user2@example.com"); err == nil {
		t.Error("expected error for cleared user2 token")
	}
	if _, err := getTokenFile("user1@example.com"); err != nil {
		t.Error("user1 token should still exist")
	}
	if _, err := getTokenFile("user3@example.com"); err != nil {
		t.Error("user3 token should still exist")
	}
}

func TestLoadTokenStoreNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	store, err := loadTokenStore()
	if err != nil {
		t.Fatalf("loadTokenStore failed: %v", err)
	}
	if store == nil {
		t.Fatal("store should not be nil")
	}
	if store.Tokens == nil {
		t.Error("Tokens map should be initialized")
	}
	if len(store.Tokens) != 0 {
		t.Error("Tokens map should be empty for non-existent file")
	}
}

func TestLoadTokenStoreInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	// Create invalid JSON file
	tokenDir := filepath.Join(tmpDir, "pipeboard")
	if err := os.MkdirAll(tokenDir, 0700); err != nil {
		t.Fatal(err)
	}
	tokenPath := filepath.Join(tokenDir, ".tokens")
	if err := os.WriteFile(tokenPath, []byte("invalid json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := loadTokenStore()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSaveTokenStorePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	store := &tokenStore{
		Tokens: map[string]string{
			"test@example.com": "encrypted-token",
		},
	}

	if err := saveTokenStore(store); err != nil {
		t.Fatalf("saveTokenStore failed: %v", err)
	}

	// Check file permissions (should be 0600)
	tokenPath := filepath.Join(tmpDir, "pipeboard", ".tokens")
	info, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("token file permissions should be 0600, got %o", perm)
	}
}

func TestTokenFilePath(t *testing.T) {
	tests := []struct {
		name           string
		xdgConfigHome  string
		expectedSuffix string
	}{
		{
			name:           "with XDG_CONFIG_HOME",
			xdgConfigHome:  "/custom/config",
			expectedSuffix: "/custom/config/pipeboard/.tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalXDG := os.Getenv("XDG_CONFIG_HOME")
			if tt.xdgConfigHome != "" {
				_ = os.Setenv("XDG_CONFIG_HOME", tt.xdgConfigHome)
			} else {
				_ = os.Unsetenv("XDG_CONFIG_HOME")
			}
			defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

			path, err := tokenFilePath()
			if err != nil {
				// Skip if we can't determine home directory
				if tt.xdgConfigHome != "" {
					t.Fatalf("tokenFilePath failed: %v", err)
				}
				return
			}

			if tt.xdgConfigHome != "" && path != tt.expectedSuffix {
				t.Errorf("path mismatch: got %q, want %q", path, tt.expectedSuffix)
			}
		})
	}
}

func TestGetTokenFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	_, err := getTokenFile("nonexistent@example.com")
	if err == nil {
		t.Error("expected error for non-existent email")
	}
}

func TestTokenStoreUpdateExisting(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	email := "test@example.com"
	token1 := "first-token"
	token2 := "second-token"

	// Store first token
	if err := storeTokenFile(email, token1); err != nil {
		t.Fatalf("storeTokenFile failed: %v", err)
	}

	// Update with second token
	if err := storeTokenFile(email, token2); err != nil {
		t.Fatalf("storeTokenFile update failed: %v", err)
	}

	// Should get the updated token
	retrieved, err := getTokenFile(email)
	if err != nil {
		t.Fatalf("getTokenFile failed: %v", err)
	}
	if retrieved != token2 {
		t.Errorf("expected updated token %q, got %q", token2, retrieved)
	}
}

func TestTokenStoreNilTokensMap(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	// Create a token file with null Tokens field
	tokenDir := filepath.Join(tmpDir, "pipeboard")
	if err := os.MkdirAll(tokenDir, 0700); err != nil {
		t.Fatal(err)
	}
	tokenPath := filepath.Join(tokenDir, ".tokens")
	if err := os.WriteFile(tokenPath, []byte(`{"tokens": null}`), 0600); err != nil {
		t.Fatal(err)
	}

	store, err := loadTokenStore()
	if err != nil {
		t.Fatalf("loadTokenStore failed: %v", err)
	}
	if store.Tokens == nil {
		t.Error("Tokens map should be initialized even when JSON has null")
	}
}

func TestStoreTokenClearToken(t *testing.T) {
	// This test uses the high-level storeToken/getStoredToken/clearToken
	// On non-darwin systems, these delegate to file-based storage
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	email := "highlevel@example.com"
	token := "highlevel-token"

	// Store
	if err := storeToken(email, token); err != nil {
		t.Fatalf("storeToken failed: %v", err)
	}

	// Get
	retrieved, err := getStoredToken(email)
	if err != nil {
		t.Fatalf("getStoredToken failed: %v", err)
	}
	if retrieved != token {
		t.Errorf("token mismatch: got %q, want %q", retrieved, token)
	}

	// Clear
	if err := clearToken(email); err != nil {
		t.Fatalf("clearToken failed: %v", err)
	}

	// Verify cleared
	_, err = getStoredToken(email)
	if err == nil {
		t.Error("expected error after clearing token")
	}
}

func TestTokenStoreJSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	// Store a token
	if err := storeTokenFile("test@example.com", "token"); err != nil {
		t.Fatalf("storeTokenFile failed: %v", err)
	}

	// Read raw file and verify it's valid JSON
	tokenPath := filepath.Join(tmpDir, "pipeboard", ".tokens")
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var store tokenStore
	if err := json.Unmarshal(data, &store); err != nil {
		t.Errorf("stored data is not valid JSON: %v", err)
	}

	// Verify structure
	if store.Tokens == nil {
		t.Error("Tokens map should exist in JSON")
	}
	if _, ok := store.Tokens["test@example.com"]; !ok {
		t.Error("token entry should exist for email")
	}
}

// TestEncryptTokenInvalidKeySize tests that encryption fails with invalid key sizes.
func TestEncryptTokenInvalidKeySize(t *testing.T) {
	// AES requires 16, 24, or 32 byte keys
	invalidKeys := [][]byte{
		make([]byte, 15), // too short
		make([]byte, 17), // invalid size
		make([]byte, 31), // invalid size
		make([]byte, 33), // too long
	}

	for _, key := range invalidKeys {
		_, err := encryptToken("test-token", key)
		if err == nil {
			t.Errorf("expected error for key size %d", len(key))
		}
	}
}

// TestDecryptTokenInvalidKeySize tests that decryption fails with invalid key sizes.
func TestDecryptTokenInvalidKeySize(t *testing.T) {
	// First encrypt with a valid key
	validKey := make([]byte, 32)
	encrypted, err := encryptToken("test-token", validKey)
	if err != nil {
		t.Fatalf("encryptToken failed: %v", err)
	}

	// Try to decrypt with invalid keys
	invalidKeys := [][]byte{
		make([]byte, 15),
		make([]byte, 17),
	}

	for _, key := range invalidKeys {
		_, err := decryptToken(encrypted, key)
		if err == nil {
			t.Errorf("expected error for key size %d", len(key))
		}
	}
}

// TestTokenFilePathNoXDG tests tokenFilePath when XDG_CONFIG_HOME is unset.
func TestTokenFilePathNoXDG(t *testing.T) {
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Unsetenv("XDG_CONFIG_HOME")
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	path, err := tokenFilePath()
	// Should not error - falls back to HOME
	if err != nil {
		t.Fatalf("tokenFilePath failed: %v", err)
	}
	// Should contain pipeboard and .tokens
	if !filepath.IsAbs(path) {
		t.Error("path should be absolute")
	}
}

// TestClearTokenFileNonExistent tests clearing a token that doesn't exist.
func TestClearTokenFileNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	// Clear a non-existent token should not error (it's already gone)
	err := clearTokenFile("nonexistent@example.com")
	if err != nil {
		t.Errorf("clearTokenFile should not error for non-existent token: %v", err)
	}
}

// TestGetMachineKeyEnvironmentVariations tests getMachineKey with different env vars.
func TestGetMachineKeyEnvironmentVariations(t *testing.T) {
	// Test with USER env var
	originalUser := os.Getenv("USER")
	_ = os.Setenv("USER", "testuser")
	defer func() { _ = os.Setenv("USER", originalUser) }()

	key1, err := getMachineKey()
	if err != nil {
		t.Fatalf("getMachineKey failed: %v", err)
	}

	// Change user, key should change
	_ = os.Setenv("USER", "differentuser")
	key2, err := getMachineKey()
	if err != nil {
		t.Fatalf("getMachineKey failed: %v", err)
	}

	if string(key1) == string(key2) {
		t.Error("different users should produce different keys")
	}
}

// TestGetMachineKeyNoUser tests getMachineKey when USER env is not set.
func TestGetMachineKeyNoUser(t *testing.T) {
	originalUser := os.Getenv("USER")
	originalUsername := os.Getenv("USERNAME")
	_ = os.Unsetenv("USER")
	_ = os.Unsetenv("USERNAME")
	defer func() {
		_ = os.Setenv("USER", originalUser)
		_ = os.Setenv("USERNAME", originalUsername)
	}()

	// Should still work with "unknown" fallback
	key, err := getMachineKey()
	if err != nil {
		t.Fatalf("getMachineKey failed: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("expected 32-byte key, got %d bytes", len(key))
	}
}
