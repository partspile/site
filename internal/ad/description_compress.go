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
		"ad description by comparing Previous to Current. Text only in " +
		"Previous was removed. Text only in Current was added. Say " +
		"\"removed\" when a fact or sentence disappeared. Say \"added\" " +
		"only when it is new in Current. Rewording is \"edited\" or " +
		"\"fixed some typos\". One short phrase or sentence. Do not quote " +
		"long passages. Do not mention yourself, AI, or Grok. Output only " +
		"the summary."
	maxDescChangeSummaryRunes = 80
	fallbackDescChangeSummary = "Description updated"
	descChangeConvID          = "desc-change-summary"
)

// summarizeDescChangeFn is the journal summarizer; tests may stub it.
var summarizeDescChangeFn = summarizeDescriptionChange

var grokCallConv = grok.CallGrokConv

func summarizeDescriptionChange(previous, current string) string {
	userPrompt := fmt.Sprintf(
		"Summarize the change from Previous to Current in at most %d "+
			"characters (Unicode runes). A sentence in Previous and "+
			"missing from Current was removed.\n\n"+
			"Previous:\n%s\n\nCurrent:\n%s",
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
