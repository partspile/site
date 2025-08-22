package verification

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/parts-pile/site/db"
)

const (
	// CodeLength is the length of verification codes
	CodeLength = 6
	// CodeExpiry is how long codes are valid
	CodeExpiry = 10 * time.Minute
	// MaxAttempts is the maximum verification attempts allowed
	MaxAttempts = 3
	// MaxFailedVerifications is the maximum failed verification attempts before account cleanup
	MaxFailedVerifications = 5
	// VerificationWindow is the time window for tracking failed verifications
	VerificationWindow = 24 * time.Hour
)

// VerificationCode represents a phone verification code
type VerificationCode struct {
	ID        int
	Phone     string
	Code      string
	ExpiresAt time.Time
	Attempts  int
	CreatedAt time.Time
}

// GenerateCode creates a new 6-digit verification code
func GenerateCode() (string, error) {
	code := ""
	for i := 0; i < CodeLength; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		code += fmt.Sprintf("%d", num.Int64())
	}
	return code, nil
}

// CreateVerificationCode stores a new verification code in the database
func CreateVerificationCode(phone, code string) error {
	expiresAt := time.Now().Add(CodeExpiry)

	_, err := db.Exec(`
		INSERT INTO PhoneVerification (phone, verification_code, expires_at, attempts) 
		VALUES (?, ?, ?, 0)
	`, phone, code, expiresAt.Format(time.RFC3339))

	if err != nil {
		return fmt.Errorf("failed to create verification code: %w", err)
	}

	log.Printf("[VERIFICATION] Created verification code for %s, expires at %s",
		phone, expiresAt.Format(time.RFC3339))
	return nil
}

// GetVerificationCode retrieves the most recent verification code for a phone
func GetVerificationCode(phone string) (*VerificationCode, error) {
	row := db.QueryRow(`
		SELECT id, phone, verification_code, expires_at, attempts, created_at
		FROM PhoneVerification 
		WHERE phone = ? 
		ORDER BY created_at DESC 
		LIMIT 1
	`, phone)

	var vc VerificationCode
	var expiresAtStr, createdAtStr string

	err := row.Scan(&vc.ID, &vc.Phone, &vc.Code, &expiresAtStr,
		&vc.Attempts, &createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("verification code not found: %w", err)
	}

	vc.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAtStr)
	vc.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

	return &vc, nil
}

// ValidateCode checks if a verification code is valid and not expired
func ValidateCode(phone, code string) (bool, error) {
	vc, err := GetVerificationCode(phone)
	if err != nil {
		return false, err
	}

	// Check if code has expired
	if time.Now().After(vc.ExpiresAt) {
		log.Printf("[VERIFICATION] Code expired for %s", phone)
		return false, nil
	}

	// Check if max attempts exceeded
	if vc.Attempts >= MaxAttempts {
		log.Printf("[VERIFICATION] Max attempts exceeded for %s", phone)
		// Track this failed verification for potential account cleanup
		TrackFailedVerification(phone)
		return false, nil
	}

	// Increment attempts
	_, err = db.Exec(`
		UPDATE PhoneVerification 
		SET attempts = attempts + 1 
		WHERE id = ?
	`, vc.ID)
	if err != nil {
		log.Printf("[VERIFICATION] Failed to update attempts: %v", err)
	}

	// Check if code matches
	if vc.Code == code {
		log.Printf("[VERIFICATION] Code validated successfully for %s", phone)
		return true, nil
	}

	log.Printf("[VERIFICATION] Invalid code for %s", phone)
	return false, nil
}

// CleanupExpiredCodes removes expired verification codes
func CleanupExpiredCodes() error {
	_, err := db.Exec(`
		DELETE FROM PhoneVerification 
		WHERE expires_at < ?
	`, time.Now().Format(time.RFC3339))

	if err != nil {
		return fmt.Errorf("failed to cleanup expired codes: %w", err)
	}

	log.Printf("[VERIFICATION] Cleaned up expired verification codes")
	return nil
}

// MarkPhoneVerified updates a user's phone verification status
func MarkPhoneVerified(userID int) error {
	_, err := db.Exec(`
		UPDATE User 
		SET phone_verified = 1, verification_code = NULL 
		WHERE id = ?
	`, userID)

	if err != nil {
		return fmt.Errorf("failed to mark phone verified: %w", err)
	}

	log.Printf("[VERIFICATION] Marked phone verified for user %d", userID)
	return nil
}

// InvalidateVerificationCodes invalidates all verification codes for a phone number
// This is called when a user replies STOP or when SMS delivery fails
func InvalidateVerificationCodes(phone string) error {
	_, err := db.Exec(`
		DELETE FROM PhoneVerification 
		WHERE phone = ?
	`, phone)

	if err != nil {
		return fmt.Errorf("failed to invalidate verification codes for %s: %w", phone, err)
	}

	log.Printf("[VERIFICATION] Invalidated all verification codes for %s", phone)
	return nil
}

// TrackFailedVerification records a failed verification attempt and cleans up if threshold exceeded
func TrackFailedVerification(phone string) error {
	// Count failed verifications in the last 24 hours
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM PhoneVerification 
		WHERE phone = ? AND created_at > ?
	`, phone, time.Now().Add(-VerificationWindow).Format(time.RFC3339)).Scan(&count)

	if err != nil {
		return fmt.Errorf("failed to count failed verifications for %s: %w", phone, err)
	}

	// If we've exceeded the threshold, clean up the account
	if count >= MaxFailedVerifications {
		log.Printf("[VERIFICATION] Max failed verifications exceeded for %s, cleaning up account", phone)
		return CleanupFailedAccount(phone)
	}

	return nil
}

// CleanupFailedAccount removes all data associated with a failed verification attempt
func CleanupFailedAccount(phone string) error {
	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete verification codes
	_, err = tx.Exec(`DELETE FROM PhoneVerification WHERE phone = ?`, phone)
	if err != nil {
		return fmt.Errorf("failed to delete verification codes: %w", err)
	}

	// Delete any partial user records (users created but not verified)
	_, err = tx.Exec(`DELETE FROM User WHERE phone = ? AND phone_verified = 0`, phone)
	if err != nil {
		return fmt.Errorf("failed to delete unverified user: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit cleanup transaction: %w", err)
	}

	log.Printf("[VERIFICATION] Successfully cleaned up failed account for %s", phone)
	return nil
}
