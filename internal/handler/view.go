package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/ui"
)

func ViewHandler(c *fiber.Ctx) error {

	view := ui.ValidateView(c.Params("view"))
	searchState := cookie.GetSearchState(c)
	categoryID := cookie.GetCategoryID(c)

	_, results, err := searchAndRender(c, searchState, categoryID, view)
	if err != nil {
		return err
	}

	cookie.SetView(c, view)

	distanceUnit := cookie.GetDistanceUnit(c)
	filters := searchStateToFilters(searchState, distanceUnit)

	return render(c, ui.SearchView(view, filters, results))
}
