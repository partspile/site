package ui

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/local"
	uiads "github.com/partspile/site/internal/ui/ads"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

func BookmarkButton(adID int, bookmarked bool, csrfToken string) g.Node {
	var imageNode g.Node
	if bookmarked {
		imageNode = Img(
			Class("w-6 h-6"),
			Src("/images/bookmark-true.svg"),
			Alt("Bookmark"),
		)
	} else {
		imageNode = Img(
			Class("w-6 h-6 dark:invert dark:opacity-80"),
			Src("/images/bookmark-false.svg"),
			Alt("Bookmark"),
		)
	}

	return iconButton(buttonProps{
		Children: []g.Node{imageNode},
		Attrs: []g.Node{
			g.If(bookmarked, hx.Delete(fmt.Sprintf("/auth/bookmark/%d", adID))),
			g.If(!bookmarked, hx.Post(fmt.Sprintf("/auth/bookmark/%d", adID))),
			hx.Headers(fmt.Sprintf(`{"X-Csrf-Token": %q}`, csrfToken)),
			hx.Target("this"),
			hx.Swap("outerHTML"),
			g.Attr("onclick", "event.stopPropagation()"),
		},
	})
}

func gridImageNode(adID, count, current int) g.Node {
	if count == 0 {
		return noImage("aspect-[4/3]")
	}
	return ImageNode(adID, count, current, "480w", "aspect-[4/3]", false)
}

func formatListedAge(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "Listed just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "Listed 1 minute ago"
		}
		return fmt.Sprintf("Listed %d minutes ago", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "Listed 1 hour ago"
		}
		return fmt.Sprintf("Listed %d hours ago", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "Listed 1 day ago"
	}
	if days < 7 {
		return fmt.Sprintf("Listed %d days ago", days)
	}
	weeks := days / 7
	if weeks == 1 {
		return "Listed a week ago"
	}
	if days < 30 {
		return fmt.Sprintf("Listed %d weeks ago", weeks)
	}
	months := days / 30
	if months == 1 {
		return "Listed a month ago"
	}
	if days < 365 {
		return fmt.Sprintf("Listed %d months ago", months)
	}
	years := days / 365
	if years == 1 {
		return "Listed a year ago"
	}
	return fmt.Sprintf("Listed %d years ago", years)
}

func newBadge() g.Node {
	return Span(
		Class("inline-flex align-middle whitespace-nowrap px-2 py-0.5 rounded-full bg-zinc-100 text-black text-xs font-medium shadow-sm"),
		g.Text("Just listed"),
	)
}

func newBadgeImageOverlay() g.Node {
	return Div(
		Class("absolute top-[10px] left-[10px] z-10 pointer-events-none"),
		newBadge(),
	)
}

func listedAgeDetailNode(createdAt time.Time) g.Node {
	return Div(
		Class("justify-self-end shrink-0 whitespace-nowrap text-right text-xs text-zinc-500"),
		g.Text(formatListedAge(createdAt)),
	)
}

func paginationDiv(nextPage int) g.Node {
	return Div(
		hx.Get(fmt.Sprintf("/api/search/?page=%d", nextPage)),
		hx.Trigger("revealed"),
		hx.Target("this"),
		hx.Swap("outerHTML"),
		hx.Include("form"),
	)
}

func adCardMeta(location string) g.Node {
	return Div(
		Class("text-xs text-zinc-500 whitespace-nowrap"),
		g.Text(location),
	)
}

func adCardTitle(title, facetLabel string) g.Node {
	return adCardTitleWithBadge(title, facetLabel, false, time.Time{})
}

func adCardTitleWithBadge(title, facetLabel string, showNewBadge bool,
	createdAt time.Time) g.Node {
	return Span(
		Class("min-w-0"),
		g.Text(title),
		g.If(facetLabel != "",
			Span(Class("text-xs text-zinc-500 whitespace-nowrap"),
				g.Text(" · "+facetLabel))),
		g.If(showNewBadge && time.Since(createdAt) < 4*time.Hour,
			Span(Class("ml-1"), newBadge())),
	)
}

func priceSpan(priceDisplay string, hasPrice bool) g.Node {
	if !hasPrice {
		return g.Text("")
	}
	return Span(
		Class("shrink-0 whitespace-nowrap text-green-600 font-semibold"),
		g.Text(priceDisplay),
	)
}

