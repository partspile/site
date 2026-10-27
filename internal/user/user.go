package user

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/password"
	"github.com/partspile/site/internal/rock"
	"github.com/partspile/site/internal/vector"
)

// UserStatus represents the status of a user
type UserStatus string

const (
	StatusActive   UserStatus = "active"
	StatusArchived UserStatus = "archived"
)

type User struct {
	ID                int
	Name              string // Decrypted (calculated field)
	EncryptedName     string
	NameNonce         string
	PasswordHash      string
	PasswordSalt      string
	PasswordAlgo      string
	PhoneE64          string // Decrypted (calculated field)
	EncryptedPhone    string
	PhoneNonce        string
	CreatedAt         time.Time
	IsAdmin           bool
	PhoneVerified     bool
	SMSOptedOut       bool
	HasAccountPicture bool
	AccountPictureURL string
	DeletedAt         *time.Time
}

const userSelectFields = `SELECT 
	id,
	encrypted_name,
	name_nonce,
	encrypted_phone,
	phone_nonce,
	password_hash,
	password_salt,
	password_algo,
	phone_verified,
	created_at,
	is_admin,
	sms_opted_out,
	has_account_picture,
	account_picture_url,
	deleted_at
FROM users`

func processUserRow(id int, encryptedNameBytes, nameNonceBytes []byte,
	encryptedPhoneBytes, phoneNonceBytes []byte, passwordHash, passwordSalt,
	passwordAlgo string, phoneVerifiedInt int, createdAt time.Time, isAdminInt int,
	smsOptedOutInt, hasAccountPictureInt int, accountPictureURL sql.NullString,
	deletedAt *time.Time, withPhone bool) (User, error) {
	var u User

	u.ID = id
	u.PasswordHash = passwordHash
	u.PasswordSalt = passwordSalt
	u.PasswordAlgo = passwordAlgo
	u.CreatedAt = createdAt
	u.DeletedAt = deletedAt

	u.EncryptedName = base64.StdEncoding.EncodeToString(encryptedNameBytes)
	u.NameNonce = base64.StdEncoding.EncodeToString(nameNonceBytes)
	u.EncryptedPhone = base64.StdEncoding.EncodeToString(encryptedPhoneBytes)
	u.PhoneNonce = base64.StdEncoding.EncodeToString(phoneNonceBytes)

	u.PhoneVerified = phoneVerifiedInt == 1
	u.IsAdmin = isAdminInt == 1
	u.SMSOptedOut = smsOptedOutInt == 1
	u.HasAccountPicture = hasAccountPictureInt == 1
	if accountPictureURL.Valid {
		u.AccountPictureURL = accountPictureURL.String
	}

	var err error
	u.Name, err = decryptName(u.ID, u.EncryptedName, u.NameNonce)
	if err != nil {
		return User{}, fmt.Errorf("failed to decrypt name: %w", err)
	}
	if withPhone {
		u.PhoneE64, err = decryptPhone(u.ID, u.EncryptedPhone, u.PhoneNonce)
		if err != nil {
			return User{}, fmt.Errorf("failed to decrypt phone: %w", err)
		}
	}

	return u, nil
}

// scanner interface for both *sql.Row and *sql.Rows
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanUserFields(s scanner) (User, error) {
	return scanUserFieldsOpt(s, true)
}

