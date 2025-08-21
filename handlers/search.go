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
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/vector"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type SearchQuery = ad.SearchQuery

type SearchCursor = ad.SearchCursor

// runEmbeddingSearch runs vector search without filters
func runEmbeddingSearch(embedding []float32, cursor string, currentUser *user.User, threshold float64, k int) ([]ad.Ad, string, error) {
	// Get results with threshold filtering at Qdrant level
	ids, nextCursor, err := vector.QuerySimilarAdIDs(embedding, k, cursor, threshold)
	if err != nil {
		return nil, "", err
	}
	log.Printf("[runEmbeddingSearch] Qdrant returned %d results (threshold: %.2f, k: %d)", len(ids), threshold, k)
	log.Printf("[runEmbeddingSearch] Qdrant result IDs: %v", ids)

	ads, err := ad.GetAdsByIDs(ids, currentUser)
	if err != nil {
		return nil, "", err
	}
	log.Printf("[runEmbeddingSearch] DB fetch returned %d ads", len(ads))
	return ads, nextCursor, nil
}

// runEmbeddingSearchWithFilter runs vector search with filters
func runEmbeddingSearchWithFilter(embedding []float32, filter *qdrant.Filter, cursor string, currentUser *user.User, threshold float64, k int) ([]ad.Ad, string, error) {
	// Get results with threshold filtering at Qdrant level
	ids, nextCursor, err := vector.QuerySimilarAdIDsWithFilter(embedding, filter, k, cursor, threshold)
	if err != nil {
		return nil, "", err
	}
	log.Printf("[runEmbeddingSearchWithFilter] Qdrant returned %d results (threshold: %.2f, k: %d)", len(ids), threshold, k)
	log.Printf("[runEmbeddingSearchWithFilter] Qdrant result IDs: %v", ids)

	ads, err := ad.GetAdsByIDs(ids, currentUser)
	if err != nil {
		return nil, "", err
	}
	log.Printf("[runEmbeddingSearchWithFilter] DB fetch returned %d ads", len(ads))
	return ads, nextCursor, nil
}

// Embedding-based search with user query
func queryEmbedding(userPrompt string, currentUser *user.User, cursor string, threshold float64, k int) ([]ad.Ad, string, error) {
	log.Printf("[queryEmbedding] Generating embedding for user query: %s", userPrompt)
	embedding, err := vector.EmbedTextCached(userPrompt)
	if err != nil {
		return nil, "", err
	}
	return runEmbeddingSearch(embedding, cursor, currentUser, threshold, k)
}

// Embedding-based search with user embedding
func userEmbedding(currentUser *user.User, cursor string, threshold float64, k int) ([]ad.Ad, string, error) {
	log.Printf("[userEmbedding] called with userID=%d, cursor=%s, threshold=%.2f", currentUser.ID, cursor, threshold)
	embedding, err := vector.GetUserPersonalizedEmbedding(currentUser.ID, false)
	if err != nil {
		log.Printf("[userEmbedding] GetUserPersonalizedEmbedding error: %v", err)
		return nil, "", err
	}
	if embedding == nil {
		log.Printf("[userEmbedding] GetUserPersonalizedEmbedding returned nil embedding")
		return nil, "", nil
	}
	return runEmbeddingSearch(embedding, cursor, currentUser, threshold, k)
}

// Embedding-based search with site-level vector
func siteEmbedding(currentUser *user.User, cursor string, threshold float64, k int) ([]ad.Ad, string, error) {
	log.Printf("[siteEmbedding] called with userID=%d, cursor=%s, threshold=%.2f", currentUser.ID, cursor, threshold)
	embedding, err := vector.GetSiteLevelVector()
	if err != nil {
		log.Printf("[siteEmbedding] GetSiteLevelVector error: %v", err)
		return nil, "", err
	}
	if embedding == nil {
		log.Printf("[siteEmbedding] GetSiteLevelVector returned nil embedding")
		return nil, "", nil
	}
	return runEmbeddingSearch(embedding, cursor, currentUser, threshold, k)
}

