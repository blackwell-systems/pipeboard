package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// Token storage: secure token storage across platforms
// - macOS: Uses keychain via security command
// - Linux: Encrypted file in ~/.config/pipeboard/
// - Windows: Encrypted file in AppData

const serviceName = "com.blackwell.pipeboard"

// storeToken securely stores a JWT token for the given email
func storeToken(email, token string) error {
	switch runtime.GOOS {
	case "darwin":
		return storeTokenKeychain(email, token)
	default:
		return storeTokenFile(email, token)
	}
}

// getStoredToken retrieves a stored JWT token for the given email
func getStoredToken(email string) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return getTokenKeychain(email)
	default:
		return getTokenFile(email)
	}
}

// clearToken removes the stored token for the given email
func clearToken(email string) error {
	switch runtime.GOOS {
	case "darwin":
		return clearTokenKeychain(email)
	default:
		return clearTokenFile(email)
	}
}

// macOS Keychain implementation

func storeTokenKeychain(email, token string) error {
	// Delete existing entry first
	_ = clearTokenKeychain(email)

	// Add new entry
	cmd := fmt.Sprintf("security add-generic-password -s '%s' -a '%s' -w '%s'",
		serviceName, email, token)
	return runCommand("sh", "-c", cmd)
}

func getTokenKeychain(email string) (string, error) {
	cmd := fmt.Sprintf("security find-generic-password -s '%s' -a '%s' -w",
		serviceName, email)
	output, err := runCommandOutput("sh", "-c", cmd)
	if err != nil {
		return "", fmt.Errorf("token not found in keychain")
	}
	return string(output), nil
}

func clearTokenKeychain(email string) error {
	cmd := fmt.Sprintf("security delete-generic-password -s '%s' -a '%s'",
		serviceName, email)
	_ = runCommand("sh", "-c", cmd) // Ignore errors if item doesn't exist
	return nil
}

// File-based implementation (Linux/Windows)
// Tokens are encrypted with a machine-specific key

type tokenStore struct {
	Tokens map[string]string `json:"tokens"` // email -> encrypted token
}

func tokenFilePath() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".config")
	}

	dir := filepath.Join(configDir, "pipeboard")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(dir, ".tokens"), nil
}

// getMachineKey derives a machine-specific encryption key
// Uses hostname + username hash as the key
func getMachineKey() ([]byte, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	if username == "" {
		username = "unknown"
	}

	// Hash hostname + username for a machine-specific key
	data := fmt.Sprintf("%s:%s:pipeboard-token-store", hostname, username)
	hash := sha256.Sum256([]byte(data))
	return hash[:], nil
}

// encryptToken encrypts a token with the machine key
func encryptToken(token string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(token), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptToken decrypts a token with the machine key
func decryptToken(encryptedToken string, key []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encryptedToken)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func loadTokenStore() (*tokenStore, error) {
	path, err := tokenFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &tokenStore{Tokens: make(map[string]string)}, nil
		}
		return nil, err
	}

	var store tokenStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}

	if store.Tokens == nil {
		store.Tokens = make(map[string]string)
	}

	return &store, nil
}

func saveTokenStore(store *tokenStore) error {
	path, err := tokenFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func storeTokenFile(email, token string) error {
	store, err := loadTokenStore()
	if err != nil {
		return err
	}

	key, err := getMachineKey()
	if err != nil {
		return err
	}

	encrypted, err := encryptToken(token, key)
	if err != nil {
		return err
	}

	store.Tokens[email] = encrypted
	return saveTokenStore(store)
}

func getTokenFile(email string) (string, error) {
	store, err := loadTokenStore()
	if err != nil {
		return "", err
	}

	encrypted, ok := store.Tokens[email]
	if !ok {
		return "", fmt.Errorf("no token found for %s", email)
	}

	key, err := getMachineKey()
	if err != nil {
		return "", err
	}

	return decryptToken(encrypted, key)
}

func clearTokenFile(email string) error {
	store, err := loadTokenStore()
	if err != nil {
		return err
	}

	delete(store.Tokens, email)
	return saveTokenStore(store)
}
