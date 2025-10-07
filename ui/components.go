package ui

import (
	"fmt"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/rock"
)

// ---- Layout Components ----

func GridContainer4(children ...g.Node) g.Node {
	return Div(
		Class("grid grid-cols-4 gap-2 p-4 bg-gray-50 border border-gray-200 rounded-lg"),
		g.Group(children),
	)
}

func contentContainer(content ...g.Node) g.Node {
	return Div(
		Class("max-w-2xl mx-auto"),
		g.Group(content),
	)
}

func sectionHeader(title string, helpText string) g.Node {
	nodes := []g.Node{
		Label(Class("block font-bold"), g.Text(title)),
	}
	if helpText != "" {
		nodes = append(nodes,
			P(
				Class("text-sm text-gray-600 mb-2"),
				g.Text(helpText),
			),
		)
	}
	return g.Group(nodes)
}

// ---- Button Components ----

type ButtonVariant string

const (
	buttonPrimary   ButtonVariant = "primary"
	ButtonSecondary ButtonVariant = "secondary"
	ButtonDanger    ButtonVariant = "danger"
)

func getButtonClass(variant ButtonVariant) string {
	baseClass := "px-4 py-2 rounded inline-block "
	switch variant {
	case buttonPrimary:
		return baseClass + "bg-blue-500 text-white hover:bg-blue-600"
	case ButtonSecondary:
		return baseClass + "text-blue-500 hover:underline"
	case ButtonDanger:
		return baseClass + "bg-red-500 text-white hover:bg-red-600"
	default:
		return baseClass + "bg-blue-500 text-white hover:bg-blue-600"
	}
}

func styledButton(text string, variant ButtonVariant, attrs ...g.Node) g.Node {
	allAttrs := append([]g.Node{Class(getButtonClass(variant))}, attrs...)
	return Button(append(allAttrs, g.Text(text))...)
}

func styledLink(text string, href string, variant ButtonVariant, attrs ...g.Node) g.Node {
	allAttrs := append([]g.Node{Href(href), Class(getButtonClass(variant))}, attrs...)
	return A(append(allAttrs, g.Text(text))...)
}

func styledLinkDisabled(text string, variant ButtonVariant) g.Node {
	return Span(
		Class(getButtonClass(variant)+" opacity-50 cursor-not-allowed"),
		g.Text(text),
	)
}

func actionButtons(buttons ...g.Node) g.Node {
	return Div(
		Class("mt-8 space-x-4"),
		g.Group(buttons),
	)
}

// ---- Message Components ----

func ValidationError(message string) g.Node {
	return Div(
		Class("bg-red-100 border-red-500 text-red-700 px-4 py-3 rounded"),
		g.Text(message),
	)
}

func SuccessMessage(message string, redirectURL string) g.Node {
	nodes := []g.Node{
		Class("bg-green-100 border-green-500 text-green-700 px-4 py-3 rounded"),
		g.Text(message),
	}
	if redirectURL != "" {
		nodes[1] = g.Text(message + "...redirecting")
		nodes = append(nodes, Script(g.Raw(fmt.Sprintf(
			"setTimeout(function() { window.location = '%s' }, %d);",
			redirectURL, config.ServerRedirectDelay.Milliseconds())),
		))
	}
	return Div(nodes...)
}

func resultContainer() g.Node {
	return Div(
		ID("result"),
		Class("mt-4"),
	)
}

func ErrorPage(code int, message string) g.Node {
	return Page(
		fmt.Sprintf("Error %d", code),
		nil, // no current user on error page
		"",  // no current path
		[]g.Node{
			pageHeader(fmt.Sprintf("Error %d", code)),
			P(g.Text(message)),
		},
	)
}

// LoaderDiv renders the loader for infinite scroll (kept for compatibility)
func loaderDiv(url string, view string) g.Node {
	return Div(
		ID("infinite-scroll-loader"),
		Class("h-4"),
		g.Attr("hx-get", url),
		g.Attr("hx-trigger", "intersect once"),
		g.Attr("hx-swap", "outerHTML"),
	)
}

func NoSearchResultsMessage() g.Node {
	return Div(
		Class("flex justify-center items-center p-8"),
		Div(
			Class("text-center"),
			P(Class("text-gray-600 text-lg"), g.Text("Found no results")),
		),
	)
}

// EmptyResponse returns an empty div for HTMX responses that don't need content
func EmptyResponse() g.Node {
	return Div()
}

// ---- Rock Components ----

