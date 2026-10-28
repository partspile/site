package ad_test

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/db/testdb"
	"github.com/partspile/site/internal/dbinit"
	"github.com/partspile/site/internal/facet"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/user"
)

const testDBEncryptionKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
const testDBHashPepper = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="

func TestMain(m *testing.M) {
	if err := chdirModuleRoot(); err != nil {
		panic(err)
	}
	testURL := testdb.PackageDatabaseURL("ad")
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

func resetAdDB(t *testing.T) {
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

func createTestAd(t *testing.T, userID int, title string) int {
	t.Helper()
	price := 1000
	currency := "USD"
	adID, err := ad.CreateAd(ad.CreateInput{
		CategoryID:  5,
		UserID:      userID,
		Title:       title,
		Description: "desc",
		Facets: map[string]facet.Value{
			"price": {Num: &price, Text: &currency},
		},
	})
	if err != nil {
		t.Fatalf("create ad %q: %v", title, err)
	}
	return adID
}

func TestActivateFreshExpiresAtKeepsCreatedAt(t *testing.T) {
	resetAdDB(t)

	u, err := user.CreateUser("expireowner", "+15559872001", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	adID := createTestAd(t, u.ID, "Activate fresh")

	oldCreated := time.Now().UTC().AddDate(0, -1, 0)
	_, err = db.Exec(`UPDATE ads SET created_at = $1 WHERE id = $2`,
		oldCreated, adID)
	if err != nil {
		t.Fatalf("backdate created_at: %v", err)
	}
	if err := ad.Pause(adID); err != nil {
		t.Fatalf("pause: %v", err)
	}

	activateAt := time.Now().UTC()
	if err := ad.Activate(adID); err != nil {
		t.Fatalf("activate: %v", err)
	}

	a, err := ad.GetAd(u.ID, adID, time.UTC)
	if err != nil {
		t.Fatalf("get ad: %v", err)
	}
	if !a.IsActive() {
		t.Fatal("expected ad to be active")
	}
	if a.CreatedAt.UTC().Sub(oldCreated).Abs() > time.Second {
		t.Fatalf("created_at changed: got %v, want ~%v",
			a.CreatedAt, oldCreated)
	}
	wantExpires := activateAt.AddDate(0, config.AdExpireMonths, 0)
	if a.ExpiresAt.UTC().Sub(wantExpires).Abs() > 2*time.Second {
		t.Fatalf("expires_at = %v, want ~%v", a.ExpiresAt, wantExpires)
	}
}

func TestRenewExtendsExpiresAt(t *testing.T) {
	resetAdDB(t)

	u, err := user.CreateUser("expirerenew", "+15559872005", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	adID := createTestAd(t, u.ID, "Renew me")

	soon := time.Now().UTC().AddDate(0, 0, 10)
	_, err = db.Exec(`UPDATE ads SET expires_at = $1 WHERE id = $2`, soon, adID)
	if err != nil {
		t.Fatalf("set expires_at: %v", err)
	}

	renewAt := time.Now().UTC()
	if err := ad.Renew(adID); err != nil {
		t.Fatalf("renew: %v", err)
	}

	a, err := ad.GetAd(u.ID, adID, time.UTC)
	if err != nil {
		t.Fatalf("get ad: %v", err)
	}
	wantExpires := renewAt.AddDate(0, config.AdExpireMonths, 0)
	if a.ExpiresAt.UTC().Sub(wantExpires).Abs() > 2*time.Second {
		t.Fatalf("expires_at = %v, want ~%v", a.ExpiresAt, wantExpires)
	}
}

func TestRenewRejectsWhenNotEligible(t *testing.T) {
	resetAdDB(t)

	u, err := user.CreateUser("expirerenew2", "+15559872006", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	adID := createTestAd(t, u.ID, "Too fresh")

	if err := ad.Renew(adID); err == nil {
		t.Fatal("expected renew to fail for fresh 3-month expires_at")
	}
}

func TestListAdsDueToExpire(t *testing.T) {
	resetAdDB(t)

	u, err := user.CreateUser("expirelist", "+15559872002", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	freshID := createTestAd(t, u.ID, "Fresh ad")
	staleID := createTestAd(t, u.ID, "Stale ad")

	past := time.Now().UTC().Add(-time.Hour)
	_, err = db.Exec(`UPDATE ads SET expires_at = $1 WHERE id = $2`,
		past, staleID)
	if err != nil {
		t.Fatalf("backdate stale expires_at: %v", err)
	}

	due, err := ad.ListAdsDueToExpire()
	if err != nil {
		t.Fatalf("list due: %v", err)
	}
	foundStale := false
	for _, row := range due {
		if row.ID == freshID {
			t.Fatal("fresh ad should not be due to expire")
		}
		if row.ID == staleID {
			foundStale = true
			if row.UserID != u.ID {
				t.Fatalf("user_id = %d, want %d", row.UserID, u.ID)
			}
		}
	}
	if !foundStale {
		t.Fatal("expected stale ad in due list")
	}

	if err := ad.Pause(staleID); err != nil {
		t.Fatalf("pause stale: %v", err)
	}
	due, err = ad.ListAdsDueToExpire()
	if err != nil {
		t.Fatalf("list due after pause: %v", err)
	}
	for _, row := range due {
		if row.ID == staleID {
			t.Fatal("paused ad should not be due to expire")
		}
	}
}

func TestCreateAdSetsExpiresAt(t *testing.T) {
	resetAdDB(t)
	u, err := user.CreateUser("expirecreate", "+15559872003", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	before := time.Now().UTC()
	adID := createTestAd(t, u.ID, "Expire create")
	a, err := ad.GetAd(u.ID, adID, time.UTC)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	want := before.AddDate(0, config.AdExpireMonths, 0)
	if a.ExpiresAt.UTC().Sub(want).Abs() > 2*time.Second {
		t.Fatalf("expires_at = %v, want ~%v", a.ExpiresAt, want)
	}
}

func TestCreateGarageSaleExpiresFromSaleEnd(t *testing.T) {
	resetAdDB(t)
	u, err := user.CreateUser("expiregarage", "+15559872004", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	catID, err := ad.GetCategoryIDByName("Garage/Estate Sale")
	if err != nil {
		t.Fatalf("garage category: %v", err)
	}
	saleType := "Garage Sale"
	start := "2026-08-01"
	end := "2026-08-02"
	addr := "123 Main St, Ann Arbor, MI"
	adID, err := ad.CreateAd(ad.CreateInput{
		CategoryID:  catID,
		UserID:      u.ID,
		Title:       "Yard sale expire test",
		Description: "desc",
		Facets: map[string]facet.Value{
			"sale_type":       {Text: &saleType},
			"sale_start_date": {Text: &start},
			"sale_end_date":   {Text: &end},
			"address":         {Text: &addr},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	a, err := ad.GetAd(u.ID, adID, time.UTC)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	want, err := ad.ExpiresAtFromSaleEnd(end)
	if err != nil {
		t.Fatal(err)
	}
	if !a.ExpiresAt.UTC().Equal(want) {
		t.Fatalf("expires_at = %v, want %v", a.ExpiresAt, want)
	}
}
