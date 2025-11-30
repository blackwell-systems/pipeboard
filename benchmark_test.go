package main

import (
	"crypto/rand"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

// Benchmark encryption/decryption operations
func BenchmarkEncrypt(b *testing.B) {
	data := make([]byte, 1024) // 1KB
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}
	passphrase := "benchmark-passphrase"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encrypt(data, passphrase)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncrypt10KB(b *testing.B) {
	data := make([]byte, 10*1024) // 10KB
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}
	passphrase := "benchmark-passphrase"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encrypt(data, passphrase)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncrypt100KB(b *testing.B) {
	data := make([]byte, 100*1024) // 100KB
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}
	passphrase := "benchmark-passphrase"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encrypt(data, passphrase)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecrypt(b *testing.B) {
	data := make([]byte, 1024) // 1KB
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}
	passphrase := "benchmark-passphrase"
	encrypted, _ := encrypt(data, passphrase)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := decrypt(encrypted, passphrase)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecrypt10KB(b *testing.B) {
	data := make([]byte, 10*1024) // 10KB
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}
	passphrase := "benchmark-passphrase"
	encrypted, _ := encrypt(data, passphrase)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := decrypt(encrypted, passphrase)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark compression operations
func BenchmarkCompress1KB(b *testing.B) {
	// Text data compresses better than random
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte('a' + (i % 26))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := compressData(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompress10KB(b *testing.B) {
	data := make([]byte, 10*1024)
	for i := range data {
		data[i] = byte('a' + (i % 26))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := compressData(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompress100KB(b *testing.B) {
	data := make([]byte, 100*1024)
	for i := range data {
		data[i] = byte('a' + (i % 26))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := compressData(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecompress10KB(b *testing.B) {
	data := make([]byte, 10*1024)
	for i := range data {
		data[i] = byte('a' + (i % 26))
	}
	compressed, _ := compressData(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := decompressData(compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark MIME detection
func BenchmarkDetectMIME(b *testing.B) {
	data := []byte(`{"key": "value", "number": 123}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detectMIME(data)
	}
}

func BenchmarkDetectMIMELarge(b *testing.B) {
	// Large JSON-like data
	data := make([]byte, 10*1024)
	copy(data, []byte(`{"key": "`))
	for i := 8; i < len(data)-2; i++ {
		data[i] = 'x'
	}
	copy(data[len(data)-2:], []byte(`"}`))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detectMIME(data)
	}
}

// Benchmark local backend operations
func BenchmarkLocalBackendPush(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "pipeboard-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	backend, err := newLocalBackend(&LocalConfig{Path: tmpDir}, "", "", 0)
	if err != nil {
		b.Fatal(err)
	}
	data := make([]byte, 1024)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}
	meta := map[string]string{"content-type": "application/octet-stream"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := backend.Push("bench-slot", data, meta)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLocalBackendPull(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "pipeboard-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	backend, err := newLocalBackend(&LocalConfig{Path: tmpDir}, "", "", 0)
	if err != nil {
		b.Fatal(err)
	}
	data := make([]byte, 1024)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}
	meta := map[string]string{"content-type": "application/octet-stream"}
	if err := backend.Push("bench-slot", data, meta); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := backend.Pull("bench-slot")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLocalBackendList(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "pipeboard-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	backend, err := newLocalBackend(&LocalConfig{Path: tmpDir}, "", "", 0)
	if err != nil {
		b.Fatal(err)
	}
	data := []byte("test data")
	meta := map[string]string{}

	// Create 100 slots
	for i := 0; i < 100; i++ {
		if err := backend.Push(filepath.Base(tmpDir)+string(rune('a'+i%26)), data, meta); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := backend.List()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark config operations
func BenchmarkConfigPath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = configPath()
	}
}

// Benchmark hash computation (used in watch mode)
func BenchmarkSHA256_1KB(b *testing.B) {
	data := make([]byte, 1024)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hashData(data)
	}
}

func BenchmarkSHA256_100KB(b *testing.B) {
	data := make([]byte, 100*1024)
	if _, err := rand.Read(data); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hashData(data)
	}
}

// hashData computes SHA256 hash - helper for benchmarking
func hashData(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// Benchmark string truncation (used in history display)
func BenchmarkTruncateString(b *testing.B) {
	s := "This is a test string that needs to be truncated to fit within the display width"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = truncateString(s, 40)
	}
}

func BenchmarkTruncateStringLong(b *testing.B) {
	s := make([]byte, 10000)
	for i := range s {
		s[i] = byte('a' + (i % 26))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = truncateString(string(s), 40)
	}
}

// Benchmark key derivation (expensive by design)
func BenchmarkDeriveKey(b *testing.B) {
	passphrase := "benchmark-passphrase"
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = deriveKey(passphrase, salt)
	}
}
