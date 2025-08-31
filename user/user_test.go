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

	// Expect transaction begin
	mock.ExpectBegin()
	
	// Expect user insert
	mock.ExpectExec("INSERT INTO User").
		WithArgs("testuser", "1234567890", "hashedpassword", "somesalt", "argon2id", "sms").
		WillReturnResult(sqlmock.NewResult(1, 1))
	
	// Expect rock inventory insert
	mock.ExpectExec("INSERT INTO UserRock").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	
	// Expect transaction commit
	mock.ExpectCommit()

	userID, err := CreateUser("testuser", "1234567890", "hashedpassword", "somesalt", "argon2id")

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
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "phone", "password_hash", "password_salt", "password_algo", "phone_verified", "verification_code", "notification_method", "email_address", "created_at", "is_admin", "deleted_at"}).
			AddRow(1, "testuser", "1234567890", "hashedpassword", "salt", "argon2id", 0, nil, "sms", nil, time.Now().Format(time.RFC3339Nano), 0, nil))

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
	mock.ExpectQuery("SELECT.*FROM User WHERE phone = \\? AND deleted_at IS NULL").
		WithArgs("1234567890").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "phone", "password_hash", "password_salt", "password_algo", "phone_verified", "verification_code", "notification_method", "email_address", "created_at", "is_admin", "deleted_at"}).
			AddRow(1, "testuser", "1234567890", "hashedpassword", "salt", "argon2id", 0, nil, "sms", nil, expectedTime.Format(time.RFC3339Nano), 0, nil))

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
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "phone", "password_hash", "password_salt", "password_algo", "phone_verified", "verification_code", "notification_method", "email_address", "created_at", "is_admin", "deleted_at"}).
			AddRow(1, "testuser", "1234567890", "hashedpassword", "salt", "argon2id", 0, nil, "sms", nil, expectedTime.Format(time.RFC3339Nano), 1, nil))

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
	mock.ExpectQuery("SELECT.*FROM User WHERE name = \\? AND deleted_at IS NULL").
		WithArgs("testuser").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "phone", "password_hash", "password_salt", "password_algo", "phone_verified", "verification_code", "notification_method", "email_address", "created_at", "is_admin", "deleted_at"}).
			AddRow(1, "testuser", "1234567890", "hashedpassword", "salt", "argon2id", 0, nil, "sms", nil, expectedTime.Format(time.RFC3339Nano), 0, nil))

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

	mock.ExpectExec("UPDATE User SET password_hash = \\?, password_salt = \\?, password_algo = \\? WHERE id = \\?").
		WithArgs("newhashedpassword", "newsalt", "argon2id", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	rowsAffected, err := UpdateUserPassword(1, "newhashedpassword", "newsalt", "argon2id")

	assert.NoError(t, err)
	assert.Equal(t, 1, rowsAffected)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateNotificationMethod(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db.SetForTesting(mockDB)

	mock.ExpectExec("UPDATE User SET notification_method = \\? WHERE id = \\?").
		WithArgs("email", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = UpdateNotificationMethod(1, "email")

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestArchiveUser(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db.SetForTesting(mockDB)

	mock.ExpectExec("UPDATE User SET deleted_at = \\? WHERE id = \\?").
		WithArgs(sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = ArchiveUser(1)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRestoreUser(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db.SetForTesting(mockDB)

	mock.ExpectExec("UPDATE User SET deleted_at = NULL WHERE id = \\?").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = RestoreUser(1)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
