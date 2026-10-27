package testdb

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/partspile/site/internal/db"
)

// DatabaseURL returns the test database connection string.
func DatabaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return "postgres://postgres:postgres@localhost:5432/partspile?sslmode=disable"
}

// PackageDatabaseURL returns a DATABASE_URL with an isolated database name
// for the given package suffix (e.g. partspile -> partspile_backup_db).
func PackageDatabaseURL(suffix string) string {
	base := DatabaseURL()
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	if i := strings.IndexByte(dbName, '?'); i >= 0 {
		dbName = dbName[:i]
	}
	if dbName == "" {
		dbName = "partspile"
	}
	u.Path = "/" + dbName + "_" + suffix
	return u.String()
}

// EnsureDatabase creates databaseURL's database if it does not exist.
func EnsureDatabase(databaseURL string) error {
	_, dbName := db.ConnectionTarget(databaseURL)
	if !validDatabaseName(dbName) {
		return fmt.Errorf("invalid database name %q", dbName)
	}

	u, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("parse database URL: %w", err)
	}
	u.Path = "/postgres"

	conn, err := pgx.Connect(context.Background(), u.String())
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer conn.Close(context.Background())

	var exists bool
	err = conn.QueryRow(context.Background(),
		`SELECT true FROM pg_database WHERE datname = $1`, dbName,
	).Scan(&exists)
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("lookup database %q: %w", dbName, err)
	}

	_, err = conn.Exec(context.Background(),
		fmt.Sprintf(`CREATE DATABASE "%s" TEMPLATE template0`, dbName),
	)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("create database %q: %w", dbName, err)
	}
	return nil
}

func validDatabaseName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

// InitSchema drops all tables and applies schema.sql.
func InitSchema() error {
	url := DatabaseURL()
	db.ResetForTest()
	if err := db.Init(url); err != nil {
		return fmt.Errorf("init db: %w", err)
	}
	return db.ResetSchema()
}
