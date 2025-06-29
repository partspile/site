package search

import (
	"database/sql"
	"log"
	"time"

	)

var db *sql.DB

// Exported for use by other packages
var DB *sql.DB

func InitDB(database *sql.DB) {
	db = database
	DB = database
}

// UserSearch represents a user's search query
type UserSearch struct {
	ID         int            `json:"id"`
	UserID     sql.NullInt64  `json:"user_id"`
	QueryString string        `json:"query_string"`
	CreatedAt  time.Time      `json:"created_at"`
}

// SaveUserSearch saves a user's search query to the database
func SaveUserSearch(userID sql.NullInt64, queryString string) error {
	_, err := db.Exec("INSERT INTO UserSearch (user_id, query_string) VALUES (?, ?)", userID, queryString)
	if err != nil {
		log.Printf("Error saving user search: %v", err)
		return err
	}
	return nil
}

// GetRecentUserSearches returns a list of recent search queries for a user
func GetRecentUserSearches(userID int, limit int) ([]UserSearch, error) {
	rows, err := db.Query("SELECT id, user_id, query_string, created_at FROM UserSearch WHERE user_id = ? ORDER BY created_at DESC LIMIT ?", userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var searches []UserSearch
	for rows.Next() {
		var s UserSearch
		var userID sql.NullInt64
		if err := rows.Scan(&s.ID, &userID, &s.QueryString, &s.CreatedAt); err != nil {
			log.Printf("Error scanning user search: %v", err)
			continue
		}
		s.UserID = userID
		searches = append(searches, s)
	}
	return searches, nil
}

// DeleteUserSearch deletes a specific user search entry
func DeleteUserSearch(searchID int, userID int) error {
	_, err := db.Exec("DELETE FROM UserSearch WHERE id = ? AND user_id = ?", searchID, userID)
	if err != nil {
		log.Printf("Error deleting user search: %v", err)
		return err
	}
	return nil
}

// DeleteAllUserSearches deletes all search entries for a user
func DeleteAllUserSearches(userID int) error {
	_, err := db.Exec("DELETE FROM UserSearch WHERE user_id = ?", userID)
	if err != nil {
		log.Printf("Error deleting all user searches: %v", err)
		return err
	}
	return nil
}

// GetTopSearches returns the most frequent search queries across all users
func GetTopSearches(limit int) ([]struct { QueryString string; Count int }, error) {
	rows, err := db.Query("SELECT query_string, COUNT(*) as count FROM UserSearch GROUP BY query_string ORDER BY count DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topSearches []struct { QueryString string; Count int }
	for rows.Next() {
		var s struct { QueryString string; Count int }
		if err := rows.Scan(&s.QueryString, &s.Count); err != nil {
			log.Printf("Error scanning top search: %v", err)
			continue
		}
		topSearches = append(topSearches, s)
	}
	return topSearches, nil
}
