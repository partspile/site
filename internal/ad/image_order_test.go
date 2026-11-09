package ad

import (
	"strings"
	"testing"
	"time"
)

func TestParseImageOrder(t *testing.T) {
	order, err := ParseImageOrder("3,1,2", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 3 || order[0] != 3 || order[1] != 1 || order[2] != 2 {
		t.Fatalf("order = %v", order)
	}

	order, err = ParseImageOrder("", 3)
	if err != nil || order != nil {
		t.Fatalf("empty: %v %v", order, err)
	}

	if _, err := ParseImageOrder("1,2", 3); err == nil {
		t.Fatal("expected incomplete order error")
	}
	if _, err := ParseImageOrder("1,2,2", 3); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestRemapDescriptionImageIndices(t *testing.T) {
	loc := time.UTC
	origAt := time.Date(2026, 1, 5, 12, 0, 0, 0, loc)
	at := time.Date(2026, 1, 7, 0, 34, 0, 0, loc)

	desc := WrapDescription("Original.", origAt, loc)
	desc = AppendHistoryEntry(desc, imagesAddedLabel, "2,3", at, loc)

	// order [3,1,2] means old 3->1, old 1->2, old 2->3
	remapped := RemapDescriptionImageIndices(desc, []int{3, 1, 2})
	parts := ParseDescriptionForDisplay(remapped)
	if len(parts.History) != 1 {
		t.Fatalf("history len = %d", len(parts.History))
	}
	got := parts.History[0].ImageIndices
	if len(got) != 2 || got[0] != 3 || got[1] != 1 {
		t.Fatalf("indices = %v, want [3 1]", got)
	}

	same := RemapDescriptionImageIndices(desc, []int{1, 2, 3})
	if same != desc {
		t.Fatal("identity order should leave description unchanged")
	}
}

func TestRemapDescriptionImageIndicesNoImageHistory(t *testing.T) {
	loc := time.UTC
	desc := WrapDescription("Original.", time.Now(), loc)
	desc = AppendHistoryEntry(
		desc, DescriptionChangeLabel, "edited", time.Now(), loc,
	)
	out := RemapDescriptionImageIndices(desc, []int{2, 1})
	if !strings.Contains(out, DescriptionChangeLabel) {
		t.Fatalf("lost change history: %q", out)
	}
}
