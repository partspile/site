package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsersCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.csv")
	body := "id,name,phone,password_hash,password_salt,password_algo,deleted\n" +
		"1,alice,+15555550100," +
		`"$argon2id$v=19$m=65536,t=3,p=2$abc$def"` +
		",salt,argon2id,false\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := loadUsersCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	key := csvUserKey(
		"$argon2id$v=19$m=65536,t=3,p=2$abc$def", "salt")
	u, ok := got[key]
	if !ok {
		t.Fatalf("missing row, got %#v", got)
	}
	if u.name != "alice" || u.phone != "+15555550100" {
		t.Fatalf("row = %#v", u)
	}
}