// Search strategy for both HandleSearch and HandleSearchPage
func performSearch(userPrompt string, currentUser *user.User, cursorStr string, threshold float64, k int) ([]ad.Ad, string, error) {
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
	log.Printf("[performSearch] userPrompt='%s', userID=%d, cursorStr='%s', threshold=%.2f, k=%d", userPrompt, userID, cursorStr, threshold, k)

	if userPrompt != "" {
		return queryEmbedding(userPrompt, currentUser, cursorStr, threshold, k)
	}

	if userPrompt == "" && userID != 0 {
		return userEmbedding(currentUser, cursorStr, threshold, k)
	}

	if userPrompt == "" && userID == 0 {
		return siteEmbedding(currentUser, cursorStr, threshold, k)
	}

	// This should never be reached, but provide a default return
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
func performGeoBoxSearch(userPrompt string, currentUser *user.User, cursorStr string, bounds *GeoBounds, threshold float64) ([]ad.Ad, string, error) {
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
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
				ads, nextCursor, err = runEmbeddingSearchWithFilter(emb, geoFilter, cursorStr, currentUser, threshold, config.QdrantSearchInitialK)
			} else {
				ads, nextCursor, err = runEmbeddingSearch(emb, cursorStr, currentUser, threshold, config.QdrantSearchInitialK)
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
			ads, nextCursor, err = runEmbeddingSearchWithFilter(embedding, geoFilter, cursorStr, currentUser, threshold, config.QdrantSearchInitialK)
		} else {
			ads, nextCursor, err = runEmbeddingSearch(embedding, cursorStr, currentUser, threshold, config.QdrantSearchInitialK)
		}
	}

	if err != nil {
		return nil, "", err
	}

	log.Printf("[performGeoBoxSearch] Found %d ads in bounding box", len(ads))
	return ads, nextCursor, nil
}

