package backup

import (
	"database/sql"
	"encoding/base64"
	"fmt"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/encryption"
)

func verifyUserDecrypt(u UserRow) error {
	if len(config.DBEncryptionKey) != 32 {
		return fmt.Errorf("DB_ENCRYPTION_KEY required to verify user %s", u.NameHash)
	}
	if _, err := decryptField(
		config.DBEncryptionKey, u.EncryptUserID,
		[]byte(u.EncryptedName), []byte(u.NameNonce),
	); err != nil {
		return fmt.Errorf("verify name for user %s: %w", u.NameHash, err)
	}
	if _, err := decryptField(
		config.DBEncryptionKey, u.EncryptUserID,
		[]byte(u.EncryptedPhone), []byte(u.PhoneNonce),
	); err != nil {
		return fmt.Errorf("verify phone for user %s: %w", u.NameHash, err)
	}
	return nil
}

func resolveUserID(u UserRow, backupKey []byte,
	fromCSV map[string]csvUser) (int, error) {
	var id int
	err := db.QueryRow(
		`SELECT id FROM users WHERE name_hash = $1`, u.NameHash,
	).Scan(&id)
	if err == nil {
		_, err = db.Exec(`
			UPDATE users SET
				has_account_picture = $1,
				account_picture_url = $2
			WHERE id = $3`,
			u.HasAccountPicture, u.AccountPictureURL, id,
		)
		if err != nil {
			return 0, fmt.Errorf(
				"update account picture for user %s: %w", u.NameHash, err)
		}
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("lookup user %s: %w", u.NameHash, err)
	}
	if len(config.DBEncryptionKey) == 0 {
		return 0, fmt.Errorf(
			"DB_ENCRYPTION_KEY required to restore new user %s",
			u.NameHash,
		)
	}

	err = db.QueryRow(`
		INSERT INTO users (
			created_at, deleted_at, is_admin,
			encrypted_name, name_nonce, name_hash,
			password_hash, password_salt, password_algo,
			encrypted_phone, phone_nonce, phone_hash,
			phone_verified, sms_opted_out, last_sms_sent_at,
			has_account_picture, account_picture_url
		) VALUES (
			$1, $2, $3,
			$4, $5, $6,
			$7, $8, $9,
			$10, $11, $12,
			$13, $14, $15,
			$16, $17
		) RETURNING id`,
		u.CreatedAt, u.DeletedAt, u.IsAdmin,
		[]byte{}, []byte{}, u.NameHash,
		u.PasswordHash, u.PasswordSalt, u.PasswordAlgo,
		[]byte{}, []byte{}, u.PhoneHash,
		u.PhoneVerified, u.SMSOptedOut, u.LastSMSSentAt,
		u.HasAccountPicture, u.AccountPictureURL,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert user %s: %w", u.NameHash, err)
	}

	name, phone, err := userPlaintext(u, backupKey, fromCSV)
	if err != nil {
		return 0, err
	}

	encName, nameNonce, err := encryption.Encrypt(
		id, name, config.DBEncryptionKey,
	)
	if err != nil {
		return 0, fmt.Errorf("encrypt name for user %s: %w", u.NameHash, err)
	}
	encPhone, phoneNonce, err := encryption.Encrypt(
		id, phone, config.DBEncryptionKey,
	)
	if err != nil {
		return 0, fmt.Errorf("encrypt phone for user %s: %w", u.NameHash, err)
	}

	encNameBytes, _ := base64.StdEncoding.DecodeString(encName)
	nameNonceBytes, _ := base64.StdEncoding.DecodeString(nameNonce)
	encPhoneBytes, _ := base64.StdEncoding.DecodeString(encPhone)
	phoneNonceBytes, _ := base64.StdEncoding.DecodeString(phoneNonce)

	nameHash := db.HashString(name)
	phoneHash := db.HashString(phone)

	_, err = db.Exec(`
		UPDATE users SET
			encrypted_name = $1, name_nonce = $2, name_hash = $3,
			encrypted_phone = $4, phone_nonce = $5, phone_hash = $6
		WHERE id = $7`,
		encNameBytes, nameNonceBytes, nameHash,
		encPhoneBytes, phoneNonceBytes, phoneHash,
		id,
	)
	if err != nil {
		return 0, fmt.Errorf("update encrypted user %s: %w", u.NameHash, err)
	}
	return id, nil
}

func userPlaintext(u UserRow, backupKey []byte,
	fromCSV map[string]csvUser) (string, string, error) {
	if fromCSV != nil {
		p, ok := fromCSV[csvUserKey(u.PasswordHash, u.PasswordSalt)]
		if !ok {
			return "", "", fmt.Errorf(
				"users.csv has no password_hash match for user %s",
				u.NameHash)
		}
		return p.name, p.phone, nil
	}

	decryptKey := backupKey
	if len(decryptKey) != 32 {
		decryptKey = config.DBEncryptionKey
	}
	name, err := decryptField(
		decryptKey, u.EncryptUserID,
		[]byte(u.EncryptedName), []byte(u.NameNonce),
	)
	if err != nil {
		return "", "", fmt.Errorf(
			"decrypt name for user %s: %w", u.NameHash, err)
	}
	phone, err := decryptField(
		decryptKey, u.EncryptUserID,
		[]byte(u.EncryptedPhone), []byte(u.PhoneNonce),
	)
	if err != nil {
		return "", "", fmt.Errorf(
			"decrypt phone for user %s: %w", u.NameHash, err)
	}
	return name, phone, nil
}

func decryptField(key []byte, encryptID int, ciphertext, nonce []byte) (string, error) {
	return encryption.Decrypt(
		encryptID,
		base64.StdEncoding.EncodeToString(ciphertext),
		base64.StdEncoding.EncodeToString(nonce),
		key,
	)
}
