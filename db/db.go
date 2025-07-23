package db

import (
	"database/sql"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db   *sql.DB
	once sync.Once
)

// Init initializes the database connection
func Init(databaseURL string) error {
	var err error
	once.Do(func() {
		db, err = sql.Open("sqlite3", databaseURL)
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

// Get returns the database connection
func Get() *sql.DB {
	if db == nil {
		panic("Database not initialized. Call db.Init() first.")
	}
	return db
}

// SetForTesting sets the database connection for testing
func SetForTesting(database *sql.DB) {
	db = database
}

// Close closes the database connection
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// Convenience methods that wrap common database operations

// Query executes a query that returns rows
func Query(query string, args ...interface{}) (*sql.Rows, error) {
	return Get().Query(query, args...)
}

// QueryRow executes a query that returns a single row
func QueryRow(query string, args ...interface{}) *sql.Row {
	return Get().QueryRow(query, args...)
}

// Exec executes a query that doesn't return rows
func Exec(query string, args ...interface{}) (sql.Result, error) {
	return Get().Exec(query, args...)
}

// Begin starts a new transaction
func Begin() (*sql.Tx, error) {
	return Get().Begin()
}
