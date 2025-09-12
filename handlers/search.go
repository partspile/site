package handlers

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/vector"
	"github.com/qdrant/go-client/qdrant"
)

// extractMapBounds extracts geographic bounding box parameters for map view
func extractMapBounds(c *fiber.Ctx) *ui.GeoBounds {
	bounds, _ := parseBounds(
		c.Query("minLat"),
		c.Query("maxLat"),
		c.Query("minLon"),
		c.Query("maxLon"))
	return bounds
}

// runEmbeddingSearch runs vector search with optional filters
func runEmbeddingSearch(embedding []float32, cursor string, currentUser *user.User, threshold float64, k int, filter *qdrant.Filter) ([]ad.Ad, string, error) {
	var ids []int
	var nextCursor string
	var err error

	// Get results with threshold filtering at Qdrant level
	ids, nextCursor, err = vector.QuerySimilarAdIDs(embedding, filter, k, cursor, threshold)

	if err != nil {
		return nil, "", err
	}
	log.Printf("[runEmbeddingSearch] Qdrant returned %d results (threshold: %.2f, k: %d)", len(ids), threshold, k)
	log.Printf("[runEmbeddingSearch] Qdrant result IDs: %v", ids)

	ads, err := ad.GetAdsByIDs(ids, currentUser)
	if err != nil {
		return nil, "", err
	}
	log.Printf("[runEmbeddingSearch] DB fetch returned %d ads", len(ads))

	return ads, nextCursor, nil
}

// Embedding-based search with user query
func queryEmbedding(userPrompt string, currentUser *user.User, cursor string, threshold float64, k int, filter *qdrant.Filter) ([]ad.Ad, string, error) {
	log.Printf("[queryEmbedding] Generating embedding for user query: %s", userPrompt)
	embedding, err := vector.GetQueryEmbedding(userPrompt)
	if err != nil {
		return nil, "", err
	}
	return runEmbeddingSearch(embedding, cursor, currentUser, threshold, k, filter)
}

// Embedding-based search with user embedding
func userEmbedding(currentUser *user.User, cursor string, threshold float64, k int, filter *qdrant.Filter) ([]ad.Ad, string, error) {
	log.Printf("[userEmbedding] called with userID=%d, cursor=%s, threshold=%.2f", currentUser.ID, cursor, threshold)
	embedding, err := vector.GetUserPersonalizedEmbedding(currentUser.ID, false)
	if err != nil {
		log.Printf("[userEmbedding] GetUserPersonalizedEmbedding error: %v", err)
		// If user has no activity, fall back to site-level embedding
		if err.Error() == "no user activity to aggregate" {
			log.Printf("[userEmbedding] User has no activity, falling back to site-level embedding")
			return siteEmbedding(cursor, threshold, k, filter)
		}
		return nil, "", err
	}
	if embedding == nil {
		log.Printf("[userEmbedding] GetUserPersonalizedEmbedding returned nil embedding, falling back to site-level embedding")
		return siteEmbedding(cursor, threshold, k, filter)
	}
	return runEmbeddingSearch(embedding, cursor, currentUser, threshold, k, filter)
}

// Embedding-based search with site-level vector
func siteEmbedding(cursor string, threshold float64, k int, filter *qdrant.Filter) ([]ad.Ad, string, error) {
	log.Printf("[siteEmbedding] called with cursor=%s, threshold=%.2f", cursor, threshold)
	embedding, err := vector.GetSiteEmbedding("default")
	if err != nil {
		log.Printf("[siteEmbedding] GetSiteEmbedding error: %v", err)
		return nil, "", err
	}
	if embedding == nil {
		log.Printf("[siteEmbedding] GetSiteEmbedding returned nil embedding")
		return nil, "", fmt.Errorf("site-level vector unavailable")
	}
	return runEmbeddingSearch(embedding, cursor, nil, threshold, k, filter)
}

// performSearch performs the search based on the user prompt and current user
func performSearch(userPrompt string, currentUser *user.User, cursorStr string, threshold float64, k int, filter *qdrant.Filter) ([]ad.Ad, string, error) {
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
	log.Printf("[performSearch] userPrompt='%s', userID=%d, cursorStr='%s', threshold=%.2f, k=%d, filter=%v", userPrompt, userID, cursorStr, threshold, k, filter)

	if userPrompt != "" {
		return queryEmbedding(userPrompt, currentUser, cursorStr, threshold, k, filter)
	}

	if userPrompt == "" && userID != 0 {
		return userEmbedding(currentUser, cursorStr, threshold, k, filter)
	}

	if userPrompt == "" && userID == 0 {
		return siteEmbedding(cursorStr, threshold, k, filter)
	}

	// This should never be reached, but provide a default return
	return nil, "", nil
}

func handleSearch(c *fiber.Ctx, viewType string) error {
	view, err := NewView(c, viewType)
	if err != nil {
		return err
	}

	ads, nextCursor, err := view.GetAds()
	if err != nil {
		return err
	}

	saveUserSearch(c)
	saveCookieLastView(c, viewType)

	return view.RenderSearchResults(ads, nextCursor)
}

func HandleSearchPage(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")

	view, err := NewView(c, c.Query("view", "list"))
	if err != nil {
		return err
	}

	ads, nextCursor, err := view.GetAds()
	if err != nil {
		return err
	}

	return view.RenderSearchPage(ads, nextCursor)
}

func HandleListView(c *fiber.Ctx) error {
	return handleSearch(c, "list")
}

func HandleGridView(c *fiber.Ctx) error {
	return handleSearch(c, "grid")
}

func HandleTreeView(c *fiber.Ctx) error {
	return handleSearch(c, "tree")
}

func HandleMapView(c *fiber.Ctx) error {
	return handleSearch(c, "map")
}

func HandleSearch(c *fiber.Ctx) error {
	return handleSearch(c, c.Query("view", "list"))
}

// HandleSearchAPI returns search results as JSON for JavaScript consumption
func HandleSearchAPI(c *fiber.Ctx) error {
	view, err := NewView(c, c.Query("view", "list"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid view type: %s", c.Query("view", "list")),
			"ads":   []ad.Ad{},
		})
	}

	ads, nextCursor, err := view.GetAds()
	if err != nil {
		log.Printf("[HandleSearchAPI] Search error: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Search failed",
			"ads":   []ad.Ad{},
		})
	}

	saveUserSearch(c)

	log.Printf("[HandleSearchAPI] ads returned: %d", len(ads))

	// Return JSON response
	return c.JSON(fiber.Map{
		"ads":        ads,
		"nextCursor": nextCursor,
		"count":      len(ads),
	})
}

// saveUserSearch saves user search and queues user for embedding update
func saveUserSearch(c *fiber.Ctx) {
	userPrompt := getQueryParam(c, "q")
	_, userID := getUser(c)
	saveUserSearchAndQueue(userPrompt, userID)
}
