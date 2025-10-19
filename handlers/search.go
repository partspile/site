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
	g "maragu.dev/gomponents"
)

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
func userEmbedding(currentUser *user.User, cursor string, threshold float64, k int, filter *qdrant.Filter) ([]int, string, error) {
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
func performSearch(userPrompt string, currentUser *user.User, cursorStr string, threshold float64, k int, filter *qdrant.Filter) ([]int, string, error) {
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
	log.Printf("[performSearch] userPrompt='%s', userID=%d, cursorStr='%s', threshold=%.2f, k=%d, filter=%v", userPrompt, userID, cursorStr, threshold, k, filter)

	if userPrompt != "" {
		return queryEmbedding(userPrompt, cursorStr, threshold, k, filter)
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

	adIDs, nextCursor, err := view.GetAdIDs()
	if err != nil {
		return err
	}

	saveUserSearch(c)
	saveCookieLastView(c, viewType)

	return view.RenderSearchResults(adIDs, nextCursor)
}

func HandleSearchPage(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")

	view, err := NewView(c, c.Query("view", "list"))
	if err != nil {
		return err
	}

	adIDs, nextCursor, err := view.GetAdIDs()
	if err != nil {
		return err
	}

	return view.RenderSearchPage(adIDs, nextCursor)
}

func HandleSearch(c *fiber.Ctx) error {
	// Check if this is a request targeting the search container (from category pills)
	target := c.Get("HX-Target")
	if target == "#searchContainer" {
		// Return the full search container with updated category pills and results
		return handleSearchContainer(c, c.Query("view", "list"))
	}

	// Default behavior - return just the search results
	return handleSearch(c, c.Query("view", "list"))
}

// handleSearchContainer handles requests targeting the search container (from category pills)
func handleSearchContainer(c *fiber.Ctx, viewType string) error {
	// Get category from query param or cookie
	categoryStr := c.Query("category", "")
	var activeAdCategory ad.AdCategory
	if categoryStr != "" {
		activeAdCategory = ad.ParseCategoryFromQuery(categoryStr)
	} else {
		activeAdCategory = getCookieAdCategoryID(c)
	}

	view, err := NewView(c, viewType)
	if err != nil {
		return err
	}

	adIDs, nextCursor, err := view.GetAdIDs()
	if err != nil {
		return err
	}

	// Convert ad IDs to full ad objects for UI rendering
	currentUser, userID := CurrentUser(c)
	ads, err := ad.GetAdsByIDs(adIDs, currentUser)
	if err != nil {
		return err
	}

	// Create loader URL for infinite scroll
	userPrompt := getQueryParam(c, "q")
	loaderURL := ui.SearchCreateLoaderURL(userPrompt, nextCursor, viewType)

	// Save category preference
	saveCookieAdCategoryID(c, activeAdCategory)

	// Render the full search container
	return render(c, ui.SearchPage(userID, userPrompt, ads, getLocation(c), loaderURL, activeAdCategory))
}

// HandleSearchQuery renders a full search page with search widget and results
func HandleSearchQuery(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")

	// Get category from query param or cookie
	categoryStr := c.Query("category", "")
	var activeAdCategory ad.AdCategory
	if categoryStr != "" {
		activeAdCategory = ad.ParseCategoryFromQuery(categoryStr)
	} else {
		activeAdCategory = getCookieAdCategoryID(c)
	}

	view, err := NewView(c, "list")
	if err != nil {
		return err
	}

	adIDs, nextCursor, err := view.GetAdIDs()
	if err != nil {
		return err
	}

	// Convert ad IDs to full ad objects for UI rendering
	currentUser, userID := CurrentUser(c)
	ads, err := ad.GetAdsByIDs(adIDs, currentUser)
	if err != nil {
		return err
	}

	// Create loader URL for infinite scroll
	userPrompt := getQueryParam(c, "q")
	loaderURL := ui.SearchCreateLoaderURL(userPrompt, nextCursor, "list")

	// Save category preference
	saveCookieAdCategoryID(c, activeAdCategory)

	// Render full page with search widget and results
	return render(c, ui.Page(
		"Search Results",
		currentUser,
		c.Path(),
		[]g.Node{
			ui.SearchPage(userID, userPrompt, ads, getLocation(c), loaderURL, activeAdCategory),
		},
	))
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
	currentUser, _ := CurrentUser(c)

	adIDs, nextCursor, err := view.GetAdIDs()
	if err != nil {
		log.Printf("[HandleSearchAPI] Search error: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Search failed",
			"ads":   []ad.Ad{},
		})
	}

	ads, err := ad.GetAdsByIDs(adIDs, currentUser)
	if err != nil {
		log.Printf("[HandleSearchAPI] GetAdsByIDs error: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to retrieve ads",
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
	_, userID := CurrentUser(c)
	saveUserSearchAndQueue(userPrompt, userID)
}

// HandleFiltersShow shows the filters area
func HandleFiltersShow(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")

	// Get view and query parameters
	view := c.Query("view", "list")
	query := c.Query("q", "")
	categoryStr := c.Query("category", "CarParts")
	category := ad.ParseCategoryFromQuery(categoryStr)

	// Return the search form with filters
	return render(c, ui.FiltersShow(view, query, category))
}
