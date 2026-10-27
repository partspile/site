package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/facet"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/location"
	"github.com/partspile/site/internal/search"
	"github.com/partspile/site/internal/ui"
	uiads "github.com/partspile/site/internal/ui/ads"
)

func searchStateToFilters(state cookie.SearchState,
	distanceUnit string) uiads.SearchFilters {
	opts := search.WithinMileOptions
	if distanceUnit == location.UnitKm {
		opts = search.WithinKmOptions
	}
	locationDisplay := searchLocationDisplay(state.Location)
	return uiads.SearchFilters{
		Facets:          state.Facets,
		Location:        state.Location,
		LocationDisplay: locationDisplay,
		Within:          state.Within,
		WithinUnit:      distanceUnit,
		WithinOptions:   opts,
	}
}

func searchLocationDisplay(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	display, ok, err := location.DisplayTextForInput(raw)
	if err != nil || !ok || display == "" {
		return raw
	}
	return display
}

func categoryOption(cat ad.Category) ui.CategoryOption {
	return ui.CategoryOption{ID: cat.ID, Name: cat.Name, ImageFile: cat.ImageFile}
}

func categoryOptions(categories []ad.Category) []ui.CategoryOption {
	out := make([]ui.CategoryOption, len(categories))
	for i, cat := range categories {
		out[i] = categoryOption(cat)
	}
	return out
}

func allCategoryOptions() []ui.CategoryOption {
	cats, _ := ad.GetCategories()
	return categoryOptions(cats)
}

func filterableFacets(cat ad.Category) []facet.Def {
	defs := cat.Facets()
	out := make([]facet.Def, 0, len(defs))
	for _, d := range defs {
		if d.Filterable {
			out = append(out, d)
		}
	}
	return out
}

func filterFacetsForExpanded(cat ad.Category, expanded bool) []facet.Def {
	if !expanded {
		return nil
	}
	return filterableFacets(cat)
}

func parseSearchParams(state cookie.SearchState, page, categoryID, userID int,
	distanceUnit string, tz *time.Location) search.Params {
	limit := config.SearchPageSize
	offset := (page - 1) * limit

	var facetFilters map[string]facet.Filter
	if state.Expanded {
		facetFilters = expandFacetFilters(state.Facets, tz)
	}

	p := search.BuildParams(
		categoryID, limit, offset,
		state.Q, state.Location, state.Within, distanceUnit,
		facetFilters,
	)
	p.UserID = userID
	p.Expanded = state.Expanded
	return p
}

// saveSearchStateFromRequest updates the search cookie and returns the new
// state (the request cookie is unchanged until the response is received).
func saveSearchStateFromRequest(c *fiber.Ctx, expanded *bool,
	fromForm bool) cookie.SearchState {
	distanceUnit := cookie.GetDistanceUnit(c)
	state := cookie.GetSearchState(c)
	if fromForm {
		state.Q = strings.TrimSpace(c.Query("q"))
		state.Location = strings.TrimSpace(c.Query("location"))
		state.Within = parseWithin(c.Query("within"), distanceUnit)
		if state.Location == "" {
			state.Within = 0
		}
		if state.Expanded {
			category := ad.GetCategory(cookie.GetCategoryID(c))
			state.Facets = parseFacetFilters(c, category)
		}
	}
	if expanded != nil {
		state.Expanded = *expanded
	}
	cookie.SetSearchState(c, state)
	return state
}

func searchVisible(state cookie.SearchState) bool {
	if state.SearchOpen {
		return true
	}
	if strings.TrimSpace(state.Q) != "" || state.Expanded || len(state.Facets) > 0 {
		return true
	}
	return false
}

func parseFacetFilters(c *fiber.Ctx,
	category ad.Category) map[string]facet.Filter {
	filters := map[string]facet.Filter{}
	for _, d := range category.Facets() {
		if !d.Filterable {
			continue
		}
		switch d.Filter {
		case facet.FilterExact:
			val := strings.TrimSpace(c.Query(d.Key))
			if val != "" && d.ValidFilterValue(val) {
				filters[d.Key] = facet.Filter{Value: &val}
			}
		case facet.FilterCheckboxes:
			if vals := parseEnumCheckboxQuery(c, d.Key, d.Enum); len(vals) > 0 {
				filters[d.Key] = facet.Filter{Values: vals}
			}
		case facet.FilterFlag:
			if c.Query(d.Key) == "1" {
				one := "1"
				filters[d.Key] = facet.Filter{Value: &one}
			}
		default:
			if d.Kind == facet.Date {
				min := parseOptionalDate(c.Query(d.Key + "_min"))
				max := parseOptionalDate(c.Query(d.Key + "_max"))
				if min != nil || max != nil {
					filters[d.Key] = facet.Filter{TextMin: min, TextMax: max}
				}
				continue
			}
			min := parseOptionalAmount(c.Query(d.Key + "_min"))
			max := parseOptionalAmount(c.Query(d.Key + "_max"))
			if min != nil || max != nil {
				filters[d.Key] = facet.Filter{Min: min, Max: max}
			}
		}
	}
	if len(filters) == 0 {
		return nil
	}
	return filters
}

func expandFacetFilters(filters map[string]facet.Filter,
	tz *time.Location) map[string]facet.Filter {
	if len(filters) == 0 {
		return filters
	}
	now := time.Now().In(tz)
	out := make(map[string]facet.Filter, len(filters))
	for k, f := range filters {
		out[k] = facet.ResolveFilterForSearch(k, f, now)
	}
	return out
}

func parseEnumCheckboxQuery(c *fiber.Ctx, key string, allowed []string) []string {
	allowedSet := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = true
	}
	var vals []string
	c.Context().QueryArgs().VisitAll(func(k, v []byte) {
		if string(k) != key {
			return
		}
		s := strings.TrimSpace(string(v))
		if s == "" || !allowedSet[s] {
			return
		}
		for _, existing := range vals {
			if existing == s {
				return
			}
		}
		vals = append(vals, s)
	})
	return vals
}

func parseOptionalDate(raw string) *string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", raw); err != nil {
		return nil
	}
	return &raw
}

func parseOptionalAmount(raw string) *int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	amount, err := strconv.Atoi(raw)
	if err != nil || amount < 0 {
		return nil
	}
	return &amount
}

func parseWithin(raw, distanceUnit string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	opts := search.WithinMileOptions
	if distanceUnit == location.UnitKm {
		opts = search.WithinKmOptions
	}
	for _, opt := range opts {
		if n == opt {
			return n
		}
	}
	return 0
}

func clearFacetFilters(state cookie.SearchState,
	categoryID int) cookie.SearchState {
	category := ad.GetCategory(categoryID)
	allowed := make(map[string]bool, len(category.FacetKeys))
	for _, d := range category.Facets() {
		allowed[d.Key] = true
	}
	for key := range state.Facets {
		if !allowed[key] {
			delete(state.Facets, key)
		}
	}
	if len(state.Facets) == 0 {
		state.Facets = nil
	}
	return state
}

// SaveSearchStateForTest exposes saveSearchStateFromRequest for integration tests.
func SaveSearchStateForTest(c *fiber.Ctx, fromForm bool) cookie.SearchState {
	return saveSearchStateFromRequest(c, nil, fromForm)
}

// BuildSearchParamsForTest exposes parseSearchParams for integration tests.
func BuildSearchParamsForTest(c *fiber.Ctx, state cookie.SearchState,
	categoryID int) search.Params {
	return parseSearchParams(
		state, c.QueryInt("page", 1), categoryID,
		local.GetUserID(c), cookie.GetDistanceUnit(c), cookie.GetTimezone(c),
	)
}
