package rockopinion

import (
	"strings"
	"testing"
	"time"

	"github.com/partspile/site/internal/imagestore"
	"github.com/partspile/site/internal/message"
	"github.com/partspile/site/internal/rock"
)

func TestBuildUserPromptIncludesReasonAndImages(t *testing.T) {
	prompt := buildUserPrompt(promptInput{
		AdTitle:       "Widget",
		AdOriginal:    "Nice widget",
		OwnerID:       1,
		InquirerID:    2,
		RockThrowerID: 2,
		RockThrownAt:  time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
		Reason:        "policy",
		ImageCount:    2,
		Tz:            time.UTC,
	})
	if !strings.Contains(prompt, "Complainant selected reason: policy") {
		t.Fatalf("missing reason in prompt: %s", prompt)
	}
	if !strings.Contains(prompt, "2 image(s) attached") {
		t.Fatalf("missing image note: %s", prompt)
	}
	if !strings.Contains(prompt, "inquirer filed a complaint") {
		t.Fatalf("missing complainant role: %s", prompt)
	}
}

func TestBuildGrokPartsTextOnlyWithoutAttach(t *testing.T) {
	parts := buildGrokParts("hello", 1, 3, false)
	if len(parts) != 1 || parts[0].Type != "text" {
		t.Fatalf("expected text-only parts, got %+v", parts)
	}
}

func TestShouldAttachImages(t *testing.T) {
	prev := opinionImageStore
	t.Cleanup(func() { opinionImageStore = prev })

	opinionImageStore = nil
	if shouldAttachImages(rock.ReasonPolicy, nil, 2) {
		t.Fatal("expected no attach without store")
	}

	opinionImageStore = imagestore.NewLocal(t.TempDir())

	if !shouldAttachImages(rock.ReasonPolicy, nil, 2) {
		t.Fatal("policy + no messages should attach")
	}
	if shouldAttachImages(rock.ReasonDeal, nil, 2) {
		t.Fatal("deal + no messages should not attach")
	}
	if shouldAttachImages(rock.ReasonPolicy,
		[]message.Message{{Content: "still available?"}}, 2) {
		t.Fatal("policy + messages without image mention should not attach")
	}
	if !shouldAttachImages(rock.ReasonDeal,
		[]message.Message{{Content: "the photo looks wrong"}}, 2) {
		t.Fatal("image mention should attach")
	}
	if shouldAttachImages(rock.ReasonPolicy, nil, 0) {
		t.Fatal("no images on ad should not attach")
	}
}

func TestMessagesMentionImages(t *testing.T) {
	if !messagesMentionImages([]message.Message{
		{Content: "look at the photo please"},
	}) {
		t.Fatal("expected image mention")
	}
	if messagesMentionImages([]message.Message{
		{Content: "is it available?"},
	}) {
		t.Fatal("expected no image mention")
	}
}
