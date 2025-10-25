package handlers

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	"github.com/qdrant/go-client/qdrant"
)

func extractSearchParams(c *fiber.Ctx) map[string]string {
	return map[string]string{
		"q":         c.Query("q"),
		"location":  c.Query("location"),
		"make":      c.Query("make"),
		"min_year":  c.Query("min_year"),
		"max_year":  c.Query("max_year"),
		"min_price": c.Query("min_price"),
		"max_price": c.Query("max_price"),
	}
}

// runEmbeddingSearch runs vector search with optional filters
func runEmbeddingSearch(embedding []float32, cursor string, threshold float64, k int, filter *qdrant.Filter) ([]int, string, error) {
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

	return ids, nextCursor, nil
}

// Embedding-based search with user query
func queryEmbedding(userPrompt string, cursor string, threshold float64, k int, filter *qdrant.Filter) ([]int, string, error) {
	log.Printf("[queryEmbedding] Generating embedding for user query: %s", userPrompt)
	embedding, err := vector.GetQueryEmbedding(userPrompt)
	if err != nil {
		return nil, "", err
	}
	return runEmbeddingSearch(embedding, cursor, threshold, k, filter)
}

// Embedding-based search with user embedding
func userEmbedding(userID int, cursor string, threshold float64, k int, filter *qdrant.Filter) ([]int, string, error) {
	log.Printf("[userEmbedding] called with userID=%d, cursor=%s, threshold=%.2f", userID, cursor, threshold)
	embedding, err := vector.GetUserPersonalizedEmbedding(userID, false)
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
	return runEmbeddingSearch(embedding, cursor, threshold, k, filter)
}

// Embedding-based search with site-level vector
func siteEmbedding(cursor string, threshold float64, k int, filter *qdrant.Filter) ([]int, string, error) {
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
	return runEmbeddingSearch(embedding, cursor, threshold, k, filter)
}

// performSearch performs the search based on the user prompt and returns IDs
func performSearch(userPrompt string, userID int, cursorStr string, threshold float64, k int, filter *qdrant.Filter) ([]int, string, error) {
	log.Printf("[performSearch] userPrompt='%s', userID=%d, cursorStr='%s', threshold=%.2f, k=%d, filter=%v", userPrompt, userID, cursorStr, threshold, k, filter)

	if userPrompt != "" {
		return queryEmbedding(userPrompt, cursorStr, threshold, k, filter)
	}

	if userPrompt == "" && userID != 0 {
		return userEmbedding(userID, cursorStr, threshold, k, filter)
	}

	if userPrompt == "" && userID == 0 {
		return siteEmbedding(cursorStr, threshold, k, filter)
	}

	// This should never be reached, but provide a default return
	return nil, "", nil
}

func handleSearch(c *fiber.Ctx, view int) error {
	userID := getUserID(c)
	params := extractSearchParams(c)

	v, err := NewView(c, view)
	if err != nil {
		return err
	}

	adIDs, nextCursor, err := v.GetAdIDs()
	if err != nil {
		return err
	}

	saveUserSearch(userID, params)

	return v.RenderSearchResults(adIDs, nextCursor)
}

func HandleSearch(c *fiber.Ctx) error {
	view := cookie.GetView(c)
	return handleSearch(c, view)
}

func HandleSearchPage(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")

	v, err := NewView(c, cookie.GetView(c))
	if err != nil {
		return err
	}

	adIDs, nextCursor, err := v.GetAdIDs()
	if err != nil {
		return err
	}

	return v.RenderSearchPage(adIDs, nextCursor)
}

// HandleFiltersShow shows the filters area
func HandleFiltersShow(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")

	// Get view and query parameters
	view := c.Query("view", "list")
	query := c.Query("q", "")
	category := AdCategory(c)

	// Return the search form with filters
	return render(c, ui.FiltersShow(view, query, category))
}
