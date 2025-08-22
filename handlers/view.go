package handlers

import (
	"fmt"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/ui"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// View interface defines the contract for different view implementations
type View interface {
	// GetAds retrieves ads for this view with appropriate search strategy
	GetAds() ([]ad.Ad, string, error)

	// RenderSearchResults renders the complete search results including container, ads, and pagination
	RenderSearchResults(ads []ad.Ad, nextCursor string) error

	// RenderSearchPage renders just the ads and infinite scroll for pagination
	RenderSearchPage(ads []ad.Ad, nextCursor string) error

	// SaveUserSearch saves user search and queues user for embedding update
	SaveUserSearch()

	// ShouldShowNoResults determines if this view should show a no-results message
	ShouldShowNoResults() bool

	// GetSearchK returns the appropriate k value for search
	GetSearchK() int
}

// ListView implements the View interface for list view
type ListView struct {
	ctx *fiber.Ctx
}

// NewListView creates a new list view
func NewListView(ctx *fiber.Ctx) *ListView {
	return &ListView{ctx: ctx}
}

func (v *ListView) GetAds() ([]ad.Ad, string, error) {
	userPrompt := v.ctx.Query("q")
	cursor := v.ctx.Query("cursor")
	threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
	currentUser, _ := CurrentUser(v.ctx)
	ads, nextCursor, err := performSearch(userPrompt, currentUser, cursor, threshold, config.QdrantSearchPageSize, nil)
	if err == nil {
		log.Printf("[ListView.GetAds] ads returned: %d", len(ads))
		log.Printf("[ListView.GetAds] Final ad order: %v", func() []int {
			result := make([]int, len(ads))
			for i, ad := range ads {
				result[i] = ad.ID
			}
			return result
		}())
	}
	return ads, nextCursor, err
}

func (v *ListView) RenderSearchResults(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	currentUser, _ := CurrentUser(v.ctx)
	userPrompt := v.ctx.Query("q")
	userID := getUserID(v.ctx)
	newAdButton := renderNewAdButton(userID)
	threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)

	// Check if we should show no results message
	if len(ads) == 0 && v.ShouldShowNoResults() {
		return render(v.ctx, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), nil, nil, currentUser, loc, "list", userPrompt, "", fmt.Sprintf("%.1f", threshold)))
	}

	// Create loader URL if there are more results
	var loaderURL string
	if nextCursor != "" {
		loaderURL = createLoaderURL(userPrompt, nextCursor, "list", threshold, nil)
	}

	return render(v.ctx, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), ads, nil, currentUser, loc, "list", userPrompt, loaderURL, fmt.Sprintf("%.1f", threshold)))
}

func (v *ListView) RenderSearchPage(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	currentUser, _ := CurrentUser(v.ctx)

	// Render ads in list view format
	for _, ad := range ads {
		render(v.ctx, ui.AdCardCompactList(ad, loc, currentUser))
		// Add separator after each ad
		render(v.ctx, Div(Class("border-b border-gray-200")))
	}

	// Add infinite scroll trigger if there are more results
	if nextCursor != "" {
		userPrompt := v.ctx.Query("q")
		threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
		nextPageURL := createLoaderURL(userPrompt, nextCursor, "list", threshold, nil)
		if nextPageURL != "" {
			renderInfiniteScrollTrigger(v.ctx, nextPageURL, "list")
		}
	}

	return nil
}

func (v *ListView) SaveUserSearch() {
	userPrompt := v.ctx.Query("q")
	userID := getUserID(v.ctx)
	saveUserSearchAndQueue(userPrompt, userID)
}

func (v *ListView) ShouldShowNoResults() bool {
	return true
}

func (v *ListView) GetSearchK() int {
	return config.QdrantSearchPageSize
}

// GridView implements the View interface for grid view
type GridView struct {
	ctx *fiber.Ctx
}

// NewGridView creates a new grid view
func NewGridView(ctx *fiber.Ctx) *GridView {
	return &GridView{ctx: ctx}
}

