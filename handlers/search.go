package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sort"
	"strings"
	"time"

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
)

// runEmbeddingSearch runs vector search with optional filters
func runEmbeddingSearch(embedding []float32, cursor string, currentUser *user.User, threshold float64, k int, filter *qdrant.Filter) ([]ad.Ad, string, error) {
	var ids []int
	var nextCursor string
	var err error

	// Get results with threshold filtering at Qdrant level
	if filter != nil {
		ids, nextCursor, err = vector.QuerySimilarAdIDsWithFilter(embedding, filter, k, cursor, threshold)
	} else {
		ids, nextCursor, err = vector.QuerySimilarAdIDs(embedding, k, cursor, threshold)
	}

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

// Embedding-based search with user query
func queryEmbedding(userPrompt string, currentUser *user.User, cursor string, threshold float64, k int, filter *qdrant.Filter) ([]ad.Ad, string, error) {
	log.Printf("[queryEmbedding] Generating embedding for user query: %s", userPrompt)
	embedding, err := vector.EmbedTextCached(userPrompt)
	if err != nil {
		return nil, "", err
	}
	return runEmbeddingSearch(embedding, cursor, currentUser, threshold, k, filter)
}

// Embedding-based search with user embedding
func userEmbedding(currentUser *user.User, cursor string, threshold float64, k int, filter *qdrant.Filter) ([]ad.Ad, string, error) {
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
	return runEmbeddingSearch(embedding, cursor, currentUser, threshold, k, filter)
}

// Embedding-based search with site-level vector
func siteEmbedding(currentUser *user.User, cursor string, threshold float64, k int, filter *qdrant.Filter) ([]ad.Ad, string, error) {
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
	log.Printf("[siteEmbedding] called with userID=%d, cursor=%s, threshold=%.2f", userID, cursor, threshold)
	embedding, err := vector.GetSiteLevelVector()
	if err != nil {
		log.Printf("[siteEmbedding] GetSiteLevelVector error: %v", err)
		return nil, "", err
	}
	if embedding == nil {
		log.Printf("[siteEmbedding] GetSiteLevelVector returned nil embedding")
		return nil, "", nil
	}
	return runEmbeddingSearch(embedding, cursor, currentUser, threshold, k, filter)
}

// Search strategy for both HandleSearch and HandleSearchPage
func performSearch(userPrompt string, currentUser *user.User, cursorStr string, threshold float64, k int, filter *qdrant.Filter) ([]ad.Ad, string, error) {
	userID := getUserIDFromUser(currentUser)
	log.Printf("[performSearch] userPrompt='%s', userID=%d, cursorStr='%s', threshold=%.2f, k=%d, filter=%v", userPrompt, userID, cursorStr, threshold, k, filter)

	if userPrompt != "" {
		return queryEmbedding(userPrompt, currentUser, cursorStr, threshold, k, filter)
	}

	if userPrompt == "" && userID != 0 {
		return userEmbedding(currentUser, cursorStr, threshold, k, filter)
	}

	if userPrompt == "" && userID == 0 {
		return siteEmbedding(currentUser, cursorStr, threshold, k, filter)
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
	if bounds == nil {
		return nil, "", fmt.Errorf("bounds cannot be nil for geo box search")
	}

	userID := getUserIDFromUser(currentUser)
	log.Printf("[performGeoBoxSearch] userPrompt='%s', userID=%d, cursorStr='%s', bounds=%+v", userPrompt, userID, cursorStr, bounds)

	// Build geo filter
	geoFilter := vector.BuildBoundingBoxGeoFilter(bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon)

	// Use performSearch with the geo filter
	return performSearch(userPrompt, currentUser, cursorStr, threshold, config.QdrantSearchInitialK, geoFilter)
}

// Render new ad button based on user login
func renderNewAdButton(userID int) g.Node {
	if userID != 0 {
		return ui.StyledLink("New Ad", "/new-ad", ui.ButtonPrimary)
	}
	return ui.StyledLinkDisabled("New Ad", ui.ButtonPrimary)
}

// saveUserSearchAndQueue saves user search and queues user for embedding update
func saveUserSearchAndQueue(userPrompt string, userID int) {
	if userPrompt != "" {
		_ = search.SaveUserSearch(sql.NullInt64{Int64: int64(userID), Valid: userID != 0}, userPrompt)
		if userID != 0 {
			// Queue user for background embedding update
			vector.GetEmbeddingQueue().QueueUserForUpdate(userID)
		}
	}
}

func handleSearch(c *fiber.Ctx, viewType string) error {
	view, err := NewView(c, viewType)
	if err != nil {
		return err
	}

	ads, nextCursor, err := view.GetAds()
	if err != nil {
		return err
	}

	view.SaveUserSearch()

	return view.RenderSearchResults(ads, nextCursor)
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

func HandleMapView(c *fiber.Ctx) error {
	return handleSearch(c, "map")
}

func HandleSearch(c *fiber.Ctx) error {
	return handleSearch(c, c.Query("view", "list"))
}

func HandleSearchPage(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")

	view, err := NewView(c, c.Query("view", "list"))
	if err != nil {
		return err
	}

	ads, nextCursor, err := view.GetAds()
	if err != nil {
		return err
	}

	if len(ads) == 0 && view.ShouldShowNoResults() {
		return nil
	}

	return view.RenderSearchPage(ads, nextCursor)
}

func HandleTreeCollapse(c *fiber.Ctx) error {
	q := c.Query("q")
	structuredQueryStr := c.Query("structured_query")
	path := c.Params("*")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	threshold := c.QueryFloat("threshold", config.QdrantSearchThreshold)

	name := parts[len(parts)-1]
	level := len(parts) - 1

	return render(c, ui.CollapsedTreeNodeWithThreshold(name, "/"+path, q, structuredQueryStr, fmt.Sprintf("%.1f", threshold), level))
}

func HandleTreeViewNavigation(c *fiber.Ctx) error {
	q := c.Query("q")
	structuredQueryStr := c.Query("structured_query")
	path := c.Params("*")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		parts = []string{}
	}
	level := len(parts)

	threshold := c.QueryFloat("threshold", config.QdrantSearchThreshold)

	var structuredQuery ad.SearchQuery
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
	userID := getUserID(c)

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

	// For now, we'll skip showing children at this level since extractChildrenFromAds was removed
	// The tree navigation will still work through direct URL navigation
	var children []string

	// Show ads if we're at a leaf level (level 4 = engine, level 5 = category)
	// or if there are no more children to show
	if level >= 4 || len(children) == 0 {
		if len(ads) > 0 {
			loc, _ := time.LoadLocation(c.Get("X-Timezone"))
			for _, ad := range ads {
				childNodes = append(childNodes, ui.AdCardCompactTree(ad, loc, currentUser))
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
	userID := getUserIDFromUser(currentUser)
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
