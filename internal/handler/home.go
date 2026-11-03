package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/ui"
)

func HomeHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	view := ui.ValidateView(cookie.GetView(c))
	searchState := cookie.GetSearchState(c)
	distanceUnit := cookie.GetDistanceUnit(c)

	_, results, err := searchAndRender(c, searchState, view)
	if err != nil {
		return err
	}

	return renderPage(c, config.ServerName,
		ui.HomePage(userID, view, searchState.Q, searchState.Expanded,
			searchVisible(searchState),
			filterFacetsForExpanded(searchState.CategoryID, searchState.Expanded),
			searchStateToFilters(searchState, distanceUnit),
			results))
}
