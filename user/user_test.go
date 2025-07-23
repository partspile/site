package user

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/parts-pile/site/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_IsArchived(t *testing.T) {
	tests := []struct {
		name     string
		user     User
		expected bool
	}{
		{
			name: "active user",
			user: User{
				ID:        1,
				Name:      "Test User",
				DeletedAt: nil,
			},
			expected: false,
		},
		{
			name: "archived user",
			user: User{
				ID:        1,
				Name:      "Test User",
				DeletedAt: &time.Time{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.user.IsArchived()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateUser(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db.SetForTesting(mockDB)

	mock.ExpectExec("INSERT INTO User").
		WithArgs("testuser", "1234567890", "hashedpassword").
		WillReturnResult(sqlmock.NewResult(1, 1))

	userID, err := CreateUser("testuser", "1234567890", "hashedpassword")

	assert.NoError(t, err)
	assert.Equal(t, 1, userID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserByID(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db.SetForTesting(mockDB)

	// Test active user
	mock.ExpectQuery("SELECT.*FROM User WHERE id = ?").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "phone", "token_balance", "password_hash", "created_at", "is_admin"}).
			AddRow(1, "testuser", "1234567890", 0.0, "hashedpassword", time.Now().Format(time.RFC3339Nano), 0))

	user, status, found := GetUserByID(1)

	assert.True(t, found)
	assert.Equal(t, StatusActive, status)
	assert.Equal(t, 1, user.ID)
	assert.Equal(t, "testuser", user.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserByPhone(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db.SetForTesting(mockDB)

	expectedTime := time.Now()
	mock.ExpectQuery("SELECT.*FROM User WHERE phone = ?").
		WithArgs("1234567890").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "phone", "token_balance", "password_hash", "created_at", "is_admin"}).
			AddRow(1, "testuser", "1234567890", 0.0, "hashedpassword", expectedTime.Format(time.RFC3339Nano), 0))

	user, err := GetUserByPhone("1234567890")

	assert.NoError(t, err)
	assert.Equal(t, 1, user.ID)
	assert.Equal(t, "testuser", user.Name)
	assert.Equal(t, "1234567890", user.Phone)
	assert.False(t, user.IsAdmin)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUser(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db.SetForTesting(mockDB)

	expectedTime := time.Now()
	mock.ExpectQuery("SELECT.*FROM User WHERE id = ?").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "phone", "token_balance", "password_hash", "created_at", "is_admin"}).
			AddRow(1, "testuser", "1234567890", 0.0, "hashedpassword", expectedTime.Format(time.RFC3339Nano), 1))

	user, err := GetUser(1)

	assert.NoError(t, err)
	assert.Equal(t, 1, user.ID)
	assert.Equal(t, "testuser", user.Name)
	assert.True(t, user.IsAdmin)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserByName(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db.SetForTesting(mockDB)

	expectedTime := time.Now()
	mock.ExpectQuery("SELECT.*FROM User WHERE name = ?").
		WithArgs("testuser").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "phone", "token_balance", "password_hash", "created_at", "is_admin"}).
			AddRow(1, "testuser", "1234567890", 0.0, "hashedpassword", expectedTime.Format(time.RFC3339Nano), 0))

	user, err := GetUserByName("testuser")

	assert.NoError(t, err)
	assert.Equal(t, 1, user.ID)
	assert.Equal(t, "testuser", user.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateUserPassword(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db.SetForTesting(mockDB)

	mock.ExpectExec("UPDATE User SET password_hash = \\? WHERE id = \\?").
		WithArgs("newhashedpassword", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	rowsAffected, err := UpdateUserPassword(1, "newhashedpassword")

	assert.NoError(t, err)
	assert.Equal(t, 1, rowsAffected)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestArchiveUser(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db.SetForTesting(mockDB)

	// Mock the transaction
	mock.ExpectBegin()

	// Mock getting user data (6 columns, no is_admin)
	mock.ExpectQuery("SELECT id, name, phone, token_balance, password_hash, created_at FROM User WHERE id = \\?").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "phone", "token_balance", "password_hash", "created_at"}).
			AddRow(1, "testuser", "1234567890", 0.0, "hashedpassword", "2023-01-01T00:00:00Z"))

	// Mock inserting into ArchivedUser (7 columns including deletion_date)
	mock.ExpectExec("INSERT INTO ArchivedUser").
		WithArgs(1, "testuser", "1234567890", 0.0, "hashedpassword", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock archiving ads
	mock.ExpectExec("INSERT INTO ArchivedAd").
		WithArgs(sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock archiving ad-car relationships
	mock.ExpectExec("INSERT INTO ArchivedAdCar").
		WithArgs(sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock updating token transactions
	mock.ExpectExec("UPDATE TokenTransaction").
		WithArgs(1, 1).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock deleting ad-car relationships
	mock.ExpectExec("DELETE FROM AdCar").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock deleting ads
	mock.ExpectExec("DELETE FROM Ad WHERE user_id = \\?").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock deleting user
	mock.ExpectExec("DELETE FROM User WHERE id = \\?").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	err = ArchiveUser(1)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAllUsers(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db.SetForTesting(mockDB)

	expectedTime := time.Now()
	mock.ExpectQuery("SELECT id, name, phone, token_balance, password_hash, created_at, is_admin FROM User").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "phone", "token_balance", "password_hash", "created_at", "is_admin"}).
			AddRow(1, "user1", "1234567890", 0.0, "hash1", expectedTime.Format(time.RFC3339Nano), 0).
			AddRow(2, "user2", "0987654321", 0.0, "hash2", expectedTime.Format(time.RFC3339Nano), 1))

	users, err := GetAllUsers()

	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, 1, users[0].ID)
	assert.Equal(t, "user1", users[0].Name)
	assert.Equal(t, 2, users[1].ID)
	assert.Equal(t, "user2", users[1].Name)
	assert.True(t, users[1].IsAdmin)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSetAdmin(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db.SetForTesting(mockDB)

	mock.ExpectExec("UPDATE User SET is_admin = \\? WHERE id = \\?").
		WithArgs(1, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = SetAdmin(1, true)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
