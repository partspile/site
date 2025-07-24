package password

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters
const (
	Time    = 1
	Memory  = 64 * 1024
	Threads = 4
	KeyLen  = 32
)

// HashPassword hashes a password with a new random salt using Argon2id
func HashPassword(password string) (hash, salt string, err error) {
	// Generate a random salt
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", "", err
	}

	// Hash the password with Argon2id
	hashBytes := argon2.IDKey([]byte(password), saltBytes, Time, Memory, Threads, KeyLen)

	// Encode both hash and salt as base64
	hash = base64.RawStdEncoding.EncodeToString(hashBytes)
	salt = base64.RawStdEncoding.EncodeToString(saltBytes)

	return hash, salt, nil
}

// VerifyPassword verifies a password against a stored hash and salt
func VerifyPassword(password, hash, salt string) bool {
	// Decode the salt from base64
	saltBytes, err := base64.RawStdEncoding.DecodeString(salt)
	if err != nil {
		return false
	}

	// Hash the password with the same salt
	hashBytes := argon2.IDKey([]byte(password), saltBytes, Time, Memory, Threads, KeyLen)
	computedHash := base64.RawStdEncoding.EncodeToString(hashBytes)

	return computedHash == hash
}

// GenerateSalt generates a new random salt
func GenerateSalt() (string, error) {
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(saltBytes), nil
}

// HashPasswordWithSalt hashes a password with a provided salt
func HashPasswordWithSalt(password, salt string) (string, error) {
	saltBytes, err := base64.RawStdEncoding.DecodeString(salt)
	if err != nil {
		return "", err
	}

	hashBytes := argon2.IDKey([]byte(password), saltBytes, Time, Memory, Threads, KeyLen)
	return base64.RawStdEncoding.EncodeToString(hashBytes), nil
}
