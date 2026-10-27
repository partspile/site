package ad

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/partspile/site/internal/service/grok"
)

const (
	compressOriginalSystemPrompt = "You summarize classified ad descriptions. " +
		"Preserve key facts: price, condition, mechanical state, " +
		"features, and contact intent. Output only the summary text."
	compressHistorySystemPrompt = "You summarize classified ad edit history. " +
		"Preserve what changed and when it mattered. " +
		"Output only the summary text."
	summarizeDescChangeSystemPrompt = "You summarize edits to a classified " +
		"ad description. Capture only important material changes in one " +
		"short phrase or sentence, such as 'fixed some typos' or 'added " +
		"service record details'. Do not quote long passages. Do not " +
		"mention yourself, AI, or Grok. Output only the summary."
	maxDescChangeSummaryRunes = 80
	fallbackDescChangeSummary = "Description updated"
	descChangeConvID          = "desc-change-summary"
)

// summarizeDescChangeFn is the journal summarizer; tests may stub it.
var summarizeDescChangeFn = summarizeDescriptionChange

var grokCallConv = grok.CallGrokConv

func summarizeDescriptionChange(previous, current string) string {
	userPrompt := fmt.Sprintf(
		"Summarize the important material changes in at most %d "+
			"characters (Unicode runes).\n\nPrevious:\n%s\n\nCurrent:\n%s",
		maxDescChangeSummaryRunes, previous, current,
	)
	out, err := grokCallConv(
		summarizeDescChangeSystemPrompt, userPrompt, descChangeConvID,
	)
	if err != nil {
		return fallbackDescChangeSummary
	}
	out = strings.TrimSpace(SanitizeAdText(out))
	if out == "" {
		return fallbackDescChangeSummary
	}
	return truncateRunes(out, maxDescChangeSummaryRunes)
}

func compressWithGrok(systemPrompt, text string, maxRunes int) (string, error) {
	userPrompt := fmt.Sprintf(
		"Summarize to at most %d characters (Unicode runes). "+
			"Keep the most important facts.\n\n%s",
		maxRunes, text,
	)
	out, err := grok.CallGrok(systemPrompt, userPrompt)
	if err != nil {
		return "", err
	}
	out = strings.TrimSpace(SanitizeAdText(out))
	return truncateRunes(out, maxRunes), nil
}

func descriptionRuneCount(desc string) int {
	return utf8.RuneCountInString(desc)
}

func truncateRunes(s string, max int) string {
	if descriptionRuneCount(s) <= max {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	n := 0
	for _, r := range s {
		if n >= max {
			break
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}
