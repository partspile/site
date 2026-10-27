package ad

import (
	"fmt"
	"strings"

	"github.com/partspile/site/internal/facet"
)

func suggestionsUserPrompt(in SuggestInput) string {
	var b strings.Builder
	b.WriteString("AD COPY:\n")
	fmt.Fprintf(&b, "Title: %s\n", in.Title)
	fmt.Fprintf(&b, "Description: %s\n", in.Description)
	fmt.Fprintf(&b, "Location: %s\n", in.Location)

	if len(in.FormalFacets) > 0 {
		b.WriteString("\nFORMAL FORM FIELDS (do not suggest)\n")
		for _, line := range in.FormalFacets {
			fmt.Fprintf(&b, "  %s\n", line)
		}
	}

	if len(in.AlreadySelected) > 0 {
		b.WriteString("\nALREADY SELECTED (do not suggest again):\n")
		for _, s := range in.AlreadySelected {
			fmt.Fprintf(&b, "  %s\n", s.PromptDisplay())
		}
	}

	remaining := maxSuggestions - len(in.AlreadySelected)
	if remaining > 0 {
		fmt.Fprintf(&b, "\nReturn at most %d new suggestions.\n", remaining)
	}

	return b.String()
}

// FormalFacetLines formats all formal facet fields for the suggestions prompt,
// including empty ones so the model knows which fields exist on the form.
func FormalFacetLines(category Category, values map[string]facet.Value) []string {
	lines := make([]string, 0, len(category.FacetKeys))
	for _, d := range category.Facets() {
		v, ok := values[d.Key]
		if ok && v.Present() {
			if s := d.FormatFull(v); s != "" {
				lines = append(lines, d.Label+": "+s)
				continue
			}
		}
		lines = append(lines, d.Label+":")
	}
	return lines
}
