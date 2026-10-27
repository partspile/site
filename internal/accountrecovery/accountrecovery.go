package accountrecovery

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nyaruka/phonenumbers"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/password"
	"github.com/partspile/site/internal/user"
)

const (
	CodeLength        = 6
	sessionTokenBytes = 32
	maxCodeAttempts   = 8

	failureUnknownPhone    = "unknown_phone"
	failureUnverifiedPhone = "unverified_phone"
)

var (
	ErrExpired        = errors.New("recovery session expired")
	ErrNotFound       = errors.New("recovery session not found")
	ErrNotVerified    = errors.New("recovery session not verified")
	ErrInvalidSMS     = errors.New("invalid recovery SMS")
	ErrNoUser         = errors.New("no verified user for phone")
	recoverSMSPattern = regexp.MustCompile(`(?i)^RECOVER\s+(\d{6})$`)
)

// Session is a newly started recovery flow (plaintext values for cookie/UI).
type Session struct {
	Token     string
	Code      string
	ExpiresAt time.Time
}

type StatusKind int

const (
	StatusPending StatusKind = iota
	StatusVerified
	StatusExpired
	StatusFailed
)

// Status is the current state of a recovery session for the browser.
type Status struct {
	Kind      StatusKind
	Username  string // set when Kind == StatusVerified
	Message   string // set when Kind == StatusFailed
	ExpiresAt time.Time
}

func hashValue(plaintext string) string {
	mac := hmac.New(sha256.New, config.JWTSecret)
	mac.Write([]byte(plaintext))
	return hex.EncodeToString(mac.Sum(nil))
}

func generateSessionToken() (string, error) {
	b := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateCode() (string, error) {
	code := ""
	for i := 0; i < CodeLength; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("failed to generate code digit: %w", err)
		}
		code += fmt.Sprintf("%d", num.Int64())
	}
	return code, nil
}

func cleanupExpired() {
	_, err := db.Exec(`DELETE FROM account_recovery WHERE expires_at <= $1`,
		time.Now().UTC())
	if err != nil {
		logger.Warn("Failed to cleanup expired recovery sessions",
			"error", err, "component", "accountrecovery")
	}
}

// Cancel removes a recovery session by plaintext session token.
func Cancel(sessionToken string) error {
	if sessionToken == "" {
		return nil
	}
	_, err := db.Exec(
		`DELETE FROM account_recovery WHERE session_token_hash = $1`,
		hashValue(sessionToken),
	)
	return err
}

// Start creates a new recovery session. Call Cancel on any prior token first.
func Start() (Session, error) {
	cleanupExpired()

	token, err := generateSessionToken()
	if err != nil {
		return Session{}, err
	}
	tokenHash := hashValue(token)
	expiresAt := time.Now().UTC().Add(config.RecoverySessionTTL)

	var code string
	for attempt := 0; attempt < maxCodeAttempts; attempt++ {
		code, err = generateCode()
		if err != nil {
			return Session{}, err
		}
		codeHash := hashValue(code)

		_, err = db.Exec(`
			INSERT INTO account_recovery (
				session_token_hash, code_hash, expires_at
			) VALUES ($1, $2, $3)
		`, tokenHash, codeHash, expiresAt)
		if err == nil {
			logger.Info("Started account recovery session",
				"component", "accountrecovery")
			return Session{Token: token, Code: code, ExpiresAt: expiresAt}, nil
		}
		if !isUniqueViolation(err) {
			return Session{}, fmt.Errorf("failed to create recovery session: %w", err)
		}
	}
	return Session{}, fmt.Errorf("failed to allocate unique recovery code")
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// ParseRecoverCode extracts the 6-digit code from an inbound SMS body.
// Expected format: RECOVER 123456 (case-insensitive).
func ParseRecoverCode(body string) (string, bool) {
	body = strings.TrimSpace(body)
	m := recoverSMSPattern.FindStringSubmatch(body)
	if m == nil {
		return "", false
	}
	return m[1], true
}

func normalizePhoneE64(fromPhone string) (string, error) {
	fromPhone = strings.TrimSpace(fromPhone)
	num, err := phonenumbers.Parse(fromPhone, "")
	if err != nil {
		if !strings.HasPrefix(fromPhone, "+") {
			num, err = phonenumbers.Parse(fromPhone, "US")
		}
		if err != nil {
			return "", err
		}
	}
	return phonenumbers.Format(num, phonenumbers.E164), nil
}

func failPendingByCode(codeHash, reason string) error {
	_, err := db.Exec(`
		UPDATE account_recovery
		SET failure = $1
		WHERE code_hash = $2
			AND user_id IS NULL
			AND failure IS NULL
			AND expires_at > $3
	`, reason, codeHash, time.Now().UTC())
	return err
}

// CompleteFromSMS binds a pending recovery session to the user who owns fromPhone.
func CompleteFromSMS(fromPhone, body string) error {
	code, ok := ParseRecoverCode(body)
	if !ok {
		return ErrInvalidSMS
	}

	cleanupExpired()
	codeHash := hashValue(code)
	now := time.Now().UTC()

	phoneE64, err := normalizePhoneE64(fromPhone)
	if err != nil {
		logger.Info("Recovery SMS phone normalize failed",
			"component", "accountrecovery")
		_ = failPendingByCode(codeHash, failureUnknownPhone)
		return ErrNoUser
	}

	u, err := user.GetByPhoneE64(phoneE64)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Info("Recovery SMS from unknown phone",
				"component", "accountrecovery")
			if err := failPendingByCode(codeHash, failureUnknownPhone); err != nil {
				return fmt.Errorf("mark recovery failure: %w", err)
			}
			return ErrNoUser
		}
		return fmt.Errorf("lookup user by phone: %w", err)
	}
	if !u.PhoneVerified {
		logger.Info("Recovery SMS from unverified phone",
			"component", "accountrecovery", "userID", u.ID)
		if err := failPendingByCode(codeHash, failureUnverifiedPhone); err != nil {
			return fmt.Errorf("mark recovery failure: %w", err)
		}
		return ErrNoUser
	}

	res, err := db.Exec(`
		UPDATE account_recovery
		SET user_id = $1, failure = NULL
		WHERE code_hash = $2
			AND user_id IS NULL
			AND failure IS NULL
			AND expires_at > $3
	`, u.ID, codeHash, now)
	if err != nil {
		return fmt.Errorf("bind recovery session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		logger.Info("Recovery SMS code not matched to pending session",
			"component", "accountrecovery", "userID", u.ID)
		return ErrNotFound
	}

	logger.Info("Recovery session verified via SMS",
		"component", "accountrecovery", "userID", u.ID)
	return nil
}