func (v *GridView) GetAds() ([]ad.Ad, string, error) {
	userPrompt := v.ctx.Query("q")
	cursor := v.ctx.Query("cursor")
	threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
	currentUser, _ := CurrentUser(v.ctx)
	ads, nextCursor, err := performSearch(userPrompt, currentUser, cursor, threshold, config.QdrantSearchPageSize, nil)
	if err == nil {
		log.Printf("[GridView.GetAds] ads returned: %d", len(ads))
		log.Printf("[GridView.GetAds] Final ad order: %v", func() []int {
			result := make([]int, len(ads))
			for i, ad := range ads {
				result[i] = ad.ID
			}
			return result
		}())
	}
	return ads, nextCursor, err
}

func (v *GridView) RenderSearchResults(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	currentUser, _ := CurrentUser(v.ctx)
	userPrompt := v.ctx.Query("q")
	userID := getUserID(v.ctx)
	newAdButton := renderNewAdButton(userID)
	threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)

	// Check if we should show no results message
	if len(ads) == 0 && v.ShouldShowNoResults() {
		return render(v.ctx, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), nil, nil, currentUser, loc, "grid", userPrompt, "", fmt.Sprintf("%.1f", threshold)))
	}

	// Create loader URL if there are more results
	var loaderURL string
	if nextCursor != "" {
		loaderURL = createLoaderURL(userPrompt, nextCursor, "grid", threshold, nil)
	}

	return render(v.ctx, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), ads, nil, currentUser, loc, "grid", userPrompt, loaderURL, fmt.Sprintf("%.1f", threshold)))
}

func (v *GridView) RenderSearchPage(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	currentUser, _ := CurrentUser(v.ctx)

	// Render ads in grid view format
	for _, ad := range ads {
		render(v.ctx, ui.AdCardExpandable(ad, loc, currentUser, "grid"))
	}

	// Add infinite scroll trigger if there are more results
	if nextCursor != "" {
		userPrompt := v.ctx.Query("q")
		threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
		nextPageURL := createLoaderURL(userPrompt, nextCursor, "grid", threshold, nil)
		if nextPageURL != "" {
			renderInfiniteScrollTrigger(v.ctx, nextPageURL, "grid")
		}
	}

	return nil
}

func (v *GridView) SaveUserSearch() {
	userPrompt := v.ctx.Query("q")
	userID := getUserID(v.ctx)
	saveUserSearchAndQueue(userPrompt, userID)
}

func (v *GridView) ShouldShowNoResults() bool {
	return true
}

func (v *GridView) GetSearchK() int {
	return config.QdrantSearchPageSize
}

// MapView implements the View interface for map view
type MapView struct {
	ctx    *fiber.Ctx
	bounds *GeoBounds
}

// NewMapView creates a new map view
func NewMapView(ctx *fiber.Ctx, bounds *GeoBounds) *MapView {
	return &MapView{ctx: ctx, bounds: bounds}
}

func (v *MapView) GetAds() ([]ad.Ad, string, error) {
	userPrompt := v.ctx.Query("q")
	cursor := v.ctx.Query("cursor")
	threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
	currentUser, _ := CurrentUser(v.ctx)
	var ads []ad.Ad
	var nextCursor string
	var err error
	if v.bounds != nil {
		ads, nextCursor, err = performGeoBoxSearch(userPrompt, currentUser, cursor, v.bounds, threshold)
	} else {
		ads, nextCursor, err = performSearch(userPrompt, currentUser, cursor, threshold, config.QdrantSearchInitialK, nil)
	}
	if err == nil {
		log.Printf("[MapView.GetAds] ads returned: %d", len(ads))
		log.Printf("[MapView.GetAds] Final ad order: %v", func() []int {
			result := make([]int, len(ads))
			for i, ad := range ads {
				result[i] = ad.ID
			}
			return result
		}())
	}
	return ads, nextCursor, err
}

func (v *MapView) RenderSearchResults(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	currentUser, _ := CurrentUser(v.ctx)
	userPrompt := v.ctx.Query("q")
	userID := getUserID(v.ctx)
	newAdButton := renderNewAdButton(userID)
	threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)

	// Check if we should show no results message
	if len(ads) == 0 && v.ShouldShowNoResults() {
		return render(v.ctx, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), nil, nil, currentUser, loc, "map", userPrompt, "", fmt.Sprintf("%.1f", threshold)))
	}

	// Create loader URL if there are more results
	var loaderURL string
	if nextCursor != "" {
		loaderURL = createLoaderURL(userPrompt, nextCursor, "map", threshold, v.bounds)
		// Add bounding box to loader URL for map view
		if v.bounds != nil {
			loaderURL += fmt.Sprintf("&minLat=%.6f&maxLat=%.6f&minLon=%.6f&maxLon=%.6f",
				v.bounds.MinLat, v.bounds.MaxLat, v.bounds.MinLon, v.bounds.MaxLon)
		}
	}

	return render(v.ctx, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), ads, nil, currentUser, loc, "map", userPrompt, loaderURL, fmt.Sprintf("%.1f", threshold)))
}

