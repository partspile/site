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

func AdDetail(ad ad.Ad, loc *time.Location, userID int, view string) g.Node {
	return Div(
		ID(adID(ad)),
		Class("border rounded-lg shadow-lg bg-white flex flex-col relative my-4 mx-2 col-span-full"),
		imageNode(ad, view),
		Div(
			Class("p-4 flex flex-col gap-2"),
			// Title and buttons row
			Div(
				Class("flex flex-row items-center justify-between mb-2"),
				Div(Class("font-semibold text-xl truncate"), titleNode(ad)),
				Div(Class("flex flex-row items-center gap-2 ml-2"),
					g.If(userID != 0, BookmarkButton(ad)),
					g.If(userID != 0, messageButton(ad, userID)),
					editButton(ad, userID),
					deleteButton(ad, userID),
				),
			),
			// Age and location row
			Div(
				Class("flex flex-row items-center justify-between text-xs text-gray-500 mb-2"),
				Div(Class("text-gray-400"), ageNode(ad, loc)),
				locationFlagNode(ad),
			),
			// Description
			Div(Class("text-base mt-2"), g.Text(ad.Description)),
		),
	)
}

func priceOverlayNode(ad ad.Ad) g.Node {
	return Div(
		Class("absolute top-0 left-0 bg-white text-green-600 text-base font-normal px-2 rounded-br-md z-10"),
		priceNode(ad),
	)
}

func closeButtonOverlayNode(ad ad.Ad, view string) g.Node {
	return Button(
		Type("button"),
		Class("absolute -top-2 -right-2 bg-gray-800 bg-opacity-80 text-white text-2xl font-bold rounded-full w-10 h-10 flex items-center justify-center shadow-lg z-30 hover:bg-gray-700 focus:outline-none"),
		hx.Get(fmt.Sprintf("/ad/card/%d?view=%s", ad.ID, view)),
		hx.Target(adTarget(ad)),
		hx.Swap("outerHTML"),
		g.Text("×"),
	)
}

// adCarouselImageSrc generates a single signed B2 image URL for carousel context
func AdCarouselImageSrc(adID int, idx int) string {
	prefix := fmt.Sprintf("%d/", adID)
	token, err := b2util.GetB2DownloadTokenForPrefixCached(prefix)
	if err != nil || token == "" {
		// Return empty string when B2 images aren't available - browser will show broken image
		return ""
	}

	return config.GetB2ImageURL(adID, idx, "1200w", token)
}

func AdCarouselImage(adID int, idx int) g.Node {
	return Img(
		Class("object-contain w-full h-full bg-gray-100 transition-opacity duration-200"),
		ID(fmt.Sprintf("ad-carousel-img-%d", adID)),
		Src(AdCarouselImageSrc(adID, idx)),
		Alt(fmt.Sprintf("Image %d", idx)),
	)
}

func AdNoImage() g.Node {
	return Div(
		Class("absolute inset-0 bg-gray-100 flex items-center justify-center"),
		Div(
			Class("text-gray-400 text-sm"),
			g.Text("No Image"),
		),
	)
}

func imageNode(ad ad.Ad, view string) g.Node {
	return Div(
		Class("relative w-full bg-gray-100 overflow-visible"),
		Style("height: 60vh; min-height: 500px; max-height: 800px;"),
		priceOverlayNode(ad),
		closeButtonOverlayNode(ad, view),
		Div(
			Class("relative w-full h-full flex flex-col overflow-hidden rounded-t-lg"),
			Div(
				Class("flex-1 flex items-center justify-center"),
				g.If(ad.ImageCount > 0, AdCarouselImage(ad.ID, 1)),
				g.If(ad.ImageCount == 0, AdNoImage()),
			),
			Div(
				Class("flex-shrink-0 p-4"),
				g.If(ad.ImageCount > 0, thumbnails(ad)),
			),
		),
	)
}

// adThumbnailImageSrc generates a single signed B2 image URL for thumbnail context
func adThumbnailImageSrc(adID int, idx int) string {
	prefix := fmt.Sprintf("%d/", adID)
	token, err := b2util.GetB2DownloadTokenForPrefixCached(prefix)
	if err != nil || token == "" {
		// Return empty string when B2 images aren't available - browser will show broken image
		return ""
	}

	return config.GetB2ImageURL(adID, idx, "160w", token)
}

func adThumbnailImage(adID int, idx int, alt string) g.Node {
	src := adThumbnailImageSrc(adID, idx)

	return Img(
		Src(src),
		Alt(alt),
		Class("object-contain w-full aspect-square bg-gray-100"),
	)
}

func thumbnails(ad ad.Ad) g.Node {
	return Div(
		Class("flex flex-row gap-2 mt-2 px-4 justify-center"),
		g.Group(func() []g.Node {
			nodes := []g.Node{}
			for i := 1; i <= ad.ImageCount; i++ {
				nodes = append(nodes, Button(
					Type("button"),
					Class("border rounded w-16 h-16 overflow-hidden p-0 focus:outline-none"),
					hx.Get(fmt.Sprintf("/ad/image/%d/%d", ad.ID, i)),
					hx.Target(fmt.Sprintf("#ad-carousel-img-%d", ad.ID)),
					hx.Swap("outerHTML"),
					adThumbnailImage(ad.ID, i, fmt.Sprintf("Image %d", i)),
				))
			}
			return nodes
		}()),
	)
}

func messageButton(ad ad.Ad, userID int) g.Node {
	// Don't show message button if user is viewing their own ad
	if userID == ad.UserID {
		return g.Node(nil)
	}

	// Don't show message button if user is not logged in
	if userID == 0 {
		return g.Node(nil)
	}

	return Button(
		Type("button"),
		Class("ml-2 focus:outline-none z-20"),
		Title("Message seller"),
		hx.Get(fmt.Sprintf("/messages/inline/%d?view=%s", ad.ID, "tree")),
		hx.Target(adTarget(ad)),
		hx.Swap("outerHTML"),
		Img(
			Src("/images/message.svg"),
			Alt("Message"),
			Class("w-6 h-6 inline align-middle text-blue-500 hover:text-blue-700"),
		),
	)
}

func deleteButton(ad ad.Ad, userID int) g.Node {
	if userID != ad.UserID {
		return g.Node(nil)
	}

	return Button(
		Type("button"),
		Class("ml-2 focus:outline-none"),
		hx.Delete(fmt.Sprintf("/delete-ad/%d", ad.ID)),
		hx.Target(adTarget(ad)),
		hx.Swap("delete"),
		hx.Confirm("Are you sure you want to delete this ad? This action cannot be undone."),
		Img(
			Src("/images/trashcan.svg"),
			Alt("Delete"),
			Class("w-6 h-6 inline align-middle text-red-500 hover:text-red-700"),
		),
	)
}

func editButton(ad ad.Ad, userID int) g.Node {
	if userID != ad.UserID {
		return g.Node(nil)
	}

	return Button(
		Type("button"),
		Class("ml-2 focus:outline-none"),
		hx.Get(fmt.Sprintf("/ad/edit-partial/%d", ad.ID)),
		hx.Target(adTarget(ad)),
		hx.Swap("outerHTML"),
		Img(
			Src("/images/edit.svg"),
			Alt("Edit"),
			Class("w-6 h-6 inline align-middle text-blue-500 hover:text-blue-700"),
		),
	)
}
