package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/currency"
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
		filterCats[i] = uiads.CategoryOption{
			ID: cat.ID, Name: cat.Name, ImageFile: cat.ImageFile,
		}
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

// filterableFacetsForSearch returns filter pills for a category, in facet
// registry order. That order is shared across categories, so a pill stays
// put when the category changes. categoryID 0 is the facets every category
// has. Category JSON order is the new-ad form, not the pill rank.
func filterableFacetsForSearch(categoryID int) []facet.Def {
	allowed := map[string]bool{}
	if categoryID > 0 {
		for _, key := range ad.GetCategory(categoryID).FacetKeys {
			allowed[key] = true
		}
		return filterableInRegistryOrder(allowed)
	}
	cats, _ := ad.GetCategories()
	if len(cats) == 0 {
		return nil
	}
	for i, cat := range cats {
		keys := map[string]bool{}
		for _, key := range cat.FacetKeys {
			keys[key] = true
		}
		if i == 0 {
			allowed = keys
			continue
		}
		for key := range allowed {
			if !keys[key] {
				delete(allowed, key)
			}
		}
	}
	return filterableInRegistryOrder(allowed)
}

func filterableInRegistryOrder(allowed map[string]bool) []facet.Def {
	out := make([]facet.Def, 0, len(allowed))
	for _, d := range facet.All() {
		if !d.Filterable || !allowed[d.Key] {
			continue
		}
		out = append(out, d)
	}
	return out
}

func parseSearchParams(state cookie.SearchState, page, userID int,
	distanceUnit string) search.Params {
	limit := config.SearchPageSize
	offset := (page - 1) * limit

	p := search.BuildParams(
		state.CategoryID, limit, offset,
		state.Q, state.Location, state.Within, distanceUnit,
		state.Facets,
	)
	p.UserID = userID
	return p
}

// saveSearchStateFromRequest updates the search cookie and returns the new
// state (the request cookie is unchanged until the response is received).
func saveSearchStateFromRequest(c *fiber.Ctx,
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
	if queryHas(c, "category") {
		newCatID := parseCategoryFilter(c.Query("category"))
		if newCatID != state.CategoryID {
			state.CategoryID = newCatID
			state = clearFacetFilters(state, newCatID)
		} else if fromForm {
			state.Facets = parseFacetFilters(c,
				filterableFacetsForSearch(state.CategoryID))
		}
	} else if fromForm {
		state.Facets = parseFacetFilters(c,
			filterableFacetsForSearch(state.CategoryID))
	}
	cookie.SetSearchState(c, state)
	return state
}

// applyFilterModalSave overlays one facet from the modal form onto cookie state.
func applyFilterModalSave(c *fiber.Ctx, d facet.Def) cookie.SearchState {
	state := cookie.GetSearchState(c)
	if queryHas(c, "q") {
		state.Q = strings.TrimSpace(c.Query("q"))
	}
	parsed := parseFacetFilters(c, []facet.Def{d})
	if state.Facets == nil {
		state.Facets = map[string]facet.Filter{}
	}
	if f, ok := parsed[d.Key]; ok {
		state.Facets[d.Key] = f
	} else {
		delete(state.Facets, d.Key)
	}
	if len(state.Facets) == 0 {
		state.Facets = nil
	}
	state = clearFacetFilters(state, state.CategoryID)
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
			if d.Kind == facet.Money && c.Query(d.Key+"_wanted") == "1" {
				wanted := currency.WantedFilter
				filters[d.Key] = facet.Filter{Value: &wanted}
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
	return saveSearchStateFromRequest(c, fromForm)
}

// BuildSearchParamsForTest exposes parseSearchParams for integration tests.
func BuildSearchParamsForTest(c *fiber.Ctx,
	state cookie.SearchState) search.Params {
	return parseSearchParams(
		state, c.QueryInt("page", 1),
		local.GetUserID(c), cookie.GetDistanceUnit(c),
	)
}
