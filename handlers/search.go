package handlers

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	"github.com/qdrant/go-client/qdrant"
)

// runEmbeddingSearch runs vector search with optional filters
func runEmbeddingSearch(embedding []float32, cursor uint64, threshold float64, k int, filter *qdrant.Filter) ([]int, uint64, error) {
	var ids []int
	var err error

	// Get results with threshold filtering at Qdrant level
	ids, cursor, err = vector.QuerySimilarAdIDs(embedding, filter, k, cursor, threshold)

	if err != nil {
		return nil, 0, err
	}
	logger.Debug("Qdrant returned results",
		"component", "runEmbeddingSearch", "count", len(ids),
		"threshold", threshold, "k", k, "ids", ids)

	return ids, cursor, nil
}

// Embedding-based search with user query
func queryEmbedding(userPrompt string, cursor uint64, threshold float64, k int, filter *qdrant.Filter) ([]int, uint64, error) {
	logger.Debug("Generating embedding for user query",
		"component", "queryEmbedding", "userPrompt", userPrompt)
	embedding, err := vector.GetQueryEmbedding(userPrompt)
	if err != nil {
		return nil, 0, err
	}
	return runEmbeddingSearch(embedding, cursor, threshold, k, filter)
}

// Embedding-based search with user embedding
func userEmbedding(userID int, cursor uint64, threshold float64, k int, filter *qdrant.Filter) ([]int, uint64, error) {
	logger.Debug("userEmbedding called",
		"component", "userEmbedding", "userID", userID,
		"cursor", cursor, "threshold", threshold)
	embedding, err := vector.GetUserPersonalizedEmbedding(userID, false)
	if err != nil {
		logger.Debug("GetUserPersonalizedEmbedding error",
			"error", err, "component", "userEmbedding",
			"userID", userID)
		// If user has no activity, fall back to site-level embedding
		if err.Error() == "no user activity to aggregate" {
			logger.Debug("User has no activity, falling back to site-level embedding",
				"component", "userEmbedding", "userID", userID)
			return siteEmbedding(cursor, threshold, k, filter)
		}
		return nil, 0, err
	}
	if embedding == nil {
		logger.Debug("GetUserPersonalizedEmbedding returned nil embedding, falling back to site-level embedding",
			"component", "userEmbedding", "userID", userID)
		return siteEmbedding(cursor, threshold, k, filter)
	}
	return runEmbeddingSearch(embedding, cursor, threshold, k, filter)
}

// Embedding-based search with site-level vector
func siteEmbedding(cursor uint64, threshold float64, k int, filter *qdrant.Filter) ([]int, uint64, error) {
	logger.Debug("siteEmbedding called",
		"component", "siteEmbedding", "cursor", cursor,
		"threshold", threshold)
	embedding, err := vector.GetSiteEmbedding("default")
	if err != nil {
		logger.Error("GetSiteEmbedding error",
			"error", err, "component", "siteEmbedding")
		return nil, 0, err
	}
	if embedding == nil {
		logger.Warn("GetSiteEmbedding returned nil embedding",
			"component", "siteEmbedding")
		return nil, 0, fmt.Errorf("site-level vector unavailable")
	}
	return runEmbeddingSearch(embedding, cursor, threshold, k, filter)
}

// performSearch performs the search based on the user prompt and returns IDs
func performSearch(userPrompt string, userID int, cursor uint64, threshold float64, k int, filter *qdrant.Filter) ([]int, uint64, error) {
	logger.Debug("performSearch called",
		"component", "performSearch", "userPrompt", userPrompt,
		"userID", userID, "cursor", cursor, "threshold", threshold,
		"k", k, "filter", filter)

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

func parseCursor(cursorStr string) (uint64, error) {
	if cursorStr == "" {
		return 0, nil
	}
	cursor, err := strconv.ParseUint(cursorStr, 10, 64)
	if err != nil {
		logger.Debug("Failed to parse cursor offset",
			"error", err, "component", "parseCursor")
		return 0, err
	}
	return cursor, nil
}

func getAdIDs(c *fiber.Ctx) ([]int, uint64, error) {
	q := c.FormValue("q")
	cursorStr := c.FormValue("cursor")
	userID := local.GetUserID(c)
	filter := buildSearchFilter(c)
	threshold := config.QdrantSearchThreshold
	k := config.QdrantSearchPageSize

	cursor, err := parseCursor(cursorStr)
	if err != nil {
		return nil, 0, err
	}

	adIDs, cursor, err := performSearch(q, userID, cursor, threshold, k, filter)

	if err == nil {
		logger.Debug("ad IDs returned",
			"component", "getAdIDs", "count", len(adIDs),
			"adIDs", adIDs)
	}

	return adIDs, cursor, err
}

func handleSearch(c *fiber.Ctx, view int) error {
	v, err := NewView(c, view)
	if err != nil {
		return err
	}

	adIDs, nextCursor, err := v.GetAdIDs()
	if err != nil {
		return err
	}

	return v.RenderSearchResults(adIDs, nextCursor)
}

func HandleSearch(c *fiber.Ctx) error {
	view := cookie.GetView(c)
	return handleSearch(c, view)
}

func HandleSearchPage(c *fiber.Ctx) error {
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
	userID := local.GetUserID(c)
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
	params := c.Queries()

	// Return the search widget
	return render(c, ui.SearchWidget(userID, view, adCategory, params, showFilters))
}

// HandleAdCategoryModal shows the ad category selector modal
func HandleAdCategoryModal(c *fiber.Ctx) error {
	adCategory := cookie.GetAdCategory(c)

	// Return the category modal
	return render(c, ui.AdCategoryModal(adCategory))
}

// HandleSwitchAdCategory switches the ad category and returns updated search container
func HandleSwitchAdCategory(c *fiber.Ctx) error {
	// Extract ad category from URL parameter
	adCategoryStr := c.Params("adCategory")
	adCategory, err := strconv.Atoi(adCategoryStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ad category")
	}

	// Validate that the category exists
	if !ad.IsValidCategory(adCategory) {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid ad category")
	}

	// Set the new ad category cookie
	cookie.SetAdCategory(c, adCategory)

	// Get current user and view settings
	userID := local.GetUserID(c)
	view := cookie.GetView(c)

	// Clear form parameters when switching categories
	params := make(map[string]string)

	// Return the updated search container
	return render(c, ui.SearchContainer(userID, view, adCategory, params))
}
