package ui

import (
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ---- Button Components ----

// buttonOption represents configuration options for buttons
type buttonOption func(*buttonConfig)

type buttonConfig struct {
	href       string
	disabled   bool
	buttonType string
	class      string
	attributes []g.Node
}

// withHref makes the button a link with the specified href
func withHref(href string) buttonOption {
	return func(c *buttonConfig) {
		c.href = href
	}
}

// withDisabled makes the button disabled
func withDisabled() buttonOption {
	return func(c *buttonConfig) {
		c.disabled = true
	}
}

// withType sets the button type (button, submit, etc.)
func withType(buttonType string) buttonOption {
	return func(c *buttonConfig) {
		c.buttonType = buttonType
	}
}

// withClass adds additional CSS classes
func withClass(class string) buttonOption {
	return func(c *buttonConfig) {
		c.class = class
	}
}

// withAttributes adds additional g.Node attributes
func withAttributes(attrs ...g.Node) buttonOption {
	return func(c *buttonConfig) {
		c.attributes = append(c.attributes, attrs...)
	}
}

// buttonStyled creates a styled button with the given text, base class, and options
func buttonStyled(text, baseClass string, options ...buttonOption) g.Node {
	config := &buttonConfig{}

	// Apply options
	for _, option := range options {
		option(config)
	}

	// Build class string
	class := baseClass
	if config.class != "" {
		class += " " + config.class
	}
	if config.disabled {
		class += " opacity-50 cursor-not-allowed"
	}

	// Build attributes
	attrs := []g.Node{Class(class)}
	if config.buttonType != "" {
		attrs = append(attrs, Type(config.buttonType))
	}
	attrs = append(attrs, config.attributes...)
	attrs = append(attrs, g.Text(text))

	// Return appropriate element
	if config.href != "" {
		if config.disabled {
			return Span(Class(class), g.Text(text))
		}
		attrs = append([]g.Node{Href(config.href)}, attrs...) // Add Href at start, keep all other attrs
		return A(attrs...)
	}

	return Button(attrs...)
}

// button creates a primary button (blue background)
func button(text string, options ...buttonOption) g.Node {
	return buttonStyled(text, "px-4 py-2 rounded inline-block bg-blue-500 text-white hover:bg-blue-600", options...)
}

// buttonSecondary creates a secondary button (blue text, underlined on hover)
func buttonSecondary(text string, options ...buttonOption) g.Node {
	return buttonStyled(text, "px-4 py-2 rounded inline-block text-blue-500 hover:underline", options...)
}

// buttonDanger creates a danger button (red background)
func buttonDanger(text string, options ...buttonOption) g.Node {
	return buttonStyled(text, "px-4 py-2 rounded inline-block bg-red-500 text-white hover:bg-red-600", options...)
}

func actionButtons(buttons ...g.Node) g.Node {
	return Div(
		Class("mt-8 space-x-4"),
		g.Group(buttons),
	)
}
