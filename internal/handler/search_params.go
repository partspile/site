package handler

import (
	"strconv"
	"strings"

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
	cats := allCategoryOptions()
	filterCats := make([]uiads.CategoryOption, len(cats))
	for i, cat := range cats {
		filterCats[i] = uiads.CategoryOption{ID: cat.ID, Name: cat.Name}
	}
	return uiads.SearchFilters{
		CategoryID:      state.CategoryID,
		Categories:      filterCats,
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

func filterableFacets(defs []facet.Def) []facet.Def {
	out := make([]facet.Def, 0, len(defs))
	for _, d := range defs {
		if d.Filterable {
			out = append(out, d)
		}
	}
	return out
}

// filterableFacetsForSearch returns facets for the filter panel. When
// categoryID is 0 (all categories), it is the intersection of filterable
// facets shared by every category, in registry order.
func filterableFacetsForSearch(categoryID int) []facet.Def {
	if categoryID > 0 {
		return filterableFacets(ad.GetCategory(categoryID).Facets())
	}
	cats, _ := ad.GetCategories()
	if len(cats) == 0 {
		return nil
	}
	common := map[string]bool{}
	for i, cat := range cats {
		keys := map[string]bool{}
		for _, key := range cat.FacetKeys {
			keys[key] = true
		}
		if i == 0 {
			common = keys
			continue
		}
		for key := range common {
			if !keys[key] {
				delete(common, key)
			}
		}
	}
	out := make([]facet.Def, 0, len(common))
	for _, d := range facet.All() {
		if !d.Filterable || !common[d.Key] {
			continue
		}
		out = append(out, d)
	}
	return out
}

func filterFacetsForExpanded(categoryID int, expanded bool) []facet.Def {
	if !expanded {
		return nil
	}
	return filterableFacetsForSearch(categoryID)
}

func parseSearchParams(state cookie.SearchState, page, userID int,
	distanceUnit string) search.Params {
	limit := config.SearchPageSize
	offset := (page - 1) * limit

	categoryID := 0
	var facetFilters map[string]facet.Filter
	if state.Expanded {
		categoryID = state.CategoryID
		facetFilters = state.Facets
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
	}
	// Category select is included whenever #search-widget is posted; parse it
	// even when only expanding filters so a category change refreshes facets.
	if queryHas(c, "category") {
		newCatID := parseCategoryFilter(c.Query("category"))
		if newCatID != state.CategoryID {
			state.CategoryID = newCatID
			state = clearFacetFilters(state, newCatID)
		} else if fromForm && state.Expanded {
			state.Facets = parseFacetFilters(c,
				filterableFacetsForSearch(state.CategoryID))
		}
	} else if fromForm && state.Expanded {
		state.Facets = parseFacetFilters(c,
			filterableFacetsForSearch(state.CategoryID))
	}
	if expanded != nil {
		state.Expanded = *expanded
	}
	cookie.SetSearchState(c, state)
	return state
}

func queryHas(c *fiber.Ctx, key string) bool {
	found := false
	c.Context().QueryArgs().VisitAll(func(k, _ []byte) {
		if string(k) == key {
			found = true
		}
	})
	return found
}

func searchVisible(state cookie.SearchState) bool {
	if state.SearchOpen {
		return true
	}
	if strings.TrimSpace(state.Q) != "" || state.Expanded ||
		state.CategoryID > 0 || len(state.Facets) > 0 {
		return true
	}
	return false
}

func parseCategoryFilter(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" {
		return 0
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0
	}
	cats, _ := ad.GetCategories()
	for _, cat := range cats {
		if cat.ID == id {
			return id
		}
	}
	return 0
}

func parseFacetFilters(c *fiber.Ctx,
	defs []facet.Def) map[string]facet.Filter {
	filters := map[string]facet.Filter{}
	for _, d := range defs {
		if !d.Filterable {
			continue
		}
		switch d.Filter {
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
	allowed := make(map[string]bool)
	for _, d := range filterableFacetsForSearch(categoryID) {
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
func BuildSearchParamsForTest(c *fiber.Ctx,
	state cookie.SearchState) search.Params {
	return parseSearchParams(
		state, c.QueryInt("page", 1),
		local.GetUserID(c), cookie.GetDistanceUnit(c),
	)
}
