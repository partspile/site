package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sort"
	"strings"
	"time"

	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/search"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type SearchQuery = ad.SearchQuery

type SearchCursor = ad.SearchCursor

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper to fetch ads by Qdrant result IDs
func fetchAdsByIDs(ids []string, userID int) ([]ad.Ad, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Convert string IDs to int IDs
	intIDs := make([]int, 0, len(ids))
	for _, idStr := range ids {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}
		intIDs = append(intIDs, id)
	}

	// Fetch all ads in a single optimized query
	var ads []ad.Ad
	var err error
	if userID > 0 {
		// Use the optimized function that includes bookmark status
		ads, err = ad.GetAdsByIDsOptimizedWithBookmarks(intIDs, userID)
	} else {
		// Use the optimized function for anonymous users
		ads, err = ad.GetAdsByIDsOptimized(intIDs)
	}
	if err != nil {
		return nil, err
	}

	log.Printf("[fetchAdsByIDs] Returning ads in order: %v", func() []int {
		result := make([]int, len(ads))
		for i, ad := range ads {
			result[i] = ad.ID
		}
		return result
	}())
	return ads, nil
}

// runEmbeddingSearchWithFilter runs vector search with filters
func runEmbeddingSearchWithFilter(embedding []float32, filter *qdrant.Filter, cursor string, userID int, threshold float64) ([]ad.Ad, string, error) {
	// Get results with threshold filtering at Qdrant level
	results, nextCursor, err := vector.QuerySimilarAdsWithFilter(embedding, filter, config.QdrantSearchPageSize, cursor, threshold)
	if err != nil {
		return nil, "", err
	}
	log.Printf("[runEmbeddingSearchWithFilter] Qdrant returned %d results (threshold: %.2f)", len(results), threshold)

	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	log.Printf("[runEmbeddingSearchWithFilter] Qdrant result IDs: %v", ids)
	ads, _ := fetchAdsByIDs(ids, userID)
	log.Printf("[runEmbeddingSearchWithFilter] DB fetch returned %d ads", len(ads))
	return ads, nextCursor, nil
}

// runEmbeddingSearch runs vector search without filters
func runEmbeddingSearch(embedding []float32, cursor string, userID int, threshold float64) ([]ad.Ad, string, error) {
	// Get results with threshold filtering at Qdrant level
	results, nextCursor, err := vector.QuerySimilarAds(embedding, config.QdrantSearchPageSize, cursor, threshold)
	if err != nil {
		return nil, "", err
	}
	log.Printf("[runEmbeddingSearch] Qdrant returned %d results (threshold: %.2f)", len(results), threshold)

	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	log.Printf("[runEmbeddingSearch] Qdrant result IDs: %v", ids)
	ads, _ := fetchAdsByIDs(ids, userID)
	log.Printf("[runEmbeddingSearch] DB fetch returned %d ads", len(ads))
	return ads, nextCursor, nil
}

// runEmbeddingSearchWithFilterMap runs vector search with filters for map view (200 results)
func runEmbeddingSearchWithFilterMap(embedding []float32, filter *qdrant.Filter, cursor string, userID int, threshold float64) ([]ad.Ad, string, error) {
	// Get results with threshold filtering at Qdrant level
	results, nextCursor, err := vector.QuerySimilarAdsWithFilter(embedding, filter, config.QdrantSearchInitialK, cursor, threshold)
	if err != nil {
		return nil, "", err
	}
	log.Printf("[runEmbeddingSearchWithFilterMap] Qdrant returned %d results (threshold: %.2f)", len(results), threshold)

	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	log.Printf("[runEmbeddingSearchWithFilterMap] Qdrant result IDs: %v", ids)
	ads, _ := fetchAdsByIDs(ids, userID)
	log.Printf("[runEmbeddingSearchWithFilterMap] DB fetch returned %d ads", len(ads))
	return ads, nextCursor, nil
}

// runEmbeddingSearchMap runs vector search without filters for map view (200 results)
func runEmbeddingSearchMap(embedding []float32, cursor string, userID int, threshold float64) ([]ad.Ad, string, error) {
	// Get results with threshold filtering at Qdrant level
	results, nextCursor, err := vector.QuerySimilarAds(embedding, config.QdrantSearchInitialK, cursor, threshold)
	if err != nil {
		return nil, "", err
	}
	log.Printf("[runEmbeddingSearchMap] Qdrant returned %d results (threshold: %.2f)", len(results), threshold)

	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	log.Printf("[runEmbeddingSearchMap] Qdrant result IDs: %v", ids)
	ads, _ := fetchAdsByIDs(ids, userID)
	log.Printf("[runEmbeddingSearchMap] DB fetch returned %d ads", len(ads))
	return ads, nextCursor, nil
}

// Try embedding-based search with user prompt
func tryQueryEmbedding(userPrompt, cursor string, userID int, threshold float64) ([]ad.Ad, string, error) {
	log.Printf("[search] Generating embedding for user query: %s", userPrompt)
	embedding, err := vector.EmbedTextCached(userPrompt)
	if err != nil {
		return nil, "", err
	}
	return runEmbeddingSearch(embedding, cursor, userID, threshold)
}