func scanUserFieldsOpt(s scanner, withPhone bool) (User, error) {
	var id int
	var encryptedNameBytes, nameNonceBytes []byte
	var encryptedPhoneBytes, phoneNonceBytes []byte
	var phoneVerifiedInt, isAdminInt, smsOptedOutInt int
	var hasAccountPictureInt int
	var accountPictureURL sql.NullString
	var passwordHash, passwordSalt, passwordAlgo string
	var createdAt time.Time
	var deletedAt *time.Time

	err := s.Scan(
		&id,
		&encryptedNameBytes,
		&nameNonceBytes,
		&encryptedPhoneBytes,
		&phoneNonceBytes,
		&passwordHash,
		&passwordSalt,
		&passwordAlgo,
		&phoneVerifiedInt,
		&createdAt,
		&isAdminInt,
		&smsOptedOutInt,
		&hasAccountPictureInt,
		&accountPictureURL,
		&deletedAt,
	)
	if err != nil {
		return User{}, err
	}

	return processUserRow(
		id,
		encryptedNameBytes, nameNonceBytes,
		encryptedPhoneBytes, phoneNonceBytes,
		passwordHash, passwordSalt, passwordAlgo,
		phoneVerifiedInt,
		createdAt,
		isAdminInt,
		smsOptedOutInt,
		hasAccountPictureInt,
		accountPictureURL,
		deletedAt,
		withPhone,
	)
}

func getUserBy(whereClause string, args ...any) (User, error) {
	query := userSelectFields + " WHERE " + whereClause
	return scanUserFields(db.QueryRow(query, args...))
}

func GetByID(id int) (User, error) {
	return getUserBy("id = $1 AND deleted_at IS NULL", id)
}

// GetByIDIncludingDeleted returns a user by ID, including deleted users
func GetByIDIncludingDeleted(id int) (User, error) {
	return getUserBy("id = $1", id)
}

// Exists checks if a user exists and is not deleted (lightweight check without decryption)
func Exists(id int) bool {
	var exists int
	err := db.QueryRow("SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL", id).Scan(&exists)
	return err == nil
}