func AdGridNode(userID, adID, imageCount, nextPage int, priceDisplay, title,
	location, facetLabel, csrfToken string, hasPrice bool, createdAt time.Time,
	active, bookmarked, isLast bool, rockCount int) g.Node {
	class := "flex flex-col cursor-pointer gap-1 py-3"
	if !active {
		class += " bg-red-100 dark:bg-red-900 border border-red-300 dark:border-red-700 rounded-lg"
	}

	node := A(
		Href("/ad/"+strconv.Itoa(adID)),
		Class(class),
		Div(
			Class("relative"),
			gridImageNode(adID, imageCount, 1),
			g.If(time.Since(createdAt) < 4*time.Hour, newBadgeImageOverlay()),
		),
		Div(
			Class("flex items-start gap-2 pt-1 min-w-0"),
			priceSpan(priceDisplay, hasPrice),
			Span(
				Class("min-w-0 flex-1 text-right text-xs text-zinc-500"),
				g.Text(location),
			),
		),
		Div(
			Class("flex items-start gap-2 min-w-0"),
			g.If(local.IsLoggedIn(userID) && bookmarked, BookmarkButton(adID, bookmarked, csrfToken)),
			g.If(rockCount > 0, RockIcons(adID, rockCount)),
			adCardTitle(title, facetLabel),
		),
	)

	if isLast {
		return g.Group([]g.Node{
			node,
			paginationDiv(nextPage),
		})
	}

	return node
}

func AdListNode(userID, adID int, priceDisplay, title, location,
	facetLabel string, hasPrice bool, createdAt time.Time, active, bookmarked bool,
	csrfToken string, isLast bool, nextPage int, rockCount int) g.Node {
	class := "grid grid-cols-[minmax(0,1fr)_auto_auto] " +
		"gap-x-2 items-baseline py-2 cursor-pointer"
	if active {
		class += " hover:bg-zinc-50 dark:hover:bg-zinc-800"
	} else {
		class += " bg-red-100 dark:bg-red-900 border border-red-300 dark:border-red-700 rounded-lg"
	}

	node := A(
		Href("/ad/"+strconv.Itoa(adID)),
		Class(class),
		Div(
			Class("flex items-start gap-2 min-w-0"),
			g.If(local.IsLoggedIn(userID) && bookmarked, BookmarkButton(adID, bookmarked, csrfToken)),
			g.If(rockCount > 0, RockIcons(adID, rockCount)),
			adCardTitleWithBadge(title, facetLabel, true, createdAt),
		),
		Div(
			Class("text-right"),
			adCardMeta(location),
		),
		Div(
			Class("justify-self-end"),
			priceSpan(priceDisplay, hasPrice),
		),
	)

	if isLast {
		return g.Group([]g.Node{
			node,
			paginationDiv(nextPage),
		})
	}

	return node
}

func deletedWatermark() g.Node {
	return statusWatermark("DELETED")
}

func pausedWatermark() g.Node {
	return statusWatermark("PAUSED")
}

func statusWatermark(text string) g.Node {
	return Div(
		Class("absolute top-0 left-0 right-0 bottom-0 flex items-center justify-center pointer-events-none z-50"),
		Div(
			Class("font-bold text-8xl text-red-500 transform rotate-[-45deg]"),
			g.Text(text),
		),
	)
}

func testWatermark() g.Node {
	return Div(
		Class("absolute top-0 left-0 right-0 bottom-0 flex items-center justify-center pointer-events-none z-50"),
		Div(
			Class("watermark-test text-center whitespace-pre-line"),
			g.Text("TEST\nAD"),
		),
	)
}

func shareButton(adID int) g.Node {
	return iconButton(buttonProps{
		ImageSrc: "/images/share.svg",
		Alt:      "Share Ad",
		Class:    "dark:invert dark:opacity-80",
		Attrs: []g.Node{
			hx.Get(fmt.Sprintf("/api/ad/%d/share", adID)),
			hx.Target("body"),
			hx.Swap("beforeend"),
		},
	})
}

func deleteButton(adID int) g.Node {
	return iconButton(buttonProps{
		ImageSrc: "/images/trashcan.svg",
		Alt:      "Remove Ad",
		Class:    "dark:invert dark:opacity-80",
		Attrs: []g.Node{
			hx.Get(fmt.Sprintf("/auth/ad/%d/remove-modal", adID)),
			hx.Target("body"),
			hx.Swap("beforeend"),
		},
	})
}

