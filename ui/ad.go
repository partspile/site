package ui

import (
	"fmt"
	"strings"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/user"
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
func location(ad ad.Ad) g.Node {
	var city string
	if ad.City.Valid {
		city = ad.City.String
	}
	var adminArea string
	if ad.AdminArea.Valid {
		adminArea = ad.AdminArea.String
	}
	var country string
	if ad.Country.Valid {
		country = ad.Country.String
	}

	// Return nil if no location data
	if city == "" && adminArea == "" && country == "" {
		return nil
	}

	// Build location text
	var locationText string
	if city != "" && adminArea != "" {
		locationText = city + ", " + adminArea
	} else if city != "" {
		locationText = city
	} else if adminArea != "" {
		locationText = adminArea
	}

	// Return Div with flag and location text
	return Div(
		Class("flex items-center"),
		g.Text(countryFlag(country)),
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

// ageNode returns age text
func ageNode(ad ad.Ad, loc *time.Location) g.Node {
	agoStr := formatAdAge(ad.CreatedAt.In(loc))
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
func BookmarkButton(ad ad.Ad) g.Node {
	var hxMethod g.Node
	if ad.Bookmarked {
		hxMethod = hx.Delete(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID))
	} else {
		hxMethod = hx.Post(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID))
	}

	return iconButton(
		bookmarkIconSrc(ad.Bookmarked),
		"Bookmark",
		"Toggle bookmark",
		hxMethod,
		hx.Target("this"),
		hx.Swap("outerHTML"),
		ID(fmt.Sprintf("bookmark-btn-%d", ad.ID)),
		g.Attr("onclick", "event.stopPropagation()"),
	)
}

func AdPage(adObj ad.AdDetail, currentUser *user.User, userID int, path string, loc *time.Location, view string) g.Node {
	return Page(
		fmt.Sprintf("Ad %d - Parts Pile", adObj.ID),
		currentUser,
		path,
		[]g.Node{
			AdDetail(adObj, loc, userID, view),
		},
	)
}

// ---- Ad Components ----
