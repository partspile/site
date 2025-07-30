package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/grok"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/search"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	"github.com/parts-pile/site/vehicle"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

const sysPrompt = `You are an expert vehicle parts assistant.

Your job is to extract a structured query from a user's search request.

Extract the make, years, models, engine sizes, category, and subcategory from
the user's search request.  Use your best judgement as a vehicle parts export
to fill out the structured query as much as possible.  When filling out the
structured query, only use values from the lists below, and not the user's values.
For example, if user entered "Ford", the structure query would use "FORD".

<Makes>
%s
</Makes>

<Years>
%s
</Years>

<Models>
%s
</Models>

<EngineSizes>
%s
</EngineSizes>

<Categories>
%s
</Categories>

<SubCategories>
%s
</SubCategories>

Return JSON encoding this Go structure with the vehicle parts data:

struct {
	Make        string
	Years       []string
	Models      []string
	EngineSizes []string
	Category    string
	SubCategory string
}

Only return the JSON.  Nothing else.
`

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
	ads := make([]ad.Ad, 0, len(ids))
	for _, idStr := range ids {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}
		adObj, _, ok := ad.GetAdByID(id)
		if ok {
			if userID > 0 {
				bookmarked, _ := ad.IsAdBookmarkedByUser(userID, adObj.ID)
				adObj.Bookmarked = bookmarked
			}
			ads = append(ads, adObj)
		}
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

// Run embedding-based search and fetch ads
func runEmbeddingSearch(embedding []float32, cursor string, userID int, threshold float32) ([]ad.Ad, string, error) {
	// Get results with threshold filtering at Qdrant level
	results, nextCursor, err := vector.QuerySimilarAds(embedding, config.VectorSearchPageSize, cursor, threshold)
	if err != nil {
		return nil, "", err
	}
	log.Printf("[runEmbeddingSearch] Qdrant returned %d results (threshold: %.1f)", len(results), threshold)

	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	log.Printf("[runEmbeddingSearch] Qdrant result IDs: %v", ids)
	ads, _ := fetchAdsByIDs(ids, userID)
	log.Printf("[runEmbeddingSearch] DB fetch returned %d ads", len(ads))
	return ads, nextCursor, nil
}

// Try embedding-based search with user prompt
func tryQueryEmbedding(userPrompt, cursor string, userID int, threshold float32) ([]ad.Ad, string, error) {
	log.Printf("[search] Generating embedding for user query: %s", userPrompt)
	embedding, err := vector.EmbedText(userPrompt)
	if err != nil {
		return nil, "", err
	}
	return runEmbeddingSearch(embedding, cursor, userID, threshold)
}

// Try embedding-based search with user embedding
func tryUserEmbedding(userID int, cursor string, threshold float32) ([]ad.Ad, string, error) {
	embedding, err := vector.GetUserPersonalizedEmbedding(userID, false)
	if err != nil || embedding == nil {
		return nil, "", err
	}
	return runEmbeddingSearch(embedding, cursor, userID, threshold)
}

