package user

import (
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/encryption"
)

func EncryptName(userID int,
	name string) (encryptedName, nonce string, err error) {
	return encryption.Encrypt(userID, name, config.DBEncryptionKey)
}

func decryptName(userID int, encryptedName,
	nonce string) (name string, err error) {
	return encryption.Decrypt(userID, encryptedName, nonce, config.DBEncryptionKey)
}

func EncryptPhone(userID int,
	phone string) (encryptedPhone, nonce string, err error) {
	return encryption.Encrypt(userID, phone, config.DBEncryptionKey)
}

func decryptPhone(userID int, encryptedPhone,
	nonce string) (phone string, err error) {
	return encryption.Decrypt(userID, encryptedPhone, nonce, config.DBEncryptionKey)
}
