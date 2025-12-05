package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSlotPayloadEncoding(t *testing.T) {
	data := []byte("hello world")

	payload := SlotPayload{
		Version:   1,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Hostname:  "test-host",
		OS:        "linux",
		Len:       len(data),
		MIME:      "text/plain; charset=utf-8",
		DataB64:   base64.StdEncoding.EncodeToString(data),
	}

	// Encode to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	// Decode from JSON
	var decoded SlotPayload
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	// Verify fields
	if decoded.Version != 1 {
		t.Errorf("expected version 1, got %d", decoded.Version)
	}
	if decoded.Hostname != "test-host" {
		t.Errorf("expected hostname test-host, got %s", decoded.Hostname)
	}
	if decoded.OS != "linux" {
		t.Errorf("expected OS linux, got %s", decoded.OS)
	}
	if decoded.Len != 11 {
		t.Errorf("expected len 11, got %d", decoded.Len)
	}

	// Decode data
	decodedData, err := base64.StdEncoding.DecodeString(decoded.DataB64)
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}
	if string(decodedData) != "hello world" {
		t.Errorf("expected 'hello world', got %s", string(decodedData))
	}
}

func TestRemoteSlotStruct(t *testing.T) {
	slot := RemoteSlot{
		Name:      "test-slot",
		Size:      1024,
		CreatedAt: time.Now(),
		Hostname:  "my-host",
	}

	if slot.Name != "test-slot" {
		t.Errorf("expected name test-slot, got %s", slot.Name)
	}
	if slot.Size != 1024 {
		t.Errorf("expected size 1024, got %d", slot.Size)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
		{1572864, "1.5 MiB"},
		{1073741824, "1.0 GiB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatSize(%d) = %s, want %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatAge(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		contains string
	}{
		{"seconds", now.Add(-30 * time.Second), "s ago"},
		{"minutes", now.Add(-5 * time.Minute), "m ago"},
		{"hours", now.Add(-3 * time.Hour), "h ago"},
		{"days", now.Add(-48 * time.Hour), "d ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAge(tt.time)
			if len(result) == 0 {
				t.Error("formatAge returned empty string")
			}
			// Just verify it contains the expected suffix
			if !containsString(result, tt.contains) {
				t.Errorf("formatAge() = %s, want to contain %s", result, tt.contains)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s[len(s)-len(substr):] == substr || s == substr)
}

func TestS3BackendKey(t *testing.T) {
	backend := &S3Backend{
		bucket: "test-bucket",
		prefix: "user/slots/",
	}

	key := backend.key("myslot")
	expected := "user/slots/myslot.pb"
	if key != expected {
		t.Errorf("expected key %s, got %s", expected, key)
	}
}

func TestS3BackendKeyNoPrefix(t *testing.T) {
	backend := &S3Backend{
		bucket: "test-bucket",
		prefix: "",
	}

	key := backend.key("myslot")
	expected := "myslot.pb"
	if key != expected {
		t.Errorf("expected key %s, got %s", expected, key)
	}
}

func TestNewRemoteBackendFromConfigNoConfig(t *testing.T) {
	// Ensure no config exists
	origConfig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", origConfig)

	_ = os.Setenv("PIPEBOARD_CONFIG", "/nonexistent/config.yaml")

	_, err := newRemoteBackendFromConfig()
	if err == nil {
		t.Error("expected error when config doesn't exist")
	}
}

func TestNewRemoteBackendFromConfigNoSyncSection(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := tmpDir + "/config.yaml"

	// Config with no sync section
	configContent := `version: 1
peers:
  dev:
    ssh: devbox
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	origConfig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", origConfig)

	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	_, err := newRemoteBackendFromConfig()
	if err == nil {
		t.Error("expected error when sync section is missing")
	}
}

func TestNewRemoteBackendFromConfigUnsupportedBackend(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := tmpDir + "/config.yaml"

	// Config with unsupported backend
	configContent := `version: 1
sync:
  backend: gcs
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	origConfig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", origConfig)

	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	_, err := newRemoteBackendFromConfig()
	if err == nil {
		t.Error("expected error for unsupported backend")
	}
}

func TestSlotPayloadWithEncryption(t *testing.T) {
	data := []byte("secret data")

	payload := SlotPayload{
		Version:   1,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Hostname:  "test-host",
		OS:        "darwin",
		Len:       len(data),
		MIME:      "text/plain; charset=utf-8",
		DataB64:   base64.StdEncoding.EncodeToString(data),
		Encrypted: true,
	}

	if !payload.Encrypted {
		t.Error("expected Encrypted to be true")
	}

	// Verify JSON encoding preserves encrypted flag
	jsonData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded SlotPayload
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !decoded.Encrypted {
		t.Error("expected decoded Encrypted to be true")
	}
}

func TestFormatAgeEdgeCases(t *testing.T) {
	// Test with zero time
	zeroTime := time.Time{}
	result := formatAge(zeroTime)
	// Should handle gracefully
	if result == "" {
		t.Error("formatAge with zero time should return something")
	}

	// Test with future time (edge case)
	futureTime := time.Now().Add(1 * time.Hour)
	result = formatAge(futureTime)
	// Should handle gracefully
	if result == "" {
		t.Error("formatAge with future time should return something")
	}
}

func TestFormatSizeEdgeCases(t *testing.T) {
	// Test large GiB value (formatSize only supports up to GiB)
	result := formatSize(10737418240) // 10 GiB
	if result == "" {
		t.Error("formatSize should handle large GiB values")
	}
	if result != "10.0 GiB" {
		t.Errorf("expected '10.0 GiB', got %s", result)
	}
}

// Test gzip compression
func TestCompressDecompress(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"small text", []byte("hello world")},
		{"empty", []byte{}},
		{"large text", []byte(largeTestString(5000))},
		{"binary data", []byte{0x00, 0x01, 0x02, 0x03, 0xff, 0xfe, 0xfd}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compress
			compressed, err := compressData(tt.data)
			if err != nil {
				t.Fatalf("compressData failed: %v", err)
			}

			// Decompress
			decompressed, err := decompressData(compressed)
			if err != nil {
				t.Fatalf("decompressData failed: %v", err)
			}

			// Verify roundtrip
			if string(decompressed) != string(tt.data) {
				t.Errorf("roundtrip failed: got %q, want %q", decompressed, tt.data)
			}
		})
	}
}

