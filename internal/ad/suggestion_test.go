package ad

import (
	"strings"
	"testing"

	"github.com/partspile/site/internal/facet"
)

func TestParseSuggestResponse(t *testing.T) {
	raw := `[{"label":"transmission","value":"manual"},{"label":"fuel","value":"gasoline"}]`
	got, err := parseSuggestResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
	if got[0].Label != "transmission" || got[0].Value != "manual" {
		t.Fatalf("first=%+v", got[0])
	}
}

func TestParseSuggestResponseCodeFence(t *testing.T) {
	raw := "```json\n[{\"label\":\"color\",\"value\":\"red\"}]\n```"
	got, err := parseSuggestResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Label != "color" {
		t.Fatalf("got=%+v", got)
	}
}

func TestParseFormSuggestion(t *testing.T) {
	raw := "transmission" + suggestionFormSep + " manual "
	s, ok := ParseFormSuggestion(raw)
	if !ok || s.Label != "transmission" || s.Value != "manual" {
		t.Fatalf("s=%+v ok=%v", s, ok)
	}
}

func TestParseSuggestionFormValue(t *testing.T) {
	label, value, ok := ParseSuggestionFormValue("transmission" + suggestionFormSep + "manual")
	if !ok || label != "transmission" || value != "manual" {
		t.Fatalf("label=%q value=%q ok=%v", label, value, ok)
	}
}

func TestDedupeSuggestions(t *testing.T) {
	got := dedupeSuggestions([]Suggestion{
		{Label: "fuel", Value: "gas"},
		{Label: "fuel", Value: "diesel"},
		{Label: "fuel", Value: "gas"},
		{Label: "", Value: "x"},
	})
	if len(got) != 2 {
		t.Fatalf("len=%d got=%+v", len(got), got)
	}
}

func TestTagsJSONRoundTrip(t *testing.T) {
	in := []Suggestion{
		{Label: "fuel", Value: "diesel"},
		{Label: "transmission", Value: "automatic"},
	}
	raw := tagsJSON(in)
	got, err := parseTagsJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Label != "fuel" {
		t.Fatalf("got=%+v", got)
	}
}

func TestNormalizeSuggestionTruncates(t *testing.T) {
	long := strings.Repeat("a", 40)
	n, ok := normalizeSuggestion(Suggestion{Label: long, Value: long})
	if !ok {
		t.Fatal("expected ok")
	}
	if len(n.Label) != maxSuggestionLabelLen || len(n.Value) != maxSuggestionValueLen {
		t.Fatalf("label=%d value=%d", len(n.Label), len(n.Value))
	}
}

func TestSuggestionDisplay(t *testing.T) {
	s := Suggestion{Label: "heated seats", Value: "yes"}
	if s.Display() != "heated seats" {
		t.Fatalf("Display=%q", s.Display())
	}
	if s.PromptDisplay() != "heated seats: yes" {
		t.Fatalf("PromptDisplay=%q", s.PromptDisplay())
	}
}

func TestUsefulSuggestion(t *testing.T) {
	facets := formalFacetKeySet(map[string]string{"price": "Price", "mileage": "Mileage"})

	if _, ok := usefulSuggestion(Suggestion{Label: "heated seats", Value: "yes"}, facets); !ok {
		t.Fatal("binary yes should be allowed")
	}
	if _, ok := usefulSuggestion(Suggestion{Label: "fuel", Value: ""}, facets); ok {
		t.Fatal("empty value should be rejected")
	}
	if _, ok := usefulSuggestion(Suggestion{Label: "fuel", Value: "diesel"}, facets); !ok {
		t.Fatal("expected useful")
	}
	if _, ok := usefulSuggestion(Suggestion{Label: "fuel", Value: "fuel"}, facets); ok {
		t.Fatal("label=value should be rejected")
	}
	if _, ok := usefulSuggestion(Suggestion{Label: "price", Value: "5000"}, facets); ok {
		t.Fatal("formal facet should be rejected")
	}
}

func TestFormalFacetLinesIncludesEmpty(t *testing.T) {
	cat := Category{FacetKeys: []string{"year", "condition", "mileage", "price", "title_status"}}
	year, mileage, price := 2026, 124, 14
	km, usd := "km", "USD"
	lines := FormalFacetLines(cat, map[string]facet.Value{
		"year":    {Num: &year},
		"mileage": {Num: &mileage, Text: &km},
		"price":   {Num: &price, Text: &usd},
	})
	want := []string{
		"Year: 2026",
		"Condition:",
		"Mileage: 124 km",
		"Price: $ 14",
		"Title Status:",
	}
	if len(lines) != len(want) {
		t.Fatalf("lines=%v", lines)
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("line[%d] = %q, want %q", i, lines[i], w)
		}
	}
}

func TestSuggestionsUserPromptFormalFacets(t *testing.T) {
	prompt := suggestionsUserPrompt(SuggestInput{
		Title:       "1971 Chevrolet corvette",
		Description: "Barn find",
		Location:    "Denver",
		FormalFacets: []string{
			"Condition: Used - Good",
			"Price: $13,950",
			"Year: 1975",
		},
		AlreadySelected: []Suggestion{{Label: "transmission", Value: "manual"}},
	})
	for _, want := range []string{
		"FORMAL FORM FIELDS (do not suggest)",
		"Condition: Used - Good",
		"Price: $13,950",
		"Year: 1975",
		"ALREADY SELECTED (do not suggest again)",
		"transmission: manual",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestFormatTagUpdates(t *testing.T) {
	old := []Suggestion{
		{Label: "Clean title", Value: "yes"},
		{Label: "A/C", Value: "yes"},
	}
	new := []Suggestion{
		{Label: "Clean title", Value: "yes"},
		{Label: "Manual", Value: "yes"},
	}
	body := FormatTagUpdates(old, new)
	if !strings.Contains(body, "Added: Manual") {
		t.Errorf("body = %q", body)
	}
	if !strings.Contains(body, "Removed: A/C") {
		t.Errorf("body = %q", body)
	}
	if FormatTagUpdates(old, old) != "" {
		t.Fatal("expected no change body")
	}
}