// Try embedding-based search with user embedding
func tryUserEmbedding(userID int, cursor string, threshold float64) ([]ad.Ad, string, error) {
	embedding, err := vector.GetUserPersonalizedEmbedding(userID, false)
	if err != nil || embedding == nil {
		return nil, "", err
	}
	return runEmbeddingSearch(embedding, cursor, userID, threshold)
}

// Search strategy for both HandleSearch and HandleSearchPage
func performSearch(userPrompt string, userID int, cursorStr string, threshold float64) ([]ad.Ad, string, error) {
	log.Printf("[performSearch] userPrompt='%s', userID=%d, cursorStr='%s', threshold=%.2f", userPrompt, userID, cursorStr, threshold)
	if userPrompt != "" {
		ads, nextCursor, _ := tryQueryEmbedding(userPrompt, cursorStr, userID, threshold)
		log.Printf("[performSearch] tryQueryEmbedding: found %d ads", len(ads))
		if len(ads) > 0 {
			return ads, nextCursor, nil
		}
	}
	if userPrompt == "" && userID != 0 {
		ads, nextCursor, _ := tryUserEmbedding(userID, cursorStr, threshold)
		log.Printf("[performSearch] tryUserEmbedding: found %d ads", len(ads))
		if len(ads) > 0 {
			return ads, nextCursor, nil
		}
		// Fallback to site-level vector for logged-in users with no personalized embedding
		log.Printf("[performSearch] No personalized embedding found for user %d, falling back to site-level vector", userID)
		emb, err := vector.GetSiteLevelVector()
		log.Printf("[performSearch] GetSiteLevelVector returned emb=%v, err=%v", emb != nil, err)
		if err == nil && emb != nil {
			log.Printf("[performSearch] site-level vector length: %d", len(emb))
			if len(emb) > 0 {
				log.Printf("[performSearch] site-level vector first 5 values: %v", emb[:min(5, len(emb))])
			}
			log.Printf("[performSearch] About to call runEmbeddingSearch with site-level vector")
			ads, nextCursor, _ := runEmbeddingSearch(emb, cursorStr, userID, threshold)
			log.Printf("[performSearch] site-level vector: found %d ads", len(ads))
			if len(ads) > 0 {
				return ads, nextCursor, nil
			}
		} else {
			log.Printf("[performSearch] site-level vector error: %v", err)
		}
	}
	if userPrompt == "" && userID == 0 {
		emb, err := vector.GetSiteLevelVector()
		log.Printf("[performSearch] GetSiteLevelVector returned emb=%v, err=%v", emb != nil, err)
		if err == nil && emb != nil {
			log.Printf("[performSearch] site-level vector length: %d", len(emb))
			if len(emb) > 0 {
				log.Printf("[performSearch] site-level vector first 5 values: %v", emb[:min(5, len(emb))])
			}
			log.Printf("[performSearch] About to call runEmbeddingSearch with site-level vector")
			ads, nextCursor, _ := runEmbeddingSearch(emb, cursorStr, userID, threshold)
			log.Printf("[performSearch] site-level vector: found %d ads", len(ads))
			if len(ads) > 0 {
				return ads, nextCursor, nil
			}
		} else {
			log.Printf("[performSearch] site-level vector error: %v", err)
		}
	}
	log.Printf("[performSearch] No ads found for given parameters.")
	return nil, "", nil
}

// GeoBounds represents a geographic bounding box
type GeoBounds struct {
	MinLat float64
	MaxLat float64
	MinLon float64
	MaxLon float64
}

// performGeoBoxSearch performs search with geo bounding box filtering
func performGeoBoxSearch(userPrompt string, userID int, cursorStr string, bounds *GeoBounds, threshold float64) ([]ad.Ad, string, error) {
	log.Printf("[performGeoBoxSearch] userPrompt='%s', userID=%d, cursorStr='%s', bounds=%+v", userPrompt, userID, cursorStr, bounds)

	// Build geo filter if bounds are provided
	var geoFilter *qdrant.Filter
	if bounds != nil {
		geoFilter = vector.BuildBoundingBoxGeoFilter(bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon)
	}

	var ads []ad.Ad
	var nextCursor string
	var err error

	// If no query, use site-level vector + geo filter
	if userPrompt == "" {
		emb, err := vector.GetSiteLevelVector()
		if err != nil {
			log.Printf("[performGeoBoxSearch] GetSiteLevelVector error: %v", err)
			return nil, "", err
		}
		if emb != nil && len(emb) > 0 {
			if geoFilter != nil {
				ads, nextCursor, err = runEmbeddingSearchWithFilterMap(emb, geoFilter, cursorStr, userID, threshold)
			} else {
				ads, nextCursor, err = runEmbeddingSearchMap(emb, cursorStr, userID, threshold)
			}
		}
	}

	// If query provided, use query embedding + geo filter
	if userPrompt != "" {
		embedding, err := vector.EmbedTextCached(userPrompt)
		if err != nil {
			log.Printf("[performGeoBoxSearch] EmbedText error: %v", err)
			return nil, "", err
		}
		if geoFilter != nil {
			ads, nextCursor, err = runEmbeddingSearchWithFilterMap(embedding, geoFilter, cursorStr, userID, threshold)
		} else {
			ads, nextCursor, err = runEmbeddingSearchMap(embedding, cursorStr, userID, threshold)
		}
	}

	if err != nil {
		return nil, "", err
	}

	if len(ads) == 0 {
		log.Printf("[performGeoBoxSearch] No ads found for given parameters.")
		return nil, "", nil
	}

	return ads, nextCursor, nil
}

