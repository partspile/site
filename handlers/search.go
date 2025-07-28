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
	"github.com/parts-pile/site/grok"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/search"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	"github.com/parts-pile/site/vehicle"
	g "maragu.dev/gomponents"
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

// Helper to fetch ads by Pinecone result IDs
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
func runEmbeddingSearch(embedding []float32, cursor string, userID int) ([]ad.Ad, string, error) {
	results, nextCursor, err := vector.QuerySimilarAds(embedding, 10, cursor)
	if err != nil {
		return nil, "", err
	}
	log.Printf("[runEmbeddingSearch] Pinecone returned %d results", len(results))
	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	log.Printf("[runEmbeddingSearch] Pinecone result IDs: %v", ids)
	ads, _ := fetchAdsByIDs(ids, userID)
	log.Printf("[runEmbeddingSearch] DB fetch returned %d ads", len(ads))
	return ads, nextCursor, nil
}

// Try embedding-based search with user prompt
func tryQueryEmbedding(userPrompt, cursor string, userID int) ([]ad.Ad, string, error) {
	log.Printf("[search] Generating embedding for user query: %s", userPrompt)
	embedding, err := vector.EmbedText(userPrompt)
	if err != nil {
		return nil, "", err
	}
	return runEmbeddingSearch(embedding, cursor, userID)
}

// Try embedding-based search with user embedding
func tryUserEmbedding(userID int, cursor string) ([]ad.Ad, string, error) {
	embedding, err := vector.GetUserPersonalizedEmbedding(userID, false)
	if err != nil || embedding == nil {
		return nil, "", err
	}
	return runEmbeddingSearch(embedding, cursor, userID)
}

// Search strategy for both HandleSearch and HandleSearchPage
func performSearch(userPrompt string, userID int, cursor *SearchCursor, cursorStr string) ([]ad.Ad, string, error) {
	log.Printf("[performSearch] userPrompt='%s', userID=%d, cursorStr='%s'", userPrompt, userID, cursorStr)
	if userPrompt != "" {
		ads, nextCursor, _ := tryQueryEmbedding(userPrompt, cursorStr, userID)
		log.Printf("[performSearch] tryQueryEmbedding: found %d ads", len(ads))
		if len(ads) > 0 {
			return ads, nextCursor, nil
		}
	}
	if userPrompt == "" && userID != 0 {
		ads, nextCursor, _ := tryUserEmbedding(userID, cursorStr)
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
			ads, nextCursor, _ := runEmbeddingSearch(emb, cursorStr, userID)
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
			ads, nextCursor, _ := runEmbeddingSearch(emb, cursorStr, userID)
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

// Render loader if there are more results
func renderLoaderIfNeeded(c *fiber.Ctx, userPrompt, nextCursor string) {
	if nextCursor != "" {
		loaderURL := fmt.Sprintf("/search-page?q=%s&cursor=%s", htmlEscape(userPrompt), htmlEscape(nextCursor))
		render(c, ui.LoaderDiv(loaderURL))
	}
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

	userID := getUserID(c)
	log.Printf("[HandleSearch] userPrompt='%s', userID=%d", userPrompt, userID)
	ads, nextCursor, err := performSearch(userPrompt, userID, nil, "")
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

	render(c, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), ads, nil, userID, loc, view, userPrompt))
	renderLoaderIfNeeded(c, userPrompt, nextCursor)
	return nil
}

