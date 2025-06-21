package ui

import (
	"fmt"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/config"
)

// ---- Layout Components ----

func GridContainer(cols int, children ...g.Node) g.Node {
	return Div(
		Class(fmt.Sprintf("grid grid-cols-%d gap-4", cols)),
		g.Group(children),
	)
}

func ContentContainer(content ...g.Node) g.Node {
	return Div(
		Class("max-w-2xl mx-auto"),
		g.Group(content),
	)
}

func SectionHeader(title string, helpText string) g.Node {
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
	ButtonPrimary   ButtonVariant = "primary"
	ButtonSecondary ButtonVariant = "secondary"
	ButtonDanger    ButtonVariant = "danger"
)

func getButtonClass(variant ButtonVariant) string {
	baseClass := "px-4 py-2 rounded inline-block "
	switch variant {
	case ButtonPrimary:
		return baseClass + "bg-blue-500 text-white hover:bg-blue-600"
	case ButtonSecondary:
		return baseClass + "text-blue-500 hover:underline"
	case ButtonDanger:
		return baseClass + "bg-red-500 text-white hover:bg-red-600"
	default:
		return baseClass + "bg-blue-500 text-white hover:bg-blue-600"
	}
}

func StyledButton(text string, variant ButtonVariant, attrs ...g.Node) g.Node {
	allAttrs := append([]g.Node{Class(getButtonClass(variant))}, attrs...)
	return Button(append(allAttrs, g.Text(text))...)
}

func StyledLink(text string, href string, variant ButtonVariant, attrs ...g.Node) g.Node {
	allAttrs := append([]g.Node{Href(href), Class(getButtonClass(variant))}, attrs...)
	return A(append(allAttrs, g.Text(text))...)
}

func StyledLinkDisabled(text string, variant ButtonVariant) g.Node {
	return Span(
		Class(getButtonClass(variant)+" opacity-50 cursor-not-allowed"),
		g.Text(text),
	)
}

func DeleteButton(id int) g.Node {
	return StyledButton("Delete Ad", ButtonDanger,
		hx.Delete(fmt.Sprintf("/delete-ad/%d", id)),
		hx.Confirm("Are you sure you want to delete this ad? This action cannot be undone."),
		hx.Target("#result"),
	)
}

func BackToListingsButton() g.Node {
	return A(
		Href("/"),
		Class("text-blue-500 hover:underline"),
		g.Text("← Back to listings"),
	)
}

func ActionButtons(buttons ...g.Node) g.Node {
	return Div(
		Class("mt-8 space-x-4"),
		g.Group(buttons),
	)
}

// ---- Message Components ----

func ValidationErrorContainer() g.Node {
	return Div(
		ID("validationError"),
		Class("hidden bg-red-100 border-red-500 text-red-700 px-4 py-3 rounded mb-4"),
	)
}

func ValidationError(message string) g.Node {
	return Div(
		Class("bg-red-100 border-red-500 text-red-700 px-4 py-3 rounded"),
		g.Text(message),
	)
}

func SuccessMessage(message string, redirectScript string) g.Node {
	return Div(
		Class("bg-green-100 border-green-500 text-green-700 px-4 py-3 rounded"),
		g.Text(message),
		Script(g.Raw(redirectScript)),
	)
}

func SuccessMessageWithRedirect(message string, redirectURL string) g.Node {
	return Div(
		Class("bg-green-100 border-green-500 text-green-700 px-4 py-3 rounded"),
		g.Raw(fmt.Sprintf(`
			<div>%s</div>
			<script>setTimeout(function() { window.location = '%s' }, %d)</script>
		`, message+" Redirecting...", redirectURL, config.RedirectDelay.Milliseconds())),
	)
}

func ResultContainer() g.Node {
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
			PageHeader(fmt.Sprintf("Error %d", code)),
			P(g.Text(message)),
		},
	)
}