// Get user ID from context
func getUserID(c *fiber.Ctx) int {
	currentUser, _ := CurrentUser(c)
	if currentUser != nil {
		return currentUser.ID
	}
	return 0
}

// Get location from context
func getLocation(c *fiber.Ctx) *time.Location {
	loc, _ := time.LoadLocation(c.Get("X-Timezone"))
	return loc
}

// Render new ad button based on user login
func renderNewAdButton(c *fiber.Ctx) g.Node {
	currentUser, _ := CurrentUser(c)
	if currentUser != nil {
		return ui.StyledLink("New Ad", "/new-ad", ui.ButtonPrimary)
	}
	return ui.StyledLinkDisabled("New Ad", ui.ButtonPrimary)
}

func HandleSearch(c *fiber.Ctx) error {
	userPrompt := c.Query("q")
	view := c.FormValue("view")
	if view == "" {
		view = "list"
	}

	// Get threshold from query parameter, default to config value
	thresholdStr := c.Query("threshold")
	threshold := config.QdrantSearchThreshold
	if thresholdStr != "" {
		if thresholdVal, err := strconv.ParseFloat(thresholdStr, 32); err == nil {
			threshold = thresholdVal
		}
	}

	userID := getUserID(c)
	log.Printf("[HandleSearch] userPrompt='%s', userID=%d, threshold=%.2f", userPrompt, userID, threshold)

	// Extract bounding box parameters for map view
	var bounds *GeoBounds
	if view == "map" {
		minLatStr := c.Query("minLat")
		maxLatStr := c.Query("maxLat")
		minLonStr := c.Query("minLon")
		maxLonStr := c.Query("maxLon")

		if minLatStr != "" && maxLatStr != "" && minLonStr != "" && maxLonStr != "" {
			minLat, err1 := strconv.ParseFloat(minLatStr, 64)
			maxLat, err2 := strconv.ParseFloat(maxLatStr, 64)
			minLon, err3 := strconv.ParseFloat(minLonStr, 64)
			maxLon, err4 := strconv.ParseFloat(maxLonStr, 64)

			if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
				bounds = &GeoBounds{
					MinLat: minLat,
					MaxLat: maxLat,
					MinLon: minLon,
					MaxLon: maxLon,
				}
				log.Printf("[HandleSearch] Using bounding box: %+v", bounds)
			}
		}
	}

	var ads []ad.Ad
	var nextCursor string
	var err error

	// Use geo search for map view with bounds, regular search otherwise
	if view == "map" && bounds != nil {
		ads, nextCursor, err = performGeoBoxSearch(userPrompt, userID, "", bounds, threshold)
	} else {
		ads, nextCursor, err = performSearch(userPrompt, userID, "", threshold)
	}

	if err != nil {
		return err
	}
	log.Printf("[HandleSearch] ads returned: %d", len(ads))
	log.Printf("[HandleSearch] Final ad order: %v", func() []int {
		result := make([]int, len(ads))
		for i, ad := range ads {
			result[i] = ad.ID
		}
		return result
	}())

	if userPrompt != "" {
		_ = search.SaveUserSearch(sql.NullInt64{Int64: int64(userID), Valid: userID != 0}, userPrompt)
		if userID != 0 {
			// Queue user for background embedding update
			vector.GetEmbeddingQueue().QueueUserForUpdate(userID)
		}
	}

	loc := getLocation(c)
	newAdButton := renderNewAdButton(c)

	// Show no results message if no ads found, but only for list/grid views
	if (view == "list" || view == "grid") && len(ads) == 0 {
		render(c, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), nil, nil, userID, loc, view, userPrompt, "", fmt.Sprintf("%.1f", threshold)))
		return nil
	}

	// Create loader URL if there are more results
	var loaderURL string
	if nextCursor != "" {
		loaderURL = fmt.Sprintf("/search-page?q=%s&cursor=%s&view=%s&threshold=%.1f", htmlEscape(userPrompt), htmlEscape(nextCursor), htmlEscape(view), threshold)
		// Add bounding box to loader URL for map view
		if view == "map" && bounds != nil {
			loaderURL += fmt.Sprintf("&minLat=%.6f&maxLat=%.6f&minLon=%.6f&maxLon=%.6f",
				bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon)
		}
	}

	render(c, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), ads, nil, userID, loc, view, userPrompt, loaderURL, fmt.Sprintf("%.1f", threshold)))

	return nil
}

