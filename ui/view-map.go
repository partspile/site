package ui

import (
	"fmt"
	"time"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/config"
)

func MapViewResults(ads []ad.Ad, userID int, loc *time.Location, bounds *GeoBounds) g.Node {
	var viewContent = NoSearchResultsMessage()

	if len(ads) > 0 {
		viewContent = mapNode(ads, userID, loc, bounds)
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
			// Get the first image index
			firstIdx := 1
			if len(ad.ImageOrderSlice) > 0 {
				firstIdx = ad.ImageOrderSlice[0]
			}

			// Generate the image URL
			imageURL := generateMapImageURL(ad.ID, firstIdx)

			adDataElements = append(adDataElements,
				Div(
					Class("hidden"),
					g.Attr("data-ad-id", fmt.Sprintf("%d", ad.ID)),
					g.Attr("data-lat", fmt.Sprintf("%f", ad.Latitude.Float64)),
					g.Attr("data-lon", fmt.Sprintf("%f", ad.Longitude.Float64)),
					g.Attr("data-title", ad.Title),
					g.Attr("data-price", fmt.Sprintf("%.2f", ad.Price)),
					g.Attr("data-image", imageURL),
				),
			)
		}
	}
	return adDataElements
}

// generateMapImageURL generates a signed B2 image URL for map popup context
func generateMapImageURL(adID int, idx int) string {
	prefix := fmt.Sprintf("%d/", adID)
	token, err := b2util.GetB2DownloadTokenForPrefixCached(prefix)
	if err != nil || token == "" {
		// Return empty string when B2 images aren't available - browser will show broken image
		return ""
	}

	base := fmt.Sprintf("%s/%d/%d", config.B2FileServerURL, adID, idx)
	// Use 480w for map popups - good balance of quality and file size
	return fmt.Sprintf("%s-480w.webp?Authorization=%s", base, token)
}

// MapDataOnly returns just the map data container for HTMX updates
func MapDataOnly(ads []ad.Ad, userID int, loc *time.Location) g.Node {
	return Div(
		ID("map-data"),
		Class("hidden"),
		g.Group(createAdDataElements(ads)),
	)
}

func mapNode(ads []ad.Ad, userID int, loc *time.Location, bounds *GeoBounds) g.Node {
	// Create initialization script with saved map bounds
	var initScript string
	if bounds != nil {
		initScript = fmt.Sprintf("initMap({minLat: %f, maxLat: %f, minLon: %f, maxLon: %f});",
			bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon)
	} else {
		initScript = "initMap();"
	}

	return Div(
		ID("map-view"),
		Class("w-full"),
		// Map container with explicit styling
		Div(
			Class("h-96 w-full rounded border bg-gray-50"),
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
				g.Raw(initScript),
			),
		),
		// Container for ad details below the map
		Div(
			ID("map-ad-details"),
			Class("mt-4"),
		),
	)
}