func restoreButton(adID int, csrfToken string) g.Node {
	return iconButton(buttonProps{
		ImageSrc: "/images/restore.svg",
		Alt:      "Restore Ad",
		Class:    "dark:invert dark:opacity-80",
		Attrs: []g.Node{
			hx.Post(fmt.Sprintf("/auth/ad/%d/restore", adID)),
			hx.Headers(fmt.Sprintf(`{"X-Csrf-Token": %q}`, csrfToken)),
			hx.Target("body"),
			hx.Swap("outerHTML"),
			g.Attr("hx-confirm", "Are you sure you want to restore this paused ad?"),
		},
	})
}

func messageButton(adID int) g.Node {
	return iconButton(buttonProps{
		ImageSrc: "/images/message.svg",
		Alt:      "Message",
		Class:    "dark:invert dark:opacity-80",
		Attrs: []g.Node{
			hx.Get(fmt.Sprintf("/api/ad/%d/new-conversation", adID)),
			hx.Target("#page-content"),
			hx.Swap("beforeend"),
		},
	})
}

func adButtons(adID, userID, ownerID int, bookmarked, active, inactive, reachable bool,
	csrfToken string) g.Node {
	isOwner := local.IsLoggedIn(userID) && userID == ownerID
	return Div(
		Class("flex shrink-0 items-center gap-2 justify-self-end"),
		g.If(active && local.IsLoggedIn(userID),
			BookmarkButton(adID, bookmarked, csrfToken)),
		g.If(active && reachable && !isOwner, messageButton(adID)),
		g.If(active && isOwner, editButton(adID)),
		g.If(active && isOwner, deleteButton(adID)),
		g.If(inactive && isOwner, restoreButton(adID, csrfToken)),
		g.If(active, shareButton(adID)),
	)
}

func editButton(adID int) g.Node {
	return iconButton(buttonProps{
		ImageSrc: "/images/edit.svg",
		Alt:      "Edit",
		Class:    "dark:invert dark:opacity-80",
		Attrs: []g.Node{
			g.Attr("onclick", fmt.Sprintf(
				"event.stopPropagation(); window.location.href=%q;",
				fmt.Sprintf("/auth/ad/%d/edit", adID),
			)),
		},
	})
}

func copyButton(path string, copied bool) g.Node {
	var text string
	var attrs []g.Node

	if copied {
		text = "Copied!"
		attrs = []g.Node{
			hx.Get(fmt.Sprintf("/api/ad/share/copy?copied=false&path=%s", url.QueryEscape(path))),
			hx.Target("#copy-link-button"),
			hx.Swap("outerHTML"),
			hx.Trigger("load delay:2s"),
		}
	} else {
		text = "Copy"
		attrs = []g.Node{
			g.Attr("onclick", fmt.Sprintf(`navigator.clipboard.writeText(%q);`, path)),
			hx.Get(fmt.Sprintf("/api/ad/share/copy?copied=true&path=%s", url.QueryEscape(path))),
			hx.Target("#copy-link-button"),
			hx.Swap("outerHTML"),
			hx.Trigger("click"),
		}
	}

	return standardButton(buttonProps{
		Type:  "button",
		ID:    "copy-link-button",
		Class: "flex items-center justify-center gap-2 transition-all duration-200 min-w-[140px]",
		Attrs: attrs,
		Children: []g.Node{
			Div(
				Class("flex items-center justify-center gap-2 whitespace-nowrap"),
				Img(
					Src("/images/copy.svg"),
					Alt("Copy"),
					Class("w-4 h-4"),
				),
				g.Text(text),
			),
		},
	})
}

func CopyButton(path string) g.Node {
	return copyButton(path, false)
}

func CopyButtonCopied(path string) g.Node {
	return copyButton(path, true)
}

func AdShareModal(path, flyerHref string) g.Node {
	return g.Group([]g.Node{
		modalBackdrop("ad-share"),
		Div(
			ID("ad-share-modal"),
			Class("fixed inset-0 flex items-center justify-center z-50 p-8 pointer-events-none"),
			Div(
				Class("bg-white dark:bg-zinc-800 rounded-lg w-full max-w-md shadow-2xl border-2 border-zinc-300 dark:border-zinc-600 flex flex-col pointer-events-auto"),
				Div(
					Class("flex items-center justify-between p-6 border-b border-zinc-200 dark:border-zinc-700 flex-shrink-0"),
					H3(Class("text-xl font-bold text-zinc-900 dark:text-zinc-200"), g.Text("Share Ad")),
					modalClose("ad-share"),
				),
				Div(
					Class("p-6 flex flex-col gap-6"),
					shareAdLinkSection(path),
					g.If(flyerHref != "",
						shareFlyerSection(flyerHref)),
				),
			),
		),
	})
}

