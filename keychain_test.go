package main

import (
	"os"
	"testing"
)

// TestTokenStorage tests cross-platform token storage
func TestTokenStorage(t *testing.T) {
	email := "keychain-test@example.com"
	token := "test-token-12345"

	t.Run("store and retrieve", func(t *testing.T) {
		// Store token
		if err := storeToken(email, token); err != nil {
			t.Fatalf("storeToken failed: %v", err)
		}

		// Retrieve token
		retrieved, err := getStoredToken(email)
		if err != nil {
			t.Fatalf("getStoredToken failed: %v", err)
		}

		if retrieved != token {
			t.Errorf("expected %s, got %s", token, retrieved)
		}

		// Cleanup
		_ = clearToken(email)
	})

	t.Run("retrieve nonexistent token", func(t *testing.T) {
		_, err := getStoredToken("nonexistent@example.com")
		if err == nil {
			t.Error("expected error for nonexistent token")
		}
	})

	t.Run("clear token", func(t *testing.T) {
		// Store token
		if err := storeToken(email, token); err != nil {
			t.Fatalf("storeToken failed: %v", err)
		}

		// Clear token
		if err := clearToken(email); err != nil {
			t.Fatalf("clearToken failed: %v", err)
		}

		// Verify it's gone
		_, err := getStoredToken(email)
		if err == nil {
			t.Error("token should not exist after clear")
		}
	})

	t.Run("overwrite existing token", func(t *testing.T) {
		email2 := "overwrite-test@example.com"
		defer func() { _ = clearToken(email2) }()

		// Store first token
		if err := storeToken(email2, "first-token"); err != nil {
			t.Fatalf("first storeToken failed: %v", err)
		}

		// Overwrite with second token
		if err := storeToken(email2, "second-token"); err != nil {
			t.Fatalf("second storeToken failed: %v", err)
		}

		// Should get the second token
		retrieved, err := getStoredToken(email2)
		if err != nil {
			t.Fatalf("getStoredToken failed: %v", err)
		}
		if retrieved != "second-token" {
			t.Errorf("expected second-token, got %s", retrieved)
		}
	})

	t.Run("multiple users", func(t *testing.T) {
		user1 := "user1@example.com"
		user2 := "user2@example.com"
		defer func() { _ = clearToken(user1) }()
		defer func() { _ = clearToken(user2) }()

		// Store tokens for two different users
		if err := storeToken(user1, "token1"); err != nil {
			t.Fatalf("storeToken user1 failed: %v", err)
		}
		if err := storeToken(user2, "token2"); err != nil {
			t.Fatalf("storeToken user2 failed: %v", err)
		}

		// Retrieve both tokens
		token1, err := getStoredToken(user1)
		if err != nil || token1 != "token1" {
			t.Errorf("user1 token mismatch: got %s, err %v", token1, err)
		}

		token2, err := getStoredToken(user2)
		if err != nil || token2 != "token2" {
			t.Errorf("user2 token mismatch: got %s, err %v", token2, err)
		}
	})
}

// TestGetMachineKey tests machine key generation
func TestGetMachineKey(t *testing.T) {
	t.Run("returns consistent key", func(t *testing.T) {
		key1, err := getMachineKey()
		if err != nil {
			t.Fatalf("getMachineKey failed: %v", err)
		}

		key2, err := getMachineKey()
		if err != nil {
			t.Fatalf("getMachineKey failed: %v", err)
		}

		if len(key1) != 32 {
			t.Errorf("expected 32-byte key, got %d bytes", len(key1))
		}

		// Should be deterministic for same machine
		if string(key1) != string(key2) {
			t.Error("machine key should be consistent across calls")
		}
	})

	t.Run("key is non-empty", func(t *testing.T) {
		key, err := getMachineKey()
		if err != nil {
			t.Fatalf("getMachineKey failed: %v", err)
		}

		allZero := true
		for _, b := range key {
			if b != 0 {
				allZero = false
				break
			}
		}

		if allZero {
			t.Error("machine key should not be all zeros")
		}
	})
}

