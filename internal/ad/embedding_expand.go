package ad

import (
	"fmt"
	"strings"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/logger"
)

const embeddingExpandConvID = "ad-embedding-expand-v2"

// Keep expansion short: extra restated copy dilutes distinctive terms
// and pulls every "pickup part" ad toward generic pickup queries.
const maxEmbeddingExpandRunes = 400

const embeddingExpandSystemPrompt = `You write a short alias list used only
to encode a classified ad for semantic search. Use the CATEGORY and AD
COPY in the user message.

Output plain prose, at most 60 words. No title, no selling copy.

Add search names a knowledgeable buyer would type that are NOT already
in the AD COPY. Include, when the copy implies them:

- brand/make aliases (IH, Chevy, International Harvester, etc.)
- model codes and nicknames
- body style or item type (pickup, sedan, tractor, carburetor, etc.)
- common fitment names for parts (years, platform)

Do not invent specs, years, or parts the copy does not support.
Do not repeat phrases already in the title, description, or tags.
If nothing useful is missing, output nothing.
`

func ExpandForEmbedding(in EmbeddingInput) string {
	if config.GrokAPIKey == "" {
		return ""
	}
	if strings.TrimSpace(in.Title) == "" &&
		strings.TrimSpace(in.Description) == "" {
		return ""
	}
	resp, err := grokCallConv(
		embeddingExpandSystemPrompt,
		embeddingExpandUserPrompt(in),
		embeddingExpandConvID,
	)
	if err != nil {
		logger.Warn("embedding expand: grok call failed",
			"adID", in.ID, "error", err)
		return ""
	}
	out := clipExpansion(resp)
	if out != "" {
		logger.Debug("embedding expand", "adID", in.ID, "text", out)
	}
	return out
}

func clipExpansion(s string) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= maxEmbeddingExpandRunes {
		return s
	}
	return string(r[:maxEmbeddingExpandRunes])
}

func embeddingExpandUserPrompt(in EmbeddingInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "CATEGORY: %s\n\n", in.CategoryName)
	b.WriteString("AD COPY:\n")
	fmt.Fprintf(&b, "Title: %s\n", in.Title)
	fmt.Fprintf(&b, "Description: %s\n",
		ParseDescriptionForDisplay(in.Description).Body)
	if len(in.TagLabels) > 0 {
		fmt.Fprintf(&b, "Tags: %s\n", strings.Join(in.TagLabels, ", "))
	}
	if in.City != "" || in.AdminArea != "" {
		fmt.Fprintf(&b, "Location: %s, %s, %s\n",
			in.City, in.AdminArea, in.Country)
	}
	var fields []string
	for _, d := range in.FacetDefs {
		v, ok := in.Facets[d.Key]
		if !ok || !v.Present() {
			continue
		}
		if s := d.FormatFull(v); s != "" {
			fields = append(fields, d.Label+": "+s)
		}
	}
	if len(fields) > 0 {
		b.WriteString("\nFIELDS:\n")
		for _, line := range fields {
			fmt.Fprintf(&b, "  %s\n", line)
		}
	}
	return b.String()
}
