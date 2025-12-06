package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestNewHostedBackend tests the hosted backend constructor
func TestNewHostedBackend(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		_, err := newHostedBackend(nil, "none", "", 0)
		if err == nil {
			t.Error("expected error for nil config")
		}
	})

	t.Run("empty URL", func(t *testing.T) {
		cfg := &HostedConfig{Email: "test@example.com"}
		_, err := newHostedBackend(cfg, "none", "", 0)
		if err == nil {
			t.Error("expected error for empty URL")
		}
	})

	t.Run("encryption without passphrase", func(t *testing.T) {
		cfg := &HostedConfig{
			URL:   "https://example.com",
			Email: "test@example.com",
		}
		_, err := newHostedBackend(cfg, "aes256", "", 0)
		if err == nil {
			t.Error("expected error for aes256 without passphrase")
		}
	})

	t.Run("not logged in", func(t *testing.T) {
		cfg := &HostedConfig{
			URL:   "https://example.com",
			Email: "nonexistent@example.com",
		}
		_, err := newHostedBackend(cfg, "none", "", 0)
		if err == nil {
			t.Error("expected error when not logged in")
		}
	})
}

// TestHostedBackendPush tests the Push operation
func TestHostedBackendPush(t *testing.T) {
	// Create test JWT token
	email := "test-hosted-push@example.com"
	token := "test-jwt-token"
	if err := storeToken(email, token); err != nil {
		t.Fatalf("failed to store token: %v", err)
	}
	defer clearToken(email)

	t.Run("successful push", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("expected PUT, got %s", r.Method)
			}
			if r.Header.Get("Authorization") != "Bearer "+token {
				t.Error("missing or incorrect Authorization header")
			}
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		cfg := &HostedConfig{
			URL:   server.URL,
			Email: email,
		}
		backend, err := newHostedBackend(cfg, "none", "", 0)
		if err != nil {
			t.Fatalf("newHostedBackend failed: %v", err)
		}

		err = backend.Push("test-slot", []byte("test data"), map[string]string{})
		if err != nil {
			t.Errorf("Push failed: %v", err)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		cfg := &HostedConfig{
			URL:   server.URL,
			Email: email,
		}
		backend, err := newHostedBackend(cfg, "none", "", 0)
		if err != nil {
			t.Fatalf("newHostedBackend failed: %v", err)
		}

		err = backend.Push("test-slot", []byte("test data"), map[string]string{})
		if err == nil {
			t.Error("expected error for unauthorized")
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
		}))
		defer server.Close()

		cfg := &HostedConfig{
			URL:   server.URL,
			Email: email,
		}
		backend, err := newHostedBackend(cfg, "none", "", 0)
		if err != nil {
			t.Fatalf("newHostedBackend failed: %v", err)
		}

		err = backend.Push("test-slot", []byte("test data"), map[string]string{})
		if err == nil {
			t.Error("expected error for server error")
		}
	})

	t.Run("with encryption", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		cfg := &HostedConfig{
			URL:   server.URL,
			Email: email,
		}
		backend, err := newHostedBackend(cfg, "aes256", "test-passphrase", 0)
		if err != nil {
			t.Fatalf("newHostedBackend failed: %v", err)
		}

		err = backend.Push("test-slot", []byte("test data"), map[string]string{})
		if err != nil {
			t.Errorf("Push with encryption failed: %v", err)
		}
	})

	t.Run("with content type", func(t *testing.T) {
		var receivedContentType string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedContentType = r.Header.Get("Content-Type")
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		cfg := &HostedConfig{
			URL:   server.URL,
			Email: email,
		}
		backend, err := newHostedBackend(cfg, "none", "", 0)
		if err != nil {
			t.Fatalf("newHostedBackend failed: %v", err)
		}

		err = backend.Push("test-slot", []byte("test data"), map[string]string{"mime": "text/plain"})
		if err != nil {
			t.Errorf("Push failed: %v", err)
		}
		if receivedContentType != "text/plain" {
			t.Errorf("expected Content-Type text/plain, got %s", receivedContentType)
		}
	})
}