func HandleSearchPage(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")
	userPrompt := c.Query("q")
	cursorStr := c.Query("cursor")
	view := c.Query("view")
	if view == "" {
		view = "list"
	}

	// Get threshold from query parameter, default to config value
	thresholdStr := c.Query("threshold")
	threshold := config.QdrantSearchThreshold
	if thresholdStr != "" {
		if thresholdVal, err := strconv.ParseFloat(thresholdStr, 32); err == nil {
			threshold = thresholdVal
		}
	}

	userID := getUserID(c)
	log.Printf("[HandleSearchPage] userPrompt='%s', cursorStr='%s', userID=%d, view='%s', threshold=%.2f", userPrompt, cursorStr, userID, view, threshold)

	ads, nextCursor, err := performSearch(userPrompt, userID, cursorStr, threshold)
	if err != nil {
		log.Printf("[HandleSearchPage] performSearch error: %v", err)
		return err
	}
	log.Printf("[HandleSearchPage] Found %d ads, nextCursor='%s'", len(ads), nextCursor)

	loc := getLocation(c)

	// Determine if there are more results for infinite scroll
	var nextPageURL string
	if nextCursor != "" {
		nextPageURL = fmt.Sprintf("/search-page?q=%s&cursor=%s&view=%s", htmlEscape(userPrompt), htmlEscape(nextCursor), htmlEscape(view))
	}

	// Show no results message if no ads found, but only for list/grid views
	if (view == "list" || view == "grid") && len(ads) == 0 {
		return nil
	}

	// For list view, render the ads in compact list format with separators
	if view == "list" {
		log.Printf("[HandleSearchPage] Rendering %d ads in list view", len(ads))
		for _, ad := range ads {
			render(c, ui.AdCardCompactList(ad, loc, ad.Bookmarked, userID))
			// Add separator after each ad
			render(c, Div(Class("border-b border-gray-200")))
		}
	} else if view == "grid" {
		log.Printf("[HandleSearchPage] Rendering %d ads in grid view", len(ads))
		// For grid view, render the ads in expandable format without separators
		for _, ad := range ads {
			render(c, ui.AdCardExpandable(ad, loc, ad.Bookmarked, userID, "grid"))
		}
	}

	// Add infinite scroll trigger if there are more results
	if nextPageURL != "" {
		log.Printf("[HandleSearchPage] Adding infinite scroll trigger with URL: %s", nextPageURL)

		// Create trigger that matches the view style
		if view == "grid" {
			// Grid trigger should be a grid item
			render(c, Div(
				Class("h-4"),
				g.Attr("hx-get", nextPageURL),
				g.Attr("hx-trigger", "revealed"),
				g.Attr("hx-swap", "outerHTML"),
			))
		} else {
			// List trigger
			render(c, Div(
				Class("h-4"),
				g.Attr("hx-get", nextPageURL),
				g.Attr("hx-trigger", "revealed"),
				g.Attr("hx-swap", "outerHTML"),
			))
		}
	} else {
		log.Printf("[HandleSearchPage] No infinite scroll trigger - no more results")
	}

	return nil
}

func EncodeCursor(c SearchCursor) string {
	jsonCursor, _ := json.Marshal(c)
	return base64.StdEncoding.EncodeToString(jsonCursor)
}

func DecodeCursor(s string) (SearchCursor, error) {
	var c SearchCursor
	if s == "" {
		return c, nil
	}
	jsonCursor, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(jsonCursor, &c)
	return c, err
}

func FilterAds(query SearchQuery, ads []ad.Ad) []ad.Ad {
	if query.IsEmpty() {
		return ads
	}
	var filteredAds []ad.Ad
	for _, ad := range ads {
		var makeMatch, yearMatch, modelMatch, engineMatch bool

		if query.Make == "" || ad.Make == query.Make {
			makeMatch = true
		}

		if len(query.Years) == 0 || anyStringInSlice(ad.Years, query.Years) {
			yearMatch = true
		}

		if len(query.Models) == 0 || anyStringInSlice(ad.Models, query.Models) {
			modelMatch = true
		}

		if len(query.EngineSizes) == 0 || anyStringInSlice(ad.Engines, query.EngineSizes) {
			engineMatch = true
		}

		if makeMatch && yearMatch && modelMatch && engineMatch {
			filteredAds = append(filteredAds, ad)
		}
	}
	return filteredAds
}

func GetNextPage(query SearchQuery, cursor *SearchCursor, limit int, userID int) ([]ad.Ad, *SearchCursor, error) {
	// Get filtered page from database
	ads, hasMore, err := ad.GetFilteredAdsPageDB(query, cursor, limit, userID)
	if err != nil {
		return nil, nil, err
	}

	var nextCursor *SearchCursor
	if hasMore && len(ads) > 0 {
		lastAd := ads[len(ads)-1]
		nextCursor = &SearchCursor{
			Query:      query,
			LastID:     lastAd.ID,
			LastPosted: lastAd.CreatedAt,
		}
	}

	return ads, nextCursor, nil
}

