package templates

import (
	"fmt"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
)

// ---- Page Layout ----

// Page creates the base HTML page template with common head elements and layout
func Page(title string, content []g.Node) g.Node {
	return HTML(
		Head(
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			Title(title),
			Link(Rel("stylesheet"), Href("https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css")),
			Script(Src("https://unpkg.com/htmx.org@1.9.10")),
		),
		Body(
			Div(
				Class("container mx-auto px-4 py-8"),
				g.Group(content),
			),
		),
	)
}

// PageHeader creates a standard page header
func PageHeader(text string) g.Node {
	return H1(Class("text-4xl font-bold mb-8"), g.Text(text))
}

// ---- Layout Components ----

// GridContainer creates a standard grid container
func GridContainer(cols int, children ...g.Node) g.Node {
	return Div(
		Class(fmt.Sprintf("grid grid-cols-%d gap-4", cols)),
		g.Group(children),
	)
}

// ContentContainer wraps page content in a max-width container for readability
func ContentContainer(content ...g.Node) g.Node {
	return Div(
		Class("max-w-2xl mx-auto"),
		g.Group(content),
	)
}

// SectionHeader creates a section header with optional help text
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

// ---- Form Components ----

// FormContainer creates a standard form container with validation error space
func FormContainer(formID string, content ...g.Node) g.Node {
	return Form(
		ID(formID),
		Class("space-y-6"),
		ValidationErrorContainer(),
		g.Group(content),
	)
}

// FormGroup creates a standard form group with label and input
func FormGroup(labelText string, fieldID string, input g.Node) g.Node {
	return Div(
		Class("space-y-2"),
		Label(For(fieldID), Class("block"), g.Text(labelText)),
		input,
	)
}

// Checkbox creates a standard checkbox with label
func Checkbox(id string, value string, label string, checked bool, disabled bool, attrs ...g.Node) g.Node {
	inputAttrs := []g.Node{
		Type("checkbox"),
		Name(id),
		Value(value),
		ID(id + "-" + value),
	}

	if checked {
		inputAttrs = append(inputAttrs, Checked())
	}
	if disabled {
		inputAttrs = append(inputAttrs, Disabled())
		inputAttrs = append(inputAttrs, g.Attr("class", "opacity-50 cursor-not-allowed"))
	}

	inputAttrs = append(inputAttrs, attrs...)

	labelNode := Label(
		For(id+"-"+value),
		func() g.Node {
			if disabled {
				return Class("text-gray-400")
			}
			return g.Text("")
		}(),
		g.Text(label),
	)

	return Div(
		Class("flex items-center space-x-2"),
		Input(inputAttrs...),
		labelNode,
	)
}

// ---- Button Components ----

// ButtonVariant defines the style variant for a button
type ButtonVariant string

const (
	ButtonPrimary   ButtonVariant = "primary"
	ButtonSecondary ButtonVariant = "secondary"
	ButtonDanger    ButtonVariant = "danger"
)

// getButtonClass returns the CSS classes for a button variant
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

// StyledButton creates a styled button for actions (form submission, state changes)
func StyledButton(text string, variant ButtonVariant, attrs ...g.Node) g.Node {
	allAttrs := append([]g.Node{Class(getButtonClass(variant))}, attrs...)
	return Button(append(allAttrs, g.Text(text))...)
}

// StyledLink creates a styled link for navigation
func StyledLink(text string, href string, variant ButtonVariant, attrs ...g.Node) g.Node {
	allAttrs := append([]g.Node{Href(href), Class(getButtonClass(variant))}, attrs...)
	return A(append(allAttrs, g.Text(text))...)
}

// DeleteButton creates a standard delete button with confirmation
func DeleteButton(id int) g.Node {
	return StyledButton("Delete Ad", ButtonDanger,
		hx.Delete(fmt.Sprintf("/delete-ad/%d", id)),
		hx.Confirm("Are you sure you want to delete this ad? This action cannot be undone."),
		hx.Target("#result"),
	)
}

// BackToListingsButton creates a standard back button
func BackToListingsButton() g.Node {
	return A(
		Href("/"),
		Class("text-blue-500 hover:underline"),
		g.Text("← Back to listings"),
	)
}

// ActionButtons creates a container for action buttons with consistent spacing
func ActionButtons(buttons ...g.Node) g.Node {
	return Div(
		Class("mt-8 space-x-4"),
		g.Group(buttons),
	)
}

// ---- Message Components ----

// ValidationErrorContainer creates a container for validation errors
func ValidationErrorContainer() g.Node {
	return Div(
		ID("validationError"),
		Class("hidden bg-red-100 border-red-500 text-red-700 px-4 py-3 rounded mb-4"),
	)
}

// ValidationError creates a validation error message
func ValidationError(message string) g.Node {
	return Div(
		Class("bg-red-100 border-red-500 text-red-700 px-4 py-3 rounded"),
		g.Text(message),
	)
}

// SuccessMessage creates a success message with optional redirect script
func SuccessMessage(message string, redirectScript string) g.Node {
	return Div(
		Class("bg-green-100 border-green-500 text-green-700 px-4 py-3 rounded"),
		g.Text(message),
		Script(g.Raw(redirectScript)),
	)
}

// SuccessMessageWithRedirect creates a standard success message with redirect
func SuccessMessageWithRedirect(message string, redirectURL string) g.Node {
	return Div(
		Class("bg-green-100 border-green-500 text-green-700 px-4 py-3 rounded"),
		g.Raw(fmt.Sprintf(`
			<div>%s</div>
			<script>setTimeout(function() { window.location = '%s' }, 1000)</script>
		`, message+" Redirecting...", redirectURL)),
	)
}

// ResultContainer creates a container for async operation results (like HTMX responses)
func ResultContainer() g.Node {
	return Div(
		ID("result"),
		Class("mt-4"),
	)
}

// ---- Ad Components ----

// AdDetails creates a standardized ad details display
func AdDetails(ad ad.Ad) g.Node {
	return GridContainer(1,
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Years: %v", ad.Years))),
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Models: %v", ad.Models))),
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Engines: %v", ad.Engines))),
		P(Class("mt-4"), g.Text(ad.Description)),
		P(Class("text-2xl font-bold mt-4"), g.Text(fmt.Sprintf("$%.2f", ad.Price))),
	)
}

// AdCard renders a single ad card for use in lists
func AdCard(ad ad.Ad) g.Node {
	return A(
		Href(fmt.Sprintf("/ad/%d", ad.ID)),
		Class("block border p-4 mb-4 rounded hover:bg-gray-50"),
		Div(
			H3(Class("text-xl font-bold"), g.Text(ad.Make)),
			P(Class("text-gray-600"), g.Text(fmt.Sprintf("Years: %v", ad.Years))),
			P(Class("text-gray-600"), g.Text(fmt.Sprintf("Models: %v", ad.Models))),
			P(Class("mt-2"), g.Text(ad.Description)),
			P(Class("text-xl font-bold mt-2"), g.Text(fmt.Sprintf("$%.2f", ad.Price))),
		),
	)
}
