package vector

import (
	"strings"
	"testing"

	"github.com/partspile/site/internal/ad"
)

func TestBuildAdEmbeddingPromptIncludesExpansion(t *testing.T) {
	got := buildAdEmbeddingPrompt(ad.EmbeddingInput{
		Title:       "S120",
		Description: "Farm truck.",
	}, "International Harvester pickup truck.")
	for _, want := range []string{
		"Title: S120",
		"Farm truck.",
		"Search background:",
		"International Harvester pickup truck.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestBuildAdEmbeddingPromptOmitsEmptyExpansion(t *testing.T) {
	got := buildAdEmbeddingPrompt(ad.EmbeddingInput{Title: "S120"}, "  ")
	if strings.Contains(got, "Search background:") {
		t.Fatal("empty expansion should not add a background block")
	}
}
