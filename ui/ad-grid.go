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
	// Get the first image index
	firstIdx := 1
	if len(ad.ImageOrderSlice) > 0 {
		firstIdx = ad.ImageOrderSlice[0]
	}

	imageNode := Div(
		Class("relative w-full h-48 bg-gray-100 overflow-hidden"),
		adGridImage(ad.ID, firstIdx, ad.Title),
		Div(
			Class("absolute top-0 left-0 bg-white text-green-600 text-base font-normal px-2 rounded-br-md"),
			priceNode(ad),
		),
	)

	return Div(
		ID(adID(ad)),
		Class("border rounded-lg shadow-sm bg-white flex flex-col cursor-pointer hover:shadow-md transition-shadow"),

		hx.Get(fmt.Sprintf("/ad/detail/%d?view=grid", ad.ID)),
		hx.Target(adTarget(ad)),
		hx.Swap("outerHTML"),

		Div(
			Class("rounded-t-lg overflow-hidden"),
			imageNode,
		),
		Div(
			Class("p-2 flex flex-col gap-1"),
			// Title and bookmark row
			Div(
				Class("flex flex-row items-center justify-between"),
				Div(Class("font-semibold text-base truncate"), titleNode(ad)),
				g.If(userID != 0, BookmarkButton(ad)),
			),
			// Age and location row
			Div(
				Class("flex flex-row items-center justify-between text-xs text-gray-500"),
				Div(Class("text-gray-400"), ageNode(ad, loc)),
				locationFlagNode(ad),
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

	base := fmt.Sprintf("%s/%d/%d", config.B2FileServerURL, adID, idx)
	// Use 480w for grid cards - good balance of quality and file size for fixed grid layout
	return fmt.Sprintf("%s-480w.webp?Authorization=%s", base, token)
}

func adGridImage(adID int, idx int, alt string) g.Node {
	src := adGridImageSrc(adID, idx)

	return Img(
		Src(src),
		Alt(alt),
		Class("object-contain w-full aspect-square bg-gray-100"),
	)
}