// Get user ID from context
func getUserID(c *fiber.Ctx) int {
	currentUser, err := CurrentUser(c)
	if err != nil {
		log.Printf("[DEBUG] CurrentUser error: %v", err)
	}
	if currentUser != nil {
		log.Printf("[DEBUG] getUserID returning userID=%d", currentUser.ID)
		return currentUser.ID
	}
	log.Printf("[DEBUG] getUserID returning userID=0 (no current user)")
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

	currentUser, _ := CurrentUser(c)
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
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
		ads, nextCursor, err = performGeoBoxSearch(userPrompt, currentUser, "", bounds, threshold)
	} else if view == "map" {
		// For map view without bounds, use map-specific search functions
		ads, nextCursor, err = performSearch(userPrompt, currentUser, "", threshold, config.QdrantSearchInitialK)
	} else {
		ads, nextCursor, err = performSearch(userPrompt, currentUser, "", threshold, config.QdrantSearchPageSize)
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

	currentUser, _ := CurrentUser(c)
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
	log.Printf("[HandleSearchPage] userPrompt='%s', cursorStr='%s', userID=%d, view='%s', threshold=%.2f", userPrompt, cursorStr, userID, view, threshold)

	ads, nextCursor, err := performSearch(userPrompt, currentUser, cursorStr, threshold, config.QdrantSearchPageSize)
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
			render(c, ui.AdCardCompactList(ad, loc, userID))
			// Add separator after each ad
			render(c, Div(Class("border-b border-gray-200")))
		}
	} else if view == "grid" {
		log.Printf("[HandleSearchPage] Rendering %d ads in grid view", len(ads))
		// For grid view, render the ads in expandable format without separators
		for _, ad := range ads {
			render(c, ui.AdCardExpandable(ad, loc, userID, "grid"))
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
			ads, err = getTreeAdsForSearchWithFilter(q, treePath, currentUser, threshold)
		} else {
			// Use vector search with threshold-based filtering for search queries without tree path
			log.Printf("[tree-view] Using vector search for query: %s, threshold: %.2f", q, threshold)
			ads, err = getTreeAdsForSearch(q, currentUser, threshold)
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
				childNodes = append(childNodes, ui.AdCardCompactTree(ad, loc, userID))
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
	ads, nextCursor, err = performSearch(userPrompt, currentUser, "", threshold, config.QdrantSearchPageSize)
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
		ads, nextCursor, err = performGeoBoxSearch(userPrompt, currentUser, "", bounds, threshold)
	} else if view == "map" {
		// For map view without bounds, use map-specific search functions
		ads, nextCursor, err = performSearch(userPrompt, currentUser, "", threshold, config.QdrantSearchInitialK)
	} else {
		ads, nextCursor, err = performSearch(userPrompt, currentUser, "", threshold, config.QdrantSearchPageSize)
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
func getTreeAdsForSearch(userPrompt string, currentUser *user.User, threshold float64) ([]ad.Ad, error) {
	// Generate embedding for search query
	log.Printf("[tree-search] Generating embedding for tree search query: %s", userPrompt)
	embedding, err := vector.EmbedTextCached(userPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Get results with threshold filtering at Qdrant level (larger limit for tree building)
	intIDs, _, err := vector.QuerySimilarAdIDs(embedding, config.QdrantSearchInitialK, "", threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to query Qdrant: %w", err)
	}
	log.Printf("[tree-search] Qdrant returned %d results (threshold: %.2f)", len(intIDs), threshold)
	log.Printf("[tree-search] Qdrant result IDs: %v", intIDs)
	var ads []ad.Ad
	ads, _ = ad.GetAdsByIDs(intIDs, currentUser)
	log.Printf("[tree-search] DB fetch returned %d ads", len(ads))
	return ads, nil
}

// getTreeAdsForSearchWithFilter gets ads for tree view with filtering
func getTreeAdsForSearchWithFilter(userPrompt string, treePath map[string]string, currentUser *user.User, threshold float64) ([]ad.Ad, error) {
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
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
	intIDs, _, err := vector.QuerySimilarAdIDsWithFilter(embedding, filter, config.QdrantSearchInitialK, "", threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to query Qdrant with filter: %w", err)
	}
	log.Printf("[getTreeAdsForSearchWithFilter] Qdrant returned %d results (threshold: %.2f)", len(intIDs), threshold)
	log.Printf("[getTreeAdsForSearchWithFilter] Qdrant result IDs: %v", intIDs)
	var ads []ad.Ad
	ads, _ = ad.GetAdsByIDs(intIDs, currentUser)
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
		// TODO: Implement subcategory filtering using SubCategoryID
		// For now, skip subcategory level filtering
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
		// TODO: Implement subcategory extraction using SubCategoryID
		// For now, skip subcategory level extraction
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

// HandleSearchAPI returns search results as JSON for JavaScript consumption
func HandleSearchAPI(c *fiber.Ctx) error {
	userPrompt := strings.TrimSpace(c.Query("q", ""))
	view := c.Query("view", "list")
	threshold := c.QueryFloat("threshold", 0.6)

	// Get current user
	currentUser, _ := CurrentUser(c)
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}

	log.Printf("[HandleSearchAPI] userPrompt='%s', userID=%d, view=%s, threshold=%.2f", userPrompt, userID, view, threshold)

	// Parse bounding box parameters
	var bounds *GeoBounds
	if minLatStr := c.Query("minLat"); minLatStr != "" {
		if maxLatStr := c.Query("maxLat"); maxLatStr != "" {
			if minLonStr := c.Query("minLon"); minLonStr != "" {
				if maxLonStr := c.Query("maxLon"); maxLonStr != "" {
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
						log.Printf("[HandleSearchAPI] Using bounding box: %+v", bounds)
					}
				}
			}
		}
	}

	// Perform search
	var ads []ad.Ad
	var nextCursor string
	var err error

	// Use geo search for map view with bounds, regular search otherwise
	if view == "map" && bounds != nil {
		ads, nextCursor, err = performGeoBoxSearch(userPrompt, currentUser, "", bounds, threshold)
	} else if view == "map" {
		// For map view without bounds, use map-specific search functions
		ads, nextCursor, err = performSearch(userPrompt, currentUser, "", threshold, config.QdrantSearchInitialK)
	} else {
		ads, nextCursor, err = performSearch(userPrompt, currentUser, "", threshold, config.QdrantSearchPageSize)
	}

	if err != nil {
		log.Printf("[HandleSearchAPI] Search error: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Search failed",
			"ads":   []ad.Ad{},
		})
	}

	log.Printf("[HandleSearchAPI] ads returned: %d", len(ads))

	// Return JSON response
	return c.JSON(fiber.Map{
		"ads":        ads,
		"nextCursor": nextCursor,
		"count":      len(ads),
	})
}
