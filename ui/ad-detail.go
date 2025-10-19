package ui

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/config"
)

func AdDetail(ad ad.Ad, loc *time.Location, userID int, view string) g.Node {
	// Determine full class strings based on deleted status
	var containerClass, contentClass string
	if ad.IsArchived() {
		containerClass = "rounded-lg shadow-xl/50 bg-red-100 flex flex-col relative my-4 mx-2 col-span-full overflow-hidden"
		contentClass = "p-4 flex flex-col bg-red-100"
	} else {
		containerClass = "rounded-lg shadow-xl/50 flex flex-col relative my-4 mx-2 col-span-full overflow-hidden"
		contentClass = "p-4 flex flex-col"
	}

	isOwner := userID == ad.UserID && userID != 0

	return Div(
		ID(adID(ad)),
		Class(containerClass),
		imageNode(ad, view),
		closeButton(ad, view),
		g.If(ad.IsArchived(), deletedWatermark()),
		Div(
			Class(contentClass),
			// Title row
			Div(
				Class("font-semibold text-xl"),
				titleNode(ad),
			),
			// Age and location row with inline edit for owner
			Div(
				Class("flex flex-row items-center justify-between text-xs text-gray-500 mb-4"),
				Div(Class("text-gray-400"), ageNode(ad, loc)),
				g.If(isOwner && !ad.IsArchived(),
					locationEditable(ad),
				),
				g.If(!isOwner || ad.IsArchived(),
					location(ad),
				),
			),
			// Price row with inline edit for owner and action buttons
			Div(
				Class("flex flex-row items-center justify-between mb-4"),
				g.If(isOwner && !ad.IsArchived(),
					priceEditable(ad),
				),
				g.If(!isOwner || ad.IsArchived(),
					price(ad),
				),
				Div(Class("flex flex-row items-center gap-2 ml-2"),
					// For active ads: show bookmark, message, and delete
					g.If(!ad.IsArchived() && userID != 0, BookmarkButton(ad)),
					g.If(!ad.IsArchived() && userID != 0, messageButton(ad, userID)),
					g.If(!ad.IsArchived(), deleteButton(ad, userID)),
					// For deleted ads: show restore button (owner only)
					g.If(ad.IsArchived(), restoreButton(ad, userID)),
					// Share button (visible to everyone)
					shareButton(ad),
					// Duplicate button (logged in users only)
					g.If(userID != 0, duplicateButton(ad)),
				),
			),
			// Description with inline edit for owner
			g.If(isOwner && !ad.IsArchived(),
				descriptionEditable(ad),
			),
			g.If(!isOwner || ad.IsArchived(),
				description(ad),
			),
			// Part type path
			partTypePath(ad),
		),
	)
}

func deletedWatermark() g.Node {
	return Div(
		Class("absolute top-0 left-0 right-0 bottom-0 flex items-center justify-center pointer-events-none z-50"),
		Div(
			Class("font-bold transform rotate-[-45deg]"),
			Style("user-select: none; font-size: 8rem; color: transparent; -webkit-text-stroke: 3px rgba(220, 38, 38, 0.4); text-stroke: 3px rgba(220, 38, 38, 0.4);"),
			g.Text("DELETED"),
		),
	)
}

func closeButton(ad ad.Ad, view string) g.Node {
	return Button(
		Type("button"),
		Class("absolute top-2 right-2 bg-white border-2 border-gray-800 rounded-full w-10 h-10 flex items-center justify-center shadow-lg z-30 hover:bg-gray-100 focus:outline-none cursor-pointer"),
		hx.Get(fmt.Sprintf("/ad/card/%d?view=%s", ad.ID, view)),
		hx.Target(adTarget(ad)),
		hx.Swap("outerHTML"),
		icon("/images/close.svg", "Close", "w-6 h-6"),
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
		Class("object-cover w-full aspect-[4/3]"),
		ID(fmt.Sprintf("ad-carousel-img-%d", adID)),
		Src(AdCarouselImageSrc(adID, idx)),
		Alt(fmt.Sprintf("Image %d", idx)),
	)
}

func AdNoImage() g.Node {
	return Div(
		Class("flex items-center justify-center h-24 bg-gray-100 text-gray-400 text-sm"),
		g.Text("No Image"),
	)
}

