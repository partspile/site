package ad

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestEnsureDescriptionFitsNoOp(t *testing.T) {
	desc := "short description"
	got, err := EnsureDescriptionFits(desc, time.Now(), time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if got != desc {
		t.Errorf("got %q", got)
	}
}

func TestTruncateRunes(t *testing.T) {
	got := truncateRunes("hello world", 5)
	if got != "hello" {
		t.Errorf("got %q", got)
	}
	if truncateRunes("hi", 5) != "hi" {
		t.Error("expected short string unchanged")
	}
}

func TestSummarizeDescriptionChangeUsesGrok(t *testing.T) {
	orig := grokCallConv
	grokCallConv = func(systemPrompt, userPrompt, convID string) (string, error) {
		if systemPrompt != summarizeDescChangeSystemPrompt {
			t.Errorf("system prompt = %q", systemPrompt)
		}
		if !strings.Contains(systemPrompt, "Say \"removed\"") ||
			!strings.Contains(systemPrompt, "Say \"added\"") {
			t.Errorf("system prompt missing direction rules: %q",
				systemPrompt)
		}
		if convID != descChangeConvID {
			t.Errorf("convID = %q", convID)
		}
		if !strings.Contains(userPrompt,
			"missing from Current was removed") {
			t.Errorf("user prompt missing removal rule: %q", userPrompt)
		}
		if !strings.Contains(userPrompt, "Previous:\nOld copy.") {
			t.Errorf("user prompt missing previous: %q", userPrompt)
		}
		if !strings.Contains(userPrompt, "Current:\nNew copy.") {
			t.Errorf("user prompt missing current: %q", userPrompt)
		}
		return "  added service record details  ", nil
	}
	t.Cleanup(func() { grokCallConv = orig })

	got := summarizeDescriptionChange("Old copy.", "New copy.")
	if got != "added service record details" {
		t.Errorf("got %q", got)
	}
}

func TestSummarizeDescriptionChangeFallbackOnError(t *testing.T) {
	orig := grokCallConv
	grokCallConv = func(string, string, string) (string, error) {
		return "", fmt.Errorf("grok down")
	}
	t.Cleanup(func() { grokCallConv = orig })

	got := summarizeDescriptionChange("Old.", "New.")
	if got != fallbackDescChangeSummary {
		t.Errorf("got %q", got)
	}
}

func TestSummarizeDescriptionChangeFallbackOnEmpty(t *testing.T) {
	orig := grokCallConv
	grokCallConv = func(string, string, string) (string, error) {
		return "   ", nil
	}
	t.Cleanup(func() { grokCallConv = orig })

	got := summarizeDescriptionChange("Old.", "New.")
	if got != fallbackDescChangeSummary {
		t.Errorf("got %q", got)
	}
}

func TestSummarizeDescriptionChangeTruncates(t *testing.T) {
	orig := grokCallConv
	grokCallConv = func(string, string, string) (string, error) {
		return strings.Repeat("x", maxDescChangeSummaryRunes+20), nil
	}
	t.Cleanup(func() { grokCallConv = orig })

	got := summarizeDescriptionChange("Old.", "New.")
	if utf8.RuneCountInString(got) != maxDescChangeSummaryRunes {
		t.Errorf("len = %d, want %d", utf8.RuneCountInString(got),
			maxDescChangeSummaryRunes)
	}
}
