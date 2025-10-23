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
	return getAdIDs(v.ctx)
}

func (v *ListView) RenderSearchResults(adIDs []int, nextCursor string) error {
	userPrompt := getQueryParam(v.ctx, "q")
	u := getUser(v.ctx)
	loc := getLocation(v.ctx)

	// Convert ad IDs to full ad objects for UI rendering
	ads, err := ad.GetAdsByIDs(adIDs, u)
	if err != nil {
		return err
	}

	// Create loader URL for infinite scroll
	loaderURL := ui.SearchCreateLoaderURL(userPrompt, nextCursor, "list")

	return render(v.ctx, ui.ListViewResults(ads, u.ID, loc, loaderURL))
}

func (v *ListView) RenderSearchPage(adIDs []int, nextCursor string) error {
	userPrompt := getQueryParam(v.ctx, "q")
	u := getUser(v.ctx)
	loc := getLocation(v.ctx)

	// Convert ad IDs to full ad objects for UI rendering
	ads, err := ad.GetAdsByIDs(adIDs, u)
	if err != nil {
		return err
	}

	// Create loader URL for infinite scroll
	loaderURL := ui.SearchCreateLoaderURL(userPrompt, nextCursor, "list")

	return render(v.ctx, ui.ListViewPage(ads, u.ID, loc, loaderURL))
}