func imageNode(ad ad.Ad, view string) g.Node {
	return Div(
		Class("w-full h-full flex flex-col overflow-hidden rounded-t-lg"),
		carouselImageContainer(ad, 1),
		g.If(ad.ImageCount > 0, Div(
			thumbnails(ad),
		)),
	)
}

func carouselImageContainer(ad ad.Ad, currentIdx int) g.Node {
	containerID := fmt.Sprintf("carousel-image-container-%d", ad.ID)

	return Div(
		ID(containerID),
		Class("relative group"),
		g.If(ad.ImageCount > 0, AdCarouselImage(ad.ID, currentIdx)),
		g.If(ad.ImageCount == 0, AdNoImage()),
		g.If(ad.ImageCount > 1, carouselNavButtons(ad, currentIdx)),
	)
}

func carouselNavButtons(ad ad.Ad, currentIdx int) g.Node {
	if ad.ImageCount == 0 {
		return g.Node(nil)
	}

	prevIdx := (currentIdx-2+ad.ImageCount)%ad.ImageCount + 1
	nextIdx := currentIdx%ad.ImageCount + 1

	return g.Group([]g.Node{
		// Left button
		Button(
			Type("button"),
			Class("absolute left-2 top-1/2 transform -translate-y-1/2 bg-white/50 rounded-full w-10 h-10 flex items-center justify-center shadow-lg hover:bg-white/60 focus:outline-none cursor-pointer z-20 opacity-100 md:opacity-0 md:group-hover:opacity-100 md:transition-opacity"),
			hx.Get(fmt.Sprintf("/ad/image/%d/%d", ad.ID, prevIdx)),
			hx.Target(fmt.Sprintf("#carousel-image-container-%d", ad.ID)),
			hx.Swap("outerHTML"),
			icon("/images/left.svg", "Previous", "w-6 h-6"),
		),
		// Right button
		Button(
			Type("button"),
			Class("absolute right-2 top-1/2 transform -translate-y-1/2 bg-white/50 rounded-full w-10 h-10 flex items-center justify-center shadow-lg hover:bg-white/60 focus:outline-none cursor-pointer z-20 opacity-100 md:opacity-0 md:group-hover:opacity-100 md:transition-opacity"),
			hx.Get(fmt.Sprintf("/ad/image/%d/%d", ad.ID, nextIdx)),
			hx.Target(fmt.Sprintf("#carousel-image-container-%d", ad.ID)),
			hx.Swap("outerHTML"),
			icon("/images/right.svg", "Next", "w-6 h-6"),
		),
	})
}

// CarouselImageContainer creates the carousel image container for HTMX swapping
func CarouselImageContainer(ad ad.Ad, currentIdx int) g.Node {
	return carouselImageContainer(ad, currentIdx)
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
		Class("object-cover w-full h-full"),
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
					Class("rounded w-16 h-16 overflow-hidden"),
					hx.Get(fmt.Sprintf("/ad/image/%d/%d", ad.ID, i)),
					hx.Target(fmt.Sprintf("#carousel-image-container-%d", ad.ID)),
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

	return iconButton(
		"/images/message.svg",
		"Message",
		"Message seller",
		hx.Get(fmt.Sprintf("/modal/ad/message/%d", ad.ID)),
		hx.Target("body"),
		hx.Swap("beforeend"),
	)
}

func deleteButton(ad ad.Ad, userID int) g.Node {
	if userID != ad.UserID {
		return g.Node(nil)
	}

	return iconButton(
		"/images/trashcan.svg",
		"Delete",
		"Delete ad",
		hx.Delete(fmt.Sprintf("/delete-ad/%d", ad.ID)),
		hx.Target(adTarget(ad)),
		hx.Swap("delete"),
		hx.Confirm("Are you sure you want to delete this ad? This action cannot be undone."),
	)
}

func restoreButton(ad ad.Ad, userID int) g.Node {
	if userID != ad.UserID {
		return g.Node(nil)
	}

	return iconButton(
		"/images/restore.svg",
		"Restore",
		"Restore ad",
		hx.Post(fmt.Sprintf("/restore-ad/%d", ad.ID)),
		hx.Target(adTarget(ad)),
		hx.Swap("outerHTML"),
		hx.Confirm("Are you sure you want to restore this ad?"),
	)
}

