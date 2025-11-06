package db

import (
	"database/sql"
	"log"
	"sync"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var (
	db   *sqlx.DB
	once sync.Once
)

// Init initializes the database connection
func Init(databaseURL string) error {
	var err error
	once.Do(func() {
		db, err = sqlx.Open("sqlite3", databaseURL)
		if err != nil {
			log.Printf("Failed to open database: %v", err)
			return
		}

		// Test the connection
		if err = db.Ping(); err != nil {
			log.Printf("Failed to ping database: %v", err)
			return
		}

		log.Printf("Database initialized successfully: %s", databaseURL)
	})
	return err
}

// SetForTesting sets the database connection for testing
func SetForTesting(database *sqlx.DB) {
	db = database
}

// Query executes a query that returns rows
func Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.Query(query, args...)
}

// QueryRow executes a query that returns a single row
func QueryRow(query string, args ...interface{}) *sql.Row {
	return db.QueryRow(query, args...)
}

// Exec executes a query that doesn't return rows
func Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.Exec(query, args...)
}

// Begin starts a new transaction
func Begin() (*sql.Tx, error) {
	return db.Begin()
}

// Select executes a query and scans the results into dest (slice)
func Select(dest interface{}, query string, args ...interface{}) error {
	return db.Select(dest, query, args...)
}

// Get executes a query and scans a single row into dest
func Get(dest interface{}, query string, args ...interface{}) error {
	return db.Get(dest, query, args...)
}

// Ping verifies the database connection is alive
func Ping() error {
	return db.Ping()
}