func HandleTreeCollapse(c *fiber.Ctx) error {
	q := c.Query("q")
	structuredQueryStr := c.Query("structured_query")
	path := c.Params("*")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Get threshold from query parameter, default to config value
	thresholdStr := c.Query("threshold")
	threshold := config.QdrantSearchThreshold
	if thresholdStr != "" {
		if thresholdVal, err := strconv.ParseFloat(thresholdStr, 32); err == nil {
			threshold = thresholdVal
		}
	}

	name := parts[len(parts)-1]
	level := len(parts) - 1

	return render(c, ui.CollapsedTreeNodeWithThreshold(name, "/"+path, q, structuredQueryStr, fmt.Sprintf("%.1f", threshold), level))
}

func TreeView(c *fiber.Ctx) error {
	q := c.Query("q")
	structuredQueryStr := c.Query("structured_query")
	path := c.Params("*")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		parts = []string{}
	}
	level := len(parts)

	// Get threshold from query parameter or form data, default to config value
	thresholdStr := c.Query("threshold")
	if thresholdStr == "" {
		thresholdStr = c.FormValue("threshold")
	}
	threshold := config.QdrantSearchThreshold
	if thresholdStr != "" {
		if thresholdVal, err := strconv.ParseFloat(thresholdStr, 32); err == nil {
			threshold = thresholdVal
		}
	}

	var structuredQuery SearchQuery
	if structuredQueryStr != "" {
		err := json.Unmarshal([]byte(structuredQueryStr), &structuredQuery)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid structured_query")
		}
	}

	var childNodes []g.Node
	var err error

	// Get ads for the current node
	currentUser, _ := CurrentUser(c)
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}

	var ads []ad.Ad
	if q != "" {
		// If we have a tree path, use vector search with metadata filtering
		if len(parts) > 0 && parts[0] != "" {
			// Build tree path from parts
			treePath := make(map[string]string)
			if len(parts) >= 1 && parts[0] != "" {
				treePath["make"] = parts[0]
			}
			if len(parts) >= 2 && parts[1] != "" {
				treePath["year"] = parts[1]
			}
			if len(parts) >= 3 && parts[2] != "" {
				treePath["model"] = parts[2]
			}
			if len(parts) >= 4 && parts[3] != "" {
				treePath["engine"] = parts[3]
			}

			log.Printf("[tree-view] Using vector search with metadata filtering for query: %s, tree path: %+v, threshold: %.2f", q, treePath, threshold)
			ads, err = getTreeAdsForSearchWithFilter(q, treePath, userID, threshold)
		} else {
			// Use vector search with threshold-based filtering for search queries without tree path
			log.Printf("[tree-view] Using vector search for query: %s, threshold: %.2f", q, threshold)
			ads, err = getTreeAdsForSearch(q, userID, threshold)
		}
	} else {
		// Use SQL-based filtering for browse mode
		log.Printf("[tree-view] Using SQL-based filtering for browse mode")
		// For tree view, we need ads to extract children from, not to display
		// So we get all ads that match the current path/structure
		ads, err = part.GetAdsForTreeView(parts, structuredQuery, userID)
	}

	if err != nil {
		return err
	}

	// Sort ads by CreatedAt DESC, ID DESC (same as list/grid view)
	sort.Slice(ads, func(i, j int) bool {
		if ads[i].CreatedAt.Equal(ads[j].CreatedAt) {
			return ads[i].ID > ads[j].ID
		}
		return ads[i].CreatedAt.After(ads[j].CreatedAt)
	})

	// At the root, only show a blank tree if there are no children (makes with ads)
	if level == 0 {
		var children []string
		if structuredQuery.Make != "" {
			adsForMake, err := part.GetAdsForNodeStructured([]string{structuredQuery.Make}, structuredQuery, userID)
			if err != nil {
				return err
			}
			if len(adsForMake) > 0 {
				children = []string{structuredQuery.Make}
			}
		} else {
			// For vector search (q != "") or SQL-based browsing (q == ""), get all makes
			children, err = part.GetMakes("")
			if err != nil {
				return err
			}
		}
		if len(children) == 0 {
			return render(c, ui.EmptyResponse())
		}
	}

	log.Printf("[tree-view] Processing %d ads for node at level %d", len(ads), level)

	// Extract unique values from ads to determine available children
	children := extractChildrenFromAds(ads, level, parts, structuredQuery)

	// Show ads if we're at a leaf level (level 4 = engine, level 5 = category)
	// or if there are no more children to show
	if level >= 4 || len(children) == 0 {
		if len(ads) > 0 {
			loc, _ := time.LoadLocation(c.Get("X-Timezone"))
			for _, ad := range ads {
				childNodes = append(childNodes, ui.AdCardCompactTree(ad, loc, ad.Bookmarked, userID))
			}
		} else {
			// Show "no results" message when no ads found (for all cases)
			childNodes = append(childNodes, ui.NoSearchResultsMessage())
		}
	}

	// Only show children that actually have ads
	// We need to check each child to see if it has ads
	var validChildren []string
	for _, child := range children {
		// Create a new path with this child
		childPath := append(parts, child)

		// Check if this child has any ads
		var childAds []ad.Ad
		var err error
		if q != "" {
			// For vector search, filter the existing ads instead of making new vector search calls
			childAds = filterAdsForChildPath(ads, childPath, level)
			err = nil // No error since we're just filtering existing ads
		} else {
			// For SQL search, check if this child has ads
			childAds, err = part.GetAdsForTreeView(childPath, structuredQuery, userID)
		}

		if err == nil && len(childAds) > 0 {
			validChildren = append(validChildren, child)
		}
	}

	// Show only children that have ads
	if len(validChildren) > 0 {
		for _, child := range validChildren {
			childNodes = append(childNodes, ui.CollapsedTreeNodeWithThreshold(child, "/"+path+"/"+child, q, structuredQueryStr, fmt.Sprintf("%.1f", threshold), level+1))
		}
	}

	if level == 0 {
		return render(c, g.Group(childNodes))
	}

	name := parts[len(parts)-1]
	return render(c, ui.ExpandedTreeNodeWithThreshold(name, "/"+path, q, structuredQueryStr, fmt.Sprintf("%.1f", threshold), level, g.Group(childNodes)))
}

