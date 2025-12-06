package ui

import (
	"fmt"
	"net/url"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/minio"
)

// AdGridNode renders a grid view of ad
func AdGridNode(a ad.Ad, userID int, loc *time.Location, csrfToken string) g.Node {
	var containerClass string = "flex flex-col cursor-pointer"
	if a.IsArchived() {
		containerClass += " bg-red-100"
	}

	return Div(
		ID(adID(a)),
		Class(containerClass),

		hx.Get(fmt.Sprintf("/ad/detail/%d?view=grid", a.ID)),
		hx.Target(adTarget(a)),
		hx.Swap("outerHTML show:top transition:true"),

		gridImageNode(a),
		Div(
			Class("p-2 flex flex-col gap-1"),

			// Price and bookmark row
			Div(
				Class("flex flex-row items-center justify-between"),
				Div(
					Class("text-green-600 font-semibold text-base"),
					Class("font-semibold text-base"),
					priceNode(a),
				),
				g.If(userID != 0, BookmarkButton(a, csrfToken)),
			),

			// Title
			titleNode(a),

			// Age and location row
			Div(
				Class("flex flex-row items-center justify-between text-xs text-gray-500"),
				ageNode(a, loc, false),
				location(a),
			),
		),
	)
}

// adGridImageSrc generates a single MinIO image URL for grid context
func adGridImageSrc(adID int, idx int) string {
	url, err := minio.GetImageURL(adID, idx, "480w")
	if err != nil || url == "" {
		// Return empty string when MinIO images aren't available - browser will show broken image
		return ""
	}

	return url
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

func gridImageNode(a ad.Ad) g.Node {
	if a.ImageCount == 0 {
		return gridNoImage()
	}
	return gridImageWithNav(a.ID, a.ImageCount, a.Title, 1)
}

func gridImageWithNav(adID, imageCount int, title string, currentIdx int) g.Node {
	containerID := fmt.Sprintf("grid-image-container-%d", adID)

	return Div(
		ID(containerID),
		Class("relative group"),
		gridImageWithIndex(adID, currentIdx, title),
		g.If(imageCount > 1, gridNavButtons(adID, imageCount, title, currentIdx)),
	)
}

func gridNavButtons(adID, imageCount int, title string, currentIdx int) g.Node {
	prevIdx := (currentIdx-2+imageCount)%imageCount + 1
	nextIdx := currentIdx%imageCount + 1

	// URL encode title for query parameter
	titleParam := ""
	if title != "" {
		titleParam = fmt.Sprintf("&title=%s", url.QueryEscape(title))
	}

	return g.Group([]g.Node{
		// Left button
		Button(
			Type("button"),
			Class("absolute left-2 top-1/2 transform -translate-y-1/2 bg-white/50 rounded-full w-10 h-10 flex items-center justify-center shadow-lg hover:bg-white/60 focus:outline-none cursor-pointer z-20 opacity-100 md:opacity-0 md:group-hover:opacity-100 md:transition-opacity"),
			hx.Get(fmt.Sprintf("/ad/grid-image/%d/%d?count=%d%s", adID, prevIdx, imageCount, titleParam)),
			hx.Target(fmt.Sprintf("#grid-image-container-%d", adID)),
			hx.Swap("outerHTML"),
			g.Attr("onclick", "event.stopPropagation()"),
			icon("/images/left.svg", "Previous", "w-6 h-6"),
		),
		// Right button
		Button(
			Type("button"),
			Class("absolute right-2 top-1/2 transform -translate-y-1/2 bg-white/50 rounded-full w-10 h-10 flex items-center justify-center shadow-lg hover:bg-white/60 focus:outline-none cursor-pointer z-20 opacity-100 md:opacity-0 md:group-hover:opacity-100 md:transition-opacity"),
			hx.Get(fmt.Sprintf("/ad/grid-image/%d/%d?count=%d%s", adID, nextIdx, imageCount, titleParam)),
			hx.Target(fmt.Sprintf("#grid-image-container-%d", adID)),
			hx.Swap("outerHTML"),
			g.Attr("onclick", "event.stopPropagation()"),
			icon("/images/right.svg", "Next", "w-6 h-6"),
		),
	})
}

// GridImageWithNav creates the grid image container for HTMX swapping
// Can be called with either full ad object or just adID/imageCount/title/currentIdx
func GridImageWithNav(adID, imageCount int, title string, currentIdx int) g.Node {
	return gridImageWithNav(adID, imageCount, title, currentIdx)
}
