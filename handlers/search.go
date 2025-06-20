package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/templates"
)

func HandleSearch(w http.ResponseWriter, r *http.Request) {
	userPrompt := r.URL.Query().Get("q")
	query, err := ParseSearchQuery(userPrompt)
	if err != nil {
		http.Error(w, "Could not parse query", http.StatusBadRequest)
		return
	}

	ads, nextCursor, err := GetNextPage(query, nil, 10)
	if err != nil {
		http.Error(w, "Could not get ads", http.StatusInternalServerError)
		return
	}

	loc, _ := time.LoadLocation(r.Header.Get("X-Timezone"))

	adsMap := make(map[int]ad.Ad)
	for _, ad := range ads {
		adsMap[ad.ID] = ad
	}

	// For the initial search, we render the whole container.
	templates.SearchResultsContainer(templates.SearchSchema(query), adsMap, loc).Render(w)

	// Add the loader if there are more results
	if nextCursor != nil {
		nextCursorStr := EncodeCursor(*nextCursor)
		loaderURL := fmt.Sprintf("/search-page?q=%s&cursor=%s",
			htmlEscape(userPrompt),
			htmlEscape(nextCursorStr))
		loaderHTML := fmt.Sprintf(`<div id="loader" hx-get="%s" hx-trigger="revealed" hx-swap="outerHTML">Loading more...</div>`, loaderURL)
		fmt.Fprint(w, loaderHTML)
	}
}

func HandleSearchPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	userPrompt := r.URL.Query().Get("q")
	cursorStr := r.URL.Query().Get("cursor")

	if cursorStr == "" {
		// This page should not be called without a cursor.
		return
	}

	cursor, err := DecodeCursor(cursorStr)
	if err != nil {
		http.Error(w, "Invalid cursor", http.StatusBadRequest)
		return
	}

	ads, nextCursor, err := GetNextPage(cursor.Query, &cursor, 10)
	if err != nil {
		http.Error(w, "Could not get ads", http.StatusInternalServerError)
		return
	}

	loc, _ := time.LoadLocation(r.Header.Get("X-Timezone"))

	// For subsequent loads, we just render the new ad cards, and the next loader
	for _, ad := range ads {
		templates.AdCard(ad, loc).Render(w)
	}

	if nextCursor != nil {
		nextCursorStr := EncodeCursor(*nextCursor)
		loaderURL := fmt.Sprintf("/search-page?q=%s&cursor=%s",
			htmlEscape(userPrompt),
			htmlEscape(nextCursorStr))
		loaderHTML := fmt.Sprintf(`<div id="loader" hx-get="%s" hx-trigger="revealed" hx-swap="outerHTML">Loading more...</div>`, loaderURL)
		fmt.Fprint(w, loaderHTML)
	}
}
