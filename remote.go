package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
	Version   int    `json:"version"`
	CreatedAt string `json:"created_at"`
	Hostname  string `json:"hostname"`
	OS        string `json:"os"`
	Len       int    `json:"len"`
	MIME      string `json:"mime"`
	DataB64   string `json:"data_b64"`
}

// RemoteSlot represents metadata about a stored slot
type RemoteSlot struct {
	Name      string
	Size      int64
	CreatedAt time.Time
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
	client *s3.Client
	bucket string
	prefix string
	sse    string
}

func newRemoteBackendFromConfig() (RemoteBackend, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	switch cfg.Sync.Backend {
	case "s3":
		return newS3Backend(cfg.Sync.S3)
	default:
		return nil, fmt.Errorf("unsupported backend: %s", cfg.Sync.Backend)
	}
}

func newS3Backend(cfg *S3Config) (*S3Backend, error) {
	ctx := context.Background()

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
		client: client,
		bucket: cfg.Bucket,
		prefix: cfg.Prefix,
		sse:    cfg.SSE,
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

	payload := SlotPayload{
		Version:   1,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Hostname:  hostname,
		OS:        runtime.GOOS,
		Len:       len(data),
		MIME:      "text/plain; charset=utf-8",
		DataB64:   base64.StdEncoding.EncodeToString(data),
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

	ctx := context.Background()
	_, err = b.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("uploading to S3: %w", err)
	}

	return nil
}

func (b *S3Backend) Pull(slot string) ([]byte, map[string]string, error) {
	ctx := context.Background()

	result, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(b.key(slot)),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("fetching from S3: %w", err)
	}
	defer func() { _ = result.Body.Close() }()

	jsonData, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading S3 object: %w", err)
	}

	var payload SlotPayload
	if err := json.Unmarshal(jsonData, &payload); err != nil {
		return nil, nil, fmt.Errorf("decoding payload: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(payload.DataB64)
	if err != nil {
		return nil, nil, fmt.Errorf("decoding base64 data: %w", err)
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

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(b.bucket),
		Prefix: aws.String(b.prefix),
	}

	result, err := b.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("listing S3 objects: %w", err)
	}

	var slots []RemoteSlot
	for _, obj := range result.Contents {
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
