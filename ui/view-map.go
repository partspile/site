package ui

import (
	"fmt"
	"time"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
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
		loadMapResources(),
	)
}

func adMapNode(ads []ad.Ad, userID int, loc *time.Location) g.Node {
	// Create hidden data elements for each ad with coordinates
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

// loadMapResources loads Leaflet CSS, JS, and map.js directly in the body
func loadMapResources() g.Node {
	return g.Group([]g.Node{
		// Load Leaflet CSS
		Link(
			Rel("stylesheet"),
			Href(config.LeafletCSSURL),
		),
		// Load Leaflet JS
		Script(
			Type("text/javascript"),
			Src(config.LeafletJSURL),
		),
		// Load map.js
		Script(
			Type("text/javascript"),
			Src("/js/map.js"),
		),
	})
}
