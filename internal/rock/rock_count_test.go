package rock_test

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
	"github.com/partspile/site/internal/encryption"
	"github.com/partspile/site/internal/facet"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/rock"
	"github.com/partspile/site/internal/seed"
	"github.com/partspile/site/internal/user"
)

const testDBEncryptionKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
const testDBHashPepper = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="

func TestMain(m *testing.M) {
	if err := chdirModuleRoot(); err != nil {
		panic(err)
	}
	testURL := testdb.PackageDatabaseURL("rock")
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
	if err := seed.LoadAll(); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := ad.LoadCategories(); err != nil {
		t.Fatalf("load categories: %v", err)
	}
}

func adRockCount(t *testing.T, adID int) int {
	t.Helper()
	var n int
	err := db.QueryRow(`SELECT rock_count FROM ads WHERE id = $1`, adID).Scan(&n)
	if err != nil {
		t.Fatalf("query rock_count: %v", err)
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

func TestAdRockCountSyncOnThrowUnthrow(t *testing.T) {
	resetSchema(t)

	owner, err := user.CreateUser("rockowner", "+15559873001", "password1")
	if err != nil {
		t.Fatalf("owner: %v", err)
	}
	buyer, err := user.CreateUser("rockbuyer", "+15559873002", "password1")
	if err != nil {
		t.Fatalf("buyer: %v", err)
	}
	adID := createPricedAd(t, owner.ID, "Rock Count Part")
	if adRockCount(t, adID) != 0 {
		t.Fatalf("initial rock_count = %d", adRockCount(t, adID))
	}

	convID := createConv(t, adID, owner.ID, buyer.ID)
	if err := rock.ThrowRock(buyer.ID, convID, rock.ReasonPolicy); err != nil {
		t.Fatalf("throw: %v", err)
	}
	if got := adRockCount(t, adID); got != 1 {
		t.Fatalf("after throw rock_count = %d, want 1", got)
	}

	if err := rock.UnthrowRock(buyer.ID, convID); err != nil {
		t.Fatalf("unthrow: %v", err)
	}
	if got := adRockCount(t, adID); got != 0 {
		t.Fatalf("after unthrow rock_count = %d, want 0", got)
	}
}

func TestOwnerThrowDoesNotChangeAdRockCount(t *testing.T) {
	resetSchema(t)

	owner, err := user.CreateUser("rockowner2", "+15559873003", "password1")
	if err != nil {
		t.Fatalf("owner: %v", err)
	}
	buyer, err := user.CreateUser("rockbuyer2", "+15559873004", "password1")
	if err != nil {
		t.Fatalf("buyer: %v", err)
	}
	adID := createPricedAd(t, owner.ID, "Owner Rock Part")
	convID := createConv(t, adID, owner.ID, buyer.ID)

	if err := rock.ThrowRock(owner.ID, convID, rock.ReasonConduct); err != nil {
		t.Fatalf("owner throw: %v", err)
	}
	if got := adRockCount(t, adID); got != 0 {
		t.Fatalf("owner throw should not bump ad rock_count, got %d", got)
	}
}