func shareAdLinkSection(path string) g.Node {
	return Div(
		Class("flex flex-col gap-2"),
		Label(
			Class("text-sm font-medium text-zinc-700 dark:text-zinc-300"),
			For("ad-link-input"),
			g.Text("Ad Link"),
		),
		Div(
			Class("flex flex-wrap items-center gap-2"),
			Input(
				ID("ad-link-input"),
				Type("text"),
				Value(path),
				g.Attr("readonly", ""),
				g.Attr("onfocus", "this.select();"),
				Class("grow px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-zinc-50 dark:bg-zinc-700 text-sm text-zinc-900 dark:text-zinc-200"),
			),
			copyButton(path, false),
		),
	)
}

func shareFlyerSection(href string) g.Node {
	return Div(
		Class("flex flex-col gap-2 pt-4 border-t border-zinc-200 "+
			"dark:border-zinc-700"),
		Div(
			Class("text-sm font-medium text-zinc-700 dark:text-zinc-300"),
			g.Text("Flyer"),
		),
		P(
			Class("text-sm text-zinc-600 dark:text-zinc-400"),
			g.Text("Print a one-page flyer with this ad's photos, "+
				"details, and a QR code. Save as PDF from the print "+
				"dialog if you prefer."),
		),
		standardButton(buttonProps{
			Href:  href,
			Text:  "Print flyer",
			Class: "self-start",
		}),
	)
}

func Ad(d AdDetail, userID int, csrfToken string) []g.Node {
	nodes := []g.Node{}
	if userID == d.OwnerID && d.Active {
		nodes = append(nodes, adExpireToolbar(d.ID, d.ExpiresAt, csrfToken))
	}
	nodes = append(nodes, Div(
		Class("flex flex-col relative rounded-none sm:rounded-lg shadow-lg dark:shadow-xl dark:shadow-zinc-900/50 my-4 -mx-6 sm:mx-2 col-span-full overflow-hidden bg-white dark:bg-zinc-800 border-y sm:border border-zinc-200 dark:border-zinc-700"),
		Div(
			Class("relative"),
			g.If(d.ImageCount > 0, ImageNodeWithThumbnails(d.ID, d.ImageCount, 1, "1200w", "aspect-[4/3] w-full", true)),
			g.If(d.ImageCount == 0, noImage("h-32 w-full")),
			g.If(time.Since(d.CreatedAt) < 4*time.Hour, newBadgeImageOverlay()),
		),
		g.If(d.Deleted, deletedWatermark()),
		g.If(d.Inactive, pausedWatermark()),
		g.If(d.IsTest && d.Active, testWatermark()),
		Div(
			Class("p-3 sm:p-6 flex flex-col bg-white dark:bg-zinc-800"),
			Div(
				Class("grid grid-cols-[minmax(0,1fr)_auto] gap-x-2 gap-y-1 items-start"),
				Div(
					Class("flex items-center gap-2 min-w-0"),
					g.If(d.RockCount > 0, RockIcons(d.ID, d.RockCount)),
					adCardTitle(d.Title, d.FacetLabel),
				),
				adButtons(d.ID, userID, d.OwnerID, d.Bookmarked, d.Active, d.Inactive, d.Reachable, csrfToken),
				Div(
					Class("flex flex-wrap items-baseline gap-x-2 gap-y-0.5 min-w-0"),
					priceSpan(d.PriceDisplay, d.HasPrice),
					Span(Class("min-w-0 text-xs text-zinc-500"), g.Text(d.Location)),
				),
				listedAgeDetailNode(d.CreatedAt),
			),
			descriptionDisplay(
				d.ID,
				d.DescriptionOriginal,
				d.FacetDetails,
				d.Tags,
				d.DescriptionHistory,
				d.ShowLoginForDetails,
			),
		),
	))
	return nodes
}

