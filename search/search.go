package search

import (
	"time"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
)

// UserSearch represents a user's search query
type UserSearch struct {
	ID          int       `db:"id"`
	UserID      int       `db:"user_id"`
	QueryString string    `db:"query_string"`
	CreatedAt   time.Time `db:"created_at"`
}

// TopSearch represents a popular search query with its count
type TopSearch struct {
	QueryString string `db:"query_string"`
	Count       int    `db:"count"`
}

// SaveUserSearch saves a user's search query to the database
func SaveUserSearch(userID int, queryString string) error {
	_, err := db.Exec("INSERT INTO UserSearch (user_id, query_string) VALUES ($1, $2)", userID, queryString)
	if err != nil {
		logger.Error("Error saving user search",
			"error", err, "component", "search", "userID", userID)
		return err
	}
	return nil
}

// GetRecentUserSearches returns a list of recent search queries for a user
func GetRecentUserSearches(userID int, limit int) ([]UserSearch, error) {
	query := "SELECT id, user_id, query_string, created_at FROM UserSearch WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2"
	var searches []UserSearch
	err := db.Select(&searches, query, userID, limit)
	return searches, err
}

// DeleteUserSearch deletes a specific user search entry
func DeleteUserSearch(searchID int, userID int) error {
	_, err := db.Exec("DELETE FROM UserSearch WHERE id = $1 AND user_id = $2", searchID, userID)
	if err != nil {
		logger.Error("Error deleting user search",
			"error", err, "component", "search", "searchID", searchID, "userID", userID)
		return err
	}
	return nil
}

// DeleteAllUserSearches deletes all search entries for a user
func DeleteAllUserSearches(userID int) error {
	_, err := db.Exec("DELETE FROM UserSearch WHERE user_id = $1", userID)
	if err != nil {
		logger.Error("Error deleting all user searches",
			"error", err, "component", "search", "userID", userID)
		return err
	}
	return nil
}

// GetTopSearches returns the most frequent search queries across all users
func GetTopSearches(limit int) ([]TopSearch, error) {
	query := "SELECT query_string, COUNT(*) as count FROM UserSearch GROUP BY query_string ORDER BY count DESC LIMIT $1"
	var topSearches []TopSearch
	err := db.Select(&topSearches, query, limit)
	return topSearches, err
}
