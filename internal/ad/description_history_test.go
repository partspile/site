package ad

import (
	"strings"
	"testing"
	"time"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/entrylog"
	"github.com/partspile/site/internal/facet"
)

func TestSplitDescription(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	origAt := time.Date(2026, 1, 5, 12, 0, 0, 0, loc)
	editAt := time.Date(2026, 1, 7, 0, 34, 0, 0, loc)

	desc := WrapDescription("Triumph tr7 $3500 obo.", origAt, loc)
	desc = AppendHistoryEntry(
		desc, DescriptionAdditionLabel, "Has the 2.0L engine.", editAt, loc,
	)

	gotOrig, gotHist := SplitDescription(desc)
	if gotOrig != "Triumph tr7 $3500 obo." {
		t.Errorf("original = %q, want Triumph tr7 $3500 obo.", gotOrig)
	}
	if !strings.Contains(gotHist, DescriptionAdditionLabel) {
		t.Errorf("history = %q", gotHist)
	}
	if !strings.Contains(gotHist, entrylog.Marker) {
		t.Fatal("expected entry marker in history")
	}

	gotOrig, gotHist = SplitDescription(WrapDescription("no history here", origAt, loc))
	if gotOrig != "no history here" || gotHist != "" {
		t.Errorf("no history split failed: orig=%q hist=%q", gotOrig, gotHist)
	}
}

func TestAppendHistoryEntryPrependsNewest(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	origAt := time.Date(2026, 1, 5, 12, 0, 0, 0, loc)
	first := time.Date(2026, 1, 7, 0, 34, 0, 0, loc)
	second := time.Date(2026, 1, 7, 18, 47, 0, 0, loc)

	desc := WrapDescription("Original text.", origAt, loc)
	desc = AppendHistoryEntry(
		desc,
		DescriptionAdditionLabel,
		"First edit.",
		first,
		loc,
	)
	desc = AppendHistoryEntry(
		desc,
		"Price change",
		"Price dropped from $3,400 to $3,000",
		second,
		loc,
	)

	parts := ParseDescriptionForDisplay(desc)
	if parts.Original != "Original text." {
		t.Errorf("original = %q", parts.Original)
	}
	if parts.Body != "Original text." {
		t.Errorf("body = %q", parts.Body)
	}
	if len(parts.History) != 2 {
		t.Fatalf("got %d history entries, want 2", len(parts.History))
	}
	if !strings.Contains(parts.History[0].Header, "Price change") {
		t.Errorf("newest first: %q", parts.History[0].Header)
	}
	if !strings.Contains(parts.History[1].Header, DescriptionAdditionLabel) {
		t.Errorf("oldest second: %q", parts.History[1].Header)
	}
}

func TestFoldDescriptionAdditions(t *testing.T) {
	loc := time.UTC
	origAt := time.Date(2026, 1, 5, 12, 0, 0, 0, loc)
	first := time.Date(2026, 1, 7, 0, 34, 0, 0, loc)
	second := time.Date(2026, 1, 8, 10, 0, 0, 0, loc)

	desc := WrapDescription("Original.", origAt, loc)
	desc = AppendHistoryEntry(desc, DescriptionAdditionLabel, "asdfasdf", first, loc)
	desc = AppendHistoryEntry(desc, "Price change", "now free", first, loc)
	desc = AppendHistoryEntry(desc, DescriptionAdditionLabel, "more info", second, loc)

	got, changed := FoldDescriptionAdditions(desc)
	if !changed {
		t.Fatal("expected fold to change description")
	}
	parts := ParseDescriptionForDisplay(got)
	if parts.Original != "Original.\n\nasdfasdf\n\nmore info" {
		t.Errorf("original = %q", parts.Original)
	}
	if parts.Body != parts.Original {
		t.Errorf("body = %q, want original", parts.Body)
	}
	if len(parts.History) != 1 {
		t.Fatalf("history len = %d, want 1", len(parts.History))
	}
	if !strings.Contains(parts.History[0].Header, "Price change") {
		t.Errorf("kept history = %q", parts.History[0].Header)
	}
	if _, changed := FoldDescriptionAdditions(got); changed {
		t.Fatal("second fold should be a no-op")
	}
}

func TestFoldDescriptionAdditionsEmptyOriginal(t *testing.T) {
	loc := time.UTC
	origAt := time.Date(2026, 1, 5, 12, 0, 0, 0, loc)
	addAt := time.Date(2026, 1, 7, 0, 34, 0, 0, loc)

	desc := WrapDescription("", origAt, loc)
	desc = AppendHistoryEntry(
		desc, DescriptionAdditionLabel, "only addition", addAt, loc,
	)
	got, changed := FoldDescriptionAdditions(desc)
	if !changed {
		t.Fatal("expected fold")
	}
	parts := ParseDescriptionForDisplay(got)
	if parts.Original != "only addition" {
		t.Errorf("original = %q", parts.Original)
	}
	if len(parts.History) != 0 {
		t.Fatalf("history = %+v", parts.History)
	}
}