func adExpireToolbar(adID int, expiresAt time.Time, csrfToken string) g.Node {
	return Div(
		Class("flex items-center justify-between gap-2 mx-2 mt-4 mb-0 col-span-full"),
		Div(
			Class("flex items-center gap-2 min-w-0 text-xs text-zinc-500"),
			Span(g.Text(formatExpiresIn(expiresAt))),
			g.If(ad.RenewEligible(expiresAt, time.Now()),
				renewExpireLink(adID, csrfToken)),
		),
		Div(
			Class("flex items-center gap-2 shrink-0"),
			newAdLink(),
			copyAdLink(adID),
		),
	)
}

func renewExpireLink(adID int, csrfToken string) g.Node {
	return A(
		Href("#"),
		Class("underline cursor-pointer"),
		hx.Post(fmt.Sprintf("/auth/ad/%d/renew", adID)),
		hx.Headers(fmt.Sprintf(`{"X-Csrf-Token": %q}`, csrfToken)),
		hx.Target("body"),
		hx.Swap("outerHTML"),
		g.Text("Renew"),
	)
}

func copyAdLink(adID int) g.Node {
	const label = "Copy ad"
	return A(
		Href(fmt.Sprintf("/auth/ad/%d/copy", adID)),
		Class("flex-shrink-0 cursor-pointer"),
		g.Attr("aria-label", label),
		g.Attr("title", label),
		Img(
			Class("w-6 h-6 dark:invert dark:opacity-80"),
			Src("/images/copy.svg"),
			Alt(label),
		),
	)
}

func formatExpiresIn(expiresAt time.Time) string {
	return formatExpiresInAt(expiresAt, time.Now())
}

func formatExpiresInAt(expiresAt, now time.Time) string {
	if expiresAt.Sub(now) < 24*time.Hour {
		return "Expires soon"
	}
	// Calendar months match AdExpireMonths (AddDate months), not
	// fixed 30-day blocks — else a fresh 3-month grant reads as
	// "3 months 1 day" / "3 months 2 days".
	months, days := calendarMonthsDays(now, expiresAt)

	monthPart := ""
	if months == 1 {
		monthPart = "1 month"
	} else if months > 1 {
		monthPart = fmt.Sprintf("%d months", months)
	}
	dayPart := ""
	if days == 1 {
		dayPart = "1 day"
	} else if days > 1 {
		dayPart = fmt.Sprintf("%d days", days)
	}

	switch {
	case monthPart != "" && dayPart != "":
		return fmt.Sprintf("Expires in %s %s", monthPart, dayPart)
	case monthPart != "":
		return "Expires in " + monthPart
	case dayPart != "":
		return "Expires in " + dayPart
	default:
		return "Expires soon"
	}
}

// calendarMonthsDays is whole calendar months and leftover days from
// from→to using calendar dates (same basis as time.AddDate months).
func calendarMonthsDays(from, to time.Time) (months, days int) {
	from = from.In(to.Location())
	y1, m1, d1 := from.Date()
	y2, m2, d2 := to.Date()

	months = (y2-y1)*12 + int(m2-m1)
	days = d2 - d1
	if days < 0 {
		months--
		// Day 0 of to's month is the last day of the previous month.
		lastPrev := time.Date(y2, m2, 0, 0, 0, 0, 0, to.Location()).Day()
		days += lastPrev
	}
	if months < 0 {
		return 0, 0
	}
	return months, days
}

func descriptionDisplay(adID int, original string, facetDetails []string,
	tags []string, history []AdHistoryEntry, showLoginForDetails bool) g.Node {
	var nodes []g.Node
	if len(facetDetails) > 0 {
		nodes = append(nodes, adFacetList(facetDetails))
	}
	nodes = append(nodes, Div(
		Class("whitespace-pre-wrap"),
		uiads.DescriptionTextWithLinks(original),
	))
	if len(tags) > 0 {
		nodes = append(nodes, adPills(tags, "mt-3"))
	}
	if len(history) > 0 {
		entryNodes := make([]g.Node, len(history))
		for i, e := range history {
			entryNodes[i] = descriptionHistoryEntry(adID, e)
		}
		nodes = append(nodes, Div(
			Class("mt-4 pt-4 border-t border-zinc-200 dark:border-zinc-600 space-y-4"),
			g.Group(entryNodes),
		))
	} else if showLoginForDetails {
		nodes = append(nodes, Div(
			Class("mt-4 pt-4 border-t border-zinc-200 dark:border-zinc-600"),
			A(
				Href("/login?return="+url.QueryEscape(
					fmt.Sprintf("/ad/%d", adID))),
				Class("text-sm text-blue-600 dark:text-blue-400 hover:underline"),
				g.Text("Login to see more details..."),
			),
		))
	}
	return Div(Class("text-base mt-4"), g.Group(nodes))
}

