package search

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveUserSearch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	InitDB(db)

	userID := sql.NullInt64{Int64: 1, Valid: true}
	queryString := "test search"

	mock.ExpectExec("INSERT INTO UserSearch \\(user_id, query_string\\) VALUES \\(\\?, \\?\\)").
		WithArgs(userID, queryString).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = SaveUserSearch(userID, queryString)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveUserSearch_Anonymous(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	InitDB(db)

	userID := sql.NullInt64{Valid: false}
	queryString := "anonymous search"

	mock.ExpectExec("INSERT INTO UserSearch \\(user_id, query_string\\) VALUES \\(\\?, \\?\\)").
		WithArgs(userID, queryString).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = SaveUserSearch(userID, queryString)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRecentUserSearches(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	InitDB(db)

	userID := 1
	limit := 5

	expectedSearches := []UserSearch{
		{ID: 1, UserID: sql.NullInt64{Int64: 1, Valid: true}, QueryString: "search 1", CreatedAt: time.Now()},
		{ID: 2, UserID: sql.NullInt64{Int64: 1, Valid: true}, QueryString: "search 2", CreatedAt: time.Now()},
	}

	rows := sqlmock.NewRows([]string{"id", "user_id", "query_string", "created_at"})
	for _, search := range expectedSearches {
		rows.AddRow(search.ID, search.UserID, search.QueryString, search.CreatedAt.Format(time.RFC3339Nano))
	}

	mock.ExpectQuery("SELECT id, user_id, query_string, created_at FROM UserSearch WHERE user_id = \\? ORDER BY created_at DESC LIMIT \\?").
		WithArgs(userID, limit).
		WillReturnRows(rows)

	searches, err := GetRecentUserSearches(userID, limit)

	assert.NoError(t, err)
	assert.Len(t, searches, 2)
	assert.Equal(t, expectedSearches[0].QueryString, searches[0].QueryString)
	assert.Equal(t, expectedSearches[1].QueryString, searches[1].QueryString)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRecentUserSearches_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	InitDB(db)

	userID := 1
	limit := 5

	mock.ExpectQuery("SELECT id, user_id, query_string, created_at FROM UserSearch WHERE user_id = \\? ORDER BY created_at DESC LIMIT \\?").
		WithArgs(userID, limit).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "query_string", "created_at"}))

	searches, err := GetRecentUserSearches(userID, limit)

	assert.NoError(t, err)
	assert.Len(t, searches, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteUserSearch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	InitDB(db)

	searchID := 1
	userID := 1

	mock.ExpectExec("DELETE FROM UserSearch WHERE id = \\? AND user_id = \\?").
		WithArgs(searchID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = DeleteUserSearch(searchID, userID)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteAllUserSearches(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	InitDB(db)

	userID := 1

	mock.ExpectExec("DELETE FROM UserSearch WHERE user_id = \\?").
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 3))

	err = DeleteAllUserSearches(userID)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTopSearches(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	InitDB(db)

	limit := 10

	expectedSearches := []struct {
		QueryString string
		Count       int
	}{
		{QueryString: "engine", Count: 15},
		{QueryString: "brake", Count: 12},
		{QueryString: "tire", Count: 8},
	}

	rows := sqlmock.NewRows([]string{"query_string", "count"})
	for _, search := range expectedSearches {
		rows.AddRow(search.QueryString, search.Count)
	}

	mock.ExpectQuery("SELECT query_string, COUNT\\(\\*\\) as count FROM UserSearch GROUP BY query_string ORDER BY count DESC LIMIT \\?").
		WithArgs(limit).
		WillReturnRows(rows)

	searches, err := GetTopSearches(limit)

	assert.NoError(t, err)
	assert.Len(t, searches, 3)
	assert.Equal(t, expectedSearches[0].QueryString, searches[0].QueryString)
	assert.Equal(t, expectedSearches[0].Count, searches[0].Count)
	assert.Equal(t, expectedSearches[1].QueryString, searches[1].QueryString)
	assert.Equal(t, expectedSearches[1].Count, searches[1].Count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTopSearches_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	InitDB(db)

	limit := 10

	mock.ExpectQuery("SELECT query_string, COUNT\\(\\*\\) as count FROM UserSearch GROUP BY query_string ORDER BY count DESC LIMIT \\?").
		WithArgs(limit).
		WillReturnRows(sqlmock.NewRows([]string{"query_string", "count"}))

	searches, err := GetTopSearches(limit)

	assert.NoError(t, err)
	assert.Len(t, searches, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}
