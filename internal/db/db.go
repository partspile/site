package db

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

var (
	db          *sqlx.DB
	once        sync.Once
	databaseURL string
	hashPepper  []byte
)

func Init(url string) error {
	var err error
	once.Do(func() {
		databaseURL = url
		db, err = sqlx.Open("pgx", url)
		if err != nil {
			return
		}

		if err = db.Ping(); err != nil {
			return
		}
	})
	return err
}

// ConnectionTarget returns host and database name from a PostgreSQL URL (no credentials).
func ConnectionTarget(databaseURL string) (host, database string) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", ""
	}
	host = u.Host
	database = strings.TrimPrefix(u.Path, "/")
	if i := strings.IndexByte(database, '?'); i >= 0 {
		database = database[:i]
	}
	return host, database
}

// ResetForTest closes the database and allows Init to run again. Test use only.
func ResetForTest() {
	if db != nil {
		_ = db.Close()
	}
	db = nil
	databaseURL = ""
	once = sync.Once{}
}

// SetHashPepper configures the HMAC key for HashString. Required before
// hashing name/phone lookup values.
func SetHashPepper(pepper []byte) {
	hashPepper = pepper
}

func Query(query string, args ...any) (*sql.Rows, error) {
	return db.Query(query, args...)
}

func QueryRow(query string, args ...any) *sql.Row {
	return db.QueryRow(query, args...)
}

func Exec(query string, args ...any) (sql.Result, error) {
	return db.Exec(query, args...)
}

func Begin() (*sql.Tx, error) {
	return db.Begin()
}

func Select(dest any, query string, args ...any) error {
	return db.Select(dest, query, args...)
}

func Ping() error {
	return db.Ping()
}

// CheckSchema reports whether the schema has been applied (Admin TUI init, etc.).
func CheckSchema() error {
	var exists bool
	err := QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'categories'
		)`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("checking database schema: %w", err)
	}
	if !exists {
		return fmt.Errorf("database not initialized — restore a backup via Admin TUI")
	}
	return nil
}

func QueryJSON(dst any, query string, args ...any) error {
	var jsonResult []byte
	err := db.QueryRow(query, args...).Scan(&jsonResult)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonResult, dst)
}

func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// HashString returns a peppered HMAC-SHA256 hex digest for DB lookups
// (name_hash / phone_hash). Without the pepper, E.164 / username
// dictionaries cannot be matched offline against a DB dump alone.
func HashString(value string) string {
	if len(hashPepper) == 0 {
		panic("db: hash pepper not configured")
	}
	mac := hmac.New(sha256.New, hashPepper)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}