func failureMessage(failure string) string {
	switch failure {
	case failureUnknownPhone, failureUnverifiedPhone:
		return "No account is registered for the phone that sent that text. " +
			"Text from your registered number, or create an account first."
	default:
		return "Account recovery failed. Please try again."
	}
}

// GetStatus returns the status of a recovery session for the browser cookie.
func GetStatus(sessionToken string) (Status, error) {
	if sessionToken == "" {
		return Status{Kind: StatusExpired}, nil
	}

	cleanupExpired()

	var userID sql.NullInt64
	var failure sql.NullString
	var expiresAt time.Time
	err := db.QueryRow(`
		SELECT user_id, failure, expires_at
		FROM account_recovery
		WHERE session_token_hash = $1
	`, hashValue(sessionToken)).Scan(&userID, &failure, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Status{Kind: StatusExpired}, nil
		}
		return Status{}, fmt.Errorf("get recovery status: %w", err)
	}

	if !expiresAt.After(time.Now().UTC()) {
		_ = Cancel(sessionToken)
		return Status{Kind: StatusExpired}, nil
	}

	if failure.Valid && failure.String != "" {
		return Status{
			Kind:      StatusFailed,
			Message:   failureMessage(failure.String),
			ExpiresAt: expiresAt,
		}, nil
	}

	if !userID.Valid {
		return Status{Kind: StatusPending, ExpiresAt: expiresAt}, nil
	}

	u, err := user.GetByID(int(userID.Int64))
	if err != nil {
		_ = Cancel(sessionToken)
		return Status{Kind: StatusExpired}, nil
	}

	return Status{
		Kind:      StatusVerified,
		Username:  u.Name,
		ExpiresAt: expiresAt,
	}, nil
}

// ResetPassword updates the verified user's password and consumes the session.
func ResetPassword(sessionToken, newPassword, confirmPassword string) error {
	if sessionToken == "" {
		return ErrNotFound
	}
	if err := password.ValidatePasswordConfirmation(newPassword, confirmPassword); err != nil {
		return err
	}
	if err := password.ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

	cleanupExpired()

	tokenHash := hashValue(sessionToken)
	var id int
	var userID sql.NullInt64
	var failure sql.NullString
	var expiresAt time.Time
	err := db.QueryRow(`
		SELECT id, user_id, failure, expires_at
		FROM account_recovery
		WHERE session_token_hash = $1
	`, tokenHash).Scan(&id, &userID, &failure, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("load recovery session: %w", err)
	}
	if !expiresAt.After(time.Now().UTC()) {
		_, _ = db.Exec(`DELETE FROM account_recovery WHERE id = $1`, id)
		return ErrExpired
	}
	if failure.Valid && failure.String != "" {
		return ErrNotVerified
	}
	if !userID.Valid {
		return ErrNotVerified
	}

	newHash, newSalt, err := password.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`UPDATE users SET password_hash = $1, password_salt = $2,
			password_algo = $3 WHERE id = $4 AND deleted_at IS NULL`,
		newHash, newSalt, "argon2id", int(userID.Int64),
	)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	res, err := tx.Exec(`DELETE FROM account_recovery WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("consume recovery session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit password reset: %w", err)
	}

	logger.Info("Recovery password reset completed",
		"component", "accountrecovery", "userID", int(userID.Int64))
	return nil
}
