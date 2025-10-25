package handlers

import (
	"fmt"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/search"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	"github.com/qdrant/go-client/qdrant"
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
	return handleSearch(c, ui.ViewList)
}

func HandleGridView(c *fiber.Ctx) error {
	return handleSearch(c, ui.ViewGrid)
}

func HandleTreeView(c *fiber.Ctx) error {
	return handleSearch(c, ui.ViewTree)
}

// saveUserSearchAndQueue saves user search and queues user for embedding update
func saveUserSearchAndQueue(userID int, params map[string]string) {
	q := params["q"]
	if q != "" {
		_ = search.SaveUserSearch(userID, q)
		// Queue user for background embedding update
		vector.QueueUserForUpdate(userID)
	}
}

// getAdIDs performs the common ad ID retrieval logic
func getAdIDs(ctx *fiber.Ctx) ([]int, uint64, error) {
	q := ctx.Query("q")
	cursorStr := ctx.Query("cursor")
	userID := getUserID(ctx)
	filter := buildSearchFilter(ctx)
	threshold := config.QdrantSearchThreshold
	k := config.QdrantSearchPageSize

	cursor, err := strconv.ParseUint(cursorStr, 10, 64)
	if err != nil {
		log.Printf("[getAdIDs] Failed to parse cursor offset: %v", err)
		return nil, 0, err
	}

	adIDs, cursor, err := performSearch(q, userID, cursor, threshold, k, filter)

	if err == nil {
		log.Printf("[getAdIDs] ad IDs returned: %d", len(adIDs))
		log.Printf("[getAdIDs] Final ad ID order: %v", adIDs)
	}

	return adIDs, cursor, err
}

// buildSearchFilter builds a combined Qdrant filter from all available filter parameters
func buildSearchFilter(c *fiber.Ctx) *qdrant.Filter {
	var conditions []*qdrant.Condition
	params := extractSearchParams(c)

	// Ad category condition (required filter)
	adCat := AdCategory(c)
	adCategoryCondition := qdrant.NewMatchInt("ad_category_id", int64(adCat))
	conditions = append(conditions, adCategoryCondition)
	log.Printf("[buildSearchFilters] Added ad_category_id filter: %d", adCat)

	// Location condition (geo radius) - only apply if location is provided
	locationText := params["location"]
	if locationText != "" {
		radiusStr := params["radius"]
		var radius float64 = 25 // default to 25 miles
		if radiusStr != "" {
			if _, err := fmt.Sscanf(radiusStr, "%f", &radius); err != nil {
				radius = 25 // default to 25 miles if invalid
			}
		}

		lat, lon, err := resolveLocationForFilter(locationText)
		if err != nil {
			log.Printf("[buildSearchFilters] Error resolving location '%s': %v", locationText, err)
		} else {
			// Convert miles to meters and create geo radius condition
			radiusMeters := radius * 1609.34
			geoCondition := qdrant.NewGeoRadius("location", lat, lon, float32(radiusMeters))
			conditions = append(conditions, geoCondition)
			log.Printf("[buildSearchFilters] Added location filter: %s (%.6f, %.6f) radius %.0f miles", locationText, lat, lon, radius)
		}
	}

	// Make condition (exact string match)
	makeFilter := params["make"]
	if makeFilter != "" {
		makeCondition := qdrant.NewMatch("make", makeFilter)
		conditions = append(conditions, makeCondition)
		log.Printf("[buildSearchFilters] Added make filter: %s", makeFilter)
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

	// Category condition (exact string match)
	categoryFilter := c.Query("ad_category")
	if categoryFilter != "" {
		categoryCondition := qdrant.NewMatch("category", categoryFilter)
		conditions = append(conditions, categoryCondition)
		log.Printf("[buildSearchFilters] Added category filter: %s", categoryFilter)
	}

	// Create filter with all filter conditions (always has ad_category)
	filter := &qdrant.Filter{
		Must: conditions,
	}

	log.Printf("[buildSearchFilters] Built filter with %d filter conditions", len(conditions))
	return filter
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