func description(ad ad.Ad) g.Node {
	return Div(Class("text-base mt-2 whitespace-pre-wrap"), g.Text(ad.Description))
}

func price(ad ad.Ad) g.Node {
	return Div(Class("text-2xl font-bold text-green-600"), priceNode(ad))
}

// Editable field components
func priceEditable(ad ad.Ad) g.Node {
	return Div(
		Class("flex items-center gap-3"),
		price(ad),
		button("Edit",
			withClass("px-4 h-10"),
			withAttributes(
				hx.Get(fmt.Sprintf("/modal/ad/price/%d", ad.ID)),
				hx.Target("body"),
				hx.Swap("beforeend"),
			),
		),
	)
}

func locationEditable(ad ad.Ad) g.Node {
	return Div(
		Class("flex items-center gap-2"),
		location(ad),
		button("Edit",
			withClass("px-4 h-10"),
			withAttributes(
				hx.Get(fmt.Sprintf("/modal/ad/location/%d", ad.ID)),
				hx.Target("body"),
				hx.Swap("beforeend"),
			),
		),
	)
}

func descriptionEditable(ad ad.Ad) g.Node {
	return Div(
		Class("mt-2"),
		description(ad),
		button("Edit",
			withClass("px-4 h-10"),
			withAttributes(
				hx.Get(fmt.Sprintf("/modal/ad/description/%d", ad.ID)),
				hx.Target("body"),
				hx.Swap("beforeend"),
			),
		),
	)
}

// Modal button components
func modalCloseButton() g.Node {
	return Button(
		Type("button"),
		Class("bg-white border-2 border-gray-800 rounded-full w-10 h-10 flex items-center justify-center shadow-lg hover:bg-gray-100 focus:outline-none cursor-pointer"),
		g.Attr("onclick", "this.closest('.fixed').remove()"),
		icon("/images/close.svg", "Close", "w-6 h-6"),
	)
}

// copyIcon creates a copy icon for the modal
func copyIcon() g.Node {
	return icon("/images/copy.svg", "Copy", "w-5 h-5 inline")
}

func modalCopyButton(copyButtonID, urlInputID, copyFeedbackID string) g.Node {
	return button("Copy",
		withClass("px-6 py-3 font-medium shadow-md transition flex items-center gap-2"),
		withAttributes(
			ID(copyButtonID),
			g.Attr("onclick", fmt.Sprintf(`
				const urlInput = document.getElementById('%s');
				const fullURL = urlInput.value;
				navigator.clipboard.writeText(fullURL).then(() => {
					const feedback = document.getElementById('%s');
					feedback.classList.remove('hidden');
					setTimeout(() => {
						feedback.classList.add('hidden');
					}, 2000);
				});
			`, urlInputID, copyFeedbackID)),
			copyIcon(),
		),
	)
}

// Modal dialog components
type editModalConfig struct {
	modalID       string
	title         string
	apiEndpoint   string
	formContent   g.Node
	submitBtnText string
}

func editModal(ad ad.Ad, cfg editModalConfig) g.Node {
	return Div(
		ID(cfg.modalID),
		Class("fixed inset-0 bg-black/30 flex items-center justify-center z-50 p-8"),
		g.Attr("onclick", "this.remove()"),
		Div(
			Class("bg-white rounded-lg w-full shadow-2xl border-2 border-gray-300 flex flex-col overflow-hidden"),
			Style("max-width: 500px; max-height: 70vh"),
			g.Attr("onclick", "event.stopPropagation()"),
			Div(Class("p-8 overflow-y-auto flex-1"),
				H3(Class("text-2xl font-bold mb-6 text-gray-900"), g.Text(cfg.title)),
				Form(
					hx.Post(cfg.apiEndpoint),
					hx.Target(adTarget(ad)),
					hx.Swap("outerHTML"),
					g.Attr("hx-on::after-swap", "this.closest('.fixed').remove();"),
					cfg.formContent,
					Div(
						Class("flex gap-3 justify-end"),
						modalCloseButton(),
						button(cfg.submitBtnText,
							withType("submit"),
							withClass("px-6 py-3 font-medium shadow-md transition"),
						),
					),
				),
			),
		),
	)
}

