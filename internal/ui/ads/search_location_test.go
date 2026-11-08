package ads

import (
	"bytes"
	"strings"
	"testing"
)

func TestLocationSummaryText(t *testing.T) {
	t.Run("no location", func(t *testing.T) {
		got := LocationSummaryText(SearchFilters{})
		if got != "No location set" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("with resolved display", func(t *testing.T) {
		got := LocationSummaryText(SearchFilters{
			Location:        "97345",
			LocationDisplay: "Corvallis, OR",
			Within:          50,
			WithinUnit:      "mi",
		})
		want := "Corvallis, OR - Within 50 miles"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
	t.Run("falls back to raw location", func(t *testing.T) {
		got := LocationSummaryText(SearchFilters{
			Location:   "Corvallis, Oregon",
			Within:     50,
			WithinUnit: "mi",
		})
		want := "Corvallis, Oregon - Within 50 miles"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}

func TestLocationSummaryPin(t *testing.T) {
	empty := renderLocationBar(t, SearchFilters{})
	if strings.Contains(empty, "location-pin") {
		t.Fatal("pin should be hidden when no location is set")
	}

	set := renderLocationBar(t, SearchFilters{
		Location:        "Philomath",
		LocationDisplay: "Philomath",
		Within:          100,
	})
	if !strings.Contains(set, "location-pin") {
		t.Fatal("expected a pin when location is set")
	}
	if !strings.Contains(set, "Philomath - Within 100 miles") {
		t.Fatalf("missing location label: %s", set)
	}
}

func renderLocationBar(t *testing.T, f SearchFilters) string {
	t.Helper()
	var buf bytes.Buffer
	if err := SearchLocationBar(f).Render(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
