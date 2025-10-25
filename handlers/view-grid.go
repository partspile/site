package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/ui"
)

// GridView implements the View interface for grid view
type GridView struct {
	ctx *fiber.Ctx
}

// NewGridView creates a new grid view
func NewGridView(ctx *fiber.Ctx) *GridView {
	return &GridView{ctx: ctx}
}

func (v *GridView) GetAdIDs() ([]int, uint64, error) {
	return getAdIDs(v.ctx)
}

func (v *GridView) RenderSearchResults(adIDs []int, cursor uint64) error {
	q := v.ctx.Query("q")
	userID := getUserID(v.ctx)
	loc := getLocation(v.ctx)

	// Convert ad IDs to full ad objects for UI rendering
	ads, err := ad.GetAdsByIDs(adIDs, userID)
	if err != nil {
		return err
	}

	// Create loader URL for infinite scroll
	loaderURL := ui.SearchCreateLoaderURL(q, cursor)

	return render(v.ctx, ui.GridViewResults(ads, userID, loc, loaderURL))
}

func (v *GridView) RenderSearchPage(adIDs []int, cursor uint64) error {
	q := v.ctx.Query("q")
	userID := getUserID(v.ctx)
	loc := getLocation(v.ctx)

	// Convert ad IDs to full ad objects for UI rendering
	ads, err := ad.GetAdsByIDs(adIDs, userID)
	if err != nil {
		return err
	}
	// Create loader URL for infinite scroll
	loaderURL := ui.SearchCreateLoaderURL(q, cursor)

	return render(v.ctx, ui.GridViewPage(ads, userID, loc, loaderURL))
}
