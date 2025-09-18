package handlers

import (
	"fmt"
	"log"

	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/search"
	"github.com/parts-pile/site/vector"
	"github.com/qdrant/go-client/qdrant"
)

// View interface defines the contract for different view implementations
type View interface {
	// GetAdIDs retrieves ad IDs for this view with appropriate search strategy
	GetAdIDs() ([]int, string, error)

	// RenderSearchResults renders the complete search results including container, ads, and pagination
	RenderSearchResults(adIDs []int, nextCursor string) error

	// RenderSearchPage renders just the ads and infinite scroll for pagination
	RenderSearchPage(adIDs []int, nextCursor string) error
}

// saveUserSearchAndQueue saves user search and queues user for embedding update
func saveUserSearchAndQueue(userPrompt string, userID int) {
	if userPrompt != "" {
		_ = search.SaveUserSearch(sql.NullInt64{Int64: int64(userID), Valid: userID != 0}, userPrompt)
		if userID != 0 {
			// Queue user for background embedding update
			vector.QueueUserForUpdate(userID)
		}
	}
}

// getAdIDs performs the common ad ID retrieval logic
func getAdIDs(ctx *fiber.Ctx, geoFilter *qdrant.Filter) ([]int, string, error) {
	userPrompt := getQueryParam(ctx, "q")
	cursor := getQueryParam(ctx, "cursor")
	threshold := getThreshold(ctx)
	currentUser, _ := CurrentUser(ctx)

	// Use QdrantSearchInitialK for geo searches, QdrantSearchPageSize for regular searches
	limit := config.QdrantSearchPageSize
	if geoFilter != nil {
		limit = config.QdrantSearchInitialK
	}

	adIDs, nextCursor, err := performSearch(userPrompt, currentUser, cursor, threshold, limit, geoFilter)

	if err == nil {
		log.Printf("[getAdIDs] ad IDs returned: %d", len(adIDs))
		log.Printf("[getAdIDs] Final ad ID order: %v", adIDs)
	}

	return adIDs, nextCursor, err
}

// NewView creates the appropriate view implementation based on view type
func NewView(ctx *fiber.Ctx, viewType string) (View, error) {
	switch viewType {
	case "list":
		return NewListView(ctx), nil
	case "grid":
		return NewGridView(ctx), nil
	case "map":
		return NewMapView(ctx), nil
	case "tree":
		return NewTreeView(ctx), nil
	default:
		return nil, fmt.Errorf("invalid view type: %s", viewType)
	}
}
