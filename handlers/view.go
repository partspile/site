package handlers

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/search"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
)

// Track last search query per user to avoid duplicate saves
var lastUserSearch = make(map[int]string)
var lastSearchMutex sync.RWMutex

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

// saveUserSearchAndQueue saves user search and queues user for embedding update
// Only saves if the query is different from the last saved search
func saveUserSearchAndQueue(userID int, params map[string]string) {
	q := params["q"]
	if q == "" {
		return
	}

	// Normalize the query for comparison (case-insensitive, trimmed)
	currentQuery := strings.TrimSpace(strings.ToLower(q))

	// Check if this matches the last saved search for this user
	lastSearchMutex.RLock()
	lastQuery, exists := lastUserSearch[userID]
	lastSearchMutex.RUnlock()

	// Only save if the query is different
	if !exists || currentQuery != lastQuery {
		_ = search.SaveUserSearch(userID, q)
		// Queue user for background embedding update
		vector.QueueUserForUpdate(userID)

		// Update the in-memory map
		lastSearchMutex.Lock()
		lastUserSearch[userID] = currentQuery
		lastSearchMutex.Unlock()

		logger.Debug("Saved new user search",
			"component", "saveUserSearchAndQueue", "userID", userID,
			"query", q)
	} else {
		logger.Debug("Skipping duplicate search save",
			"component", "saveUserSearchAndQueue", "userID", userID,
			"query", q)
	}
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