// TestHostedBackendPull tests the Pull operation
func TestHostedBackendPull(t *testing.T) {
	email := "test-hosted-pull@example.com"
	token := "test-jwt-token"
	if err := storeToken(email, token); err != nil {
		t.Fatalf("failed to store token: %v", err)
	}
	defer clearToken(email)

	t.Run("successful pull", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			response := slotDataResponse{
				Name:          "test-slot",
				EncryptedData: "dGVzdCBkYXRh", // base64("test data")
				ContentType:   "text/plain",
				SizeBytes:     9,
				UpdatedAt:     "2025-12-06T00:00:00Z",
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := &HostedConfig{
			URL:   server.URL,
			Email: email,
		}
		backend, err := newHostedBackend(cfg, "none", "", 0)
		if err != nil {
			t.Fatalf("newHostedBackend failed: %v", err)
		}

		data, meta, err := backend.Pull("test-slot")
		if err != nil {
			t.Errorf("Pull failed: %v", err)
		}
		if string(data) != "test data" {
			t.Errorf("expected 'test data', got %s", string(data))
		}
		if meta["mime"] != "text/plain" {
			t.Errorf("expected mime type text/plain, got %s", meta["mime"])
		}
	})

	t.Run("not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("slot not found"))
		}))
		defer server.Close()

		cfg := &HostedConfig{
			URL:   server.URL,
			Email: email,
		}
		backend, err := newHostedBackend(cfg, "none", "", 0)
		if err != nil {
			t.Fatalf("newHostedBackend failed: %v", err)
		}

		_, _, err = backend.Pull("nonexistent")
		if err == nil {
			t.Error("expected error for not found")
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := slotDataResponse{
				Name:          "test-slot",
				EncryptedData: "!!!invalid-base64!!!",
				ContentType:   "text/plain",
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := &HostedConfig{
			URL:   server.URL,
			Email: email,
		}
		backend, err := newHostedBackend(cfg, "none", "", 0)
		if err != nil {
			t.Fatalf("newHostedBackend failed: %v", err)
		}

		_, _, err = backend.Pull("test-slot")
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})
}

// TestHostedBackendList tests the List operation
func TestHostedBackendList(t *testing.T) {
	email := "test-hosted-list@example.com"
	token := "test-jwt-token"
	if err := storeToken(email, token); err != nil {
		t.Fatalf("failed to store token: %v", err)
	}
	defer clearToken(email)

	t.Run("empty list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]slotMetadataResponse{})
		}))
		defer server.Close()

		cfg := &HostedConfig{
			URL:   server.URL,
			Email: email,
		}
		backend, err := newHostedBackend(cfg, "none", "", 0)
		if err != nil {
			t.Fatalf("newHostedBackend failed: %v", err)
		}

		slots, err := backend.List()
		if err != nil {
			t.Errorf("List failed: %v", err)
		}
		if len(slots) != 0 {
			t.Errorf("expected empty list, got %d slots", len(slots))
		}
	})

	t.Run("with slots", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := []slotMetadataResponse{
				{
					Name:        "slot1",
					ContentType: "text/plain",
					SizeBytes:   100,
					UpdatedAt:   "2025-12-06T00:00:00Z",
				},
				{
					Name:        "slot2",
					ContentType: "image/png",
					SizeBytes:   2048,
					UpdatedAt:   "2025-12-05T00:00:00Z",
				},
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := &HostedConfig{
			URL:   server.URL,
			Email: email,
		}
		backend, err := newHostedBackend(cfg, "none", "", 0)
		if err != nil {
			t.Fatalf("newHostedBackend failed: %v", err)
		}

		slots, err := backend.List()
		if err != nil {
			t.Errorf("List failed: %v", err)
		}
		if len(slots) != 2 {
			t.Errorf("expected 2 slots, got %d", len(slots))
		}
		if slots[0].Name != "slot1" {
			t.Errorf("expected slot1, got %s", slots[0].Name)
		}
		if slots[0].Size != 100 {
			t.Errorf("expected size 100, got %d", slots[0].Size)
		}
	})
}

// TestHostedBackendDelete tests the Delete operation
func TestHostedBackendDelete(t *testing.T) {
	email := "test-hosted-delete@example.com"
	token := "test-jwt-token"
	if err := storeToken(email, token); err != nil {
		t.Fatalf("failed to store token: %v", err)
	}
	defer clearToken(email)

	t.Run("successful delete", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		cfg := &HostedConfig{
			URL:   server.URL,
			Email: email,
		}
		backend, err := newHostedBackend(cfg, "none", "", 0)
		if err != nil {
			t.Fatalf("newHostedBackend failed: %v", err)
		}

		err = backend.Delete("test-slot")
		if err != nil {
			t.Errorf("Delete failed: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		cfg := &HostedConfig{
			URL:   server.URL,
			Email: email,
		}
		backend, err := newHostedBackend(cfg, "none", "", 0)
		if err != nil {
			t.Fatalf("newHostedBackend failed: %v", err)
		}

		err = backend.Delete("nonexistent")
		if err == nil {
			t.Error("expected error for not found")
		}
	})
}

