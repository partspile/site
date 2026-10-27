package backup_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/backup"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/db/testdb"
	"github.com/partspile/site/internal/encryption"
	"github.com/partspile/site/internal/facet"
	"github.com/partspile/site/internal/imagestore"
	"github.com/partspile/site/internal/journal"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/message"
	"github.com/partspile/site/internal/seed"
	"github.com/partspile/site/internal/user"
	"github.com/partspile/site/internal/vector"
	"github.com/pgvector/pgvector-go"
)

const keyA = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
const keyB = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE="
const testDBHashPepper = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="

func TestMain(m *testing.M) {
	if err := chdirModuleRoot(); err != nil {
		panic(err)
	}
	testURL := testdb.PackageDatabaseURL("backup")
	if err := testdb.EnsureDatabase(testURL); err != nil {
		panic(err)
	}
	os.Setenv("DATABASE_URL", testURL)
	os.Setenv("DB_ENCRYPTION_KEY", keyA)
	os.Setenv("DB_HASH_PEPPER", testDBHashPepper)
	setDBKey(keyA)
	setHashPepper(testDBHashPepper)
	if err := logger.Init("error", "text", os.DevNull); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func setDBKey(b64 string) {
	key, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		panic(err)
	}
	reflect.ValueOf(&config.DBEncryptionKey).Elem().Set(reflect.ValueOf(key))
}

func setHashPepper(b64 string) {
	pepper, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		panic(err)
	}
	reflect.ValueOf(&config.DBHashPepper).Elem().Set(reflect.ValueOf(pepper))
	db.SetHashPepper(pepper)
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

