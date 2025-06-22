package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/grok"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vehicle"
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

func HandleSearch(c *fiber.Ctx) error {
	userPrompt := c.Query("q")
	query, err := ParseSearchQuery(userPrompt)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Could not parse query")
	}

	ads, nextCursor, err := GetNextPage(query, nil, 10)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Could not get ads")
	}

	loc, _ := time.LoadLocation(c.Get("X-Timezone"))

	adsMap := make(map[int]ad.Ad)
	for _, ad := range ads {
		adsMap[ad.ID] = ad
	}

	// For the initial search, we render the whole container.
	render(c, ui.SearchResultsContainer(ui.SearchSchema(query), adsMap, loc))

	// Add the loader if there are more results
	if nextCursor != nil {
		nextCursorStr := EncodeCursor(*nextCursor)
		loaderURL := fmt.Sprintf("/search-page?q=%s&cursor=%s",
			htmlEscape(userPrompt),
			htmlEscape(nextCursorStr))
		loaderHTML := fmt.Sprintf(`<div id="loader" hx-get="%s" hx-trigger="revealed" hx-swap="outerHTML">Loading more...</div>`, loaderURL)
		fmt.Fprint(c.Response().BodyWriter(), loaderHTML)
	}
	return nil
}

func HandleSearchPage(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")
	userPrompt := c.Query("q")
	cursorStr := c.Query("cursor")

	if cursorStr == "" {
		// This page should not be called without a cursor.
		return nil
	}

	cursor, err := DecodeCursor(cursorStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid cursor")
	}

	ads, nextCursor, err := GetNextPage(cursor.Query, &cursor, 10)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Could not get ads")
	}

	loc, _ := time.LoadLocation(c.Get("X-Timezone"))

	// For subsequent loads, we just render the new ad cards, and the next loader
	for _, ad := range ads {
		render(c, ui.AdCard(ad, loc))
	}

	if nextCursor != nil {
		nextCursorStr := EncodeCursor(*nextCursor)
		loaderURL := fmt.Sprintf("/search-page?q=%s&cursor=%s",
			htmlEscape(userPrompt),
			htmlEscape(nextCursorStr))
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
	if query.Make == "" && len(query.Years) == 0 && len(query.Models) == 0 &&
		len(query.EngineSizes) == 0 && query.Category == "" && query.SubCategory == "" {
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

func GetNextPage(query SearchQuery, cursor *SearchCursor, limit int) ([]ad.Ad, *SearchCursor, error) {
	// Get filtered page from database
	ads, hasMore, err := ad.GetFilteredAdsPageDB(query, cursor, limit)
	if err != nil {
		return nil, nil, err
	}

	// Create next cursor if there are more results
	var nextCursor *SearchCursor
	if hasMore && len(ads) > 0 {
		last := ads[len(ads)-1]
		nextCursor = &SearchCursor{
			Query:      query,
			LastID:     last.ID,
			LastPosted: last.CreatedAt,
		}
	}

	return ads, nextCursor, nil
}
