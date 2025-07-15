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

// Helper to fetch ads by Pinecone result IDs
func fetchAdsByIDs(ids []string) ([]ad.Ad, error) {
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
			ads = append(ads, adObj)
		}
	}
	return ads, nil
}

func HandleSearch(c *fiber.Ctx) error {
	userPrompt := c.Query("q")
	view := c.FormValue("view")
	if view == "" {
		view = "list"
	}

	var ads []ad.Ad
	var nextCursor string
	var usedVectorSearch bool

	currentUser, _ := CurrentUser(c)
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}

	if userPrompt != "" {
		// Embedding-based search for q != ""
		embedding, err := vector.EmbedText(userPrompt)
		if err == nil {
			results, cursor, err := vector.QuerySimilarAds(embedding, 10, "")
			if err == nil {
				log.Printf("[search] Embedding-based search (query embedding) used in HandleSearch with q='%s'", userPrompt)
				ids := make([]string, len(results))
				for i, r := range results {
					ids[i] = r.ID
				}
				ads, _ = fetchAdsByIDs(ids)
				nextCursor = cursor
				usedVectorSearch = true
			}
		}
		// Save the user's search query
		_ = search.SaveUserSearch(sql.NullInt64{Int64: int64(userID), Valid: userID != 0}, userPrompt)
		if userID != 0 && userPrompt != "" {
			go vector.GetUserPersonalizedEmbedding(userID, true)
		}
	}

	if !usedVectorSearch {
		if userPrompt == "" && userID != 0 {
			// Personalized feed for logged-in user
			embedding, err := vector.GetUserPersonalizedEmbedding(userID, false)
			if err == nil && embedding != nil {
				results, cursor, err := vector.QuerySimilarAds(embedding, 10, "")
				if err == nil {
					log.Printf("[search] Embedding-based search (user embedding) used in HandleSearch for userID=%d", userID)
					ids := make([]string, len(results))
					for i, r := range results {
						ids[i] = r.ID
					}
					ads, _ = fetchAdsByIDs(ids)
					nextCursor = cursor
					usedVectorSearch = true
				}
			}
			if err != nil {
				log.Printf("[embedding] User embedding error for userID=%d: %v", userID, err)
			}
		}
	}

	if !usedVectorSearch {
		log.Printf("[search] Fallback to SQL-based search in HandleSearch (q='%s', userID=%d)", userPrompt, userID)
		// Fallback to old SQL-based logic
		query, err := ParseSearchQuery(userPrompt)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Could not parse query")
		}
		ads, _, err = GetNextPage(query, nil, 10, userID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Could not get ads")
		}
	}

	loc, _ := time.LoadLocation(c.Get("X-Timezone"))

	var newAdButton g.Node
	if currentUser != nil {
		newAdButton = ui.StyledLink("New Ad", "/new-ad", ui.ButtonPrimary)
	} else {
		newAdButton = ui.StyledLinkDisabled("New Ad", ui.ButtonPrimary)
	}

	render(c, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), ads, nil, userID, loc, view, userPrompt))

	if nextCursor != "" {
		loaderURL := fmt.Sprintf("/search-page?q=%s&cursor=%s", htmlEscape(userPrompt), htmlEscape(nextCursor))
		loaderHTML := fmt.Sprintf(`<div id="loader" hx-get="%s" hx-trigger="revealed" hx-swap="outerHTML">Loading more...</div>`, loaderURL)
		fmt.Fprint(c.Response().BodyWriter(), loaderHTML)
	}
	return nil
}

