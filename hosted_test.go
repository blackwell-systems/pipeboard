package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestNewHostedBackendValidation tests configuration validation for the hosted backend.
func TestNewHostedBackendValidation(t *testing.T) {
	// Setup temp directory for token storage
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	// Pre-store a valid token for tests that need it
	testEmail := "test@example.com"
	if err := storeToken(testEmail, "valid-token"); err != nil {
		t.Fatalf("failed to store test token: %v", err)
	}

	tests := []struct {
		name       string
		cfg        *HostedConfig
		encryption string
		passphrase string
		wantErr    bool
		errContain string
	}{
		{
			name:       "nil config",
			cfg:        nil,
			encryption: "",
			passphrase: "",
			wantErr:    true,
			errContain: "missing",
		},
		{
			name:       "empty URL",
			cfg:        &HostedConfig{URL: "", Email: testEmail},
			encryption: "",
			passphrase: "",
			wantErr:    true,
			errContain: "missing",
		},
		{
			name:       "encryption without passphrase",
			cfg:        &HostedConfig{URL: "https://example.com", Email: testEmail},
			encryption: "aes256",
			passphrase: "",
			wantErr:    true,
			errContain: "passphrase required",
		},
		{
			name:       "valid config with encryption",
			cfg:        &HostedConfig{URL: "https://example.com", Email: testEmail},
			encryption: "aes256",
			passphrase: "secret",
			wantErr:    false,
		},
		{
			name:       "valid config without encryption",
			cfg:        &HostedConfig{URL: "https://example.com", Email: testEmail},
			encryption: "",
			passphrase: "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := newHostedBackend(tt.cfg, tt.encryption, tt.passphrase, 0)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.errContain != "" && !stringContains(err.Error(), tt.errContain) {
					t.Errorf("error should contain %q, got: %v", tt.errContain, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if backend == nil {
					t.Error("backend should not be nil on success")
				}
			}
		})
	}
}

// TestNewHostedBackendNoToken tests that backend creation fails when no token is stored.
func TestNewHostedBackendNoToken(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	cfg := &HostedConfig{
		URL:   "https://example.com",
		Email: "notoken@example.com",
	}

	_, err := newHostedBackend(cfg, "", "", 0)
	if err == nil {
		t.Error("expected error when no token is stored")
	}
	if !stringContains(err.Error(), "not logged in") {
		t.Errorf("error should mention login, got: %v", err)
	}
}

// TestHostedBackendPush tests pushing data to the hosted backend using a mock server.
func TestHostedBackendPush(t *testing.T) {
	tests := []struct {
		name         string
		serverStatus int
		serverBody   string
		wantErr      bool
		errContain   string
	}{
		{
			name:         "success",
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name:         "created",
			serverStatus: http.StatusCreated,
			wantErr:      false,
		},
		{
			name:         "unauthorized",
			serverStatus: http.StatusUnauthorized,
			wantErr:      true,
			errContain:   "unauthorized",
		},
		{
			name:         "server error",
			serverStatus: http.StatusInternalServerError,
			serverBody:   "internal error",
			wantErr:      true,
			errContain:   "500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request format
				if r.Method != http.MethodPut {
					t.Errorf("expected PUT, got %s", r.Method)
				}
				if r.Header.Get("Authorization") == "" {
					t.Error("missing Authorization header")
				}

				w.WriteHeader(tt.serverStatus)
				if tt.serverBody != "" {
					_, _ = w.Write([]byte(tt.serverBody))
				}
			}))
			defer server.Close()

			backend := &HostedBackend{
				baseURL:    server.URL,
				token:      "test-token",
				httpClient: server.Client(),
			}

			err := backend.Push("test-slot", []byte("test data"), nil)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				} else if tt.errContain != "" && !stringContains(err.Error(), tt.errContain) {
					t.Errorf("error should contain %q, got: %v", tt.errContain, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestHostedBackendPushWithEncryption tests that data is encrypted before pushing.
func TestHostedBackendPushWithEncryption(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture the request body to verify encryption
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		receivedBody = body[:n]
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	backend := &HostedBackend{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
		encryption: "aes256",
		passphrase: "test-passphrase",
	}

	originalData := []byte("sensitive data that should be encrypted")
	err := backend.Push("encrypted-slot", originalData, nil)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Verify data was encrypted (should not match original)
	if string(receivedBody) == string(originalData) {
		t.Error("data should be encrypted before sending")
	}
}

// TestHostedBackendPull tests pulling data from the hosted backend.
func TestHostedBackendPull(t *testing.T) {
	slotData := slotDataResponse{
		Name:          "test-slot",
		EncryptedData: base64.StdEncoding.EncodeToString([]byte("plain data")),
		ContentType:   "text/plain",
		UpdatedAt:     "2024-01-15T10:30:00Z",
	}

	tests := []struct {
		name         string
		serverStatus int
		serverBody   interface{}
		wantErr      bool
		errContain   string
	}{
		{
			name:         "success",
			serverStatus: http.StatusOK,
			serverBody:   slotData,
			wantErr:      false,
		},
		{
			name:         "not found",
			serverStatus: http.StatusNotFound,
			wantErr:      true,
			errContain:   "not found",
		},
		{
			name:         "unauthorized",
			serverStatus: http.StatusUnauthorized,
			wantErr:      true,
			errContain:   "unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET, got %s", r.Method)
				}

				w.WriteHeader(tt.serverStatus)
				if tt.serverBody != nil {
					_ = json.NewEncoder(w).Encode(tt.serverBody)
				}
			}))
			defer server.Close()

			backend := &HostedBackend{
				baseURL:    server.URL,
				token:      "test-token",
				httpClient: server.Client(),
			}

			data, meta, err := backend.Pull("test-slot")
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				} else if tt.errContain != "" && !stringContains(err.Error(), tt.errContain) {
					t.Errorf("error should contain %q, got: %v", tt.errContain, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if string(data) != "plain data" {
					t.Errorf("data mismatch: got %q", string(data))
				}
				if meta["mime"] != "text/plain" {
					t.Errorf("mime mismatch: got %q", meta["mime"])
				}
			}
		})
	}
}