func PriceEditModal(ad ad.Ad, view string) g.Node {
	modalID := fmt.Sprintf("price-modal-%d", ad.ID)
	return editModal(ad, editModalConfig{
		modalID:       modalID,
		title:         "Update Price",
		apiEndpoint:   fmt.Sprintf("/api/update-ad-price/%d", ad.ID),
		submitBtnText: "Save",
		formContent: Div(Class("mb-6"),
			Label(For("price"), Class("block text-sm font-semibold text-gray-800 mb-3"), g.Text("Price")),
			Input(
				Type("number"),
				ID("price"),
				Name("price"),
				Class("w-full p-3 border-2 border-gray-300 rounded-lg focus:border-blue-500 focus:ring-2 focus:ring-blue-200 transition"),
				Step("0.01"),
				Min("0"),
				Value(fmt.Sprintf("%.2f", ad.Price)),
				Required(),
				g.Attr("autofocus"),
			),
		),
	})
}

func LocationEditModal(ad ad.Ad, view string) g.Node {
	modalID := fmt.Sprintf("location-modal-%d", ad.ID)
	currentLocation := ""
	if ad.RawLocation.Valid {
		currentLocation = ad.RawLocation.String
	}
	return editModal(ad, editModalConfig{
		modalID:       modalID,
		title:         "Update Location",
		apiEndpoint:   fmt.Sprintf("/api/update-ad-location/%d", ad.ID),
		submitBtnText: "Save",
		formContent: Div(Class("mb-6"),
			Label(For("location"), Class("block text-sm font-semibold text-gray-800 mb-3"), g.Text("Location (Zipcode or City)")),
			Input(
				Type("text"),
				ID("location"),
				Name("location"),
				Class("w-full p-3 border-2 border-gray-300 rounded-lg focus:border-blue-500 focus:ring-2 focus:ring-blue-200 transition"),
				Placeholder("e.g., 90210 or Portland, OR"),
				Value(currentLocation),
				g.Attr("autofocus"),
			),
			Div(Class("text-sm text-gray-600 mt-2 bg-blue-50 p-2 rounded"),
				g.Text("Enter a zipcode, city, or address. We'll resolve it automatically.")),
		),
	})
}

func DescriptionEditModal(ad ad.Ad, view string) g.Node {
	modalID := fmt.Sprintf("description-modal-%d", ad.ID)
	return editModal(ad, editModalConfig{
		modalID:       modalID,
		title:         "Add to Description",
		apiEndpoint:   fmt.Sprintf("/api/update-ad-description/%d", ad.ID),
		submitBtnText: "Add",
		formContent: g.Group([]g.Node{
			Div(
				Class("mb-6 p-4 bg-gray-100 rounded-lg border-2 border-gray-200 max-h-40 overflow-y-auto"),
				Div(Class("text-sm font-semibold text-gray-800 mb-3"), g.Text("Current Description:")),
				Div(Class("text-sm whitespace-pre-wrap text-gray-700"), g.Text(ad.Description)),
			),
			Div(Class("mb-6"),
				Label(For("description_addition"), Class("block text-sm font-semibold text-gray-800 mb-3"), g.Text("Add to Description")),
				Textarea(
					ID("description_addition"),
					Name("description_addition"),
					Class("w-full p-3 border-2 border-gray-300 rounded-lg focus:border-blue-500 focus:ring-2 focus:ring-blue-200 transition"),
					Rows("4"),
					MaxLength("500"),
					Placeholder("Add additional information (will be timestamped and appended)"),
					g.Attr("autofocus"),
				),
				Div(Class("text-sm text-gray-600 mt-2 bg-blue-50 p-2 rounded"),
					g.Text("Your addition will be appended with a timestamp. Total description must remain under 500 characters."),
				),
			),
		}),
	})
}

func shareButton(ad ad.Ad) g.Node {
	return iconButton(
		"/images/share.svg",
		"Share",
		"Share ad",
		hx.Get(fmt.Sprintf("/modal/ad/share/%d", ad.ID)),
		hx.Target("body"),
		hx.Swap("beforeend"),
	)
}

func duplicateButton(ad ad.Ad) g.Node {
	return iconLink(
		"/images/duplicate.svg",
		"Duplicate",
		"Duplicate ad",
		fmt.Sprintf("/duplicate-ad/%d", ad.ID),
	)
}

