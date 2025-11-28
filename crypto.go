package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	saltSize   = 16
	nonceSize  = 12
	keySize    = 32 // AES-256
	iterations = 100000
)

// deriveKey derives a 256-bit key from a passphrase using PBKDF2
func deriveKey(passphrase string, salt []byte) []byte {
	return pbkdf2.Key([]byte(passphrase), salt, iterations, keySize, sha256.New)
}

// encrypt encrypts data using AES-256-GCM with a passphrase
// Returns: salt (16 bytes) + nonce (12 bytes) + ciphertext
func encrypt(data []byte, passphrase string) ([]byte, error) {
	if passphrase == "" {
		return nil, errors.New("passphrase cannot be empty")
	}

	// Generate random salt
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	// Derive key from passphrase
	key := deriveKey(passphrase, salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate random nonce
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt
	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// Combine: salt + nonce + ciphertext
	result := make([]byte, saltSize+nonceSize+len(ciphertext))
	copy(result[:saltSize], salt)
	copy(result[saltSize:saltSize+nonceSize], nonce)
	copy(result[saltSize+nonceSize:], ciphertext)

	return result, nil
}

// decrypt decrypts data that was encrypted with encrypt()
func decrypt(data []byte, passphrase string) ([]byte, error) {
	if passphrase == "" {
		return nil, errors.New("passphrase cannot be empty")
	}

	if len(data) < saltSize+nonceSize+16 { // 16 = minimum GCM tag size
		return nil, errors.New("ciphertext too short")
	}

	// Extract salt, nonce, ciphertext
	salt := data[:saltSize]
	nonce := data[saltSize : saltSize+nonceSize]
	ciphertext := data[saltSize+nonceSize:]

	// Derive key from passphrase
	key := deriveKey(passphrase, salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed (wrong passphrase?)")
	}

	return plaintext, nil
}
