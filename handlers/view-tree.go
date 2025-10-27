package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/local"
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

func (v *TreeView) GetAdIDs() ([]int, uint64, error) {
	// Check if any filters are applied (search query, location, make, year range, price range)
	hasFilters := v.ctx.Query("q") != "" ||
		v.ctx.Query("location") != "" ||
		v.ctx.Query("make") != "" ||
		v.ctx.Query("min_year") != "" ||
		v.ctx.Query("max_year") != "" ||
		v.ctx.Query("min_price") != "" ||
		v.ctx.Query("max_price") != ""

	if !hasFilters {
		// Browse mode - no search query and no filters, return empty slice for full tree
		return []int{}, 0, nil
	}

	// Search mode - get ad IDs from vector search for tree filtering
	return getAdIDs(v.ctx)
}

func (v *TreeView) RenderSearchResults(adIDs []int, cursor uint64) error {
	userPrompt := v.ctx.Query("q")
	adCat := cookie.GetAdCategory(v.ctx)
	userID := local.GetUserID(v.ctx)
	return render(v.ctx, ui.TreeViewResults(adIDs, adCat, userPrompt, userID))
}

func (v *TreeView) RenderSearchPage(adIDs []int, cursor uint64) error {
	return nil
}
