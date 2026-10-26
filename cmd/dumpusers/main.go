package main

import (
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"os"

	"github.com/rocky-ads/site/internal/config"
	"github.com/rocky-ads/site/internal/db"
	"github.com/rocky-ads/site/internal/encryption"
)

func main() {
	if len(os.Args) > 1 {
		fmt.Fprintf(os.Stderr, `Usage: dumpusers

Writes a CSV of users to stdout. Names and phones are decrypted
with DB_ENCRYPTION_KEY. password_hash is the stored Argon2id
hash, not the password that was typed.

Requires DATABASE_URL and DB_ENCRYPTION_KEY.
`)
		os.Exit(2)
	}
	if len(config.DBEncryptionKey) != 32 {
		fmt.Fprintln(os.Stderr,
			"DB_ENCRYPTION_KEY must be set (32-byte base64)")
		os.Exit(1)
	}
	if err := db.Init(config.DatabaseURL); err != nil {
		fmt.Fprintf(os.Stderr, "database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	var rows []struct {
		ID             int    `db:"id"`
		EncryptedName  []byte `db:"encrypted_name"`
		NameNonce      []byte `db:"name_nonce"`
		EncryptedPhone []byte `db:"encrypted_phone"`
		PhoneNonce     []byte `db:"phone_nonce"`
		PasswordHash   string `db:"password_hash"`
		PasswordSalt   string `db:"password_salt"`
		PasswordAlgo   string `db:"password_algo"`
		Deleted        bool   `db:"deleted"`
	}
	err := db.Select(&rows, `
		SELECT id, encrypted_name, name_nonce,
		       encrypted_phone, phone_nonce,
		       password_hash, password_salt, password_algo,
		       deleted_at IS NOT NULL AS deleted
		FROM users
		ORDER BY id`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "select users: %v\n", err)
		os.Exit(1)
	}

	out := make([][]string, 0, len(rows))
	for _, u := range rows {
		name, err := encryption.Decrypt(
			u.ID,
			base64.StdEncoding.EncodeToString(u.EncryptedName),
			base64.StdEncoding.EncodeToString(u.NameNonce),
			config.DBEncryptionKey,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "user %d name: %v\n", u.ID, err)
			os.Exit(1)
		}
		phone, err := encryption.Decrypt(
			u.ID,
			base64.StdEncoding.EncodeToString(u.EncryptedPhone),
			base64.StdEncoding.EncodeToString(u.PhoneNonce),
			config.DBEncryptionKey,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "user %d phone: %v\n", u.ID, err)
			os.Exit(1)
		}
		deleted := "false"
		if u.Deleted {
			deleted = "true"
		}
		out = append(out, []string{
			fmt.Sprint(u.ID), name, phone,
			u.PasswordHash, u.PasswordSalt, u.PasswordAlgo,
			deleted,
		})
	}

	w := csv.NewWriter(os.Stdout)
	err = w.Write([]string{
		"id", "name", "phone",
		"password_hash", "password_salt", "password_algo",
		"deleted",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "csv: %v\n", err)
		os.Exit(1)
	}
	for _, rec := range out {
		if err := w.Write(rec); err != nil {
			fmt.Fprintf(os.Stderr, "csv: %v\n", err)
			os.Exit(1)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		fmt.Fprintf(os.Stderr, "csv: %v\n", err)
		os.Exit(1)
	}
}
