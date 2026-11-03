package ui

import (
	"fmt"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// renderRockIconsByOrdinal renders clickable rock icons using ordinal positions
func renderRockIconsByOrdinal(rockCount int, urlPattern string, id int) g.Node {
	if rockCount <= 0 {
		return g.Raw("")
	}

	var icons []g.Node
	for i := range rockCount {
		icons = append(icons,
			Div(
				Class("rock-icon-container flex items-center gap-0.5 cursor-pointer hover:opacity-80 transition-opacity"),
				g.Attr("onclick", "event.stopPropagation();"),
				hx.Get(fmt.Sprintf(urlPattern, id, i)),
				hx.Target("body"),
				hx.Swap("beforeend"),
				Img(
					Src("/images/flag.svg"),
					Alt("Flag raised"),
					Class("w-5 h-5 flex-shrink-0"),
					Title("Flag raised — click to view dispute assessment"),
				),
			),
		)
	}

	return Div(
		Class("flex items-center gap-0.5"),
		g.Group(icons),
	)
}

// RockIcons renders clickable rock icons for an ad
// Each icon links to a rock by ordinal position
// Only shows rocks bound to the ad (when inquirer throws)
func RockIcons(adID int, rockCount int) g.Node {
	return renderRockIconsByOrdinal(rockCount, "/ad/%d/rock/%d", adID)
}

// UserRockIcons renders clickable rock icons for a user
// Each icon links to a rock by ordinal position
// Shows rocks bound to the user (when owner throws)
func UserRockIcons(userID int, rockCount int) g.Node {
	return renderRockIconsByOrdinal(rockCount, "/auth/user/%d/rock/%d", userID)
}

func StaticRockIcons(rockCount int) g.Node {
	icons := make([]g.Node, 0, rockCount)
	for range rockCount {
		icons = append(icons, Img(
			Src("/images/flag.svg"),
			Alt("Flag raised"),
			Class("w-5 h-5 flex-shrink-0"),
		))
	}
	return Span(
		Class("inline-flex items-center gap-0.5"),
		g.Group(icons),
	)
}

func rockThrowLabel(currentUserID, ownerID int, inquirerName string) string {
	if currentUserID == ownerID {
		return fmt.Sprintf("Raise flag on %s", inquirerName)
	}
	return "Raise flag on ad"
}

// RockThrowLink renders a button to throw/unthrow a rock in the conversation modal
func RockThrowLink(d ConversationModalData) g.Node {
	if !d.HasThrownRock && !d.CanThrowRock {
		return g.Raw("")
	}

	var attrs []g.Node
	var label string

	if d.HasThrownRock {
		label = "Lower flag"
		attrs = []g.Node{
			hx.Delete(fmt.Sprintf(
				"/auth/conversation/%d/rock/unthrow", d.ConversationID)),
			hx.Headers(fmt.Sprintf(`{"X-Csrf-Token": %q}`, d.CSRFToken)),
			hx.Target(ConversationMessagesSelector(d.ConversationID)),
			hx.Swap(conversationMessagesAppendSwap()),
		}
	} else if d.CanThrowRock {
		label = rockThrowLabel(d.CurrentUserID, d.OwnerID, d.InquirerName)
		confirmURL := fmt.Sprintf(
			"/auth/conversation/%d/rock/confirm", d.ConversationID)
		if d.ConversationID == 0 {
			confirmURL = fmt.Sprintf("/auth/ad/%d/rock/confirm", d.AdID)
		}
		attrs = []g.Node{
			hx.Get(confirmURL),
			hx.Target("body"),
			hx.Swap("beforeend"),
		}
	} else {
		return g.Raw("")
	}

	return Button(
		g.Group(attrs),
		Type("button"),
		Class("flex-shrink-0 px-4 py-2 bg-red-500 text-white rounded-md hover:bg-red-600 transition-colors"),
		Title(label),
		g.Attr("aria-label", label),
		g.Attr("aria-pressed", fmt.Sprintf("%t", d.HasThrownRock)),
		g.Text(label),
	)
}

func flagCountLabel(n int) string {
	if n == 1 {
		return "1 flag"
	}
	return fmt.Sprintf("%d flags", n)
}

// RockCountBadge shows user's remaining rocks
func RockCountBadge(rockCount int) g.Node {
	if rockCount <= 0 {
		return g.Raw("")
	}

	return Span(
		Class("px-2 py-1 bg-orange-100 dark:bg-orange-900 text-orange-800 dark:text-orange-200 rounded-full text-xs font-medium"),
		g.Text(flagCountLabel(rockCount)),
	)
}