// TestHostedBackendPullWithDecryption tests that encrypted data is decrypted on pull.
func TestHostedBackendPullWithDecryption(t *testing.T) {
	// First encrypt some data
	originalData := []byte("secret message")
	passphrase := "test-passphrase"
	encrypted, err := encrypt(originalData, passphrase)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	slotData := slotDataResponse{
		Name:          "encrypted-slot",
		EncryptedData: base64.StdEncoding.EncodeToString(encrypted),
		ContentType:   "application/octet-stream",
		UpdatedAt:     "2024-01-15T10:30:00Z",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(slotData)
	}))
	defer server.Close()

	backend := &HostedBackend{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
		encryption: "aes256",
		passphrase: passphrase,
	}

	data, _, err := backend.Pull("encrypted-slot")
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	if string(data) != string(originalData) {
		t.Errorf("decrypted data mismatch: got %q, want %q", string(data), string(originalData))
	}
}

// TestHostedBackendList tests listing slots from the hosted backend.
func TestHostedBackendList(t *testing.T) {
	metadata := []slotMetadataResponse{
		{Name: "slot1", ContentType: "text/plain", SizeBytes: 100, UpdatedAt: "2024-01-15T10:30:00Z"},
		{Name: "slot2", ContentType: "application/json", SizeBytes: 256, UpdatedAt: "2024-01-14T09:00:00Z"},
	}

	tests := []struct {
		name         string
		serverStatus int
		serverBody   interface{}
		wantErr      bool
		wantSlots    int
	}{
		{
			name:         "success",
			serverStatus: http.StatusOK,
			serverBody:   metadata,
			wantErr:      false,
			wantSlots:    2,
		},
		{
			name:         "empty list",
			serverStatus: http.StatusOK,
			serverBody:   []slotMetadataResponse{},
			wantErr:      false,
			wantSlots:    0,
		},
		{
			name:         "unauthorized",
			serverStatus: http.StatusUnauthorized,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET, got %s", r.Method)
				}

				w.WriteHeader(tt.serverStatus)
				if tt.serverBody != nil {
					_ = json.NewEncoder(w).Encode(tt.serverBody)
				}
			}))
			defer server.Close()

			backend := &HostedBackend{
				baseURL:    server.URL,
				token:      "test-token",
				httpClient: server.Client(),
			}

			slots, err := backend.List()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(slots) != tt.wantSlots {
					t.Errorf("expected %d slots, got %d", tt.wantSlots, len(slots))
				}
			}
		})
	}
}

