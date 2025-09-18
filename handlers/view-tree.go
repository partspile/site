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
	userPrompt := getQueryParam(v.ctx, "q")

	if userPrompt == "" {
		// Browse mode - return empty slice, tree will be built using unfiltered SQL queries
		return []int{}, "", nil
	}

	// Search mode - get ad IDs from vector search for tree filtering
	return getAdIDs(v.ctx, nil)
}

func (v *TreeView) RenderSearchResults(adIDs []int, nextCursor string) error {
	userPrompt := getQueryParam(v.ctx, "q")
	return render(v.ctx, ui.TreeViewResults(adIDs, userPrompt))
}

func (v *TreeView) RenderSearchPage(adIDs []int, nextCursor string) error {
	return nil
}