// TestSignup tests the Signup function
func TestSignup(t *testing.T) {
	t.Run("successful signup", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}

			var req signupRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error("failed to decode signup request")
			}
			if req.Email != "test@example.com" {
				t.Errorf("expected test@example.com, got %s", req.Email)
			}

			response := authResponse{
				Token:  "jwt-token-123",
				UserID: "user-id-123",
				Email:  "test@example.com",
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		email := "test@example.com"
		defer clearToken(email)

		err := Signup(server.URL, email, "password123")
		if err != nil {
			t.Errorf("Signup failed: %v", err)
		}

		// Verify token was stored
		token, err := getStoredToken(email)
		if err != nil {
			t.Error("token should be stored after signup")
		}
		if token != "jwt-token-123" {
			t.Errorf("expected jwt-token-123, got %s", token)
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("email already exists"))
		}))
		defer server.Close()

		err := Signup(server.URL, "test@example.com", "password123")
		if err == nil {
			t.Error("expected error for duplicate email")
		}
	})
}

// TestLogin tests the Login function
func TestLogin(t *testing.T) {
	t.Run("successful login", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req loginRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error("failed to decode login request")
			}

			response := authResponse{
				Token:  "login-token-456",
				UserID: "user-id-456",
				Email:  req.Email,
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		email := "login@example.com"
		defer clearToken(email)

		err := Login(server.URL, email, "correct-password")
		if err != nil {
			t.Errorf("Login failed: %v", err)
		}

		// Verify token was stored
		token, err := getStoredToken(email)
		if err != nil {
			t.Error("token should be stored after login")
		}
		if token != "login-token-456" {
			t.Errorf("expected login-token-456, got %s", token)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("invalid credentials"))
		}))
		defer server.Close()

		err := Login(server.URL, "test@example.com", "wrong-password")
		if err == nil {
			t.Error("expected error for wrong password")
		}
	})
}

// TestLogout tests the Logout function
func TestLogout(t *testing.T) {
	email := "logout-test@example.com"
	token := "test-token-to-clear"

	t.Run("successful logout", func(t *testing.T) {
		// Store a token first
		if err := storeToken(email, token); err != nil {
			t.Fatalf("failed to store token: %v", err)
		}

		// Verify token exists
		if _, err := getStoredToken(email); err != nil {
			t.Error("token should exist before logout")
		}

		// Logout
		if err := Logout(email); err != nil {
			t.Errorf("Logout failed: %v", err)
		}

		// Verify token was cleared
		if _, err := getStoredToken(email); err == nil {
			t.Error("token should be cleared after logout")
		}
	})

	t.Run("logout without token", func(t *testing.T) {
		// Should not error even if no token exists
		err := Logout("nonexistent@example.com")
		if err != nil {
			t.Errorf("Logout should not error for nonexistent token: %v", err)
		}
	})
}

// TestHostedBackendWithEncryption tests end-to-end encrypted push/pull
func TestHostedBackendWithEncryption(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping test that requires secure storage in CI")
	}

	email := "encrypt-test@example.com"
	token := "test-jwt-token"
	passphrase := "test-encryption-passphrase"

	if err := storeToken(email, token); err != nil {
		t.Fatalf("failed to store token: %v", err)
	}
	defer clearToken(email)

	// Storage for the encrypted data on the "server"
	var storedData string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			// Store the encrypted data (base64-encode it as the real backend would)
			body, _ := io.ReadAll(r.Body)
			storedData = base64.StdEncoding.EncodeToString(body)
			w.WriteHeader(http.StatusCreated)

		case http.MethodGet:
			// Return the stored encrypted data
			response := slotDataResponse{
				Name:          "test-slot",
				EncryptedData: storedData,
				ContentType:   "text/plain",
				SizeBytes:     len(storedData),
				UpdatedAt:     "2025-12-06T00:00:00Z",
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	cfg := &HostedConfig{
		URL:   server.URL,
		Email: email,
	}

	// Push encrypted data
	backend1, err := newHostedBackend(cfg, "aes256", passphrase, 0)
	if err != nil {
		t.Fatalf("newHostedBackend failed: %v", err)
	}

	originalData := []byte("secret clipboard data")
	err = backend1.Push("test-slot", originalData, map[string]string{})
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Pull and decrypt
	backend2, err := newHostedBackend(cfg, "aes256", passphrase, 0)
	if err != nil {
		t.Fatalf("newHostedBackend failed: %v", err)
	}

	pulledData, _, err := backend2.Pull("test-slot")
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	if string(pulledData) != string(originalData) {
		t.Errorf("decrypted data mismatch: expected %s, got %s", string(originalData), string(pulledData))
	}
}
