package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/ui"
	g "maragu.dev/gomponents"
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
	adCat := cookie.GetAdCategory(v.ctx)

	if !IsSearching(v.ctx, adCat) {
		// Browse mode - no search query and no filters, return empty slice for full tree
		return []int{}, 0, nil
	}

	// Search mode - get ad IDs from vector search for tree filtering
	return getAdIDs(v.ctx)
}

func (v *TreeView) RenderSearchResults(adIDs []int, cursor uint64) error {
	adCat := cookie.GetAdCategory(v.ctx)
	userID := local.GetUserID(v.ctx)
	csrfToken := local.GetCSRFToken(v.ctx)

	// Check if any filters are applied to differentiate between browse mode and search with no results
	hasFilters := IsSearching(v.ctx, adCat)

	var viewContent g.Node

	if !hasFilters {
		// No filters applied - show full browse tree
		viewContent = ui.TreeBrowseFirstField(adCat)
	} else if len(adIDs) > 0 {
		// Has filters and results - show filtered tree
		viewContent = ui.TreeSearchFirstField(adIDs, adCat)
	} else {
		// Has filters but no results - show no results message
		viewContent = ui.NoSearchResultsMessage()
	}

	return render(v.ctx, ui.TreeViewResults(viewContent, userID, csrfToken))
}

func (v *TreeView) RenderSearchPage(adIDs []int, cursor uint64) error {
	return nil
}