// TestEncryptDecryptToken tests token encryption/decryption
func TestEncryptDecryptToken(t *testing.T) {
	key, err := getMachineKey()
	if err != nil {
		t.Fatalf("getMachineKey failed: %v", err)
	}

	t.Run("encrypt and decrypt", func(t *testing.T) {
		originalToken := "jwt-token-example-123"

		encrypted, err := encryptToken(originalToken, key)
		if err != nil {
			t.Fatalf("encryptToken failed: %v", err)
		}

		if encrypted == originalToken {
			t.Error("encrypted token should differ from original")
		}

		decrypted, err := decryptToken(encrypted, key)
		if err != nil {
			t.Fatalf("decryptToken failed: %v", err)
		}

		if decrypted != originalToken {
			t.Errorf("expected %s, got %s", originalToken, decrypted)
		}
	})

	t.Run("empty token", func(t *testing.T) {
		encrypted, err := encryptToken("", key)
		if err != nil {
			t.Fatalf("encryptToken failed for empty string: %v", err)
		}

		decrypted, err := decryptToken(encrypted, key)
		if err != nil {
			t.Fatalf("decryptToken failed: %v", err)
		}

		if decrypted != "" {
			t.Errorf("expected empty string, got %s", decrypted)
		}
	})

	t.Run("invalid encrypted data", func(t *testing.T) {
		_, err := decryptToken("invalid-base64!!!", key)
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})

	t.Run("wrong key", func(t *testing.T) {
		originalToken := "jwt-token-example"
		encrypted, err := encryptToken(originalToken, key)
		if err != nil {
			t.Fatalf("encryptToken failed: %v", err)
		}

		// Try to decrypt with a different key
		wrongKey := make([]byte, 32)
		for i := range wrongKey {
			wrongKey[i] = 0xFF
		}

		_, err = decryptToken(encrypted, wrongKey)
		if err == nil {
			t.Error("expected error when decrypting with wrong key")
		}
	})

	t.Run("corrupted ciphertext", func(t *testing.T) {
		originalToken := "jwt-token-example"
		encrypted, err := encryptToken(originalToken, key)
		if err != nil {
			t.Fatalf("encryptToken failed: %v", err)
		}

		// Corrupt the ciphertext by modifying a character
		corrupted := encrypted[:len(encrypted)-1] + "X"

		_, err = decryptToken(corrupted, key)
		if err == nil {
			t.Error("expected error for corrupted ciphertext")
		}
	})

	t.Run("too short ciphertext", func(t *testing.T) {
		// Create a ciphertext that's too short to contain a valid nonce
		_, err := decryptToken("AA==", key) // Just 1 byte after base64 decode
		if err == nil {
			t.Error("expected error for ciphertext too short")
		}
	})
}

// TestTokenFilePath tests token file path generation
func TestTokenFilePath(t *testing.T) {
	t.Run("respects XDG_CONFIG_HOME", func(t *testing.T) {
		originalXDG := os.Getenv("XDG_CONFIG_HOME")
		defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

		testDir := t.TempDir()
		_ = os.Setenv("XDG_CONFIG_HOME", testDir)

		path, err := tokenFilePath()
		if err != nil {
			t.Fatalf("tokenFilePath failed: %v", err)
		}

		expectedSuffix := "/pipeboard/.tokens"
		if len(path) < len(expectedSuffix) || path[len(path)-len(expectedSuffix):] != expectedSuffix {
			t.Errorf("expected path to end with %s, got %s", expectedSuffix, path)
		}
	})
}

// TestTokenStoreLoadSave tests loading and saving the token store
func TestTokenStoreLoadSave(t *testing.T) {
	// Set up a temporary config directory
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	testDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", testDir)

	email := "store-test@example.com"
	token := "test-token-789"

	t.Run("save and load", func(t *testing.T) {
		// Store token
		if err := storeTokenFile(email, token); err != nil {
			t.Fatalf("storeTokenFile failed: %v", err)
		}

		// Load token
		retrieved, err := getTokenFile(email)
		if err != nil {
			t.Fatalf("getTokenFile failed: %v", err)
		}

		if retrieved != token {
			t.Errorf("expected %s, got %s", token, retrieved)
		}

		// Cleanup
		clearTokenFile(email)
	})

	t.Run("load nonexistent store", func(t *testing.T) {
		// Use a fresh temp dir
		testDir2 := t.TempDir()
		os.Setenv("XDG_CONFIG_HOME", testDir2)

		store, err := loadTokenStore()
		if err != nil {
			t.Fatalf("loadTokenStore should create empty store: %v", err)
		}

		if len(store.Tokens) != 0 {
			t.Errorf("expected empty token map, got %d entries", len(store.Tokens))
		}
	})

	t.Run("clear nonexistent token", func(t *testing.T) {
		// Should not error when clearing a token that doesn't exist
		err := clearTokenFile("doesnotexist@example.com")
		if err != nil {
			t.Errorf("clearTokenFile should not error for nonexistent token: %v", err)
		}
	})
}
