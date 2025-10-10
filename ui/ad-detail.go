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
	// Determine full class strings based on deleted status
	var containerClass, contentClass string
	if ad.IsArchived() {
		containerClass = "border rounded-lg shadow-lg bg-red-100 flex flex-col relative my-4 mx-2 col-span-full overflow-hidden"
		contentClass = "p-4 flex flex-col gap-2 bg-red-100"
	} else {
		containerClass = "border rounded-lg shadow-lg bg-white flex flex-col relative my-4 mx-2 col-span-full overflow-hidden"
		contentClass = "p-4 flex flex-col gap-2 bg-white"
	}

	isOwner := userID == ad.UserID && userID != 0

	return Div(
		ID(adID(ad)),
		Class(containerClass),
		imageNode(ad, view),
		g.If(ad.IsArchived(), deletedWatermark()),
		Div(
			Class(contentClass),
			// Title and buttons row
			Div(
				Class("flex flex-row items-center justify-between mb-2"),
				Div(Class("font-semibold text-xl truncate"), titleNode(ad)),
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
			// Price row with inline edit for owner
			Div(
				Class("flex flex-row items-center gap-2 mb-2"),
				g.If(isOwner && !ad.IsArchived(),
					priceEditable(ad),
				),
				g.If(!isOwner || ad.IsArchived(),
					price(ad),
				),
			),
			// Age and location row with inline edit for owner
			Div(
				Class("flex flex-row items-center justify-between text-xs text-gray-500 mb-2"),
				Div(Class("text-gray-400"), ageNode(ad, loc)),
				g.If(isOwner && !ad.IsArchived(),
					locationEditable(ad),
				),
				g.If(!isOwner || ad.IsArchived(),
					location(ad),
				),
			),
			// Description with inline edit for owner
			g.If(isOwner && !ad.IsArchived(),
				descriptionEditable(ad),
			),
			g.If(!isOwner || ad.IsArchived(),
				description(ad),
			),
		),
		// Modal dialogs (hidden by default)
		g.If(isOwner && !ad.IsArchived(), priceEditModal(ad, view)),
		g.If(isOwner && !ad.IsArchived(), locationEditModal(ad, view)),
		g.If(isOwner && !ad.IsArchived(), descriptionEditModal(ad, view)),
		shareModal(ad),
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
	// Determine full class string based on deleted status
	var imageContainerClass string
	if ad.IsArchived() {
		imageContainerClass = "relative w-full bg-red-100 overflow-visible"
	} else {
		imageContainerClass = "relative w-full bg-gray-100 overflow-visible"
	}

	return Div(
		Class(imageContainerClass),
		Style("height: 60vh; min-height: 500px; max-height: 800px;"),
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
		Title("Delete ad"),
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

func restoreButton(ad ad.Ad, userID int) g.Node {
	if userID != ad.UserID {
		return g.Node(nil)
	}

	return Button(
		Type("button"),
		Class("ml-2 focus:outline-none"),
		Title("Restore ad"),
		hx.Post(fmt.Sprintf("/restore-ad/%d", ad.ID)),
		hx.Target(adTarget(ad)),
		hx.Swap("outerHTML"),
		hx.Confirm("Are you sure you want to restore this ad?"),
		Img(
			Src("/images/restore.svg"),
			Alt("Restore"),
			Class("w-6 h-6 inline align-middle text-green-600 hover:text-green-700"),
		),
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
		Button(
			Type("button"),
			Class("px-4 bg-blue-500 text-white rounded hover:bg-blue-600"),
			Style("height: 40px"),
			g.Attr("onclick", fmt.Sprintf("document.getElementById('price-modal-%d').classList.remove('hidden')", ad.ID)),
			g.Text("Edit"),
		),
	)
}

func locationEditable(ad ad.Ad) g.Node {
	return Div(
		Class("flex items-center gap-2"),
		location(ad),
		Button(
			Type("button"),
			Class("px-4 bg-blue-500 text-white rounded hover:bg-blue-600"),
			Style("height: 40px"),
			g.Attr("onclick", fmt.Sprintf("document.getElementById('location-modal-%d').classList.remove('hidden')", ad.ID)),
			g.Text("Edit"),
		),
	)
}

func descriptionEditable(ad ad.Ad) g.Node {
	return Div(
		Class("mt-2"),
		description(ad),
		Button(
			Type("button"),
			Class("px-4 bg-blue-500 text-white rounded hover:bg-blue-600"),
			Style("height: 40px"),
			g.Attr("onclick", fmt.Sprintf("document.getElementById('description-modal-%d').classList.remove('hidden')", ad.ID)),
			g.Text("Edit"),
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
		Class("hidden fixed inset-0 bg-black/30 flex items-center justify-center z-50 p-8"),
		g.Attr("onclick", fmt.Sprintf("if (event.target.id === '%s') this.classList.add('hidden')", cfg.modalID)),
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
					g.Attr("hx-on::after-request", fmt.Sprintf("document.getElementById('%s').classList.add('hidden')", cfg.modalID)),
					cfg.formContent,
					Div(
						Class("flex gap-3 justify-end"),
						Button(
							Type("button"),
							Class("px-6 py-3 bg-gray-200 text-gray-800 rounded-lg hover:bg-gray-300 font-medium transition"),
							g.Attr("onclick", fmt.Sprintf("document.getElementById('%s').classList.add('hidden')", cfg.modalID)),
							g.Text("Cancel"),
						),
						Button(
							Type("submit"),
							Class("px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-medium shadow-md transition"),
							g.Text(cfg.submitBtnText),
						),
					),
				),
			),
		),
	)
}

func priceEditModal(ad ad.Ad, view string) g.Node {
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

func locationEditModal(ad ad.Ad, view string) g.Node {
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

func descriptionEditModal(ad ad.Ad, view string) g.Node {
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
	return Button(
		Type("button"),
		Class("ml-2 focus:outline-none"),
		Title("Share ad"),
		g.Attr("onclick", fmt.Sprintf(
			"document.getElementById('share-modal-%d').classList.remove('hidden')",
			ad.ID)),
		Img(
			Src("/images/share.svg"),
			Alt("Share"),
			Class("w-6 h-6 inline align-middle text-blue-500 hover:text-blue-700"),
		),
	)
}

func duplicateButton(ad ad.Ad) g.Node {
	return A(
		Href(fmt.Sprintf("/duplicate-ad/%d", ad.ID)),
		Class("ml-2 focus:outline-none"),
		Title("Duplicate ad"),
		Img(
			Src("/images/duplicate.svg"),
			Alt("Duplicate"),
			Class("w-6 h-6 inline align-middle text-blue-500 hover:text-blue-700"),
		),
	)
}

func shareModal(ad ad.Ad) g.Node {
	modalID := fmt.Sprintf("share-modal-%d", ad.ID)
	adPath := fmt.Sprintf("/ad/%d", ad.ID)
	urlInputID := fmt.Sprintf("ad-url-%d", ad.ID)
	copyButtonID := fmt.Sprintf("copy-button-%d", ad.ID)
	copyFeedbackID := fmt.Sprintf("copy-feedback-%d", ad.ID)

	return Div(
		ID(modalID),
		Class("hidden fixed inset-0 bg-black/30 flex items-center justify-center z-50 p-8"),
		g.Attr("onclick", fmt.Sprintf(
			"if (event.target.id === '%s') this.classList.add('hidden')",
			modalID)),
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
					Button(
						Type("button"),
						Class("px-6 py-3 bg-gray-200 text-gray-800 rounded-lg hover:bg-gray-300 font-medium transition"),
						g.Attr("onclick", fmt.Sprintf(
							"document.getElementById('%s').classList.add('hidden')",
							modalID)),
						g.Text("Close"),
					),
					Button(
						Type("button"),
						ID(copyButtonID),
						Class("px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-medium shadow-md transition flex items-center gap-2"),
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
						Img(
							Src("/images/copy.svg"),
							Alt("Copy"),
							Class("w-5 h-5 inline"),
						),
						g.Text("Copy"),
					),
				),
			),
		),
		// Script to set the full URL when modal is opened
		Script(
			g.Raw(fmt.Sprintf(`
				(function() {
					const modal = document.getElementById('%s');
					const urlInput = document.getElementById('%s');
					const observer = new MutationObserver(function(mutations) {
						mutations.forEach(function(mutation) {
							if (!modal.classList.contains('hidden') && urlInput.value === '') {
								urlInput.value = window.location.origin + '%s';
							}
						});
					});
					observer.observe(modal, { attributes: true, attributeFilter: ['class'] });
				})();
			`, modalID, urlInputID, adPath)),
		),
	)
}
