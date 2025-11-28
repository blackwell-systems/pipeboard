package main

import (
	"bytes"
	"testing"
)

func TestDeriveKey(t *testing.T) {
	passphrase := "test-passphrase"
	salt := []byte("0123456789abcdef") // 16 bytes

	key := deriveKey(passphrase, salt)

	if len(key) != keySize {
		t.Errorf("expected key size %d, got %d", keySize, len(key))
	}

	// Same inputs should produce same key
	key2 := deriveKey(passphrase, salt)
	if !bytes.Equal(key, key2) {
		t.Error("deriveKey should be deterministic")
	}

	// Different salt should produce different key
	salt2 := []byte("abcdef0123456789")
	key3 := deriveKey(passphrase, salt2)
	if bytes.Equal(key, key3) {
		t.Error("different salt should produce different key")
	}

	// Different passphrase should produce different key
	key4 := deriveKey("different-passphrase", salt)
	if bytes.Equal(key, key4) {
		t.Error("different passphrase should produce different key")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name       string
		plaintext  []byte
		passphrase string
	}{
		{
			name:       "simple text",
			plaintext:  []byte("hello world"),
			passphrase: "test-pass",
		},
		{
			name:       "empty data",
			plaintext:  []byte(""),
			passphrase: "test-pass",
		},
		{
			name:       "binary data",
			plaintext:  []byte{0x00, 0x01, 0x02, 0xff, 0xfe, 0xfd},
			passphrase: "test-pass",
		},
		{
			name:       "large data",
			plaintext:  bytes.Repeat([]byte("x"), 10000),
			passphrase: "test-pass",
		},
		{
			name:       "unicode passphrase",
			plaintext:  []byte("test data"),
			passphrase: "パスワード🔐",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := encrypt(tt.plaintext, tt.passphrase)
			if err != nil {
				t.Fatalf("encrypt() error: %v", err)
			}

			// Encrypted should be larger than plaintext (salt + nonce + tag)
			minSize := len(tt.plaintext) + saltSize + nonceSize + 16
			if len(encrypted) < minSize {
				t.Errorf("encrypted data too small: got %d, want at least %d", len(encrypted), minSize)
			}

			decrypted, err := decrypt(encrypted, tt.passphrase)
			if err != nil {
				t.Fatalf("decrypt() error: %v", err)
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("decrypted data doesn't match original")
			}
		})
	}
}

func TestEncryptEmptyPassphrase(t *testing.T) {
	_, err := encrypt([]byte("test"), "")
	if err == nil {
		t.Error("encrypt with empty passphrase should return error")
	}
}

func TestDecryptEmptyPassphrase(t *testing.T) {
	_, err := decrypt([]byte("some-data-here-with-enough-bytes"), "")
	if err == nil {
		t.Error("decrypt with empty passphrase should return error")
	}
}

func TestDecryptTooShort(t *testing.T) {
	_, err := decrypt([]byte("short"), "test-pass")
	if err == nil {
		t.Error("decrypt with too-short data should return error")
	}
}

func TestDecryptWrongPassphrase(t *testing.T) {
	plaintext := []byte("secret message")
	encrypted, err := encrypt(plaintext, "correct-passphrase")
	if err != nil {
		t.Fatalf("encrypt() error: %v", err)
	}

	_, err = decrypt(encrypted, "wrong-passphrase")
	if err == nil {
		t.Error("decrypt with wrong passphrase should return error")
	}
}

func TestDecryptCorruptedData(t *testing.T) {
	plaintext := []byte("secret message")
	encrypted, err := encrypt(plaintext, "test-pass")
	if err != nil {
		t.Fatalf("encrypt() error: %v", err)
	}

	// Corrupt a byte in the ciphertext portion
	encrypted[len(encrypted)-5] ^= 0xff

	_, err = decrypt(encrypted, "test-pass")
	if err == nil {
		t.Error("decrypt with corrupted data should return error")
	}
}

func TestEncryptProducesDifferentOutput(t *testing.T) {
	plaintext := []byte("same message")
	passphrase := "same-passphrase"

	encrypted1, err := encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatalf("encrypt() error: %v", err)
	}

	encrypted2, err := encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatalf("encrypt() error: %v", err)
	}

	// Due to random salt and nonce, outputs should differ
	if bytes.Equal(encrypted1, encrypted2) {
		t.Error("encrypt should produce different output each time (random salt/nonce)")
	}

	// But both should decrypt to the same plaintext
	decrypted1, _ := decrypt(encrypted1, passphrase)
	decrypted2, _ := decrypt(encrypted2, passphrase)

	if !bytes.Equal(decrypted1, decrypted2) {
		t.Error("both encrypted values should decrypt to same plaintext")
	}
}
