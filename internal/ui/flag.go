package ui

import (
	"fmt"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// renderFlagIconsByOrdinal renders clickable flag icons using ordinal positions
func renderFlagIconsByOrdinal(flagCount int, urlPattern string, id int) g.Node {
	if flagCount <= 0 {
		return g.Raw("")
	}

	var icons []g.Node
	for i := range flagCount {
		icons = append(icons,
			Div(
				Class("flag-icon-container flex items-center gap-0.5 cursor-pointer hover:opacity-80 transition-opacity"),
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

// FlagIcons renders clickable flag icons for an ad
// Each icon links to a flag by ordinal position
// Only shows flags bound to the ad (when inquirer throws)
func FlagIcons(adID int, flagCount int) g.Node {
	return renderFlagIconsByOrdinal(flagCount, "/ad/%d/flag/%d", adID)
}

// UserFlagIcons renders clickable flag icons for a user
// Each icon links to a flag by ordinal position
// Shows flags bound to the user (when owner throws)
func UserFlagIcons(userID int, flagCount int) g.Node {
	return renderFlagIconsByOrdinal(flagCount, "/auth/user/%d/flag/%d", userID)
}

func StaticFlagIcons(flagCount int) g.Node {
	icons := make([]g.Node, 0, flagCount)
	for range flagCount {
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

func flagThrowLabel(currentUserID, ownerID int, inquirerName string) string {
	if currentUserID == ownerID {
		return fmt.Sprintf("Raise flag on %s", inquirerName)
	}
	return "Raise flag on ad"
}

// FlagThrowLink renders a button to throw/lower a flag in the conversation modal
func FlagThrowLink(d ConversationModalData) g.Node {
	if !d.HasRaisedFlag && !d.CanRaiseFlag {
		return g.Raw("")
	}

	var attrs []g.Node
	var label string

	if d.HasRaisedFlag {
		label = "Lower flag"
		attrs = []g.Node{
			hx.Delete(fmt.Sprintf(
				"/auth/conversation/%d/flag/lower", d.ConversationID)),
			hx.Headers(fmt.Sprintf(`{"X-Csrf-Token": %q}`, d.CSRFToken)),
			hx.Target(ConversationMessagesSelector(d.ConversationID)),
			hx.Swap(conversationMessagesAppendSwap()),
		}
	} else if d.CanRaiseFlag {
		label = flagThrowLabel(d.CurrentUserID, d.OwnerID, d.InquirerName)
		confirmURL := fmt.Sprintf(
			"/auth/conversation/%d/flag/confirm", d.ConversationID)
		if d.ConversationID == 0 {
			confirmURL = fmt.Sprintf("/auth/ad/%d/flag/confirm", d.AdID)
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
		g.Attr("aria-pressed", fmt.Sprintf("%t", d.HasRaisedFlag)),
		g.Text(label),
	)
}

func flagCountLabel(n int) string {
	if n == 1 {
		return "1 flag"
	}
	return fmt.Sprintf("%d flags", n)
}

// FlagCountBadge shows user's remaining flags
func FlagCountBadge(flagCount int) g.Node {
	if flagCount <= 0 {
		return g.Raw("")
	}

	return Span(
		Class("px-2 py-1 bg-orange-100 dark:bg-orange-900 text-orange-800 dark:text-orange-200 rounded-full text-xs font-medium"),
		g.Text(flagCountLabel(flagCount)),
	)
}