func TestFoldDescriptionAdditionsNoOp(t *testing.T) {
	desc := WrapDescription("no additions", time.Now(), time.UTC)
	got, changed := FoldDescriptionAdditions(desc)
	if changed || got != desc {
		t.Fatalf("changed=%v got=%q", changed, got)
	}
}

func TestAppendHistoryEntry(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	origAt := time.Date(2026, 1, 5, 12, 0, 0, 0, loc)
	at := time.Date(2026, 1, 7, 0, 34, 0, 0, loc)

	desc := WrapDescription("Original text.", origAt, loc)
	desc = AppendHistoryEntry(
		desc,
		DescriptionAdditionLabel,
		"Has the 2.0L engine.",
		at,
		loc,
	)
	if !strings.Contains(desc, entrylog.Marker) {
		t.Fatal("expected history marker in stored description")
	}
	orig, _ := SplitDescription(desc)
	if orig != "Original text." {
		t.Fatalf("original portion = %q", orig)
	}
	display := DisplayDescription(desc)
	if strings.Contains(display, entrylog.Marker) {
		t.Fatal("marker visible after DisplayDescription")
	}
	if !strings.Contains(display, "1/7/2026 12:34 am  Description Addition") {
		t.Errorf("display missing header: %q", display)
	}
	if !strings.Contains(display, "Has the 2.0L engine.") {
		t.Errorf("display missing body: %q", display)
	}
}

func TestBuildFieldChangeEntries(t *testing.T) {
	oldAmt, newAmt := 3400, 3000
	code := "USD"
	oldAd := Ad{
		Title: "Old title",
		Facets: map[string]facet.Value{
			"price": {Num: &oldAmt, Text: &code},
		},
	}
	newFacets := map[string]facet.Value{
		"price": {Num: &newAmt, Text: &code},
	}
	cat := Category{FacetKeys: []string{"price"}}

	entries := BuildFieldChangeEntries(
		oldAd, "New title", "", newFacets, cat,
	)
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].label != "Title change" {
		t.Errorf("first label = %q", entries[0].label)
	}
	if entries[1].label != "Price change" {
		t.Errorf("second label = %q", entries[1].label)
	}
	if !strings.Contains(entries[1].body, "Price dropped") {
		t.Errorf("price body = %q", entries[1].body)
	}
}

func TestFormatLocationHistoryChangeNoOp(t *testing.T) {
	old := Ad{
		RawLocation: "97333",
		City:        "Corvallis",
		AdminArea:   "OR",
		Country:     "US",
	}
	if body := formatLocationHistoryChange(old, "97333", Category{}); body != "" {
		t.Fatalf("want no change, got %q", body)
	}
}

func TestSanitizeStripsHistoryMarker(t *testing.T) {
	if got := SanitizeAdText("before\u001eafter"); got != "beforeafter" {
		t.Errorf("SanitizeAdText = %q", got)
	}
	if got := SanitizeAdText("before\u001fafter"); got != "beforeafter" {
		t.Errorf("SanitizeAdText end marker = %q", got)
	}
}

func TestFormatLocationHistoryChangeAddressCategory(t *testing.T) {
	addr := "123 Main St"
	old := Ad{
		Facets: map[string]facet.Value{
			"address": {Text: &addr},
		},
	}
	garage := Category{FacetKeys: []string{"address"}}

	if body := formatLocationHistoryChange(old, "456 Oak Ave", garage); body != "Address changed" {
		t.Fatalf("change = %q, want Address changed", body)
	}
	if body := formatLocationHistoryChange(old, "", garage); body != "Address removed" {
		t.Fatalf("remove = %q, want Address removed", body)
	}
	if body := formatLocationHistoryChange(Ad{}, "123 Main St", garage); body != "Address changed" {
		t.Fatalf("set = %q, want Address changed", body)
	}
	if body := formatLocationHistoryChange(old, addr, garage); body != "" {
		t.Fatalf("no-op = %q, want empty", body)
	}
}

func TestApplyDescriptionEditReplacesBodyAndJournalsSummary(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	origAt := time.Date(2026, 1, 5, 12, 0, 0, 0, loc)
	editAt := time.Date(2026, 1, 7, 0, 34, 0, 0, loc)

	orig := summarizeDescChangeFn
	summarizeDescChangeFn = func(previous, current string) string {
		if previous != "Original text." || current != "Updated text." {
			t.Errorf("summarize args previous=%q current=%q", previous, current)
		}
		return "fixed some typos"
	}
	t.Cleanup(func() { summarizeDescChangeFn = orig })

	desc := WrapDescription("Original text.", origAt, loc)
	desc, err := applyDescriptionEdit(desc, "Updated text.", editAt, loc)
	if err != nil {
		t.Fatal(err)
	}
	parts := ParseDescriptionForDisplay(desc)
	if parts.Original != "Updated text." {
		t.Errorf("original = %q", parts.Original)
	}
	if parts.Body != "Updated text." {
		t.Errorf("body = %q", parts.Body)
	}
	if len(parts.History) != 1 {
		t.Fatalf("got %d history entries, want 1", len(parts.History))
	}
	if !strings.Contains(parts.History[0].Header, DescriptionChangeLabel) {
		t.Errorf("header = %q", parts.History[0].Header)
	}
	if parts.History[0].Body != "fixed some typos" {
		t.Errorf("summary = %q", parts.History[0].Body)
	}
}

