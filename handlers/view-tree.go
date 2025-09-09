package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/ui"
)

// TreeView implements the View interface for tree view
type TreeView struct {
	ctx *fiber.Ctx
}

// NewTreeView creates a new tree view
func NewTreeView(ctx *fiber.Ctx) *TreeView {
	return &TreeView{ctx: ctx}
}

func (v *TreeView) GetAds() ([]ad.Ad, string, error) {
	return getAds(v.ctx, nil)
}

func (v *TreeView) RenderSearchResults(ads []ad.Ad, nextCursor string) error {
	userPrompt := getQueryParam(v.ctx, "q")
	threshold := getThreshold(v.ctx)
	_, userID := getUser(v.ctx)

	// Create loader URL if there are more results
	loaderURL := ui.SearchCreateLoaderURL(userPrompt, nextCursor, "tree", threshold, nil)

	loc := getLocation(v.ctx)
	return render(v.ctx, ui.TreeViewRenderResults(ads, userID, loc, userPrompt, loaderURL, threshold))
}

func (v *TreeView) RenderSearchPage(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	_, userID := getUser(v.ctx)

	// Create loader URL for infinite scroll
	var loaderURL string
	if nextCursor != "" {
		userPrompt := getQueryParam(v.ctx, "q")
		threshold := getThreshold(v.ctx)
		loaderURL = ui.SearchCreateLoaderURL(userPrompt, nextCursor, "tree", threshold, nil)
	}

	// Render the page content using UI function
	return render(v.ctx, ui.TreeViewRenderPage(ads, userID, loc, loaderURL))
}