func (v *MapView) RenderSearchPage(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	currentUser, _ := CurrentUser(v.ctx)

	// Render ads in map view format (same as list for now)
	for _, ad := range ads {
		render(v.ctx, ui.AdCardCompactList(ad, loc, currentUser))
		// Add separator after each ad
		render(v.ctx, Div(Class("border-b border-gray-200")))
	}

	// Add infinite scroll trigger if there are more results
	if nextCursor != "" {
		userPrompt := v.ctx.Query("q")
		threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
		nextPageURL := createLoaderURL(userPrompt, nextCursor, "map", threshold, v.bounds)
		// Add bounding box to loader URL for map view
		if v.bounds != nil {
			nextPageURL += fmt.Sprintf("&minLat=%.6f&maxLat=%.6f&minLon=%.6f&maxLon=%.6f",
				v.bounds.MinLat, v.bounds.MaxLat, v.bounds.MinLon, v.bounds.MaxLon)
		}
		if nextPageURL != "" {
			renderInfiniteScrollTrigger(v.ctx, nextPageURL, "map")
		}
	}

	return nil
}

func (v *MapView) SaveUserSearch() {
	userPrompt := v.ctx.Query("q")
	userID := getUserID(v.ctx)
	saveUserSearchAndQueue(userPrompt, userID)
}

func (v *MapView) ShouldShowNoResults() bool {
	return false // Map view continues to show empty map
}

func (v *MapView) GetSearchK() int {
	return config.QdrantSearchInitialK
}

// TreeView implements the View interface for tree view
type TreeView struct {
	ctx *fiber.Ctx
}

// NewTreeView creates a new tree view
func NewTreeView(ctx *fiber.Ctx) *TreeView {
	return &TreeView{ctx: ctx}
}

func (v *TreeView) GetAds() ([]ad.Ad, string, error) {
	userPrompt := v.ctx.Query("q")
	cursor := v.ctx.Query("cursor")
	threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
	currentUser, _ := CurrentUser(v.ctx)
	ads, nextCursor, err := performSearch(userPrompt, currentUser, cursor, threshold, config.QdrantSearchPageSize, nil)
	if err == nil {
		log.Printf("[TreeView.GetAds] ads returned: %d", len(ads))
		log.Printf("[TreeView.GetAds] Final ad order: %v", func() []int {
			result := make([]int, len(ads))
			for i, ad := range ads {
				result[i] = ad.ID
			}
			return result
		}())
	}
	return ads, nextCursor, err
}

func (v *TreeView) RenderSearchResults(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	currentUser, _ := CurrentUser(v.ctx)
	userPrompt := v.ctx.Query("q")
	userID := getUserID(v.ctx)
	newAdButton := renderNewAdButton(userID)
	threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)

	// Check if we should show no results message
	if len(ads) == 0 && v.ShouldShowNoResults() {
		return render(v.ctx, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), nil, nil, currentUser, loc, "tree", userPrompt, "", fmt.Sprintf("%.1f", threshold)))
	}

	// Create loader URL if there are more results
	var loaderURL string
	if nextCursor != "" {
		loaderURL = createLoaderURL(userPrompt, nextCursor, "tree", threshold, nil)
	}

	return render(v.ctx, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), ads, nil, currentUser, loc, "tree", userPrompt, loaderURL, fmt.Sprintf("%.1f", threshold)))
}

func (v *TreeView) RenderSearchPage(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	currentUser, _ := CurrentUser(v.ctx)

	// Render ads in tree view format (same as list for now)
	for _, ad := range ads {
		render(v.ctx, ui.AdCardCompactTree(ad, loc, currentUser))
		// Add separator after each ad
		render(v.ctx, Div(Class("border-b border-gray-200")))
	}

	// Add infinite scroll trigger if there are more results
	if nextCursor != "" {
		userPrompt := v.ctx.Query("q")
		threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
		nextPageURL := createLoaderURL(userPrompt, nextCursor, "tree", threshold, nil)
		if nextPageURL != "" {
			renderInfiniteScrollTrigger(v.ctx, nextPageURL, "tree")
		}
	}

	return nil
}