// TestHostedBackendDelete tests deleting a slot from the hosted backend.
func TestHostedBackendDelete(t *testing.T) {
	tests := []struct {
		name         string
		serverStatus int
		wantErr      bool
		errContain   string
	}{
		{
			name:         "success",
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name:         "not found",
			serverStatus: http.StatusNotFound,
			wantErr:      true,
			errContain:   "not found",
		},
		{
			name:         "unauthorized",
			serverStatus: http.StatusUnauthorized,
			wantErr:      true,
			errContain:   "unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("expected DELETE, got %s", r.Method)
				}

				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			backend := &HostedBackend{
				baseURL:    server.URL,
				token:      "test-token",
				httpClient: server.Client(),
			}

			err := backend.Delete("test-slot")
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				} else if tt.errContain != "" && !stringContains(err.Error(), tt.errContain) {
					t.Errorf("error should contain %q, got: %v", tt.errContain, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestSignup tests the Signup function with mock server.
func TestSignup(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	tests := []struct {
		name         string
		serverStatus int
		serverBody   interface{}
		wantErr      bool
		errContain   string
	}{
		{
			name:         "success",
			serverStatus: http.StatusCreated,
			serverBody:   authResponse{Token: "new-token", UserID: "123", Email: "test@example.com"},
			wantErr:      false,
		},
		{
			name:         "conflict",
			serverStatus: http.StatusConflict,
			wantErr:      true,
			errContain:   "already exists",
		},
		{
			name:         "server error",
			serverStatus: http.StatusInternalServerError,
			serverBody:   "internal error",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/auth/signup" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}

				var req signupRequest
				_ = json.NewDecoder(r.Body).Decode(&req)
				if req.Email == "" || req.Password == "" {
					t.Error("missing email or password in request")
				}

				w.WriteHeader(tt.serverStatus)
				if tt.serverBody != nil {
					_ = json.NewEncoder(w).Encode(tt.serverBody)
				}
			}))
			defer server.Close()

			err := Signup(server.URL, "test@example.com", "password123")
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				} else if tt.errContain != "" && !stringContains(err.Error(), tt.errContain) {
					t.Errorf("error should contain %q, got: %v", tt.errContain, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				// Verify token was stored
				token, err := getStoredToken("test@example.com")
				if err != nil {
					t.Error("token should be stored after successful signup")
				}
				if token != "new-token" {
					t.Errorf("stored token mismatch: got %q", token)
				}
			}
		})
	}
}

// TestLogin tests the Login function with mock server.
func TestLogin(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	tests := []struct {
		name         string
		serverStatus int
		serverBody   interface{}
		wantErr      bool
		errContain   string
	}{
		{
			name:         "success",
			serverStatus: http.StatusOK,
			serverBody:   authResponse{Token: "login-token", UserID: "123", Email: "login@example.com"},
			wantErr:      false,
		},
		{
			name:         "invalid credentials 401",
			serverStatus: http.StatusUnauthorized,
			wantErr:      true,
			errContain:   "invalid email or password",
		},
		{
			name:         "invalid credentials 404",
			serverStatus: http.StatusNotFound,
			wantErr:      true,
			errContain:   "invalid email or password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/auth/login" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}

				w.WriteHeader(tt.serverStatus)
				if tt.serverBody != nil {
					_ = json.NewEncoder(w).Encode(tt.serverBody)
				}
			}))
			defer server.Close()

			err := Login(server.URL, "login@example.com", "password123")
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				} else if tt.errContain != "" && !stringContains(err.Error(), tt.errContain) {
					t.Errorf("error should contain %q, got: %v", tt.errContain, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				// Verify token was stored
				token, err := getStoredToken("login@example.com")
				if err != nil {
					t.Error("token should be stored after successful login")
				}
				if token != "login-token" {
					t.Errorf("stored token mismatch: got %q", token)
				}
			}
		})
	}
}

// TestLogout tests the Logout function.
func TestLogout(t *testing.T) {
	tmpDir := t.TempDir()
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", originalXDG) }()

	email := "logout@example.com"

	// Store a token first
	if err := storeToken(email, "token-to-clear"); err != nil {
		t.Fatalf("storeToken failed: %v", err)
	}

	// Verify token exists
	_, err := getStoredToken(email)
	if err != nil {
		t.Fatal("token should exist before logout")
	}

	// Logout
	if err := Logout(email); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	// Verify token is cleared
	_, err = getStoredToken(email)
	if err == nil {
		t.Error("token should be cleared after logout")
	}
}

// TestHostedBackendRequestHeaders tests that correct headers are sent.
func TestHostedBackendRequestHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authorization header format
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("unexpected Authorization header: %s", auth)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]slotMetadataResponse{})
	}))
	defer server.Close()

	backend := &HostedBackend{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

	_, _ = backend.List()
}

