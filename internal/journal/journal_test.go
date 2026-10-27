package journal

import (
	"strings"
	"testing"
	"time"

	"github.com/partspile/site/internal/entrylog"
)

func TestAppendAndParseMessage(t *testing.T) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 7, 10, 22, 33, 0, 0, loc)

	j := AppendMessage("", 12, "Is this still available?", at, loc)
	entries := Parse(j)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Kind != Message {
		t.Errorf("kind = %q", entries[0].Kind)
	}
	if entries[0].UserID != 12 {
		t.Errorf("user = %d", entries[0].UserID)
	}
	if entries[0].Body != "Is this still available?" {
		t.Errorf("body = %q", entries[0].Body)
	}
	if !entries[0].At.Equal(at) {
		t.Errorf("at = %v, want %v", entries[0].At, at)
	}
}

func TestAppendOldestFirst(t *testing.T) {
	loc := time.UTC
	t1 := time.Date(2026, 7, 10, 22, 33, 0, 0, loc)
	t2 := time.Date(2026, 7, 10, 22, 35, 0, 0, loc)
	t3 := time.Date(2026, 7, 10, 22, 40, 0, 0, loc)

	j := AppendMessage("", 12, "hello", t1, loc)
	j = AppendRock(j, RockThrown, 12, "policy", t2, loc)
	j = AppendRock(j, RockUnthrown, 12, "", t3, loc)

	entries := Parse(j)
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
	if entries[0].Kind != Message || entries[1].Kind != RockThrown ||
		entries[2].Kind != RockUnthrown {
		t.Fatalf("unexpected kinds: %+v", entries)
	}
	if entries[1].Body != "policy" {
		t.Errorf("thrown reason = %q", entries[1].Body)
	}
	if !strings.Contains(j, "message  sender:12") {
		t.Errorf("missing message meta: %q", j)
	}
	if !strings.Contains(j, "rock thrown  user:12") {
		t.Errorf("missing rock thrown: %q", j)
	}
}

func TestAppendAndParseClose(t *testing.T) {
	loc := time.UTC
	t1 := time.Date(2026, 7, 10, 22, 33, 0, 0, loc)
	t2 := time.Date(2026, 7, 10, 22, 40, 0, 0, loc)

	j := AppendMessage("", 12, "hello", t1, loc)
	j = AppendClose(j, AdDeleted, 34, t2, loc)

	entries := Parse(j)
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[1].Kind != AdDeleted || entries[1].UserID != 34 {
		t.Fatalf("close entry: %+v", entries[1])
	}
	if !strings.Contains(j, "ad deleted  user:34") {
		t.Errorf("missing ad deleted: %q", j)
	}
}

func TestAppendAndParsePause(t *testing.T) {
	loc := time.UTC
	t1 := time.Date(2026, 7, 10, 22, 33, 0, 0, loc)
	t2 := time.Date(2026, 7, 10, 22, 40, 0, 0, loc)
	t3 := time.Date(2026, 7, 10, 22, 45, 0, 0, loc)

	j := AppendMessage("", 12, "hello", t1, loc)
	j = AppendClose(j, AdPaused, 34, t2, loc)
	j = AppendClose(j, AdUnpaused, 34, t3, loc)

	entries := Parse(j)
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
	if entries[1].Kind != AdPaused || entries[2].Kind != AdUnpaused {
		t.Fatalf("kinds: %+v %+v", entries[1], entries[2])
	}
}

func TestLastMessagePreviewSkipsRock(t *testing.T) {
	loc := time.UTC
	t1 := time.Date(2026, 7, 10, 22, 33, 0, 0, loc)
	t2 := time.Date(2026, 7, 10, 22, 35, 0, 0, loc)
	t3 := time.Date(2026, 7, 10, 22, 40, 0, 0, loc)

	j := AppendMessage("", 1, "first", t1, loc)
	j = AppendMessage(j, 2, "second", t2, loc)
	j = AppendRock(j, RockThrown, 1, "deal", t3, loc)

	content, at, ok := LastMessagePreview(j)
	if !ok {
		t.Fatal("expected preview")
	}
	if content != "second" {
		t.Errorf("content = %q", content)
	}
	if !at.Equal(t2) {
		t.Errorf("at = %v", at)
	}
}

func TestSanitizeStripsMarkers(t *testing.T) {
	got := entrylog.SanitizeText("before\u001eafter\u001fend")
	if got != "beforeafterend" {
		t.Errorf("SanitizeText = %q", got)
	}
}

func TestFirstEntryAt(t *testing.T) {
	loc := time.UTC
	t1 := time.Date(2026, 7, 10, 22, 33, 0, 0, loc)
	j := AppendMessage("", 1, "hi", t1, loc)
	at, ok := FirstEntryAt(j)
	if !ok || !at.Equal(t1) {
		t.Fatalf("FirstEntryAt = %v %v", at, ok)
	}
	if _, ok := FirstEntryAt(""); ok {
		t.Fatal("empty journal should not have first entry")
	}
}