// SessionAuth returns password salt and admin flag for an active user
// without decryption. ok is false if the user does not exist or is deleted.
func SessionAuth(id int) (salt string, isAdmin bool, ok bool) {
	var isAdminInt int
	err := db.QueryRow(
		`SELECT password_salt, is_admin FROM users
		 WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&salt, &isAdminInt)
	if err != nil {
		return "", false, false
	}
	return salt, isAdminInt == 1, true
}

func GetByPhoneE64(phoneE64 string) (User, error) {
	phoneHash := db.HashString(phoneE64)
	return getUserBy("phone_hash = $1 AND deleted_at IS NULL", phoneHash)
}

func GetByName(name string) (User, error) {
	nameHash := db.HashString(name)
	return getUserBy("name_hash = $1 AND deleted_at IS NULL", nameHash)
}

var (
	// ErrUserAlreadyExists is returned when username or live phone is taken.
	ErrUserAlreadyExists = errors.New("user already exists")
	// ErrPhoneSame is returned when changing to the current phone number.
	ErrPhoneSame = errors.New("phone number is unchanged")
)

// PhoneHoldDuration is how long a deleted account's phone stays unavailable.
const PhoneHoldDuration = 10 * 24 * time.Hour

// PhoneStatus describes whether a phone may be claimed.
type PhoneStatus int

const (
	PhoneAvailable PhoneStatus = iota
	PhoneActive
	PhoneHeld
)

// PhoneAvailability is the result of checking whether a phone can be used.
type PhoneAvailability struct {
	Status        PhoneStatus
	DaysRemaining int
}

// PhoneHoldError is returned when a phone is within the post-delete hold.
type PhoneHoldError struct {
	DaysRemaining int
}

func (e *PhoneHoldError) Error() string {
	n := e.DaysRemaining
	if n < 1 {
		n = 1
	}
	if n == 1 {
		return "This phone number will be available in 1 day. Please come back then."
	}
	return fmt.Sprintf(
		"This phone number will be available in %d days. Please come back then.",
		n,
	)
}

// UsernameTaken reports whether username is reserved, including deleted users.
func UsernameTaken(username string) (bool, error) {
	nameHash := db.HashString(username)
	var id int
	err := db.QueryRow(`
		SELECT id FROM users WHERE name_hash = $1 LIMIT 1
	`, nameHash).Scan(&id)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

// CheckPhoneAvailability reports whether phoneE64 can be claimed.
// excludeUserID skips that live user (for change-phone on the same account).
func CheckPhoneAvailability(phoneE64 string,
	excludeUserID int) (PhoneAvailability, error) {
	phoneHash := db.HashString(phoneE64)

	var activeID int
	err := db.QueryRow(`
		SELECT id FROM users
		WHERE phone_hash = $1 AND deleted_at IS NULL
		LIMIT 1
	`, phoneHash).Scan(&activeID)
	if err == nil {
		if excludeUserID != 0 && activeID == excludeUserID {
			return PhoneAvailability{Status: PhoneAvailable}, nil
		}
		return PhoneAvailability{Status: PhoneActive}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return PhoneAvailability{}, err
	}

	var deletedAt time.Time
	err = db.QueryRow(`
		SELECT deleted_at FROM users
		WHERE phone_hash = $1
			AND deleted_at IS NOT NULL
			AND deleted_at > $2
		ORDER BY deleted_at DESC
		LIMIT 1
	`, phoneHash, time.Now().UTC().Add(-PhoneHoldDuration)).Scan(&deletedAt)
	if err == nil {
		holdUntil := deletedAt.Add(PhoneHoldDuration)
		return PhoneAvailability{
			Status:        PhoneHeld,
			DaysRemaining: holdDaysRemaining(holdUntil),
		}, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return PhoneAvailability{Status: PhoneAvailable}, nil
	}
	return PhoneAvailability{}, err
}

func holdDaysRemaining(holdUntil time.Time) int {
	remaining := time.Until(holdUntil)
	if remaining <= 0 {
		return 0
	}
	days := int(remaining.Hours() / 24)
	if remaining > time.Duration(days)*24*time.Hour {
		days++
	}
	if days < 1 {
		days = 1
	}
	return days
}

func phoneAvailabilityError(a PhoneAvailability) error {
	switch a.Status {
	case PhoneActive:
		return ErrUserAlreadyExists
	case PhoneHeld:
		return &PhoneHoldError{DaysRemaining: a.DaysRemaining}
	default:
		return nil
	}
}

// CreateUser creates a new user with phone verification in a transaction.
// Usernames are reserved forever (including deleted). Phones may be reused
// after the post-delete hold once no live user holds them.
func CreateUser(username, phoneE64, plainPassword string) (User, error) {
	passwordHash, passwordSalt, err := password.HashPassword(plainPassword)
	if err != nil {
		return User{}, fmt.Errorf("failed to hash password: %w", err)
	}

	taken, err := UsernameTaken(username)
	if err != nil {
		return User{}, fmt.Errorf("failed to check username: %w", err)
	}
	if taken {
		return User{}, ErrUserAlreadyExists
	}

	avail, err := CheckPhoneAvailability(phoneE64, 0)
	if err != nil {
		return User{}, fmt.Errorf("failed to check phone: %w", err)
	}
	if err := phoneAvailabilityError(avail); err != nil {
		return User{}, err
	}

	tx, err := db.Begin()
	if err != nil {
		return User{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	nameHash := db.HashString(username)
	phoneHash := db.HashString(phoneE64)

	var userID int
	err = tx.QueryRow(`
		INSERT INTO users (
			encrypted_name, name_nonce, name_hash,
			password_hash, password_salt, password_algo,
			encrypted_phone, phone_nonce, phone_hash,
			phone_verified, is_admin
		) VALUES ($1, $2, $3, $4, $5, 'argon2id', $6, $7, $8, 0, 0)
		RETURNING id
	`, []byte{}, []byte{}, nameHash,
		passwordHash, passwordSalt,
		[]byte{}, []byte{}, phoneHash).Scan(&userID)
	if err != nil {
		return User{}, fmt.Errorf("failed to create user: %w", err)
	}

	encryptedName, nameNonce, err := EncryptName(int(userID), username)
	if err != nil {
		return User{}, fmt.Errorf("failed to encrypt name: %w", err)
	}
	encryptedNameBytes, _ := base64.StdEncoding.DecodeString(encryptedName)
	nameNonceBytes, _ := base64.StdEncoding.DecodeString(nameNonce)

	encryptedPhone, phoneNonce, err := EncryptPhone(int(userID), phoneE64)
	if err != nil {
		return User{}, fmt.Errorf("failed to encrypt phone: %w", err)
	}
	encryptedPhoneBytes, _ := base64.StdEncoding.DecodeString(encryptedPhone)
	phoneNonceBytes, _ := base64.StdEncoding.DecodeString(phoneNonce)

	_, err = tx.Exec(`
		UPDATE users SET
			encrypted_name = $1, name_nonce = $2,
			encrypted_phone = $3, phone_nonce = $4,
			phone_verified = 1
		WHERE id = $5
	`, encryptedNameBytes, nameNonceBytes,
		encryptedPhoneBytes, phoneNonceBytes,
		userID)
	if err != nil {
		return User{}, fmt.Errorf("failed to update user with encrypted fields: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return User{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	u, err := GetByID(int(userID))
	if err != nil {
		return User{}, fmt.Errorf("failed to get created user: %w", err)
	}

	return u, nil
}

// UpdatePhone changes a live user's phone after availability checks.
func UpdatePhone(userID int, phoneE64 string) error {
	u, err := GetByID(userID)
	if err != nil {
		return err
	}
	if u.PhoneE64 == phoneE64 {
		return ErrPhoneSame
	}

	avail, err := CheckPhoneAvailability(phoneE64, userID)
	if err != nil {
		return fmt.Errorf("failed to check phone: %w", err)
	}
	if err := phoneAvailabilityError(avail); err != nil {
		return err
	}

	encryptedPhone, phoneNonce, err := EncryptPhone(userID, phoneE64)
	if err != nil {
		return fmt.Errorf("failed to encrypt phone: %w", err)
	}
	encryptedPhoneBytes, _ := base64.StdEncoding.DecodeString(encryptedPhone)
	phoneNonceBytes, _ := base64.StdEncoding.DecodeString(phoneNonce)
	phoneHash := db.HashString(phoneE64)

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE users SET
			encrypted_phone = $1, phone_nonce = $2, phone_hash = $3,
			phone_verified = 1
		WHERE id = $4 AND deleted_at IS NULL
	`, encryptedPhoneBytes, phoneNonceBytes, phoneHash, userID)
	if err != nil {
		return fmt.Errorf("failed to update phone: %w", err)
	}

	return tx.Commit()
}

