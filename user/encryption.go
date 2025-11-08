package user

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/encryption"
)

// EncryptName encrypts a user's name
func EncryptName(userID int, name string) (encryptedName, nonce string, err error) {
	return encryption.Encrypt(userID, name, config.UserEncryptionKey)
}

// DecryptName decrypts a user's name
func DecryptName(userID int, encryptedName, nonce string) (name string, err error) {
	return encryption.Decrypt(userID, encryptedName, nonce, config.UserEncryptionKey)
}

// EncryptPhone encrypts a user's phone number
func EncryptPhone(userID int, phone string) (encryptedPhone, nonce string, err error) {
	return encryption.Encrypt(userID, phone, config.UserEncryptionKey)
}

// DecryptPhone decrypts a user's phone number
func DecryptPhone(userID int, encryptedPhone, nonce string) (phone string, err error) {
	return encryption.Decrypt(userID, encryptedPhone, nonce, config.UserEncryptionKey)
}

// EncryptEmailAddress encrypts a user's email address
func EncryptEmailAddress(userID int, emailAddress string) (encryptedEmail, nonce string, err error) {
	return encryption.Encrypt(userID, emailAddress, config.UserEncryptionKey)
}

// DecryptEmailAddress decrypts a user's email address
func DecryptEmailAddress(userID int, encryptedEmail, nonce string) (emailAddress string, err error) {
	return encryption.Decrypt(userID, encryptedEmail, nonce, config.UserEncryptionKey)
}

// HashForLookup creates a SHA256 hash of a value for database lookups
// This allows us to search for users by phone/name without decrypting all records
func HashForLookup(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

