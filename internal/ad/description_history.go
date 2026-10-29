package ad

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/currency"
	"github.com/partspile/site/internal/entrylog"
	"github.com/partspile/site/internal/facet"
	"github.com/partspile/site/internal/location"
)

const OriginalLabel = "original"

// DescriptionAdditionLabel is the legacy append-via-edit history marker.
const DescriptionAdditionLabel = "Description Addition"

// DescriptionChangeLabel is the history label for a description rewrite.
const DescriptionChangeLabel = "Description change"

// HistoryEntryDisplay is one parsed edit-history block for UI rendering.
type HistoryEntryDisplay struct {
	Header       string
	Body         string
	ImageIndices []int
}

// DescriptionDisplay holds original ad text and edit history for display.
type DescriptionDisplay struct {
	// Original is the current listing text.
	Original string
	// Body is the current listing text (same as Original).
	Body    string
	History []HistoryEntryDisplay
}

// DisplayDescription renders stored description as plain text for display.
func DisplayDescription(desc string) string {
	parts := ParseDescriptionForDisplay(desc)
	var b strings.Builder
	if parts.Original != "" {
		b.WriteString(parts.Original)
	}
	for _, h := range parts.History {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(h.Header)
		if h.Body != "" {
			b.WriteString("\n\n")
			b.WriteString(h.Body)
		}
	}
	return b.String()
}

// WrapDescription stores plain body text as the original entry.
func WrapDescription(body string, at time.Time, tz *time.Location) string {
	body = strings.TrimSpace(SanitizeAdText(body))
	return entrylog.BuildBlock(OriginalLabel, "", body, at, tz)
}

// ParseDescriptionForDisplay splits stored description into original and history.
func ParseDescriptionForDisplay(desc string) DescriptionDisplay {
	original, historyBlocks := splitDescriptionBlocks(desc)
	out := DescriptionDisplay{
		Original: original,
		Body:     original,
	}
	for _, b := range historyBlocks {
		header := formatDisplayHeader(b)
		indices := imageIndicesFromHistoryEntry(header, b.Body)
		body := b.Body
		if len(indices) > 0 {
			body = ""
		}
		out.History = append(out.History, HistoryEntryDisplay{
			Header:       header,
			Body:         body,
			ImageIndices: indices,
		})
	}
	return out
}

// applyDescriptionEdit replaces the original body when the posted
// description differs, and journals a short Grok summary of the change.
func applyDescriptionEdit(desc, newBody string, at time.Time,
	tz *time.Location) (string, error) {
	newBody = strings.TrimSpace(SanitizeAdText(newBody))
	if utf8.RuneCountInString(newBody) > config.MaxAdDescriptionLength {
		return "", fmt.Errorf(
			"description must be at most %d characters",
			config.MaxAdDescriptionLength,
		)
	}
	previous := ParseDescriptionForDisplay(desc).Original
	if previous == newBody {
		return desc, nil
	}
	desc = replaceOriginalBody(desc, newBody, at, tz)
	summary := summarizeDescChangeFn(previous, newBody)
	return AppendHistoryEntry(
		desc, DescriptionChangeLabel, summary, at, tz,
	), nil
}

func replaceOriginalBody(desc, newBody string, at time.Time,
	tz *time.Location) string {
	newBody = strings.TrimSpace(SanitizeAdText(newBody))
	blocks := entrylog.Parse(desc)
	if len(blocks) == 0 {
		return WrapDescription(newBody, at, tz)
	}
	orig := blocks[0]
	orig.Body = newBody
	return entrylog.Join(append([]entrylog.Block{orig}, blocks[1:]...), tz)
}

// SplitDescription separates the original body from server history.
func SplitDescription(desc string) (original, history string) {
	orig, blocks := splitDescriptionBlocks(desc)
	if len(blocks) == 0 {
		return orig, ""
	}
	return orig, joinHistoryBlocks(blocks)
}

func splitDescriptionBlocks(desc string) (original string,
	history []entrylog.Block) {
	blocks := entrylog.Parse(desc)
	if len(blocks) == 0 {
		return desc, nil
	}
	if blocks[0].Label == OriginalLabel {
		return blocks[0].Body, blocks[1:]
	}
	return blocks[0].Body, blocks[1:]
}

func joinHistoryBlocks(blocks []entrylog.Block) string {
	if len(blocks) == 0 {
		return ""
	}
	var b strings.Builder
	for i, block := range blocks {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(entrylog.BuildBlock(
			block.Label, block.Meta, block.Body, block.At, nil,
		))
	}
	return b.String()
}

func formatDisplayHeader(b entrylog.Block) string {
	return strings.ToLower(b.At.Format("1/2/2006 3:04 pm")) +
		"  " + b.Label
}

const imagesAddedLabel = "Images Added"

func formatImageAdditionBody(startIndex, count int) string {
	if count <= 0 {
		return ""
	}
	parts := make([]string, count)
	for i := range parts {
		parts[i] = strconv.Itoa(startIndex + i)
	}
	return strings.Join(parts, ",")
}