func SetSMSOptOut(userID int, optedOut bool) error {
	val := 0
	if optedOut {
		val = 1
	}
	_, err := db.Exec("UPDATE users SET sms_opted_out = $1 WHERE id = $2", val, userID)
	return err
}

// SetSMSOptOutByPhoneE64 sets sms_opted_out for the active user with this phone.
func SetSMSOptOutByPhoneE64(phoneE64 string, optedOut bool) error {
	u, err := GetByPhoneE64(phoneE64)
	if err != nil {
		return err
	}
	return SetSMSOptOut(u.ID, optedOut)
}

func UpdatePassword(userID int, hash, salt, algo string) error {
	_, err := db.Exec("UPDATE users SET password_hash = $1, password_salt = $2, password_algo = $3 WHERE id = $4", hash, salt, algo, userID)
	return err
}

// GetAllUsers returns all users (including deleted) with optional sorting.
// Phones are not decrypted; use GetByName / GetByID when a phone is needed.
func GetAllUsers(sortBy string, sortOrder string) ([]User, error) {
	validSortColumns := map[string]bool{
		"id":         true,
		"name":       true,
		"is_admin":   true,
		"created_at": true,
		"deleted_at": true,
	}

	if !validSortColumns[sortBy] {
		sortBy = "id"
	}

	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "ASC"
	}

	query := userSelectFields

	needsInMemorySort := sortBy == "name"
	if !needsInMemorySort {
		query += " ORDER BY " + sortBy + " " + sortOrder
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		u, err := scanUserFieldsOpt(rows, false)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if needsInMemorySort {
		sort.Slice(users, func(i, j int) bool {
			less := users[i].Name < users[j].Name
			if sortOrder == "DESC" {
				return !less
			}
			return less
		})
	}

	return users, nil
}

