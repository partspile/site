package ui

import (
	"fmt"
	"time"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
)

func MapViewResults(ads []ad.Ad, userID int, loc *time.Location) g.Node {
	var viewContent = NoSearchResultsMessage()

	if len(ads) > 0 {
		viewContent = adMapNode(ads, userID, loc)
	}

	return Div(
		ID("searchResults"),
		ViewToggleButtons("map"),
		viewContent,
	)
}

// createAdDataElements creates hidden data elements for each ad with coordinates
func createAdDataElements(ads []ad.Ad) []g.Node {
	var adDataElements []g.Node
	for _, ad := range ads {
		if ad.Latitude.Valid && ad.Longitude.Valid {
			adDataElements = append(adDataElements,
				Div(
					Class("hidden"),
					g.Attr("data-ad-id", fmt.Sprintf("%d", ad.ID)),
					g.Attr("data-lat", fmt.Sprintf("%f", ad.Latitude.Float64)),
					g.Attr("data-lon", fmt.Sprintf("%f", ad.Longitude.Float64)),
					g.Attr("data-title", ad.Title),
					g.Attr("data-price", fmt.Sprintf("%.2f", ad.Price)),
				),
			)
		}
	}
	return adDataElements
}

// MapDataOnly returns just the map data container for HTMX updates
func MapDataOnly(ads []ad.Ad, userID int, loc *time.Location) g.Node {
	return Div(
		ID("map-data"),
		Class("hidden"),
		g.Group(createAdDataElements(ads)),
	)
}

func adMapNode(ads []ad.Ad, userID int, loc *time.Location) g.Node {
	return Div(
		ID("map-view"),
		Class("h-96 w-full rounded border bg-gray-50"),
		// Map container with explicit styling
		Div(
			ID("map-container"),
			Class("h-full w-full"),
			Style("border-radius: inherit; overflow: hidden;"),
		),
		// Hidden inputs for bounding box
		Input(Type("hidden"), ID("min-lat"), Name("minLat")),
		Input(Type("hidden"), ID("max-lat"), Name("maxLat")),
		Input(Type("hidden"), ID("min-lon"), Name("minLon")),
		Input(Type("hidden"), ID("max-lon"), Name("maxLon")),
		// Hidden data container for HTMX updates
		Div(
			ID("map-data"),
			Class("hidden"),
			g.Group(createAdDataElements(ads)),
		),
		// Initialize map after all elements are created
		Script(
			Type("text/javascript"),
			g.Raw("initMap();"),
		),
	)
}
