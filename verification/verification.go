package verification

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"math/big"
	"time"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
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
	ID               int
	Phone            string
	Code             string
	Attempts         int
	RegistrationName string
	CreatedAt        time.Time
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
	_, err := db.Exec(`
		INSERT INTO PhoneVerification (phone, verification_code, attempts) 
		VALUES ($1, $2, 0)
	`, phone, code)

	if err != nil {
		return fmt.Errorf("failed to create verification code: %w", err)
	}

	logger.Info("Created verification code",
		"component", "verification", "phone", phone)
	return nil
}

// CreateRegistrationVerificationCode stores a new verification code with registration name
func CreateRegistrationVerificationCode(phone, code, name string) error {
	_, err := db.Exec(`
		INSERT INTO PhoneVerification (phone, verification_code, attempts, registration_name) 
		VALUES ($1, $2, 0, $3)
	`, phone, code, name)

	if err != nil {
		return fmt.Errorf("failed to create registration verification code: %w", err)
	}

	logger.Info("Created registration verification code",
		"component", "verification", "phone", phone, "name", name)
	return nil
}

// GetVerificationCode retrieves the most recent verification code for a phone
func GetVerificationCode(phone string) (*VerificationCode, error) {
	row := db.QueryRow(`
		SELECT id, phone, verification_code, attempts, registration_name, created_at
		FROM PhoneVerification 
		WHERE phone = $1 
		ORDER BY created_at DESC 
		LIMIT 1
	`, phone)

	var vc VerificationCode

	err := row.Scan(&vc.ID, &vc.Phone, &vc.Code,
		&vc.Attempts, &vc.RegistrationName, &vc.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("verification code not found: %w", err)
	}

	// Ensure timestamp is in UTC
	vc.CreatedAt = vc.CreatedAt.UTC()

	return &vc, nil
}

// ValidateCode checks if a verification code is valid and not expired
func ValidateCode(phone, code string) (bool, error) {
	vc, err := GetVerificationCode(phone)
	if err != nil {
		return false, err
	}

	// Check if code has expired (calculate from created_at + CodeExpiry)
	expiresAt := vc.CreatedAt.Add(CodeExpiry)
	if time.Now().UTC().After(expiresAt) {
		logger.Warn("Code expired",
			"component", "verification", "phone", phone,
			"now", time.Now().UTC().Format(time.RFC3339),
			"createdAt", vc.CreatedAt.Format(time.RFC3339),
			"expiresAt", expiresAt.Format(time.RFC3339))
		return false, nil
	}

	// Check if max attempts exceeded
	if vc.Attempts >= MaxAttempts {
		logger.Warn("Max attempts exceeded",
			"component", "verification", "phone", phone)
		// Track this failed verification for potential account cleanup
		TrackFailedVerification(phone)
		return false, nil
	}

	// Increment attempts
	_, err = db.Exec(`
		UPDATE PhoneVerification 
		SET attempts = attempts + 1 
		WHERE id = $1
	`, vc.ID)
	if err != nil {
		logger.Error("Failed to update attempts",
			"error", err, "component", "verification")
	}

	// Check if code matches using constant-time comparison to prevent timing attacks
	// Convert codes to byte slices for comparison
	codeBytes := []byte(vc.Code)
	inputBytes := []byte(code)

	// Ensure both slices are the same length for constant-time comparison
	if len(codeBytes) != len(inputBytes) {
		logger.Warn("Invalid code (length mismatch)",
			"component", "verification", "phone", phone)
		return false, nil
	}

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare(codeBytes, inputBytes) == 1 {
		logger.Info("Code validated successfully",
			"component", "verification", "phone", phone)
		return true, nil
	}

	logger.Warn("Invalid code",
		"component", "verification", "phone", phone)
	return false, nil
}

// CleanupExpiredCodes removes expired verification codes
func CleanupExpiredCodes() error {
	// Delete codes where created_at + CodeExpiry < now
	expiryThreshold := time.Now().UTC().Add(-CodeExpiry)
	_, err := db.Exec(`
		DELETE FROM PhoneVerification 
		WHERE created_at < $1
	`, expiryThreshold)

	if err != nil {
		return fmt.Errorf("failed to cleanup expired codes: %w", err)
	}

	logger.Info("Cleaned up expired verification codes",
		"component", "verification")
	return nil
}

// MarkPhoneVerified updates a user's phone verification status
func MarkPhoneVerified(userID int) error {
	_, err := db.Exec(`
		UPDATE Users 
		SET phone_verified = 1, verification_code = '' 
		WHERE id = $1
	`, userID)

	if err != nil {
		return fmt.Errorf("failed to mark phone verified: %w", err)
	}

	logger.Info("Marked phone verified for user",
		"component", "verification", "userID", userID)
	return nil
}

// InvalidateVerificationCodes invalidates all verification codes for a phone number
// This is called when a user replies STOP or when SMS delivery fails
func InvalidateVerificationCodes(phone string) error {
	_, err := db.Exec(`
		DELETE FROM PhoneVerification 
		WHERE phone = $1
	`, phone)

	if err != nil {
		return fmt.Errorf("failed to invalidate verification codes for %s: %w", phone, err)
	}

	logger.Info("Invalidated all verification codes for phone",
		"component", "verification", "phone", phone)
	return nil
}

// TrackFailedVerification records a failed verification attempt and cleans up if threshold exceeded
func TrackFailedVerification(phone string) error {
	// Count failed verifications in the last 24 hours
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM PhoneVerification 
		WHERE phone = $1 AND created_at > $2
	`, phone, time.Now().UTC().Add(-VerificationWindow)).Scan(&count)

	if err != nil {
		return fmt.Errorf("failed to count failed verifications for %s: %w", phone, err)
	}

	// If we've exceeded the threshold, clean up the account
	if count >= MaxFailedVerifications {
		logger.Warn("Max failed verifications exceeded, cleaning up account",
			"component", "verification", "phone", phone)
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
	_, err = tx.Exec(`DELETE FROM PhoneVerification WHERE phone = $1`, phone)
	if err != nil {
		return fmt.Errorf("failed to delete verification codes: %w", err)
	}

	// Delete any partial user records (users created but not verified)
	_, err = tx.Exec(`DELETE FROM Users WHERE phone = $1 AND phone_verified = 0`, phone)
	if err != nil {
		return fmt.Errorf("failed to delete unverified user: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit cleanup transaction: %w", err)
	}

	logger.Info("Successfully cleaned up failed account",
		"component", "verification", "phone", phone)
	return nil
}
