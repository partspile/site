package ad

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/service/grok"
)

const (
	maxSuggestions        = 12 // max pills on form and saved per ad
	maxSuggestionLabelLen = 32
	maxSuggestionValueLen = 32
	suggestionFormSep     = "\x1f"
	binarySuggestionValue = "yes"
)

type Suggestion struct {
	Label string
	Value string
}

type SuggestInput struct {
	CategoryID      int
	CategoryName    string
	Title           string
	Description     string
	Location        string
	Facets          map[string]string // formal facet key -> display label
	FormalFacets    []string          // filled formal fields, e.g. "Condition: Used - Good"
	AlreadySelected []Suggestion
}

func isBinaryValue(value string) bool {
	return value == binarySuggestionValue
}

func (s Suggestion) Display() string {
	if isBinaryValue(s.Value) {
		return s.Label
	}
	return s.Label + ": " + s.Value
}

// PromptDisplay formats a suggestion for LLM context (includes ": yes" for binary).
func (s Suggestion) PromptDisplay() string {
	if isBinaryValue(s.Value) {
		return s.Label + ": yes"
	}
	return s.Label + ": " + s.Value
}

func (s Suggestion) Key() string {
	return strings.ToLower(s.Label) + suggestionFormSep + strings.ToLower(s.Value)
}

func EncodeSuggestionFormValue(label, value string) string {
	return label + suggestionFormSep + value
}

func ParseSuggestionFormValue(raw string) (label, value string, ok bool) {
	i := strings.IndexByte(raw, suggestionFormSep[0])
	if i < 0 {
		return "", "", false
	}
	return raw[:i], raw[i+1:], true
}

func normalizeSuggestion(s Suggestion) (Suggestion, bool) {
	label := strings.TrimSpace(s.Label)
	if label == "" {
		return Suggestion{}, false
	}
	if len(label) > maxSuggestionLabelLen {
		label = label[:maxSuggestionLabelLen]
	}
	value := strings.TrimSpace(s.Value)
	if value == "" {
		return Suggestion{}, false
	}
	if strings.EqualFold(value, binarySuggestionValue) {
		value = binarySuggestionValue
	}
	if len(value) > maxSuggestionValueLen {
		value = value[:maxSuggestionValueLen]
	}
	return Suggestion{Label: label, Value: value}, true
}

// ParseFormSuggestion parses an encoded form checkbox value into a normalized suggestion.
func ParseFormSuggestion(raw string) (Suggestion, bool) {
	label, value, ok := ParseSuggestionFormValue(raw)
	if !ok {
		return Suggestion{}, false
	}
	return normalizeSuggestion(Suggestion{Label: label, Value: value})
}

func dedupeSuggestions(suggestions []Suggestion) []Suggestion {
	seen := make(map[string]struct{}, len(suggestions))
	out := make([]Suggestion, 0, len(suggestions))
	for _, s := range suggestions {
		n, ok := normalizeSuggestion(s)
		if !ok {
			continue
		}
		key := n.Key()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, n)
		if len(out) >= maxSuggestions {
			break
		}
	}
	return out
}

// FormatTagUpdates describes added and removed tag pills.
func FormatTagUpdates(old, new []Suggestion) string {
	old = dedupeSuggestions(old)
	new = dedupeSuggestions(new)
	oldKeys := make(map[string]Suggestion, len(old))
	for _, s := range old {
		oldKeys[s.Key()] = s
	}
	newKeys := make(map[string]Suggestion, len(new))
	for _, s := range new {
		newKeys[s.Key()] = s
	}
	var added, removed []string
	for key, s := range newKeys {
		if _, ok := oldKeys[key]; !ok {
			added = append(added, s.Display())
		}
	}
	for key, s := range oldKeys {
		if _, ok := newKeys[key]; !ok {
			removed = append(removed, s.Display())
		}
	}
	if len(added) == 0 && len(removed) == 0 {
		return ""
	}
	var parts []string
	if len(added) > 0 {
		parts = append(parts, "Added: "+strings.Join(added, ", "))
	}
	if len(removed) > 0 {
		parts = append(parts, "Removed: "+strings.Join(removed, ", "))
	}
	return strings.Join(parts, "\n")
}

// TagsJSON serializes tags for storage in ads.tags.
func TagsJSON(suggestions []Suggestion) string {
	return tagsJSON(suggestions)
}

