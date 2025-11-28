package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
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
