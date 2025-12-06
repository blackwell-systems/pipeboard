package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HostedConfig contains configuration for the hosted backend.
// This is the configuration stored in the config.yaml file.
type HostedConfig struct {
	URL   string `yaml:"url"`   // Base URL of the mobile backend (e.g., https://pipeboard.example.com)
	Email string `yaml:"email"` // User email for authentication
	// Token is stored securely in keychain/encrypted file, not in config file
}

// HostedBackend implements RemoteBackend for the Pipeboard mobile backend.
// It provides HTTP-based clipboard sync between desktop CLI and mobile devices.
// Data is encrypted client-side using AES-256-GCM before transmission.
type HostedBackend struct {
	baseURL    string       // Base URL of the backend API
	email      string       // User's email address
	token      string       // JWT authentication token
	httpClient *http.Client // HTTP client with 30s timeout
	encryption string       // Encryption mode: "none" or "aes256"
	passphrase string       // Encryption passphrase (empty if encryption is "none")
	ttlDays    int          // TTL for slots (0 = never expires)
}

// API request/response models matching mobile backend API contract

// signupRequest contains user credentials for account creation
type signupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginRequest contains user credentials for authentication
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// authResponse contains JWT token returned after successful authentication
type authResponse struct {
	Token  string `json:"token"`   // JWT token for subsequent API requests
	UserID string `json:"user_id"` // User's unique identifier
	Email  string `json:"email"`   // Confirmed email address
}

// slotMetadataResponse contains slot information without the actual data
type slotMetadataResponse struct {
	Name        string `json:"name"`         // Slot name
	ContentType string `json:"content_type"` // MIME type (e.g., "text/plain", "image/png")
	SizeBytes   int    `json:"size_bytes"`   // Size of encrypted data
	UpdatedAt   string `json:"updated_at"`   // ISO 8601 timestamp
}

// slotDataResponse contains both metadata and the encrypted slot data
type slotDataResponse struct {
	Name          string `json:"name"`           // Slot name
	EncryptedData string `json:"encrypted_data"` // Base64-encoded encrypted payload
	ContentType   string `json:"content_type"`   // MIME type
	SizeBytes     int    `json:"size_bytes"`     // Size of encrypted data
	UpdatedAt     string `json:"updated_at"`     // ISO 8601 timestamp
}

// newHostedBackend creates a new HostedBackend instance.
// It retrieves the JWT token from secure storage (keychain or encrypted file).
// Returns an error if the user is not logged in or if encryption config is invalid.
func newHostedBackend(cfg *HostedConfig, encryption, passphrase string, ttlDays int) (*HostedBackend, error) {
	if cfg == nil || cfg.URL == "" {
		return nil, fmt.Errorf("hosted config is missing or incomplete")
	}

	// Validate encryption config - encryption without passphrase is invalid
	if encryption == "aes256" && passphrase == "" {
		return nil, fmt.Errorf("passphrase required when encryption is set to aes256")
	}

	// Retrieve JWT token from secure storage (Keychain on macOS, encrypted file elsewhere)
	token, err := getStoredToken(cfg.Email)
	if err != nil {
		return nil, fmt.Errorf("not logged in: %w\nRun 'pipeboard login' to authenticate", err)
	}

	return &HostedBackend{
		baseURL:    cfg.URL,
		email:      cfg.Email,
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		encryption: encryption,
		passphrase: passphrase,
		ttlDays:    ttlDays,
	}, nil
}

// Push uploads encrypted data to a slot
func (h *HostedBackend) Push(slot string, data []byte, meta map[string]string) error {
	// Encrypt data if configured
	payload := data
	var err error
	if h.encryption == "aes256" {
		payload, err = encrypt(data, h.passphrase)
		if err != nil {
			return fmt.Errorf("encryption failed: %w", err)
		}
	}

	// Determine content type from metadata or detect
	contentType := meta["mime"]
	if contentType == "" {
		contentType = detectMIME(data)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/slots/%s", h.baseURL, slot)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+h.token)
	req.Header.Set("Content-Type", contentType)

	// Send request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode == 401 {
		return fmt.Errorf("unauthorized: token expired or invalid\nRun 'pipeboard login' to re-authenticate")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Pull downloads and decrypts data from a slot
func (h *HostedBackend) Pull(slot string) ([]byte, map[string]string, error) {
	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/slots/%s", h.baseURL, slot)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+h.token)

	// Send request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode == 401 {
		return nil, nil, fmt.Errorf("unauthorized: token expired or invalid\nRun 'pipeboard login' to re-authenticate")
	}
	if resp.StatusCode == 404 {
		return nil, nil, fmt.Errorf("slot '%s' not found", slot)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("pull failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var slotData slotDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&slotData); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Decode base64
	encryptedData, err := base64.StdEncoding.DecodeString(slotData.EncryptedData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Decrypt data if configured
	data := encryptedData
	if h.encryption == "aes256" {
		data, err = decrypt(encryptedData, h.passphrase)
		if err != nil {
			return nil, nil, fmt.Errorf("decryption failed: %w", err)
		}
	}

	// Return data with metadata
	meta := map[string]string{
		"mime":       slotData.ContentType,
		"updated_at": slotData.UpdatedAt,
	}

	return data, meta, nil
}

// List returns all slots for the authenticated user
func (h *HostedBackend) List() ([]RemoteSlot, error) {
	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/slots", h.baseURL)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+h.token)

	// Send request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("unauthorized: token expired or invalid\nRun 'pipeboard login' to re-authenticate")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var metadata []slotMetadataResponse
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to RemoteSlot
	slots := make([]RemoteSlot, len(metadata))
	for i, m := range metadata {
		updatedAt, _ := time.Parse(time.RFC3339, m.UpdatedAt)
		slots[i] = RemoteSlot{
			Name:      m.Name,
			Size:      int64(m.SizeBytes),
			CreatedAt: updatedAt, // Use updated_at as created_at since backend doesn't return it
		}
	}

	return slots, nil
}

// Delete removes a slot from the backend
func (h *HostedBackend) Delete(slot string) error {
	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/slots/%s", h.baseURL, slot)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+h.token)

	// Send request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode == 401 {
		return fmt.Errorf("unauthorized: token expired or invalid\nRun 'pipeboard login' to re-authenticate")
	}
	if resp.StatusCode == 404 {
		return fmt.Errorf("slot '%s' not found", slot)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Authentication functions

// Signup creates a new user account
func Signup(baseURL, email, password string) error {
	client := &http.Client{Timeout: 30 * time.Second}

	// Create request
	reqBody := signupRequest{
		Email:    email,
		Password: password,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/auth/signup", baseURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode == 409 {
		return fmt.Errorf("account already exists for this email")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("signup failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var authResp authResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	if err := json.Unmarshal(body, &authResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Store token
	if err := storeToken(email, authResp.Token); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	return nil
}

// Login authenticates and stores the JWT token
func Login(baseURL, email, password string) error {
	client := &http.Client{Timeout: 30 * time.Second}

	// Create request
	reqBody := loginRequest{
		Email:    email,
		Password: password,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/auth/login", baseURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode == 401 || resp.StatusCode == 404 {
		return fmt.Errorf("invalid email or password")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var authResp authResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	if err := json.Unmarshal(body, &authResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Store token
	if err := storeToken(email, authResp.Token); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	return nil
}

// Logout clears the stored token
func Logout(email string) error {
	return clearToken(email)
}
