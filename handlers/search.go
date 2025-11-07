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

func extractSearchParams(c *fiber.Ctx) map[string]string {
	return map[string]string{
		"q":         c.Query("q"),
		"location":  c.Query("location"),
		"make":      c.Query("make"),
		"min_year":  c.Query("min_year"),
		"max_year":  c.Query("max_year"),
		"model":     c.Query("model"),
		"engine":    c.Query("engine"),
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

func buildSearchFilter(c *fiber.Ctx) *qdrant.Filter {
	var conditions []*qdrant.Condition

	params := extractSearchParams(c)

	ac := cookie.GetAdCategory(c)
	conditions = append(conditions, qdrant.NewMatchInt("ad_category_id", int64(ac)))
	logger.Debug("Added ad_category_id filter",
		"component", "buildSearchFilters", "adCategoryID", ac)

	// Get category fields to determine which filters to apply
	fieldConfigs, err := spec.GetCategoryFields(ac)
	if err != nil {
		logger.Warn("Failed to get category fields for search filter",
			"error", err, "component", "buildSearchFilters", "adCategoryID", ac)
		// Continue with basic filters only
	} else {
		// Add filters for spec fields that exist in this category
		for _, fieldConfig := range fieldConfigs {
			fieldName := fieldConfig.Name
			fieldValue := params[fieldName]

			// Skip if no value provided
			if fieldValue == "" {
				// Special handling for year range (min_year/max_year)
				if fieldConfig.Name == "year" {
					minYearStr := params["min_year"]
					maxYearStr := params["max_year"]
					if minYearStr == "" && maxYearStr == "" {
						continue
					}
					// Handle year range below
				} else {
					continue
				}
			}

			// Handle fields based on their Qdrant type
			switch fieldConfig.QdrantType {
			case "keyword":
				// Single-value keyword fields (make, breed, part_subcategory)
				if fieldValue != "" {
					condition := qdrant.NewMatch(fieldName, fieldValue)
					conditions = append(conditions, condition)
					logger.Debug("Added keyword filter",
						"component", "buildSearchFilters", "field", fieldName, "value", fieldValue)
				}
			case "keyword_array":
				// Multi-value keyword fields (years, models, engines)
				if fieldConfig.Name == "year" {
					// Special handling for year range
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
							for year := minYear; year <= maxYear; year++ {
								yearList = append(yearList, strconv.Itoa(year))
							}
						} else if hasMin {
							currentYear := 2024
							for year := minYear; year <= currentYear+5; year++ {
								yearList = append(yearList, strconv.Itoa(year))
							}
						} else if hasMax {
							for year := 1900; year <= maxYear; year++ {
								yearList = append(yearList, strconv.Itoa(year))
							}
						}

						if len(yearList) > 0 {
							// Use plural form for Qdrant metadata key
							yearCondition := qdrant.NewMatchKeywords("years", yearList...)
							conditions = append(conditions, yearCondition)
							logger.Debug("Added year keywords filter",
								"component", "buildSearchFilters", "yearList", yearList)
						}
					}
				} else {
					// Other keyword_array fields (models, engines)
					if fieldValue != "" {
						// Use plural form for Qdrant metadata key
						pluralName := fieldName + "s"
						condition := qdrant.NewMatchKeywords(pluralName, fieldValue)
						conditions = append(conditions, condition)
						logger.Debug("Added keyword_array filter",
							"component", "buildSearchFilters", "field", fieldName, "value", fieldValue)
					}
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
	}

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
			logger.Debug("Failed to parse cursor offset",
				"error", err, "component", "getAdIDs")
			return nil, 0, err
		}
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
	makes := spec.GetMakes(adCategory)

	return render(c, ui.MakeFilterOptions(makes))
}

// HandleFilterModels handles the model filter dropdown for search filters
func HandleFilterModels(c *fiber.Ctx) error {
	adCategory := cookie.GetAdCategory(c)

	// Get parent field values from query parameters
	makeValue := c.Query("make")
	minYear := c.Query("min_year")
	maxYear := c.Query("max_year")

	// Build parent values map
	parentValues := make(map[string]string)
	if makeValue != "" {
		parentValues["make"] = makeValue
	}
	if minYear != "" || maxYear != "" {
		// For year range, we need to convert to a list
		// For now, use empty string - GetFieldValues will handle it
		// TODO: Handle year range properly for model filtering
	}

	// Get models
	models, err := spec.GetFieldValues(adCategory, "model", parentValues)
	if err != nil {
		logger.Warn("Failed to get models for filter",
			"error", err, "component", "filter-models", "adCategory", adCategory)
		models = []string{}
	}

	return render(c, ui.ModelFilterOptions(models))
}

// HandleFilterEngines handles the engine filter dropdown for search filters
func HandleFilterEngines(c *fiber.Ctx) error {
	adCategory := cookie.GetAdCategory(c)

	// Get parent field values from query parameters
	makeValue := c.Query("make")
	minYear := c.Query("min_year")
	maxYear := c.Query("max_year")
	modelValue := c.Query("model")

	// Build parent values map
	parentValues := make(map[string]string)
	if makeValue != "" {
		parentValues["make"] = makeValue
	}
	if modelValue != "" {
		parentValues["model"] = modelValue
	}
	if minYear != "" || maxYear != "" {
		// For year range, we need to convert to a list
		// For now, use empty string - GetFieldValues will handle it
		// TODO: Handle year range properly for engine filtering
	}

	// Get engines
	engines, err := spec.GetFieldValues(adCategory, "engine", parentValues)
	if err != nil {
		logger.Warn("Failed to get engines for filter",
			"error", err, "component", "filter-engines", "adCategory", adCategory)
		engines = []string{}
	}

	return render(c, ui.EngineFilterOptions(engines))
}