// TestHostedBackendPushContentType tests that content type is set correctly.
func TestHostedBackendPushContentType(t *testing.T) {
	tests := []struct {
		name         string
		meta         map[string]string
		data         []byte
		expectedMime string
	}{
		{
			name:         "explicit mime",
			meta:         map[string]string{"mime": "application/json"},
			data:         []byte("{}"),
			expectedMime: "application/json",
		},
		{
			name:         "detect html",
			meta:         nil,
			data:         []byte(`<!DOCTYPE html><html><body>test</body></html>`),
			expectedMime: "text/html; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contentType := r.Header.Get("Content-Type")
				if contentType != tt.expectedMime {
					t.Errorf("expected Content-Type %q, got %q", tt.expectedMime, contentType)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			backend := &HostedBackend{
				baseURL:    server.URL,
				token:      "test-token",
				httpClient: server.Client(),
			}

			_ = backend.Push("slot", tt.data, tt.meta)
		})
	}
}

// stringContains checks if s contains substr (helper for error checking).
// Named differently from containsString in remote_test.go to avoid redeclaration.
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestHostedBackendPullInvalidJSON tests that Pull handles invalid JSON gracefully.
func TestHostedBackendPullInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	backend := &HostedBackend{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

	_, _, err := backend.Pull("test-slot")
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

// TestHostedBackendPullInvalidBase64 tests that Pull handles invalid base64 gracefully.
func TestHostedBackendPullInvalidBase64(t *testing.T) {
	slotData := slotDataResponse{
		Name:          "test-slot",
		EncryptedData: "not-valid-base64!!!",
		ContentType:   "text/plain",
		UpdatedAt:     "2024-01-15T10:30:00Z",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(slotData)
	}))
	defer server.Close()

	backend := &HostedBackend{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

	_, _, err := backend.Pull("test-slot")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

// TestHostedBackendListInvalidJSON tests that List handles invalid JSON gracefully.
func TestHostedBackendListInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not valid json array"))
	}))
	defer server.Close()

	backend := &HostedBackend{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

	_, err := backend.List()
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

// TestHostedBackendListServerError tests List with server errors.
func TestHostedBackendListServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	backend := &HostedBackend{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

	_, err := backend.List()
	if err == nil {
		t.Error("expected error for server error response")
	}
	if !stringContains(err.Error(), "500") {
		t.Errorf("error should contain status code, got: %v", err)
	}
}

// TestHostedBackendDeleteServerError tests Delete with server errors.
func TestHostedBackendDeleteServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	backend := &HostedBackend{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

	err := backend.Delete("test-slot")
	if err == nil {
		t.Error("expected error for server error response")
	}
}

// TestHostedBackendPullServerError tests Pull with server errors.
func TestHostedBackendPullServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	backend := &HostedBackend{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

	_, _, err := backend.Pull("test-slot")
	if err == nil {
		t.Error("expected error for server error response")
	}
}

// TestHostedBackendPullDecryptionError tests that Pull handles decryption errors.
func TestHostedBackendPullDecryptionError(t *testing.T) {
	// Send data that looks like encrypted but isn't valid
	slotData := slotDataResponse{
		Name:          "test-slot",
		EncryptedData: base64.StdEncoding.EncodeToString([]byte("not encrypted data")),
		ContentType:   "application/octet-stream",
		UpdatedAt:     "2024-01-15T10:30:00Z",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(slotData)
	}))
	defer server.Close()

	backend := &HostedBackend{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
		encryption: "aes256",
		passphrase: "test-passphrase",
	}

	_, _, err := backend.Pull("test-slot")
	if err == nil {
		t.Error("expected error for invalid encrypted data")
	}
	if !stringContains(err.Error(), "decryption failed") {
		t.Errorf("error should mention decryption failure, got: %v", err)
	}
}

// TestSignupServerError tests Signup with server errors.
func TestSignupServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	err := Signup(server.URL, "test@example.com", "password")
	if err == nil {
		t.Error("expected error for server error response")
	}
}

// TestLoginServerError tests Login with server errors.
func TestLoginServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	err := Login(server.URL, "test@example.com", "password")
	if err == nil {
		t.Error("expected error for server error response")
	}
}

// TestHostedBackendNetworkError tests handling of network errors.
func TestHostedBackendNetworkError(t *testing.T) {
	// Use an invalid URL to trigger network error
	backend := &HostedBackend{
		baseURL:    "http://localhost:99999",
		token:      "test-token",
		httpClient: &http.Client{},
	}

	// Push should fail
	err := backend.Push("slot", []byte("data"), nil)
	if err == nil {
		t.Error("expected error for network error")
	}

	// Pull should fail
	_, _, err = backend.Pull("slot")
	if err == nil {
		t.Error("expected error for network error")
	}

	// List should fail
	_, err = backend.List()
	if err == nil {
		t.Error("expected error for network error")
	}

	// Delete should fail
	err = backend.Delete("slot")
	if err == nil {
		t.Error("expected error for network error")
	}
}