func descriptionHistoryEntry(adID int, e AdHistoryEntry) g.Node {
	imageNodes := make([]g.Node, len(e.ImageIndices))
	for i, idx := range e.ImageIndices {
		class := "w-16 h-16 object-cover rounded border " +
			"border-zinc-200 dark:border-zinc-600"
		src := AdImageSrc(adID, idx, "160w")
		if src == "" {
			imageNodes[i] = GenerateSVG(adID, idx, "160w", class)
		} else {
			imageNodes[i] = Img(
				Src(src),
				Alt(fmt.Sprintf("Added image %d", idx)),
				Class(class),
			)
		}
	}
	return Div(
		Class("text-sm text-blue-700 dark:text-blue-300"),
		Div(
			Class("flex items-start gap-2"),
			Img(
				Src("/images/edit.svg"),
				Alt("Edit"),
				Class("w-4 h-4 mt-0.5 shrink-0 dark:invert dark:opacity-80"),
			),
			Div(
				Class("min-w-0"),
				Div(Class("font-medium"), g.Text(e.Header)),
				g.If(e.Body != "",
					Div(
						Class("whitespace-pre-wrap mt-1 "+
							"text-blue-600/90 dark:text-blue-200/90"),
						uiads.DescriptionTextWithLinks(e.Body),
					),
				),
				g.If(len(e.ImageIndices) > 0,
					Div(
						Class("flex flex-wrap gap-2 mt-2"),
						g.Group(imageNodes),
					),
				),
			),
		),
	)
}

func AdUnavailable() []g.Node {
	return []g.Node{
		Div(
			Class("text-center py-16"),
			Div(
				Class("mb-6 flex justify-center"),
				Img(
					Src("/images/trashcan.svg"),
					Alt("Unavailable"),
					Class("w-24 h-24"),
				),
			),
			H2(
				Class("text-3xl font-bold mb-4"),
				g.Text("Ad Unavailable"),
			),
			P(
				Class("text-lg text-zinc-600 dark:text-zinc-400 mb-8"),
				g.Text("This ad is no longer available."),
			),
			standardButton(buttonProps{
				Text: "Back to Home",
				Href: "/",
			}),
		),
	}
}

func AdRemoveModal(adID int, csrfToken string) g.Node {
	csrfHeader := fmt.Sprintf(`{"X-Csrf-Token": %q}`, csrfToken)
	return g.Group([]g.Node{
		modalBackdrop("ad-remove"),
		Div(
			ID("ad-remove-modal"),
			Class("fixed inset-0 flex items-center justify-center z-50 p-8 pointer-events-none"),
			Div(
				Class("bg-white dark:bg-zinc-800 rounded-lg w-full max-w-md shadow-2xl border-2 border-zinc-300 dark:border-zinc-600 flex flex-col pointer-events-auto"),
				Div(
					Class("flex items-center justify-between p-6 border-b border-zinc-200 dark:border-zinc-700 flex-shrink-0"),
					H3(Class("text-xl font-bold text-zinc-900 dark:text-zinc-200"), g.Text("Pause or delete?")),
					modalClose("ad-remove"),
				),
				Div(
					Class("p-6 flex flex-col gap-6"),
					Div(
						Class("flex flex-col gap-2"),
						H4(Class("font-semibold text-zinc-900 dark:text-zinc-200"), g.Text("Pause ad")),
						P(
							Class("text-sm text-zinc-600 dark:text-zinc-400"),
							g.Text("Hide this ad from search and listings. You can turn it back on later from My Ads."),
						),
						standardButton(buttonProps{
							Text:  "Pause Ad",
							Class: "bg-amber-600 hover:bg-amber-700 w-full text-center",
							Attrs: []g.Node{
								hx.Post(fmt.Sprintf("/auth/ad/%d/pause", adID)),
								hx.Headers(csrfHeader),
								hx.Swap("none"),
							},
						}),
					),
					Div(
						Class("flex flex-col gap-2"),
						H4(Class("font-semibold text-zinc-900 dark:text-zinc-200"), g.Text("Delete forever")),
						P(
							Class("text-sm text-zinc-600 dark:text-zinc-400"),
							g.Text("Permanently remove this ad. It cannot be restored. Existing conversations stay in your inbox but messaging will stop."),
						),
						standardButton(buttonProps{
							Text:  "Delete Forever",
							Class: "bg-red-600 hover:bg-red-700 w-full text-center",
							Attrs: []g.Node{
								hx.Delete(fmt.Sprintf("/auth/ad/%d/delete", adID)),
								hx.Headers(csrfHeader),
								hx.Swap("none"),
								g.Attr("hx-confirm", "Permanently delete this ad? This cannot be undone."),
							},
						}),
					),
				),
			),
		),
	})
}