func HandleListView(c *fiber.Ctx) error {
	return handleViewSwitch(c, "list")
}

func HandleTreeViewContent(c *fiber.Ctx) error {
	return handleViewSwitch(c, "tree")
}

// handleViewSwitch is a unified handler for switching between list and tree views
func handleViewSwitch(c *fiber.Ctx, view string) error {
	currentUser, _ := CurrentUser(c)
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
	userPrompt := c.FormValue("q")
	if userPrompt == "" {
		userPrompt = c.Query("q")
	}

	// Get threshold from query parameter or form data, default to config value
	thresholdStr := c.Query("threshold")
	if thresholdStr == "" {
		thresholdStr = c.FormValue("threshold")
	}
	threshold := config.QdrantSearchThreshold
	if thresholdStr != "" {
		if thresholdVal, err := strconv.ParseFloat(thresholdStr, 32); err == nil {
			threshold = thresholdVal
		}
	}

	var ads []ad.Ad
	var nextCursor string
	var err error

	// Use the same performSearch function that HandleSearch uses
	ads, nextCursor, err = performSearch(userPrompt, userID, "", threshold)
	if err != nil {
		log.Printf("[handleViewSwitch] performSearch error: %v", err)
		ads = []ad.Ad{}
	}

	loc, _ := time.LoadLocation(c.Get("X-Timezone"))

	var newAdButton g.Node
	if currentUser != nil {
		newAdButton = ui.StyledLink("New Ad", "/new-ad", ui.ButtonPrimary)
	} else {
		newAdButton = ui.StyledLinkDisabled("New Ad", ui.ButtonPrimary)
	}

	selectedView := c.FormValue("selected_view")
	if selectedView == "" {
		selectedView = view
	}
	c.Cookie(&fiber.Cookie{
		Name:     "last_view",
		Value:    selectedView,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: false,
		Path:     "/",
	})

	// Create loader URL if there are more results
	var loaderURL string
	if nextCursor != "" {
		loaderURL = fmt.Sprintf("/search-page?q=%s&cursor=%s&view=%s&threshold=%.1f", htmlEscape(userPrompt), htmlEscape(nextCursor), htmlEscape(selectedView), threshold)
	}

	// Only show no-results message for list and grid views
	if (view == "list" || view == "grid") && len(ads) == 0 {
		render(c, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), nil, nil, userID, loc, selectedView, userPrompt, "", fmt.Sprintf("%.1f", threshold)))
		return nil
	}

	return render(c, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), ads, nil, userID, loc, selectedView, userPrompt, loaderURL, fmt.Sprintf("%.1f", threshold)))
}

func HandleGridView(c *fiber.Ctx) error {
	return handleViewSwitch(c, "grid")
}

func HandleMapView(c *fiber.Ctx) error {
	// Extract bounding box parameters
	minLatStr := c.Query("minLat")
	maxLatStr := c.Query("maxLat")
	minLonStr := c.Query("minLon")
	maxLonStr := c.Query("maxLon")

	// Parse bounding box if provided
	var bounds *GeoBounds
	if minLatStr != "" && maxLatStr != "" && minLonStr != "" && maxLonStr != "" {
		minLat, err1 := strconv.ParseFloat(minLatStr, 64)
		maxLat, err2 := strconv.ParseFloat(maxLatStr, 64)
		minLon, err3 := strconv.ParseFloat(minLonStr, 64)
		maxLon, err4 := strconv.ParseFloat(maxLonStr, 64)

		if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
			bounds = &GeoBounds{
				MinLat: minLat,
				MaxLat: maxLat,
				MinLon: minLon,
				MaxLon: maxLon,
			}
			log.Printf("[HandleMapView] Using bounding box: %+v", bounds)
		} else {
			log.Printf("[HandleMapView] Error parsing bounding box: %v, %v, %v, %v", err1, err2, err3, err4)
		}
	}

	return handleViewSwitchWithGeo(c, "map", bounds)
}

