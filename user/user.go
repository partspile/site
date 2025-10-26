package user

import (
	"fmt"
	"time"

	"github.com/parts-pile/site/db"
)

// Table name constants
const (
	TableUser = "User"
)

// Notification method constants
const (
	NotificationMethodSMS    = "sms"
	NotificationMethodEmail  = "email"
	NotificationMethodSignal = "signal"
)

// UserStatus represents the status of a user
type UserStatus string

const (
	StatusActive   UserStatus = "active"
	StatusArchived UserStatus = "archived"
)

type User struct {
	ID                 int
	Name               string
	Phone              string
	PasswordHash       string
	PasswordSalt       string
	PasswordAlgo       string
	PhoneVerified      bool
	VerificationCode   *string
	NotificationMethod string
	EmailAddress       *string
	CreatedAt          time.Time
	IsAdmin            bool
	DeletedAt          *time.Time
}

// IsArchived returns true if the user has been archived
func (u User) IsArchived() bool {
	return u.DeletedAt != nil
}

// CreateUser inserts a new user into the database and initializes their rock inventory
func CreateUser(name, phone, passwordHash, passwordSalt, passwordAlgo string) (int, error) {
	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Create user
	res, err := tx.Exec(`INSERT INTO User (name, phone, password_hash, password_salt, password_algo, phone_verified, notification_method) VALUES (?, ?, ?, ?, ?, 0, ?)`, name, phone, passwordHash, passwordSalt, passwordAlgo, NotificationMethodSMS)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()

	// Initialize rock inventory
	_, err = tx.Exec(`INSERT INTO UserRock (user_id, rock_count) VALUES (?, 3)`, id)
	if err != nil {
		return 0, err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return int(id), nil
}

// GetUserByID retrieves a user by ID from either active or archived tables
// Returns the user, its status, and whether it was found
func GetUserByID(id int) (User, UserStatus, bool) {
	// Try to get user (includes both active and archived)
	user, err := GetUser(id)
	if err == nil {
		if user.IsArchived() {
			return user, StatusArchived, true
		}
		return user, StatusActive, true
	}

	return User{}, StatusActive, false
}

// GetUserByPhone retrieves a user by phone number
func GetUserByPhone(phone string) (User, error) {
	row := db.QueryRow(`SELECT id, name, phone, password_hash,
		password_salt, password_algo, phone_verified, verification_code,
		notification_method, email_address, created_at, is_admin, deleted_at
		FROM User WHERE phone = ? AND deleted_at IS NULL`, phone)
	var u User
	var createdAt time.Time
	var isAdmin int
	var phoneVerified int
	var verificationCode *string
	var notificationMethod string
	var emailAddress *string
	var deletedAt *time.Time
	err := row.Scan(&u.ID, &u.Name, &u.Phone, &u.PasswordHash, &u.PasswordSalt, &u.PasswordAlgo, &phoneVerified, &verificationCode, &notificationMethod, &emailAddress, &createdAt, &isAdmin, &deletedAt)
	if err != nil {
		return User{}, err
	}
	u.CreatedAt = createdAt
	u.IsAdmin = isAdmin == 1
	u.PhoneVerified = phoneVerified == 1
	u.VerificationCode = verificationCode
	u.NotificationMethod = notificationMethod
	u.EmailAddress = emailAddress
	u.DeletedAt = deletedAt
	return u, nil
}

// GetUser retrieves a user by ID (includes both active and archived users)
func GetUser(id int) (User, error) {
	row := db.QueryRow(`SELECT id, name, phone, password_hash, password_salt, password_algo, phone_verified, verification_code, notification_method, email_address, created_at, is_admin, deleted_at FROM User WHERE id = ?`, id)
	var u User
	var createdAt time.Time
	var isAdmin int
	var phoneVerified int
	var verificationCode *string
	var notificationMethod string
	var emailAddress *string
	var deletedAt *time.Time
	err := row.Scan(&u.ID, &u.Name, &u.Phone, &u.PasswordHash, &u.PasswordSalt, &u.PasswordAlgo, &phoneVerified, &verificationCode, &notificationMethod, &emailAddress, &createdAt, &isAdmin, &deletedAt)
	if err != nil {
		return User{}, err
	}
	u.CreatedAt = createdAt
	u.IsAdmin = isAdmin == 1
	u.PhoneVerified = phoneVerified == 1
	u.VerificationCode = verificationCode
	u.NotificationMethod = notificationMethod
	u.EmailAddress = emailAddress
	u.DeletedAt = deletedAt

	return u, nil
}

// GetUserByName retrieves a user by name (username)
func GetUserByName(name string) (User, error) {
	row := db.QueryRow(`SELECT id, name, phone, password_hash,
		password_salt, password_algo, phone_verified, verification_code,
		notification_method, email_address, created_at, is_admin, deleted_at
		FROM User WHERE name = ? AND deleted_at IS NULL`, name)
	var u User
	var createdAt time.Time
	var isAdmin int
	var phoneVerified int
	var verificationCode *string
	var notificationMethod string
	var emailAddress *string
	var deletedAt *time.Time
	err := row.Scan(&u.ID, &u.Name, &u.Phone, &u.PasswordHash, &u.PasswordSalt, &u.PasswordAlgo, &phoneVerified, &verificationCode, &notificationMethod, &emailAddress, &createdAt, &isAdmin, &deletedAt)
	if err != nil {
		return User{}, err
	}
	u.CreatedAt = createdAt
	u.IsAdmin = isAdmin == 1
	u.PhoneVerified = phoneVerified == 1
	u.VerificationCode = verificationCode
	u.NotificationMethod = notificationMethod
	u.EmailAddress = emailAddress
	u.DeletedAt = deletedAt

	return u, nil
}

// UpdateUserPassword updates a user's password hash, salt, and algo
func UpdateUserPassword(userID int, newHash, newSalt, newAlgo string) (int, error) {
	res, err := db.Exec(`UPDATE User SET password_hash = ?, password_salt = ?, password_algo = ? WHERE id = ?`, newHash, newSalt, newAlgo, userID)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}

// MarkPhoneVerified marks a user's phone number as verified
func MarkPhoneVerified(userID int) error {
	_, err := db.Exec(`UPDATE User SET phone_verified = 1, verification_code = NULL WHERE id = ?`, userID)
	if err != nil {
		return fmt.Errorf("failed to mark phone verified: %w", err)
	}
	return nil
}

// UpdateNotificationMethod updates the user's notification method preference
func UpdateNotificationMethod(userID int, method string) error {
	_, err := db.Exec(`UPDATE User SET notification_method = ? WHERE id = ?`, method, userID)
	return err
}

// UpdateNotificationPreferences updates both notification method and email address
func UpdateNotificationPreferences(userID int, method string, emailAddress *string) error {
	_, err := db.Exec(`UPDATE User SET notification_method = ?, email_address = ? WHERE id = ?`, method, emailAddress, userID)
	return err
}

// ArchiveUser archives a user using soft delete
func ArchiveUser(id int) error {
	_, err := db.Exec("UPDATE User SET deleted_at = ? WHERE id = ?",
		time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}

// RestoreUser restores an archived user by clearing the deleted_at field
func RestoreUser(userID int) error {
	_, err := db.Exec("UPDATE User SET deleted_at = NULL WHERE id = ?", userID)
	return err
}
