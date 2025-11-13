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
	"github.com/parts-pile/site/spec"
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
		// If embeddings aren't ready yet (e.g., after database rebuild),
		// return empty results instead of an error
		if err.Error() == "no embeddings found for popular ads" {
			logger.Info("Site embeddings not ready yet, returning empty results",
				"component", "siteEmbedding", "error", err)
			return []int{}, 0, nil
		}
		logger.Error("GetSiteEmbedding error",
			"error", err, "component", "siteEmbedding")
		return nil, 0, err
	}
	if embedding == nil {
		logger.Warn("GetSiteEmbedding returned nil embedding, returning empty results",
			"component", "siteEmbedding")
		return []int{}, 0, nil
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

func buildSearchFilter(c *fiber.Ctx) *qdrant.Filter {
	var conditions []*qdrant.Condition

	ac := cookie.GetAdCategory(c)
	conditions = append(conditions, qdrant.NewMatchInt("ad_category_id", int64(ac)))
	logger.Debug("Added ad_category_id filter",
		"component", "buildSearchFilters", "adCategoryID", ac)

	// Get category fields to determine which filters to apply
	fieldConfigs := spec.GetCategoryFields(ac)
	// Add filters for spec fields that exist in this category
	for _, fieldConfig := range fieldConfigs {
		fieldName := fieldConfig.Name
		fieldValue := c.FormValue(fieldName)
		minKey := "min_" + fieldName
		maxKey := "max_" + fieldName
		minValue := c.FormValue(minKey)
		maxValue := c.FormValue(maxKey)

		// Handle fields based on their Qdrant type
		switch fieldConfig.QdrantType {
		case "keyword":
			// Single-value keyword fields
			if fieldValue != "" {
				condition := qdrant.NewMatch(fieldName, fieldValue)
				conditions = append(conditions, condition)
				logger.Debug("Added keyword filter",
					"component", "buildSearchFilters", "field", fieldName, "value", fieldValue)
			}
		case "keyword_array":
			// Multi-value keyword fields - check for range or single value
			if minValue != "" || maxValue != "" {
				// Range filter for integer keyword_array fields (like year)
				if fieldConfig.FieldType == "int" {
					var minVal, maxVal int
					var hasMin, hasMax bool

					if minValue != "" {
						if val, err := strconv.Atoi(minValue); err == nil {
							minVal = val
							hasMin = true
						}
					}
					if maxValue != "" {
						if val, err := strconv.Atoi(maxValue); err == nil {
							maxVal = val
							hasMax = true
						}
					}

					// Convert range to list of values
					var valueList []string
					if hasMin && hasMax {
						for val := minVal; val <= maxVal; val++ {
							valueList = append(valueList, strconv.Itoa(val))
						}
					} else if hasMin {
						// Default max to current year + 5 for year fields, or reasonable max
						maxVal = 2030
						if fieldName == "year" {
							maxVal = 2029
						}
						for val := minVal; val <= maxVal; val++ {
							valueList = append(valueList, strconv.Itoa(val))
						}
					} else if hasMax {
						// Default min to 1900 for year fields, or reasonable min
						minVal = 1900
						if fieldName == "year" {
							minVal = 1900
						}
						for val := minVal; val <= maxVal; val++ {
							valueList = append(valueList, strconv.Itoa(val))
						}
					}

					if len(valueList) > 0 {
						condition := qdrant.NewMatchKeywords(fieldName, valueList...)
						conditions = append(conditions, condition)
						logger.Debug("Added range keywords filter",
							"component", "buildSearchFilters", "field", fieldName, "valueList", valueList)
					}
				}
			} else if fieldValue != "" {
				// Single value for keyword_array fields
				condition := qdrant.NewMatchKeywords(fieldName, fieldValue)
				conditions = append(conditions, condition)
				logger.Debug("Added keyword_array filter",
					"component", "buildSearchFilters", "field", fieldName, "value", fieldValue)
			}
		case "integer":
			// Integer fields - could add range support here if needed
			if fieldValue != "" {
				logger.Debug("Integer field filter not yet implemented",
					"component", "buildSearchFilters", "field", fieldName)
			}
		case "float":
			// Float fields - could add range support here if needed
			if fieldValue != "" {
				logger.Debug("Float field filter not yet implemented",
					"component", "buildSearchFilters", "field", fieldName)
			}
		}
	}

	// Location condition (geo radius) - only apply if location is provided
	l := c.FormValue("location")
	if l != "" {
		rStr := c.FormValue("radius")
		var r float64 = 25 // default to 25 miles
		if rStr != "" {
			if _, err := fmt.Sscanf(rStr, "%f", &r); err != nil {
				r = 25 // default to 25 miles if invalid
			}
		}

		lat, lon, err := resolveLocationForFilter(l)
		if err != nil {
			logger.Debug("Error resolving location",
				"error", err, "component", "buildSearchFilters",
				"location", l)
		} else {
			// Convert miles to meters and create geo radius condition
			rm := r * 1609.34
			geoCondition := qdrant.NewGeoRadius("location", lat, lon, float32(rm))
			conditions = append(conditions, geoCondition)
			logger.Debug("Added location filter",
				"component", "buildSearchFilters", "location", l,
				"lat", lat, "lon", lon, "radiusMiles", r)
		}
	}

	// Price condition (range - min/max price)
	minPriceStr := c.FormValue("min_price")
	maxPriceStr := c.FormValue("max_price")
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
			logger.Debug("Added price range filter",
				"component", "buildSearchFilters", "minPrice",
				minPrice, "maxPrice", maxPrice)
		}
	}

	// Rock count filter - exclude ads with too many rocks
	rockCondition := qdrant.NewRange("rock_count", &qdrant.Range{
		Gte: qdrant.PtrOf(0.0),
		Lte: qdrant.PtrOf(float64(config.MaxRockCount)),
	})
	conditions = append(conditions, rockCondition)
	logger.Debug("Added rock_count filter",
		"component", "buildSearchFilters", "maxRockCount",
		config.MaxRockCount)

	// Create filter, all conditions MUST match
	filter := &qdrant.Filter{
		Must: conditions,
	}

	logger.Debug("Built filter",
		"component", "buildSearchFilters", "conditionCount",
		len(conditions))
	return filter
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
