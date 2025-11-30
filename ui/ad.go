package ui

import (
	"fmt"
	"strings"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
)

func adID(ad ad.Ad) string {
	return fmt.Sprintf("ad-%d", ad.ID)
}

func adTarget(ad ad.Ad) string {
	return fmt.Sprintf("closest #%s", adID(ad))
}

// priceNode returns price text
func priceNode(ad ad.Ad) g.Node {
	return g.Text(fmt.Sprintf("$%.0f", ad.Price))
}

// titleNode returns title text without styling
func titleNode(ad ad.Ad) g.Node {
	return g.Text(ad.Title)
}

// Returns the Unicode flag for a given country code (e.g., "US" -> 🇺🇸)
func countryFlag(country string) string {
	if len(country) != 2 {
		return ""
	}
	code := strings.ToUpper(country)
	return string(rune(int32(code[0])-'A'+0x1F1E6)) + string(rune(int32(code[1])-'A'+0x1F1E6))
}

// location returns a Div containing flag and location text
func location(a ad.Ad) g.Node {

	// Return nil if no location data
	if a.City == "" && a.AdminArea == "" && a.Country == "" {
		return nil
	}

	// Build location text
	var locationText string
	if a.City != "" && a.AdminArea != "" {
		locationText = a.City + ", " + a.AdminArea
	} else if a.City != "" {
		locationText = a.City
	} else if a.AdminArea != "" {
		locationText = a.AdminArea
	}

	// Return Div with flag and location text
	return Div(
		Class("flex items-center"),
		g.Text(countryFlag(a.Country)),
		Span(Class("ml-1"), g.Text(locationText)),
	)
}

// Helper to format ad age as Xm, Xh, Xd, Xmo, or Xy Xmo
func formatAdAge(t time.Time) string {
	d := time.Since(t)
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}

	days := int(d.Hours() / 24)
	if days <= 31 {
		return fmt.Sprintf("%dd", days)
	}

	// Calculate months and years
	now := time.Now()
	years := now.Year() - t.Year()
	months := int(now.Month()) - int(t.Month())

	// Adjust for day of month
	if now.Day() < t.Day() {
		months--
	}

	// Adjust years if months went negative
	if months < 0 {
		years--
		months += 12
	}

	if years > 0 {
		if months > 0 {
			return fmt.Sprintf("%dy %dmo", years, months)
		}
		return fmt.Sprintf("%dy", years)
	}

	return fmt.Sprintf("%dmo", months)
}

// isNewAd checks if an ad is less than 4 hours old
func isNewAd(ad ad.Ad, loc *time.Location) bool {
	createdAt := ad.CreatedAt.In(loc)
	d := time.Since(createdAt)
	return d < 4*time.Hour
}

// newBadge returns the "New!" badge with orange pill styling
func newBadge() g.Node {
	return Span(
		Class("px-2 py-0.5 rounded-full border border-orange-500 text-orange-500 text-xs font-medium"),
		g.Text("New!"),
	)
}

// ageNode returns age text, optionally showing both "New!" badge and age when ad is new
func ageNode(ad ad.Ad, loc *time.Location, showBoth bool) g.Node {
	createdAt := ad.CreatedAt.In(loc)
	agoStr := formatAdAge(createdAt)

	if isNewAd(ad, loc) {
		if showBoth {
			return Div(
				Class("flex items-center gap-2"),
				newBadge(),
				g.Text(agoStr),
			)
		}
		return newBadge()
	}

	return g.Text(agoStr)
}

// bookmarkIconSrc returns the bookmark icon source based on state
func bookmarkIconSrc(bookmarked bool) string {
	if bookmarked {
		return "/images/bookmark-true.svg"
	}
	return "/images/bookmark-false.svg"
}

// BookmarkButton returns the bookmark toggle button
func BookmarkButton(a ad.Ad, csrfToken string) g.Node {
	var hxMethod g.Node
	if a.Bookmarked {
		hxMethod = hx.Delete(fmt.Sprintf("/api/bookmark-ad/%d", a.ID))
	} else {
		hxMethod = hx.Post(fmt.Sprintf("/api/bookmark-ad/%d", a.ID))
	}

	return iconButton(
		bookmarkIconSrc(a.Bookmarked),
		"Bookmark",
		"Toggle bookmark",
		hxMethod,
		hx.Target("this"),
		hx.Swap("outerHTML"),
		hxHeadersCSRFToken(csrfToken),
		ID(fmt.Sprintf("bookmark-btn-%d", a.ID)),
		g.Attr("onclick", "event.stopPropagation()"),
	)
}

func AdPage(a ad.AdDetail, userID int, userName, csrfToken string, path string, loc *time.Location) g.Node {
	return Page(
		fmt.Sprintf("Ad %d - Parts Pile", a.ID),
		userID,
		userName,
		path,
		csrfToken,
		[]g.Node{
			AdDetail(a, userID, loc, csrfToken, ""),
		},
	)
}

// ---- Ad Components ----