func imageIndicesFromHistoryEntry(header, body string) []int {
	if !strings.Contains(header, imagesAddedLabel) {
		return nil
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}
	var indices []int
	for _, part := range strings.Split(body, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 {
			continue
		}
		indices = append(indices, n)
	}
	return indices
}

// AppendHistoryEntry prepends a labeled, timestamped history block (newest first).
func AppendHistoryEntry(desc, label, body string, at time.Time,
	tz *time.Location) string {
	body = strings.TrimSpace(SanitizeAdText(body))
	if body == "" && label != "Description compressed" &&
		label != "History compressed" {
		return desc
	}
	return entrylog.PrependAfterFirst(desc, label, "", body, at, tz)
}

func historyEntryText(b entrylog.Block) string {
	header := formatDisplayHeader(b)
	if b.Body == "" {
		return header
	}
	return header + "\n\n" + b.Body
}

// EnsureDescriptionFits compresses description text until it fits MaxAdDescriptionLength.
func EnsureDescriptionFits(desc string, at time.Time,
	tz *time.Location) (string, error) {
	return ensureDescriptionFits(desc, at, tz)
}

func ensureDescriptionFits(desc string, at time.Time,
	tz *time.Location) (string, error) {
	max := config.MaxAdDescriptionLength
	if descriptionRuneCount(desc) <= max {
		return desc, nil
	}

	original, historyBlocks := splitDescriptionBlocks(desc)

	targetOriginal := max / 2
	if targetOriginal < 100 {
		targetOriginal = 100
	}
	compressed, err := compressWithGrok(
		compressOriginalSystemPrompt, original, targetOriginal,
	)
	if err != nil {
		return "", fmt.Errorf("compress description: %w", err)
	}
	original = compressed
	blocks := entrylog.Parse(desc)
	origAt := at
	if len(blocks) > 0 && blocks[0].Label == OriginalLabel {
		origAt = blocks[0].At
	}
	desc = entrylog.Join(
		append([]entrylog.Block{{
			Label: OriginalLabel,
			Body:  original,
			At:    origAt,
		}}, historyBlocks...),
		tz,
	)
	desc = AppendHistoryEntry(
		desc,
		"Description compressed",
		"Original description was summarized to make room for edit history.",
		at,
		tz,
	)
	if descriptionRuneCount(desc) <= max {
		return desc, nil
	}

	_, historyBlocks = splitDescriptionBlocks(desc)
	keepRecent := 2
	if len(historyBlocks) <= keepRecent {
		return truncateRunes(desc, max), nil
	}

	recentEntries := historyBlocks[:keepRecent]
	oldEntries := historyBlocks[keepRecent:]
	var oldParts []string
	for _, e := range oldEntries {
		oldParts = append(oldParts, historyEntryText(e))
	}
	oldText := strings.Join(oldParts, "\n\n")
	targetHistory := max / 4
	if targetHistory < 80 {
		targetHistory = 80
	}
	compressedHistory, err := compressWithGrok(
		compressHistorySystemPrompt, oldText, targetHistory,
	)
	if err != nil {
		return "", fmt.Errorf("compress history: %w", err)
	}
	mergedOld := entrylog.Block{
		Label: "History compressed",
		Body:  compressedHistory,
		At:    at,
	}
	allHistory := append(recentEntries, mergedOld)
	origBlock := entrylog.Block{
		Label: OriginalLabel,
		Body:  original,
		At:    origAt,
	}
	desc = entrylog.Join(append([]entrylog.Block{origBlock}, allHistory...), tz)
	if descriptionRuneCount(desc) <= max {
		return desc, nil
	}
	return truncateRunes(desc, max), nil
}

func facetValuesEqual(a, b facet.Value) bool {
	if a.Num == nil && b.Num != nil || a.Num != nil && b.Num == nil {
		return false
	}
	if a.Num != nil && b.Num != nil && *a.Num != *b.Num {
		return false
	}
	aVals := a.MultiEnumValues()
	bVals := b.MultiEnumValues()
	if len(aVals) > 0 || len(bVals) > 0 {
		if len(aVals) != len(bVals) {
			return false
		}
		for i := range aVals {
			if aVals[i] != bVals[i] {
				return false
			}
		}
		return true
	}
	if a.Text == nil && b.Text != nil || a.Text != nil && b.Text == nil {
		return false
	}
	if a.Text != nil && b.Text != nil && *a.Text != *b.Text {
		return false
	}
	return true
}

