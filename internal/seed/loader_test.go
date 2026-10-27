package seed

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/entrylog"
)

func TestMain(m *testing.M) {
	if err := chdirModuleRoot(); err != nil {
		fmt.Fprintf(os.Stderr, "chdir module root: %v\n", err)
		os.Exit(1)
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

func TestAssembleDescription(t *testing.T) {
	createdAt := time.Date(2024, 7, 4, 0, 0, 0, 0, time.UTC)
	history := []HistoryEntryJSON{
		{
			Label: "Description Addition",
			Body:  "Available for local pickup only.",
			At:    "2024-09-12T15:30:00-04:00",
		},
	}
	got, err := AssembleDescription("Original body.", history, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, entrylog.Marker) {
		t.Fatal("expected entry marker")
	}
	if !strings.Contains(got, ad.OriginalLabel) {
		t.Fatal("expected original entry label")
	}
	parts := ad.ParseDescriptionForDisplay(got)
	if parts.Original != "Original body." {
		t.Errorf("original = %q", parts.Original)
	}
	if len(parts.History) != 1 {
		t.Fatalf("got %d history entries", len(parts.History))
	}
}

func TestPartsCategoriesIncludeLocalPickup(t *testing.T) {
	cats, err := readCategoryJSON()
	if err != nil {
		t.Fatal(err)
	}
	wantPickup := map[string]bool{
		"Car & Truck Parts":            true,
		"Motorcycle Parts":             true,
		"Bicycle Parts":                true,
		"Agricultural Equipment Parts": true,
	}
	found := make(map[string]bool)
	for _, cat := range cats {
		hasPickup := false
		for _, key := range cat.Facets {
			if key == "local_pickup" {
				hasPickup = true
				break
			}
		}
		if want, ok := wantPickup[cat.Name]; ok {
			found[cat.Name] = true
			if hasPickup != want {
				t.Errorf("%s local_pickup=%v, want %v",
					cat.Name, hasPickup, want)
			}
			continue
		}
		if hasPickup {
			t.Errorf("%s should not have local_pickup facet", cat.Name)
		}
	}
	for name := range wantPickup {
		if !found[name] {
			t.Errorf("missing category %s", name)
		}
	}
}

func TestSeedTagsJSON(t *testing.T) {
	aj := adJSON{
		Suggestions: []SuggestionJSON{
			{Label: "OEM", Value: "yes"},
			{Label: "side", Value: "left"},
		},
	}
	raw := ad.TagsJSON(seedSuggestions(aj))
	if raw == "[]" {
		t.Fatal("expected tags json")
	}
	if !strings.Contains(raw, "OEM") {
		t.Fatalf("unexpected json: %s", raw)
	}
}
