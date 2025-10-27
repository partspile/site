package handlers

import (
	"fmt"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	"github.com/parts-pile/site/vehicle"
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

func buildSearchFilter(c *fiber.Ctx) *qdrant.Filter {
	var conditions []*qdrant.Condition

	params := extractSearchParams(c)

	ac := cookie.GetAdCategory(c)
	conditions = append(conditions, qdrant.NewMatchInt("ad_category_id", int64(ac)))
	log.Printf("[buildSearchFilters] Added ad_category_id filter: %d", ac)

	// Location condition (geo radius) - only apply if location is provided
	l := params["location"]
	if l != "" {
		rStr := params["radius"]
		var r float64 = 25 // default to 25 miles
		if rStr != "" {
			if _, err := fmt.Sscanf(rStr, "%f", &r); err != nil {
				r = 25 // default to 25 miles if invalid
			}
		}

		lat, lon, err := resolveLocationForFilter(l)
		if err != nil {
			log.Printf("[buildSearchFilters] Error resolving location '%s': %v", l, err)
		} else {
			// Convert miles to meters and create geo radius condition
			rm := r * 1609.34
			geoCondition := qdrant.NewGeoRadius("location", lat, lon, float32(rm))
			conditions = append(conditions, geoCondition)
			log.Printf("[buildSearchFilters] Added location filter: %s (%.6f, %.6f) radius %.0f miles", l, lat, lon, r)
		}
	}

	// Make condition (exact string match)
	make := params["make"]
	if make != "" {
		makeCondition := qdrant.NewMatch("make", make)
		conditions = append(conditions, makeCondition)
		log.Printf("[buildSearchFilters] Added make filter: %s", make)
	}

	// Year condition (range - min/max years converted to keywords)
	minYearStr := params["min_year"]
	maxYearStr := params["max_year"]
	if minYearStr != "" || maxYearStr != "" {
		var minYear, maxYear int
		var hasMin, hasMax bool

		if minYearStr != "" {
			if year, err := strconv.Atoi(minYearStr); err == nil {
				minYear = year
				hasMin = true
			}
		}
		if maxYearStr != "" {
			if year, err := strconv.Atoi(maxYearStr); err == nil {
				maxYear = year
				hasMax = true
			}
		}

		// Convert year range to list of year strings
		var yearList []string
		if hasMin && hasMax {
			// Both min and max specified
			for year := minYear; year <= maxYear; year++ {
				yearList = append(yearList, strconv.Itoa(year))
			}
		} else if hasMin {
			// Only min specified - assume reasonable max (current year + 5)
			currentYear := 2024
			for year := minYear; year <= currentYear+5; year++ {
				yearList = append(yearList, strconv.Itoa(year))
			}
		} else if hasMax {
			// Only max specified - assume reasonable min (1900)
			for year := 1900; year <= maxYear; year++ {
				yearList = append(yearList, strconv.Itoa(year))
			}
		}

		if len(yearList) > 0 {
			yearCondition := qdrant.NewMatchKeywords("years", yearList...)
			conditions = append(conditions, yearCondition)
			log.Printf("[buildSearchFilters] Added year keywords filter: %v", yearList)
		}
	}

	// Price condition (range - min/max price)
	minPriceStr := params["min_price"]
	maxPriceStr := params["max_price"]
	if minPriceStr != "" || maxPriceStr != "" {
		var minPrice, maxPrice *float64

		if minPriceStr != "" {
			if price, err := strconv.ParseFloat(minPriceStr, 64); err == nil {
				minPrice = &price
			}
		}
		if maxPriceStr != "" {
			if price, err := strconv.ParseFloat(maxPriceStr, 64); err == nil {
				maxPrice = &price
			}
		}

		if minPrice != nil || maxPrice != nil {
			priceCondition := qdrant.NewRange("price", &qdrant.Range{
				Gte: minPrice,
				Lte: maxPrice,
			})
			conditions = append(conditions, priceCondition)
			log.Printf("[buildSearchFilters] Added price range filter: min=%v, max=%v", minPrice, maxPrice)
		}
	}

	// Rock count filter - exclude ads with too many rocks
	rockCondition := qdrant.NewRange("rock_count", &qdrant.Range{
		Gte: qdrant.PtrOf(0.0),
		Lte: qdrant.PtrOf(float64(config.MaxRockCount)),
	})
	conditions = append(conditions, rockCondition)
	log.Printf("[buildSearchFilters] Added rock_count filter: 0-%d (integer field)", config.MaxRockCount)

	// Create filter, all conditions MUST match
	filter := &qdrant.Filter{
		Must: conditions,
	}

	log.Printf("[buildSearchFilters] Built filter with %d filter conditions", len(conditions))
	return filter
}

func getAdIDs(c *fiber.Ctx) ([]int, uint64, error) {
	q := c.Query("q")
	cursorStr := c.Query("cursor")
	userID := local.GetUserID(c)
	filter := buildSearchFilter(c)
	threshold := config.QdrantSearchThreshold
	k := config.QdrantSearchPageSize

	cursor := uint64(0)
	if cursorStr != "" {
		var err error
		cursor, err = strconv.ParseUint(cursorStr, 10, 64)
		if err != nil {
			log.Printf("[getAdIDs] Failed to parse cursor offset: %v", err)
			return nil, 0, err
		}
	}

	adIDs, cursor, err := performSearch(q, userID, cursor, threshold, k, filter)

	if err == nil {
		log.Printf("[getAdIDs] ad IDs returned: %d", len(adIDs))
		log.Printf("[getAdIDs] Final ad ID order: %v", adIDs)
	}

	return adIDs, cursor, err
}

func handleSearch(c *fiber.Ctx, view int) error {
	userID := local.GetUserID(c)

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
	params := extractSearchParams(c)

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

	// Extract search parameters from form data
	params := extractSearchParams(c)

	// Return the updated search container
	return render(c, ui.SearchContainer(userID, view, adCategory, params))
}

// HandleFilterMakes handles the make filter dropdown for search filters
func HandleFilterMakes(c *fiber.Ctx) error {
	adCategory := cookie.GetAdCategory(c)
	makes := vehicle.GetMakes(adCategory)

	return render(c, ui.MakeFilterOptions(makes))
}
