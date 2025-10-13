package ui

import (
	"fmt"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ---- Icon Components ----

// IconButton creates a standardized icon button with consistent styling
func iconButton(iconSrc, alt, title string, attrs ...g.Node) g.Node {
	return Button(
		Type("button"),
		Class("ml-2 focus:outline-none cursor-pointer"),
		Title(title),
		g.Group(attrs),
		Img(
			Src(iconSrc),
			Alt(alt),
			Class("w-6 h-6 inline align-middle"),
		),
	)
}

// IconLink creates a standardized icon link with consistent styling
func iconLink(iconSrc, alt, title, href string, attrs ...g.Node) g.Node {
	return A(
		Href(href),
		Class("ml-2 focus:outline-none cursor-pointer"),
		Title(title),
		g.Group(attrs),
		Img(
			Src(iconSrc),
			Alt(alt),
			Class("w-6 h-6 inline align-middle"),
		),
	)
}

// Icon creates a standardized icon image
func icon(iconSrc, alt string, classes ...string) g.Node {
	class := "w-6 h-6 inline align-middle"
	for _, c := range classes {
		class += " " + c
	}

	return Img(
		Src(iconSrc),
		Alt(alt),
		Class(class),
	)
}

// NavigationIcon creates an icon for navigation menu items
func navigationIcon(iconSrc, alt string) g.Node {
	return Img(
		Src(iconSrc),
		Alt(alt),
		Class("w-6 h-6 mr-2"),
	)
}

// LargeIcon creates a large icon for special contexts
func largeIcon(iconSrc, alt string, size string) g.Node {
	return Img(
		Src(iconSrc),
		Alt(alt),
		Class(fmt.Sprintf("w-%s h-%s", size, size)),
	)
}
