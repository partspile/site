package handlers

import (
	"github.com/gofiber/fiber/v2"
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

func (v *TreeView) GetAdIDs() ([]int, string, error) {
	// Check if any filters are applied (search query, location, make, year range, price range)
	hasFilters := getQueryParam(v.ctx, "q") != "" ||
		getQueryParam(v.ctx, "location") != "" ||
		getQueryParam(v.ctx, "make") != "" ||
		getQueryParam(v.ctx, "min_year") != "" ||
		getQueryParam(v.ctx, "max_year") != "" ||
		getQueryParam(v.ctx, "min_price") != "" ||
		getQueryParam(v.ctx, "max_price") != ""

	if !hasFilters {
		// Browse mode - no search query and no filters, return empty slice for full tree
		return []int{}, "", nil
	}

	// Search mode - get ad IDs from vector search for tree filtering
	return getAdIDs(v.ctx)
}

func (v *TreeView) RenderSearchResults(adIDs []int, nextCursor string) error {
	userPrompt := getQueryParam(v.ctx, "q")
	category := AdCategory(v.ctx)
	return render(v.ctx, ui.TreeViewResults(adIDs, userPrompt, category))
}

func (v *TreeView) RenderSearchPage(adIDs []int, nextCursor string) error {
	return nil
}
