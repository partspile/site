package messaging

import (
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/encryption"
)

// EncryptMessage encrypts a message for a conversation
func EncryptMessage(conversationID int, plaintext string) (encryptedContent, nonce string, err error) {
	return encryption.Encrypt(conversationID, plaintext, config.MessageEncryptionKey)
}

// DecryptMessage decrypts an encrypted message for a conversation
func DecryptMessage(conversationID int, encryptedContent, nonce string) (plaintext string, err error) {
	return encryption.Decrypt(conversationID, encryptedContent, nonce, config.MessageEncryptionKey)
}
