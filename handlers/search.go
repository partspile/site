package handlers

import (
	"fmt"
	"log"
	"strconv"

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
func runEmbeddingSearch(embedding []float32, cursor uint64, threshold float64, k int, filter *qdrant.Filter) ([]int, uint64, error) {
	var ids []int
	var err error

	// Get results with threshold filtering at Qdrant level
	ids, cursor, err = vector.QuerySimilarAdIDs(embedding, filter, k, cursor, threshold)

	if err != nil {
		return nil, 0, err
	}
	log.Printf("[runEmbeddingSearch] Qdrant returned %d results (threshold: %.2f, k: %d)", len(ids), threshold, k)
	log.Printf("[runEmbeddingSearch] Qdrant result IDs: %v", ids)

	return ids, cursor, nil
}

// Embedding-based search with user query
func queryEmbedding(userPrompt string, cursor uint64, threshold float64, k int, filter *qdrant.Filter) ([]int, uint64, error) {
	log.Printf("[queryEmbedding] Generating embedding for user query: %s", userPrompt)
	embedding, err := vector.GetQueryEmbedding(userPrompt)
	if err != nil {
		return nil, 0, err
	}
	return runEmbeddingSearch(embedding, cursor, threshold, k, filter)
}

// Embedding-based search with user embedding
func userEmbedding(userID int, cursor uint64, threshold float64, k int, filter *qdrant.Filter) ([]int, uint64, error) {
	log.Printf("[userEmbedding] called with userID=%d, cursor=%d, threshold=%.2f", userID, cursor, threshold)
	embedding, err := vector.GetUserPersonalizedEmbedding(userID, false)
	if err != nil {
		log.Printf("[userEmbedding] GetUserPersonalizedEmbedding error: %v", err)
		// If user has no activity, fall back to site-level embedding
		if err.Error() == "no user activity to aggregate" {
			log.Printf("[userEmbedding] User has no activity, falling back to site-level embedding")
			return siteEmbedding(cursor, threshold, k, filter)
		}
		return nil, 0, err
	}
	if embedding == nil {
		log.Printf("[userEmbedding] GetUserPersonalizedEmbedding returned nil embedding, falling back to site-level embedding")
		return siteEmbedding(cursor, threshold, k, filter)
	}
	return runEmbeddingSearch(embedding, cursor, threshold, k, filter)
}

// Embedding-based search with site-level vector
func siteEmbedding(cursor uint64, threshold float64, k int, filter *qdrant.Filter) ([]int, uint64, error) {
	log.Printf("[siteEmbedding] called with cursor=%d, threshold=%.2f", cursor, threshold)
	embedding, err := vector.GetSiteEmbedding("default")
	if err != nil {
		log.Printf("[siteEmbedding] GetSiteEmbedding error: %v", err)
		return nil, 0, err
	}
	if embedding == nil {
		log.Printf("[siteEmbedding] GetSiteEmbedding returned nil embedding")
		return nil, 0, fmt.Errorf("site-level vector unavailable")
	}
	return runEmbeddingSearch(embedding, cursor, threshold, k, filter)
}

// performSearch performs the search based on the user prompt and returns IDs
func performSearch(userPrompt string, userID int, cursor uint64, threshold float64, k int, filter *qdrant.Filter) ([]int, uint64, error) {
	log.Printf("[performSearch] userPrompt='%s', userID=%d, cursor=%d, threshold=%.2f, k=%d, filter=%v", userPrompt, userID, cursor, threshold, k, filter)

	if userPrompt != "" {
		return queryEmbedding(userPrompt, cursor, threshold, k, filter)
	}

	if userPrompt == "" && userID != 0 {
		return userEmbedding(userID, cursor, threshold, k, filter)
	}

	if userPrompt == "" && userID == 0 {
		return siteEmbedding(cursor, threshold, k, filter)
	}

	// This should never be reached, but provide a default return
	return nil, 0, nil
}

func handleSearch(c *fiber.Ctx, view int) error {
	userID := getUserID(c)

	v, err := NewView(c, view)
	if err != nil {
		return err
	}

	adIDs, nextCursor, err := v.GetAdIDs()
	if err != nil {
		return err
	}

	if userID != 0 {
		params := extractSearchParams(c)
		saveUserSearchAndQueue(userID, params)
	}

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

// HandleSearchWidget handles the search widget with query parameters
func HandleSearchWidget(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")

	userID := getUserID(c)
	view := cookie.GetView(c)
	adCategory := cookie.GetAdCategory(c)

	// Handle new-ad-category parameter - set cookie if provided
	if newAdCategoryStr := c.Query("new-ad-category"); newAdCategoryStr != "" {
		if newAdCategory, err := strconv.Atoi(newAdCategoryStr); err == nil {
			cookie.SetAdCategory(c, newAdCategory)
			adCategory = newAdCategory
		}
	}

	// Parse show-filters parameter
	showFilters := c.Query("show-filters") == "true"

	// Extract search parameters
	params := extractSearchParams(c)

	// Return the search widget
	return render(c, ui.SearchWidget(userID, view, adCategory, params, showFilters))
}