func TestBackupRestoreCrossKey(t *testing.T) {
	if err := testdb.InitSchema(); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	if err := seed.LoadAll(); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := ad.LoadCategories(); err != nil {
		t.Fatalf("load categories: %v", err)
	}

	setDBKey(keyA)
	alice, err := user.CreateUser("alice", "+15559876543", "secret")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	bob, err := user.CreateUser("bob", "+15559876544", "secret")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	price := 12000
	currency := "USD"
	adID, err := ad.CreateAd(ad.CreateInput{
		CategoryID:  5,
		UserID:      alice.ID,
		Title:       "Backup Test Car",
		Description: "Round trip restore test",
		Facets: map[string]facet.Value{
			"price": {Num: &price, Text: &currency},
		},
		ImageCount: 1,
	})
	if err != nil {
		t.Fatalf("create ad: %v", err)
	}

	wantEmb := make([]float32, config.OllamaEmbeddingDimensions)
	wantEmb[0] = 1
	wantMeta := map[string]any{"category_id": 5}
	if err := vector.UpsertAdEmbeddings(
		[]int{adID}, [][]float32{wantEmb}, []map[string]any{wantMeta},
	); err != nil {
		t.Fatalf("upsert embedding: %v", err)
	}

	var convID int
	err = db.QueryRow(`
		INSERT INTO conversations (
			ad_id, owner_id, inquirer_id, inquirer_has_unread, journal
		) VALUES ($1, $2, $3, 1, '')
		RETURNING id`,
		adID, alice.ID, bob.ID,
	).Scan(&convID)
	if err != nil {
		t.Fatalf("insert conversation: %v", err)
	}
	_, err = message.CreateMessage(convID, bob.ID, "Is this still available?")
	if err != nil {
		t.Fatalf("create message: %v", err)
	}

	imageDir := t.TempDir()
	store := imagestore.NewLocal(imageDir)
	var jpegBuf bytes.Buffer
	px := image.NewRGBA(image.Rect(0, 0, 1, 1))
	if err := jpeg.Encode(&jpegBuf, px, nil); err != nil {
		t.Fatalf("encode test jpeg: %v", err)
	}
	imageData := jpegBuf.Bytes()
	if err := store.Put(adID, 1, "480w", imageData); err != nil {
		t.Fatalf("put image: %v", err)
	}
	if err := store.PutUserAccount(alice.ID, imageData); err != nil {
		t.Fatalf("put account picture: %v", err)
	}
	if err := user.ConfirmAccountPicture(alice.ID); err != nil {
		t.Fatalf("confirm account picture: %v", err)
	}

	outDir := t.TempDir()
	archive := filepath.Join(outDir, "cross.tar.gz")
	path, err := backup.BackupToArchive(archive, store, false, false)
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("archive missing: %v", err)
	}

	keyABytes, _ := base64.StdEncoding.DecodeString(keyA)
	setDBKey(keyB)
	restoreStore := imagestore.NewLocal(t.TempDir())
	if err := backup.RestoreFromArchive(path, restoreStore, keyABytes, false, false); err != nil {
		t.Fatalf("restore: %v", err)
	}

	var restoredID int
	var expiresAt time.Time
	err = db.QueryRow(`
		SELECT id, expires_at
		FROM ads WHERE title = $1`, "Backup Test Car",
	).Scan(&restoredID, &expiresAt)
	if err != nil {
		t.Fatalf("query restored ad: %v", err)
	}
	if expiresAt.IsZero() {
		t.Fatal("expected restored expires_at")
	}

	var gotVec pgvector.Vector
	var gotMeta string
	err = db.QueryRow(`
		SELECT embedding, vector_metadata::text
		FROM ads WHERE id = $1`, restoredID,
	).Scan(&gotVec, &gotMeta)
	if err != nil {
		t.Fatalf("query restored embedding: %v", err)
	}
	gotSlice := gotVec.Slice()
	if len(gotSlice) != len(wantEmb) {
		t.Fatalf("embedding len = %d, want %d", len(gotSlice), len(wantEmb))
	}
	for i := range wantEmb {
		if gotSlice[i] != wantEmb[i] {
			t.Fatalf("embedding[%d] = %v, want %v", i, gotSlice[i], wantEmb[i])
		}
	}
	if !strings.Contains(gotMeta, `"category_id": 5`) &&
		!strings.Contains(gotMeta, `"category_id":5`) {
		t.Fatalf("vector_metadata = %q", gotMeta)
	}

	var restoredRockCount int
	err = db.QueryRow(
		`SELECT rock_count FROM ads WHERE id = $1`, restoredID,
	).Scan(&restoredRockCount)
	if err != nil {
		t.Fatalf("query restored rock_count: %v", err)
	}
	if restoredRockCount != 0 {
		t.Fatalf("rock_count = %d, want 0", restoredRockCount)
	}

	restoredAlice, err := user.GetByName("alice")
	if err != nil {
		t.Fatalf("get alice under key B: %v", err)
	}
	if restoredAlice.Name != "alice" {
		t.Fatalf("alice name = %q", restoredAlice.Name)
	}
	if !restoredAlice.HasAccountPicture {
		t.Fatal("expected restored alice account picture flag")
	}
	restoredAccount, err := restoreStore.GetUserAccount(restoredAlice.ID)
	if err != nil {
		t.Fatalf("get restored account picture: %v", err)
	}
	if string(restoredAccount) != string(imageData) {
		t.Fatal("restored account picture mismatch")
	}

	// Seed admin has no ads; backup must still include all users.
	admin, err := user.GetByName("admin")
	if err != nil {
		t.Fatalf("get seed admin after restore: %v", err)
	}
	if !admin.IsAdmin {
		t.Fatal("expected restored admin.is_admin")
	}
	if _, err := user.GetByName("test"); err != nil {
		t.Fatalf("get seed test after restore: %v", err)
	}

	var sealedJournal string
	err = db.QueryRow(
		`SELECT journal FROM conversations WHERE ad_id = $1`, restoredID,
	).Scan(&sealedJournal)
	if err != nil {
		t.Fatalf("query journal: %v", err)
	}
	var convID2 int
	err = db.QueryRow(
		`SELECT id FROM conversations WHERE ad_id = $1`, restoredID,
	).Scan(&convID2)
	if err != nil {
		t.Fatalf("query conv id: %v", err)
	}
	plain, err := encryption.Open(convID2, sealedJournal, config.DBEncryptionKey)
	if err != nil {
		t.Fatalf("open journal: %v", err)
	}
	content, _, ok := journal.LastMessagePreview(plain)
	if !ok {
		t.Fatal("expected message in restored journal")
	}
	if content != "Is this still available?" {
		t.Fatalf("message = %q", content)
	}

	restoredImage, err := restoreStore.Get(restoredID, 1, "480w")
	if err != nil {
		t.Fatalf("get restored image: %v", err)
	}
	if string(restoredImage) != string(imageData) {
		t.Fatal("restored image data mismatch")
	}
}

func TestDefaultArchivePath(t *testing.T) {
	ts := time.Date(2026, 7, 22, 17, 20, 45, 0, time.UTC)
	got := backup.DefaultArchivePath(ts)
	want := filepath.Join("backups", "backup-20260722-172045.tar.gz")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSealOpenRoundTrip(t *testing.T) {
	key, _ := base64.StdEncoding.DecodeString(keyA)
	sealed, err := encryption.Seal(42, "hello journal", key)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := encryption.Open(42, sealed, key)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "hello journal" {
		t.Fatalf("got %q", plain)
	}
	passthrough, err := encryption.Open(1, "not sealed", key)
	if err != nil {
		t.Fatal(err)
	}
	if passthrough != "not sealed" {
		t.Fatalf("got %q", passthrough)
	}
}
