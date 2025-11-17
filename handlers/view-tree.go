package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/spec"
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
	// Check if search query is present
	hasSearchQuery := v.ctx.FormValue("q") != ""

	// Check if any field filters are applied using field implementations
	adCat := cookie.GetAdCategory(v.ctx)
	fields := spec.GetCategoryFields(adCat)
	locationResolver := spec.LocationResolver(ResolveLocation)

	hasFieldFilters := false
	for _, field := range fields {
		condition := field.BuildFilterCondition(v.ctx, locationResolver)
		if condition != nil {
			hasFieldFilters = true
			break
		}
	}

	hasFilters := hasSearchQuery || hasFieldFilters

	if !hasFilters {
		// Browse mode - no search query and no filters, return empty slice for full tree
		return []int{}, 0, nil
	}

	// Search mode - get ad IDs from vector search for tree filtering
	return getAdIDs(v.ctx)
}

func (v *TreeView) RenderSearchResults(adIDs []int, cursor uint64) error {
	q := v.ctx.FormValue("q")
	adCat := cookie.GetAdCategory(v.ctx)
	userID := local.GetUserID(v.ctx)
	csrfToken := local.GetCSRFToken(v.ctx)
	return render(v.ctx, ui.TreeViewResults(adIDs, adCat, q, userID, csrfToken))
}

func (v *TreeView) RenderSearchPage(adIDs []int, cursor uint64) error {
	return nil
}