func ShareModal(ad ad.Ad) g.Node {
	modalID := fmt.Sprintf("share-modal-%d", ad.ID)
	adPath := fmt.Sprintf("/ad/%d", ad.ID)
	urlInputID := fmt.Sprintf("ad-url-%d", ad.ID)
	copyButtonID := fmt.Sprintf("copy-button-%d", ad.ID)
	copyFeedbackID := fmt.Sprintf("copy-feedback-%d", ad.ID)

	return Div(
		ID(modalID),
		Class("fixed inset-0 bg-black/30 flex items-center justify-center z-50 p-8"),
		g.Attr("onclick", "this.remove()"),
		Div(
			Class("bg-white rounded-lg w-full shadow-2xl border-2 border-gray-300 flex flex-col overflow-hidden"),
			Style("max-width: 500px;"),
			g.Attr("onclick", "event.stopPropagation()"),
			Div(Class("p-8"),
				H3(Class("text-2xl font-bold mb-6 text-gray-900"),
					g.Text("Share Ad")),
				Div(Class("mb-6"),
					Label(Class("block text-sm font-semibold text-gray-800 mb-3"),
						g.Text("Ad Link")),
					Input(
						Type("text"),
						ID(urlInputID),
						Class("w-full p-3 border-2 border-gray-300 rounded-lg bg-gray-50"),
						Value(""),
						g.Attr("readonly"),
					),
					Div(
						ID(copyFeedbackID),
						Class("hidden mt-2 text-sm text-green-600 font-medium"),
						g.Text("✓ Copied to clipboard!"),
					),
				),
				Div(
					Class("flex gap-3 justify-end"),
					modalCloseButton(),
					modalCopyButton(copyButtonID, urlInputID, copyFeedbackID),
				),
			),
		),
		// Script to set the full URL when modal is opened
		Script(
			g.Raw(fmt.Sprintf(`
				(function() {
					const modal = document.getElementById('%s');
					const urlInput = document.getElementById('%s');
					// Set URL immediately since modal is now visible when loaded
					urlInput.value = window.location.origin + '%s';
				})();
			`, modalID, urlInputID, adPath)),
		),
	)
}

func MessageModal(ad ad.Ad, conversationContent g.Node) g.Node {
	modalID := fmt.Sprintf("message-modal-%d", ad.ID)

	return Div(
		ID(modalID),
		Class("fixed inset-0 bg-black/30 flex items-center justify-center z-50 p-8"),
		g.Attr("onclick", "this.remove()"),
		Div(
			Class("bg-white rounded-lg w-full shadow-2xl border-2 border-gray-300 flex flex-col overflow-hidden"),
			Style("max-width: 600px; max-height: 80vh"),
			g.Attr("onclick", "event.stopPropagation()"),
			Div(Class("p-6 flex flex-col h-full"),
				Div(
					Class("flex items-center justify-between mb-4"),
					H3(Class("text-xl font-bold text-gray-900"),
						g.Text("Message Seller")),
					Button(
						Type("button"),
						Class("bg-white border-2 border-gray-800 rounded-full w-10 h-10 flex items-center justify-center shadow-lg hover:bg-gray-100 focus:outline-none cursor-pointer"),
						g.Attr("onclick", "this.closest('.fixed').remove()"),
						icon("/images/close.svg", "Close", "w-6 h-6"),
					),
				),
				Div(
					Class("flex-1 overflow-hidden"),
					conversationContent,
				),
			),
		),
	)
}

func formatYearRanges(years []string) string {
	if len(years) == 0 {
		return ""
	}
	if len(years) == 1 {
		return years[0]
	}

	// Convert strings to integers for proper sorting
	yearInts := make([]int, len(years))
	for i, yearStr := range years {
		fmt.Sscanf(yearStr, "%d", &yearInts[i])
	}

	// Sort years numerically
	sort.Ints(yearInts)

	// Group consecutive years into ranges
	var ranges []string
	start := yearInts[0]
	end := yearInts[0]

	for i := 1; i < len(yearInts); i++ {
		if yearInts[i] == end+1 {
			// Consecutive year, extend range
			end = yearInts[i]
		} else {
			// Non-consecutive year, close current range and start new one
			if start == end {
				ranges = append(ranges, fmt.Sprintf("%d", start))
			} else {
				ranges = append(ranges, fmt.Sprintf("%d-%d", start, end))
			}
			start = yearInts[i]
			end = yearInts[i]
		}
	}

	// Add the last range
	if start == end {
		ranges = append(ranges, fmt.Sprintf("%d", start))
	} else {
		ranges = append(ranges, fmt.Sprintf("%d-%d", start, end))
	}

	return strings.Join(ranges, ", ")
}