func HandleSearchPage(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")
	userPrompt := c.Query("q")
	cursorStr := c.Query("cursor")

	userID := getUserID(c)
	var cursor *SearchCursor
	if cursorStr != "" {
		curs, err2 := DecodeCursor(cursorStr)
		if err2 == nil {
			cursor = &curs
		}
	}

	ads, nextCursor, err := performSearch(userPrompt, userID, cursor, cursorStr)
	if err != nil {
		return err
	}

	loc := getLocation(c)
	for _, ad := range ads {
		render(c, ui.AdCardExpandable(ad, loc, ad.Bookmarked, userID))
	}

	renderLoaderIfNeeded(c, userPrompt, nextCursor)
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
		log.Printf("[tree-view] Using vector search for query: %s", q)
		ads, err = getTreeAdsForSearch(q, userID)
		if err != nil {
			log.Printf("[tree-view] Vector search failed, falling back to SQL: %v", err)
			// Fallback to SQL-based filtering
			ads, err = part.GetAdsForNodeStructured(parts, structuredQuery, userID)
		}
	} else {
		// Use SQL-based filtering for browse mode
		log.Printf("[tree-view] Using SQL-based filtering for browse mode")
		ads, err = part.GetAdsForNodeStructured(parts, structuredQuery, userID)
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

	loc, _ := time.LoadLocation(c.Get("X-Timezone"))

	log.Printf("[tree-view] Rendering %d ads for node at level %d", len(ads), level)
	for _, ad := range ads {
		childNodes = append(childNodes, ui.AdCardCompactTree(ad, loc, ad.Bookmarked, userID))
	}

	// Get children for the next level, filtered by structured query
	var children []string
	switch level {
	case 0: // Root, get makes
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
		}
	case 1: // Make, get years
		if len(structuredQuery.Years) > 0 {
			children = structuredQuery.Years
		} else {
			children, err = part.GetYearsForMake(parts[0], "")
		}
	case 2: // Year, get models
		if len(structuredQuery.Models) > 0 {
			children = structuredQuery.Models
		} else {
			children, err = part.GetModelsForMakeYear(parts[0], parts[1], "")
		}
	case 3: // Model, get engines
		if len(structuredQuery.EngineSizes) > 0 {
			children = structuredQuery.EngineSizes
		} else {
			children, err = part.GetEnginesForMakeYearModel(parts[0], parts[1], parts[2], "")
		}
	case 4: // Engine, get categories
		if structuredQuery.Category != "" {
			children = []string{structuredQuery.Category}
		} else {
			children, err = part.GetCategoriesForMakeYearModelEngine(parts[0], parts[1], parts[2], parts[3], "")
		}
	case 5: // Category, get subcategories
		if structuredQuery.SubCategory != "" {
			children = []string{structuredQuery.SubCategory}
		} else {
			children, err = part.GetSubCategoriesForMakeYearModelEngineCategory(parts[0], parts[1], parts[2], parts[3], parts[4], "")
		}
	}
	if err != nil {
		return err
	}

	// Always show ads for the current node, then show children if they exist
	if len(children) > 0 {
		for _, child := range children {
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

	var ads []ad.Ad
	var err error

	// Use the same performSearch function that HandleSearch uses
	ads, _, err = performSearch(userPrompt, userID, nil, "")
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

	return render(c, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), ads, nil, userID, loc, selectedView, userPrompt))
}

func HandleGridView(c *fiber.Ctx) error {
	return handleViewSwitch(c, "grid")
}

func HandleMapView(c *fiber.Ctx) error {
	return handleViewSwitch(c, "map")
}

// getTreeAdsForSearch performs vector search with threshold-based filtering for tree view
func getTreeAdsForSearch(userPrompt string, userID int) ([]ad.Ad, error) {
	// Generate embedding for search query
	log.Printf("[tree-search] Generating embedding for tree search query: %s", userPrompt)
	embedding, err := vector.EmbedText(userPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Get more results than needed to filter by threshold
	const initialK = 200
	results, _, err := vector.QuerySimilarAds(embedding, initialK, "")
	if err != nil {
		return nil, fmt.Errorf("failed to query Pinecone: %w", err)
	}

	log.Printf("[tree-search] Pinecone returned %d results", len(results))

	// Filter by similarity threshold
	const (
		minResultsThreshold = float32(0.7) // High quality threshold
		fallbackThreshold   = float32(0.5) // Lower threshold if not enough results
		maxResults          = 100          // Maximum results to show
		minResults          = 10           // Minimum results before using fallback
	)

	var filteredResults []vector.AdResult
	threshold := minResultsThreshold

	// First pass: filter with high threshold
	for _, result := range results {
		if result.Score >= threshold {
			filteredResults = append(filteredResults, result)
		}
	}

	log.Printf("[tree-search] High threshold (%.1f) filtered to %d results", threshold, len(filteredResults))

	// Fallback: if not enough results, use lower threshold
	if len(filteredResults) < minResults && threshold == minResultsThreshold {
		threshold = fallbackThreshold
		filteredResults = nil
		for _, result := range results {
			if result.Score >= threshold {
				filteredResults = append(filteredResults, result)
			}
		}
		log.Printf("[tree-search] Fallback threshold (%.1f) filtered to %d results", threshold, len(filteredResults))
	}

	// Limit to max results
	if len(filteredResults) > maxResults {
		filteredResults = filteredResults[:maxResults]
		log.Printf("[tree-search] Limited to %d results", maxResults)
	}

	// Fetch ads from DB
	ids := make([]string, len(filteredResults))
	for i, r := range filteredResults {
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