func imagesField(maxImagesPerAd int) g.Node {
	return imageUploadField(maxImagesPerAd, 0, false)
}

func editImagesField(adID, existingCount, maxImagesPerAd int) g.Node {
	if existingCount == 0 {
		return imageUploadField(maxImagesPerAd, 0, false)
	}

	existingNodes := make([]g.Node, existingCount)
	for i := 1; i <= existingCount; i++ {
		class := "object-cover rounded w-[90px] h-[90px]"
		src := AdImageSrc(adID, i, "160w")
		if src == "" {
			existingNodes[i-1] = GenerateSVG(adID, i, "160w", class)
		} else {
			existingNodes[i-1] = Img(
				Src(src),
				Alt(fmt.Sprintf("Image %d", i)),
				Class(class),
			)
		}
	}

	nodes := []g.Node{
		label("Images"),
		Div(
			Class("text-sm text-zinc-500 dark:text-zinc-400"),
			g.Text("Original images cannot be changed."),
		),
		Div(
			ID("existing-image-preview"),
			Class("flex flex-row flex-wrap gap-2"),
			g.Group(existingNodes),
		),
	}

	if existingCount >= maxImagesPerAd {
		nodes = append(nodes, Div(
			Class("text-sm text-zinc-500 dark:text-zinc-400"),
			g.Text(fmt.Sprintf(
				"Maximum of %d images reached.", maxImagesPerAd,
			)),
		))
	} else {
		nodes = append(nodes,
			Div(
				Class("text-sm text-zinc-500 dark:text-zinc-400"),
				g.Text("Add more images (optional)"),
			),
			imageUploadControls(maxImagesPerAd, existingCount, true),
		)
	}

	return Div(Class("space-y-2"), g.Group(nodes))
}

func imageUploadField(maxImagesPerAd, existingCount int, appendMode bool) g.Node {
	return Div(
		Class("space-y-2"),
		label("Images"),
		imageUploadControls(maxImagesPerAd, existingCount, appendMode),
	)
}

func imageUploadControls(maxImagesPerAd, existingCount int,
	appendMode bool) g.Node {
	appendFlag := "false"
	if appendMode {
		appendFlag = "true"
	}
	return Div(
		Input(
			Type("file"),
			ID("images"),
			Name("images"),
			Class("hidden"),
			// application/pdf is only to coax Android Chrome into offering
			// Camera; non-images are rejected in image-preview.js.
			g.Attr("accept", "image/*, application/pdf"),
			g.Attr("multiple"),
			g.Attr("onchange", "previewImages(this)"),
		),
		Div(
			ID("upload-area"),
			Class("border border-zinc-300 dark:border-zinc-600 rounded p-6 "+
				"hover:border-blue-400 hover:bg-blue-50 dark:hover:bg-zinc-800 "+
				"transition-colors duration-200 cursor-pointer"),
			g.Attr("onclick", "handleUploadClick()"),
			g.Attr("ondragover", "event.preventDefault(); "+
				"this.classList.add('border-blue-400', 'bg-blue-50')"),
			g.Attr("ondragleave", "this.classList.remove('border-blue-400', "+
				"'bg-blue-50')"),
			g.Attr("ondrop", "event.preventDefault(); "+
				"this.classList.remove('border-blue-400', 'bg-blue-50'); "+
				"handleDrop(event)"),
			Div(
				ID("upload-content"),
				Class("flex flex-col items-center space-y-4"),
				Div(
					Class("flex flex-col items-center space-y-2"),
					Div(
						Class("w-12 h-12 bg-blue-100 dark:bg-zinc-700 "+
							"rounded-full flex items-center justify-center"),
						g.Raw(`<svg class="w-6 h-6 text-blue-600 dark:text-blue-400" `+
							`fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" `+
							`stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
						</svg>`),
					),
					Div(
						Class("text-lg font-medium text-zinc-700 dark:text-zinc-300"),
						g.Text("Upload Images"),
					),
					Div(
						Class("text-sm text-zinc-500 dark:text-zinc-400"),
						g.Text("Click to browse or drag and drop"),
					),
				),
			),
			Div(
				ID("image-preview"),
				Class("hidden image-preview flex flex-row flex-wrap "+
					"gap-2 justify-center mt-4"),
			),
		),
		g.Raw(fmt.Sprintf(
			`<script>const MAX_IMAGES_PER_AD = %d;`+
				`const EXISTING_IMAGE_COUNT = %d;`+
				`const IMAGE_APPEND_MODE = %s;</script>`,
			maxImagesPerAd, existingCount, appendFlag,
		)),
		g.Raw(`<script src="/js/image-preview.js" defer></script>`),
	)
}

