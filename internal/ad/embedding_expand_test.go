package ad

import (
	"strings"
	"testing"

	"github.com/partspile/site/internal/config"
)

func TestEmbeddingExpandUserPrompt(t *testing.T) {
	in := EmbeddingInput{
		CategoryName: "Cars & Trucks",
		Title:        "1957 International Harvester S120 factory 4x4",
		Description:  "Farm truck from Havre Montana.",
		TagLabels:    []string{"4x4"},
	}
	got := embeddingExpandUserPrompt(in)
	for _, want := range []string{
		"CATEGORY: Cars & Trucks",
		"Title: 1957 International Harvester S120 factory 4x4",
		"Farm truck from Havre Montana.",
		"Tags: 4x4",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("prompt missing %q:\n%s", want, got)
		}
	}
}

func TestEmbeddingExpandSystemPromptIsGeneric(t *testing.T) {
	if strings.Contains(embeddingExpandSystemPrompt, "Cars & Trucks") {
		t.Fatal("system prompt should not name a category")
	}
	if !strings.Contains(embeddingExpandSystemPrompt, "CATEGORY") {
		t.Fatal("system prompt should mention CATEGORY")
	}
	if !strings.Contains(embeddingExpandSystemPrompt, "NOT already") {
		t.Fatal("system prompt should add missing names only")
	}
	if !strings.Contains(embeddingExpandSystemPrompt, "body style") {
		t.Fatal("system prompt should ask for body style")
	}
}

func TestClipExpansion(t *testing.T) {
	if got := clipExpansion("  IH pickup  "); got != "IH pickup" {
		t.Fatalf("trim got %q", got)
	}
	long := strings.Repeat("a", maxEmbeddingExpandRunes+10)
	if got := clipExpansion(long); len([]rune(got)) != maxEmbeddingExpandRunes {
		t.Fatalf("len=%d", len([]rune(got)))
	}
}

func TestExpandForEmbeddingEmptyCopy(t *testing.T) {
	if got := ExpandForEmbedding(EmbeddingInput{}); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestExpandForEmbeddingUsesGrok(t *testing.T) {
	prevKey := config.GrokAPIKey
	prevFn := grokCallConv
	t.Cleanup(func() {
		config.GrokAPIKey = prevKey
		grokCallConv = prevFn
	})
	config.GrokAPIKey = "test-key"
	grokCallConv = func(sys, user, conv string) (string, error) {
		if !strings.Contains(user, "CATEGORY: Cars & Trucks") {
			t.Error("user prompt missing category")
		}
		if conv != embeddingExpandConvID {
			t.Errorf("conv=%s", conv)
		}
		if !strings.Contains(sys, "semantic search") {
			t.Error("system prompt missing search role")
		}
		return "International Harvester pickup truck.", nil
	}
	got := ExpandForEmbedding(EmbeddingInput{
		CategoryName: "Cars & Trucks",
		Title:        "S120",
		Description:  "Farm truck.",
	})
	if got != "International Harvester pickup truck." {
		t.Fatalf("got %q", got)
	}
}
