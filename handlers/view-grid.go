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

func (v *GridView) GetAdIDs() ([]int, string, error) {
	return getAdIDs(v.ctx, nil)
}

func (v *GridView) RenderSearchResults(ads []ad.Ad, nextCursor string) error {
	userPrompt := getQueryParam(v.ctx, "q")
	threshold := getThreshold(v.ctx)
	_, userID := getUser(v.ctx)
	loc := getLocation(v.ctx)

	// Create loader URL for infinite scroll
	loaderURL := ui.SearchCreateLoaderURL(userPrompt, nextCursor, "grid", threshold, nil)

	return render(v.ctx, ui.GridViewResults(ads, userID, loc, loaderURL))
}

func (v *GridView) RenderSearchPage(ads []ad.Ad, nextCursor string) error {
	userPrompt := getQueryParam(v.ctx, "q")
	threshold := getThreshold(v.ctx)
	_, userID := getUser(v.ctx)
	loc := getLocation(v.ctx)

	// Create loader URL for infinite scroll
	loaderURL := ui.SearchCreateLoaderURL(userPrompt, nextCursor, "grid", threshold, nil)

	return render(v.ctx, ui.GridViewPage(ads, userID, loc, loaderURL))
}
