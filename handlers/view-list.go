package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/ui"
)

// ListView implements the View interface for list view
type ListView struct {
	ctx *fiber.Ctx
}

// NewListView creates a new list view
func NewListView(ctx *fiber.Ctx) *ListView {
	return &ListView{ctx: ctx}
}

func (v *ListView) GetAdIDs() ([]int, string, error) {
	return getAdIDs(v.ctx, nil)
}

func (v *ListView) RenderSearchResults(ads []ad.Ad, nextCursor string) error {
	userPrompt := getQueryParam(v.ctx, "q")
	threshold := getThreshold(v.ctx)
	_, userID := getUser(v.ctx)
	loc := getLocation(v.ctx)

	// Create loader URL for infinite scroll
	loaderURL := ui.SearchCreateLoaderURL(userPrompt, nextCursor, "list", threshold, nil)

	return render(v.ctx, ui.ListViewResults(ads, userID, loc, loaderURL))
}

func (v *ListView) RenderSearchPage(ads []ad.Ad, nextCursor string) error {
	userPrompt := getQueryParam(v.ctx, "q")
	threshold := getThreshold(v.ctx)
	_, userID := getUser(v.ctx)
	loc := getLocation(v.ctx)

	// Create loader URL for infinite scroll
	loaderURL := ui.SearchCreateLoaderURL(userPrompt, nextCursor, "list", threshold, nil)

	return render(v.ctx, ui.ListViewPage(ads, userID, loc, loaderURL))
}