func RockIcon(count int, resolved bool) g.Node {
	if resolved {
		return Span(
			Class("inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800"),
			g.Text("🪨"),
			Span(Class("ml-1"), g.Text(fmt.Sprintf("%d", count))),
		)
	}
	return Span(
		Class("inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-red-100 text-red-800"),
		g.Text("🪨"),
		Span(Class("ml-1"), g.Text(fmt.Sprintf("%d", count))),
	)
}

func RockButton(adID int, canThrow bool, rockCount int) g.Node {
	if canThrow {
		return Button(
			Type("button"),
			Class("inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-red-100 text-red-800 hover:bg-red-200 transition-colors"),
			hx.Post(fmt.Sprintf("/api/throw-rock/%d", adID)),
			hx.Target(fmt.Sprintf("#rock-section-%d", adID)),
			hx.Swap("outerHTML"),
			g.Text("🪨"),
			Span(Class("ml-1"), g.Text("Throw Rock")),
		)
	}
	return Span(
		Class("inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-500 cursor-not-allowed"),
		g.Text("🪨"),
		Span(Class("ml-1"), g.Text("No Rocks")),
	)
}

func RockSection(adID, rockCount int, canThrow bool, userRockCount int) g.Node {
	if rockCount == 0 && !canThrow {
		return Div()
	}

	nodes := []g.Node{}

	// Show existing rocks
	if rockCount > 0 {
		nodes = append(nodes,
			Div(
				Class("flex items-center gap-2 mb-2"),
				RockIcon(rockCount, false),
				Button(
					Type("button"),
					Class("text-xs text-blue-600 hover:underline"),
					hx.Get(fmt.Sprintf("/api/ad-rocks/%d/conversations", adID)),
					hx.Target(fmt.Sprintf("#rock-conversations-%d", adID)),
					hx.Swap("innerHTML"),
					g.Text("View Conversations"),
				),
			),
			Div(
				ID(fmt.Sprintf("rock-conversations-%d", adID)),
				Class("hidden"),
			),
		)
	}

	// Show rock button if user can throw
	if canThrow {
		nodes = append(nodes,
			Div(
				Class("flex items-center gap-2"),
				RockButton(adID, canThrow, userRockCount),
				Span(
					Class("text-xs text-gray-500"),
					g.Text(fmt.Sprintf("You have %d rocks", userRockCount)),
				),
			),
		)
	}

	return Div(
		Class("border-t pt-2 mt-2"),
		g.Group(nodes),
	)
}

func RockConversations(adID int, rocks []rock.AdRock) g.Node {
	if len(rocks) == 0 {
		return Div(
			Class("text-sm text-gray-500"),
			g.Text("No rocks thrown at this ad."),
		)
	}

	var conversationNodes []g.Node
	for _, rock := range rocks {
		status := "Active"
		statusClass := "bg-red-100 text-red-800"
		if rock.ResolvedAt != nil {
			status = "Resolved"
			statusClass = "bg-green-100 text-green-800"
		}

		conversationNodes = append(conversationNodes,
			Div(
				Class("border rounded p-3 mb-2"),
				Div(
					Class("flex items-center justify-between mb-2"),
					Span(
						Class("text-sm font-medium"),
						g.Text(fmt.Sprintf("Rock by %s", rock.ThrowerName)),
					),
					Span(
						Class(fmt.Sprintf("inline-flex items-center px-2 py-1 rounded-full text-xs font-medium %s", statusClass)),
						g.Text(status),
					),
				),
				Div(
					Class("text-xs text-gray-500 mb-2"),
					g.Text(fmt.Sprintf("Thrown on %s", rock.CreatedAt.Format("Jan 2, 2006"))),
				),
				Div(
					Class("flex items-center gap-2"),
					Button(
						Type("button"),
						Class("text-xs text-blue-600 hover:underline"),
						hx.Get(fmt.Sprintf("/conversation/%d", rock.ConversationID)),
						hx.Target("#conversation-content"),
						hx.Swap("innerHTML"),
						g.Text("View Conversation"),
					),
					g.If(rock.ResolvedAt == nil,
						Button(
							Type("button"),
							Class("text-xs text-green-600 hover:underline"),
							hx.Post(fmt.Sprintf("/api/resolve-rock/%d", rock.ID)),
							hx.Target(fmt.Sprintf("#rock-section-%d", adID)),
							hx.Swap("outerHTML"),
							g.Text("Resolve & Return Rock"),
						),
					),
				),
			),
		)
	}

	return Div(
		Class("space-y-2"),
		H4(
			Class("text-sm font-medium mb-2"),
			g.Text("Rock Conversations"),
		),
		g.Group(conversationNodes),
	)
}
