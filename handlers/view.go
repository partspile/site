package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/ui"
)

// View interface defines the contract for different view implementations
type View interface {
	// GetAdIDs retrieves ad IDs for this view with appropriate search strategy
	GetAdIDs() ([]int, uint64, error)

	// RenderSearchResults renders the complete search results including container, ads, and pagination
	RenderSearchResults(adIDs []int, cursor uint64) error

	// RenderSearchPage renders just the ads and infinite scroll for pagination
	RenderSearchPage(adIDs []int, cursor uint64) error
}

func HandleListView(c *fiber.Ctx) error {
	cookie.SetView(c, ui.ViewList)
	return handleSearch(c, ui.ViewList)
}

func HandleGridView(c *fiber.Ctx) error {
	cookie.SetView(c, ui.ViewGrid)
	return handleSearch(c, ui.ViewGrid)
}

func HandleTreeView(c *fiber.Ctx) error {
	cookie.SetView(c, ui.ViewTree)
	return handleSearch(c, ui.ViewTree)
}

// NewView creates the appropriate view implementation based on view type
func NewView(ctx *fiber.Ctx, view int) (View, error) {
	switch view {
	case ui.ViewList:
		return NewListView(ctx), nil
	case ui.ViewGrid:
		return NewGridView(ctx), nil
	case ui.ViewTree:
		return NewTreeView(ctx), nil
	default:
		return nil, fmt.Errorf("invalid view: %d", view)
	}
}
