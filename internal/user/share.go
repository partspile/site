package user

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"errors"

	"github.com/partspile/site/internal/config"
)

const (
	shareNonceSize = 12
	shareIDSize    = 4
	shareTagSize   = 16
	shareRawSize   = shareNonceSize + shareIDSize + shareTagSize
)

func ShareToken(userID int) (string, error) {
	key := config.ShareSecret
	if len(key) != 32 {
		return "", errors.New("share secret not configured")
	}
	if userID < 1 {
		return "", sql.ErrNoRows
	}

	var idBytes [shareIDSize]byte
	binary.BigEndian.PutUint32(idBytes[:], uint32(userID))
	nonce := shareNonce(key, idBytes[:])

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ct := gcm.Seal(nil, nonce, idBytes[:], nil)
	out := append(nonce, ct...)
	return base64.RawURLEncoding.EncodeToString(out), nil
}

func shareNonce(key, idBytes []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("share-nonce"))
	mac.Write(idBytes)
	return mac.Sum(nil)[:shareNonceSize]
}

func ValidShareToken(token string) bool {
	_, err := parseShareToken(token)
	return err == nil
}

func GetIDByShareToken(token string) (int, error) {
	id, err := parseShareToken(token)
	if err != nil {
		return 0, err
	}
	if !Exists(id) {
		return 0, sql.ErrNoRows
	}
	return id, nil
}

func parseShareToken(token string) (int, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != shareRawSize {
		return 0, sql.ErrNoRows
	}
	key := config.ShareSecret
	if len(key) != 32 {
		return 0, sql.ErrNoRows
	}
	nonce, ct := raw[:shareNonceSize], raw[shareNonceSize:]
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, sql.ErrNoRows
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return 0, sql.ErrNoRows
	}
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil || len(pt) != shareIDSize {
		return 0, sql.ErrNoRows
	}
	id := int(binary.BigEndian.Uint32(pt))
	if id < 1 {
		return 0, sql.ErrNoRows
	}
	return id, nil
}
