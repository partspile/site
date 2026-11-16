package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/facet"
	"github.com/partspile/site/internal/param"
	"github.com/partspile/site/internal/ui"
	uiads "github.com/partspile/site/internal/ui/ads"
	g "maragu.dev/gomponents"
)

func SwitchCategoryHandler(c *fiber.Ctx) error {
	redirect := categorySwitchRedirect(c)

	categoryID := param.GetCategoryID(c)
	cookie.SetCategoryID(c, categoryID)

	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", redirect)
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(redirect, fiber.StatusFound)
}

func ModalRemoveHandler(c *fiber.Ctx) error {
	name := c.Params("name")
	return render(c, g.Group(ui.RemoveModal(name)))
}

func categorySwitchRedirect(c *fiber.Ctx) string {
	if redirect := safeReturnPath(c.Query("return")); redirect != "" {
		return redirect
	}
	return "/"
}

func renderSearchFiltersResponse(c *fiber.Ctx, state cookie.SearchState,
	closeModal string) error {
	_, results, err := searchFromRequest(c, state)
	if err != nil {
		return err
	}

	distanceUnit := cookie.GetDistanceUnit(c)
	filterFacets := filterableFacetsForSearch(state.CategoryID)
	filters := searchStateToFilters(state, distanceUnit)

	nodes := []g.Node{
		ui.SearchAreaOOB(state.Q, filterFacets, filters),
		ui.SearchResultsOOB(results),
	}
	if closeModal != "" {
		nodes = append(nodes,
			ui.RemoveModal(uiads.FilterModalName(closeModal))...)
	}
	return render(c, g.Group(nodes))
}

// SearchFiltersHandler refreshes filter pills and results after a category
// change or flag-pill toggle.
func SearchFiltersHandler(c *fiber.Ctx) error {
	state := saveSearchStateFromRequest(c, true)
	return renderSearchFiltersResponse(c, state, "")
}

func FilterModalHandler(c *fiber.Ctx) error {
	key := c.Params("key")
	d, ok := facet.Get(key)
	if !ok || !d.Filterable || d.Filter == facet.FilterFlag {
		return c.SendStatus(fiber.StatusNotFound)
	}
	state := cookie.GetSearchState(c)
	filter := state.Facets[key]
	return render(c, ui.SearchFilterModal(d, filter))
}

func FilterModalSaveHandler(c *fiber.Ctx) error {
	key := c.Params("key")
	d, ok := facet.Get(key)
	if !ok || !d.Filterable || d.Filter == facet.FilterFlag {
		return c.SendStatus(fiber.StatusNotFound)
	}
	state := applyFilterModalSave(c, d)
	return renderSearchFiltersResponse(c, state, key)
}

func SearchPageHandler(c *fiber.Ctx) error {
	state := saveSearchStateFromRequest(c, true)
	page, results, err := searchFromRequest(c, state)
	if err != nil {
		return err
	}
	return render(c, ui.SearchResultsResponse(page, results))
}
