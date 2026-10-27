package user

import (
	"database/sql"
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
)

const testDBEncryptionKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
const testDBHashPepper = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="

func TestMain(m *testing.M) {
	if err := chdirModuleRoot(); err != nil {
		panic(err)
	}
	testURL := testdb.PackageDatabaseURL("user")
	if err := testdb.EnsureDatabase(testURL); err != nil {
		panic(err)
	}
	os.Setenv("DATABASE_URL", testURL)
	os.Setenv("DB_ENCRYPTION_KEY", testDBEncryptionKey)
	os.Setenv("DB_HASH_PEPPER", testDBHashPepper)
	if key, err := base64.StdEncoding.DecodeString(testDBEncryptionKey); err == nil {
		reflect.ValueOf(&config.DBEncryptionKey).Elem().Set(reflect.ValueOf(key))
	}
	if pepper, err := base64.StdEncoding.DecodeString(testDBHashPepper); err == nil {
		reflect.ValueOf(&config.DBHashPepper).Elem().Set(reflect.ValueOf(pepper))
		db.SetHashPepper(pepper)
	}
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

func TestUsernameTombstone(t *testing.T) {
	resetSchema(t)

	u, err := CreateUser("tombstone", "+15559871001", "password1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := DeleteUser(u.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	taken, err := UsernameTaken("tombstone")
	if err != nil {
		t.Fatalf("UsernameTaken: %v", err)
	}
	if !taken {
		t.Fatal("expected deleted username to remain taken")
	}

	_, err = CreateUser("tombstone", "+15559871002", "password1")
	if !errors.Is(err, ErrUserAlreadyExists) {
		t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestPhoneHoldAndReuse(t *testing.T) {
	resetSchema(t)

	const phone = "+15559871010"
	u, err := CreateUser("holduser", phone, "password1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := DeleteUser(u.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	avail, err := CheckPhoneAvailability(phone, 0)
	if err != nil {
		t.Fatalf("availability: %v", err)
	}
	if avail.Status != PhoneHeld {
		t.Fatalf("status = %v, want Held", avail.Status)
	}
	if avail.DaysRemaining < 1 || avail.DaysRemaining > 10 {
		t.Fatalf("DaysRemaining = %d, want 1..10", avail.DaysRemaining)
	}

	_, err = CreateUser("newholduser", phone, "password1")
	var holdErr *PhoneHoldError
	if !errors.As(err, &holdErr) {
		t.Fatalf("expected PhoneHoldError, got %v", err)
	}

	_, err = db.Exec(`
		UPDATE users SET deleted_at = $1 WHERE id = $2
	`, time.Now().UTC().Add(-11*24*time.Hour), u.ID)
	if err != nil {
		t.Fatalf("backdate deleted_at: %v", err)
	}

	avail, err = CheckPhoneAvailability(phone, 0)
	if err != nil {
		t.Fatalf("availability after hold: %v", err)
	}
	if avail.Status != PhoneAvailable {
		t.Fatalf("status = %v, want Available", avail.Status)
	}

	u2, err := CreateUser("newholduser", phone, "password1")
	if err != nil {
		t.Fatalf("reuse phone after hold: %v", err)
	}
	if u2.PhoneE64 != phone {
		t.Fatalf("phone = %q", u2.PhoneE64)
	}
}

func TestUpdatePhoneGuards(t *testing.T) {
	resetSchema(t)

	alice, err := CreateUser("alicephone", "+15559871101", "password1")
	if err != nil {
		t.Fatalf("create alice: %v", err)
	}
	bob, err := CreateUser("bobphone", "+15559871102", "password1")
	if err != nil {
		t.Fatalf("create bob: %v", err)
	}

	if err := UpdatePhone(alice.ID, alice.PhoneE64); !errors.Is(err, ErrPhoneSame) {
		t.Fatalf("same phone: got %v, want ErrPhoneSame", err)
	}

	if err := UpdatePhone(alice.ID, bob.PhoneE64); !errors.Is(err, ErrUserAlreadyExists) {
		t.Fatalf("active phone: got %v, want ErrUserAlreadyExists", err)
	}

	const newPhone = "+15559871103"
	if err := UpdatePhone(alice.ID, newPhone); err != nil {
		t.Fatalf("update phone: %v", err)
	}
	updated, err := GetByID(alice.ID)
	if err != nil {
		t.Fatalf("get alice: %v", err)
	}
	if updated.PhoneE64 != newPhone {
		t.Fatalf("phone = %q, want %q", updated.PhoneE64, newPhone)
	}

	// Old phone is immediately free after change.
	avail, err := CheckPhoneAvailability("+15559871101", 0)
	if err != nil {
		t.Fatalf("old phone availability: %v", err)
	}
	if avail.Status != PhoneAvailable {
		t.Fatalf("old phone status = %v, want Available", avail.Status)
	}

	if err := DeleteUser(bob.ID); err != nil {
		t.Fatalf("delete bob: %v", err)
	}
	avail, err = CheckPhoneAvailability(bob.PhoneE64, alice.ID)
	if err != nil {
		t.Fatalf("held phone: %v", err)
	}
	if avail.Status != PhoneHeld {
		t.Fatalf("status = %v, want Held", avail.Status)
	}
	err = UpdatePhone(alice.ID, bob.PhoneE64)
	var holdErr *PhoneHoldError
	if !errors.As(err, &holdErr) {
		t.Fatalf("expected PhoneHoldError, got %v", err)
	}
}

func TestPhoneActiveBlocks(t *testing.T) {
	resetSchema(t)

	_, err := CreateUser("activeone", "+15559871201", "password1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	avail, err := CheckPhoneAvailability("+15559871201", 0)
	if err != nil {
		t.Fatalf("availability: %v", err)
	}
	if avail.Status != PhoneActive {
		t.Fatalf("status = %v, want Active", avail.Status)
	}
	_, err = CreateUser("activetwo", "+15559871201", "password1")
	if !errors.Is(err, ErrUserAlreadyExists) {
		t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestSetSMSOptOutByPhoneE64(t *testing.T) {
	resetSchema(t)

	phone := "+15559871301"
	u, err := CreateUser("optoutuser", phone, "password1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.SMSOptedOut {
		t.Fatal("new user should not be opted out")
	}

	if err := SetSMSOptOutByPhoneE64(phone, true); err != nil {
		t.Fatalf("opt out: %v", err)
	}
	got, err := GetByID(u.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.SMSOptedOut {
		t.Fatal("expected opted out after STOP sync")
	}

	if err := SetSMSOptOutByPhoneE64(phone, false); err != nil {
		t.Fatalf("opt in: %v", err)
	}
	got, err = GetByID(u.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.SMSOptedOut {
		t.Fatal("expected opted in after START sync")
	}

	err = SetSMSOptOutByPhoneE64("+15559871399", true)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unknown phone: got %v, want sql.ErrNoRows", err)
	}
}