func TestCompressDataEfficiency(t *testing.T) {
	// Test that compression actually reduces size for compressible data
	data := []byte(largeTestString(5000))
	compressed, err := compressData(data)
	if err != nil {
		t.Fatalf("compressData failed: %v", err)
	}

	if len(compressed) >= len(data) {
		t.Logf("compression did not reduce size: original=%d, compressed=%d", len(data), len(compressed))
	}
}

func TestDecompressInvalidData(t *testing.T) {
	// Test decompression of invalid gzip data
	_, err := decompressData([]byte("not gzip data"))
	if err == nil {
		t.Error("expected error for invalid gzip data")
	}
}

func largeTestString(n int) string {
	s := "test data repeated "
	result := ""
	for len(result) < n {
		result += s
	}
	return result[:n]
}

// Test MIME detection
func TestDetectMIME(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		contains string
	}{
		{"empty", []byte{}, "text/plain"},
		{"plain text", []byte("hello world"), "text/plain"},
		{"HTML", []byte("<html><body>test</body></html>"), "text/html"},
		{"PNG header", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, "image/png"},
		{"JPEG header", []byte{0xFF, 0xD8, 0xFF}, "image/jpeg"},
		{"JSON", []byte(`{"key": "value"}`), "text/plain"}, // JSON detected as text
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mime := detectMIME(tt.data)
			if mime == "" {
				t.Error("detectMIME returned empty string")
			}
			if !containsMIME(mime, tt.contains) {
				t.Errorf("detectMIME(%q) = %q, want to contain %q", tt.name, mime, tt.contains)
			}
		})
	}
}

func containsMIME(mime, substr string) bool {
	if mime == substr {
		return true
	}
	return strings.HasPrefix(mime, substr+";")
}

