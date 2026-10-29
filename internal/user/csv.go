package user

import (
	"encoding/csv"
	"io"
	"strconv"

	"github.com/partspile/site/internal/db"
)

// WriteCSV writes every user, including deleted accounts. Names and
// phones are decrypted. password_hash is the stored hash.
func WriteCSV(w io.Writer) (int, error) {
	rows, err := db.Query(userSelectFields + " ORDER BY id")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		u, err := scanUserFieldsOpt(rows, true)
		if err != nil {
			return 0, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	cw := csv.NewWriter(w)
	err = cw.Write([]string{
		"id", "name", "phone",
		"password_hash", "password_salt", "password_algo",
		"deleted",
	})
	if err != nil {
		return 0, err
	}
	for _, u := range users {
		deleted := "false"
		if u.DeletedAt != nil {
			deleted = "true"
		}
		err = cw.Write([]string{
			strconv.Itoa(u.ID), u.Name, u.PhoneE64,
			u.PasswordHash, u.PasswordSalt, u.PasswordAlgo,
			deleted,
		})
		if err != nil {
			return 0, err
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return 0, err
	}
	return len(users), nil
}