// Search strategy for both HandleSearch and HandleSearchPage
func performSearch(userPrompt string, userID int, cursorStr string, threshold float32) ([]ad.Ad, string, error) {
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

	// Get threshold from request, default to config value
	thresholdStr := c.Query("threshold")
	threshold := float32(config.VectorSearchThreshold)
	if thresholdStr != "" {
		if thresholdVal, err := strconv.ParseFloat(thresholdStr, 32); err == nil {
			threshold = float32(thresholdVal)
		}
	}

	userID := getUserID(c)
	log.Printf("[HandleSearch] userPrompt='%s', userID=%d, threshold=%.2f", userPrompt, userID, threshold)
	ads, nextCursor, err := performSearch(userPrompt, userID, "", threshold)
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
		render(c, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), nil, nil, userID, loc, view, userPrompt, "", fmt.Sprintf("%.2f", threshold)))
		return nil
	}

	// Create loader URL if there are more results
	var loaderURL string
	if nextCursor != "" {
		loaderURL = fmt.Sprintf("/search-page?q=%s&cursor=%s&view=%s&threshold=%.2f", htmlEscape(userPrompt), htmlEscape(nextCursor), htmlEscape(view), threshold)
	}

	render(c, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), ads, nil, userID, loc, view, userPrompt, loaderURL, fmt.Sprintf("%.2f", threshold)))

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

	// Get threshold from request, default to config value
	thresholdStr := c.Query("threshold")
	threshold := float32(config.VectorSearchThreshold)
	if thresholdStr != "" {
		if thresholdVal, err := strconv.ParseFloat(thresholdStr, 32); err == nil {
			threshold = float32(thresholdVal)
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
		nextPageURL = fmt.Sprintf("/search-page?q=%s&cursor=%s&view=%s&threshold=%.2f", htmlEscape(userPrompt), htmlEscape(nextCursor), htmlEscape(view), threshold)
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

func ParseSearchQuery(q string) (SearchQuery, error) {
	if q == "" {
		return SearchQuery{}, nil
	}

	allMakes := vehicle.GetMakes()
	allYears := vehicle.GetYearRange()
	allModels := vehicle.GetAllModels()
	allEngineSizes := vehicle.GetAllEngineSizes()

	categories, err := part.GetAllCategories()
	if err != nil {
		return SearchQuery{}, fmt.Errorf("error getting categories: %w", err)
	}
	allCategories := make([]string, len(categories))
	for i, c := range categories {
		allCategories[i] = c.Name
	}

	subCategories, err := part.GetAllSubCategories()
	if err != nil {
		return SearchQuery{}, fmt.Errorf("error getting subcategories: %w", err)
	}
	allSubCategories := make([]string, len(subCategories))
	for i, sc := range subCategories {
		allSubCategories[i] = sc.Name
	}

	prompt := fmt.Sprintf(sysPrompt,
		strings.Join(allMakes, "\n"),
		strings.Join(allYears, "\n"),
		strings.Join(allModels, "\n"),
		strings.Join(allEngineSizes, "\n"),
		strings.Join(allCategories, "\n"),
		strings.Join(allSubCategories, "\n"),
	)

	var query SearchQuery
	resp, err := grok.CallGrok(prompt, q)
	if err != nil {
		return SearchQuery{}, fmt.Errorf("error grokking query: %w", err)
	}

	err = json.Unmarshal([]byte(resp), &query)
	if err != nil {
		return SearchQuery{}, fmt.Errorf("error unmarshalling grok response: %w", err)
	}

	return query, nil
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

	name := parts[len(parts)-1]
	level := len(parts) - 1

	return render(c, ui.CollapsedTreeNode(name, "/"+path, q, structuredQueryStr, level))
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

	// Get threshold from request, default to config value
	thresholdStr := c.Query("threshold")
	threshold := float32(config.VectorSearchThreshold)
	if thresholdStr != "" {
		if thresholdVal, err := strconv.ParseFloat(thresholdStr, 32); err == nil {
			threshold = float32(thresholdVal)
		}
	}

	var structuredQuery SearchQuery
	if structuredQueryStr != "" {
		err := json.Unmarshal([]byte(structuredQueryStr), &structuredQuery)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid structured_query")
		}
	} else {
		structuredQuery, _ = ParseSearchQuery(q)
		// If we just parsed it, re-marshal for passing to children
		structuredQueryStrBytes, _ := json.Marshal(structuredQuery)
		structuredQueryStr = string(structuredQueryStrBytes)
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
		// Use vector search with threshold-based filtering for search queries
		log.Printf("[tree-view] Using vector search for query: %s with threshold: %.2f", q, threshold)
		ads, err = getTreeAdsForSearch(q, userID, threshold)
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
			children, err = part.GetMakes("")
			if err != nil {
				return err
			}
		}
		if len(children) == 0 {
			return c.SendString("")
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
			// For vector search, we need to check if this child path has ads
			// We'll use the same vector search but filter by the child path
			childAds, err = part.GetAdsForTreeView(childPath, structuredQuery, userID)
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
			childNodes = append(childNodes, ui.CollapsedTreeNode(child, "/"+path+"/"+child, q, structuredQueryStr, level+1))
		}
	}

	if level == 0 {
		return render(c, g.Group(childNodes))
	}

	name := parts[len(parts)-1]
	return render(c, ui.ExpandedTreeNode(name, "/"+path, q, structuredQueryStr, level, g.Group(childNodes)))
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

	// Get threshold from request, default to config value
	thresholdStr := c.Query("threshold")
	threshold := float32(config.VectorSearchThreshold)
	if thresholdStr != "" {
		if thresholdVal, err := strconv.ParseFloat(thresholdStr, 32); err == nil {
			threshold = float32(thresholdVal)
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
		loaderURL = fmt.Sprintf("/search-page?q=%s&cursor=%s&view=%s&threshold=%.2f", htmlEscape(userPrompt), htmlEscape(nextCursor), htmlEscape(selectedView), threshold)
	}

	// Only show no-results message for list and grid views
	if (view == "list" || view == "grid") && len(ads) == 0 {
		render(c, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), nil, nil, userID, loc, selectedView, userPrompt, "", fmt.Sprintf("%.2f", threshold)))
		return nil
	}

	return render(c, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), ads, nil, userID, loc, selectedView, userPrompt, loaderURL, fmt.Sprintf("%.2f", threshold)))
}

func HandleGridView(c *fiber.Ctx) error {
	return handleViewSwitch(c, "grid")
}

func HandleMapView(c *fiber.Ctx) error {
	return handleViewSwitch(c, "map")
}

// getTreeAdsForSearch performs vector search with threshold-based filtering for tree view
// Uses larger limit to build complete tree structure
func getTreeAdsForSearch(userPrompt string, userID int, threshold float32) ([]ad.Ad, error) {
	// Generate embedding for search query
	log.Printf("[tree-search] Generating embedding for tree search query: %s", userPrompt)
	embedding, err := vector.EmbedText(userPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Get results with threshold filtering at Qdrant level (larger limit for tree building)
	results, _, err := vector.QuerySimilarAds(embedding, config.VectorSearchInitialK, "", threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to query Qdrant: %w", err)
	}

	log.Printf("[tree-search] Qdrant returned %d results (threshold: %.1f)", len(results), threshold)

	// Fetch ads from DB
	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}

	log.Printf("[tree-search] Fetching %d ads from DB", len(ids))
	ads, err := fetchAdsByIDs(ids, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ads: %w", err)
	}

	log.Printf("[tree-search] Successfully fetched %d ads for tree view", len(ads))
	return ads, nil
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
