package ui

import (
	"fmt"
	"net/url"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

const textFieldClass = "w-full p-2 border border-zinc-300 dark:border-zinc-200 rounded-md " +
	"bg-transparent"

// Autofocus marks a field for initial focus. HTMX focuses [autofocus] after swaps.
func Autofocus() g.Node {
	return g.Attr("autofocus", "autofocus")
}

func labeledTextInput(labelText, fieldID string, attrs ...g.Node) g.Node {
	inputAttrs := append([]g.Node{Class(textFieldClass), ID(fieldID)}, attrs...)
	return Div(
		fieldLabel(labelText, fieldID),
		Input(inputAttrs...),
	)
}

func labeledSelect(labelText, fieldID string, attrs ...g.Node) g.Node {
	selectAttrs := append([]g.Node{Class(textFieldClass), ID(fieldID)}, attrs...)
	return Div(
		fieldLabel(labelText, fieldID),
		Select(selectAttrs...),
	)
}

// PasswordFieldView holds state for a password input with visibility toggle.
type PasswordFieldView struct {
	Name         string
	ID           string // DOM id stem; defaults to Name when empty
	Autocomplete string
	Value        string
	Visible      bool
	Autofocus    bool
	// PreventAutofill uses readonly-until-focus so password managers
	// don't fill the field on page load (e.g. change-password forms
	// that also include a username for the save prompt).
	PreventAutofill bool
}

func passwordFieldID(view PasswordFieldView) string {
	id := view.ID
	if id == "" {
		id = view.Name
	}
	return "password-field-" + id
}

func passwordFieldWrapID(view PasswordFieldView) string {
	return passwordFieldID(view) + "-wrap"
}

func passwordFieldToggleURL(view PasswordFieldView, visible bool) string {
	vis := "false"
	if visible {
		vis = "true"
	}
	id := view.ID
	if id == "" {
		id = view.Name
	}
	urlStr := fmt.Sprintf(
		"/api/password-field?name=%s&id=%s&visible=%s&autocomplete=%s",
		url.QueryEscape(view.Name),
		url.QueryEscape(id),
		vis,
		url.QueryEscape(view.Autocomplete),
	)
	if view.PreventAutofill {
		urlStr += "&prevent_autofill=true"
	}
	return urlStr
}

// PasswordField renders a password input with an HTMX visibility toggle.
func PasswordField(view PasswordFieldView) g.Node {
	fieldID := passwordFieldID(view)
	inputType := "password"
	iconSrc := "/images/eye.svg"
	ariaLabel := "Show password"
	ariaPressed := "false"
	if view.Visible {
		inputType = "text"
		iconSrc = "/images/eye-off.svg"
		ariaLabel = "Hide password"
		ariaPressed = "true"
	}

	inputAttrs := []g.Node{
		Class(textFieldClass + " pr-10"),
		Type(inputType),
		Name(view.Name),
		ID(fieldID),
		MaxLength("32"),
		g.Attr("autocomplete", view.Autocomplete),
		Required(),
	}
	if view.Value != "" {
		inputAttrs = append(inputAttrs, Value(view.Value))
	}
	if view.Autofocus {
		inputAttrs = append(inputAttrs, Autofocus())
	}
	if view.PreventAutofill {
		inputAttrs = append(inputAttrs,
			g.Attr("readonly", "readonly"),
			g.Attr("onfocus", "this.removeAttribute('readonly')"),
		)
	}

	wrapID := passwordFieldWrapID(view)
	toggleVisible := !view.Visible
	return Div(
		ID(wrapID),
		Class("relative"),
		Input(inputAttrs...),
		Button(
			Type("button"),
			Class(
				"absolute right-2 top-1/2 -translate-y-1/2 p-1 "+
					"rounded hover:bg-zinc-100 dark:hover:bg-zinc-700",
			),
			g.Attr("aria-label", ariaLabel),
			g.Attr("aria-pressed", ariaPressed),
			hx.Post(passwordFieldToggleURL(view, toggleVisible)),
			hx.Target("#"+wrapID),
			hx.Swap("outerHTML"),
			hx.Include("#"+wrapID),
			Img(
				Src(iconSrc),
				Alt(""),
				Class("w-5 h-5 dark:invert dark:opacity-80"),
			),
		),
	)
}

func labeledPasswordField(labelText, name, autocomplete string,
	autofocus bool) g.Node {
	return labeledPasswordFieldID(
		labelText, name, "", autocomplete, autofocus, false,
	)
}

func labeledPasswordFieldID(labelText, name, id, autocomplete string,
	autofocus, preventAutofill bool) g.Node {
	view := PasswordFieldView{
		Name:            name,
		ID:              id,
		Autocomplete:    autocomplete,
		Autofocus:       autofocus,
		PreventAutofill: preventAutofill,
	}
	return Div(
		fieldLabel(labelText, passwordFieldID(view)),
		PasswordField(view),
	)
}

// buttonProps contains properties for creating a button
type buttonProps struct {
	Text     string
	Href     string
	Type     string // "button", "submit", "reset" - only used when Href is empty
	ID       string // Optional ID for the button
	Class    string // Additional classes to append
	Disabled bool   // If true, button will be disabled (for links, renders as disabled button instead)
	Attrs    []g.Node
	Children []g.Node // If provided, Text is ignored
	// Icon button properties
	ImageSrc string // The image source path (e.g., "/images/share.svg")
	Alt      string // Alt text for the image
}

// standardButton creates a standardized button element.
// If Href is provided, returns an anchor element styled as a button.
// Otherwise, returns a button element.
// If Disabled is true and Href is set, renders a disabled button instead of a link.
func standardButton(props buttonProps) g.Node {
	baseClasses := "inline-block px-6 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"

	var classes string
	if props.Class != "" {
		classes = baseClasses + " " + props.Class
	} else {
		classes = baseClasses
	}

	// Add disabled styling if disabled
	if props.Disabled {
		classes += " opacity-50 cursor-not-allowed"
	}

	allAttrs := []g.Node{Class(classes)}
	if props.ID != "" {
		allAttrs = append(allAttrs, ID(props.ID))
	}
	allAttrs = append(allAttrs, props.Attrs...)

	if props.Href != "" && !props.Disabled {
		// Return anchor element (only if not disabled)
		allAttrs = append(allAttrs, Href(props.Href))
		var content g.Node
		if len(props.Children) > 0 {
			content = g.Group(props.Children)
		} else {
			content = g.Text(props.Text)
		}
		allNodes := append(allAttrs, content)
		return A(allNodes...)
	}

	// Return button element (or disabled button if Href was set but Disabled is true)
	if props.Type == "" {
		props.Type = "button"
	}
	allAttrs = append(allAttrs, Type(props.Type))
	if props.Disabled {
		allAttrs = append(allAttrs, Disabled())
	}
	var content g.Node
	if len(props.Children) > 0 {
		content = g.Group(props.Children)
	} else {
		content = g.Text(props.Text)
	}
	allNodes := append(allAttrs, content)
	return Button(allNodes...)
}

// iconButton creates a standardized icon button element
func iconButton(props buttonProps) g.Node {
	classes := "flex-shrink-0 cursor-pointer"

	allAttrs := []g.Node{
		Type("button"),
		Class(classes),
	}
	allAttrs = append(allAttrs, props.Attrs...)

	var imageNode g.Node
	if len(props.Children) > 0 {
		imageNode = g.Group(props.Children)
	} else {
		imageClasses := "w-6 h-6"
		if props.Class != "" {
			imageClasses += " " + props.Class
		}
		imageNode = Img(
			Class(imageClasses),
			Src(props.ImageSrc),
			Alt(props.Alt),
		)
	}

	allNodes := append(allAttrs, imageNode)
	return Button(allNodes...)
}

func checkbox(name string, value string, label string, checked bool,
	disabled bool, attrs ...g.Node) g.Node {
	inputClass := "mt-0.5 h-5 w-5 shrink-0 accent-blue-600 cursor-pointer"
	labelClass := "text-sm leading-snug text-zinc-600 dark:text-zinc-300 cursor-pointer"
	if disabled {
		inputClass += " opacity-50 cursor-not-allowed"
		labelClass = "text-sm leading-snug text-zinc-400"
	}

	inputAttrs := []g.Node{
		Type("checkbox"),
		Name(name),
		Value(value),
		ID(name + "-" + value),
		Class(inputClass),
	}

	if checked {
		inputAttrs = append(inputAttrs, Checked())
	}
	if disabled {
		inputAttrs = append(inputAttrs, Disabled())
	}

	inputAttrs = append(inputAttrs, attrs...)

	return Div(
		Class("flex items-start gap-3"),
		Input(inputAttrs...),
		Label(
			For(name+"-"+value),
			Class(labelClass),
			g.Text(label),
		),
	)
}

func fieldLabel(text, forID string) g.Node {
	return Label(
		For(forID),
		Class("block text-base font-medium mb-1 text-zinc-900 dark:text-zinc-200"),
		g.Text(text),
	)
}

// label renders a section heading styled like a form label.
func label(text string) g.Node {
	return Span(
		Class("block text-base font-medium mb-1 text-zinc-900 dark:text-zinc-200"),
		g.Text(text),
	)
}

// inputText creates a standardized text input element with dark mode support
func inputText(name, placeholder string, isRequired bool,
	attrs ...g.Node) g.Node {
	inputAttrs := []g.Node{
		Type("text"),
		Name(name),
		Class(textFieldClass),
		Placeholder(placeholder),
		g.If(isRequired, Required()),
	}
	inputAttrs = append(inputAttrs, attrs...)
	return Input(inputAttrs...)
}

// pageTitle creates a standardized H1 page title element
func pageTitle(text string) g.Node {
	return H1(
		Class("text-3xl font-bold"),
		g.Text(text),
	)
}

// RemoveModal returns swap-oob delete elements to remove a modal and its backdrop
func RemoveModal(name string) []g.Node {
	return []g.Node{
		Div(
			ID(name+"-modal-backdrop"),
			hx.SwapOOB("delete"),
		),
		Div(
			ID(name+"-modal"),
			hx.SwapOOB("delete"),
		),
	}
}

// modalBackdrop creates a backdrop div that removes itself (and modal) when clicked
func modalBackdrop(name string) g.Node {
	return Div(
		ID(name+"-modal-backdrop"),
		Class("fixed inset-0 bg-black/30 z-40"),
		hx.Get(fmt.Sprintf("/api/modal-remove/%s", name)),
		hx.Swap("none"),
		hx.Trigger("click"),
	)
}

// modalClose creates a standardized close button for modals
func modalClose(name string) g.Node {
	return Button(
		Type("button"),
		Class("bg-white dark:bg-zinc-700 border-2 border-zinc-800 dark:border-zinc-500 rounded-full w-8 h-8 flex items-center justify-center shadow-lg hover:bg-zinc-100 dark:hover:bg-zinc-600 focus:outline-none cursor-pointer"),
		hx.Get(fmt.Sprintf("/api/modal-remove/%s", name)),
		hx.Swap("none"),
		Img(
			Src("/images/close.svg"),
			Alt("Close"),
			Class("w-4 h-4 dark:invert"),
		),
	)
}
