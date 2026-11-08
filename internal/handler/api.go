package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/param"
	"github.com/partspile/site/internal/ui"
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

func renderFilterPanelResponse(c *fiber.Ctx, state cookie.SearchState) error {
	_, results, err := searchFromRequest(c, state)
	if err != nil {
		return err
	}

	distanceUnit := cookie.GetDistanceUnit(c)
	visible := searchVisible(state)
	filterFacets := filterFacetsForExpanded(state.CategoryID, state.Expanded)
	filters := searchStateToFilters(state, distanceUnit)

	return render(c, g.Group([]g.Node{
		ui.SearchAreaOOB(state.Q, state.Expanded, filterFacets, filters, visible),
		ui.SearchToggleOOB(visible),
		ui.SearchResultsOOB(results),
	}))
}

func ShowFiltersHandler(c *fiber.Ctx) error {
	expanded := true
	// fromForm false preserves cookie q/location when the expand button
	// does not include the search widget; category is still applied when
	// the category select includes the widget and posts "category".
	state := saveSearchStateFromRequest(c, &expanded, false)
	state.SearchOpen = true
	cookie.SetSearchState(c, state)
	return renderFilterPanelResponse(c, state)
}

func HideFiltersHandler(c *fiber.Ctx) error {
	expanded := false
	state := saveSearchStateFromRequest(c, &expanded, true)
	return renderFilterPanelResponse(c, state)
}

func ToggleSearchHandler(c *fiber.Ctx) error {
	state := cookie.GetSearchState(c)
	state.SearchOpen = true
	cookie.SetSearchState(c, state)
	return renderFilterPanelResponse(c, state)
}

func SearchPageHandler(c *fiber.Ctx) error {
	state := saveSearchStateFromRequest(c, nil, true)
	page, results, err := searchFromRequest(c, state)
	if err != nil {
		return err
	}
	return render(c, ui.SearchResultsResponse(page, results))
}
