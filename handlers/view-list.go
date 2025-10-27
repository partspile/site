package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/local"
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

func (v *ListView) GetAdIDs() ([]int, uint64, error) {
	return getAdIDs(v.ctx)
}

func (v *ListView) RenderSearchResults(adIDs []int, cursor uint64) error {
	q := v.ctx.Query("q")
	userID := local.GetUserID(v.ctx)
	userName := local.GetUserName(v.ctx)
	loc := getLocation(v.ctx)

	// Convert ad IDs to full ad objects for UI rendering
	ads, err := ad.GetAdsByIDs(adIDs, userID)
	if err != nil {
		return err
	}

	// Create loader URL for infinite scroll
	loaderURL := ui.SearchCreateLoaderURL(q, cursor)

	return render(v.ctx, ui.ListViewResults(ads, userID, userName, loc, loaderURL))
}

func (v *ListView) RenderSearchPage(adIDs []int, cursor uint64) error {
	q := v.ctx.Query("q")
	userID := local.GetUserID(v.ctx)
	userName := local.GetUserName(v.ctx)
	loc := getLocation(v.ctx)

	// Convert ad IDs to full ad objects for UI rendering
	ads, err := ad.GetAdsByIDs(adIDs, userID)
	if err != nil {
		return err
	}

	// Create loader URL for infinite scroll
	loaderURL := ui.SearchCreateLoaderURL(q, cursor)

	return render(v.ctx, ui.ListViewPage(ads, userID, userName, loc, loaderURL))
}