func (v *TreeView) SaveUserSearch() {
	userPrompt := v.ctx.Query("q")
	userID := getUserID(v.ctx)
	saveUserSearchAndQueue(userPrompt, userID)
}

func (v *TreeView) ShouldShowNoResults() bool {
	return true
}

func (v *TreeView) GetSearchK() int {
	return config.QdrantSearchPageSize
}

// extractMapBounds extracts geographic bounding box parameters for map view
func extractMapBounds(c *fiber.Ctx) *GeoBounds {
	minLatStr := c.Query("minLat")
	maxLatStr := c.Query("maxLat")
	minLonStr := c.Query("minLon")
	maxLonStr := c.Query("maxLon")

	if minLatStr == "" || maxLatStr == "" || minLonStr == "" || maxLonStr == "" {
		return nil
	}

	minLat, err1 := strconv.ParseFloat(minLatStr, 64)
	maxLat, err2 := strconv.ParseFloat(maxLatStr, 64)
	minLon, err3 := strconv.ParseFloat(minLonStr, 64)
	maxLon, err4 := strconv.ParseFloat(maxLonStr, 64)

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return nil
	}

	return &GeoBounds{
		MinLat: minLat,
		MaxLat: maxLat,
		MinLon: minLon,
		MaxLon: maxLon,
	}
}

// createLoaderURL creates the loader URL for pagination
func createLoaderURL(userPrompt, nextCursor, view string, threshold float64, bounds *GeoBounds) string {
	if nextCursor == "" {
		return ""
	}

	loaderURL := fmt.Sprintf("/search-page?q=%s&cursor=%s&view=%s&threshold=%.1f",
		htmlEscape(userPrompt), htmlEscape(nextCursor), htmlEscape(view), threshold)

	// Add bounding box to loader URL for map view
	if view == "map" && bounds != nil {
		loaderURL += fmt.Sprintf("&minLat=%.6f&maxLat=%.6f&minLon=%.6f&maxLon=%.6f",
			bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon)
	}

	return loaderURL
}

// renderInfiniteScrollTrigger renders the infinite scroll trigger for pagination
func renderInfiniteScrollTrigger(c *fiber.Ctx, nextPageURL, view string) {
	if nextPageURL == "" {
		log.Printf("[renderInfiniteScrollTrigger] No infinite scroll trigger - no more results")
		return
	}

	log.Printf("[renderInfiniteScrollTrigger] Adding infinite scroll trigger with URL: %s", nextPageURL)

	// Create trigger that matches the view style
	render(c, Div(
		Class("h-4"),
		g.Attr("hx-get", nextPageURL),
		g.Attr("hx-trigger", "revealed"),
		g.Attr("hx-swap", "outerHTML"),
	))
}

// NewView creates the appropriate view implementation based on view type
func NewView(ctx *fiber.Ctx, viewType string) (View, error) {
	switch viewType {
	case "list":
		return NewListView(ctx), nil
	case "grid":
		return NewGridView(ctx), nil
	case "map":
		bounds := extractMapBounds(ctx)
		return NewMapView(ctx, bounds), nil
	case "tree":
		return NewTreeView(ctx), nil
	default:
		return nil, fmt.Errorf("invalid view type: %s", viewType)
	}
}

// HandleSearchAPI returns search results as JSON for JavaScript consumption
func HandleSearchAPI(c *fiber.Ctx) error {
	view, err := NewView(c, c.Query("view", "list"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid view type: %s", c.Query("view", "list")),
			"ads":   []ad.Ad{},
		})
	}

	ads, nextCursor, err := view.GetAds()
	if err != nil {
		log.Printf("[HandleSearchAPI] Search error: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Search failed",
			"ads":   []ad.Ad{},
		})
	}

	view.SaveUserSearch()

	log.Printf("[HandleSearchAPI] ads returned: %d", len(ads))

	// Return JSON response
	return c.JSON(fiber.Map{
		"ads":        ads,
		"nextCursor": nextCursor,
		"count":      len(ads),
	})
}
