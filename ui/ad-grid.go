package ui

import (
	"fmt"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/user"
)

// AdGridNode renders a grid view of ad
func AdGridNode(ad ad.Ad, u *user.User, loc *time.Location) g.Node {
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
				g.If(u != nil, BookmarkButton(ad)),
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

	return b2util.GetB2ImageURL(adID, idx, "480w", token)
}

func gridImageWithIndex(adID int, idx int, alt string) g.Node {
	src := adGridImageSrc(adID, idx)

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
	return gridImageWithNav(ad, 1)
}

func gridImageWithNav(ad ad.Ad, currentIdx int) g.Node {
	containerID := fmt.Sprintf("grid-image-container-%d", ad.ID)

	return Div(
		ID(containerID),
		Class("relative group"),
		gridImageWithIndex(ad.ID, currentIdx, ad.Title),
		g.If(ad.ImageCount > 1, gridNavButtons(ad, currentIdx)),
	)
}

func gridNavButtons(ad ad.Ad, currentIdx int) g.Node {
	prevIdx := (currentIdx-2+ad.ImageCount)%ad.ImageCount + 1
	nextIdx := currentIdx%ad.ImageCount + 1

	return g.Group([]g.Node{
		// Left button
		Button(
			Type("button"),
			Class("absolute left-2 top-1/2 transform -translate-y-1/2 bg-white/50 rounded-full w-10 h-10 flex items-center justify-center shadow-lg hover:bg-white/60 focus:outline-none cursor-pointer z-20 opacity-100 md:opacity-0 md:group-hover:opacity-100 md:transition-opacity"),
			hx.Get(fmt.Sprintf("/ad/grid-image/%d/%d", ad.ID, prevIdx)),
			hx.Target(fmt.Sprintf("#grid-image-container-%d", ad.ID)),
			hx.Swap("outerHTML"),
			g.Attr("onclick", "event.stopPropagation()"),
			icon("/images/left.svg", "Previous", "w-6 h-6"),
		),
		// Right button
		Button(
			Type("button"),
			Class("absolute right-2 top-1/2 transform -translate-y-1/2 bg-white/50 rounded-full w-10 h-10 flex items-center justify-center shadow-lg hover:bg-white/60 focus:outline-none cursor-pointer z-20 opacity-100 md:opacity-0 md:group-hover:opacity-100 md:transition-opacity"),
			hx.Get(fmt.Sprintf("/ad/grid-image/%d/%d", ad.ID, nextIdx)),
			hx.Target(fmt.Sprintf("#grid-image-container-%d", ad.ID)),
			hx.Swap("outerHTML"),
			g.Attr("onclick", "event.stopPropagation()"),
			icon("/images/right.svg", "Next", "w-6 h-6"),
		),
	})
}

// GridImageWithNav creates the grid image container for HTMX swapping
func GridImageWithNav(ad ad.Ad, currentIdx int) g.Node {
	return gridImageWithNav(ad, currentIdx)
}
