package flag_test

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/db/testdb"
	"github.com/partspile/site/internal/dbinit"
	"github.com/partspile/site/internal/encryption"
	"github.com/partspile/site/internal/facet"
	"github.com/partspile/site/internal/flag"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/user"
)

const testDBEncryptionKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
const testDBHashPepper = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="

func TestMain(m *testing.M) {
	if err := chdirModuleRoot(); err != nil {
		panic(err)
	}
	testURL := testdb.PackageDatabaseURL("flag")
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
	if err := dbinit.LoadCategories(); err != nil {
		t.Fatalf("load categories: %v", err)
	}
	if err := ad.LoadCategories(); err != nil {
		t.Fatalf("cache categories: %v", err)
	}
}

func adFlagCount(t *testing.T, adID int) int {
	t.Helper()
	var n int
	err := db.QueryRow(`SELECT flag_count FROM ads WHERE id = $1`, adID).Scan(&n)
	if err != nil {
		t.Fatalf("query flag_count: %v", err)
	}
	return n
}

func createConv(t *testing.T, adID, ownerID, inquirerID int) int {
	t.Helper()
	var convID int
	err := db.QueryRow(`
		INSERT INTO conversations (
			ad_id, owner_id, inquirer_id, journal
		) VALUES ($1, $2, $3, '')
		RETURNING id`,
		adID, ownerID, inquirerID,
	).Scan(&convID)
	if err != nil {
		t.Fatalf("conversation: %v", err)
	}
	sealed, err := encryption.Seal(convID, "", config.DBEncryptionKey)
	if err != nil {
		t.Fatalf("seal journal: %v", err)
	}
	_, err = db.Exec(
		`UPDATE conversations SET journal = $1 WHERE id = $2`,
		sealed, convID,
	)
	if err != nil {
		t.Fatalf("store journal: %v", err)
	}
	return convID
}

func createPricedAd(t *testing.T, userID int, title string) int {
	t.Helper()
	price := 100
	currency := "USD"
	adID, err := ad.CreateAd(ad.CreateInput{
		CategoryID:  5,
		UserID:      userID,
		Title:       title,
		Description: "test",
		Facets: map[string]facet.Value{
			"price": {Num: &price, Text: &currency},
		},
	})
	if err != nil {
		t.Fatalf("create ad: %v", err)
	}
	return adID
}

func TestAdFlagCountSyncOnThrowLower(t *testing.T) {
	resetSchema(t)

	owner, err := user.CreateUser("flagowner", "+15559873001", "password1")
	if err != nil {
		t.Fatalf("owner: %v", err)
	}
	buyer, err := user.CreateUser("flagbuyer", "+15559873002", "password1")
	if err != nil {
		t.Fatalf("buyer: %v", err)
	}
	adID := createPricedAd(t, owner.ID, "Flag Count Part")
	if adFlagCount(t, adID) != 0 {
		t.Fatalf("initial flag_count = %d", adFlagCount(t, adID))
	}

	convID := createConv(t, adID, owner.ID, buyer.ID)
	if err := flag.RaiseFlag(buyer.ID, convID, flag.ReasonPolicy); err != nil {
		t.Fatalf("throw: %v", err)
	}
	if got := adFlagCount(t, adID); got != 1 {
		t.Fatalf("after throw flag_count = %d, want 1", got)
	}

	if err := flag.LowerFlag(buyer.ID, convID); err != nil {
		t.Fatalf("lower: %v", err)
	}
	if got := adFlagCount(t, adID); got != 0 {
		t.Fatalf("after lower flag_count = %d, want 0", got)
	}
}

func TestOwnerThrowDoesNotChangeAdFlagCount(t *testing.T) {
	resetSchema(t)

	owner, err := user.CreateUser("flagowner2", "+15559873003", "password1")
	if err != nil {
		t.Fatalf("owner: %v", err)
	}
	buyer, err := user.CreateUser("flagbuyer2", "+15559873004", "password1")
	if err != nil {
		t.Fatalf("buyer: %v", err)
	}
	adID := createPricedAd(t, owner.ID, "Owner Flag Part")
	convID := createConv(t, adID, owner.ID, buyer.ID)

	if err := flag.RaiseFlag(owner.ID, convID, flag.ReasonConduct); err != nil {
		t.Fatalf("owner throw: %v", err)
	}
	if got := adFlagCount(t, adID); got != 0 {
		t.Fatalf("owner throw should not bump ad flag_count, got %d", got)
	}
}
