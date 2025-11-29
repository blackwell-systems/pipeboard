package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// LocalConfig holds configuration for local file backend
type LocalConfig struct {
	Path string `yaml:"path,omitempty"` // directory for slot storage
}

// LocalBackend implements RemoteBackend using local filesystem
type LocalBackend struct {
	path       string
	encryption string
	passphrase string
	ttlDays    int
}

func newLocalBackend(cfg *LocalConfig, encryption, passphrase string, ttlDays int) (*LocalBackend, error) {
	// Validate encryption config
	if encryption == "aes256" && passphrase == "" {
		return nil, fmt.Errorf("passphrase required when encryption is set to aes256")
	}

	path := cfg.Path
	if path == "" {
		// Default to ~/.config/pipeboard/slots
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("could not determine home directory: %w", err)
			}
			configDir = filepath.Join(home, ".config")
		}
		path = filepath.Join(configDir, "pipeboard", "slots")
	}

	// Ensure directory exists
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, fmt.Errorf("creating slots directory: %w", err)
	}

	return &LocalBackend{
		path:       path,
		encryption: encryption,
		passphrase: passphrase,
		ttlDays:    ttlDays,
	}, nil
}

func (b *LocalBackend) slotPath(slot string) string {
	return filepath.Join(b.path, slot+".pb")
}

func (b *LocalBackend) Push(slot string, data []byte, meta map[string]string) error {
	hostname := meta["hostname"]
	if hostname == "" {
		hostname, _ = os.Hostname()
	}

	// Detect MIME type before any transformations
	mimeType := detectMIME(data)

	// Store original data for processing
	storeData := data
	compressed := false
	encrypted := false

	// Apply gzip compression for data > 1KB (saves storage)
	if len(data) > 1024 {
		compressedData, err := compressData(data)
		if err == nil && len(compressedData) < len(data) {
			// Only use compression if it actually reduces size
			storeData = compressedData
			compressed = true
		}
	}

	// Apply client-side encryption if configured (after compression)
	if b.encryption == "aes256" && b.passphrase != "" {
		encData, err := encrypt(storeData, b.passphrase)
		if err != nil {
			return fmt.Errorf("encrypting data: %w", err)
		}
		storeData = encData
		encrypted = true
	}

	payload := SlotPayload{
		Version:    1,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		Hostname:   hostname,
		OS:         runtime.GOOS,
		Len:        len(data), // Original length before compression/encryption
		MIME:       mimeType,
		Encrypted:  encrypted,
		Compressed: compressed,
		DataB64:    base64.StdEncoding.EncodeToString(storeData),
	}

	// Set expiry time if TTL configured
	if b.ttlDays > 0 {
		payload.ExpiresAt = time.Now().UTC().AddDate(0, 0, b.ttlDays).Format(time.RFC3339)
	}

	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding payload: %w", err)
	}

	if err := os.WriteFile(b.slotPath(slot), jsonData, 0600); err != nil {
		return fmt.Errorf("writing slot file: %w", err)
	}

	return nil
}

func (b *LocalBackend) Pull(slot string) ([]byte, map[string]string, error) {
	jsonData, err := os.ReadFile(b.slotPath(slot))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("slot %q not found", slot)
		}
		return nil, nil, fmt.Errorf("reading slot file: %w", err)
	}

	var payload SlotPayload
	if err := json.Unmarshal(jsonData, &payload); err != nil {
		return nil, nil, fmt.Errorf("decoding payload: %w", err)
	}

	// Check if slot has expired
	if payload.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, payload.ExpiresAt)
		if err == nil && time.Now().UTC().After(expiresAt) {
			// Auto-delete expired slot
			_ = b.Delete(slot)
			return nil, nil, fmt.Errorf("slot %q has expired", slot)
		}
	}

	data, err := base64.StdEncoding.DecodeString(payload.DataB64)
	if err != nil {
		return nil, nil, fmt.Errorf("decoding base64 data: %w", err)
	}

	// Decrypt if the payload was encrypted (before decompression)
	if payload.Encrypted {
		if b.passphrase == "" {
			return nil, nil, fmt.Errorf("slot is encrypted but no passphrase configured")
		}
		decData, err := decrypt(data, b.passphrase)
		if err != nil {
			return nil, nil, fmt.Errorf("decrypting data: %w", err)
		}
		data = decData
	}

	// Decompress if the payload was compressed (after decryption)
	if payload.Compressed {
		decompressedData, err := decompressData(data)
		if err != nil {
			return nil, nil, fmt.Errorf("decompressing data: %w", err)
		}
		data = decompressedData
	}

	meta := map[string]string{
		"hostname":   payload.Hostname,
		"os":         payload.OS,
		"created_at": payload.CreatedAt,
		"mime":       payload.MIME,
	}

	return data, meta, nil
}

func (b *LocalBackend) List() ([]RemoteSlot, error) {
	entries, err := os.ReadDir(b.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []RemoteSlot{}, nil
		}
		return nil, fmt.Errorf("reading slots directory: %w", err)
	}

	var slots []RemoteSlot
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".pb") {
			continue
		}

		slotName := strings.TrimSuffix(name, ".pb")

		info, err := entry.Info()
		if err != nil {
			continue
		}

		slots = append(slots, RemoteSlot{
			Name:      slotName,
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}

	return slots, nil
}

func (b *LocalBackend) Delete(slot string) error {
	err := os.Remove(b.slotPath(slot))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("slot %q not found", slot)
		}
		return fmt.Errorf("deleting slot file: %w", err)
	}
	return nil
}