func HandleSearchPage(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")
	userPrompt := c.Query("q")
	cursorStr := c.Query("cursor")

	currentUser, _ := CurrentUser(c)
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}

	var ads []ad.Ad
	var nextCursor string
	var usedVectorSearch bool

	if userPrompt != "" {
		embedding, err := vector.EmbedText(userPrompt)
		if err == nil {
			results, cursor, err := vector.QuerySimilarAds(embedding, 10, cursorStr)
			if err == nil {
				log.Printf("[search] Embedding-based search (query embedding) used in HandleSearchPage with q='%s'", userPrompt)
				ids := make([]string, len(results))
				for i, r := range results {
					ids[i] = r.ID
				}
				ads, _ = fetchAdsByIDs(ids)
				nextCursor = cursor
				usedVectorSearch = true
			}
		}
	}

	if !usedVectorSearch {
		if userPrompt == "" && userID != 0 {
			// Personalized feed for logged-in user
			embedding, err := vector.GetUserPersonalizedEmbedding(userID, false)
			if err == nil && embedding != nil {
				results, cursor, err := vector.QuerySimilarAds(embedding, 10, cursorStr)
				if err == nil {
					log.Printf("[search] Embedding-based search (user embedding) used in HandleSearchPage for userID=%d", userID)
					ids := make([]string, len(results))
					for i, r := range results {
						ids[i] = r.ID
					}
					ads, _ = fetchAdsByIDs(ids)
					nextCursor = cursor
					usedVectorSearch = true
				}
			}
			if err != nil {
				log.Printf("[embedding] User embedding error for userID=%d: %v", userID, err)
			}
		}
	}

	if !usedVectorSearch {
		log.Printf("[search] Fallback to SQL-based search in HandleSearchPage (q='%s', userID=%d)", userPrompt, userID)
		// Fallback to old SQL-based logic
		cursor, err := DecodeCursor(cursorStr)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid cursor")
		}
		ads, _, err = GetNextPage(cursor.Query, &cursor, 10, userID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Could not get ads")
		}
	}

	loc, _ := time.LoadLocation(c.Get("X-Timezone"))

	for _, ad := range ads {
		render(c, ui.AdCardExpandable(ad, loc, ad.Bookmarked, userID))
	}

	if nextCursor != "" {
		loaderURL := fmt.Sprintf("/search-page?q=%s&cursor=%s", htmlEscape(userPrompt), htmlEscape(nextCursor))
		loaderHTML := fmt.Sprintf(`<div id="loader" hx-get="%s" hx-trigger="revealed" hx-swap="outerHTML">Loading more...</div>`, loaderURL)
		fmt.Fprint(c.Response().BodyWriter(), loaderHTML)
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

	// Get ads for the current node (filtered by structured query)
	currentUser, _ := CurrentUser(c)
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
	ads, err := part.GetAdsForNodeStructured(parts, structuredQuery, userID)
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

	for _, ad := range ads {
		childNodes = append(childNodes, ui.AdCardExpandable(ad, loc, ad.Bookmarked, userID))
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

	// If there are children, render them; otherwise, render ads at the leaf
	if len(children) > 0 {
		for _, child := range children {
			childNodes = append(childNodes, ui.CollapsedTreeNode(child, "/"+path+"/"+child, q, structuredQueryStr, level+1))
		}
	} // else, childNodes already contains the ads

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
	userPrompt := c.Query("q")
	if userPrompt == "" {
		userPrompt = c.FormValue("q")
	}
	structuredQuery := c.FormValue("structured_query")

	var ads []ad.Ad
	var usedVectorSearch bool
	var err error

	if userPrompt != "" {
		embedding, err := vector.EmbedText(userPrompt)
		if err == nil {
			results, _, err := vector.QuerySimilarAds(embedding, 10, "")
			if err == nil {
				log.Printf("[search] Embedding-based search (query embedding) used for view '%s' with q='%s'", view, userPrompt)
				ids := make([]string, len(results))
				for i, r := range results {
					ids[i] = r.ID
				}
				ads, _ = fetchAdsByIDs(ids)
				usedVectorSearch = true
			}
		}
	}
	if !usedVectorSearch {
		if userPrompt == "" && userID != 0 {
			embedding, err := vector.GetUserPersonalizedEmbedding(userID, false)
			if err == nil && embedding != nil {
				results, _, err := vector.QuerySimilarAds(embedding, 10, "")
				if err == nil {
					log.Printf("[search] Embedding-based search (user embedding) used for view '%s' for userID=%d", view, userID)
					ids := make([]string, len(results))
					for i, r := range results {
						ids[i] = r.ID
					}
					ads, _ = fetchAdsByIDs(ids)
					usedVectorSearch = true
				}
			}
			if err != nil {
				log.Printf("[embedding] User embedding error for userID=%d: %v", userID, err)
			}
		}
	}
	if !usedVectorSearch {
		log.Printf("[search] Fallback to SQL-based search for view '%s' (q='%s', userID=%d)", view, userPrompt, userID)
		var query SearchQuery
		if structuredQuery != "" {
			err := json.Unmarshal([]byte(structuredQuery), &query)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Invalid structured_query")
			}
		} else {
			query, err = ParseSearchQuery(userPrompt)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Could not parse query")
			}
		}
		ads, _, err = GetNextPage(query, nil, 10, userID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Could not get ads")
		}
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