func pathSegmentLink(text string, accumulatedSegments []string) g.Node {
	searchQuery := strings.Join(accumulatedSegments, " ")
	searchURL := fmt.Sprintf("/search-query?q=%s", url.QueryEscape(searchQuery))

	return A(
		Href(searchURL),
		Class("hover:underline hover:text-blue-600"),
		g.Text(text),
	)
}

func intersperseSeparators(nodes []g.Node) g.Node {
	if len(nodes) == 0 {
		return g.Node(nil)
	}

	result := []g.Node{nodes[0]}
	for i := 1; i < len(nodes); i++ {
		result = append(result, g.Text(" | "), nodes[i])
	}
	return g.Group(result)
}

// commaSeparatedLinks splits comma-separated text and makes each item clickable
func commaSeparatedLinks(text string, accumulatedSegments []string) g.Node {
	// Split by comma and trim whitespace
	items := strings.Split(text, ",")
	for i, item := range items {
		items[i] = strings.TrimSpace(item)
	}

	if len(items) == 1 {
		// Single item, just return a regular link
		return pathSegmentLink(items[0], accumulatedSegments)
	}

	// Multiple items, create links for each with comma separators
	var linkNodes []g.Node
	for i, item := range items {
		if i > 0 {
			linkNodes = append(linkNodes, g.Text(", "))
		}
		// Create new accumulated segments with this specific item
		newSegments := make([]string, len(accumulatedSegments))
		copy(newSegments, accumulatedSegments)
		// Replace the last segment (which was the full comma-separated text) with just this item
		newSegments[len(newSegments)-1] = item
		linkNodes = append(linkNodes, pathSegmentLink(item, newSegments))
	}

	return g.Group(linkNodes)
}

func partTypePath(ad ad.Ad) g.Node {
	var segments []string  // Keep strings for building cumulative queries
	var linkNodes []g.Node // Build link nodes

	// Add make
	if ad.Make != "" {
		segments = append(segments, ad.Make)
		linkNodes = append(linkNodes, pathSegmentLink(ad.Make, segments))
	}

	// Add years (format as ranges when possible, comma-separated for individual years)
	if len(ad.Years) > 0 {
		yearRanges := formatYearRanges(ad.Years)
		segments = append(segments, yearRanges)
		linkNodes = append(linkNodes, commaSeparatedLinks(yearRanges, segments))
	}

	// Add models (join multiple with comma)
	if len(ad.Models) > 0 {
		models := strings.Join(ad.Models, ", ")
		segments = append(segments, models)
		linkNodes = append(linkNodes, commaSeparatedLinks(models, segments))
	}

	// Add engines (join multiple with comma)
	if len(ad.Engines) > 0 {
		engines := strings.Join(ad.Engines, ", ")
		segments = append(segments, engines)
		linkNodes = append(linkNodes, commaSeparatedLinks(engines, segments))
	}

	// Add category
	if ad.PartCategory.Valid && ad.PartCategory.String != "" {
		segments = append(segments, ad.PartCategory.String)
		linkNodes = append(linkNodes, pathSegmentLink(ad.PartCategory.String, segments))
	}

	// Add subcategory
	if ad.PartSubcategory.Valid && ad.PartSubcategory.String != "" {
		segments = append(segments, ad.PartSubcategory.String)
		linkNodes = append(linkNodes, pathSegmentLink(ad.PartSubcategory.String, segments))
	}

	// If no path parts, return empty node
	if len(linkNodes) == 0 {
		return g.Node(nil)
	}

	// Intersperse links with " | " separators
	return Div(
		Class("text-xs text-gray-500 mt-4 pt-2 border-t border-gray-200"),
		intersperseSeparators(linkNodes),
	)
}
