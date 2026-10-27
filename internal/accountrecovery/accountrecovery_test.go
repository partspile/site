package accountrecovery

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/db/testdb"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/user"
)

const (
	testDBEncryptionKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	testDBHashPepper    = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="
	testJWTSecret       = "test-jwt-secret-key-for-accountrecovery-tests"
)

func TestMain(m *testing.M) {
	if err := chdirModuleRoot(); err != nil {
		panic(err)
	}
	testURL := testdb.PackageDatabaseURL("accountrecovery")
	if err := testdb.EnsureDatabase(testURL); err != nil {
		panic(err)
	}
	os.Setenv("DATABASE_URL", testURL)
	os.Setenv("DB_ENCRYPTION_KEY", testDBEncryptionKey)
	os.Setenv("DB_HASH_PEPPER", testDBHashPepper)
	os.Setenv("JWT_SECRET", testJWTSecret)
	if key, err := base64.StdEncoding.DecodeString(testDBEncryptionKey); err == nil {
		reflect.ValueOf(&config.DBEncryptionKey).Elem().Set(reflect.ValueOf(key))
	}
	if pepper, err := base64.StdEncoding.DecodeString(testDBHashPepper); err == nil {
		reflect.ValueOf(&config.DBHashPepper).Elem().Set(reflect.ValueOf(pepper))
		db.SetHashPepper(pepper)
	}
	reflect.ValueOf(&config.JWTSecret).Elem().Set(reflect.ValueOf([]byte(testJWTSecret)))
	if err := logger.Init("error", "text", ""); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func chdirModuleRoot() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return os.Chdir(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func resetSchema(t *testing.T) {
	t.Helper()
	if err := testdb.InitSchema(); err != nil {
		t.Fatalf("init schema: %v", err)
	}
}

func TestParseRecoverCode(t *testing.T) {
	tests := []struct {
		body string
		code string
		ok   bool
	}{
		{"RECOVER 123456", "123456", true},
		{"recover 000001", "000001", true},
		{"  RECOVER 999999  ", "999999", true},
		{"RECOVER123456", "", false},
		{"123456", "", false},
		{"STOP", "", false},
		{"RECOVER 12345", "", false},
		{"RECOVER 1234567", "", false},
		{"HELLO RECOVER 123456", "", false},
	}
	for _, tt := range tests {
		code, ok := ParseRecoverCode(tt.body)
		if ok != tt.ok || code != tt.code {
			t.Errorf("ParseRecoverCode(%q) = %q,%v; want %q,%v",
				tt.body, code, ok, tt.code, tt.ok)
		}
	}
}

func TestStartStatusCompleteReset(t *testing.T) {
	resetSchema(t)

	const phone = "+15559873001"
	u, err := user.CreateUser("recoverme", phone, "oldpassword")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	session, err := Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(session.Code) != CodeLength {
		t.Fatalf("code length: %d", len(session.Code))
	}

	st, err := GetStatus(session.Token)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if st.Kind != StatusPending {
		t.Fatalf("want pending, got %v", st.Kind)
	}
	if st.ExpiresAt.IsZero() {
		t.Fatal("pending ExpiresAt")
	}

	err = CompleteFromSMS(phone, "RECOVER "+session.Code)
	if err != nil {
		t.Fatalf("CompleteFromSMS: %v", err)
	}

	st, err = GetStatus(session.Token)
	if err != nil {
		t.Fatalf("GetStatus verified: %v", err)
	}
	if st.Kind != StatusVerified || st.Username != "recoverme" {
		t.Fatalf("want verified recoverme, got %+v", st)
	}

	err = ResetPassword(session.Token, "newpassword", "newpassword")
	if err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}

	st, err = GetStatus(session.Token)
	if err != nil {
		t.Fatalf("GetStatus after reset: %v", err)
	}
	if st.Kind != StatusExpired {
		t.Fatalf("want expired after consume, got %v", st.Kind)
	}

	err = ResetPassword(session.Token, "anotherpass", "anotherpass")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("second reset: want ErrNotFound, got %v", err)
	}

	got, err := user.GetByID(u.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.PasswordSalt == u.PasswordSalt {
		t.Fatal("password salt should change after reset")
	}
}

func TestCompleteUnknownPhoneFailsSession(t *testing.T) {
	resetSchema(t)

	session, err := Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	err = CompleteFromSMS("+15035550199", "RECOVER "+session.Code)
	if !errors.Is(err, ErrNoUser) {
		t.Fatalf("want ErrNoUser, got %v", err)
	}

	st, err := GetStatus(session.Token)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if st.Kind != StatusFailed {
		t.Fatalf("want StatusFailed, got %v", st.Kind)
	}
	if st.Message == "" {
		t.Fatal("expected failure message")
	}
}

func TestCancelAndExpire(t *testing.T) {
	resetSchema(t)

	session, err := Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := Cancel(session.Token); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	st, err := GetStatus(session.Token)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if st.Kind != StatusExpired {
		t.Fatalf("want expired after cancel, got %v", st.Kind)
	}

	session, err = Start()
	if err != nil {
		t.Fatalf("Start 2: %v", err)
	}
	_, err = db.Exec(`
		UPDATE account_recovery
		SET expires_at = $1
		WHERE session_token_hash = $2
	`, time.Now().UTC().Add(-time.Minute), hashValue(session.Token))
	if err != nil {
		t.Fatalf("force expire: %v", err)
	}
	st, err = GetStatus(session.Token)
	if err != nil {
		t.Fatalf("GetStatus expired: %v", err)
	}
	if st.Kind != StatusExpired {
		t.Fatalf("want expired, got %v", st.Kind)
	}
}

func TestResetRequiresVerification(t *testing.T) {
	resetSchema(t)

	session, err := Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	err = ResetPassword(session.Token, "newpassword", "newpassword")
	if !errors.Is(err, ErrNotVerified) {
		t.Fatalf("want ErrNotVerified, got %v", err)
	}
}

func TestCodeUniqueness(t *testing.T) {
	resetSchema(t)

	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		s, err := Start()
		if err != nil {
			t.Fatalf("Start %d: %v", i, err)
		}
		if seen[s.Code] {
			t.Fatalf("duplicate code %s", s.Code)
		}
		seen[s.Code] = true
	}
}
