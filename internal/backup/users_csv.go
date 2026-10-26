package backup

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const usersCSVName = "users.csv"

type csvUser struct {
	name  string
	phone string
}

func usersCSVPath() string {
	return filepath.Join(DefaultDir, usersCSVName)
}

func csvUserKey(hash, salt string) string {
	return hash + "\x00" + salt
}

// loadUsersCSV reads a dumpusers CSV. Rows are keyed by password_hash
// and password_salt, which the archive stores in the clear.
func loadUsersCSV(path string) (map[string]csvUser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("%s is empty", path)
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	nameCol, err := csvCol(header, "name")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	phoneCol, err := csvCol(header, "phone")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	hashCol, err := csvCol(header, "password_hash")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	saltCol, err := csvCol(header, "password_salt")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	need := nameCol
	for _, col := range []int{phoneCol, hashCol, saltCol} {
		if col > need {
			need = col
		}
	}

	out := make(map[string]csvUser)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		if len(rec) <= need {
			return nil, fmt.Errorf("%s: short row", path)
		}
		key := csvUserKey(rec[hashCol], rec[saltCol])
		if _, ok := out[key]; ok {
			return nil, fmt.Errorf("%s: duplicate password hash", path)
		}
		out[key] = csvUser{name: rec[nameCol], phone: rec[phoneCol]}
	}
	return out, nil
}

func csvCol(header []string, name string) (int, error) {
	for i, h := range header {
		h = strings.TrimPrefix(strings.TrimSpace(h), "\uFEFF")
		if h == name {
			return i, nil
		}
	}
	return 0, fmt.Errorf("missing %s column", name)
}
