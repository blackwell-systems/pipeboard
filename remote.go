package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// SlotPayload is the JSON envelope stored in remote slots
type SlotPayload struct {
	Version    int    `json:"version"`
	CreatedAt  string `json:"created_at"`
	ExpiresAt  string `json:"expires_at,omitempty"` // RFC3339 timestamp for TTL
	Hostname   string `json:"hostname"`
	OS         string `json:"os"`
	Len        int    `json:"len"`
	MIME       string `json:"mime"`
	Encrypted  bool   `json:"encrypted,omitempty"`  // true if data is client-side encrypted
	Compressed bool   `json:"compressed,omitempty"` // true if data is gzip compressed
	DataB64    string `json:"data_b64"`
}

// compressData compresses data using gzip
func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// decompressData decompresses gzip data
func decompressData(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return io.ReadAll(r)
}

// detectMIME detects the MIME type of data
func detectMIME(data []byte) string {
	if len(data) == 0 {
		return "text/plain; charset=utf-8"
	}
	// Use http.DetectContentType for accurate detection
	mimeType := http.DetectContentType(data)
	return mimeType
}

// retryWithBackoff retries an operation with exponential backoff
func retryWithBackoff(maxRetries int, operation func() error) error {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := operation(); err != nil {
			lastErr = err
			// Don't retry on non-transient errors
			if strings.Contains(err.Error(), "NoSuchKey") ||
				strings.Contains(err.Error(), "AccessDenied") ||
				strings.Contains(err.Error(), "InvalidAccessKeyId") {
				return err
			}
			// Exponential backoff with jitter
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
			time.Sleep(backoff + jitter)
			continue
		}
		return nil
	}
	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}

// RemoteSlot represents metadata about a stored slot
type RemoteSlot struct {
	Name      string
	Size      int64
	CreatedAt time.Time
	ExpiresAt time.Time // Zero value means no expiry
	Hostname  string
}

// RemoteBackend defines the interface for remote clipboard sync
type RemoteBackend interface {
	Push(slot string, data []byte, meta map[string]string) error
	Pull(slot string) ([]byte, map[string]string, error)
	List() ([]RemoteSlot, error)
	Delete(slot string) error
}

// S3Backend implements RemoteBackend using AWS S3
type S3Backend struct {
	client     *s3.Client
	bucket     string
	prefix     string
	sse        string
	encryption string // "none" or "aes256" for client-side encryption
	passphrase string // passphrase for client-side encryption
	ttlDays    int    // TTL in days (0 = never expires)
}

func newRemoteBackendFromConfig() (RemoteBackend, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	switch cfg.Sync.Backend {
	case "s3":
		return newS3Backend(cfg.Sync.S3, cfg.Sync.Encryption, cfg.Sync.Passphrase, cfg.Sync.TTLDays)
	case "local":
		return newLocalBackend(cfg.Sync.Local, cfg.Sync.Encryption, cfg.Sync.Passphrase, cfg.Sync.TTLDays)
	case "hosted":
		return newHostedBackend(cfg.Sync.Hosted, cfg.Sync.Encryption, cfg.Sync.Passphrase, cfg.Sync.TTLDays)
	default:
		return nil, fmt.Errorf("unsupported backend: %s", cfg.Sync.Backend)
	}
}

func newS3Backend(cfg *S3Config, encryption, passphrase string, ttlDays int) (*S3Backend, error) {
	ctx := context.Background()

	// Validate encryption config
	if encryption == "aes256" && passphrase == "" {
		return nil, fmt.Errorf("passphrase required when encryption is set to aes256")
	}

	var awsCfg aws.Config
	var err error

	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	if cfg.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(cfg.Profile))
	}

	// Check for explicit credentials in environment (useful for testing)
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				os.Getenv("AWS_ACCESS_KEY_ID"),
				os.Getenv("AWS_SECRET_ACCESS_KEY"),
				os.Getenv("AWS_SESSION_TOKEN"),
			),
		))
	}

	awsCfg, err = config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)

	return &S3Backend{
		client:     client,
		bucket:     cfg.Bucket,
		prefix:     cfg.Prefix,
		sse:        cfg.SSE,
		encryption: encryption,
		passphrase: passphrase,
		ttlDays:    ttlDays,
	}, nil
}

func (b *S3Backend) key(slot string) string {
	return path.Join(b.prefix, slot+".pb")
}

func (b *S3Backend) Push(slot string, data []byte, meta map[string]string) error {
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

	// Apply gzip compression for data > 1KB (saves bandwidth/storage)
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

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encoding payload: %w", err)
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(b.bucket),
		Key:         aws.String(b.key(slot)),
		Body:        bytes.NewReader(jsonData),
		ContentType: aws.String("application/json"),
	}

	// Apply server-side encryption
	switch b.sse {
	case "AES256":
		input.ServerSideEncryption = types.ServerSideEncryptionAes256
	case "aws:kms":
		input.ServerSideEncryption = types.ServerSideEncryptionAwsKms
	}

	// Use retry with exponential backoff for network resilience
	return retryWithBackoff(3, func() error {
		ctx := context.Background()
		_, err := b.client.PutObject(ctx, input)
		if err != nil {
			return fmt.Errorf("uploading to S3: %w", err)
		}
		return nil
	})
}

func (b *S3Backend) Pull(slot string) ([]byte, map[string]string, error) {
	var jsonData []byte

	// Use retry with exponential backoff for network resilience
	err := retryWithBackoff(3, func() error {
		ctx := context.Background()
		result, err := b.client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(b.bucket),
			Key:    aws.String(b.key(slot)),
		})
		if err != nil {
			return fmt.Errorf("fetching from S3: %w", err)
		}
		defer func() { _ = result.Body.Close() }()

		jsonData, err = io.ReadAll(result.Body)
		if err != nil {
			return fmt.Errorf("reading S3 object: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
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

func (b *S3Backend) List() ([]RemoteSlot, error) {
	ctx := context.Background()

	// Note: Unlike LocalBackend, we don't check expiry here because it would
	// require fetching each object's content (expensive S3 GET requests).
	// Expired slots are cleaned up lazily on Pull() instead.

	// Use paginator to handle more than 1000 objects
	paginator := s3.NewListObjectsV2Paginator(b.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(b.bucket),
		Prefix: aws.String(b.prefix),
	})

	var slots []RemoteSlot
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing S3 objects: %w", err)
		}

		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)

			// Skip if not a .pb file
			if !strings.HasSuffix(key, ".pb") {
				continue
			}

			// Extract slot name
			name := strings.TrimPrefix(key, b.prefix)
			name = strings.TrimPrefix(name, "/")
			name = strings.TrimSuffix(name, ".pb")

			slot := RemoteSlot{
				Name:      name,
				Size:      aws.ToInt64(obj.Size),
				CreatedAt: aws.ToTime(obj.LastModified),
			}

			// Try to get hostname from object metadata (optional, may require HEAD request)
			// For now, we'll get it when showing details
			slots = append(slots, slot)
		}
	}

	return slots, nil
}

func (b *S3Backend) Delete(slot string) error {
	ctx := context.Background()

	_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(b.key(slot)),
	})
	if err != nil {
		return fmt.Errorf("deleting from S3: %w", err)
	}

	return nil
}

// formatSize returns a human-readable size string
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMG"[exp])
}

// formatAge returns a human-readable age string
func formatAge(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

// formatTimeUntil returns a human-readable time until string
func formatTimeUntil(t time.Time) string {
	d := time.Until(t)

	if d < 0 {
		return "expired"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
