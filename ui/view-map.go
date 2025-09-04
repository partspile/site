package ui

import (
	"fmt"
	"time"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
)

func RenderMapViewEmpty(query string, threshold float64, userID int) g.Node {
	return Div(
		ID("searchResults"),
		SearchWidget(userID, "map", query, threshold),
		ViewToggleButtons("map"),
		NoSearchResultsMessage(),
	)
}

func RenderMapViewResults(ads []ad.Ad, userID int, loc *time.Location, query string, loaderURL string, threshold float64) g.Node {
	// For map view, always show the map (even if empty)
	adsMap := make(map[int]ad.Ad, len(ads))
	for _, ad := range ads {
		adsMap[ad.ID] = ad
	}
	viewContent := MapView(adsMap, loc)

	return Div(
		ID("searchResults"),
		SearchWidget(userID, "map", query, threshold),
		ViewToggleButtons("map"),
		viewContent,
	)
}

func RenderMapViewPage(ads []ad.Ad, userID int, loc *time.Location, loaderURL string) g.Node {
	// For map view pagination, we need to add new ads to the existing map
	// This would typically involve JavaScript to add markers to the existing map
	// For now, we'll return the hidden data elements for the new ads

	var adDataElements []g.Node
	for _, ad := range ads {
		if ad.Latitude != nil && ad.Longitude != nil {
			adDataElements = append(adDataElements,
				Div(
					Class("hidden"),
					g.Attr("data-ad-id", fmt.Sprintf("%d", ad.ID)),
					g.Attr("data-lat", fmt.Sprintf("%f", *ad.Latitude)),
					g.Attr("data-lon", fmt.Sprintf("%f", *ad.Longitude)),
					g.Attr("data-title", ad.Title),
					g.Attr("data-price", fmt.Sprintf("%.2f", ad.Price)),
				),
			)
		}
	}

	// Add infinite scroll trigger if there are more results
	if loaderURL != "" {
		trigger := Div(
			Class("h-4"),
			g.Attr("hx-get", loaderURL),
			g.Attr("hx-trigger", "revealed"),
			g.Attr("hx-swap", "outerHTML"),
		)
		adDataElements = append(adDataElements, trigger)
	}

	return g.Group(adDataElements)
}

func MapView(ads map[int]ad.Ad, loc *time.Location) g.Node {
	// Create hidden data elements for each ad with coordinates
	var adDataElements []g.Node
	for _, ad := range ads {
		if ad.Latitude != nil && ad.Longitude != nil {
			adDataElements = append(adDataElements,
				Div(
					Class("hidden"),
					g.Attr("data-ad-id", fmt.Sprintf("%d", ad.ID)),
					g.Attr("data-lat", fmt.Sprintf("%f", *ad.Latitude)),
					g.Attr("data-lon", fmt.Sprintf("%f", *ad.Longitude)),
					g.Attr("data-title", ad.Title),
					g.Attr("data-price", fmt.Sprintf("%.2f", ad.Price)),
				),
			)
		}
	}

	return Div(
		ID("map-view"),
		Class("h-96 w-full rounded border bg-gray-50"),
		// Map container with explicit styling
		Div(
			ID("map-container"),
			Class("h-full w-full z-10"),
			Style("min-height: 384px; position: relative;"),
		),
		// Hidden inputs for bounding box
		Input(Type("hidden"), ID("min-lat"), Name("minLat")),
		Input(Type("hidden"), ID("max-lat"), Name("maxLat")),
		Input(Type("hidden"), ID("min-lon"), Name("minLon")),
		Input(Type("hidden"), ID("max-lon"), Name("maxLon")),
		// Hidden ad data elements
		g.Group(adDataElements),
	)
}

// View-specific loader URL creation function
func CreateMapViewLoaderURL(userPrompt, nextCursor string, threshold float64, bounds *GeoBounds) string {
	if nextCursor == "" {
		return ""
	}
	loaderURL := fmt.Sprintf("/search-page?q=%s&cursor=%s&view=map&threshold=%.1f",
		htmlEscape(userPrompt), htmlEscape(nextCursor), threshold)

	// Add bounding box to loader URL for map view
	if bounds != nil {
		loaderURL += fmt.Sprintf("&minLat=%.6f&maxLat=%.6f&minLon=%.6f&maxLon=%.6f",
			bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon)
	}
	return loaderURL
}
