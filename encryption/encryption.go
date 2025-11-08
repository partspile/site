package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/hkdf"
)

const (
	NonceSize = 12 // For AES-GCM
	KeySize   = 32 // 256 bits
)

// Encrypt encrypts plaintext data using an ID
// Returns encrypted content (base64) and nonce (base64)
func Encrypt(id int, plaintext string, masterKey []byte) (encryptedContent, nonce string, err error) {
	// Derive key for this ID
	key, err := deriveKey(id, masterKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to derive key: %w", err)
	}

	// Create AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonceBytes := make([]byte, NonceSize)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	plaintextBytes := []byte(plaintext)
	ciphertext := aesGCM.Seal(nil, nonceBytes, plaintextBytes, nil)

	// Encode as base64 for storage
	encryptedContent = base64.StdEncoding.EncodeToString(ciphertext)
	nonce = base64.StdEncoding.EncodeToString(nonceBytes)

	return encryptedContent, nonce, nil
}

// Decrypt decrypts encrypted data using an ID
func Decrypt(id int, encryptedContent, nonce string, masterKey []byte) (plaintext string, err error) {
	// Derive key for this ID
	key, err := deriveKey(id, masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to derive key: %w", err)
	}

	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedContent)
	if err != nil {
		return "", fmt.Errorf("invalid encrypted content: %w", err)
	}

	nonceBytes, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", fmt.Errorf("invalid nonce: %w", err)
	}

	// Create AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt
	plaintextBytes, err := aesGCM.Open(nil, nonceBytes, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintextBytes), nil
}

// deriveKey derives an encryption key for a specific ID
// Uses HKDF-SHA256 to derive key from ID and master key
// Keys are derived on-the-fly, never stored
func deriveKey(id int, masterKey []byte) ([]byte, error) {
	if len(masterKey) != KeySize {
		return nil, fmt.Errorf("invalid master key size: expected %d, got %d", KeySize, len(masterKey))
	}

	// Convert ID to bytes for HKDF salt
	idBytes := []byte(fmt.Sprintf("%d", id))

	// Derive key using HKDF-SHA256
	hkdf := hkdf.New(sha256.New, masterKey, idBytes, []byte("message-encryption-v1"))

	key := make([]byte, KeySize)
	if _, err := hkdf.Read(key); err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	return key, nil
}

// VerifyEncryptedData verifies that encrypted data is valid
func VerifyEncryptedData(encryptedContent, nonce string) error {
	if encryptedContent == "" {
		return fmt.Errorf("encrypted content is empty")
	}
	if nonce == "" {
		return fmt.Errorf("nonce is empty")
	}

	// Verify base64 encoding
	if _, err := base64.StdEncoding.DecodeString(encryptedContent); err != nil {
		return fmt.Errorf("invalid encrypted content encoding: %w", err)
	}
	if _, err := base64.StdEncoding.DecodeString(nonce); err != nil {
		return fmt.Errorf("invalid nonce encoding: %w", err)
	}

	return nil
}