// handleViewSwitchWithGeo is a unified handler for switching between views with geo filtering
func handleViewSwitchWithGeo(c *fiber.Ctx, view string, bounds *GeoBounds) error {
	currentUser, _ := CurrentUser(c)
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
	userPrompt := c.FormValue("q")
	if userPrompt == "" {
		userPrompt = c.Query("q")
	}

	// Get threshold from query parameter or form data, default to config value
	thresholdStr := c.Query("threshold")
	if thresholdStr == "" {
		thresholdStr = c.FormValue("threshold")
	}
	threshold := config.QdrantSearchThreshold
	if thresholdStr != "" {
		if thresholdVal, err := strconv.ParseFloat(thresholdStr, 32); err == nil {
			threshold = thresholdVal
		}
	}

	var ads []ad.Ad
	var nextCursor string
	var err error

	// Use geo search for map view, regular search for other views
	if view == "map" && bounds != nil {
		ads, nextCursor, err = performGeoBoxSearch(userPrompt, userID, "", bounds, threshold)
	} else {
		ads, nextCursor, err = performSearch(userPrompt, userID, "", threshold)
	}

	if err != nil {
		log.Printf("[handleViewSwitchWithGeo] search error: %v", err)
		ads = []ad.Ad{}
	}

	loc, _ := time.LoadLocation(c.Get("X-Timezone"))

	var newAdButton g.Node
	if currentUser != nil {
		newAdButton = ui.StyledLink("New Ad", "/new-ad", ui.ButtonPrimary)
	} else {
		newAdButton = ui.StyledLinkDisabled("New Ad", ui.ButtonPrimary)
	}

	selectedView := c.FormValue("selected_view")
	if selectedView == "" {
		selectedView = view
	}
	c.Cookie(&fiber.Cookie{
		Name:     "last_view",
		Value:    selectedView,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: false,
		Path:     "/",
	})

	// Create loader URL if there are more results
	var loaderURL string
	if nextCursor != "" {
		loaderURL = fmt.Sprintf("/search-page?q=%s&cursor=%s&view=%s&threshold=%.1f", htmlEscape(userPrompt), htmlEscape(nextCursor), htmlEscape(selectedView), threshold)
		// Add bounding box to loader URL for map view
		if view == "map" && bounds != nil {
			loaderURL += fmt.Sprintf("&minLat=%.6f&maxLat=%.6f&minLon=%.6f&maxLon=%.6f",
				bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon)
		}
	}

	// Only show no-results message for list and grid views
	if (view == "list" || view == "grid") && len(ads) == 0 {
		render(c, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), nil, nil, userID, loc, selectedView, userPrompt, "", fmt.Sprintf("%.1f", threshold)))
		return nil
	}

	return render(c, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), ads, nil, userID, loc, selectedView, userPrompt, loaderURL, fmt.Sprintf("%.1f", threshold)))
}

// getTreeAdsForSearch performs vector search with threshold-based filtering for tree view
// Uses larger limit to build complete tree structure
func getTreeAdsForSearch(userPrompt string, userID int, threshold float64) ([]ad.Ad, error) {
	// Generate embedding for search query
	log.Printf("[tree-search] Generating embedding for tree search query: %s", userPrompt)
	embedding, err := vector.EmbedTextCached(userPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Get results with threshold filtering at Qdrant level (larger limit for tree building)
	results, _, err := vector.QuerySimilarAds(embedding, config.QdrantSearchInitialK, "", threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to query Qdrant: %w", err)
	}
	log.Printf("[tree-search] Qdrant returned %d results (threshold: %.2f)", len(results), threshold)

	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	log.Printf("[tree-search] Qdrant result IDs: %v", ids)
	ads, _ := fetchAdsByIDs(ids, userID)
	log.Printf("[tree-search] DB fetch returned %d ads", len(ads))
	return ads, nil
}

// getTreeAdsForSearchWithFilter gets ads for tree view with filtering
func getTreeAdsForSearchWithFilter(userPrompt string, treePath map[string]string, userID int, threshold float64) ([]ad.Ad, error) {
	log.Printf("[getTreeAdsForSearchWithFilter] userPrompt='%s', treePath=%+v, userID=%d, threshold=%.2f", userPrompt, treePath, userID, threshold)

	var embedding []float32
	var err error

	if userPrompt != "" {
		embedding, err = vector.EmbedTextCached(userPrompt)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding: %w", err)
		}
	} else {
		embedding, err = vector.GetSiteLevelVector()
		if err != nil {
			return nil, fmt.Errorf("failed to get site-level vector: %w", err)
		}
	}

	// Build tree filter
	filter := vector.BuildTreeFilter(treePath)
	if filter == nil {
		log.Printf("[getTreeAdsForSearchWithFilter] No tree filter built, returning empty results")
		return []ad.Ad{}, nil
	}

	// Get results with filtering at Qdrant level (larger limit for tree building)
	results, _, err := vector.QuerySimilarAdsWithFilter(embedding, filter, config.QdrantSearchInitialK, "", threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to query Qdrant with filter: %w", err)
	}
	log.Printf("[getTreeAdsForSearchWithFilter] Qdrant returned %d results (threshold: %.2f)", len(results), threshold)

	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	log.Printf("[getTreeAdsForSearchWithFilter] Qdrant result IDs: %v", ids)
	ads, _ := fetchAdsByIDs(ids, userID)
	log.Printf("[getTreeAdsForSearchWithFilter] DB fetch returned %d ads", len(ads))
	return ads, nil
}