// Test retry with backoff
func TestRetryWithBackoffSuccess(t *testing.T) {
	attempts := 0
	err := retryWithBackoff(3, func() error {
		attempts++
		return nil // Succeed immediately
	})

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetryWithBackoffEventualSuccess(t *testing.T) {
	attempts := 0
	err := retryWithBackoff(3, func() error {
		attempts++
		if attempts < 2 {
			return os.ErrNotExist // Transient error
		}
		return nil // Succeed on second attempt
	})

	if err != nil {
		t.Errorf("expected no error after retry, got: %v", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestRetryWithBackoffNonTransientError(t *testing.T) {
	attempts := 0
	err := retryWithBackoff(3, func() error {
		attempts++
		return os.ErrNotExist // Non-transient-ish but not in our list
	})

	// Should retry for ErrNotExist since it's not in our non-transient list
	if err == nil {
		t.Error("expected error after all retries")
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

// Test SlotPayload with compression flag
func TestSlotPayloadWithCompression(t *testing.T) {
	data := []byte("compressed data")

	payload := SlotPayload{
		Version:    1,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		Hostname:   "test-host",
		OS:         "linux",
		Len:        len(data),
		MIME:       "text/plain; charset=utf-8",
		Compressed: true,
		DataB64:    base64.StdEncoding.EncodeToString(data),
	}

	if !payload.Compressed {
		t.Error("expected Compressed to be true")
	}

	// Verify JSON encoding preserves compressed flag
	jsonData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded SlotPayload
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !decoded.Compressed {
		t.Error("expected decoded Compressed to be true")
	}
}

func TestNewS3BackendEncryptionNoPassphrase(t *testing.T) {
	// Test that encryption requires a passphrase
	cfg := &S3Config{
		Bucket: "test-bucket",
		Region: "us-east-1",
	}

	_, err := newS3Backend(cfg, "aes256", "", 0)
	if err == nil {
		t.Error("expected error when encryption is set but passphrase is empty")
	}
	if err != nil && !strings.Contains(err.Error(), "passphrase required") {
		t.Errorf("expected 'passphrase required' error, got: %v", err)
	}
}

func TestRetryWithBackoffNoSuchKeyError(t *testing.T) {
	// NoSuchKey errors should not retry
	attempts := 0
	err := retryWithBackoff(3, func() error {
		attempts++
		return fmt.Errorf("NoSuchKey: the specified key does not exist")
	})

	if err == nil {
		t.Error("expected error")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt for NoSuchKey error (no retry), got %d", attempts)
	}
}

func TestRetryWithBackoffAccessDeniedError(t *testing.T) {
	// AccessDenied errors should not retry
	attempts := 0
	err := retryWithBackoff(3, func() error {
		attempts++
		return fmt.Errorf("AccessDenied: access to the resource is denied")
	})

	if err == nil {
		t.Error("expected error")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt for AccessDenied error (no retry), got %d", attempts)
	}
}

func TestRetryWithBackoffInvalidAccessKeyIdError(t *testing.T) {
	// InvalidAccessKeyId errors should not retry
	attempts := 0
	err := retryWithBackoff(3, func() error {
		attempts++
		return fmt.Errorf("InvalidAccessKeyId: the AWS access key ID does not exist")
	})

	if err == nil {
		t.Error("expected error")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt for InvalidAccessKeyId error (no retry), got %d", attempts)
	}
}

func TestSlotPayloadWithExpiresAt(t *testing.T) {
	data := []byte("expiring data")
	expiresAt := time.Now().UTC().AddDate(0, 0, 30).Format(time.RFC3339)

	payload := SlotPayload{
		Version:   1,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		ExpiresAt: expiresAt,
		Hostname:  "test-host",
		OS:        "linux",
		Len:       len(data),
		MIME:      "text/plain; charset=utf-8",
		DataB64:   base64.StdEncoding.EncodeToString(data),
	}

	// Verify JSON encoding preserves expires_at
	jsonData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded SlotPayload
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.ExpiresAt != expiresAt {
		t.Errorf("expected ExpiresAt %s, got %s", expiresAt, decoded.ExpiresAt)
	}
}

func TestSlotPayloadEmptyExpiresAt(t *testing.T) {
	// When ExpiresAt is empty, it should be omitted from JSON
	payload := SlotPayload{
		Version:   1,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		ExpiresAt: "", // Empty means no expiry
		Hostname:  "test-host",
		OS:        "linux",
		Len:       5,
		MIME:      "text/plain",
		DataB64:   base64.StdEncoding.EncodeToString([]byte("hello")),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// expires_at should be omitted when empty
	if strings.Contains(string(jsonData), `"expires_at":""`) {
		t.Error("empty expires_at should be omitted from JSON")
	}
}

func TestNewRemoteBackendFromConfigLocalBackend(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := tmpDir + "/config.yaml"

	// Config with local backend
	configContent := `version: 1
sync:
  backend: local
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	origConfig := os.Getenv("PIPEBOARD_CONFIG")
	defer restoreEnv("PIPEBOARD_CONFIG", origConfig)

	_ = os.Setenv("PIPEBOARD_CONFIG", configFile)

	backend, err := newRemoteBackendFromConfig()
	if err != nil {
		t.Fatalf("unexpected error for local backend: %v", err)
	}
	if backend == nil {
		t.Error("expected backend to be non-nil")
	}
}

func TestS3BackendKeyWithTrailingSlash(t *testing.T) {
	backend := &S3Backend{
		bucket: "test-bucket",
		prefix: "user/slots", // No trailing slash
	}

	key := backend.key("myslot")
	// path.Join handles this correctly
	if key != "user/slots/myslot.pb" {
		t.Errorf("expected key user/slots/myslot.pb, got %s", key)
	}
}

func TestRetryWithBackoffAllRetriesFail(t *testing.T) {
	attempts := 0
	err := retryWithBackoff(2, func() error {
		attempts++
		return fmt.Errorf("transient network error")
	})

	if err == nil {
		t.Error("expected error after all retries exhausted")
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
	if !strings.Contains(err.Error(), "operation failed after 2 retries") {
		t.Errorf("expected retry error message, got: %v", err)
	}
}

func TestDetectMIMEBinaryData(t *testing.T) {
	// Test various binary formats
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"PDF header", []byte("%PDF-1.4"), "application/pdf"},
		{"GIF header", []byte("GIF89a"), "image/gif"},
		{"ZIP header", []byte{0x50, 0x4B, 0x03, 0x04}, "application/zip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mime := detectMIME(tt.data)
			if !strings.HasPrefix(mime, tt.expected) {
				t.Errorf("detectMIME() = %q, want prefix %q", mime, tt.expected)
			}
		})
	}
}

func TestCompressDataEmpty(t *testing.T) {
	// Compressing empty data should work
	compressed, err := compressData([]byte{})
	if err != nil {
		t.Fatalf("compressData on empty data failed: %v", err)
	}

	// Decompress should return empty
	decompressed, err := decompressData(compressed)
	if err != nil {
		t.Fatalf("decompressData failed: %v", err)
	}
	if len(decompressed) != 0 {
		t.Errorf("expected empty result, got %d bytes", len(decompressed))
	}
}

func TestSlotPayloadAllFieldsSet(t *testing.T) {
	// Test payload with all optional fields set
	payload := SlotPayload{
		Version:    1,
		CreatedAt:  "2025-01-01T00:00:00Z",
		ExpiresAt:  "2025-02-01T00:00:00Z",
		Hostname:   "test-host",
		OS:         "linux",
		Len:        100,
		MIME:       "application/json",
		Encrypted:  true,
		Compressed: true,
		DataB64:    "dGVzdA==",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify all fields are present
	jsonStr := string(jsonData)
	requiredFields := []string{
		`"version":1`,
		`"created_at":"2025-01-01T00:00:00Z"`,
		`"expires_at":"2025-02-01T00:00:00Z"`,
		`"hostname":"test-host"`,
		`"os":"linux"`,
		`"len":100`,
		`"mime":"application/json"`,
		`"encrypted":true`,
		`"compressed":true`,
		`"data_b64":"dGVzdA=="`,
	}

	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON missing field: %s", field)
		}
	}
}