func adFacetList(items []string) g.Node {
	listItems := make([]g.Node, len(items))
	for i, s := range items {
		listItems[i] = Li(g.Text(s))
	}
	return Ul(
		Class("list-disc list-inside mb-3 text-sm text-zinc-700 dark:text-zinc-300"),
		g.Group(listItems),
	)
}

func adPills(labels []string, spacing string) g.Node {
	nodes := make([]g.Node, len(labels))
	for i, s := range labels {
		nodes[i] = Span(
			Class("inline-block px-3 py-1 rounded-full border border-zinc-300 dark:border-zinc-600 text-sm text-zinc-700 dark:text-zinc-300"),
			g.Text(s),
		)
	}
	return Div(
		Class("flex flex-wrap gap-2 "+spacing),
		g.Group(nodes),
	)
}

func adFormStatus() g.Node {
	return Div(
		ID("ad-form-status"),
		Class("hidden text-blue-600 dark:text-blue-400 "+
			"text-sm whitespace-nowrap"),
		Span(ID("ad-form-status-text")),
	)
}

func newAdForm(fields g.Node) g.Node {
	cfg := uiads.NewFormConfig(uiads.AdFormConfig{}.Defaults)
	return adForm(cfg, fields)
}

func adForm(cfg uiads.AdFormConfig, fields g.Node) g.Node {
	mode := "create"
	if cfg.Mode == uiads.AdFormEdit {
		mode = "edit"
	}
	formAttrs := []g.Node{
		Class("space-y-8 mt-8"),
		ID(cfg.FormID),
		g.Attr("novalidate", ""),
		g.Attr("data-ad-post-url", cfg.PostURL),
		g.Attr("data-ad-form-mode", mode),
		g.Attr("onsubmit", "return submitAdForm(event)"),
	}

	children := []g.Node{fields}
	if cfg.Mode == uiads.AdFormCreate {
		children = append(children, imagesField(config.MaxImagesPerAd))
	} else if cfg.Mode == uiads.AdFormEdit {
		children = append(children, editImagesField(
			cfg.AdID, cfg.Values.ImageCount, config.MaxImagesPerAd,
		))
	}
	children = append(children,
		Div(
			Class("flex items-center gap-4 flex-wrap"),
			standardButton(buttonProps{
				Type: "submit",
				Text: cfg.SubmitLabel,
			}),
			adFormStatus(),
			ErrorDiv(""),
		),
		g.Raw(`<script src="/js/image-upload.js" defer></script>`),
	)

	return Form(append(formAttrs, g.Group(children))...)
}

func NewAd(category CategoryOption, categories []CategoryOption,
	fields g.Node) []g.Node {
	return []g.Node{
		Div(Class("mb-4"),
			CategorySelect(category, categories, "/auth/ad/new")),
		pageTitle("Create New Ad"),
		newAdForm(fields),
	}
}

func EditAd(category CategoryOption, cfg uiads.AdFormConfig,
	fields g.Node) []g.Node {
	imagePath := "/images/category/" + category.ImageFile
	return []g.Node{
		Div(
			Class("py-2 px-5 flex items-center gap-2 rounded-full border-2 mb-4 "+
				"border-zinc-300 bg-zinc-100 dark:bg-zinc-800 dark:border-zinc-600"),
			Img(
				Src(imagePath),
				Alt("Category icon"),
				Class("w-6 h-6 dark:invert dark:opacity-80"),
			),
			Span(g.Text(category.Name)),
		),
		pageTitle("Edit Ad"),
		adForm(cfg, fields),
	}
}
