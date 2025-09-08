package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	"maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/user"
)

// ---- Page Layout ----

func Page(title string, currentUser *user.User, currentPath string, content []g.Node) g.Node {
	return components.HTML5(components.HTML5Props{
		Title:    title,
		Language: "en",
		Head: []g.Node{
			Link(Rel("icon"), Type("image/png"), Href("/images/favicon-32x32.png"), g.Attr("sizes", "32x32")),
			Link(Rel("icon"), Type("image/png"), Href("/images/favicon-16x16.png"), g.Attr("sizes", "16x16")),
			Link(
				Rel("stylesheet"),
				Href(config.TailwindCSSURL),
			),
			// Leaflet CSS for map functionality
			Link(
				Rel("stylesheet"),
				Href(config.LeafletCSSURL),
			),
			Script(
				Type("text/javascript"),
				Src(config.HTMXURL),
				Defer(),
			),
			Script(
				Type("text/javascript"),
				Src(config.HTMXSSEURL),
				Defer(),
			),
			// Leaflet JS for map functionality
			Script(
				Type("text/javascript"),
				Src(config.LeafletJSURL),
				Defer(),
			),
			// Custom map functionality
			Script(
				Type("text/javascript"),
				Src("/js/map.js"),
				Defer(),
			),
			// Script(
			// 	Type("text/javascript"),
			// 	g.Raw("if(window.htmx){htmx.logAll()} else {document.addEventListener('htmx:load',function(){htmx.logAll()})}"),
			// ),
		},
		Body: []g.Node{
			Div(
				Class("container mx-auto px-4 py-8"),
				hx.Headers(`js:{'X-Timezone': Intl.DateTimeFormat().resolvedOptions().timeZone}`),
				navigation(currentUser, currentPath),
				g.Group(content),
			),
		},
	})
}

func pageHeader(text string) g.Node {
	return H1(Class("text-4xl font-bold mb-8"), g.Text(text))
}
