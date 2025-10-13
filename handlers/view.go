package handlers

import (
	"fmt"
	"log"
	"strconv"

	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
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

func HandleListView(c *fiber.Ctx) error {
	return handleSearch(c, "list")
}

func HandleGridView(c *fiber.Ctx) error {
	return handleSearch(c, "grid")
}

func HandleTreeView(c *fiber.Ctx) error {
	return handleSearch(c, "tree")
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
func getAdIDs(ctx *fiber.Ctx) ([]int, string, error) {
	userPrompt := getQueryParam(ctx, "q")
	cursor := getQueryParam(ctx, "cursor")
	currentUser, _ := CurrentUser(ctx)
	filter := buildSearchFilter(ctx)
	threshold := config.QdrantSearchThreshold
	k := config.QdrantSearchPageSize

	adIDs, nextCursor, err := performSearch(userPrompt, currentUser, cursor, threshold, k, filter)

	if err == nil {
		log.Printf("[getAdIDs] ad IDs returned: %d", len(adIDs))
		log.Printf("[getAdIDs] Final ad ID order: %v", adIDs)
	}

	return adIDs, nextCursor, err
}

// buildSearchFilter builds a combined Qdrant filter from all available filter parameters
func buildSearchFilter(ctx *fiber.Ctx) *qdrant.Filter {
	var conditions []*qdrant.Condition

	// Location filter (geo radius) - only apply if location is provided
	locationText := getQueryParam(ctx, "location")
	if locationText != "" {
		radiusStr := getQueryParam(ctx, "radius")
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

	// Make filter (exact string match)
	makeFilter := getQueryParam(ctx, "make")
	if makeFilter != "" {
		makeCondition := qdrant.NewMatch("make", makeFilter)
		conditions = append(conditions, makeCondition)
		log.Printf("[buildSearchFilters] Added make filter: %s", makeFilter)
	}

	// Year filter (range - min/max years converted to keywords)
	minYearStr := getQueryParam(ctx, "min_year")
	maxYearStr := getQueryParam(ctx, "max_year")
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

	// Price filter (range - min/max price)
	minPriceStr := getQueryParam(ctx, "min_price")
	maxPriceStr := getQueryParam(ctx, "max_price")
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

	// If no conditions, return nil (no filter)
	if len(conditions) == 0 {
		return nil
	}

	// Create filter with all conditions
	filter := &qdrant.Filter{
		Must: conditions,
	}

	log.Printf("[buildSearchFilters] Built filter with %d conditions", len(conditions))
	return filter
}

// resolveLocationForFilter resolves a location text to lat/lon coordinates
// Checks database first, then uses Grok if not found, but doesn't store the result
func resolveLocationForFilter(locationText string) (latitude, longitude float64, err error) {
	if locationText == "" {
		return 0, 0, fmt.Errorf("empty location text")
	}

	// First try to find existing location in database
	lat, lon, found, err := ad.GetLocationCoordinates(locationText)
	if err != nil {
		return 0, 0, fmt.Errorf("database error looking up location: %w", err)
	}
	if found {
		return lat, lon, nil
	}

	// If not found in database, use Grok to resolve it but don't store
	loc, err := ad.ResolveLocation(locationText)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to resolve location: %w", err)
	}

	return *loc.Latitude, *loc.Longitude, nil
}

// NewView creates the appropriate view implementation based on view type
func NewView(ctx *fiber.Ctx, viewType string) (View, error) {
	switch viewType {
	case "list":
		return NewListView(ctx), nil
	case "grid":
		return NewGridView(ctx), nil
	case "tree":
		return NewTreeView(ctx), nil
	default:
		return nil, fmt.Errorf("invalid view type: %s", viewType)
	}
}