// filterAdsForChildPath filters existing ads to find those that match a child path
func filterAdsForChildPath(ads []ad.Ad, childPath []string, level int) []ad.Ad {
	var filteredAds []ad.Ad

	for _, ad := range ads {
		// Check if this ad matches the child path
		if matchesChildPath(ad, childPath, level) {
			filteredAds = append(filteredAds, ad)
		}
	}

	return filteredAds
}

// matchesChildPath checks if an ad matches a specific child path
func matchesChildPath(ad ad.Ad, childPath []string, level int) bool {
	switch level {
	case 0: // Root level - check make
		if len(childPath) >= 1 && childPath[0] != "" {
			// URL decode the make value from the path
			decodedMake, err := url.QueryUnescape(childPath[0])
			if err != nil {
				decodedMake = childPath[0] // fallback to original if decoding fails
			}
			return ad.Make == decodedMake
		}
	case 1: // Make level - check year
		if len(childPath) >= 2 && childPath[1] != "" {
			for _, year := range ad.Years {
				if year == childPath[1] {
					return true
				}
			}
			return false
		}
	case 2: // Year level - check model
		if len(childPath) >= 3 && childPath[2] != "" {
			// URL decode the model value from the path
			decodedModel, err := url.QueryUnescape(childPath[2])
			if err != nil {
				decodedModel = childPath[2] // fallback to original if decoding fails
			}
			for _, model := range ad.Models {
				if model == decodedModel {
					return true
				}
			}
			return false
		}
	case 3: // Model level - check engine
		if len(childPath) >= 4 && childPath[3] != "" {
			// URL decode the engine value from the path
			decodedEngine, err := url.QueryUnescape(childPath[3])
			if err != nil {
				decodedEngine = childPath[3] // fallback to original if decoding fails
			}
			for _, engine := range ad.Engines {
				if engine == decodedEngine {
					return true
				}
			}
			return false
		}
	case 4: // Engine level - check category
		if len(childPath) >= 5 && childPath[4] != "" {
			// URL decode the category value from the path
			decodedCategory, err := url.QueryUnescape(childPath[4])
			if err != nil {
				decodedCategory = childPath[4] // fallback to original if decoding fails
			}
			return ad.Category == decodedCategory
		}
	case 5: // Category level - check subcategory
		if len(childPath) >= 6 && childPath[5] != "" {
			// URL decode the subcategory value from the path
			decodedSubCategory, err := url.QueryUnescape(childPath[5])
			if err != nil {
				decodedSubCategory = childPath[5] // fallback to original if decoding fails
			}
			return ad.SubCategory == decodedSubCategory
		}
	}

	return true // Default to true if no specific filtering needed
}

// extractChildrenFromAds extracts unique values from ads to determine tree children
func extractChildrenFromAds(ads []ad.Ad, level int, parts []string, structuredQuery SearchQuery) []string {
	// Use a map to track unique values
	uniqueValues := make(map[string]bool)

	switch level {
	case 0: // Root level - extract makes
		for _, ad := range ads {
			if ad.Make != "" {
				uniqueValues[ad.Make] = true
			}
		}
	case 1: // Make level - extract years
		for _, ad := range ads {
			for _, year := range ad.Years {
				if year != "" {
					uniqueValues[year] = true
				}
			}
		}
	case 2: // Year level - extract models
		for _, ad := range ads {
			for _, model := range ad.Models {
				if model != "" {
					uniqueValues[model] = true
				}
			}
		}
	case 3: // Model level - extract engines
		for _, ad := range ads {
			for _, engine := range ad.Engines {
				if engine != "" {
					uniqueValues[engine] = true
				}
			}
		}
	case 4: // Engine level - extract categories
		for _, ad := range ads {
			if ad.Category != "" {
				uniqueValues[ad.Category] = true
			}
		}
	case 5: // Category level - extract subcategories
		for _, ad := range ads {
			if ad.SubCategory != "" {
				uniqueValues[ad.SubCategory] = true
			}
		}
	}

	// Convert map keys to slice and sort
	var children []string
	for value := range uniqueValues {
		children = append(children, value)
	}
	sort.Strings(children)

	log.Printf("[extractChildrenFromAds] Level %d: extracted %d children from %d ads", level, len(children), len(ads))
	return children
}