func tagsJSON(suggestions []Suggestion) string {
	suggestions = dedupeSuggestions(suggestions)
	if len(suggestions) == 0 {
		return "[]"
	}
	data, err := json.Marshal(suggestions)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func parseTagsJSON(raw string) ([]Suggestion, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil, nil
	}
	var suggestions []Suggestion
	if err := json.Unmarshal([]byte(raw), &suggestions); err != nil {
		return nil, fmt.Errorf("parse suggestions json: %w", err)
	}
	return dedupeSuggestions(suggestions), nil
}

// LoadTags reads tags JSON for one ad (detail page only).
func LoadTags(a *Ad) error {
	var raw string
	err := db.QueryRow(
		`SELECT COALESCE(tags, '[]') FROM ads WHERE id = $1`,
		a.ID,
	).Scan(&raw)
	if err != nil {
		return fmt.Errorf("load tags: %w", err)
	}
	tags, err := parseTagsJSON(raw)
	if err != nil {
		return err
	}
	a.Tags = tags
	return nil
}

func GenerateSuggestions(in SuggestInput) ([]Suggestion, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.Location = strings.TrimSpace(in.Location)
	if in.Title == "" && in.Description == "" {
		return nil, nil
	}

	systemPrompt := suggestionsSystemPrompt(in.CategoryName, in.Facets)
	userPrompt := suggestionsUserPrompt(in)

	resp, err := grok.CallGrokConv(systemPrompt, userPrompt, suggestionsConvID(in.CategoryID))
	if err != nil {
		logger.Warn("generate suggestions: grok call failed", "error", err)
		return nil, nil
	}

	parsed, err := parseSuggestResponse(resp)
	if err != nil {
		logger.Warn("generate suggestions: parse failed", "error", err)
		return nil, nil
	}

	remaining := maxSuggestions - len(in.AlreadySelected)
	if remaining <= 0 {
		return nil, nil
	}

	facetKeys := formalFacetKeySet(in.Facets)
	selectedKeys := make(map[string]struct{}, len(in.AlreadySelected))
	selectedLabels := make(map[string]struct{}, len(in.AlreadySelected))
	for _, s := range in.AlreadySelected {
		selectedKeys[s.Key()] = struct{}{}
		selectedLabels[strings.ToLower(s.Label)] = struct{}{}
	}

	out := make([]Suggestion, 0, len(parsed))
	for _, s := range parsed {
		n, ok := usefulSuggestion(s, facetKeys)
		if !ok {
			continue
		}
		if _, dup := selectedKeys[n.Key()]; dup {
			continue
		}
		if _, taken := selectedLabels[strings.ToLower(n.Label)]; taken {
			continue
		}
		out = append(out, n)
		if len(out) >= remaining {
			break
		}
	}
	return out, nil
}

func formalFacetKeySet(facets map[string]string) map[string]struct{} {
	set := make(map[string]struct{}, len(facets))
	for k := range facets {
		set[strings.ToLower(k)] = struct{}{}
	}
	return set
}

func usefulSuggestion(s Suggestion,
	facetSet map[string]struct{}) (Suggestion, bool) {
	n, ok := normalizeSuggestion(s)
	if !ok {
		return Suggestion{}, false
	}
	if strings.EqualFold(n.Label, n.Value) {
		return Suggestion{}, false
	}
	if _, dup := facetSet[strings.ToLower(n.Label)]; dup {
		return Suggestion{}, false
	}
	return n, true
}

type suggestResponseJSON struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func parseSuggestResponse(resp string) ([]Suggestion, error) {
	resp = strings.TrimSpace(resp)
	resp = trimCodeFence(resp)

	var items []suggestResponseJSON
	if err := json.Unmarshal([]byte(resp), &items); err != nil {
		return nil, err
	}

	out := make([]Suggestion, 0, len(items))
	for _, item := range items {
		out = append(out, Suggestion{Label: item.Label, Value: item.Value})
	}
	return out, nil
}

func trimCodeFence(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) < 2 {
		return s
	}
	end := len(lines) - 1
	for end > 0 && strings.TrimSpace(lines[end]) == "" {
		end--
	}
	if strings.HasPrefix(strings.TrimSpace(lines[end]), "```") {
		return strings.Join(lines[1:end], "\n")
	}
	return strings.Join(lines[1:], "\n")
}
