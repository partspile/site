package ui

import (
	"fmt"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/config"
)

// AdGridNode renders a grid view of ad
func AdGridNode(ad ad.Ad, loc *time.Location, userID int) g.Node {
	var containerClass string = "flex flex-col cursor-pointer"
	if ad.IsArchived() {
		containerClass += " bg-red-100"
	}

	return Div(
		ID(adID(ad)),
		Class(containerClass),

		hx.Get(fmt.Sprintf("/ad/detail/%d?view=grid", ad.ID)),
		hx.Target(adTarget(ad)),
		hx.Swap("outerHTML"),

		gridImageNode(ad),
		Div(
			Class("p-2 flex flex-col gap-1"),

			// Price and bookmark row
			Div(
				Class("flex flex-row items-center justify-between"),
				Div(
					Class("text-green-600 font-semibold text-base"),
					Class("font-semibold text-base"),
					priceNode(ad),
				),
				g.If(userID != 0, BookmarkButton(ad)),
			),

			// Title
			titleNode(ad),

			// Age and location row
			Div(
				Class("flex flex-row items-center justify-between text-xs text-gray-500"),
				ageNode(ad, loc),
				location(ad),
			),
		),
	)
}

// adGridImageSrc generates a single signed B2 image URL for grid context
func adGridImageSrc(adID int, idx int) string {
	prefix := fmt.Sprintf("%d/", adID)
	token, err := b2util.GetB2DownloadTokenForPrefixCached(prefix)
	if err != nil || token == "" {
		// Return empty string when B2 images aren't available - browser will show broken image
		return ""
	}

	return config.GetB2ImageURL(adID, idx, "480w", token)
}

func gridImage(adID int, alt string) g.Node {
	src := adGridImageSrc(adID, 1)

	return Img(
		Class("rounded-md w-full h-60 object-cover"),
		Class("rounded-md w-full h-48 object-cover"),
		Src(src),
		Alt(alt),
	)
}

func gridNoImage() g.Node {
	return Div(
		Class("rounded-md w-full h-60 bg-gray-100 border border-gray-300 flex items-center justify-center text-gray-500"),
		Class("rounded-md w-full h-48 bg-gray-100 border border-gray-300 flex items-center justify-center text-gray-500"),
		g.Text("No Image"),
	)
}

func gridImageNode(ad ad.Ad) g.Node {
	if ad.ImageCount == 0 {
		return gridNoImage()
	}
	return gridImage(ad.ID, ad.Title)
}
