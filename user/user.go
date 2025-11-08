package user

import (
	"database/sql"
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
	SMSOptedOut        bool
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

	// Compute hashes for lookups
	nameHash := HashForLookup(name)
	phoneHash := HashForLookup(phone)
	
	// Encrypt with placeholder ID (0) - we'll re-encrypt with real ID after insert
	encryptedName, nameNonce, err := EncryptName(0, name)
	if err != nil {
		return 0, err
	}
	encryptedPhone, phoneNonce, err := EncryptPhone(0, phone)
	if err != nil {
		return 0, err
	}

	// Insert user with encrypted data
	res, err := tx.Exec(`INSERT INTO User (encrypted_name, name_nonce, name_hash, encrypted_phone, phone_nonce, phone_hash, password_hash, password_salt, password_algo, phone_verified, notification_method) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?)`, 
		encryptedName, nameNonce, nameHash, encryptedPhone, phoneNonce, phoneHash, passwordHash, passwordSalt, passwordAlgo, NotificationMethodSMS)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	userID := int(id)

	// Re-encrypt with the real user ID
	encryptedName, nameNonce, err = EncryptName(userID, name)
	if err != nil {
		return 0, err
	}
	encryptedPhone, phoneNonce, err = EncryptPhone(userID, phone)
	if err != nil {
		return 0, err
	}
	
	// Update with properly encrypted data
	_, err = tx.Exec(`UPDATE User SET encrypted_name = ?, name_nonce = ?, encrypted_phone = ?, phone_nonce = ? WHERE id = ?`,
		encryptedName, nameNonce, encryptedPhone, phoneNonce, userID)
	if err != nil {
		return 0, err
	}

	// Initialize rock inventory
	_, err = tx.Exec(`INSERT INTO UserRock (user_id, rock_count) VALUES (?, 3)`, userID)
	if err != nil {
		return 0, err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return userID, nil
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

// GetUserByPhone retrieves a user by phone number (uses hash for lookup)
func GetUserByPhone(phone string) (User, error) {
	phoneHash := HashForLookup(phone)
	row := db.QueryRow(`SELECT id, encrypted_name, name_nonce, encrypted_phone, phone_nonce, password_hash,
		password_salt, password_algo, phone_verified, verification_code,
		notification_method, encrypted_email_address, email_address_nonce, created_at, is_admin, sms_opted_out, deleted_at
		FROM User WHERE phone_hash = ? AND deleted_at IS NULL`, phoneHash)
	var u User
	var createdAt time.Time
	var isAdmin int
	var phoneVerified int
	var verificationCode *string
	var notificationMethod string
	var encryptedName, nameNonce, encryptedPhone, phoneNonce string
	var encryptedEmail, emailNonce sql.NullString
	var smsOptedOut int
	var deletedAt *time.Time
	err := row.Scan(&u.ID, &encryptedName, &nameNonce, &encryptedPhone, &phoneNonce, &u.PasswordHash, &u.PasswordSalt, &u.PasswordAlgo, &phoneVerified, &verificationCode, &notificationMethod, &encryptedEmail, &emailNonce, &createdAt, &isAdmin, &smsOptedOut, &deletedAt)
	if err != nil {
		return User{}, err
	}
	
	// Decrypt name and phone
	u.Name, err = DecryptName(u.ID, encryptedName, nameNonce)
	if err != nil {
		return User{}, fmt.Errorf("failed to decrypt name: %w", err)
	}
	u.Phone, err = DecryptPhone(u.ID, encryptedPhone, phoneNonce)
	if err != nil {
		return User{}, fmt.Errorf("failed to decrypt phone: %w", err)
	}
	
	// Decrypt email if present
	if encryptedEmail.Valid && emailNonce.Valid {
		email, err := DecryptEmailAddress(u.ID, encryptedEmail.String, emailNonce.String)
		if err == nil {
			u.EmailAddress = &email
		}
	}
	
	u.CreatedAt = createdAt
	u.IsAdmin = isAdmin == 1
	u.PhoneVerified = phoneVerified == 1
	u.VerificationCode = verificationCode
	u.NotificationMethod = notificationMethod
	u.SMSOptedOut = smsOptedOut == 1
	u.DeletedAt = deletedAt
	return u, nil
}

// GetUser retrieves a user by ID (includes both active and archived users)
func GetUser(id int) (User, error) {
	row := db.QueryRow(`SELECT id, encrypted_name, name_nonce, encrypted_phone, phone_nonce, password_hash, password_salt, password_algo, phone_verified, verification_code, notification_method, encrypted_email_address, email_address_nonce, created_at, is_admin, sms_opted_out, deleted_at FROM User WHERE id = ?`, id)
	var u User
	var createdAt time.Time
	var isAdmin int
	var phoneVerified int
	var verificationCode *string
	var notificationMethod string
	var encryptedName, nameNonce, encryptedPhone, phoneNonce string
	var encryptedEmail, emailNonce sql.NullString
	var smsOptedOut int
	var deletedAt *time.Time
	err := row.Scan(&u.ID, &encryptedName, &nameNonce, &encryptedPhone, &phoneNonce, &u.PasswordHash, &u.PasswordSalt, &u.PasswordAlgo, &phoneVerified, &verificationCode, &notificationMethod, &encryptedEmail, &emailNonce, &createdAt, &isAdmin, &smsOptedOut, &deletedAt)
	if err != nil {
		return User{}, err
	}
	
	// Decrypt name and phone
	u.Name, err = DecryptName(u.ID, encryptedName, nameNonce)
	if err != nil {
		return User{}, fmt.Errorf("failed to decrypt name: %w", err)
	}
	u.Phone, err = DecryptPhone(u.ID, encryptedPhone, phoneNonce)
	if err != nil {
		return User{}, fmt.Errorf("failed to decrypt phone: %w", err)
	}
	
	// Decrypt email if present
	if encryptedEmail.Valid && emailNonce.Valid {
		email, err := DecryptEmailAddress(u.ID, encryptedEmail.String, emailNonce.String)
		if err == nil {
			u.EmailAddress = &email
		}
	}
	
	u.CreatedAt = createdAt
	u.IsAdmin = isAdmin == 1
	u.PhoneVerified = phoneVerified == 1
	u.VerificationCode = verificationCode
	u.NotificationMethod = notificationMethod
	u.SMSOptedOut = smsOptedOut == 1
	u.DeletedAt = deletedAt

	return u, nil
}

// GetUserByName retrieves a user by name (username, uses hash for lookup)
func GetUserByName(name string) (User, error) {
	nameHash := HashForLookup(name)
	row := db.QueryRow(`SELECT id, encrypted_name, name_nonce, encrypted_phone, phone_nonce, password_hash,
		password_salt, password_algo, phone_verified, verification_code,
		notification_method, encrypted_email_address, email_address_nonce, created_at, is_admin, sms_opted_out, deleted_at
		FROM User WHERE name_hash = ? AND deleted_at IS NULL`, nameHash)
	var u User
	var createdAt time.Time
	var isAdmin int
	var phoneVerified int
	var verificationCode *string
	var notificationMethod string
	var encryptedName, nameNonce, encryptedPhone, phoneNonce string
	var encryptedEmail, emailNonce sql.NullString
	var smsOptedOut int
	var deletedAt *time.Time
	err := row.Scan(&u.ID, &encryptedName, &nameNonce, &encryptedPhone, &phoneNonce, &u.PasswordHash, &u.PasswordSalt, &u.PasswordAlgo, &phoneVerified, &verificationCode, &notificationMethod, &encryptedEmail, &emailNonce, &createdAt, &isAdmin, &smsOptedOut, &deletedAt)
	if err != nil {
		return User{}, err
	}
	
	// Decrypt name and phone
	u.Name, err = DecryptName(u.ID, encryptedName, nameNonce)
	if err != nil {
		return User{}, fmt.Errorf("failed to decrypt name: %w", err)
	}
	u.Phone, err = DecryptPhone(u.ID, encryptedPhone, phoneNonce)
	if err != nil {
		return User{}, fmt.Errorf("failed to decrypt phone: %w", err)
	}
	
	// Decrypt email if present
	if encryptedEmail.Valid && emailNonce.Valid {
		email, err := DecryptEmailAddress(u.ID, encryptedEmail.String, emailNonce.String)
		if err == nil {
			u.EmailAddress = &email
		}
	}
	
	u.CreatedAt = createdAt
	u.IsAdmin = isAdmin == 1
	u.PhoneVerified = phoneVerified == 1
	u.VerificationCode = verificationCode
	u.NotificationMethod = notificationMethod
	u.SMSOptedOut = smsOptedOut == 1
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
	_, err := db.Exec(`UPDATE User SET phone_verified = 1, verification_code = '' WHERE id = ?`, userID)
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
	if emailAddress == nil || *emailAddress == "" {
		// Clear email
		_, err := db.Exec(`UPDATE User SET notification_method = ?, encrypted_email_address = NULL, email_address_nonce = NULL WHERE id = ?`, method, userID)
		return err
	}
	
	// Encrypt email address
	encryptedEmail, emailNonce, err := EncryptEmailAddress(userID, *emailAddress)
	if err != nil {
		return fmt.Errorf("failed to encrypt email: %w", err)
	}
	
	_, err = db.Exec(`UPDATE User SET notification_method = ?, encrypted_email_address = ?, email_address_nonce = ? WHERE id = ?`, method, encryptedEmail, emailNonce, userID)
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

// SetSMSOptOut sets the SMS opt-out flag for a user
func SetSMSOptOut(userID int, optedOut bool) error {
	val := 0
	if optedOut {
		val = 1
	}
	_, err := db.Exec("UPDATE User SET sms_opted_out = ? WHERE id = ?", val, userID)
	return err
}

// IsUnreachable checks if a user is unreachable for notifications
// A user is unreachable if they have SMS opted out AND their notification method is SMS
func IsUnreachable(userID int) bool {
	u, err := GetUser(userID)
	if err != nil {
		return false
	}
	return u.SMSOptedOut && u.NotificationMethod == NotificationMethodSMS
}
