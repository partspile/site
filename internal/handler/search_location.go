package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/ui"
	uiads "github.com/partspile/site/internal/ui/ads"
	g "maragu.dev/gomponents"
)

func SearchLocationModalHandler(c *fiber.Ctx) error {
	distanceUnit := cookie.GetDistanceUnit(c)
	state := cookie.GetSearchState(c)
	filters := searchStateToFilters(state, distanceUnit)
	return render(c, ui.SearchLocationModal(filters))
}

func SearchLocationSaveHandler(c *fiber.Ctx) error {
	state := saveSearchStateFromRequest(c, nil, true)

	view, _, results, err := searchFromRequest(c, state)
	if err != nil {
		return err
	}

	distanceUnit := cookie.GetDistanceUnit(c)
	filters := searchStateToFilters(state, distanceUnit)

	nodes := []g.Node{
		uiads.SearchLocationOOB(filters),
		ui.SearchResultsOOB(view, results),
	}
	nodes = append(nodes, ui.RemoveModal("search-location")...)
	return render(c, g.Group(nodes))
}
