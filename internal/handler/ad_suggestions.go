package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/currency"
	"github.com/partspile/site/internal/facet"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/param"
	uiads "github.com/partspile/site/internal/ui/ads"
)

func SuggestionsHandler(c *fiber.Ctx) error {
	category := ad.GetCategory(cookie.GetCategoryID(c))
	return renderSuggestions(c, category.ID, category)
}

func EditSuggestionsHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)

	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	tz := cookie.GetTimezone(c)
	a, err := ad.GetAd(userID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if a.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "You are not the owner of this ad")
	}
	if !a.IsActive() {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot edit a deleted or inactive ad")
	}

	category := ad.GetCategory(a.CategoryID)
	return renderSuggestions(c, a.CategoryID, category)
}

func renderSuggestions(c *fiber.Ctx, categoryID int, category ad.Category) error {
	selected := parseSuggestionFormValues(c)
	suggested, _ := ad.GenerateSuggestions(suggestInputFrom(c, categoryID, category, selected))
	merged := mergeSuggestionOptions(selected, suggested)
	return render(c, uiads.SuggestionsPartial(merged))
}

func suggestInputFrom(c *fiber.Ctx, categoryID int, category ad.Category,
	selected []ad.Suggestion) ad.SuggestInput {
	facets := category.Facets()
	formalFacets := make(map[string]string, len(facets))
	for _, d := range facets {
		formalFacets[d.Key] = d.Label
	}
	facetValues := parseFormFacetValues(c, category)
	locText := c.FormValue("location")
	if ad.HasLocationFacet(category) {
		locText = ad.LocationTextFromFacets(category, facetValues)
	}
	return ad.SuggestInput{
		CategoryID:      categoryID,
		CategoryName:    category.Name,
		Title:           c.FormValue("title"),
		Description:     c.FormValue("description"),
		Location:        locText,
		Facets:          formalFacets,
		FormalFacets:    ad.FormalFacetLines(category, facetValues),
		AlreadySelected: selected,
	}
}

// parseFormFacetValues reads facet values from the ad form without validation.
func parseFormFacetValues(c *fiber.Ctx,
	category ad.Category) map[string]facet.Value {
	values := make(map[string]facet.Value)
	for _, d := range category.Facets() {
		switch d.Kind {
		case facet.Money:
			if c.FormValue("price_free") == "1" {
				amount := 0
				code := currency.Normalize(c.FormValue("price_currency"))
				if !currency.IsSupported(code) {
					code = defaultCurrencyForUser(c)
				}
				values[d.Key] = facet.Value{Num: &amount, Text: &code}
				continue
			}
			raw := strings.TrimSpace(c.FormValue(d.Key))
			if raw == "" {
				continue
			}
			n, err := strconv.Atoi(raw)
			if err != nil || n < 0 {
				continue
			}
			code := currency.Normalize(c.FormValue("price_currency"))
			if !currency.IsSupported(code) {
				code = defaultCurrencyForUser(c)
			}
			values[d.Key] = facet.Value{Num: &n, Text: &code}
		case facet.Enum:
			val := strings.TrimSpace(c.FormValue(d.Key))
			if val != "" {
				values[d.Key] = facet.Value{Text: &val}
			}
		case facet.Date:
			raw := strings.TrimSpace(c.FormValue(d.Key))
			if raw == "" {
				continue
			}
			if v, err := facet.ParseDateValue(raw); err == nil {
				values[d.Key] = v
			}
		case facet.MultiEnum:
			vals := parseFormEnumCheckboxes(c, d.Key, d.Enum)
			if len(vals) > 0 {
				values[d.Key] = facet.EncodeMultiEnum(vals)
			}
		case facet.Flag:
			if c.FormValue(d.Key) == "1" {
				values[d.Key] = facet.EncodeFlag(true)
			}
		case facet.Location:
			raw := strings.TrimSpace(c.FormValue(d.Key))
			if raw != "" {
				values[d.Key] = facet.Value{Text: &raw}
			}
		default:
			num, err := parseOptionalFacet(c.FormValue(d.Key))
			if err != nil || num == nil {
				continue
			}
			v := facet.Value{Num: num}
			if len(d.Units) > 0 {
				unit := strings.TrimSpace(c.FormValue(d.Key + "_unit"))
				if unit == "" {
					unit = cookie.GetDistanceUnit(c)
				}
				if d.ValidUnit(unit) {
					u := d.NormalizeUnit(unit)
					v.Text = &u
				}
			}
			values[d.Key] = v
		}
	}
	return values
}

func parseSuggestionFormValues(c *fiber.Ctx) []ad.Suggestion {
	var suggestions []ad.Suggestion
	c.Context().QueryArgs().VisitAll(func(k, v []byte) {
		if string(k) != uiads.SuggestionsFormName() {
			return
		}
		if s, ok := ad.ParseFormSuggestion(string(v)); ok {
			suggestions = append(suggestions, s)
		}
	})
	return dedupeSuggestions(suggestions)
}

func dedupeSuggestions(suggestions []ad.Suggestion) []ad.Suggestion {
	seen := make(map[string]struct{}, len(suggestions))
	out := make([]ad.Suggestion, 0, len(suggestions))
	for _, s := range suggestions {
		key := s.Key()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	return out
}

func mergeSuggestionOptions(selected []ad.Suggestion,
	suggested []ad.Suggestion) []uiads.SuggestionOption {
	seen := make(map[string]struct{}, len(selected)+len(suggested))
	out := make([]uiads.SuggestionOption, 0, len(selected)+len(suggested))

	for _, s := range selected {
		key := s.Key()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, uiads.SuggestionOption{
			Label:    s.Label,
			Value:    s.Value,
			Selected: true,
		})
	}
	for _, s := range suggested {
		key := s.Key()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, uiads.SuggestionOption{
			Label:    s.Label,
			Value:    s.Value,
			Selected: false,
		})
	}
	return out
}

func parseAdSuggestions(c *fiber.Ctx) []ad.Suggestion {
	name := uiads.SuggestionsFormName()
	var suggestions []ad.Suggestion
	add := func(raw string) {
		if s, ok := ad.ParseFormSuggestion(raw); ok {
			suggestions = append(suggestions, s)
		}
	}
	// Urlencoded request bodies expose fields via PostArgs.
	c.Context().PostArgs().VisitAll(func(k, v []byte) {
		if string(k) == name {
			add(string(v))
		}
	})
	// Multipart request bodies (the ad form uses them for image uploads)
	// are not present in PostArgs, so read them from the multipart form.
	if form, err := c.MultipartForm(); err == nil && form != nil {
		for _, v := range form.Value[name] {
			add(v)
		}
	}
	return suggestions
}