func TestApplyDescriptionEditNoOpWhenUnchanged(t *testing.T) {
	loc := time.UTC
	origAt := time.Date(2026, 1, 5, 12, 0, 0, 0, loc)
	desc := WrapDescription("Same text.", origAt, loc)

	called := false
	orig := summarizeDescChangeFn
	summarizeDescChangeFn = func(string, string) string {
		called = true
		return "should not run"
	}
	t.Cleanup(func() { summarizeDescChangeFn = orig })

	got, err := applyDescriptionEdit(desc, "Same text.", origAt, loc)
	if err != nil {
		t.Fatal(err)
	}
	if got != desc {
		t.Errorf("expected unchanged description")
	}
	if called {
		t.Error("summarizer should not run when text is unchanged")
	}
}

func TestApplyDescriptionEditKeepsOtherHistory(t *testing.T) {
	loc := time.UTC
	origAt := time.Date(2026, 1, 5, 12, 0, 0, 0, loc)
	priceAt := time.Date(2026, 1, 6, 12, 0, 0, 0, loc)
	editAt := time.Date(2026, 1, 7, 12, 0, 0, 0, loc)

	orig := summarizeDescChangeFn
	summarizeDescChangeFn = func(previous, current string) string {
		if previous != "Original." {
			t.Errorf("previous = %q", previous)
		}
		if current != "Rewritten listing with service records." {
			t.Errorf("current = %q", current)
		}
		return "added service record details"
	}
	t.Cleanup(func() { summarizeDescChangeFn = orig })

	desc := WrapDescription("Original.", origAt, loc)
	desc = AppendHistoryEntry(
		desc, "Price change", "Price dropped from $3,400 to $3,000",
		priceAt, loc,
	)
	desc, err := applyDescriptionEdit(
		desc, "Rewritten listing with service records.", editAt, loc,
	)
	if err != nil {
		t.Fatal(err)
	}
	parts := ParseDescriptionForDisplay(desc)
	if parts.Original != "Rewritten listing with service records." {
		t.Errorf("original = %q", parts.Original)
	}
	if len(parts.History) != 2 {
		t.Fatalf("history = %+v", parts.History)
	}
	if !strings.Contains(parts.History[0].Header, DescriptionChangeLabel) {
		t.Errorf("newest = %q", parts.History[0].Header)
	}
	if parts.History[0].Body != "added service record details" {
		t.Errorf("summary = %q", parts.History[0].Body)
	}
	if !strings.Contains(parts.History[1].Header, "Price change") {
		t.Errorf("kept = %q", parts.History[1].Header)
	}
}

func TestApplyDescriptionEditRejectsTooLong(t *testing.T) {
	loc := time.UTC
	desc := WrapDescription("short", time.Now(), loc)
	long := strings.Repeat("x", config.MaxAdDescriptionLength+1)
	_, err := applyDescriptionEdit(desc, long, time.Now(), loc)
	if err == nil {
		t.Fatal("expected length error")
	}
}

func TestImageIndicesFromHistoryEntry(t *testing.T) {
	indices := imageIndicesFromHistoryEntry(
		"1/7/2026 12:34 am  Images Added", "3,4,5",
	)
	if len(indices) != 3 || indices[0] != 3 || indices[2] != 5 {
		t.Fatalf("indices = %v", indices)
	}
	if imageIndicesFromHistoryEntry("Price change", "3,4") != nil {
		t.Fatal("expected nil for non-image entry")
	}
}

func TestAppendHistoryEntryImagesAdded(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	origAt := time.Date(2026, 1, 5, 12, 0, 0, 0, loc)
	at := time.Date(2026, 1, 7, 0, 34, 0, 0, loc)

	desc := WrapDescription("Original.", origAt, loc)
	desc = AppendHistoryEntry(
		desc, imagesAddedLabel, "2,3", at, loc,
	)
	parts := ParseDescriptionForDisplay(desc)
	if len(parts.History) != 1 {
		t.Fatalf("got %d entries", len(parts.History))
	}
	if len(parts.History[0].ImageIndices) != 2 {
		t.Fatalf("indices = %v", parts.History[0].ImageIndices)
	}
	if parts.History[0].Body != "" {
		t.Fatalf("body should be hidden: %q", parts.History[0].Body)
	}
}