func DeleteUser(userID int) error {
	now := time.Now()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"UPDATE users SET deleted_at = $1 WHERE id = $2", now, userID,
	)
	if err != nil {
		return err
	}

	rows, err := tx.Query(`
		SELECT id FROM ads
		WHERE user_id = $1 AND deleted_at IS NULL
	`, userID)
	if err != nil {
		return err
	}
	var adIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		adIDs = append(adIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = tx.Exec(`
		UPDATE ads
		SET deleted_at = $1, inactive_at = NULL
		WHERE user_id = $2 AND deleted_at IS NULL
	`, now, userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		UPDATE conversations SET owner_has_unread = 0 WHERE owner_id = $1
	`, userID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
		UPDATE conversations SET inquirer_has_unread = 0 WHERE inquirer_id = $1
	`, userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		UPDATE sms_notification_queue
		SET status = 'suppressed', processed_at = $1
		WHERE recipient_user_id = $2 AND status = 'pending'
	`, now, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	_ = rock.UnthrowActiveForUser(userID)
	for _, adID := range adIDs {
		_ = rock.UnthrowActiveForAd(adID)
		_ = vector.DeleteAdEmbedding(adID)
		_, _ = db.Exec(`
			DELETE FROM rock_opinions
			WHERE conversation_id IN (
				SELECT id FROM conversations WHERE ad_id = $1
			)
		`, adID)
	}
	return nil
}

func PromoteToAdmin(userID int) error {
	_, err := db.Exec("UPDATE users SET is_admin = 1 WHERE id = $1", userID)
	return err
}

func DemoteFromAdmin(userID int) error {
	_, err := db.Exec("UPDATE users SET is_admin = 0 WHERE id = $1", userID)
	return err
}

// ConfirmAccountPicture marks that users/{id}/account.jpg was uploaded.
func ConfirmAccountPicture(userID int) error {
	_, err := db.Exec(
		`UPDATE users SET has_account_picture = 1 WHERE id = $1`, userID,
	)
	return err
}

// ClearAccountPicture clears the picture flag (caller deletes MinIO object).
func ClearAccountPicture(userID int) error {
	_, err := db.Exec(
		`UPDATE users SET has_account_picture = 0 WHERE id = $1`, userID,
	)
	return err
}

// SetAccountPictureURL sets or clears the optional click-through URL.
func SetAccountPictureURL(userID int, pictureURL string) error {
	var arg any
	if pictureURL == "" {
		arg = nil
	} else {
		arg = pictureURL
	}
	_, err := db.Exec(
		`UPDATE users SET account_picture_url = $1 WHERE id = $2`,
		arg, userID,
	)
	return err
}

var testPhonePattern = regexp.MustCompile(`^\+1555010\d{4}$`)

// IsTestPhoneE64 reports whether phoneE64 is a seeded test account.
func IsTestPhoneE64(phoneE64 string) bool {
	return testPhonePattern.MatchString(phoneE64)
}

// IsTestUser reports whether userID belongs to a seeded test account.
func IsTestUser(userID int) (bool, error) {
	u, err := GetByID(userID)
	if err != nil {
		return false, err
	}
	return IsTestPhoneE64(u.PhoneE64), nil
}

// IsReachable reports whether the user can be contacted via the platform.
// Users must verify their phone during registration before they can receive messages.
// TODO: incorporate notification preferences and SMS opt-out.
func IsReachable(userID int) (bool, error) {
	u, err := GetByID(userID)
	if err != nil {
		return false, err
	}
	return u.PhoneVerified, nil
}