func formatPriceChange(old, new facet.Value) string {
	oldAmt, oldCur, oldOK := priceFromValue(old)
	newAmt, newCur, newOK := priceFromValue(new)
	if !oldOK && !newOK {
		return ""
	}
	if !oldOK && newOK {
		if newAmt == 0 {
			return "Listed as FREE"
		}
		return fmt.Sprintf("Price set to %s", currency.Format(newAmt, newCur))
	}
	if oldOK && !newOK {
		return fmt.Sprintf("Price removed (was %s)", currency.Format(oldAmt, oldCur))
	}
	if oldAmt == 0 && newAmt == 0 {
		return ""
	}
	if oldAmt == 0 {
		return fmt.Sprintf("Price set to %s", currency.Format(newAmt, newCur))
	}
	if newAmt == 0 {
		return fmt.Sprintf("Listed as FREE (was %s)", currency.Format(oldAmt, oldCur))
	}
	oldStr := currency.Format(oldAmt, oldCur)
	newStr := currency.Format(newAmt, newCur)
	if newAmt < oldAmt {
		return fmt.Sprintf("Price dropped from %s to %s", oldStr, newStr)
	}
	if newAmt > oldAmt {
		return fmt.Sprintf("Price increased from %s to %s", oldStr, newStr)
	}
	if oldCur != newCur {
		return fmt.Sprintf("Price currency changed from %s to %s", oldStr, newStr)
	}
	return ""
}

func priceFromValue(v facet.Value) (amount int, code string, ok bool) {
	if v.Num == nil {
		return 0, "", false
	}
	code = currency.Default
	if v.Text != nil {
		code = *v.Text
	}
	return *v.Num, code, true
}

func formatFacetChange(d facet.Def, old, new facet.Value) string {
	oldStr := d.FormatFull(old)
	newStr := d.FormatFull(new)
	if oldStr == "" && newStr == "" {
		return ""
	}
	if oldStr == "" {
		return fmt.Sprintf("%s set to %s", d.Label, newStr)
	}
	if newStr == "" {
		return fmt.Sprintf("%s removed (was %s)", d.Label, oldStr)
	}
	return fmt.Sprintf("%s changed from %s to %s", d.Label, oldStr, newStr)
}

func formatTitleChange(oldTitle, newTitle string) string {
	if oldTitle == newTitle {
		return ""
	}
	return fmt.Sprintf("Title changed from %q to %q", oldTitle, newTitle)
}

func formatLocationChange(oldText, newText string) string {
	oldText = strings.TrimSpace(oldText)
	newText = strings.TrimSpace(newText)
	if oldText == newText {
		return ""
	}
	if oldText == "" {
		return fmt.Sprintf("Location set to %s", newText)
	}
	if newText == "" {
		return fmt.Sprintf("Location removed (was %s)", oldText)
	}
	return fmt.Sprintf("Location changed from %s to %s", oldText, newText)
}

func formatLocationHistoryChange(oldAd Ad, newLocationText string,
	category Category) string {
	newRaw := strings.TrimSpace(newLocationText)
	if UsesFullAddressDisplay(category) {
		oldRaw := fullAddressText(oldAd)
		if strings.EqualFold(oldRaw, newRaw) {
			return ""
		}
		if newRaw == "" {
			return "Address removed"
		}
		return "Address changed"
	}

	oldRaw := strings.TrimSpace(oldAd.RawLocation)
	if strings.EqualFold(oldRaw, newRaw) {
		return ""
	}

	oldDisplay := location.DisplayText(
		oldAd.City, oldAd.AdminArea, oldAd.Country,
	)
	newDisplay := ""
	if newRaw != "" {
		display, _, _ := location.DisplayTextForInput(newRaw)
		newDisplay = display
	}
	return formatLocationChange(oldDisplay, newDisplay)
}

// BuildFieldChangeEntries returns history entry bodies for changed ad fields.
func BuildFieldChangeEntries(oldAd Ad, newTitle string, newLocationText string,
	newFacets map[string]facet.Value, category Category) []struct {
	label string
	body  string
} {
	var entries []struct {
		label string
		body  string
	}
	oldTitle := oldAd.Title
	newTitle = strings.TrimSpace(SanitizeAdText(newTitle))
	if body := formatTitleChange(oldTitle, newTitle); body != "" {
		entries = append(entries, struct {
			label string
			body  string
		}{"Title change", body})
	}
	if body := formatLocationHistoryChange(oldAd, newLocationText, category); body != "" {
		label := "Location change"
		if UsesFullAddressDisplay(category) {
			label = "Address change"
		}
		entries = append(entries, struct {
			label string
			body  string
		}{label, body})
	}
	for _, d := range category.Facets() {
		oldV, oldOK := oldAd.Facets[d.Key]
		newV, newOK := newFacets[d.Key]
		if !oldOK {
			oldV = facet.Value{}
		}
		if !newOK {
			newV = facet.Value{}
		}
		if facetValuesEqual(oldV, newV) {
			continue
		}
		var body string
		var label string
		if d.Key == "price" {
			body = formatPriceChange(oldV, newV)
			label = "Price change"
		} else if d.Kind == facet.Location {
			continue
		} else {
			body = formatFacetChange(d, oldV, newV)
			label = d.Label + " change"
		}
		if body != "" {
			entries = append(entries, struct {
				label string
				body  string
			}{label, body})
		}
	}
	return entries
}
