package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/ui"
)

func HomeHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	searchState := cookie.GetSearchState(c)
	distanceUnit := cookie.GetDistanceUnit(c)

	_, results, err := searchAndRender(c, searchState)
	if err != nil {
		return err
	}

	return renderPage(c, "",
		ui.HomePage(userID, searchState.Q, searchState.Expanded,
			filterFacetsForExpanded(searchState.CategoryID, searchState.Expanded),
			searchStateToFilters(searchState, distanceUnit),
			results))
}
